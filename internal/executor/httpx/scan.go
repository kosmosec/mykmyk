package httpx

import (
	"bytes"
	"fmt"
	"os"

	"github.com/kosmosec/mykmyk/internal/binary"
)

func scan(host string, ports []string, args []string) (string, error) {
	if _, err := os.Stat(host); os.IsNotExist(err) {
		os.Mkdir(host, 0775)
	}
	input := createInput(host, ports)

	output, _, err := binary.Run("httpx", args, &input)
	if err != nil {
		return "", err
	}
	return output.String(), nil
}

func createInput(host string, ports []string) bytes.Buffer {
	input := bytes.Buffer{}
	for _, p := range ports {
		hostWithPort := fmt.Sprintf("%s:%s\n", host, p)
		input.Write([]byte(hostWithPort))
	}
	return input
}
