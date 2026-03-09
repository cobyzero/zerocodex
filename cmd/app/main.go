package main

import (
	"log"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"github.com/cobyzero/zerocodex/internal/adapters/filesystem"
	"github.com/cobyzero/zerocodex/internal/adapters/llm"
	"github.com/cobyzero/zerocodex/internal/adapters/storage"
	"github.com/cobyzero/zerocodex/internal/adapters/ui"
	"github.com/cobyzero/zerocodex/internal/application"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, falling back to environment variables")
	}

	a := app.NewWithID("com.cobyzero.zerocodex")
	w := a.NewWindow("ZeroCodex - AI Coding Agent")

	repo := &filesystem.ProjectFS{}
	deepseekClient := llm.NewDeepSeekClient()
	projectStore, err := newProjectStore()
	if err != nil {
		log.Printf("project store disabled: %v", err)
	}

	selectProject := &application.SelectProject{
		Repo:  repo,
		Store: projectStore,
	}

	chat := &application.Chat{
		Repo:   repo,
		Client: deepseekClient,
	}

	ui.BuildWindow(w, selectProject, chat)

	w.Resize(fyne.NewSize(1180, 760))
	w.ShowAndRun()
}

func newProjectStore() (*storage.SQLiteProjectStore, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	dbPath := filepath.Join(configDir, "zerocodex", "projects.db")
	return storage.NewSQLiteProjectStore(dbPath)
}
