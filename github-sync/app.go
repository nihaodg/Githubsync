package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/go-github/v60/github"
	"golang.org/x/oauth2"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

type App struct {
	ctx         context.Context
	token       string
	username    string
	storagePath string
	ghClient    *github.Client
}

type Config struct {
	Token       string `json:"token"`
	Username    string `json:"username"`
	StoragePath string `json:"storage_path"`
}

type RepoInfo struct {
	Name         string `json:"name"`
	LocalPath    string `json:"local_path"`
	RemoteURL    string `json:"remote_url"`
	LastSyncTime string `json:"last_sync_time"`
	Branch       string `json:"branch"`
	CommitSHA    string `json:"commit_sha"`
}

type CloneResult struct {
	Success   bool   `json:"success"`
	LocalPath string `json:"local_path"`
	RemoteURL string `json:"remote_url"`
	Error     string `json:"error"`
}

type StatusResult struct {
	Clean bool         `json:"clean"`
	Files []FileStatus `json:"files"`
}

type FileStatus struct {
	Path     string `json:"path"`
	Status   string `json:"status"`
	Staged   bool   `json:"staged"`
	Modified bool   `json:"modified"`
}

type LogEntry struct {
	SHA       string `json:"sha"`
	Message   string `json:"message"`
	Author    string `json:"author"`
	Timestamp string `json:"timestamp"`
}

func NewApp() *App {
	app := &App{}

	// Use USERPROFILE on Windows, HOME on Linux/Mac
	homeDir := os.Getenv("USERPROFILE")
	if homeDir == "" {
		homeDir = os.Getenv("HOME")
	}
	if homeDir == "" {
		homeDir = "."
	}

	app.storagePath = filepath.Join(homeDir, "GitHubSync", "repos")
	os.MkdirAll(app.storagePath, 0755)
	return app
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.loadConfig()
}

func (a *App) loadConfig() {
	// Use same path logic as in ValidateAndSaveConfig
	configDir := os.Getenv("APPDATA")
	if configDir == "" {
		configDir = os.Getenv("XDG_CONFIG_HOME")
	}
	if configDir == "" {
		homeDir := os.Getenv("HOME")
		if homeDir == "" {
			homeDir = "."
		}
		configDir = filepath.Join(homeDir, ".config", "GitHubSync")
	} else {
		configDir = filepath.Join(configDir, "GitHubSync")
	}

	configPath := filepath.Join(configDir, "config.enc")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return
	}

	decrypted, err := a.decrypt(data)
	if err != nil {
		return
	}

	var cfg Config
	if err := json.Unmarshal(decrypted, &cfg); err != nil {
		return
	}

	a.token = cfg.Token
	a.username = cfg.Username
	a.storagePath = cfg.StoragePath

	if a.token != "" {
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: a.token})
		tc := oauth2.NewClient(context.Background(), ts)
		a.ghClient = github.NewClient(tc)
	}
}

func (a *App) ValidateAndSaveConfig(token, storagePath string) (string, error) {
	// Step 1: Validate token with GitHub
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(context.Background(), ts)
	client := github.NewClient(tc)

	user, _, err := client.Users.Get(context.Background(), "")
	if err != nil {
		return "", fmt.Errorf("step1_token_validation_failed: %v", err)
	}

	// Step 2: Setup storage path
	a.token = token
	a.username = user.GetLogin()
	a.storagePath = storagePath
	a.ghClient = client

	if a.storagePath == "" {
		a.storagePath = filepath.Join(os.Getenv("USERPROFILE"), "GitHubSync", "repos")
	}

	// Try to create storage dir
	if err := os.MkdirAll(a.storagePath, 0755); err != nil {
		return "", fmt.Errorf("step2_create_storage_dir_failed: %v", err)
	}

	cfg := Config{
		Token:       a.token,
		Username:    a.username,
		StoragePath: a.storagePath,
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("step3_json_marshal_failed: %v", err)
	}

	encrypted, err := a.encrypt(data)
	if err != nil {
		return "", fmt.Errorf("step4_encrypt_failed: %v", err)
	}

	// Step 5: Determine config directory
	configDir := os.Getenv("APPDATA")
	if configDir == "" {
		configDir = os.Getenv("XDG_CONFIG_HOME")
	}
	if configDir == "" {
		homeDir := os.Getenv("HOME")
		if homeDir == "" {
			homeDir = "."
		}
		configDir = filepath.Join(homeDir, ".config", "GitHubSync")
	} else {
		configDir = filepath.Join(configDir, "GitHubSync")
	}

	// Try to create config dir
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("step5_create_config_dir_failed: path=%s, error=%v", configDir, err)
	}

	configPath := filepath.Join(configDir, "config.enc")

	// Step 6: Write config file
	if err := os.WriteFile(configPath, encrypted, 0600); err != nil {
		return "", fmt.Errorf("step6_write_config_failed: path=%s, error=%v", configPath, err)
	}

	return a.username, nil
}

