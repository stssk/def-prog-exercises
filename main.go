package main

import (
	"context"
	"log"
	"net/http"

	_ "github.com/glebarez/go-sqlite"
	"github.com/stssk/def-prog-exercises/app"
)

func main() {
	ctx := context.Background()
	auth := app.Auth(ctx)

	sm := http.NewServeMux()
	sm.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if auth.IsLogged(r) {
			http.Redirect(w, r, "/notes/", http.StatusFound)
		} else {
			http.Redirect(w, r, "/auth/", http.StatusFound)
		}
	})
	sm.HandleFunc("/echo", app.Echo)
	sm.Handle("/auth/", auth)
	sm.Handle("/notes/", app.Notes(ctx, auth))

	addr := "localhost:8080"
	s := &http.Server{
		Addr:    addr,
		Handler: sm,
	}
	log.Println("Ready to accept connections on " + addr)
	log.Fatal(s.ListenAndServe())
}
