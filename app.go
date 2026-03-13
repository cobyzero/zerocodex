package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/cobyzero/zerocodex/internal/adapters/filesystem"
	"github.com/cobyzero/zerocodex/internal/adapters/llm"
	"github.com/cobyzero/zerocodex/internal/adapters/secrets"
	"github.com/cobyzero/zerocodex/internal/adapters/storage"
	"github.com/cobyzero/zerocodex/internal/application"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx context.Context

	repo             *filesystem.ProjectFS
	projectStore     *storage.SQLiteProjectStore
	fileContextStore *storage.SQLiteFileContextStore
	chatHistoryStore *storage.SQLiteChatHistoryStore
	selectProject    *application.SelectProject
	chat             *application.Chat
	keyring          *secrets.KeyringStore

	mu             sync.Mutex
	currentProject string
	status         string
	activity       string
	apiKeyReady    bool
}

type ProjectInfo struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type BootstrapData struct {
	APIKeyConfigured bool          `json:"apiKeyConfigured"`
	CurrentProject   string        `json:"currentProject"`
	Status           string        `json:"status"`
	Activity         string        `json:"activity"`
	SavedProjects    []ProjectInfo `json:"savedProjects"`
	Transcript       []Message     `json:"transcript"`
}

type OperationResult struct {
	APIKeyConfigured bool          `json:"apiKeyConfigured"`
	Status           string        `json:"status"`
	Activity         string        `json:"activity"`
	CurrentProject   string        `json:"currentProject"`
	SavedProjects    []ProjectInfo `json:"savedProjects"`
	Transcript       []Message     `json:"transcript"`
}

func NewApp() *App {
	repo := &filesystem.ProjectFS{}
	dbPath, _ := localDBPath()

	projectStore, _ := newProjectStore(dbPath)
	fileContextStore, _ := newFileContextStore(dbPath)
	chatHistoryStore, _ := newChatHistoryStore(dbPath)

	selectProject := &application.SelectProject{
		Repo:  repo,
		Store: projectStore,
	}

	chat := &application.Chat{
		Repo:         repo,
		ContextStore: fileContextStore,
		HistoryStore: chatHistoryStore,
	}

	app := &App{
		repo:             repo,
		projectStore:     projectStore,
		fileContextStore: fileContextStore,
		chatHistoryStore: chatHistoryStore,
		selectProject:    selectProject,
		chat:             chat,
		keyring:          secrets.NewKeyringStore(),
		status:           "Ready",
	}

	if key, err := app.keyring.LoadDeepSeekAPIKey(); err == nil && strings.TrimSpace(key) != "" {
		app.chat.Client = llm.NewDeepSeekClient(strings.TrimSpace(key))
		app.apiKeyReady = true
	}

	return app
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) shutdown(ctx context.Context) {
	closeIfPossible(a.projectStore)
	closeIfPossible(a.fileContextStore)
	closeIfPossible(a.chatHistoryStore)
}

func (a *App) Bootstrap() BootstrapData {
	return BootstrapData{
		APIKeyConfigured: a.hasAPIKey(),
		CurrentProject:   a.currentProjectPath(),
		Status:           a.currentStatus(),
		Activity:         a.currentActivity(),
		SavedProjects:    a.savedProjects(),
		Transcript:       a.transcript(a.currentProjectPath()),
	}
}

func (a *App) SaveAPIKey(apiKey string) (OperationResult, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return OperationResult{}, validationError{msg: "api key is required"}
	}
	if err := a.keyring.SaveDeepSeekAPIKey(apiKey); err != nil {
		return OperationResult{}, err
	}
	a.setAPIKey(apiKey)
	return a.snapshot(), nil
}

func (a *App) DeleteAPIKey() (OperationResult, error) {
	if err := a.keyring.DeleteDeepSeekAPIKey(); err != nil {
		return OperationResult{}, err
	}
	a.clearAPIKey()
	return a.snapshot(), nil
}

func (a *App) OpenProjectDialog() (string, error) {
	defaultDir := a.currentProjectPath()
	if defaultDir == "" {
		if cwd, err := os.Getwd(); err == nil {
			defaultDir = cwd
		}
	}
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:                "Selecciona un proyecto",
		DefaultDirectory:     defaultDir,
		CanCreateDirectories: true,
	})
}

func (a *App) SelectProject(path string) (OperationResult, error) {
	if !a.hasAPIKey() {
		return OperationResult{}, validationError{msg: "configure the api key first"}
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return OperationResult{}, validationError{msg: "project path is required"}
	}

	a.setStatus("Indexing", "Building local project context...")
	selected, err := a.selectProject.Execute(path)
	if err != nil {
		a.setStatus("Ready", "")
		return OperationResult{}, err
	}
	if err := a.chat.AnalyzeProject(selected, nil); err != nil {
		a.setCurrentProject(selected)
		a.setStatus("Index failed", "")
		return OperationResult{}, err
	}

	a.setCurrentProject(selected)
	a.setStatus("Ready", "")
	return a.snapshot(), nil
}

