package orders

import (
	elevio "Driver-go"
	network "Network-go"
	"heislab-sanntid/config"
	"heislab-sanntid/elevator/elev_struct"
	"sync"
	"time"
)

type OrderState int

const (
	NONE OrderState = iota
	NEW
	CONFIRMED
	ASSIGNED
	COMPLETED
)

type HallOrders [config.N_FLOORS][config.N_BUTTONS - 1]OrderState

type HallOrdersAllElevators map[string]HallOrders

type AllElevatorStates map[string]elev_struct.Elevator

type ElevstateHallorderPair struct { //TODO endre navn
	elevatorState elev_struct.Elevator
	hallOrders    HallOrders
}

func initHallOrders() HallOrders {
	var hallOrders HallOrders
	for floor := 0; floor < config.N_FLOORS; floor++ {
		for btn := 0; btn < config.N_BUTTONS-1; btn++ {
			hallOrders[floor][btn] = NONE
		}
	}
	return hallOrders
}

func initHallOrdersAllElevators(id string) HallOrdersAllElevators {
	allHallOrders := make(HallOrdersAllElevators)
	allHallOrders[id] = initHallOrders()
	return allHallOrders
}

func initAllElevatorStates(id string) AllElevatorStates {
	allElevatorStates := make(AllElevatorStates)
	allElevatorStates[id] = elev_struct.ElevatorInit(id)
	return allElevatorStates
}

func confirmHallOrders(
	order_confirmed_chan chan<- elevio.ButtonEvent,
	hall_light_chan chan<- elev_struct.LightEvent,
	allHallOrders *HallOrdersAllElevators,
	availableElevators *map[string]bool,
	dataMutex *sync.RWMutex) {

	for {
		time.Sleep(10 * time.Millisecond)

		dataMutex.RLock()

		for floor := 0; floor < config.N_FLOORS; floor++ {
			for btn := 0; btn < config.N_BUTTONS-1; btn++ {

				shouldConfirm := true

				for id, isAvailable := range *availableElevators {
					if isAvailable {
						if orders, ok := (*allHallOrders)[id]; ok {
							if orders[floor][btn] != NEW {
								shouldConfirm = false
								break
							}
						} else {
							shouldConfirm = false
							break
						}
					}
				}

				if shouldConfirm {
					order_confirmed_chan <- elevio.ButtonEvent{Floor: floor, Button: elevio.ButtonType(btn)}
				}
			}
		}
		dataMutex.RUnlock()
	}
}

func resetHallOrders(
	order_reset_chan chan<- elevio.ButtonEvent,
	hall_light_chan chan<- elev_struct.LightEvent,
	allHallOrders *HallOrdersAllElevators,
	availableElevators *map[string]bool,
	dataMutex *sync.RWMutex) {

	for {
		time.Sleep(10 * time.Millisecond)

		dataMutex.RLock()

		for floor := 0; floor < config.N_FLOORS; floor++ {
			for btn := 0; btn < config.N_BUTTONS-1; btn++ {

				shouldReset := true

				for id, isAvailable := range *availableElevators {
					if isAvailable {
						if orders, ok := (*allHallOrders)[id]; ok {
							if orders[floor][btn] != COMPLETED {
								shouldReset = false
								break
							}
						} else {
							shouldReset = false
							break
						}
					}
				}

				if shouldReset {
					order_reset_chan <- elevio.ButtonEvent{Floor: floor, Button: elevio.ButtonType(btn)}
				}
			}
		}
		dataMutex.RUnlock()
	}
}

func unnasignHallOrders(id string, allHallOrders *HallOrdersAllElevators, dataMutex *sync.RWMutex) {
	dataMutex.Lock()
	if orders, ok := (*allHallOrders)[id]; ok {
		for floor := 0; floor < config.N_FLOORS; floor++ {
			for btn := 0; btn < config.N_BUTTONS-1; btn++ {
				if orders[floor][btn] == ASSIGNED {
					orders[floor][btn] = CONFIRMED
				}
			}
		}
		(*allHallOrders)[id] = orders
	}
	dataMutex.Unlock()
}

