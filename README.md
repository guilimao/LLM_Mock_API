# LLM Mock API Server

基于 OpenAI/OpenRouter 风格请求结构的 Go 语言 Mock 服务，支持流式输出、Reasoning、工具调用编排、多轮 tool round-trip 和故障注入。

## 功能特性

- 支持 SSE 流式输出
- 支持 reasoning enabled / excluded 模式
- 支持通过 `#CHAIN` / `#CHAIN_STEPn` 精确控制返回内容
- 支持由 Mock API 产出 `tool_calls`，由测试端执行真实工具
- 支持多轮 `assistant -> tool -> assistant` 回环测试
- 支持故障模拟

## 快速开始

### 1. 安装依赖

```bash
go mod tidy
```

### 2. 启动服务

```bash
go run .

# 或指定参数
go run . -port=8080 -debug=true
```

### 3. 基础请求

```bash
curl http://localhost:8080/health

curl -X POST http://localhost:8080/api/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{"messages":[{"role":"user","content":"Hello"}]}'
```

## API 端点

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/v1/chat/completions` | POST | 对话补全 |
| `/api/v1/models` | GET | 列出可用模型 |
| `/fault-presets` | GET | 列出故障预设 |
| `/health` | GET | 健康检查 |

说明：

## 对话链语法

通过 system message 中的链指令控制响应。

### 单阶段

```text
#CHAIN: reasoning-content-tool_calls
```

### 多阶段

```text
#CHAIN_STEP1: tool_calls{name=get_weather,args={"location":"Shanghai"}}
#CHAIN_STEP2: content{text=Tool results received}
```

### 规则

- `-` 表示同一阶段内顺序节点
- `,` 表示并发分段
- `parallel:` 表示并发节点
- `#CHAIN:` 等价于 `#CHAIN_STEP1:`
- 当消息历史中没有已完成的 tool round 时，命中 `STEP1`
- 已完成 1 轮 tool round 时，命中 `STEP2`
- 已完成 2 轮 tool round 时，命中 `STEP3`
- 若目标阶段不存在，回退到最后一个已定义阶段

### 节点类型

| 类型 | 说明 | 示例 |
|------|------|------|
| `reasoning` | reasoning 文本 | `reasoning{text=Let me think}` |
| `content` | 普通文本 | `content{text=Hello}` |
| `tool_calls` | 工具调用 | `tool_calls{name=get_weather,args={"location":"Shanghai"}}` |
| `mixed` | reasoning + content | `mixed{reasoning=...,content=...}` |
| `image` | 图片占位输出 | `image{url=https://example.com/a.png}` |
| `audio` | 音频占位输出 | `audio{data=...}` |

## 工具调用的新用法

当前推荐的工具测试流程：

1. 客户端在请求中声明真实 `tools`
2. system prompt 使用 `#CHAIN_STEP1` 指定需要返回的 `tool_calls`
3. Mock API 返回标准 OpenAI 风格 `tool_calls`
4. 客户端执行真实工具
5. 客户端把结果作为 `role=tool` 消息回传
6. Mock API 根据 `#CHAIN_STEP2` 继续返回后续内容

### `tool_calls` 参数规则

首个工具调用：

```text
tool_calls{name=get_weather,args={"location":"Shanghai"}}
```

多个工具调用：

```text
tool_calls{
  id=call_weather_1,
  name=get_weather,
  args={"location":"Shanghai"},
  id2=call_search_1,
  name2=search,
  args2={"query":"weather shanghai"}
}
```

规则说明：

- `name` 和 `args` 必填
- `id` 可选；不填时服务端自动生成
- 第二个及之后的工具调用使用 `name2` / `args2` / `id2`、`name3` / `args3` / `id3`
- `args` 必须是合法 JSON
- 服务端不会根据 `tools` 自动推断参数，只按 `CHAIN` 显式定义返回
- 即使 `CHAIN` 中的工具名与请求 `tools` 不一致，也不会拦截；若开启 trace，会通过响应头暴露 mismatch

### 多轮 round-trip 示例

第一轮：请求模型产出工具调用

```bash
curl -X POST http://localhost:8080/api/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{
    "messages": [
      {
        "role": "system",
        "content": "#CHAIN_STEP1: tool_calls{name=get_weather,args={\"location\":\"Shanghai\"}}\n#CHAIN_STEP2: content{text=Tool results received.}"
      },
      {
        "role": "user",
        "content": "What is the weather?"
      }
    ],
    "stream": true,
    "tools": [
      {
        "type": "function",
        "function": {
          "name": "get_weather",
          "description": "Get weather information"
        }
      }
    ]
  }'
```

第二轮：客户端执行真实工具后把结果传回

