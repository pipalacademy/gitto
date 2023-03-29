package main

import (
    "errors"
    "net/http"
    "path/filepath"
    "fmt"
    "log"
    "os"
    "os/exec"
    "github.com/google/uuid"
    "strings"
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
    log.Println("Initialzed GIT_ROOT to", GIT_ROOT)
}

// The repo will be at {Root}/{Id}/{Name}.git
type GitRepo struct {
    Root string `json:"-"`
    Id string `json:"id"`
    Name string `json:"name"`
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
    log.Println("InitGitURL", repo.GitURL)
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
        Id: id,
        Name: name,
    }

    err = repo.initRepo()
    if err != nil {
        msg := fmt.Sprintf("Failed to create repository (%s)", err)
        return GitRepo{}, errors.New(msg)
    }

    err = repo.installHook()
    if err != nil {
        msg := fmt.Sprintf("Failed to create repository (%s)", err)
        return GitRepo{}, errors.New(msg)
    }

    log.Println("Created new repo at", repo.GetPath())
    return repo, nil
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
func (repo *GitRepo) installHook() error {
    hook_path := filepath.Join(repo.GetPath(), "hooks", "post-receive")
    gitto_path, err := filepath.Abs(os.Args[0])
    if err != nil {
        log.Printf("%s: failed to install post-receive hook (%s)\n", repo.Id, err)
        return err
    }

    err = os.Symlink(gitto_path, hook_path)
    if err != nil{
        log.Printf("%s: failed to install post-receive hook (%s)\n", repo.Id, err)
        return err
    }
    return nil
}