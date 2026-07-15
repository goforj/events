package events

import "encoding/json"

// Codec marshals and unmarshals event payloads.
// @group Options
//
// Example: define a custom codec
//
//	var codec events.Codec
//	fmt.Println(codec == nil)
//	// Output: true
type Codec interface {
	// Marshal encodes one event value for transport.
	Marshal(v any) ([]byte, error)
	// Unmarshal decodes one transport payload into a handler destination.
	Unmarshal(data []byte, v any) error
}

type jsonCodec struct{}

// Marshal uses the stable JSON payload format provided by the default codec.
func (jsonCodec) Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

// Unmarshal decodes the default JSON payload into the handler's exact destination type.
func (jsonCodec) Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
