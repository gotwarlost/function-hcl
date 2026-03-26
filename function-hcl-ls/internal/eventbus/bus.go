// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package eventbus

import (
	"sync"
)

const channelSize = 10

// topic represents a generic subscription topic
type topic[T any] struct {
	l           sync.Mutex
	subscribers []subscriber[T]
}

// subscriber represents a subscriber to a topic
type subscriber[T any] struct {
	identifier string
	// channel is the channel to which all events of the topic are sent
	channel chan<- T
}

// newTopic creates a new topic
func newTopic[T any]() *topic[T] {
	return &topic[T]{}
}

// subscribe adds a subscriber to a topic
func (eb *topic[T]) subscribe(identifier string) <-chan T {
	channel := make(chan T, channelSize)
	eb.l.Lock()
	defer eb.l.Unlock()

	eb.subscribers = append(eb.subscribers, subscriber[T]{
		identifier: identifier,
		channel:    channel,
	})
	return channel
}

// publish sends an event to all subscribers of a specific topic
func (eb *topic[T]) publish(event T) {
	eb.l.Lock()
	defer eb.l.Unlock()
	for _, s := range eb.subscribers {
		s.channel <- event
	}
}
