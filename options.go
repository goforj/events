package events

// Option configures root bus behavior.
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
func WithCodec(codec Codec) Option {
	return func(o *options) {
		if codec != nil {
			o.codec = codec
		}
	}
}
