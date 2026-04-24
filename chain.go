package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ChainParser handles parsing of dialogue chain specifications
type ChainParser struct{}

// NewChainParser creates a new chain parser
func NewChainParser() *ChainParser {
	return &ChainParser{}
}

// Parse parses a dialogue chain string into a structured format
// Format example: "reasoning-content-tool_calls-reasoning-content-tool_calls, tool_calls, tool_calls-reasoning-content"
// - segments are separated by ", " (comma + space) for sequential execution
// - nodes within a segment are separated by "-" for ordered execution within the segment
// - all nodes in a segment marked with concurrency=true are executed concurrently
func (p *ChainParser) Parse(chainStr string, reasoningEnabled bool, reasoningExcluded bool) (*ParsedChain, error) {
	if chainStr == "" {
		// Default chain: simple content response
		// If reasoning is enabled but excluded, don't generate reasoning content
		if reasoningEnabled && !reasoningExcluded {
			return &ParsedChain{
				Segments: [][]ChainNode{
					{{Type: NodeTypeReasoning, Reasoning: "Let me think about this step by step..."},
						{Type: NodeTypeContent, Content: "This is a default response from the mock API with reasoning enabled."}},
				},
			}, nil
		}
		return &ParsedChain{
			Segments: [][]ChainNode{
				{{Type: NodeTypeContent, Content: "This is a default response from the mock API."}},
			},
		}, nil
	}

	chainStr = strings.TrimSpace(chainStr)

	// Split into segments (separated by comma for concurrent groups)
	segmentStrs := splitTopLevel(chainStr, ',')

	segments := make([][]ChainNode, 0, len(segmentStrs))

	for _, segStr := range segmentStrs {
		segStr = strings.TrimSpace(segStr)
		if segStr == "" {
			continue
		}

		// Split segment into nodes (separated by dash for sequential nodes)
		nodeStrs := splitTopLevel(segStr, '-')

		nodes := make([]ChainNode, 0, len(nodeStrs))
		for _, nodeStr := range nodeStrs {
			nodeStr = strings.TrimSpace(nodeStr)
			if nodeStr == "" {
				continue
			}

			// Check if this segment should be concurrent (marked with prefix "parallel:")
			concurrent := false
			if strings.HasPrefix(nodeStr, "parallel:") {
				concurrent = true
				nodeStr = strings.TrimPrefix(nodeStr, "parallel:")
				nodeStr = strings.TrimSpace(nodeStr)
			}

			node, err := p.parseNode(nodeStr, reasoningEnabled, reasoningExcluded)
			if err != nil {
				return nil, fmt.Errorf("failed to parse node '%s': %w", nodeStr, err)
			}
			node.Concurrency = concurrent
			nodes = append(nodes, *node)
		}

		if len(nodes) > 0 {
			segments = append(segments, nodes)
		}
	}

	return &ParsedChain{Segments: segments}, nil
}

// parseNode parses a single node string into a ChainNode
func (p *ChainParser) parseNode(nodeStr string, reasoningEnabled bool, reasoningExcluded bool) (*ChainNode, error) {
	// Check for extended format with parameters: "type{param1=value1,param2=value2}"
	baseType, params := p.extractParams(nodeStr)

	switch strings.ToLower(baseType) {
	case "reasoning", "think", "thinking":
		if !reasoningEnabled || reasoningExcluded {
			// Return empty content node if reasoning is disabled or excluded
			fmt.Printf("[DEBUG parseNode] Reasoning disabled or excluded, converting reasoning node to empty content\n")
			return &ChainNode{Type: NodeTypeContent, Content: ""}, nil
		}
		return p.parseReasoningNode(params)
	case "content", "text", "msg", "message":
		return p.parseContentNode(params)
	case "tool", "tools", "tool_calls", "function", "functions":
		return p.parseToolCallsNode(params)
	case "mixed", "combo":
		return p.parseMixedNode(params, reasoningEnabled, reasoningExcluded)
	case "image", "img":
		return p.parseImageNode(params)
	case "audio", "voice":
		return p.parseAudioNode(params)
	default:
		// If unrecognized, treat as content with the string as content
		return &ChainNode{
			Type:    NodeTypeContent,
			Content: nodeStr,
		}, nil
	}
}