func (a *App) GetConfig() (*Config, error) {
	return &Config{
		Token:       a.token,
		Username:    a.username,
		StoragePath: a.storagePath,
	}, nil
}

func (a *App) Clone(repoURL, name string) (*CloneResult, error) {
	result := &CloneResult{Success: false}

	if a.token == "" {
		result.Error = "请先配置 GitHub Token"
		return result, fmt.Errorf(result.Error)
	}

	// 判断 name 是否为完整路径
	var localPath string
	if filepath.IsAbs(name) || (len(name) > 1 && name[1] == ':') {
		// name 是完整路径（如 C:\xxx 或 D:\xxx），直接使用
		localPath = name
	} else {
		// name 只是文件夹名，拼接到 storagePath
		localPath = filepath.Join(a.storagePath, name)
	}

	// 检查目录是否存在
	if _, err := os.Stat(filepath.Dir(localPath)); os.IsNotExist(err) {
		// 确保父目录存在
		os.MkdirAll(filepath.Dir(localPath), 0755)
	}

	if _, err := os.Stat(localPath); err == nil {
		result.Error = "仓库已存在"
		result.LocalPath = localPath
		return result, fmt.Errorf(result.Error)
	}

	parsedURL := convertToHttpsUrl(repoURL)
	if parsedURL == "" {
		result.Error = "无效的 GitHub URL"
		return result, fmt.Errorf(result.Error)
	}

	auth := &http.BasicAuth{
		Username: "x-access-token",
		Password: a.token,
	}

	_, err := git.PlainClone(localPath, false, &git.CloneOptions{
		URL:  parsedURL,
		Auth: auth,
	})
	if err != nil {
		result.Error = err.Error()
		return result, err
	}

	result.Success = true
	result.LocalPath = localPath
	result.RemoteURL = parsedURL

	return result, nil
}

func (a *App) Status(repoPath string) (*StatusResult, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, err
	}

	workTree, err := repo.Worktree()
	if err != nil {
		return nil, err
	}

	status, err := workTree.Status()
	if err != nil {
		return nil, err
	}

	result := &StatusResult{
		Files: make([]FileStatus, 0),
		Clean: status.IsClean(),
	}

	for path, s := range status {
		fileStatus := FileStatus{
			Path: path,
		}

		switch {
		case s.Worktree == '?' || s.Worktree == '!' || s.Staging == '?':
			fileStatus.Status = "untracked"
			fileStatus.Staged = false
			fileStatus.Modified = false
		case s.Worktree == 'M' || s.Worktree == 'D' || s.Worktree == 'A':
			fileStatus.Status = "modified"
			fileStatus.Modified = true
			fileStatus.Staged = s.Staging == 'M' || s.Staging == 'A' || s.Staging == 'D'
		case s.Staging == 'M' || s.Staging == 'A' || s.Staging == 'D':
			fileStatus.Status = "staged"
			fileStatus.Staged = true
			fileStatus.Modified = false
		default:
			fileStatus.Status = "unchanged"
			fileStatus.Staged = false
			fileStatus.Modified = false
		}

		result.Files = append(result.Files, fileStatus)
	}

	return result, nil
}

