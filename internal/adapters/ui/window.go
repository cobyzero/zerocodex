package ui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
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

	projectLabel := widget.NewLabel("No project selected")
	projectLabel.Wrapping = fyne.TextWrapBreak

	statusValue := widget.NewLabel("Idle")
	statusValue.Importance = widget.SuccessImportance

	chatView := widget.NewRichTextFromMarkdown("_No conversation yet. Select a project and send a prompt._")
	chatView.Wrapping = fyne.TextWrapWord
	chatScroll := container.NewVScroll(chatView)

	updateChat := func() {
		md := transcript.String()
		if strings.TrimSpace(md) == "" {
			md = "_No conversation yet. Select a project and send a prompt._"
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
			transcript.WriteString("\n\n")
		}
		switch role {
		case "user":
			transcript.WriteString("**You**\n")
			transcript.WriteString("> " + content)
		case "assistant":
			transcript.WriteString("**ZeroCodex**\n")
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
	prompt.SetPlaceHolder("Describe your coding task...")
	prompt.SetMinRowsVisible(2)

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
		appendMessage("system", "Project selected: "+path)
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
				appendMessage("system", "Project selected: "+path)
				refreshProjects(path)
			} else if err != nil {
				dialog.ShowError(err, w)
			}
		}, w)
	})
	selectBtn.Icon = theme.FolderOpenIcon()

	runBtn := widget.NewButtonWithIcon("Run Agent", theme.MediaPlayIcon(), nil)
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
		statusValue.SetText("Running...")
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
				statusValue.SetText("Idle")
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

	sidebarTitle := widget.NewLabelWithStyle("Projects", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	sidebar := container.NewBorder(
		container.NewVBox(sidebarTitle, selectBtn),
		nil,
		nil,
		nil,
		container.NewVScroll(projectsList),
	)

	topBar := container.NewBorder(
		nil,
		nil,
		nil,
		container.NewHBox(clearBtn, statusValue),
		projectLabel,
	)

	composer := container.NewBorder(nil, nil, nil, runBtn, prompt)
	mainPane := container.NewBorder(topBar, composer, nil, nil, chatScroll)
	split := container.NewHSplit(sidebar, mainPane)
	split.SetOffset(0.26)

	runBtn.Text = "Send"
	runBtn.Icon = theme.MailSendIcon()

	w.SetContent(split)
	refreshProjects("")
}
