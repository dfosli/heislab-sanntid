package elevator

import (
	"heislab-sanntid/config"
	"heislab-sanntid/elevator/elev_struct"
	"heislab-sanntid/elevator/elevio"
	"heislab-sanntid/elevator/fsm"
	"heislab-sanntid/elevator/requests"
)

const (
	N_FLOORS  int = config.N_FLOORS
	N_BUTTONS int = config.N_BUTTONS
	DOOR_OPEN_TIME = config.DOOR_OPEN_TIME
	STUCK_TIME = config.STUCK_TIME
)

func RunElevator(
	
) {}