// extractParams extracts base type and parameters from a node string
func (p *ChainParser) extractParams(nodeStr string) (string, map[string]string) {
	params := make(map[string]string)

	// Find opening brace
	braceIdx := strings.Index(nodeStr, "{")
	if braceIdx == -1 {
		return nodeStr, params
	}

	// Find closing brace (accounting for nesting)
	baseType := strings.TrimSpace(nodeStr[:braceIdx])
	content := nodeStr[braceIdx+1:]

	// Find matching closing brace
	depth := 1
	closeIdx := -1
	for i, ch := range content {
		if ch == '{' {
			depth++
		} else if ch == '}' {
			depth--
			if depth == 0 {
				closeIdx = i
				break
			}
		}
	}

	if closeIdx == -1 {
		return nodeStr, params
	}

	paramStr := content[:closeIdx]

	// Parse key=value pairs
	pairs := splitTopLevel(paramStr, ',')
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		eqIdx := strings.Index(pair, "=")
		if eqIdx == -1 {
			// Boolean flag
			params[pair] = "true"
		} else {
			key := strings.TrimSpace(pair[:eqIdx])
			value := strings.TrimSpace(pair[eqIdx+1:])
			params[key] = value
		}
	}

	return baseType, params
}

func (p *ChainParser) parseReasoningNode(params map[string]string) (*ChainNode, error) {
	node := &ChainNode{
		Type:      NodeTypeReasoning,
		Reasoning: getParam(params, "text", "Let me think about this step by step..."),
	}

	// Parse speed parameters
	if speed := p.parseSpeed(params); speed != nil {
		node.Speed = speed
	}

	// Parse fault parameters
	if fault := p.parseFault(params); fault != nil {
		node.Fault = fault
	}

	return node, nil
}

func (p *ChainParser) parseContentNode(params map[string]string) (*ChainNode, error) {
	node := &ChainNode{
		Type:    NodeTypeContent,
		Content: getParam(params, "text", "This is a simulated response."),
	}

	// Parse speed parameters
	if speed := p.parseSpeed(params); speed != nil {
		node.Speed = speed
	}

	// Parse fault parameters
	if fault := p.parseFault(params); fault != nil {
		node.Fault = fault
	}

	return node, nil
}

func (p *ChainParser) parseToolCallsNode(params map[string]string) (*ChainNode, error) {
	node := &ChainNode{
		Type:      NodeTypeToolCalls,
		ToolCalls: []SimulatedToolCall{},
	}

	for toolCount := 1; ; toolCount++ {
		suffix := strconv.Itoa(toolCount)
		name := getIndexedParam(params, "name", suffix)
		if name == "" {
			name = getIndexedParam(params, "tool", suffix)
		}
		if toolCount == 1 && name == "" {
			name = strings.TrimSpace(getParam(params, "name", ""))
			if name == "" {
				name = strings.TrimSpace(getParam(params, "tool", ""))
			}
		}
		if name == "" {
			if toolCount == 1 {
				return nil, fmt.Errorf("tool_calls requires at least one tool name")
			}
			break
		}

		argsStr := getIndexedParam(params, "args", suffix)
		if toolCount == 1 && argsStr == "" {
			argsStr = getParam(params, "args", "")
		}
		if argsStr == "" {
			return nil, fmt.Errorf("tool '%s' is missing args", name)
		}
		if !json.Valid([]byte(argsStr)) {
			return nil, fmt.Errorf("tool '%s' has invalid args JSON", name)
		}

		toolCall := SimulatedToolCall{
			ID:        getIndexedParam(params, "id", suffix),
			Name:      name,
			Arguments: json.RawMessage(argsStr),
		}
		if toolCount == 1 && toolCall.ID == "" {
			toolCall.ID = getParam(params, "id", "")
		}

		node.ToolCalls = append(node.ToolCalls, toolCall)
	}

	// Parse speed parameters
	if speed := p.parseSpeed(params); speed != nil {
		node.Speed = speed
	}

	// Parse fault parameters
	if fault := p.parseFault(params); fault != nil {
		node.Fault = fault
	}

	return node, nil
}

func (p *ChainParser) parseMixedNode(params map[string]string, reasoningEnabled bool, reasoningExcluded bool) (*ChainNode, error) {
	node := &ChainNode{
		Type:      NodeTypeMixed,
		Content:   getParam(params, "content", ""),
		Reasoning: getParam(params, "reasoning", ""),
	}

	// Only include reasoning if enabled and not excluded
	if reasoningEnabled && !reasoningExcluded && node.Reasoning == "" {
		node.Reasoning = "Analyzing the request..."
	} else if reasoningExcluded {
		node.Reasoning = "" // Clear reasoning if excluded
	}
	if node.Content == "" {
		node.Content = "Here's my response."
	}

	// Parse speed parameters
	if speed := p.parseSpeed(params); speed != nil {
		node.Speed = speed
	}

	// Parse fault parameters
	if fault := p.parseFault(params); fault != nil {
		node.Fault = fault
	}

	return node, nil
}

