// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

// Package logging provider logging facilities.
package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// PerfLogger is a nil logger than can be set to a real one to log timing information from multiple places.
var PerfLogger *log.Logger = NopLogger()

func NopLogger() *log.Logger {
	return log.New(io.Discard, "", 0)
}

func NewLogger(w io.Writer) *log.Logger {
	return log.New(w, "", log.LstdFlags|log.Lshortfile)
}

// FileLogger wraps a file-based logger.
type FileLogger struct {
	l *log.Logger
	f *os.File
}

// NewFileLogger creates a new file-based logger.
func NewFileLogger(rawPath string) (*FileLogger, error) {
	path, err := parseLogPath(rawPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse path: %w", err)
	}

	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("please provide absolute log path to prevent ambiguity (given: %q)",
			path)
	}

	mode := os.O_TRUNC | os.O_CREATE | os.O_WRONLY
	file, err := os.OpenFile(path, mode, 0o600)
	if err != nil {
		return nil, err
	}

	return &FileLogger{
		l: NewLogger(file),
		f: file,
	}, nil
}

// Writer returns the underlying io.Writer for the file logger.
func (fl *FileLogger) Writer() io.Writer {
	return fl.f
}

func parseLogPath(rawPath string) (string, error) {
	tpl, err := newPath("log-file").Parse(rawPath)
	if err != nil {
		return "", err
	}

	buf := &strings.Builder{}
	err = tpl.Execute(buf, nil)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func ValidateExecLogPath(rawPath string) error {
	_, err := parseExecLogPathTemplate("", rawPath)
	return err
}

func ParseExecLogPath(method string, rawPath string) (string, error) {
	tpl, err := parseExecLogPathTemplate(method, rawPath)
	if err != nil {
		return "", err
	}

	buf := &strings.Builder{}
	err = tpl.Execute(buf, nil)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func parseExecLogPathTemplate(method string, rawPath string) (templatedPath, error) {
	tpl := newPath("tf-log-file")
	methodFunc := func() string {
		return method
	}
	tpl = tpl.Funcs(template.FuncMap{
		"method": methodFunc,
		// DEPRECATED
		"args": methodFunc,
	})
	return tpl.Parse(rawPath)
}

// Logger returns the underlying logger.
func (fl *FileLogger) Logger() *log.Logger {
	return fl.l
}

// Close closes the log file.
func (fl *FileLogger) Close() error {
	return fl.f.Close()
}
