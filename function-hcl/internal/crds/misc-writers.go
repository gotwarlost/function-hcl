package crds

import (
	"log"
	"os"
	"time"
)

// nop writer for embedding.
type nop struct{}

func (nop) StartImage(_ Image) error               { return nil }
func (nop) Write(_ ObjectMetadata, _ []byte) error { return nil }
func (nop) EndImage() error                        { return nil }

// ProgressWriter is a writer than writes progress log messages when a new image is started and ended.
type ProgressWriter struct {
	nop
	logger *log.Logger
	start  time.Time
}

// NewProgressWriter returns a progress writer.
func NewProgressWriter(logger *log.Logger) *ProgressWriter {
	if logger == nil {
		logger = log.New(os.Stderr, "", 0)
	}
	return &ProgressWriter{logger: logger}
}

func (w *ProgressWriter) StartImage(image Image) error {
	v := image.Version
	if v != "" {
		v = " (" + v + ")"
	}
	w.logger.Printf("process %s%s", image.Name, v)
	w.start = time.Now()
	return nil
}

func (w *ProgressWriter) EndImage() error {
	tt := time.Since(w.start).Round(10 * time.Millisecond)
	w.logger.Println("  processed in [", tt, "]")
	return nil
}

type WarningWriter struct {
	nop
	logger           *log.Logger
	current          Image
	seenImageNames   map[string]Image
	seenObjects      map[ObjectMetadata]Image
	currentImageSeen bool
}

// NewWarningWriter returns a warning writer.
func NewWarningWriter(logger *log.Logger) *WarningWriter {
	if logger == nil {
		logger = log.New(os.Stderr, "", 0)
	}
	return &WarningWriter{
		logger:         logger,
		seenImageNames: map[string]Image{},
		seenObjects:    map[ObjectMetadata]Image{},
	}
}

func (w *WarningWriter) StartImage(image Image) error {
	w.currentImageSeen = false
	dup, ok := w.seenImageNames[image.Name]
	if ok {
		w.logger.Printf("[warn] image %s already processed as %s", image.String(), dup.String())
		w.currentImageSeen = true
	}
	w.current = image
	return nil
}

func (w *WarningWriter) Write(m ObjectMetadata, _ []byte) error {
	dup, ok := w.seenObjects[m]
	if ok && !w.currentImageSeen {
		w.logger.Printf("[warn] object %s in image %s already processed in %s", m, w.current.String(), dup.String())
	}
	w.seenObjects[m] = w.current
	return nil
}

// MultiWriter delegates to one or more writers.
type MultiWriter struct {
	writers []Writer
}

// NewMultiWriter creates a MultiWriter.
func NewMultiWriter(writers ...Writer) *MultiWriter {
	return &MultiWriter{writers: writers}
}

func (m *MultiWriter) StartImage(img Image) error {
	for _, w := range m.writers {
		if err := w.StartImage(img); err != nil {
			return err
		}
	}
	return nil
}

func (m *MultiWriter) Write(meta ObjectMetadata, content []byte) error {
	for _, w := range m.writers {
		if err := w.Write(meta, content); err != nil {
			return err
		}
	}
	return nil
}

func (m *MultiWriter) EndImage() error {
	for _, w := range m.writers {
		if err := w.EndImage(); err != nil {
			return err
		}
	}
	return nil
}
