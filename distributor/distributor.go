package distributor

import (
	"encoding/json"
	"fmt"
	"heislab-sanntid/elevator/elev_struct"
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


//TODO: finish this function
func formatInputForDistributor(hallOrders *HallOrders, activeElevators []int, allElevatorStates []elev_struct.Elevator) any {
	/* input format for distributor looks like this:
	{
    "hallRequests" : 
        [[Boolean, Boolean], ...],
    "states" : 
        {
            "id_1" : {
                "behaviour"     : < "idle" | "moving" | "doorOpen" >
                "floor"         : NonNegativeInteger
                "direction"     : < "up" | "down" | "stop" >
                "cabRequests"   : [Boolean, ...]
            },
            "id_2" : {...}
        }
	}
	*/
	temporaryStruct = Struct{
		//Fill in here
	}
	data, _ := json.MarshalIndent(p, "", "  ")
	fmt.Println(string(data))
	
}