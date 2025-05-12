package main

import (
	"fmt"
	"strings"
)

type GoLintSpec struct {
	LintArgs []string          `json:"lintArgs"`
	Sources  GoLintSpecSources `json:"sources"`
	Output   GoLintSpecOutput  `json:"output"`
}

type GoLintSpecSources struct {
	Path                string   `json:"path"`
	Include             []string `json:"include"`
	Exclude             []string `json:"exclude"`
	GolangCILintVersion string   `json:"golangCILintVersion"`
}

type GoLintSpecOutput struct {
	CodeClimateDaggerFileName string `json:"codeClimateDaggerFileName"`
	CodeClimateHostFilePath   string `json:"codeClimateHostFilePath"`
}

func (s GoLintSpec) Plan(brick Brick) map[string]string {
	plan := map[string]string{
		"lint": s.lintScript(brick),
	}
	for _, phase := range brick.Metadata.ExtraPhases {
		plan[phase] = plan["lint"]
	}
	return plan
}

func (s GoLintSpec) lintScript(brick Brick) string {
	src := "host | directory "
	if s.Sources.Path != "" {
		src += s.Sources.Path
	} else {
		src += "."
	}
	if len(s.Sources.Include) > 0 {
		src += ` --include "` + strings.Join(s.Sources.Include, `","`) + `"`
	}
	if len(s.Sources.Exclude) > 0 {
		src += ` --exclude "` + strings.Join(s.Sources.Exclude, `","`) + `"`
	}

	baseCmd := brick.ModuleRef + " --source $(" + src + ") | lint"
	if s.Sources.GolangCILintVersion != "" {
		baseCmd += " --golangcilint-version " + s.Sources.GolangCILintVersion
	}
	if len(s.LintArgs) > 0 {
		baseCmd += " " + strings.Join(s.LintArgs, " ")
	}

	var cmd string
	if s.Output.CodeClimateDaggerFileName != "" {
		cmd += fmt.Sprintf("%s=$(%s | code-climate-file)", s.Output.CodeClimateDaggerFileName, baseCmd)
		if s.Output.CodeClimateHostFilePath != "" {
			cmd += fmt.Sprintf("\n$%s | export %s", s.Output.CodeClimateDaggerFileName, s.Output.CodeClimateHostFilePath)
		}
	} else {
		if s.Output.CodeClimateHostFilePath != "" {
			cmd += fmt.Sprintf("%s | code-climate-file | export %s", baseCmd, s.Output.CodeClimateHostFilePath)
		}
	}
	cmd += "\n_echo\n" + baseCmd + " | assert"

	return cmd
}
