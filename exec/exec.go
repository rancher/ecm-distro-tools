package exec

import (
	"bytes"
	"errors"
	"os/exec"
)

func RunCommand(dir, cmd string, args ...string) (string, error) {
	command := exec.Command(cmd, args...)

	var outb, errb bytes.Buffer
	command.Stdout = &outb
	command.Stderr = &errb
	command.Dir = dir
	if err := command.Run(); err != nil {
		return "", errors.New(errb.String())
	}

	return outb.String(), nil
}
