package httpx

import (
	"bufio"
	"fmt"
	"os"
)

type cache struct{}

func (c *cache) get(target string, key string) ([]string, bool) {
	path := fmt.Sprintf("./%s/%s", target, key)
	file, err := os.Open(path)
	if err != nil {
		return nil, false
	}
	defer file.Close()
	var urls []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		urls = append(urls, scanner.Text())
	}
	return urls, true

}
