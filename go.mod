module github.com/matthewmueller/livereload

go 1.22.0

require (
	github.com/livebud/sse v0.0.3
	github.com/livebud/watcher v0.0.3
	github.com/matryer/is v1.4.1
	github.com/matthewmueller/httpbuf v0.0.2
)

require (
	github.com/bep/debounce v1.2.1 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/sabhiram/go-gitignore v0.0.0-20210923224102-525f6e181f06 // indirect
	golang.org/x/sync v0.3.0 // indirect
	golang.org/x/sys v0.0.0-20220908164124-27713097b956 // indirect
)

replace github.com/matthewmueller/httpbuf => ../httpbuf
