package main

import (
	"encoding/json"
	"time"
)

// ============ Chat Request Models ============

type ChatRequest struct {
	Messages           []ChatMessage          `json:"messages" binding:"required"`
	Model              string                 `json:"model,omitempty"`
	Models             []string               `json:"models,omitempty"`
	Stream             bool                   `json:"stream,omitempty"`
	StreamOptions      *ChatStreamOptions     `json:"stream_options,omitempty"`
	Temperature        *float64               `json:"temperature,omitempty"`
	TopP               *float64               `json:"top_p,omitempty"`
	MaxCompletionTokens *int                  `json:"max_completion_tokens,omitempty"`
	MaxTokens          *int                   `json:"max_tokens,omitempty"`
	Reasoning          *ChatReasoningConfig   `json:"reasoning,omitempty"`
	Tools              []ChatFunctionTool     `json:"tools,omitempty"`
	ToolChoice         interface{}            `json:"tool_choice,omitempty"`
	ResponseFormat     interface{}            `json:"response_format,omitempty"`
	Stop               interface{}            `json:"stop,omitempty"`
	Seed               *int                   `json:"seed,omitempty"`
	FrequencyPenalty   *float64               `json:"frequency_penalty,omitempty"`
	PresencePenalty    *float64               `json:"presence_penalty,omitempty"`
	LogitBias          map[string]float64     `json:"logit_bias,omitempty"`
	Logprobs           bool                   `json:"logprobs,omitempty"`
	TopLogprobs        *int                   `json:"top_logprobs,omitempty"`
	User               string                 `json:"user,omitempty"`
	Metadata           map[string]string      `json:"metadata,omitempty"`
	ParallelToolCalls  *bool                  `json:"parallel_tool_calls,omitempty"`
	Modalities         []string               `json:"modalities,omitempty"`
	Provider           *ProviderConfig        `json:"provider,omitempty"`
	Plugins            []PluginConfig         `json:"plugins,omitempty"`
	SessionID          string                 `json:"session_id,omitempty"`
	Trace              *TraceConfig           `json:"trace,omitempty"`
	ServiceTier        string                 `json:"service_tier,omitempty"`
	Debug              *ChatDebugOptions      `json:"debug,omitempty"`
}

type ChatMessage struct {
	Role         string      `json:"role"`
	Content      interface{} `json:"content,omitempty"`
	Name         string      `json:"name,omitempty"`
	ToolCalls    []ChatToolCall `json:"tool_calls,omitempty"`
	ToolCallID   string      `json:"tool_call_id,omitempty"`
	Reasoning    string      `json:"reasoning,omitempty"`
	Refusal      string      `json:"refusal,omitempty"`
}

type ChatReasoningConfig struct {
	Enabled  *bool       `json:"enabled,omitempty"`
	Effort   string      `json:"effort,omitempty"`
	Summary  interface{} `json:"summary,omitempty"`
}

type ChatStreamOptions struct {
	IncludeUsage bool `json:"include_usage,omitempty"`
}

type ChatFunctionTool struct {
	Type     string           `json:"type"`
	Function FunctionDefinition `json:"function"`
}

type FunctionDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
	Strict      *bool           `json:"strict,omitempty"`
}

type ProviderConfig struct {
	AllowFallbacks       *bool         `json:"allow_fallbacks,omitempty"`
	RequireParameters    *bool         `json:"require_parameters,omitempty"`
	DataCollection       string        `json:"data_collection,omitempty"`
	ZDR                  *bool         `json:"zdr,omitempty"`
	EnforceDistillable   *bool         `json:"enforce_distillable_text,omitempty"`
	Order                []string      `json:"order,omitempty"`
	Only                 []string      `json:"only,omitempty"`
	Ignore               []string      `json:"ignore,omitempty"`
	Quantizations        []string      `json:"quantizations,omitempty"`
	Sort                 interface{}   `json:"sort,omitempty"`
	MaxPrice             *MaxPriceConfig `json:"max_price,omitempty"`
	PreferredMinThroughput interface{} `json:"preferred_min_throughput,omitempty"`
	PreferredMaxLatency  interface{}   `json:"preferred_max_latency,omitempty"`
}

