package utils

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func Authenticate(username, password string) bool {
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
