package middleware

import (
	"github.com/valyala/fasthttp"
)

type Middleware interface {
	Handle() func(next fasthttp.RequestHandler) fasthttp.RequestHandler
}

type Middlewares struct {
	Handles []Middleware
}

var DefaultMiddlewares = Middlewares{
	Handles: []Middleware{&Logger{}, &Recovery{}},
}

// NewMiddlewares returns middlewares
func NewMiddlewares(middlewares ...Middleware) Middlewares {
	return Middlewares{Handles: append([]Middleware{}, middlewares...)}
}

func (m Middlewares) Apply(h fasthttp.RequestHandler) fasthttp.RequestHandler {
	for i := len(m.Handles) - 1; i > -1; i-- {
		invoke := m.Handles[i].Handle()
		h = invoke(h)
	}
	return h
}

// Append copy all middleware layers to newLayers, then append middlewares to newLayers, then return a new middleware onion.
func (m Middlewares) Append(middlewares ...Middleware) Middlewares {
	m.Handles = append(m.Handles, middlewares...)
	return m
}
