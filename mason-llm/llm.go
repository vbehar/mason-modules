package main

import (
	"context"
	"fmt"

	"dagger/mason-llm/internal/dagger"
)

type MasonLlm struct{}

type LlmResult struct {
	Llm *dagger.LLM
}

func (r *LlmResult) Result(ctx context.Context) (string, error) {
	result, err := r.Llm.Env().Output("result").AsString(ctx)
	if err != nil {
		return "", err
	}
	return result, nil
}

func (r *LlmResult) ResultFile(ctx context.Context) (*dagger.File, error) {
	result, err := r.Result(ctx)
	if err != nil {
		return nil, err
	}
	return dag.File("result", result), nil
}

func (r *LlmResult) ProviderInfo(ctx context.Context) (string, error) {
	provider, err := r.Llm.Provider(ctx)
	if err != nil {
		return "", err
	}
	model, err := r.Llm.Model(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("LLM Provider: %s (%s)", provider, model), nil
}

func (r *LlmResult) TokensInfo(ctx context.Context) (string, error) {
	inputTokens, err := r.Llm.TokenUsage().InputTokens(ctx)
	if err != nil {
		return "", err
	}
	outputTokens, err := r.Llm.TokenUsage().OutputTokens(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("LLM Tokens used: %d (%d in, %d out)", inputTokens+outputTokens, inputTokens, outputTokens), nil
}
