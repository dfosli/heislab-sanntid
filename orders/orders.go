package orders

import (
	elevio "Driver-go"
	//"fmt"
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

func initAllElevators(id string) types.AllElevators {
	allElevators := make(types.AllElevators)
	allElevators[id] = elev_struct.ElevatorInit(id)
	return allElevators
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
	allHallOrders *HallOrdersAllElevators,
	availableElevators *map[string]bool,
	dataMutex *sync.RWMutex) {

	for {
		time.Sleep(10 * time.Millisecond)
		var ordersToConfirm []elevio.ButtonEvent

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
					ordersToConfirm = append(ordersToConfirm, elevio.ButtonEvent{Floor: floor, Button: elevio.ButtonType(btn)})
				}
			}
		}
		dataMutex.RUnlock()

		for _, event := range ordersToConfirm {
			select {
			case order_confirmed_chan <- event:
			default:
			}
		}
	}
}

func resetHallOrders(
	order_reset_chan chan<- elevio.ButtonEvent,
	allHallOrders *HallOrdersAllElevators,
	availableElevators *map[string]bool,
	dataMutex *sync.RWMutex) {

	for {
		time.Sleep(10 * time.Millisecond)
		var ordersToReset []elevio.ButtonEvent

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
					ordersToReset = append(ordersToReset, elevio.ButtonEvent{Floor: floor, Button: elevio.ButtonType(btn)})
				}
			}
		}
		dataMutex.RUnlock()

		for _, event := range ordersToReset {
			select {
			case order_reset_chan <- event:
			default:
			}
		}
	}
}

func unassignHallOrders(hallOrders HallOrders) HallOrders {
	for floor := 0; floor < config.N_FLOORS; floor++ {
		for btn := 0; btn < config.N_BUTTONS-1; btn++ {
			if hallOrders[floor][btn] == ASSIGNED || hallOrders[floor][btn] == CONFIRMED {
				hallOrders[floor][btn] = NEW
			}
		}
	}
	return hallOrders
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
			allHallOrders[id] = unassignHallOrders(allHallOrders[id])
		}
	}
	allHallOrders[lost_id] = initHallOrders()

	return allHallOrders
}

func checkStuckAndUpdateAvailable(elevator elev_struct.Elevator, allHallOrders HallOrdersAllElevators, availableElevators map[string]bool) (HallOrdersAllElevators, map[string]bool) {
	if elevator.Stuck && availableElevators[elevator.ID] {
		availableElevators[elevator.ID] = false
		allHallOrders = lostPeerReassignOrders(elevator.ID, allHallOrders, availableElevators)
	} else if !elevator.Stuck && !availableElevators[elevator.ID] {
		availableElevators[elevator.ID] = true
	}
	return allHallOrders, availableElevators
}

