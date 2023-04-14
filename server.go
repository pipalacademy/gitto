//
// Server functionality of gitto
//
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/cgi"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Config struct {
	Root  string
	Token string
}

// Loads the config from the env
func (config *Config) load() {
	root, ok := os.LookupEnv("GITTO_ROOT")
	if !ok {
		root = "git"
	}

	root, err := filepath.Abs(root)
	if err != nil {
		log.Fatalf("Unable to resolve GITTO_ROOT: %s", err)
	}
	config.Root = root

	token, ok := os.LookupEnv("GITTO_API_TOKEN")
	if !ok {
		log.Fatalf("Please specify env variable GITTO_API_TOKEN")
	}
	config.Token = token
}

var config Config

type NewRepoRequest struct {
	Name string `json:"name"`
}

type Webhook struct {
	URL string `json:"url"`
}

type WebhookRequest struct {
	GitURL        string `json:"git_url"`
	GitCommitHash string `json:"git_commit_hash"`
}

func Serve() {
	config.load()

	http.HandleFunc("/api/", handleAPI)
	http.HandleFunc("/", gitHttpBackend)

	fmt.Println("http://localhost:7878/")
	log.Fatal(http.ListenAndServe(":7878", nil))
}

func getAuthToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(strings.ToLower(header), "bearer ") {
		log.Println("header no bearer")
		return ""
	}

	parts := strings.SplitN(header, " ", 2)

	if len(parts) == 2 {
		return parts[1]
	} else {
		return ""
	}
}

var REGEX_REPO = regexp.MustCompile("^/api/repos/([0-9a-f]+)$")
var REGEX_HOOK = regexp.MustCompile("^/api/repos/([0-9a-f]+)/hook$")

func handleAPI(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	token := getAuthToken(r)

	if token == "" {
		w.WriteHeader(401)
		w.Header().Set("WWW-Authenticate", "Bearer")
		return
	} else if token != config.Token {
		w.WriteHeader(403)
		return
	}

	if path == "/api/repos" {
		apiCreateRepo(w, r)
		return
	}

	matches := REGEX_REPO.FindStringSubmatch(path)
	if matches != nil {
		apiRepo(w, r, matches[1])
		return
	}

	matches = REGEX_HOOK.FindStringSubmatch(path)
	if matches != nil {
		apiRepoHook(w, r, matches[1])
		return
	}

	w.WriteHeader(404)
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

func apiRepo(w http.ResponseWriter, r *http.Request, repo_id string) {
	repo := GetRepo(repo_id)

	if repo == nil {
		w.WriteHeader(404)
		return
	}
	repo.InitGitURL(r)

	body, _ := json.Marshal(repo)
	w.Write(body)
}

func apiRepoHook(w http.ResponseWriter, r *http.Request, repo_id string) {
	repo := GetRepo(repo_id)

	if repo == nil {
		w.WriteHeader(404)
		return
	}

	if r.Method == "GET" {
		hook := Webhook{
			URL: repo.GetWebhookURL(),
		}
		w.Header().Set("Content-type", "application/json")
		body, _ := json.Marshal(hook)
		w.Write(body)
		return
	} else if r.Method == "POST" {
		var hook Webhook

		err := json.NewDecoder(r.Body).Decode(&hook)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		repo.SetWebhookURL(hook.URL)

		w.Header().Set("Content-type", "application/json")
		body, _ := json.Marshal(hook)
		w.Write(body)
		return
	}
}

func gitHttpBackend(w http.ResponseWriter, r *http.Request) {
	handler := cgi.Handler{
		Path: "/usr/bin/git",
		Args: []string{"http-backend"},
		Env: []string{
			"GIT_PROJECT_ROOT=" + config.Root,
			"GIT_HTTP_EXPORT_ALL=",
			"REMOTE_USER=git",
		},
	}
	handler.ServeHTTP(w, r)
}

// tests if this command has been called from post-receive hook
func isPostReceive() bool {
	return strings.HasSuffix(os.Args[0], "post-receive")
}

func handlePostReceive() {
	pwd, _ := os.Getwd()
	repo := RepoFromPath(pwd)
	if repo != nil {
		repo.TriggerWebhook()
	} else {
		fmt.Printf("Failed to find the repo: %s", pwd)
	}
}

func main() {
	if isPostReceive() {
		handlePostReceive()
	} else {
		Serve()
	}
}
