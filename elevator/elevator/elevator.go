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
	drvButtonsChan <-chan elevio.ButtonEvent,
	drvFloorsChan <-chan int,
	drvObstrChan <-chan bool,
	reassignedHallOrdersChan <-chan [N_FLOORS][N_BUTTONS - 1]bool,
	completedOrderChan chan<- elevio.ButtonEvent,
	elevOutChan chan<- elev_struct.Elevator) {

	elevator := elev_struct.ElevatorInit(id)
	elevator.Floor = elevio.GetFloor()

	doorTimer := time.NewTimer(DOOR_OPEN_TIME)
	stuckTimer := time.NewTimer(STALL_TIME)
	publishTicker := time.NewTicker(20 * time.Millisecond)
	defer publishTicker.Stop()

	for {
		select {
		case reassignedHallOrders := <-reassignedHallOrdersChan:
			previousRequests := elevator.Requests
			for floor := 0; floor < N_FLOORS; floor++ {
				for btn := 0; btn < N_BUTTONS-1; btn++ {
					elevator.Requests[floor][btn] = reassignedHallOrders[floor][btn]
				}
			}
			for floor := 0; floor < N_FLOORS; floor++ {
				for btn := 0; btn < N_BUTTONS-1; btn++ {
					if reassignedHallOrders[floor][btn] && !previousRequests[floor][btn] {
						elevator = state_machine.OnRequestButtonPress(
							elevator,
							floor,
							elevio.ButtonType(btn),
							doorTimer,
							stuckTimer,
							completedOrderChan,
						)
					}
				}
			}

		case btnEvent := <-drvButtonsChan:
			if btnEvent.Button == elevio.BT_Cab {
				elevator = state_machine.OnRequestButtonPress(
					elevator,
					btnEvent.Floor,
					btnEvent.Button,
					doorTimer,
					stuckTimer,
					completedOrderChan,
				)
				elevOutChan <- elevator
				continue
			}

			elevatorWithNewOrder := elevator
			elevatorWithNewOrder.Requests[btnEvent.Floor][btnEvent.Button] = true
			elevOutChan <- elevatorWithNewOrder

		case newFloor := <-drvFloorsChan:
			elevator = state_machine.OnFloorArrival(elevator, newFloor, doorTimer, completedOrderChan)
			stuckTimer.Reset(STALL_TIME)
			if elevator.Stuck {
				elevator.Stuck = false
			}

		case obstructionSwitch := <-drvObstrChan:
			if obstructionSwitch {
				state_machine.OnObstruction(elevator, doorTimer)
				elevator.Obstructed = true
			} else if !obstructionSwitch && elevator.Obstructed {
				elevator.Obstructed = false
			}

		case <-doorTimer.C:
			elevator = state_machine.OnDoorTimeout(elevator, doorTimer, completedOrderChan)
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

		case <-publishTicker.C:
			if elevator.Obstructed {
				state_machine.OnObstruction(elevator, doorTimer)
			}
			elevOutChan <- elevator
		}
	}
}

func ElevatorInit(
	id string,
	port string,
	reassignedOrders <-chan [N_FLOORS][N_BUTTONS - 1]bool,
	completedOrder chan<- elevio.ButtonEvent,
	elevOut chan<- elev_struct.Elevator) {

	elevio.Init("localhost:"+port, config.N_FLOORS)

	startFloor := elevio.GetFloor()
	if startFloor == -1 {
		elevio.SetMotorDirection(elevio.MD_Down)
		for {
			floor := elevio.GetFloor()
			if floor != -1 {
				elevio.SetMotorDirection(elevio.MD_Stop)
				break
			}
		}
	}

	drvButtons := make(chan elevio.ButtonEvent)
	drvFloors := make(chan int)
	drvObstr := make(chan bool)

	go elevio.PollButtons(drvButtons)
	go elevio.PollFloorSensor(drvFloors)
	go elevio.PollObstructionSwitch(drvObstr)

	go RunElevator(id, drvButtons, drvFloors, drvObstr, reassignedOrders, completedOrder, elevOut)
}
