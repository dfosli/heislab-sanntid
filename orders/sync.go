package orders

import (
	"heislab-sanntid/config"
	"heislab-sanntid/distributor"
)

// functions

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

//UnassignOrder: assigns orders all orders that are assigned to an elevator not in the ActiveElevators list back to NEW, so they can be assigned to active elevators instead.
//will bi triggered by a change in the ActiveElevators list, and will return true if local hallOrders was updated, false if not.
func reassignedOrders(hallOrders *HallOrders, activeElevators []int) HallOrders {
	//TODO: format input so that it matches what distributor expects,
	//  and removes inactive elevators from hallOrders object, so distributor only assigns to active elevators.
	distributor.CallDistributor(hallOrders) 
}