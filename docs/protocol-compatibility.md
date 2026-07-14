# 协议兼容矩阵

网关优先使用入站协议对应的上游原生端点。只有 Provider 类型或端点能力不一致时才执行转换。转换目标是保留明确支持的语义，而不是把未知字段静默删除。

| 入站协议 | 同协议上游 | OpenAI Chat / Responses 互转 | Anthropic / Gemini / OpenAI 跨协议 |
|---|---|---|---|
| OpenAI Chat Completions | 原样转发并替换模型名 | 文本、流式、工具调用与结果、多模态内容、JSON Schema、reasoning、usage | 基础文本、system、temperature、top_p、stop；工具和多模态会明确拒绝 |
| OpenAI Responses | 原生 `/v1/responses` | 文本、流式、函数工具、多模态图片/文件、JSON Schema、reasoning、usage | 先转换为 Chat 语义；超出基础文本能力的字段会明确拒绝 |
| Anthropic Messages | 原样转发并替换模型名 | 不适用 | 基础文本、system、max_tokens、temperature、top_p、stop_sequences；工具、thinking 与非文本内容会明确拒绝 |
| Gemini generateContent | 原样转发，模型放在 URL | 不适用 | 基础文本、systemInstruction、generationConfig 的常用采样参数；tools、safetySettings、cachedContent 与非文本 parts 会明确拒绝 |

## 流式边界

- 同协议上游的 SSE/流式响应优先透传。
- OpenAI Responses 与 Chat Completions 之间支持标准 SSE 生命周期和工具调用增量。
- 上游忽略 `stream: true` 返回完整 JSON 时，网关会转换为对应的 SSE 结果。
- Anthropic/Gemini 与 OpenAI 的跨协议流转换只保证基础文本事件；复杂工具事件应使用同协议 Provider。
- 已向客户端发送响应字节后发生中断，网关不会再切换 Key。

## 错误行为

发现不支持的跨协议字段时，请求在调用上游之前返回 `400`。这项行为用于防止工具调用、安全配置或多模态内容被静默忽略。需要完整语义时，应配置同协议 Provider 或让客户端使用上游原生协议。
