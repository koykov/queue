package victoria

import "time"

type Option func(writer *writer)

func WithPrecision(precision time.Duration) Option {
	return func(writer *writer) {
		writer.prec = precision
	}
}
