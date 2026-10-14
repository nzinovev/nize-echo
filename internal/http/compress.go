package http

import (
	"compress/gzip"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/andybalholm/brotli"
)

// minCompressSize is the smallest known body worth compressing.
const minCompressSize = 512

// compress encodes response bodies with brotli or gzip, whichever the client
// accepts (brotli preferred). Upgrades, HEAD requests, bodies that are already
// encoded and binary media types pass through unchanged.
func compress(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Accept-Encoding")
		enc := negotiateEncoding(r.Header.Get("Accept-Encoding"))
		if enc == "" || r.Method == http.MethodHead || r.Header.Get("Upgrade") != "" {
			next.ServeHTTP(w, r)
			return
		}
		cw := &compressWriter{ResponseWriter: w, encoding: enc}
		defer cw.close()
		next.ServeHTTP(cw, r)
	})
}

// negotiateEncoding picks "br", "gzip" or "" from an Accept-Encoding header.
func negotiateEncoding(header string) string {
	var br, gz bool
	for part := range strings.SplitSeq(header, ",") {
		name, params, _ := strings.Cut(strings.TrimSpace(part), ";")
		if q, ok := strings.CutPrefix(strings.TrimSpace(params), "q="); ok {
			if v, err := strconv.ParseFloat(q, 64); err == nil && v == 0 {
				continue
			}
		}
		switch strings.ToLower(strings.TrimSpace(name)) {
		case "br":
			br = true
		case "gzip":
			gz = true
		}
	}
	switch {
	case br:
		return "br"
	case gz:
		return "gzip"
	}
	return ""
}

// compressWriter decides on the first write whether to compress, based on
// the response headers the handler has set by then.
type compressWriter struct {
	http.ResponseWriter
	encoding    string
	enc         io.WriteCloser // nil until compression starts
	decided     bool
	wroteHeader bool
	status      int
}

func (cw *compressWriter) WriteHeader(code int) {
	if cw.wroteHeader {
		return
	}
	cw.wroteHeader = true
	cw.status = code
	// Informational and body-less responses go straight through.
	if code < http.StatusOK || code == http.StatusNoContent || code == http.StatusNotModified {
		cw.decided = true
		cw.ResponseWriter.WriteHeader(code)
	}
}

func (cw *compressWriter) Write(b []byte) (int, error) {
	if !cw.wroteHeader {
		cw.WriteHeader(http.StatusOK)
	}
	if !cw.decided {
		cw.decide(len(b))
	}
	if cw.enc != nil {
		return cw.enc.Write(b)
	}
	return cw.ResponseWriter.Write(b)
}

// decide starts compression if the response qualifies, then sends headers.
func (cw *compressWriter) decide(firstWrite int) {
	cw.decided = true
	h := cw.Header()
	if h.Get("Content-Encoding") == "" && compressible(h, firstWrite) {
		h.Set("Content-Encoding", cw.encoding)
		h.Del("Content-Length")
		if etag := h.Get("ETag"); etag != "" && !strings.HasPrefix(etag, "W/") {
			h.Set("ETag", "W/"+etag)
		}
		switch cw.encoding {
		case "br":
			cw.enc = brotli.NewWriterLevel(cw.ResponseWriter, 5)
		default:
			cw.enc = gzip.NewWriter(cw.ResponseWriter)
		}
	}
	cw.ResponseWriter.WriteHeader(cw.status)
}

// compressible reports whether a response with these headers is worth
// encoding.
func compressible(h http.Header, firstWrite int) bool {
	if cl := h.Get("Content-Length"); cl != "" {
		if n, err := strconv.Atoi(cl); err == nil && n < minCompressSize {
			return false
		}
	} else if firstWrite == 0 {
		return false
	}
	ct := h.Get("Content-Type")
	if ct == "" {
		return false
	}
	mt, _, err := mime.ParseMediaType(ct)
	if err != nil {
		return false
	}
	switch {
	case strings.HasPrefix(mt, "text/"):
		return mt != "text/event-stream"
	case mt == "image/svg+xml":
		return true
	case strings.HasPrefix(mt, "application/"):
		return mt == "application/json" || mt == "application/javascript" ||
			mt == "application/xml" || mt == "application/manifest+json" ||
			strings.HasSuffix(mt, "+xml") || strings.HasSuffix(mt, "+json")
	}
	return false
}

// Flush flushes encoded data so far to the client.
func (cw *compressWriter) Flush() {
	if !cw.decided && cw.wroteHeader {
		cw.decide(0)
	}
	if f, ok := cw.enc.(interface{ Flush() error }); ok {
		_ = f.Flush()
	}
	_ = http.NewResponseController(cw.ResponseWriter).Flush()
}

// Unwrap lets http.ResponseController reach the underlying writer.
func (cw *compressWriter) Unwrap() http.ResponseWriter { return cw.ResponseWriter }

// close finishes the encoded stream. If the handler wrote a status but no
// body, the status is sent now; if it wrote nothing at all, nothing is sent,
// so a recovering middleware outside can still write its own response.
func (cw *compressWriter) close() {
	if cw.wroteHeader && !cw.decided {
		cw.decided = true
		cw.ResponseWriter.WriteHeader(cw.status)
	}
	if cw.enc != nil {
		_ = cw.enc.Close()
	}
}
