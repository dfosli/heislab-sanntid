package network
	
import (
	"fmt"
	"Network-go/network/localip"
	"os"
	"flag"
)

func NetworkInit() {

	var id string
	flag.StringVar(&id, "id", "", "id of this peer")
	flag.Parse()

	if id == "" {
		localIP, err := localip.LocalIP()
		if err != nil {
			fmt.Println(err)
			localIP = "DISCONNECTED"
		}
		id = fmt.Sprintf("peer-%s-%d", localIP, os.Getpid())
	}
}

func NetworkSend(struct) {
	
}

func NetworkRecive() struct {

}