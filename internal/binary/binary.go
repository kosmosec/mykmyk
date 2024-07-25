package binary

import (
	"bytes"
	"io"
	"os/exec"
)

// Run run binaryName with args and stdin optinally.
func Run(binaryName string, args []string, stdin io.Reader) (*bytes.Buffer, *bytes.Buffer, error) {

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	binaryPath, err := exec.LookPath(binaryName)
	if err != nil {
		return nil, nil, err
	}

	cmd := exec.Command(binaryPath, args...)
	if stdin != nil {
		cmd.Stdin = stdin
	}
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Start()
	if err != nil {
		return nil, nil, err
	}

	cmd.Wait()

	return &stdout, &stderr, nil
}
