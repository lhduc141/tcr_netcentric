// server.go (updated with proper DEF logic like heal)

package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"path/filepath"
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
		sendMessage(conn, "🔐 Username: ")
		username := readFromConn(conn)
		sendMessage(conn, "🔐 Password: ")
		password := readFromConn(conn)

		if !authenticate(username, password) {
			sendMessage(conn, "Authentication failed.\n")
			conn.Close()
			continue
		}

		player := &Player{
			Conn:     conn,
			Username: username,
			Exp:      getExp(username),
			Level:    getLevel(username),
			Mana:     5,
		}
		mu.Lock()
		players = append(players, player)
		mu.Unlock()
		sendMessage(conn, fmt.Sprintf("✅ Welcome %s! Waiting for other player...\n", username))
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
	}
	currentTurn = 0
	commands = [2]string{"", ""}

	for {
		currentPlayer := players[currentTurn]
		sendMessage(currentPlayer.Conn, fmt.Sprintf("🔁 Your turn (Mana: %d): ", currentPlayer.Mana))
		input := readFromConn(currentPlayer.Conn)
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
				sendMessage(p.Conn, fmt.Sprintf("🔋 Mana = %d\n", p.Mana))
			}
			commands = [2]string{"", ""}
		}
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
		if !t1Dead && t2Dead {
			applyDamageToTower(t1, target1, lane, p1, p2)
		} else if !t2Dead && t1Dead {
			applyDamageToTower(t2, target2, lane, p2, p1)
		}
		return
	}
	if len(troops1) > 0 {
		applyDamageToTower(troops1[0], target1, lane, p1, p2)
		return
	}
	if len(troops2) > 0 {
		applyDamageToTower(troops2[0], target2, lane, p2, p1)
	}
}

func applyDamageToTower(t *models.Troop, target *models.Tower, lane int, attacker *Player, defender *Player) {
	dmg := t.ATK
	if rand.Float64() < target.CRIT {
		dmg = int(float64(dmg) * 1.2)
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
	if target.HP <= 0 {
		target.HP = 0
		attacker.DestroyedTowers[lane] = true
		defender.DestroyedTowers[lane] = true
		broadcast(fmt.Sprintf("🏰 Tower %d of opponent destroyed!", lane+1))

		if lane == 2 {
			broadcast(fmt.Sprintf("🎉 %s wins the game!", attacker.Username))
			updateExp(attacker.Username, 30)
			updateExp(defender.Username, 10)
			time.Sleep(2 * time.Second)
			for _, p := range players {
				sendMessage(p.Conn, "🔌 Game over. Disconnecting...\n")
				p.Conn.Close()
			}
			os.Exit(0)
		}

		attacker.Troops[lane] = []*models.Troop{}
		defender.Troops[lane] = []*models.Troop{}
	}

}

func handleCommand(playerIndex int, input string) bool {
	p := players[playerIndex]
	//opponentIndex := 1 - playerIndex
	parts := strings.Fields(input)
	if len(parts) == 0 {
		sendMessage(p.Conn, "Invalid command\n")
		return false
	}

	switch parts[0] {
	case "end":
		sendMessage(p.Conn, "🔚 You ended your turn.\n")
		return true

	case "summon":
		if len(parts) != 4 {
			sendMessage(p.Conn, "Usage: summon <TroopName> <guard|king> <Number>\n")
			return false
		}

		troopName := parts[1]
		target := strings.ToLower(parts[2])
		countStr := parts[3]

		num, err := strconv.Atoi(countStr)
		if err != nil || num > 2 {
			return false
		}

		troop, err := utils.LoadTroopSpecs(troopName)
		if err != nil {
			return false
		}

		if p.Mana < troop.Mana {
			sendMessage(p.Conn, fmt.Sprintf("Not enough mana (%d required, %d available)\n", troop.Mana, p.Mana))
			return false
		}

		var lane int
		if target == "guard" {
			lane = num - 1
			if p.DestroyedTowers[lane] {
				sendMessage(p.Conn, fmt.Sprintf("❌ Guard Tower %d is destroyed. Cannot summon here.\n", num))
				return false
			}
		} else if target == "king" {
			if num != 0 && num != 1 {
				sendMessage(p.Conn, "❌ Invalid King number. Use 'king 0' (defend) or 'king 1' (attack)\n")
				return false
			}

			if num == 1 {
				// tấn công vào King của đối thủ (chỉ khi đối thủ mất cả 2 guard)
				opponentIndex := 1 - playerIndex
				if !players[opponentIndex].DestroyedTowers[0] || !players[opponentIndex].DestroyedTowers[1] {
					sendMessage(p.Conn, "❌ Opponent's Guard Towers are still standing. Cannot attack King.\n")
					return false
				}
			} else if num == 0 {
				// phòng thủ King của chính mình (chỉ khi bản thân đã mất cả 2 guard)
				if !p.DestroyedTowers[0] || !p.DestroyedTowers[1] {
					sendMessage(p.Conn, "❌ You must lose both Guard Towers to defend King.\n")
					return false
				}
			}

			lane = 2 // King lane
		} else {
			sendMessage(p.Conn, "Invalid target. Use 'guard' or 'king'\n")
			return false
		}

		troopInstance := troop
		troopInstance.Owner = p.Username
		troopInstance.Lane = lane + 1
		troopInstance.TimeToTarget = 5 * time.Second
		troopInstance.Alive = true

		p.Troops[lane] = append(p.Troops[lane], &troopInstance)
		p.Mana -= troop.Mana

		broadcast(fmt.Sprintf("⚔️ %s summoned %s to %s %d. Mana: %d\n", p.Username, troop.Name, target, num, p.Mana))

		if troop.Special == "heal" {
			idx := utils.HealTower(playerTowers[playerIndex][:])
			broadcast(fmt.Sprintf("❤️ Queen healed tower %d by 300 HP!\n", idx+1))
		}
		return true
	default:
		sendMessage(p.Conn, "Unknown command\n")
		return false
	}
}

func authenticate(username, password string) bool {
	path := filepath.Join("data", "players.json")
	file, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var data map[string]map[string]interface{}
	err = json.Unmarshal(file, &data)
	if err != nil {
		return false
	}
	if user, ok := data[username]; ok {
		return user["password"] == password
	}
	return false
}

func getExp(username string) int {
	file, _ := os.ReadFile("player_data.json")
	var data map[string]map[string]interface{}
	json.Unmarshal(file, &data)
	if user, ok := data[username]; ok {
		return int(user["exp"].(float64))
	}
	return 0
}

func getLevel(username string) int {
	return getExp(username) / 100
}

func sendMessage(conn net.Conn, msg string) {
	conn.Write([]byte(msg))
}

func readFromConn(conn net.Conn) string {
	reader := bufio.NewReader(conn)
	input, err := reader.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			conn.Close()
		}
		return ""
	}
	return strings.TrimSpace(input)
}

func broadcast(msg string) {
	for _, p := range players {
		sendMessage(p.Conn, msg)
	}
}

func updateExp(username string, delta int) {
	path := filepath.Join("data", "players.json")
	file, err := os.ReadFile(path)
	if err != nil {
		log.Println("Cannot read player file:", err)
		return
	}
	var data map[string]map[string]interface{}
	json.Unmarshal(file, &data)
	if _, ok := data[username]; ok {
		old := int(data[username]["exp"].(float64))
		data[username]["exp"] = old + delta
	}
	updated, _ := json.MarshalIndent(data, "", "  ")
	os.WriteFile(path, updated, 0644)
}
