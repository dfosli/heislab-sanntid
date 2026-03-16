package main

import (
	"fmt"
	"heislab-sanntid/elevator/elev_struct"
	"heislab-sanntid/orders"
	"strings"
)

func main() {
	// EDIT HERE: choose the elevator id you want assignments for.
	id := "id_1"

	// EDIT HERE: choose hall-order states to send into ReassignedOrders.
	// Buttons are [0]=hall up, [1]=hall down.
	//hallOrders[floor][btn]
	hallOrders := orders.HallOrders{}
	hallOrders[0][0] = orders.CONFIRMED
	hallOrders[1][1] = orders.ASSIGNED
	hallOrders[2][0] = orders.NEW
	hallOrders[1][0] = orders.CONFIRMED
	hallOrders[2][1] = orders.ASSIGNED
	hallOrders[3][0] = orders.NEW

	// EDIT HERE: choose which elevators are currently available.
	availableElevators := map[string]bool{
		"id_1": true,
		"id_2": true,
		"id_3": false,
	}

	// EDIT HERE: set elevator states for the same keys used above.
	allElevatorStates := map[string]elev_struct.Elevator{
		"id_1": makeElevator("id_1", 0, elev_struct.Idle),
		"id_2": makeElevator("id_2", 2, elev_struct.Moving),
		"id_3": makeElevator("id_3", 3, elev_struct.DoorOpen),
	}

	printScenario(id, hallOrders, availableElevators, allElevatorStates)

	//reassigned, err := orders.ReassignOrders(id, hallOrders, availableElevators, allElevatorStates)
	// if err != nil {
	// 	fmt.Printf("ReassignOrders failed: %v\n", err)
	// 	return
	// }

	//fmt.Printf("\nReturned hall assignments for id=%q:\n%s\n", id, formatBoolMatrix(reassigned))
}

func makeElevator(id string, floor int, state elev_struct.State) elev_struct.Elevator {
	e := elev_struct.ElevatorInit(id)
	e.Floor = floor
	e.State = state
	return e
}

func printScenario(id string, hallOrders orders.HallOrders, availableElevators map[string]bool, allElevatorStates map[string]elev_struct.Elevator) {
	fmt.Println("Manual ReassignOrders test")
	fmt.Printf("Requesting assignments for id=%q\n", id)
	fmt.Printf("Input hallOrders (state matrix):\n%s\n", formatStateMatrix(hallOrders))
	fmt.Printf("Available elevators: %v\n", availableElevators)
	fmt.Printf("Elevator states:\n")
	for elevID, st := range allElevatorStates {
		fmt.Printf("  - %s: floor=%d state=%s\n", elevID, st.Floor, elevatorStateString(st.State))
	}
}

func formatStateMatrix(hallOrders orders.HallOrders) string {
	var b strings.Builder
	b.WriteString("floor: [up, down]\n")
	for floor := 0; floor < len(hallOrders); floor++ {
		b.WriteString(fmt.Sprintf("  %d: [%s, %s]\n",
			floor,
			orderStateString(hallOrders[floor][0]),
			orderStateString(hallOrders[floor][1]),
		))
	}
	return b.String()
}

func formatBoolMatrix(hallOrders [][]bool) string {
	var b strings.Builder
	b.WriteString("floor: [up, down]\n")
	for floor := 0; floor < len(hallOrders); floor++ {
		if len(hallOrders[floor]) < 2 {
			b.WriteString(fmt.Sprintf("  %d: %v\n", floor, hallOrders[floor]))
			continue
		}
		b.WriteString(fmt.Sprintf("  %d: [%t, %t]\n", floor, hallOrders[floor][0], hallOrders[floor][1]))
	}
	return b.String()
}

func elevatorStateString(s elev_struct.State) string {
	switch s {
	case elev_struct.Idle:
		return "Idle"
	case elev_struct.Moving:
		return "Moving"
	case elev_struct.DoorOpen:
		return "DoorOpen"
	default:
		return fmt.Sprintf("Unknown(%d)", s)
	}
}

func orderStateString(s orders.OrderState) string {
	switch s {
	case orders.NONE:
		return "NONE"
	case orders.NEW:
		return "NEW"
	case orders.CONFIRMED:
		return "CONFIRMED"
	case orders.ASSIGNED:
		return "ASSIGNED"
	case orders.COMPLETED:
		return "COMPLETED"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", s)
	}
}