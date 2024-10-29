package rdp

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
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

type Rdp struct {
	Type          api.TaskType
	Name          string
	Source        string
	Port          int
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

func (r *Rdp) Run(ctx context.Context, in interface{}, db *sql.DB) error {
	task, err := r.unmarshal(in)
	if err != nil {
		return err
	}

	r.args = task.Args
	resultCh := make(chan scanResult, r.Concurrency)
	var wg sync.WaitGroup
	limiter := make(chan bool, r.Concurrency)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-r.consumer:
				if !ok {
					wg.Wait()
					close(resultCh)
					return
				}
				r.once.Do((func() {
					r.waitForTask()
				}))
				target := msg.Targets[0]
				if isRDPPortExists(r.Port, msg.Ports) {
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
								result, err := r.scanTarget(target, r.Port, msg, db)
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
	}()

	for rMsg := range resultCh {
		if rMsg.err != nil {
			return err
		}
		r.saveToFile(rMsg.target, rMsg.result)
		liveOutput(r.Name, rMsg.target, rMsg.result)
		r.collectOutput(rMsg.target, rMsg.result)
	}
	r.signalDoneTask()
	return nil
}

func (r *Rdp) New(task api.Task, sns *sns.SNS, creds credsmanager.Credentials) abstract.Executor {
	return &Rdp{
		Type:          task.Type,
		Name:          task.Name,
		Source:        task.Source,
		Port:          task.Port,
		Concurrency:   task.Concurrency,
		waitFor:       task.WaitFor,
		isActive:      task.Active,
		isCacheActive: task.UseCache,
		output:        make([]model.Output, 0),
		credsManager:  creds,
		sns:           sns,
	}
}

func (r *Rdp) GetConcurrency() int {
	return r.Concurrency
}

func (r *Rdp) SetDoneSignal(signalDone chan bool) {
	r.signalDone = signalDone
}

func (r *Rdp) SetWaitForSignal(waitForSignal chan bool) {
	r.waitForSignal = waitForSignal
}

func (r *Rdp) GetWaitFor() string {
	return r.waitFor
}

func (r *Rdp) GetWaitForSignal() chan bool {
	return r.waitForSignal
}

func (r *Rdp) GetDoneSignal() chan bool {
	return r.signalDone
}

func (r *Rdp) Output() []model.Output {
	return r.output
}

func (r *Rdp) GetType() api.TaskType {
	return r.Type
}

func (r *Rdp) GetSource() string {
	return r.Source
}
func (r *Rdp) HasSource() bool {
	if r.Source != "" {
		return true
	}
	return false
}

func (r *Rdp) GetName() string {
	return r.Name
}

func (r *Rdp) SetConsumer(c chan model.Message) {
	r.consumer = c
}

func (r *Rdp) IsActive() bool {
	return r.isActive
}

func (r *Rdp) unmarshal(in interface{}) (*Task, error) {
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

func (r *Rdp) waitForTask() {
	if r.waitForSignal != nil {
		<-r.waitForSignal
	}
}
func (r *Rdp) signalDoneTask() {
	if r.signalDone != nil {
		r.signalDone <- true
	}
}

func isRDPPortExists(desiredPort int, ports []string) bool {
	for _, p := range ports {
		pInt, err := strconv.Atoi(p)
		if err != nil {
			return false
		}
		if pInt == desiredPort {
			return true
		}
	}
	return false
}

func (r *Rdp) scanTarget(target string, port int, msg model.Message, db *sql.DB) (string, error) {
	cached, found := r.cache.get(target, r.Name)
	if r.isCacheActive && found {
		return cached, nil
	}
	err := status.AddTaskToStatus(db, r.Name, target, target)
	if err != nil {
		return "", err
	}
	output := scan(target, port, r.args, r.credsManager)
	err = status.UpdateDoneTaskInStatus(db, r.Name, target, target)
	if err != nil {
		return "", err
	}
	return output, nil
}

func (r *Rdp) saveToFile(target string, output string) error {
	path := fmt.Sprintf("./%s/%s", target, r.Name)
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
	fmt.Printf("Rdp scan %s, output for %s\n%s\n", taskName, target, output)
}

func (r *Rdp) collectOutput(target string, output string) {
	result := strings.Split(output, "\n")
	o := model.Output{
		Type:    r.Type,
		Name:    r.Name,
		Target:  target,
		Results: model.Results{Data: result},
	}
	r.mu.Lock()
	r.output = append(r.output, o)
	r.mu.Unlock()
}
