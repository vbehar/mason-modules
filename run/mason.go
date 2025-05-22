package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"dagger/run/internal/dagger"
)

func (m *Run) RenderPlan(ctx context.Context, blueprint *dagger.Directory) (*dagger.Directory, error) {
	fileNames, err := blueprint.Entries(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get blueprint directory entries: %w", err)
	}

	outDirectory := dag.Directory()
	for _, fileName := range fileNames {
		data, err := blueprint.File(fileName).Contents(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get file contents for %s: %w", fileName, err)
		}

		var brick Brick
		err = json.Unmarshal([]byte(data), &brick)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal file %s: %w", fileName, err)
		}

		switch strings.ToLower(brick.Kind) {
		case "runbinary":
			var spec RunBinarySpec
			err = json.Unmarshal(brick.Spec, &spec)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal spec for file %s: %w", fileName, err)
			}
			outDirectory = addPlanToDirectory(spec.Plan, outDirectory, brick)
		default:
			fmt.Println("Unknown kind:", brick.Kind, "for file", fileName)
		}
	}

	return outDirectory, nil
}

func addPlanToDirectory(
	planFunc func(Brick) map[string]string,
	dir *dagger.Directory,
	brick Brick,
) *dagger.Directory {
	for filename, script := range planFunc(brick) {
		dir = dir.WithNewFile(filename+".dagger", script)
	}
	return dir
}

type Brick struct {
	Kind      string `json:"kind"`
	ModuleRef string `json:"moduleRef"`
	Metadata  struct {
		Name        string   `json:"name"`
		ExtraPhases []string `json:"extraPhases"`
		PostRun     PostRun  `json:"postRun"`
	} `json:"metadata"`
	Spec json.RawMessage `json:"spec"`
}

func (b Brick) Filename() string {
	var filename string
	switch b.Metadata.PostRun {
	case PostRunAlways:
		filename += "postrun_"
	case PostRunOnSuccess:
		filename += "postrun_on_success_"
	case PostRunOnFailure:
		filename += "postrun_on_failure_"
	}
	filename += strings.ToLower(b.Metadata.Name)
	return filename
}

type PostRun string

const (
	PostRunAlways    PostRun = "always"
	PostRunOnSuccess PostRun = "on_success"
	PostRunOnFailure PostRun = "on_failure"
	PostRunNever     PostRun = ""
)
