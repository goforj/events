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
	eventType    reflect.Type
	fn           reflect.Value
	expectsCtx   bool
	returnsError bool
}

var nextHandlerID atomic.Uint64

func newRegisteredHandler(handler any) (registeredHandler, error) {
	if handler == nil {
		return registeredHandler{}, ErrInvalidHandler
	}
	fn := reflect.ValueOf(handler)
	typ := fn.Type()
	if typ.Kind() != reflect.Func {
		return registeredHandler{}, fmt.Errorf("%w: handler must be a function", ErrInvalidHandler)
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
		if !typ.In(0).Implements(contextType) {
			return registeredHandler{}, fmt.Errorf("%w: first argument must implement context.Context", ErrInvalidHandler)
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
	if !h.returnsError || len(out) == 0 || out[0].IsNil() {
		return nil
	}
	return out[0].Interface().(error)
}

func (h registeredHandler) decodePayload(codec Codec, payload []byte) (reflect.Value, error) {
	target := h.eventType
	decodeTarget := reflect.New(indirectType(target))
	if err := codec.Unmarshal(payload, decodeTarget.Interface()); err != nil {
		return reflect.Value{}, err
	}
	if target.Kind() == reflect.Pointer {
		return decodeTarget, nil
	}
	return decodeTarget.Elem(), nil
}

func sampleEventValue(typ reflect.Type) reflect.Value {
	if typ.Kind() == reflect.Pointer {
		return reflect.New(indirectType(typ))
	}
	return reflect.Zero(typ)
}
