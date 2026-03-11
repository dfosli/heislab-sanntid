package distributor

import (
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

func FormatInputForDistributor(hallRequests [][]bool, activeElevators []int, allElevatorStates []elev_struct.Elevator) any { //TODO endre activeElev. list til map
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
	}	//TODO: direction og state til string funksjon

	states := make(map[string]elev_struct.Elevator, len(activeElevators))

	for _, id := range activeElevators {
		state := allElevatorStates[id]
		states[id] = state //!feil her
	}

	fullInput := DistributorInput{
		HallRequests: hallRequests,
		States:       states,
	}
	
	data, _ := json.MarshalIndent(fullInput, "", "  ")
	fmt.Println(string(data))
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