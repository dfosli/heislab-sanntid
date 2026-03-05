package orders

import (
	elevio "Driver-go"
	"Network-go/peers"
	"heislab-sanntid/config"
	"heislab-sanntid/elevator/elev_struct"
)

type OrderState int

const (
	NONE OrderState = iota
	NEW
	CONFIRMED
	ASSIGNED
	COMPLETED
)

type HallOrders [config.N_FLOORS][config.N_BUTTONS - 1]OrderState

type HallOrdersAllElevators [config.N_ELEVATORS]HallOrders

type AllElevatorStates [config.N_ELEVATORS]elev_struct.Elevator

type NetworkMessage struct {
	elevatorState elev_struct.Elevator
	hallOrders    HallOrders
}

func InitHallOrders() HallOrders {
	var hallOrders HallOrders
	for floor := 0; floor < config.N_FLOORS; floor++ {
		for btn := 0; btn < config.N_BUTTONS; btn++ {
			hallOrders[floor][btn] = NONE
		}
	}
	return hallOrders
}

func InitHallOrdersAllElevators() HallOrdersAllElevators {
	var allHallOrders HallOrdersAllElevators
	for elev := 0; elev < config.N_ELEVATORS; elev++ {
		allHallOrders[elev] = InitHallOrders()
	}
	return allHallOrders
}

func InitAllElevatorStates() AllElevatorStates {
	var allElevatorStates AllElevatorStates
	for elev := 0; elev < config.N_ELEVATORS; elev++ {
		allElevatorStates[elev] = elev_struct.ElevatorInit(elev)
	}
	return allElevatorStates
}

func ConfirmHallOrders(
	order_confirmed_chan chan<- elevio.ButtonEvent,
	hall_light_chan chan<- elev_struct.LightEvent,
	allHallOrders *HallOrdersAllElevators,
	availibleElevators *[config.N_ELEVATORS]bool) {

	for {
		//sleep?
		currentAllHallOrders := *allHallOrders
		currentAvailibleElevators := *availibleElevators

		for floor := 0; floor < config.N_FLOORS; floor++ {
			for btn := 0; btn < config.N_BUTTONS-1; btn++ {

				shouldConfirm := true

				for elev := 0; elev < config.N_ELEVATORS; elev++ {
					if !currentAvailibleElevators[elev] {
						continue
					}
					if currentAllHallOrders[elev][floor][btn] != NEW {
						shouldConfirm = false
						break
					}
				}

				if shouldConfirm {
					for elev := 0; elev < config.N_ELEVATORS; elev++ {
						if currentAvailibleElevators[elev] {
							currentAllHallOrders[elev][floor][btn] = CONFIRMED
						}
					}
					hall_light_chan <- elev_struct.LightEvent{Floor: floor, Button: elevio.ButtonType(btn), On: true}
					order_confirmed_chan <- elevio.ButtonEvent{Floor: floor, Button: elevio.ButtonType(btn)}
				}
			}
		}
	}
}

func ResetHallOrders(
	order_reset_chan chan<- elevio.ButtonEvent,
	hall_light_chan chan<- elev_struct.LightEvent,
	allHallOrders *HallOrdersAllElevators,
	availibleElevators *[config.N_ELEVATORS]bool) {

	for {
		//sleep?
		currentAllHallOrders := *allHallOrders
		currentAvailibleElevators := *availibleElevators

		for floor := 0; floor < config.N_FLOORS; floor++ {
			for btn := 0; btn < config.N_BUTTONS-1; btn++ {

				shouldReset := true

				for elev := 0; elev < config.N_ELEVATORS; elev++ {
					if !currentAvailibleElevators[elev] {
						continue
					}
					if currentAllHallOrders[elev][floor][btn] != COMPLETED {
						shouldReset = false
						break
					}
				}

				if shouldReset {
					for elev := 0; elev < config.N_ELEVATORS; elev++ {
						if currentAvailibleElevators[elev] {
							currentAllHallOrders[elev][floor][btn] = CONFIRMED
						}
					}
					hall_light_chan <- elev_struct.LightEvent{Floor: floor, Button: elevio.ButtonType(btn), On: false}
					order_reset_chan <- elevio.ButtonEvent{Floor: floor, Button: elevio.ButtonType(btn)}
				}
			}
		}
	}
}

func RunOrderManager(
	Id int,
	local_elevator_chan <-chan elev_struct.Elevator,
	assigned_orders_chan chan<- elevio.ButtonEvent,
	completed_order_chan <-chan elevio.ButtonEvent,
	clear_local_hall_orders_chan chan<- bool,
	network_Tx chan<- NetworkMessage,
	network_Rx <-chan NetworkMessage,
	peer_update_chan <-chan peers.PeerUpdate) {
	//mer parametre sikkert

	var allHallOrders HallOrdersAllElevators = InitHallOrdersAllElevators()
	var allElevatorStates = InitAllElevatorStates()
	var availibleElevators [config.N_ELEVATORS]bool
	availibleElevators[Id] = true //TODO: sett andre heiser?

	order_confirmed_chan := make(chan elevio.ButtonEvent, config.BUFFER_SIZE)
	order_reset_chan := make(chan elevio.ButtonEvent, config.BUFFER_SIZE)
	hall_light_chan := make(chan elev_struct.LightEvent, config.BUFFER_SIZE)

	go ConfirmHallOrders(order_confirmed_chan, hall_light_chan, &allHallOrders, &availibleElevators)
	go ResetHallOrders(order_reset_chan, hall_light_chan, &allHallOrders, &availibleElevators)

	for {
		select {
		case peerUpdate := <-peer_update_chan:
		//TODO: oppdater availibleElevators

		case newLocalElevator := <-local_elevator_chan:
			allElevatorStates[newLocalElevator.ID] = newLocalElevator
			//TODO: stuck? ny ordre?
			network_Tx <- NetworkMessage{elevatorState: newLocalElevator, hallOrders: allHallOrders[Id]}

		case newNetworkMessage := <-network_Rx:
			allElevatorStates[newNetworkMessage.elevatorState.ID] = newNetworkMessage.elevatorState
			allHallOrders[newNetworkMessage.elevatorState.ID] = newNetworkMessage.hallOrders
			//TODO: stuck? ny ordre?

		case newCompletedOrder := <-completed_order_chan:
			allHallOrders[Id][newCompletedOrder.Floor][newCompletedOrder.Button] = COMPLETED
			network_Tx <- NetworkMessage{elevatorState: allElevatorStates[Id], hallOrders: allHallOrders[Id]}

		case <-order_confirmed_chan:
			//TODO: kjør distribution
			//sett til assigned
			//hvis assigned til oss, send til elevator

		case newResetOrder := <-order_reset_chan:
			allHallOrders[Id][newResetOrder.Floor][newResetOrder.Button] = NONE

		case hallLightEvent := <-hall_light_chan:
			elevio.SetButtonLamp(hallLightEvent.Button, hallLightEvent.Floor, hallLightEvent.On)
		}
	}
}
