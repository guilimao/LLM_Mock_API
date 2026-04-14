package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	// Parse command line flags
	var (
		port       = flag.String("port", "8080", "Server port")
		host       = flag.String("host", "localhost", "Server host")
		model      = flag.String("model", "mock/llm-model", "Default model name")
		debug      = flag.Bool("debug", false, "Enable debug mode")
		logDir     = flag.String("log-dir", "./logs", "Directory for request logs")
		disableLog = flag.Bool("no-log", false, "Disable request logging")
	)
	flag.Parse()
	
	// Set Gin mode
	if !*debug {
		gin.SetMode(gin.ReleaseMode)
	}
	
	// Create router
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(CORSMiddleware())
	
	// 添加请求日志中间件
	if !*disableLog {
		requestLogger := NewRequestLogger(*logDir)
		router.Use(requestLogger.LogMiddleware())
	}
	
	if *debug {
		router.Use(gin.Logger())
	}
	
	// Create handler
	handler := NewHandler(*model)
	
	// Register routes
	handler.RegisterRoutes(router)
	
	// Print startup message
	fmt.Printf("╔═══════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║                 LLM Mock API Server                           ║\n")
	fmt.Printf("╠═══════════════════════════════════════════════════════════════╣\n")
	fmt.Printf("║  Server:   http://%s:%s                                      ║\n", *host, *port)
	fmt.Printf("║  API:      http://%s:%s/api/v1/chat/completions              ║\n", *host, *port)
	fmt.Printf("║  Health:   http://%s:%s/health                               ║\n", *host, *port)
	fmt.Printf("║  Tools:    http://%s:%s/test-tools                           ║\n", *host, *port)
	fmt.Printf("╠═══════════════════════════════════════════════════════════════╣\n")
	fmt.Printf("║  Features:                                                    ║\n")
	fmt.Printf("║    ✓ Streaming responses (SSE)                               ║\n")
	fmt.Printf("║    ✓ Reasoning mode (enable with reasoning.effort)           ║\n")
	fmt.Printf("║    ✓ Tool calls simulation                                   ║\n")
	fmt.Printf("║    ✓ Multimodal output                                       ║\n")
	fmt.Printf("║    ✓ Fault simulation                                        ║\n")
	fmt.Printf("║    ✓ Dialogue chain control                                  ║\n")
	fmt.Printf("║    ✓ Request/Response logging                                ║\n")
	fmt.Printf("╠═══════════════════════════════════════════════════════════════╣\n")
	if !*disableLog {
		fmt.Printf("║  Log Dir:  %s\n", *logDir)
		fmt.Printf("╚═══════════════════════════════════════════════════════════════╝\n")
	} else {
		fmt.Printf("║  Logging:  Disabled (--no-log)                               ║\n")
		fmt.Printf("╚═══════════════════════════════════════════════════════════════╝\n")
	}
	fmt.Println()
	
	// Print example usage
	printUsageExamples()
	
	// Start server
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
	fmt.Println("     -d '{" +
		"\"messages\":[{\"role\":\"user\",\"content\":\"Hello\"}]," +
		"\"model\":\"mock/llm-model\"}'")
	fmt.Println()
	
	fmt.Println("2. Streaming response:")
	fmt.Println("   curl -X POST http://localhost:8080/api/v1/chat/completions \\")
	fmt.Println("     -H 'Content-Type: application/json' \\")
	fmt.Println("     -d '{" +
		"\"messages\":[{\"role\":\"user\",\"content\":\"Hello\"}]," +
		"\"stream\":true}'")
	fmt.Println()
	
	fmt.Println("3. With reasoning enabled:")
	fmt.Println("   curl -X POST http://localhost:8080/api/v1/chat/completions \\")
	fmt.Println("     -H 'Content-Type: application/json' \\")
	fmt.Println("     -d '{" +
		"\"messages\":[{\"role\":\"user\",\"content\":\"Solve 2+2\"}]," +
		"\"reasoning\":{\"effort\":\"medium\"}," +
		"\"stream\":true}'")
	fmt.Println()
	
	fmt.Println("4. With dialogue chain (system prompt):")
	fmt.Println("   curl -X POST http://localhost:8080/api/v1/chat/completions \\")
	fmt.Println("     -H 'Content-Type: application/json' \\")
	fmt.Println("     -d '{")
	fmt.Println("       \"messages\":[")
	fmt.Println("         {\"role\":\"system\",\"content\":\"#CHAIN: reasoning-content-tool_calls\"},")
	fmt.Println("         {\"role\":\"user\",\"content\":\"What's the weather?\"}")
	fmt.Println("       ],")
	fmt.Println("       \"stream\":true,")
	fmt.Println("       \"tools\":[{")
	fmt.Println("         \"type\":\"function\",")
	fmt.Println("         \"function\":{")
	fmt.Println("           \"name\":\"get_weather\",")
	fmt.Println("           \"description\":\"Get weather information\"")
	fmt.Println("         }")
	fmt.Println("       }]")
	fmt.Println("     }'")
	fmt.Println()
	
	fmt.Println("5. With fault simulation:")
	fmt.Println("   curl -X POST http://localhost:8080/api/v1/chat/completions \\")
	fmt.Println("     -H 'Content-Type: application/json' \\")
	fmt.Println("     -d '{")
	fmt.Println("       \"messages\":[{")
	fmt.Println("         \"role\":\"system\",")
	fmt.Println("         \"content\":\"#CHAIN: content{fault=delay,fault_duration=2s,text=Delayed response}\"")
	fmt.Println("       },{")
	fmt.Println("         \"role\":\"user\",\"content\":\"Hello\"}")
	fmt.Println("       ],")
	fmt.Println("       \"stream\":true")
	fmt.Println("     }'")
	fmt.Println()
	
	fmt.Println("6. List available test tools:")
	fmt.Println("   curl http://localhost:8080/test-tools")
	fmt.Println()
	
	fmt.Println("7. Invoke a test tool:")
	fmt.Println("   curl -X POST http://localhost:8080/test-tools/get_weather/invoke \\")
	fmt.Println("     -H 'Content-Type: application/json' \\")
	fmt.Println("     -d '{\"location\":\"Beijing\",\"unit\":\"celsius\"}'")
	fmt.Println()
	
	fmt.Println("8. List fault presets:")
	fmt.Println("   curl http://localhost:8080/fault-presets")
	fmt.Println()
	
	fmt.Println("Dialogue Chain Syntax:")
	fmt.Println("  - Nodes are separated by '-' for sequential execution")
	fmt.Println("  - Segments are separated by ',' for concurrent groups")
	fmt.Println("  - Use 'parallel:' prefix for concurrent nodes in a segment")
	fmt.Println()
	fmt.Println("  Node Types:")
	fmt.Println("    reasoning    - Thinking/reasoning content")
	fmt.Println("    content      - Regular text content")
	fmt.Println("    tool_calls   - Tool/function calls")
	fmt.Println("    mixed        - Combination of reasoning + content")
	fmt.Println("    image        - Image output")
	fmt.Println("    audio        - Audio output")
	fmt.Println()
	fmt.Println("  Parameters:")
	fmt.Println("    text=<text>               - Content text")
	fmt.Println("    reasoning=<text>          - Reasoning text")
	fmt.Println("    char_delay=<duration>     - Delay between characters")
	fmt.Println("    chunk_size=<n>            - Characters per chunk")
	fmt.Println("    chunk_delay=<duration>    - Delay between chunks")
	fmt.Println("    fault=<type>              - Fault type to simulate")
	fmt.Println("    fault_prob=<0.0-1.0>      - Fault probability")
	fmt.Println("    fault_duration=<duration> - Fault duration")
	fmt.Println()
	fmt.Println("  Examples:")
	fmt.Println("    reasoning-content-tool_calls")
	fmt.Println("    reasoning{text=Let me think...}-content{text=Here's the answer}")
	fmt.Println("    tool_calls{name=get_weather,args={\"loc\":\"NYC\"}}")
	fmt.Println("    content{fault=delay,fault_duration=1s,text=Slow response}")
	fmt.Println("    reasoning, parallel:tool_calls, content")
	fmt.Println()
}
