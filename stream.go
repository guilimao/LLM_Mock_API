package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// StreamHandler handles streaming responses
type StreamHandler struct {
	chainParser   *ChainParser
	faultSim      *FaultSimulator
	modelName     string
}

// NewStreamHandler creates a new stream handler
func NewStreamHandler(modelName string) *StreamHandler {
	return &StreamHandler{
		chainParser: NewChainParser(),
		faultSim:    NewFaultSimulator(),
		modelName:   modelName,
	}
}

// GenerateStream generates a streaming response
func (sh *StreamHandler) GenerateStream(c *gin.Context, req ChatRequest, chain *ParsedChain, reasoningEnabled bool) error {
	// Set headers for SSE
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	
	// Get the gin writer which supports Flusher
	writer := c.Writer
	flusher := writer
	
	// Generate unique ID
	streamID := generateStreamID()
	created := time.Now().Unix()
	
	// Send initial role message
	if err := sh.sendRoleChunk(writer, flusher, streamID, created, 0); err != nil {
		return err
	}
	
	choiceIndex := 0
	bytesSent := 0
	
	// Process each segment
	for segIdx, segment := range chain.Segments {
		// Process nodes within segment
		for nodeIdx, node := range segment {
			isLastNode := (segIdx == len(chain.Segments)-1) && (nodeIdx == len(segment)-1)
			
			if err := sh.processNode(writer, flusher, node, streamID, created, choiceIndex, &bytesSent, isLastNode); err != nil {
				// Client disconnected or other error
				return nil
			}
		}
		
		// If this is a concurrent segment, handle concurrency
		if len(segment) > 1 && sh.hasConcurrentNodes(segment) {
			// Concurrent processing already handled in processNode
			choiceIndex++
		}
	}
	
	// Send final [DONE] message
	if _, err := writer.WriteString("data: [DONE]\n\n"); err != nil {
		return err
	}
	flusher.Flush()
	
	return nil
}

func (sh *StreamHandler) sendRoleChunk(writer gin.ResponseWriter, flusher gin.ResponseWriter, id string, created int64, index int) error {
	chunk := ChatStreamChunk{
		ID:      id,
		Object:  "chat.completion.chunk",
		Created: created,
		Model:   sh.modelName,
		Choices: []ChatStreamChoice{
			{
				Index: index,
				Delta: ChatStreamDelta{
					Role: "assistant",
				},
				FinishReason: nil,
			},
		},
	}
	
	return sh.sendChunk(writer, flusher, chunk, nil, nil)
}

func (sh *StreamHandler) processNode(writer gin.ResponseWriter, flusher gin.ResponseWriter, node ChainNode, id string, created int64, index int, bytesSent *int, isLastNode bool) error {
	// Apply transmission speed settings
	speed := node.Speed
	if speed == nil {
		speed = &TransmissionSpeed{
			CharDelay:  5 * time.Millisecond,
			ChunkSize:  10,
			ChunkDelay: 20 * time.Millisecond,
		}
	}
	
	// Create fault injector if fault is configured
	var faultInjector func(string) (string, error)
	if node.Fault != nil {
		faultInjector = sh.faultSim.StreamingFaultInjector(node.Fault, bytesSent)
	}
	
	switch node.Type {
	case NodeTypeReasoning:
		return sh.streamReasoning(writer, flusher, node, id, created, index, speed, faultInjector, isLastNode)
	case NodeTypeContent:
		return sh.streamContent(writer, flusher, node, id, created, index, speed, faultInjector, isLastNode)
	case NodeTypeToolCalls:
		return sh.streamToolCalls(writer, flusher, node, id, created, index, speed, faultInjector, isLastNode)
	case NodeTypeMixed:
		return sh.streamMixed(writer, flusher, node, id, created, index, speed, faultInjector, isLastNode)
	default:
		return sh.streamContent(writer, flusher, node, id, created, index, speed, faultInjector, isLastNode)
	}
}

func (sh *StreamHandler) streamReasoning(writer gin.ResponseWriter, flusher gin.ResponseWriter, node ChainNode, id string, created int64, index int, speed *TransmissionSpeed, faultInjector func(string) (string, error), isLastNode bool) error {
	text := node.Reasoning
	if text == "" {
		text = "Analyzing the request..."
	}
	
	// Stream reasoning content in chunks
	chunks := sh.splitIntoChunks(text, speed.ChunkSize)
	
	for i, chunk := range chunks {
		isLastChunk := (i == len(chunks)-1) && isLastNode
		
		streamChunk := ChatStreamChunk{
			ID:      id,
			Object:  "chat.completion.chunk",
			Created: created,
			Model:   sh.modelName,
			Choices: []ChatStreamChoice{
				{
					Index: index,
					Delta: ChatStreamDelta{
						Reasoning: chunk,
					},
					FinishReason: nil,
				},
			},
		}
		
		if isLastChunk {
			finishReason := "stop"
			streamChunk.Choices[0].FinishReason = &finishReason
		}
		
		if err := sh.sendChunk(writer, flusher, streamChunk, speed, faultInjector); err != nil {
			return err
		}
		
		if speed.ChunkDelay > 0 {
			time.Sleep(speed.ChunkDelay)
		}
	}
	
	return nil
}

