package main

import "testing"

func TestParseToolCallsNode(t *testing.T) {
	parser := NewChainParser()

	chain, err := parser.Parse(`tool_calls{id=call_weather_1,name=get_weather,args={"location":"Shanghai"},id2=call_search_1,name2=search,args2={"query":"weather"}}`, false, false)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if len(chain.Segments) != 1 || len(chain.Segments[0]) != 1 {
		t.Fatalf("unexpected chain shape: %#v", chain.Segments)
	}

	toolCalls := chain.Segments[0][0].ToolCalls
	if len(toolCalls) != 2 {
		t.Fatalf("expected 2 tool calls, got %d", len(toolCalls))
	}
	if toolCalls[0].ID != "call_weather_1" || toolCalls[0].Name != "get_weather" {
		t.Fatalf("unexpected first tool call: %#v", toolCalls[0])
	}
	if string(toolCalls[1].Arguments) != `{"query":"weather"}` {
		t.Fatalf("unexpected second tool args: %s", string(toolCalls[1].Arguments))
	}
}

func TestParseToolCallsNodeRequiresNameAndValidJSON(t *testing.T) {
	parser := NewChainParser()

	if _, err := parser.Parse(`tool_calls{args={"location":"Shanghai"}}`, false, false); err == nil {
		t.Fatal("expected missing tool name to fail")
	}

	if _, err := parser.Parse(`tool_calls{name=get_weather,args={location:"Shanghai"}}`, false, false); err == nil {
		t.Fatal("expected invalid JSON args to fail")
	}
}
