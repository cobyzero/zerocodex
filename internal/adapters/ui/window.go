package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/cobyzero/zerocodex/internal/application"
)

func BuildWindow(
	w fyne.Window,
	selectProject application.SelectProjectUseCase,
) {

	projectLabel := widget.NewLabel("No project selected")

	selectBtn := widget.NewButton("Select Project", func() {

		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {

			if uri == nil {
				return
			}

			project, _ := selectProject.Execute(uri.Path())

			if project != nil {
				projectLabel.SetText(project.Path)
			}

		}, w)

	})

	chat := widget.NewMultiLineEntry()

	prompt := widget.NewEntry()

	runBtn := widget.NewButton("Run", func() {})

	top := container.NewHBox(selectBtn, projectLabel)

	bottom := container.NewBorder(nil, nil, nil, runBtn, prompt)

	layout := container.NewBorder(top, bottom, nil, nil, chat)

	w.SetContent(layout)
}
