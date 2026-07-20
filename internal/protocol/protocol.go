package protocol

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"local-ai-gateway/internal/model"
	"local-ai-gateway/internal/router"
)

type ConvertedRequest struct {
	Body   []byte
	Model  string
	Stream bool
}

type Usage struct {
	PromptTokens     *int
	CompletionTokens *int
	TotalTokens      *int
}

var geminiPathRE = regexp.MustCompile(`/v1(?:beta)?/models/([^:/]+)(?::[^/]+)?`)

func DetectInbound(path string) string {
	if (strings.Contains(path, "/v1beta/models/") || strings.Contains(path, "/v1/models/")) &&
		(strings.Contains(path, ":generateContent") || strings.Contains(path, ":streamGenerateContent")) {
		return router.ProtocolGemini
	}
	if strings.HasSuffix(path, "/messages") {
		return router.ProtocolAnthropic
	}
	if strings.Contains(path, "/responses") {
		return router.ProtocolOpenAIResponses
	}
	return router.ProtocolOpenAI
}

func ExtractPathModel(path string) string {
	m := geminiPathRE.FindStringSubmatch(path)
	if len(m) < 2 {
		return ""
	}
	v, err := url.PathUnescape(m[1])
	if err != nil {
		return m[1]
	}
	return v
}

func UpstreamPath(inboundPath, upstreamProtocol, modelName string, stream bool) string {
	switch upstreamProtocol {
	case router.ProtocolOpenAIResponses:
		return "/v1/responses"
	case router.ProtocolAnthropic:
		return "/v1/messages"
	case router.ProtocolGemini:
		action := ":generateContent"
		if stream || strings.Contains(inboundPath, ":streamGenerateContent") {
			action = ":streamGenerateContent"
		}
		modelName = model.NormalizeModelID(model.ProviderGeminiCompatible, modelName)
		return "/v1beta/models/" + url.PathEscape(modelName) + action
	default:
		if strings.Contains(inboundPath, "/responses") {
			return "/v1/chat/completions"
		}
		if strings.Contains(inboundPath, "/completions") && !strings.Contains(inboundPath, "/chat/completions") {
			return "/v1/completions"
		}
		return "/v1/chat/completions"
	}
}

func ConvertRequest(body []byte, inbound, upstream, upstreamModel, pathModel string) (ConvertedRequest, error) {
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return ConvertedRequest{}, fmt.Errorf("decode request JSON: %w", err)
	}
	if err := validateCrossProtocolRequest(raw, inbound, upstream); err != nil {
		return ConvertedRequest{}, err
	}
	modelName := stringField(raw, "model")
	if modelName == "" {
		modelName = pathModel
	}
	if inbound == router.ProtocolOpenAIResponses && upstream == router.ProtocolOpenAI {
		converted := responsesToChatCompletions(raw, upstreamModel)
		out, err := json.Marshal(converted)
		return ConvertedRequest{Body: out, Model: modelName, Stream: boolField(converted, "stream")}, err
	}
	if inbound == router.ProtocolOpenAI && upstream == router.ProtocolOpenAIResponses {
		converted := chatCompletionsToResponses(raw, upstreamModel)
		out, err := json.Marshal(converted)
		return ConvertedRequest{Body: out, Model: modelName, Stream: boolField(converted, "stream")}, err
	}
	if inbound == upstream {
		if upstream == router.ProtocolGemini {
			delete(raw, "model")
		} else {
			raw["model"] = upstreamModel
		}
		out, err := json.Marshal(raw)
		return ConvertedRequest{Body: out, Model: modelName, Stream: boolField(raw, "stream")}, err
	}

	var converted any
	switch {
	case inbound == router.ProtocolAnthropic && upstream == router.ProtocolOpenAI:
		converted = anthropicToOpenAI(raw, upstreamModel)
	case inbound == router.ProtocolOpenAI && upstream == router.ProtocolAnthropic:
		converted = openAIToAnthropic(raw, upstreamModel)
	case inbound == router.ProtocolGemini && upstream == router.ProtocolOpenAI:
		converted = geminiToOpenAI(raw, upstreamModel)
	case inbound == router.ProtocolOpenAI && upstream == router.ProtocolGemini:
		converted = openAIToGemini(raw)
	case inbound == router.ProtocolGemini && upstream == router.ProtocolAnthropic:
		converted = geminiToAnthropic(raw, upstreamModel)
	case inbound == router.ProtocolAnthropic && upstream == router.ProtocolGemini:
		converted = anthropicToGemini(raw)
	default:
		raw["model"] = upstreamModel
		converted = raw
	}
	out, err := json.Marshal(converted)
	return ConvertedRequest{Body: out, Model: modelName, Stream: boolField(raw, "stream")}, err
}

func ConvertResponse(body []byte, inbound, upstream string) []byte {
	out, err := ConvertResponseChecked(body, inbound, upstream)
	if err != nil {
		return body
	}
	return out
}

