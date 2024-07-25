package exec

import (
	"get.porter.sh/porter/pkg/exec"
	"get.porter.sh/porter/pkg/exec/builder"
)

type Task exec.Instruction

func (s Task) GetCommand() string {
	return s.Command
}

func (s Task) GetArguments() []string {
	return s.Arguments
}

func (s Task) GetSuffixArguments() []string {
	return s.SuffixArguments
}

func (s Task) GetFlags() builder.Flags {
	return s.Flags
}

func (s Task) SuppressesOutput() bool {
	return s.SuppressOutput
}

func (s Task) GetOutputs() []builder.Output {
	outputs := make([]builder.Output, len(s.Outputs))
	for i := range s.Outputs {
		outputs[i] = s.Outputs[i]
	}
	return outputs
}

func (s Task) GetWorkingDir() string {
	return s.WorkingDir
}
