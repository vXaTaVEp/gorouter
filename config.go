package gorouter

import "time"

type Config interface {
	GetAddress() string
	GetReadTimeout() time.Duration
	GetWriteTimeout() time.Duration
	GetIdleTimeout() time.Duration
}