// ConvertResponseChecked validates successful upstream JSON before returning it
// to the client. A malformed 2xx response is an upstream protocol failure, not
// a successful opaque response.
func ConvertResponseChecked(body []byte, inbound, upstream string) ([]byte, error) {
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode successful %s response: %w", upstream, err)
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("successful %s response is not a JSON object", upstream)
	}
	if inbound == upstream {
		return body, nil
	}
	var converted any
	switch {
	case inbound == router.ProtocolOpenAIResponses && upstream == router.ProtocolOpenAI:
		converted = chatCompletionResponseToResponses(raw)
	case inbound == router.ProtocolOpenAI && upstream == router.ProtocolOpenAIResponses:
		converted = responsesResponseToChatCompletions(raw)
	case inbound == router.ProtocolAnthropic && upstream == router.ProtocolOpenAI:
		converted = openAIResponseToAnthropic(raw)
	case inbound == router.ProtocolOpenAI && upstream == router.ProtocolAnthropic:
		converted = anthropicResponseToOpenAI(raw)
	case inbound == router.ProtocolGemini && upstream == router.ProtocolOpenAI:
		converted = openAIResponseToGemini(raw)
	case inbound == router.ProtocolOpenAI && upstream == router.ProtocolGemini:
		converted = geminiResponseToOpenAI(raw)
	case inbound == router.ProtocolGemini && upstream == router.ProtocolAnthropic:
		converted = anthropicResponseToGemini(raw)
	case inbound == router.ProtocolAnthropic && upstream == router.ProtocolGemini:
		converted = geminiResponseToAnthropic(raw)
	default:
		return nil, fmt.Errorf("unsupported response conversion from %s to %s", upstream, inbound)
	}
	out, err := json.Marshal(converted)
	if err != nil {
		return nil, fmt.Errorf("encode converted %s response: %w", inbound, err)
	}
	return out, nil
}

func ExtractUsage(body []byte) Usage {
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return Usage{}
	}
	if u, ok := raw["usage"].(map[string]any); ok {
		p := intFromAny(u["prompt_tokens"], u["input_tokens"])
		c := intFromAny(u["completion_tokens"], u["output_tokens"])
		t := intFromAny(u["total_tokens"])
		if t == nil && p != nil && c != nil {
			n := *p + *c
			t = &n
		}
		return Usage{PromptTokens: p, CompletionTokens: c, TotalTokens: t}
	}
	if u, ok := raw["usageMetadata"].(map[string]any); ok {
		return Usage{
			PromptTokens:     intFromAny(u["promptTokenCount"]),
			CompletionTokens: intFromAny(u["candidatesTokenCount"]),
			TotalTokens:      intFromAny(u["totalTokenCount"]),
		}
	}
	return Usage{}
}

func openAIToAnthropic(in map[string]any, modelName string) map[string]any {
	var system []string
	var messages []map[string]any
	for _, item := range slice(in["messages"]) {
		msg := asMap(item)
		role := stringField(msg, "role")
		if role == "system" || role == "developer" {
			system = append(system, contentText(msg["content"]))
			continue
		}
		outRole := "user"
		if role == "assistant" {
			outRole = "assistant"
		}
		messages = append(messages, map[string]any{"role": outRole, "content": []map[string]any{{"type": "text", "text": contentText(msg["content"])}}})
	}
	out := map[string]any{"model": modelName, "messages": messages, "max_tokens": intOr(in["max_tokens"], intOr(in["max_completion_tokens"], 4096)), "stream": boolField(in, "stream")}
	if len(system) > 0 {
		out["system"] = strings.Join(system, "\n\n")
	}
	if v, ok := in["temperature"]; ok {
		out["temperature"] = v
	}
	if v, ok := in["top_p"]; ok {
		out["top_p"] = v
	}
	if v, ok := in["stop"]; ok {
		out["stop_sequences"] = v
	}
	return out
}

func anthropicToOpenAI(in map[string]any, modelName string) map[string]any {
	var messages []map[string]any
	if sys := contentText(in["system"]); sys != "" {
		messages = append(messages, map[string]any{"role": "system", "content": sys})
	}
	for _, item := range slice(in["messages"]) {
		msg := asMap(item)
		role := "user"
		if stringField(msg, "role") == "assistant" {
			role = "assistant"
		}
		messages = append(messages, map[string]any{"role": role, "content": contentText(msg["content"])})
	}
	out := map[string]any{"model": modelName, "messages": messages, "stream": boolField(in, "stream")}
	if v, ok := in["max_tokens"]; ok {
		out["max_tokens"] = v
	}
	if v, ok := in["temperature"]; ok {
		out["temperature"] = v
	}
	if v, ok := in["top_p"]; ok {
		out["top_p"] = v
	}
	if v, ok := in["stop_sequences"]; ok {
		out["stop"] = v
	}
	return out
}

func responsesToChatCompletions(in map[string]any, modelName string) map[string]any {
	out := map[string]any{
		"model":    modelName,
		"messages": responsesMessages(in),
	}
	if v, ok := in["stream"]; ok {
		out["stream"] = v
	}
	if v, ok := in["temperature"]; ok {
		out["temperature"] = v
	}
	if v, ok := in["top_p"]; ok {
		out["top_p"] = v
	}
	if v, ok := in["max_output_tokens"]; ok {
		out["max_completion_tokens"] = v
	} else if v, ok := in["max_tokens"]; ok {
		out["max_tokens"] = v
	} else if v, ok := in["max_completion_tokens"]; ok {
		out["max_completion_tokens"] = v
	}
	copyFields(out, in, "frequency_penalty", "presence_penalty", "parallel_tool_calls", "prompt_cache_key", "safety_identifier", "seed", "service_tier", "stop", "user")
	if tools := responsesToolsToChat(in["tools"]); len(tools) > 0 {
		out["tools"] = tools
	}
	if choice := responsesToolChoiceToChat(in["tool_choice"]); choice != nil {
		out["tool_choice"] = choice
	}
	if text := asMap(in["text"]); len(text) > 0 {
		if format := asMap(text["format"]); len(format) > 0 {
			out["response_format"] = responsesFormatToChat(format)
		}
	}
	if reasoning := asMap(in["reasoning"]); len(reasoning) > 0 {
		if effort, ok := reasoning["effort"]; ok {
			out["reasoning_effort"] = effort
		}
	}
	if boolField(in, "stream") {
		out["stream_options"] = map[string]any{"include_usage": true}
	}
	return out
}

