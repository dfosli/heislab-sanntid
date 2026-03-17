package orders

import (
	elevio "Driver-go"
	"fmt"
	"heislab-sanntid/config"
	"heislab-sanntid/elevator/elev_struct"
	network "heislab-sanntid/network"
	types "heislab-sanntid/types"
	"maps"
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

func initHallOrdersAllElevators(id string) HallOrdersAllElevators {
	allHallOrders := make(HallOrdersAllElevators)
	allHallOrders[id] = setAllOrders(NONE)
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

func setAllOrders(orderState OrderState) HallOrders {
	var hallOrders HallOrders
	for floor := 0; floor < config.N_FLOORS; floor++ {
		for btn := 0; btn < config.N_BUTTONS-1; btn++ {
			hallOrders[floor][btn] = orderState
		}
	}
	return hallOrders
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

		localOrders, localOk := (*allHallOrders)[localID]

		for floor := 0; floor < config.N_FLOORS; floor++ {
			for btn := 0; btn < config.N_BUTTONS-1; btn++ {
				allAtLeastNew := true
				for elevID, isAvailable := range *availableElevators {
					if !isAvailable {
						continue
					}
					orders, ok := (*allHallOrders)[elevID]
					if !ok {
						allAtLeastNew = false
						break
					}
					if orders[floor][btn] < NEW {
						allAtLeastNew = false
						break
					}
				}

				localOrder := OrderState(NONE)
				if localOk {
					localOrder = localOrders[floor][btn]
				}
				shouldConfirm := allAtLeastNew && localOrder == NEW

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

		for floor := 0; floor < config.N_FLOORS; floor++ {
			for btn := 0; btn < config.N_BUTTONS-1; btn++ {
				allCompletedOrNone := true
				atLeastOneCompleted := false
				for elevID, isAvailable := range *availableElevators {
					if !isAvailable {
						continue
					}
					orders, ok := (*allHallOrders)[elevID]
					if !ok {
						allCompletedOrNone = false
						break
					}
					state := orders[floor][btn]
					if state == COMPLETED {
						atLeastOneCompleted = true
					} else if state != NONE {
						allCompletedOrNone = false
						break
					}
				}

				shouldReset := allCompletedOrNone && atLeastOneCompleted
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

func rollbackHallOrders(hallOrders HallOrders) HallOrders {
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
			allHallOrders[id] = rollbackHallOrders(allHallOrders[id])
		}
	}
	allHallOrders[unavailableID] = setAllOrders(NONE)
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
	availableElevators map[string]bool,
	allHallOrders HallOrdersAllElevators,
	allElevators types.AllElevators,
	allCabOrders types.CabOrders) {

	if !availableElevators[remoteElevatorMsg.Elevator.ID] {
		return
	}

	allElevators[remoteElevatorMsg.Elevator.ID] = remoteElevatorMsg.Elevator
	allHallOrders[remoteElevatorMsg.Elevator.ID] = remoteElevatorMsg.HallOrders

	mergeCabOrders(allCabOrders, remoteElevatorMsg.CabOrders, remoteElevatorMsg.Elevator.ID, remoteElevatorMsg.CabOrdersRecovering)
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
			fmt.Printf("peer update case, peers: %v\n", peerUpdate.Peers)
			dataMutex.Lock()
			for _, peer := range peerUpdate.Peers {
				if peer != id {
					if _, ok := availableElevators[peer]; !ok {
						availableElevators[peer] = true
						allHallOrders[peer] = setAllOrders(NONE)
						allElevators[peer] = elev_struct.ElevatorInit(peer)
					} else if !availableElevators[peer] {
						availableElevators[peer] = true

						for elevID, isAvailable := range availableElevators {
							if isAvailable {
								if orders, ok := allHallOrders[elevID]; ok {
									orders = rollbackHallOrders(orders)
									allHallOrders[elevID] = orders
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
			cabOrdersSnapshot := maps.Clone(allCabOrders)
			dataMutex.Unlock()

			if wasAvailable != isAvailable {
				network.SetPeerTxEnable(isAvailable)
			}

			network.NetworkSend(elevatorSnapshot, hallOrdersSnapshot, cabOrdersSnapshot, recoveringCabOrders)

		case remoteElevatorMsg := <-network.NetworkRxChan():
			if remoteElevatorMsg.Elevator.ID == id {
				continue
			}

			dataMutex.Lock()
			applyRemoteElevatorUpdate(id, remoteElevatorMsg, availableElevators, allHallOrders, allElevators, allCabOrders)

			shouldSend := false
			var recoveredCabOrders [config.N_FLOORS]bool
			if time.Now().Before(cabOrderRecoveryDeadline) {
				shouldSend = true
				recoveredCabOrders = recoverLocalCabOrders(id, allCabOrders, allElevators)
			}
			dataMutex.Unlock()

			if shouldSend {
				select {
				case recoveredCabOrdersChan <- recoveredCabOrders:
				default:
				}
			}

		case newCompletedOrder := <-completedOrderChan:
			if newCompletedOrder.Button == elevio.BT_Cab {
				continue
			}

			fmt.Printf("completedOrderChan case: floor: %d, button: %d\n", newCompletedOrder.Floor, newCompletedOrder.Button)

			shouldSend := false
			var elevatorSnapshot elev_struct.Elevator
			var hallOrdersSnapshot HallOrders
			var cabOrdersSnapshot types.CabOrders

			dataMutex.Lock()
			if orders, ok := allHallOrders[id]; ok {
				orders[newCompletedOrder.Floor][newCompletedOrder.Button] = COMPLETED
				allHallOrders[id] = orders

				elevatorSnapshot = allElevators[id]
				hallOrdersSnapshot = allHallOrders[id]
				cabOrdersSnapshot = maps.Clone(allCabOrders)
				shouldSend = true
			}

			dataMutex.Unlock()

			if shouldSend {
				network.NetworkSend(elevatorSnapshot, hallOrdersSnapshot, cabOrdersSnapshot, time.Now().Before(cabOrderRecoveryDeadline))
			}

		case orderToConfirm := <-orderConfirmedChan:
			fmt.Printf("ConfirmedChan case, floor: %d, button: %d\n", orderToConfirm.Floor, orderToConfirm.Button)
			dataMutex.Lock()

			localOrders := allHallOrders[id]
			localOrders[orderToConfirm.Floor][orderToConfirm.Button] = CONFIRMED
			allHallOrders[id] = localOrders

			hallOrdersForId, err := ReassignOrders(id, allHallOrders[id], availableElevators, allElevators)
			if err != nil {
				dataMutex.Unlock()
				continue
			}

			allHallOrders[id] = setOrdersToAssigned(hallOrdersForId, allHallOrders[id])

			elevatorSnapshot := allElevators[id]
			hallOrdersSnapshot := allHallOrders[id]
			cabOrdersSnapshot := maps.Clone(allCabOrders)
			dataMutex.Unlock()

			hallLightChan <- elev_struct.LightEvent{Floor: orderToConfirm.Floor, Button: elevio.ButtonType(orderToConfirm.Button), On: true}

			reassignedHallOrdersChan <- hallOrdersForId

			network.NetworkSend(elevatorSnapshot, hallOrdersSnapshot, cabOrdersSnapshot, time.Now().Before(cabOrderRecoveryDeadline))

		case orderToReset := <-orderResetChan:
			fmt.Printf("ResetChan case, floor: %d, button: %d\n", orderToReset.Floor, orderToReset.Button)
			dataMutex.Lock()
			localOrders := allHallOrders[id]
			localOrders[orderToReset.Floor][orderToReset.Button] = NONE
			allHallOrders[id] = localOrders
			dataMutex.Unlock()

			hallLightChan <- elev_struct.LightEvent{Floor: orderToReset.Floor, Button: elevio.ButtonType(orderToReset.Button), On: false}

		case hallLightEvent := <-hallLightChan:
			fmt.Printf("hallLightChan case, floor: %d, button: %d, on: %v\n", hallLightEvent.Floor, hallLightEvent.Button, hallLightEvent.On)
			elevio.SetButtonLamp(hallLightEvent.Button, hallLightEvent.Floor, hallLightEvent.On)

		case <-networkResendTicker.C:
			dataMutex.RLock()
			elevatorSnapshot := allElevators[id]
			hallOrdersSnapshot := allHallOrders[id]
			cabOrdersSnapshot := maps.Clone(allCabOrders)
			recoveringCabOrders := time.Now().Before(cabOrderRecoveryDeadline)
			dataMutex.RUnlock()

			network.NetworkSend(elevatorSnapshot, hallOrdersSnapshot, cabOrdersSnapshot, recoveringCabOrders)
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
	go resetHallOrders(orderResetChan, &allHallOrders, &availableElevators, &dataMutex)

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
