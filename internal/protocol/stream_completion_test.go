package protocol

import (
	"strings"
	"testing"

	"local-ai-gateway/internal/router"
)

func TestNativeStreamTracksProtocolTerminalEvents(t *testing.T) {
	tests := []struct{ name, protocol, partial, terminal string }{
		{"chat", router.ProtocolOpenAI, `{"choices":[{"index":0,"delta":{"content":"partial"},"finish_reason":null}]}`, `[DONE]`},
		{"responses", router.ProtocolOpenAIResponses, `{"type":"response.output_text.delta","delta":"partial"}`, `{"type":"response.completed","response":{"status":"completed"}}`},
		{"anthropic", router.ProtocolAnthropic, `{"type":"content_block_delta","delta":{"type":"text_delta","text":"partial"}}`, `{"type":"message_stop"}`},
		{"gemini", router.ProtocolGemini, `{"candidates":[{"content":{"parts":[{"text":"partial"}]}}]}`, `{"candidates":[{"finishReason":"STOP"}]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := &StreamState{DisableAggregate: true}
			if _, err := ConvertStreamEvent("", []byte(tt.partial), tt.protocol, tt.protocol, state); err != nil {
				t.Fatal(err)
			}
			if state.UpstreamFinished {
				t.Fatal("partial delta marked as upstream completion")
			}
			if _, err := FinalizeStream(router.ProtocolOpenAIResponses, state); err == nil {
				t.Fatal("partial stream finalized")
			}
			if _, err := ConvertStreamEvent("", []byte(tt.terminal), tt.protocol, tt.protocol, state); err != nil {
				t.Fatal(err)
			}
			if !state.UpstreamFinished {
				t.Fatal("terminal event not tracked in native stream")
			}
		})
	}
}

func TestChatCompletionWaitsForAllObservedChoices(t *testing.T) {
	state := &StreamState{}
	for _, data := range []string{
		`{"choices":[{"index":0,"delta":{"content":"one"}},{"index":1,"delta":{"content":"two"}}]}`,
		`{"choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
	} {
		_, _ = ConvertStreamEvent("", []byte(data), router.ProtocolOpenAI, router.ProtocolOpenAI, state)
	}
	if state.UpstreamFinished {
		t.Fatal("unfinished second choice ignored")
	}
	_, _ = ConvertStreamEvent("", []byte(`{"choices":[{"index":1,"delta":{},"finish_reason":"stop"}]}`), router.ProtocolOpenAI, router.ProtocolOpenAI, state)
	if !state.UpstreamFinished {
		t.Fatal("all finished choices not recognized")
	}
}

func TestChatFinishReasonDoesNotHideMissingRequestedChoice(t *testing.T) {
	state := &StreamState{ExpectedChoices: 2}
	_, _ = ConvertStreamEvent("", []byte(`{"choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`), router.ProtocolOpenAI, router.ProtocolOpenAI, state)
	if state.UpstreamFinished {
		t.Fatal("missing second requested choice marked as completed")
	}
	_, _ = ConvertStreamEvent("", []byte(`{"choices":[{"index":1,"delta":{},"finish_reason":"stop"}]}`), router.ProtocolOpenAI, router.ProtocolOpenAI, state)
	if !state.UpstreamFinished {
		t.Fatal("all requested choices were not recognized")
	}
}

func TestNonChatDoneMarkerDoesNotCompletePartialStream(t *testing.T) {
	for _, upstream := range []string{router.ProtocolOpenAIResponses, router.ProtocolAnthropic, router.ProtocolGemini} {
		state := &StreamState{}
		_, _ = ConvertStreamEvent("", []byte(`[DONE]`), upstream, upstream, state)
		if state.UpstreamFinished {
			t.Fatalf("non-protocol marker completed %s", upstream)
		}
	}
}

func TestLengthLimitedChatBecomesIncompleteResponsesStream(t *testing.T) {
	for _, terminal := range []bool{false, true} {
		state := &StreamState{}
		_, err := ConvertStreamEvent("", []byte(`{"choices":[{"delta":{"content":"partial"},"finish_reason":"length"}]}`), router.ProtocolOpenAIResponses, router.ProtocolOpenAI, state)
		if err != nil {
			t.Fatal(err)
		}
		var frames []StreamFrame
		if terminal {
			frames, err = ConvertStreamEvent("", []byte(`[DONE]`), router.ProtocolOpenAIResponses, router.ProtocolOpenAI, state)
		} else {
			frames, err = FinalizeStream(router.ProtocolOpenAIResponses, state)
		}
		if err != nil {
			t.Fatal(err)
		}
		var text strings.Builder
		for _, frame := range frames {
			text.Write(frame.Data)
		}
		if !strings.Contains(text.String(), `"type":"response.incomplete"`) || !strings.Contains(text.String(), `"reason":"max_output_tokens"`) || strings.Contains(text.String(), `"type":"response.completed"`) {
			t.Fatalf("length-limited completion = %s", text.String())
		}
	}
}
