package main

import (
	"flag"
	"fmt"
	"heislab-sanntid/elevator/elevator"
	network "heislab-sanntid/network"
	"heislab-sanntid/network/network/localip"
	"os"
)

func main() {

	var id string
	flag.StringVar(&id, "id", "", "id of this peer")
	flag.Parse()

	if id == "" {
		localIP, err := localip.LocalIP()
		if err != nil {
			fmt.Println(err)
			localIP = "DISCONNECTED"
		}
		id = fmt.Sprintf("peer-%s-%d", localIP, os.Getpid())
	}

	network.NetworkInit(id)
	elevator.ElevatorInit(id)

	// KUN FOR Å SIMULERE EN ENKELT HEIS
	// go func() { //black hole for channels, channels blocker programmet hvis ingen leser fra dem
	// 	for {
	// 		select {
	// 		case e := <-elev_out:
	// 			for f := 0; f < config.N_FLOORS; f++ { //setter alle lys her, kun for simulatoren
	// 				for btn := 0; btn < config.N_BUTTONS; btn++ {
	// 					elevio.SetButtonLamp(elevio.ButtonType(btn), f, e.Requests[f][btn])
	// 					time.Sleep(10 * time.Millisecond)
	// 				}
	// 			}
	// 		case <-clear_order:
	// 		case <-clear_local_hall_orders:
	// 		}
	// 	}
	// }()

	for {
	}
}
