package main

import (
	"heislab-sanntid/config"
	"heislab-sanntid/elevator/elev_struct"
	"heislab-sanntid/elevator/elevator"
	"heislab-sanntid/elevator/elevio"
    "time"
)

func main() {

	elevio.Init("localhost:15657", config.N_FLOORS)
	id := 0 //TODO: få id fra -id flag i kommando elns

	// Driver
	drv_buttons := make(chan elevio.ButtonEvent)
	drv_floors := make(chan int)
	drv_obstr := make(chan bool)

	// Elevator
	elev_out := make(chan elev_struct.Elevator)
	clear_local_hall_orders := make(chan bool, config.BUFFER_SIZE)
	clear_order := make(chan elevio.ButtonEvent, config.BUFFER_SIZE)
	assigned_orders := make(chan elevio.ButtonEvent, config.BUFFER_SIZE)

	go elevio.PollButtons(drv_buttons)
	go elevio.PollFloorSensor(drv_floors)
	go elevio.PollObstructionSwitch(drv_obstr)

	go elevator.RunElevator(id, drv_buttons, drv_floors, drv_obstr, clear_local_hall_orders, clear_order, assigned_orders, elev_out)

	// KUN FOR Å SIMULERE EN ENKELT HEIS
	go func() { //black hole for channels, channels blocker programmet hvis ingen leser fra dem
		for {
			select {
			case e :=<-elev_out:
				for f := 0; f < config.N_FLOORS; f++ { //setter alle lys her, kun for simulatoren
					for btn := 0; btn < config.N_BUTTONS; btn++ {
						elevio.SetButtonLamp(elevio.ButtonType(btn), f, e.Requests[f][btn])
						time.Sleep(10 * time.Millisecond)
					}
				}
			case <-clear_order:
			case <-clear_local_hall_orders:
			}
		}
	}()

	for {
	}
}
