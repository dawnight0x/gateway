package protocol

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"local-ai-gateway/internal/router"
)

type StreamFrame struct {
	Event string
	Data  []byte
}

type StreamState struct {
	ID                   string
	Model                string
	Created              int64
	Started              bool
	ContentOpen          bool
	FinishSent           bool
	Finished             bool
	Sequence             int
	Text                 strings.Builder
	Reasoning            strings.Builder
	Usage                Usage
	MessageAdded         bool
	TextOutputIndex      int
	ReasoningAdded       bool
	ReasoningOutputIndex int
	NextOutputIndex      int
	Tools                map[int]*streamToolState
	DisableAggregate     bool
	MaxAggregateBytes    int
	MaxToolArgumentBytes int
	aggregateBytes       int
	ResourceIDs          []string
	UpstreamError        error
}

type upstreamStreamError struct {
	protocol string
	message  string
}

func (e *upstreamStreamError) Error() string {
	return e.protocol + " stream error: " + e.message
}

func newUpstreamStreamError(protocolName, message string) error {
	return &upstreamStreamError{protocol: protocolName, message: stringOr(message, "upstream error")}
}

func IsUpstreamStreamError(err error) bool {
	var target *upstreamStreamError
	return errors.As(err, &target)
}

type streamToolState struct {
	ID          string
	CallID      string
	Name        string
	Arguments   strings.Builder
	OutputIndex int
	ChatIndex   int
	Added       bool
	Done        bool
}

type normalizedToolCall struct {
	Index     int
	ID        string
	CallID    string
	Name      string
	Arguments string
	Done      bool
}

type normalizedStreamEvent struct {
	ID          string
	Model       string
	Text        string
	Reasoning   string
	ToolCalls   []normalizedToolCall
	Finish      string
	Usage       Usage
	Done        bool
	ResourceIDs []string
}

func EnableStreaming(req ConvertedRequest, upstream string) (ConvertedRequest, error) {
	req.Stream = true
	if upstream == router.ProtocolGemini {
		return req, nil
	}
	var raw map[string]any
	if err := json.Unmarshal(req.Body, &raw); err != nil {
		return req, err
	}
	raw["stream"] = true
	out, err := json.Marshal(raw)
	if err != nil {
		return req, err
	}
	req.Body = out
	return req, nil
}

func ConvertJSONResponseToStream(body []byte, inbound, upstream string) ([]StreamFrame, Usage, []string, error) {
	converted, err := ConvertResponseChecked(body, inbound, upstream)
	if err != nil {
		return nil, Usage{}, nil, err
	}
	var raw map[string]any
	if err := json.Unmarshal(converted, &raw); err != nil {
		return nil, Usage{}, nil, fmt.Errorf("decode upstream JSON stream fallback: %w", err)
	}
	state := &StreamState{ID: stringField(raw, "id"), Model: stringField(raw, "model"), Created: nowUnix(), Usage: ExtractUsage(converted), ResourceIDs: ResponseResourceIDs(converted)}
	if n := intFromAny(raw["created"]); n != nil {
		state.Created = int64(*n)
	} else if n := intFromAny(raw["created_at"]); n != nil {
		state.Created = int64(*n)
	}
	var events []normalizedStreamEvent
	switch inbound {
	case router.ProtocolOpenAI:
		choice := firstMap(raw["choices"])
		message := asMap(choice["message"])
		var tools []normalizedToolCall
		for index, rawCall := range slice(message["tool_calls"]) {
			call := asMap(rawCall)
			function := asMap(call["function"])
			tools = append(tools, normalizedToolCall{Index: index, ID: stringField(call, "id"), CallID: stringField(call, "id"), Name: stringField(function, "name"), Arguments: stringField(function, "arguments"), Done: true})
		}
		events = append(events, normalizedStreamEvent{Text: contentText(message["content"]), Reasoning: stringOr(message["reasoning_content"], stringField(message, "reasoning")), ToolCalls: tools})
		finish := stringOr(choice["finish_reason"], "stop")
		events = append(events, normalizedStreamEvent{Finish: finish, Usage: state.Usage, Done: true})
	case router.ProtocolOpenAIResponses:
		for _, rawItem := range slice(raw["output"]) {
			item := asMap(rawItem)
			switch stringField(item, "type") {
			case "message":
				events = append(events, normalizedStreamEvent{Text: contentText(item["content"])})
			case "reasoning":
				events = append(events, normalizedStreamEvent{Reasoning: contentText(item["summary"])})
			case "function_call":
				index := len(events)
				events = append(events, normalizedStreamEvent{ToolCalls: []normalizedToolCall{{Index: index, ID: stringField(item, "id"), CallID: stringField(item, "call_id"), Name: stringField(item, "name"), Arguments: stringField(item, "arguments"), Done: true}}})
			}
		}
		finish := "stop"
		if stringField(raw, "status") == "incomplete" {
			finish = "length"
		}
		events = append(events, normalizedStreamEvent{Finish: finish, Usage: state.Usage, Done: true})
	default:
		return nil, Usage{}, nil, fmt.Errorf("JSON stream fallback is unsupported for inbound protocol %q", inbound)
	}
	var frames []StreamFrame
	for _, event := range events {
		if err := updateStreamState(state, event); err != nil {
			return nil, Usage{}, nil, err
		}
		var rendered []StreamFrame
		var err error
		if inbound == router.ProtocolOpenAI {
			rendered, err = renderOpenAIStream(event, state)
		} else {
			rendered, err = renderResponsesStream(event, state)
		}
		if err != nil {
			return nil, Usage{}, nil, err
		}
		frames = append(frames, rendered...)
	}
	return frames, state.Usage, state.ResourceIDs, nil
}

