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
type AllCabOrders = types.AllCabOrders
type AllElevators = types.AllElevators
type HallOrders = types.HallOrders

const (
	NONE      = types.NONE
	NEW       = types.NEW
	CONFIRMED = types.CONFIRMED
	ASSIGNED  = types.ASSIGNED
	COMPLETED = types.COMPLETED
)

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
	orderConfirmedCh chan<- elevio.ButtonEvent,
	allHallOrders AllHallOrders,
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

		localOrders, localOk := allHallOrders[localID]

		for floor := 0; floor < config.N_FLOORS; floor++ {
			for btn := 0; btn < config.N_BUTTONS-1; btn++ {
				allAtLeastNew := true
				for elevID, isAvailable := range availableElevators {
					if !isAvailable && elevID != localID {
						continue
					}
					orders, ok := allHallOrders[elevID]
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
			orderConfirmedCh <- event
		}
	}
}

func resetHallOrders(
	localID string,
	orderResetCh chan<- elevio.ButtonEvent,
	allHallOrders AllHallOrders,
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

		for floor := 0; floor < config.N_FLOORS; floor++ {
			for btn := 0; btn < config.N_BUTTONS-1; btn++ {
				allCompletedOrNone := true
				atLeastOneCompleted := false
				for elevID, isAvailable := range availableElevators {
					if !isAvailable && elevID != localID {
						continue
					}
					orders, ok := allHallOrders[elevID]
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
			orderResetCh <- event
		}
	}
}

func applyLocalElevatorUpdate(
	localID string,
	localElevator elev_struct.Elevator,
	availableElevators map[string]bool,
	allHallOrders AllHallOrders,
	allElevators types.AllElevators,
	allCabOrders types.AllCabOrders) {

	allElevators[localID] = localElevator
	allCabOrders[localID] = elev_struct.GetCabOrders(localElevator)

	wasAvailable := availableElevators[localElevator.ID]

	if localElevator.Stuck && availableElevators[localElevator.ID] {
		availableElevators[localElevator.ID] = false
		handleElevatorUnavailable(localID, localElevator.ID, allHallOrders)
	} else if !localElevator.Stuck && !availableElevators[localElevator.ID] {
		availableElevators[localElevator.ID] = true
	}

	isAvailable := availableElevators[localElevator.ID]
	if wasAvailable != isAvailable {
		network.SetPeerTxEnable(isAvailable)
	}

	allHallOrders[localID] = AddNewLocalOrder(allHallOrders[localID], localElevator.Requests)
}

func applyRemoteElevatorUpdate(
	localID string,
	remoteElevatorMsg network.NetworkMsg,
	availableElevators map[string]bool,
	allHallOrders AllHallOrders,
	allElevators types.AllElevators,
	allCabOrders types.AllCabOrders) {

	mergeCabOrders(allCabOrders, remoteElevatorMsg.AllCabOrders, remoteElevatorMsg.Elevator.ID, remoteElevatorMsg.CabOrdersRecovering)

	if !availableElevators[remoteElevatorMsg.Elevator.ID] && !remoteElevatorMsg.Elevator.Stuck {
		return
	}

	allElevators[remoteElevatorMsg.Elevator.ID] = remoteElevatorMsg.Elevator
	allHallOrders[remoteElevatorMsg.Elevator.ID] = remoteElevatorMsg.HallOrders
	allHallOrders[localID] = UpdateLocalHallOrders(allHallOrders[localID], remoteElevatorMsg.HallOrders)
}

func runOrderManager(
	id string,
	localElevatorCh <-chan types.Elevator,
	completedOrderCh <-chan elevio.ButtonEvent,
	reassignedLocalHallOrdersCh chan<- [config.N_FLOORS][config.N_BUTTONS - 1]bool,
	recoveredCabOrdersCh chan<- [config.N_FLOORS]bool,
	orderConfirmedCh <-chan elevio.ButtonEvent,
	orderResetCh <-chan elevio.ButtonEvent,
	allHallOrders AllHallOrders,
	allElevators AllElevators,
	allCabOrders AllCabOrders,
	availableElevators map[string]bool,
	dataMutex *sync.RWMutex) {

	networkResendTicker := time.NewTicker(100 * time.Millisecond)
	defer networkResendTicker.Stop()
	cabOrderRecoveryDeadline := time.Now().Add(5 * time.Second)

	sendNetworkUpdate := func() {
		dataMutex.RLock()
		elevator := allElevators[id]
		hallOrders := allHallOrders[id]
		cabOrders := maps.Clone(allCabOrders)
		recovering := time.Now().Before(cabOrderRecoveryDeadline)
		dataMutex.RUnlock()
		network.NetworkSend(elevator, hallOrders, cabOrders, recovering)
	}

	for {
		select {
		case peerUpdate := <-network.Peers():
			fmt.Printf("peer update case, peers: %v\n", peerUpdate.Peers)
			dataMutex.Lock()
			for _, peer := range peerUpdate.Peers {
				if peer == id {
					continue
				}
				if _, ok := availableElevators[peer]; !ok {
					availableElevators[peer] = true
					allHallOrders[peer] = setAllOrders(NONE)
					allElevators[peer] = elev_struct.ElevatorInit(peer)
				} else if !availableElevators[peer] {
					availableElevators[peer] = true
				}
			}

			for _, lostPeer := range peerUpdate.Lost {
				if lostPeer == id {
					continue
				}
				if availableElevators[lostPeer] {
					availableElevators[lostPeer] = false
					if availableElevators[id] {
						handleElevatorUnavailable(id, lostPeer, allHallOrders)
					}
				}
			}
			dataMutex.Unlock()
			sendNetworkUpdate()

		case localElevator := <-localElevatorCh:
			dataMutex.Lock()
			applyLocalElevatorUpdate(id, localElevator, availableElevators, allHallOrders, allElevators, allCabOrders)
			dataMutex.Unlock()
			sendNetworkUpdate()

		case remoteElevatorMsg := <-network.NetworkRxChan():
			if remoteElevatorMsg.Elevator.ID == id {
				continue
			}
			dataMutex.Lock()
			applyRemoteElevatorUpdate(id, remoteElevatorMsg, availableElevators, allHallOrders, allElevators, allCabOrders)
			var recoveredCabOrders [config.N_FLOORS]bool
			recovering := time.Now().Before(cabOrderRecoveryDeadline)
			if recovering {
				recoveredCabOrders = recoverLocalCabOrders(id, allCabOrders, allElevators)
			}
			dataMutex.Unlock()
			if recovering {
				select {
				case recoveredCabOrdersCh <- recoveredCabOrders:
				default:
				}
			}

		case newCompletedOrder := <-completedOrderCh:
			if newCompletedOrder.Button == elevio.BT_Cab {
				continue
			}
			fmt.Printf("completedOrderCh case: floor: %d, button: %d\n", newCompletedOrder.Floor, newCompletedOrder.Button)
			dataMutex.Lock()
			orders := allHallOrders[id]
			orders[newCompletedOrder.Floor][newCompletedOrder.Button] = COMPLETED
			allHallOrders[id] = orders
			dataMutex.Unlock()
			sendNetworkUpdate()

		case orderToConfirm := <-orderConfirmedCh:
			dataMutex.Lock()
			localOrders := allHallOrders[id]
			if localOrders[orderToConfirm.Floor][orderToConfirm.Button] != NEW {
				dataMutex.Unlock()
				continue
			}

			fmt.Printf("ConfirmedCh case, floor: %d, button: %d\n", orderToConfirm.Floor, orderToConfirm.Button)
			localOrders[orderToConfirm.Floor][orderToConfirm.Button] = CONFIRMED
			allHallOrders[id] = localOrders

			hallOrdersForId := [config.N_FLOORS][config.N_BUTTONS - 1]bool{}
			if availableElevators[id] {
				var err error = nil
				hallOrdersForId, err = ReassignOrders(id, allHallOrders[id], availableElevators, allElevators)
				if err != nil {
					fmt.Printf("Error reassigning orders: %v\n", err)
					localOrders[orderToConfirm.Floor][orderToConfirm.Button] = NEW
					allHallOrders[id] = localOrders
					dataMutex.Unlock()
					continue
				}
			}

			allHallOrders[id] = setOrdersToAssigned(hallOrdersForId, allHallOrders[id])

			dataMutex.Unlock()
			elevio.SetButtonLamp(orderToConfirm.Button, orderToConfirm.Floor, true)
			reassignedLocalHallOrdersCh <- hallOrdersForId
			sendNetworkUpdate()

		case orderToReset := <-orderResetCh:
			dataMutex.Lock()
			localOrders := allHallOrders[id]
			if localOrders[orderToReset.Floor][orderToReset.Button] != COMPLETED {
				dataMutex.Unlock()
				continue
			}

			fmt.Printf("ResetCh case, floor: %d, button: %d\n", orderToReset.Floor, orderToReset.Button)

			localOrders[orderToReset.Floor][orderToReset.Button] = NONE
			allHallOrders[id] = localOrders
			dataMutex.Unlock()

			elevio.SetButtonLamp(orderToReset.Button, orderToReset.Floor, false)
			sendNetworkUpdate()

		case <-networkResendTicker.C:
			sendNetworkUpdate()
		}
	}
}

func OrdersInit(id string,
	reassignLocalHallOrdersCh chan<- [config.N_FLOORS][config.N_BUTTONS - 1]bool,
	recoveredCabOrdersCh chan<- [config.N_FLOORS]bool,
	completedOrderCh <-chan elevio.ButtonEvent,
	localElevatorCh <-chan types.Elevator) {

	allHallOrders := make(AllHallOrders)
	allHallOrders[id] = setAllOrders(NONE)

	allElevators := make(AllElevators)
	allElevators[id] = elev_struct.ElevatorInit(id)

	allCabOrders := make(AllCabOrders)
	allCabOrders[id] = elev_struct.GetCabOrders(allElevators[id])

	availableElevators := map[string]bool{id: true}

	var dataMutex sync.RWMutex

	orderConfirmedCh := make(chan elevio.ButtonEvent, config.BUFFER_SIZE)
	orderResetCh := make(chan elevio.ButtonEvent, config.BUFFER_SIZE)

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

	go runOrderManager(
		id,
		localElevatorCh,
		completedOrderCh,
		reassignLocalHallOrdersCh,
		recoveredCabOrdersCh,
		orderConfirmedCh,
		orderResetCh,
		allHallOrders,
		allElevators,
		allCabOrders,
		availableElevators,
		&dataMutex)

}
