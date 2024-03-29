package middleware

// copyright AdhityaRamadhanus
// this file coming from : https://github.com/AdhityaRamadhanus/fasthttpcors/blob/master/cors.go

import (
	"net/http"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
)

// Options is struct that defined cors properties
type Options struct {
	AllowedOrigins   []string
	AllowedHeaders   []string
	AllowMaxAge      int
	AllowedMethods   []string
	ExposedHeaders   []string
	AllowCredentials bool
	Debug            bool
}

type CorsHandler struct {
	allowedOriginsAll bool
	allowedOrigins    []string
	allowedHeadersAll bool
	allowedHeaders    []string
	allowedMethods    []string
	exposedHeaders    []string
	allowCredentials  bool
	maxAge            int
}

var defaultOptions = &Options{
	AllowedOrigins: []string{"*"},
	AllowedHeaders: []string{"*"},
	AllowedMethods: []string{
		http.MethodOptions,
		http.MethodHead,
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
	},
}

func NewOption(origins []string) *Options {
	return &Options{
		AllowedOrigins: origins,
		AllowedHeaders: []string{"*"},
		AllowedMethods: []string{
			http.MethodOptions,
			http.MethodHead,
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
		},
		AllowCredentials: false,
		Debug:            false,
	}
}

func DefaultCorsHandler() *CorsHandler {
	return NewCorsHandler(*defaultOptions)
}

func NewCorsHandler(options Options) *CorsHandler {
	cors := &CorsHandler{
		allowedOrigins:   options.AllowedOrigins,
		allowedHeaders:   options.AllowedHeaders,
		allowCredentials: options.AllowCredentials,
		allowedMethods:   options.AllowedMethods,
		exposedHeaders:   options.ExposedHeaders,
		maxAge:           options.AllowMaxAge,
	}

	if len(cors.allowedOrigins) == 0 {
		cors.allowedOrigins = defaultOptions.AllowedOrigins
		cors.allowedOriginsAll = true
	} else {
		for _, v := range options.AllowedOrigins {
			if v == "*" {
				cors.allowedOrigins = defaultOptions.AllowedOrigins
				cors.allowedOriginsAll = true
				break
			}
		}
	}
	if len(cors.allowedHeaders) == 0 {
		cors.allowedHeaders = defaultOptions.AllowedHeaders
		cors.allowedHeadersAll = true
	} else {
		for _, v := range options.AllowedHeaders {
			if v == "*" {
				cors.allowedHeadersAll = true
				break
			}
		}
	}
	if len(cors.allowedMethods) == 0 {
		cors.allowedMethods = defaultOptions.AllowedMethods
	}
	return cors
}

func (c *CorsHandler) Handle() func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(h fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			if string(ctx.Method()) == "OPTIONS" {
				c.handlePreflight(ctx)
				ctx.SetStatusCode(200)
			} else {
				c.handleActual(ctx)
				h(ctx)
			}
		}
	}
}

func (c *CorsHandler) handlePreflight(ctx *fasthttp.RequestCtx) {
	originHeader := string(ctx.Request.Header.Peek("Origin"))
	if len(originHeader) == 0 || c.isAllowedOrigin(originHeader) == false {
		log.Errorf("Origin %s  is not in %v", originHeader, c.allowedOrigins)
		return
	}

	method := string(ctx.Request.Header.Peek("Access-Control-Request-Method"))
	if !c.isAllowedMethod(method) {
		log.Errorf("Method %s is not in %v", method, c.allowedMethods)
		return
	}

	var headers []string
	if len(ctx.Request.Header.Peek("Access-Control-Request-Headers")) > 0 {
		headers = strings.Split(string(ctx.Request.Header.Peek("Access-Control-Request-Headers")), ",")
	}
	if !c.areHeadersAllowed(headers) {
		log.Errorf("Headers is %v not in %v", headers, c.allowedHeaders)
		return
	}

	ctx.Response.Header.Set("Access-Control-Allow-Origin", originHeader)
	ctx.Response.Header.Set("Access-Control-Allow-Methods", method)
	if len(headers) > 0 {
		ctx.Response.Header.Set("Access-Control-Allow-Headers", strings.Join(headers, ", "))
	}
	if c.allowCredentials {
		ctx.Response.Header.Set("Access-Control-Allow-Credentials", "true")
	}
	if c.maxAge > 0 {
		ctx.Response.Header.Set("Access-Control-Max-Age", strconv.Itoa(c.maxAge))
	}
}

func (c *CorsHandler) handleActual(ctx *fasthttp.RequestCtx) {
	originHeader := string(ctx.Request.Header.Peek("Origin"))
	if len(originHeader) == 0 || c.isAllowedOrigin(originHeader) == false {
		log.Errorf("Origin %v is not in %v", originHeader, c.allowedOrigins)
		return
	}

	ctx.Response.Header.Set("Access-Control-Allow-Origin", originHeader)
	if len(c.exposedHeaders) > 0 {
		ctx.Response.Header.Set("Access-Control-Expose-Headers", strings.Join(c.exposedHeaders, ", "))
	}
	if c.allowCredentials {
		ctx.Response.Header.Set("Access-Control-Allow-Credentials", "true")
	}
}

func (c *CorsHandler) isAllowedOrigin(originHeader string) bool {
	if c.allowedOriginsAll {
		return true
	}
	for _, val := range c.allowedOrigins {
		if val == originHeader {
			return true
		}
	}
	return false
}

func (c *CorsHandler) isAllowedMethod(methodHeader string) bool {
	if len(c.allowedMethods) == 0 {
		return false
	}
	if methodHeader == "OPTIONS" {
		return true
	}
	for _, m := range c.allowedMethods {
		if m == methodHeader {
			return true
		}
	}
	return false
}

func (c *CorsHandler) areHeadersAllowed(headers []string) bool {
	if c.allowedHeadersAll || len(headers) == 0 {
		return true
	}
	for _, header := range headers {
		found := false
		for _, h := range c.allowedHeaders {
			if h == header {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	return true
}
