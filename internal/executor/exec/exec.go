// Package exec is based on porter implementation
package exec

import (
	"context"
	"database/sql"
	"io/ioutil"
	"strings"
	"sync"

	"get.porter.sh/porter/pkg/exec"
	"get.porter.sh/porter/pkg/exec/builder"
	"get.porter.sh/porter/pkg/yaml"
	"github.com/kosmosec/mykmyk/internal/api"
	"github.com/kosmosec/mykmyk/internal/credsmanager"
	"github.com/kosmosec/mykmyk/internal/executor/abstract"
	"github.com/kosmosec/mykmyk/internal/model"
	"github.com/kosmosec/mykmyk/internal/sns"
)

type Exec struct {
	Type          api.TaskType
	Name          string
	Source        string
	Concurrency   int
	isActive      bool
	isCacheActive bool
	waitFor       string
	waitForSignal chan bool
	signalDone    chan bool
	output        []model.Output
	mu            sync.Mutex
	sns           *sns.SNS
	consumer      chan model.Message
}

func (e *Exec) Output() []model.Output {
	return e.output
}

func (e *Exec) Run(ctx context.Context, in interface{}, db *sql.DB) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			e.waitForTask()
			task, err := e.unmarshal(in)
			if err != nil {
				return err
			}

			m := exec.New()

			m.Out = ioutil.Discard
			output, err := builder.ExecuteStep(m.Context, task)
			if err != nil {
				return err
			}
			result := model.Output{
				Type:    e.Type,
				Name:    e.Name,
				Target:  strings.Join(task.GetArguments(), " "),
				Results: model.Results{Data: []string{output}},
			}
			e.mu.Lock()
			e.output = append(e.output, result)
			e.mu.Unlock()
			// // TODO: collect which output should be run
			// err = ProcessJsonPathOutputs(m.Context, task, output)
			// if err != nil {
			// 	return err
			// }

			// // TODO: copy other methods and get rid of saving to file...
			// err = builder.ProcessRegexOutputs(m.Context, task, output)
			// if err != nil {
			// 	return err
			// }
			// err = builder.ProcessFileOutputs(m.Context, task)
			e.signalDoneTask()
			return nil
		}
	}

}

func (e *Exec) unmarshal(in interface{}) (*Task, error) {
	var execTask Task
	raw, err := yaml.Marshal(in)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(raw, &execTask)
	if err != nil {
		return nil, err
	}

	return &execTask, nil
}

func (e *Exec) New(task api.Task, sns *sns.SNS, creds credsmanager.Credentials) abstract.Executor {
	return &Exec{
		Type:          task.Type,
		Name:          task.Name,
		Source:        task.Source,
		waitFor:       task.WaitFor,
		Concurrency:   task.Concurrency,
		isActive:      task.Active,
		isCacheActive: task.UseCache,
		output:        make([]model.Output, 0),
		sns:           sns,
	}
}

func (e *Exec) GetConcurrency() int {
	return e.Concurrency
}

func (e *Exec) signalDoneTask() {
	if e.signalDone != nil {
		e.signalDone <- true
	}
}

func (e *Exec) waitForTask() {
	if e.waitForSignal != nil {
		<-e.waitForSignal
	}
}

func (e *Exec) SetDoneSignal(signalDone chan bool) {
	e.signalDone = signalDone
}

func (e *Exec) SetWaitForSignal(waitForSignal chan bool) {
	e.waitForSignal = waitForSignal
}

func (e *Exec) GetWaitForSignal() chan bool {
	return e.waitForSignal
}

func (e *Exec) GetDoneSignal() chan bool {
	return e.signalDone
}

func (e *Exec) GetWaitFor() string {
	return e.waitFor
}

func (e *Exec) GetType() api.TaskType {
	return e.Type
}

func (e *Exec) GetSource() string {
	return e.Source
}

func (e *Exec) HasSource() bool {
	if e.Source != "" {
		return true
	}
	return false
}

func (e *Exec) GetName() string {
	return e.Name
}

func (e *Exec) SetConsumer(c chan model.Message) {
	e.consumer = c
}

func (e *Exec) IsActive() bool {
	return e.isActive
}
