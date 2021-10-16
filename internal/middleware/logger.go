package middleware

import (
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
)

type Logger struct {
}

func NewLogger() *Logger{
	return &Logger{}
}

func (l *Logger) Handle() func (next fasthttp.RequestHandler) fasthttp.RequestHandler{
	return func(h fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			startTime := time.Now()
			h(ctx)

			if ctx.Response.StatusCode() < 400 {
				log.Infof("user access code %d time %v, method %s path %s client-real-ip %s",
				ctx.Response.StatusCode(), time.Since(startTime), string(ctx.Method()), ctx.Path(),ctx.Request.Header.Peek("X-Real-IP"))
			} else {
				log.Warnf("user access code %d time %v, method %s path %s client-real-ip %s",
					ctx.Response.StatusCode(), time.Since(startTime), string(ctx.Method()), ctx.Path(),ctx.Request.Header.Peek("X-Real-IP"))
			}
		}
	}
}
