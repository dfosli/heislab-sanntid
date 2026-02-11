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

const (
	Idle State = iota
	Moving
	DoorOpen
)

type Orders [N_FLOORS][N_BUTTONS]bool

type Elevator struct {
	State  State
	Floor  int
	Dir    elevio.MotorDirection
	Orders Orders
	ID     int
}

func ElevatorInit(id int) Elevator {
	return Elevator{
		State: Idle,
		Floor: -1,
		Dir:   elevio.MD_Stop,
		ID:    id,
	}
}

type DirStatePair struct {
	Dir   elevio.MotorDirection
	State State
}
