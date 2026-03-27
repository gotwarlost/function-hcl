// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package filesystem

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/document"
	"github.com/google/go-cmp/cmp"
)

func TestFilesystem_ReadFile_osOnly(t *testing.T) {
	tmpDir := t.TempDir()
	f, err := os.Create(filepath.Join(tmpDir, "testfile"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	content := "lorem ipsum"
	_, err = f.WriteString(content)
	if err != nil {
		t.Fatal(err)
	}

	fs := New(testDocumentStore{})
	b, err := fs.ReadFile(filepath.Join(tmpDir, "testfile"))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != content {
		t.Fatalf("expected content to match %q, given: %q",
			content, string(b))
	}

	_, err = fs.ReadFile(filepath.Join(tmpDir, "not-existing"))
	if err == nil {
		t.Fatal("expected file to not exist")
	}

	if !os.IsNotExist(err) {
		t.Fatalf("expected file to not exist, given error: %s", err)
	}
}

func TestFilesystem_ReadFile_memOnly(t *testing.T) {
	testHandle := document.HandleFromURI("file:///tmp/test.tf")
	content := "test content"

	fs := New(testDocumentStore{
		testHandle: &document.Document{
			Dir:        testHandle.Dir,
			Filename:   testHandle.Filename,
			LanguageID: "terraform",
			Version:    0,
			Text:       []byte(content),
		},
	})

	b, err := fs.ReadFile(testHandle.FullPath())
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != content {
		t.Fatalf("expected content to match %q, given: %q",
			content, string(b))
	}

	_, err = fs.ReadFile(filepath.Join("tmp", "not-existing"))
	if err == nil {
		t.Fatal("expected file to not exist")
	}

	if !os.IsNotExist(err) {
		t.Fatalf("expected file to not exist, given error: %s", err)
	}
}

func TestFilesystem_ReadFile_memAndOs(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "testfile")

	f, err := os.Create(testPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	osContent := "os content"
	_, err = f.WriteString(osContent)
	if err != nil {
		t.Fatal(err)
	}

	testHandle := document.HandleFromPath(testPath)
	memContent := "in-mem content"
	fs := New(testDocumentStore{
		testHandle: &document.Document{
			Dir:        testHandle.Dir,
			Filename:   testHandle.Filename,
			LanguageID: "terraform",
			Version:    0,
			Text:       []byte(memContent),
		},
	})

	b, err := fs.ReadFile(testPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != memContent {
		t.Fatalf("expected content to match %q, given: %q",
			memContent, string(b))
	}

	_, err = fs.ReadFile(filepath.Join(tmpDir, "not-existing"))
	if err == nil {
		t.Fatal("expected file to not exist")
	}

	if !os.IsNotExist(err) {
		t.Fatalf("expected file to not exist, given error: %s", err)
	}
}

func TestFilesystem_ReadDir_memAndOs(t *testing.T) {
	tmpDir := t.TempDir()

	f, err := os.Create(filepath.Join(tmpDir, "osfile"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	testHandle := document.HandleFromPath(filepath.Join(tmpDir, "memfile"))
	fs := New(testDocumentStore{
		testHandle: &document.Document{
			Dir:        testHandle.Dir,
			Filename:   testHandle.Filename,
			LanguageID: "terraform",
			Version:    0,
			Text:       []byte("test"),
		},
	})

	fis, err := fs.ReadDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	expectedFis := []string{"memfile", "osfile"}
	names := namesFromFileInfos(fis)
	if diff := cmp.Diff(expectedFis, names); diff != "" {
		t.Fatalf("file list mismatch: %s", diff)
	}
}

func TestFilesystem_ReadDir_memFsOnly(t *testing.T) {
	tmpDir := t.TempDir()

	testHandle := document.HandleFromPath(filepath.Join(tmpDir, "memfile"))
	fs := New(testDocumentStore{
		testHandle: &document.Document{
			Dir:        testHandle.Dir,
			Filename:   testHandle.Filename,
			LanguageID: "terraform",
			Version:    0,
			Text:       []byte("test"),
		},
	})

	fis, err := fs.ReadDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	expectedFis := []string{"memfile"}
	names := namesFromFileInfos(fis)
	if diff := cmp.Diff(expectedFis, names); diff != "" {
		t.Fatalf("file list mismatch: %s", diff)
	}
}

func TestFilesystem_Open_osOnly(t *testing.T) {
	tmpDir := t.TempDir()

	f, err := os.Create(filepath.Join(tmpDir, "testfile"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	content := "lorem ipsum"
	_, err = f.WriteString(content)
	if err != nil {
		t.Fatal(err)
	}

	fs := New(testDocumentStore{})
	f1, err := fs.Open(filepath.Join(tmpDir, "testfile"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f1.Close() }()

	f2, err := fs.Open(filepath.Join(tmpDir, "not-existing"))
	if err == nil {
		defer func() { _ = f2.Close() }()
		t.Fatal("expected file to not exist")
	}

	if !os.IsNotExist(err) {
		t.Fatalf("expected file to not exist, given error: %s", err)
	}
}

func TestFilesystem_Open_memOnly(t *testing.T) {
	tmpDir := t.TempDir()

	path := filepath.Join(tmpDir, "test.tf")
	testHandle := document.HandleFromPath(path)

	fs := New(testDocumentStore{
		testHandle: &document.Document{
			Dir:        testHandle.Dir,
			Filename:   testHandle.Filename,
			LanguageID: "terraform",
			Version:    0,
			Text:       []byte("test"),
		},
	})

	f1, err := fs.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f1.Close() }()

	f2, err := fs.Open(filepath.Join("tmp", "not-existing"))
	if err == nil {
		defer func() { _ = f2.Close() }()
		t.Fatal("expected file to not exist")
	}

	if !os.IsNotExist(err) {
		t.Fatalf("expected file to not exist, given error: %s", err)
	}
}

func TestFilesystem_Open_memAndOs(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "testfile")

	f, err := os.Create(testPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	osContent := "os content"
	_, err = f.WriteString(osContent)
	if err != nil {
		t.Fatal(err)
	}

	testHandle := document.HandleFromPath(testPath)
	memContent := "in-mem content"

	fs := New(testDocumentStore{
		testHandle: &document.Document{
			Dir:        testHandle.Dir,
			Filename:   testHandle.Filename,
			LanguageID: "terraform",
			Version:    0,
			Text:       []byte(memContent),
		},
	})

	f1, err := fs.Open(testPath)
	if err != nil {
		t.Fatal(err)
	}
	fi, err := f1.Stat()
	if err != nil {
		t.Fatal(err)
	}
	size := int(fi.Size())
	if size != len(memContent) {
		t.Fatalf("expected size to match %d, given: %d",
			len(memContent), size)
	}

	_, err = fs.Open(filepath.Join(tmpDir, "not-existing"))
	if err == nil {
		t.Fatal("expected file to not exist")
	}

	if !os.IsNotExist(err) {
		t.Fatalf("expected file to not exist, given error: %s", err)
	}
}

func namesFromFileInfos(entries []fs.DirEntry) []string {
	names := make([]string, len(entries))
	for i, entry := range entries {
		names[i] = entry.Name()
	}
	return names
}

type testDocumentStore map[document.Handle]*document.Document

func (ds testDocumentStore) Get(dh document.Handle) (*document.Document, error) {
	doc, ok := ds[dh]
	if !ok {
		return nil, document.NotFound(dh.FullURI())
	}
	return doc, nil
}

func (ds testDocumentStore) List(dirHandle document.DirHandle) []*document.Document {
	var docs []*document.Document
	for dh, doc := range ds {
		if dh.Dir == dirHandle {
			docs = append(docs, doc)
		}
	}
	return docs
}
