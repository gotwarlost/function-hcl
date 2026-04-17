package store

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	s := New(nil)
	require.NotNil(t, s)
	assert.NotNil(t, s.queue)
	assert.NotNil(t, s.files)
	assert.NotNil(t, s.sources)
	assert.NotNil(t, s.seenDirs)
	assert.Nil(t, s.onNoCRDSources)
}

func TestNew_WithCallback(t *testing.T) {
	called := false
	callback := func(dir string) {
		called = true
	}
	s := New(callback)
	require.NotNil(t, s)
	assert.NotNil(t, s.onNoCRDSources)

	// Invoke callback to verify it's set correctly
	s.onNoCRDSources("/test")
	assert.True(t, called)
}

func TestDiscoverSourceStore_CallsCallbackWhenNoSourcesFound(t *testing.T) {
	// Create a temp directory with no CRD sources
	tmpDir, err := os.MkdirTemp("", "crd-store-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	var callbackDir string
	var callCount atomic.Int32
	callback := func(dir string) {
		callbackDir = dir
		callCount.Add(1)
	}

	s := New(callback)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Start(ctx)

	// Register the directory - this should trigger discovery
	s.RegisterOpenDir(tmpDir)

	// Wait for the queue to process by polling on the expected condition
	require.Eventually(t, func() bool {
		return callCount.Load() == 1
	}, time.Second, 10*time.Millisecond, "callback should be called once")
	assert.Equal(t, tmpDir, callbackDir, "callback should receive the directory path")
}

func TestDiscoverSourceStore_NoCallbackWhenSourcesExist(t *testing.T) {
	// Create a temp directory with .crds subdirectory
	tmpDir, err := os.MkdirTemp("", "crd-store-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	crdsDir := filepath.Join(tmpDir, ".crds")
	err = os.Mkdir(crdsDir, 0755)
	require.NoError(t, err)

	var callCount atomic.Int32
	callback := func(dir string) {
		callCount.Add(1)
	}

	s := New(callback)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Start(ctx)

	// Register the directory
	s.RegisterOpenDir(tmpDir)

	// Wait for the queue to process
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, int32(0), callCount.Load(), "callback should not be called when sources exist")
}

func TestDiscoverSourceStore_CallbackOnlyOnFirstDiscovery(t *testing.T) {
	// Create a temp directory with no CRD sources
	tmpDir, err := os.MkdirTemp("", "crd-store-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	var callCount atomic.Int32
	callback := func(dir string) {
		callCount.Add(1)
	}

	s := New(callback)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Start(ctx)

	// Register the same directory multiple times
	s.RegisterOpenDir(tmpDir)
	s.RegisterOpenDir(tmpDir)
	s.RegisterOpenDir(tmpDir)

	// Wait for the queue to process by polling on the expected condition
	require.Eventually(t, func() bool {
		return callCount.Load() == 1
	}, time.Second, 100*time.Millisecond, "callback should be called once")
}

func TestDiscoverSourceStore_NilCallbackDoesNotPanic(t *testing.T) {
	// Create a temp directory with no CRD sources
	tmpDir, err := os.MkdirTemp("", "crd-store-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	s := New(nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Start(ctx)

	// This should not panic even with nil callback
	require.NotPanics(t, func() {
		s.RegisterOpenDir(tmpDir)
		time.Sleep(100 * time.Millisecond)
	})
}
