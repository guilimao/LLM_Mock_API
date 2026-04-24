package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestExtractChainFromMessagesSelectsSteps(t *testing.T) {
	handler := NewHandler("mock/test-model")
	req := ChatRequest{
		Messages: []ChatMessage{
			{Role: "system", Content: "#CHAIN_STEP1: tool_calls{name=get_weather,args={\"location\":\"Shanghai\"}}\n#CHAIN_STEP2: content{text=done}\n#CHAIN_STEP3: content{text=fallback}"},
			{Role: "user", Content: "weather"},
			{
				Role: "assistant",
				ToolCalls: []ChatToolCall{{
					ID:   "call_1",
					Type: "function",
					Function: FunctionCall{
						Name:      "get_weather",
						Arguments: `{"location":"Shanghai"}`,
					},
				}},
			},
			{Role: "tool", ToolCallID: "call_1", Content: `{"temperature":26}`},
		},
	}

	chain, trace, err := handler.extractChainFromMessages(req, false, false, NewRequestExecutionOptions(req))
	if err != nil {
		t.Fatalf("extractChainFromMessages returned error: %v", err)
	}
	if trace.SelectedStep != 2 || trace.FallbackUsed {
		t.Fatalf("unexpected trace: %#v", trace)
	}
	if got := chain.Segments[0][0].Content; got != "done" {
		t.Fatalf("expected step 2 content, got %q", got)
	}

	fallbackReq := ChatRequest{
		Messages: []ChatMessage{
			{Role: "system", Content: "#CHAIN_STEP1: content{text=first}\n#CHAIN_STEP3: content{text=final}"},
			{Role: "user", Content: "weather"},
			{Role: "assistant", ToolCalls: []ChatToolCall{{ID: "call_1", Type: "function", Function: FunctionCall{Name: "get_weather", Arguments: `{"location":"Shanghai"}`}}}},
			{Role: "tool", ToolCallID: "call_1", Content: `{"temperature":26}`},
		},
	}

	chain, trace, err = handler.extractChainFromMessages(fallbackReq, false, false, NewRequestExecutionOptions(fallbackReq))
	if err != nil {
		t.Fatalf("fallback extractChainFromMessages returned error: %v", err)
	}
	if trace.SelectedStep != 3 || !trace.FallbackUsed {
		t.Fatalf("expected fallback to step 3, got %#v", trace)
	}
	if got := chain.Segments[0][0].Content; got != "final" {
		t.Fatalf("expected fallback content, got %q", got)
	}
}

func TestChatCompletionsNonStreamToolCalls(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewHandler("mock/test-model")

	reqBody := ChatRequest{
		Messages: []ChatMessage{
			{Role: "system", Content: "#CHAIN: tool_calls{name=get_weather,args={\"location\":\"Shanghai\"}}"},
			{Role: "user", Content: "weather"},
		},
		Tools: []ChatFunctionTool{{
			Type: "function",
			Function: FunctionDefinition{
				Name:        "get_weather",
				Description: "Get weather",
			},
		}},
		Debug: &ChatDebugOptions{
			Deterministic: true,
			ChainTrace:    true,
		},
		Seed: intPtr(7),
	}
	expectedOpts := NewRequestExecutionOptions(reqBody)
	expectedResponseID := expectedOpts.ResponseID()
	expectedToolCallID := expectedOpts.NextToolCallID(1)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	body, _ := json.Marshal(reqBody)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/chat/completions", bytes.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")

	handler.ChatCompletions(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", recorder.Code, recorder.Body.String())
	}
	if recorder.Header().Get("X-Mock-Chain-Step") != "1" {
		t.Fatalf("expected chain trace header, got %q", recorder.Header().Get("X-Mock-Chain-Step"))
	}

	var resp ChatResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.ID != expectedResponseID {
		t.Fatalf("unexpected deterministic response id: %s", resp.ID)
	}
	if len(resp.Choices) != 1 || resp.Choices[0].FinishReason == nil || *resp.Choices[0].FinishReason != "tool_calls" {
		t.Fatalf("unexpected finish reason: %#v", resp.Choices)
	}
	if len(resp.Choices[0].Message.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(resp.Choices[0].Message.ToolCalls))
	}
	toolCall := resp.Choices[0].Message.ToolCalls[0]
	if toolCall.ID != expectedToolCallID {
		t.Fatalf("unexpected deterministic tool call id: %s", toolCall.ID)
	}
	if toolCall.Function.Name != "get_weather" {
		t.Fatalf("unexpected tool name: %s", toolCall.Function.Name)
	}
}

