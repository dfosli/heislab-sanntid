package main

import (
	"fmt"
	"heislab-sanntid/distributor"
)

func main() {
	input := map[string]any{
		"hallRequests": [][]bool{{false, false}, {true, false}, {false, false}, {false, true}},
		"states": map[string]any{
			"one": map[string]any{
				"behaviour":   "moving",
				"floor":       2,
				"direction":   "up",
				"cabRequests": []bool{false, false, true, true},
			},
			"two": map[string]any{
				"behaviour":   "idle",
				"floor":       0,
				"direction":   "stop",
				"cabRequests": []bool{false, false, false, false},
			},
		},
	}

	output, err := distributor.CallDistributor(input)
	if err != nil {
		fmt.Printf("CallDistributor error: %v\n", err)
		return
	}

	fmt.Printf("CallDistributor output: %s\n", string(output))
}
