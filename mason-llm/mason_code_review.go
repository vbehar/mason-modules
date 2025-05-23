package main

import (
	"fmt"
	"strings"

	"github.com/vbehar/mason-sdk-go"
)

type LLMCodeReviewSpec struct {
	LLM                    LLMCodeReviewSpecLLM                  `json:"llm"`
	Workspace              LLMCodeReviewSpecInputSourceDirectory `json:"workspace"`
	AdditionalInputs       []LLMCodeReviewSpecInput              `json:"AdditionalInputs"`
	AdditionalInstructions string                                `json:"AdditionalInstructions"`
	Output                 LLMCodeReviewSpecOutput               `json:"output"`
}

type LLMCodeReviewSpecLLM struct {
	Model       string `json:"model"`
	MaxAPICalls int    `json:"maxAPICalls"`
}

type LLMCodeReviewSpecInput struct {
	Name        string                       `json:"name"`
	Description string                       `json:"description"`
	Source      LLMCodeReviewSpecInputSource `json:"source"`
}

type LLMCodeReviewSpecInputSource struct {
	DaggerFileName string                                `json:"daggerFileName"`
	Directory      LLMCodeReviewSpecInputSourceDirectory `json:"directory"`
}

type LLMCodeReviewSpecInputSourceDirectory struct {
	Path    string   `json:"path"`
	Include []string `json:"include"`
	Exclude []string `json:"exclude"`
}

type LLMCodeReviewSpecOutput struct {
	DaggerFileName string `json:"daggerFileName"`
	HostFilePath   string `json:"hostFilePath"`
}

func (s LLMCodeReviewSpec) Plan(brick mason.Brick) map[string]string {
	plan := map[string]string{
		"review_" + brick.Filename(): s.script(brick),
	}
	for _, phase := range brick.Metadata.ExtraPhases {
		plan[phase+"_"+brick.Filename()] = s.script(brick)
	}
	return plan
}

func (s LLMCodeReviewSpec) script(brick mason.Brick) string {
	brickName := strings.ReplaceAll(brick.Metadata.Name, "-", "_")

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

	baseCmd := brick.ModuleRef + " | review-code $(" + workspace + ") --env $(" + env + ") --llm $(" + llm + ") --additional-instructions \"" + s.AdditionalInstructions + "\""

	var cmd string
	cmd += fmt.Sprintf("%s_code_review=$( %s )\n", brickName, baseCmd)
	cmd += fmt.Sprintf("$%s_code_review | provider-info\n.echo\n", brickName)
	cmd += fmt.Sprintf("$%s_code_review | tokens-info\n.echo\n", brickName)
	cmd += fmt.Sprintf("%s=$( $%s_code_review | result-file )\n", s.Output.DaggerFileName, brickName)
	if s.Output.HostFilePath != "" {
		cmd += fmt.Sprintf(".echo -n \"Local output file: \"\n$%s | export %s\n", s.Output.DaggerFileName, s.Output.HostFilePath)
	}

	return cmd
}
