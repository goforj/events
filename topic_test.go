package events

import "testing"

type userCreated struct{}

type customTopicEvent struct{}

// Topic supplies the explicit topic used to verify TopicEvent overrides.
func (customTopicEvent) Topic() string { return "custom.topic" }

// TestResolveTopicUsesTypeName verifies unnamed events derive topics from their concrete type.
func TestResolveTopicUsesTypeName(t *testing.T) {
	topic, _, err := resolveTopic(userCreated{})
	if err != nil {
		t.Fatalf("resolveTopic returned error: %v", err)
	}
	if topic != "user.created" {
		t.Fatalf("topic = %q, want %q", topic, "user.created")
	}
}

// TestResolveTopicUsesOverride verifies TopicEvent implementations control their published topic.
func TestResolveTopicUsesOverride(t *testing.T) {
	topic, _, err := resolveTopic(customTopicEvent{})
	if err != nil {
		t.Fatalf("resolveTopic returned error: %v", err)
	}
	if topic != "custom.topic" {
		t.Fatalf("topic = %q, want %q", topic, "custom.topic")
	}
}

// TestResolveTopicRejectsNil verifies nil events cannot produce a topic.
func TestResolveTopicRejectsNil(t *testing.T) {
	var event *userCreated
	if _, _, err := resolveTopic(event); err != ErrNilEvent {
		t.Fatalf("resolveTopic error = %v, want %v", err, ErrNilEvent)
	}
}
