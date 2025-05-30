package models

import "time"

type Troop struct {
	Name         string
	HP           int
	ATK          int
	DEF          int
	Mana         int
	EXP          int
	Special      string
	Lane         int
	TimeToTarget time.Duration
	Owner        string
	Alive        bool

	Mode string
}
