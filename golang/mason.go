package main

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"dagger/golang/internal/dagger"
)

func (m *Golang) RenderPlan(ctx context.Context, blueprint *dagger.Directory) (*dagger.Directory, error) {
	fileNames, err := blueprint.Entries(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get blueprint directory entries: %w", err)
	}

	outDirectory := dag.Directory()
	for _, fileName := range fileNames {
		name := strings.TrimSuffix(fileName, filepath.Ext(fileName))
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
		case "gobinary":
			var spec GoBinarySpec
			err = json.Unmarshal(brick.Spec, &spec)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal spec for file %s: %w", fileName, err)
			}
			outDirectory = addPlanToDirectory(spec.Plan, outDirectory, brick, name)
		case "gotest":
			var spec GoTestSpec
			err = json.Unmarshal(brick.Spec, &spec)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal spec for file %s: %w", fileName, err)
			}
			outDirectory = addPlanToDirectory(spec.Plan, outDirectory, brick, name)
		case "golint":
			var spec GoLintSpec
			err = json.Unmarshal(brick.Spec, &spec)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal spec for file %s: %w", fileName, err)
			}
			outDirectory = addPlanToDirectory(spec.Plan, outDirectory, brick, name)
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
	name string,
) *dagger.Directory {
	scriptByPhase := planFunc(brick)
	for phase, script := range scriptByPhase {
		dir = dir.WithNewFile(
			fmt.Sprintf("%s_%s.dagger", phase, name),
			script,
		)
	}
	return dir
}

type Brick struct {
	Kind      string `json:"kind"`
	ModuleRef string `json:"moduleRef"`
	Metadata  struct {
		Name string `json:"name"`
	} `json:"metadata"`
	Spec json.RawMessage `json:"spec"`
}
