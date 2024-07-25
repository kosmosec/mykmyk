package abstract

import (
	"context"
	"database/sql"

	"github.com/kosmosec/mykmyk/internal/api"
	"github.com/kosmosec/mykmyk/internal/model"
)

type Executor interface {
	Run(ctx context.Context, in interface{}, db *sql.DB) error
	Output() []model.Output
	GetSource() string
	HasSource() bool
	IsActive() bool
	GetName() string
	GetType() api.TaskType
	SetWaitForSignal(chan bool)
	GetWaitForSignal() chan bool
	GetWaitFor() string
	SetDoneSignal(chan bool)
	GetDoneSignal() chan bool
	GetConcurrency() int
	SetConsumer(c chan model.Message)
}
