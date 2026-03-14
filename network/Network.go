package network

import (
	"heislab-sanntid/network/network/bcast"
	"heislab-sanntid/network/network/peers"
	"heislab-sanntid/types"
)

type NetworkMsg struct {
	Elevator   types.Elevator
	HallOrders types.HallOrders
}

var (
	networkTx    chan NetworkMsg
	networkRx    chan NetworkMsg
	peerUpdateCh chan peers.PeerUpdate
	peerTxEnable chan bool
)

func NetworkInit(id string) {

	peerUpdateCh = make(chan peers.PeerUpdate)
	// We can disable/enable the transmitter after it has been started.
	// This could be used to signal that we are somehow "unavailable".
	peerTxEnable = make(chan bool)

	go peers.Transmitter(15647, id, peerTxEnable)
	go peers.Receiver(15647, peerUpdateCh)

	// We make channels for sending and receiving our custom data types
	networkTx = make(chan NetworkMsg)
	networkRx = make(chan NetworkMsg)
	// ... and start the transmitter/receiver pair on some port
	// These functions can take any number of channels! It is also possible to
	//  start multiple transmitters/receivers on the same port.
	go bcast.Transmitter(16569, networkTx)
	go bcast.Receiver(16569, networkRx)
}
func NetworkSend(elevator types.Elevator, hallOrders types.HallOrders) {
	msg := NetworkMsg{Elevator: elevator, HallOrders: hallOrders}
	networkTx <- msg
}

func NetworkRxChan() <-chan NetworkMsg {
	return networkRx
}

func Peers() <-chan peers.PeerUpdate {
	return peerUpdateCh
}

func SetPeerTxEnable(enable bool) {
	peerTxEnable <- enable
}

func HasVisiblePeers() bool {
	select {
	case peers := <-peerUpdateCh:
		if len(peers.Peers) > 0 {
			return true
		}
		return false
	default:
		return false
	}
}
