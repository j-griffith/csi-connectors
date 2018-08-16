package main

import (
	"flag"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/j-griffith/csi-connectors/iscsi"
)

var (
	portals   = flag.String("portals", "192.168.1.107:3260", "Comma delimited.  Eg: 1.1.1.1,2.2.2.2")
	iqn       = flag.String("iqn", "iqn.2010-10.org.openstack:volume-eb393993-73d0-4e39-9ef4-b5841e244ced", "")
	multipath = flag.Bool("multipath", false, "")
	username  = flag.String("username", "86Jx6hXYqDYpKamtgx4d", "")
	password  = flag.String("password", "Qj3MuzmHu8cJBpkv", "")
	lun       = flag.Int("lun", 1, "")
)

func main() {
	flag.Parse()
	tgtp := strings.Split(*portals, ",")

	c := iscsi.Connector{
		AuthType:      "chap",
		TargetIqn:     *iqn,
		TargetPortals: tgtp,
		SessionSecrets: iscsi.Secrets{
			UserName: "",
			Password: ""},
		Lun:       int32(*lun),
		Multipath: *multipath,
	}
	path, err := iscsi.Connect(c)
	log.Printf("path is: %s\n", path)
	if err != nil {
		log.Printf("err is: %s\n", err.Error())
	}
	time.Sleep(3 * 100 * time.Millisecond)
	out, err := exec.Command("ls", "/dev/disk/by-path/").CombinedOutput()
	if err != nil {
		fmt.Printf(err.Error())
	}
	fmt.Printf("disk by path: %s\n", out)
	iscsi.Disconnect(c.TargetIqn, c.TargetPortals)
	time.Sleep(3 * 100 * time.Millisecond)
	out, err = exec.Command("ls", "/dev/disk/by-path/").CombinedOutput()
	if err != nil {
		fmt.Printf(err.Error())
	}
	fmt.Printf("disk by path: %s\n", out)

}
