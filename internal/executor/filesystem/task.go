package filesystem

type Task struct {
	Input  string   `yaml:"input,omitempty"`
	Args   []string `yaml:"args,omitempty"`
	Source string   `yaml:"source,omitempty"`
}
