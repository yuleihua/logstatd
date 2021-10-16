package middleware

import (
	"path/filepath"
	"runtime"

	log "github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
)

type Recovery struct {
}

func NewRecovery() *Recovery {
	return &Recovery{}
}

func (r *Recovery) Handle() func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(h fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Errorf("panic:%v", rec)
					for skip := 2; ; skip++ {
						pc, f, n, ok := runtime.Caller(skip)
						if !ok {
							break
						}
						log.Errorf("%v():%s:%d",
							filepath.Base(runtime.FuncForPC(pc).Name()), f, n)
					}
					ctx.Response.SetStatusCode(fasthttp.StatusInternalServerError)
					return
				}
			}()
			h(ctx)
		}
	}
}
