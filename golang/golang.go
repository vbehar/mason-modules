package main

import (
	"context"
	"path/filepath"
	"strings"

	"dagger/golang/internal/dagger"
)

const (
	baseRunImage   = "cgr.dev/chainguard/wolfi-base:latest"
	baseBuildImage = "cgr.dev/chainguard/go:latest-dev"
)

type Golang struct {
	Source *dagger.Directory
	Module string
}

func New(
	// the root directory of the source code
	// +optional
	source *dagger.Directory,
	// the subdirectory in the source directory where the go module is located
	// in case of multi-modules repository
	// +optional
	module string,
) *Golang {
	return &Golang{
		Source: source,
		Module: module,
	}
}

func (g *Golang) BaseBuildContainer() *dagger.Container {
	return dag.Container().From(baseBuildImage)
}

func (g *Golang) BaseRunContainer(
	// +optional
	platform dagger.Platform,
) *dagger.Container {
	return dag.Container(dagger.ContainerOpts{
		Platform: platform,
	}).From(baseRunImage)
}

func (g *Golang) Container(
	// +optional
	baseContainer *dagger.Container,
) *dagger.Container {
	ctr := baseContainer
	if ctr == nil {
		ctr = g.BaseBuildContainer()
	}

	ctr = ctr.
		WithEnvVariable("CGO_ENABLED", "0").
		WithEnvVariable("GOPATH", "/go").
		WithEnvVariable("GOMODCACHE", "/go/pkg/mod").
		WithMountedCache("/go/pkg/mod", dag.CacheVolume("go-mod")).
		WithEnvVariable("GOCACHE", "/go/build-cache").
		WithMountedCache("/go/build-cache", dag.CacheVolume("go-build"))

	ctr = ctr.
		WithWorkdir(filepath.Join("/src", g.Module)).
		WithDirectory("/src", g.Source, dagger.ContainerWithDirectoryOpts{
			Include: []string{"**/go.mod", "**/go.sum"},
		}).
		WithExec([]string{"go", "mod", "download"}).
		WithoutDirectory("/src")

	return ctr
}

// Build a binary for a specific platform - defaulting to the current platform
func (g *Golang) BuildBinary(
	ctx context.Context,
	// Go OS target
	// Default to the default platform's OS
	// +optional
	goOs string,
	// Go architecture target
	// Default to the default platform's architecture
	// +optional
	goArch string,
	// "go build" extra arguments
	// +optional
	args []string,
	// Name of the output file
	// Default to "{os}_{arch}"
	// +optional
	outputFileName string,
	// +optional
	baseContainer *dagger.Container,
) *dagger.File {
	if goOs == "" || goArch == "" {
		defaultPlatform, _ := dag.DefaultPlatform(ctx) //nolint:errcheck // don't care
		defaultOs, defaultArch, _ := extractPlatform(defaultPlatform)
		if goOs == "" {
			goOs = defaultOs
		}
		if goArch == "" {
			goArch = defaultArch
		}
	}

	if outputFileName == "" {
		outputFileName = goOs + "_" + goArch
	}
	outputFile := "/" + outputFileName
	args = append([]string{"go", "build", "-o", outputFile}, args...)

	return g.Container(baseContainer).
		WithDirectory("/src", g.Source, dagger.ContainerWithDirectoryOpts{
			Exclude: []string{"**/*_test.go"},
		}).
		WithEnvVariable("GOOS", goOs).
		WithEnvVariable("GOARCH", goArch).
		WithExec(args).
		File(outputFile).
		WithName(outputFileName)
}

func extractPlatform(platform dagger.Platform) (os, arch string, ok bool) {
	elems := strings.Split(string(platform), "/")
	if len(elems) < 2 {
		return "", "", false
	}
	return elems[0], elems[1], true
}
