//
// Server functionality of gitto
//
package main

import (
    "encoding/json"
    "net/http/cgi"
    "net/http"
    "fmt"
    "log"
    "strings"
    "os"
)

type NewRepoRequest struct {
    Name string `json:"name"`
}

func Serve() {
    http.HandleFunc("/api/repos", apiCreateRepo)
    http.HandleFunc("/", gitHttpBackend)

    fmt.Println("http://localhost:8080/")
    http.ListenAndServe(":8080", nil)
}

func apiCreateRepo(w http.ResponseWriter, r *http.Request) {
    log.Printf("Req: %s %s\n", r.Host, r.URL.Path)

    if r.Method != "POST" {
        w.WriteHeader(405)
        return
    }

    var req NewRepoRequest

    err := json.NewDecoder(r.Body).Decode(&req)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    repo, err := NewRepo(req.Name)
    if err != nil {
        http.Error(w, "Failed to create repo", 500)
    }
    repo.InitGitURL(r)

    body, _ := json.Marshal(repo)
    w.Write(body)
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

// tests if this command has been called from post-receive hook
func isPostReceive() bool {
    return strings.HasSuffix(os.Args[0], "post-receive")
}
func handlePostReceive() {
    // TODO
    fmt.Println("handling post-receive")
}

func main() {
    if isPostReceive() {
        handlePostReceive()
    } else {
        Serve()
    }
}
