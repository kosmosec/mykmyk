package nc

import (
	"fmt"
	"io/ioutil"
)

type cache struct{}

func (c *cache) get(target string, key string) (string, bool) {
	path := fmt.Sprintf("./%s/%s", target, key)
	output, err := ioutil.ReadFile(path)
	if err != nil {
		return "", false
	}
	return string(output), true
}
