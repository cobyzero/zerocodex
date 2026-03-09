package ui

import (
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/cobyzero/zerocodex/internal/application"
)

func BuildWindow(
	w fyne.Window,
	selectProject *application.SelectProject,
	chat *application.Chat,
) {
	currentProject := ""
	isRunning := false
	var transcript strings.Builder
	savedProjects, _ := selectProject.ListSaved()

	projectLabel := widget.NewLabel("Ningun proyecto seleccionado")
	projectLabel.Wrapping = fyne.TextWrapBreak
	projectLabel.TextStyle = fyne.TextStyle{Bold: true}

	statusValue := widget.NewLabel("Listo")
	statusValue.Importance = widget.SuccessImportance

	chatView := widget.NewRichTextFromMarkdown("_Sin mensajes. Selecciona un proyecto y escribe tu solicitud._")
	chatView.Wrapping = fyne.TextWrapWord
	chatScroll := container.NewVScroll(chatView)

	updateChat := func() {
		md := transcript.String()
		if strings.TrimSpace(md) == "" {
			md = "_Sin mensajes. Selecciona un proyecto y escribe tu solicitud._"
		}
		chatView.ParseMarkdown(md)
		chatView.Refresh()
		chatScroll.ScrollToBottom()
	}

	appendMessage := func(role, content string) {
		content = strings.TrimSpace(content)
		if content == "" {
			return
		}
		if transcript.Len() > 0 {
			transcript.WriteString("\n\n---\n\n")
		}
		switch role {
		case "user":
			transcript.WriteString("### Tu\n")
			transcript.WriteString("> " + content)
		case "assistant":
			transcript.WriteString("### ZeroCodex\n")
			transcript.WriteString(content)
		case "system":
			transcript.WriteString("`System` " + content)
		case "error":
			transcript.WriteString("`Error` " + content)
		default:
			transcript.WriteString(content)
		}
		updateChat()
	}

	prompt := widget.NewMultiLineEntry()
	prompt.SetPlaceHolder("Escribe lo que quieres cambiar en el proyecto...")
	prompt.SetMinRowsVisible(2)

	sectionTitle := func(icon fyne.Resource, title string) fyne.CanvasObject {
		return container.NewHBox(
			widget.NewIcon(icon),
			widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		)
	}

	glassPanel := func(content fyne.CanvasObject) fyne.CanvasObject {
		bg := canvas.NewRectangle(color.NRGBA{R: 22, G: 25, B: 31, A: 210})
		bg.StrokeColor = color.NRGBA{R: 66, G: 75, B: 92, A: 220}
		bg.StrokeWidth = 1
		return container.NewStack(bg, container.NewPadded(content))
	}

	projectsList := widget.NewList(
		func() int { return len(savedProjects) },
		func() fyne.CanvasObject {
			label := widget.NewLabel("")
			label.Truncation = fyne.TextTruncateEllipsis
			return label
		},
		func(i widget.ListItemID, obj fyne.CanvasObject) {
			if i < 0 || i >= len(savedProjects) {
				obj.(*widget.Label).SetText("")
				return
			}
			obj.(*widget.Label).SetText(savedProjects[i])
		},
	)
	projectsList.OnSelected = func(id widget.ListItemID) {
		if isRunning {
			return
		}
		if id < 0 || id >= len(savedProjects) {
			return
		}
		path := savedProjects[id]
		currentProject = path
		projectLabel.SetText(path)
		appendMessage("system", "Proyecto seleccionado: "+path)
	}

	refreshProjects := func(selectPath string) {
		savedProjects, _ = selectProject.ListSaved()
		projectsList.Refresh()
		if strings.TrimSpace(selectPath) == "" {
			return
		}
		for i, p := range savedProjects {
			if p == selectPath {
				projectsList.Select(i)
				return
			}
		}
	}

	selectBtn := widget.NewButton("Select Project", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if uri == nil {
				return
			}
			path, err := selectProject.Execute(uri.Path())
			if err == nil && path != "" {
				currentProject = path
				projectLabel.SetText(path)
				appendMessage("system", "Proyecto seleccionado: "+path)
				refreshProjects(path)
			} else if err != nil {
				dialog.ShowError(err, w)
			}
		}, w)
	})
	selectBtn.Icon = theme.FolderOpenIcon()
	selectBtn.Importance = widget.HighImportance

	runBtn := widget.NewButtonWithIcon("Enviar", theme.MailSendIcon(), nil)
	runBtn.Importance = widget.HighImportance
	runAgent := func() {
		if isRunning {
			return
		}
		userPrompt := strings.TrimSpace(prompt.Text)
		if userPrompt == "" {
			return
		}

		prompt.SetText("")
		appendMessage("user", userPrompt)

		isRunning = true
		statusValue.SetText("Procesando...")
		statusValue.Importance = widget.WarningImportance
		runBtn.Disable()
		selectBtn.Disable()

		projectPath := currentProject
		go func() {
			response, err := chat.Execute(projectPath, userPrompt, func(msg string) {
				fyne.Do(func() {
					appendMessage("system", msg)
				})
			})

			fyne.Do(func() {
				if err != nil {
					appendMessage("error", err.Error())
				} else {
					changeMD := buildGitChangesMarkdown(projectPath)
					if changeMD != "" {
						response = response + "\n\n---\n\n" + changeMD
					}
					appendMessage("assistant", response)
				}
				isRunning = false
				statusValue.SetText("Listo")
				statusValue.Importance = widget.SuccessImportance
				runBtn.Enable()
				selectBtn.Enable()
			})
		}()
	}
	runBtn.OnTapped = runAgent

	clearBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		transcript.Reset()
		updateChat()
	})
	clearBtn.Importance = widget.LowImportance

	brandTitle := widget.NewLabelWithStyle("ZeroCodex", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	brandSub := widget.NewLabel("AI Coding Agent")
	brandSub.Importance = widget.MediumImportance

	sidebarContent := container.NewVBox(
		container.NewPadded(container.NewHBox(widget.NewIcon(theme.ComputerIcon()), brandTitle)),
		container.NewPadded(brandSub),
		widget.NewSeparator(),
		sectionTitle(theme.StorageIcon(), "Proyectos Recientes"),
		container.NewPadded(selectBtn),
		container.NewPadded(container.NewVScroll(projectsList)),
	)
	sidebar := container.NewPadded(glassPanel(sidebarContent))

	statusBadge := container.NewHBox(
		widget.NewIcon(theme.ConfirmIcon()),
		statusValue,
	)

	topBarLeft := container.NewVBox(
		sectionTitle(theme.FolderIcon(), "Proyecto Activo"),
		projectLabel,
	)

	topBarContent := container.NewBorder(
		nil,
		nil,
		topBarLeft,
		container.NewHBox(clearBtn, layout.NewSpacer(), statusBadge),
		nil,
	)
	topBar := container.NewPadded(glassPanel(topBarContent))

	chatHeader := container.NewPadded(sectionTitle(theme.MailSendIcon(), "Conversacion"))
	chatArea := glassPanel(container.NewBorder(chatHeader, nil, nil, nil, container.NewPadded(chatScroll)))

	composerContent := container.NewBorder(nil, nil, nil, runBtn, prompt)
	composer := container.NewPadded(glassPanel(composerContent))

	mainPane := container.NewBorder(topBar, composer, nil, nil, container.NewPadded(chatArea))
	split := container.NewHSplit(sidebar, mainPane)
	split.SetOffset(0.25)

	w.SetContent(split)
	refreshProjects("")
}
