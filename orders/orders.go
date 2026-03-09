package orders

import (
	elevio "Driver-go"
	network "Network-go"
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

type ElevstateHallorderPair struct {
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
	availableElevators *[config.N_ELEVATORS]bool) {

	for {
		//sleep?
		currentAllHallOrders := *allHallOrders
		currentavailableElevators := *availableElevators

		for floor := 0; floor < config.N_FLOORS; floor++ {
			for btn := 0; btn < config.N_BUTTONS-1; btn++ {

				shouldConfirm := true

				for elev := 0; elev < config.N_ELEVATORS; elev++ {
					if currentAllHallOrders[elev][floor][btn] != NEW && currentavailableElevators[elev] {
						shouldConfirm = false
						break
					}
				}

				if shouldConfirm {
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
	availableElevators *[config.N_ELEVATORS]bool) {

	for {
		//sleep?
		currentAllHallOrders := *allHallOrders
		currentavailableElevators := *availableElevators

		for floor := 0; floor < config.N_FLOORS; floor++ {
			for btn := 0; btn < config.N_BUTTONS-1; btn++ {

				shouldReset := true

				for elev := 0; elev < config.N_ELEVATORS; elev++ {
					if currentAllHallOrders[elev][floor][btn] != COMPLETED && currentavailableElevators[elev] {
						shouldReset = false
						break
					}
				}

				if shouldReset {
					order_reset_chan <- elevio.ButtonEvent{Floor: floor, Button: elevio.ButtonType(btn)}
				}
			}
		}
	}
}

func RunOrderManager(
	id string,
	local_elevator_chan <-chan elev_struct.Elevator,
	assign_order_chan chan<- elevio.ButtonEvent,
	completed_order_chan <-chan elevio.ButtonEvent,
	clear_local_hall_orders_chan chan<- bool,
	// networkTx chan<- ElevstateHallorderPair,
	// networkRx <-chan ElevstateHallorderPair,
	//peer_update_chan <-chan peers.PeerUpdate
) {
	// Ikke accsess direkte til variabler fra network. DB

	var allHallOrders HallOrdersAllElevators = InitHallOrdersAllElevators()
	var allElevatorStates = InitAllElevatorStates()
	var availableElevators = make(map[string]bool)
	availableElevators[id] = true //TODO: sett andre heiser? (Bør kanskje heller oppdatere denne basert på peer updates. DB)

	order_confirmed_chan := make(chan elevio.ButtonEvent, config.BUFFER_SIZE)
	order_reset_chan := make(chan elevio.ButtonEvent, config.BUFFER_SIZE)
	hall_light_chan := make(chan elev_struct.LightEvent, config.BUFFER_SIZE)

	go ConfirmHallOrders(order_confirmed_chan, hall_light_chan, &allHallOrders, &availableElevators)
	go ResetHallOrders(order_reset_chan, hall_light_chan, &allHallOrders, &availableElevators)

	for {
		select {
		// Unsure if peers returns IDs. Will be tested. DB
		case peerUpdate := <-network.Peers():
			for _, peer := range peerUpdate.Peers {
				if peer != id {
					availableElevators[peer] = true
				}
			}

			// Needed? DB
			availableElevators[peerUpdate.New] = true

			for _, lostPeer := range peerUpdate.Lost {
				delete(availableElevators, lostPeer)
			}

		case localElevator := <-local_elevator_chan:
			allElevatorStates[localElevator.ID] = localElevator
			//TODO: er stuck? har ny ordre?
			networkTx <- ElevstateHallorderPair{elevatorState: localElevator, hallOrders: allHallOrders[Id]}

			//network.NetworkSend()

		case stateOrderPair := <-networkRx:
			allElevatorStates[stateOrderPair.elevatorState.ID] = stateOrderPair.elevatorState
			allHallOrders[stateOrderPair.elevatorState.ID] = stateOrderPair.hallOrders
			//TODO: er stuck? har ny ordre?

		case newCompletedOrder := <-completed_order_chan:
			allHallOrders[Id][newCompletedOrder.Floor][newCompletedOrder.Button] = COMPLETED
			networkTx <- ElevstateHallorderPair{elevatorState: allElevatorStates[Id], hallOrders: allHallOrders[Id]}

		case orderToConfirm := <-order_confirmed_chan:
			for elev := 0; elev < config.N_ELEVATORS; elev++ {
				if availableElevators[elev] {
					allHallOrders[elev][orderToConfirm.Floor][orderToConfirm.Button] = CONFIRMED
				}
			}
			hall_light_chan <- elev_struct.LightEvent{Floor: orderToConfirm.Floor, Button: elevio.ButtonType(orderToConfirm.Button), On: true}
			//TODO: kjør distribution
			//sett til assigned
			//hvis assigned til oss, send til elevator

		case orderToReset := <-order_reset_chan:
			for elev := 0; elev < config.N_ELEVATORS; elev++ {
				if availableElevators[elev] {
					allHallOrders[elev][orderToReset.Floor][orderToReset.Button] = NONE
				}
			}
			hall_light_chan <- elev_struct.LightEvent{Floor: orderToReset.Floor, Button: elevio.ButtonType(orderToReset.Button), On: false}

		case hallLightEvent := <-hall_light_chan:
			elevio.SetButtonLamp(hallLightEvent.Button, hallLightEvent.Floor, hallLightEvent.On)
		}
	}
}
