package nmap

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strconv"
	"sync"

	nmapWrapper "github.com/Ullaakut/nmap/v3"
	"github.com/kosmosec/mykmyk/internal/api"
	"github.com/kosmosec/mykmyk/internal/credsmanager"
	"github.com/kosmosec/mykmyk/internal/executor/abstract"
	"github.com/kosmosec/mykmyk/internal/model"
	"github.com/kosmosec/mykmyk/internal/sns"
	"github.com/kosmosec/mykmyk/internal/status"
	"gopkg.in/yaml.v2"
)

type Nmap struct {
	Type          api.TaskType
	Name          string
	Source        string
	Concurrency   int
	waitForSignal chan bool
	signalDone    chan bool
	waitFor       string
	output        []model.Output
	mu            sync.Mutex
	cache         cache
	isActive      bool
	isCacheActive bool
	sns           *sns.SNS
	consumer      chan model.Message
	args          []string
}

type scanResult struct {
	target string
	result *nmapWrapper.Run
	err    error
}

func (n *Nmap) scanTarget(target string, msg model.Message, db *sql.DB) (*nmapWrapper.Run, error) {
	fmt.Printf("[+] Nmap scanning for %s started\n", target)
	cached, found := n.cache.get(target, n.Name)
	if n.isCacheActive && found {
		return cached, nil
	}
	err := status.AddTaskToStatus(db, n.Name, target, target)
	if err != nil {
		return nil, err
	}
	result, warnings, err := scan(target, msg.Ports, n.args, n.Name)
	if err != nil {
		return nil, err
	}
	err = status.UpdateDoneTaskInStatus(db, n.Name, target, target)
	if err != nil {
		return nil, err
	}
	log.Printf("nmap: %s for %s", warnings, target)
	return result, nil
}

func (n *Nmap) Run(ctx context.Context, in interface{}, db *sql.DB) error {
	n.waitForTask()
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
				limiter <- true
				wg.Add(1)
				// nmap gets one IP with multiple ports
				target := msg.Targets[0]
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
							result, err := n.scanTarget(target, msg, db)
							resultCh <- scanResult{target: target, result: result, err: err}
							return
						}
					}

				}(ctx, target, msg)
			}
		}
	}()

	for r := range resultCh {
		if r.err != nil && !errors.Is(err, ErrEmptyNmapScanResult) {
			n.sns.CloseTopic(n.Name)
			n.signalDoneTask()
			return err
		}
		liveOutput(n.Name, r.target, r.result)
		n.sendMessageToSNS(r.result, r.target)
		n.collectOutput(r.target, r.result)
	}
	n.sns.CloseTopic(n.Name)
	n.signalDoneTask()
	return nil
}

func liveOutput(taskName string, target string, result *nmapWrapper.Run) {
	output := convertNmapResultToString(result, target)
	fmt.Printf("Nmap scan %s, output for %s\n%s\n", taskName, target, output)
}

func (n *Nmap) collectOutput(target string, nmapResult *nmapWrapper.Run) {
	output := convertNmapResultToString(nmapResult, target)

	result := model.Output{
		Type:    n.Type,
		Name:    n.Name,
		Target:  target,
		Results: model.Results{Data: output},
	}
	n.mu.Lock()
	n.output = append(n.output, result)
	n.mu.Unlock()
}

func (n *Nmap) GetConcurrency() int {
	return n.Concurrency
}

func (n *Nmap) sendMessageToSNS(scanResult *nmapWrapper.Run, host string) error {
	nmapMsg, err := convertToNmapMessage(scanResult, host)
	if err != nil {
		return err
	}
	n.sns.SendMessage(n.Name, nmapMsg)

	return nil
}

func convertToNmapMessage(result *nmapWrapper.Run, target string) (model.Message, error) {
	if result.Hosts == nil {
		return model.Message{}, ErrEmptyNmapScanResult
	}
	ports := make([]string, 0)
	for _, host := range result.Hosts {
		for _, p := range host.Ports {
			ports = append(ports, strconv.Itoa(int(p.ID)))
		}
	}
	return model.Message{Targets: []string{target}, Ports: ports}, nil
}

func convertNmapResultToString(result *nmapWrapper.Run, target string) []string {
	// if result.Hosts == nil {
	// 	return "", ErrEmptyNmapScanResult //errors.New("no hosts in nmap result")
	// }
	//output := bytes.Buffer{}
	output := make([]string, 0)
	for _, host := range result.Hosts {
		for _, p := range host.Ports {
			o := fmt.Sprintf("\t%-10s %-18s %-18s\n", strconv.Itoa(int(p.ID)), p.Service.Name, p.Service.Product)
			//output.Write([]byte(o))
			output = append(output, o)
		}
	}
	return output //output.String()
}

func (n *Nmap) New(task api.Task, sns *sns.SNS, creds credsmanager.Credentials) abstract.Executor {
	return &Nmap{
		Type:          task.Type,
		Name:          task.Name,
		Source:        task.Source,
		Concurrency:   task.Concurrency,
		waitFor:       task.WaitFor,
		isActive:      task.Active,
		isCacheActive: task.UseCache,
		output:        make([]model.Output, 0),
		cache:         cache{},
		sns:           sns,
	}
}

func (n *Nmap) signalDoneTask() {
	if n.signalDone != nil {
		n.signalDone <- true
	}
}

func (n *Nmap) waitForTask() {
	if n.waitForSignal != nil {
		<-n.waitForSignal
	}
}

func (n *Nmap) SetDoneSignal(signalDone chan bool) {
	n.signalDone = signalDone
}

func (n *Nmap) SetWaitForSignal(waitForSignal chan bool) {
	n.waitForSignal = waitForSignal
}

func (n *Nmap) GetWaitForSignal() chan bool {
	return n.waitForSignal
}

func (n *Nmap) GetDoneSignal() chan bool {
	return n.signalDone
}

func (n *Nmap) GetWaitFor() string {
	return n.waitFor
}

func (n *Nmap) GetType() api.TaskType {
	return n.Type
}

func (n *Nmap) Output() []model.Output {
	return n.output
}

func (n *Nmap) GetSource() string {
	return n.Source
}

func (n *Nmap) HasSource() bool {
	if n.Source != "" {
		return true
	}
	return false
}

func (n *Nmap) GetName() string {
	return n.Name
}

func (n *Nmap) unmarshal(in interface{}) (*Task, error) {
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

func (n *Nmap) SetConsumer(c chan model.Message) {
	n.consumer = c
}

func (n *Nmap) IsActive() bool {
	return n.isActive
}
