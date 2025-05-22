package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"dagger/golang/internal/dagger"
)

const (
	codeClimateFilePath = "/output/code-climate.json"
)

func (g *Golang) Lint(
	ctx context.Context,
	// "golangci-lint run" extra arguments
	// +optional
	args []string,
	// +optional
	baseContainer *dagger.Container,
	// The version of the golangci-lint tool to use.
	// See https://github.com/golangci/golangci-lint/releases
	// +optional
	// +default="2.1.5"
	golangcilintVersion string,
) (*LintRun, error) {
	ctr := g.Container(baseContainer).
		WithFile("/usr/local/bin/golangci-lint", g.golangciLintFile(ctx, golangcilintVersion)).
		WithDirectory("/src", g.Source).
		WithWorkdir(filepath.Join("/src", g.Module)).
		WithEnvVariable("GOLANGCI_LINT_CACHE", "/go/lint-cache").
		WithMountedCache("/go/lint-cache", dag.CacheVolume("golangci-lint")).
		WithExec(append([]string{
			"golangci-lint", "run",
			"--output.text.path=stdout",
			"--output.code-climate.path=" + codeClimateFilePath,
		}, args...), dagger.ContainerWithExecOpts{
			Expect: dagger.ReturnTypeAny,
		})

	exitCode, err := ctr.ExitCode(ctx)
	if err != nil {
		return nil, err
	}

	return &LintRun{
		Ctr:      ctr,
		ExitCode: exitCode,
	}, nil
}

type LintRun struct {
	Ctr      *dagger.Container
	ExitCode int
}

func (l *LintRun) Assert(ctx context.Context) (string, error) {
	output, err := l.Ctr.Stdout(ctx)
	output = strings.TrimSpace(output)
	if err != nil {
		return output, err
	}
	if l.ExitCode != 0 {
		return output, fmt.Errorf("golangci-lint failed with exit code %d:\n%s", l.ExitCode, output)
	}
	return output, nil
}

func (l *LintRun) CodeClimateFile() *dagger.File {
	return l.Ctr.File(codeClimateFilePath)
}

func (l *LintRun) Reports() *dagger.Directory {
	return dag.Directory().
		WithFile("code-climate.json", l.CodeClimateFile())
}

func (g *Golang) golangciLintFile(ctx context.Context, golangcilintVersion string) *dagger.File {
	platform, _ := dag.DefaultPlatform(ctx) //nolint:errcheck // don't care
	os, arch, ok := extractPlatform(platform)
	if !ok {
		os = "linux"
		arch = "amd64"
	}

	u := fmt.Sprintf("https://github.com/golangci/golangci-lint/releases/download/v%[1]s/golangci-lint-%[1]s-%[2]s-%[3]s.tar.gz",
		golangcilintVersion, os, arch,
	)
	tarGzFile := dag.HTTP(u)

	return g.BaseRunContainer(platform).
		WithFile("/golangci-lint.tar.gz", tarGzFile).
		WithExec([]string{"tar", "xzf", "/golangci-lint.tar.gz"}).
		File(fmt.Sprintf("/golangci-lint-%s-%s-%s/golangci-lint", golangcilintVersion, os, arch))
}