func (a *App) Commit(repoPath, message string) (*LogEntry, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, err
	}

	workTree, err := repo.Worktree()
	if err != nil {
		return nil, err
	}

	_, err = workTree.Commit(message, &git.CommitOptions{
		All: true,
	})
	if err != nil {
		return nil, err
	}

	head, err := repo.Head()
	if err != nil {
		return nil, err
	}

	return &LogEntry{
		SHA:     head.Hash().String(),
		Message: message,
	}, nil
}

func (a *App) Pull(repoPath string) error {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return err
	}

	workTree, err := repo.Worktree()
	if err != nil {
		return err
	}

	auth := &http.BasicAuth{
		Username: "x-access-token",
		Password: a.token,
	}

	return workTree.Pull(&git.PullOptions{
		Auth: auth,
	})
}

func (a *App) Push(repoPath string) error {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return err
	}

	auth := &http.BasicAuth{
		Username: "x-access-token",
		Password: a.token,
	}

	return repo.Push(&git.PushOptions{
		Auth: auth,
	})
}

func (a *App) Log(repoPath string, limit int) ([]LogEntry, error) {
	if limit <= 0 {
		limit = 50
	}

	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, err
	}

	head, err := repo.Head()
	if err != nil {
		return nil, err
	}

	cIter, err := repo.Log(&git.LogOptions{From: head.Hash()})
	if err != nil {
		return nil, err
	}
	defer cIter.Close()

	var entries []LogEntry
	count := 0

	err = cIter.ForEach(func(c *object.Commit) error {
		if count >= limit {
			return fmt.Errorf("limit reached")
		}

		entries = append(entries, LogEntry{
			SHA:       c.Hash.String(),
			Message:   c.Message,
			Author:    c.Author.Name,
			Timestamp: c.Author.When.Format(time.RFC3339),
		})
		count++
		return nil
	})

	if err != nil && err.Error() != "limit reached" {
		return nil, err
	}

	return entries, nil
}

func (a *App) GetRepos() ([]RepoInfo, error) {
	reposPath := filepath.Join(os.Getenv("APPDATA"), "GitHubSync", "repos.json")
	data, err := os.ReadFile(reposPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []RepoInfo{}, nil
		}
		return nil, err
	}

	var repos []RepoInfo
	if err := json.Unmarshal(data, &repos); err != nil {
		return nil, err
	}

	return repos, nil
}

func (a *App) SaveRepos(repos []RepoInfo) error {
	reposPath := filepath.Join(os.Getenv("APPDATA"), "GitHubSync", "repos.json")
	os.MkdirAll(filepath.Dir(reposPath), 0755)

	data, err := json.Marshal(repos)
	if err != nil {
		return err
	}

	return os.WriteFile(reposPath, data, 0644)
}

func (a *App) GetStoragePath() string {
	return a.storagePath
}

func (a *App) ListGithubRepos() ([]string, error) {
	if a.ghClient == nil {
		return nil, fmt.Errorf("github client not initialized")
	}

	var allRepos []*github.Repository
	opts := &github.RepositoryListOptions{
		Visibility:  "all",
		Sort:        "updated",
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		repos, resp, err := a.ghClient.Repositories.List(context.Background(), "", opts)
		if err != nil {
			return nil, err
		}

		allRepos = append(allRepos, repos...)

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	var names []string
	for _, repo := range allRepos {
		names = append(names, repo.GetFullName())
	}

	return names, nil
}

func convertToHttpsUrl(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)

	rawURL = strings.ReplaceAll(rawURL, "git@github.com:", "https://github.com/")
	rawURL = strings.ReplaceAll(rawURL, "git://github.com/", "https://github.com/")
	rawURL = strings.ReplaceAll(rawURL, "http://github.com/", "https://github.com/")

	if !strings.HasPrefix(rawURL, "https://github.com/") {
		return ""
	}

	rawURL = strings.TrimSuffix(rawURL, ".git")

	return rawURL
}

func (a *App) encrypt(data []byte) ([]byte, error) {
	key := []byte("GitHubSync2024SecureKeyForAES32B")
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, data, nil), nil
}

func (a *App) decrypt(data []byte) ([]byte, error) {
	key := []byte("GitHubSync2024SecureKeyForAES32B")
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}