type MaxPriceConfig struct {
	Prompt     interface{} `json:"prompt,omitempty"`
	Completion interface{} `json:"completion,omitempty"`
	Image      interface{} `json:"image,omitempty"`
	Audio      interface{} `json:"audio,omitempty"`
	Request    interface{} `json:"request,omitempty"`
}

type PluginConfig struct {
	ID             string                 `json:"id"`
	Enabled        *bool                  `json:"enabled,omitempty"`
	AllowedModels  []string               `json:"allowed_models,omitempty"`
	MaxResults     int                    `json:"max_results,omitempty"`
	SearchPrompt   string                 `json:"search_prompt,omitempty"`
	Engine         string                 `json:"engine,omitempty"`
	IncludeDomains []string               `json:"include_domains,omitempty"`
	ExcludeDomains []string               `json:"exclude_domains,omitempty"`
	PDF            map[string]interface{} `json:"pdf,omitempty"`
}

type TraceConfig struct {
	TraceID        string      `json:"trace_id,omitempty"`
	TraceName      string      `json:"trace_name,omitempty"`
	SpanName       string      `json:"span_name,omitempty"`
	GenerationName string      `json:"generation_name,omitempty"`
	ParentSpanID   string      `json:"parent_span_id,omitempty"`
}

type ChatDebugOptions struct {
	// Add debug options as needed
}

// ============ Chat Response Models ============

type ChatResponse struct {
	ID               string           `json:"id"`
	Object           string           `json:"object"`
	Created          int64            `json:"created"`
	Model            string           `json:"model"`
	Choices          []ChatChoice     `json:"choices"`
	Usage            *ChatUsage       `json:"usage,omitempty"`
	SystemFingerprint string          `json:"system_fingerprint,omitempty"`
	ServiceTier      string           `json:"service_tier,omitempty"`
}

type ChatChoice struct {
	Index        int           `json:"index"`
	Message      ChatMessage   `json:"message"`
	FinishReason *string       `json:"finish_reason,omitempty"`
	Logprobs     interface{}   `json:"logprobs,omitempty"`
}

type ChatUsage struct {
	PromptTokens            int                     `json:"prompt_tokens"`
	CompletionTokens        int                     `json:"completion_tokens"`
	TotalTokens             int                     `json:"total_tokens"`
	PromptTokensDetails     *PromptTokensDetails    `json:"prompt_tokens_details,omitempty"`
	CompletionTokensDetails *CompletionTokensDetails `json:"completion_tokens_details,omitempty"`
}

type PromptTokensDetails struct {
	CachedTokens     int `json:"cached_tokens,omitempty"`
	CacheWriteTokens int `json:"cache_write_tokens,omitempty"`
	AudioTokens      int `json:"audio_tokens,omitempty"`
	VideoTokens      int `json:"video_tokens,omitempty"`
}

type CompletionTokensDetails struct {
	ReasoningTokens            int `json:"reasoning_tokens,omitempty"`
	AudioTokens                int `json:"audio_tokens,omitempty"`
	AcceptedPredictionTokens   int `json:"accepted_prediction_tokens,omitempty"`
	RejectedPredictionTokens   int `json:"rejected_prediction_tokens,omitempty"`
}

// ============ Streaming Models ============

type ChatStreamChunk struct {
	ID               string            `json:"id"`
	Object           string            `json:"object"`
	Created          int64             `json:"created"`
	Model            string            `json:"model"`
	Choices          []ChatStreamChoice `json:"choices"`
	Usage            *ChatUsage        `json:"usage,omitempty"`
	SystemFingerprint string           `json:"system_fingerprint,omitempty"`
	ServiceTier      string            `json:"service_tier,omitempty"`
	Error            *ErrorInfo        `json:"error,omitempty"`
}

type ChatStreamChoice struct {
	Index        int             `json:"index"`
	Delta        ChatStreamDelta `json:"delta"`
	FinishReason *string         `json:"finish_reason,omitempty"`
	Logprobs     interface{}     `json:"logprobs,omitempty"`
}

