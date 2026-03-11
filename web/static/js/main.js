document.addEventListener("DOMContentLoaded", () => {
    const dropZone = document.getElementById("drop-zone");
    const fileInput = document.getElementById("file-input");
    const progressSection = document.getElementById("progress-section");
    const progressFill = document.getElementById("progress-fill");
    const statusText = document.getElementById("status-text");
    const resultSection = document.getElementById("result-section");
    const downloadLink = document.getElementById("download-link");
    const errorSection = document.getElementById("error-section");
    const errorMessage = document.getElementById("error-message");
    const dynamicSlot = document.getElementById("dynamic-slot");

    dropZone.addEventListener("dragover", (event) => {
        event.preventDefault();
        dropZone.classList.add("dragover");
    });

    dropZone.addEventListener("dragleave", () => {
        dropZone.classList.remove("dragover");
    });

    dropZone.addEventListener("drop", (event) => {
        event.preventDefault();
        dropZone.classList.remove("dragover");
        if (event.dataTransfer.files.length > 0) {
            handleFile(event.dataTransfer.files[0]);
        }
    });

    dropZone.addEventListener("keydown", (event) => {
        if (event.key === "Enter" || event.key === " ") {
            event.preventDefault();
            fileInput.click();
        }
    });

    fileInput.addEventListener("change", (event) => {
        if (event.target.files.length > 0) {
            handleFile(event.target.files[0]);
        }
    });

    function handleFile(file) {
        if (!file.name.toLowerCase().endsWith(".zip")) {
            showError("请上传 .zip 格式的压缩包。");
            return;
        }

        resetTransientUI();
        dropZone.style.display = "none";
        progressSection.style.display = "block";
        updateProgress(6, "正在上传并分析…");
        uploadFile(file);
    }

    function uploadFile(file) {
        const formData = new FormData();
        formData.append("file", file);

        const xhr = new XMLHttpRequest();
        xhr.open("POST", "/api/analyze", true);

        xhr.upload.onprogress = (event) => {
            if (!event.lengthComputable) {
                return;
            }
            const percent = Math.max(8, Math.min(82, (event.loaded / event.total) * 80));
            updateProgress(percent, "正在上传并分析…");
        };

        xhr.onload = () => {
            if (xhr.status === 200) {
                const response = JSON.parse(xhr.responseText);
                updateProgress(100, "分析完成");
                showAnalysisReport(response.report, response.session_id);
                return;
            }

            let message = "分析失败。";
            try {
                const response = JSON.parse(xhr.responseText);
                message = response.error || message;
            } catch (error) {
                console.error(error);
            }
            showError(message);
        };

        xhr.onerror = () => {
            showError("网络连接失败，无法上传文件。");
        };

        xhr.send(formData);
    }

    function startConversion(sessionId) {
        const formData = new FormData();
        formData.append("session_id", sessionId);

        const sourceInput = document.getElementById("selected-source");
        const targetInput = document.getElementById("selected-target");
        const namespaceInput = document.getElementById("namespace-input");

        if (sourceInput && sourceInput.value) {
            formData.append("source_format", sourceInput.value);
        }

        if (targetInput && targetInput.value) {
            formData.append("target_format", targetInput.value);
        }

        if (namespaceInput && namespaceInput.value.trim()) {
            formData.append("namespace", namespaceInput.value.trim());
        }

        progressSection.style.display = "block";
        updateProgress(0, "正在执行转换…");

        const reportSection = document.getElementById("report-section");
        if (reportSection) {
            reportSection.style.display = "none";
        }

        const xhr = new XMLHttpRequest();
        xhr.open("POST", "/api/convert", true);

        xhr.onload = () => {
            if (xhr.status === 200) {
                const response = JSON.parse(xhr.responseText);
                updateProgress(100, "转换完成");
                showResult(response.download_url);
                return;
            }

            let message = "转换失败。";
            try {
                const response = JSON.parse(xhr.responseText);
                message = response.error || message;
            } catch (error) {
                console.error(error);
            }
            showError(message);
        };

        let pseudoProgress = 0;
        const timer = setInterval(() => {
            if (xhr.readyState === 4) {
                clearInterval(timer);
                return;
            }
            if (pseudoProgress < 90) {
                pseudoProgress += 4;
                updateProgress(pseudoProgress, "正在执行转换…");
            }
        }, 180);

        xhr.send(formData);
    }

    function showAnalysisReport(report, sessionId) {
        progressSection.style.display = "none";
        dynamicSlot.innerHTML = "";

        const supportedPlugins = report.supported_plugins || [
            { id: "ItemsAdder", name: "ItemsAdder", icon: "/static/images/itemsadder.webp" },
            { id: "Nexo", name: "Nexo", icon: "/static/images/nexo.webp" },
            { id: "Oraxen", name: "Oraxen", icon: "/static/images/oraxen.webp" },
            { id: "CraftEngine", name: "CraftEngine", icon: "/static/images/craftengine.webp" },
            { id: "MythicCrucible", name: "MythicCrucible", icon: "/static/images/mythiccrucible.webp" }
        ];

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
            plugins: supportedPlugins,
            selected: selectedSource,
            selectable: (plugin) => report.source_formats.includes(plugin.id)
        });

        const targetGrid = renderPluginGrid({
            plugins: supportedPlugins,
            selected: selectedTarget,
            selectable: (plugin) => report.available_targets.includes(plugin.id)
        });

        dynamicSlot.insertAdjacentHTML("beforeend", `
            <section class="report-card" id="report-section">
                <div class="report-header">
                    <div>
                        <span class="section-chip">分析结果</span>
                        <h3>先确认结构，再决定怎么转。</h3>
                    </div>
                    <p class="analysis-summary">
                        检测到 ${escapeHtml(report.source_formats.join(" / ") || "未知格式")}，
                        可用目标为 ${escapeHtml(report.available_targets.join(" / ") || "暂无")}。
                    </p>
                </div>
                ${warningHtml}
                <div class="plugin-selection">
                    <div class="plugin-column">
                        <h4>源插件</h4>
                        <div class="plugin-grid" id="source-plugins-grid">${sourceGrid}</div>
                    </div>
                    <div class="selection-arrow">→</div>
                    <div class="plugin-column">
                        <h4>目标插件</h4>
                        <div class="plugin-grid" id="target-plugins-grid">${targetGrid}</div>
                    </div>
                </div>

                <input type="hidden" id="selected-source" value="${selectedSource}">
                <input type="hidden" id="selected-target" value="${selectedTarget}">

                <div class="report-grid">
                    <div class="report-item report-item-wide">
                        <span class="label">当前文件</span>
                        <span class="filename-badge">${escapeHtml(report.filename || "未知文件")}</span>
                    </div>
                    <div class="report-item">
                        <span class="label">命名空间</span>
                        <input
                            type="text"
                            id="namespace-input"
                            class="text-input"
                            placeholder="留空则使用默认值"
                            title="仅允许小写字母、数字、下划线、连字符和点"
                        >
                    </div>
                    <div class="report-item">
                        <span class="label">包含内容</span>
                        <div class="value">${escapeHtml((report.content_types || []).join(" · ") || "未识别")}</div>
                    </div>
                    <div class="report-item">
                        <span class="label">完整性检查</span>
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
                    <button id="start-convert-btn" class="btn-primary" ${selectedTarget ? "" : "disabled"}>开始转换</button>
                    <button class="btn-secondary" type="button" onclick="location.reload()">重新选择文件</button>
                </div>
            </section>
        `);

        const sourceGridElement = document.getElementById("source-plugins-grid");
        const targetGridElement = document.getElementById("target-plugins-grid");
        const startButton = document.getElementById("start-convert-btn");

        sourceGridElement.addEventListener("click", (event) => {
            const card = event.target.closest(".plugin-card.selectable");
            if (!card) {
                return;
            }
            selectedSource = card.dataset.id;
            document.getElementById("selected-source").value = selectedSource;
            sourceGridElement.querySelectorAll(".plugin-card").forEach((node) => node.classList.remove("selected"));
            card.classList.add("selected");
        });

        targetGridElement.addEventListener("click", (event) => {
            const card = event.target.closest(".plugin-card.selectable");
            if (!card) {
                return;
            }
            selectedTarget = card.dataset.id;
            document.getElementById("selected-target").value = selectedTarget;
            targetGridElement.querySelectorAll(".plugin-card").forEach((node) => node.classList.remove("selected"));
            card.classList.add("selected");
            startButton.disabled = false;
        });

        startButton.addEventListener("click", () => startConversion(sessionId));
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

    function showResult(url) {
        progressSection.style.display = "none";
        dynamicSlot.innerHTML = "";
        resultSection.style.display = "block";
        downloadLink.href = url;
    }

    function showError(message) {
        progressSection.style.display = "none";
        dynamicSlot.innerHTML = "";
        resultSection.style.display = "none";
        dropZone.style.display = "none";
        errorSection.style.display = "block";
        errorMessage.textContent = message;
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

    setInterval(() => {
        fetch("/api/heartbeat", { method: "POST" }).catch(() => {
            console.log("Heartbeat failed.");
        });
    }, 2000);
});
