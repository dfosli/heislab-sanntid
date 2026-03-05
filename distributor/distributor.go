package distributor

import (
	"encoding/json"
	"fmt"
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