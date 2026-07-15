package events

import (
	"context"
	"fmt"
	"reflect"
	"sync/atomic"
)

var (
	contextType = reflect.TypeOf((*context.Context)(nil)).Elem()
	errorType   = reflect.TypeOf((*error)(nil)).Elem()
)

type registeredHandler struct {
	id           uint64
	topic        string
	generation   *transportGeneration
	eventType    reflect.Type
	fn           reflect.Value
	expectsCtx   bool
	returnsError bool
}

var nextHandlerID atomic.Uint64

// newRegisteredHandler validates the complete handler signature before registration mutates bus state.
func newRegisteredHandler(handler any) (registeredHandler, error) {
	if handler == nil {
		return registeredHandler{}, ErrInvalidHandler
	}
	fn := reflect.ValueOf(handler)
	typ := fn.Type()
	if typ.Kind() != reflect.Func {
		return registeredHandler{}, fmt.Errorf("%w: handler must be a function", ErrInvalidHandler)
	}
	if fn.IsNil() {
		return registeredHandler{}, fmt.Errorf("%w: handler function must not be nil", ErrInvalidHandler)
	}
	if typ.NumIn() < 1 || typ.NumIn() > 2 {
		return registeredHandler{}, fmt.Errorf("%w: handler must accept 1 or 2 arguments", ErrInvalidHandler)
	}
	if typ.NumOut() > 1 {
		return registeredHandler{}, fmt.Errorf("%w: handler must return zero or one value", ErrInvalidHandler)
	}

	h := registeredHandler{id: nextHandlerID.Add(1), fn: fn}
	eventIndex := 0
	if typ.NumIn() == 2 {
		if typ.In(0) != contextType {
			return registeredHandler{}, fmt.Errorf("%w: first argument must be context.Context", ErrInvalidHandler)
		}
		h.expectsCtx = true
		eventIndex = 1
	}
	h.eventType = typ.In(eventIndex)
	if h.eventType.Kind() == reflect.Interface {
		return registeredHandler{}, fmt.Errorf("%w: event argument must be a concrete type", ErrInvalidHandler)
	}
	if typ.NumOut() == 1 {
		if !typ.Out(0).Implements(errorType) {
			return registeredHandler{}, fmt.Errorf("%w: return value must be error", ErrInvalidHandler)
		}
		h.returnsError = true
	}

	sample := sampleEventValue(h.eventType)
	topic, _, err := resolveTopic(sample.Interface())
	if err != nil {
		return registeredHandler{}, fmt.Errorf("%w: resolve handler topic: %v", ErrInvalidHandler, err)
	}
	h.topic = topic
	return h, nil
}

// invoke adapts the supported handler signatures after their shapes have been validated.
func (h registeredHandler) invoke(ctx context.Context, value reflect.Value) error {
	args := make([]reflect.Value, 0, 2)
	if h.expectsCtx {
		if ctx == nil {
			ctx = context.Background()
		}
		args = append(args, reflect.ValueOf(ctx))
	}
	args = append(args, value)
	out := h.fn.Call(args)
	if !h.returnsError || len(out) == 0 {
		return nil
	}
	if isNilableKind(out[0].Kind()) && out[0].IsNil() {
		return nil
	}
	return out[0].Interface().(error)
}

// decodePayload asks the codec to populate the exact handler parameter type.
func (h registeredHandler) decodePayload(codec Codec, payload []byte) (reflect.Value, error) {
	decodeTarget := reflect.New(h.eventType)
	if err := codec.Unmarshal(payload, decodeTarget.Interface()); err != nil {
		return reflect.Value{}, err
	}
	return decodeTarget.Elem(), nil
}

// sampleEventValue creates a non-nil outer pointer so topic resolution can inspect pointer handlers.
func sampleEventValue(typ reflect.Type) reflect.Value {
	if typ.Kind() == reflect.Pointer {
		value := reflect.New(typ.Elem())
		if value.Type() != typ {
			value = value.Convert(typ)
		}
		return value
	}
	return reflect.Zero(typ)
}

// isNilableKind identifies result kinds on which reflect.Value.IsNil is valid.
func isNilableKind(kind reflect.Kind) bool {
	switch kind {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return true
	default:
		return false
	}
}
