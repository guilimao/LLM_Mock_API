package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestLogger 请求日志记录器
type RequestLogger struct {
	logDir     string
	mutex      sync.RWMutex
	enableFile bool // 是否启用文件日志
}

// LogEntry 单个日志条目
type LogEntry struct {
	RequestID   string      `json:"request_id"`
	Timestamp   string      `json:"timestamp"`
	Method      string      `json:"method"`
	Path        string      `json:"path"`
	ClientIP    string      `json:"client_ip"`
	UserAgent   string      `json:"user_agent"`
	Request     interface{} `json:"request,omitempty"`
	Response    interface{} `json:"response,omitempty"`
	StatusCode  int         `json:"status_code"`
	Duration    string      `json:"duration"`
	Error       string      `json:"error,omitempty"`
}

// NewRequestLogger 创建新的请求日志记录器
func NewRequestLogger(logDir string) *RequestLogger {
	logger := &RequestLogger{
		logDir:     logDir,
		enableFile: true,
	}

	// 创建日志目录
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Printf("Warning: Failed to create log directory: %v\n", err)
		logger.enableFile = false
	}

	return logger
}

// LogMiddleware Gin 中间件 - 记录请求和响应
func (rl *RequestLogger) LogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 生成请求ID
		requestID := uuid.New().String()[:8]
		c.Set("request_id", requestID)

		// 记录开始时间
		startTime := time.Now()

		// 读取请求体
		var requestBody interface{}
		if c.Request.Body != nil && c.Request.ContentLength > 0 {
			bodyBytes, _ := io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			// 尝试解析JSON
			var jsonBody map[string]interface{}
			if err := json.Unmarshal(bodyBytes, &jsonBody); err == nil {
				requestBody = jsonBody
			} else {
				requestBody = string(bodyBytes)
			}
		}

		// 创建响应写入器来捕获响应
		blw := &bodyLogWriter{
			body:           bytes.NewBufferString(""),
			ResponseWriter: c.Writer,
		}
		c.Writer = blw

		// 打印请求信息到终端
		rl.printRequestInfo(requestID, c, requestBody)

		// 继续处理请求
		c.Next()

		// 计算处理时间
		duration := time.Since(startTime)

		// 解析响应体
		var responseBody interface{}
		if blw.body.Len() > 0 {
			var jsonResponse map[string]interface{}
			if err := json.Unmarshal(blw.body.Bytes(), &jsonResponse); err == nil {
				responseBody = jsonResponse
			} else {
				responseBody = blw.body.String()
			}
		}

		// 创建日志条目
		entry := LogEntry{
			RequestID:  requestID,
			Timestamp:  startTime.Format("2006-01-02 15:04:05.000"),
			Method:     c.Request.Method,
			Path:       c.Request.URL.Path,
			ClientIP:   c.ClientIP(),
			UserAgent:  c.Request.UserAgent(),
			Request:    requestBody,
			Response:   responseBody,
			StatusCode: c.Writer.Status(),
			Duration:   duration.String(),
		}

		// 如果有错误，记录错误信息
		if len(c.Errors) > 0 {
			entry.Error = c.Errors.String()
		}

		// 打印响应信息到终端
		rl.printResponseInfo(requestID, entry.StatusCode, duration, blw.body.Len())

		// 写入日志文件
		rl.writeToFile(entry)
	}
}