func (sh *StreamHandler) streamContent(writer gin.ResponseWriter, flusher gin.ResponseWriter, node ChainNode, id string, created int64, index int, speed *TransmissionSpeed, faultInjector func(string) (string, error), isLastNode bool) error {
	text := node.Content
	
	// Handle multimodal content
	if len(node.Multimodal) > 0 {
		// For now, just send text representation of multimodal content
		for _, mm := range node.Multimodal {
			switch mm.Type {
			case "image":
				text += fmt.Sprintf("\n[Image: %s]\n", mm.URL)
			case "audio":
				text += fmt.Sprintf("\n[Audio: %s]\n", mm.Text)
			}
		}
	}
	
	if text == "" {
		text = " "
	}
	
	// Stream content in chunks
	chunks := sh.splitIntoChunks(text, speed.ChunkSize)
	
	for i, chunk := range chunks {
		isLastChunk := (i == len(chunks)-1) && isLastNode
		
		streamChunk := ChatStreamChunk{
			ID:      id,
			Object:  "chat.completion.chunk",
			Created: created,
			Model:   sh.modelName,
			Choices: []ChatStreamChoice{
				{
					Index: index,
					Delta: ChatStreamDelta{
						Content: chunk,
					},
					FinishReason: nil,
				},
			},
		}
		
		if isLastChunk {
			finishReason := "stop"
			streamChunk.Choices[0].FinishReason = &finishReason
		}
		
		if err := sh.sendChunk(writer, flusher, streamChunk, speed, faultInjector); err != nil {
			return err
		}
		
		if speed.ChunkDelay > 0 && !isLastChunk {
			time.Sleep(speed.ChunkDelay)
		}
	}
	
	return nil
}

func (sh *StreamHandler) streamToolCalls(writer gin.ResponseWriter, flusher gin.ResponseWriter, node ChainNode, id string, created int64, index int, speed *TransmissionSpeed, faultInjector func(string) (string, error), isLastNode bool) error {
	if len(node.ToolCalls) == 0 {
		return nil
	}
	
	// Stream tool calls
	for toolIdx, toolCall := range node.ToolCalls {
		isLastTool := (toolIdx == len(node.ToolCalls)-1) && isLastNode
		
		// Send tool call ID and name first
		toolChunk := ChatStreamChunk{
			ID:      id,
			Object:  "chat.completion.chunk",
			Created: created,
			Model:   sh.modelName,
			Choices: []ChatStreamChoice{
				{
					Index: index,
					Delta: ChatStreamDelta{
						ToolCalls: []ChatStreamToolCall{
							{
								Index: toolIdx,
								ID:    toolCall.ID,
								Type:  "function",
								Function: ChatStreamFunctionCall{
									Name: toolCall.Name,
								},
							},
						},
					},
					FinishReason: nil,
				},
			},
		}
		
		if err := sh.sendChunk(writer, flusher, toolChunk, speed, faultInjector); err != nil {
			return err
		}
		
		time.Sleep(speed.ChunkDelay)
		
		// Stream arguments in chunks
		argsStr := string(toolCall.Arguments)
		argChunks := sh.splitIntoChunks(argsStr, speed.ChunkSize)
		
		for argIdx, argChunk := range argChunks {
			isLastArg := (argIdx == len(argChunks)-1) && isLastTool
			
			argToolChunk := ChatStreamChunk{
				ID:      id,
				Object:  "chat.completion.chunk",
				Created: created,
				Model:   sh.modelName,
				Choices: []ChatStreamChoice{
					{
						Index: index,
						Delta: ChatStreamDelta{
							ToolCalls: []ChatStreamToolCall{
								{
									Index: toolIdx,
									Function: ChatStreamFunctionCall{
										Arguments: argChunk,
									},
								},
							},
						},
						FinishReason: nil,
					},
				},
			}
			
			if isLastArg {
				finishReason := "tool_calls"
				argToolChunk.Choices[0].FinishReason = &finishReason
			}
			
			if err := sh.sendChunk(writer, flusher, argToolChunk, speed, faultInjector); err != nil {
				return err
			}
			
			if speed.ChunkDelay > 0 && !isLastArg {
				time.Sleep(speed.ChunkDelay)
			}
		}
	}
	
	return nil
}

func (sh *StreamHandler) streamMixed(writer gin.ResponseWriter, flusher gin.ResponseWriter, node ChainNode, id string, created int64, index int, speed *TransmissionSpeed, faultInjector func(string) (string, error), isLastNode bool) error {
	// First stream reasoning if present
	if node.Reasoning != "" {
		reasoningNode := ChainNode{
			Type:      NodeTypeReasoning,
			Reasoning: node.Reasoning,
			Speed:     speed,
			Fault:     node.Fault,
		}
		if err := sh.streamReasoning(writer, flusher, reasoningNode, id, created, index, speed, faultInjector, false); err != nil {
			return err
		}
	}
	
	// Then stream content
	contentNode := ChainNode{
		Type:       NodeTypeContent,
		Content:    node.Content,
		Multimodal: node.Multimodal,
		Speed:      speed,
		Fault:      node.Fault,
	}
	if err := sh.streamContent(writer, flusher, contentNode, id, created, index, speed, faultInjector, isLastNode); err != nil {
		return err
	}
	
	return nil
}

