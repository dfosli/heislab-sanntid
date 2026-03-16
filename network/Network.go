package network

import (
	"heislab-sanntid/config"
	"heislab-sanntid/network/network/bcast"
	"heislab-sanntid/network/network/peers"
	"heislab-sanntid/types"
)

type NetworkMsg struct {
	Elevator            types.Elevator
	HallOrders          types.HallOrders
	CabOrders           types.CabOrders
	CabOrdersRecovering bool
}

var (
	networkTx    chan NetworkMsg
	networkRx    chan NetworkMsg
	peerUpdateCh chan peers.PeerUpdate
	peerTxEnable chan bool
)

func NetworkInit(id string) {

	peerUpdateCh = make(chan peers.PeerUpdate, config.BUFFER_SIZE)
	peerTxEnable = make(chan bool)

	go peers.Transmitter(15647, id, peerTxEnable)
	go peers.Receiver(15647, peerUpdateCh)
	networkTx = make(chan NetworkMsg, config.BUFFER_SIZE)
	networkRx = make(chan NetworkMsg, config.BUFFER_SIZE)
	go bcast.Transmitter(16569, networkTx)
	go bcast.Receiver(16569, networkRx)
}

func NetworkSend(elevator types.Elevator, hallOrders types.HallOrders, cabOrders types.CabOrders, recovering bool) {
	msg := NetworkMsg{Elevator: elevator, HallOrders: hallOrders, CabOrders: cabOrders, CabOrdersRecovering: recovering}
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

// func HasVisiblePeers() bool { //! denne funksjonen leser rett fra kanalen, og "stjeler" oppdateringer fra hovedløkken i orders.go.
// 	select {
// 	case peers := <-peerUpdateCh:
// 		if len(peers.Peers) > 0 {
// 			return true
// 		}
// 		return false
// 	default:
// 		return false
// 	}
// }
