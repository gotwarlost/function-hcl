// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package logging

import (
	"io"
	"os"
	"strings"
	"text/template"
	"time"
)

type templatedPath interface {
	Parse(text string) (*template.Template, error)
	Funcs(funcMap template.FuncMap) *template.Template
	Execute(wr io.Writer, data interface{}) error
}

func newPath(name string) templatedPath {
	tpl := template.New(name)
	tpl = tpl.Funcs(template.FuncMap{
		"timestamp": time.Now().Local().Unix,
		"pid":       os.Getpid,
		"ppid":      os.Getppid,
	})

	return tpl
}

func ParseRawPath(name string, rawPath string) (string, error) {
	tpl, err := newPath(name).Parse(rawPath)
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
