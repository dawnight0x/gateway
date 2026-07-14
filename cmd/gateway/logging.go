package main

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

type rotatingLogWriter struct {
	mu         sync.Mutex
	path       string
	file       *os.File
	maxBytes   int64
	maxBackups int
}

func setupLogging(path string, maxSizeMB, maxBackups int) (io.WriteCloser, error) {
	writer, err := newRotatingLogWriter(path, int64(maxSizeMB)<<20, maxBackups)
	if err != nil {
		return nil, err
	}
	var output io.Writer = writer
	if stdoutAvailable() {
		output = io.MultiWriter(os.Stdout, writer)
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(output, &slog.HandlerOptions{Level: slog.LevelInfo})))
	slog.Info("logging initialized", "path", path, "max_size_mb", maxSizeMB, "max_backups", maxBackups)
	return writer, nil
}

func newRotatingLogWriter(path string, maxBytes int64, maxBackups int) (*rotatingLogWriter, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, err
	}
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return nil, err
	}
	return &rotatingLogWriter{path: path, file: file, maxBytes: maxBytes, maxBackups: maxBackups}, nil
}

func (w *rotatingLogWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.maxBytes > 0 {
		info, err := w.file.Stat()
		if err != nil {
			return 0, err
		}
		if info.Size() > 0 && info.Size()+int64(len(p)) > w.maxBytes {
			if err := w.rotateLocked(); err != nil {
				return 0, err
			}
		}
	}
	return w.file.Write(p)
}

func (w *rotatingLogWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	return err
}

func (w *rotatingLogWriter) rotateLocked() error {
	if err := w.file.Close(); err != nil {
		return err
	}
	for index := w.maxBackups - 1; index >= 1; index-- {
		source := w.path + "." + strconv.Itoa(index)
		target := w.path + "." + strconv.Itoa(index+1)
		if _, err := os.Stat(source); err == nil {
			_ = os.Remove(target)
			if err := os.Rename(source, target); err != nil {
				return err
			}
		}
	}
	if w.maxBackups > 0 {
		_ = os.Remove(w.path + ".1")
		if err := os.Rename(w.path, w.path+".1"); err != nil && !os.IsNotExist(err) {
			return err
		}
	} else if err := os.Remove(w.path); err != nil && !os.IsNotExist(err) {
		return err
	}
	file, err := os.OpenFile(w.path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	w.file = file
	return nil
}

func stdoutAvailable() bool {
	if os.Stdout == nil {
		return false
	}
	_, err := os.Stdout.Stat()
	return err == nil
}
