package utils

import (
	"math"
	"math/rand"
	"tcr_netcentric/models"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func HealTower(towers []*models.Tower) int {
	minHP := math.MaxInt32
	target := -1

	for i, t := range towers {
		if t.HP > 0 && t.HP < minHP {
			minHP = t.HP
			target = i
		}
	}

	if target != -1 {
		towers[target].HP += 300
	}

	return target
}
