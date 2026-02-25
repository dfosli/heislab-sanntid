package network

import (
	"time"
	"fmt"
	"Network-go/network/bcast"
)

// Create channel for each peer. Mutex to lock the list of channels.
type HeartbeatManager struct {
	id string
	heartbeatInterval time.Duration
}

type HeartBeat struct {
	Message string
}

func HeartbeatInit(id string, peerAddrs []string, heartbeatInterval time.Duration) (*HeartbeatManager, error) {

	return &HeartbeatManager{
		id: id,
		heartbeatInterval: heartbeatInterval,
	}, nil

}

func (heartBeat *HeartbeatManager) sender() {

	heartBeatMsg := HeartBeat{"Hello from" + heartBeat.id}
	bcast.Transmitter(16569, heartBeatMsg)
}

func (heartBeat *HeartbeatManager) receiver() {
	go bcast.Receiver(16569, helloRx)

	for {
		select {		
			case a := <-helloRx:
			fmt.Printf("Received: %#v\n", a)
		}
	}
}

func (heartBeat *HeartbeatManager) Heartbeatstart() {
	go heartBeat.receiver()
	go heartBeat.sender()
}
