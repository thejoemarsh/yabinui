package ssh

import (
	"errors"
	"os/exec"
	"strings"
	"syscall"
)

// Entry is a single saved SSH session.
type Entry struct {
	Name    string
	Command string // raw ssh command line, e.g. "ssh user@host -p 22"
}

// Launch spawns a new terminal window running Entry.Command, detached
// from this process. terminal is a command prefix like "alacritty --hold -e"
// — the ssh command is appended as additional arguments.
func (e Entry) Launch(terminal string) error {
	tparts := strings.Fields(strings.TrimSpace(terminal))
	if len(tparts) == 0 {
		return errors.New("terminal command is empty")
	}
	cparts := strings.Fields(strings.TrimSpace(e.Command))
	if len(cparts) == 0 {
		return errors.New("ssh command is empty")
	}

	args := append([]string{}, tparts[1:]...)
	args = append(args, cparts...)

	cmd := exec.Command(tparts[0], args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	return cmd.Start()
}
