package orders

import (
	elevio "Driver-go"
	"heislab-sanntid/config"
	"heislab-sanntid/elevator/elev_struct"
	network "heislab-sanntid/network"
	types "heislab-sanntid/types"
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

func toNetworkHallOrders(hallOrders HallOrders) types.HallOrders {
	var networkHallOrders types.HallOrders
	for floor := 0; floor < config.N_FLOORS; floor++ {
		for btn := 0; btn < config.N_BUTTONS-1; btn++ {
			networkHallOrders[floor][btn] = types.OrderState(hallOrders[floor][btn])
		}
	}
	return networkHallOrders
}

func fromNetworkHallOrders(networkHallOrders types.HallOrders) HallOrders {
	var hallOrders HallOrders
	for floor := 0; floor < config.N_FLOORS; floor++ {
		for btn := 0; btn < config.N_BUTTONS-1; btn++ {
			hallOrders[floor][btn] = OrderState(networkHallOrders[floor][btn])
		}
	}
	return hallOrders
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

func unassignHallOrders(id string, allHallOrders HallOrdersAllElevators) HallOrders {
	if orders, ok := allHallOrders[id]; ok {
		for floor := 0; floor < config.N_FLOORS; floor++ {
			for btn := 0; btn < config.N_BUTTONS-1; btn++ {
				if orders[floor][btn] == ASSIGNED {
					orders[floor][btn] = CONFIRMED
				}
			}
		}
	}
	return allHallOrders[id]
}

func setOrdersToAssigned(assignedOrders [][]bool, hallOrders HallOrders) HallOrders {
	for floor := 0; floor < config.N_FLOORS; floor++ {
		for btn := 0; btn < config.N_BUTTONS-1; btn++ {
			if assignedOrders[floor][btn] {
					hallOrders[floor][btn] = ASSIGNED
				}
			}
		}
	return hallOrders
}

func lostPeerReassignOrders(lost_id string, allHallOrders HallOrdersAllElevators, availableElevators map[string]bool) HallOrdersAllElevators {
	for id, isAvailable := range availableElevators {
		if isAvailable && id != lost_id {
			allHallOrders[id] = unassignHallOrders(id, allHallOrders)
		}
	}
	allHallOrders[lost_id] = initHallOrders()

	return allHallOrders
}

func RunOrderManager(
	id string,
	localElevatorChan <-chan elev_struct.Elevator,
	assignOrderChan chan<- elevio.ButtonEvent,
	completedOrderChan <-chan elevio.ButtonEvent,
	clearLocalHallOrdersChan chan<- bool,
	hallLightChan chan elev_struct.LightEvent,
	orderConfirmedChan <-chan elevio.ButtonEvent,
	orderResetChan <-chan elevio.ButtonEvent,
	allHallOrders HallOrdersAllElevators,
	allElevatorStates AllElevatorStates,
	availableElevators map[string]bool,
	dataMutex *sync.RWMutex) {

	for {
		select {
		// Unsure if peers returns IDs. Will be tested. DB
		case peerUpdate := <-network.Peers():
			dataMutex.Lock()
			for _, peer := range peerUpdate.Peers {
				if peer != id {
					availableElevators[peer] = true
					// unassigne ordre her slik at de blir fordelt på de tilgjengelige?
				}
			}

			// Needed? DB
			// Could leed to fatal problems. DB
			availableElevators[peerUpdate.New] = true

			for _, lostPeer := range peerUpdate.Lost {
				availableElevators[lostPeer] = false
				allHallOrders = lostPeerReassignOrders(lostPeer, allHallOrders, availableElevators)
				clearLocalHallOrdersChan <- true
			}
			dataMutex.Unlock()

		case localElevator := <-localElevatorChan:
			allElevatorStates[id] = localElevator

			if localElevator.Stuck && availableElevators[id] {
				dataMutex.Lock()
				availableElevators[id] = false
				allHallOrders = lostPeerReassignOrders(id, allHallOrders, availableElevators)
				dataMutex.Unlock()
				clearLocalHallOrdersChan <- true
			} else if !localElevator.Stuck && !availableElevators[id] {
				dataMutex.Lock()
				availableElevators[id] = true
				dataMutex.Unlock()
			}

			//network.NetworkSend()
			
		case remoteElevator := <-network.NetworkRxChan():
			newHallOrder := UpdateLocalHallOrdersIfPossible(allHallOrders[id], fromNetworkHallOrders(remoteElevator.HallOrders))//this function is added now as long as the HallORders stuff is not working
			
			dataMutex.Lock()
			allHallOrders[id] = newHallOrder
			dataMutex.Unlock()


		case newCompletedOrder := <-completedOrderChan:
			dataMutex.Lock()
			if orders, ok := allHallOrders[id]; ok {
				orders[newCompletedOrder.Floor][newCompletedOrder.Button] = COMPLETED
				allHallOrders[id] = orders
				//network.NetworkSend()
			}
			dataMutex.Unlock()

		case orderToConfirm := <-orderConfirmedChan:
			dataMutex.Lock()
			for id, isAvailable := range availableElevators {
				if isAvailable {
					if orders, ok := allHallOrders[id]; ok {
						orders[orderToConfirm.Floor][orderToConfirm.Button] = CONFIRMED
						allHallOrders[id] = orders
					}
				}
			}
			hallLightChan <- elev_struct.LightEvent{Floor: orderToConfirm.Floor, Button: elevio.ButtonType(orderToConfirm.Button), On: true}

			hallOrdersForId, _ := ReassignOrders(id, allHallOrders[id], availableElevators, allElevatorStates)
			allHallOrders[id] = setOrdersToAssigned(hallOrdersForId, allHallOrders[id])
			dataMutex.Unlock()

			for floor := 0; floor < config.N_FLOORS; floor++ {
				for btn := 0; btn < config.N_BUTTONS-1; btn++ {
					if hallOrdersForId[floor][btn] {
						assignOrderChan <- elevio.ButtonEvent{Floor: floor, Button: elevio.ButtonType(btn)}
					}
				}
			}

			//network.NetworkSend()

		case orderToReset := <-orderResetChan:
			dataMutex.Lock()
			for id, isAvailable := range availableElevators {
				if isAvailable {
					if orders, ok := allHallOrders[id]; ok {
						orders[orderToReset.Floor][orderToReset.Button] = NONE
						allHallOrders[id] = orders
					}
				}
			}
			hallLightChan <- elev_struct.LightEvent{Floor: orderToReset.Floor, Button: elevio.ButtonType(orderToReset.Button), On: false}
			dataMutex.Unlock()

		case hallLightEvent := <-hallLightChan:
			elevio.SetButtonLamp(hallLightEvent.Button, hallLightEvent.Floor, hallLightEvent.On)

		default:
			//network.NetworkSend()
		}
	}
}

func OrdersInit(id string,
	clear_local_hall_orders_chan chan<- bool,
	completed_order_chan <-chan elevio.ButtonEvent,
	assign_order_chan chan<- elevio.ButtonEvent,
	local_elevator_chan <-chan elev_struct.Elevator){

	var allHallOrders HallOrdersAllElevators = initHallOrdersAllElevators(id) //bruk mutex rundt denne
	var allElevatorStates = initAllElevatorStates(id)
	var availableElevators = make(map[string]bool) //bruk mutex rundt denne
	availableElevators[id] = true
	var dataMutex sync.RWMutex

	order_confirmed_chan := make(chan elevio.ButtonEvent, config.BUFFER_SIZE)
	order_reset_chan := make(chan elevio.ButtonEvent, config.BUFFER_SIZE)
	hall_light_chan := make(chan elev_struct.LightEvent, config.BUFFER_SIZE)

	go confirmHallOrders(order_confirmed_chan, hall_light_chan, &allHallOrders, &availableElevators, &dataMutex)
	go resetHallOrders(order_reset_chan, hall_light_chan, &allHallOrders, &availableElevators, &dataMutex)

	go RunOrderManager(
		id, 
		local_elevator_chan, 
		assign_order_chan, 
		completed_order_chan, 
		clear_local_hall_orders_chan, 
		hall_light_chan, 
		order_confirmed_chan, 
		order_reset_chan, 
		allHallOrders, 
		allElevatorStates, 
		availableElevators, 
		&dataMutex)
}