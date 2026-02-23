package fsm

import (
	elevio "Driver-go"
	"heislab-sanntid/config"
	"heislab-sanntid/elevator/elev_struct"
	"heislab-sanntid/elevator/requests"
	"time"
)

const (
	DOOR_OPEN_TIME = config.DOOR_OPEN_TIME
	STUCK_TIME     = config.STUCK_TIME
)

func SetAllLights(e elev_struct.Elevator) {
	for f := 0; f < config.N_FLOORS; f++ {
		for btn := 0; btn < config.N_BUTTONS; btn++ {
			elevio.SetButtonLamp(elevio.ButtonType(btn), f, e.Requests[f][btn])
		}
	}
}

func OnInitBetweenFloors(e elev_struct.Elevator) elev_struct.Elevator {
	localElevator := e
	elevio.SetMotorDirection(elevio.MD_Down)
	localElevator.Dir = elevio.MD_Down
	localElevator.State = elev_struct.Moving
	return localElevator
}

func OnRequestButtonPress(
	e elev_struct.Elevator,
	btnFloor int,
	btnType elevio.ButtonType,
	doorTimer *time.Timer,
	stuckTimer *time.Timer,
	clear_order_chan chan<- elevio.ButtonEvent) elev_struct.Elevator {

	localElevator := e

	switch localElevator.State {
	case elev_struct.DoorOpen:
		if requests.RequestsShouldClearImmediately(localElevator, btnFloor, btnType) {
			clear_order_chan <- elevio.ButtonEvent{Floor: btnFloor, Button: btnType}
			doorTimer.Reset(DOOR_OPEN_TIME)
			stuckTimer.Reset(STUCK_TIME)
		} else {
			localElevator.Requests[btnFloor][btnType] = true
		}

	case elev_struct.Moving:
		localElevator.Requests[btnFloor][btnType] = true

	case elev_struct.Idle:
		localElevator.Requests[btnFloor][btnType] = true
		var nextAction elev_struct.DirStatePair = requests.RequestsChooseDirection(localElevator)
		localElevator.Dir = nextAction.Dir
		localElevator.State = nextAction.State
		switch nextAction.State {
		case elev_struct.DoorOpen:
			elevio.SetDoorOpenLamp(true)
			doorTimer.Reset(DOOR_OPEN_TIME)
			stuckTimer.Reset(STUCK_TIME)
			localElevator = requests.RequestsClearAtCurrentFloor(localElevator, clear_order_chan)

		case elev_struct.Moving:
			elevio.SetMotorDirection(localElevator.Dir)
			stuckTimer.Reset(STUCK_TIME)

		case elev_struct.Idle:
		}
	}

	elev_struct.SetCabLights(localElevator)

	return localElevator
}

func OnFloorArrival(e elev_struct.Elevator, newFloor int, doorTimer *time.Timer, clear_order_chan chan<- elevio.ButtonEvent) elev_struct.Elevator {
	localElevator := e
	localElevator.Floor = newFloor
	elevio.SetFloorIndicator(localElevator.Floor)

	switch localElevator.State {
	case elev_struct.Moving:
		if requests.RequestsShouldStop(localElevator) {
			elevio.SetMotorDirection(elevio.MD_Stop)
			elevio.SetDoorOpenLamp(true)
			localElevator.State = elev_struct.DoorOpen
			localElevator = requests.RequestsClearAtCurrentFloor(localElevator, clear_order_chan)
			doorTimer.Reset(DOOR_OPEN_TIME)
			elev_struct.SetCabLights(localElevator)
		}
	}

	return localElevator
}

func OnDoorTimeout(e elev_struct.Elevator, doorTimer *time.Timer, clear_order_chan chan<- elevio.ButtonEvent) elev_struct.Elevator {
	localElevator := e

	switch localElevator.State {
	case elev_struct.DoorOpen:
		nextAction := requests.RequestsChooseDirection(localElevator)
		localElevator.Dir = nextAction.Dir
		localElevator.State = nextAction.State

		switch localElevator.State {
		case elev_struct.DoorOpen:
			doorTimer.Reset(DOOR_OPEN_TIME)
			localElevator = requests.RequestsClearAtCurrentFloor(localElevator, clear_order_chan)
			elev_struct.SetCabLights(localElevator)

		case elev_struct.Moving:
			fallthrough

		case elev_struct.Idle:
			elevio.SetDoorOpenLamp(false)
			elevio.SetMotorDirection(localElevator.Dir)
		}
	}
	return localElevator
}

func OnObstruction(e elev_struct.Elevator, doorTimer *time.Timer) {
	localElevator := e

	switch localElevator.State {
	case elev_struct.DoorOpen:
		doorTimer.Reset(DOOR_OPEN_TIME)
	}
}
