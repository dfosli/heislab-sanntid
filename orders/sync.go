package orders

import (
	"heislab-sanntid/config"
	"heislab-sanntid/distributor"
	"heislab-sanntid/elevator/elev_struct"
)

func shouldUpdateLocalHallOrders(localHallOrders *HallOrders, allElevatorHallOrders *HallOrdersAllElevators) bool {
	for elev := 0; elev < config.N_ELEVATORS; elev++ {
		externalHallOrders := allElevatorHallOrders[elev]
		for floor := 0; floor < config.N_FLOORS; floor++ {
			for btn := 0; btn < config.N_BUTTONS; btn++ {
				if localHallOrders[floor][btn] == COMPLETED && externalHallOrders[floor][btn] == NONE {
					return true
				}
				if localHallOrders[floor][btn] < externalHallOrders[floor][btn] {
					if localHallOrders[floor][btn] == NONE && externalHallOrders[floor][btn] == COMPLETED {
						return false
					}
					return true
				}
			}
		}
	}
	return false
}
func updateLocalHallOrders(hallOrders *HallOrders, floor int, btn int, orderState OrderState) bool {
	if hallOrders[floor][btn] < orderState {
		hallOrders[floor][btn] = orderState
		return true
	}
	return false
}
func addNewOrder(hallOrders *HallOrders, floor int, btn int) bool {
	if hallOrders[floor][btn] == NONE {
		hallOrders[floor][btn] = NEW
		return true
	}
	return false
}

func hallOrdersToBoolMatrix(hallOrders *HallOrders) [][]bool {
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


func reassignedOrders(hallOrders *HallOrders, activeElevators []int,allElevatorStates []elev_struct.Elevator, elevator_id int) HallOrders {
	hallRequests := hallOrdersToBoolMatrix(hallOrders)
	formattedOrders := distributor.FormatInputForDistributor(hallRequests, activeElevators, allElevatorStates)
	output, err := distributor.CallDistributor(formattedOrders)
	if err != nil {
		return *hallOrders
	}

	_ = output
	_ = elevator_id
	//TODO: output must be parsed from json back to HallOrders struct
	return *hallOrders
}