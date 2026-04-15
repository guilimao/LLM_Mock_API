package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests
type Handler struct {
	chainParser   *ChainParser
	streamHandler *StreamHandler
	testTools     TestToolRegistry
}

// NewHandler creates a new handler
func NewHandler(defaultModel string) *Handler {
	return &Handler{
		chainParser:   NewChainParser(),
		streamHandler: NewStreamHandler(defaultModel),
		testTools:     DefaultTestTools,
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
	
	// Test tool routes
	router.GET("/test-tools", h.ListTestTools)
	router.GET("/test-tools/:name", h.GetTestTool)
	router.POST("/test-tools/:name/invoke", h.InvokeTestTool)
	
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
	
	// Determine model
	model := req.Model
	if model == "" {
		model = "mock/llm-model"
	}
	h.streamHandler.modelName = model
	
	// Extract chain from system prompt
	chain, err := h.extractChainFromMessages(req.Messages, reasoningEnabled)
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
	
	// Handle streaming vs non-streaming
	if req.Stream {
		// Streaming response
		err := h.streamHandler.GenerateStream(c, req, chain, reasoningEnabled)
		if err != nil {
			// If streaming failed mid-way, we can't send JSON error
			fmt.Printf("Streaming error: %v\n", err)
		}
	} else {
		// Non-streaming response
		response := h.streamHandler.GenerateNonStream(req, chain, reasoningEnabled)
		c.JSON(http.StatusOK, response)
	}
}

// isReasoningEnabled checks if reasoning is enabled in the request
// Priority: 1. Check if Enabled field is explicitly set to true
//           2. Fall back to checking if effort is not "none"
func (h *Handler) isReasoningEnabled(req ChatRequest) bool {
	if req.Reasoning != nil {
		// Priority 1: Check Enabled field if it's explicitly set
		if req.Reasoning.Enabled != nil {
			return *req.Reasoning.Enabled
		}
		
		// Priority 2: Fall back to effort check
		// Reasoning is enabled if effort is not "none" or empty
		effort := strings.ToLower(req.Reasoning.Effort)
		if effort != "none" && effort != "" {
			return true
		}
	}
	return false
}

// extractChainFromMessages extracts the dialogue chain from messages
func (h *Handler) extractChainFromMessages(messages []ChatMessage, reasoningEnabled bool) (*ParsedChain, error) {
	// Look for system message with chain specification
	for _, msg := range messages {
		if msg.Role == "system" {
			content := h.extractStringContent(msg.Content)
			
			// Check if content contains chain specification
			// Chain spec format: #CHAIN: <chain-definition>
			if idx := strings.Index(content, "#CHAIN:"); idx != -1 {
				chainStr := strings.TrimSpace(content[idx+7:])
				// Extract until end of line or comment
				if nlIdx := strings.IndexAny(chainStr, "\n\r"); nlIdx != -1 {
					chainStr = chainStr[:nlIdx]
				}
				return h.chainParser.Parse(chainStr, reasoningEnabled)
			}
			
			// Also check for @chain directive
			if idx := strings.Index(content, "@chain"); idx != -1 {
				chainStr := strings.TrimSpace(content[idx+6:])
				if nlIdx := strings.IndexAny(chainStr, "\n\r"); nlIdx != -1 {
					chainStr = chainStr[:nlIdx]
				}
				return h.chainParser.Parse(chainStr, reasoningEnabled)
			}
		}
	}
	
	// Default chain if no specification found
	return h.chainParser.Parse("", reasoningEnabled)
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

// ListTestTools lists available test tools
func (h *Handler) ListTestTools(c *gin.Context) {
	c.JSON(http.StatusOK, h.testTools)
}

// GetTestTool gets a specific test tool specification
func (h *Handler) GetTestTool(c *gin.Context) {
	name := c.Param("name")
	tool, ok := h.testTools.Tools[name]
	if !ok {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error: struct {
				Message string `json:"message"`
				Type    string `json:"type,omitempty"`
				Code    string `json:"code,omitempty"`
			}{
				Message: fmt.Sprintf("Test tool '%s' not found", name),
				Type:    "not_found",
				Code:    "404",
			},
		})
		return
	}
	
	c.JSON(http.StatusOK, tool)
}

// InvokeTestTool invokes a test tool and returns the result
func (h *Handler) InvokeTestTool(c *gin.Context) {
	name := c.Param("name")
	tool, ok := h.testTools.Tools[name]
	if !ok {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error: struct {
				Message string `json:"message"`
				Type    string `json:"type,omitempty"`
				Code    string `json:"code,omitempty"`
			}{
				Message: fmt.Sprintf("Test tool '%s' not found", name),
				Type:    "not_found",
				Code:    "404",
			},
		})
		return
	}
	
	// Parse arguments
	var args map[string]interface{}
	if err := c.ShouldBindJSON(&args); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: struct {
				Message string `json:"message"`
				Type    string `json:"type,omitempty"`
				Code    string `json:"code,omitempty"`
			}{
				Message: fmt.Sprintf("Invalid arguments: %v", err),
				Type:    "invalid_request_error",
				Code:    "400",
			},
		})
		return
	}
	
	// Find matching response scenario
	var response TestToolResponse
	for _, resp := range tool.Responses {
		if resp.Scenario == "success" {
			response = resp
			break
		}
	}
	
	// Apply latency simulation
	if tool.Latency != nil {
		latencyRange := tool.Latency.Max - tool.Latency.Min
		if latencyRange > 0 {
			latency := time.Duration(tool.Latency.Min+rand.Intn(latencyRange)) * time.Millisecond
			time.Sleep(latency)
		}
	} else if response.Latency > 0 {
		time.Sleep(time.Duration(response.Latency) * time.Millisecond)
	}
	
	// Return error if configured
	if response.Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": response.Error,
		})
		return
	}
	
	// Return result
	var result interface{}
	if err := json.Unmarshal(response.Result, &result); err != nil {
		result = response.Result
	}
	
	c.JSON(http.StatusOK, gin.H{
		"tool":      name,
		"arguments": args,
		"result":    result,
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
			"multimodal",
			"fault_simulation",
		},
	})
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
