package middleware

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
)

func md5Base64(content string) string {
	hash := md5.New()
	hash.Write([]byte(content))
	return base64.StdEncoding.EncodeToString(hash.Sum(nil))
}

func NewSign(uri, body, timestamp, appSecret string) string {
	s := fmt.Sprintf("%s%s%s%s", uri, body, timestamp, appSecret)
	hash := md5Base64(s)
	log.Infof("input:%s", s)
	return base64.StdEncoding.EncodeToString([]byte(hash))
}

func VerifySign(uri, body, timestamp, appSecret, sig string, duration int64) bool {
	if sig == "" {
		return false
	}

	inTimestamp, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		log.Errorf("invalid signature timestamp %s", timestamp)
		return false
	}
	now := time.Now().Unix()

	if now > inTimestamp+duration {
		return false
	}

	verifyCode := NewSign(uri, body, timestamp, appSecret)
	if verifyCode != sig {
		log.Errorf("invalid signature, input:%s, want:%s", sig, verifyCode)
		return false
	}

	return true
}

const (
	SignCode      = "X-Sign"
	SignTimestamp = "X-Sign-Timestamp"
	SignAppid     = "X-App-Id"
)

type Authorization struct {
	appid    string
	secret   string
	duration int64
}

func (a *Authorization) Verify(ctx *fasthttp.RequestCtx) bool {
	appid := string(ctx.Request.Header.Peek(SignAppid))
	if appid != a.appid {
		return false
	}

	code := string(ctx.Request.Header.Peek(SignCode))
	timestamp := string(ctx.Request.Header.Peek(SignTimestamp))

	var body string
	if string(ctx.Method()) != fasthttp.MethodGet {
		body = string(ctx.PostBody())
	}
	uri := string(ctx.URI().Path())

	return VerifySign(uri, body, timestamp, a.secret, code, a.duration)
}

func (a *Authorization) Handle() func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(h fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			if a.Verify(ctx) {
				h(ctx)
			} else {
				ctx.Response.SetStatusCode(fasthttp.StatusForbidden)
			}
		}
	}
}