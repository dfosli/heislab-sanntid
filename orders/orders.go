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

type OrderState = types.OrderState
type HallOrders = types.HallOrders

const (
	NONE      = types.NONE
	NEW       = types.NEW
	CONFIRMED = types.CONFIRMED
	ASSIGNED  = types.ASSIGNED
	COMPLETED = types.COMPLETED
)

type HallOrdersAllElevators map[string]HallOrders

type AllElevatorStates map[string]elev_struct.Elevator

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

func toNetworkHallOrders(hallOrders HallOrders) HallOrders {
	var networkHallOrders HallOrders
	for floor := 0; floor < config.N_FLOORS; floor++ {
		for btn := 0; btn < config.N_BUTTONS-1; btn++ {
			networkHallOrders[floor][btn] = OrderState(hallOrders[floor][btn])
		}
	}
	return networkHallOrders
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

func setOrdersToAssigned(id string, allHallOrders HallOrdersAllElevators, availableElevators map[string]bool) HallOrdersAllElevators {
	for floor := 0; floor < config.N_FLOORS; floor++ {
		for btn := 0; btn < config.N_BUTTONS-1; btn++ {
			if allHallOrders[id][floor][btn] == CONFIRMED {
				for elevId, isAvailable := range availableElevators {
					if isAvailable {
						orders := allHallOrders[elevId]
						orders[floor][btn] = ASSIGNED
						allHallOrders[elevId] = orders
					}
				}
			}
		}
	}
	return allHallOrders
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
	// networkTx chan<- ElevstateHallorderPair,
	// networkRx <-chan ElevstateHallorderPair,
	//peer_update_chan <-chan peers.PeerUpdate
) {
	// Ikke accsess direkte til variabler fra network. DB

	var allHallOrders HallOrdersAllElevators = initHallOrdersAllElevators(id) //bruk mutex rundt denne
	var allElevatorStates = initAllElevatorStates(id)
	var availableElevators = make(map[string]bool) //bruk mutex rundt denne
	availableElevators[id] = true
	var dataMutex sync.RWMutex

	orderConfirmedChan := make(chan elevio.ButtonEvent, config.BUFFER_SIZE)
	orderResetChan := make(chan elevio.ButtonEvent, config.BUFFER_SIZE)
	hallLightChan := make(chan elev_struct.LightEvent, config.BUFFER_SIZE)

	go confirmHallOrders(orderConfirmedChan, hallLightChan, &allHallOrders, &availableElevators, &dataMutex)
	go resetHallOrders(orderResetChan, hallLightChan, &allHallOrders, &availableElevators, &dataMutex)

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
			// Could leed to fatal problems. DB
			availableElevators[peerUpdate.New] = true

			for _, lostPeer := range peerUpdate.Lost {
				delete(availableElevators, lostPeer) //! delete? eller sett til false?
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

			//TODO: ny ordre? hvis vi har none og den har true, sett til new
			//network.NetworkSend()

		case remoteElevator := <-network.NetworkRxChan():
			newHallOrder := UpdateLocalHallOrdersIfPossible(allHallOrders[id], remoteElevator.HallOrders)

			dataMutex.Lock()
			allHallOrders[id] = newHallOrder
			dataMutex.Unlock()

		case newCompletedOrder := <-completedOrderChan:
			dataMutex.Lock()
			if orders, ok := allHallOrders[id]; ok {
				orders[newCompletedOrder.Floor][newCompletedOrder.Button] = COMPLETED
				allHallOrders[id] = orders
				//TODO: network send
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
			dataMutex.Unlock()

			//TODO: kjør distribution

			dataMutex.Lock()
			allHallOrders = setOrdersToAssigned(id, allHallOrders, availableElevators)
			dataMutex.Unlock()

			//hvis assigned til oss, send til elevator
			//loope gjennom assigned orders, og sende button_events til assign_order_chan for hver

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
		}
	}
}
