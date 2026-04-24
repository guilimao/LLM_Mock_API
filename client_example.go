//go:build ignore

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const baseURL = "http://localhost:8080"

func main() {
	runRoundTripExample()
}

func runRoundTripExample() {
	initialReq := ChatRequest{
		Messages: []ChatMessage{
			{
				Role: "system",
				Content: "#CHAIN_STEP1: tool_calls{name=get_weather,args={\"location\":\"Shanghai\"}}\n" +
					"#CHAIN_STEP2: content{text=Tool result has been consumed.}",
			},
			{Role: "user", Content: "What is the weather?"},
		},
		Stream: true,
		Tools: []ChatFunctionTool{
			{
				Type: "function",
				Function: FunctionDefinition{
					Name:        "get_weather",
					Description: "Get weather information",
				},
			},
		},
		Debug: &ChatDebugOptions{
			Deterministic: true,
			ChainTrace:    true,
		},
		Seed: intPtr(7),
	}

	fmt.Println("Step 1: model emits tool_calls")
	var toolCall ChatToolCall
	err := sendStreamingRequest(initialReq, func(chunk ChatStreamChunk) {
		if len(chunk.Choices) == 0 {
			return
		}
		for _, tc := range chunk.Choices[0].Delta.ToolCalls {
			if tc.ID != "" {
				toolCall.ID = tc.ID
				toolCall.Type = tc.Type
			}
			if tc.Function.Name != "" {
				toolCall.Function.Name = tc.Function.Name
			}
			if tc.Function.Arguments != "" {
				toolCall.Function.Arguments += tc.Function.Arguments
			}
		}
	})
	if err != nil {
		fmt.Printf("stream error: %v\n", err)
		return
	}

	fmt.Printf("Tool call: %s(%s) [%s]\n", toolCall.Function.Name, toolCall.Function.Arguments, toolCall.ID)

	toolResult := `{"temperature":26,"condition":"sunny"}`
	secondReq := ChatRequest{
		Messages: []ChatMessage{
			initialReq.Messages[0],
			initialReq.Messages[1],
			{
				Role:      "assistant",
				Content:   "",
				ToolCalls: []ChatToolCall{toolCall},
			},
			{
				Role:       "tool",
				ToolCallID: toolCall.ID,
				Content:    toolResult,
			},
		},
	}

	fmt.Println("Step 2: client sends real tool result back")
	resp, err := sendRequest(secondReq)
	if err != nil {
		fmt.Printf("request error: %v\n", err)
		return
	}

	if len(resp.Choices) > 0 {
		fmt.Printf("Final response: %s\n", resp.Choices[0].Message.Content)
	}
}

func sendRequest(req ChatRequest) (*ChatResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(baseURL+"/api/v1/chat/completions", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(data, &chatResp); err != nil {
		return nil, err
	}
	return &chatResp, nil
}

func sendStreamingRequest(req ChatRequest, onChunk func(ChatStreamChunk)) error {
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequest(http.MethodPost, baseURL+"/api/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		if payload == "[DONE]" {
			break
		}

		var chunk ChatStreamChunk
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			return err
		}
		onChunk(chunk)
	}
	return scanner.Err()
}

type ChatRequest struct {
	Messages []ChatMessage      `json:"messages"`
	Stream   bool               `json:"stream,omitempty"`
	Tools    []ChatFunctionTool `json:"tools,omitempty"`
	Debug    *ChatDebugOptions  `json:"debug,omitempty"`
	Seed     *int               `json:"seed,omitempty"`
}

type ChatMessage struct {
	Role       string         `json:"role"`
	Content    string         `json:"content,omitempty"`
	ToolCalls  []ChatToolCall `json:"tool_calls,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
}

type ChatToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ChatFunctionTool struct {
	Type     string             `json:"type"`
	Function FunctionDefinition `json:"function"`
}

type FunctionDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type ChatDebugOptions struct {
	Deterministic bool `json:"deterministic,omitempty"`
	ChainTrace    bool `json:"chain_trace,omitempty"`
}

type ChatResponse struct {
	Choices []struct {
		Message ChatMessage `json:"message"`
	} `json:"choices"`
}

type ChatStreamChunk struct {
	Choices []struct {
		Delta struct {
			ToolCalls []struct {
				Index    int    `json:"index"`
				ID       string `json:"id,omitempty"`
				Type     string `json:"type,omitempty"`
				Function struct {
					Name      string `json:"name,omitempty"`
					Arguments string `json:"arguments,omitempty"`
				} `json:"function,omitempty"`
			} `json:"tool_calls,omitempty"`
		} `json:"delta"`
	} `json:"choices"`
}

func intPtr(v int) *int {
	return &v
}
