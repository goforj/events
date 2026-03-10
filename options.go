package events

// Option configures root bus behavior.
// @group Options
//
// Example: keep an option for later bus construction
//
//	opt := events.WithCodec(nil)
//	fmt.Println(opt != nil)
//	// Output: true
type Option func(*options)

type options struct {
	codec Codec
}

func (o *options) apply(opts []Option) {
	for _, opt := range opts {
		if opt != nil {
			opt(o)
		}
	}
}

// WithCodec overrides the default event codec.
// @group Options
//
// Example: construct a bus with a custom codec
//
//	bus, _ := events.NewSync(events.WithCodec(nil))
//	fmt.Println(bus.Driver())
//	// Output: sync
func WithCodec(codec Codec) Option {
	return func(o *options) {
		if codec != nil {
			o.codec = codec
		}
	}
}
