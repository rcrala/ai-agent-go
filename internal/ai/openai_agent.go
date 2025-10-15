package ai

import (
	"context"
)

// OpenAIEvaluator implements CodeEvaluator for OpenAI
// Wraps the OpenAIClient and config

type OpenAIEvaluator struct {
	Client *OpenAIClient
}

func NewOpenAIEvaluator(cfg AIAgentConfig) *OpenAIEvaluator {
	return &OpenAIEvaluator{
		Client: NewOpenAIClient(cfg.Key, cfg.Model, cfg.MaxTokens, cfg.Temperature),
	}
}

func (o *OpenAIEvaluator) Evaluate(ctx context.Context, fileName, code string) (*EvaluationResult, error) {
	return EvaluateCode(ctx, o.Client.Client, fileName, code, o.Client.Model, o.Client.MaxTokens, float32(o.Client.Temperature))
}
