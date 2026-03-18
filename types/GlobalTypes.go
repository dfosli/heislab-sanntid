package types

import (
	"heislab-sanntid/config"
	"heislab-sanntid/elevator/elev_struct"
)

const (
	NONE OrderState = iota
	NEW
	CONFIRMED
	ASSIGNED
	COMPLETED
)

type OrderState int
type Elevator = elev_struct.Elevator
type HallOrders = [config.N_FLOORS][config.N_BUTTONS - 1]OrderState

type AllHallOrders map[string]HallOrders
type AllCabOrders map[string][config.N_FLOORS]bool
type AllElevators map[string]Elevator
