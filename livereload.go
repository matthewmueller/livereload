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
// events at the given path.
func (r *Reloader) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
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
			Type: "reload",
			Data: data.Bytes(),
		})
		if err != nil {
			r.log.Error("livereload: failed to reload", "error", err, "events", data.String())
		}
		return nil
	})
}

// Client-side livereload script that we attach to the end of the body
const liveScript = `
<script type="text/javascript">
const es = new EventSource(%[1]q)
// Handle the eventsource connection
es.addEventListener("open", function(e) {
	console.debug("livereload: connected to", %[1]q)
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
// Default reload implementation. This can be overriden with
// e.stopImmediatePropagation() to prevent the default behavior.
window.addEventListener("reload", function(e) {
	console.debug("livereload: window got 'reload' event")
	document.location.reload()
})
window.addEventListener("beforeunload", function() {
	console.debug("livereload: closing event source")
	es.close()
})
</script>
`

func rewrite(data []byte, url string) ([]byte, bool) {
	index := bytes.Index(data, []byte("</body>"))
	if index < 0 {
		return data, false
	}
	script := fmt.Sprintf(liveScript, url)
	data = append(data[:index], append([]byte(script), data[index:]...)...)
	return data, true
}
