# TTK4145 Elevator Project

## Project Overview

A distributed real-time elevator control system written in Go. Uses a peer-to-peer design, where elevators communicate over UDP to sync orders and coordinate assignment, handle failures, and recover state. Each node runs an identical program and discovers peers automatically.

---

## Modules

### `config`
Defines system-wide constants.

### `types`
Shared types and the `OrderState` enum used across packages:
- **NONE** -> **NEW** -> **CONFIRMED** -> **ASSIGNED** -> **COMPLETED** -> **NONE**

### `elevator`
Manages the local elevator hardware and behavior.

- **`elevio`**: Hardware abstraction layer (driver). Opens a TCP connection to the elevator simulator and exposes functions for motor, lamps, and sensors. Polling goroutines push hardware events onto channels.
- **`elev_struct`**: Defines the `Elevator` struct, containing elevator state and helper functions.
- **`state_machine`**: Implements the elevator FSM with handlers for button presses, floor arrivals, door timeouts, and obstruction events.
- **`requests`**: Algorithms that determine elevator direction, whether to stop at a floor, and how to clear completed requests.

### `network`
Handles all communication over UDP.

- **`Network.go`**: Public interface. Defines the `NetworkMsg` struct, and exposes `NetworkInit`, `NetworkSend`, `NetworkRxChan`, `Peers`, and `SetPeerTxEnable`.
- **`bcast`**: Broadcasts and receives JSON-encoded messages on port 23879.
- **`peers`**: Heartbeat protocol on port 27023. Nodes broadcast their ID every 15ms. A node is considered lost after 500ms of silence. Produces `PeerUpdate` events (new peer, lost peers, current peers).

### `orders`
Distributed order management with consensus-based confirmation.

- **`orders.go`**: Contains central goroutine (`runOrderManager`), which handles peer updates, local and remote elevator updates, triggers order redistribution, manages stuck elevator detection, and sends network broadcasts every 100ms. Also contains two additional goroutines: `confirmHallOrders` and `resetHallOrders`. They signal orders to advance **NEW** -> **CONFIRMED** and **COMPLETED** -> **NONE** respectively, when there is consensus between elevators. These goroutines "own" these order state transitions.
- **`sync.go`**: Algorithms for synchronizing order states from local elevator and between peers, recovering cab orders after restart, and reassigning orders from unavailable elevators.

### `distributor`
Wraps an external executable (`hall_request_assigner`) that computes optimal order assignments, using a "reassign all orders on update" approach. Helper functions format system state to JSON, runs the executable, and parses the output back into per-elevator hall order assignments.

---

## Program Flow

### Startup (`main.go`)

- Parse CLI flags `-id` and `-port`
- Initialize four channels:
  - `elevOutCh` - local elevator state -> order manager
  - `reassignLocalHallOrdersCh` - new assignments -> local elevator
  - `recoveredCabOrdersCh` - recovered cab orders -> local elevator
  - `completedOrderCh` - completed orders from local elevator -> order manager
- `network.NetworkInit(id)` - start UDP broadcast and heartbeat
- `elevator.ElevatorInit(...)` - connect to hardware, initialize floor, start FSM
- `orders.OrdersInit(...)` - initialize order state, start goroutines

### Elevator Operation

- Hardware polling goroutines run continuously:
  - `PollButtons` - sends button press events
  - `PollFloorSensor` - sends floor arrival events
  - `PollObstructionSwitch` - sends obstruction events
- `RunElevator` (FSM loop) reacts to those events:
  - `OnRequestButtonPress` - sets request, transitions state
  - `OnFloorArrival` - stops or continues, clears completed requests
  - `OnDoorTimeout` - closes door, chooses next direction
  - `OnObstruction` - extends door open timer
- Elevator publishes its state periodically via `elevOutCh` to `runOrderManager`

### Order Lifecycle

1. Button press detected by `RunElevator`
2. `runOrderManager` receives elevator update via `elevOutCh`, and `AddNewLocalOrder` marks the order as **NEW**
3. Network broadcasts current state to all peers
4. All peers acknowledge **NEW**, `confirmHallOrders` goroutine signals, `runOrderManager` marks order as **CONFIRMED**
5. `ReassignOrders` calls the external distributor executable, and orders assigned to itself are sent via `reassignLocalHallOrdersCh` to `RunElevator`. Order is marked as **ASSIGNED**.
6. Elevator services the order and reaches the target floor, `completedOrderCh` -> `runOrderManager`, marks order **COMPLETED**.
7. All peers sync to **COMPLETED**, `resetHallOrders` signals, `runOrderManager` resets order to **NONE**.

### Fault Handling

**Stuck elevator:**
- `stuckTimer` expires while Moving -> elevator sets `Stuck = true` which is sent with state through `elevOutCh`.
- `runOrderManager` detects `Stuck`:
  - Sets `availableElevators[id] = false`
  - Calls `SetPeerTxEnable(false)` to stop broadcasting, makes other elevators mark us as unavailable and not assign orders to us
  - Calls `handleElevatorUnavailable` to reset its hall orders

**Peer disconnect:**
- `peers.Receiver` detects heartbeat timeout and signals peer is lost through `PeerUpdate`
- `runOrderManager` calls `handleElevatorUnavailable` to reassign the lost elevator's **CONFIRMED**/**ASSIGNED** orders to remaining peers

**Cab order recovery after restart:**
- Node restarts with `CabOrdersRecovering = true` (5-second window)
- Peers send their stored copy of this node's cab orders via `NetworkMsg`
- `recoverLocalCabOrders` reconstructs the cab request matrix
- After 5 seconds the recovery window closes

---
