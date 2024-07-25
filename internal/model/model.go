package model

import (
	"github.com/kosmosec/mykmyk/internal/api"
)

type Output struct {
	Type    api.TaskType
	Name    string
	Target  string
	Results Results
}

type Results struct {
	Data        []string
	ReportPaths []string
}

type Message struct {
	Targets []string
	Ports   []string
}
