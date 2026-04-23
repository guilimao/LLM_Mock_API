# LLM Mock API Server

基于OpenRouter API规范的Go语言Mock服务器，支持流式传输、思考模式、工具调用和故障模拟。

## 功能特性

- ✅ **流式传输** - 支持SSE (Server-Sent Events) 流式响应
- ✅ **思考模式** - 模拟Reasoning Enabled/Disabled
- ✅ **工具调用** - 支持模拟工具调用测试
- ✅ **多模态输出** - 支持图片、音频输出
- ✅ **对话链条** - 通过system_prompt精细控制服务端输出
- ✅ **故障模拟** - 支持延迟、中断、丢包、JSON损坏等故障
- ✅ **并发执行** - 支持并行工具调用

## 快速开始

### 1. 安装依赖

```bash
cd LLM_Mock_API
go mod tidy
```

### 2. 启动服务器

```bash
go run .
# 或指定端口
go run . -port=8080 -debug=true
```

### 3. 测试API

```bash
# 健康检查
curl http://localhost:8080/health

# 简单对话
curl -X POST http://localhost:8080/api/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{"messages":[{"role":"user","content":"Hello"}]}'

# 流式响应
curl -X POST http://localhost:8080/api/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{"messages":[{"role":"user","content":"Hello"}],"stream":true}'
```

## API 端点

| 端点 | 方法 | 描述 |
|------|------|------|
| `/api/v1/chat/completions` | POST | 对话补全 |
| `/api/v1/models` | GET | 列出可用模型 |
| `/test-tools` | GET | 列出测试工具 |
| `/test-tools/{name}/invoke` | POST | 调用测试工具 |
| `/fault-presets` | GET | 列出故障预设 |
| `/health` | GET | 健康检查 |

## 对话链条语法

在system消息中使用`#CHAIN:`指令定义对话流程：

### 基础语法

```
#CHAIN: node1-node2-node3, node4, node5-node6
```

- `-` 分隔顺序执行的节点
- `,` 分隔可并发的段落
- `parallel:` 前缀表示并发节点

### 节点类型

| 类型 | 说明 | 示例 |
|------|------|------|
| `reasoning` | 思考内容 | `reasoning{text=分析中...}` |
| `content` | 普通文本 | `content{text=你好}` |
| `tool_calls` | 工具调用 | `tool_calls{name=func,args={}}` |
| `mixed` | 混合类型 | `mixed{reasoning=...,content=...}` |
| `image` | 图片 | `image{url=...}` |
| `audio` | 音频 | `audio{data=...}` |

### 参数

| 参数 | 说明 | 示例 |
|------|------|------|
| `text` | 文本内容 | `text=Hello` |
| `reasoning` | 思考文本 | `reasoning=Let me think` |
| `char_delay` | 字符延迟 | `char_delay=10ms` |
| `chunk_size` | 块大小 | `chunk_size=5` |
| `chunk_delay` | 块延迟 | `chunk_delay=50ms` |
| `fault` | 故障类型 | `fault=delay` |
| `fault_prob` | 故障概率 | `fault_prob=0.5` |

### 示例链条

```
#CHAIN: reasoning-content-tool_calls
#CHAIN: reasoning{text=Step 1}-content{text=Result 1}-reasoning{text=Step 2}-content{text=Result 2}
#CHAIN: tool_calls{name=get_weather,args={"loc":"NYC"}}, content{text=Done}
#CHAIN: reasoning, parallel:tool_calls, content
#CHAIN: content{fault=delay,fault_duration=2s}
```

## 思考模式 (Reasoning)

支持 OpenRouter 最佳实践的推理配置，兼容 OpenAI 和 Anthropic 两种风格：

### 配置选项

| 参数 | 类型 | 说明 | 示例 |
|------|------|------|------|
| `enabled` | boolean | 显式启用/禁用推理 | `true` |
| `effort` | string | OpenAI风格: xhigh/high/medium/low/minimal/none | `"high"` |
| `max_tokens` | integer | Anthropic风格: 推理token限制 | `2000` |
| `exclude` | boolean | 从响应中排除推理内容(仅统计) | `false` |

**注意**: `effort` 和 `max_tokens` 不能同时使用，只能二选一。

### 示例 1: OpenAI 风格 (effort)

```json
{
  "messages": [{"role": "user", "content": "Explain quantum computing"}],
  "reasoning": {
    "effort": "high"
  },
  "stream": true
}
```

### 示例 2: Anthropic 风格 (max_tokens)

```json
{
  "messages": [{"role": "user", "content": "Explain quantum computing"}],
  "reasoning": {
    "max_tokens": 2000
  },
  "stream": true
}
```

### 示例 3: 排除推理内容 (仅统计)

当 `exclude: true` 时，推理 tokens 仍会生成并计入 usage，但不会出现在响应内容中：

```json
{
  "messages": [{"role": "user", "content": "Complex problem"}],
  "reasoning": {
    "effort": "high",
    "exclude": true
  },
  "stream": false
}
```

响应示例：
```json
{
  "choices": [{
    "message": {
      "content": "The answer is 42.",
      "reasoning": ""  // 推理内容被排除
    }
  }],
  "usage": {
    "completion_tokens": 150,
    "completion_tokens_details": {
      "reasoning_tokens": 100  // 但统计中仍包含推理tokens
    }
  }
}
```

