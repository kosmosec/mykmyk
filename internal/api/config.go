package api

const DefaultConfigName = "config.yml"

type (
	Config struct {
		Workflow   Workflow `yaml:"workflow,omitempty"`
		OutputFile string   `yaml:"outputFile,omitempty"`
	}
	Workflow struct {
		Tasks []Task `yaml:"tasks,omitempty"`
	}

	Task struct {
		Name        string      `yaml:"name,omitempty"`
		Type        TaskType    `yaml:"type,omitempty"`
		Active      bool        `yaml:"active,omitempty"`
		UseCache    bool        `yaml:"useCache,omitempty"`
		Source      string      `yaml:"source,omitempty"`
		Concurrency int         `yaml:"concurrency,omitempty"`
		WaitFor     string      `yaml:"waitFor,omitempty"`
		Run         interface{} `yaml:"run,omitempty"`
		Prefix      string      `yaml:"prefix,omitempty"`
		Credentials bool        `yaml:"credentials,omitempty"`
	}
)