func (p *ChainParser) parseImageNode(params map[string]string) (*ChainNode, error) {
	node := &ChainNode{
		Type: NodeTypeContent,
		Multimodal: []MultimediaContent{
			{
				Type:     "image",
				URL:      getParam(params, "url", "https://mock.example.com/image.png"),
				MimeType: getParam(params, "mime", "image/png"),
			},
		},
	}

	// Also include text content if provided
	if text := getParam(params, "text", ""); text != "" {
		node.Content = text
	}

	return node, nil
}

func (p *ChainParser) parseAudioNode(params map[string]string) (*ChainNode, error) {
	node := &ChainNode{
		Type: NodeTypeContent,
		Multimodal: []MultimediaContent{
			{
				Type:     "audio",
				Data:     getParam(params, "data", "UklGRiQAAABXQVZFZm10IBAAAAABAAEARKwAAIhYAQACABAAZGF0YQAAAAA="),
				MimeType: getParam(params, "mime", "audio/wav"),
				Text:     getParam(params, "transcript", "Audio content"),
			},
		},
	}

	return node, nil
}

func (p *ChainParser) parseSpeed(params map[string]string) *TransmissionSpeed {
	speed := &TransmissionSpeed{
		CharDelay:  10 * time.Millisecond,
		ChunkSize:  10,
		ChunkDelay: 50 * time.Millisecond,
	}

	modified := false

	if v := getParam(params, "char_delay", ""); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			speed.CharDelay = d
			modified = true
		} else if ms, err := strconv.Atoi(v); err == nil {
			speed.CharDelay = time.Duration(ms) * time.Millisecond
			modified = true
		}
	}

	if v := getParam(params, "chunk_size", ""); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			speed.ChunkSize = n
			modified = true
		}
	}

	if v := getParam(params, "chunk_delay", ""); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			speed.ChunkDelay = d
			modified = true
		} else if ms, err := strconv.Atoi(v); err == nil {
			speed.ChunkDelay = time.Duration(ms) * time.Millisecond
			modified = true
		}
	}

	if modified {
		return speed
	}
	return nil
}

func (p *ChainParser) parseFault(params map[string]string) *FaultConfig {
	faultType := FaultType(getParam(params, "fault", ""))
	if faultType == "" || faultType == FaultTypeNone {
		return nil
	}

	fault := &FaultConfig{
		Type: faultType,
	}

	if v := getParam(params, "fault_prob", "0.5"); v != "" {
		if p, err := strconv.ParseFloat(v, 64); err == nil {
			fault.Probability = p
		}
	}

	if v := getParam(params, "fault_duration", ""); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			fault.Duration = d
		} else if ms, err := strconv.Atoi(v); err == nil {
			fault.Duration = time.Duration(ms) * time.Millisecond
		}
	}

	if v := getParam(params, "fault_after", ""); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			fault.AfterBytes = n
		}
	}

	if v := getParam(params, "fault_recovery", ""); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			fault.RecoveryAfter = d
		} else if ms, err := strconv.Atoi(v); err == nil {
			fault.RecoveryAfter = time.Duration(ms) * time.Millisecond
		}
	}

	if v := getParam(params, "fault_corruption", "0.5"); v != "" {
		if p, err := strconv.ParseFloat(v, 64); err == nil {
			fault.CorruptionLevel = p
		}
	}

	return fault
}

// Helper functions

func getParam(params map[string]string, key, defaultValue string) string {
	if v, ok := params[key]; ok {
		return v
	}
	return defaultValue
}

func getIndexedParam(params map[string]string, key, suffix string) string {
	return strings.TrimSpace(getParam(params, key+suffix, ""))
}

// splitTopLevel splits a string by a separator, but only at the top level
// (not inside braces or brackets)
func splitTopLevel(s string, sep rune) []string {
	var result []string
	var current strings.Builder
	depth := 0

	for _, ch := range s {
		switch ch {
		case '{', '[', '(':
			depth++
			current.WriteRune(ch)
		case '}', ']', ')':
			depth--
			current.WriteRune(ch)
		case sep:
			if depth == 0 {
				result = append(result, current.String())
				current.Reset()
			} else {
				current.WriteRune(ch)
			}
		default:
			current.WriteRune(ch)
		}
	}

	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result
}