### 示例 4: 显式启用/禁用

```json
{
  "reasoning": {
    "enabled": true,
    "effort": "medium"
  }
}
```

```json
{
  "reasoning": {
    "enabled": false
  }
}
```

## 工具调用

测试工具规范支持客户端实现模拟工具调用：

```bash
# 列出工具
curl http://localhost:8080/test-tools

# 调用工具
curl -X POST http://localhost:8080/test-tools/get_weather/invoke \
  -H 'Content-Type: application/json' \
  -d '{"location": "Beijing", "unit": "celsius"}'
```

可用工具：
- `get_weather` - 获取天气
- `calculate` - 数学计算
- `search` - 搜索
- `generate_image` - 生成图片

## 故障模拟

### 故障类型

| 类型 | 说明 |
|------|------|
| `delay` | 延迟 |
| `interrupt` | 连接中断 |
| `packet_loss` | 丢包 |
| `corruption` | 数据损坏 |
| `partial_json` | JSON截断 |
| `malformed_json` | 畸形JSON |
| `timeout` | 超时 |

### 使用故障预设

```bash
curl http://localhost:8080/fault-presets
```

### 在对话中注入故障

```json
{
  "messages": [{
    "role": "system",
    "content": "#CHAIN: content{fault=delay,fault_duration=3s}"
  }, {
    "role": "user",
    "content": "Hello"
  }],
  "stream": true
}
```

## 完整示例

### 1. 基础流式对话

```bash
curl -X POST http://localhost:8080/api/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{
    "messages": [{"role": "user", "content": "Hello"}],
    "stream": true
  }'
```

### 2. 带思考的流式对话

```bash
curl -X POST http://localhost:8080/api/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{
    "messages": [{"role": "user", "content": "2+2=?"}],
    "reasoning": {"effort": "high"},
    "stream": true
  }'
```

### 3. 自定义对话链条

```bash
curl -X POST http://localhost:8080/api/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{
    "messages": [{
      "role": "system",
      "content": "#CHAIN: reasoning{text=Let me calculate}-content{text=The answer is 4}"
    }, {
      "role": "user",
      "content": "2+2=?"
    }],
    "stream": true
  }'
```

### 4. 带工具调用的对话

```bash
curl -X POST http://localhost:8080/api/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{
    "messages": [{
      "role": "system",
      "content": "#CHAIN: content-tool_calls-reasoning-content"
    }, {
      "role": "user",
      "content": "What's the weather in Beijing?"
    }],
    "stream": true,
    "tools": [{
      "type": "function",
      "function": {
        "name": "get_weather",
        "description": "Get weather"
      }
    }]
  }'
```

### 5. 并发工具调用

```bash
curl -X POST http://localhost:8080/api/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{
    "messages": [{
      "role": "system",
      "content": "#CHAIN: reasoning, parallel:tool_calls, content"
    }, {
      "role": "user",
      "content": "Get weather and calculate"
    }],
    "stream": true
  }'
```

### 6. 故障模拟

```bash
curl -X POST http://localhost:8080/api/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{
    "messages": [{
      "role": "system",
      "content": "#CHAIN: content{fault=interrupt,fault_prob=0.5}"
    }, {
      "role": "user",
      "content": "Hello"
    }],
    "stream": true
  }'
```

### 7. 慢速传输

```bash
curl -X POST http://localhost:8080/api/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{
    "messages": [{
      "role": "system",
      "content": "#CHAIN: content{char_delay=100ms,chunk_size=1,text=Slow response}"
    }, {
      "role": "user",
      "content": "Hello"
    }],
    "stream": true
  }'
```

### 8. JSON损坏测试

```bash
curl -X POST http://localhost:8080/api/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{
    "messages": [{
      "role": "system",
      "content": "#CHAIN: content{fault=malformed_json,fault_prob=1.0}"
    }, {
      "role": "user",
      "content": "Hello"
    }],
    "stream": true
  }'
```

### 9. OpenRouter 风格推理配置 (exclude)

```bash
# 推理内容被生成但不包含在响应中（仅用于统计）
curl -X POST http://localhost:8080/api/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{
    "messages": [{"role": "user", "content": "Explain quantum computing"}],
    "reasoning": {
      "effort": "high",
      "exclude": true
    },
    "stream": true
  }'
```

### 10. Anthropic 风格推理配置 (max_tokens)

```bash
# 使用 max_tokens 而不是 effort (Anthropic风格)
curl -X POST http://localhost:8080/api/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{
    "messages": [{"role": "user", "content": "Analyze this complex problem"}],
    "reasoning": {
      "max_tokens": 2000,
      "exclude": false
    },
    "stream": true
  }'
```

## 项目结构

```
LLM_Mock_API/
├── main.go        # 入口程序
├── models.go      # 数据模型
├── chain.go       # 对话链条解析
├── stream.go      # 流式处理
├── failure.go     # 故障模拟
├── handler.go     # HTTP处理器
├── go.mod         # Go模块
└── README.md      # 说明文档
```

## 命令行参数

```bash
go run . [options]

Options:
  -host string    Server host (default "localhost")
  -port string    Server port (default "8080")
  -model string   Default model name (default "mock/llm-model")
  -debug bool     Enable debug mode (default false)
```

## 许可证

MIT License
