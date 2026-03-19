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
	AllCabOrders        types.AllCabOrders
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
	peerTxEnable = make(chan bool, 1)

	go peers.Transmitter(27023, id, peerTxEnable)
	go peers.Receiver(27023, peerUpdateCh)
	networkTx = make(chan NetworkMsg, config.BUFFER_SIZE)
	networkRx = make(chan NetworkMsg, config.BUFFER_SIZE)
	go bcast.Transmitter(23879, networkTx)
	go bcast.Receiver(23879, networkRx)
}

func NetworkSend(elevator types.Elevator, hallOrders types.HallOrders, cabOrders types.AllCabOrders, recovering bool) {
	msg := NetworkMsg{Elevator: elevator, HallOrders: hallOrders, AllCabOrders: cabOrders, CabOrdersRecovering: recovering}
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