type ChatStreamDelta struct {
	Role             string             `json:"role,omitempty"`
	Content          string             `json:"content,omitempty"`
	Reasoning        string             `json:"reasoning,omitempty"`
	Refusal          string             `json:"refusal,omitempty"`
	ToolCalls        []ChatStreamToolCall `json:"tool_calls,omitempty"`
	Audio            interface{}        `json:"audio,omitempty"`
}

type ChatStreamToolCall struct {
	Index     int                      `json:"index"`
	ID        string                   `json:"id,omitempty"`
	Type      string                   `json:"type,omitempty"`
	Function  ChatStreamFunctionCall   `json:"function,omitempty"`
}

type ChatStreamFunctionCall struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

type ChatToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function FunctionCall     `json:"function"`
}

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ErrorInfo struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// ============ Content Types ============

type ChatContentText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type ChatContentImage struct {
	Type     string               `json:"type"`
	ImageURL ChatContentImageURL  `json:"image_url"`
}

type ChatContentImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"`
}

type ChatContentAudio struct {
	Type       string                `json:"type"`
	InputAudio ChatContentAudioInput `json:"input_audio"`
}

type ChatContentAudioInput struct {
	Data   string `json:"data"`
	Format string `json:"format"`
}

// ============ Test Tool Specification ============

// TestToolSpec defines the specification for test tools
// This allows clients to implement simulated tool calls
type TestToolSpec struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  TestToolParameters     `json:"parameters"`
	Responses   []TestToolResponse     `json:"responses"`
	Latency     *TestToolLatency       `json:"latency,omitempty"`
}

type TestToolParameters struct {
	Type       string                     `json:"type"`
	Properties map[string]json.RawMessage `json:"properties"`
	Required   []string                   `json:"required,omitempty"`
}

type TestToolResponse struct {
	Scenario    string          `json:"scenario"`
	Condition   string          `json:"condition,omitempty"`
	Result      json.RawMessage `json:"result"`
	Latency     int             `json:"latency_ms,omitempty"`
	Error       *TestToolError  `json:"error,omitempty"`
}

type TestToolError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type TestToolLatency struct {
	Min int `json:"min_ms"`
	Max int `json:"max_ms"`
}

// ============ Dialogue Chain Models ============

// ChainNode represents a single node in the dialogue chain
type ChainNode struct {
	Type           NodeType               `json:"type"`
	Content        string                 `json:"content,omitempty"`
	Reasoning      string                 `json:"reasoning,omitempty"`
	ToolCalls      []SimulatedToolCall    `json:"tool_calls,omitempty"`
	Multimodal     []MultimediaContent    `json:"multimedia,omitempty"`
	Speed          *TransmissionSpeed     `json:"speed,omitempty"`
	Fault          *FaultConfig           `json:"fault,omitempty"`
	Concurrency    bool                   `json:"concurrency,omitempty"`
}

// NodeType represents the type of chain node
type NodeType string

const (
	NodeTypeContent    NodeType = "content"
	NodeTypeReasoning  NodeType = "reasoning"
	NodeTypeToolCalls  NodeType = "tool_calls"
	NodeTypeMixed      NodeType = "mixed"
)

// SimulatedToolCall represents a tool call in the chain
type SimulatedToolCall struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
	Result    interface{}     `json:"result,omitempty"`
}

// MultimediaContent represents multi-modal output content
type MultimediaContent struct {
	Type     string `json:"type"`
	URL      string `json:"url,omitempty"`
	Data     string `json:"data,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
	Text     string `json:"text,omitempty"`
}

// TransmissionSpeed controls the output speed
type TransmissionSpeed struct {
	CharDelay      time.Duration `json:"char_delay_ms"`
	ChunkSize      int           `json:"chunk_size"`
	ChunkDelay     time.Duration `json:"chunk_delay_ms"`
}

// FaultConfig defines fault simulation configuration
type FaultConfig struct {
	Type            FaultType         `json:"type"`
	Probability     float64           `json:"probability"`
	Duration        time.Duration     `json:"duration,omitempty"`
	AfterBytes      int               `json:"after_bytes,omitempty"`
	RecoveryAfter   time.Duration     `json:"recovery_after,omitempty"`
	CorruptionLevel float64           `json:"corruption_level,omitempty"`
}

// FaultType represents the type of fault to simulate
type FaultType string

const (
	FaultTypeNone           FaultType = "none"
	FaultTypeDelay          FaultType = "delay"
	FaultTypeInterrupt      FaultType = "interrupt"
	FaultTypePacketLoss     FaultType = "packet_loss"
	FaultTypeCorruption     FaultType = "corruption"
	FaultTypeTimeout        FaultType = "timeout"
	FaultTypePartialJSON    FaultType = "partial_json"
	FaultTypeMalformedJSON  FaultType = "malformed_json"
)

// ParsedChain represents a parsed dialogue chain
type ParsedChain struct {
	Segments [][]ChainNode
}

// ============ Error Response Models ============

type ErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type,omitempty"`
		Code    string `json:"code,omitempty"`
	} `json:"error"`
}

