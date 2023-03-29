package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

var GIT_ROOT string = "git"

func init() {
	root, ok := os.LookupEnv("GITTO_ROOT")
	if ok {
		GIT_ROOT = root
	}
	var err error
	GIT_ROOT, err = filepath.Abs(GIT_ROOT)
	if err != nil {
		log.Fatalf("Unable to resolve GITTO_ROOT: %s", err)
	}
}

// The repo will be at {Root}/{Id}/{Name}.git
type GitRepo struct {
	Root   string `json:"-"`
	Id     string `json:"id"`
	Name   string `json:"name"`
	GitURL string `json:"git_url"`
}

func (repo *GitRepo) GetPath() string {
	name := repo.Name + ".git"
	return filepath.Join(repo.Root, repo.Id, name)
}

// Initialize the Git URL using the base URL from the http Request
func (repo *GitRepo) InitGitURL(r *http.Request) {
	scheme := "https"
	host := r.Host

	// Hack to make it work for localhost
	if strings.HasPrefix(host, "localhost") {
		scheme = "http"
	}

	repo.GitURL = fmt.Sprintf("%s://%s/%s/%s.git", scheme, host, repo.Id, repo.Name)
}

func GetRepo(id string) *GitRepo {
	path := filepath.Join(GIT_ROOT, id)

	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return nil
	}

	matches, _ := filepath.Glob(path + "/*.git")
	if len(matches) == 0 {
		return nil
	}

	name := filepath.Base(matches[0])
	name = strings.Split(name, ".")[0]

	return &GitRepo{
		Root: GIT_ROOT,
		Id:   id,
		Name: name,
	}
}

func RepoFromPath(path string) *GitRepo {
	// TODO: don't ignore errors
	if !strings.HasSuffix(path, ".git") {
		return nil
	}
	parent := filepath.Dir(path)
	root := filepath.Dir(parent)

	id := filepath.Base(parent)
	name := strings.Replace(filepath.Base(path), ".git", "", -1)

	return &GitRepo{
		Root: root,
		Id:   id,
		Name: name,
	}
}

func NewRepo(name string) (GitRepo, error) {
	uuid_, err := uuid.NewRandom()

	// convert uuid to string and remove hyphens
	id := strings.Replace(uuid_.String(), "-", "", -1)

	if err != nil {
		return GitRepo{}, err
	}

	repo := GitRepo{
		Root: GIT_ROOT,
		Id:   id,
		Name: name,
	}

	err = repo.initRepo()
	if err != nil {
		msg := fmt.Sprintf("Failed to create repository (%s)", err)
		return GitRepo{}, errors.New(msg)
	}

	err = repo.installPostReceive()
	if err != nil {
		msg := fmt.Sprintf("Failed to create repository (%s)", err)
		return GitRepo{}, errors.New(msg)
	}

	log.Println("Created new repo at", repo.GetPath())
	return repo, nil
}

func (repo *GitRepo) GetCommitHash() string {
	path := repo.GetPath()
	cmd := exec.Command(
		"git", "-C", path,
		"rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		log.Printf("%s: failed to get commit hash (%s)\n", repo.Id, err)
		return ""
	}
	return strings.TrimSpace(string(out))
}

// initializes the repo by invoking git init
func (repo *GitRepo) initRepo() error {
	path := repo.GetPath()
	cmd := exec.Command(
		"git", "init",
		"--bare",
		"--initial-branch", "main",
		path)
	err := cmd.Run()
	if err != nil {
		log.Printf("%s: failed to initialize git repo (%s)\n", repo.Id, err)
		return err
	}
	return nil
}

// install post-receive hook
func (repo *GitRepo) installPostReceive() error {
	hook_path := filepath.Join(repo.GetPath(), "hooks", "post-receive")
	gitto_path, err := filepath.Abs(os.Args[0])
	if err != nil {
		log.Printf("%s: failed to install post-receive hook (%s)\n", repo.Id, err)
		return err
	}

	err = os.Symlink(gitto_path, hook_path)
	if err != nil {
		log.Printf("%s: failed to install post-receive hook (%s)\n", repo.Id, err)
		return err
	}
	return nil
}

// Returns the Webhook URL if exists, or empty string
func (repo *GitRepo) GetWebhookURL() string {
	path := filepath.Join(repo.GetPath(), "hooks", "webhook.txt")

	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	url := string(content)
	return strings.TrimSpace(url)
}

// Returns the Webhook URL if exists, or empty string
func (repo *GitRepo) SetWebhookURL(url string) error {
	path := filepath.Join(repo.GetPath(), "hooks", "webhook.txt")
	return os.WriteFile(path, []byte(url), 0755)
}

func (repo *GitRepo) TriggerWebhook() {
	// fmt.Println("Triggering webhook", repo.GetWebhookURL())

	url := repo.GetWebhookURL()
	if url == "" {
		return
	}

	values := map[string]string{
		"repo_id":         repo.Id,
		"repo_name":       repo.Name,
		"git_commit_hash": repo.GetCommitHash(),
	}
	json_data, _ := json.Marshal(values)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(json_data))

	if err != nil {
		log.Fatal(err)
	}
	io.Copy(os.Stdout, resp.Body)
}
