package distributor

import (
	elevio "Driver-go"
	"bytes"
	"encoding/json"
	"fmt"
	"heislab-sanntid/elevator/elev_struct"
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

func stateToString(elevatorState elev_struct.Elevator) string {
	stateStrings := map[elev_struct.State]string{
		elev_struct.Idle: "idle",
		elev_struct.Moving: "moving",
		elev_struct.DoorOpen: "doorOpen",
	}
	return stateStrings[elevatorState.State]
}
func directionToString(elevatorState elev_struct.Elevator) string {
	directionStrings := map[elevio.MotorDirection]string{
		elevio.MD_Up: "up",
		elevio.MD_Down: "down",
		elevio.MD_Stop: "stop",
	}
	return directionStrings[elevatorState.Dir]
}
func FormatInputForDistributor(hallRequests [][]bool, availableElevators map[string]bool, allElevatorStates map[string]elev_struct.Elevator) any {
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
		State string `json:"state"`
		Floor int    `json:"floor"`
		Direction string `json:"direction"`
		CabRequests []bool `json:"cabRequests"`
	}
	type DistributorInput struct {
    	HallRequests [][]bool                 `json:"hallRequests"`
    	States       map[string]StateInputForDistributor `json:"states"`
	}	

	states := make(map[string]StateInputForDistributor, len(availableElevators))

	for id, isActive := range availableElevators {
		var cabRequests []bool
		if !isActive {
			continue
		}
		for floor := 0; floor < elev_struct.N_FLOORS; floor++ {
			cabRequests = append(cabRequests, allElevatorStates[id].Requests[floor][elevio.BT_Cab])
		}
		states[id] = StateInputForDistributor{
			State: stateToString(allElevatorStates[id]),
			Floor: allElevatorStates[id].Floor,
			Direction: directionToString(allElevatorStates[id]),
			CabRequests: cabRequests,
		}
	}

	fullInput := DistributorInput{
		HallRequests: hallRequests,
		States:       states,
	}
	
	debugData, _ := json.MarshalIndent(fullInput, "", "  ")
	fmt.Println(string(debugData))

	data, _ := json.Marshal(fullInput)
	return data
}

func ParseDistributorOutput(output []byte) (map[string][][]bool, error) {
    var assignments map[string][][]bool
    if err := json.Unmarshal(output, &assignments); err != nil {
        return nil, err
    }
    return assignments, nil
}

func HallOrdersForID(output []byte, id string) ([][]bool, bool, error) {
    assignments, err := ParseDistributorOutput(output)
    if err != nil {
        return nil, false, err
    }
    hallOrders, ok := assignments[id]
	
    return hallOrders, ok, nil
}