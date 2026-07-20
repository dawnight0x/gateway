package protocol

import (
	"encoding/json"
	"strings"
	"testing"

	"local-ai-gateway/internal/router"
)

func TestDetectGeminiPath(t *testing.T) {
	if got := DetectInbound("/v1beta/models/gemini-2.5-pro:generateContent"); got != router.ProtocolGemini {
		t.Fatalf("protocol = %s", got)
	}
	if got := DetectInbound("/v1/models/gemini-2.5-pro:streamGenerateContent"); got != router.ProtocolGemini {
		t.Fatalf("stream protocol = %s", got)
	}
	if got := ExtractPathModel("/v1beta/models/gemini-2.5-pro:streamGenerateContent"); got != "gemini-2.5-pro" {
		t.Fatalf("model = %s", got)
	}
}

func FuzzProtocolConversionDoesNotPanic(f *testing.F) {
	f.Add([]byte(`{"model":"gpt","messages":[{"role":"user","content":"hello"}]}`), uint8(0), uint8(1))
	f.Add([]byte(`{"contents":[{"parts":[{"text":"hello"}]}]}`), uint8(2), uint8(0))
	protocols := []string{router.ProtocolOpenAI, router.ProtocolOpenAIResponses, router.ProtocolAnthropic, router.ProtocolGemini}
	f.Fuzz(func(t *testing.T, body []byte, inboundIndex, upstreamIndex uint8) {
		inbound := protocols[int(inboundIndex)%len(protocols)]
		upstream := protocols[int(upstreamIndex)%len(protocols)]
		converted, _ := ConvertRequest(body, inbound, upstream, "mapped-model", "path-model")
		_ = ConvertResponse(body, inbound, upstream)
		_ = ExtractUsage(body)
		if len(converted.Body) > 0 && !json.Valid(converted.Body) {
			t.Fatalf("successful conversion returned invalid JSON: %q", converted.Body)
		}
	})
}

func TestOpenAIToGemini(t *testing.T) {
	body := []byte(`{"model":"gemini-2.5-pro","messages":[{"role":"system","content":"Be direct."},{"role":"user","content":"Hello"}],"max_tokens":256}`)
	out, err := ConvertRequest(body, router.ProtocolOpenAI, router.ProtocolGemini, "gemini-2.5-pro", "")
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	_ = json.Unmarshal(out.Body, &raw)
	contents := raw["contents"].([]any)
	first := contents[0].(map[string]any)
	parts := first["parts"].([]any)
	if got := parts[0].(map[string]any)["text"]; got != "Hello" {
		t.Fatalf("text = %v", got)
	}
}

func TestCrossProtocolConversionRejectsUnsupportedFeaturesInsteadOfDroppingThem(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		inbound  string
		upstream string
	}{
		{"OpenAI tools", `{"model":"gpt","messages":[{"role":"user","content":"hi"}],"tools":[{"type":"function","function":{"name":"weather"}}]}`, router.ProtocolOpenAI, router.ProtocolAnthropic},
		{"OpenAI image", `{"model":"gpt","messages":[{"role":"user","content":[{"type":"image_url","image_url":{"url":"https://example.test/a.png"}}]}]}`, router.ProtocolOpenAI, router.ProtocolGemini},
		{"Anthropic tool", `{"model":"claude","messages":[{"role":"user","content":"hi"}],"tools":[{"name":"weather"}]}`, router.ProtocolAnthropic, router.ProtocolOpenAI},
		{"Gemini safety", `{"contents":[{"parts":[{"text":"hi"}]}],"safetySettings":[{"category":"x"}]}`, router.ProtocolGemini, router.ProtocolOpenAI},
		{"Responses state", `{"model":"gpt","input":"hi","previous_response_id":"resp_1"}`, router.ProtocolOpenAIResponses, router.ProtocolOpenAI},
		{"Responses built-in tool", `{"model":"gpt","input":"hi","tools":[{"type":"web_search"}]}`, router.ProtocolOpenAIResponses, router.ProtocolOpenAI},
		{"Chat multiple choices", `{"model":"gpt","messages":[{"role":"user","content":"hi"}],"n":2}`, router.ProtocolOpenAI, router.ProtocolOpenAIResponses},
		{"Chat logprobs", `{"model":"gpt","messages":[{"role":"user","content":"hi"}],"logprobs":true}`, router.ProtocolOpenAI, router.ProtocolOpenAIResponses},
		{"Chat audio input", `{"model":"gpt","messages":[{"role":"user","content":[{"type":"input_audio","input_audio":{"data":"AAAA","format":"wav"}}]}]}`, router.ProtocolOpenAI, router.ProtocolOpenAIResponses},
		{"Chat assistant refusal", `{"model":"gpt","messages":[{"role":"assistant","content":null,"refusal":"cannot comply"}]}`, router.ProtocolOpenAI, router.ProtocolOpenAIResponses},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := ConvertRequest([]byte(test.body), test.inbound, test.upstream, "mapped", ""); err == nil || !strings.Contains(err.Error(), "does not support") {
				t.Fatalf("conversion error = %v", err)
			}
		})
	}
}

