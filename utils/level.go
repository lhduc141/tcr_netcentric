package utils

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

func GetExp(username string) int {
	file, _ := os.ReadFile("player_data.json")
	var data map[string]map[string]interface{}
	json.Unmarshal(file, &data)
	if user, ok := data[username]; ok {
		return int(user["exp"].(float64))
	}
	return 0
}

func GetLevel(exp int) int {
	required := 100.0
	level := 0
	for float64(exp) >= required {
		exp -= int(required)
		required *= 1.1
		level++
	}
	return level
}

func UpdateExp(username string, delta int) {
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
