// Package events provides typed in-process dispatch and a transport facade for
// distributed pub/sub drivers. Event payloads use JSON by default, handlers run
// in registration order on the synchronous bus, and context-bound handles share
// one concurrency-safe subscription registry.
package events
