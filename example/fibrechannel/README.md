## Fibre Channel Connector Usage
The fibre channel connector is a library that can be imported to create your own fibre channel CSI driver. Simply import it and get access to the necessary functions.

`import "github.com/j-griffith/fibrechannel"`

The Connect, Disconnect, and MountDisk functions are used to Attach, Detach, and Mount a Fibre Channel disk.

## Running Fibre Channel Connector Example

### Edit main-fc.go file: 
To be able to connect to a fc device you would need to edit the main-fc.go file to match the WWNs, WWIDs, and LUN, on your system.

### Build test file: 
`go build main-fc.go`

### Run: 
`./main-fc.go`

#### You should see an output similar to this: 

~~~~
[root@storageqe-18 fibrechannel]# ./main-fc
TRACE: 2018/09/07 15:49:57 fibrechannel.go:196: Connecting fibre channel volume
TRACE: 2018/09/07 15:49:57 fibrechannel.go:68: Found matching device: sdi under dm-* device path /sys/block/dm-1
TRACE: 2018/09/07 15:49:57 fibrechannel.go:149: fc: find disk: /dev/sdi, dm: /dev/dm-1
2018/09/07 15:49:57 Path is: /dev/dm-1
TRACE: 2018/09/07 15:49:57 fibrechannel.go:209: Disconnecting fibre channel volume
TRACE: 2018/09/07 15:49:57 fibrechannel.go:224: fc: DetachDisk devicePath: /dev/dm-1, dstPath: /dev/dm-1, devices: [/dev/sdc /dev/sdi /dev/sdo]
TRACE: 2018/09/07 15:49:57 fibrechannel.go:277: fc: remove device from scsi-subsystem: path: /sys/block/sdc/device/delete
TRACE: 2018/09/07 15:49:57 fibrechannel.go:277: fc: remove device from scsi-subsystem: path: /sys/block/sdi/device/delete
TRACE: 2018/09/07 15:49:57 fibrechannel.go:277: fc: remove device from scsi-subsystem: path: /sys/block/sdo/device/delete
~~~~

### [A CSI driver that utilizes this Connector](https://github.com/mathu97/fibrechannel-kubernetes-csi-driver)
