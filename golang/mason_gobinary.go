package main

import (
	"fmt"
	"strings"

	"github.com/vbehar/mason-sdk-go"
)

type GoBinarySpec struct {
	OS        string              `json:"os"`
	Arch      string              `json:"arch"`
	Packages  []string            `json:"packages"`
	BuildArgs []string            `json:"buildArgs"`
	Sources   GoBinarySpecSources `json:"sources"`
	Output    GoBinarySpecOutput  `json:"output"`
}

type GoBinarySpecSources struct {
	Path    string   `json:"path"`
	Include []string `json:"include"`
	Exclude []string `json:"exclude"`
}

type GoBinarySpecOutput struct {
	DaggerFileName string `json:"daggerFileName"`
	HostFilePath   string `json:"hostFilePath"`
}

func (s GoBinarySpec) Plan(brick mason.Brick) map[string]string {
	plan := map[string]string{
		"package_" + brick.Filename(): s.packageScript(brick),
	}
	for _, phase := range brick.Metadata.ExtraPhases {
		plan[phase+"_"+brick.Filename()] = plan["package_"+brick.Filename()]
	}
	return plan
}

func (s GoBinarySpec) packageScript(brick mason.Brick) string {
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

	cmd := brick.ModuleRef + " --source $(" + src + ") | build-binary"
	if s.OS != "" {
		cmd += " --go-os " + s.OS
	}
	if s.Arch != "" {
		cmd += " --go-arch " + s.Arch
	}
	if len(s.BuildArgs) > 0 || len(s.Packages) > 0 {
		args := append(s.BuildArgs, s.Packages...)
		cmd += ` --args "` + strings.Join(args, `","`) + `"`
	}
	if s.Output.DaggerFileName != "" {
		cmd += " --output-file-name " + s.Output.DaggerFileName
		cmd = fmt.Sprintf("%s=$(%s)", s.Output.DaggerFileName, cmd)
		if s.Output.HostFilePath != "" {
			cmd += fmt.Sprintf("\n$%s | export %s", s.Output.DaggerFileName, s.Output.HostFilePath)
		}
	} else {
		if s.Output.HostFilePath != "" {
			cmd += fmt.Sprintf(" | export %s", s.Output.HostFilePath)
		}
	}

	return cmd
}
