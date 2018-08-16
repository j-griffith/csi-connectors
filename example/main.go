package main

import (
	"log"

	"github.com/j-griffith/csi-connectors/iscsi"
)

func main() {
	c := iscsi.Connector{}
	c.AuthType = "chap"
	c.TargetIqn = "iqn.2010-10.org.openstack:volume-eb393993-73d0-4e39-9ef4-b5841e244ced"
	c.TargetPortals = []string{"192.168.1.107:3260"}
	c.SessionSecrets.UserName = "86Jx6hXYqDYpKamtgx4d"
	c.SessionSecrets.Password = "Qj3MuzmHu8cJBpkv"
	c.Lun = 1
	path, err := iscsi.Connect(c)
	log.Printf("path is: %s\n", path)
	if err != nil {
		log.Printf("err is: %s\n", err.Error())

	}

}
