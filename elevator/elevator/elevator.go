package elevator

import (
	"heislab-sanntid/config"
	"heislab-sanntid/elevator/elev_struct"
	"heislab-sanntid/elevator/elevio"
	"heislab-sanntid/elevator/fsm"
	"log"
	"time"
)

const (
	N_FLOORS       int = config.N_FLOORS
	N_BUTTONS      int = config.N_BUTTONS
	DOOR_OPEN_TIME     = config.DOOR_OPEN_TIME
	STUCK_TIME         = config.STUCK_TIME
)

func RunElevator(
	id int,
	drv_buttons_chan <-chan elevio.ButtonEvent,
	drv_floors_chan <-chan int,
	drv_obstr_chan <-chan bool,
	clear_local_hall_orders_chan <-chan bool,
	clear_order_chan chan<- elevio.ButtonEvent,
	assigned_orders_chan chan elevio.ButtonEvent,
	elev_out_chan chan<- elev_struct.Elevator) {

	elevator := elev_struct.ElevatorInit(id)
	elevator.Floor = elevio.GetFloor()
	if elevator.Floor == -1 { //? Mulig å loope helt til man treffer en floor i stedet
		fsm.OnInitBetweenFloors(elevator)
	}

	doorTimer := time.NewTimer(DOOR_OPEN_TIME)
	stuckTimer := time.NewTimer(STUCK_TIME)

	for {
		select {
		case <-clear_local_hall_orders_chan:
			elevator = elev_struct.ClearHallOrders(elevator)

		case btnEvent := <-drv_buttons_chan: //TODO: lage kopi og sende på out channel. Assigne til seg selv hvis cab order.
			assigned_orders_chan <- btnEvent //assigner alle til seg selv siden distribution ikke er implementert

		case btnEvent := <-assigned_orders_chan:
			elevator = fsm.OnRequestButtonPress(elevator, btnEvent.Floor, btnEvent.Button, doorTimer, stuckTimer, clear_order_chan)

		case newFloor := <-drv_floors_chan:
			elevator = fsm.OnFloorArrival(elevator, newFloor, doorTimer, clear_order_chan)
			stuckTimer.Reset(STUCK_TIME)
			if elevator.Stuck {
				elevator.Stuck = false
			}

		case obstructionSwitch := <-drv_obstr_chan:
			if obstructionSwitch {
				fsm.OnObstruction(elevator, doorTimer)
				elevator.Obstructed = true
			} else if !obstructionSwitch && elevator.Obstructed {
				elevator.Obstructed = false
			}

		case <-doorTimer.C:
			elevator = fsm.OnDoorTimeout(elevator, doorTimer, clear_order_chan)
			stuckTimer.Reset(STUCK_TIME)
			if elevator.Stuck {
				elevator.Stuck = false
			}

		case <-stuckTimer.C:
			if elevator.State != elev_struct.Idle {
				elevator.Stuck = true
				elevator = elev_struct.ClearHallOrders(elevator)
				log.Printf("stuck timer case")
			}

		default:
			elev_out_chan <- elevator

			if elevator.Obstructed {
				fsm.OnObstruction(elevator, doorTimer)
			}
		}
	}
}
