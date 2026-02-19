package elev_struct

import (
	"heislab-sanntid/config"
	"heislab-sanntid/elevator/elevio"
)

const (
	N_FLOORS  int = config.N_FLOORS
	N_BUTTONS int = config.N_BUTTONS
)

type State int
type Stuck bool

const (
	Idle State = iota
	Moving
	DoorOpen
)

type Requests [N_FLOORS][N_BUTTONS]bool

type Elevator struct {
	State    State
	Floor    int
	Dir      elevio.MotorDirection
	Requests Requests
	ID       int
	Stuck    Stuck
}

type DirStatePair struct {
	Dir   elevio.MotorDirection
	State State
}

func ElevatorInit(id int) Elevator {
	return Elevator{
		State: Idle,
		Floor: -1,
		Dir:   elevio.MD_Stop,
		ID:    id,
		Stuck: false,
	}
}

func ClearHallOrders(e Elevator) Elevator {
	localElevator := e

	for f := 0; f < N_FLOORS; f++ {
		for btn := 0; btn < N_BUTTONS; btn++ {
			e.Requests[f][btn] = false
		}
	}
	return localElevator
}

func GetCabOrders(e Elevator) [N_FLOORS]bool {
	var cabOrders [N_FLOORS]bool

	for f := 0; f < N_FLOORS; f++ {
		cabOrders[f] = e.Requests[f][elevio.BT_Cab]
	}
	return cabOrders
}

func SetCabLights(e Elevator) {
	for f := 0; f < N_FLOORS; f++ {
		elevio.SetButtonLamp(elevio.BT_Cab, f, e.Requests[f][elevio.BT_Cab])
	}
}

func StateToString(state State) string {
	switch state {
	case Idle:
		return "idle"
	case DoorOpen:
		return "doorOpen"
	case Moving:
		return "moving"
	}
	return "INVALID STATE"
}

func MotorDirectionToString(dirn elevio.MotorDirection) string {
	switch dirn {
	case elevio.MD_Up:
		return "up"
	case elevio.MD_Down:
		return "down"
	case elevio.MD_Stop:
		return "stop"
	}
	return "INVALID DIRECTION"
}
