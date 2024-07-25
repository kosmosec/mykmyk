package filesystem

import (
	"fmt"
	"io/ioutil"
	"strings"
)

func loadScope(filePath string) ([]string, error) {
	rawScope, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read scope from file %v: %w", filePath, err)
	}
	targets := strings.Split(string(rawScope), "\n")
	targetList := make([]string, 0)
	for _, t := range targets {
		if t != "" {
			targetList = append(targetList, strings.TrimSpace(t))
		}
	}
	return targetList, nil
}
