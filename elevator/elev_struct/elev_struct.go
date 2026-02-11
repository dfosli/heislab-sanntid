package elevstruct

import (
	"heislab-sanntid/elevator/elevio"
	"heislab-sanntid/config"
)

const (
	N_FLOORS  int = config.N_FLOORS
	N_BUTTONS 	  = config.N_BUTTONS
)

type State int

const (
	Idle State = iota
	Moving
	DoorOpen
)

