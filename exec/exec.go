package exec

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
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

func RunTemplatedScript(path, fileName, script string, args interface{}) (string, error) {
	if _, err := os.Stat(path); err != nil {
		return "", err
	}
	scriptPath := filepath.Join(path, fileName)
	f, err := os.Create(scriptPath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if err := os.Chmod(scriptPath, 0755); err != nil {
		return "", err
	}
	tmpl, err := template.New(script).Parse(script)
	if err != nil {
		return "", err
	}
	if err := tmpl.Execute(f, args); err != nil {
		return "", err
	}
	output, err := RunCommand(path, "bash", "./"+fileName)
	if err != nil {
		return "", err
	}
	return output, nil
}
