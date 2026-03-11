document.addEventListener("DOMContentLoaded", () => {
    const dropZone = document.getElementById("drop-zone");
    const previewStartBtn = document.getElementById("preview-start-btn");
    const progressSection = document.getElementById("progress-section");
    const progressFill = document.getElementById("progress-fill");
    const statusText = document.getElementById("status-text");
    const resultSection = document.getElementById("result-section");
    const errorSection = document.getElementById("error-section");
    const dynamicSlot = document.getElementById("dynamic-slot");

    const mockReport = {
        filename: "demo-pack.zip",
        source_formats: ["ItemsAdder"],
        available_targets: ["CraftEngine"],
        content_types: ["物品", "贴图", "模型"],
        completeness: {
            items_config: true,
            categories_config: true,
            resource_files: true
        },
        details: {
            item_count: 128,
            texture_count: 96,
            model_count: 42
        },
        warnings: ["这是静态预览页面，数据仅用于展示界面效果。"],
        supported_plugins: [
            { id: "ItemsAdder", name: "ItemsAdder", icon: "./static/images/itemsadder.webp" },
            { id: "Nexo", name: "Nexo", icon: "./static/images/nexo.webp" },
            { id: "Oraxen", name: "Oraxen", icon: "./static/images/oraxen.webp" },
            { id: "CraftEngine", name: "CraftEngine", icon: "./static/images/craftengine.webp" },
            { id: "MythicCrucible", name: "MythicCrucible", icon: "./static/images/mythiccrucible.webp" }
        ]
    };

    previewStartBtn.addEventListener("click", startPreview);
    dropZone.addEventListener("click", startPreview);
    dropZone.addEventListener("keydown", (event) => {
        if (event.key === "Enter" || event.key === " ") {
            event.preventDefault();
            startPreview();
        }
    });

    function startPreview() {
        resetTransientUI();
        dropZone.style.display = "none";
        progressSection.style.display = "block";
        runProgress([
            { percent: 18, text: "正在准备演示内容…" },
            { percent: 52, text: "正在展示识别结果…" },
            { percent: 100, text: "已准备完成" }
        ], () => showAnalysisReport(mockReport));
    }

    function showAnalysisReport(report) {
        progressSection.style.display = "none";
        dynamicSlot.innerHTML = "";

        let selectedSource = report.source_formats[0] || "";
        let selectedTarget = report.available_targets[0] || "";

        const warningHtml = (report.warnings || []).length > 0
            ? `
                <div class="warning-box">
                    ${report.warnings.map((warning) => `<p>${warning}</p>`).join("")}
                </div>
            `
            : "";

        const sourceGrid = renderPluginGrid({
            plugins: report.supported_plugins,
            selected: selectedSource,
            selectable: (plugin) => report.source_formats.includes(plugin.id)
        });

        const targetGrid = renderPluginGrid({
            plugins: report.supported_plugins,
            selected: selectedTarget,
            selectable: (plugin) => report.available_targets.includes(plugin.id)
        });

        dynamicSlot.insertAdjacentHTML("beforeend", `
            <section class="report-card" id="report-section">
                <div class="report-header">
                    <div>
                        <span class="section-chip">已识别</span>
                        <h3>确认一下，然后开始处理。</h3>
                    </div>
                    <p class="analysis-summary">
                        已识别为 ${escapeHtml(report.source_formats.join(" / ") || "未知类型")}，
                        可转换为 ${escapeHtml(report.available_targets.join(" / ") || "暂不支持")}。
                    </p>
                </div>
                ${warningHtml}
                <div class="plugin-selection">
                    <div class="plugin-column">
                        <h4>当前类型</h4>
                        <div class="plugin-grid" id="source-plugins-grid">${sourceGrid}</div>
                    </div>
                    <div class="selection-arrow">→</div>
                    <div class="plugin-column">
                        <h4>转换为</h4>
                        <div class="plugin-grid" id="target-plugins-grid">${targetGrid}</div>
                    </div>
                </div>

                <div class="report-grid">
                    <div class="report-item report-item-wide">
                        <span class="label">当前文件</span>
                        <span class="filename-badge">${escapeHtml(report.filename || "未知文件")}</span>
                    </div>
                    <div class="report-item">
                        <span class="label">资源名称</span>
                        <input
                            type="text"
                            id="namespace-input"
                            class="text-input"
                            placeholder="示例：adventure_pack"
                            value="demo_pack"
                        >
                    </div>
                    <div class="report-item">
                        <span class="label">包含内容</span>
                        <div class="value">${escapeHtml((report.content_types || []).join(" · ") || "暂未识别")}</div>
                    </div>
                    <div class="report-item">
                        <span class="label">文件检查</span>
                        <ul class="check-list">
                            <li class="${report.completeness.items_config ? "ok" : "fail"}">物品配置</li>
                            <li class="${report.completeness.categories_config ? "ok" : "fail"}">分类配置</li>
                            <li class="${report.completeness.resource_files ? "ok" : "fail"}">资源文件</li>
                        </ul>
                    </div>
                    <div class="report-item">
                        <span class="label">统计</span>
                        <ul class="stats-list">
                            <li>物品 ${report.details.item_count}</li>
                            <li>贴图 ${report.details.texture_count}</li>
                            <li>模型 ${report.details.model_count}</li>
                        </ul>
                    </div>
                </div>

                <div class="analysis-actions">
                    <button id="start-convert-btn" class="btn-primary" type="button">继续演示</button>
                    <button class="btn-secondary" type="button" onclick="location.reload()">重新开始</button>
                </div>
            </section>
        `);

        document.getElementById("start-convert-btn").addEventListener("click", startMockConversion);
    }

    function startMockConversion() {
        const reportSection = document.getElementById("report-section");
        if (reportSection) {
            reportSection.style.display = "none";
        }

        progressSection.style.display = "block";
        runProgress([
            { percent: 12, text: "正在整理内容…" },
            { percent: 46, text: "正在生成演示结果…" },
            { percent: 82, text: "正在准备完成界面…" },
            { percent: 100, text: "处理完成" }
        ], showResult);
    }

    function showResult() {
        progressSection.style.display = "none";
        dynamicSlot.innerHTML = "";
        resultSection.style.display = "block";
    }

    function runProgress(steps, done) {
        let index = 0;
        const tick = () => {
            const step = steps[index];
            updateProgress(step.percent, step.text);
            index += 1;
            if (index >= steps.length) {
                window.setTimeout(done, 240);
                return;
            }
            window.setTimeout(tick, 420);
        };
        tick();
    }

    function renderPluginGrid({ plugins, selected, selectable }) {
        return plugins.map((plugin) => {
            const canSelect = selectable(plugin);
            const classes = [
                "plugin-card",
                canSelect ? "selectable" : "",
                plugin.id === selected ? "selected" : ""
            ].filter(Boolean).join(" ");

            return `
                <div class="${classes}" data-id="${plugin.id}">
                    <div class="plugin-icon">
                        <img src="${plugin.icon}" alt="${plugin.name}">
                    </div>
                    <div class="plugin-name">${plugin.name}</div>
                </div>
            `;
        }).join("");
    }

    function updateProgress(percent, text) {
        progressFill.style.width = `${percent}%`;
        statusText.textContent = text;
    }

    function resetTransientUI() {
        errorSection.style.display = "none";
        resultSection.style.display = "none";
        dynamicSlot.innerHTML = "";
    }

    function escapeHtml(value) {
        return String(value)
            .replaceAll("&", "&amp;")
            .replaceAll("<", "&lt;")
            .replaceAll(">", "&gt;")
            .replaceAll("\"", "&quot;")
            .replaceAll("'", "&#39;");
    }
});
