package httpapi

import (
	"bufio"
	"context"
	"crypto/rand"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"
)

type ctxKey int

const correlationKey ctxKey = 1

const correlationHeader = "X-Correlation-ID"

// correlationID accepts a client UUID v4 in X-Correlation-ID or mints one,
// echoes it on the response and stores it in the request context.
func correlationID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(correlationHeader)
		if !isUUIDv4(id) {
			id = newUUID()
		}
		w.Header().Set(correlationHeader, id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), correlationKey, id)))
	})
}

func correlationFrom(ctx context.Context) string {
	id, _ := ctx.Value(correlationKey).(string)
	return id
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (sw *statusWriter) WriteHeader(code int) {
	sw.status = code
	sw.ResponseWriter.WriteHeader(code)
}

// Unwrap lets http.ResponseController reach the underlying writer.
func (sw *statusWriter) Unwrap() http.ResponseWriter { return sw.ResponseWriter }

// Hijack is needed by the WebSocket upgrader.
func (sw *statusWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return http.NewResponseController(sw.ResponseWriter).Hijack()
}

func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		s.metrics.Inc("http_requests_total", "method", r.Method, "code", strconv.Itoa(sw.status/100)+"xx")
		s.log.Info("http", "correlation_id", correlationFrom(r.Context()), "method", r.Method,
			"path", r.URL.Path, "status", sw.status, "ms", time.Since(start).Milliseconds(), "ip", clientIP(r))
	})
}

func (s *Server) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				s.log.Error("panic", "correlation_id", correlationFrom(r.Context()), "path", r.URL.Path, "panic", v)
				writeErr(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong.")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// clientIP trusts X-Real-IP because the API only listens on 127.0.0.1 /
// the docker network behind nginx, which sets it from CF-Connecting-IP.
func clientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" && net.ParseIP(ip) != nil {
		return ip
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	return host
}

// newUUID returns a random RFC 4122 v4 UUID (never v1: no MAC/time leak).
func newUUID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
