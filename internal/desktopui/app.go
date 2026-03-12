//go:build cgo

package desktopui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/Arbousier1/Minecraft-Config-Converter/internal/analyzer"
	"github.com/Arbousier1/Minecraft-Config-Converter/internal/workflow"
)

func Run(baseDir string) error {
	service, err := workflow.New(baseDir)
	if err != nil {
		return err
	}

	desktop := app.NewWithID("github.com.Arbousier1.MCC")
	window := desktop.NewWindow("MCC")
	window.Resize(fyne.NewSize(1120, 760))

	var currentSession *workflow.Session

	fileValue := widget.NewLabel("No package selected")
	fileValue.Wrapping = fyne.TextWrapWord

	sourceSelect := widget.NewSelect(nil, nil)
	sourceSelect.PlaceHolder = "Analyze a package first"
	sourceSelect.Disable()

	targetSelect := widget.NewSelect([]string{"CraftEngine"}, nil)
	targetSelect.SetSelected("CraftEngine")
	targetSelect.Disable()

	namespaceEntry := widget.NewEntry()
	namespaceEntry.SetPlaceHolder("Optional namespace override")

	formatsValue := widget.NewLabel("Not analyzed")
	formatsValue.Wrapping = fyne.TextWrapWord
	contentValue := widget.NewLabel("-")
	contentValue.Wrapping = fyne.TextWrapWord
	completenessValue := widget.NewLabel("-")
	completenessValue.Wrapping = fyne.TextWrapWord
	detailsValue := widget.NewLabel("-")
	detailsValue.Wrapping = fyne.TextWrapWord
	statusValue := widget.NewLabel("Ready")
	statusValue.Wrapping = fyne.TextWrapWord

	warningsValue := widget.NewLabel("No warnings")
	warningsValue.Wrapping = fyne.TextWrapWord

	logOutput := widget.NewMultiLineEntry()
	logOutput.Disable()
	logOutput.Wrapping = fyne.TextWrapWord
	logOutput.SetMinRowsVisible(16)

	progress := widget.NewProgressBarInfinite()
	progress.Hide()

	var openButton *widget.Button
	var analyzeButton *widget.Button
	var convertButton *widget.Button

	appendLog := func(message string) {
		line := fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), message)
		if logOutput.Text == "" {
			logOutput.SetText(line)
			return
		}
		logOutput.Append("\n" + line)
	}

	setBusy := func(busy bool, status string) {
		if busy {
			progress.Show()
			openButton.Disable()
			analyzeButton.Disable()
			convertButton.Disable()
			sourceSelect.Disable()
			targetSelect.Disable()
			namespaceEntry.Disable()
		} else {
			progress.Hide()
			openButton.Enable()
			analyzeButton.Enable()
			namespaceEntry.Enable()
			if currentSession != nil {
				sourceSelect.Enable()
				if len(currentSession.AvailableTargets) > 0 {
					targetSelect.Enable()
					convertButton.Enable()
				}
			}
		}
		statusValue.SetText(status)
	}

	updateReport := func(session *workflow.Session) {
		currentSession = session
		fileValue.SetText(session.SourceZipPath)
		formatsValue.SetText(joinOrFallback(session.Report.Formats, ", ", "Unknown"))
		contentValue.SetText(joinOrFallback(session.Report.ContentTypes, ", ", "None"))
		completenessValue.SetText(formatCompleteness(session.Report))
		detailsValue.SetText(formatDetails(session.Report))
		warningsValue.SetText(reportWarnings(session))

		sourceSelect.SetOptions(session.AvailableSourceFormat)
		if len(session.AvailableSourceFormat) > 0 {
			sourceSelect.SetSelected(session.AvailableSourceFormat[0])
			sourceSelect.Enable()
		} else {
			sourceSelect.ClearSelected()
			sourceSelect.Disable()
		}

		targetSelect.SetOptions(session.AvailableTargets)
		if len(session.AvailableTargets) > 0 {
			targetSelect.SetSelected(session.AvailableTargets[0])
			targetSelect.Enable()
			convertButton.Enable()
		} else {
			targetSelect.ClearSelected()
			targetSelect.Disable()
			convertButton.Disable()
		}

		if session.ItemsAdderNamespace != "" {
			namespaceEntry.SetText(session.ItemsAdderNamespace)
		} else {
			namespaceEntry.SetText("")
		}
	}

	runAnalyze := func(zipPath string) {
		if strings.TrimSpace(zipPath) == "" {
			dialog.NewInformation("Select Package", "Choose a source zip package first.", window).Show()
			return
		}

		go func() {
			fyne.Do(func() {
				setBusy(true, "Analyzing package...")
				appendLog("Analyzing " + filepath.Base(zipPath))
			})

			session, err := service.AnalyzeZip(zipPath)

			fyne.Do(func() {
				if err != nil {
					setBusy(false, "Analyze failed")
					appendLog("Analyze failed: " + err.Error())
					dialog.NewError(err, window).Show()
					return
				}

				updateReport(session)
				setBusy(false, "Analysis complete")
				appendLog("Detected source formats: " + joinOrFallback(session.AvailableSourceFormat, ", ", "none"))
			})
		}()
	}

	openSourceDialog := func() {
		fileDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.NewError(err, window).Show()
				return
			}
			if reader == nil {
				return
			}

			zipPath := reader.URI().Path()
			_ = reader.Close()
			runAnalyze(zipPath)
		}, window)
		fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{".zip"}))
		fileDialog.SetTitleText("Open Source Package")
		fileDialog.Show()
	}

	openButton = widget.NewButtonWithIcon("Open ZIP", theme.FolderOpenIcon(), openSourceDialog)
	analyzeButton = widget.NewButtonWithIcon("Analyze Again", theme.ViewRefreshIcon(), func() {
		if currentSession != nil {
			runAnalyze(currentSession.SourceZipPath)
			return
		}
		openSourceDialog()
	})
	convertButton = widget.NewButtonWithIcon("Convert And Save", theme.DocumentSaveIcon(), func() {
		if currentSession == nil {
			dialog.NewInformation("Analyze First", "Analyze a package before converting it.", window).Show()
			return
		}
		if sourceSelect.Selected == "" {
			dialog.NewInformation("Source Format", "Choose the source format to convert.", window).Show()
			return
		}
		if targetSelect.Selected == "" {
			dialog.NewInformation("Target Format", "Choose the target format.", window).Show()
			return
		}

		saveDialog := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
			if err != nil {
				dialog.NewError(err, window).Show()
				return
			}
			if writer == nil {
				return
			}

			savePath := writer.URI().Path()
			_ = writer.Close()

			go func() {
				fyne.Do(func() {
					setBusy(true, "Converting package...")
					appendLog(fmt.Sprintf("Converting %s -> %s", sourceSelect.Selected, targetSelect.Selected))
				})

				outputPath, convertErr := service.Convert(currentSession, sourceSelect.Selected, targetSelect.Selected, strings.TrimSpace(namespaceEntry.Text), savePath)

				fyne.Do(func() {
					if convertErr != nil {
						setBusy(false, "Convert failed")
						appendLog("Convert failed: " + convertErr.Error())
						dialog.NewError(convertErr, window).Show()
						return
					}

					setBusy(false, "Convert complete")
					appendLog("Saved converted archive to " + outputPath)
					dialog.NewInformation("Conversion Complete", "Saved archive to:\n"+outputPath, window).Show()
				})
			}()
		}, window)
		saveDialog.SetFilter(storage.NewExtensionFileFilter([]string{".zip"}))
		saveDialog.SetFileName(workflow.SuggestedArchiveName(currentSession.OriginalFilename, targetSelect.Selected))
		saveDialog.SetTitleText("Save Converted Archive")
		saveDialog.Show()
	})
	convertButton.Disable()

	sourceSelect.OnChanged = func(selected string) {
		if currentSession == nil {
			return
		}
		if selected == "ItemsAdder" && currentSession.ItemsAdderNamespace != "" && strings.TrimSpace(namespaceEntry.Text) == "" {
			namespaceEntry.SetText(currentSession.ItemsAdderNamespace)
		}
	}

	summaryForm := widget.NewForm(
		widget.NewFormItem("Package", fileValue),
		widget.NewFormItem("Formats", formatsValue),
		widget.NewFormItem("Content", contentValue),
		widget.NewFormItem("Completeness", completenessValue),
		widget.NewFormItem("Details", detailsValue),
	)

	controlsForm := widget.NewForm(
		widget.NewFormItem("Source", sourceSelect),
		widget.NewFormItem("Target", targetSelect),
		widget.NewFormItem("Namespace", namespaceEntry),
	)

	leftPane := container.NewVBox(
		widget.NewCard("Workflow", "", controlsForm),
		widget.NewCard("Package Report", "", summaryForm),
		widget.NewCard("Warnings", "", warningsValue),
	)
	rightPane := widget.NewCard("Activity", "", logOutput)
	center := container.NewHSplit(leftPane, rightPane)
	center.Offset = 0.48

	topBar := container.NewHBox(openButton, analyzeButton, convertButton)
	bottomBar := container.NewHBox(progress, statusValue)

	window.SetContent(container.NewBorder(topBar, bottomBar, nil, nil, center))
	window.ShowAndRun()
	return nil
}

func joinOrFallback(items []string, separator, fallback string) string {
	if len(items) == 0 {
		return fallback
	}
	return strings.Join(items, separator)
}

func formatCompleteness(report analyzer.Report) string {
	return fmt.Sprintf(
		"Items=%t  Categories=%t  Resources=%t",
		report.Completeness.ItemsConfig,
		report.Completeness.CategoriesConfig,
		report.Completeness.ResourceFiles,
	)
}

func formatDetails(report analyzer.Report) string {
	return fmt.Sprintf(
		"Items: %d\nTextures: %d\nModels: %d",
		report.Details.ItemCount,
		report.Details.TextureCount,
		report.Details.ModelCount,
	)
}

func reportWarnings(session *workflow.Session) string {
	if session == nil {
		return "No warnings"
	}

	warnings := make([]string, 0, 2)
	if contains(session.Report.Formats, "CraftEngine") && len(session.AvailableTargets) > 0 {
		warnings = append(warnings, "Detected existing CraftEngine content. Conversion may overwrite current files.")
	}
	if len(session.AvailableTargets) == 0 {
		warnings = append(warnings, "No supported conversion target detected for this package.")
	}
	if len(warnings) == 0 {
		return "No warnings"
	}
	return strings.Join(warnings, "\n")
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
