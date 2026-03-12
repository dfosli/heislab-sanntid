package elevator

import (
	elevio "Driver-go"
	"heislab-sanntid/config"
	"heislab-sanntid/elevator/elev_struct"
	"heislab-sanntid/elevator/state_machine"
	"log"
	"time"
)

const (
	N_FLOORS       int = config.N_FLOORS
	N_BUTTONS      int = config.N_BUTTONS
	DOOR_OPEN_TIME     = config.DOOR_OPEN_TIME
	STALL_TIME         = config.STALL_TIME
)

func RunElevator(
	id string,
	drv_buttons_chan <-chan elevio.ButtonEvent,
	drv_floors_chan <-chan int,
	drv_obstr_chan <-chan bool,
	clear_local_hall_orders_chan <-chan bool,
	completed_order_chan chan<- elevio.ButtonEvent,
	assigned_orders_chan chan elevio.ButtonEvent,
	elev_out_chan chan<- elev_struct.Elevator) {

	elevator := elev_struct.ElevatorInit(id)
	elevator.Floor = elevio.GetFloor()

	if elevator.Floor == -1 {
		elevio.SetMotorDirection(elevio.MD_Down)
		for {
			floor := elevio.GetFloor()
			if floor != -1 {
				elevio.SetMotorDirection(elevio.MD_Stop)
				break
			}
		}
	}

	doorTimer := time.NewTimer(DOOR_OPEN_TIME)
	stuckTimer := time.NewTimer(STALL_TIME)

	for {
		select {
		case <-clear_local_hall_orders_chan:
			elevator = elev_struct.ClearLocalHallOrders(elevator)

		case btnEvent := <-drv_buttons_chan:
			elevatorWithNewOrder := elevator
			elevatorWithNewOrder.Requests[btnEvent.Floor][btnEvent.Button] = true

			if btnEvent.Button == elevio.BT_Cab {
				assigned_orders_chan <- btnEvent
			}

			elev_out_chan <- elevatorWithNewOrder

		case btnEvent := <-assigned_orders_chan:
			elevator = state_machine.OnRequestButtonPress(elevator, btnEvent.Floor, btnEvent.Button, doorTimer, stuckTimer, completed_order_chan)

		case newFloor := <-drv_floors_chan:
			elevator = state_machine.OnFloorArrival(elevator, newFloor, doorTimer, completed_order_chan)
			stuckTimer.Reset(STALL_TIME)
			if elevator.Stuck {
				elevator.Stuck = false
			}

		case obstructionSwitch := <-drv_obstr_chan:
			if obstructionSwitch {
				state_machine.OnObstruction(elevator, doorTimer)
				elevator.Obstructed = true
			} else if !obstructionSwitch && elevator.Obstructed {
				elevator.Obstructed = false
			}

		case <-doorTimer.C:
			elevator = state_machine.OnDoorTimeout(elevator, doorTimer, completed_order_chan)
			stuckTimer.Reset(STALL_TIME)
			if elevator.Stuck {
				elevator.Stuck = false
			}

		case <-stuckTimer.C:
			if elevator.State != elev_struct.Idle {
				elevator.Stuck = true
				elevator = elev_struct.ClearLocalHallOrders(elevator)
				log.Printf("stuck timer case")
			}

		default:
			elev_out_chan <- elevator

			if elevator.Obstructed {
				state_machine.OnObstruction(elevator, doorTimer)
			}
		}
	}
}

func ElevatorInit(id string) {
	elevio.Init("localhost:15657", config.N_FLOORS)

	drv_buttons := make(chan elevio.ButtonEvent)
	drv_floors := make(chan int)
	drv_obstr := make(chan bool)

	elev_out := make(chan elev_struct.Elevator)
	clear_local_hall_orders := make(chan bool, config.BUFFER_SIZE)
	clear_order := make(chan elevio.ButtonEvent, config.BUFFER_SIZE)
	assigned_orders := make(chan elevio.ButtonEvent, config.BUFFER_SIZE)

	go elevio.PollButtons(drv_buttons)
	go elevio.PollFloorSensor(drv_floors)
	go elevio.PollObstructionSwitch(drv_obstr)

	go RunElevator(id, drv_buttons, drv_floors, drv_obstr, clear_local_hall_orders, clear_order, assigned_orders, elev_out)
}
