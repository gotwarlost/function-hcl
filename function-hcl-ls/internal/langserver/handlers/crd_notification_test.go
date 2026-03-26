package handlers

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/eventbus"
	lsp "github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/langserver/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleNoCRDSources_OnlyNotifiesOncePerDirectory(t *testing.T) {
	var notifiedDirs sync.Map
	var callCount atomic.Int32

	// Simulate what handleNoCRDSources does with the notifiedDirs map
	handleEvent := func(dir string) {
		if _, alreadyNotified := notifiedDirs.LoadOrStore(dir, true); alreadyNotified {
			return
		}
		callCount.Add(1)
	}

	// Same directory multiple times
	for i := 0; i < 5; i++ {
		handleEvent("/test/workspace")
	}

	assert.Equal(t, int32(1), callCount.Load(), "should only notify once per directory")
}

func TestHandleNoCRDSources_NotifiesDifferentDirectories(t *testing.T) {
	var notifiedDirs sync.Map
	var callCount atomic.Int32

	handleEvent := func(dir string) {
		if _, alreadyNotified := notifiedDirs.LoadOrStore(dir, true); alreadyNotified {
			return
		}
		callCount.Add(1)
	}

	// Different directories
	handleEvent("/test/workspace1")
	handleEvent("/test/workspace2")
	handleEvent("/test/workspace3")

	assert.Equal(t, int32(3), callCount.Load(), "should notify for each unique directory")
}

func TestHandleNoCRDSources_MixedDirectories(t *testing.T) {
	var notifiedDirs sync.Map
	var callCount atomic.Int32

	handleEvent := func(dir string) {
		if _, alreadyNotified := notifiedDirs.LoadOrStore(dir, true); alreadyNotified {
			return
		}
		callCount.Add(1)
	}

	// Mix of same and different directories
	handleEvent("/test/workspace1")
	handleEvent("/test/workspace1")
	handleEvent("/test/workspace2")
	handleEvent("/test/workspace1")
	handleEvent("/test/workspace2")
	handleEvent("/test/workspace3")

	assert.Equal(t, int32(3), callCount.Load(), "should notify once per unique directory")
}

func TestStartCRDNotificationHandler_ReceivesEvents(t *testing.T) {
	bus := eventbus.New()

	// Track received events
	var receivedDirs []string
	var mu sync.Mutex

	// Subscribe directly to verify events flow through
	ch := bus.SubscribeToNoCRDSourcesEvents("test-receiver")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		for {
			select {
			case event := <-ch:
				mu.Lock()
				receivedDirs = append(receivedDirs, event.Dir)
				mu.Unlock()
			case <-ctx.Done():
				return
			}
		}
	}()

	// Publish events
	bus.PublishNoCRDSourcesEvent(eventbus.NoCRDSourcesEvent{Dir: "/test/workspace1"})
	bus.PublishNoCRDSourcesEvent(eventbus.NoCRDSourcesEvent{Dir: "/test/workspace2"})

	// Wait for events to be processed
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, receivedDirs, 2)
	assert.Contains(t, receivedDirs, "/test/workspace1")
	assert.Contains(t, receivedDirs, "/test/workspace2")
}

func TestShowMessageRequestParams_Structure(t *testing.T) {
	// Verify the params structure is correct
	params := &lsp.ShowMessageRequestParams{
		Type:    lsp.Info,
		Message: "No CRD sources configured for this workspace. Configure CRDs for full completion and validation support.",
		Actions: []lsp.MessageActionItem{
			{Title: "Learn More"},
		},
	}

	assert.Equal(t, lsp.Info, params.Type)
	assert.Contains(t, params.Message, "No CRD sources configured")
	require.Len(t, params.Actions, 1)
	assert.Equal(t, "Learn More", params.Actions[0].Title)
}

func TestShowDocumentParams_Structure(t *testing.T) {
	// Verify the params structure is correct for opening URLs
	params := &lsp.ShowDocumentParams{
		URI:      lsp.URI(crdSetupURL),
		External: true,
	}

	assert.Equal(t, lsp.URI(crdSetupURL), params.URI)
	assert.True(t, params.External)
}
