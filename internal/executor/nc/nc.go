package nc

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

type Nc struct {
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
	args          []string
}

type scanResult struct {
	target string
	result string
	err    error
}

func (n *Nc) Run(ctx context.Context, in interface{}, db *sql.DB) error {
	task, err := n.unmarshal(in)
	if err != nil {
		return err
	}

	n.args = task.Args

	resultCh := make(chan scanResult, n.Concurrency)
	var wg sync.WaitGroup
	limiter := make(chan bool, n.Concurrency)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-n.consumer:
				if !ok {
					wg.Wait()
					close(resultCh)
					return
				}
				n.once.Do(func() {
					n.waitForTask()
				})
				limiter <- true
				wg.Add(1)
				target := msg.Targets[0]
				go func(ctx context.Context, targe string, msg model.Message) {
					defer func() {
						wg.Done()
						<-limiter
					}()
					for {
						select {
						case <-ctx.Done():
							return
						default:
							result, err := n.scanTarget(target, msg, db)
							resultCh <- scanResult{target: target, result: result, err: err}
							return
						}
					}

				}(ctx, target, msg)
			}
		}
		// for msg := range n.consumer {
		// 	n.once.Do(func() {
		// 		n.waitForTask()
		// 	})
		// 	limiter <- true
		// 	wg.Add(1)
		// 	target := msg.Targets[0]
		// 	go func(targe string, msg model.Message) {
		// 		defer func() {
		// 			wg.Done()
		// 			<-limiter
		// 		}()
		// 		result, err := n.scanTarget(target, msg)
		// 		resultCh <- scanResult{target: target, result: result, err: err}
		// 	}(target, msg)
		// }
		// wg.Wait()
		// close(resultCh)
	}()

	for r := range resultCh {
		if r.err != nil {
			return err
		}
		// maybe without liveOutput or parametirezd true/false
		n.saveToFile(r.target, r.result)
		liveOutput(n.Name, r.target, r.result)
		n.collectOutput(r.target, r.result)
	}
	n.signalDoneTask()
	return nil
}

func (n *Nc) New(task api.Task, sns *sns.SNS, creds credsmanager.Credentials) abstract.Executor {
	return &Nc{
		Type:          task.Type,
		Name:          task.Name,
		Source:        task.Source,
		Concurrency:   task.Concurrency,
		waitFor:       task.WaitFor,
		isActive:      task.Active,
		isCacheActive: task.UseCache,
		output:        make([]model.Output, 0),
		sns:           sns,
	}
}

func (n *Nc) GetConcurrency() int {
	return n.Concurrency
}

func (n *Nc) SetDoneSignal(signalDone chan bool) {
	n.signalDone = signalDone
}

func (n *Nc) SetWaitForSignal(waitForSignal chan bool) {
	n.waitForSignal = waitForSignal
}

func (n *Nc) GetWaitFor() string {
	return n.waitFor
}

func (n *Nc) GetWaitForSignal() chan bool {
	return n.waitForSignal
}

func (n *Nc) GetDoneSignal() chan bool {
	return n.signalDone
}

func (n *Nc) Output() []model.Output {
	return n.output
}

func (n *Nc) GetType() api.TaskType {
	return n.Type
}

func (n *Nc) GetSource() string {
	return n.Source
}
func (n *Nc) HasSource() bool {
	if n.Source != "" {
		return true
	}
	return false
}

func (n *Nc) GetName() string {
	return n.Name
}

func (n *Nc) SetConsumer(c chan model.Message) {
	n.consumer = c
}

func (n *Nc) IsActive() bool {
	return n.isActive
}

func (n *Nc) signalDoneTask() {
	if n.signalDone != nil {
		n.signalDone <- true
	}
}

func (n *Nc) unmarshal(in interface{}) (*Task, error) {
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

func (n *Nc) waitForTask() {
	if n.waitForSignal != nil {
		<-n.waitForSignal
	}
}

func (n *Nc) scanTarget(target string, msg model.Message, db *sql.DB) (string, error) {
	fmt.Printf("[+] Netcat scanning for %s started\n", target)
	cached, found := n.cache.get(target, n.Name)
	if n.isCacheActive && found {
		return cached, nil
	}
	for _, portToStatus := range msg.Ports {
		err := status.AddTaskToStatus(db, n.Name, target, fmt.Sprintf("%s:%s", target, portToStatus))
		if err != nil {
			return "", err
		}
	}
	output, err := scan(target, msg.Ports, n.args, n.Name, db)
	if err != nil {
		return "", err
	}
	return output, nil
}

func (n *Nc) saveToFile(target string, output string) error {
	path := fmt.Sprintf("./%s/%s", target, n.Name)
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
	fmt.Printf("Netcat scan %s, output for %s\n%s", taskName, target, output)
}

func (n *Nc) collectOutput(target string, output string) {
	r := strings.Split(output, "\n")
	o := model.Output{
		Type:    n.Type,
		Name:    n.Name,
		Target:  target,
		Results: model.Results{Data: r},
	}
	n.mu.Lock()
	n.output = append(n.output, o)
	n.mu.Unlock()

}
