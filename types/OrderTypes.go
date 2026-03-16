package types

import (
	"heislab-sanntid/config"
)

type OrderState int
type HallOrders [config.N_FLOORS][config.N_BUTTONS - 1]OrderState
type CabOrders map[string][config.N_FLOORS]bool
const (
	NONE OrderState = iota
	NEW
	CONFIRMED
	ASSIGNED
	COMPLETED
)

type HallOrdersAllElevators map[string]HallOrders
