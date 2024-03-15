package livereload_test

import (
	"context"
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

func TestStatic(t *testing.T) {
	log := slog.Default()
	is := is.New(t)
	lr := livereload.New(log)
	fsys := fstest.MapFS{
		"index.html":  &fstest.MapFile{Data: []byte("<html><body>hello world</body></html>")},
		"error.txt":   &fstest.MapFile{Data: []byte("some error")},
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
	is.True(strings.Contains(string(body), "<html><body>hello world</body></html>"))
	is.True(strings.Contains(string(body), `new EventSource("/livereload")`))

	// Test error.txt
	res, err = http.Get(server.URL + "/error.txt")
	is.NoErr(err)
	is.Equal(res.StatusCode, 200)
	// Gets overwritten by livereload so the livereload script works
	is.Equal(res.Header.Get("Content-Type"), "text/html; charset=utf-8")
	is.Equal(res.Header.Get("Cache-Control"), "no-cache, no-store, must-revalidate")
	is.Equal(res.Header.Get("Last-Modified"), "0")
	body, err = io.ReadAll(res.Body)
	is.NoErr(err)
	is.True(strings.Contains(string(body), `some error`))
	is.True(strings.Contains(string(body), `new EventSource("/livereload")`))

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

	// Test that we get events when we call Reload
	stream, err := sse.Dial(log, server.URL+"/livereload")
	is.NoErr(err)
	defer stream.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	lr.Reload(ctx, &sse.Event{Data: []byte("hello")})
	event, err := stream.Next(ctx)
	is.NoErr(err)
	is.Equal(string(event.Data), "hello")
}
