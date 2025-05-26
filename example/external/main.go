package main

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/livebud/mux"
)

func main() {
	fsys := http.FileServer(http.Dir("example/external/public"))
	router := mux.New()
	router.Get("/", fsys.ServeHTTP)
	router.Get("/about", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `<html><body><h1>About</h1><script src="http://localhost:35729/livereload"></script></body></html>`)
	})
	fmt.Println("Server started at http://localhost:3000")
	if err := http.ListenAndServe(":3000", router); err != nil {
		slog.Error("Error in server", "error", err)
		return
	}
}
