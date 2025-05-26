package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/livebud/mux"
	"github.com/matthewmueller/livereload"
	"github.com/matthewmueller/socket"
	"golang.org/x/sync/errgroup"
)

func main() {
	ctx := context.Background()
	fsys := http.FileServer(http.Dir("example/embedded/public"))
	lr := livereload.New(slog.Default())
	router := mux.New()
	router.Get("/", fsys.ServeHTTP)
	router.Get("/about", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "<html><body><h1>About Page</h1></body></html>")
	})
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return lr.Watch(ctx, ".")
	})
	eg.Go(func() error {
		fmt.Println("Server started at http://localhost:3000")
		return socket.ListenAndServe(ctx, ":3000", lr.Middleware(router))
	})
	if err := eg.Wait(); err != nil {
		slog.Error("Error in server", "error", err)
		return
	}
}
