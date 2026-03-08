package events

import "encoding/json"

// Codec marshals and unmarshals event payloads.
//
// Example: define a custom codec
//
//	var codec events.Codec
//	fmt.Println(codec == nil)
//	// Output: true
type Codec interface {
	Marshal(v any) ([]byte, error)
	Unmarshal(data []byte, v any) error
}

type jsonCodec struct{}

func (jsonCodec) Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (jsonCodec) Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
