package config

import "time"

const (
	N_FLOORS    int = 4
	N_BUTTONS   int = 3
	N_ELEVATORS int = 3

	DOOR_OPEN_TIME time.Duration = 3 * time.Second
	STALL_TIME     time.Duration = 5 * time.Second

	BUFFER_SIZE int = 10
)