func RunOrderManager(
	id string,
	localElevatorChan <-chan types.Elevator,
	assignOrderChan chan<- elevio.ButtonEvent,
	completedOrderChan <-chan elevio.ButtonEvent,
	clearLocalHallOrdersChan chan<- bool,
	hallLightChan chan elev_struct.LightEvent,
	orderConfirmedChan <-chan elevio.ButtonEvent,
	orderResetChan <-chan elevio.ButtonEvent,
	allHallOrders HallOrdersAllElevators,
	allElevators types.AllElevators,
	availableElevators map[string]bool,
	dataMutex *sync.RWMutex) {

	for {
		select {
		// Unsure if peers returns IDs. Will be tested. DB
		case peerUpdate := <-network.Peers():
			dataMutex.Lock()
			for _, peer := range peerUpdate.Peers {
				if peer != id {
					if _, ok := availableElevators[peer]; !ok {
						availableElevators[peer] = true
						allHallOrders[peer] = initHallOrders()
						allElevators[peer] = elev_struct.ElevatorInit(peer)
					} else {
						availableElevators[peer] = true
					}
					// unassigne ordre her slik at de blir fordelt på de tilgjengelige?
					// men da vil det kanskje reassignes hver gang det kommer en peer update...
				}
			}

			// Needed? DB
			// Could leed to fatal problems. DB
			// availableElevators[peerUpdate.New] = true //! New starter som [] og det blir da lagt til en ny heis med tom id. Har derfor kommentert ut

			for _, lostPeer := range peerUpdate.Lost {
				if lostPeer == id {
					continue
				}
				if availableElevators[lostPeer] {
					availableElevators[lostPeer] = false
					allHallOrders = lostPeerReassignOrders(lostPeer, allHallOrders, availableElevators)
				}
			}
			dataMutex.Unlock()

		case localElevator := <-localElevatorChan:
			allElevators[id] = localElevator

			dataMutex.Lock()
			allHallOrders, availableElevators = checkStuckAndUpdateAvailable(localElevator, allHallOrders, availableElevators)
			allHallOrders[id] = AddNewLocalOrder(allHallOrders[id], localElevator.Requests)
			
			network.NetworkSend(allElevators[id], allHallOrders[id])
			dataMutex.Unlock()

		case remoteElevator := <-network.NetworkRxChan():
			dataMutex.Lock()
			allElevators[remoteElevator.Elevator.ID] = remoteElevator.Elevator

			newHallOrder := UpdateLocalHallOrdersIfPossible(allHallOrders[id], fromNetworkHallOrders(remoteElevator.HallOrders)) //this function is added now as long as the HallOrders stuff is not working
			allHallOrders[id] = newHallOrder
			
			allHallOrders, availableElevators = checkStuckAndUpdateAvailable(remoteElevator.Elevator, allHallOrders, availableElevators)
			dataMutex.Unlock()

		case newCompletedOrder := <-completedOrderChan:
			dataMutex.Lock()
			if orders, ok := allHallOrders[id]; ok {
				orders[newCompletedOrder.Floor][newCompletedOrder.Button] = COMPLETED
				allHallOrders[id] = orders

				// type NetworkMsg struct {
				// 	ID            string
				// 	HallOrders    types.HallOrders
				// 	ElevatorState types.ElevatorState
				// }

				network.NetworkSend(allElevators[id], allHallOrders[id])
			}
			dataMutex.Unlock()

		case orderToConfirm := <-orderConfirmedChan:
			dataMutex.Lock()

			minOneAvailable := false
			for _, isAvailable := range availableElevators {
				if isAvailable {
					minOneAvailable = true
					break
				}
			}
			if !minOneAvailable {
				dataMutex.Unlock()
				continue
			}

			for elev_id, isAvailable := range availableElevators {
				if isAvailable {
					if orders, ok := allHallOrders[elev_id]; ok {
						orders[orderToConfirm.Floor][orderToConfirm.Button] = CONFIRMED
						allHallOrders[elev_id] = orders
					}
				}
			}

			hallOrdersForId, err := ReassignOrders(id, allHallOrders[id], availableElevators, allElevators)
			if err != nil {
				dataMutex.Unlock()
				continue
			}

			allHallOrders[id] = setOrdersToAssigned(hallOrdersForId, allHallOrders[id])			
			dataMutex.Unlock()

			hallLightChan <- elev_struct.LightEvent{Floor: orderToConfirm.Floor, Button: elevio.ButtonType(orderToConfirm.Button), On: true}

			clearLocalHallOrdersChan <- true
			for floor := 0; floor < config.N_FLOORS; floor++ {
				for btn := 0; btn < config.N_BUTTONS-1; btn++ {
					if hallOrdersForId[floor][btn] {
						assignOrderChan <- elevio.ButtonEvent{Floor: floor, Button: elevio.ButtonType(btn)}
					}
				}
			}

			dataMutex.RLock()
			network.NetworkSend(allElevators[id], allHallOrders[id])
			dataMutex.RUnlock()

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
			dataMutex.Unlock()

			hallLightChan <- elev_struct.LightEvent{Floor: orderToReset.Floor, Button: elevio.ButtonType(orderToReset.Button), On: false}

		case hallLightEvent := <-hallLightChan:
			elevio.SetButtonLamp(hallLightEvent.Button, hallLightEvent.Floor, hallLightEvent.On)

		default:
			dataMutex.RLock()
			network.NetworkSend(allElevators[id], allHallOrders[id])
			dataMutex.RUnlock()
		}
	}
}

func OrdersInit(id string,
	clear_local_hall_orders_chan chan<- bool,
	completed_order_chan <-chan elevio.ButtonEvent,
	assign_order_chan chan<- elevio.ButtonEvent,
	localElevatorChan <-chan types.Elevator) {

	var allHallOrders HallOrdersAllElevators = initHallOrdersAllElevators(id) //bruk mutex rundt denne
	var allElevators = initAllElevators(id)
	var availableElevators = make(map[string]bool) //bruk mutex rundt denne
	availableElevators[id] = true
	var dataMutex sync.RWMutex

	order_confirmed_chan := make(chan elevio.ButtonEvent, config.BUFFER_SIZE)
	order_reset_chan := make(chan elevio.ButtonEvent, config.BUFFER_SIZE)
	hall_light_chan := make(chan elev_struct.LightEvent, config.BUFFER_SIZE)

	go confirmHallOrders(order_confirmed_chan, &allHallOrders, &availableElevators, &dataMutex)
	go resetHallOrders(order_reset_chan, &allHallOrders, &availableElevators, &dataMutex)

	go RunOrderManager(
		id,
		localElevatorChan,
		assign_order_chan,
		completed_order_chan,
		clear_local_hall_orders_chan,
		hall_light_chan,
		order_confirmed_chan,
		order_reset_chan,
		allHallOrders,
		allElevators,
		availableElevators,
		&dataMutex)
}
