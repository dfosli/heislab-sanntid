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
	"time"
)

func main() {

	var id string
	flag.StringVar(&id, "id", "", "id of this peer")
	var port string
	flag.StringVar(&port, "port", "", "port of this elevator")
	flag.Parse()

	if id == "" {
		localIP, err := localip.LocalIP()
		if err != nil {
			fmt.Println(err)
			localIP = "DISCONNECTED"
		}
		id = fmt.Sprintf("peer-%s-%d", localIP, os.Getpid())
	}

	elevOutChan := make(chan elev_struct.Elevator, config.BUFFER_SIZE)
	clearLocalHallOrdersChan := make(chan bool, config.BUFFER_SIZE)
	completedOrderChan := make(chan elevio.ButtonEvent, config.BUFFER_SIZE)
	assignedOrdersChan := make(chan elevio.ButtonEvent, config.BUFFER_SIZE)

	error := network.NetworkInit(id)
	if error != nil {
		fmt.Println(error)
		return
	}

	elevator.ElevatorInit(id, port, clearLocalHallOrdersChan, completedOrderChan, assignedOrdersChan, elevOutChan)
	orders.OrdersInit(id, clearLocalHallOrdersChan, completedOrderChan, assignedOrdersChan, elevOutChan)

	for {
		time.Sleep(100 * time.Millisecond)
	}
}
