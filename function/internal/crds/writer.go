package crds

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func writeDoc(output io.Writer, content []byte) error {
	_, err := io.WriteString(output, "---\n")
	if err != nil {
		return err
	}
	_, err = output.Write(content)
	return err
}

// StreamWriter writes multiple YAML documents to an output stream.
type StreamWriter struct {
	nop
	output io.Writer
}

// NewStreamWriter creates a StreamWriter.
func NewStreamWriter(output io.Writer) *StreamWriter {
	return &StreamWriter{output: output}
}

func (w *StreamWriter) Write(_ ObjectMetadata, content []byte) error {
	return writeDoc(w.output, content)
}

// SplitFileWriter writes one file per document to an output directory.
type SplitFileWriter struct {
	nop
	outputDir string
}

// NewSplitFileWriter creates a SplitFileWriter.
func NewSplitFileWriter(outputDir string) (*SplitFileWriter, error) {
	err := os.MkdirAll(outputDir, 0o755)
	if err != nil {
		return nil, err
	}
	return &SplitFileWriter{outputDir: outputDir}, nil
}

func (w *SplitFileWriter) Write(m ObjectMetadata, content []byte) error {
	outFile := filepath.Join(w.outputDir, m.Filename())
	return os.WriteFile(outFile, content, 0o644)
}

// SplitImageWriter writes a file for every image processed and an additional one for any local files.
type SplitImageWriter struct {
	outputDir   string
	currentFile *os.File
}

// NewSplitImageWriter creates a SplitImageWriter.
func NewSplitImageWriter(outputDir string) (*SplitImageWriter, error) {
	err := os.MkdirAll(outputDir, 0o755)
	if err != nil {
		return nil, err
	}
	return &SplitImageWriter{outputDir: outputDir}, nil
}

func (w *SplitImageWriter) StartImage(image Image) error {
	var err error
	w.currentFile, err = os.Create(filepath.Join(w.outputDir, image.Filename()))
	if err != nil {
		return err
	}
	_, err = io.WriteString(w.currentFile, fmt.Sprintf("# source image: %s\n", image))
	return err
}

func (w *SplitImageWriter) Write(_ ObjectMetadata, content []byte) error {
	return writeDoc(w.currentFile, content)
}

func (w *SplitImageWriter) EndImage() error {
	err := w.currentFile.Close()
	w.currentFile = nil
	return err
}