func ConvertStreamEvent(event string, data []byte, inbound, upstream string, state *StreamState) ([]StreamFrame, error) {
	normalized, err := normalizeStreamEvent(event, data, upstream)
	if inbound == upstream {
		if err == nil {
			if err := updateStreamState(state, normalized); err != nil {
				return nil, err
			}
		} else if IsUpstreamStreamError(err) {
			state.UpstreamError = err
		}
		return []StreamFrame{{Event: event, Data: append([]byte(nil), data...)}}, nil
	}
	if err != nil {
		return nil, err
	}
	if err := updateStreamState(state, normalized); err != nil {
		return nil, err
	}
	switch inbound {
	case router.ProtocolOpenAI:
		return renderOpenAIStream(normalized, state)
	case router.ProtocolOpenAIResponses:
		return renderResponsesStream(normalized, state)
	case router.ProtocolAnthropic:
		return renderAnthropicStream(normalized, state)
	case router.ProtocolGemini:
		return renderGeminiStream(normalized, state)
	default:
		return nil, fmt.Errorf("unsupported inbound stream protocol %q", inbound)
	}
}

func normalizeStreamEvent(event string, data []byte, upstream string) (normalizedStreamEvent, error) {
	if strings.TrimSpace(string(data)) == "[DONE]" {
		return normalizedStreamEvent{Done: true}, nil
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return normalizedStreamEvent{}, fmt.Errorf("decode %s stream event: %w", upstream, err)
	}
	switch upstream {
	case router.ProtocolOpenAI:
		if raw["error"] != nil {
			return normalizedStreamEvent{}, newUpstreamStreamError("openai", streamErrorMessage(raw))
		}
		choice := firstMap(raw["choices"])
		delta := asMap(choice["delta"])
		text := contentText(delta["content"])
		if text == "" {
			text = contentText(choice["text"])
		}
		var toolCalls []normalizedToolCall
		for _, rawCall := range slice(delta["tool_calls"]) {
			call := asMap(rawCall)
			function := asMap(call["function"])
			index := intOr(call["index"], len(toolCalls))
			toolCalls = append(toolCalls, normalizedToolCall{Index: index, ID: stringField(call, "id"), CallID: stringField(call, "id"), Name: stringField(function, "name"), Arguments: stringField(function, "arguments")})
		}
		if legacy := asMap(delta["function_call"]); len(legacy) > 0 {
			toolCalls = append(toolCalls, normalizedToolCall{Index: 0, Name: stringField(legacy, "name"), Arguments: stringField(legacy, "arguments")})
		}
		return normalizedStreamEvent{
			ID:        stringField(raw, "id"),
			Model:     stringField(raw, "model"),
			Text:      text,
			Reasoning: stringOr(delta["reasoning_content"], stringField(delta, "reasoning")),
			ToolCalls: toolCalls,
			Finish:    stringField(choice, "finish_reason"),
			Usage:     ExtractUsage(data),
		}, nil
	case router.ProtocolOpenAIResponses:
		typeName := stringField(raw, "type")
		if typeName == "" {
			typeName = event
		}
		switch typeName {
		case "response.created", "response.in_progress":
			response := asMap(raw["response"])
			return normalizedStreamEvent{ID: stringField(response, "id"), Model: stringField(response, "model"), Usage: responsesUsageFromMap(asMap(response["usage"])), ResourceIDs: responseResourceIDsMap(response)}, nil
		case "response.output_text.delta":
			return normalizedStreamEvent{Text: stringField(raw, "delta")}, nil
		case "response.reasoning_summary_text.delta":
			return normalizedStreamEvent{Reasoning: stringField(raw, "delta")}, nil
		case "response.output_item.added", "response.output_item.done":
			item := asMap(raw["item"])
			if stringField(item, "type") != "function_call" {
				return normalizedStreamEvent{}, nil
			}
			return normalizedStreamEvent{ToolCalls: []normalizedToolCall{{Index: intOr(raw["output_index"], 0), ID: stringField(item, "id"), CallID: stringField(item, "call_id"), Name: stringField(item, "name"), Arguments: stringField(item, "arguments"), Done: typeName == "response.output_item.done"}}}, nil
		case "response.function_call_arguments.delta":
			return normalizedStreamEvent{ToolCalls: []normalizedToolCall{{Index: intOr(raw["output_index"], 0), ID: stringField(raw, "item_id"), Arguments: stringField(raw, "delta")}}}, nil
		case "response.completed", "response.incomplete":
			response := asMap(raw["response"])
			finish := "stop"
			if typeName == "response.incomplete" || stringField(response, "status") == "incomplete" {
				finish = "length"
			}
			return normalizedStreamEvent{ID: stringField(response, "id"), Model: stringField(response, "model"), Finish: finish, Usage: responsesUsageFromMap(asMap(response["usage"])), Done: true, ResourceIDs: responseResourceIDsMap(response)}, nil
		case "response.failed", "error":
			return normalizedStreamEvent{}, newUpstreamStreamError("responses", streamErrorMessage(raw))
		default:
			return normalizedStreamEvent{}, nil
		}
	case router.ProtocolAnthropic:
		typeName := stringField(raw, "type")
		if typeName == "" {
			typeName = event
		}
		switch typeName {
		case "message_start":
			message := asMap(raw["message"])
			return normalizedStreamEvent{
				ID:    stringField(message, "id"),
				Model: stringField(message, "model"),
				Usage: usageFromMap(asMap(message["usage"]), "input_tokens", "output_tokens", "total_tokens"),
			}, nil
		case "content_block_start":
			block := asMap(raw["content_block"])
			return normalizedStreamEvent{Text: stringField(block, "text")}, nil
		case "content_block_delta":
			delta := asMap(raw["delta"])
			return normalizedStreamEvent{Text: stringField(delta, "text")}, nil
		case "message_delta":
			delta := asMap(raw["delta"])
			return normalizedStreamEvent{
				Finish: stringField(delta, "stop_reason"),
				Usage:  usageFromMap(asMap(raw["usage"]), "input_tokens", "output_tokens", "total_tokens"),
			}, nil
		case "message_stop":
			return normalizedStreamEvent{Done: true}, nil
		case "ping", "content_block_stop":
			return normalizedStreamEvent{}, nil
		case "error":
			return normalizedStreamEvent{}, newUpstreamStreamError("anthropic", streamErrorMessage(raw))
		default:
			return normalizedStreamEvent{}, nil
		}
	case router.ProtocolGemini:
		if errBody := asMap(raw["error"]); len(errBody) > 0 {
			return normalizedStreamEvent{}, newUpstreamStreamError("gemini", streamErrorMessage(raw))
		}
		candidate := firstMap(raw["candidates"])
		content := asMap(candidate["content"])
		finish := stringField(candidate, "finishReason")
		return normalizedStreamEvent{
			Model:  stringField(raw, "modelVersion"),
			Text:   partsText(content["parts"]),
			Finish: finish,
			Usage:  ExtractUsage(data),
			Done:   finish != "",
		}, nil
	default:
		return normalizedStreamEvent{}, fmt.Errorf("unsupported upstream stream protocol %q", upstream)
	}
}

