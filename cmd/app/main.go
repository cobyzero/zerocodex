package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"github.com/cobyzero/zerocodex/internal/adapters/filesystem"
	"github.com/cobyzero/zerocodex/internal/adapters/ui"
	"github.com/cobyzero/zerocodex/internal/application"
)

func main() {

	a := app.New()
	w := a.NewWindow("ZeroCodex")

	repo := &filesystem.ProjectFS{}

	selectProject := application.SelectProjectUseCase{
		Repo: repo,
	}

	ui.BuildWindow(w, selectProject)

	w.Resize(fyne.NewSize(900, 700))
	w.ShowAndRun()
}
