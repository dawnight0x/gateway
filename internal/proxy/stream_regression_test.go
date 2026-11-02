package proxy

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"local-ai-gateway/internal/config"
	"local-ai-gateway/internal/model"
	"local-ai-gateway/internal/protocol"
	"local-ai-gateway/internal/router"
)

func TestCleanEOFWithoutTerminalEventIsStreamInterruption(t *testing.T) {
	tests := []struct{ name, inbound, upstream, event, data string }{
		{"native-chat", router.ProtocolOpenAI, router.ProtocolOpenAI, "", `{"choices":[{"delta":{"content":"partial"},"finish_reason":null}]}`},
		{"converted-chat-responses", router.ProtocolOpenAIResponses, router.ProtocolOpenAI, "", `{"choices":[{"delta":{"content":"partial"},"finish_reason":null}]}`},
		{"native-responses", router.ProtocolOpenAIResponses, router.ProtocolOpenAIResponses, "response.output_text.delta", `{"type":"response.output_text.delta","delta":"partial"}`},
		{"converted-responses-chat", router.ProtocolOpenAI, router.ProtocolOpenAIResponses, "response.output_text.delta", `{"type":"response.output_text.delta","delta":"partial"}`},
		{"native-anthropic", router.ProtocolAnthropic, router.ProtocolAnthropic, "content_block_delta", `{"type":"content_block_delta","delta":{"type":"text_delta","text":"partial"}}`},
		{"converted-anthropic-chat", router.ProtocolOpenAI, router.ProtocolAnthropic, "content_block_delta", `{"type":"content_block_delta","delta":{"type":"text_delta","text":"partial"}}`},
		{"native-gemini", router.ProtocolGemini, router.ProtocolGemini, "", `{"candidates":[{"content":{"parts":[{"text":"partial"}]}}]}`},
		{"converted-gemini-chat", router.ProtocolOpenAI, router.ProtocolGemini, "", `{"candidates":[{"content":{"parts":[{"text":"partial"}]}}]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &Service{cfg: config.Default()}
			response := &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Request: httptest.NewRequest(http.MethodPost, "http://upstream", nil), Body: io.NopCloser(strings.NewReader("event: " + tt.event + "\ndata: " + tt.data + "\n\n"))}
			recorder := httptest.NewRecorder()
			result := svc.pipeStream(recorder, response, model.Key{}, tt.inbound, tt.upstream)
			if result.ok || !result.committed || result.retryable || result.errorType != "stream_interrupted" {
				t.Fatalf("truncated stream result = %+v", result)
			}
			body := recorder.Body.String()
			if !strings.Contains(body, "partial") || strings.Contains(body, "response.completed") || strings.Contains(body, "data: [DONE]") {
				t.Fatalf("truncated stream gained a completion marker: %s", body)
			}
		})
	}
}

func TestTruncatedConvertedStreamLogsFailureAndNeverRetriesAfterCommit(t *testing.T) {
	var chatCalls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/responses" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		chatCalls.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"partial\"},\"finish_reason\":null}]}\n\n")
	}))
	defer upstream.Close()
	st := testStore(t)
	ctx := context.Background()
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "p", Name: "p", Type: model.ProviderOpenAICompatible, BaseURL: upstream.URL, Enabled: true})
	for _, id := range []string{"first", "second"} {
		_, _ = st.UpsertKey(ctx, model.Key{ID: id, ProviderID: "p", Name: id, Secret: id, Enabled: true})
	}
	key, _ := st.CreateGatewayKey(ctx, "client")
	cfg := config.Default()
	cfg.Routing.RetryAmbiguousErrors = true
	gw := testGateway(t, st, cfg)
	defer gw.Close()
	req, _ := http.NewRequest(http.MethodPost, gw.URL+"/v1/responses", strings.NewReader(`{"model":"gpt","stream":true,"input":"hi"}`))
	req.Header.Set("Authorization", "Bearer "+key.Plaintext)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(res.Body)
	_ = res.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if chatCalls.Load() != 1 || strings.Contains(string(body), "response.completed") || !strings.Contains(string(body), "partial") {
		t.Fatalf("calls=%d body=%s", chatCalls.Load(), body)
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		logs, err := st.ListLogs(ctx, 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(logs) > 0 {
			if logs[0].ErrorType != "stream_interrupted" {
				t.Fatalf("truncated stream log = %+v", logs[0])
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("interrupted stream was not logged")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestStreamingFallbackBodyReadsAreBounded(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusTooManyRequests} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			release := make(chan struct{})
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.Copy(io.Discard, r.Body)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				w.(http.Flusher).Flush()
				select {
				case <-release:
				case <-r.Context().Done():
				}
			}))
			defer upstream.Close()
			defer close(release)
			cfg := config.Default()
			cfg.Routing.TimeoutSeconds = 1
			svc := New(nil, nil, cfg)
			req := httptest.NewRequest(http.MethodPost, "http://gateway", nil)
			start := time.Now()
			result := svc.forward(httptest.NewRecorder(), req, upstream.URL, model.Key{}, router.ProtocolOpenAI, router.ProtocolOpenAI, "test", protocol.ConvertedRequest{Body: []byte(`{"stream":true}`), Stream: true})
			if elapsed := time.Since(start); elapsed > 2500*time.Millisecond {
				t.Fatalf("body read ignored configured timeout: %s", elapsed)
			}
			if result.ok || result.committed || result.errorType != "timeout" || !result.ambiguous || !result.retryable {
				t.Fatalf("stalled streaming fallback result = %+v", result)
			}
		})
	}
}

func TestStreamingJSONFallbackHonorsClientCancellation(t *testing.T) {
	release := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	defer upstream.Close()
	defer close(release)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	timer := time.AfterFunc(150*time.Millisecond, cancel)
	defer timer.Stop()
	req := httptest.NewRequest(http.MethodPost, "http://gateway", nil).WithContext(ctx)
	svc := New(nil, nil, config.Default())
	start := time.Now()
	result := svc.forward(httptest.NewRecorder(), req, upstream.URL, model.Key{}, router.ProtocolOpenAI, router.ProtocolOpenAI, "test", protocol.ConvertedRequest{Body: []byte(`{"stream":true}`), Stream: true})
	if result.ok || !strings.Contains(result.message, "canceled") || time.Since(start) > time.Second {
		t.Fatalf("cancellation result = %+v", result)
	}
}

func TestStreamIdleReaderTracksBytesInsideUnfinishedSSELine(t *testing.T) {
	body, writer := io.Pipe()
	defer body.Close()
	reader := newStreamIdleReader(body, 300*time.Millisecond)
	defer reader.stop()
	go func() {
		defer writer.Close()
		for range 6 {
			if _, err := io.WriteString(writer, "partial"); err != nil {
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()
	data, err := io.ReadAll(reader)
	if err != nil || string(data) != strings.Repeat("partial", 6) {
		t.Fatalf("active byte stream timed out: data=%q err=%v", data, err)
	}
}

func TestStreamIdleReaderInterruptsBlockedRead(t *testing.T) {
	body, writer := io.Pipe()
	defer body.Close()
	defer writer.Close()
	reader := newStreamIdleReader(body, 150*time.Millisecond)
	defer reader.stop()
	start := time.Now()
	_, err := io.ReadAll(reader)
	if err == nil || !strings.Contains(err.Error(), "idle timeout") || time.Since(start) > time.Second {
		t.Fatalf("idle read: %v", err)
	}
}
