package middleware

import "github.com/valyala/fasthttp"

// AuthFunc is your custom auth function type
type AuthFunc func(ctx *fasthttp.RequestCtx) bool

// AuthMiddleware accepts a customer auth function and then returns a middleware which only accepts auth passed request.
// If auth function returns false, it will term the HTTP request and response 403 status code
func AuthMiddleware(auth AuthFunc) func (next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(h fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			if auth(ctx) {
				h(ctx)
			} else {
				ctx.Response.SetStatusCode(fasthttp.StatusForbidden)
			}
		}
	}
}

