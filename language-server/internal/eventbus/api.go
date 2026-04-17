// Package eventbus provides facilities for asynchronous processing of events
// mediated by a type-safe bus.
package eventbus

import (
	"log"

	"github.com/crossplane-contrib/function-hcl/language-server/internal/document"
	"github.com/crossplane-contrib/function-hcl/language-server/internal/langserver/protocol"
	"github.com/crossplane-contrib/function-hcl/language-server/internal/utils/logging"
	"github.com/hashicorp/hcl/v2"
)

// EditEvent is an event to signal that a file in a directory has changed.
//
// It is usually emitted when a document is changed via a language server
// text edit event.
type EditEvent struct {
	Doc        document.Handle
	LanguageID string
}

// ChangeWatchEvent is the event that is emitted when a client notifies
// the language server that a directory or file was changed outside the
// editor.
type ChangeWatchEvent struct {
	// RawPath contains an OS specific path to the file or directory that was
	// changed. Usually extracted from the URI.
	RawPath string
	// IsDir is true if we were able to determine that the path is a directory.
	// This is not set for delete events.
	IsDir bool
	// ChangeType specifies if the file or directory was created, updated, or deleted.
	ChangeType protocol.FileChangeType
}

// OpenEvent is an event to signal that a document is open in the editor.
//
// It is usually emitted when a document is opened via a language server
// text synchronization event.
type OpenEvent struct {
	Doc        document.Handle
	LanguageID string
}

// DiagnosticsEvent is an event to signal that diagnostics are available for a file.
//
// It is emitted after parsing a file to notify the language server
// to publish diagnostics to the client.
type DiagnosticsEvent struct {
	Doc   document.Handle
	Diags hcl.Diagnostics
}

// NoCRDSourcesEvent is an event to signal that no CRD sources were found
// for a workspace directory. This is used to prompt the user to configure
// CRD sources.
type NoCRDSourcesEvent struct {
	Dir string // workspace directory with no CRD config
}

// EventBus is a simple event bus that allows for subscribing to and publishing
// events of a specific type.
//
// It has a static list of topics. Each topic can have multiple subscribers.
// When an event is published to a topic, it is sent to all subscribers.
type EventBus struct {
	logger            *log.Logger
	openTopic         *topic[OpenEvent]
	changeTopic       *topic[EditEvent]
	changeWatchTopic  *topic[ChangeWatchEvent]
	diagnosticsTopic  *topic[DiagnosticsEvent]
	noCRDSourcesTopic *topic[NoCRDSourcesEvent]
}

// New creates an event bus.
func New() *EventBus {
	return &EventBus{
		logger:            logging.LoggerFor(logging.ModuleEventBus),
		openTopic:         newTopic[OpenEvent](),
		changeTopic:       newTopic[EditEvent](),
		changeWatchTopic:  newTopic[ChangeWatchEvent](),
		diagnosticsTopic:  newTopic[DiagnosticsEvent](),
		noCRDSourcesTopic: newTopic[NoCRDSourcesEvent](),
	}
}

// PublishOpenEvent publishes a document open event.
func (b *EventBus) PublishOpenEvent(e OpenEvent) {
	b.logger.Printf("bus: -> publish open event %s %s", e.Doc.Dir.Path(), e.Doc.Filename)
	b.openTopic.publish(e)
}

// SubscribeToOpenEvents adds a subscriber to process a document open event.
func (b *EventBus) SubscribeToOpenEvents(identifier string) <-chan OpenEvent {
	b.logger.Printf("bus: %q subscribe to open events", identifier)
	return b.openTopic.subscribe(identifier)
}

// PublishEditEvent publishes a document edit event.
func (b *EventBus) PublishEditEvent(e EditEvent) {
	b.logger.Printf("bus: -> publish change event %s %s", e.Doc.Dir.Path(), e.Doc.Filename)
	b.changeTopic.publish(e)
}

// SubscribeToEditEvents adds a subscriber to process document edit events.
func (b *EventBus) SubscribeToEditEvents(identifier string) <-chan EditEvent {
	b.logger.Printf("bus: %q subscribed to change events", identifier)
	return b.changeTopic.subscribe(identifier)
}

// PublishChangeWatchEvent publishes a change watch event.
func (b *EventBus) PublishChangeWatchEvent(e ChangeWatchEvent) {
	b.logger.Printf("bus: -> publish change watch event %s", e.RawPath)
	b.changeWatchTopic.publish(e)
}

// SubscribeToChangeWatchEvents adds a subscriber to process change watch events.
func (b *EventBus) SubscribeToChangeWatchEvents(identifier string) <-chan ChangeWatchEvent {
	b.logger.Printf("bus: %q subscribed to change watch events", identifier)
	return b.changeWatchTopic.subscribe(identifier)
}

// PublishDiagnosticsEvent publishes a diagnostics event.
func (b *EventBus) PublishDiagnosticsEvent(e DiagnosticsEvent) {
	b.logger.Printf("bus: -> publish diagnostics event %s %s (%d diags)", e.Doc.Dir.Path(), e.Doc.Filename, len(e.Diags))
	b.diagnosticsTopic.publish(e)
}

// SubscribeToDiagnosticsEvents adds a subscriber to process diagnostics events.
func (b *EventBus) SubscribeToDiagnosticsEvents(identifier string) <-chan DiagnosticsEvent {
	b.logger.Printf("bus: %q subscribed to diagnostics events", identifier)
	return b.diagnosticsTopic.subscribe(identifier)
}

// PublishNoCRDSourcesEvent publishes a no CRD sources event.
func (b *EventBus) PublishNoCRDSourcesEvent(e NoCRDSourcesEvent) {
	b.logger.Printf("bus: -> publish no CRD sources event %s", e.Dir)
	b.noCRDSourcesTopic.publish(e)
}

// SubscribeToNoCRDSourcesEvents adds a subscriber to process no CRD sources events.
func (b *EventBus) SubscribeToNoCRDSourcesEvents(identifier string) <-chan NoCRDSourcesEvent {
	b.logger.Printf("bus: %q subscribed to no CRD sources events", identifier)
	return b.noCRDSourcesTopic.subscribe(identifier)
}