func (a *App) RemoveProject(path string) (OperationResult, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return OperationResult{}, validationError{msg: "project path is required"}
	}
	if err := a.selectProject.RemoveSaved(path); err != nil {
		return OperationResult{}, err
	}
	if path == a.currentProjectPath() {
		a.setCurrentProject("")
	}
	return a.snapshot(), nil
}

func (a *App) RunPrompt(prompt string) (OperationResult, error) {
	if !a.hasAPIKey() {
		return OperationResult{}, validationError{msg: "configure the api key first"}
	}
	projectPath := a.currentProjectPath()
	if strings.TrimSpace(projectPath) == "" {
		return OperationResult{}, validationError{msg: "select a project first"}
	}
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return OperationResult{}, validationError{msg: "prompt is required"}
	}

	a.setStatus("Running", "Thinking...")
	_, err := a.chat.Execute(projectPath, prompt, func(msg string) {
		a.setActivity(msg)
	})
	if err != nil {
		a.setStatus("Ready", "")
		return OperationResult{}, err
	}
	a.setStatus("Ready", "")
	return a.snapshot(), nil
}

func (a *App) snapshot() OperationResult {
	return OperationResult{
		APIKeyConfigured: a.hasAPIKey(),
		Status:           a.currentStatus(),
		Activity:         a.currentActivity(),
		CurrentProject:   a.currentProjectPath(),
		SavedProjects:    a.savedProjects(),
		Transcript:       a.transcript(a.currentProjectPath()),
	}
}

func (a *App) hasAPIKey() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.apiKeyReady
}

func (a *App) setAPIKey(apiKey string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.chat.Client = llm.NewDeepSeekClient(apiKey)
	a.apiKeyReady = true
	a.status = "Ready"
	a.activity = ""
}

func (a *App) clearAPIKey() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.chat.Client = nil
	a.apiKeyReady = false
	a.status = "Ready"
	a.activity = ""
}

func (a *App) setCurrentProject(path string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.currentProject = path
}

func (a *App) currentProjectPath() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.currentProject
}

func (a *App) setStatus(status, activity string) {
	a.mu.Lock()
	a.status = status
	a.activity = activity
	a.mu.Unlock()
	a.emitProgress(status, activity)
}

func (a *App) setActivity(activity string) {
	a.mu.Lock()
	a.activity = activity
	status := a.status
	a.mu.Unlock()
	a.emitProgress(status, activity)
}

func (a *App) currentStatus() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.status
}

func (a *App) currentActivity() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.activity
}

func (a *App) emitProgress(status, activity string) {
	if a.ctx == nil {
		return
	}
	runtime.EventsEmit(a.ctx, "agent:progress", map[string]string{
		"status":   status,
		"activity": activity,
	})
}

func (a *App) savedProjects() []ProjectInfo {
	paths, err := a.selectProject.ListSaved()
	if err != nil {
		return nil
	}
	out := make([]ProjectInfo, 0, len(paths))
	for _, path := range paths {
		out = append(out, ProjectInfo{
			Name: filepath.Base(path),
			Path: path,
		})
	}
	return out
}

func (a *App) transcript(projectPath string) []Message {
	if a.chat.HistoryStore == nil || strings.TrimSpace(projectPath) == "" {
		return nil
	}
	entries, err := a.chat.HistoryStore.ListRecent(projectPath, 100)
	if err != nil {
		return nil
	}
	out := make([]Message, 0, len(entries))
	for _, entry := range entries {
		role := entry.Role
		switch role {
		case "assistant", "user", "error":
		default:
			role = "system"
		}
		out = append(out, Message{
			Role:    role,
			Content: strings.TrimSpace(entry.Content),
		})
	}
	return out
}

func localDBPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "zerocodex", "projects.db"), nil
}

func newProjectStore(dbPath string) (*storage.SQLiteProjectStore, error) {
	if strings.TrimSpace(dbPath) == "" {
		return nil, validationError{msg: "invalid db path"}
	}
	return storage.NewSQLiteProjectStore(dbPath)
}

func newFileContextStore(dbPath string) (*storage.SQLiteFileContextStore, error) {
	if strings.TrimSpace(dbPath) == "" {
		return nil, validationError{msg: "invalid db path"}
	}
	return storage.NewSQLiteFileContextStore(dbPath)
}

func newChatHistoryStore(dbPath string) (*storage.SQLiteChatHistoryStore, error) {
	if strings.TrimSpace(dbPath) == "" {
		return nil, validationError{msg: "invalid db path"}
	}
	return storage.NewSQLiteChatHistoryStore(dbPath)
}

func closeIfPossible(v interface{ Close() error }) {
	if v == nil {
		return
	}
	_ = v.Close()
}

type validationError struct {
	msg string
}

func (e validationError) Error() string {
	return e.msg
}