func TestChatCompletionsStreamingRoundTripAndDeterministic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewHandler("mock/test-model")

	reqBody := ChatRequest{
		Messages: []ChatMessage{
			{Role: "system", Content: "#CHAIN_STEP1: tool_calls{name=get_weather,args={\"location\":\"Shanghai\"},name2=search,args2={\"query\":\"weather\"}}\n#CHAIN_STEP2: content{text=Tool results received}"},
			{Role: "user", Content: "weather"},
			{
				Role: "assistant",
				ToolCalls: []ChatToolCall{
					{ID: "call_a", Type: "function", Function: FunctionCall{Name: "get_weather", Arguments: `{"location":"Shanghai"}`}},
					{ID: "call_b", Type: "function", Function: FunctionCall{Name: "search", Arguments: `{"query":"weather"}`}},
				},
			},
			{Role: "tool", ToolCallID: "call_a", Content: `{"temperature":26}`},
			{Role: "tool", ToolCallID: "call_b", Content: `{"results":[1]}`},
		},
		Stream: true,
		Debug: &ChatDebugOptions{
			Deterministic: true,
			ChainTrace:    true,
		},
		Seed: intPtr(7),
	}
	expectedOpts := NewRequestExecutionOptions(reqBody)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	body, _ := json.Marshal(reqBody)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/chat/completions", bytes.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")

	handler.ChatCompletions(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", recorder.Code, recorder.Body.String())
	}
	if recorder.Header().Get("X-Mock-Chain-Step") != "2" {
		t.Fatalf("expected selected step 2, got %q", recorder.Header().Get("X-Mock-Chain-Step"))
	}
	if recorder.Header().Get("Content-Type") != "text/event-stream" {
		t.Fatalf("expected SSE content type, got %q", recorder.Header().Get("Content-Type"))
	}

	var chunks []ChatStreamChunk
	scanner := bufio.NewScanner(strings.NewReader(recorder.Body.String()))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		if payload == "[DONE]" {
			break
		}

		var chunk ChatStreamChunk
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			t.Fatalf("failed to decode chunk %q: %v", payload, err)
		}
		chunks = append(chunks, chunk)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scanner error: %v", err)
	}

	if len(chunks) < 2 {
		t.Fatalf("expected at least role + content chunks, got %d", len(chunks))
	}
	if chunks[0].ID != expectedOpts.ResponseID() {
		t.Fatalf("unexpected stream id: %s", chunks[0].ID)
	}
	if chunks[0].Choices[0].Delta.Role != "assistant" {
		t.Fatalf("expected first chunk to set assistant role, got %#v", chunks[0].Choices[0].Delta)
	}

	var content strings.Builder
	var finishReason string
	for _, chunk := range chunks[1:] {
		content.WriteString(chunk.Choices[0].Delta.Content)
		if chunk.Choices[0].FinishReason != nil {
			finishReason = *chunk.Choices[0].FinishReason
		}
	}
	if content.String() != "Tool results received" {
		t.Fatalf("unexpected streamed content: %q", content.String())
	}
	if finishReason != "stop" {
		t.Fatalf("expected stop finish reason, got %q", finishReason)
	}
}

func intPtr(v int) *int {
	return &v
}
