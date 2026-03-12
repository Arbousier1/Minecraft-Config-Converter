//go:build cgo

package desktopui

import (
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/Arbousier1/Minecraft-Config-Converter/internal/analyzer"
	"github.com/Arbousier1/Minecraft-Config-Converter/internal/workflow"
)

type pluginDef struct {
	ID   string
	Name string
}

type workflowStep struct {
	ID     string
	Label  string
	Helper string
}

var pluginDefs = []pluginDef{
	{ID: "ItemsAdder", Name: "ItemsAdder"},
	{ID: "Nexo", Name: "Nexo"},
	{ID: "Oraxen", Name: "Oraxen"},
	{ID: "CraftEngine", Name: "CraftEngine"},
	{ID: "MythicCrucible", Name: "MythicCrucible"},
}

var workflowSteps = []workflowStep{
	{ID: "import", Label: "Import Package", Helper: "ZIP"},
	{ID: "inspect", Label: "Inspect Content", Helper: "Analyzer"},
	{ID: "convert", Label: "Run Conversion", Helper: "Converter"},
	{ID: "export", Label: "Export Result", Helper: "Output"},
}

var stageLabels = map[string]string{
	"idle":       "Waiting For Package",
	"analyzing":  "Analyzing",
	"ready":      "Ready To Convert",
	"converting": "Converting",
	"done":       "Completed",
	"error":      "Error",
}

