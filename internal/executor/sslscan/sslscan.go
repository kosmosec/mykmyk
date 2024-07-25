package sslscan

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/kosmosec/mykmyk/internal/api"
	"github.com/kosmosec/mykmyk/internal/credsmanager"
	"github.com/kosmosec/mykmyk/internal/executor/abstract"
	"github.com/kosmosec/mykmyk/internal/model"
	"github.com/kosmosec/mykmyk/internal/sns"
	"github.com/kosmosec/mykmyk/internal/status"
	"gopkg.in/yaml.v2"
)

type SSLScan struct {
	Type          api.TaskType
	Name          string
	Source        string
	Concurrency   int
	prefix        string
	waitForSignal chan bool
	signalDone    chan bool
	waitFor       string
	output        []model.Output
	mu            sync.Mutex
	once          sync.Once
	cache         cache
	isActive      bool
	isCacheActive bool
	sns           *sns.SNS
	consumer      chan model.Message
	args          []string
}

type scanResult struct {
	target        string
	result        string
	pathsToReport []string
	err           error
}

func (s *SSLScan) Run(ctx context.Context, in interface{}, db *sql.DB) error {
	task, err := s.unmarshal(in)
	if err != nil {
		return err
	}

	s.args = task.Args

	resultCh := make(chan scanResult, s.Concurrency)
	var wg sync.WaitGroup
	limiter := make(chan bool, s.Concurrency)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-s.consumer:
				if !ok {
					wg.Wait()
					close(resultCh)
					return
				}
				s.once.Do(func() {
					s.waitForTask()
				})
				limiter <- true
				wg.Add(1)
				target, err := getHost(msg.Targets[0])
				if err != nil {
					resultCh <- scanResult{target: target, err: err}
					return
				}
				go func(ctx context.Context, target string, msg model.Message) {
					defer func() {
						wg.Done()
						<-limiter
					}()
					for {
						select {
						case <-ctx.Done():
							return
						default:
							result, pathsToReport, err := s.scanTarget(target, msg, db)
							resultCh <- scanResult{target: target, result: result, pathsToReport: pathsToReport, err: err}
							return
						}
					}

				}(ctx, target, msg)
			}
		}
	}()

	outputToFile := make(map[string]string)
	for r := range resultCh {
		if r.err != nil {
			return err
		}
		liveOutput(s.Name, r.target, r.result)
		outputToFile[r.target] = r.result
		s.collectOutput(r.target, r.result, r.pathsToReport)
	}
	s.saveToFile(outputToFile)
	s.signalDoneTask()
	return nil
}

func (s *SSLScan) New(task api.Task, sns *sns.SNS, creds credsmanager.Credentials) abstract.Executor {
	return &SSLScan{
		Type:          task.Type,
		Name:          task.Name,
		Source:        task.Source,
		Concurrency:   task.Concurrency,
		waitFor:       task.WaitFor,
		isActive:      task.Active,
		isCacheActive: task.UseCache,
		prefix:        task.Prefix,
		output:        make([]model.Output, 0),
		sns:           sns,
	}
}

func (s *SSLScan) GetConcurrency() int {
	return s.Concurrency
}

func (s *SSLScan) SetDoneSignal(signalDone chan bool) {
	s.signalDone = signalDone
}

func (s *SSLScan) SetWaitForSignal(waitForSignal chan bool) {
	s.waitForSignal = waitForSignal
}

func (s *SSLScan) GetWaitFor() string {
	return s.waitFor
}

func (s *SSLScan) GetWaitForSignal() chan bool {
	return s.waitForSignal
}

func (s *SSLScan) GetDoneSignal() chan bool {
	return s.signalDone
}

func (s *SSLScan) Output() []model.Output {
	return s.output
}

func (s *SSLScan) GetType() api.TaskType {
	return s.Type
}

func (s *SSLScan) GetSource() string {
	return s.Source
}
func (s *SSLScan) HasSource() bool {
	if s.Source != "" {
		return true
	}
	return false
}

func (s *SSLScan) GetName() string {
	return s.Name
}

func (s *SSLScan) SetConsumer(c chan model.Message) {
	s.consumer = c
}

func (s *SSLScan) IsActive() bool {
	return s.isActive
}

func (s *SSLScan) unmarshal(in interface{}) (*Task, error) {
	var task Task
	raw, err := yaml.Marshal(in)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(raw, &task); err != nil {
		return nil, err
	}
	return &task, err
}

func (s *SSLScan) waitForTask() {
	if s.waitForSignal != nil {
		<-s.waitForSignal
	}
}

func getHost(rawUrl string) (string, error) {
	u, err := url.Parse(rawUrl)
	if err != nil {
		return "", err
	}

	h := strings.Split(u.Host, ":")
	if len(h) == 1 {
		return h[0], nil
	} else {
		host, _, _ := net.SplitHostPort(u.Host)
		return host, nil
	}
}

func (s *SSLScan) scanTarget(target string, msg model.Message, db *sql.DB) (string, []string, error) {
	fmt.Printf("[+] SSLscan scanning for %s started\n", target)
	cached, reportPaths, found := s.cache.get(target, s.Name)
	if s.isCacheActive && found {
		return cached, reportPaths, nil
	}
	for _, targetToStatus := range msg.Targets {
		err := status.AddTaskToStatus(db, s.Name, target, targetToStatus)
		if err != nil {
			return "", nil, err
		}
	}
	result, pathsToReport, err := scan(target, msg.Targets, s.args, db, s.Name)
	if err != nil {
		return "", nil, err
	}
	return result, pathsToReport, nil
}

func liveOutput(taskName string, target string, result string) {
	fmt.Printf("SSLScan scan %s, output for %s\n%s\n", taskName, target, result)
}

func (s *SSLScan) saveToFile(toSave map[string]string) error {
	for target, outputs := range toSave {
		path := fmt.Sprintf("./%s/%s", target, s.Name)
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		defer f.Close()
		if _, err = f.WriteString(outputs); err != nil {
			return err
		}
	}
	return nil
}

func (s *SSLScan) collectOutput(target string, scanned string, pathsToReport []string) {
	f := strings.Split(scanned, "\n")
	r := model.Results{
		Data:        f,
		ReportPaths: pathsToReport,
	}
	output := model.Output{
		Type:    s.Type,
		Name:    s.Name,
		Target:  target,
		Results: r,
	}
	s.mu.Lock()
	s.output = append(s.output, output)
	s.mu.Unlock()
}

func (s *SSLScan) signalDoneTask() {
	if s.signalDone != nil {
		s.signalDone <- true
	}
}
