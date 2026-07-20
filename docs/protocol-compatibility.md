# 协议兼容矩阵

网关优先使用入站协议对应的上游原生端点。只有 Provider 类型或端点能力不一致时才执行转换。转换目标是保留明确支持的语义，而不是把未知字段静默删除。

启用的逻辑模型路由会先把客户端模型 ID 展开为按优先级排列的实际上游模型和 Provider。转换发生在候选确定之后，因此每次原生请求或协议转换都会使用该候选配置的实际上游模型名。Provider 开启模型白名单后，白名单会同时约束逻辑路由目标、未匹配逻辑路由的 `model_map` 结果和 `/v1/models` 输出；没有映射时保持客户端模型名。

| 入站协议 | 同协议上游 | OpenAI Chat / Responses 互转 | Anthropic / Gemini / OpenAI 跨协议 |
|---|---|---|---|
| OpenAI Chat Completions | 原样转发并替换模型名 | 文本、流式、工具调用与结果、多模态图片/文件、JSON Schema、reasoning、verbosity、usage；音频会明确拒绝 | 基础文本、system、temperature、top_p、stop；工具和多模态会明确拒绝 |
| OpenAI Responses | 原生 `/v1/responses` | 文本、流式、函数工具、多模态图片/文件、JSON Schema、reasoning、verbosity、usage；有状态字段不转换 | 先转换为 Chat 语义；超出基础文本能力的字段会明确拒绝 |
| Anthropic Messages | 原样转发并替换模型名 | 不适用 | 基础文本、system、max_tokens、temperature、top_p、stop_sequences；工具、thinking 与非文本内容会明确拒绝 |
| Gemini generateContent | 原样转发，模型放在 URL；自动规范化官方列表中的 `models/` 前缀 | 不适用 | 基础文本、systemInstruction、generationConfig 的常用采样参数；tools、safetySettings、cachedContent 与非文本 parts 会明确拒绝 |

## 流式边界

- 同协议上游的 SSE/流式响应优先透传。
- OpenAI Responses 与 Chat Completions 之间支持标准 SSE 生命周期和工具调用增量。
- 上游忽略 `stream: true` 返回完整 JSON 时，网关会转换为对应的 SSE 结果。
- Anthropic/Gemini 与 OpenAI 的跨协议流转换只保证基础文本事件；复杂工具事件应使用同协议 Provider。
- 已向客户端发送响应字节后发生中断，网关不会再切换 Key。
- 转换状态总量限制为 64 MiB，单个工具参数限制为 16 MiB；每次下游写入使用 `stream_write_timeout_seconds` 滚动超时。

## Responses 状态亲和性

网关会持久记录 `response.id`、`conversation.id` 与生成它们的 Provider/Key。包含 `previous_response_id`、`conversation`、文件、向量库或容器引用的请求不会跨 Key 重试，也不会回退到 Chat Completions。若资源没有已记录的亲和性且存在多个可选上游，网关返回 `409 affinity_unknown`，避免把资源 ID 猜测性地发送给错误的 Provider。

`background`、`previous_response_id`、`conversation`、`include`、`max_tool_calls`、内置工具和其他 Responses 独有能力只能使用原生 Responses 上游。Chat 的 `n > 1`、音频（包括消息内容中的 `input_audio`）、logprobs、prediction 等独有能力也不会回退到 Responses。

## 错误行为

发现不支持的跨协议字段时，请求在调用转换端点之前返回 `400 protocol_error`。上游返回无法解析或结构错误的 2xx JSON 时返回 `502 protocol_error`，该 Key 不会被标记为成功。需要完整语义时，应配置同协议 Provider 或让客户端使用上游原生协议。

上游明确返回模型不存在、未启用或无模型权限时归类为 `model_unavailable`，兼容常见结构化错误码、下划线/连字符和中英文消息，不会误判成端点不支持并触发 Chat/Responses 双端点重试。确定的“模型不存在/不支持”会冷却整个 Provider/模型组合；模型权限或未启用错误只冷却失败的 Key/模型组合。网关会按对应作用域跳过不可用候选，再尝试其他 Key、Provider 或下一备用模型。
