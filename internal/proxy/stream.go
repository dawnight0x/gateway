package proxy

import (
	"bufio"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"local-ai-gateway/internal/model"
	"local-ai-gateway/internal/protocol"
	"local-ai-gateway/internal/router"
)

type streamOutput struct {
	w            http.ResponseWriter
	status       int
	committed    bool
	flusher      http.Flusher
	controller   *http.ResponseController
	writeTimeout time.Duration
	beforeCommit func()
}

const (
	maxSSELineSize        = 8 << 20
	maxSSEEventSize       = 16 << 20
	maxStreamToolArgument = 16 << 20
)

func newStreamOutput(w http.ResponseWriter, status int, writeTimeout time.Duration) *streamOutput {
	flusher, _ := w.(http.Flusher)
	return &streamOutput{w: w, status: status, flusher: flusher, controller: http.NewResponseController(w), writeTimeout: writeTimeout}
}

func (o *streamOutput) commit() {
	if o.committed {
		return
	}
	if o.beforeCommit != nil {
		o.beforeCommit()
		o.beforeCommit = nil
	}
	h := o.w.Header()
	h.Set("Content-Type", "text/event-stream; charset=utf-8")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	o.w.WriteHeader(o.status)
	o.committed = true
}

func (o *streamOutput) write(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	if o.writeTimeout > 0 {
		_ = o.controller.SetWriteDeadline(time.Now().Add(o.writeTimeout))
	}
	o.commit()
	if _, err := o.w.Write(data); err != nil {
		return err
	}
	if err := o.controller.Flush(); err != nil {
		if errors.Is(err, http.ErrNotSupported) && o.flusher != nil {
			o.flusher.Flush()
			return nil
		}
		return err
	}
	return nil
}

func (s *Service) pipeStream(w http.ResponseWriter, resp *http.Response, key model.Key, inbound, upstream string) attemptResult {
	output := newStreamOutput(w, resp.StatusCode, time.Duration(s.cfg.Routing.StreamWriteTimeoutSeconds)*time.Second)
	output.beforeCommit = func() { forwardResponseMetadataHeaders(w.Header(), resp.Header) }
	if !s.cfg.Routing.StreamRetryBeforeFirstByte {
		output.commit()
	}

	state := &protocol.StreamState{
		DisableAggregate:     inbound != router.ProtocolOpenAIResponses || upstream == router.ProtocolOpenAIResponses,
		MaxAggregateBytes:    maxStreamAggregate,
		MaxToolArgumentBytes: maxStreamToolArgument,
	}
	var event string
	var dataLines []string
	dataBytes := 0
	recordedResources := 0
	processedEvents := 0
	processEvent := func() error {
		if len(dataLines) == 0 {
			event = ""
			dataBytes = 0
			return nil
		}
		data := []byte(strings.Join(dataLines, "\n"))
		processedEvents++
		frames, err := protocol.ConvertStreamEvent(event, data, inbound, upstream, state)
		event = ""
		dataLines = dataLines[:0]
		dataBytes = 0
		if err != nil {
			return err
		}
		if inbound == router.ProtocolOpenAIResponses && len(state.ResourceIDs) > recordedResources {
			s.recordResponseAffinity(resp.Request.Context(), state.ResourceIDs[recordedResources:], key)
			recordedResources = len(state.ResourceIDs)
		}
		for _, frame := range frames {
			if err := output.write(encodeSSEFrame(frame)); err != nil {
				return err
			}
		}
		return nil
	}

	var idleExpired atomic.Bool
	idleTimeout := time.Duration(s.cfg.Routing.StreamIdleTimeoutSeconds) * time.Second
	idleTimer := time.AfterFunc(idleTimeout, func() {
		idleExpired.Store(true)
		_ = resp.Body.Close()
	})
	defer idleTimer.Stop()

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 32<<10), maxSSELineSize)
	for scanner.Scan() {
		if idleExpired.Load() {
			return streamFailure(output, resp.StatusCode, state.Usage, errors.New("upstream stream idle timeout"), s.cfg.Routing.StreamRetryBeforeFirstByte)
		}
		idleTimer.Reset(idleTimeout)
		line := strings.TrimSuffix(scanner.Text(), "\r")
		switch {
		case line == "":
			if err := processEvent(); err != nil {
				return streamFailure(output, resp.StatusCode, state.Usage, err, s.cfg.Routing.StreamRetryBeforeFirstByte)
			}
		case strings.HasPrefix(line, "event:"):
			event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case strings.HasPrefix(line, "data:"):
			value := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			dataBytes += len(value)
			if dataBytes > maxSSEEventSize {
				return streamFailure(output, resp.StatusCode, state.Usage, fmt.Errorf("upstream SSE event exceeds %d MiB limit", maxSSEEventSize>>20), s.cfg.Routing.StreamRetryBeforeFirstByte)
			}
			dataLines = append(dataLines, value)
		}
	}
	readErr := scanner.Err()
	if idleExpired.Load() {
		readErr = errors.New("upstream stream idle timeout")
	}
	if readErr != nil {
		return streamFailure(output, resp.StatusCode, state.Usage, readErr, s.cfg.Routing.StreamRetryBeforeFirstByte)
	}
	if len(dataLines) > 0 {
		if err := processEvent(); err != nil {
			return streamFailure(output, resp.StatusCode, state.Usage, err, s.cfg.Routing.StreamRetryBeforeFirstByte)
		}
	}
	if processedEvents == 0 {
		if !output.committed {
			return attemptResult{status: http.StatusBadGateway, errorType: "empty_response", message: "empty upstream stream", retryable: true, ambiguous: true}
		}
		return attemptResult{committed: true, status: resp.StatusCode, errorType: "empty_response", message: "empty upstream stream", ambiguous: true}
	}
	if inbound != upstream && !state.Finished {
		frames, err := protocol.ConvertStreamEvent("", []byte("[DONE]"), inbound, upstream, state)
		if err != nil {
			return streamFailure(output, resp.StatusCode, state.Usage, err, false)
		}
		for _, frame := range frames {
			if err := output.write(encodeSSEFrame(frame)); err != nil {
				return streamFailure(output, resp.StatusCode, state.Usage, err, false)
			}
		}
	}
	resourceIDs := append([]string(nil), state.ResourceIDs...)
	if state.ID != "" {
		resourceIDs = append(resourceIDs, state.ID)
	}
	return attemptResult{ok: true, committed: true, status: resp.StatusCode, usage: state.Usage, responseResourceIDs: resourceIDs}
}

func streamFailure(output *streamOutput, status int, usage protocol.Usage, err error, retryBeforeFirstByte bool) attemptResult {
	if !output.committed {
		return attemptResult{status: http.StatusBadGateway, errorType: "upstream_error", message: err.Error(), retryable: retryBeforeFirstByte, ambiguous: true}
	}
	return attemptResult{committed: true, status: status, errorType: "stream_interrupted", message: err.Error(), usage: usage}
}

func encodeSSEFrame(frame protocol.StreamFrame) []byte {
	var out strings.Builder
	if frame.Event != "" {
		out.WriteString("event: ")
		out.WriteString(frame.Event)
		out.WriteByte('\n')
	}
	for _, line := range strings.Split(string(frame.Data), "\n") {
		out.WriteString("data: ")
		out.WriteString(line)
		out.WriteByte('\n')
	}
	out.WriteByte('\n')
	return []byte(out.String())
}
