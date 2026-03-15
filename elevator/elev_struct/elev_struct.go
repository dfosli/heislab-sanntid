package elev_struct

import (
	elevio "Driver-go"
	"heislab-sanntid/config"
)

const (
	N_FLOORS  int = config.N_FLOORS
	N_BUTTONS int = config.N_BUTTONS
)

type State int

const (
	Idle State = iota
	Moving
	DoorOpen
)

type Requests [N_FLOORS][N_BUTTONS]bool

type Elevator struct {
	State      State
	Floor      int
	Dir        elevio.MotorDirection
	Requests   Requests
	ID         string
	Stuck      bool
	Obstructed bool
}

type DirStatePair struct {
	Dir   elevio.MotorDirection
	State State
}

type LightEvent struct {
	Floor  int
	Button elevio.ButtonType
	On     bool
}

func ElevatorInit(id string) Elevator {
	return Elevator{
		State:      Idle,
		Floor:      -1,
		Dir:        elevio.MD_Stop,
		ID:         id,
		Stuck:      false,
		Obstructed: false,
	}
}

func ClearLocalHallOrders(elev Elevator) Elevator {
	for f := 0; f < N_FLOORS; f++ {
		for btn := 0; btn < N_BUTTONS - 1; btn++ {
			elev.Requests[f][btn] = false
		}
	}
	return elev
}

func GetCabOrders(elev Elevator) [N_FLOORS]bool {
	var cabOrders [N_FLOORS]bool

	for f := 0; f < N_FLOORS; f++ {
		cabOrders[f] = elev.Requests[f][elevio.BT_Cab]
	}
	return cabOrders
}

func SetCabLights(elev Elevator) {
	for f := 0; f < N_FLOORS; f++ {
		elevio.SetButtonLamp(elevio.BT_Cab, f, elev.Requests[f][elevio.BT_Cab])
	}
}
