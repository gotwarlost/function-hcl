package eventbus

import (
	"sync"
	"testing"
	"time"

	"github.com/crossplane-contrib/function-hcl/language-server/internal/document"
	"github.com/crossplane-contrib/function-hcl/language-server/internal/langserver/protocol"
	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// created by Claude and looks reasonable.

func createTestDocHandle() document.Handle {
	return document.HandleFromPath("/test/dir/file.hcl")
}

func TestNew(t *testing.T) {
	bus := New()
	require.NotNil(t, bus)
	assert.NotNil(t, bus.logger)
	assert.NotNil(t, bus.openTopic)
	assert.NotNil(t, bus.changeTopic)
	assert.NotNil(t, bus.changeWatchTopic)
	assert.NotNil(t, bus.diagnosticsTopic)
	assert.NotNil(t, bus.noCRDSourcesTopic)
}

func TestPublishOpenEvent(t *testing.T) {
	bus := New()
	ch := bus.SubscribeToOpenEvents("test-subscriber")

	testDoc := createTestDocHandle()
	event := OpenEvent{
		Doc:        testDoc,
		LanguageID: "hcl",
	}

	// Publish event in a goroutine to avoid blocking
	go bus.PublishOpenEvent(event)

	// Receive the event
	select {
	case received := <-ch:
		assert.Equal(t, event.Doc.Filename, received.Doc.Filename)
		assert.Equal(t, event.LanguageID, received.LanguageID)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for open event")
	}
}

func TestPublishEditEvent(t *testing.T) {
	bus := New()
	ch := bus.SubscribeToEditEvents("test-subscriber")

	testDoc := createTestDocHandle()
	event := EditEvent{
		Doc:        testDoc,
		LanguageID: "hcl",
	}

	go bus.PublishEditEvent(event)

	select {
	case received := <-ch:
		assert.Equal(t, event.Doc.Filename, received.Doc.Filename)
		assert.Equal(t, event.LanguageID, received.LanguageID)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for edit event")
	}
}

func TestPublishChangeWatchEvent(t *testing.T) {
	bus := New()
	ch := bus.SubscribeToChangeWatchEvents("test-subscriber")

	event := ChangeWatchEvent{
		RawPath:    "/test/dir/file.hcl",
		IsDir:      false,
		ChangeType: protocol.Changed,
	}

	go bus.PublishChangeWatchEvent(event)

	select {
	case received := <-ch:
		assert.Equal(t, event.RawPath, received.RawPath)
		assert.Equal(t, event.IsDir, received.IsDir)
		assert.Equal(t, event.ChangeType, received.ChangeType)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for change watch event")
	}
}

func TestMultipleSubscribers(t *testing.T) {
	bus := New()

	// Create multiple subscribers
	ch1 := bus.SubscribeToEditEvents("subscriber-1")
	ch2 := bus.SubscribeToEditEvents("subscriber-2")
	ch3 := bus.SubscribeToEditEvents("subscriber-3")

	testDoc := createTestDocHandle()
	event := EditEvent{
		Doc:        testDoc,
		LanguageID: "hcl",
	}

	// Use WaitGroup to ensure all subscribers receive the event
	var wg sync.WaitGroup
	wg.Add(3)

	checkEvent := func(ch <-chan EditEvent, name string) {
		defer wg.Done()
		select {
		case received := <-ch:
			assert.Equal(t, event.Doc.Filename, received.Doc.Filename)
			assert.Equal(t, event.LanguageID, received.LanguageID)
		case <-time.After(1 * time.Second):
			t.Errorf("timeout waiting for event on %s", name)
		}
	}

	// Start subscribers
	go checkEvent(ch1, "subscriber-1")
	go checkEvent(ch2, "subscriber-2")
	go checkEvent(ch3, "subscriber-3")

	// Publish event
	go bus.PublishEditEvent(event)

	wg.Wait()
}

func TestMultipleEvents(t *testing.T) {
	bus := New()
	ch := bus.SubscribeToOpenEvents("test-subscriber")

	testDoc := createTestDocHandle()
	eventCount := 5

	// Publish multiple events
	go func() {
		for i := 0; i < eventCount; i++ {
			event := OpenEvent{
				Doc:        testDoc,
				LanguageID: "hcl",
			}
			bus.PublishOpenEvent(event)
		}
	}()

	// Receive all events
	received := 0
	timeout := time.After(2 * time.Second)
	for received < eventCount {
		select {
		case event := <-ch:
			assert.Equal(t, testDoc.Filename, event.Doc.Filename)
			received++
		case <-timeout:
			t.Fatalf("timeout: only received %d/%d events", received, eventCount)
		}
	}

	assert.Equal(t, eventCount, received)
}

func TestConcurrentPublishAndSubscribe(t *testing.T) {
	bus := New()
	subscriberCount := 10
	eventCount := 20

	var wg sync.WaitGroup
	wg.Add(subscriberCount)

	testDoc := createTestDocHandle()

	// Create multiple subscribers
	for i := 0; i < subscriberCount; i++ {
		go func(id int) {
			defer wg.Done()
			ch := bus.SubscribeToEditEvents(string(rune(id)))

			receivedCount := 0
			timeout := time.After(3 * time.Second)
			for receivedCount < eventCount {
				select {
				case event := <-ch:
					assert.Equal(t, testDoc.Filename, event.Doc.Filename)
					receivedCount++
				case <-timeout:
					t.Errorf("subscriber %d: timeout after receiving %d/%d events", id, receivedCount, eventCount)
					return
				}
			}
		}(i)
	}

	// Give subscribers time to initialize
	time.Sleep(100 * time.Millisecond)

	// Publish events concurrently
	for i := 0; i < eventCount; i++ {
		go func() {
			event := EditEvent{
				Doc:        testDoc,
				LanguageID: "hcl",
			}
			bus.PublishEditEvent(event)
		}()
	}

	wg.Wait()
}

func TestDifferentEventTypes(t *testing.T) {
	bus := New()

	openCh := bus.SubscribeToOpenEvents("open-subscriber")
	editCh := bus.SubscribeToEditEvents("edit-subscriber")
	watchCh := bus.SubscribeToChangeWatchEvents("watch-subscriber")

	testDoc := createTestDocHandle()

	// Publish different event types
	go func() {
		bus.PublishOpenEvent(OpenEvent{
			Doc:        testDoc,
			LanguageID: "hcl",
		})
		bus.PublishEditEvent(EditEvent{
			Doc:        testDoc,
			LanguageID: "hcl",
		})
		bus.PublishChangeWatchEvent(ChangeWatchEvent{
			RawPath:    "/test/path",
			IsDir:      true,
			ChangeType: protocol.Created,
		})
	}()

	// Verify each event type is received on the correct channel
	receivedOpen := false
	receivedEdit := false
	receivedWatch := false

	timeout := time.After(2 * time.Second)
	for !receivedOpen || !receivedEdit || !receivedWatch {
		select {
		case event := <-openCh:
			assert.Equal(t, testDoc.Filename, event.Doc.Filename)
			receivedOpen = true
		case event := <-editCh:
			assert.Equal(t, testDoc.Filename, event.Doc.Filename)
			receivedEdit = true
		case event := <-watchCh:
			assert.Equal(t, "/test/path", event.RawPath)
			assert.True(t, event.IsDir)
			receivedWatch = true
		case <-timeout:
			t.Fatal("timeout waiting for events")
		}
	}

	assert.True(t, receivedOpen, "should receive open event")
	assert.True(t, receivedEdit, "should receive edit event")
	assert.True(t, receivedWatch, "should receive watch event")
}

func TestChangeWatchEventTypes(t *testing.T) {
	bus := New()
	ch := bus.SubscribeToChangeWatchEvents("test-subscriber")

	testCases := []struct {
		name       string
		event      ChangeWatchEvent
		wantPath   string
		wantIsDir  bool
		wantChange protocol.FileChangeType
	}{
		{
			name: "file created",
			event: ChangeWatchEvent{
				RawPath:    "/test/file.hcl",
				IsDir:      false,
				ChangeType: protocol.Created,
			},
			wantPath:   "/test/file.hcl",
			wantIsDir:  false,
			wantChange: protocol.Created,
		},
		{
			name: "directory changed",
			event: ChangeWatchEvent{
				RawPath:    "/test/dir",
				IsDir:      true,
				ChangeType: protocol.Changed,
			},
			wantPath:   "/test/dir",
			wantIsDir:  true,
			wantChange: protocol.Changed,
		},
		{
			name: "file deleted",
			event: ChangeWatchEvent{
				RawPath:    "/test/deleted.hcl",
				IsDir:      false,
				ChangeType: protocol.Deleted,
			},
			wantPath:   "/test/deleted.hcl",
			wantIsDir:  false,
			wantChange: protocol.Deleted,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			go bus.PublishChangeWatchEvent(tc.event)

			select {
			case received := <-ch:
				assert.Equal(t, tc.wantPath, received.RawPath)
				assert.Equal(t, tc.wantIsDir, received.IsDir)
				assert.Equal(t, tc.wantChange, received.ChangeType)
			case <-time.After(1 * time.Second):
				t.Fatal("timeout waiting for change watch event")
			}
		})
	}
}

