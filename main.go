package main

import (
	"context"
	"log"
	"net/http"

	_ "github.com/glebarez/go-sqlite"

	"github.com/empijei/def-prog-exercises/app"
	"github.com/empijei/def-prog-exercises/safeauth"
)

func main() {
	// The root/startup context has all rights.
	ctx := safeauth.Grant(context.Background(), "read", "write", "delete")
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
		Addr: addr,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r = auth.Preprocess(r)
			sm.ServeHTTP(w, r)
		}),
	}
	log.Println("Ready to accept connections on " + addr)
	log.Fatal(s.ListenAndServe())
}
