package smb

import (
	"context"
	"database/sql"
	"fmt"
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

type Smb struct {
	Type          api.TaskType
	Name          string
	Source        string
	Concurrency   int
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
	credsManager  credsmanager.Credentials
	args          []string
}

type scanResult struct {
	target string
	result string
	err    error
}

func (s *Smb) Run(ctx context.Context, in interface{}, db *sql.DB) error {
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
				s.once.Do((func() {
					s.waitForTask()
				}))
				target := msg.Targets[0]
				if isSMBPortExists(msg.Ports) {
					limiter <- true
					wg.Add(1)
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
								result, err := s.scanTarget(target, msg, db)
								resultCh <- scanResult{target: target, result: result, err: err}
								return
							}
						}

					}(ctx, target, msg)
				} else {
					resultCh <- scanResult{target: target, result: "", err: nil}
				}
			}
		}
		// for msg := range s.consumer {
		// 	s.once.Do((func() {
		// 		s.waitForTask()
		// 	}))
		// 	target := msg.Targets[0]
		// 	if isSMBPortExists(msg.Ports) {
		// 		limiter <- true
		// 		wg.Add(1)
		// 		go func(target string, msg model.Message) {
		// 			defer func() {
		// 				wg.Done()
		// 				<-limiter
		// 			}()
		// 			result, err := s.scanTarget(target, msg)
		// 			resultCh <- scanResult{target: target, result: result, err: err}
		// 		}(target, msg)
		// 	} else {
		// 		resultCh <- scanResult{target: target, result: "", err: nil}
		// 	}
		// }
		// wg.Wait()
		// close(resultCh)
	}()

	for r := range resultCh {
		if r.err != nil {
			return err
		}
		s.saveToFile(r.target, r.result)
		liveOutput(s.Name, r.target, r.result)
		s.collectOutput(r.target, r.result)
	}
	s.signalDoneTask()
	return nil
}

func (s *Smb) New(task api.Task, sns *sns.SNS, creds credsmanager.Credentials) abstract.Executor {
	return &Smb{
		Type:          task.Type,
		Name:          task.Name,
		Source:        task.Source,
		Concurrency:   task.Concurrency,
		waitFor:       task.WaitFor,
		isActive:      task.Active,
		isCacheActive: task.UseCache,
		output:        make([]model.Output, 0),
		credsManager:  creds,
		sns:           sns,
	}
}

func (s *Smb) GetConcurrency() int {
	return s.Concurrency
}

func (s *Smb) SetDoneSignal(signalDone chan bool) {
	s.signalDone = signalDone
}

func (s *Smb) SetWaitForSignal(waitForSignal chan bool) {
	s.waitForSignal = waitForSignal
}

func (s *Smb) GetWaitFor() string {
	return s.waitFor
}

func (s *Smb) GetWaitForSignal() chan bool {
	return s.waitForSignal
}

func (s *Smb) GetDoneSignal() chan bool {
	return s.signalDone
}

func (s *Smb) Output() []model.Output {
	return s.output
}

func (s *Smb) GetType() api.TaskType {
	return s.Type
}

func (s *Smb) GetSource() string {
	return s.Source
}
func (s *Smb) HasSource() bool {
	if s.Source != "" {
		return true
	}
	return false
}

func (s *Smb) GetName() string {
	return s.Name
}

func (s *Smb) SetConsumer(c chan model.Message) {
	s.consumer = c
}

func (s *Smb) IsActive() bool {
	return s.isActive
}

func (s *Smb) unmarshal(in interface{}) (*Task, error) {
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

func (s *Smb) waitForTask() {
	if s.waitForSignal != nil {
		<-s.waitForSignal
	}
}
func (s *Smb) signalDoneTask() {
	if s.signalDone != nil {
		s.signalDone <- true
	}
}

func isSMBPortExists(ports []string) bool {
	for _, p := range ports {
		if p == "445" {
			return true
		}
	}
	return false
}

func (s *Smb) scanTarget(target string, msg model.Message, db *sql.DB) (string, error) {
	cached, found := s.cache.get(target, s.Name)
	if s.isCacheActive && found {
		return cached, nil
	}
	err := status.AddTaskToStatus(db, s.Name, target, target)
	if err != nil {
		return "", err
	}
	output, err := scan(target, s.args, s.credsManager)
	if err != nil {
		return "", err
	}
	err = status.UpdateDoneTaskInStatus(db, s.Name, target, target)
	if err != nil {
		return "", err
	}
	return output, nil
}

func (s *Smb) saveToFile(target string, output string) error {
	path := fmt.Sprintf("./%s/%s", target, s.Name)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err = f.WriteString(output + "\n"); err != nil {
		return err
	}

	return nil
}

func liveOutput(taskName string, target string, output string) {
	fmt.Printf("SMB scan %s, output for %s\n%s\n", taskName, target, output)
}

func (s *Smb) collectOutput(target string, output string) {
	r := strings.Split(output, "\n")
	o := model.Output{
		Type:    s.Type,
		Name:    s.Name,
		Target:  target,
		Results: model.Results{Data: r},
	}
	s.mu.Lock()
	s.output = append(s.output, o)
	s.mu.Unlock()
}
