package middleware

type MiddlewareOption struct {
	key   string
	value any
}

func WithRetryServerErrorsOption(option bool) MiddlewareOption {
	return MiddlewareOption{key: "RetryServerErrors", value: option}
}
