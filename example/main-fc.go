package main

import (
	"github.com/mathu97/csi-connectors/fibrechannel"
		"log"
	"time"
	"os/exec"
	"fmt"
)

func main() {
	c := fibrechannel.Connector{}
	//Host5 and host6 respectively
	c.TargetWWNs = []string{"10000000c9a02834", "10000000c9a02835"}
	c.Lun = "1"
	dp, err := fibrechannel.Connect(c)
	log.Printf("Path is: %s\n", dp)
	if err != nil {
		log.Printf("Error from Connect: %s\n", err)
	}

	time.Sleep(3 * 100 * time.Millisecond)
	out, _ := exec.Command("ls /dev/disk/by-path/").CombinedOutput()
	fmt.Printf("disk by path: %s\n", out)
	fibrechannel.Disconnect(c, dp)
	fmt.Printf("disk by path: %s\n", out)
	out, _ = exec.Command("ls /dev/disk/by-path/").CombinedOutput()
}