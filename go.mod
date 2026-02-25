module heislab-sanntid

go 1.16

require (
	Driver-go v0.0.0
	Network-go v0.0.0
)

replace Driver-go => ./elevator/elevio

replace Network-go => ./network
