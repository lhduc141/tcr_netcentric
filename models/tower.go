package models

type Tower struct {
	Name     string  `json:"Name"`
	HP       int     `json:"HP"`
	ATK      int     `json:"ATK"`
	DEF      int     `json:"DEF"`
	CRIT     float64 `json:"CRIT"`
	EXPValue int     `json:"EXPValue"`
}
