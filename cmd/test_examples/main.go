package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type ChatRequest struct {
	Messages          []ChatMessage      `json:"messages"`
	Model             string             `json:"model,omitempty"`
	Stream            bool               `json:"stream,omitempty"`
	Reasoning         *ReasoningConfig   `json:"reasoning,omitempty"`
	Tools             []ChatFunctionTool `json:"tools,omitempty"`
	ParallelToolCalls *bool              `json:"parallel_tool_calls,omitempty"`
	Debug             *DebugOptions      `json:"debug,omitempty"`
	Seed              *int               `json:"seed,omitempty"`
}

type ChatMessage struct {
	Role       string         `json:"role"`
	Content    interface{}    `json:"content,omitempty"`
	ToolCalls  []ChatToolCall `json:"tool_calls,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
	Reasoning  string         `json:"reasoning,omitempty"`
}

type ReasoningConfig struct {
	Effort  string `json:"effort,omitempty"`
	Exclude *bool  `json:"exclude,omitempty"`
}

type ChatFunctionTool struct {
	Type     string             `json:"type"`
	Function FunctionDefinition `json:"function"`
}

type FunctionDefinition struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Parameters  interface{} `json:"parameters,omitempty"`
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

type DebugOptions struct {
	Deterministic bool `json:"deterministic,omitempty"`
	ChainTrace    bool `json:"chain_trace,omitempty"`
}

type Scenario struct {
	Key  string
	Name string
	Run  func(baseURL string) error
}

func main() {
	baseURL := flag.String("base-url", "http://localhost:8080", "Mock API base URL")
	flag.Parse()

	scenarios := []Scenario{
		{Key: "1", Name: "基础流式请求", Run: runBasicStreaming},
		{Key: "2", Name: "期望返回 reasoning 的单次流式请求", Run: runReasoningStreaming},
		{Key: "3", Name: "期望返回单个 tool_calls 的单次流式请求", Run: runSingleToolCallStreaming},
		{Key: "4", Name: "期望返回并行 tool_calls 的单次流式请求", Run: runParallelToolCallsStreaming},
		{Key: "5", Name: "CHAIN 控制的多轮带 reasoning 流式对话", Run: runMultiRoundReasoning},
		{Key: "6", Name: "CHAIN 控制的多轮带 tool calls 流式对话", Run: runMultiRoundToolCalls},
		{Key: "7", Name: "高延迟下缓慢响应的单次流式请求", Run: runSlowStreaming},
		{Key: "8", Name: "带有 JSON 格式错误的单次流式请求", Run: runMalformedJSONStreaming},
		{Key: "9", Name: "带有图像返回的单次流式请求", Run: runImageStreaming},
		{Key: "10", Name: "带有音频返回的单次流式请求", Run: runAudioStreaming},
		{Key: "11", Name: "超时请求", Run: runTimeoutScenario},
		{Key: "12", Name: "健康检查", Run: runHealthCheck},
		{Key: "13", Name: "列出可用模型", Run: runListModels},
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Println("========================================")
		fmt.Println("LLM Mock API Test Examples (Go)")
		fmt.Println("========================================")
		for _, scenario := range scenarios {
			fmt.Printf("%s. %s\n", scenario.Key, scenario.Name)
		}
		fmt.Println("0. 运行全部")
		fmt.Println("q. 退出")
		fmt.Print("请选择测试项: ")

		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("读取输入失败: %v\n", err)
			return
		}

		choice := strings.TrimSpace(input)
		fmt.Println()

		switch strings.ToLower(choice) {
		case "q":
			return
		case "0":
			for _, scenario := range scenarios {
				if err := runScenario(*baseURL, scenario); err != nil {
					fmt.Printf("场景执行失败: %v\n", err)
				}
			}
		default:
			found := false
			for _, scenario := range scenarios {
				if scenario.Key == choice {
					found = true
					if err := runScenario(*baseURL, scenario); err != nil {
						fmt.Printf("场景执行失败: %v\n", err)
					}
					break
				}
			}
			if !found {
				fmt.Println("无效选项")
			}
		}

		fmt.Println()
		fmt.Print("按回车继续...")
		_, _ = reader.ReadString('\n')
		fmt.Println()
	}
}

func runScenario(baseURL string, scenario Scenario) error {
	fmt.Println("========================================")
	fmt.Printf("场景 %s: %s\n", scenario.Key, scenario.Name)
	fmt.Println("========================================")
	return scenario.Run(baseURL)
}

func runBasicStreaming(baseURL string) error {
	req := ChatRequest{
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello, please stream a basic reply."},
		},
		Stream: true,
	}
	return doStreamingJSONRequest(baseURL, "/api/v1/chat/completions", req, 0)
}

func runReasoningStreaming(baseURL string) error {
	req := ChatRequest{
		Messages: []ChatMessage{
			{Role: "user", Content: "Explain why the sky appears blue."},
		},
		Reasoning: &ReasoningConfig{Effort: "high"},
		Stream:    true,
	}
	return doStreamingJSONRequest(baseURL, "/api/v1/chat/completions", req, 0)
}

func runSingleToolCallStreaming(baseURL string) error {
	req := ChatRequest{
		Messages: []ChatMessage{
			{Role: "system", Content: `#CHAIN: tool_calls{id=call_weather_1,name=get_weather,args={"location":"Shanghai"}}`},
			{Role: "user", Content: "Call the weather tool once."},
		},
		Stream: true,
		Tools: []ChatFunctionTool{
			functionTool("get_weather", "Get the current weather for a city"),
		},
	}
	return doStreamingJSONRequest(baseURL, "/api/v1/chat/completions", req, 0)
}

