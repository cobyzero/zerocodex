package ui

import (
	"image/color"
	"strconv"
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
	isAnalyzing := false
	var transcript strings.Builder
	savedProjects, _ := selectProject.ListSaved()

	projectLabel := widget.NewLabel("Ningun proyecto seleccionado")
	projectLabel.Wrapping = fyne.TextWrapBreak
	projectLabel.TextStyle = fyne.TextStyle{Bold: true}

	statusValue := widget.NewLabel("Listo")
	statusValue.Importance = widget.SuccessImportance
	analyzeLabel := widget.NewLabel("")
	analyzeBar := widget.NewProgressBar()
	analyzeBar.Min = 0
	analyzeBar.Max = 1
	analyzeBar.Hide()

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

	var runBtn *widget.Button
	var selectBtn *widget.Button

	updateRunState := func() {
		canRun := !isRunning && !isAnalyzing && strings.TrimSpace(currentProject) != ""
		if canRun {
			runBtn.Enable()
			return
		}
		runBtn.Disable()
	}

	startProjectAnalysis := func(path string) {
		currentProject = path
		projectLabel.SetText(path)
		appendMessage("system", "Proyecto seleccionado: "+path)

		isAnalyzing = true
		statusValue.SetText("Analizando proyecto...")
		statusValue.Importance = widget.WarningImportance
		analyzeLabel.SetText("Analizando contexto del proyecto...")
		analyzeBar.Max = 1
		analyzeBar.SetValue(0)
		analyzeBar.Show()
		selectBtn.Disable()
		updateRunState()

		go func() {
			err := chat.AnalyzeProject(path, func(done, total int) {
				fyne.Do(func() {
					if total <= 0 {
						analyzeBar.Max = 1
						analyzeBar.SetValue(0)
						return
					}
					analyzeBar.Max = float64(total)
					analyzeBar.SetValue(float64(done))
					analyzeLabel.SetText("Analizando contexto del proyecto... (" + strconv.Itoa(done) + "/" + strconv.Itoa(total) + ")")
				})
			})

			fyne.Do(func() {
				isAnalyzing = false
				analyzeLabel.SetText("")
				analyzeBar.Hide()
				if err != nil {
					statusValue.SetText("Error de analisis")
					statusValue.Importance = widget.DangerImportance
					appendMessage("error", "Analisis de contexto fallido: "+err.Error())
				} else {
					statusValue.SetText("Listo")
					statusValue.Importance = widget.SuccessImportance
					appendMessage("system", "Analisis completado. Ya puedes consultar.")
				}
				selectBtn.Enable()
				updateRunState()
			})
		}()
	}

	projectsList.OnSelected = func(id widget.ListItemID) {
		if isRunning || isAnalyzing {
			return
		}
		if id < 0 || id >= len(savedProjects) {
			return
		}
		startProjectAnalysis(savedProjects[id])
	}

	selectBtn = widget.NewButton("Select Project", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if uri == nil {
				return
			}
			path, err := selectProject.Execute(uri.Path())
			if err == nil && path != "" {
				refreshProjects(path)
				startProjectAnalysis(path)
			} else if err != nil {
				dialog.ShowError(err, w)
			}
		}, w)
	})
	selectBtn.Icon = theme.FolderOpenIcon()
	selectBtn.Importance = widget.HighImportance

	runBtn = widget.NewButtonWithIcon("Enviar", theme.MailSendIcon(), nil)
	runBtn.Importance = widget.HighImportance
	runAgent := func() {
		if isRunning || isAnalyzing {
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
		selectBtn.Disable()
		updateRunState()

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
				if !isAnalyzing {
					statusValue.SetText("Listo")
					statusValue.Importance = widget.SuccessImportance
				}
				selectBtn.Enable()
				updateRunState()
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
		container.NewVBox(analyzeLabel, analyzeBar),
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
	updateRunState()
}