func chatCompletionsToResponses(in map[string]any, modelName string) map[string]any {
	out := map[string]any{"model": modelName, "input": chatMessagesToResponses(in["messages"])}
	copyFields(out, in, "stream", "temperature", "top_p", "frequency_penalty", "presence_penalty", "parallel_tool_calls", "metadata", "store", "service_tier", "prompt_cache_key", "safety_identifier", "user")
	if v, ok := in["max_completion_tokens"]; ok {
		out["max_output_tokens"] = v
	} else if v, ok := in["max_tokens"]; ok {
		out["max_output_tokens"] = v
	}
	if tools := chatToolsToResponses(in["tools"]); len(tools) > 0 {
		out["tools"] = tools
	}
	if choice := chatToolChoiceToResponses(in["tool_choice"]); choice != nil {
		out["tool_choice"] = choice
	}
	if format := asMap(in["response_format"]); len(format) > 0 {
		out["text"] = map[string]any{"format": chatFormatToResponses(format)}
	}
	if effort, ok := in["reasoning_effort"]; ok {
		out["reasoning"] = map[string]any{"effort": effort}
	} else if reasoning := asMap(in["reasoning"]); len(reasoning) > 0 {
		out["reasoning"] = reasoning
	}
	return out
}

func responsesMessages(in map[string]any) []map[string]any {
	var messages []map[string]any
	if instructions := contentText(in["instructions"]); instructions != "" {
		messages = append(messages, map[string]any{"role": "system", "content": instructions})
	}
	if existing := slice(in["messages"]); len(existing) > 0 {
		for _, item := range existing {
			messages = append(messages, responseInputMessages(item)...)
		}
	}
	switch input := in["input"].(type) {
	case string:
		if input != "" {
			messages = append(messages, map[string]any{"role": "user", "content": input})
		}
	case []any:
		for _, item := range input {
			messages = append(messages, responseInputMessages(item)...)
		}
	default:
		if text := contentText(input); text != "" {
			messages = append(messages, map[string]any{"role": "user", "content": text})
		}
	}
	if len(messages) == 0 {
		messages = append(messages, map[string]any{"role": "user", "content": ""})
	}
	return messages
}

func responseInputMessages(item any) []map[string]any {
	msg := asMap(item)
	switch stringField(msg, "type") {
	case "function_call_output":
		return []map[string]any{{"role": "tool", "tool_call_id": stringField(msg, "call_id"), "content": contentText(msg["output"])}}
	case "function_call":
		callID := stringField(msg, "call_id")
		if callID == "" {
			callID = stringField(msg, "id")
		}
		return []map[string]any{{"role": "assistant", "content": nil, "tool_calls": []map[string]any{{"id": callID, "type": "function", "function": map[string]any{"name": stringField(msg, "name"), "arguments": stringOr(msg["arguments"], "{}")}}}}}
	}
	if converted, ok := responseInputMessage(item); ok {
		return []map[string]any{converted}
	}
	return nil
}

func responseInputMessage(item any) (map[string]any, bool) {
	msg := asMap(item)
	if len(msg) == 0 {
		if text := contentText(item); text != "" {
			return map[string]any{"role": "user", "content": text}, true
		}
		return nil, false
	}
	if typ := stringField(msg, "type"); typ != "" && typ != "message" && typ != "input_text" && typ != "output_text" {
		return nil, false
	}
	role := stringField(msg, "role")
	switch role {
	case "assistant":
		role = "assistant"
	case "system", "developer":
		role = "system"
	default:
		role = "user"
	}
	content := responsesContentToChat(msg["content"])
	if content == nil {
		content = responsesContentToChat(msg["text"])
	}
	if content == nil {
		return nil, false
	}
	return map[string]any{"role": role, "content": content}, true
}

func openAIToGemini(in map[string]any) map[string]any {
	var contents []map[string]any
	var system map[string]any
	for _, item := range slice(in["messages"]) {
		msg := asMap(item)
		role := stringField(msg, "role")
		if role == "system" || role == "developer" {
			system = map[string]any{"parts": []map[string]any{{"text": contentText(msg["content"])}}}
			continue
		}
		gRole := "user"
		if role == "assistant" {
			gRole = "model"
		}
		contents = append(contents, map[string]any{"role": gRole, "parts": []map[string]any{{"text": contentText(msg["content"])}}})
	}
	out := map[string]any{"contents": contents}
	if system != nil {
		out["systemInstruction"] = system
	}
	cfg := map[string]any{}
	if v, ok := in["temperature"]; ok {
		cfg["temperature"] = v
	}
	if v, ok := in["top_p"]; ok {
		cfg["topP"] = v
	}
	if v, ok := in["stop"]; ok {
		cfg["stopSequences"] = v
	}
	if v, ok := in["max_tokens"]; ok {
		cfg["maxOutputTokens"] = v
	} else if v, ok := in["max_completion_tokens"]; ok {
		cfg["maxOutputTokens"] = v
	}
	if len(cfg) > 0 {
		out["generationConfig"] = cfg
	}
	return out
}

