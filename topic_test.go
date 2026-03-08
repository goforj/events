package events

import "testing"

type userCreated struct{}

type customTopicEvent struct{}

func (customTopicEvent) Topic() string { return "custom.topic" }

func TestResolveTopicUsesTypeName(t *testing.T) {
	topic, _, err := resolveTopic(userCreated{})
	if err != nil {
		t.Fatalf("resolveTopic returned error: %v", err)
	}
	if topic != "user.created" {
		t.Fatalf("topic = %q, want %q", topic, "user.created")
	}
}

func TestResolveTopicUsesOverride(t *testing.T) {
	topic, _, err := resolveTopic(customTopicEvent{})
	if err != nil {
		t.Fatalf("resolveTopic returned error: %v", err)
	}
	if topic != "custom.topic" {
		t.Fatalf("topic = %q, want %q", topic, "custom.topic")
	}
}

func TestResolveTopicRejectsNil(t *testing.T) {
	var event *userCreated
	if _, _, err := resolveTopic(event); err != ErrNilEvent {
		t.Fatalf("resolveTopic error = %v, want %v", err, ErrNilEvent)
	}
}
