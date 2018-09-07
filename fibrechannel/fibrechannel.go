package fibrechannel

import (
	"fmt"
	"github.com/j-griffith/csi-connectors/logger"
	"io/ioutil"
	"k8s.io/kubernetes/pkg/util/mount"
	volumeutil "k8s.io/kubernetes/pkg/volume/util"
	"os"

	"path"
	"path/filepath"
	"strings"
)

var log *logger.Logger

//Connector provides a struct to hold all of the needed parameters to make our fibrechannel connection
type Connector struct {
	VolumeName string
	TargetWWNs []string
	Lun        string
	WWIDs      []string
}

type FCMounter struct {
	ReadOnly     bool
	FsType       string
	MountOptions []string
	Mounter      *mount.SafeFormatAndMount
	Exec         mount.Exec
	DeviceUtil   volumeutil.DeviceUtil
	TargetPath   string
}

func init() {
	// TODO: add a handle to configure loggers after init
	// also, make default for trace to go to discard when you're done messing around
	log = logger.NewLogger(os.Stdout, os.Stdout, os.Stdout, os.Stderr)
}

func getMultipathDisk(path string) (string, error) {
	// Follow link to destination directory
	devicePath, err := os.Readlink(path)
	if err != nil {
		log.Error.Printf("failed reading link: %s -- error: %s\n", path, err.Error())
		return "", err
	}
	sdevice := filepath.Base(devicePath)
	// If destination directory is already identified as a multipath device,
	// just return its path
	if strings.HasPrefix(sdevice, "dm-") {
		return path, nil
	}
	// Fallback to iterating through all the entries under /sys/block/dm-* and
	// check to see if any have an entry under /sys/block/dm-*/slaves matching
	// the device the symlink was pointing at
	dmpaths, _ := filepath.Glob("/sys/block/dm-*")
	for _, dmpath := range dmpaths {
		sdevices, _ := filepath.Glob(filepath.Join(dmpath, "slaves", "*"))
		for _, spath := range sdevices {
			s := filepath.Base(spath)
			if sdevice == s {
				// We've found a matching entry, return the path for the
				// dm-* device it was found under
				p := filepath.Join("/dev", filepath.Base(dmpath))
				log.Trace.Printf("Found matching device: %s under dm-* device path %s", sdevice, dmpath)
				return p, nil
			}
		}
	}
	return "", fmt.Errorf("Couldn't find dm-* path for path: %s, found non dm-* path: %s", path, devicePath)
}

func scsiHostRescan() {
	scsi_path := "/sys/class/scsi_host/"
	if dirs, err := ioutil.ReadDir(scsi_path); err == nil {
		for _, f := range dirs {
			name := scsi_path + f.Name() + "/scan"
			data := []byte("- - -")
			ioutil.WriteFile(name, data, 0666)
		}
	}
}

func searchDisk(c Connector) (string, error) {
	var diskIds []string
	var disk string
	var dm string

	if len(c.TargetWWNs) != 0 {
		diskIds = c.TargetWWNs
	} else {
		diskIds = c.WWIDs
	}

	rescaned := false
	// two-phase search:
	// first phase, search existing device path, if a multipath dm is found, exit loop
	// otherwise, in second phase, rescan scsi bus and search again, return with any findings
	for true {

		for _, diskId := range diskIds {
			if len(c.TargetWWNs) != 0 {
				disk, dm = findDisk(diskId, c.Lun)
			} else {
				disk, dm = findDiskWWIDs(diskId)
			}
			// if multipath device is found, break
			if dm != "" {

				break
			}
		}
		// if a dm is found, exit loop
		if rescaned || dm != "" {
			break
		}
		// rescan and search again
		// rescan scsi bus
		scsiHostRescan()
		rescaned = true
	}
	// if no disk matches input wwn and lun, exit
	if disk == "" && dm == "" {
		return "", fmt.Errorf("no fc disk found")
	}

	// if multipath devicemapper device is found, use it; otherwise use raw disk
	if dm != "" {
		return dm, nil
	}

	return disk, nil
}

// given a wwn and lun, find the device and associated devicemapper parent
func findDisk(wwn, lun string) (string, string) {
	FC_PATH := "-fc-0x" + wwn + "-lun-" + lun
	DEV_PATH := "/dev/disk/by-path/"
	if dirs, err := ioutil.ReadDir(DEV_PATH); err == nil {
		for _, f := range dirs {
			name := f.Name()
			if strings.Contains(name, FC_PATH) {
				if disk, err1 := filepath.EvalSymlinks(DEV_PATH + name); err1 == nil {
					dm, err2 := getMultipathDisk(DEV_PATH + name)
					if err2 == nil {
						log.Trace.Printf("fc: find disk: %v, dm: %v", disk, dm)
						return disk, dm
					}
				}
			}
		}
	}
	return "", ""
}

