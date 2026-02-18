package fsm

import (
	"heislab-sanntid/elevator/elev_struct"
	"heislab-sanntid/elevator/elevio"
	"heislab-sanntid/elevator/requests"
)

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
	btnType elevio.ButtonType) elev_struct.Elevator {

	localElevator := e

	switch localElevator.State {
	case elev_struct.DoorOpen:
		if requests.RequestsShouldClearImmediately(localElevator, btnFloor, btnType) {
			//TODO: door timer start/reset, stuck timer reset
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
			//TODO: door timer start/reset, stuck timer reset
			localElevator = requests.RequestsClearAtCurrentFloor(localElevator)

		case elev_struct.Moving:
			elevio.SetMotorDirection(localElevator.Dir)
			//TODO: reset stuck timer

		case elev_struct.Idle:
		}
	}

	elev_struct.SetCabLights(localElevator)

	return localElevator
}

func OnFloorArrival(e elev_struct.Elevator, newFloor int) elev_struct.Elevator {
	localElevator := e
	localElevator.Floor = newFloor
	elevio.SetFloorIndicator(localElevator.Floor)

	switch localElevator.State{
	case elev_struct.Moving:
		if requests.RequestsShouldStop(localElevator) {
			elevio.SetMotorDirection(elevio.MD_Stop)
			elevio.SetDoorOpenLamp(true)
			localElevator = requests.RequestsClearAtCurrentFloor(localElevator)
			//TODO: door timer start/reset
			elev_struct.SetCabLights(localElevator)
			localElevator.State = elev_struct.DoorOpen
		}

	default:
	}

	return localElevator
}

func OnDoorTimeout(e elev_struct.Elevator) {}

func OnObstruction(e elev_struct.Elevator) {}
