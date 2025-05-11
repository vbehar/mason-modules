package main

import (
	"context"
	"fmt"
	"path/filepath"

	"dagger/golang/internal/dagger"
)

const (
	goTestJUnitFilePath = "/output/tests-report.xml"
	goTestJSONFilePath  = "/output/tests-report.json"
)

func (g *Golang) Test(
	ctx context.Context,
	// "go test" extra arguments
	// +optional
	args []string,
	// +optional
	baseContainer *dagger.Container,
	// The version of the gotestsum tool to use.
	// See https://github.com/gotestyourself/gotestsum/releases
	// +optional
	// +default="1.12.1"
	gotestsumVersion string,
	// The version of the tparse tool to use.
	// See https://github.com/mfridman/tparse/releases
	// +optional
	// +default="0.17.0"
	tparseVersion string,
) (*TestRun, error) {
	cmd := append([]string{
		"gotestsum",
		"--junitfile", goTestJUnitFilePath,
		"--jsonfile", goTestJSONFilePath,
		"--",
	}, args...)

	ctr := g.Container(baseContainer).
		WithFile("/usr/local/bin/gotestsum", g.goTestSumFile(ctx, gotestsumVersion)).
		WithFile("/usr/local/bin/tparse", g.tParseFile(ctx, tparseVersion), dagger.ContainerWithFileOpts{
			Permissions: 0755,
		}).
		WithDirectory("/src", g.Source).
		WithWorkdir(filepath.Join("/src", g.Module)).
		WithExec(cmd, dagger.ContainerWithExecOpts{
			Expect: dagger.ReturnTypeAny,
		}).
		WithExec([]string{"tparse", "-file", goTestJSONFilePath}, dagger.ContainerWithExecOpts{
			Expect: dagger.ReturnTypeAny,
		})

	exitCode, err := ctr.ExitCode(ctx)
	if err != nil {
		return nil, err
	}

	return &TestRun{
		Ctr:      ctr,
		ExitCode: exitCode,
	}, nil
}

type TestRun struct {
	Ctr      *dagger.Container
	ExitCode int
}

func (t *TestRun) Assert(ctx context.Context) (string, error) {
	output, err := t.Ctr.Stdout(ctx)
	if err != nil {
		return output, err
	}
	if t.ExitCode != 0 {
		return output, fmt.Errorf("go test failed with exit code %d: %s", t.ExitCode, output)
	}
	return output, nil
}

func (t *TestRun) JUnitFile() *dagger.File {
	return t.Ctr.File(goTestJUnitFilePath)
}

func (t *TestRun) JsonFile() *dagger.File {
	return t.Ctr.File(goTestJSONFilePath)
}

func (t *TestRun) Reports() *dagger.Directory {
	return dag.Directory().
		WithFile("tests-junit-report.xml", t.JUnitFile()).
		WithFile("tests-report.json", t.JsonFile())
}

func (g *Golang) goTestSumFile(ctx context.Context, gotestsumVersion string) *dagger.File {
	platform, _ := dag.DefaultPlatform(ctx) //nolint:errcheck // don't care
	os, arch, ok := extractPlatform(platform)
	if !ok {
		os = "linux"
		arch = "amd64"
	}

	u := fmt.Sprintf("https://github.com/gotestyourself/gotestsum/releases/download/v%[1]s/gotestsum_%[1]s_%[2]s_%[3]s.tar.gz",
		gotestsumVersion, os, arch,
	)
	tarGzFile := dag.HTTP(u)

	return g.BaseRunContainer(platform).
		WithFile("/gotestsum.tar.gz", tarGzFile).
		WithExec([]string{"tar", "xzf", "/gotestsum.tar.gz"}).
		File("/gotestsum")
}

func (g *Golang) tParseFile(ctx context.Context, tparseVersion string) *dagger.File {
	platform, _ := dag.DefaultPlatform(ctx) //nolint:errcheck // don't care
	os, arch, ok := extractPlatform(platform)
	if !ok {
		os = "linux"
		arch = "amd64"
	}
	if arch == "amd64" {
		arch = "x86_64"
	}

	u := fmt.Sprintf("https://github.com/mfridman/tparse/releases/download/v%s/tparse_%s_%s",
		tparseVersion, os, arch,
	)
	return dag.HTTP(u)
}
