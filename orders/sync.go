package orders

import (
	"elevio"
	"heislab-sanntid/config"
	"heislab-sanntid/distributor"
	"heislab-sanntid/elevator/elev_struct"
	"heislab-sanntid/network"
	"heislab-sanntid/types"

	"fmt"
)

func syncLocalHallOrders(localHallOrders HallOrders, remoteHallOrders HallOrders) HallOrders {
	for floor := 0; floor < config.N_FLOORS; floor++ {
		for btn := 0; btn < config.N_BUTTONS-1; btn++ {
			if localHallOrders[floor][btn] < remoteHallOrders[floor][btn] {
				if localHallOrders[floor][btn] == NONE && remoteHallOrders[floor][btn] == COMPLETED {
					continue
				}
				if localHallOrders[floor][btn] == NONE && remoteHallOrders[floor][btn] >= CONFIRMED && remoteHallOrders[floor][btn] < COMPLETED {
					localHallOrders[floor][btn] = NEW
					continue
				}
				if localHallOrders[floor][btn] == NEW && remoteHallOrders[floor][btn] >= CONFIRMED && remoteHallOrders[floor][btn] < COMPLETED {
					continue
				}
				localHallOrders[floor][btn] = remoteHallOrders[floor][btn]
			}
		}
	}
	return localHallOrders
}

func addNewLocalOrder(hallOrders HallOrders, requests elev_struct.Requests) HallOrders {
	for floor := 0; floor < config.N_FLOORS; floor++ {
		for btn := 0; btn < config.N_BUTTONS-1; btn++ {
			if requests[floor][btn] && hallOrders[floor][btn] == NONE {
				hallOrders[floor][btn] = NEW
			}
		}
	}
	return hallOrders
}

func reassignOrders(id string, hallOrders HallOrders, availableElevators map[string]bool, allElevators types.AllElevators) ([config.N_FLOORS][config.N_BUTTONS - 1]bool, error) {
	hallRequests := hallOrdersToBoolMatrix(hallOrders)
	formattedOrders, err := distributor.FormatInputForDistributor(hallRequests, availableElevators, allElevators)
	if err != nil {
		return [config.N_FLOORS][config.N_BUTTONS - 1]bool{}, fmt.Errorf("format input for distributor: %w", err)
	}

	allReassignedHallOrders, err := distributor.CallDistributor(formattedOrders)
	if err != nil {
		return [config.N_FLOORS][config.N_BUTTONS - 1]bool{}, fmt.Errorf("call distributor: %w", err)
	}

	hallOrderForID, err := distributor.HallOrdersForID(allReassignedHallOrders, id)
	if err != nil {
		return [config.N_FLOORS][config.N_BUTTONS - 1]bool{}, fmt.Errorf("parse distributor output: %w", err)
	}

	return hallOrderForID, nil
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

func hallOrdersToBoolMatrix(hallOrders HallOrders) [config.N_FLOORS][config.N_BUTTONS - 1]bool {
	hallRequests := [config.N_FLOORS][config.N_BUTTONS - 1]bool{}
	for floor := 0; floor < len(hallOrders); floor++ {
		for btn := 0; btn < len(hallOrders[floor]); btn++ {
			state := hallOrders[floor][btn]
			hallRequests[floor][btn] = state == CONFIRMED || state == ASSIGNED
		}
	}
	return hallRequests
}

func hasCabOrders(cabOrders [config.N_FLOORS]bool) bool {
	for floor := 0; floor < config.N_FLOORS; floor++ {
		if cabOrders[floor] {
			return true
		}
	}
	return false
}

func mergeCabOrders(allCabOrders types.AllCabOrders, remoteCabOrders types.AllCabOrders, remoteID string, remoteRecovering bool) {
	for id, cabOrders := range remoteCabOrders {
		if remoteRecovering && id == remoteID && !hasCabOrders(cabOrders) {
			continue
		}
		allCabOrders[id] = cabOrders
	}
}

func recoverLocalCabOrders(localID string, allCabOrders types.AllCabOrders, allElevators types.AllElevators) [config.N_FLOORS]bool {
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

	allElevators[localID] = localElevator
	allCabOrders[localID] = elev_struct.GetCabOrders(localElevator)

	return recoveredOrders
}

func handleElevatorUnavailable(localID string, unavailableID string, allHallOrders AllHallOrders) {
	allHallOrders[unavailableID] = setAllOrders(NONE)
	if localID != unavailableID {
		allHallOrders[localID] = rollbackHallOrders(allHallOrders[localID])
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

	allHallOrders[localID] = addNewLocalOrder(allHallOrders[localID], localElevator.Requests)
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
	allHallOrders[localID] = syncLocalHallOrders(allHallOrders[localID], remoteElevatorMsg.HallOrders)
}
