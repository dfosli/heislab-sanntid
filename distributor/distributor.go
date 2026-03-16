package distributor

import (
	elevio "Driver-go"
	"bytes"
	"encoding/json"
	"fmt"
	"heislab-sanntid/config"
	"heislab-sanntid/elevator/elev_struct"
	"heislab-sanntid/types"
	"os/exec"
)

func CallDistributor(input any) ([]byte, error) {
	var jsonData []byte
	switch v := input.(type) {
	case []byte:
		jsonData = v
	case json.RawMessage:
		jsonData = v
	default:
		var err error
		jsonData, err = json.Marshal(input)
		if err != nil {
			return nil, fmt.Errorf("marshal error: %w", err)
		}
	}

	cmd := exec.Command("./distributor/hall_request_assigner.exe")
	cmd.Stdin = bytes.NewReader(append(jsonData, '\n'))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("distributor error: %w\nOutput: %s", err, string(output))
	}

	return output, nil
}

func stateToString(elevator types.Elevator) string {
	stateStrings := map[elev_struct.State]string{
		elev_struct.Idle:     "idle",
		elev_struct.Moving:   "moving",
		elev_struct.DoorOpen: "doorOpen",
	}
	return stateStrings[elevator.State]
}
func directionToString(elevator types.Elevator) string {
	directionStrings := map[elevio.MotorDirection]string{
		elevio.MD_Up:   "up",
		elevio.MD_Down: "down",
		elevio.MD_Stop: "stop",
	}
	return directionStrings[elevator.Dir]
}

func FormatInputForDistributor(hallRequests [config.N_FLOORS][config.N_BUTTONS - 1]bool, availableElevators map[string]bool, allElevators types.AllElevators) ([]byte, error) {

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
	type StateInputForDistributor struct {
		State       string                `json:"state"`
		Floor       int                   `json:"floor"`
		Direction   string                `json:"direction"`
		CabRequests [config.N_FLOORS]bool `json:"cabRequests"`
	}
	type DistributorInput struct {
		HallRequests [config.N_FLOORS][config.N_BUTTONS - 1]bool `json:"hallRequests"`
		States       map[string]StateInputForDistributor         `json:"states"`
	}

	states := make(map[string]StateInputForDistributor, len(availableElevators))

	for id, isActive := range availableElevators {
		var cabRequests [config.N_FLOORS]bool
		if !isActive {
			continue
		}
		for floor := 0; floor < elev_struct.N_FLOORS; floor++ {
			cabRequests[floor] = allElevators[id].Requests[floor][elevio.BT_Cab]
		}
		floor := allElevators[id].Floor
		if floor < 0 {
			floor = 0
		}
		states[id] = StateInputForDistributor{
			State:       stateToString(allElevators[id]),
			Floor:       floor,
			Direction:   directionToString(allElevators[id]),
			CabRequests: cabRequests,
		}
	}
	if len(states) == 0 {
		return nil, fmt.Errorf("no active elevators")
	}

	fullInput := DistributorInput{
		HallRequests: hallRequests,
		States:       states,
	}

	// debugData, _ := json.MarshalIndent(fullInput, "", "  ")
	// fmt.Println(string(debugData))

	data, err := json.Marshal(fullInput)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}


	return data, nil
}

func ParseDistributorOutput(output []byte) (map[string][config.N_FLOORS][config.N_BUTTONS - 1]bool, error) {
	var assignments map[string][config.N_FLOORS][config.N_BUTTONS - 1]bool
	if err := json.Unmarshal(output, &assignments); err != nil {
		return nil, fmt.Errorf("unmarshal distributor output: %w", err)
	}
	if assignments == nil {
		return nil, fmt.Errorf("distributor output was empty")
	}
	return assignments, nil
}

func HallOrdersForID(output []byte, id string) ([config.N_FLOORS][config.N_BUTTONS - 1]bool, error) {
	assignments, err := ParseDistributorOutput(output)
	if err != nil {
		return [config.N_FLOORS][config.N_BUTTONS - 1]bool{}, err
	}

	hallOrders, exists := assignments[id]
	if !exists {
		return [config.N_FLOORS][config.N_BUTTONS - 1]bool{}, fmt.Errorf("missing assignments for id %s", id)
	}
	return hallOrders, nil
}
