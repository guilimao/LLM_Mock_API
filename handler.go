package main

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests
type Handler struct {
	chainParser   *ChainParser
	streamHandler *StreamHandler
}

// NewHandler creates a new handler
func NewHandler(defaultModel string) *Handler {
	return &Handler{
		chainParser:   NewChainParser(),
		streamHandler: NewStreamHandler(defaultModel),
	}
}

// RegisterRoutes registers all routes
func (h *Handler) RegisterRoutes(router *gin.Engine) {
	// API routes
	api := router.Group("/api/v1")
	{
		api.POST("/chat/completions", h.ChatCompletions)
		api.GET("/models", h.ListModels)
	}

	// Health check
	router.GET("/health", h.HealthCheck)

	// Fault preset routes
	router.GET("/fault-presets", h.ListFaultPresets)
}

// ChatCompletions handles chat completion requests
func (h *Handler) ChatCompletions(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: struct {
				Message string `json:"message"`
				Type    string `json:"type,omitempty"`
				Code    string `json:"code,omitempty"`
			}{
				Message: fmt.Sprintf("Invalid request: %v", err),
				Type:    "invalid_request_error",
				Code:    "400",
			},
		})
		return
	}
	// Determine if reasoning is enabled
	reasoningEnabled := h.isReasoningEnabled(req)

	// Determine if reasoning should be excluded from response
	reasoningExcluded := h.isReasoningExcluded(req)

	// Determine model
	model := req.Model
	if model == "" {
		model = "mock/llm-model"
	}
	h.streamHandler.modelName = model
	execOpts := NewRequestExecutionOptions(req)

	// Extract chain from system prompt
	chain, trace, err := h.extractChainFromMessages(req, reasoningEnabled, reasoningExcluded, execOpts)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: struct {
				Message string `json:"message"`
				Type    string `json:"type,omitempty"`
				Code    string `json:"code,omitempty"`
			}{
				Message: fmt.Sprintf("Failed to parse chain: %v", err),
				Type:    "invalid_request_error",
				Code:    "400",
			},
		})
		return
	}
	h.writeChainTraceHeaders(c, execOpts, trace)
	// Handle streaming vs non-streaming
	if req.Stream {
		// Streaming response
		err := h.streamHandler.GenerateStream(c, req, chain, reasoningEnabled, reasoningExcluded, execOpts)
		if err != nil {
			// If streaming failed mid-way, we can't send JSON error
			fmt.Printf("Streaming error: %v\n", err)
		}
	} else {
		// Non-streaming response
		response := h.streamHandler.GenerateNonStream(req, chain, reasoningEnabled, reasoningExcluded, execOpts)
		c.JSON(http.StatusOK, response)
	}
}

// isReasoningEnabled checks if reasoning is enabled in the request
// Priority: 1. Check if Enabled field is explicitly set
//  2. Check if max_tokens is specified (Anthropic-style)
//  3. Fall back to checking if effort is not "none"
func (h *Handler) isReasoningEnabled(req ChatRequest) bool {
	if req.Reasoning != nil {
		// Priority 1: Check Enabled field if it's explicitly set
		if req.Reasoning.Enabled != nil {
			return *req.Reasoning.Enabled
		}

		// Priority 2: Check if max_tokens is specified (Anthropic-style reasoning)
		if req.Reasoning.MaxTokens != nil && *req.Reasoning.MaxTokens > 0 {
			return true
		}

		// Priority 3: Fall back to effort check
		// Reasoning is enabled if effort is not "none" or empty
		effort := strings.ToLower(req.Reasoning.Effort)
		result := effort != "none" && effort != ""
		return result
	}
	return false
}

// isReasoningExcluded checks if reasoning tokens should be excluded from response
func (h *Handler) isReasoningExcluded(req ChatRequest) bool {
	if req.Reasoning != nil && req.Reasoning.Exclude != nil {
		return *req.Reasoning.Exclude
	}
	return false
}

// getReasoningMaxTokens gets the max_tokens value for reasoning (Anthropic-style)
func (h *Handler) getReasoningMaxTokens(req ChatRequest) *int {
	if req.Reasoning != nil {
		return req.Reasoning.MaxTokens
	}
	return nil
}

// getReasoningEffort gets the effort value for reasoning (OpenAI-style)
func (h *Handler) getReasoningEffort(req ChatRequest) string {
	if req.Reasoning != nil {
		return req.Reasoning.Effort
	}
	return ""
}

// extractChainFromMessages extracts the dialogue chain from messages
func (h *Handler) extractChainFromMessages(req ChatRequest, reasoningEnabled bool, reasoningExcluded bool, execOpts *RequestExecutionOptions) (*ParsedChain, ChainSelectionTrace, error) {
	stepChains := h.extractChainSteps(req.Messages)
	selectedStep, fallbackUsed := selectChainStep(stepChains, countCompletedToolRounds(req.Messages))

	chainStr := ""
	if selectedStep > 0 {
		chainStr = stepChains[selectedStep]
	}

	chain, err := h.chainParser.Parse(chainStr, reasoningEnabled, reasoningExcluded)
	if err != nil {
		return nil, ChainSelectionTrace{}, err
	}

	trace := ChainSelectionTrace{
		SelectedStep: selectedStep,
		FallbackUsed: fallbackUsed,
	}
	trace.ToolMismatch = h.normalizeChainToolCalls(chain, req.Tools, execOpts)
	return chain, trace, nil
}

