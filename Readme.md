# livereload

[![Go Reference](https://pkg.go.dev/badge/github.com/matthewmueller/livereload.svg)](https://pkg.go.dev/github.com/matthewmueller/livereload)

Live-reload middleware for your server. Meant to be drop-n-go, so it's fairly opinionated. Built on top of the lower-level [sse package](https://github.com/livebud/sse).

## Features

- Minimal API for serving, watching and live reloading.
- Customize how livereload behaves a custom event listener.
- Detects the content type and ignores non-HTML files.
- Robust file watching built on top of `fsnotify`.

## Install

```sh
go get github.com/matthewmueller/livereload
```

## Usage

```go
func main() {
  ctx := context.Background()
  fsys := http.FileServer(http.Dir("example/public"))
  fmt.Println("Server started at http://localhost:3000")
  lr := livereload.New(slog.Default())
  go lr.Watch(ctx, ".")
  http.ListenAndServe(":3000", lr.Middleware(fsys))
}
```

### Customize live-reload behavior

By default, any HTML or text document will be reloaded upon change. If you'd like to better control this behavior, you can implement your own `reload` event handler:

```html
<!DOCTYPE html>
<html lang="en">
  <head> </head>
  <body>
    <h1>Hello</h1>
  </body>
  <script>
    window.addEventListener("reload", (e) => {
      // Prevent the default livereload handler from running
      e.stopImmediatePropagation()
      // Listen all the events the watcher discovered
      const events = e.detail.events || []
      for (let event of events) {
        console.log(event.op, event.path)
      }
    })
  </script>
</html>
```

### Publishing your own events

You can publish your own events with `lr.Publish`, for example:

```go
watcher.Watch(ctx, "example", func(path string) error {
  return lr.Publish(ctx, &livereload.Event{
    Type: "reload",
    Data: []byte(path),
  })
})
```

This will trigger the `reload` event on the client-side.

## Contributors

- Matt Mueller ([@mattmueller](https://twitter.com/mattmueller))

## License

MIT
