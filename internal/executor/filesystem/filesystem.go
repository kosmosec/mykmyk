package filesystem

import (
	"context"
	"database/sql"

	"github.com/kosmosec/mykmyk/internal/api"
	"github.com/kosmosec/mykmyk/internal/credsmanager"
	"github.com/kosmosec/mykmyk/internal/executor/abstract"
	"github.com/kosmosec/mykmyk/internal/model"
	"github.com/kosmosec/mykmyk/internal/sns"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type Filesystem struct {
	Type          api.TaskType
	Name          string
	Source        string
	Concurrency   int
	waitForSignal chan bool
	waitFor       string
	signalDone    chan bool
	isActive      bool
	isCacheActive bool
	output        []model.Output
	sns           *sns.SNS
	consumer      chan model.Message
}

func (f *Filesystem) Run(ctx context.Context, in interface{}, db *sql.DB) error {
	f.waitForTask()
	task, err := f.unmarshal(in)
	if err != nil {
		return err
	}

	switch f.Name {
	case "scope":
		scope, err := loadScope(task.Input)
		if err != nil {
			return errors.Errorf("unable to load scope", err)
		}
		for _, s := range scope {
			m := model.Message{Targets: []string{s}}
			f.sns.SendMessage(f.Name, m)
		}
		f.sns.CloseTopic(f.Name)
	default:
		return errors.Errorf("unsupported filesystem task", err)
	}
	f.signalDoneTask()

	return nil
}

func (f *Filesystem) signalDoneTask() {
	if f.signalDone != nil {
		f.signalDone <- true
	}
}

func (f *Filesystem) waitForTask() {
	if f.waitForSignal != nil {
		<-f.waitForSignal
	}
}

func (f *Filesystem) New(task api.Task, sns *sns.SNS, creds credsmanager.Credentials) abstract.Executor {
	return &Filesystem{
		Type:          task.Type,
		Name:          task.Name,
		Source:        task.Source,
		waitFor:       task.WaitFor,
		Concurrency:   task.Concurrency,
		output:        make([]model.Output, 0),
		isActive:      task.Active,
		isCacheActive: task.UseCache,
		sns:           sns,
	}
}

func (f *Filesystem) GetConcurrency() int {
	return f.Concurrency
}

func (f *Filesystem) SetDoneSignal(signalDone chan bool) {
	f.signalDone = signalDone
}

func (f *Filesystem) SetWaitForSignal(waitForSignal chan bool) {
	f.waitForSignal = waitForSignal
}

func (f *Filesystem) GetWaitForSignal() chan bool {
	return f.waitForSignal
}

func (f *Filesystem) GetDoneSignal() chan bool {
	return f.signalDone
}

func (f *Filesystem) GetWaitFor() string {
	return f.waitFor
}

func (f *Filesystem) Output() []model.Output {
	return f.output
}

func (f *Filesystem) GetType() api.TaskType {
	return f.Type
}

func (f *Filesystem) GetSource() string {
	return f.Source
}

func (f *Filesystem) HasSource() bool {
	if f.Source != "" {
		return true
	}
	return false
}

func (f *Filesystem) GetName() string {
	return f.Name
}

func (f *Filesystem) unmarshal(in interface{}) (*Task, error) {
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

func (f *Filesystem) SetConsumer(c chan model.Message) {
	f.consumer = c
}

func (f *Filesystem) IsActive() bool {
	return f.isActive
}