func geminiToOpenAI(in map[string]any, modelName string) map[string]any {
	var messages []map[string]any
	if sys := asMap(in["systemInstruction"]); len(sys) > 0 {
		messages = append(messages, map[string]any{"role": "system", "content": partsText(sys["parts"])})
	}
	for _, item := range slice(in["contents"]) {
		msg := asMap(item)
		role := "user"
		if stringField(msg, "role") == "model" {
			role = "assistant"
		}
		messages = append(messages, map[string]any{"role": role, "content": partsText(msg["parts"])})
	}
	out := map[string]any{"model": modelName, "messages": messages, "stream": false}
	if cfg := asMap(in["generationConfig"]); len(cfg) > 0 {
		if v, ok := cfg["temperature"]; ok {
			out["temperature"] = v
		}
		if v, ok := cfg["maxOutputTokens"]; ok {
			out["max_tokens"] = v
		}
		if v, ok := cfg["topP"]; ok {
			out["top_p"] = v
		}
		if v, ok := cfg["stopSequences"]; ok {
			out["stop"] = v
		}
	}
	return out
}

func validateCrossProtocolRequest(raw map[string]any, inbound, upstream string) error {
	if inbound == upstream {
		return nil
	}
	if inbound == router.ProtocolOpenAIResponses && upstream == router.ProtocolOpenAI {
		return validateResponsesToChat(raw)
	}
	if inbound == router.ProtocolOpenAI && upstream == router.ProtocolOpenAIResponses {
		return validateChatToResponses(raw)
	}
	switch inbound {
	case router.ProtocolOpenAI:
		if _, hasPrompt := raw["prompt"]; hasPrompt && len(slice(raw["messages"])) == 0 {
			return unsupportedCrossProtocol(inbound, upstream, "legacy completions prompt")
		}
		for _, field := range []string{"tools", "tool_choice", "functions", "function_call", "response_format", "modalities", "audio", "logprobs", "top_logprobs"} {
			if meaningful(raw[field]) {
				return unsupportedCrossProtocol(inbound, upstream, field)
			}
		}
		for _, item := range slice(raw["messages"]) {
			message := asMap(item)
			if role := stringField(message, "role"); role == "tool" || role == "function" {
				return unsupportedCrossProtocol(inbound, upstream, role+" message")
			}
			if meaningful(message["tool_calls"]) || meaningful(message["function_call"]) {
				return unsupportedCrossProtocol(inbound, upstream, "message tool calls")
			}
			if err := validateTextOnlyContent(message["content"], inbound, upstream); err != nil {
				return err
			}
		}
	case router.ProtocolAnthropic:
		for _, field := range []string{"tools", "tool_choice", "thinking"} {
			if meaningful(raw[field]) {
				return unsupportedCrossProtocol(inbound, upstream, field)
			}
		}
		for _, item := range slice(raw["messages"]) {
			if err := validateTextOnlyContent(asMap(item)["content"], inbound, upstream); err != nil {
				return err
			}
		}
	case router.ProtocolGemini:
		for _, field := range []string{"tools", "toolConfig", "safetySettings", "cachedContent"} {
			if meaningful(raw[field]) {
				return unsupportedCrossProtocol(inbound, upstream, field)
			}
		}
		for _, item := range slice(raw["contents"]) {
			for _, part := range slice(asMap(item)["parts"]) {
				block := asMap(part)
				if len(block) != 1 || block["text"] == nil {
					return unsupportedCrossProtocol(inbound, upstream, "non-text content part")
				}
			}
		}
		if cfg := asMap(raw["generationConfig"]); len(cfg) > 0 {
			for field, value := range cfg {
				switch field {
				case "maxOutputTokens", "temperature", "topP", "stopSequences":
				default:
					if meaningful(value) {
						return unsupportedCrossProtocol(inbound, upstream, "generationConfig."+field)
					}
				}
			}
		}
	}
	return nil
}

func validateResponsesToChat(raw map[string]any) error {
	for _, field := range []string{"background", "conversation", "previous_response_id", "include", "max_tool_calls", "prompt", "truncation"} {
		if meaningful(raw[field]) {
			return unsupportedCrossProtocol(router.ProtocolOpenAIResponses, router.ProtocolOpenAI, field)
		}
	}
	if value, ok := raw["store"]; ok && value == true {
		return unsupportedCrossProtocol(router.ProtocolOpenAIResponses, router.ProtocolOpenAI, "store=true state semantics")
	}
	for _, rawTool := range slice(raw["tools"]) {
		if toolType := stringOr(asMap(rawTool)["type"], "function"); toolType != "function" {
			return unsupportedCrossProtocol(router.ProtocolOpenAIResponses, router.ProtocolOpenAI, "built-in tool "+toolType)
		}
	}
	if choice := asMap(raw["tool_choice"]); len(choice) > 0 {
		if toolType := stringField(choice, "type"); toolType != "" && toolType != "function" && toolType != "allowed_tools" {
			return unsupportedCrossProtocol(router.ProtocolOpenAIResponses, router.ProtocolOpenAI, "tool_choice "+toolType)
		}
	}
	for _, rawItem := range slice(raw["input"]) {
		item := asMap(rawItem)
		typeName := stringOr(item["type"], "message")
		switch typeName {
		case "message":
			if err := validateResponsesContentForChat(item["content"]); err != nil {
				return err
			}
		case "function_call", "function_call_output":
		default:
			return unsupportedCrossProtocol(router.ProtocolOpenAIResponses, router.ProtocolOpenAI, "input item "+typeName)
		}
	}
	return nil
}

