package main

import (
    "heislab-sanntid/elevator/elevio" 
    "heislab-sanntid/config" 
    "heislab-sanntid/elevator/elev_struct" 
    "heislab-sanntid/elevator/elevator" 
)

const (
	N_FLOORS  int = config.N_FLOORS
	N_BUTTONS int = config.N_BUTTONS
	DOOR_OPEN_TIME = config.DOOR_OPEN_TIME
	STUCK_TIME = config.STUCK_TIME
)

func main() {

    elevio.Init("localhost:15657", N_FLOORS)
    
    // Driver
    drv_buttons := make(chan elevio.ButtonEvent)
    drv_floors  := make(chan int)
    drv_obstr   := make(chan bool)

    // Elevator
    elev_out := make(chan elev_struct.Elevator)
    clear_local_hall_orders := make(chan bool)
	clear_order := make(chan elevio.ButtonEvent)
	assigned_orders := make(chan elevio.ButtonEvent)
    
    go elevio.PollButtons(drv_buttons)
    go elevio.PollFloorSensor(drv_floors)
    go elevio.PollObstructionSwitch(drv_obstr)

    go elevator.RunElevator(0, drv_buttons, drv_floors, drv_obstr, clear_local_hall_orders, clear_order, assigned_orders, elev_out)

    for {}   
}
