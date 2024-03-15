package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/matthewmueller/livereload"
)

func main() {
	ctx := context.Background()
	fsys := http.FileServer(http.Dir("example/public"))
	fmt.Println("Server started at http://localhost:3000")
	lr := livereload.New(slog.Default())
	go lr.Watch(ctx, ".")
	http.ListenAndServe(":3000", lr.Middleware(fsys))
}
