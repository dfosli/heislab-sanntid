package distributor

import (
	elevio "Driver-go"
	"bytes"
	"encoding/json"
	"fmt"
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
<<<<<<< HEAD
func FormatInputForDistributor(hallRequests [][]bool, availableElevators map[string]bool, allElevatorStates map[string]elev_struct.Elevator) ([]byte, error) {
=======
func FormatInputForDistributor(hallRequests [][]bool, availableElevators map[string]bool, allElevators types.AllElevators) any {
>>>>>>> David4
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
		State       string `json:"state"`
		Floor       int    `json:"floor"`
		Direction   string `json:"direction"`
		CabRequests []bool `json:"cabRequests"`
	}
	type DistributorInput struct {
		HallRequests [][]bool                            `json:"hallRequests"`
		States       map[string]StateInputForDistributor `json:"states"`
	}

	states := make(map[string]StateInputForDistributor, len(availableElevators))

	for id, isActive := range availableElevators {
		var cabRequests []bool
		if !isActive {
			continue
		}
		for floor := 0; floor < elev_struct.N_FLOORS; floor++ {
			cabRequests = append(cabRequests, allElevators[id].Requests[floor][elevio.BT_Cab])
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

	debugData, _ := json.MarshalIndent(fullInput, "", "  ")
	fmt.Println(string(debugData))

	data, err := json.Marshal(fullInput)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	return data, nil
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