func streamErrorMessage(raw map[string]any) string {
	for _, body := range []map[string]any{asMap(raw["error"]), asMap(asMap(raw["response"])["error"])} {
		if message := stringField(body, "message"); message != "" {
			return message
		}
	}
	if message, ok := raw["error"].(string); ok && message != "" {
		return message
	}
	return "upstream error"
}

func updateStreamState(state *StreamState, event normalizedStreamEvent) error {
	if event.ID != "" {
		state.ID = event.ID
	}
	if event.Model != "" {
		state.Model = event.Model
	}
	for _, id := range event.ResourceIDs {
		if id != "" && !containsString(state.ResourceIDs, id) {
			state.ResourceIDs = append(state.ResourceIDs, id)
		}
	}
	if state.Created == 0 {
		state.Created = nowUnix()
	}
	if !state.DisableAggregate {
		additional := len(event.Text) + len(event.Reasoning)
		if state.MaxAggregateBytes > 0 && state.aggregateBytes+additional > state.MaxAggregateBytes {
			return fmt.Errorf("stream aggregate exceeds %d MiB limit", state.MaxAggregateBytes>>20)
		}
		state.aggregateBytes += additional
		if event.Text != "" {
			state.Text.WriteString(event.Text)
		}
		if event.Reasoning != "" {
			state.Reasoning.WriteString(event.Reasoning)
		}
	}
	mergeUsage(&state.Usage, event.Usage)
	return nil
}

