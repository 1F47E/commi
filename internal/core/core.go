package core

import (
	"context"
	"errors"
	"fmt"
)

var (
	ErrEmptySystemPrompt = errors.New("system prompt cannot be empty")
	ErrEmptyDiffs        = errors.New("diffs cannot be empty")
)

type LLMClient interface {
	GenerateCommitMessage(ctx context.Context, systemPrompt, status, diffs, subject string) (string, error)
}

type Core struct {
	client LLMClient
}

func NewCore(client LLMClient) *Core {
	if client == nil {
		panic("LLM client cannot be nil")
	}
	return &Core{
		client: client,
	}
}

type CommitMessage struct {
	Title   string
	Message string
}

type GenerateOptions struct {
	SystemPrompt string
	Status       string
	Diffs        string
	Subject      string
}

func (o *GenerateOptions) validate() error {
	if o.SystemPrompt == "" {
		return ErrEmptySystemPrompt
	}
	if o.Diffs == "" {
		return ErrEmptyDiffs
	}
	return nil
}

func (c *Core) GenerateCommit(ctx context.Context, opts GenerateOptions) (*CommitMessage, error) {
	if err := opts.validate(); err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}

	xmlContent, err := c.client.GenerateCommitMessage(
		ctx,
		opts.SystemPrompt,
		opts.Status,
		opts.Diffs,
		opts.Subject,
	)
	if err != nil {
		return nil, fmt.Errorf("LLM client failed: %w", err)
	}

	commit, err := parseCommitMessage(xmlContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	return commit, nil
}
