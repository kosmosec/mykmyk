package nmap

import (
	"fmt"
	"os"

	nmapWrapper "github.com/Ullaakut/nmap/v3"
)

type cache struct{}

func (c *cache) get(target string, key string) (*nmapWrapper.Run, bool) {
	path := fmt.Sprintf("./%s/%s.xml", target, key)
	cachedResult, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var scanResult nmapWrapper.Run
	err = nmapWrapper.Parse(cachedResult, &scanResult)
	if err != nil {
		return nil, false
	}
	return &scanResult, true
}