func runParallelToolCallsStreaming(baseURL string) error {
	parallel := true
	req := ChatRequest{
		Messages: []ChatMessage{
			{Role: "system", Content: `#CHAIN: tool_calls{id=call_weather_1,name=get_weather,args={"location":"Shanghai"},id2=call_search_1,name2=search,args2={"query":"Shanghai weather"}}`},
			{Role: "user", Content: "Call the weather and search tools in parallel."},
		},
		Stream:            true,
		ParallelToolCalls: &parallel,
		Tools: []ChatFunctionTool{
			functionTool("get_weather", "Get the current weather for a city"),
			functionTool("search", "Search for public information"),
		},
	}
	return doStreamingJSONRequest(baseURL, "/api/v1/chat/completions", req, 0)
}

func runMultiRoundReasoning(baseURL string) error {
	firstReasoning := "Analyzing the first user request..."
	firstContent := "Here is the first answer."
	firstReq := ChatRequest{
		Messages: []ChatMessage{
			{Role: "system", Content: fmt.Sprintf(`#CHAIN: reasoning{text=%s}-content{text=%s}`, firstReasoning, firstContent)},
			{Role: "user", Content: "Please answer my first question."},
		},
		Stream: true,
	}
	if err := doStreamingJSONRequest(baseURL, "/api/v1/chat/completions", firstReq, 0); err != nil {
		return err
	}

	secondReasoning := "Analyzing the follow-up user request..."
	secondContent := "Here is the follow-up answer."
	secondReq := ChatRequest{
		Messages: []ChatMessage{
			{Role: "system", Content: fmt.Sprintf(`#CHAIN: reasoning{text=%s}-content{text=%s}`, secondReasoning, secondContent)},
			{Role: "user", Content: "Please answer my first question."},
			{Role: "assistant", Content: firstContent, Reasoning: firstReasoning},
			{Role: "user", Content: "Please continue with a follow-up answer."},
		},
		Stream: true,
	}
	return doStreamingJSONRequest(baseURL, "/api/v1/chat/completions", secondReq, 0)
}

func runMultiRoundToolCalls(baseURL string) error {
	firstReasoning := "I need to decide whether a tool call is required."
	firstContent := "I will call the weather tool first."
	secondReasoning := "I have the tool result and can finish the answer."
	secondContent := "The weather tool result has been incorporated into the final answer."
	systemPrompt := fmt.Sprintf(
		"#CHAIN_STEP1: reasoning{text=%s}-content{text=%s}-tool_calls{id=call_weather_1,name=get_weather,args={\"location\":\"Shanghai\"}}\n#CHAIN_STEP2: reasoning{text=%s}-content{text=%s}",
		firstReasoning,
		firstContent,
		secondReasoning,
		secondContent,
	)

	firstReq := ChatRequest{
		Messages: []ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: "What is the weather in Shanghai?"},
		},
		Stream: true,
		Tools: []ChatFunctionTool{
			functionTool("get_weather", "Get the current weather for a city"),
		},
	}
	if err := doStreamingJSONRequest(baseURL, "/api/v1/chat/completions", firstReq, 0); err != nil {
		return err
	}

	secondReq := ChatRequest{
		Messages: []ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: "What is the weather in Shanghai?"},
			{
				Role:      "assistant",
				Content:   firstContent,
				Reasoning: firstReasoning,
				ToolCalls: []ChatToolCall{
					{
						ID:   "call_weather_1",
						Type: "function",
						Function: FunctionCall{
							Name:      "get_weather",
							Arguments: `{"location":"Shanghai"}`,
						},
					},
				},
			},
			{
				Role:       "tool",
				ToolCallID: "call_weather_1",
				Content:    `{"temperature":26,"condition":"sunny"}`,
			},
		},
		Stream: true,
		Tools: []ChatFunctionTool{
			functionTool("get_weather", "Get the current weather for a city"),
		},
	}
	return doStreamingJSONRequest(baseURL, "/api/v1/chat/completions", secondReq, 0)
}

