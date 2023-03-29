//
// Server functionality of gitto
//
package main

import (
	"net/http/cgi"
	"net/http"
	"fmt"
)
func Serve() {
	http.HandleFunc("/", gitHttpBackend)

	fmt.Println("http://localhost:8080/")
	http.ListenAndServe(":8080", nil)
}

func gitHttpBackend(w http.ResponseWriter, r *http.Request) {
	handler := cgi.Handler{
		Path: "/usr/bin/git",
		Args: []string{"http-backend"},
		Env: []string {
			"GIT_PROJECT_ROOT=" + GIT_ROOT,
			"GIT_HTTP_EXPORT_ALL=",
		},
	}
	handler.ServeHTTP(w, r)
}

func main() {
	Serve()
}