func responseResourceIDsMap(raw map[string]any) []string {
	ids := []string{}
	if id := stringField(raw, "id"); id != "" {
		ids = append(ids, id)
	}
	if conversation, ok := raw["conversation"].(string); ok && conversation != "" {
		ids = append(ids, conversation)
	} else if id := stringField(asMap(raw["conversation"]), "id"); id != "" {
		ids = append(ids, id)
	}
	return ids
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func renderOpenAIStream(event normalizedStreamEvent, state *StreamState) ([]StreamFrame, error) {
	var frames []StreamFrame
	delta := map[string]any{}
	if !state.Started {
		delta["role"] = "assistant"
		state.Started = true
	}
	if event.Text != "" {
		delta["content"] = event.Text
	}
	if event.Reasoning != "" {
		delta["reasoning_content"] = event.Reasoning
	}
	var toolDeltas []map[string]any
	for _, call := range event.ToolCalls {
		tool := ensureStreamTool(state, call.Index)
		if call.ID != "" {
			tool.ID = call.ID
		}
		if call.CallID != "" {
			tool.CallID = call.CallID
		}
		if call.Name != "" {
			tool.Name = call.Name
		}
		function := map[string]any{}
		if call.Name != "" && !tool.Added {
			function["name"] = call.Name
		}
		arguments, err := appendToolArguments(state, tool, call.Arguments, call.Done)
		if err != nil {
			return nil, err
		}
		if arguments != "" {
			function["arguments"] = arguments
		}
		converted := map[string]any{"index": tool.ChatIndex}
		if !tool.Added {
			id := tool.CallID
			if id == "" {
				id = tool.ID
			}
			if id != "" {
				converted["id"] = id
			}
			converted["type"] = "function"
			tool.Added = true
		}
		if len(function) > 0 {
			converted["function"] = function
		}
		if len(converted) > 1 {
			toolDeltas = append(toolDeltas, converted)
		}
	}
	if len(toolDeltas) > 0 {
		delta["tool_calls"] = toolDeltas
	}
	finish := openAIFinishReason(event.Finish)
	if finish == "stop" && len(state.Tools) > 0 {
		finish = "tool_calls"
	}
	if len(delta) > 0 || finish != "" || hasUsage(event.Usage) {
		choice := map[string]any{"index": 0, "delta": delta, "finish_reason": nil}
		if finish != "" {
			choice["finish_reason"] = finish
		}
		payload := map[string]any{
			"id":      streamID(state, "chatcmpl"),
			"object":  "chat.completion.chunk",
			"created": state.Created,
			"model":   state.Model,
			"choices": []map[string]any{choice},
		}
		if hasUsage(event.Usage) {
			payload["usage"] = openAIUsage(state.Usage)
		}
		frames = append(frames, mustJSONFrame("", payload))
		if finish != "" {
			state.FinishSent = true
		}
	}
	if event.Done && !state.Finished {
		if finish == "" && !state.FinishSent {
			finish = "stop"
			if len(state.Tools) > 0 {
				finish = "tool_calls"
			}
			payload := map[string]any{
				"id": streamID(state, "chatcmpl"), "object": "chat.completion.chunk", "created": state.Created, "model": state.Model,
				"choices": []map[string]any{{"index": 0, "delta": map[string]any{}, "finish_reason": finish}},
			}
			if hasUsage(state.Usage) {
				payload["usage"] = openAIUsage(state.Usage)
			}
			frames = append(frames, mustJSONFrame("", payload))
		}
		state.Finished = true
		frames = append(frames, StreamFrame{Data: []byte("[DONE]")})
	}
	return frames, nil
}

func renderAnthropicStream(event normalizedStreamEvent, state *StreamState) ([]StreamFrame, error) {
	var frames []StreamFrame
	if !state.Started && (event.Text != "" || event.Finish != "" || event.Done || hasUsage(event.Usage) || state.ID != "" || state.Model != "") {
		state.Started = true
		frames = append(frames,
			mustJSONFrame("message_start", map[string]any{
				"type": "message_start",
				"message": map[string]any{
					"id": streamID(state, "msg"), "type": "message", "role": "assistant", "model": state.Model,
					"content": []any{}, "stop_reason": nil, "stop_sequence": nil,
					"usage": map[string]any{"input_tokens": usageValue(state.Usage.PromptTokens), "output_tokens": 0},
				},
			}),
			mustJSONFrame("content_block_start", map[string]any{
				"type": "content_block_start", "index": 0,
				"content_block": map[string]any{"type": "text", "text": ""},
			}),
		)
		state.ContentOpen = true
	}
	if event.Text != "" {
		frames = append(frames, mustJSONFrame("content_block_delta", map[string]any{
			"type": "content_block_delta", "index": 0,
			"delta": map[string]any{"type": "text_delta", "text": event.Text},
		}))
	}
	if event.Finish != "" || hasUsage(event.Usage) {
		frames = append(frames, mustJSONFrame("message_delta", map[string]any{
			"type":  "message_delta",
			"delta": map[string]any{"stop_reason": anthropicFinishReason(event.Finish), "stop_sequence": nil},
			"usage": map[string]any{"output_tokens": usageValue(state.Usage.CompletionTokens)},
		}))
		state.FinishSent = true
	}
	if event.Done && !state.Finished {
		if !state.Started {
			startFrames, _ := renderAnthropicStream(normalizedStreamEvent{Finish: "stop"}, state)
			frames = append(frames, startFrames...)
		}
		if !state.FinishSent {
			frames = append(frames, mustJSONFrame("message_delta", map[string]any{
				"type":  "message_delta",
				"delta": map[string]any{"stop_reason": "end_turn", "stop_sequence": nil},
				"usage": map[string]any{"output_tokens": usageValue(state.Usage.CompletionTokens)},
			}))
		}
		if state.ContentOpen {
			frames = append(frames, mustJSONFrame("content_block_stop", map[string]any{"type": "content_block_stop", "index": 0}))
		}
		frames = append(frames, mustJSONFrame("message_stop", map[string]any{"type": "message_stop"}))
		state.Finished = true
	}
	return frames, nil
}

func renderGeminiStream(event normalizedStreamEvent, state *StreamState) ([]StreamFrame, error) {
	if event.Text == "" && event.Finish == "" && !hasUsage(event.Usage) {
		return nil, nil
	}
	candidate := map[string]any{"index": 0}
	if event.Text != "" {
		candidate["content"] = map[string]any{"role": "model", "parts": []map[string]any{{"text": event.Text}}}
	}
	if event.Finish != "" {
		candidate["finishReason"] = geminiFinishReason(event.Finish)
	}
	payload := map[string]any{"candidates": []map[string]any{candidate}}
	if hasUsage(event.Usage) {
		payload["usageMetadata"] = geminiUsage(state.Usage)
	}
	return []StreamFrame{mustJSONFrame("", payload)}, nil
}

func renderResponsesStream(event normalizedStreamEvent, state *StreamState) ([]StreamFrame, error) {
	var frames []StreamFrame
	responseID := streamID(state, "resp")
	itemID := streamID(state, "msg")
	if !state.Started {
		state.Started = true
		frames = append(frames,
			responsesFrame(state, "response.created", map[string]any{"response": responsesObject(state, responseID, itemID, "in_progress", false)}),
			responsesFrame(state, "response.in_progress", map[string]any{"response": responsesObject(state, responseID, itemID, "in_progress", false)}),
		)
	}
	if event.Reasoning != "" {
		reasoningID := streamID(state, "rs")
		if !state.ReasoningAdded {
			state.ReasoningAdded = true
			state.ReasoningOutputIndex = state.NextOutputIndex
			state.NextOutputIndex++
			frames = append(frames,
				responsesFrame(state, "response.output_item.added", map[string]any{"output_index": state.ReasoningOutputIndex, "item": map[string]any{"id": reasoningID, "type": "reasoning", "status": "in_progress", "summary": []any{}}}),
				responsesFrame(state, "response.reasoning_summary_part.added", map[string]any{"item_id": reasoningID, "output_index": state.ReasoningOutputIndex, "summary_index": 0, "part": map[string]any{"type": "summary_text", "text": ""}}),
			)
		}
		frames = append(frames, responsesFrame(state, "response.reasoning_summary_text.delta", map[string]any{"item_id": reasoningID, "output_index": state.ReasoningOutputIndex, "summary_index": 0, "delta": event.Reasoning}))
	}
	if event.Text != "" {
		if !state.MessageAdded {
			state.MessageAdded = true
			state.TextOutputIndex = state.NextOutputIndex
			state.NextOutputIndex++
			state.ContentOpen = true
			frames = append(frames,
				responsesFrame(state, "response.output_item.added", map[string]any{"output_index": state.TextOutputIndex, "item": map[string]any{"id": itemID, "type": "message", "status": "in_progress", "role": "assistant", "content": []any{}}}),
				responsesFrame(state, "response.content_part.added", map[string]any{"item_id": itemID, "output_index": state.TextOutputIndex, "content_index": 0, "part": map[string]any{"type": "output_text", "text": "", "annotations": []any{}}}),
			)
		}
		frames = append(frames, responsesFrame(state, "response.output_text.delta", map[string]any{
			"item_id": itemID, "output_index": state.TextOutputIndex, "content_index": 0, "delta": event.Text,
		}))
	}
	for _, call := range event.ToolCalls {
		tool := ensureStreamTool(state, call.Index)
		if call.ID != "" {
			tool.ID = call.ID
		}
		if call.CallID != "" {
			tool.CallID = call.CallID
		}
		if call.Name != "" {
			tool.Name = call.Name
		}
		if !tool.Added {
			tool.Added = true
			tool.OutputIndex = state.NextOutputIndex
			state.NextOutputIndex++
			if tool.ID == "" {
				tool.ID = fmt.Sprintf("fc_%d_%d", state.Created, call.Index)
			}
			if tool.CallID == "" {
				tool.CallID = fmt.Sprintf("call_%d_%d", state.Created, call.Index)
			}
			frames = append(frames, responsesFrame(state, "response.output_item.added", map[string]any{"output_index": tool.OutputIndex, "item": responseToolItem(tool, "in_progress")}))
		}
		arguments, err := appendToolArguments(state, tool, call.Arguments, call.Done)
		if err != nil {
			return nil, err
		}
		if arguments != "" {
			frames = append(frames, responsesFrame(state, "response.function_call_arguments.delta", map[string]any{"item_id": tool.ID, "output_index": tool.OutputIndex, "delta": arguments}))
		}
	}
	if (event.Finish != "" || event.Done) && !state.FinishSent {
		state.FinishSent = true
		frames = append(frames, finishResponsesOutput(state, itemID)...)
	}
	if event.Done && !state.Finished {
		frames = append(frames, responsesFrame(state, "response.completed", map[string]any{"response": responsesObject(state, responseID, itemID, "completed", true)}))
		state.Finished = true
	}
	return frames, nil
}

func responsesFrame(state *StreamState, event string, fields map[string]any) StreamFrame {
	payload := map[string]any{"type": event, "sequence_number": state.Sequence}
	state.Sequence++
	for key, value := range fields {
		payload[key] = value
	}
	return mustJSONFrame(event, payload)
}

func responsesObject(state *StreamState, responseID, itemID, status string, completed bool) map[string]any {
	output := []any{}
	if completed {
		for outputIndex := 0; outputIndex < state.NextOutputIndex; outputIndex++ {
			if state.ReasoningAdded && state.ReasoningOutputIndex == outputIndex {
				output = append(output, map[string]any{"id": streamID(state, "rs"), "type": "reasoning", "status": "completed", "summary": []map[string]any{{"type": "summary_text", "text": state.Reasoning.String()}}})
			}
			if state.MessageAdded && state.TextOutputIndex == outputIndex {
				output = append(output, map[string]any{"id": itemID, "type": "message", "status": "completed", "role": "assistant", "content": []map[string]any{{"type": "output_text", "text": state.Text.String(), "annotations": []any{}}}})
			}
			for _, tool := range state.Tools {
				if tool.OutputIndex == outputIndex {
					output = append(output, responseToolItem(tool, "completed"))
				}
			}
		}
	}
	return map[string]any{
		"id": responseID, "object": "response", "created_at": state.Created, "status": status, "model": state.Model,
		"output": output, "output_text": state.Text.String(), "usage": responsesUsage(state.Usage),
	}
}

func finishResponsesOutput(state *StreamState, itemID string) []StreamFrame {
	var frames []StreamFrame
	if state.ReasoningAdded {
		reasoningID := streamID(state, "rs")
		part := map[string]any{"type": "summary_text", "text": state.Reasoning.String()}
		item := map[string]any{"id": reasoningID, "type": "reasoning", "status": "completed", "summary": []map[string]any{part}}
		frames = append(frames,
			responsesFrame(state, "response.reasoning_summary_text.done", map[string]any{"item_id": reasoningID, "output_index": state.ReasoningOutputIndex, "summary_index": 0, "text": state.Reasoning.String()}),
			responsesFrame(state, "response.reasoning_summary_part.done", map[string]any{"item_id": reasoningID, "output_index": state.ReasoningOutputIndex, "summary_index": 0, "part": part}),
			responsesFrame(state, "response.output_item.done", map[string]any{"output_index": state.ReasoningOutputIndex, "item": item}),
		)
	}
	if state.MessageAdded {
		part := map[string]any{"type": "output_text", "text": state.Text.String(), "annotations": []any{}}
		item := map[string]any{"id": itemID, "type": "message", "status": "completed", "role": "assistant", "content": []map[string]any{part}}
		frames = append(frames,
			responsesFrame(state, "response.output_text.done", map[string]any{"item_id": itemID, "output_index": state.TextOutputIndex, "content_index": 0, "text": state.Text.String()}),
			responsesFrame(state, "response.content_part.done", map[string]any{"item_id": itemID, "output_index": state.TextOutputIndex, "content_index": 0, "part": part}),
			responsesFrame(state, "response.output_item.done", map[string]any{"output_index": state.TextOutputIndex, "item": item}),
		)
		state.ContentOpen = false
	}
	for _, tool := range state.Tools {
		if !tool.Added || tool.Done {
			continue
		}
		tool.Done = true
		frames = append(frames,
			responsesFrame(state, "response.function_call_arguments.done", map[string]any{"item_id": tool.ID, "output_index": tool.OutputIndex, "arguments": tool.Arguments.String()}),
			responsesFrame(state, "response.output_item.done", map[string]any{"output_index": tool.OutputIndex, "item": responseToolItem(tool, "completed")}),
		)
	}
	return frames
}

func mustJSONFrame(event string, payload any) StreamFrame {
	data, _ := json.Marshal(payload)
	return StreamFrame{Event: event, Data: data}
}

func streamID(state *StreamState, prefix string) string {
	if state.ID == "" {
		return fmt.Sprintf("%s_%d", prefix, state.Created)
	}
	if strings.HasPrefix(state.ID, prefix+"_") || strings.HasPrefix(state.ID, prefix+"-") {
		return state.ID
	}
	base := state.ID
	for _, known := range []string{"chatcmpl-", "chatcmpl_", "resp_", "msg_"} {
		base = strings.TrimPrefix(base, known)
	}
	return prefix + "_" + base
}

func mergeUsage(dst *Usage, src Usage) {
	if src.PromptTokens != nil {
		dst.PromptTokens = src.PromptTokens
	}
	if src.CompletionTokens != nil {
		dst.CompletionTokens = src.CompletionTokens
	}
	if src.TotalTokens != nil {
		dst.TotalTokens = src.TotalTokens
	}
	if dst.TotalTokens == nil && dst.PromptTokens != nil && dst.CompletionTokens != nil {
		total := *dst.PromptTokens + *dst.CompletionTokens
		dst.TotalTokens = &total
	}
}

func ensureStreamTool(state *StreamState, index int) *streamToolState {
	if state.Tools == nil {
		state.Tools = map[int]*streamToolState{}
	}
	if tool := state.Tools[index]; tool != nil {
		return tool
	}
	tool := &streamToolState{ChatIndex: len(state.Tools)}
	state.Tools[index] = tool
	return tool
}

func appendToolArguments(state *StreamState, tool *streamToolState, value string, complete bool) (string, error) {
	if value == "" {
		return "", nil
	}
	existing := tool.Arguments.String()
	delta := value
	if complete && existing != "" {
		if value == existing {
			return "", nil
		}
		if strings.HasPrefix(value, existing) {
			delta = strings.TrimPrefix(value, existing)
		}
	}
	if state.MaxToolArgumentBytes > 0 && tool.Arguments.Len()+len(delta) > state.MaxToolArgumentBytes {
		return "", fmt.Errorf("stream tool arguments exceed %d MiB per-call limit", state.MaxToolArgumentBytes>>20)
	}
	if state.MaxAggregateBytes > 0 && state.aggregateBytes+len(delta) > state.MaxAggregateBytes {
		return "", fmt.Errorf("stream aggregate exceeds %d MiB limit", state.MaxAggregateBytes>>20)
	}
	state.aggregateBytes += len(delta)
	tool.Arguments.WriteString(delta)
	return delta, nil
}

func responseToolItem(tool *streamToolState, status string) map[string]any {
	return map[string]any{
		"id": tool.ID, "type": "function_call", "status": status, "call_id": tool.CallID,
		"name": tool.Name, "arguments": tool.Arguments.String(),
	}
}

func responsesUsageFromMap(raw map[string]any) Usage {
	return usageFromMap(raw, "input_tokens", "output_tokens", "total_tokens")
}

func usageFromMap(raw map[string]any, promptKey, completionKey, totalKey string) Usage {
	return Usage{
		PromptTokens:     intFromAny(raw[promptKey]),
		CompletionTokens: intFromAny(raw[completionKey]),
		TotalTokens:      intFromAny(raw[totalKey]),
	}
}

func hasUsage(usage Usage) bool {
	return usage.PromptTokens != nil || usage.CompletionTokens != nil || usage.TotalTokens != nil
}

func usageValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func openAIUsage(usage Usage) map[string]any {
	return map[string]any{
		"prompt_tokens": usageValue(usage.PromptTokens), "completion_tokens": usageValue(usage.CompletionTokens), "total_tokens": usageValue(usage.TotalTokens),
	}
}

func geminiUsage(usage Usage) map[string]any {
	return map[string]any{
		"promptTokenCount": usageValue(usage.PromptTokens), "candidatesTokenCount": usageValue(usage.CompletionTokens), "totalTokenCount": usageValue(usage.TotalTokens),
	}
}

func responsesUsage(usage Usage) map[string]any {
	return map[string]any{
		"input_tokens": usageValue(usage.PromptTokens), "output_tokens": usageValue(usage.CompletionTokens), "total_tokens": usageValue(usage.TotalTokens),
	}
}

func openAIFinishReason(reason string) string {
	switch strings.ToLower(reason) {
	case "", "none":
		return ""
	case "max_tokens", "max_token", "max_tokens_exceeded", "length":
		return "length"
	case "tool_use", "tool_calls":
		return "tool_calls"
	default:
		return "stop"
	}
}

func anthropicFinishReason(reason string) any {
	switch openAIFinishReason(reason) {
	case "":
		return nil
	case "length":
		return "max_tokens"
	case "tool_calls":
		return "tool_use"
	default:
		return "end_turn"
	}
}

func geminiFinishReason(reason string) string {
	switch openAIFinishReason(reason) {
	case "length":
		return "MAX_TOKENS"
	case "tool_calls":
		return "STOP"
	default:
		return "STOP"
	}
}
