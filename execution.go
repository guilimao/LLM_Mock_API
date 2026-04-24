package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

type ChainSelectionTrace struct {
	SelectedStep int
	FallbackUsed bool
	ToolMismatch bool
}

type RequestExecutionOptions struct {
	Deterministic bool
	ChainTrace    bool
	Created       int64
	responseID    string
	rng           *rand.Rand
}

func NewRequestExecutionOptions(req ChatRequest) *RequestExecutionOptions {
	deterministic := req.Debug != nil && req.Debug.Deterministic
	seed := time.Now().UnixNano()
	if deterministic {
		seed = 1
		if req.Seed != nil {
			seed = int64(*req.Seed)
		}
	}

	rng := rand.New(rand.NewSource(seed))
	created := time.Now().Unix()
	if deterministic {
		created = seed
		if created < 0 {
			created = -created
		}
	}

	return &RequestExecutionOptions{
		Deterministic: deterministic,
		ChainTrace:    req.Debug != nil && req.Debug.ChainTrace,
		Created:       created,
		responseID:    fmt.Sprintf("chatcmpl-%08x", rng.Uint32()),
		rng:           rng,
	}
}

func (o *RequestExecutionOptions) ResponseID() string {
	return o.responseID
}

func (o *RequestExecutionOptions) NextToolCallID(index int) string {
	return fmt.Sprintf("call_%08x_%d", o.rng.Uint32(), index)
}

func collectDefinedTools(tools []ChatFunctionTool) map[string]struct{} {
	if len(tools) == 0 {
		return nil
	}

	names := make(map[string]struct{}, len(tools))
	for _, tool := range tools {
		name := strings.TrimSpace(tool.Function.Name)
		if name != "" {
			names[name] = struct{}{}
		}
	}
	return names
}
