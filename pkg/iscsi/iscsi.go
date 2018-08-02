package iscsi

import (
	"errors"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

var (
	// Info provides a handle to a std logger to use for Info level events
	Info *log.Logger
	// Warning provides a handle to a std logger to use for Warning level events
	Warning *log.Logger
	// Error provides a handle to a std logger to use for Error level events
	Error *log.Logger
	mutex *sync.Mutex
)

type sessionInfo struct {
	Protocol string
	ID       int32
	Portal   string
	IQN      string
	Name     string
}

func issueCmd(name string, args ...string) ([]byte, error) {
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		Warning.Printf("cmd: %s %s, returned an error: %s", name, args, err)
	}
	return out, err
}

func init() {
	Info = log.New(os.Stdout,
		"INFO: ",
		log.Ldate|log.Ltime|log.Lshortfile)
	Warning = log.New(os.Stdout,
		"WARNING: ",
		log.Ldate|log.Ltime|log.Lshortfile)
	Error = log.New(os.Stderr,
		"ERROR: ",
		log.Ldate|log.Ltime|log.Lshortfile)
	_, err := issueCmd("iscsiadm", "-h")
	if err != nil {
		Error.Fatalf("check for iscsiadm failed, unable to init package: %v", err)
	}
}

func parseSessions(lines []byte) ([]sessionInfo, error) {
	entries := strings.Split(strings.TrimSpace(string(lines)), "\n")
	r := strings.NewReplacer("[", "",
		"]", "")

	var sessions []sessionInfo
	for _, entry := range entries {
		e := strings.Fields(entry)
		if len(e) < 4 {
			Info.Printf("won't process session info with less than 4 fields: %v", e)
			continue
		}
		protocol := e[0]
		id := r.Replace(e[1])
		id64, _ := strconv.ParseInt(id, 10, 32)

		s := sessionInfo{
			Protocol: protocol,
			ID:       int32(id64),
			Portal:   string(e[2]),
			IQN:      e[3],
			Name:     strings.Split(e[3], ":")[1],
		}
		sessions = append(sessions, s)
	}
	return sessions, nil

}

func getCurrentSessions() ([]sessionInfo, error) {
	out, err := issueCmd("iscsiadm", "-m", "session")
	if err != nil {
		Error.Printf("failed to get current iSCSI sessions: %s", err)
		return nil, err
	}
	sessions, err := parseSessions(out)
	return sessions, err
}

func sessionExists(tgtIQN string) []sessionInfo {
	sessions, err := getCurrentSessions()
	if err != nil {

	}
	var existingSessions []sessionInfo
	for _, s := range sessions {
		if tgtIQN == s.IQN {
			existingSessions = append(existingSessions, s)
		}
	}
	return existingSessions

}

// ConnectMultipathVolume is used specifically for multipath support
func ConnectMultipathVolume() {
	errors.New("not implemented")
}

// ConnectVolume attempts to connect the volume to this host
func ConnectVolume(tPortal, tIqn string, tLun int32) {
	// we lock around connect and disconnect due to races on busy systems
	mutex.Lock()
	defer mutex.Unlock()

	// First make sure we're not already connected
	sessions := sessionExists(tIqn)
	if len(sessions) > 0 {
		// We already have existing iscsi session(s) for this iqn, let's check for a device
		// and such that we can pass back and be done

	}

}

// DisconnectVolume attempts to disconnect a connected Volume, returns "ok" if volume is not connected
func DisconnectVolume() {
	// we lock around connect and disconnect due to races on busy systems
	mutex.Lock()
	defer mutex.Unlock()

}

// GetInitiators provides a helper to gather initiator info from the Node
func GetInitiators() {

}

// GetSessions will query the system for active/open iscsi sessions for the specified tgt
func GetSessions() {

}

// DoDiscovery will issue a std iscsiadm discovery on the Node
func DoDiscovery() {

}

// IsTgtDiscovered checks if we're able to discover the tgt
func IsTgtDiscovered() {

}

// GetDevice will retrieve the device file of a connected Volume
func GetDevice() {

}
