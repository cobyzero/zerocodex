package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/cobyzero/zerocodex/internal/adapters/filesystem"
	"github.com/cobyzero/zerocodex/internal/adapters/llm"
	"github.com/cobyzero/zerocodex/internal/adapters/secrets"
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
	repo := &filesystem.ProjectFS{}
	dbPath, err := localDBPath()
	if err != nil {
		log.Printf("local db path unavailable: %v", err)
	}

	projectStore, err := newProjectStore(dbPath)
	if err != nil {
		log.Printf("project store disabled: %v", err)
	}

	fileContextStore, err := newFileContextStore(dbPath)
	if err != nil {
		log.Printf("file context cache disabled: %v", err)
	}

	showAPIKeySetup(a, repo, projectStore, fileContextStore)
	a.Run()
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
		return nil, &validationError{msg: "invalid db path"}
	}
	return storage.NewSQLiteProjectStore(dbPath)
}

func newFileContextStore(dbPath string) (*storage.SQLiteFileContextStore, error) {
	if strings.TrimSpace(dbPath) == "" {
		return nil, &validationError{msg: "invalid db path"}
	}
	return storage.NewSQLiteFileContextStore(dbPath)
}

func showAPIKeySetup(
	a fyne.App,
	repo *filesystem.ProjectFS,
	projectStore *storage.SQLiteProjectStore,
	fileContextStore *storage.SQLiteFileContextStore,
) {
	store := secrets.NewKeyringStore()
	setupWindow := a.NewWindow("ZeroCodex Setup")
	setupWindow.Resize(fyne.NewSize(560, 300))

	title := widget.NewLabelWithStyle("Configura tu API key de DeepSeek", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	description := widget.NewLabel("Se guarda en el llavero seguro del sistema y se usara para esta sesion.")
	description.Wrapping = fyne.TextWrapWord

	apiKeyEntry := widget.NewPasswordEntry()
	apiKeyEntry.SetPlaceHolder("sk-...")

	if savedKey, err := store.LoadDeepSeekAPIKey(); err == nil && strings.TrimSpace(savedKey) != "" {
		apiKeyEntry.SetText(savedKey)
	}

	openMain := func(apiKey string) {
		mainWindow := a.NewWindow("ZeroCodex - AI Coding Agent")

		selectProject := &application.SelectProject{
			Repo:  repo,
			Store: projectStore,
		}

		chat := &application.Chat{
			Repo:         repo,
			Client:       llm.NewDeepSeekClient(apiKey),
			ContextStore: fileContextStore,
		}

		ui.BuildWindow(mainWindow, selectProject, chat)
		mainWindow.Resize(fyne.NewSize(1180, 760))
		mainWindow.Show()
		setupWindow.Close()
	}

	saveAndContinue := widget.NewButtonWithIcon("Guardar y continuar", theme.ConfirmIcon(), func() {
		apiKey := strings.TrimSpace(apiKeyEntry.Text)
		if apiKey == "" {
			dialog.ShowError(errEmptyAPIKey(), setupWindow)
			return
		}

		if err := store.SaveDeepSeekAPIKey(apiKey); err != nil {
			dialog.ShowError(err, setupWindow)
			return
		}
		openMain(apiKey)
	})
	saveAndContinue.Importance = widget.HighImportance

	useEnvBtn := widget.NewButton("Usar DEEPSEEK_API_KEY del entorno", func() {
		apiKey := strings.TrimSpace(os.Getenv("DEEPSEEK_API_KEY"))
		if apiKey == "" {
			dialog.ShowError(errEmptyEnvAPIKey(), setupWindow)
			return
		}
		openMain(apiKey)
	})

	content := container.NewPadded(container.NewVBox(
		title,
		description,
		widget.NewSeparator(),
		widget.NewLabel("API Key"),
		apiKeyEntry,
		container.NewHBox(saveAndContinue, useEnvBtn),
	))
	setupWindow.SetContent(content)
	setupWindow.Show()
}

func errEmptyAPIKey() error {
	return &validationError{msg: "Ingresa una API key valida para continuar."}
}

func errEmptyEnvAPIKey() error {
	return &validationError{msg: "No existe DEEPSEEK_API_KEY en el entorno."}
}

type validationError struct {
	msg string
}

func (e *validationError) Error() string {
	return e.msg
}
