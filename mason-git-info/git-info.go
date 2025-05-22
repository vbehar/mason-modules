package main

import (
	"context"
	"strings"

	"dagger/mason-git-info/internal/dagger"
)

const (
	// use fixed base images for reproductible builds and improved caching
	// the base image: https://images.chainguard.dev/directory/image/wolfi-base/overview
	// retrieve the latest sha256 hash with: `crane digest cgr.dev/chainguard/wolfi-base:latest`
	// and to retrieve its creation time: `crane config cgr.dev/chainguard/wolfi-base:latest | jq .created`
	// This one is from 2025-05-12T17:08:42Z
	baseImage = "cgr.dev/chainguard/wolfi-base:latest@sha256:3525626232d33ca137d020474cdf7659bc29b3c85b02d46c5ecf766cd72bbc59"
)

// GitInfo contains information about a git reference
type MasonGitInfo struct {
	GitDirectory *dagger.Directory
	Container    *dagger.Container
}

// New returns a new GitInfo instance with information about the git reference
func New(
	ctx context.Context,
	// directory containing the git repository
	// can be either the worktree (including the .git subdirectory)
	// or the .git directory iteself
	// +optional
	gitDirectory *dagger.Directory,
	// base container to use for git commands
	// default to cgr.dev/chainguard/wolfi-base:latest with git installed
	// +optional
	gitBaseContainer *dagger.Container,
) *MasonGitInfo {
	ctr := gitBaseContainer
	if ctr == nil {
		ctr = dag.Container().
			From(baseImage).
			WithExec([]string{
				"apk", "add", "--update", "--no-cache",
				"git",
			})
	}
	if gitDirectory != nil {
		ctr = ctr.
			WithMountedDirectory("/workdir", gitDirectory)
	}
	ctr = ctr.
		WithWorkdir("/workdir").
		WithExec([]string{"git", "config", "--global", "--add", "safe.directory", "/workdir"})

	return &MasonGitInfo{
		GitDirectory: gitDirectory,
		Container:    ctr,
	}
}

func (g *MasonGitInfo) InfoFile() *dagger.File {
	return dag.GitInfo(g.GitDirectory).JSONFile()
}

func (g *MasonGitInfo) BranchName(ctx context.Context) (string, error) {
	return dag.GitInfo(g.GitDirectory).Branch(ctx)
}

func (g *MasonGitInfo) RepoURL(ctx context.Context) (string, error) {
	return dag.GitInfo(g.GitDirectory).URL(ctx)
}

func (g *MasonGitInfo) DiffFile(ctx context.Context) (*dagger.File, error) {
	var fullDiff string

	diffFromOrigin, err := g.Container.WithExec([]string{
		"git", "diff", "origin",
	}, dagger.ContainerWithExecOpts{
		Expect: dagger.ReturnTypeAny,
	}).Stdout(ctx)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(diffFromOrigin) != "" {
		fullDiff = diffFromOrigin
	}

	localDiff, err := g.Container.WithExec([]string{
		"git", "diff",
	}, dagger.ContainerWithExecOpts{
		Expect: dagger.ReturnTypeAny,
	}).Stdout(ctx)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(localDiff) != "" {
		fullDiff += "\n" + localDiff
	}

	newFiles, err := g.Container.WithExec([]string{
		"sh", "-c",
		"git ls-files --others --exclude-standard -z | xargs -0 -n 1 git --no-pager diff /dev/null || true",
	}, dagger.ContainerWithExecOpts{
		Expect: dagger.ReturnTypeAny,
	}).Stdout(ctx)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(newFiles) != "" {
		fullDiff += "\n" + newFiles
	}

	return dag.File("git-diff", fullDiff), nil
}

func (g *MasonGitInfo) RawCmdAsFile(args []string) *dagger.File {
	return g.Container.
		WithExec(args, dagger.ContainerWithExecOpts{
			RedirectStdout: "/tmp/stdout",
		}).
		File("/tmp/stdout")
}
