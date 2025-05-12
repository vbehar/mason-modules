package main

import (
	"dagger/run/internal/dagger"
	"strings"
)

type RunBinarySpec struct {
	Platform  dagger.Platform  `json:"platform"`
	BaseImage string           `json:"baseImage"`
	Binaries  []RunBinaryEntry `json:"binaries"`
	Command   []string         `json:"command"`
}

type RunBinaryEntry struct {
	Source RunBinarySource `json:"source"`
	Path   string          `json:"path"`
}

type RunBinarySource struct {
	DaggerFileName string `json:"daggerFileName"`
}

func (s RunBinarySpec) Plan(brick Brick) map[string]string {
	plan := map[string]string{
		"run": s.runScript(brick),
	}
	for _, phase := range brick.Metadata.ExtraPhases {
		plan[phase] = plan["run"]
	}
	return plan
}

func (s RunBinarySpec) runScript(_ Brick) string {
	cmd := "container"
	if s.Platform != "" {
		cmd += " --platform " + string(s.Platform)
	}
	cmd += " | from " + s.BaseImage
	for _, binary := range s.Binaries {
		daggerFileName := binary.Source.DaggerFileName
		if daggerFileName != "" {
			cmd += " | with-file " + binary.Path + " $" + daggerFileName
		}
	}
	cmd += " | with-exec " + strings.Join(s.Command, " ")
	cmd += " | stdout"
	return cmd
}