func TestConvertResponseCheckedRejectsMalformedSuccessfulPayload(t *testing.T) {
	if _, err := ConvertResponseChecked([]byte(`not-json`), router.ProtocolOpenAI, router.ProtocolOpenAI); err == nil {
		t.Fatal("malformed native response was accepted")
	}
	if _, err := ConvertResponseChecked([]byte(`[]`), router.ProtocolOpenAIResponses, router.ProtocolOpenAI); err == nil {
		t.Fatal("non-object cross-protocol response was accepted")
	}
}

func TestProviderResourceRefsFindsStateAndHostedResources(t *testing.T) {
	refs, stateful, err := ProviderResourceRefs([]byte(`{"previous_response_id":"resp_1","conversation":{"id":"conv_1"},"input":[{"content":[{"type":"input_file","file_id":"file_1"}]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if !stateful || len(refs) != 2 {
		t.Fatalf("refs = %#v stateful = %v", refs, stateful)
	}
}

func TestNativeGeminiKeepsModelOnlyInPath(t *testing.T) {
	body := []byte(`{"model":"public-gemini","contents":[{"role":"user","parts":[{"text":"Hello"}]}]}`)
	out, err := ConvertRequest(body, router.ProtocolGemini, router.ProtocolGemini, "gemini-2.5-pro", "public-gemini")
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(out.Body, &raw); err != nil {
		t.Fatal(err)
	}
	if _, ok := raw["model"]; ok {
		t.Fatalf("native Gemini body unexpectedly contains model: %s", out.Body)
	}
	path := UpstreamPath("/v1beta/models/public-gemini:generateContent", router.ProtocolGemini, "models/gemini-2.5-pro", false)
	if path != "/v1beta/models/gemini-2.5-pro:generateContent" {
		t.Fatalf("path = %s", path)
	}
}

func TestOpenAIResponseToGemini(t *testing.T) {
	body := []byte(`{"choices":[{"message":{"content":"Done"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":2,"total_tokens":3}}`)
	out := ConvertResponse(body, router.ProtocolGemini, router.ProtocolOpenAI)
	var raw map[string]any
	_ = json.Unmarshal(out, &raw)
	candidates := raw["candidates"].([]any)
	content := candidates[0].(map[string]any)["content"].(map[string]any)
	parts := content["parts"].([]any)
	if got := parts[0].(map[string]any)["text"]; got != "Done" {
		t.Fatalf("text = %v", got)
	}
}

func TestOpenAIResponsesRequestToChatCompletions(t *testing.T) {
	body := []byte(`{"model":"gpt-5","instructions":"Be concise.","input":[{"type":"message","role":"user","content":[{"type":"input_text","text":"Hello"}]}],"max_output_tokens":123,"stream":true}`)
	out, err := ConvertRequest(body, router.ProtocolOpenAIResponses, router.ProtocolOpenAI, "auto", "")
	if err != nil {
		t.Fatal(err)
	}
	if !out.Stream {
		t.Fatal("expected stream flag")
	}
	var raw map[string]any
	_ = json.Unmarshal(out.Body, &raw)
	if got := raw["model"]; got != "auto" {
		t.Fatalf("model = %v", got)
	}
	if got := int(raw["max_completion_tokens"].(float64)); got != 123 {
		t.Fatalf("max_completion_tokens = %d", got)
	}
	messages := raw["messages"].([]any)
	if len(messages) != 2 {
		t.Fatalf("messages = %d", len(messages))
	}
	if got := messages[0].(map[string]any)["role"]; got != "system" {
		t.Fatalf("first role = %v", got)
	}
	if got := messages[1].(map[string]any)["content"]; got != "Hello" {
		t.Fatalf("content = %v", got)
	}
}

func TestResponsesToolsAndMultimodalToChatCompletions(t *testing.T) {
	body := []byte(`{"model":"gpt-5","input":[{"type":"message","role":"user","content":[{"type":"input_text","text":"inspect"},{"type":"input_image","image_url":"https://example.test/a.png","detail":"high"}]},{"type":"function_call_output","call_id":"call_1","output":"sunny"}],"tools":[{"type":"function","name":"weather","description":"Weather","parameters":{"type":"object"},"strict":true}],"tool_choice":{"type":"function","name":"weather"},"text":{"format":{"type":"json_schema","name":"answer","schema":{"type":"object"},"strict":true},"verbosity":"high"},"reasoning":{"effort":"high"},"max_output_tokens":321,"stream":true}`)
	out, err := ConvertRequest(body, router.ProtocolOpenAIResponses, router.ProtocolOpenAI, "gpt-5", "")
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(out.Body, &raw); err != nil {
		t.Fatal(err)
	}
	if raw["max_completion_tokens"] != float64(321) || asMap(raw["stream_options"])["include_usage"] != true {
		t.Fatalf("token/stream mapping = %#v", raw)
	}
	messages := slice(raw["messages"])
	if len(messages) != 2 || stringField(asMap(messages[1]), "role") != "tool" {
		t.Fatalf("messages = %#v", messages)
	}
	content := slice(asMap(messages[0])["content"])
	if len(content) != 2 || stringField(asMap(content[1]), "type") != "image_url" {
		t.Fatalf("multimodal content = %#v", content)
	}
	tool := asMap(slice(raw["tools"])[0])
	if stringField(asMap(tool["function"]), "name") != "weather" {
		t.Fatalf("tools = %#v", raw["tools"])
	}
	if stringField(asMap(raw["response_format"]), "type") != "json_schema" || raw["verbosity"] != "high" || raw["reasoning_effort"] != "high" {
		t.Fatalf("format/reasoning = %#v", raw)
	}
}

func TestChatCompletionsToolsToResponses(t *testing.T) {
	body := []byte(`{"model":"gpt-5","messages":[{"role":"user","content":[{"type":"text","text":"weather"},{"type":"image_url","image_url":{"url":"https://example.test/a.png","detail":"low"}}]},{"role":"assistant","content":null,"tool_calls":[{"id":"call_1","type":"function","function":{"name":"weather","arguments":"{\"city\":\"Paris\"}"}}]},{"role":"tool","tool_call_id":"call_1","content":"sunny"}],"tools":[{"type":"function","function":{"name":"weather","parameters":{"type":"object"},"strict":true}}],"tool_choice":{"type":"function","function":{"name":"weather"}},"response_format":{"type":"json_schema","json_schema":{"name":"answer","schema":{"type":"object"}}},"verbosity":"low","reasoning_effort":"medium","max_completion_tokens":222,"stream":true}`)
	out, err := ConvertRequest(body, router.ProtocolOpenAI, router.ProtocolOpenAIResponses, "gpt-5", "")
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	_ = json.Unmarshal(out.Body, &raw)
	if raw["max_output_tokens"] != float64(222) || !boolField(raw, "stream") {
		t.Fatalf("request = %#v", raw)
	}
	input := slice(raw["input"])
	if len(input) != 3 || stringField(asMap(input[1]), "type") != "function_call" || stringField(asMap(input[2]), "type") != "function_call_output" {
		t.Fatalf("input = %#v", input)
	}
	if stringField(asMap(slice(raw["tools"])[0]), "name") != "weather" || stringField(asMap(raw["tool_choice"]), "name") != "weather" {
		t.Fatalf("tools = %#v choice = %#v", raw["tools"], raw["tool_choice"])
	}
	if stringField(asMap(asMap(raw["text"])["format"]), "type") != "json_schema" || stringField(asMap(raw["text"]), "verbosity") != "low" || stringField(asMap(raw["reasoning"]), "effort") != "medium" {
		t.Fatalf("format/reasoning = %#v", raw)
	}
}

func TestToolResponsesConvertBothDirections(t *testing.T) {
	chat := []byte(`{"id":"chatcmpl-1","created":1700000000,"model":"gpt-5","choices":[{"message":{"role":"assistant","content":null,"tool_calls":[{"id":"call_1","type":"function","function":{"name":"weather","arguments":"{\"city\":\"Paris\"}"}}]},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":2,"completion_tokens":3,"total_tokens":5}}`)
	responses := ConvertResponse(chat, router.ProtocolOpenAIResponses, router.ProtocolOpenAI)
	if !strings.Contains(string(responses), `"type":"function_call"`) || !strings.Contains(string(responses), `"call_id":"call_1"`) {
		t.Fatalf("responses = %s", responses)
	}
	back := ConvertResponse(responses, router.ProtocolOpenAI, router.ProtocolOpenAIResponses)
	if !strings.Contains(string(back), `"finish_reason":"tool_calls"`) || !strings.Contains(string(back), `"tool_calls"`) {
		t.Fatalf("chat = %s", back)
	}
}

func TestToolStreamsConvertBothDirections(t *testing.T) {
	responsesState := &StreamState{}
	responsesFrames := convertTestStream(t, responsesState, router.ProtocolOpenAIResponses, router.ProtocolOpenAI, []streamTestEvent{
		{data: `{"id":"chatcmpl-test","model":"gpt-5","choices":[{"delta":{"role":"assistant","tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"weather","arguments":"{\"city\":"}}]},"finish_reason":null}]}`},
		{data: `{"id":"chatcmpl-test","model":"gpt-5","choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"Paris\"}"}}]},"finish_reason":"tool_calls"}]}`},
		{data: `[DONE]`},
	})
	joined := streamFrameText(responsesFrames)
	for _, expected := range []string{"response.output_item.added", "response.function_call_arguments.delta", "response.function_call_arguments.done", `"name":"weather"`, "response.completed"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("responses stream missing %q: %s", expected, joined)
		}
	}

	chatState := &StreamState{}
	chatFrames := convertTestStream(t, chatState, router.ProtocolOpenAI, router.ProtocolOpenAIResponses, []streamTestEvent{
		{event: "response.created", data: `{"type":"response.created","response":{"id":"resp_1","model":"gpt-5","status":"in_progress"}}`},
		{event: "response.output_item.added", data: `{"type":"response.output_item.added","output_index":0,"item":{"id":"fc_1","type":"function_call","call_id":"call_1","name":"weather","arguments":""}}`},
		{event: "response.function_call_arguments.delta", data: `{"type":"response.function_call_arguments.delta","output_index":0,"item_id":"fc_1","delta":"{\"city\":\"Paris\"}"}`},
		{event: "response.output_item.done", data: `{"type":"response.output_item.done","output_index":0,"item":{"id":"fc_1","type":"function_call","call_id":"call_1","name":"weather","arguments":"{\"city\":\"Paris\"}"}}`},
		{event: "response.completed", data: `{"type":"response.completed","response":{"id":"resp_1","model":"gpt-5","status":"completed","usage":{"input_tokens":2,"output_tokens":3,"total_tokens":5}}}`},
	})
	joined = streamFrameText(chatFrames)
	for _, expected := range []string{`"tool_calls"`, `"name":"weather"`, `"finish_reason":"tool_calls"`, `"total_tokens":5`, "[DONE]"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("chat stream missing %q: %s", expected, joined)
		}
	}
}

func TestJSONResponsesCanBeRenderedAsStreams(t *testing.T) {
	chatBody := []byte(`{"id":"chatcmpl-json","created":1700000000,"model":"gpt-5","choices":[{"message":{"role":"assistant","content":"hello"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":2,"total_tokens":3}}`)
	chatFrames, usage, _, err := ConvertJSONResponseToStream(chatBody, router.ProtocolOpenAI, router.ProtocolOpenAI)
	if err != nil {
		t.Fatal(err)
	}
	chat := streamFrameText(chatFrames)
	if !strings.Contains(chat, `"content":"hello"`) || !strings.Contains(chat, `"finish_reason":"stop"`) || !strings.Contains(chat, "[DONE]") || usage.TotalTokens == nil || *usage.TotalTokens != 3 {
		t.Fatalf("chat stream = %s usage = %#v", chat, usage)
	}

	responseBody := []byte(`{"id":"resp_json","object":"response","created_at":1700000000,"status":"completed","model":"gpt-5","output":[{"id":"msg_1","type":"message","role":"assistant","content":[{"type":"output_text","text":"hello"}]}],"usage":{"input_tokens":1,"output_tokens":2,"total_tokens":3}}`)
	responseFrames, usage, resourceIDs, err := ConvertJSONResponseToStream(responseBody, router.ProtocolOpenAIResponses, router.ProtocolOpenAIResponses)
	if err != nil {
		t.Fatal(err)
	}
	responses := streamFrameText(responseFrames)
	for _, expected := range []string{"response.created", "response.content_part.added", "response.output_text.delta", "response.completed", `"total_tokens":3`} {
		if !strings.Contains(responses, expected) {
			t.Fatalf("responses stream missing %q: %s", expected, responses)
		}
	}
	if usage.TotalTokens == nil || *usage.TotalTokens != 3 {
		t.Fatalf("usage = %#v", usage)
	}
	if len(resourceIDs) != 1 || resourceIDs[0] != "resp_json" {
		t.Fatalf("resource IDs = %#v", resourceIDs)
	}
}

func TestChatCompletionsResponseToOpenAIResponses(t *testing.T) {
	body := []byte(`{"id":"chatcmpl-test","object":"chat.completion","created":1700000000,"model":"gpt-5","choices":[{"message":{"role":"assistant","content":"Done"},"finish_reason":"stop"}],"usage":{"prompt_tokens":2,"completion_tokens":3,"total_tokens":5}}`)
	out := ConvertResponse(body, router.ProtocolOpenAIResponses, router.ProtocolOpenAI)
	var raw map[string]any
	_ = json.Unmarshal(out, &raw)
	if got := raw["object"]; got != "response" {
		t.Fatalf("object = %v", got)
	}
	if got := raw["output_text"]; got != "Done" {
		t.Fatalf("output_text = %v", got)
	}
	usage := raw["usage"].(map[string]any)
	if got := int(usage["total_tokens"].(float64)); got != 5 {
		t.Fatalf("total_tokens = %d", got)
	}
}

func TestAnthropicStreamConvertsToOpenAI(t *testing.T) {
	state := &StreamState{}
	frames := convertTestStream(t, state, router.ProtocolOpenAI, router.ProtocolAnthropic, []streamTestEvent{
		{event: "message_start", data: `{"type":"message_start","message":{"id":"msg_test","model":"claude","usage":{"input_tokens":4,"output_tokens":0}}}`},
		{event: "content_block_delta", data: `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"hello"}}`},
		{event: "message_delta", data: `{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":2}}`},
		{event: "message_stop", data: `{"type":"message_stop"}`},
	})
	joined := streamFrameText(frames)
	for _, expected := range []string{`"role":"assistant"`, `"content":"hello"`, `"finish_reason":"stop"`, "[DONE]"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("converted stream missing %q: %s", expected, joined)
		}
	}
	if state.Usage.PromptTokens == nil || *state.Usage.PromptTokens != 4 || state.Usage.CompletionTokens == nil || *state.Usage.CompletionTokens != 2 {
		t.Fatalf("usage = %#v", state.Usage)
	}
}

func TestOpenAIStreamConvertsToAnthropic(t *testing.T) {
	state := &StreamState{}
	frames := convertTestStream(t, state, router.ProtocolAnthropic, router.ProtocolOpenAI, []streamTestEvent{
		{data: `{"id":"chatcmpl-test","model":"gpt","choices":[{"delta":{"role":"assistant","content":"hello"},"finish_reason":null}]}`},
		{data: `{"id":"chatcmpl-test","model":"gpt","choices":[{"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":3,"completion_tokens":1,"total_tokens":4}}`},
		{data: `[DONE]`},
	})
	joined := streamFrameText(frames)
	for _, expected := range []string{"message_start", "content_block_delta", `"text":"hello"`, "message_stop"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("converted stream missing %q: %s", expected, joined)
		}
	}
}

func TestGeminiStreamConvertsToOpenAIResponses(t *testing.T) {
	state := &StreamState{}
	frames := convertTestStream(t, state, router.ProtocolOpenAIResponses, router.ProtocolGemini, []streamTestEvent{
		{data: `{"modelVersion":"gemini","candidates":[{"content":{"role":"model","parts":[{"text":"hello"}]}}]}`},
		{data: `{"modelVersion":"gemini","candidates":[{"content":{"role":"model","parts":[{"text":" world"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":2,"candidatesTokenCount":2,"totalTokenCount":4}}`},
	})
	joined := streamFrameText(frames)
	for _, expected := range []string{"response.created", "response.output_text.delta", "hello", " world", "response.completed", `"total_tokens":4`} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("converted stream missing %q: %s", expected, joined)
		}
	}
}

