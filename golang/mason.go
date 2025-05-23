package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"dagger/golang/internal/dagger"

	"github.com/vbehar/mason-sdk-go"
)

func (m *Golang) RenderPlan(ctx context.Context, blueprint *dagger.Directory) (*dagger.Directory, error) {
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

		var brick mason.Brick
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
			outDirectory = addPlanToDirectory(spec.Plan, outDirectory, brick)
		case "gotest":
			var spec GoTestSpec
			err = json.Unmarshal(brick.Spec, &spec)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal spec for file %s: %w", fileName, err)
			}
			outDirectory = addPlanToDirectory(spec.Plan, outDirectory, brick)
		case "golint":
			var spec GoLintSpec
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
	planFunc func(mason.Brick) map[string]string,
	dir *dagger.Directory,
	brick mason.Brick,
) *dagger.Directory {
	for filename, script := range planFunc(brick) {
		dir = dir.WithNewFile(filename+".dagger", script)
	}
	return dir
}
