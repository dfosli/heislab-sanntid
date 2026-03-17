package main

import (
	elevio "Driver-go"
	"flag"
	"fmt"
	"heislab-sanntid/config"
	"heislab-sanntid/elevator/elev_struct"
	"heislab-sanntid/elevator/elevator"
	network "heislab-sanntid/network"
	"heislab-sanntid/network/network/localip"
	"heislab-sanntid/orders"
	"os"
)

func parseFlags() (string, string) {
	var id string
	var port string

	flag.StringVar(&id, "id", "", "id of this peer")
	flag.StringVar(&port, "port", "", "port of this elevator")
	flag.Parse()

	if id == "" {
		localIP, err := localip.LocalIP()
		if err != nil {
			fmt.Println("Warning:", err)
			localIP = "DISCONNECTED"
		}
		id = fmt.Sprintf("peer-%s-%d", localIP, os.Getpid())
	}

	if port == "" {
		fmt.Println("Error: No port provided. Exiting.")
		os.Exit(1)
	}

	return id, port
}

func initChannels() (
	chan elev_struct.Elevator,
	chan [config.N_FLOORS][config.N_BUTTONS - 1]bool,
	chan [config.N_FLOORS]bool,
	chan elevio.ButtonEvent,
) {
	return make(chan elev_struct.Elevator, config.BUFFER_SIZE),
		make(chan [config.N_FLOORS][config.N_BUTTONS - 1]bool, config.BUFFER_SIZE),
		make(chan [config.N_FLOORS]bool, config.BUFFER_SIZE),
		make(chan elevio.ButtonEvent, config.BUFFER_SIZE)
}

func main() {

	id, port := parseFlags()

	elevOutCh, reassignedOrdersCh, recoveredCabOrdersCh, completedOrderCh := initChannels()

	network.NetworkInit(id)

	elevator.ElevatorInit(
		id,
		port,
		reassignedOrdersCh,
		recoveredCabOrdersCh,
		completedOrderCh,
		elevOutCh)

	orders.OrdersInit(id,
		reassignedOrdersCh,
		recoveredCabOrdersCh,
		completedOrderCh,
		elevOutCh)

	select {}
}
