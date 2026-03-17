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

func UpdateLocalHallOrders(localHallOrders HallOrders, remoteHallOrders HallOrders, hallLightChan chan<- elev_struct.LightEvent) HallOrders {
	for floor := 0; floor < config.N_FLOORS; floor++ {
		for btn := 0; btn < config.N_BUTTONS-1; btn++ {
			oldState := localHallOrders[floor][btn]
			newState := oldState

			if newState == COMPLETED && remoteHallOrders[floor][btn] == NONE {
				newState = NONE
			}
			if newState < remoteHallOrders[floor][btn] {
				if newState == NONE && remoteHallOrders[floor][btn] == COMPLETED {
					continue
				}
				newState = remoteHallOrders[floor][btn]
			}

			if newState == oldState {
				continue
			}

			localHallOrders[floor][btn] = newState

			newActive := newState != NONE && newState != COMPLETED
			hallLightChan <- elev_struct.LightEvent{Floor: floor, Button: elevio.ButtonType(btn), On: newActive}
		}
	}

	return localHallOrders
}

func AddNewLocalOrder(hallOrders HallOrders, requests elev_struct.Requests) HallOrders {
	for floor := 0; floor < config.N_FLOORS; floor++ {
		for btn := 0; btn < config.N_BUTTONS-1; btn++ {
			if requests[floor][btn] && hallOrders[floor][btn] == NONE {
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
			hallRequests[floor][btn] = state != NONE && state != COMPLETED
		}
	}
	return hallRequests
}

func ReassignOrders(id string, hallOrders HallOrders, availableElevators map[string]bool, allElevators types.AllElevators) ([config.N_FLOORS][config.N_BUTTONS - 1]bool, error) {
	hallRequests := hallOrdersToBoolMatrix(hallOrders)
	fmt.Println(hallRequests)
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
