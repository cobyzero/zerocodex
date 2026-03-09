package ui

import (
	"strings"

	"fyne.io/fyne/v2"
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

	title := widget.NewLabelWithStyle("ZeroCodex Agent", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	subtitle := widget.NewLabel("Professional coding assistant for your project")
	subtitle.Importance = widget.MediumImportance

	projectLabel := widget.NewLabel("No project selected")
	projectLabel.Wrapping = fyne.TextWrapBreak

	statusValue := widget.NewLabelWithStyle("Idle", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
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
			transcript.WriteString("\n\n---\n\n")
		}
		switch role {
		case "user":
			transcript.WriteString("### You\n")
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
	prompt.SetPlaceHolder("Describe your coding task...")
	prompt.SetMinRowsVisible(3)

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
		statusValue.SetText("Running")
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

	clearBtn := widget.NewButtonWithIcon("Clear Chat", theme.DeleteIcon(), func() {
		transcript.Reset()
		updateChat()
	})
	clearBtn.Importance = widget.LowImportance

	header := container.NewVBox(
		title,
		subtitle,
	)

	projectCard := widget.NewCard("Workspace", "", projectLabel)
	statusCard := widget.NewCard("Agent Status", "", statusValue)
	leftPanel := container.NewVBox(
		selectBtn,
		projectCard,
		statusCard,
		widget.NewSeparator(),
		widget.NewLabel("Guidance"),
		widget.NewLabel("- Ask specific tasks for better answers."),
		widget.NewLabel("- Include target files when possible."),
		widget.NewLabel("- Keep prompts short and explicit."),
		layout.NewSpacer(),
		clearBtn,
	)

	chatCard := widget.NewCard("Conversation", "", chatScroll)
	composer := container.NewBorder(nil, nil, nil, runBtn, prompt)
	rightPanel := container.NewBorder(nil, composer, nil, nil, chatCard)

	content := container.NewHSplit(leftPanel, rightPanel)
	content.SetOffset(0.30)
	layout := container.NewBorder(header, nil, nil, nil, content)

	w.SetContent(layout)
}