func (sh *StreamHandler) sendChunk(writer gin.ResponseWriter, flusher gin.ResponseWriter, chunk ChatStreamChunk, speed *TransmissionSpeed, faultInjector func(string) (string, error)) error {
	data, err := json.Marshal(chunk)
	if err != nil {
		return err
	}
	
	sseData := "data: " + string(data) + "\n\n"
	
	// Apply fault injection if configured
	if faultInjector != nil {
		modified, err := faultInjector(sseData)
		if err != nil {
			// If fault caused an error, still try to send what we have
			// but mark it as potentially corrupted
			sseData = modified
			// Log the fault
			fmt.Printf("[Fault Injected] Type: error, Error: %v\n", err)
		} else if modified != sseData {
			sseData = modified
		}
	}
	
	_, err = writer.WriteString(sseData)
	if err != nil {
		return err
	}
	
	flusher.Flush()
	
	return nil
}

func (sh *StreamHandler) splitIntoChunks(text string, chunkSize int) []string {
	if chunkSize <= 0 {
		chunkSize = 10
	}
	
	var chunks []string
	runes := []rune(text)
	
	for i := 0; i < len(runes); i += chunkSize {
		end := i + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[i:end]))
	}
	
	return chunks
}

func (sh *StreamHandler) hasConcurrentNodes(segment []ChainNode) bool {
	for _, node := range segment {
		if node.Concurrency {
			return true
		}
	}
	return false
}

func generateStreamID() string {
	return fmt.Sprintf("chatcmpl-%d%d", time.Now().Unix(), rand.Intn(1000000))
}

// Non-streaming response generation
func (sh *StreamHandler) GenerateNonStream(req ChatRequest, chain *ParsedChain, reasoningEnabled bool) *ChatResponse {
	response := &ChatResponse{
		ID:      generateStreamID(),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   sh.modelName,
		Choices: []ChatChoice{},
		Usage: &ChatUsage{
			PromptTokens:     estimateTokens(req.Messages),
			CompletionTokens: 0,
			TotalTokens:      0,
		},
	}
	
	// Build response content from chain
	var content strings.Builder
	var reasoning strings.Builder
	var toolCalls []ChatToolCall
	
	for _, segment := range chain.Segments {
		for _, node := range segment {
			switch node.Type {
			case NodeTypeReasoning:
				if reasoningEnabled {
					reasoning.WriteString(node.Reasoning)
				}
			case NodeTypeContent:
				content.WriteString(node.Content)
			case NodeTypeMixed:
				if reasoningEnabled && node.Reasoning != "" {
					reasoning.WriteString(node.Reasoning)
				}
				content.WriteString(node.Content)
			case NodeTypeToolCalls:
				for _, tc := range node.ToolCalls {
					toolCalls = append(toolCalls, ChatToolCall{
						ID:   tc.ID,
						Type: "function",
						Function: FunctionCall{
							Name:      tc.Name,
							Arguments: string(tc.Arguments),
						},
					})
				}
			}
		}
	}
	
	message := ChatMessage{
		Role:    "assistant",
		Content: content.String(),
	}
	
	if reasoningEnabled && reasoning.Len() > 0 {
		message.Reasoning = reasoning.String()
	}
	
	if len(toolCalls) > 0 {
		message.ToolCalls = toolCalls
	}
	
	finishReason := "stop"
	if len(toolCalls) > 0 {
		finishReason = "tool_calls"
	}
	
	response.Choices = append(response.Choices, ChatChoice{
		Index:        0,
		Message:      message,
		FinishReason: &finishReason,
	})
	
	// Estimate completion tokens
	completionTokens := estimateTokensString(content.String() + reasoning.String())
	response.Usage.CompletionTokens = completionTokens
	response.Usage.TotalTokens = response.Usage.PromptTokens + completionTokens
	
	if reasoningEnabled && reasoning.Len() > 0 {
		response.Usage.CompletionTokensDetails = &CompletionTokensDetails{
			ReasoningTokens: estimateTokensString(reasoning.String()),
		}
	}
	
	return response
}

// estimateTokens estimates token count from messages (rough approximation)
func estimateTokens(messages []ChatMessage) int {
	total := 0
	for _, msg := range messages {
		switch content := msg.Content.(type) {
		case string:
			total += estimateTokensString(content)
		case []interface{}:
			for _, item := range content {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if text, ok := itemMap["text"].(string); ok {
						total += estimateTokensString(text)
					}
				}
			}
		}
		total += 4 // Message overhead
	}
	return total
}

func estimateTokensString(s string) int {
	// Rough approximation: 1 token ≈ 4 characters for English
	return len([]rune(s)) / 4
}
