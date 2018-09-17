package fibrechannel

import (
		"testing"
		"os"
	"fmt"
)

func TestFindDisk(t *testing.T){

	testWwn := "12345678"
	fcPath := "-fc-0x" + testWwn + "-lun-1"

	devPath := "/dev/disk/by-path/"
	os.MkdirAll( devPath, 0755)
	_, err := os.Create(devPath + "x" + fcPath)

	if err != nil {
		fmt.Printf("unable to create file: %v", err)
	}

	disk, dm := findDisk(testWwn, "1")

	if disk == "" && dm == "" {
		t.Error("Could not find disk with given WWN")
	}

	os.Remove(devPath + "x"+ fcPath)

}

//func TestFindDiskWWIDs (t *testing.T){
//
//	testWWID := "12345678"
//	DevID := "/dev/disk/by-id/"
//
//	appFs.MkdirAll(DevID, 0755)
//	afero.WriteFile(appFs, DevID + "scsi-" + testWWID, []byte("file b"), 0644)
//	disk, dm := findDiskWWIDs(testWWID)
//
//	if disk == "" && dm == "" {
//		t.Error("Could not find disk with given WWID")
//	}
//
//
//}