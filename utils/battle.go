package utils

import (
	"math/rand"
	"tcr_netcentric/models"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func CalculateDamage(attacker models.Troop, defenderDEF int, critChance float64) int {
	damage := attacker.ATK
	if rand.Float64() < critChance {
		damage = int(float64(attacker.ATK) * 1.2)
	}
	result := damage - defenderDEF
	if result < 0 {
		return 0
	}
	return result
}

func HealTower(towers []*models.Tower) int {
	minHP := 999999
	index := -1
	for i, t := range towers {
		if t.HP < minHP {
			minHP = t.HP
			index = i
		}
	}
	if index != -1 {
		towers[index].HP += 300
	}
	return index
}
