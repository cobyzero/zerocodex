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

	// Colores modernos
	primaryColor := color.NRGBA{R: 41, G: 98, B: 255, A: 255}
	successColor := color.NRGBA{R: 46, G: 204, B: 113, A: 255}
	warningColor := color.NRGBA{R: 241, G: 196, B: 15, A: 255}
	dangerColor := color.NRGBA{R: 231, G: 76, B: 60, A: 255}
	sidebarBg := color.NRGBA{R: 22, G: 25, B: 31, A: 240}
	cardBg := color.NRGBA{R: 30, G: 33, B: 40, A: 230}
	borderColor := color.NRGBA{R: 66, G: 75, B: 92, A: 180}
	textPrimary := color.NRGBA{R: 240, G: 242, B: 245, A: 255}
	textSecondary := color.NRGBA{R: 170, G: 175, B: 185, A: 255}

	projectLabel := widget.NewLabel("Ningun proyecto seleccionado")
	projectLabel.Wrapping = fyne.TextWrapBreak
	projectLabel.TextStyle = fyne.TextStyle{Bold: true}
	projectLabel.Importance = widget.MediumImportance

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
	prompt.Wrapping = fyne.TextWrapWord

	sectionTitle := func(icon fyne.Resource, title string) fyne.CanvasObject {
		iconWidget := widget.NewIcon(icon)
		titleWidget := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
		titleWidget.Importance = widget.HighImportance
		return container.NewHBox(iconWidget, titleWidget)
	}

	roundedCard := func(content fyne.CanvasObject, bgColor color.Color) fyne.CanvasObject {
		bg := canvas.NewRectangle(bgColor)
		bg.StrokeColor = borderColor
		bg.StrokeWidth = 1
		bg.CornerRadius = 12
		return container.NewStack(bg, container.NewPadded(content))
	}

	gradientCard := func(content fyne.CanvasObject) fyne.CanvasObject {
		// Fondo con gradiente sutil
		bg := canvas.NewRectangle(cardBg)
		bg.StrokeColor = borderColor
		bg.StrokeWidth = 1
		bg.CornerRadius = 12
		
		// Efecto de brillo en la parte superior
		highlight := canvas.NewRectangle(color.NRGBA{R: 255, G: 255, B: 255, A: 15})
		highlight.CornerRadius = 12
		highlight.SetMinSize(fyne.NewSize(0, 2))
		
		return container.NewStack(
			bg,
			highlight,
			container.NewPadded(content),
		)
	}

	statusBadge := func(text string, importance widget.Importance) fyne.CanvasObject {
		badge := widget.NewLabel(text)
		badge.Importance = importance
		badge.TextStyle = fyne.TextStyle{Bold: true}
		
		bg := canvas.NewRectangle(color.Transparent)
		bg.CornerRadius = 8
		
		switch importance {
		case widget.SuccessImportance:
			bg.FillColor = color.NRGBA{R: 46, G: 204, B: 113, A: 30}
			bg.StrokeColor = successColor
		case widget.WarningImportance:
			bg.FillColor = color.NRGBA{R: 241, G: 196, B: 15, A: 30}
			bg.StrokeColor = warningColor
		case widget.DangerImportance:
			bg.FillColor = color.NRGBA{R: 231, G: 76, B: 60, A: 30}
			bg.StrokeColor = dangerColor
		default:
			bg.FillColor = color.NRGBA{R: 66, G: 75, B: 92, A: 30}
			bg.StrokeColor = borderColor
		}
		bg.StrokeWidth = 1
		
		return container.NewStack(
			bg,
			container.NewPadded(badge),
		)
	}

	projectsList := widget.NewList(
		func() int { return len(savedProjects) },
		func() fyne.CanvasObject {
			label := widget.NewLabel("")
			label.Truncation = fyne.TextTruncateEllipsis
			label.Importance = widget.MediumImportance
			return container.NewPadded(label)
		},
		func(i widget.ListItemID, obj fyne.CanvasObject) {
			if i < 0 || i >= len(savedProjects) {
				obj.(*container.Padded).Objects[0].(*widget.Label).SetText("")
				return
			}
			obj.(*container.Padded).Objects[0].(*widget.Label).SetText(savedProjects[i])
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
			runBtn.Importance = widget.HighImportance
			return
		}
		runBtn.Disable()
		runBtn.Importance = widget.MediumImportance
	}

	startProjectAnalysis := func(path string) {
		currentProject = path
		projectLabel.SetText(path)
		appendMessage("system", "Proyecto seleccionado: "+path)

		isAnalyzing = true
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
					appendMessage("error", "Analisis de contexto fallido: "+err.Error())
				} else {
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

	brandTitle := widget.NewLabelWithStyle("ZeroCodex", fyne.TextAlignLeading, fyne.TextStyle{Bold: true, Italic: true})
	brandTitle.TextSize = 24
	brandSub := widget.NewLabel("AI Coding Agent")
	brandSub.Importance = widget.MediumImportance
	brandSub.TextSize = 12

	// Sidebar con fondo oscuro
	sidebarBgRect := canvas.NewRectangle(sidebarBg)
	sidebarBgRect.CornerRadius = 0
	
	sidebarContent := container.NewVBox(
		container.NewPadded(container.NewHBox(
			widget.NewIcon(theme.ComputerIcon()),
			container.NewVBox(brandTitle, brandSub),
		)),
		widget.NewSeparator(),
		container.NewPadded(sectionTitle(theme.StorageIcon(), "Proyectos Recientes")),
		container.NewPadded(selectBtn),
		container.NewPadded(roundedCard(
			container.NewVScroll(projectsList),
			color.NRGBA{R: 25, G: 28, B: 35, A: 200},
		)),
	)
	sidebar := container.NewStack(
		sidebarBgRect,
		container.NewPadded(sidebarContent),
	)

	// Top bar mejorada
	projectSection := container.NewVBox(
		sectionTitle(theme.FolderIcon(), "Proyecto Activo"),
		roundedCard(
			container.NewPadded(projectLabel),
			color.NRGBA{R: 25, G: 28, B: 35, A: 200},
		),
	)

	statusSection := container.NewHBox(
		widget.NewIcon(theme.ConfirmIcon()),
		statusBadge("Listo", widget.SuccessImportance),
	)

	topBarContent := container.NewBorder(
		nil,
		nil,
		projectSection,
		container.NewHBox(
			clearBtn,
			layout.NewSpacer(),
			statusSection,
		),
		container.NewVBox(
			analyzeLabel,
			analyzeBar,
		),
	)
	topBar := container.NewPadded(gradientCard(topBarContent))

	// Chat area mejorada
	chatHeader := container.NewPadded(sectionTitle(theme.MailSendIcon(), "Conversacion"))
	chatArea := gradientCard(
		container.NewBorder(
			chatHeader,
			nil,
			nil,
			nil,
			container.NewPadded(chatScroll),
		),
	)

	// Composer mejorado
	composerContent := container.NewBorder(
		nil,
		nil,
		nil,
		container.NewPadded(runBtn),
		container.NewPadded(prompt),
	)
	composer := container.NewPadded(gradientCard(composerContent))

	// Layout principal
	mainPane := container.NewBorder(
		topBar,
		composer,
		nil,
		nil,
		container.NewPadded(chatArea),
	)
	
	split := container.NewHSplit(sidebar, mainPane)
	split.SetOffset(0.25)

	w.SetContent(split)
	refreshProjects("")
	updateRunState()
	
	// Aplicar tema oscuro por defecto
	w.Canvas().SetTheme(theme.DarkTheme())
}