// given a wwid, find the device and associated devicemapper parent
func findDiskWWIDs(wwid string) (string, string) {
	// Example wwid format:
	//   3600508b400105e210000900000490000
	//   <VENDOR NAME> <IDENTIFIER NUMBER>
	// Example of symlink under by-id:
	//   /dev/by-id/scsi-3600508b400105e210000900000490000
	//   /dev/by-id/scsi-<VENDOR NAME>_<IDENTIFIER NUMBER>
	// The wwid could contain white space and it will be replaced
	// underscore when wwid is exposed under /dev/by-id.

	FC_PATH := "scsi-" + wwid
	DEV_ID := "/dev/disk/by-id/"
	if dirs, err := ioutil.ReadDir(DEV_ID); err == nil {
		for _, f := range dirs {
			name := f.Name()
			if name == FC_PATH {
				disk, err := filepath.EvalSymlinks(DEV_ID + name)
				if err != nil {
					log.Error.Printf("fc: failed to find a corresponding disk from symlink[%s], error %v", DEV_ID+name, err)
					return "", ""
				}
				dm, err1 := getMultipathDisk(DEV_ID + name)
				if err1 == nil {
					log.Trace.Printf("fc: find disk: %v, dm: %v", disk, dm)
					return disk, dm
				}

			}
		}
	}
	log.Error.Printf("fc: failed to find a disk [%s]", DEV_ID+FC_PATH)
	return "", ""
}

// Connect attempts to connect a fc volume to this node using the provided Connector info
func Connect(c Connector) (string, error) {
	devicePath, err := searchDisk(c)

	if err != nil {
		log.Error.Printf("unable to find disk given WWNN or WWIDs")
		return "", err
	}

	return devicePath, nil
}

func Disconnect(c Connector, devicePath string) error {
	var devices []string
	dstPath, err := filepath.EvalSymlinks(devicePath)

	if err != nil {
		return err
	}

	if strings.HasPrefix(dstPath, "/dev/dm-") {
		devices = FindSlaveDevicesOnMultipath(dstPath)
	} else {
		// Add single devicepath to devices
		devices = append(devices, dstPath)
	}

	log.Trace.Printf("fc: DetachDisk devicePath: %v, dstPath: %v, devices: %v", devicePath, dstPath, devices)

	var lastErr error

	for _, device := range devices {
		err := detachFCDisk(device)
		if err != nil {
			log.Error.Printf("fc: detachFCDisk failed. device: %v err: %v", device, err)
			lastErr = fmt.Errorf("fc: detachFCDisk failed. device: %v err: %v", device, err)
		}
	}

	if lastErr != nil {
		log.Error.Printf("fc: last error occurred during detach disk:\n%v", lastErr)
		return lastErr
	}

	return nil
}

func FindSlaveDevicesOnMultipath(dm string) []string {
	var devices []string
	// Split path /dev/dm-1 into "", "dev", "dm-1"
	parts := strings.Split(dm, "/")
	if len(parts) != 3 || !strings.HasPrefix(parts[1], "dev") {
		return devices
	}
	disk := parts[2]
	slavesPath := path.Join("/sys/block/", disk, "/slaves/")
	if files, err := ioutil.ReadDir(slavesPath); err == nil {
		for _, f := range files {
			devices = append(devices, path.Join("/dev/", f.Name()))
		}
	}
	return devices
}

// detachFCDisk removes scsi device file such as /dev/sdX from the node.
func detachFCDisk(devicePath string) error {
	// Remove scsi device from the node.
	if !strings.HasPrefix(devicePath, "/dev/") {
		return fmt.Errorf("fc detach disk: invalid device name: %s", devicePath)
	}
	arr := strings.Split(devicePath, "/")
	dev := arr[len(arr)-1]
	removeFromScsiSubsystem(dev)
	return nil
}

// Removes a scsi device based upon /dev/sdX name
func removeFromScsiSubsystem(deviceName string) {
	fileName := "/sys/block/" + deviceName + "/device/delete"
	log.Trace.Printf("fc: remove device from scsi-subsystem: path: %s", fileName)
	data := []byte("1")
	ioutil.WriteFile(fileName, data, 0666)
}

func MountDisk(mnter FCMounter, devicePath string) error {
	mntPath := mnter.TargetPath
	notMnt, err := mnter.Mounter.IsLikelyNotMountPoint(mntPath)

	if err != nil {
		return fmt.Errorf("Heuristic determination of mount point failed: %v", err)
	}

	if !notMnt {
		log.Trace.Printf("fc: %s already mounted", mnter.TargetPath)
	}

	if err = mnter.Mounter.FormatAndMount(devicePath, mnter.TargetPath, mnter.FsType, nil); err != nil {
		return fmt.Errorf("fc: failed to mount fc volume %s [%s] to %s, error %v", devicePath, mnter.FsType, mnter.TargetPath, err)
	}

	return nil
}