func TestOpenAIStreamConvertsToGemini(t *testing.T) {
	state := &StreamState{}
	frames := convertTestStream(t, state, router.ProtocolGemini, router.ProtocolOpenAI, []streamTestEvent{
		{data: `{"choices":[{"delta":{"content":"hello"},"finish_reason":null}]}`},
		{data: `{"choices":[{"delta":{},"finish_reason":"length"}],"usage":{"prompt_tokens":1,"completion_tokens":2,"total_tokens":3}}`},
		{data: `[DONE]`},
	})
	joined := streamFrameText(frames)
	for _, expected := range []string{`"text":"hello"`, `"finishReason":"MAX_TOKENS"`, `"totalTokenCount":3`} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("converted stream missing %q: %s", expected, joined)
		}
	}
}

func TestStreamToolArgumentsRespectPerCallLimit(t *testing.T) {
	state := &StreamState{MaxAggregateBytes: 64, MaxToolArgumentBytes: 5}
	_, err := ConvertStreamEvent("", []byte(`{"id":"chatcmpl-test","choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_1","function":{"name":"tool","arguments":"123456"}}]}}]}`), router.ProtocolOpenAIResponses, router.ProtocolOpenAI, state)
	if err == nil || !strings.Contains(err.Error(), "tool arguments") {
		t.Fatalf("tool argument limit error = %v", err)
	}
}

