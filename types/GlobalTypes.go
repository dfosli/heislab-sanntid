package types

import (
	"heislab-sanntid/config"
	"heislab-sanntid/elevator/elev_struct"
)

type OrderState int
type Elevator = elev_struct.Elevator

type AllHallOrders [config.N_FLOORS][config.N_BUTTONS - 1]OrderState
type AllCabOrders map[string][config.N_FLOORS]bool
type AllElevators map[string]Elevator

const (
	NONE OrderState = iota
	NEW
	CONFIRMED
	ASSIGNED
	COMPLETED
)
