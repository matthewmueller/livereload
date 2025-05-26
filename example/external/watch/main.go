package main

import (
	"context"
	"log/slog"

	"github.com/matthewmueller/livereload"
	"golang.org/x/sync/errgroup"
)

func main() {
	ctx := context.Background()
	log := slog.Default()
	lr := livereload.New(log)
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return lr.Watch(ctx, ".")
	})
	eg.Go(func() error {
		log.Info("Server started at http://localhost:35729")
		return lr.ListenAndServe(ctx, ":35729")
	})
	if err := eg.Wait(); err != nil {
		log.Error("Error in server", "error", err)
		return
	}
}
