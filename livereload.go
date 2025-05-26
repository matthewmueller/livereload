package livereload

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/livebud/sse"
	"github.com/livebud/watcher"
	"github.com/matthewmueller/httpbuf"
	"github.com/matthewmueller/socket"
)

// Event is an server-sent event (SSE) that you can send to the browser
type Event = sse.Event

func New(log *slog.Logger) *Reloader {
	return &Reloader{"/livereload", log, sse.New(log)}
}

type Reloader struct {
	Path string
	log  *slog.Logger
	sse  *sse.Handler
}

// Middleware that rewrites the response body to include the livereload script
// for HTML or plain text responses. It also serves the livereload server-sent
// events at the given path. Unlike ListenAndServe, this is meant to be embedded
// into an existing HTTP server.
func (r *Reloader) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// TODO: Check method
		if req.URL.Path == r.Path && req.Header.Get("Accept") == "text/event-stream" {
			r.sse.ServeHTTP(w, req)
			return
		}
		// Wrap the response writer to capture the response body
		rw := httpbuf.Wrap(w)
		defer rw.Flush()
		next.ServeHTTP(rw, req)
		// Inject the live reload script
		body, rewrote := rewrite(rw.Body, r.Path)
		if !rewrote {
			return
		}
		rw.Body = body
		rw.Header().Set("Content-Length", strconv.Itoa(len(body)))
		rw.Header().Set("Content-Type", "text/html; charset=utf-8")
		// Don't cache re-written responses
		rw.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		rw.Header().Set("Last-Modified", "0")
	})
}

// Reload sends a message to browser with the given event. Event data should
// follow the format "op:path;op:path" format in Watch.
func (r *Reloader) Publish(ctx context.Context, event *Event) error {
	return r.sse.Publish(ctx, event)
}

// Watch a directory for changes and send the events to the browser
func (r *Reloader) Watch(ctx context.Context, watchDir string) error {
	return watcher.Watch(ctx, watchDir, func(events []watcher.Event) error {
		var data bytes.Buffer
		for i, event := range events {
			r.log.Debug("livereload: got event", "event", event)
			if i > 0 {
				data.WriteString(";")
			}
			data.WriteString(event.String())
		}
		err := r.Publish(ctx, &Event{
			Type:  "reload",
			Data:  data.Bytes(),
			Retry: 1000, // Retry after 1 second
		})
		if err != nil {
			r.log.Error("livereload: failed to reload", "error", err, "events", data.String())
		}
		return nil
	})
}

// ListenAndServe serves the client and the server-sent events at the given address.
// Unlike Middleware, it's meant to be used as a standalone server.
func (r *Reloader) ListenAndServe(ctx context.Context, addr string) error {
	return socket.ListenAndServe(ctx, addr, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Check if the request is for the event stream or the live reload script
		if req.Method != http.MethodGet || req.URL.Path != r.Path {
			http.Error(w, "livereload: not found", http.StatusNotFound)
			return
		}
		// Handle the server-sent event stream
		if req.Header.Get("Accept") == "text/event-stream" {
			r.sse.ServeHTTP(w, req)
			return
		}

		// Serve the live reload script
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Last-Modified", "0")
		w.Write(fmt.Appendf(nil, liveScript, getFullUrl(req)))
	}))
}

// Client-side livereload script that we attach to the end of the body
const liveScript = `
(function() {
	const es = new EventSource(%[1]q)
	let needsReload = false

	// Handle the eventsource connection
	es.addEventListener("open", function(e) {
		console.debug("livereload: connected to", %[1]q)
		if (needsReload) {
			console.debug("livereload: reloading page")
			document.dispatchEvent(new CustomEvent("reload", {
				bubbles: true,
				detail: {}
			}))
		}
		needsReload = false
	})

	// Handle errors
	es.addEventListener("error", function (e) {
		if (es.readyState === EventSource.CONNECTING) {
			console.debug("livereload: connection lost, reconnecting...")
			needsReload = true
		} else if (es.readyState === EventSource.CLOSED) {
			console.debug("livereload: connection closed")
		} else {
			console.debug("livereload: error", e)
		}
	})

	// Handle reload events
	es.addEventListener("reload", function(e) {
		console.debug("livereload: eventsource got 'reload' event")
		const evs = e.data.split(";")
		const events = []
		for (let i = 0; i < evs.length; i++) {
			const event = evs[i].split(":")
			if (event.length !== 2) continue
			const op = event[0]
			const path = event[1]
			events.push({ op, path })
		}
		document.dispatchEvent(new CustomEvent("reload", {
			bubbles: true,
			detail: { events }
		}))
	})

	// Close the event source when the browser window is closed
	window.addEventListener("beforeunload", function() {
		console.debug("livereload: closing event source")
		es.close()
	})

	// Default reload implementation. This can be overriden with
	// e.stopImmediatePropagation() to prevent the default behavior.
	let reloading = false
	window.addEventListener("reload", function(e) {
		console.debug("livereload: window got 'reload' event")
		if (reloading) {
			console.debug("livereload: already reloading, ignoring")
			return
		}
		reloading = true
		ready().then(() => {
			reloading = false
			document.location.reload()
		})
	})

	// Wait for the page to be ready before reloading
	function ready() {
		return new Promise((resolve, reject) => {
			function loop(attempt) {
				fetch(document.location.href).then(res => {
					console.debug("livereload: ready to reload")
					resolve()
				}).catch(err => {
					const waitMs = backoff(attempt)
					console.debug("livereload: not ready yet, trying again in "+waitMs+"ms", err.message)
					setTimeout(() => loop(attempt + 1), waitMs)
				})
			}
			loop(0)
		})
	}

	// S-curve backoff (start at 200ms, goes up to 5 seconds over 200 attempts)
	function backoff(attempt) {
		const minDelay = 200;       // 200 ms
		const maxDelay = 5000;      // 5 seconds
		const scale = 0.2;          // curve steepness
		const shift = 100;          // midpoint
		const ratio = 1 / (1 + Math.exp(-scale * (attempt - shift)));
		const delay = minDelay + ratio * (maxDelay - minDelay);
		return Math.round(delay);
	}
})()
`

func rewrite(data []byte, url string) ([]byte, bool) {
	index := bytes.Index(data, []byte("</body>"))
	if index < 0 {
		return data, false
	}
	const open = `<script type="text/javascript">`
	const close = `</script>`
	script := open + "\n" + fmt.Sprintf(liveScript, url) + "\n" + close
	data = append(data[:index], append([]byte(script), data[index:]...)...)
	return data, true
}

func getFullUrl(req *http.Request) string {
	scheme := "http"
	if req.TLS != nil {
		scheme = "https"
	}
	host := req.Host
	if host == "" {
		host = req.URL.Host
	}
	return fmt.Sprintf("%s://%s%s", scheme, host, req.URL.Path)
}
