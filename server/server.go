package server

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"tcr_netcentric/models"
	"tcr_netcentric/utils"
	"time"
)

type Player struct {
	Conn            net.Conn
	Username        string
	Exp             int
	Level           int
	Mana            int
	Troops          [3][]*models.Troop
	DestroyedTowers [3]bool
}

var players []*Player
var mu sync.Mutex
var playerTowers [2][3]*models.Tower
var currentTurn int
var commands [2]string

func Main() {
	ln, err := net.Listen("tcp", ":9000")
	if err != nil {
		log.Fatalf("Cannot listen: %v", err)
	}
	defer ln.Close()
	fmt.Println("🚀 Server is starting...")

	for len(players) < 2 {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("Connection error:", err)
			continue
		}
		utils.SendMessage(conn, "🔐 Username: ")
		username := utils.ReadFromConn(conn)
		utils.SendMessage(conn, "🔐 Password: ")
		password := utils.ReadFromConn(conn)

		if !utils.Authenticate(username, password) {
			utils.SendMessage(conn, "Authentication failed.\n")
			conn.Close()
			continue
		}

		exp := utils.GetExp(username)
		player := &Player{
			Conn:     conn,
			Username: username,
			Exp:      exp,
			Level:    utils.GetLevel(exp),
			Mana:     5,
		}

		mu.Lock()
		players = append(players, player)
		mu.Unlock()
		utils.SendMessage(conn, fmt.Sprintf("✅ Welcome %s! Waiting for other player...\n", username))
	}

	broadcast("🟢 Game started! Please take turns to summon your troops.\n")
	loadedTowers, _ := utils.LoadTowerSpecs()
	for i := 0; i < 2; i++ {
		playerTowers[i][0] = &models.Tower{}
		playerTowers[i][1] = &models.Tower{}
		playerTowers[i][2] = &models.Tower{}

		*playerTowers[i][0] = loadedTowers["GuardTower"]
		*playerTowers[i][1] = loadedTowers["GuardTower"]
		*playerTowers[i][2] = loadedTowers["KingTower"]

		level := players[i].Level
		scaleTowerStats(playerTowers[i][0], level)
		scaleTowerStats(playerTowers[i][1], level)
		scaleTowerStats(playerTowers[i][2], level)
	}

	currentTurn = 0
	commands = [2]string{"", ""}

	for {
		currentPlayer := players[currentTurn]
		utils.SendMessage(currentPlayer.Conn, fmt.Sprintf("🔁 Your turn (Mana: %d): ", currentPlayer.Mana))
		input := utils.ReadFromConn(currentPlayer.Conn)
		if input == "" {
			continue
		}

		success := handleCommand(currentTurn, input)
		if success {
			commands[currentTurn] = input
			currentTurn = (currentTurn + 1) % 2
		}
		if commands[0] != "" && commands[1] != "" {
			for lane := 0; lane < 3; lane++ {
				resolveLaneCombat(lane)
			}
			for _, p := range players {
				p.Mana += 2
				if p.Mana > 10 {
					p.Mana = 10
				}
				utils.SendMessage(p.Conn, fmt.Sprintf("🔋 Mana = %d\n", p.Mana))
			}
			commands = [2]string{"", ""}
		}
	}
}

