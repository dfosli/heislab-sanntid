package orders

import (
	elevio "Driver-go"
	"fmt"
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

func initCabOrdersAllElevators(id string) types.CabOrders {
	allCabOrders := make(types.CabOrders)
	allCabOrders[id] = elev_struct.GetCabOrders(elev_struct.ElevatorInit(id))
	return allCabOrders
}

func hasCabOrders(cabOrders [config.N_FLOORS]bool) bool {
	for floor := 0; floor < config.N_FLOORS; floor++ {
		if cabOrders[floor] {
			return true
		}
	}
	return false
}

func mergeCabOrders(allCabOrders types.CabOrders, remoteCabOrders types.CabOrders, remoteID string, remoteRecovering bool) {
	for id, cabOrders := range remoteCabOrders {
		if remoteRecovering && id == remoteID && !hasCabOrders(cabOrders) {
			continue //!This can theoretically cause caborder loss if a recovering elevator receives a caborder Very quickly, before it receives its caborders from other elevators.
		}
		allCabOrders[id] = cabOrders
	}
}

func recoverLocalCabOrders(localID string, allCabOrders types.CabOrders, allElevators types.AllElevators) [config.N_FLOORS]bool {
	var recoveredOrders [config.N_FLOORS]bool

	localCabOrders, ok := allCabOrders[localID]
	if !ok {
		return recoveredOrders
	}

	localElevator, ok := allElevators[localID]
	if !ok {
		return recoveredOrders
	}

	for floor := 0; floor < config.N_FLOORS; floor++ {
		if !localCabOrders[floor] || localElevator.Requests[floor][elevio.BT_Cab] {
			continue
		}

		localElevator.Requests[floor][elevio.BT_Cab] = true
		recoveredOrders[floor] = true
	}

	// allElevators[localID] = localElevator
	allCabOrders[localID] = elev_struct.GetCabOrders(localElevator)

	return recoveredOrders
}

func cloneCabOrders(cabOrders types.CabOrders) types.CabOrders {
	clonedCabOrders := make(types.CabOrders, len(cabOrders))
	for id, localCabOrders := range cabOrders {
		clonedCabOrders[id] = localCabOrders
	}
	return clonedCabOrders
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

func applyLocalElevatorUpdate(
	localID string,
	localElevator elev_struct.Elevator,
	allHallOrders HallOrdersAllElevators,
	allElevators types.AllElevators,
	allCabOrders types.CabOrders,
	availableElevators map[string]bool) (types.Elevator, HallOrders) {

	allElevators[localID] = localElevator
	allCabOrders[localID] = elev_struct.GetCabOrders(localElevator)

	if localElevator.Stuck && availableElevators[localElevator.ID] {
		availableElevators[localElevator.ID] = false
		handleElevatorUnavailable(localElevator.ID, allHallOrders, availableElevators)
	} else if !localElevator.Stuck && !availableElevators[localElevator.ID] {
		availableElevators[localElevator.ID] = true
	}

	allHallOrders[localID] = AddNewLocalOrder(allHallOrders[localID], localElevator.Requests)

	return allElevators[localID], allHallOrders[localID]
}

func applyRemoteElevatorUpdate(
	localID string,
	remoteElevatorMsg network.NetworkMsg,
	allHallOrders HallOrdersAllElevators,
	allElevators types.AllElevators,
	allCabOrders types.CabOrders) {

	allElevators[remoteElevatorMsg.Elevator.ID] = remoteElevatorMsg.Elevator
	allHallOrders[remoteElevatorMsg.Elevator.ID] = remoteElevatorMsg.HallOrders

	mergeCabOrders(allCabOrders, remoteElevatorMsg.CabOrders, remoteElevatorMsg.Elevator.ID, remoteElevatorMsg.Recovering)
	allHallOrders[localID] = UpdateLocalHallOrders(allHallOrders[localID], remoteElevatorMsg.HallOrders)
}

func RunOrderManager(
	id string,
	localElevatorChan <-chan types.Elevator,
	completedOrderChan <-chan elevio.ButtonEvent,
	reassignedHallOrdersChan chan<- [config.N_FLOORS][config.N_BUTTONS - 1]bool,
	recoveredCabOrdersChan chan<- [config.N_FLOORS]bool,
	hallLightChan chan elev_struct.LightEvent,
	orderConfirmedChan <-chan elevio.ButtonEvent,
	orderResetChan <-chan elevio.ButtonEvent,
	allHallOrders HallOrdersAllElevators,
	allElevators types.AllElevators,
	allCabOrders types.CabOrders,
	availableElevators map[string]bool,
	dataMutex *sync.RWMutex) {

	networkResendTicker := time.NewTicker(100 * time.Millisecond)
	defer networkResendTicker.Stop()
	cabOrderRecoveryDeadline := time.Now().Add(2 * time.Second)

	for {
		select {
		case peerUpdate := <-network.Peers():
			fmt.Printf("peer update case, peers: %v", peerUpdate.Peers)
			dataMutex.Lock()
			for _, peer := range peerUpdate.Peers {
				if peer != id {
					if _, ok := availableElevators[peer]; !ok {
						availableElevators[peer] = true
						allHallOrders[peer] = initHallOrders()
						allElevators[peer] = elev_struct.ElevatorInit(peer)
					} else if !availableElevators[peer] {
						availableElevators[peer] = true

						for id, isAvailable := range availableElevators {
							if isAvailable {
								if orders, ok := allHallOrders[id]; ok {
									orders = reopenDistributedHallOrders(orders)
									allHallOrders[id] = orders
								}
							}
						}

					}
				}
			}

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
			//fmt.Println("localElevator case")
			dataMutex.Lock()
			wasAvailable := availableElevators[id]
			elevatorSnapshot, hallOrdersSnapshot := applyLocalElevatorUpdate(id, localElevator, allHallOrders, allElevators, allCabOrders, availableElevators)
			isAvailable := availableElevators[id]
			recoveringCabOrders := time.Now().Before(cabOrderRecoveryDeadline)
			allCabOrdersSnapshot := cloneCabOrders(allCabOrders)
			dataMutex.Unlock()

			if wasAvailable != isAvailable {
				network.SetPeerTxEnable(isAvailable)
			}

			network.NetworkSend(elevatorSnapshot, hallOrdersSnapshot, allCabOrdersSnapshot, recoveringCabOrders)

		case remoteElevatorMsg := <-network.NetworkRxChan():
			if remoteElevatorMsg.Elevator.ID == id {
				continue
			}

			dataMutex.Lock()
			applyRemoteElevatorUpdate(id, remoteElevatorMsg, allHallOrders, allElevators, allCabOrders)

			if time.Now().Before(cabOrderRecoveryDeadline) {
				recoveredCabOrders := recoverLocalCabOrders(id, allCabOrders, allElevators)
				recoveredCabOrdersChan <- recoveredCabOrders
			}
			dataMutex.Unlock()

		case newCompletedOrder := <-completedOrderChan:
			if newCompletedOrder.Button == elevio.BT_Cab {
				continue
			}

			fmt.Println("completedOrderChan case")

			shouldSend := false
			var elevatorSnapshot elev_struct.Elevator
			var hallOrdersSnapshot HallOrders
			var allCabOrdersSnapshot types.CabOrders

			dataMutex.Lock()
			if orders, ok := allHallOrders[id]; ok {
				orders[newCompletedOrder.Floor][newCompletedOrder.Button] = COMPLETED
				allHallOrders[id] = orders

				elevatorSnapshot = allElevators[id]
				hallOrdersSnapshot = allHallOrders[id]
				allCabOrdersSnapshot = cloneCabOrders(allCabOrders)
				shouldSend = true
			}

			dataMutex.Unlock()

			if shouldSend {
				network.NetworkSend(elevatorSnapshot, hallOrdersSnapshot, allCabOrdersSnapshot, time.Now().Before(cabOrderRecoveryDeadline))
			}

		case orderToConfirm := <-orderConfirmedChan:
			fmt.Printf("ConfirmedChan case, floor: %d, button: %d", orderToConfirm.Floor, orderToConfirm.Button)
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
			allCabOrdersSnapshot := cloneCabOrders(allCabOrders)
			dataMutex.Unlock()

			hallLightChan <- elev_struct.LightEvent{Floor: orderToConfirm.Floor, Button: elevio.ButtonType(orderToConfirm.Button), On: true}

			reassignedHallOrdersChan <- hallOrdersForId

			network.NetworkSend(elevatorSnapshot, hallOrdersSnapshot, allCabOrdersSnapshot, time.Now().Before(cabOrderRecoveryDeadline))

		case orderToReset := <-orderResetChan:
			fmt.Printf("ResetChan case, floor: %d, button: %d", orderToReset.Floor, orderToReset.Button)
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
			fmt.Printf("hallLightChan case, floor: %d, button: %d, on: %v", hallLightEvent.Floor, hallLightEvent.Button, hallLightEvent.On)
			elevio.SetButtonLamp(hallLightEvent.Button, hallLightEvent.Floor, hallLightEvent.On)

		case <-networkResendTicker.C:
			dataMutex.RLock()
			elevatorSnapshot := allElevators[id]
			hallOrdersSnapshot := allHallOrders[id]
			allCabOrdersSnapshot := cloneCabOrders(allCabOrders)
			recoveringCabOrders := time.Now().Before(cabOrderRecoveryDeadline)
			dataMutex.RUnlock()

			network.NetworkSend(elevatorSnapshot, hallOrdersSnapshot, allCabOrdersSnapshot, recoveringCabOrders)
		}
	}
}

func OrdersInit(id string,
	reassignLocalHallOrdersChan chan<- [config.N_FLOORS][config.N_BUTTONS - 1]bool,
	recoveredCabOrdersChan chan<- [config.N_FLOORS]bool,
	completedOrderChan <-chan elevio.ButtonEvent,
	localElevatorChan <-chan types.Elevator) {

	var allHallOrders HallOrdersAllElevators = initHallOrdersAllElevators(id) //bruk mutex rundt denne
	var allElevators = initAllElevators(id)
	var allCabOrders = initCabOrdersAllElevators(id)
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
		recoveredCabOrdersChan,
		hallLightChan,
		orderConfirmedChan,
		orderResetChan,
		allHallOrders,
		allElevators,
		allCabOrders,
		availableElevators,
		&dataMutex)
}
