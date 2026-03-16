package orders

import (
	elevio "Driver-go"
	"heislab-sanntid/config"
	"heislab-sanntid/elevator/elev_struct"
	network "heislab-sanntid/network"
	types "heislab-sanntid/types"
	"log"
	"sync"
	"time"
	//"fmt"
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

type HallOrdersAllElevators = types.HallOrdersAllElevators

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

func confirmHallOrders(
	localID string,
	order_confirmed_chan chan<- elevio.ButtonEvent,
	allHallOrders *HallOrdersAllElevators,
	availableElevators *map[string]bool,
	dataMutex *sync.RWMutex) {

	var confirmAlreadySent [config.N_FLOORS][config.N_BUTTONS - 1]bool

	for {
		time.Sleep(10 * time.Millisecond)
		var ordersToConfirm []elevio.ButtonEvent

		dataMutex.RLock()
		hasAvailable := false
		for _, isAvailable := range *availableElevators {
			if isAvailable {
				hasAvailable = true
				break
			}
		}
		if !hasAvailable {
			dataMutex.RUnlock()
			continue
		}

		localOrders, ok := (*allHallOrders)[localID]
		if !ok {
			dataMutex.RUnlock()
			continue
		}

		for floor := 0; floor < config.N_FLOORS; floor++ {
			for btn := 0; btn < config.N_BUTTONS-1; btn++ {
				shouldConfirm := localOrders[floor][btn] == NEW
				if shouldConfirm && !confirmAlreadySent[floor][btn] {
					confirmAlreadySent[floor][btn] = true
					ordersToConfirm = append(ordersToConfirm, elevio.ButtonEvent{Floor: floor, Button: elevio.ButtonType(btn)})
				} else if !shouldConfirm {
					confirmAlreadySent[floor][btn] = false
				}
			}
		}
		dataMutex.RUnlock()

		for _, event := range ordersToConfirm {
			order_confirmed_chan <- event
		}
	}
}

func resetHallOrders(
	localID string,
	order_reset_chan chan<- elevio.ButtonEvent,
	allHallOrders *HallOrdersAllElevators,
	availableElevators *map[string]bool,
	dataMutex *sync.RWMutex) {

	var resetAlreadySent [config.N_FLOORS][config.N_BUTTONS - 1]bool

	for {
		time.Sleep(10 * time.Millisecond)
		var ordersToReset []elevio.ButtonEvent

		dataMutex.RLock()
		hasAvailable := false
		for _, isAvailable := range *availableElevators {
			if isAvailable {
				hasAvailable = true
				break
			}
		}
		if !hasAvailable {
			dataMutex.RUnlock()
			continue
		}

		localOrders, ok := (*allHallOrders)[localID]
		if !ok {
			dataMutex.RUnlock()
			continue
		}

		for floor := 0; floor < config.N_FLOORS; floor++ {
			for btn := 0; btn < config.N_BUTTONS-1; btn++ {
				shouldReset := localOrders[floor][btn] == COMPLETED
				if shouldReset && !resetAlreadySent[floor][btn] {
					resetAlreadySent[floor][btn] = true
					ordersToReset = append(ordersToReset, elevio.ButtonEvent{Floor: floor, Button: elevio.ButtonType(btn)})
				} else if !shouldReset {
					resetAlreadySent[floor][btn] = false
				}
			}
		}
		dataMutex.RUnlock()

		for _, event := range ordersToReset {
			order_reset_chan <- event
		}
	}
}

func reopenDistributedHallOrders(hallOrders HallOrders) HallOrders {
	for floor := 0; floor < config.N_FLOORS; floor++ {
		for btn := 0; btn < config.N_BUTTONS-1; btn++ {
			if hallOrders[floor][btn] == ASSIGNED || hallOrders[floor][btn] == CONFIRMED {
				hallOrders[floor][btn] = NEW
			}
		}
	}
	return hallOrders
}

func setOrdersToAssigned(assignedOrders [config.N_FLOORS][config.N_BUTTONS - 1]bool, hallOrders HallOrders) HallOrders {
	for floor := 0; floor < config.N_FLOORS; floor++ {
		for btn := 0; btn < config.N_BUTTONS-1; btn++ {
			if assignedOrders[floor][btn] {
				hallOrders[floor][btn] = ASSIGNED
			}
		}
	}
	return hallOrders
}

func handleElevatorUnavailable(unavailableID string, allHallOrders HallOrdersAllElevators, availableElevators map[string]bool) {
	for id, isAvailable := range availableElevators {
		if isAvailable && id != unavailableID {
			allHallOrders[id] = reopenDistributedHallOrders(allHallOrders[id])
		}
	}
	allHallOrders[unavailableID] = initHallOrders()
}

func applyAvailabilityTransition(elevator elev_struct.Elevator, allHallOrders HallOrdersAllElevators, availableElevators map[string]bool) {
	if elevator.Stuck && availableElevators[elevator.ID] {
		availableElevators[elevator.ID] = false
		handleElevatorUnavailable(elevator.ID, allHallOrders, availableElevators)
	} else if !elevator.Stuck && !availableElevators[elevator.ID] {
		availableElevators[elevator.ID] = true
	}
}

func applyLocalElevatorUpdate(
	localID string,
	localElevator elev_struct.Elevator,
	allHallOrders HallOrdersAllElevators,
	allElevators types.AllElevators,
	availableElevators map[string]bool) (types.Elevator, HallOrders) {

	allElevators[localID] = localElevator
	applyAvailabilityTransition(localElevator, allHallOrders, availableElevators)
	allHallOrders[localID] = AddNewLocalOrder(allHallOrders[localID], localElevator.Requests)

	return allElevators[localID], allHallOrders[localID]
}

func applyRemoteElevatorUpdate(
	localID string,
	remoteElevator elev_struct.Elevator,
	remoteHallOrders types.HallOrders,
	allHallOrders HallOrdersAllElevators,
	allElevators types.AllElevators,
	availableElevators map[string]bool) {

	allElevators[remoteElevator.ID] = remoteElevator
	allHallOrders[remoteElevator.ID] = remoteHallOrders
	allHallOrders[localID] = UpdateLocalHallOrders(allHallOrders[localID], remoteHallOrders)
	applyAvailabilityTransition(remoteElevator, allHallOrders, availableElevators)
}

func RunOrderManager(
	id string,
	localElevatorChan <-chan types.Elevator,
	completedOrderChan <-chan elevio.ButtonEvent,
	reassignedHallOrdersChan chan<- [config.N_FLOORS][config.N_BUTTONS - 1]bool,
	hallLightChan chan elev_struct.LightEvent,
	orderConfirmedChan <-chan elevio.ButtonEvent,
	orderResetChan <-chan elevio.ButtonEvent,
	allHallOrders HallOrdersAllElevators,
	allElevators types.AllElevators,
	availableElevators map[string]bool,
	dataMutex *sync.RWMutex) {

	networkResendTicker := time.NewTicker(100 * time.Millisecond)
	defer networkResendTicker.Stop()

	for {
		select {
		// Unsure if peers returns IDs. Will be tested. DB
		case peerUpdate := <-network.Peers():
			log.Printf("peer update case, peers: %v", peerUpdate.Peers)
			dataMutex.Lock()
			for _, peer := range peerUpdate.Peers {
				if peer != id {
					if _, ok := availableElevators[peer]; !ok {
						availableElevators[peer] = true
						allHallOrders[peer] = initHallOrders()
						allElevators[peer] = elev_struct.ElevatorInit(peer)
					} else if !availableElevators[peer] {
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
					handleElevatorUnavailable(lostPeer, allHallOrders, availableElevators)
				}
			}
			dataMutex.Unlock()

		case localElevator := <-localElevatorChan:
			//log.Printf("localElevator case")
			dataMutex.Lock()
			elevatorSnapshot, hallOrdersSnapshot := applyLocalElevatorUpdate(id, localElevator, allHallOrders, allElevators, availableElevators)
			dataMutex.Unlock()

			network.NetworkSend(elevatorSnapshot, hallOrdersSnapshot)

		case remoteElevator := <-network.NetworkRxChan():
			if remoteElevator.Elevator.ID == id {
				continue
			}

			dataMutex.Lock()
			applyRemoteElevatorUpdate(id, remoteElevator.Elevator, remoteElevator.HallOrders, allHallOrders, allElevators, availableElevators)
			dataMutex.Unlock()

		case newCompletedOrder := <-completedOrderChan:
			if newCompletedOrder.Button == elevio.BT_Cab {
				continue
			}
			log.Printf("completedOrderChan case")

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
			log.Printf("ConfirmedChan case, floor: %d, button: %d", orderToConfirm.Floor, orderToConfirm.Button)
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

			elevatorSnapshot := allElevators[id]
			hallOrdersSnapshot := allHallOrders[id]
			dataMutex.Unlock()

			hallLightChan <- elev_struct.LightEvent{Floor: orderToConfirm.Floor, Button: elevio.ButtonType(orderToConfirm.Button), On: true}

			reassignedHallOrdersChan <- hallOrdersForId

			network.NetworkSend(elevatorSnapshot, hallOrdersSnapshot)

		case orderToReset := <-orderResetChan:
			log.Printf("ResetChan case, floor: %d, button: %d", orderToReset.Floor, orderToReset.Button)
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
			log.Printf("hallLightChan case, floor: %d, button: %d, on: %v", hallLightEvent.Floor, hallLightEvent.Button, hallLightEvent.On)
			elevio.SetButtonLamp(hallLightEvent.Button, hallLightEvent.Floor, hallLightEvent.On)

		case <-networkResendTicker.C:
			dataMutex.RLock()
			elevatorSnapshot := allElevators[id]
			hallOrdersSnapshot := allHallOrders[id]
			dataMutex.RUnlock()

			network.NetworkSend(elevatorSnapshot, hallOrdersSnapshot)
		}
	}
}

func OrdersInit(id string,
	reassignLocalHallOrdersChan chan<- [config.N_FLOORS][config.N_BUTTONS - 1]bool,
	completedOrderChan <-chan elevio.ButtonEvent,
	localElevatorChan <-chan types.Elevator) {

	var allHallOrders HallOrdersAllElevators = initHallOrdersAllElevators(id) //bruk mutex rundt denne
	var allElevators = initAllElevators(id)
	var availableElevators = make(map[string]bool) //bruk mutex rundt denne
	availableElevators[id] = true
	var dataMutex sync.RWMutex

	orderConfirmedChan := make(chan elevio.ButtonEvent, config.BUFFER_SIZE)
	orderResetChan := make(chan elevio.ButtonEvent, config.BUFFER_SIZE)
	hallLightChan := make(chan elev_struct.LightEvent, config.BUFFER_SIZE)

	go confirmHallOrders(id, orderConfirmedChan, &allHallOrders, &availableElevators, &dataMutex)
	go resetHallOrders(id, orderResetChan, &allHallOrders, &availableElevators, &dataMutex)

	go RunOrderManager(
		id,
		localElevatorChan,
		completedOrderChan,
		reassignLocalHallOrdersChan,
		hallLightChan,
		orderConfirmedChan,
		orderResetChan,
		allHallOrders,
		allElevators,
		availableElevators,
		&dataMutex)
}