func handleCommand(playerIndex int, input string) bool {
	p := players[playerIndex]
	opponentIndex := 1 - playerIndex
	opponent := players[opponentIndex]

	parts := strings.Fields(input)
	if len(parts) == 0 {
		utils.SendMessage(p.Conn, "Invalid command\n")
		return false
	}

	switch parts[0] {
	case "end":
		utils.SendMessage(p.Conn, "🔚 You ended your turn.\n")
		return true

	case "summon":
		if len(parts) != 5 {
			utils.SendMessage(p.Conn, "Usage: summon <TroopName> <guard|king> <Number> <attack|defend>\n")
			return false
		}

		troopName := parts[1]
		target := strings.ToLower(parts[2])
		countStr := parts[3]
		mode := strings.ToLower(parts[4])

		num, err := strconv.Atoi(countStr)
		if err != nil || num > 2 {
			utils.SendMessage(p.Conn, "Invalid number for summon lane.\n")
			return false
		}

		troop, err := utils.LoadTroopSpecs(troopName)
		if err != nil {
			utils.SendMessage(p.Conn, "Unknown troop name.\n")
			return false
		}

		scaleStatsByLevel(&troop, p.Level)

		if p.Mana < troop.Mana {
			utils.SendMessage(p.Conn, fmt.Sprintf("Not enough mana (%d required, %d available)\n", troop.Mana, p.Mana))
			return false
		}

		var lane int
		if target == "guard" {
			lane = num - 1

			if mode == "attack" {
				if lane == 1 && !opponent.DestroyedTowers[0] {
					utils.SendMessage(p.Conn, "❌ You must destroy opponent's Guard Tower 1 before summoning to Guard Tower 2.\n")
					return false
				}
				if opponent.DestroyedTowers[lane] {
					utils.SendMessage(p.Conn, fmt.Sprintf("❌ Opponent's Guard Tower %d is destroyed. Cannot summon to attack.\n", num))
					return false
				}
			} else if mode == "defend" {
				if p.DestroyedTowers[lane] {
					utils.SendMessage(p.Conn, fmt.Sprintf("❌ Your Guard Tower %d is destroyed. Cannot summon to defend.\n", num))
					return false
				}
			} else {
				utils.SendMessage(p.Conn, "Invalid mode. Use 'attack' or 'defend'.\n")
				return false
			}
		} else if target == "king" {
			if num != 0 && num != 1 {
				utils.SendMessage(p.Conn, "❌ Invalid King number. Use 'king 0' (defend) or 'king 1' (attack)\n")
				return false
			}

			if mode == "attack" {
				if num == 1 {
					if !opponent.DestroyedTowers[0] || !opponent.DestroyedTowers[1] {
						utils.SendMessage(p.Conn, "❌ Opponent's Guard Towers are still standing. Cannot attack King.\n")
						return false
					}
				} else {
					utils.SendMessage(p.Conn, "❌ Invalid king attack number. Use 'king 1' to attack.\n")
					return false
				}
			} else if mode == "defend" {
				if num == 0 {
					if !p.DestroyedTowers[0] || !p.DestroyedTowers[1] {
						utils.SendMessage(p.Conn, "❌ You must lose both Guard Towers to defend King.\n")
						return false
					}
				} else {
					utils.SendMessage(p.Conn, "❌ Invalid king defend number. Use 'king 0' to defend.\n")
					return false
				}
			} else {
				utils.SendMessage(p.Conn, "Invalid mode. Use 'attack' or 'defend'.\n")
				return false
			}
			lane = 2
		} else {
			utils.SendMessage(p.Conn, "Invalid target. Use 'guard' or 'king'\n")
			return false
		}

		troopInstance := troop
		troopInstance.Owner = p.Username
		troopInstance.Lane = lane + 1
		troopInstance.TimeToTarget = 5 * time.Second
		troopInstance.Alive = true
		troopInstance.Mode = mode

		p.Troops[lane] = append(p.Troops[lane], &troopInstance)
		p.Mana -= troop.Mana

		broadcast(fmt.Sprintf("⚔️ %s summoned %s to %s %d (%s). Mana: %d\n", p.Username, troop.Name, target, num, mode, p.Mana))

		if troop.Special == "heal" {
			idx := utils.HealTower(playerTowers[playerIndex][:])
			broadcast(fmt.Sprintf("❤️ Queen healed tower %d by 300 HP!\n", idx+1))
		}
		return true
	default:
		utils.SendMessage(p.Conn, "Unknown command\n")
		return false
	}
}

func resolveLaneCombat(lane int) {
	p1 := players[0]
	p2 := players[1]
	troops1 := p1.Troops[lane]
	troops2 := p2.Troops[lane]
	target1 := playerTowers[1][lane]
	target2 := playerTowers[0][lane]

	if len(troops1) > 0 && len(troops2) > 0 {
		t1 := troops1[0]
		t2 := troops2[0]

		t1.HP -= t2.ATK
		t2.HP -= t1.ATK

		t1Dead := t1.HP <= 0
		t2Dead := t2.HP <= 0

		if t1Dead {
			p1.Troops[lane] = p1.Troops[lane][1:]
			broadcast(fmt.Sprintf("💀 %s's %s was defeated in lane %d", t1.Owner, t1.Name, lane+1))
		}
		if t2Dead {
			p2.Troops[lane] = p2.Troops[lane][1:]
			broadcast(fmt.Sprintf("💀 %s's %s was defeated in lane %d", t2.Owner, t2.Name, lane+1))
		}

		//Which troops win
		var winner *models.Troop
		var loser *models.Troop
		var winnerPlayer *Player
		var targetTower *models.Tower
		var defenderPlayer *Player

		if !t1Dead && t2Dead {
			winner = t1
			loser = t2
			winnerPlayer = p1
			targetTower = target1
			defenderPlayer = p2
		} else if !t2Dead && t1Dead {
			winner = t2
			loser = t1
			winnerPlayer = p2
			targetTower = target2
			defenderPlayer = p1
		} else {
			return // Same troops => die both
		}

		// Winner => lose HP
		winner.HP -= loser.ATK
		if winner.HP <= 0 {
			broadcast(fmt.Sprintf("💀 %s's %s was defeated after counter attack in lane %d", winner.Owner, winner.Name, lane+1))
			if winnerPlayer == p1 {
				p1.Troops[lane] = p1.Troops[lane][1:]
			} else {
				p2.Troops[lane] = p2.Troops[lane][1:]
			}
			return
		}

		// Win troops atk tower => tower and troop lose HP
		if winner.Mode == "attack" {
			applyDamageToTower(winner, targetTower, lane, winnerPlayer, defenderPlayer)
			applyTowerAttackOnTroop(targetTower, winner, winnerPlayer)
		}
		return
	}

	// Only player 1 sum troops
	if len(troops1) > 0 && len(troops2) == 0 {
		if troops1[0].Mode == "attack" {
			applyDamageToTower(troops1[0], target1, lane, p1, p2)
			applyTowerAttackOnTroop(target1, troops1[0], p1)
		}
		return
	}

	// Only player 2 sum troops
	if len(troops2) > 0 && len(troops1) == 0 {
		if troops2[0].Mode == "attack" {
			applyDamageToTower(troops2[0], target2, lane, p2, p1)
			applyTowerAttackOnTroop(target2, troops2[0], p2)
		}
		return
	}
}