func RunOrderManager(
	id string,
	local_elevator_chan <-chan elev_struct.Elevator,
	assign_order_chan chan<- elevio.ButtonEvent,
	completed_order_chan <-chan elevio.ButtonEvent,
	clear_local_hall_orders_chan chan<- bool,
	// networkTx chan<- ElevstateHallorderPair,
	// networkRx <-chan ElevstateHallorderPair,
	//peer_update_chan <-chan peers.PeerUpdate
) {
	// Ikke accsess direkte til variabler fra network. DB

	var allHallOrders HallOrdersAllElevators = initHallOrdersAllElevators(id) //bruk mutex rundt denne
	//var allElevatorStates = initAllElevatorStates(id)
	var availableElevators = make(map[string]bool) //bruk mutex rundt denne
	availableElevators[id] = true
	var dataMutex sync.RWMutex

	order_confirmed_chan := make(chan elevio.ButtonEvent, config.BUFFER_SIZE)
	order_reset_chan := make(chan elevio.ButtonEvent, config.BUFFER_SIZE)
	hall_light_chan := make(chan elev_struct.LightEvent, config.BUFFER_SIZE)

	go confirmHallOrders(order_confirmed_chan, hall_light_chan, &allHallOrders, &availableElevators, &dataMutex)
	go resetHallOrders(order_reset_chan, hall_light_chan, &allHallOrders, &availableElevators, &dataMutex)

	for {
		select {
		// Unsure if peers returns IDs. Will be tested. DB
		case peerUpdate := <-network.Peers():
			dataMutex.Lock()
			for _, peer := range peerUpdate.Peers {
				if peer != id {
					availableElevators[peer] = true
				}
			}

			// Needed? DB
			availableElevators[peerUpdate.New] = true

			for _, lostPeer := range peerUpdate.Lost {
				delete(availableElevators, lostPeer)
				//TODO: unassigne alle ordre slik at de kan bli assigned til andre heiser. Sette alle dens ordre til NONE (med init funksjon)
			}
			dataMutex.Unlock()

		//case localElevator := <-local_elevator_chan:
		//TODO: er stuck? har ny ordre?
		//network.NetworkSend()

		// case remoteElevator := <-networkRx:
		//TODO: er stuck? har ny ordre?
		//oppdatere allElevatorStates

		case newCompletedOrder := <-completed_order_chan:
			dataMutex.Lock()
			if orders, ok := allHallOrders[id]; ok {
				orders[newCompletedOrder.Floor][newCompletedOrder.Button] = COMPLETED
				allHallOrders[id] = orders
				//TODO: network send
			}
			dataMutex.Unlock()

		case orderToConfirm := <-order_confirmed_chan:
			dataMutex.Lock()
			for id, isAvailable := range availableElevators {
				if isAvailable {
					if orders, ok := allHallOrders[id]; ok {
						orders[orderToConfirm.Floor][orderToConfirm.Button] = CONFIRMED
						allHallOrders[id] = orders
					}
				}
			}
			hall_light_chan <- elev_struct.LightEvent{Floor: orderToConfirm.Floor, Button: elevio.ButtonType(orderToConfirm.Button), On: true}
			dataMutex.Unlock()

			//TODO: kjør distribution
			//sett ordre til assigned
			//hvis assigned til oss, send til elevator

		case orderToReset := <-order_reset_chan:
			dataMutex.Lock()
			for id, isAvailable := range availableElevators {
				if isAvailable {
					if orders, ok := allHallOrders[id]; ok {
						orders[orderToReset.Floor][orderToReset.Button] = NONE
						allHallOrders[id] = orders
					}
				}
			}
			hall_light_chan <- elev_struct.LightEvent{Floor: orderToReset.Floor, Button: elevio.ButtonType(orderToReset.Button), On: false}
			dataMutex.Unlock()

		case hallLightEvent := <-hall_light_chan:
			elevio.SetButtonLamp(hallLightEvent.Button, hallLightEvent.Floor, hallLightEvent.On)
		}
	}
}