func TestPublishDiagnosticsEvent(t *testing.T) {
	bus := New()
	ch := bus.SubscribeToDiagnosticsEvents("test-subscriber")

	testDoc := createTestDocHandle()
	diags := hcl.Diagnostics{
		{
			Severity: hcl.DiagError,
			Summary:  "Test error",
			Detail:   "Test detail",
			Subject: &hcl.Range{
				Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
				End:      hcl.Pos{Line: 1, Column: 10, Byte: 9},
				Filename: "test.hcl",
			},
		},
	}

	event := DiagnosticsEvent{
		Doc:   testDoc,
		Diags: diags,
	}

	go bus.PublishDiagnosticsEvent(event)

	select {
	case received := <-ch:
		assert.Equal(t, testDoc.Filename, received.Doc.Filename)
		assert.Equal(t, testDoc.Dir.Path(), received.Doc.Dir.Path())
		require.Len(t, received.Diags, 1)
		assert.Equal(t, hcl.DiagError, received.Diags[0].Severity)
		assert.Equal(t, "Test error", received.Diags[0].Summary)
		assert.Equal(t, "Test detail", received.Diags[0].Detail)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for diagnostics event")
	}
}

func TestPublishDiagnosticsEvent_EmptyDiags(t *testing.T) {
	bus := New()
	ch := bus.SubscribeToDiagnosticsEvents("test-subscriber")

	testDoc := createTestDocHandle()
	event := DiagnosticsEvent{
		Doc:   testDoc,
		Diags: nil,
	}

	go bus.PublishDiagnosticsEvent(event)

	select {
	case received := <-ch:
		assert.Equal(t, testDoc.Filename, received.Doc.Filename)
		assert.Nil(t, received.Diags)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for diagnostics event")
	}
}

