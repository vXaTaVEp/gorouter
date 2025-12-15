package gorouter

import "context"

type Context interface {
	GetContext() context.Context
	GetConfig() Config
	GetClaims() Claims
}

// handlerContext 处理器上下文，包含配置等信息
type handlerContext struct {
	ctx    context.Context
	config Config
	claims Claims
}

func (h handlerContext) GetContext() context.Context {
	return h.ctx
}

func (h handlerContext) GetConfig() Config {
	return h.config
}

func (h handlerContext) GetClaims() Claims {
	return h.claims
}
