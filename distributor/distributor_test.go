package distributor

import (
	"encoding/json"
	"testing"

	"heislab-sanntid/elevator/elev_struct"
	"heislab-sanntid/orders"
)

type distributorInputForTest struct {
	HallRequests [][]bool                        `json:"hallRequests"`
	States       map[string]elev_struct.Elevator `json:"states"`
}

func TestFormatInputForDistributor(t *testing.T) {
	hallOrders := orders.InitHallOrders()
	hallOrders[0][0] = orders.NEW
	hallOrders[0][1] = orders.COMPLETED
	hallOrders[1][0] = orders.CONFIRMED
	hallOrders[1][1] = orders.NONE

	allElevatorStates := make([]elev_struct.Elevator, 3)
	allElevatorStates[0] = elev_struct.ElevatorInit(0)
	allElevatorStates[0].Floor = 1
	allElevatorStates[2] = elev_struct.ElevatorInit(2)
	allElevatorStates[2].Floor = 3

	activeElevators := []int{2, -1, 0, 99}

	out := formatInputForDistributor(&hallOrders, activeElevators, allElevatorStates)
	data, ok := out.([]byte)
	
	if !ok {
		t.Fatalf("formatInputForDistributor returned %T, want []byte", out)
	}

	var got distributorInputForTest
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}

	if len(got.HallRequests) != len(hallOrders) {
		t.Fatalf("hallRequests floors = %d, want %d", len(got.HallRequests), len(hallOrders))
	}
	if len(got.HallRequests[0]) != len(hallOrders[0]) {
		t.Fatalf("hallRequests buttons = %d, want %d", len(got.HallRequests[0]), len(hallOrders[0]))
	}

	if !got.HallRequests[0][0] {
		t.Fatalf("hallRequests[0][0] = false, want true for NEW")
	}
	if got.HallRequests[0][1] {
		t.Fatalf("hallRequests[0][1] = true, want false for COMPLETED")
	}
	if !got.HallRequests[1][0] {
		t.Fatalf("hallRequests[1][0] = false, want true for CONFIRMED")
	}
	if got.HallRequests[1][1] {
		t.Fatalf("hallRequests[1][1] = true, want false for NONE")
	}

	if len(got.States) != 2 {
		t.Fatalf("states size = %d, want 2", len(got.States))
	}

	state0, ok := got.States["id_0"]
	if !ok {
		t.Fatalf("missing state for id_0")
	}
	if state0.Floor != 1 {
		t.Fatalf("id_0 floor = %d, want 1", state0.Floor)
	}

	state2, ok := got.States["id_2"]
	if !ok {
		t.Fatalf("missing state for id_2")
	}
	if state2.Floor != 3 {
		t.Fatalf("id_2 floor = %d, want 3", state2.Floor)
	}

	if _, exists := got.States["id_-1"]; exists {
		t.Fatalf("unexpected state for id_-1")
	}
	if _, exists := got.States["id_99"]; exists {
		t.Fatalf("unexpected state for id_99")
	}
}
//TODO: run this test and understand the code