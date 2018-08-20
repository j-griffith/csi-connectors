package iscsi

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/j-griffith/csi-connectors/logger"
)

var (
	log         *logger.Logger
	execCommand = exec.Command
)

type statFunc func(string) (os.FileInfo, error)
type globFunc func(string) ([]string, error)

type secrets struct {
	UserName   string
	Password   string
	UserNameIn string
	PasswordIn string
}

type iscsiSession struct {
	Protocol string
	ID       int32
	Portal   string
	IQN      string
	Name     string
}

//Connector provides a struct to hold all of the needed parameters to make our iscsi connection
type Connector struct {
	VolumeName       string
	TargetIqn        string
	TargetPortals    []string
	Port             string
	Lun              int32
	AuthType         string
	DiscoverySecrets secrets
	SessionSecrets   secrets
	Interface        string
	Multipath        bool
}

func init() {
	// TODO: add a handle to configure loggers after init
	// also, make default for trace to go to discard when you're done messing around
	log = logger.NewLogger(os.Stdout, os.Stdout, os.Stdout, os.Stderr)
}

func runCmd(cmd string, args ...string) (string, error) {
	c := execCommand(cmd, args...)
	out, err := c.CombinedOutput()
	return string(out), err
}

func parseSessions(lines string) []iscsiSession {
	entries := strings.Split(strings.TrimSpace(string(lines)), "\n")
	r := strings.NewReplacer("[", "",
		"]", "")

	var sessions []iscsiSession
	for _, entry := range entries {
		e := strings.Fields(entry)
		if len(e) < 4 {
			continue
		}
		protocol := strings.Split(e[0], ":")[0]
		id := r.Replace(e[1])
		id64, _ := strconv.ParseInt(id, 10, 32)
		portal := strings.Split(e[2], ",")[0]

		s := iscsiSession{
			Protocol: protocol,
			ID:       int32(id64),
			Portal:   portal,
			IQN:      e[3],
			Name:     strings.Split(e[3], ":")[1],
		}
		sessions = append(sessions, s)
	}
	return sessions
}

func sessionExists(tgtPortal, tgtIQN string) (bool, error) {
	sessions, err := getCurrentSessions()
	if err != nil {
		log.Error.Printf("failed to get sessions: %s\n", err.Error())
		return false, err
	}
	var existingSessions []iscsiSession
	for _, s := range sessions {
		if tgtIQN == s.IQN && tgtPortal == s.Portal {
			existingSessions = append(existingSessions, s)
		}
	}
	exists := false
	if len(existingSessions) > 0 {
		exists = true
	}
	return exists, nil
}

func extractTransportName(output string) string {
	res := regexp.MustCompile(`iface.transport_name = (.*)\n`).FindStringSubmatch(output)
	if res == nil {
		return ""
	}
	if res[1] == "" {
		return "tcp"
	}
	return res[1]
}

func getCurrentSessions() ([]iscsiSession, error) {

	out, err := runCmd("iscsiadm", "-m", "session")
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if ok && exitErr.ProcessState.Sys().(syscall.WaitStatus).ExitStatus() == 21 {
			return []iscsiSession{}, nil
		}
		return nil, err
	}
	session := parseSessions(out)
	return session, err
}

func waitForPathToExist(devicePath *string, maxRetries int, deviceTransport string) bool {
	// This makes unit testing a lot easier
	return waitForPathToExistImpl(devicePath, maxRetries, deviceTransport, os.Stat, filepath.Glob)
}

