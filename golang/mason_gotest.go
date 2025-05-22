package main

import (
	"fmt"
	"strings"
)

type GoTestSpec struct {
	Packages []string          `json:"packages"`
	TestArgs []string          `json:"testArgs"`
	Sources  GoTestSpecSources `json:"sources"`
	Output   GoTestSpecOutput  `json:"output"`
}

type GoTestSpecSources struct {
	Path    string   `json:"path"`
	Include []string `json:"include"`
	Exclude []string `json:"exclude"`
}

type GoTestSpecOutput struct {
	JUnitDaggerFileName string `json:"junitDaggerFileName"`
	JUnitHostFilePath   string `json:"junitHostFilePath"`
}

func (s GoTestSpec) Plan(brick Brick) map[string]string {
	plan := map[string]string{
		"test_" + brick.Filename(): s.testScript(brick),
	}
	for _, phase := range brick.Metadata.ExtraPhases {
		plan[phase+"_"+brick.Filename()] = plan["test_"+brick.Filename()]
	}
	return plan
}

func (s GoTestSpec) testScript(brick Brick) string {
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

	baseCmd := brick.ModuleRef + " --source $(" + src + ") | test"
	if len(s.TestArgs) > 0 {
		baseCmd += " " + strings.Join(s.TestArgs, " ")
	}
	if len(s.Packages) > 0 {
		baseCmd += " " + strings.Join(s.Packages, " ")
	}

	var cmd string
	if s.Output.JUnitDaggerFileName != "" {
		cmd += fmt.Sprintf("%s=$(%s | junit-file)", s.Output.JUnitDaggerFileName, baseCmd)
		if s.Output.JUnitHostFilePath != "" {
			cmd += fmt.Sprintf("\n$%s | export %s", s.Output.JUnitDaggerFileName, s.Output.JUnitHostFilePath)
		}
	} else {
		if s.Output.JUnitHostFilePath != "" {
			cmd += fmt.Sprintf("%s | junit-file | export %s", baseCmd, s.Output.JUnitHostFilePath)
		}
	}
	cmd += "\n.echo\n" + baseCmd + " | assert"

	return cmd
}
