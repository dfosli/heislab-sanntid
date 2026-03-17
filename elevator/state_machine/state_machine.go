package state_machine

import (
	elevio "Driver-go"
	"heislab-sanntid/config"
	"heislab-sanntid/elevator/elev_struct"
	"heislab-sanntid/elevator/requests"
	"time"
)

const (
	DOOR_OPEN_TIME = config.DOOR_OPEN_TIME
	STALL_TIME     = config.STALL_TIME
)

func OnRequestButtonPress(
	elev elev_struct.Elevator,
	btnFloor int,
	btnType elevio.ButtonType,
	doorTimer *time.Timer,
	stuckTimer *time.Timer,
	completedOrderCh chan<- elevio.ButtonEvent) elev_struct.Elevator {

	switch elev.State {
	case elev_struct.DoorOpen:
		if requests.RequestsShouldClearImmediately(elev, btnFloor, btnType) {
			completedOrderCh <- elevio.ButtonEvent{Floor: btnFloor, Button: btnType}
			doorTimer.Reset(DOOR_OPEN_TIME)
			stuckTimer.Reset(STALL_TIME)
		} else {
			elev.Requests[btnFloor][btnType] = true
		}

	case elev_struct.Moving:
		elev.Requests[btnFloor][btnType] = true

	case elev_struct.Idle:
		elev.Requests[btnFloor][btnType] = true
		var nextAction elev_struct.DirStatePair = requests.RequestsChooseDirection(elev)
		elev.Dir = nextAction.Dir
		elev.State = nextAction.State
		switch nextAction.State {
		case elev_struct.DoorOpen:
			elevio.SetDoorOpenLamp(true)
			doorTimer.Reset(DOOR_OPEN_TIME)
			stuckTimer.Reset(STALL_TIME)
			elev = requests.RequestsClearAtCurrentFloor(elev, completedOrderCh)

		case elev_struct.Moving:
			elevio.SetMotorDirection(elev.Dir)
			stuckTimer.Reset(STALL_TIME)

		case elev_struct.Idle:
		}
	}

	elev_struct.SetCabLights(elev)

	return elev
}

func OnFloorArrival(elev elev_struct.Elevator, newFloor int, doorTimer *time.Timer, completedOrderCh chan<- elevio.ButtonEvent) elev_struct.Elevator {
	elev.Floor = newFloor
	elevio.SetFloorIndicator(elev.Floor)

	switch elev.State {
	case elev_struct.Moving:
		if requests.RequestsShouldStop(elev) {
			elevio.SetMotorDirection(elevio.MD_Stop)
			elevio.SetDoorOpenLamp(true)
			elev.State = elev_struct.DoorOpen
			elev = requests.RequestsClearAtCurrentFloor(elev, completedOrderCh)
			doorTimer.Reset(DOOR_OPEN_TIME)
			elev_struct.SetCabLights(elev)
		}
	}

	return elev
}

func OnDoorTimeout(elev elev_struct.Elevator, doorTimer *time.Timer, completedOrderCh chan<- elevio.ButtonEvent) elev_struct.Elevator {
	switch elev.State {
	case elev_struct.DoorOpen:
		nextAction := requests.RequestsChooseDirection(elev)
		elev.Dir = nextAction.Dir
		elev.State = nextAction.State

		switch elev.State {
		case elev_struct.DoorOpen:
			doorTimer.Reset(DOOR_OPEN_TIME)
			elev = requests.RequestsClearAtCurrentFloor(elev, completedOrderCh)
			elev_struct.SetCabLights(elev)

		case elev_struct.Moving:
			fallthrough

		case elev_struct.Idle:
			elevio.SetDoorOpenLamp(false)
			elevio.SetMotorDirection(elev.Dir)
		}
	}
	return elev
}

func OnObstruction(elev elev_struct.Elevator, doorTimer *time.Timer) {
	switch elev.State {
	case elev_struct.DoorOpen:
		doorTimer.Reset(DOOR_OPEN_TIME)
	}
}
