package types

import (
	"heislab-sanntid/config"
	"heislab-sanntid/elevator/elev_struct"
)

type OrderState int
type HallOrders [config.N_FLOORS][config.N_BUTTONS - 1]OrderState

type ElevstateHallorderPair struct {
	ElevatorState elev_struct.Elevator
	HallOrders    HallOrders
}

const (
	NONE OrderState = iota
	NEW
	CONFIRMED
	ASSIGNED
	COMPLETED
)

type HallOrdersAllElevators [config.N_ELEVATORS]HallOrders

type AllElevatorStates [config.N_ELEVATORS]elev_struct.Elevator
