package events

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"
)

// TopicEvent overrides the derived topic for an event.
// @group Publish
type TopicEvent interface {
	Topic() string
}

func resolveTopic(event any) (string, reflect.Type, error) {
	if event == nil {
		return "", nil, ErrNilEvent
	}
	value := reflect.ValueOf(event)
	if isNilValue(value) {
		return "", nil, ErrNilEvent
	}
	typ := value.Type()
	base := indirectType(typ)
	if base.Name() == "" {
		return "", nil, fmt.Errorf("%w: unnamed event type %s", ErrEmptyTopic, typ)
	}

	if override, ok := event.(TopicEvent); ok {
		topic := strings.TrimSpace(override.Topic())
		if topic == "" {
			return "", nil, ErrEmptyTopic
		}
		return topic, typ, nil
	}

	topic := deriveTopic(base.Name())
	if topic == "" {
		return "", nil, ErrEmptyTopic
	}
	return topic, typ, nil
}

func deriveTopic(name string) string {
	parts := splitTypeWords(name)
	for i := range parts {
		parts[i] = strings.ToLower(parts[i])
	}
	return strings.Join(parts, ".")
}

func splitTypeWords(name string) []string {
	if name == "" {
		return nil
	}
	runes := []rune(name)
	start := 0
	var parts []string
	for i := 1; i < len(runes); i++ {
		prev := runes[i-1]
		curr := runes[i]
		nextLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
		if unicode.IsLower(prev) && unicode.IsUpper(curr) || (unicode.IsUpper(prev) && unicode.IsUpper(curr) && nextLower) {
			parts = append(parts, string(runes[start:i]))
			start = i
		}
	}
	parts = append(parts, string(runes[start:]))
	return parts
}

func indirectType(typ reflect.Type) reflect.Type {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	return typ
}

func isNilValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}