func waitForPathToExistImpl(devicePath *string, maxRetries int, deviceTransport string, osStat statFunc, filepathGlob globFunc) bool {
	if devicePath == nil {
		return false
	}

	for i := 0; i < maxRetries; i++ {
		var err error
		if deviceTransport == "tcp" {
			_, err = osStat(*devicePath)
		} else {
			fpath, _ := filepathGlob(*devicePath)
			if fpath == nil {
				err = os.ErrNotExist
			} else {
				// There might be a case that fpath contains multiple device paths if
				// multiple PCI devices connect to same iscsi target. We handle this
				// case at subsequent logic. Pick up only first path here.
				*devicePath = fpath[0]
			}
		}
		if err == nil {
			return true
		}
		if !os.IsNotExist(err) {
			return false
		}
		if i == maxRetries-1 {
			break
		}
		time.Sleep(time.Second)
	}
	return false
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

// Connect attempts to connect a volume to this node using the provided Connector info
func Connect(c Connector) (string, error) {
	var devicePaths []string
	iFace := "default"
	if c.Interface != "" {
		iFace = c.Interface
	}

	// make sure our iface exists and extract the transport type
	out, err := runCmd("iscsiadm", "-m", "iface", "-I", iFace, "-o", "show")
	if err != nil {
		log.Error.Printf("error in iface show: %s\n", err.Error())
		return "", err
	}
	iscsiTransport := extractTransportName(out)

	for _, p := range c.TargetPortals {
		log.Trace.Printf("process portal: %s\n", p)
		baseArgs := []string{"-m", "node", "-T", c.TargetIqn, "-p", p}

		// create our devicePath that we'll be looking for based on the transport being used
		devicePath := strings.Join([]string{"/dev/disk/by-path/ip", p, "iscsi", c.TargetIqn, "lun", fmt.Sprint(c.Lun)}, "-")
		if iscsiTransport != "tcp" {
			devicePath = strings.Join([]string{"/dev/disk/by-path/pci", "*", "ip", p, "iscsi", c.TargetIqn, "lun", fmt.Sprint(c.Lun)}, "-")
		}

		// TODO: first make sure we're not already connected/logged in
		exists, _ := sessionExists(p, c.TargetIqn)
		if exists {
			log.Info.Printf("found a session, check for device path: %s", devicePath)
			if waitForPathToExist(&devicePath, 1, iscsiTransport) {
				log.Info.Printf("found device path: %s", devicePath)
				devicePaths = append(devicePaths, devicePath)
				continue
			}
		}

		// create db entry
		args := append(baseArgs, []string{"-I", iFace, "-o", "new"}...)
		log.Trace.Printf("create the new record: %s\n", args)
		out, err := runCmd("iscsiadm", args...)
		if err != nil {
			log.Error.Printf("error: %s\n", err.Error())
			continue
		}
		log.Trace.Printf("output from new: %s\n", out)
		if c.AuthType == "chap" {
			args = append(baseArgs, []string{"-o", "update",
				"-n", "node.session.auth.authmethod", "-v", "CHAP",
				"-n", "node.session.auth.username", "-v", c.SessionSecrets.UserName,
				"-n", "node.session.auth.password", "-v", c.SessionSecrets.Password}...)
			if c.SessionSecrets.UserNameIn != "" {
				args = append(args, []string{"-n", "node.session.auth.username_in", "-v", c.SessionSecrets.UserNameIn}...)
			}
			if c.SessionSecrets.UserNameIn != "" {
				args = append(args, []string{"-n", "node.session.auth.password_in", "-v", c.SessionSecrets.PasswordIn}...)
			}
			runCmd("iscsiadm", args...)
		}
		// perform the login
		args = append(baseArgs, []string{"-l"}...)
		runCmd("iscsiadm", args...)
		if waitForPathToExist(&devicePath, 10, iscsiTransport) {
			devicePaths = append(devicePaths, devicePath)
			continue
		}
		devicePath = devicePaths[0]
		for _, path := range devicePaths {
			if path != "" {
				if mappedDevicePath, err := getMultipathDisk(path); mappedDevicePath != "" {
					devicePath = mappedDevicePath
					if err != nil {
						log.Error.Printf("failed to get multipath device path for `%s:` %s", path, err.Error())
						return "", err
					}
					break
				}
			}
		}

	}

	return devicePaths[0], nil
}

//Disconnect performs a disconnect operation on a volume
func Disconnect(tgtIqn string, portals []string) error {
	// FIXME: rework this to just take the volume name, or if we have to the path, and then derive the info we need from that
	baseArgs := []string{"-m", "node", "-T", tgtIqn}
	for _, p := range portals {
		args := append(baseArgs, []string{"-p", p, "-u"}...)
		_, err := runCmd("iscsiadm", args...)
		if err != nil {
			log.Error.Printf("failed to disconnect portal %s: %s", p, err.Error())
			return err
		}
	}
	// finally, delete the entry
	args := append(baseArgs, []string{"-o", "delete"}...)
	_, err := runCmd("iscsiadm", args...)
	if err != nil {
		log.Error.Printf("failed to delete connection to target `%s`: %s", tgtIqn, err.Error())
	}
	return err
}