func applyDamageToTower(t *models.Troop, target *models.Tower, lane int, attacker *Player, defender *Player) {
	//Destroy tower 1 before 2
	if lane == 1 && !defender.DestroyedTowers[0] {
		utils.SendMessage(attacker.Conn, "❌ You must destroy opponent's Guard Tower 1 before attacking Guard Tower 2.\n")
		return
	}

	dmg := t.ATK
	isCrit := false
	if rand.Float64() < target.CRIT {
		dmg = int(float64(dmg) * 1.2)
		isCrit = true
	}

	originalDEF := target.DEF
	originalHP := target.HP
	if target.DEF > 0 {
		if dmg >= target.DEF {
			dmg -= target.DEF
			target.DEF = 0
			target.HP -= dmg
		} else {
			target.DEF -= dmg
			dmg = 0
		}
	} else {
		target.HP -= dmg
	}
	broadcast(fmt.Sprintf("⚔️ %s's %s attacked enemy tower %d ➤ DEF: %d→%d | HP: %d→%d",
		t.Owner, t.Name, lane+1, originalDEF, target.DEF, originalHP, target.HP))
	if isCrit {
		broadcast(fmt.Sprintf("💥 Critical Hit by %s's %s on tower %d!", t.Owner, t.Name, lane+1))
	}

	if target.HP <= 0 {
		target.HP = 0
		defender.DestroyedTowers[lane] = true
		broadcast(fmt.Sprintf("🏰 Tower %d of opponent destroyed!", lane+1))

		if lane == 2 {
			broadcast(fmt.Sprintf("🎉 %s wins the game!", attacker.Username))
			utils.UpdateExp(attacker.Username, 30)
			utils.UpdateExp(defender.Username, 10)
			time.Sleep(2 * time.Second)
			for _, p := range players {
				utils.SendMessage(p.Conn, "🔌 Game over. Disconnecting...\n")
				p.Conn.Close()
			}
			os.Exit(0)
		}

		attacker.Troops[lane] = []*models.Troop{}
		defender.Troops[lane] = []*models.Troop{}
	}
}

func applyTowerAttackOnTroop(tower *models.Tower, troop *models.Troop, owner *Player) {
	damage := tower.ATK - troop.DEF
	if damage < 0 {
		damage = 0
	}
	originalHP := troop.HP
	troop.HP -= damage
	broadcast(fmt.Sprintf("🏰 Tower attacked %s's %s ➤ HP: %d→%d", troop.Owner, troop.Name, originalHP, troop.HP))

	if troop.HP <= 0 {
		troop.HP = 0
		broadcast(fmt.Sprintf("💀 %s's %s was defeated by tower attack", troop.Owner, troop.Name))
		for i, t := range owner.Troops[troop.Lane-1] {
			if t == troop {
				owner.Troops[troop.Lane-1] = append(owner.Troops[troop.Lane-1][:i], owner.Troops[troop.Lane-1][i+1:]...)
				break
			}
		}
	}
}

func scaleTowerStats(tower *models.Tower, level int) {
	if level <= 0 {
		return
	}
	scale := 1.0 + float64(level)*0.1
	tower.HP = int(float64(tower.HP) * scale)
	tower.ATK = int(float64(tower.ATK) * scale)
	tower.DEF = int(float64(tower.DEF) * scale)
}

func scaleStatsByLevel[T any](obj *T, level int) {
	scale := 1.0 + float64(level)*0.1
	switch v := any(obj).(type) {
	case *models.Troop:
		v.HP = int(float64(v.HP) * scale)
		v.ATK = int(float64(v.ATK) * scale)
		v.DEF = int(float64(v.DEF) * scale)
	case *models.Tower:
		v.HP = int(float64(v.HP) * scale)
		v.ATK = int(float64(v.ATK) * scale)
		v.DEF = int(float64(v.DEF) * scale)
	}
}

func broadcast(msg string) {
	for _, p := range players {
		utils.SendMessage(p.Conn, msg)
	}
}