func runSlowStreaming(baseURL string) error {
	req := ChatRequest{
		Messages: []ChatMessage{
			{Role: "system", Content: `#CHAIN: content{char_delay=200ms,chunk_size=1,chunk_delay=250ms,text=Slow response under high latency}`},
			{Role: "user", Content: "Please stream slowly."},
		},
		Stream: true,
	}
	return doStreamingJSONRequest(baseURL, "/api/v1/chat/completions", req, 0)
}

func runMalformedJSONStreaming(baseURL string) error {
	req := ChatRequest{
		Messages: []ChatMessage{
			{Role: "system", Content: `#CHAIN: content{fault=malformed_json,fault_prob=1.0,text=This response should contain malformed JSON}`},
			{Role: "user", Content: "Please send malformed streaming JSON."},
		},
		Stream: true,
	}
	return doStreamingJSONRequest(baseURL, "/api/v1/chat/completions", req, 0)
}

func runImageStreaming(baseURL string) error {
	req := ChatRequest{
		Messages: []ChatMessage{
			{Role: "system", Content: `#CHAIN: image{url=https://example.com/mock-image.png,text=Image response follows}`},
			{Role: "user", Content: "Please return an image."},
		},
		Stream: true,
	}
	return doStreamingJSONRequest(baseURL, "/api/v1/chat/completions", req, 0)
}

func runAudioStreaming(baseURL string) error {
	req := ChatRequest{
		Messages: []ChatMessage{
			{Role: "system", Content: `#CHAIN: audio{transcript=Audio response follows}`},
			{Role: "user", Content: "Please return audio."},
		},
		Stream: true,
	}
	return doStreamingJSONRequest(baseURL, "/api/v1/chat/completions", req, 0)
}

func runTimeoutScenario(baseURL string) error {
	req := ChatRequest{
		Messages: []ChatMessage{
			{Role: "system", Content: `#CHAIN: content{fault=timeout,fault_prob=1.0,fault_duration=10s,text=This request should timeout}`},
			{Role: "user", Content: "Please timeout."},
		},
		Stream: true,
	}
	return doStreamingJSONRequest(baseURL, "/api/v1/chat/completions", req, 3*time.Second)
}

func runHealthCheck(baseURL string) error {
	return doGET(baseURL, "/health")
}

func runListModels(baseURL string) error {
	return doGET(baseURL, "/api/v1/models")
}

func doGET(baseURL, path string) error {
	url := strings.TrimRight(baseURL, "/") + path
	fmt.Printf("Request:\nGET %s\n\n", url)

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	printResponseMeta(resp)
	fmt.Println("Raw Response Body:")
	fmt.Println(string(body))
	return nil
}

func doStreamingJSONRequest(baseURL, path string, reqBody ChatRequest, timeout time.Duration) error {
	url := strings.TrimRight(baseURL, "/") + path
	payload, err := json.MarshalIndent(reqBody, "", "  ")
	if err != nil {
		return err
	}

	fmt.Printf("Request:\nPOST %s\n", url)
	fmt.Println("Headers:")
	fmt.Println("  Content-Type: application/json")
	fmt.Println("Request Body:")
	fmt.Println(string(payload))
	fmt.Println()

	client := &http.Client{}
	if timeout > 0 {
		client.Timeout = timeout
		fmt.Printf("Client Timeout: %s\n\n", timeout)
	}

	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(httpReq)
	if err != nil {
		fmt.Printf("Request Error: %v\n", err)
		return nil
	}
	defer resp.Body.Close()

	printResponseMeta(resp)
	fmt.Println("Raw Streaming Response:")

	reader := bufio.NewReader(resp.Body)
	var raw strings.Builder
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			fmt.Print(line)
			raw.WriteString(line)
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Printf("\nStream Read Error: %v\n", err)
			break
		}
	}

	fmt.Println()
	fmt.Println("Captured Raw Response:")
	fmt.Println(raw.String())
	return nil
}

func printResponseMeta(resp *http.Response) {
	fmt.Printf("Response Status: %s\n", resp.Status)
	fmt.Println("Response Headers:")
	for key, values := range resp.Header {
		fmt.Printf("  %s: %s\n", key, strings.Join(values, ", "))
	}
	fmt.Println()
}

func functionTool(name, description string) ChatFunctionTool {
	return ChatFunctionTool{
		Type: "function",
		Function: FunctionDefinition{
			Name:        name,
			Description: description,
		},
	}
}
