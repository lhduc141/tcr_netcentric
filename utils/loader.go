package utils

import (
	"encoding/json"
	"os"
	"tcr_netcentric/models"
)

func LoadTroopSpecs(name string) (models.Troop, error) {
	file, err := os.ReadFile("data/troop_specs.json")
	if err != nil {
		return models.Troop{}, err
	}

	var troops map[string]models.Troop
	if err := json.Unmarshal(file, &troops); err != nil {
		return models.Troop{}, err
	}

	troop, ok := troops[name]
	if !ok {
		return models.Troop{}, nil
	}
	return troop, nil
}

// LoadTowerSpecs loads all tower specs from tower_specs.json
func LoadTowerSpecs() (map[string]models.Tower, error) {
	file, err := os.ReadFile("data/tower_specs.json")
	if err != nil {
		return nil, err
	}

	var towers map[string]models.Tower
	if err := json.Unmarshal(file, &towers); err != nil {
		return nil, err
	}

	return towers, nil
}
