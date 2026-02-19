package requests

import (
	"heislab-sanntid/config"
	"heislab-sanntid/elevator/elev_struct"
	"heislab-sanntid/elevator/elevio"
)

const (
	N_FLOORS  int = config.N_FLOORS
	N_BUTTONS int = config.N_BUTTONS
)

func RequestsAbove(e elev_struct.Elevator) bool {
	for f := e.Floor + 1; f < N_FLOORS; f++ {
		for btn := 0; btn < N_BUTTONS; btn++ {
			if e.Requests[f][btn] {
				return true
			}
		}
	}
	return false
}

func RequestsBelow(e elev_struct.Elevator) bool {
	for f := 0; f < e.Floor; f++ {
		for btn := 0; btn < N_BUTTONS; btn++ {
			if e.Requests[f][btn] {
				return true
			}
		}
	}
	return false
}

func RequestsHere(e elev_struct.Elevator) bool {
	for btn := 0; btn < N_BUTTONS; btn++ {
		if e.Requests[e.Floor][btn] {
			return true
		}
	}
	return false
}

func RequestsChooseDirection(e elev_struct.Elevator) elev_struct.DirStatePair {
	switch e.Dir {
	case elevio.MD_Up:
		if RequestsAbove(e) {
			return elev_struct.DirStatePair{Dir: elevio.MD_Up, State: elev_struct.Moving}
		} else if RequestsHere(e) {
			return elev_struct.DirStatePair{Dir: elevio.MD_Down, State: elev_struct.DoorOpen}
		} else if RequestsBelow(e) {
			return elev_struct.DirStatePair{Dir: elevio.MD_Down, State: elev_struct.Moving}
		} else {
			return elev_struct.DirStatePair{Dir: elevio.MD_Stop, State: elev_struct.Idle}
		}

	case elevio.MD_Down:
		if RequestsBelow(e) {
			return elev_struct.DirStatePair{Dir: elevio.MD_Down, State: elev_struct.Moving}
		} else if RequestsHere(e) {
			return elev_struct.DirStatePair{Dir: elevio.MD_Up, State: elev_struct.DoorOpen}
		} else if RequestsAbove(e) {
			return elev_struct.DirStatePair{Dir: elevio.MD_Up, State: elev_struct.Moving}
		} else {
			return elev_struct.DirStatePair{Dir: elevio.MD_Stop, State: elev_struct.Idle}
		}

	case elevio.MD_Stop:
		if RequestsHere(e) {
			return elev_struct.DirStatePair{Dir: elevio.MD_Stop, State: elev_struct.DoorOpen}
		} else if RequestsAbove(e) {
			return elev_struct.DirStatePair{Dir: elevio.MD_Up, State: elev_struct.Moving}
		} else if RequestsBelow(e) {
			return elev_struct.DirStatePair{Dir: elevio.MD_Down, State: elev_struct.Moving}
		} else {
			return elev_struct.DirStatePair{Dir: elevio.MD_Stop, State: elev_struct.Idle}
		}

	default:
		return elev_struct.DirStatePair{Dir: elevio.MD_Stop, State: elev_struct.Idle}
	}
}

func RequestsShouldStop(e elev_struct.Elevator) bool {
	switch e.Dir {
	case elevio.MD_Down:
		return e.Requests[e.Floor][elevio.BT_HallDown] ||
			e.Requests[e.Floor][elevio.BT_Cab] ||
			!RequestsBelow(e)

	case elevio.MD_Up:
		return e.Requests[e.Floor][elevio.BT_HallUp] ||
			e.Requests[e.Floor][elevio.BT_Cab] ||
			!RequestsAbove(e)

	case elevio.MD_Stop:
		fallthrough

	default:
		return true
	}
}

func RequestsShouldClearImmediately(e elev_struct.Elevator, btnFloor int, btnType elevio.ButtonType) bool {
	return e.Floor == btnFloor && ((e.Dir == elevio.MD_Up && btnType == elevio.BT_HallUp) ||
		(e.Dir == elevio.MD_Down && btnType == elevio.BT_HallDown) ||
		(e.Dir == elevio.MD_Stop) ||
		btnType == elevio.BT_Cab)
}

func RequestsClearAtCurrentFloor(e elev_struct.Elevator, clear_order_chan chan<- elevio.ButtonEvent) elev_struct.Elevator {
	e.Requests[e.Floor][elevio.BT_Cab] = false

	//TODO: sende på en channel at bestillingen er utført. Kanskje lurt å da legge inn ekstra sjekk for om bestilling fortsatt eksisterer før det sendes på channel
	switch e.Dir {
	case elevio.MD_Up:
		if !RequestsAbove(e) && !e.Requests[e.Floor][elevio.BT_HallUp] {
			e.Requests[e.Floor][elevio.BT_HallDown] = false
		}
		e.Requests[e.Floor][elevio.BT_HallUp] = false

	case elevio.MD_Down:
		if !RequestsBelow(e) && !e.Requests[e.Floor][elevio.BT_HallDown] {
			e.Requests[e.Floor][elevio.BT_HallUp] = false
		}
		e.Requests[e.Floor][elevio.BT_HallDown] = false

	case elevio.MD_Stop:
		fallthrough

	default:
		e.Requests[e.Floor][elevio.BT_HallUp] = false
		e.Requests[e.Floor][elevio.BT_HallDown] = false
	}
	return e
}