func TestPublishDiagnosticsEvent_MultipleDiags(t *testing.T) {
	bus := New()
	ch := bus.SubscribeToDiagnosticsEvents("test-subscriber")

	testDoc := createTestDocHandle()
	diags := hcl.Diagnostics{
		{
			Severity: hcl.DiagError,
			Summary:  "Error 1",
		},
		{
			Severity: hcl.DiagWarning,
			Summary:  "Warning 1",
		},
		{
			Severity: hcl.DiagError,
			Summary:  "Error 2",
		},
	}

	event := DiagnosticsEvent{
		Doc:   testDoc,
		Diags: diags,
	}

	go bus.PublishDiagnosticsEvent(event)

	select {
	case received := <-ch:
		require.Len(t, received.Diags, 3)
		assert.Equal(t, "Error 1", received.Diags[0].Summary)
		assert.Equal(t, "Warning 1", received.Diags[1].Summary)
		assert.Equal(t, "Error 2", received.Diags[2].Summary)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for diagnostics event")
	}
}

func TestDiagnosticsEventMultipleSubscribers(t *testing.T) {
	bus := New()

	ch1 := bus.SubscribeToDiagnosticsEvents("subscriber-1")
	ch2 := bus.SubscribeToDiagnosticsEvents("subscriber-2")

	testDoc := createTestDocHandle()
	diags := hcl.Diagnostics{
		{
			Severity: hcl.DiagError,
			Summary:  "Test error",
		},
	}

	event := DiagnosticsEvent{
		Doc:   testDoc,
		Diags: diags,
	}

	var wg sync.WaitGroup
	wg.Add(2)

	checkEvent := func(ch <-chan DiagnosticsEvent, name string) {
		defer wg.Done()
		select {
		case received := <-ch:
			assert.Equal(t, testDoc.Filename, received.Doc.Filename)
			require.Len(t, received.Diags, 1)
			assert.Equal(t, "Test error", received.Diags[0].Summary)
		case <-time.After(1 * time.Second):
			t.Errorf("timeout waiting for event on %s", name)
		}
	}

	go checkEvent(ch1, "subscriber-1")
	go checkEvent(ch2, "subscriber-2")

	go bus.PublishDiagnosticsEvent(event)

	wg.Wait()
}

func TestPublishNoCRDSourcesEvent(t *testing.T) {
	bus := New()
	ch := bus.SubscribeToNoCRDSourcesEvents("test-subscriber")

	event := NoCRDSourcesEvent{
		Dir: "/test/workspace",
	}

	go bus.PublishNoCRDSourcesEvent(event)

	select {
	case received := <-ch:
		assert.Equal(t, "/test/workspace", received.Dir)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for no CRD sources event")
	}
}

func TestNoCRDSourcesEventMultipleSubscribers(t *testing.T) {
	bus := New()

	ch1 := bus.SubscribeToNoCRDSourcesEvents("subscriber-1")
	ch2 := bus.SubscribeToNoCRDSourcesEvents("subscriber-2")

	event := NoCRDSourcesEvent{
		Dir: "/test/workspace",
	}

	var wg sync.WaitGroup
	wg.Add(2)

	checkEvent := func(ch <-chan NoCRDSourcesEvent, name string) {
		defer wg.Done()
		select {
		case received := <-ch:
			assert.Equal(t, "/test/workspace", received.Dir)
		case <-time.After(1 * time.Second):
			t.Errorf("timeout waiting for event on %s", name)
		}
	}

	go checkEvent(ch1, "subscriber-1")
	go checkEvent(ch2, "subscriber-2")

	go bus.PublishNoCRDSourcesEvent(event)

	wg.Wait()
}
