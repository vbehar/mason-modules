package main

import (
	"fmt"
	"strings"

	"github.com/vbehar/mason-sdk-go"
)

type LLMPipelineDebugSpec struct {
	LLM                    LLMPipelineDebugSpecLLM                  `json:"llm"`
	Workspace              LLMPipelineDebugSpecInputSourceDirectory `json:"workspace"`
	LogFilePath            string                                   `json:"logFilePath"`
	AdditionalInputs       []LLMPipelineDebugSpecInput              `json:"additionalInputs"`
	AdditionalInstructions string                                   `json:"additionalInstructions"`
	Output                 LLMPipelineDebugSpecOutput               `json:"output"`
}

type LLMPipelineDebugSpecLLM struct {
	Model       string `json:"model"`
	MaxAPICalls int    `json:"maxAPICalls"`
}

type LLMPipelineDebugSpecInput struct {
	Name        string                          `json:"name"`
	Description string                          `json:"description"`
	Source      LLMPipelineDebugSpecInputSource `json:"source"`
}

type LLMPipelineDebugSpecInputSource struct {
	DaggerFileName string                                   `json:"daggerFileName"`
	HostFilePath   string                                   `json:"hostFilePath"`
	Directory      LLMPipelineDebugSpecInputSourceDirectory `json:"directory"`
}

type LLMPipelineDebugSpecInputSourceDirectory struct {
	Path    string   `json:"path"`
	Include []string `json:"include"`
	Exclude []string `json:"exclude"`
}

type LLMPipelineDebugSpecOutput struct {
	DaggerFileName string `json:"daggerFileName"`
	HostFilePath   string `json:"hostFilePath"`
}

func (s LLMPipelineDebugSpec) Plan(brick mason.Brick) map[string]string {
	plan := make(map[string]string)
	if brick.Metadata.PostRun != "" {
		plan[brick.Filename()] = s.script(brick)
	} else {
		plan["debug_"+brick.Filename()] = s.script(brick)
	}
	return plan
}

func (s LLMPipelineDebugSpec) script(brick mason.Brick) string {
	brickName := strings.ReplaceAll(brick.Metadata.Name, "-", "_")

	if brick.Metadata.PostRun != "" && s.LogFilePath == "" {
		s.LogFilePath = "$log_file_path"
	}

	workspace := "host | directory "
	if s.Workspace.Path != "" {
		workspace += s.Workspace.Path
	} else {
		workspace += "."
	}
	if len(s.Workspace.Include) > 0 {
		workspace += ` --include "` + strings.Join(s.Workspace.Include, `","`) + `"`
	}
	if len(s.Workspace.Exclude) > 0 {
		workspace += ` --exclude "` + strings.Join(s.Workspace.Exclude, `","`) + `"`
	}

	env := "env"
	for _, input := range s.AdditionalInputs {
		switch {
		case input.Source.DaggerFileName != "":
			env += fmt.Sprintf(" | \n with-file-input %q $%s %q", input.Name, input.Source.DaggerFileName, input.Description)
		case input.Source.HostFilePath != "":
			env += fmt.Sprintf(" | \n with-file-input %q $(host | file %s) %q", input.Name, input.Source.HostFilePath, input.Description)
		case input.Source.Directory.Path != "":
			dir := fmt.Sprintf("host | directory %s", input.Source.Directory.Path)
			if len(input.Source.Directory.Include) > 0 {
				dir += ` --include "` + strings.Join(input.Source.Directory.Include, `","`) + `"`
			}
			if len(input.Source.Directory.Exclude) > 0 {
				dir += ` --exclude "` + strings.Join(input.Source.Directory.Exclude, `","`) + `"`
			}
			env += fmt.Sprintf(" | \n with-directory-input %q $( %s ) %q", input.Name, dir, input.Description)
		}
	}

	llm := "llm"
	if s.LLM.Model != "" {
		llm += fmt.Sprintf(" --model %s", s.LLM.Model)
	}
	if s.LLM.MaxAPICalls > 0 {
		llm += fmt.Sprintf(" --max-api-calls %d", s.LLM.MaxAPICalls)
	}

	baseCmd := brick.ModuleRef + " | debug-pipeline $(" + workspace + ") " + s.LogFilePath + " --env $(" + env + ") --llm $(" + llm + ") --additional-instructions \"" + s.AdditionalInstructions + "\""

	var cmd string
	cmd += fmt.Sprintf("%s_pipeline_debug=$( %s )\n", brickName, baseCmd)
	cmd += fmt.Sprintf("$%s_pipeline_debug | provider-info\n.echo\n", brickName)
	cmd += fmt.Sprintf("$%s_pipeline_debug | tokens-info\n.echo\n", brickName)
	cmd += fmt.Sprintf("%s=$( $%s_pipeline_debug | result-file )\n", s.Output.DaggerFileName, brickName)
	if s.Output.HostFilePath != "" {
		cmd += fmt.Sprintf(".echo -n \"Pipeline debug analysis: \"\n$%s | export %s\n", s.Output.DaggerFileName, s.Output.HostFilePath)
	}

	return cmd
}
