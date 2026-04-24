package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	var (
		port       = flag.String("port", "8080", "Server port")
		host       = flag.String("host", "localhost", "Server host")
		model      = flag.String("model", "mock/llm-model", "Default model name")
		debug      = flag.Bool("debug", false, "Enable debug mode")
		logDir     = flag.String("log-dir", "./logs", "Directory for request logs")
		disableLog = flag.Bool("no-log", false, "Disable request logging")
	)
	flag.Parse()

	if !*debug {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(CORSMiddleware())

	if !*disableLog {
		requestLogger := NewRequestLogger(*logDir)
		router.Use(requestLogger.LogMiddleware())
	}
	if *debug {
		router.Use(gin.Logger())
	}

	handler := NewHandler(*model)
	handler.RegisterRoutes(router)

	fmt.Println("LLM Mock API Server")
	fmt.Printf("Server: http://%s:%s\n", *host, *port)
	fmt.Printf("API:    http://%s:%s/api/v1/chat/completions\n", *host, *port)
	fmt.Printf("Health: http://%s:%s/health\n", *host, *port)
	if !*disableLog {
		fmt.Printf("Logs:   %s\n", *logDir)
	} else {
		fmt.Println("Logs:   disabled")
	}
	fmt.Println()

	printUsageExamples()

	address := fmt.Sprintf("%s:%s", *host, *port)
	if err := router.Run(address); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start server: %v\n", err)
		os.Exit(1)
	}
}

func printUsageExamples() {
	fmt.Println("Usage Examples:")
	fmt.Println()

	fmt.Println("1. Simple chat completion:")
	fmt.Println("   curl -X POST http://localhost:8080/api/v1/chat/completions \\")
	fmt.Println("     -H 'Content-Type: application/json' \\")
	fmt.Println("     -d '{\"messages\":[{\"role\":\"user\",\"content\":\"Hello\"}],\"model\":\"mock/llm-model\"}'")
	fmt.Println()

	fmt.Println("2. Streaming response:")
	fmt.Println("   curl -X POST http://localhost:8080/api/v1/chat/completions \\")
	fmt.Println("     -H 'Content-Type: application/json' \\")
	fmt.Println("     -d '{\"messages\":[{\"role\":\"user\",\"content\":\"Hello\"}],\"stream\":true}'")
	fmt.Println()

	fmt.Println("3. Staged tool round-trip:")
	fmt.Println("   curl -X POST http://localhost:8080/api/v1/chat/completions \\")
	fmt.Println("     -H 'Content-Type: application/json' \\")
	fmt.Println("     -d '{")
	fmt.Println("       \"messages\":[")
	fmt.Println("         {\"role\":\"system\",\"content\":\"#CHAIN_STEP1: tool_calls{name=get_weather,args={\\\"location\\\":\\\"Shanghai\\\"}}\\n#CHAIN_STEP2: content{text=Tool results received.}\"},")
	fmt.Println("         {\"role\":\"user\",\"content\":\"What is the weather?\"}")
	fmt.Println("       ],")
	fmt.Println("       \"stream\":true,")
	fmt.Println("       \"debug\":{\"deterministic\":true,\"chain_trace\":true},")
	fmt.Println("       \"seed\":7,")
	fmt.Println("       \"tools\":[{\"type\":\"function\",\"function\":{\"name\":\"get_weather\",\"description\":\"Get weather information\"}}]")
	fmt.Println("     }'")
	fmt.Println()

	fmt.Println("4. Continue after executing the real tool on the client:")
	fmt.Println("   curl -X POST http://localhost:8080/api/v1/chat/completions \\")
	fmt.Println("     -H 'Content-Type: application/json' \\")
	fmt.Println("     -d '{")
	fmt.Println("       \"messages\":[")
	fmt.Println("         {\"role\":\"system\",\"content\":\"#CHAIN_STEP1: tool_calls{name=get_weather,args={\\\"location\\\":\\\"Shanghai\\\"}}\\n#CHAIN_STEP2: content{text=The tool result has been consumed.}\"},")
	fmt.Println("         {\"role\":\"user\",\"content\":\"What is the weather?\"},")
	fmt.Println("         {\"role\":\"assistant\",\"tool_calls\":[{\"id\":\"call_1\",\"type\":\"function\",\"function\":{\"name\":\"get_weather\",\"arguments\":\"{\\\"location\\\":\\\"Shanghai\\\"}\"}}]},")
	fmt.Println("         {\"role\":\"tool\",\"tool_call_id\":\"call_1\",\"content\":\"{\\\"temperature\\\":26}\"}")
	fmt.Println("       ]")
	fmt.Println("     }'")
	fmt.Println()

	fmt.Println("5. Fault simulation:")
	fmt.Println("   curl -X POST http://localhost:8080/api/v1/chat/completions \\")
	fmt.Println("     -H 'Content-Type: application/json' \\")
	fmt.Println("     -d '{\"messages\":[{\"role\":\"system\",\"content\":\"#CHAIN: content{fault=delay,fault_duration=2s,text=Delayed response}\"},{\"role\":\"user\",\"content\":\"Hello\"}],\"stream\":true}'")
	fmt.Println()

	fmt.Println("6. List fault presets:")
	fmt.Println("   curl http://localhost:8080/fault-presets")
	fmt.Println()

	fmt.Println("Dialogue Chain Syntax:")
	fmt.Println("  - Nodes are separated by '-' for sequential execution")
	fmt.Println("  - Segments are separated by ',' for concurrent groups")
	fmt.Println("  - Use 'parallel:' for concurrent nodes in a segment")
	fmt.Println("  - Use '#CHAIN:' for single-stage responses")
	fmt.Println("  - Use '#CHAIN_STEPn:' for multi-round tool flows")
	fmt.Println()
	fmt.Println("Tool Call Parameters:")
	fmt.Println("  - name / args / id for the first tool call")
	fmt.Println("  - name2 / args2 / id2 for additional tool calls")
	fmt.Println()
}
