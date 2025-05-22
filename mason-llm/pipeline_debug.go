package main

import (
	"context"
	_ "embed"

	"dagger/mason-llm/internal/dagger"
)

//go:embed pipeline_debug.prompt
var pipelineDebugPrompt string

func (m MasonLlm) DebugPipeline(
	ctx context.Context,
	workspace *dagger.Directory,
	logFilePath string,
	// +optional
	gitDiff *dagger.File,
	// +optional
	gitInfo *dagger.File,
	// +optional
	env *dagger.Env,
	// +optional
	llm *dagger.LLM,
	// +optional
	additionalInstructions string,
) *LlmResult {
	if env == nil {
		env = dag.Env()
	}

	if isNull, err := env.Input("workspace").IsNull(ctx); isNull || err != nil {
		env = env.WithDirectoryInput("workspace", workspace, "The directory that contains all the files for the source code of the project.")
	}

	if isNull, err := env.Input("log_file_path").IsNull(ctx); isNull || err != nil {
		env = env.WithStringInput("log_file_path", logFilePath, "The path of the CI run output log file in the workspace directory.")
	}

	if isNull, err := env.Input("git-diff").IsNull(ctx); isNull || err != nil {
		if gitDiff == nil {
			gitDiff = dag.MasonGitInfo(dagger.MasonGitInfoOpts{GitDirectory: workspace}).DiffFile()
		}
		env = env.WithFileInput("git-diff", gitDiff, "The changes made to the source code, in the git diff format.")
	}

	if isNull, err := env.Input("git-info").IsNull(ctx); isNull || err != nil {
		if gitInfo == nil {
			gitInfo = dag.MasonGitInfo(dagger.MasonGitInfoOpts{GitDirectory: workspace}).InfoFile()
		}
		env = env.WithFileInput("git-info", gitInfo, "A JSON file that contains various information about the git repository.")
	}

	if isNull, err := env.Output("result").IsNull(ctx); isNull || err != nil {
		env = env.WithStringOutput("result", "The result of your analysis, in Markdown format.")
	}

	prompt := pipelineDebugPrompt
	if additionalInstructions != "" {
		prompt += additionalInstructions
	}

	if llm == nil {
		llm = dag.LLM()
	}
	llm = llm.
		WithPrompt(prompt).
		WithEnv(env)

	return &LlmResult{
		Llm: llm,
	}
}