func TestNativeResponsesStreamPreservesAndMarksFailureEvent(t *testing.T) {
	state := &StreamState{}
	data := []byte(`{"type":"response.failed","response":{"id":"resp_failed","status":"failed","error":{"message":"capacity exhausted"}}}`)
	frames, err := ConvertStreamEvent("response.failed", data, router.ProtocolOpenAIResponses, router.ProtocolOpenAIResponses, state)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 1 || frames[0].Event != "response.failed" || string(frames[0].Data) != string(data) {
		t.Fatalf("native failure frame = %#v", frames)
	}
	if state.UpstreamError == nil || !IsUpstreamStreamError(state.UpstreamError) || !strings.Contains(state.UpstreamError.Error(), "capacity exhausted") {
		t.Fatalf("native upstream error = %v", state.UpstreamError)
	}
}

type streamTestEvent struct {
	event string
	data  string
}

func convertTestStream(t *testing.T, state *StreamState, inbound, upstream string, events []streamTestEvent) []StreamFrame {
	t.Helper()
	var frames []StreamFrame
	for _, event := range events {
		converted, err := ConvertStreamEvent(event.event, []byte(event.data), inbound, upstream, state)
		if err != nil {
			t.Fatal(err)
		}
		frames = append(frames, converted...)
	}
	return frames
}

func streamFrameText(frames []StreamFrame) string {
	var out strings.Builder
	for _, frame := range frames {
		out.WriteString(frame.Event)
		out.Write(frame.Data)
	}
	return out.String()
}
