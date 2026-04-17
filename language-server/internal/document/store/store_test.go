// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package store

import (
	"sync"
	"testing"

	"github.com/crossplane-contrib/function-hcl/language-server/internal/document"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// created by Claude and looks reasonable

func createTestHandle(path string) document.Handle {
	return document.HandleFromPath(path)
}

func TestNew(t *testing.T) {
	store := New()
	require.NotNil(t, store)
	assert.NotNil(t, store.docs)
	assert.NotNil(t, store.logger)
	assert.Equal(t, 0, len(store.docs))
}

func TestOpen_Success(t *testing.T) {
	store := New()
	dh := createTestHandle("/test/dir/file.hcl")
	text := []byte("resource test {}")

	err := store.Open(dh, "hcl", 1, text)
	require.NoError(t, err)

	doc, err := store.Get(dh)
	require.NoError(t, err)
	assert.Equal(t, dh.Dir, doc.Dir)
	assert.Equal(t, dh.Filename, doc.Filename)
	assert.Equal(t, "hcl", doc.LanguageID)
	assert.Equal(t, 1, doc.Version)
	assert.Equal(t, text, doc.Text)
	assert.NotNil(t, doc.Lines)
	assert.False(t, doc.ModTime.IsZero())
}

func TestOpen_AlreadyExists(t *testing.T) {
	store := New()
	dh := createTestHandle("/test/dir/file.hcl")
	text := []byte("resource test {}")

	err := store.Open(dh, "hcl", 1, text)
	require.NoError(t, err)

	err = store.Open(dh, "hcl", 2, text)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestUpdate_Success(t *testing.T) {
	store := New()
	dh := createTestHandle("/test/dir/file.hcl")
	initialText := []byte("resource test {}")
	updatedText := []byte("resource updated {}")

	err := store.Open(dh, "hcl", 1, initialText)
	require.NoError(t, err)

	err = store.Update(dh, updatedText, 2)
	require.NoError(t, err)

	doc, err := store.Get(dh)
	require.NoError(t, err)
	assert.Equal(t, 2, doc.Version)
	assert.Equal(t, updatedText, doc.Text)
	assert.Equal(t, "hcl", doc.LanguageID)
}

func TestUpdate_NotFound(t *testing.T) {
	store := New()
	dh := createTestHandle("/test/dir/nonexistent.hcl")
	text := []byte("resource test {}")

	err := store.Update(dh, text, 2)
	require.Error(t, err)
	assert.True(t, document.IsNotFound(err))
}

func TestUpdate_VersionNotAscending(t *testing.T) {
	store := New()
	dh := createTestHandle("/test/dir/file.hcl")
	text := []byte("resource test {}")

	err := store.Open(dh, "hcl", 5, text)
	require.NoError(t, err)

	err = store.Update(dh, text, 3)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "version not ascending")

	err = store.Update(dh, text, 5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "version not ascending")
}

func TestClose_Success(t *testing.T) {
	store := New()
	dh := createTestHandle("/test/dir/file.hcl")
	text := []byte("resource test {}")

	err := store.Open(dh, "hcl", 1, text)
	require.NoError(t, err)

	err = store.Close(dh)
	require.NoError(t, err)

	_, err = store.Get(dh)
	require.Error(t, err)
	assert.True(t, document.IsNotFound(err))
}

func TestClose_NotFound(t *testing.T) {
	store := New()
	dh := createTestHandle("/test/dir/nonexistent.hcl")

	err := store.Close(dh)
	require.Error(t, err)
	assert.True(t, document.IsNotFound(err))
}

func TestGet_Success(t *testing.T) {
	store := New()
	dh := createTestHandle("/test/dir/file.hcl")
	text := []byte("resource test {}")

	err := store.Open(dh, "hcl", 1, text)
	require.NoError(t, err)

	doc, err := store.Get(dh)
	require.NoError(t, err)
	assert.NotNil(t, doc)
	assert.Equal(t, text, doc.Text)
	assert.Equal(t, 1, doc.Version)
}

func TestGet_NotFound(t *testing.T) {
	store := New()
	dh := createTestHandle("/test/dir/nonexistent.hcl")

	_, err := store.Get(dh)
	require.Error(t, err)
	assert.True(t, document.IsNotFound(err))
}

func TestIsDocumentOpen(t *testing.T) {
	store := New()
	dh := createTestHandle("/test/dir/file.hcl")
	text := []byte("resource test {}")

	assert.False(t, store.IsDocumentOpen(dh))

	err := store.Open(dh, "hcl", 1, text)
	require.NoError(t, err)
	assert.True(t, store.IsDocumentOpen(dh))

	err = store.Close(dh)
	require.NoError(t, err)
	assert.False(t, store.IsDocumentOpen(dh))
}

func TestHasOpenDocuments(t *testing.T) {
	store := New()
	dh1 := createTestHandle("/test/dir1/file1.hcl")
	dh2 := createTestHandle("/test/dir1/file2.hcl")
	dh3 := createTestHandle("/test/dir2/file3.hcl")
	text := []byte("resource \"test\" {}")

	dirHandle1 := document.DirHandleFromPath("/test/dir1")
	dirHandle2 := document.DirHandleFromPath("/test/dir2")
	dirHandle3 := document.DirHandleFromPath("/test/dir3")

	assert.False(t, store.HasOpenDocuments(dirHandle1))
	assert.False(t, store.HasOpenDocuments(dirHandle2))
	assert.False(t, store.HasOpenDocuments(dirHandle3))

	err := store.Open(dh1, "hcl", 1, text)
	require.NoError(t, err)
	assert.True(t, store.HasOpenDocuments(dirHandle1))
	assert.False(t, store.HasOpenDocuments(dirHandle2))

	err = store.Open(dh2, "hcl", 1, text)
	require.NoError(t, err)
	assert.True(t, store.HasOpenDocuments(dirHandle1))

	err = store.Open(dh3, "hcl", 1, text)
	require.NoError(t, err)
	assert.True(t, store.HasOpenDocuments(dirHandle1))
	assert.True(t, store.HasOpenDocuments(dirHandle2))
	assert.False(t, store.HasOpenDocuments(dirHandle3))

	err = store.Close(dh1)
	require.NoError(t, err)
	assert.True(t, store.HasOpenDocuments(dirHandle1))

	err = store.Close(dh2)
	require.NoError(t, err)
	assert.False(t, store.HasOpenDocuments(dirHandle1))
	assert.True(t, store.HasOpenDocuments(dirHandle2))
}

func TestList(t *testing.T) {
	store := New()
	dh1 := createTestHandle("/test/dir1/file1.hcl")
	dh2 := createTestHandle("/test/dir1/file2.hcl")
	dh3 := createTestHandle("/test/dir2/file3.hcl")
	text := []byte("resource test {}")

	dirHandle1 := document.DirHandleFromPath("/test/dir1")
	dirHandle2 := document.DirHandleFromPath("/test/dir2")

	err := store.Open(dh1, "hcl", 1, text)
	require.NoError(t, err)
	err = store.Open(dh2, "hcl", 1, text)
	require.NoError(t, err)
	err = store.Open(dh3, "hcl", 1, text)
	require.NoError(t, err)

	docs1 := store.List(dirHandle1)
	assert.Len(t, docs1, 2)
	filenames := []string{docs1[0].Filename, docs1[1].Filename}
	assert.Contains(t, filenames, "file1.hcl")
	assert.Contains(t, filenames, "file2.hcl")

	docs2 := store.List(dirHandle2)
	assert.Len(t, docs2, 1)
	assert.Equal(t, "file3.hcl", docs2[0].Filename)
}

func TestList_EmptyDirectory(t *testing.T) {
	store := New()
	dirHandle := document.DirHandleFromPath("/test/empty")

	docs := store.List(dirHandle)
	assert.Nil(t, docs)
}

func TestConcurrentAccess(t *testing.T) {
	store := New()
	var wg sync.WaitGroup
	numGoroutines := 10
	text := []byte("resource test {}")

	wg.Add(numGoroutines * 3)

	for i := 0; i < numGoroutines; i++ {
		i := i
		go func() {
			defer wg.Done()
			dh := createTestHandle("/test/dir/file" + string(rune('0'+i)) + ".hcl")
			err := store.Open(dh, "hcl", 1, text)
			assert.NoError(t, err)
		}()

		go func() {
			defer wg.Done()
			dh := createTestHandle("/test/dir/file" + string(rune('0'+i)) + ".hcl")
			_, _ = store.Get(dh)
		}()

		go func() {
			defer wg.Done()
			dirHandle := document.DirHandleFromPath("/test/dir")
			_ = store.HasOpenDocuments(dirHandle)
		}()
	}

	wg.Wait()

	dirHandle := document.DirHandleFromPath("/test/dir")
	assert.True(t, store.HasOpenDocuments(dirHandle))
}

func TestMultipleUpdates(t *testing.T) {
	store := New()
	dh := createTestHandle("/test/dir/file.hcl")
	initialText := []byte("version 1")

	err := store.Open(dh, "hcl", 1, initialText)
	require.NoError(t, err)

	for i := 2; i <= 10; i++ {
		updatedText := []byte("version " + string(rune('0'+i)))
		err = store.Update(dh, updatedText, i)
		require.NoError(t, err)

		doc, err := store.Get(dh)
		require.NoError(t, err)
		assert.Equal(t, i, doc.Version)
	}
}

func TestOpenMultipleDocumentsInSameDirectory(t *testing.T) {
	store := New()
	dirPath := "/test/project"
	files := []string{"main.hcl", "variables.hcl", "outputs.hcl"}

	for i, filename := range files {
		dh := createTestHandle(dirPath + "/" + filename)
		text := []byte("content for " + filename)
		err := store.Open(dh, "hcl", i+1, text)
		require.NoError(t, err)
	}

	dirHandle := document.DirHandleFromPath(dirPath)
	docs := store.List(dirHandle)
	assert.Len(t, docs, 3)

	assert.True(t, store.HasOpenDocuments(dirHandle))

	for _, filename := range files {
		dh := createTestHandle(dirPath + "/" + filename)
		assert.True(t, store.IsDocumentOpen(dh))
	}
}

func TestDocumentPreservesLanguageID(t *testing.T) {
	store := New()
	dh := createTestHandle("/test/dir/file.hcl")
	text := []byte("initial content")

	err := store.Open(dh, "terraform", 1, text)
	require.NoError(t, err)

	updatedText := []byte("updated content")
	err = store.Update(dh, updatedText, 2)
	require.NoError(t, err)

	doc, err := store.Get(dh)
	require.NoError(t, err)
	assert.Equal(t, "terraform", doc.LanguageID)
}

func TestDocumentModTimeUpdates(t *testing.T) {
	store := New()
	dh := createTestHandle("/test/dir/file.hcl")
	text := []byte("initial content")

	err := store.Open(dh, "hcl", 1, text)
	require.NoError(t, err)

	doc1, err := store.Get(dh)
	require.NoError(t, err)
	initialModTime := doc1.ModTime

	updatedText := []byte("updated content")
	err = store.Update(dh, updatedText, 2)
	require.NoError(t, err)

	doc2, err := store.Get(dh)
	require.NoError(t, err)
	assert.True(t, doc2.ModTime.After(initialModTime) || doc2.ModTime.Equal(initialModTime))
}