func Run(baseDir string) error {
	service, err := workflow.New(baseDir)
	if err != nil {
		return err
	}

	desktop := app.NewWithID("github.com.Arbousier1.MCC")
	window := desktop.NewWindow("Minecraft Config Converter")
	window.Resize(fyne.NewSize(1280, 860))

	var (
		currentSession *workflow.Session
		currentStage   = "idle"
		selectedSource string
		selectedTarget string
		lastOutputPath string
		lastStatusText = "Waiting for package"
		lastErrorText  string
	)

	fileValue := newValueLabel("No package loaded")
	summaryValue := newValueLabel("Nothing analyzed yet")
	statusValue := newValueLabel(stageLabels[currentStage])
	sessionIDValue := newMonoValueLabel("Not created")
	directionValue := newValueLabel("Not selected")
	namespaceValue := newMonoValueLabel("Auto")
	contentValue := newValueLabel("-")
	completenessValue := newValueLabel("-")
	detailsValue := newValueLabel("-")
	warningsValue := newValueLabel("No warnings")
	noteValue := newValueLabel("The desktop UI mirrors the old web layout: workflow on the left, work area in the middle, summary on the right.")
	progressStateValue := newValueLabel(lastStatusText)
	progressPercentValue := newMonoValueLabel("Idle")
	resultValue := newMonoValueLabel("No output yet")
	sourceSummaryValue := newValueLabel("No package loaded")
	filenameValue := newMonoValueLabel("No file")

	namespaceEntry := widget.NewEntry()
	namespaceEntry.SetPlaceHolder("Optional namespace override")

	logOutput := widget.NewMultiLineEntry()
	logOutput.Disable()
	logOutput.Wrapping = fyne.TextWrapWord
	logOutput.SetMinRowsVisible(14)

	progress := widget.NewProgressBarInfinite()
	progress.Hide()

	workflowLabels := make(map[string]*widget.Label, len(workflowSteps))
	workflowBox := container.NewVBox()
	for _, step := range workflowSteps {
		row := widget.NewLabel("")
		row.Wrapping = fyne.TextWrapWord
		workflowLabels[step.ID] = row
		workflowBox.Add(row)
	}

	sourceButtons := make(map[string]*widget.Button, len(pluginDefs))
	sourceStatus := make(map[string]*widget.Label, len(pluginDefs))
	sourceList := container.NewVBox()

	targetButtons := make(map[string]*widget.Button, len(pluginDefs))
	targetStatus := make(map[string]*widget.Label, len(pluginDefs))
	targetList := container.NewVBox()

	var (
		openButton    *widget.Button
		analyzeButton *widget.Button
		convertButton *widget.Button
		resetButton   *widget.Button
		exitButton    *widget.Button
	)

	appendLog := func(message string) {
		line := fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), message)
		if logOutput.Text == "" {
			logOutput.SetText(line)
			return
		}
		logOutput.Append("\n" + line)
	}

	updateSessionLabels := func() {
		if currentSession == nil {
			fileValue.SetText("No package loaded")
			filenameValue.SetText("No file")
			summaryValue.SetText("Nothing analyzed yet")
			sessionIDValue.SetText("Not created")
			contentValue.SetText("-")
			completenessValue.SetText("-")
			detailsValue.SetText("-")
			warningsValue.SetText(lastErrorTextOrDefault(lastErrorText, "No warnings"))
			sourceSummaryValue.SetText("No package loaded")
			directionValue.SetText("Not selected")
			namespaceValue.SetText(valueOrAuto(namespaceEntry.Text))
			resultValue.SetText(valueOrDefault(lastOutputPath, "No output yet"))
			return
		}

		fileValue.SetText(currentSession.SourceZipPath)
		filenameValue.SetText(currentSession.OriginalFilename)
		summaryValue.SetText(buildSummaryText(currentSession))
		sessionIDValue.SetText(currentSession.ID)
		contentValue.SetText(joinOrFallback(currentSession.Report.ContentTypes, " / ", "None"))
		completenessValue.SetText(formatCompleteness(currentSession.Report))
		detailsValue.SetText(formatDetails(currentSession.Report))
		warningsValue.SetText(reportWarnings(currentSession, lastErrorText))
		sourceSummaryValue.SetText(formatSourceSummary(currentSession, selectedSource, selectedTarget, namespaceEntry.Text))
		directionValue.SetText(formatDirection(selectedSource, selectedTarget))
		namespaceValue.SetText(valueOrAuto(namespaceEntry.Text))
		resultValue.SetText(valueOrDefault(lastOutputPath, "No output yet"))
	}

	refreshWorkflow := func() {
		active := activeWorkflowForStage(currentStage, currentSession != nil)
		for _, step := range workflowSteps {
			prefix := "  "
			if step.ID == active {
				prefix = "> "
			}

			text := fmt.Sprintf("%s%s\n%s", prefix+step.Label, "", step.Helper)
			if step.ID == active {
				text = fmt.Sprintf("%s%s\n%s", "> ", step.Label, step.Helper)
			} else {
				text = fmt.Sprintf("  %s\n%s", step.Label, step.Helper)
			}
			workflowLabels[step.ID].SetText(text)
		}
	}

	setButtonText := func(button *widget.Button, selected, available bool) {
		switch {
		case !available:
			button.SetText("Unavailable")
			button.Disable()
		case selected:
			button.SetText("Selected")
			button.Enable()
		default:
			button.SetText("Use")
			button.Enable()
		}
	}

	refreshPluginLists := func() {
		availableSources := map[string]bool{}
		availableTargets := map[string]bool{}
		if currentSession != nil {
			for _, item := range currentSession.AvailableSourceFormat {
				availableSources[item] = true
			}
			for _, item := range currentSession.AvailableTargets {
				availableTargets[item] = true
			}
		}

		for _, plugin := range pluginDefs {
			sourceAvailable := availableSources[plugin.ID]
			if currentSession == nil {
				sourceAvailable = false
			}
			setButtonText(sourceButtons[plugin.ID], selectedSource == plugin.ID, sourceAvailable)
			sourceStatus[plugin.ID].SetText(pluginStateText(sourceAvailable, selectedSource == plugin.ID))

			targetAvailable := availableTargets[plugin.ID]
			if currentSession == nil {
				targetAvailable = false
			}
			setButtonText(targetButtons[plugin.ID], selectedTarget == plugin.ID, targetAvailable)
			targetStatus[plugin.ID].SetText(pluginStateText(targetAvailable, selectedTarget == plugin.ID))
		}
	}

	setBusy := func(busy bool, stage, status string) {
		currentStage = stage
		lastStatusText = status
		statusValue.SetText(stageLabels[stage])
		progressStateValue.SetText(status)
		progressPercentValue.SetText(stage)

		if busy {
			progress.Show()
			openButton.Disable()
			analyzeButton.Disable()
			convertButton.Disable()
			resetButton.Disable()
			namespaceEntry.Disable()
		} else {
			progress.Hide()
			openButton.Enable()
			analyzeButton.Enable()
			resetButton.Enable()
			namespaceEntry.Enable()
			if currentSession != nil && selectedTarget != "" {
				convertButton.Enable()
			} else {
				convertButton.Disable()
			}
		}

		refreshWorkflow()
		updateSessionLabels()
		refreshPluginLists()
	}

	resetState := func() {
		currentSession = nil
		currentStage = "idle"
		selectedSource = ""
		selectedTarget = ""
		lastOutputPath = ""
		lastStatusText = "Waiting for package"
		lastErrorText = ""

		namespaceEntry.SetText("")
		logOutput.SetText("")
		statusValue.SetText(stageLabels[currentStage])
		progressStateValue.SetText(lastStatusText)
		progressPercentValue.SetText("Idle")
		progress.Hide()
		updateSessionLabels()
		refreshWorkflow()
		refreshPluginLists()
		convertButton.Disable()
	}

	sourceSelected := func(id string) {
		if currentSession == nil || !contains(currentSession.AvailableSourceFormat, id) {
			return
		}
		selectedSource = id
		if selectedSource == "ItemsAdder" && currentSession.ItemsAdderNamespace != "" && strings.TrimSpace(namespaceEntry.Text) == "" {
			namespaceEntry.SetText(currentSession.ItemsAdderNamespace)
		}
		updateSessionLabels()
		refreshPluginLists()
	}

	targetSelected := func(id string) {
		if currentSession == nil || !contains(currentSession.AvailableTargets, id) {
			return
		}
		selectedTarget = id
		updateSessionLabels()
		refreshPluginLists()
		if currentSession != nil && selectedTarget != "" {
			convertButton.Enable()
		}
	}

	for _, plugin := range pluginDefs {
		pluginID := plugin.ID
		sourceStatusLabel := widget.NewLabel("Unavailable")
		sourceStatusLabel.Wrapping = fyne.TextWrapWord
		sourceButton := widget.NewButton("Unavailable", func() {
			sourceSelected(pluginID)
		})
		sourceButton.Disable()
		sourceButtons[plugin.ID] = sourceButton
		sourceStatus[plugin.ID] = sourceStatusLabel
		sourceList.Add(newPluginRow(plugin, sourceStatusLabel, sourceButton))

		targetStatusLabel := widget.NewLabel("Unavailable")
		targetStatusLabel.Wrapping = fyne.TextWrapWord
		targetButton := widget.NewButton("Unavailable", func() {
			targetSelected(pluginID)
		})
		targetButton.Disable()
		targetButtons[plugin.ID] = targetButton
		targetStatus[plugin.ID] = targetStatusLabel
		targetList.Add(newPluginRow(plugin, targetStatusLabel, targetButton))
	}

	runAnalyze := func(zipPath string) {
		if strings.TrimSpace(zipPath) == "" {
			dialog.NewInformation("Select Package", "Choose a source zip package first.", window).Show()
			return
		}

		go func() {
			fyne.Do(func() {
				lastErrorText = ""
				lastOutputPath = ""
				setBusy(true, "analyzing", "Uploading and analyzing package...")
				appendLog("Analyzing package: " + zipPath)
			})

			session, analyzeErr := service.AnalyzeZip(zipPath)

			fyne.Do(func() {
				if analyzeErr != nil {
					lastErrorText = analyzeErr.Error()
					setBusy(false, "error", "Analyze failed")
					appendLog("Analyze failed: " + analyzeErr.Error())
					dialog.NewError(analyzeErr, window).Show()
					return
				}

				currentSession = session
				selectedSource = firstOrEmpty(session.AvailableSourceFormat)
				selectedTarget = firstOrEmpty(session.AvailableTargets)
				if session.ItemsAdderNamespace != "" {
					namespaceEntry.SetText(session.ItemsAdderNamespace)
				} else {
					namespaceEntry.SetText("")
				}

				setBusy(false, "ready", "Analysis complete")
				progressPercentValue.SetText("100%")
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
		if selectedSource == "" {
			dialog.NewInformation("Source Format", "Choose the source format to convert.", window).Show()
			return
		}
		if selectedTarget == "" {
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
					lastErrorText = ""
					setBusy(true, "converting", fmt.Sprintf("Converting %s -> %s ...", selectedSource, selectedTarget))
					appendLog(fmt.Sprintf("Converting %s -> %s", selectedSource, selectedTarget))
				})

				outputPath, convertErr := service.Convert(currentSession, selectedSource, selectedTarget, strings.TrimSpace(namespaceEntry.Text), savePath)

				fyne.Do(func() {
					if convertErr != nil {
						lastErrorText = convertErr.Error()
						setBusy(false, "error", "Convert failed")
						appendLog("Convert failed: " + convertErr.Error())
						dialog.NewError(convertErr, window).Show()
						return
					}

					lastOutputPath = outputPath
					setBusy(false, "done", "Conversion complete")
					progressPercentValue.SetText("100%")
					appendLog("Saved converted archive to " + outputPath)
					dialog.NewInformation("Conversion Complete", "Saved archive to:\n"+outputPath, window).Show()
				})
			}()
		}, window)
		saveDialog.SetFilter(storage.NewExtensionFileFilter([]string{".zip"}))
		saveDialog.SetFileName(workflow.SuggestedArchiveName(currentSession.OriginalFilename, selectedTarget))
		saveDialog.SetTitleText("Save Converted Archive")
		saveDialog.Show()
	})
	convertButton.Disable()

	resetButton = widget.NewButton("Reset", resetState)
	exitButton = widget.NewButton("Exit", func() {
		window.Close()
	})

	namespaceEntry.OnChanged = func(_ string) {
		updateSessionLabels()
	}

	titleBar := container.NewBorder(
		nil,
		nil,
		container.NewHBox(
			widget.NewLabelWithStyle("Minecraft Config Converter", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewSeparator(),
			widget.NewLabel("Local Conversion Session"),
		),
		container.NewHBox(resetButton, exitButton),
	)

	leftPane := container.NewVBox(
		panel("Workflow", "Current task", workflowBox),
		panel("Session", "Current state", container.NewVBox(
			dataRow("Stage", statusValue),
			dataRow("Session ID", sessionIDValue),
			dataRow("Direction", directionValue),
			dataRow("Namespace", namespaceValue),
		)),
	)

	inputPane := panel("Input", "Import and inspect the package you want to convert", container.NewVBox(
		widget.NewLabel("Select a local .zip package. This desktop shell keeps the old UI flow but saves the final result directly from the app."),
		widget.NewSeparator(),
		dataRow("Loaded Path", fileValue),
		dataRow("Filename", filenameValue),
		container.NewHBox(openButton, analyzeButton),
		progress,
		dataRow("Progress", progressStateValue),
		dataRow("State", progressPercentValue),
	))

	conversionPane := panel("Conversion Setup", "Source, target and output parameters", container.NewVBox(
		widget.NewLabelWithStyle("Source Format", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		sourceList,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Target Format", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		targetList,
		widget.NewSeparator(),
		widget.NewLabel("Namespace Override"),
		namespaceEntry,
		container.NewHBox(convertButton, layout.NewSpacer()),
	))

	inspectPane := panel("Inspection Results", "Analyzer output", container.NewVBox(
		dataRow("Summary", summaryValue),
		dataRow("Content Types", contentValue),
		dataRow("Completeness", completenessValue),
		dataRow("Details", detailsValue),
		dataRow("Warnings", warningsValue),
	))

	statsPane := panel("Runtime Status", "Current execution stage", container.NewVBox(
		dataRow("Status", statusValue),
		dataRow("Progress", progressStateValue),
		dataRow("Output", resultValue),
	))

	centerBottomLeft := container.NewVScroll(container.NewVBox(conversionPane))
	centerBottomRight := container.NewVScroll(container.NewVBox(inspectPane, statsPane))
	centerBottomSplit := container.NewHSplit(centerBottomLeft, centerBottomRight)
	centerBottomSplit.Offset = 0.58

	centerPane := container.NewBorder(
		inputPane,
		nil,
		nil,
		nil,
		centerBottomSplit,
	)

	rightPane := container.NewVBox(
		panel("Source Package", "Input summary", container.NewVBox(
			dataRow("Formats", sourceSummaryValue),
			dataRow("Selected Path", fileValue),
			dataRow("Last Result", resultValue),
		)),
		panel("Notes", "Current app behavior", noteValue),
		panel("Activity", "Session log", logOutput),
	)

	leftScroll := container.NewVScroll(leftPane)
	centerScroll := container.NewVScroll(centerPane)
	rightScroll := container.NewVScroll(rightPane)

	leftCenterSplit := container.NewHSplit(leftScroll, centerScroll)
	leftCenterSplit.Offset = 0.22

	mainSplit := container.NewHSplit(leftCenterSplit, rightScroll)
	mainSplit.Offset = 0.78

	statusBar := container.NewGridWithColumns(3,
		widget.NewLabel("MCC Desktop Shell"),
		widget.NewLabelWithStyle(lastStatusText, fyne.TextAlignCenter, fyne.TextStyle{}),
		widget.NewLabelWithStyle("No active file", fyne.TextAlignTrailing, fyne.TextStyle{}),
	)

	statusTextLeft := statusBar.Objects[0].(*widget.Label)
	statusTextCenter := statusBar.Objects[1].(*widget.Label)
	statusTextRight := statusBar.Objects[2].(*widget.Label)

	updateStatusBar := func() {
		statusTextLeft.SetText("MCC Desktop Shell")
		statusTextCenter.SetText(lastStatusText)
		if currentSession != nil {
			statusTextRight.SetText(currentSession.OriginalFilename)
		} else {
			statusTextRight.SetText("No active file")
		}
	}

	_ = statusTextLeft

	oldSetBusy := setBusy
	setBusy = func(busy bool, stage, status string) {
		oldSetBusy(busy, stage, status)
		updateStatusBar()
	}

	oldUpdateSessionLabels := updateSessionLabels
	updateSessionLabels = func() {
		oldUpdateSessionLabels()
		updateStatusBar()
	}

	resetState()
	refreshWorkflow()
	updateStatusBar()

	window.SetContent(container.NewBorder(
		titleBar,
		statusBar,
		nil,
		nil,
		mainSplit,
	))
	window.ShowAndRun()
	return nil
}

func panel(title, subtitle string, content fyne.CanvasObject) fyne.CanvasObject {
	return widget.NewCard(title, subtitle, content)
}

func dataRow(label string, value fyne.CanvasObject) fyne.CanvasObject {
	left := widget.NewLabelWithStyle(label, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	left.Wrapping = fyne.TextWrapWord
	return container.NewBorder(nil, widget.NewSeparator(), left, nil, value)
}

func newValueLabel(text string) *widget.Label {
	label := widget.NewLabel(text)
	label.Wrapping = fyne.TextWrapWord
	return label
}

func newMonoValueLabel(text string) *widget.Label {
	label := widget.NewLabelWithStyle(text, fyne.TextAlignLeading, fyne.TextStyle{Monospace: true})
	label.Wrapping = fyne.TextWrapWord
	return label
}

func newPluginRow(plugin pluginDef, status *widget.Label, button *widget.Button) fyne.CanvasObject {
	badge := widget.NewLabelWithStyle(shortPluginName(plugin.ID), fyne.TextAlignCenter, fyne.TextStyle{Bold: true, Monospace: true})
	name := widget.NewLabelWithStyle(plugin.Name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	name.Wrapping = fyne.TextWrapWord
	status.Wrapping = fyne.TextWrapWord

	left := container.NewBorder(nil, nil, container.NewGridWrap(fyne.NewSize(40, 22), badge), nil, container.NewVBox(name, status))
	return container.NewBorder(nil, widget.NewSeparator(), nil, button, left)
}

func buildSummaryText(session *workflow.Session) string {
	if session == nil {
		return "Nothing analyzed yet"
	}
	fromText := joinOrFallback(session.AvailableSourceFormat, " / ", "Unknown")
	toText := joinOrFallback(session.AvailableTargets, " / ", "None")
	return fromText + " -> " + toText
}

func formatSourceSummary(session *workflow.Session, selectedSource, selectedTarget, namespace string) string {
	if session == nil {
		return "Not analyzed"
	}

	lines := []string{
		"Detected: " + joinOrFallback(session.Report.Formats, " / ", "Unknown"),
		"Source: " + valueOrDefault(selectedSource, "Not selected"),
		"Target: " + valueOrDefault(selectedTarget, "Not selected"),
		"Namespace: " + valueOrAuto(namespace),
	}
	return strings.Join(lines, "\n")
}

func formatDirection(source, target string) string {
	return valueOrDefault(source, "Not selected") + " -> " + valueOrDefault(target, "Not selected")
}

func formatCompleteness(report analyzer.Report) string {
	parts := make([]string, 0, 3)
	if report.Completeness.ItemsConfig {
		parts = append(parts, "Items config")
	}
	if report.Completeness.CategoriesConfig {
		parts = append(parts, "Categories config")
	}
	if report.Completeness.ResourceFiles {
		parts = append(parts, "Resource files")
	}
	if len(parts) == 0 {
		return "Missing required parts"
	}
	return strings.Join(parts, " / ")
}

func formatDetails(report analyzer.Report) string {
	return fmt.Sprintf(
		"Items: %d\nTextures: %d\nModels: %d",
		report.Details.ItemCount,
		report.Details.TextureCount,
		report.Details.ModelCount,
	)
}

func reportWarnings(session *workflow.Session, lastError string) string {
	if lastError != "" {
		return lastError
	}
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

func activeWorkflowForStage(stage string, hasReport bool) string {
	switch {
	case stage == "done":
		return "export"
	case stage == "ready" || stage == "converting" || stage == "error":
		return "convert"
	case hasReport:
		return "inspect"
	default:
		return "import"
	}
}

func pluginStateText(available, selected bool) string {
	switch {
	case !available:
		return "Unavailable"
	case selected:
		return "Selected"
	default:
		return "Available"
	}
}

func shortPluginName(id string) string {
	switch id {
	case "ItemsAdder":
		return "IA"
	case "Nexo":
		return "NX"
	case "Oraxen":
		return "OR"
	case "CraftEngine":
		return "CE"
	case "MythicCrucible":
		return "MC"
	default:
		if len(id) <= 2 {
			return strings.ToUpper(id)
		}
		return strings.ToUpper(id[:2])
	}
}

func valueOrDefault(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func valueOrAuto(value string) string {
	return valueOrDefault(value, "Auto")
}

func lastErrorTextOrDefault(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func firstOrEmpty(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func joinOrFallback(items []string, separator, fallback string) string {
	if len(items) == 0 {
		return fallback
	}
	return strings.Join(items, separator)
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
