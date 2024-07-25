package finder

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	nmapWrapper "github.com/Ullaakut/nmap/v3"
)

func Find(ctx context.Context, targetFile string, portFilter string) error {
	targets, err := loadScope(targetFile)
	if err != nil {
		return err
	}
	for _, t := range targets {
		nmapScan, err := loadScan(t, "ST-scan")
		if err != nil {
			return err
		}
		for _, host := range nmapScan.Hosts {
			for _, p := range host.Ports {
				scannedPort := strconv.Itoa(int(p.ID))
				if scannedPort == portFilter {
					fmt.Println(t)
				}
			}
		}
	}
	return nil
}

func loadScan(target string, key string) (*nmapWrapper.Run, error) {
	path := fmt.Sprintf("./%s/%s.xml", target, key)
	cachedResult, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var scanResult nmapWrapper.Run
	err = nmapWrapper.Parse(cachedResult, &scanResult)
	if err != nil {
		return nil, err
	}
	return &scanResult, err
}

func loadScope(filePath string) ([]string, error) {
	rawScope, err := os.ReadFile(filePath)
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
