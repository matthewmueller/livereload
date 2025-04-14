package livereload_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/livebud/sse"
	"github.com/matryer/is"
	"github.com/matthewmueller/livereload"
)

// Pulled from: https://github.com/mathiasbynens/small
// Built with: xxd -i small.ico
var favicon = []byte{
	0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x00,
	0x18, 0x00, 0x30, 0x00, 0x00, 0x00, 0x16, 0x00, 0x00, 0x00, 0x28, 0x00,
	0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x01, 0x00,
	0x18, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0xff, 0x00, 0x00, 0x00, 0x00, 0x00,
}

func contains(haystack, needle string) error {
	if strings.Contains(haystack, needle) {
		return nil
	}
	return fmt.Errorf("expected the following to contain %s:\n\n%s", needle, haystack)
}

func notContains(haystack, needle string) error {
	if !strings.Contains(haystack, needle) {
		return nil
	}
	return fmt.Errorf("expected the following to not contain %s:\n\n%s", needle, haystack)
}

func TestStatic(t *testing.T) {
	log := slog.Default()
	is := is.New(t)
	lr := livereload.New(log)
	fsys := fstest.MapFS{
		"index.html":  &fstest.MapFile{Data: []byte("<html><body>hello world</body></html>")},
		"error.txt":   &fstest.MapFile{Data: []byte("some error")},
		"index.css":   &fstest.MapFile{Data: []byte("body { color: red }")},
		"index.js":    &fstest.MapFile{Data: []byte("console.log('hello world')")},
		"favicon.ico": &fstest.MapFile{Data: favicon},
	}
	handler := lr.Middleware(http.FileServer(http.FS(fsys)))
	server := httptest.NewServer(handler)
	defer server.Close()

	// Test index.html
	res, err := http.Get(server.URL + "/index.html")
	is.NoErr(err)
	is.Equal(res.StatusCode, 200)
	is.Equal(res.Header.Get("Content-Type"), "text/html; charset=utf-8")
	is.Equal(res.Header.Get("Cache-Control"), "no-cache, no-store, must-revalidate")
	is.Equal(res.Header.Get("Last-Modified"), "0")
	body, err := io.ReadAll(res.Body)
	is.NoErr(err)
	is.NoErr(contains(string(body), "<html><body>hello world"))
	is.NoErr(contains(string(body), `new EventSource("/livereload")`))

	// Test error.txt
	res, err = http.Get(server.URL + "/error.txt")
	is.NoErr(err)
	is.Equal(res.StatusCode, 200)
	// Gets overwritten by livereload so the livereload script works
	is.Equal(res.Header.Get("Content-Type"), "text/plain; charset=utf-8")
	is.Equal(res.Header.Get("Cache-Control"), "")
	is.Equal(res.Header.Get("Last-Modified"), "")
	body, err = io.ReadAll(res.Body)
	is.NoErr(err)
	is.NoErr(contains(string(body), `some error`))
	is.NoErr(notContains(string(body), `new EventSource("/livereload")`))

	// Test index.css
	res, err = http.Get(server.URL + "/index.css")
	is.NoErr(err)
	is.Equal(res.StatusCode, 200)
	is.Equal(res.Header.Get("Content-Type"), "text/css; charset=utf-8")
	is.Equal(res.Header.Get("Cache-Control"), "")
	is.Equal(res.Header.Get("Last-Modified"), "")
	body, err = io.ReadAll(res.Body)
	is.NoErr(err)
	is.NoErr(contains(string(body), "body { color: red }"))
	is.NoErr(notContains(string(body), `new EventSource("/livereload")`))

	// Test index.js
	res, err = http.Get(server.URL + "/index.js")
	is.NoErr(err)
	is.Equal(res.StatusCode, 200)
	is.Equal(res.Header.Get("Content-Type"), "text/javascript; charset=utf-8")
	is.Equal(res.Header.Get("Cache-Control"), "")
	is.Equal(res.Header.Get("Last-Modified"), "")
	body, err = io.ReadAll(res.Body)
	is.NoErr(err)
	is.NoErr(contains(string(body), "console.log('hello world')"))
	is.NoErr(notContains(string(body), `new EventSource("/livereload")`))

	// Test favicon.ico
	res, err = http.Get(server.URL + "/favicon.ico")
	is.NoErr(err)
	is.Equal(res.StatusCode, 200)
	is.Equal(res.Header.Get("Content-Type"), "image/x-icon")
	is.Equal(res.Header.Get("Cache-Control"), "")
	is.Equal(res.Header.Get("Last-Modified"), "")
	body, err = io.ReadAll(res.Body)
	is.NoErr(err)
	is.Equal(body, favicon)
	is.NoErr(notContains(string(body), `new EventSource("/livereload")`))

	// Test that we get events when we call Reload
	stream, err := sse.Dial(log, server.URL+"/livereload")
	is.NoErr(err)
	defer stream.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	lr.Publish(ctx, &sse.Event{
		Type: "reload",
		Data: []byte("hello"),
	})
	event, err := stream.Next(ctx)
	is.NoErr(err)
	is.Equal(string(event.Type), "reload")
	is.Equal(string(event.Data), "hello")
}
