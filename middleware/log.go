package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"chihqiang/llm-gate/logic"
	"chihqiang/llm-gate/model"

	"github.com/chihqiang/infra-go/httpx"
	"github.com/chihqiang/infra-go/jwt"
	"github.com/chihqiang/infra-go/logger"
	"github.com/chihqiang/infra-go/trace"
)

const (
	maxReqBodySize  = 1 << 20 // 1MB
	maxRespBodySize = 1 << 20 // 1MB
)

var sensitiveKeys = map[string]bool{
	"password":      true,
	"token":         true,
	"access_token":  true,
	"refresh_token": true,
	"secret":        true,
}

type responseWriter struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
	size   int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.body.Len() < maxRespBodySize {
		n := maxRespBodySize - rw.body.Len()
		if n > len(b) {
			n = len(b)
		}
		rw.body.Write(b[:n])
	}
	rw.size += len(b)
	return rw.ResponseWriter.Write(b)
}

var logWorker = newLogWorker()

type logWorkerT struct {
	jobs chan func()
}

func newLogWorker() *logWorkerT {
	w := &logWorkerT{jobs: make(chan func(), 256)}
	for range 4 {
		go func() {
			for job := range w.jobs {
				job()
			}
		}()
	}
	return w
}

func (w *logWorkerT) Submit(job func()) {
	select {
	case w.jobs <- job:
	default:
		logger.Warn("log worker queue full, dropping log")
	}
}

func Log(logLogic *logic.LogLogic, skipRoutes []string, skipMethods []string) httpx.Middleware {
	skipMethodSet := make(map[string]bool, len(skipMethods))
	for _, m := range skipMethods {
		skipMethodSet[strings.ToUpper(m)] = true
	}

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if skipMethodSet[strings.ToUpper(r.Method)] {
				next(w, r)
				return
			}
			for _, route := range skipRoutes {
				if strings.HasPrefix(r.RequestURI, route) {
					next(w, r)
					return
				}
			}

			start := time.Now()

			var reqBody []byte
			if r.Body != nil {
				limitedBody := io.LimitReader(r.Body, maxReqBodySize+1)
				var readErr error
				reqBody, readErr = io.ReadAll(limitedBody)
				if readErr != nil {
					logger.WarnCtx(r.Context(), "log: read request body failed", logger.Err(readErr))
				}
				if len(reqBody) > maxReqBodySize {
					reqBody = reqBody[:maxReqBodySize]
				}
				r.Body = io.NopCloser(bytes.NewBuffer(reqBody))
			}

			rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
			next(rw, r)

			duration := time.Since(start)

			var accountID int64
			var accountName string
			if claims := jwt.ClaimsFromContext(r.Context()); claims != nil {
				if id, ok := claims[jwt.ClaimKeyUserID].(float64); ok {
					accountID = int64(id)
				}
				name, ok := claims[jwt.ClaimKeyUsername].(string)
				if ok {
					accountName = name
				}
			}

			ua := r.UserAgent()
			browser, os := parseUserAgent(ua)

			reqURI := r.RequestURI
			reqMethod := r.Method
			reqIP := getClientIP(r)
			respStatus := rw.status
			respBody := rw.body.String()
			reqPayload := sanitizePayload(string(reqBody))
			// 在请求期间捕获 context，供后台 worker 记录链路信息（读取 value 安全，不用于 DB）
			traceCtx := r.Context()

			logWorker.Submit(func() {
				if err := logLogic.Create(&model.Log{
					RequestPath:    reqURI,
					RequestMethod:  reqMethod,
					ResponseCode:   respStatus,
					RequestPayload: reqPayload,
					RequestIP:      reqIP,
					RequestOS:      os,
					RequestBrowser: browser,
					ResponseJSON:   truncateString(respBody, maxRespBodySize),
					ProcessTime:    duration.Milliseconds(),
					RequestID:      httpx.RequestIDFromContext(traceCtx),
					TraceID:        trace.TraceIDFromContext(traceCtx),
					AccountID:      accountID,
					AccountName:    accountName,
				}); err != nil {
					logger.ErrorCtx(traceCtx, "create log failed", logger.Err(err))
				}
			})
		}
	}
}