// printRequestInfo 打印请求信息到终端
func (rl *RequestLogger) printRequestInfo(requestID string, c *gin.Context, body interface{}) {
	fmt.Println()
	fmt.Printf("┌────────────────────────────────────────────────────────────\n")
	fmt.Printf("│ 📥 [%s] %s %s\n", requestID, c.Request.Method, c.Request.URL.Path)
	fmt.Printf("├────────────────────────────────────────────────────────────\n")
	fmt.Printf("│ Client: %s\n", c.ClientIP())
	fmt.Printf("│ Time:   %s\n", time.Now().Format("2006-01-02 15:04:05"))

	// 打印请求头（关键信息）
	contentType := c.GetHeader("Content-Type")
	authorization := c.GetHeader("Authorization")
	if authorization != "" {
		// 截断Authorization显示
		if len(authorization) > 30 {
			authorization = authorization[:30] + "..."
		}
	}

	if contentType != "" {
		fmt.Printf("│ Content-Type: %s\n", contentType)
	}
	if authorization != "" {
		fmt.Printf("│ Authorization: %s\n", authorization)
	}

	// 打印请求体
	if body != nil {
		fmt.Printf("├────────────────────────────────────────────────────────────\n")
		fmt.Printf("│ 📄 Request Body:\n")
		rl.printJSON(body, "│   ")
	}

	fmt.Printf("└────────────────────────────────────────────────────────────\n")
}

// printResponseInfo 打印响应信息到终端
func (rl *RequestLogger) printResponseInfo(requestID string, statusCode int, duration time.Duration, bodySize int) {
	statusIcon := "✅"
	if statusCode >= 400 {
		statusIcon = "❌"
	}

	fmt.Printf("┌────────────────────────────────────────────────────────────\n")
	fmt.Printf("│ 📤 [%s] Response %s %d\n", requestID, statusIcon, statusCode)
	fmt.Printf("├────────────────────────────────────────────────────────────\n")
	fmt.Printf("│ Duration: %v\n", duration)
	fmt.Printf("│ Body Size: %d bytes\n", bodySize)
	fmt.Printf("└────────────────────────────────────────────────────────────\n")
}

// printJSON 格式化打印JSON
func (rl *RequestLogger) printJSON(data interface{}, prefix string) {
	jsonBytes, err := json.MarshalIndent(data, prefix, "  ")
	if err != nil {
		fmt.Printf("%s%v\n", prefix, data)
		return
	}
	fmt.Println(string(jsonBytes))
}

// writeToFile 写入日志到文件（每个请求单独文件，按URL分文件夹）
func (rl *RequestLogger) writeToFile(entry LogEntry) {
	if !rl.enableFile {
		return
	}

	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	// 将URL路径转换为文件夹名称（移除前导斜杠，替换其他斜杠为下划线）
	urlPath := entry.Path
	if urlPath == "" {
		urlPath = "root"
	} else {
		// 移除前导斜杠
		urlPath = strings.TrimPrefix(urlPath, "/")
		// 将剩余斜杠替换为下划线
		urlPath = strings.ReplaceAll(urlPath, "/", "_")
		// 处理空路径的情况
		if urlPath == "" {
			urlPath = "root"
		}
	}

	// 构建目录结构: logs/{url路径}/{日期}/
	dateStr := time.Now().Format("2006-01-02")
	targetDir := filepath.Join(rl.logDir, urlPath, dateStr)

	// 创建目录
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		fmt.Printf("Error creating log directory: %v\n", err)
		return
	}

	// 构建文件名: {时间}_{请求ID}.json
	timeStr := time.Now().Format("15-04-05")
	logFile := filepath.Join(targetDir, fmt.Sprintf("%s_%s.json", timeStr, entry.RequestID))

	// 打开文件（创建新文件）
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Printf("Error creating log file: %v\n", err)
		return
	}
	defer f.Close()

	// 写入JSON格式的日志
	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(entry); err != nil {
		fmt.Printf("Error writing log: %v\n", err)
	}
}

// GetLogFiles 获取所有日志文件列表（递归查找所有JSON文件）
func (rl *RequestLogger) GetLogFiles() ([]string, error) {
	var files []string
	
	err := filepath.Walk(rl.logDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// 只收集JSON文件，并返回相对于logDir的路径
		if !info.IsDir() && filepath.Ext(path) == ".json" {
			relPath, err := filepath.Rel(rl.logDir, path)
			if err != nil {
				return err
			}
			files = append(files, relPath)
		}
		return nil
	})
	
	return files, err
}

// bodyLogWriter 包装 ResponseWriter 以捕获响应体
type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *bodyLogWriter) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}