// TestToolRegistry holds available test tools
type TestToolRegistry struct {
	Tools map[string]TestToolSpec `json:"tools"`
}

// Default test tools specification
var DefaultTestTools = TestToolRegistry{
	Tools: map[string]TestToolSpec{
		"get_weather": {
			Name:        "get_weather",
			Description: "Get current weather information for a location",
			Parameters: TestToolParameters{
				Type: "object",
				Properties: map[string]json.RawMessage{
					"location": json.RawMessage(`{"type": "string", "description": "City name or coordinates"}`),
					"unit":     json.RawMessage(`{"type": "string", "enum": ["celsius", "fahrenheit"], "default": "celsius"}`),
				},
				Required: []string{"location"},
			},
			Responses: []TestToolResponse{
				{
					Scenario: "success",
					Result:   json.RawMessage(`{"temperature": 22, "condition": "sunny", "humidity": 65}`),
					Latency:  100,
				},
				{
					Scenario:  "not_found",
					Condition: "location=unknown",
					Error: &TestToolError{
						Code:    "LOCATION_NOT_FOUND",
						Message: "The specified location could not be found",
					},
					Latency: 50,
				},
			},
			Latency: &TestToolLatency{Min: 50, Max: 500},
		},
		"calculate": {
			Name:        "calculate",
			Description: "Perform mathematical calculations",
			Parameters: TestToolParameters{
				Type: "object",
				Properties: map[string]json.RawMessage{
					"expression": json.RawMessage(`{"type": "string", "description": "Mathematical expression to evaluate"}`),
				},
				Required: []string{"expression"},
			},
			Responses: []TestToolResponse{
				{
					Scenario: "success",
					Result:   json.RawMessage(`{"result": 42}`),
					Latency:  50,
				},
			},
			Latency: &TestToolLatency{Min: 10, Max: 100},
		},
		"search": {
			Name:        "search",
			Description: "Search for information",
			Parameters: TestToolParameters{
				Type: "object",
				Properties: map[string]json.RawMessage{
					"query": json.RawMessage(`{"type": "string", "description": "Search query"}`),
					"limit": json.RawMessage(`{"type": "integer", "description": "Maximum number of results", "default": 5}`),
				},
				Required: []string{"query"},
			},
			Responses: []TestToolResponse{
				{
					Scenario: "success",
					Result:   json.RawMessage(`{"results": [{"title": "Example", "url": "https://example.com"}], "total": 1}`),
					Latency:  200,
				},
			},
			Latency: &TestToolLatency{Min: 100, Max: 1000},
		},
		"generate_image": {
			Name:        "generate_image",
			Description: "Generate an image from a text description",
			Parameters: TestToolParameters{
				Type: "object",
				Properties: map[string]json.RawMessage{
					"prompt": json.RawMessage(`{"type": "string", "description": "Image description"}`),
					"size":   json.RawMessage(`{"type": "string", "enum": ["256x256", "512x512", "1024x1024"], "default": "512x512"}`),
				},
				Required: []string{"prompt"},
			},
			Responses: []TestToolResponse{
				{
					Scenario: "success",
					Result:   json.RawMessage(`{"url": "https://mock.example.com/image.png", "revised_prompt": "Enhanced description"}`),
					Latency:  2000,
				},
			},
			Latency: &TestToolLatency{Min: 1000, Max: 5000},
		},
	},
}