// extractStringContent extracts string content from message content
func (h *Handler) extractStringContent(content interface{}) string {
	switch v := content.(type) {
	case string:
		return v
	case []interface{}:
		var parts []string
		for _, item := range v {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if text, ok := itemMap["text"].(string); ok {
					parts = append(parts, text)
				}
			}
		}
		return strings.Join(parts, "\n")
	default:
		return ""
	}
}

// ListModels lists available models
func (h *Handler) ListModels(c *gin.Context) {
	models := []gin.H{
		{
			"id":       "mock/llm-model",
			"object":   "model",
			"created":  time.Now().Unix(),
			"owned_by": "mock",
		},
		{
			"id":       "mock/reasoning-model",
			"object":   "model",
			"created":  time.Now().Unix(),
			"owned_by": "mock",
		},
		{
			"id":       "mock/tool-model",
			"object":   "model",
			"created":  time.Now().Unix(),
			"owned_by": "mock",
		},
		{
			"id":       "mock/multimodal-model",
			"object":   "model",
			"created":  time.Now().Unix(),
			"owned_by": "mock",
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   models,
	})
}

// HealthCheck returns health status
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"version":   "1.0.0",
		"features": []string{
			"streaming",
			"reasoning",
			"tool_calls",
			"chain_steps",
			"multimodal",
			"fault_simulation",
		},
	})
}

func (h *Handler) extractChainSteps(messages []ChatMessage) map[int]string {
	steps := map[int]string{}
	for _, msg := range messages {
		if msg.Role != "system" {
			continue
		}

		content := h.extractStringContent(msg.Content)
		for _, line := range strings.Split(content, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			if strings.HasPrefix(line, "#CHAIN_STEP") {
				colonIdx := strings.Index(line, ":")
				if colonIdx == -1 {
					continue
				}
				stepLabel := strings.TrimPrefix(line[:colonIdx], "#CHAIN_STEP")
				step, err := strconv.Atoi(strings.TrimSpace(stepLabel))
				if err != nil || step <= 0 {
					continue
				}
				steps[step] = strings.TrimSpace(line[colonIdx+1:])
				continue
			}

			if strings.HasPrefix(line, "#CHAIN:") {
				steps[1] = strings.TrimSpace(line[len("#CHAIN:"):])
				continue
			}

			if strings.HasPrefix(line, "@chain") {
				steps[1] = strings.TrimSpace(line[len("@chain"):])
			}
		}
	}
	return steps
}

func countCompletedToolRounds(messages []ChatMessage) int {
	completed := 0
	awaitingToolResult := false

	for _, msg := range messages {
		if msg.Role == "assistant" && len(msg.ToolCalls) > 0 {
			awaitingToolResult = true
			continue
		}
		if msg.Role == "tool" && awaitingToolResult {
			completed++
			awaitingToolResult = false
		}
	}

	return completed
}

func selectChainStep(stepChains map[int]string, completedRounds int) (int, bool) {
	if len(stepChains) == 0 {
		return 0, false
	}

	target := completedRounds + 1
	if _, ok := stepChains[target]; ok {
		return target, false
	}

	definedSteps := make([]int, 0, len(stepChains))
	for step := range stepChains {
		definedSteps = append(definedSteps, step)
	}
	sort.Ints(definedSteps)
	return definedSteps[len(definedSteps)-1], true
}

func (h *Handler) normalizeChainToolCalls(chain *ParsedChain, tools []ChatFunctionTool, execOpts *RequestExecutionOptions) bool {
	definedTools := collectDefinedTools(tools)
	mismatch := false
	callIndex := 1

	for segIdx := range chain.Segments {
		for nodeIdx := range chain.Segments[segIdx] {
			node := &chain.Segments[segIdx][nodeIdx]
			if node.Type != NodeTypeToolCalls {
				continue
			}
			for toolIdx := range node.ToolCalls {
				toolCall := &node.ToolCalls[toolIdx]
				if toolCall.ID == "" {
					toolCall.ID = execOpts.NextToolCallID(callIndex)
				}
				callIndex++

				if definedTools == nil {
					continue
				}
				if _, ok := definedTools[toolCall.Name]; !ok {
					mismatch = true
				}
			}
		}
	}

	return mismatch
}

func (h *Handler) writeChainTraceHeaders(c *gin.Context, execOpts *RequestExecutionOptions, trace ChainSelectionTrace) {
	if !execOpts.ChainTrace {
		return
	}

	c.Header("X-Mock-Chain-Step", strconv.Itoa(trace.SelectedStep))
	c.Header("X-Mock-Chain-Fallback", strconv.FormatBool(trace.FallbackUsed))
	c.Header("X-Mock-Tool-Mismatch", strconv.FormatBool(trace.ToolMismatch))
}

// ListFaultPresets lists available fault presets
func (h *Handler) ListFaultPresets(c *gin.Context) {
	presets := make(map[string]interface{})
	for name, config := range FaultPresets {
		presets[name] = config
	}

	c.JSON(http.StatusOK, gin.H{
		"presets": presets,
	})
}

// Middleware for authentication simulation
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		// Allow requests without auth for mock server
		if authHeader == "" {
			c.Next()
			return
		}

		// Validate Bearer token format if present
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error: struct {
					Message string `json:"message"`
					Type    string `json:"type,omitempty"`
					Code    string `json:"code,omitempty"`
				}{
					Message: "Invalid authorization header format. Expected 'Bearer <token>'",
					Type:    "authentication_error",
					Code:    "401",
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// Middleware for CORS
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-OpenRouter-Title, HTTP-Referer, X-OpenRouter-Categories")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
