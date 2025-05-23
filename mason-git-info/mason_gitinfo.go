package main

import (
	"fmt"
	"strings"

	"github.com/vbehar/mason-sdk-go"
)

type GitInfoSpec struct {
	GitDirectory string              `json:"gitDirectory"`
	Outputs      []GitInfoSpecOutput `json:"outputs"`
}

type GitInfoSpecOutput struct {
	DaggerFileName string   `json:"daggerFileName"`
	HostFilePath   string   `json:"hostFilePath"`
	RawCmd         []string `json:"rawCmd"`
	Type           string   `json:"type"`
}

func (s GitInfoSpec) Plan(brick mason.Brick) map[string]string {
	plan := make(map[string]string, len(brick.Metadata.ExtraPhases))
	script := s.script(brick)
	for _, phase := range brick.Metadata.ExtraPhases {
		plan[phase+"_"+brick.Filename()] = script
	}
	return plan
}

func (s GitInfoSpec) script(brick mason.Brick) string {
	gitDirectory := "host | directory "
	if s.GitDirectory != "" {
		gitDirectory += s.GitDirectory
	} else {
		gitDirectory += "."
	}
	baseCmd := brick.ModuleRef + " --git-directory $(" + gitDirectory + ")"

	var cmd string
	for _, output := range s.Outputs {
		switch output.Type {
		case "diff":
			cmd += fmt.Sprintf("%s=$(%s | diff-file)\n", output.DaggerFileName, baseCmd)
		case "info":
			cmd += fmt.Sprintf("%s=$(%s | info-file)\n", output.DaggerFileName, baseCmd)
		default:
			cmd += fmt.Sprintf("%s=$(%s | raw-cmd-as-file \"%s\")\n", output.DaggerFileName, baseCmd, strings.Join(output.RawCmd, `","`))
		}

		if output.HostFilePath != "" {
			cmd += fmt.Sprintf("$%s | export %s\n.echo\n", output.DaggerFileName, output.HostFilePath)
		}
	}

	return cmd
}