func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if idx := strings.IndexByte(xff, ','); idx > 0 {
			return strings.TrimSpace(xff[:idx])
		}
		return xff
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	addr := r.RemoteAddr
	if idx := strings.LastIndexByte(addr, ':'); idx > 0 {
		return addr[:idx]
	}
	return addr
}

func parseUserAgent(ua string) (browser, os string) {
	ua = strings.TrimSpace(ua)
	if ua == "" {
		return "", ""
	}

	os = parseOS(ua)
	browser = parseBrowser(ua)
	return browser, os
}

func parseOS(ua string) string {
	switch {
	case strings.Contains(ua, "Windows NT 10"):
		return "Windows 10/11"
	case strings.Contains(ua, "Windows NT 6.3"):
		return "Windows 8.1"
	case strings.Contains(ua, "Windows NT 6.2"):
		return "Windows 8"
	case strings.Contains(ua, "Windows NT 6.1"):
		return "Windows 7"
	case strings.Contains(ua, "Windows"):
		return "Windows"
	case strings.Contains(ua, "Mac OS X"):
		if idx := strings.Index(ua, "Mac OS X"); idx >= 0 {
			end := strings.Index(ua[idx:], ")")
			if end > 0 {
				return strings.ReplaceAll(ua[idx:idx+end], "_", ".")
			}
		}
		return "Mac OS X"
	case strings.Contains(ua, "Android"):
		if idx := strings.Index(ua, "Android"); idx >= 0 {
			rest := ua[idx:]
			end := strings.IndexByte(rest, ';')
			if end > 0 {
				return strings.TrimSpace(rest[:end])
			}
			return rest
		}
		return "Android"
	case strings.Contains(ua, "iPhone OS"):
		if idx := strings.Index(ua, "iPhone OS"); idx >= 0 {
			end := strings.Index(ua[idx:], " ")
			if end > 0 {
				return ua[idx : idx+end]
			}
		}
		return "iOS"
	case strings.Contains(ua, "iPad"):
		return "iPadOS"
	case strings.Contains(ua, "Linux"):
		return "Linux"
	case strings.Contains(ua, "CrOS"):
		return "ChromeOS"
	default:
		return ""
	}
}

func parseBrowser(ua string) string {
	switch {
	case strings.Contains(ua, "Edg/"):
		return "Edge"
	case strings.Contains(ua, "OPR/") || strings.Contains(ua, "Opera"):
		return "Opera"
	case strings.Contains(ua, "Chrome/") && !strings.Contains(ua, "Edg/"):
		return "Chrome"
	case strings.Contains(ua, "Firefox/"):
		return "Firefox"
	case strings.Contains(ua, "Safari/") && strings.Contains(ua, "Version/"):
		return "Safari"
	case strings.Contains(ua, "MSIE") || strings.Contains(ua, "Trident/"):
		return "IE"
	case strings.Contains(ua, "curl/"):
		return "curl"
	case strings.Contains(ua, "PostmanRuntime"):
		return "Postman"
	case strings.Contains(ua, "Go-http-client"):
		return "Go-http-client"
	default:
		return ""
	}
}

func sanitizePayload(payload string) string {
	if payload == "" {
		return ""
	}
	payload = strings.TrimSpace(payload)

	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(payload), &raw); err != nil {
		return truncateString(payload, 4096)
	}

	for key, val := range raw {
		if sensitiveKeys[strings.ToLower(key)] {
			raw[key] = json.RawMessage(`"***"`)
		} else {
			s := string(val)
			if len(s) > 4096 {
				raw[key] = json.RawMessage(truncateString(s, 4096))
			}
		}
	}

	b, err := json.Marshal(raw)
	if err != nil {
		return truncateString(payload, 4096)
	}
	return string(b)
}

func truncateString(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen]
	}
	return s
}