func validateResponsesContentForChat(value any) error {
	if _, ok := value.(string); ok || value == nil {
		return nil
	}
	for _, rawPart := range slice(value) {
		typeName := stringField(asMap(rawPart), "type")
		switch typeName {
		case "input_text", "output_text", "text", "input_image", "input_file", "file":
		default:
			return unsupportedCrossProtocol(router.ProtocolOpenAIResponses, router.ProtocolOpenAI, "content block "+typeName)
		}
	}
	return nil
}

func validateChatToResponses(raw map[string]any) error {
	if n := intFromAny(raw["n"]); n != nil && *n != 1 {
		return unsupportedCrossProtocol(router.ProtocolOpenAI, router.ProtocolOpenAIResponses, "n other than 1")
	}
	for _, field := range []string{"functions", "function_call", "modalities", "audio", "prediction", "logit_bias", "logprobs", "top_logprobs", "seed", "stop", "web_search_options"} {
		if meaningful(raw[field]) {
			return unsupportedCrossProtocol(router.ProtocolOpenAI, router.ProtocolOpenAIResponses, field)
		}
	}
	for _, rawTool := range slice(raw["tools"]) {
		if toolType := stringOr(asMap(rawTool)["type"], "function"); toolType != "function" {
			return unsupportedCrossProtocol(router.ProtocolOpenAI, router.ProtocolOpenAIResponses, "tool "+toolType)
		}
	}
	return nil
}

// ProviderResourceRefs returns upstream-owned IDs that must remain on the same
// provider/key. The bool is also true for unidentifiable provider resources
// such as file/vector-store references, which still disable failover.
func ProviderResourceRefs(body []byte) ([]string, bool, error) {
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, false, err
	}
	seen := map[string]struct{}{}
	add := func(value string) {
		if value = strings.TrimSpace(value); value != "" {
			seen[value] = struct{}{}
		}
	}
	add(stringField(raw, "previous_response_id"))
	if conversation, ok := raw["conversation"].(string); ok {
		add(conversation)
	} else {
		add(stringField(asMap(raw["conversation"]), "id"))
	}
	hasResources := len(seen) > 0
	var walk func(any)
	walk = func(value any) {
		switch value := value.(type) {
		case []any:
			for _, item := range value {
				walk(item)
			}
		case map[string]any:
			for key, item := range value {
				switch key {
				case "file_id", "vector_store_id", "container_id":
					hasResources = hasResources || strings.TrimSpace(stringOr(item, "")) != ""
				case "vector_store_ids":
					hasResources = hasResources || len(slice(item)) > 0
				}
				walk(item)
			}
		}
	}
	walk(raw["input"])
	walk(raw["tools"])
	refs := make([]string, 0, len(seen))
	for id := range seen {
		refs = append(refs, id)
	}
	return refs, hasResources, nil
}

