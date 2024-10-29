package rdp

import (
	"fmt"
	"os"
)

type cache struct{}

func (c *cache) get(target string, key string) (string, bool) {
	path := fmt.Sprintf("./%s/%s", target, key)
	output, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	return string(output), true
}
