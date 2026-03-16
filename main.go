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

	elev_out_chan := make(chan elev_struct.Elevator, config.BUFFER_SIZE)
	clear_local_hall_orders_chan := make(chan bool, config.BUFFER_SIZE)
	completed_order_chan := make(chan elevio.ButtonEvent, config.BUFFER_SIZE)
	assigned_orders_chan := make(chan elevio.ButtonEvent, config.BUFFER_SIZE)

	network.NetworkInit(id)
	elevator.ElevatorInit(id, port, clear_local_hall_orders_chan, completed_order_chan, assigned_orders_chan, elev_out_chan)
	orders.OrdersInit(id, clear_local_hall_orders_chan, completed_order_chan, assigned_orders_chan, elev_out_chan)

	for {
		time.Sleep(100 * time.Millisecond)
	}
}
