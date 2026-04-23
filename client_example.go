// +build ignore

// 这是一个使用 LLM Mock API 的客户端示例
// 运行: go run client_example.go

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const baseURL = "http://localhost:8080"

func main() {
	fmt.Println("LLM Mock API 客户端示例")
	fmt.Println("========================")

	// 示例1: 简单对话
	fmt.Println("\n1. 简单对话")
	simpleChat()

	// 示例2: 流式对话
	fmt.Println("\n2. 流式对话")
	streamingChat()

	// 示例3: 带思考模式的对话
	fmt.Println("\n3. 带思考模式的对话")
	reasoningChat()

	// 示例4: 使用对话链条
	fmt.Println("\n4. 使用对话链条")
	chainChat()

	// 示例5: 工具调用
	fmt.Println("\n5. 工具调用")
	toolCallChat()

	// 示例6: 调用测试工具
	fmt.Println("\n6. 调用测试工具")
	invokeTestTool()

	// 示例7: 故障模拟
	fmt.Println("\n7. 故障模拟")
	faultSimulation()
}

func simpleChat() {
	req := ChatRequest{
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello!"},
		},
		Model: "mock/llm-model",
	}

	resp, err := sendRequest(req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	printResponse(resp)
}

func streamingChat() {
	req := ChatRequest{
		Messages: []ChatMessage{
			{Role: "user", Content: "Tell me a story"},
		},
		Stream: true,
	}

	fmt.Println("Streaming response:")
	err := sendStreamingRequest(req, func(chunk ChatStreamChunk) {
		if len(chunk.Choices) > 0 {
			delta := chunk.Choices[0].Delta
			if delta.Content != "" {
				fmt.Print(delta.Content)
			}
			if delta.Reasoning != "" {
				fmt.Printf("[Reasoning: %s]", delta.Reasoning)
			}
		}
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
	fmt.Println()
}

func reasoningChat() {
	// 示例 3.1: 使用 effort 参数 (OpenAI 风格)
	fmt.Println("\n3.1 使用 effort 参数 (high):")
	req := ChatRequest{
		Messages: []ChatMessage{
			{Role: "user", Content: "Calculate 15 * 23"},
		},
		Reasoning: &ChatReasoningConfig{
			Effort: "high",
		},
		Stream: true,
	}

	fmt.Println("Streaming with reasoning (effort=high):")
	err := sendStreamingRequest(req, func(chunk ChatStreamChunk) {
		if len(chunk.Choices) > 0 {
			delta := chunk.Choices[0].Delta
			if delta.Reasoning != "" {
				fmt.Printf("[Thinking: %s]", delta.Reasoning)
			}
			if delta.Content != "" {
				fmt.Print(delta.Content)
			}
		}
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
	fmt.Println()

	// 示例 3.2: 使用 max_tokens 参数 (Anthropic 风格)
	fmt.Println("\n3.2 使用 max_tokens 参数 (500):")
	maxTokens := 500
	req2 := ChatRequest{
		Messages: []ChatMessage{
			{Role: "user", Content: "Explain quantum computing"},
		},
		Reasoning: &ChatReasoningConfig{
			MaxTokens: &maxTokens,
		},
		Stream: true,
	}

	fmt.Println("Streaming with reasoning (max_tokens=500):")
	err = sendStreamingRequest(req2, func(chunk ChatStreamChunk) {
		if len(chunk.Choices) > 0 {
			delta := chunk.Choices[0].Delta
			if delta.Reasoning != "" {
				fmt.Printf("[Thinking: %s]", delta.Reasoning)
			}
			if delta.Content != "" {
				fmt.Print(delta.Content)
			}
		}
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
	fmt.Println()

	// 示例 3.3: 使用 enabled 参数显式启用
	fmt.Println("\n3.3 使用 enabled 参数显式启用:")
	enabled := true
	req3 := ChatRequest{
		Messages: []ChatMessage{
			{Role: "user", Content: "What is machine learning?"},
		},
		Reasoning: &ChatReasoningConfig{
			Enabled: &enabled,
			Effort:  "medium",
		},
		Stream: true,
	}

	fmt.Println("Streaming with reasoning (enabled=true):")
	err = sendStreamingRequest(req3, func(chunk ChatStreamChunk) {
		if len(chunk.Choices) > 0 {
			delta := chunk.Choices[0].Delta
			if delta.Reasoning != "" {
				fmt.Printf("[Thinking: %s]", delta.Reasoning)
			}
			if delta.Content != "" {
				fmt.Print(delta.Content)
			}
		}
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
	fmt.Println()

	// 示例 3.4: 使用 exclude 参数排除推理内容（仅统计，不显示）
	fmt.Println("\n3.4 使用 exclude=true 排除推理内容（统计中仍包含）:")
	exclude := true
	req4 := ChatRequest{
		Messages: []ChatMessage{
			{Role: "user", Content: "Explain neural networks"},
		},
		Reasoning: &ChatReasoningConfig{
			Effort:  "high",
			Exclude: &exclude,
		},
		Stream: false, // 非流式模式
	}

	fmt.Println("Non-streaming with reasoning (exclude=true):")
	resp, err := sendRequest(req4)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	printResponse(resp)
	fmt.Println("(注意: reasoning tokens 被排除在响应外，但仍计入 usage)")
}

func chainChat() {
	// 使用对话链条定义复杂的交互流程
	req := ChatRequest{
		Messages: []ChatMessage{
			{
				Role:    "system",
				Content: "#CHAIN: reasoning{text=Analyzing request}-content{text=Here's my analysis}-reasoning{text=Formulating response}-content{text=Final answer}",
			},
			{Role: "user", Content: "Explain quantum computing"},
		},
		Stream: true,
	}

	fmt.Println("Chain-based response:")
	err := sendStreamingRequest(req, func(chunk ChatStreamChunk) {
		if len(chunk.Choices) > 0 {
			delta := chunk.Choices[0].Delta
			if delta.Reasoning != "" {
				fmt.Printf("\n[Reasoning]: %s\n", delta.Reasoning)
			}
			if delta.Content != "" {
				fmt.Print(delta.Content)
			}
		}
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
	fmt.Println()
}

func toolCallChat() {
	req := ChatRequest{
		Messages: []ChatMessage{
			{
				Role:    "system",
				Content: "#CHAIN: content{text=I'll help you}-tool_calls{name=get_weather,args={\"location\":\"NYC\"}}-content{text=Based on the weather...}",
			},
			{Role: "user", Content: "What's the weather like?"},
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
	}

	fmt.Println("Tool call response:")
	err := sendStreamingRequest(req, func(chunk ChatStreamChunk) {
		if len(chunk.Choices) > 0 {
			delta := chunk.Choices[0].Delta
			if delta.Content != "" {
				fmt.Print(delta.Content)
			}
			if len(delta.ToolCalls) > 0 {
				for _, tc := range delta.ToolCalls {
					fmt.Printf("\n[Tool Call: %s(%s)]\n", tc.Function.Name, tc.Function.Arguments)
				}
			}
		}
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
	fmt.Println()
}

func invokeTestTool() {
	// 直接调用测试工具
	url := baseURL + "/test-tools/get_weather/invoke"
	args := map[string]interface{}{
		"location": "Shanghai",
		"unit":     "celsius",
	}

	jsonData, _ := json.Marshal(args)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Tool response: %s\n", string(body))
}

func faultSimulation() {
	// 测试故障模拟
	req := ChatRequest{
		Messages: []ChatMessage{
			{
				Role:    "system",
				Content: "#CHAIN: content{fault=delay,fault_duration=500ms,text=Delayed response}",
			},
			{Role: "user", Content: "Hello"},
		},
		Stream: true,
	}

	fmt.Println("Fault simulation (delay):")
	start := time.Now()
	err := sendStreamingRequest(req, func(chunk ChatStreamChunk) {
		if len(chunk.Choices) > 0 {
			delta := chunk.Choices[0].Delta
			if delta.Content != "" {
				fmt.Print(delta.Content)
			}
		}
	})
	elapsed := time.Since(start)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
	fmt.Printf("\n(Time: %v)\n", elapsed)
}

// HTTP 请求函数

func sendRequest(req ChatRequest) (*ChatResponse, error) {
	url := baseURL + "/api/v1/chat/completions"
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, err
	}

	return &chatResp, nil
}

func sendStreamingRequest(req ChatRequest, onChunk func(ChatStreamChunk)) error {
	url := baseURL + "/api/v1/chat/completions"
	jsonData, err := json.Marshal(req)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk ChatStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			// 可能是故障模拟导致的损坏数据
			fmt.Printf("[Parse error: %v]\n", err)
			continue
		}

		onChunk(chunk)
	}

	return nil
}

func printResponse(resp *ChatResponse) {
	fmt.Printf("Response ID: %s\n", resp.ID)
	fmt.Printf("Model: %s\n", resp.Model)
	for _, choice := range resp.Choices {
		fmt.Printf("Content: %s\n", choice.Message.Content)
		if choice.Message.Reasoning != "" {
			fmt.Printf("Reasoning: %s\n", choice.Message.Reasoning)
		}
		if choice.FinishReason != nil {
			fmt.Printf("Finish Reason: %s\n", *choice.FinishReason)
		}
	}
	if resp.Usage != nil {
		fmt.Printf("Tokens: %d (prompt) + %d (completion) = %d total\n",
			resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)
	}
}

// 数据模型（简化版）

type ChatRequest struct {
	Messages  []ChatMessage      `json:"messages"`
	Model     string             `json:"model,omitempty"`
	Stream    bool               `json:"stream,omitempty"`
	Reasoning *ChatReasoningConfig `json:"reasoning,omitempty"`
	Tools     []ChatFunctionTool `json:"tools,omitempty"`
}

type ChatMessage struct {
	Role    string      `json:"role"`
	Content string      `json:"content"`
}

type ChatReasoningConfig struct {
	Enabled   *bool  `json:"enabled,omitempty"`
	Effort    string `json:"effort,omitempty"`
	MaxTokens *int   `json:"max_tokens,omitempty"`
	Exclude   *bool  `json:"exclude,omitempty"`
}

type ChatFunctionTool struct {
	Type     string           `json:"type"`
	Function FunctionDefinition `json:"function"`
}

type FunctionDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type ChatResponse struct {
	ID      string        `json:"id"`
	Object  string        `json:"object"`
	Model   string        `json:"model"`
	Choices []ChatChoice  `json:"choices"`
	Usage   *ChatUsage    `json:"usage,omitempty"`
}

type ChatChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason *string     `json:"finish_reason,omitempty"`
}

type ChatUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type ChatStreamChunk struct {
	ID      string           `json:"id"`
	Object  string           `json:"object"`
	Model   string           `json:"model"`
	Choices []ChatStreamChoice `json:"choices"`
}

type ChatStreamChoice struct {
	Index        int           `json:"index"`
	Delta        ChatStreamDelta `json:"delta"`
	FinishReason *string       `json:"finish_reason,omitempty"`
}

type ChatStreamDelta struct {
	Role      string   `json:"role,omitempty"`
	Content   string   `json:"content,omitempty"`
	Reasoning string   `json:"reasoning,omitempty"`
	ToolCalls []struct {
		Index    int    `json:"index"`
		Function struct {
			Name      string `json:"name,omitempty"`
			Arguments string `json:"arguments,omitempty"`
		} `json:"function"`
	} `json:"tool_calls,omitempty"`
}

