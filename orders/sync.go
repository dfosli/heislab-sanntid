package orders

import (
	"heislab-sanntid/config"
	"heislab-sanntid/distributor"
	"heislab-sanntid/elevator/elev_struct"
)
func updateLocalHallOrders(hallOrders *HallOrders, floor int, btn int, orderState OrderState) bool {
	(*hallOrders)[floor][btn] = orderState	
	return true
}

func UpdateLocalHallOrdersIfPossible(localHallOrders HallOrders, allElevatorHallOrders HallOrdersAllElevators) HallOrders {
	for _, externalHallOrders := range allElevatorHallOrders {
		for floor := 0; floor < config.N_FLOORS; floor++ {
			for btn := 0; btn < config.N_BUTTONS-1; btn++ {
				if localHallOrders[floor][btn] == COMPLETED && externalHallOrders[floor][btn] == NONE {
					updateLocalHallOrders(&localHallOrders, floor, btn, NONE)
				}
				if localHallOrders[floor][btn] < externalHallOrders[floor][btn] {
					if localHallOrders[floor][btn] == NONE && externalHallOrders[floor][btn] == COMPLETED {
						continue
					}//this if will always update to higher order states, unless it is an update from NONE to COMPLETED
					updateLocalHallOrders(&localHallOrders, floor, btn, externalHallOrders[floor][btn])			
				}
			}
		}
	}
	return localHallOrders
}


func AddNewOrder(hallOrders *HallOrders, floor int, btn int) bool {
	if (*hallOrders)[floor][btn] == NONE {
		(*hallOrders)[floor][btn] = NEW
		return true
	}
	return false
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

func ReassignedOrders(hallOrders HallOrders, availableElevators map[string]bool, allElevatorStates map[string]elev_struct.Elevator, id string) [][]bool {	
	hallRequests := hallOrdersToBoolMatrix(hallOrders)
	formattedOrders := distributor.FormatInputForDistributor(hallRequests, availableElevators, allElevatorStates)
	allReassignedHallOrders, err := distributor.CallDistributor(formattedOrders)
	if err != nil {
		return nil
	}
	hallOrderForID, _, err := distributor.HallOrdersForID(allReassignedHallOrders, id)
	if err != nil {
		return nil
	}
	_ = id
	
	return hallOrderForID
}