func ResponseResourceIDs(body []byte) []string {
	var raw map[string]any
	if json.Unmarshal(body, &raw) != nil {
		return nil
	}
	ids := []string{}
	if id := stringField(raw, "id"); id != "" {
		ids = append(ids, id)
	}
	if conversation := raw["conversation"]; conversation != nil {
		if id, ok := conversation.(string); ok && id != "" {
			ids = append(ids, id)
		} else if id := stringField(asMap(conversation), "id"); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

func validateTextOnlyContent(value any, inbound, upstream string) error {
	if _, ok := value.(string); ok || value == nil {
		return nil
	}
	for _, item := range slice(value) {
		block := asMap(item)
		typeName := stringField(block, "type")
		if typeName != "text" && typeName != "input_text" && typeName != "output_text" {
			return unsupportedCrossProtocol(inbound, upstream, "non-text content block")
		}
	}
	return nil
}

func unsupportedCrossProtocol(inbound, upstream, feature string) error {
	return fmt.Errorf("%s to %s conversion does not support %s", inbound, upstream, feature)
}

func meaningful(value any) bool {
	if value == nil {
		return false
	}
	switch value := value.(type) {
	case string:
		return strings.TrimSpace(value) != ""
	case []any:
		return len(value) > 0
	case map[string]any:
		return len(value) > 0
	default:
		return true
	}
}

func anthropicToGemini(in map[string]any) map[string]any {
	var contents []map[string]any
	if sys := contentText(in["system"]); sys != "" {
		contents = append(contents, map[string]any{"role": "user", "parts": []map[string]any{{"text": sys}}})
	}
	for _, item := range slice(in["messages"]) {
		msg := asMap(item)
		role := "user"
		if stringField(msg, "role") == "assistant" {
			role = "model"
		}
		contents = append(contents, map[string]any{"role": role, "parts": []map[string]any{{"text": contentText(msg["content"])}}})
	}
	out := map[string]any{"contents": contents}
	cfg := map[string]any{}
	if v, ok := in["max_tokens"]; ok {
		cfg["maxOutputTokens"] = v
	}
	if v, ok := in["temperature"]; ok {
		cfg["temperature"] = v
	}
	if len(cfg) > 0 {
		out["generationConfig"] = cfg
	}
	return out
}

func geminiToAnthropic(in map[string]any, modelName string) map[string]any {
	var messages []map[string]any
	for _, item := range slice(in["contents"]) {
		msg := asMap(item)
		role := "user"
		if stringField(msg, "role") == "model" {
			role = "assistant"
		}
		messages = append(messages, map[string]any{"role": role, "content": []map[string]any{{"type": "text", "text": partsText(msg["parts"])}}})
	}
	out := map[string]any{"model": modelName, "messages": messages, "max_tokens": 4096, "stream": false}
	if sys := asMap(in["systemInstruction"]); len(sys) > 0 {
		out["system"] = partsText(sys["parts"])
	}
	if cfg := asMap(in["generationConfig"]); len(cfg) > 0 {
		if v, ok := cfg["maxOutputTokens"]; ok {
			out["max_tokens"] = v
		}
		if v, ok := cfg["temperature"]; ok {
			out["temperature"] = v
		}
	}
	return out
}

func anthropicResponseToOpenAI(in map[string]any) map[string]any {
	text := ""
	for _, part := range slice(in["content"]) {
		text += stringField(asMap(part), "text")
	}
	return map[string]any{"id": fmt.Sprintf("chatcmpl-%d", nowUnix()), "object": "chat.completion", "created": nowUnix(), "model": stringField(in, "model"), "choices": []map[string]any{{"index": 0, "message": map[string]any{"role": "assistant", "content": text}, "finish_reason": stringOr(in["stop_reason"], "stop")}}, "usage": map[string]any{"prompt_tokens": usageInt(in, "usage", "input_tokens"), "completion_tokens": usageInt(in, "usage", "output_tokens"), "total_tokens": usageInt(in, "usage", "input_tokens") + usageInt(in, "usage", "output_tokens")}}
}

func openAIResponseToAnthropic(in map[string]any) map[string]any {
	choice := firstMap(in["choices"])
	msg := asMap(choice["message"])
	return map[string]any{"id": fmt.Sprintf("msg_%d", nowUnix()), "type": "message", "role": "assistant", "model": stringField(in, "model"), "content": []map[string]any{{"type": "text", "text": contentText(msg["content"])}}, "stop_reason": stringOr(choice["finish_reason"], "end_turn"), "usage": map[string]any{"input_tokens": usageInt(in, "usage", "prompt_tokens"), "output_tokens": usageInt(in, "usage", "completion_tokens")}}
}

func chatCompletionResponseToResponses(in map[string]any) map[string]any {
	choice := firstMap(in["choices"])
	msg := asMap(choice["message"])
	text := contentText(msg["content"])
	if text == "" {
		text = contentText(choice["text"])
	}
	created := nowUnix()
	if n := intFromAny(in["created"]); n != nil {
		created = int64(*n)
	}
	prompt, completion := usageInt(in, "usage", "prompt_tokens"), usageInt(in, "usage", "completion_tokens")
	total := usageInt(in, "usage", "total_tokens")
	if total == 0 {
		total = prompt + completion
	}
	responseID := convertedID(stringField(in, "id"), "resp", created)
	var output []map[string]any
	if text != "" || len(slice(msg["tool_calls"])) == 0 {
		output = append(output, map[string]any{
			"id": convertedID(stringField(in, "id"), "msg", created), "type": "message", "status": "completed", "role": "assistant",
			"content": []map[string]any{{"type": "output_text", "text": text, "annotations": []any{}}},
		})
	}
	for index, rawCall := range slice(msg["tool_calls"]) {
		call := asMap(rawCall)
		function := asMap(call["function"])
		callID := stringField(call, "id")
		output = append(output, map[string]any{
			"id": convertedToolID(callID, responseID, index), "type": "function_call", "status": "completed", "call_id": callID,
			"name": stringField(function, "name"), "arguments": stringOr(function["arguments"], "{}"),
		})
	}
	status := "completed"
	var incomplete any
	if stringField(choice, "finish_reason") == "length" {
		status = "incomplete"
		incomplete = map[string]any{"reason": "max_output_tokens"}
	}
	result := map[string]any{
		"id":          responseID,
		"object":      "response",
		"created_at":  created,
		"status":      status,
		"model":       stringField(in, "model"),
		"output":      output,
		"output_text": text,
		"usage": map[string]any{
			"input_tokens":  prompt,
			"output_tokens": completion,
			"total_tokens":  total,
		},
	}
	if incomplete != nil {
		result["incomplete_details"] = incomplete
	}
	return result
}

func responsesResponseToChatCompletions(in map[string]any) map[string]any {
	var textParts []string
	var reasoningParts []string
	var toolCalls []map[string]any
	for _, rawItem := range slice(in["output"]) {
		item := asMap(rawItem)
		switch stringField(item, "type") {
		case "message":
			if text := contentText(item["content"]); text != "" {
				textParts = append(textParts, text)
			}
		case "function_call":
			callID := stringField(item, "call_id")
			if callID == "" {
				callID = stringField(item, "id")
			}
			toolCalls = append(toolCalls, map[string]any{"id": callID, "type": "function", "function": map[string]any{"name": stringField(item, "name"), "arguments": stringOr(item["arguments"], "{}")}})
		case "reasoning":
			if summary := contentText(item["summary"]); summary != "" {
				reasoningParts = append(reasoningParts, summary)
			}
		}
	}
	message := map[string]any{"role": "assistant", "content": strings.Join(textParts, "\n")}
	if len(toolCalls) > 0 {
		message["tool_calls"] = toolCalls
		if len(textParts) == 0 {
			message["content"] = nil
		}
	}
	if len(reasoningParts) > 0 {
		message["reasoning_content"] = strings.Join(reasoningParts, "\n")
	}
	finish := "stop"
	if len(toolCalls) > 0 {
		finish = "tool_calls"
	} else if stringField(in, "status") == "incomplete" {
		finish = "length"
	}
	created := nowUnix()
	if n := intFromAny(in["created_at"]); n != nil {
		created = int64(*n)
	}
	return map[string]any{
		"id": convertedID(stringField(in, "id"), "chatcmpl", created), "object": "chat.completion", "created": created, "model": stringField(in, "model"),
		"choices": []map[string]any{{"index": 0, "message": message, "finish_reason": finish}},
		"usage":   map[string]any{"prompt_tokens": usageInt(in, "usage", "input_tokens"), "completion_tokens": usageInt(in, "usage", "output_tokens"), "total_tokens": usageInt(in, "usage", "total_tokens")},
	}
}

func geminiResponseToOpenAI(in map[string]any) map[string]any {
	candidate := firstMap(in["candidates"])
	content := asMap(candidate["content"])
	prompt, completion, total := usageInt(in, "usageMetadata", "promptTokenCount"), usageInt(in, "usageMetadata", "candidatesTokenCount"), usageInt(in, "usageMetadata", "totalTokenCount")
	return map[string]any{"id": fmt.Sprintf("chatcmpl-%d", nowUnix()), "object": "chat.completion", "created": nowUnix(), "choices": []map[string]any{{"index": 0, "message": map[string]any{"role": "assistant", "content": partsText(content["parts"])}, "finish_reason": stringOr(candidate["finishReason"], "stop")}}, "usage": map[string]any{"prompt_tokens": prompt, "completion_tokens": completion, "total_tokens": total}}
}

func openAIResponseToGemini(in map[string]any) map[string]any {
	choice := firstMap(in["choices"])
	msg := asMap(choice["message"])
	return geminiResponse(contentText(msg["content"]), stringOr(choice["finish_reason"], "STOP"), usageInt(in, "usage", "prompt_tokens"), usageInt(in, "usage", "completion_tokens"), usageInt(in, "usage", "total_tokens"))
}

func anthropicResponseToGemini(in map[string]any) map[string]any {
	text := ""
	for _, part := range slice(in["content"]) {
		text += stringField(asMap(part), "text")
	}
	prompt, completion := usageInt(in, "usage", "input_tokens"), usageInt(in, "usage", "output_tokens")
	return geminiResponse(text, stringOr(in["stop_reason"], "STOP"), prompt, completion, prompt+completion)
}

func geminiResponseToAnthropic(in map[string]any) map[string]any {
	candidate := firstMap(in["candidates"])
	content := asMap(candidate["content"])
	return map[string]any{"id": fmt.Sprintf("msg_%d", nowUnix()), "type": "message", "role": "assistant", "content": []map[string]any{{"type": "text", "text": partsText(content["parts"])}}, "stop_reason": stringOr(candidate["finishReason"], "end_turn"), "usage": map[string]any{"input_tokens": usageInt(in, "usageMetadata", "promptTokenCount"), "output_tokens": usageInt(in, "usageMetadata", "candidatesTokenCount")}}
}

func geminiResponse(text, finish string, prompt, completion, total int) map[string]any {
	return map[string]any{"candidates": []map[string]any{{"content": map[string]any{"role": "model", "parts": []map[string]any{{"text": text}}}, "finishReason": finish, "index": 0}}, "usageMetadata": map[string]any{"promptTokenCount": prompt, "candidatesTokenCount": completion, "totalTokenCount": total}}
}

func chatMessagesToResponses(value any) []map[string]any {
	var out []map[string]any
	for _, rawMessage := range slice(value) {
		message := asMap(rawMessage)
		role := stringOr(message["role"], "user")
		if role == "tool" {
			out = append(out, map[string]any{"type": "function_call_output", "call_id": stringField(message, "tool_call_id"), "output": contentString(message["content"])})
			continue
		}
		if content := chatContentToResponses(message["content"], role); content != nil {
			out = append(out, map[string]any{"type": "message", "role": role, "content": content})
		}
		for _, rawCall := range slice(message["tool_calls"]) {
			call := asMap(rawCall)
			function := asMap(call["function"])
			out = append(out, map[string]any{
				"type": "function_call", "call_id": stringField(call, "id"), "name": stringField(function, "name"), "arguments": stringOr(function["arguments"], "{}"),
			})
		}
	}
	return out
}

func chatContentToResponses(value any, role string) any {
	if text, ok := value.(string); ok {
		typeName := "input_text"
		if role == "assistant" {
			typeName = "output_text"
		}
		return []map[string]any{{"type": typeName, "text": text}}
	}
	var parts []map[string]any
	for _, rawPart := range slice(value) {
		part := asMap(rawPart)
		switch stringField(part, "type") {
		case "text", "input_text", "output_text":
			typeName := "input_text"
			if role == "assistant" {
				typeName = "output_text"
			}
			parts = append(parts, map[string]any{"type": typeName, "text": stringField(part, "text")})
		case "image_url", "input_image":
			image := asMap(part["image_url"])
			urlValue := stringField(part, "image_url")
			if urlValue == "" {
				urlValue = stringField(image, "url")
			}
			converted := map[string]any{"type": "input_image", "image_url": urlValue}
			if detail := stringField(image, "detail"); detail != "" {
				converted["detail"] = detail
			}
			parts = append(parts, converted)
		case "file", "input_file":
			converted := map[string]any{"type": "input_file"}
			copyFields(converted, part, "file_id", "file_data", "filename")
			parts = append(parts, converted)
		}
	}
	if len(parts) == 0 {
		return nil
	}
	return parts
}

func responsesContentToChat(value any) any {
	if text, ok := value.(string); ok {
		return text
	}
	var parts []map[string]any
	for _, rawPart := range slice(value) {
		part := asMap(rawPart)
		switch stringField(part, "type") {
		case "input_text", "output_text", "text":
			parts = append(parts, map[string]any{"type": "text", "text": stringField(part, "text")})
		case "input_image":
			image := map[string]any{"url": stringField(part, "image_url")}
			if detail := stringField(part, "detail"); detail != "" {
				image["detail"] = detail
			}
			parts = append(parts, map[string]any{"type": "image_url", "image_url": image})
		case "input_file", "file":
			converted := map[string]any{"type": "file"}
			copyFields(converted, part, "file_id", "file_data", "filename")
			parts = append(parts, converted)
		}
	}
	if len(parts) == 0 {
		return nil
	}
	if len(parts) == 1 && stringField(parts[0], "type") == "text" {
		return stringField(parts[0], "text")
	}
	return parts
}

func responsesToolsToChat(value any) []map[string]any {
	var out []map[string]any
	for _, rawTool := range slice(value) {
		tool := asMap(rawTool)
		if stringOr(tool["type"], "function") != "function" {
			out = append(out, tool)
			continue
		}
		function := map[string]any{"name": stringField(tool, "name")}
		copyFields(function, tool, "description", "parameters", "strict")
		out = append(out, map[string]any{"type": "function", "function": function})
	}
	return out
}

func chatToolsToResponses(value any) []map[string]any {
	var out []map[string]any
	for _, rawTool := range slice(value) {
		tool := asMap(rawTool)
		if stringOr(tool["type"], "function") != "function" {
			out = append(out, tool)
			continue
		}
		function := asMap(tool["function"])
		converted := map[string]any{"type": "function", "name": stringField(function, "name")}
		copyFields(converted, function, "description", "parameters", "strict")
		out = append(out, converted)
	}
	return out
}

func responsesToolChoiceToChat(value any) any {
	choice := asMap(value)
	if len(choice) == 0 {
		if value != nil {
			return value
		}
		return nil
	}
	if stringField(choice, "type") == "function" {
		return map[string]any{"type": "function", "function": map[string]any{"name": stringField(choice, "name")}}
	}
	return choice
}

func chatToolChoiceToResponses(value any) any {
	choice := asMap(value)
	if len(choice) == 0 {
		if value != nil {
			return value
		}
		return nil
	}
	if stringField(choice, "type") == "function" {
		function := asMap(choice["function"])
		return map[string]any{"type": "function", "name": stringField(function, "name")}
	}
	return choice
}

func responsesFormatToChat(format map[string]any) map[string]any {
	if stringField(format, "type") != "json_schema" {
		return format
	}
	schema := map[string]any{}
	copyFields(schema, format, "name", "description", "schema", "strict")
	return map[string]any{"type": "json_schema", "json_schema": schema}
}

func chatFormatToResponses(format map[string]any) map[string]any {
	if stringField(format, "type") != "json_schema" {
		return format
	}
	converted := map[string]any{"type": "json_schema"}
	copyFields(converted, asMap(format["json_schema"]), "name", "description", "schema", "strict")
	return converted
}

func copyFields(dst, src map[string]any, names ...string) {
	for _, name := range names {
		if value, ok := src[name]; ok {
			dst[name] = value
		}
	}
}

func contentString(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	if text := contentText(value); text != "" {
		return text
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(encoded)
}

func convertedID(id, prefix string, created int64) string {
	if id == "" {
		return fmt.Sprintf("%s_%d", prefix, created)
	}
	for _, known := range []string{"chatcmpl-", "chatcmpl_", "resp_", "resp-", "msg_", "msg-"} {
		id = strings.TrimPrefix(id, known)
	}
	return prefix + "_" + id
}

func convertedToolID(callID, responseID string, index int) string {
	if callID != "" {
		return convertedID(callID, "fc", nowUnix())
	}
	return fmt.Sprintf("fc_%s_%d", strings.TrimPrefix(responseID, "resp_"), index)
}

func contentText(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case []any:
		var parts []string
		for _, p := range x {
			m := asMap(p)
			if s := stringField(m, "text"); s != "" {
				parts = append(parts, s)
			} else if s := stringField(m, "content"); s != "" {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, "\n")
	case map[string]any:
		if s := stringField(x, "text"); s != "" {
			return s
		}
	}
	return ""
}

func partsText(v any) string {
	var parts []string
	for _, item := range slice(v) {
		if s := stringField(asMap(item), "text"); s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, "\n")
}

func asMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

func slice(v any) []any {
	if s, ok := v.([]any); ok {
		return s
	}
	return nil
}

func firstMap(v any) map[string]any {
	items := slice(v)
	if len(items) == 0 {
		return map[string]any{}
	}
	return asMap(items[0])
}

func stringField(m map[string]any, key string) string {
	if s, ok := m[key].(string); ok {
		return s
	}
	return ""
}

func boolField(m map[string]any, key string) bool {
	if b, ok := m[key].(bool); ok {
		return b
	}
	return false
}

func stringOr(v any, fallback string) string {
	if s, ok := v.(string); ok && s != "" {
		return s
	}
	return fallback
}

func intOr(v any, fallback int) int {
	if n := intFromAny(v); n != nil {
		return *n
	}
	return fallback
}

func intFromAny(values ...any) *int {
	for _, v := range values {
		switch x := v.(type) {
		case float64:
			n := int(x)
			return &n
		case int:
			n := x
			return &n
		case json.Number:
			i, _ := x.Int64()
			n := int(i)
			return &n
		}
	}
	return nil
}

func usageInt(m map[string]any, usageKey, field string) int {
	if u := asMap(m[usageKey]); len(u) > 0 {
		return intOr(u[field], 0)
	}
	return 0
}

func nowUnix() int64 {
	return time.Now().Unix()
}
