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
type AllHallOrders = types.AllHallOrders

const (
	NONE      = types.NONE
	NEW       = types.NEW
	CONFIRMED = types.CONFIRMED
	ASSIGNED  = types.ASSIGNED
	COMPLETED = types.COMPLETED
)

type HallOrdersAllElevators map[string]AllHallOrders

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

func initCabOrdersAllElevators(id string) types.AllCabOrders {
	allCabOrders := make(types.AllCabOrders)
	allCabOrders[id] = elev_struct.GetCabOrders(elev_struct.ElevatorInit(id))
	return allCabOrders
}

func setAllOrders(orderState OrderState) AllHallOrders {
	var hallOrders AllHallOrders
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
	allHallOrders HallOrdersAllElevators,
	availableElevators map[string]bool,
	dataMutex *sync.RWMutex) {

	var confirmAlreadySent [config.N_FLOORS][config.N_BUTTONS - 1]bool

	for {
		time.Sleep(10 * time.Millisecond)
		var ordersToConfirm []elevio.ButtonEvent

		dataMutex.RLock()
		hasAvailable := false
		for _, isAvailable := range availableElevators {
			if isAvailable {
				hasAvailable = true
				break
			}
		}
		if !hasAvailable {
			dataMutex.RUnlock()
			continue
		}

		localOrders, ok := allHallOrders[localID]
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
	allHallOrders HallOrdersAllElevators,
	availableElevators map[string]bool,
	dataMutex *sync.RWMutex) {

	var resetAlreadySent [config.N_FLOORS][config.N_BUTTONS - 1]bool

	for {
		time.Sleep(10 * time.Millisecond)
		var ordersToReset []elevio.ButtonEvent

		dataMutex.RLock()
		hasAvailable := false
		for _, isAvailable := range availableElevators {
			if isAvailable {
				hasAvailable = true
				break
			}
		}
		if !hasAvailable {
			dataMutex.RUnlock()
			continue
		}

		localOrders, ok := allHallOrders[localID]
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

func rollbackHallOrders(hallOrders AllHallOrders) AllHallOrders {
	for floor := 0; floor < config.N_FLOORS; floor++ {
		for btn := 0; btn < config.N_BUTTONS-1; btn++ {
			if hallOrders[floor][btn] == ASSIGNED || hallOrders[floor][btn] == CONFIRMED {
				hallOrders[floor][btn] = NEW
			}
		}
	}
	return hallOrders
}

func setOrdersToAssigned(assignedOrders [config.N_FLOORS][config.N_BUTTONS - 1]bool, hallOrders AllHallOrders) AllHallOrders {
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
	allCabOrders types.AllCabOrders,
	availableElevators map[string]bool) (types.Elevator, AllHallOrders) {

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
	allCabOrders types.AllCabOrders) {

	allElevators[remoteElevatorMsg.Elevator.ID] = remoteElevatorMsg.Elevator
	allHallOrders[remoteElevatorMsg.Elevator.ID] = remoteElevatorMsg.HallOrders

	mergeCabOrders(allCabOrders, remoteElevatorMsg.AllCabOrders, remoteElevatorMsg.Elevator.ID, remoteElevatorMsg.CabOrdersRecovering)
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
	allCabOrders types.AllCabOrders,
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

						for id, isAvailable := range availableElevators {
							if isAvailable {
								if orders, ok := allHallOrders[id]; ok {
									orders = rollbackHallOrders(orders)
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

			fmt.Printf("completedOrderChan case: floor: %d, button: %d\n", newCompletedOrder.Floor, newCompletedOrder.Button)

			shouldSend := false
			var elevatorSnapshot elev_struct.Elevator
			var hallOrdersSnapshot AllHallOrders
			var cabOrdersSnapshot types.AllCabOrders

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

			for elevId, isAvailable := range availableElevators {
				if isAvailable {
					if orders, ok := allHallOrders[elevId]; ok {
						orders[orderToConfirm.Floor][orderToConfirm.Button] = CONFIRMED
						allHallOrders[elevId] = orders
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
			cabOrdersSnapshot := maps.Clone(allCabOrders)
			dataMutex.Unlock()

			hallLightChan <- elev_struct.LightEvent{Floor: orderToConfirm.Floor, Button: elevio.ButtonType(orderToConfirm.Button), On: true}

			reassignedHallOrdersChan <- hallOrdersForId

			network.NetworkSend(elevatorSnapshot, hallOrdersSnapshot, cabOrdersSnapshot, time.Now().Before(cabOrderRecoveryDeadline))

		case orderToReset := <-orderResetChan:
			fmt.Printf("ResetChan case, floor: %d, button: %d\n", orderToReset.Floor, orderToReset.Button)
			dataMutex.Lock()
			for elevId, isAvailable := range availableElevators {
				if isAvailable {
					if orders, ok := allHallOrders[elevId]; ok {
						orders[orderToReset.Floor][orderToReset.Button] = NONE
						allHallOrders[elevId] = orders
					}
				}
			}
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
	reassignLocalHallOrdersCh chan<- [config.N_FLOORS][config.N_BUTTONS - 1]bool,
	recoveredCabOrdersCh chan<- [config.N_FLOORS]bool,
	completedOrderCh <-chan elevio.ButtonEvent,
	localElevatorCh <-chan types.Elevator) {

	allHallOrders := initHallOrdersAllElevators(id)
	allElevators := initAllElevators(id)
	allCabOrders := initCabOrdersAllElevators(id)

	availableElevators := map[string]bool{id: true}
	var dataMutex sync.RWMutex

	orderConfirmedCh := make(chan elevio.ButtonEvent, config.BUFFER_SIZE)
	orderResetCh := make(chan elevio.ButtonEvent, config.BUFFER_SIZE)
	hallLightCh := make(chan elev_struct.LightEvent, config.BUFFER_SIZE)

	go confirmHallOrders(
		id,
		orderConfirmedCh,
		allHallOrders,
		availableElevators,
		&dataMutex)

	go resetHallOrders(
		id,
		orderResetCh,
		allHallOrders,
		availableElevators,
		&dataMutex)

	go RunOrderManager(
		id,
		localElevatorCh,
		completedOrderCh,
		reassignLocalHallOrdersCh,
		recoveredCabOrdersCh,
		hallLightCh,
		orderConfirmedCh,
		orderResetCh,
		allHallOrders,
		allElevators,
		allCabOrders,
		availableElevators,
		&dataMutex)

}
