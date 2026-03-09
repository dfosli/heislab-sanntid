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


func reassignedOrders(hallOrders *HallOrders, activeElevators []int,allElevatorStates []elev_struct.Elevator, elevator_id int) HallOrders {
	formattedOrders := distributor.FormatInputForDistributor(hallOrders, activeElevators, allElevatorStates)
	output, err := distributor.CallDistributor(formattedOrders)
	if err != nil {
		return *hallOrders
	}

	//TODO: output must be parsed from json back to HallOrders struct
	return output
}