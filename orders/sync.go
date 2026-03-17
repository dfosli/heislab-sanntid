package orders

import (
	elevio "Driver-go"
	"fmt"
	"heislab-sanntid/config"
	"heislab-sanntid/distributor"
	"heislab-sanntid/elevator/elev_struct"
	types "heislab-sanntid/types"
)

// func updateLocalHallOrders(hallOrders *HallOrders, floor int, btn int, orderState OrderState) bool {
// 	(*hallOrders)[floor][btn] = orderState
// 	return true //!TODO add some errorhandling here
// }
// Nuked this func since it is useless. DB.

func UpdateLocalHallOrders(localHallOrders AllHallOrders, remoteHallOrders AllHallOrders) AllHallOrders {
	for floor := 0; floor < config.N_FLOORS; floor++ {
		for btn := 0; btn < config.N_BUTTONS-1; btn++ {
			if localHallOrders[floor][btn] == COMPLETED && remoteHallOrders[floor][btn] == NONE {
				localHallOrders[floor][btn] = NONE
				//hallLightChan <- elev_struct.LightEvent{Floor: floor, Button: elevio.ButtonType(btn), On: false}
			}
			if localHallOrders[floor][btn] < remoteHallOrders[floor][btn] {
				if localHallOrders[floor][btn] == NONE && remoteHallOrders[floor][btn] == COMPLETED {
					continue
				} //this if will always update to higher order states, unless it is an update from NONE to COMPLETED
				//! should we trigger distribution if jump past barrier?
				(localHallOrders)[floor][btn] = remoteHallOrders[floor][btn]
			}
		}
	}

	return localHallOrders
}

func AddNewLocalOrder(hallOrders AllHallOrders, requests elev_struct.Requests) AllHallOrders {
	for floor := 0; floor < config.N_FLOORS; floor++ {
		for btn := 0; btn < config.N_BUTTONS-1; btn++ {
			if requests[floor][btn] && hallOrders[floor][btn] == NONE {
				hallOrders[floor][btn] = NEW
			}
		}
	}
	return hallOrders
}

func hallOrdersToBoolMatrix(hallOrders AllHallOrders) [config.N_FLOORS][config.N_BUTTONS - 1]bool {
	hallRequests := [config.N_FLOORS][config.N_BUTTONS - 1]bool{}
	for floor := 0; floor < len(hallOrders); floor++ {
		for btn := 0; btn < len(hallOrders[floor]); btn++ {
			state := hallOrders[floor][btn]
			hallRequests[floor][btn] = state != NONE && state != COMPLETED
		}
	}
	return hallRequests
}

func ReassignOrders(id string, hallOrders AllHallOrders, availableElevators map[string]bool, allElevators types.AllElevators) ([config.N_FLOORS][config.N_BUTTONS - 1]bool, error) {
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
			continue //!This can theoretically cause caborder loss if a recovering elevator receives a caborder Very quickly, before it receives its caborders from other elevators.
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

		// localElevator.Requests[floor][elevio.BT_Cab] = true
		recoveredOrders[floor] = true
	}

	// allElevators[localID] = localElevator
	allCabOrders[localID] = elev_struct.GetCabOrders(localElevator)

	return recoveredOrders
}
