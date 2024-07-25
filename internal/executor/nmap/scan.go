package nmap

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	nmapWrapper "github.com/Ullaakut/nmap/v3"
)

var ErrEmptyNmapScanResult = errors.New("nmap does not found anything")

func scan(target string, ports []string, scanOptions []string, scanName string) (*nmapWrapper.Run, *[]string, error) {

	if _, err := os.Stat(target); os.IsNotExist(err) {
		os.Mkdir(target, 0775)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1440*time.Minute)
	defer cancel()
	nmapArgs := buildScanOptions(scanOptions, target, scanName)
	result, warnings, err := doScan(ctx, target, ports, nmapArgs, scanName)
	if err != nil {
		return nil, nil, err
	}
	return result, warnings, nil
}

func doScan(ctx context.Context, target string, ports []string, scanArgs []string, scanName string) (*nmapWrapper.Run, *[]string, error) {

	scanner, err := createScanner(ctx, target, ports, scanArgs)
	if err != nil {
		return nil, nil, err
	}
	result, warnings, err := scanner.Run()
	if err != nil {
		return nil, nil, err
	}
	result.ToFile(fmt.Sprintf("./%s/%s.xml", target, scanName))
	return result, warnings, nil

}

func createScanner(ctx context.Context, target string, ports []string, scanArgs []string) (*nmapWrapper.Scanner, error) {
	var scanner *nmapWrapper.Scanner
	var err error
	if len(ports) == 0 {
		scanner, err = nmapWrapper.NewScanner(ctx,
			nmapWrapper.WithTargets(target),
			nmapWrapper.WithCustomArguments(scanArgs...),
		)
	} else {
		scanner, err = nmapWrapper.NewScanner(ctx,
			nmapWrapper.WithTargets(target),
			nmapWrapper.WithPorts(ports...),
			nmapWrapper.WithCustomArguments(scanArgs...),
		)
	}
	return scanner, err
}

func buildScanOptions(scanOptions []string, target string, scanName string) []string {
	scanOptions = buildOutput(scanOptions, target, scanName)

	return scanOptions
}

func buildOutput(scanOptions []string, host string, name string) []string {
	scanOptions = append(scanOptions, fmt.Sprintf("%s/%s", host, name))
	return scanOptions
}
