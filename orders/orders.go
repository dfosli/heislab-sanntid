package orders

import (
	"heislab-sanntid/config"
)

type OrderState int

const (
	NONE OrderState = iota
	NEW
	CONFIRMED
	ASSIGNED
	COMPLETED
)

type HallOrders [config.N_FLOORS][config.N_BUTTONS - 1]OrderState


func InitHallOrders() HallOrders{
	var hallOrders HallOrders
	for floor := 0; floor < config.N_FLOORS; floor++ {
		for btn := 0; btn < config.N_BUTTONS-1; btn++ { //må ikke disse for loopene også minkes med 1, for å fjerne cab
			hallOrders[floor][btn] = NONE
		}
	}
	return hallOrders
}

func InitHallOrdersAllElevators() [config.N_ELEVATORS]HallOrders {
	var allHallOrders [config.N_ELEVATORS]HallOrders
	for elev := 0; elev < config.N_ELEVATORS; elev++ {
		allHallOrders[elev] = InitHallOrders()
	}
	return allHallOrders
}