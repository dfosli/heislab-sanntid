package orders

import (
	"fmt"
	"heislab-sanntid/config"
	"heislab-sanntid/distributor"
	"heislab-sanntid/elevator/elev_struct"
)

func updateLocalHallOrders(hallOrders *HallOrders, floor int, btn int, orderState OrderState) bool {
	(*hallOrders)[floor][btn] = orderState
	return true //!TODO add some errorhandling here
}

func UpdateLocalHallOrdersIfPossible(localHallOrders HallOrders, remoteHallOrders HallOrders) HallOrders {
	for floor := 0; floor < config.N_FLOORS; floor++ {
		for btn := 0; btn < config.N_BUTTONS-1; btn++ {
			if localHallOrders[floor][btn] == COMPLETED && remoteHallOrders[floor][btn] == NONE {
				updateLocalHallOrders(&localHallOrders, floor, btn, NONE)
			}
			if localHallOrders[floor][btn] < remoteHallOrders[floor][btn] {
				if localHallOrders[floor][btn] == NONE && remoteHallOrders[floor][btn] == COMPLETED {
					continue
				} //this if will always update to higher order states, unless it is an update from NONE to COMPLETED
				updateLocalHallOrders(&localHallOrders, floor, btn, remoteHallOrders[floor][btn])
			}
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

func hallOrdersToBoolMatrix(hallOrders HallOrders) [][]bool {
	hallRequests := make([][]bool, len(hallOrders))
	for floor := 0; floor < len(hallOrders); floor++ {
		hallRequests[floor] = make([]bool, len(hallOrders[floor]))
		for btn := 0; btn < len(hallOrders[floor]); btn++ {
			state := hallOrders[floor][btn]
			hallRequests[floor][btn] = state != NONE && state != COMPLETED
		}
	}
	return hallRequests
}

func ReassignOrders(id string, hallOrders HallOrders, availableElevators map[string]bool, allElevatorStates map[string]elev_struct.Elevator) ([][]bool, error) {

	hallRequests := hallOrdersToBoolMatrix(hallOrders)
	formattedOrders := distributor.FormatInputForDistributor(hallRequests, availableElevators, allElevatorStates)
	if formattedOrders == nil {
		return nil, fmt.Errorf("no available elevators to assign orders")
	}
	allReassignedHallOrders, err := distributor.CallDistributor(formattedOrders)

	if err != nil {
		return nil, fmt.Errorf("call distributor: %w", err)
	}

	hallOrderForID, ok, err := distributor.HallOrdersForID(allReassignedHallOrders, id)
	if err != nil {
		return nil, fmt.Errorf("parse distributor output: %w", err)
	}
	if !ok {
		return nil, fmt.Errorf("missing assignments for id %s", id)
	}
	return hallOrderForID, nil
}
