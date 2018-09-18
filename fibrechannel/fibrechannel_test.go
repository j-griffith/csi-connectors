package fibrechannel

import (
	"os"
	"testing"
	"time"
)

type fakeFileInfo struct {
	name string
}

func (fi *fakeFileInfo) Name() string {
	return fi.name
}

func (fi *fakeFileInfo) Size() int64 {
	return 0
}

func (fi *fakeFileInfo) Mode() os.FileMode {
	return 777
}

func (fi *fakeFileInfo) ModTime() time.Time {
	return time.Now()
}
func (fi *fakeFileInfo) IsDir() bool {
	return false
}

func (fi *fakeFileInfo) Sys() interface{} {
	return nil
}

type fakeIOHandler struct{}

func (handler *fakeIOHandler) ReadDir(dirname string) ([]os.FileInfo, error) {
	switch dirname {
	case "/dev/disk/by-path/":
		f := &fakeFileInfo{
			name: "pci-0000:41:00.0-fc-0x500a0981891b8dc5-lun-0",
		}
		return []os.FileInfo{f}, nil
	case "/sys/block/":
		f := &fakeFileInfo{
			name: "dm-1",
		}
		return []os.FileInfo{f}, nil
	case "/dev/disk/by-id/":
		f := &fakeFileInfo{
			name: "scsi-3600508b400105e210000900000490000",
		}
		return []os.FileInfo{f}, nil
	}
	return nil, nil
}

func (handler *fakeIOHandler) Lstat(name string) (os.FileInfo, error) {
	return nil, nil
}

func (handler *fakeIOHandler) EvalSymlinks(path string) (string, error) {
	return "/dev/sda", nil
}

func (handler *fakeIOHandler) WriteFile(filename string, data []byte, perm os.FileMode) error {
	return nil
}

func TestSearchDisk(t *testing.T) {
	fakeConnector := Connector{
		VolumeName: "fakeVol",
		TargetWWNs: []string{"500a0981891b8dc5"},
		Lun:        "0",
	}

	devicePath, error := searchDisk(fakeConnector, &fakeIOHandler{})

	if devicePath == "" || error != nil {
		t.Errorf("no fc disk found")
	}
}

func TestInvalidWWN(t *testing.T) {
	testWwn := "INVALIDWWN"
	disk, dm := findDisk(testWwn, "1", &fakeIOHandler{})

	if disk != "" && dm != "" {
		t.Error("Found a disk with WWN that does not Exist")
	}
}

func TestInvalidWWID(t *testing.T) {
	testWWID := "INVALIDWWID"
	disk, dm := findDiskWWIDs(testWWID, &fakeIOHandler{})

	if disk != "" && dm != "" {
		t.Error("Found a disk with WWID that does not Exist")
	}
}
