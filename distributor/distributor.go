package distributor

import (
	"encoding/json"
	"fmt"
	"heislab-sanntid/elevator/elev_struct"
	"heislab-sanntid/orders"
	"os/exec"
)

func CallDistributor(input any) ([]byte, error) {
	jsonData, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	cmd := exec.Command("./distributor/hall_request_assigner.exe")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe error: %w", err)
	}

	go func() {
		_, _ = stdin.Write(append(jsonData, '\n'))
		_ = stdin.Close()
	}()

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("distributor error: %w\nOutput: %s", err, string(output))
	}

	return output, nil
}



func FormatInputForDistributor(hallOrders *orders.HallOrders, activeElevators []int, allElevatorStates []elev_struct.Elevator) any {
	/* input format for distributor looks like this:
	{
    "hallRequests" : 
        [[Boolean, Boolean], ...],
    "states" : 
        {
            "id_1" : {
                "state"     : < "idle" | "moving" | "doorOpen" >
                "floor"         : NonNegativeInteger
                "direction"     : < "up" | "down" | "stop" >
                "cabRequests"   : [Boolean, ...]
            },
            "id_2" : {...}
        }
	}
	*/
	type DistributorInput struct {
    	HallRequests [][]bool                 `json:"hallRequests"`
    	States       map[string]elev_struct.Elevator `json:"states"`
	}	
	hallRequests := make([][]bool, len(hallOrders))
	for floor := 0; floor < len(hallOrders); floor++ {
		hallRequests[floor] = make([]bool, len(hallOrders[floor]))
		for btn := 0; btn < len(hallOrders[floor]); btn++ {
			orderState := hallOrders[floor][btn]
			hallRequests[floor][btn] = orderState != orders.NONE && orderState != orders.COMPLETED 
		}
	}
	
	states := make(map[string]elev_struct.Elevator, len(activeElevators))

	for _, id := range activeElevators {
		if id < 0 || id >= len(allElevatorStates) {
			continue
		}
		state := allElevatorStates[id]
		states[fmt.Sprintf("id_%d", id)] = state
	}

	fullInput := DistributorInput{
		HallRequests: hallRequests,
		States:       states,
	}
	
	data, _ := json.MarshalIndent(fullInput, "", "  ")
	fmt.Println(string(data))
	return data
}