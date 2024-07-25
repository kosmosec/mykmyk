package nuclei

import (
	"bytes"
	"fmt"
	"os"

	"github.com/kosmosec/mykmyk/internal/binary"
)

func scan(host string, urls []string, args []string) (string, error) {
	if _, err := os.Stat(host); os.IsNotExist(err) {
		os.Mkdir(host, 0775)
	}

	input := createInput(urls)
	output, _, err := binary.Run("nuclei", args, &input)
	if err != nil {
		return "", err
	}
	return output.String(), nil

}

func createInput(urls []string) bytes.Buffer {
	input := bytes.Buffer{}
	for _, u := range urls {
		input.WriteString(fmt.Sprintf("%s\n", u))
	}
	return input
}