```bash
curl -X POST http://localhost:8080/api/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{
    "messages": [
      {
        "role": "system",
        "content": "#CHAIN_STEP1: tool_calls{name=get_weather,args={\"location\":\"Shanghai\"}}\n#CHAIN_STEP2: content{text=Tool results received.}"
      },
      {
        "role": "user",
        "content": "What is the weather?"
      },
      {
        "role": "assistant",
        "tool_calls": [
          {
            "id": "call_1",
            "type": "function",
            "function": {
              "name": "get_weather",
              "arguments": "{\"location\":\"Shanghai\"}"
            }
          }
        ]
      },
      {
        "role": "tool",
        "tool_call_id": "call_1",
        "content": "{\"temperature\":26,\"condition\":\"sunny\"}"
      }
    ]
  }'
```

## Debug 与可复现能力

`ChatRequest.debug` 当前支持：

```json
{
  "debug": {
    "deterministic": true,
    "chain_trace": true
  },
  "seed": 7
}
```

### `deterministic`

开启后，若请求带 `seed`，以下内容会稳定：

- response id
- 自动生成的 tool call id
- created 时间戳

适合做快照测试、回归测试。

### `chain_trace`

开启后，服务端会通过响应头返回链路选择信息：

- `X-Mock-Chain-Step`
- `X-Mock-Chain-Fallback`
- `X-Mock-Tool-Mismatch`

说明：

- 不会污染标准响应 JSON
- 适合调试当前命中了哪个阶段、是否发生阶段回退、是否存在工具声明不一致

## Reasoning

支持两种风格：

- OpenAI 风格：`reasoning.effort`
- Anthropic 风格：`reasoning.max_tokens`

### 示例 1：effort

```json
{
  "messages": [{"role": "user", "content": "Explain quantum computing"}],
  "reasoning": {
    "effort": "high"
  },
  "stream": true
}
```

### 示例 2：max_tokens

```json
{
  "messages": [{"role": "user", "content": "Explain quantum computing"}],
  "reasoning": {
    "max_tokens": 2000
  },
  "stream": true
}
```

### 示例 3：exclude

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

`exclude=true` 时：

- reasoning token 仍计入 usage
- reasoning 内容不出现在响应 message 中

## 故障注入

### 支持的故障类型

| 类型 | 说明 |
|------|------|
| `delay` | 延迟 |
| `interrupt` | 中断 |
| `packet_loss` | 丢包 |
| `corruption` | 数据损坏 |
| `partial_json` | JSON 截断 |
| `malformed_json` | 畸形 JSON |
| `timeout` | 超时 |

### 查询预设

```bash
curl http://localhost:8080/fault-presets
```

### 在链中注入故障

```json
{
  "messages": [
    {
      "role": "system",
      "content": "#CHAIN: content{fault=delay,fault_duration=3s,text=Delayed response}"
    },
    {
      "role": "user",
      "content": "Hello"
    }
  ],
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

### 2. 带 reasoning 的流式对话

```bash
curl -X POST http://localhost:8080/api/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{
    "messages": [{"role": "user", "content": "2+2=?"}],
    "reasoning": {"effort": "high"},
    "stream": true
  }'
```

### 3. 自定义单阶段 CHAIN

```bash
curl -X POST http://localhost:8080/api/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{
    "messages": [
      {
        "role": "system",
        "content": "#CHAIN: reasoning{text=Let me calculate}-content{text=The answer is 4}"
      },
      {
        "role": "user",
        "content": "2+2=?"
      }
    ],
    "stream": true
  }'
```

### 4. staged tool round-trip

```bash
curl -X POST http://localhost:8080/api/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{
    "messages": [
      {
        "role": "system",
        "content": "#CHAIN_STEP1: tool_calls{name=get_weather,args={\"location\":\"Shanghai\"}}\n#CHAIN_STEP2: content{text=The tool result has been consumed.}"
      },
      {
        "role": "user",
        "content": "What is the weather?"
      }
    ],
    "stream": true,
    "debug": {
      "deterministic": true,
      "chain_trace": true
    },
    "seed": 7,
    "tools": [
      {
        "type": "function",
        "function": {
          "name": "get_weather",
          "description": "Get weather"
        }
      }
    ]
  }'
```

### 5. 并行工具调用

```bash
curl -X POST http://localhost:8080/api/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{
    "messages": [
      {
        "role": "system",
        "content": "#CHAIN: tool_calls{name=get_weather,args={\"location\":\"Shanghai\"},name2=search,args2={\"query\":\"weather shanghai\"}}"
      },
      {
        "role": "user",
        "content": "Use tools"
      }
    ],
    "parallel_tool_calls": true,
    "stream": true
  }'
```

### 6. 故障模拟

```bash
curl -X POST http://localhost:8080/api/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{
    "messages": [
      {
        "role": "system",
        "content": "#CHAIN: content{fault=interrupt,fault_prob=0.5}"
      },
      {
        "role": "user",
        "content": "Test interruption"
      }
    ],
    "stream": true
  }'
```

## 测试

```bash
go test ./...
```

## 命令行参数

```text
-port string     Server port (default "8080")
-host string     Server host (default "localhost")
-model string    Default model name (default "mock/llm-model")
-debug           Enable debug mode
-log-dir string  Directory for request logs (default "./logs")
-no-log          Disable request logging
```
