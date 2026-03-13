package main

import (
	"flag"
	"fmt"
	elevio "Driver-go"
	"heislab-sanntid/config"
	"heislab-sanntid/elevator/elev_struct"
	"heislab-sanntid/elevator/elevator"
	network "heislab-sanntid/network"
	"heislab-sanntid/network/network/localip"
	"heislab-sanntid/orders"
	"os"
)

func main() {

	var id string
	flag.StringVar(&id, "id", "", "id of this peer")
	flag.Parse()

	// ... or alternatively, we can use the local IP address.
	// (But since we can run multiple programs on the same PC, we also append the
	//  process ID)
	if id == "" {
		localIP, err := localip.LocalIP()
		if err != nil {
			fmt.Println(err)
			localIP = "DISCONNECTED"
		}
		id = fmt.Sprintf("peer-%s-%d", localIP, os.Getpid())
	}

	elev_out_chan := make(chan elev_struct.Elevator)
	clear_local_hall_orders_chan := make(chan bool, config.BUFFER_SIZE)
	completed_order_chan := make(chan elevio.ButtonEvent, config.BUFFER_SIZE)
	assigned_orders_chan := make(chan elevio.ButtonEvent, config.BUFFER_SIZE)

	network.NetworkInit(id)
	elevator.ElevatorInit(id, clear_local_hall_orders_chan, completed_order_chan, assigned_orders_chan, elev_out_chan)
	orders.OrdersInit(id, clear_local_hall_orders_chan, completed_order_chan, assigned_orders_chan, elev_out_chan)

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
