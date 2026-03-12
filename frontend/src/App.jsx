import { useEffect, useMemo, useState } from "react";
import {
  Alert,
  Avatar,
  Box,
  Button,
  Chip,
  CircularProgress,
  Divider,
  LinearProgress,
  Stack,
  TextField,
  Typography
} from "@mui/material";
import ArrowForwardRoundedIcon from "@mui/icons-material/ArrowForwardRounded";
import CloudUploadRoundedIcon from "@mui/icons-material/CloudUploadRounded";
import FolderZipRoundedIcon from "@mui/icons-material/FolderZipRounded";
import SettingsEthernetRoundedIcon from "@mui/icons-material/SettingsEthernetRounded";
import TaskAltRoundedIcon from "@mui/icons-material/TaskAltRounded";
import TopicRoundedIcon from "@mui/icons-material/TopicRounded";

const defaultPlugins = [
  { id: "ItemsAdder", name: "ItemsAdder", icon: "/static/images/itemsadder.webp" },
  { id: "Nexo", name: "Nexo", icon: "/static/images/nexo.webp" },
  { id: "Oraxen", name: "Oraxen", icon: "/static/images/oraxen.webp" },
  { id: "CraftEngine", name: "CraftEngine", icon: "/static/images/craftengine.webp" },
  { id: "MythicCrucible", name: "MythicCrucible", icon: "/static/images/mythiccrucible.webp" }
];

const workflowItems = [
  { id: "import", label: "导入文件", helper: "ZIP" },
  { id: "inspect", label: "检查内容", helper: "Analyzer" },
  { id: "convert", label: "执行转换", helper: "Converter" },
  { id: "export", label: "导出结果", helper: "Output" }
];

const stageLabels = {
  idle: "等待上传",
  analyzing: "分析中",
  ready: "待确认",
  converting: "转换中",
  done: "已完成",
  error: "发生错误"
};

function Panel({ title, subtitle, children, grow = false }) {
  return (
    <Box
      sx={{
        border: "1px solid",
        borderColor: "divider",
        bgcolor: "background.paper",
        minHeight: 0,
        display: "flex",
        flexDirection: "column",
        ...(grow ? { flex: 1 } : null)
      }}
    >
      <Box
        sx={{
          px: 2,
          py: 1.25,
          borderBottom: "1px solid",
          borderColor: "divider",
          bgcolor: (theme) => theme.palette.panel.header
        }}
      >
        <Typography variant="subtitle2" sx={{ fontWeight: 700, mb: subtitle ? 0.25 : 0 }}>
          {title}
        </Typography>
        {subtitle ? (
          <Typography variant="body2" color="text.secondary">
            {subtitle}
          </Typography>
        ) : null}
      </Box>
      <Box sx={{ p: 2, minHeight: 0, flex: 1 }}>{children}</Box>
    </Box>
  );
}

function SidebarItem({ label, helper, active }) {
  return (
    <Box
      sx={{
        px: 1.25,
        py: 1,
        border: "1px solid",
        borderColor: active ? "primary.main" : "divider",
        bgcolor: active ? "action.selected" : "transparent"
      }}
    >
      <Typography variant="body2" fontWeight={600}>
        {label}
      </Typography>
      <Typography variant="caption" color="text.secondary">
        {helper}
      </Typography>
    </Box>
  );
}

function PluginListItem({ plugin, selected, disabled, onClick }) {
  return (
    <Button
      fullWidth
      variant="text"
      disabled={disabled}
      onClick={onClick}
      sx={{
        justifyContent: "flex-start",
        px: 1.25,
        py: 1,
        borderRadius: 0,
        border: "1px solid",
        borderColor: selected ? "primary.main" : "divider",
        bgcolor: selected ? "action.selected" : "background.paper",
        color: "text.primary",
        opacity: disabled ? 0.45 : 1
      }}
    >
      <Stack direction="row" spacing={1.25} alignItems="center" sx={{ width: "100%" }}>
        <Avatar src={plugin.icon} alt={plugin.name} variant="rounded" sx={{ width: 34, height: 34 }} />
        <Box sx={{ textAlign: "left", minWidth: 0 }}>
          <Typography variant="body2" fontWeight={600} noWrap>
            {plugin.name}
          </Typography>
          <Typography variant="caption" color="text.secondary">
            {disabled ? "不可用" : selected ? "已选择" : "可选"}
          </Typography>
        </Box>
      </Stack>
    </Button>
  );
}

function DataRow({ label, value, mono = false }) {
  return (
    <Box
      sx={{
        display: "grid",
        gridTemplateColumns: "108px minmax(0, 1fr)",
        gap: 1.25,
        py: 0.75,
        borderBottom: "1px solid",
        borderColor: "divider"
      }}
    >
      <Typography variant="caption" color="text.secondary">
        {label}
      </Typography>
      <Typography
        variant="body2"
        sx={mono ? { fontFamily: '"Cascadia Mono", "Consolas", monospace' } : null}
      >
        {value}
      </Typography>
    </Box>
  );
}

function MetricRow({ label, value }) {
  return (
    <Stack direction="row" justifyContent="space-between" alignItems="center">
      <Typography variant="body2" color="text.secondary">
        {label}
      </Typography>
      <Typography variant="body2" fontWeight={700}>
        {value}
      </Typography>
    </Stack>
  );
}

export default function App() {
  const [dragActive, setDragActive] = useState(false);
  const [stage, setStage] = useState("idle");
  const [progress, setProgress] = useState(0);
  const [statusText, setStatusText] = useState("等待上传文件");
  const [error, setError] = useState("");
  const [downloadUrl, setDownloadUrl] = useState("");
  const [report, setReport] = useState(null);
  const [sessionId, setSessionId] = useState("");
  const [selectedSource, setSelectedSource] = useState("");
  const [selectedTarget, setSelectedTarget] = useState("");
  const [namespace, setNamespace] = useState("");

  const supportedPlugins = report?.supported_plugins?.length ? report.supported_plugins : defaultPlugins;
  const sourceFormats = report?.source_formats ?? [];
  const availableTargets = report?.available_targets ?? [];
  const warnings = report?.warnings ?? [];
  const completeness = report?.completeness ?? {};
  const details = report?.details ?? {};

  useEffect(() => {
    const timer = window.setInterval(() => {
      fetch("/api/heartbeat", { method: "POST" }).catch(() => {});
    }, 2000);
    return () => window.clearInterval(timer);
  }, []);

  const summaryText = useMemo(() => {
    if (!report) {
      return "未加载文件";
    }
    const fromText = sourceFormats.length ? sourceFormats.join(" / ") : "未知";
    const toText = availableTargets.length ? availableTargets.join(" / ") : "无";
    return `${fromText} -> ${toText}`;
  }, [availableTargets, report, sourceFormats]);

  const activeNav = stage === "done" ? "export" : stage === "ready" || stage === "converting" ? "convert" : report ? "inspect" : "import";

  function resetState() {
    setDragActive(false);
    setStage("idle");
    setProgress(0);
    setStatusText("等待上传文件");
    setError("");
    setDownloadUrl("");
    setReport(null);
    setSessionId("");
    setSelectedSource("");
    setSelectedTarget("");
    setNamespace("");
  }

  async function shutdownApp() {
    if (!window.confirm("确定要关闭 MCC 吗？")) {
      return;
    }
    if (typeof window.quitApp === "function") {
      await window.quitApp();
      return;
    }
    await fetch("/api/shutdown", { method: "POST" }).catch(() => {});
  }

  function handleFile(file) {
    if (!file || !file.name.toLowerCase().endsWith(".zip")) {
      setError("请上传 `.zip` 格式的压缩包。");
      setStage("error");
      return;
    }

    setStage("analyzing");
    setProgress(6);
    setStatusText("正在上传并分析文件...");
    setError("");
    setDownloadUrl("");
    setReport(null);

    const formData = new FormData();
    formData.append("file", file);

    const xhr = new XMLHttpRequest();
    xhr.open("POST", "/api/analyze", true);

    xhr.upload.onprogress = (event) => {
      if (!event.lengthComputable) {
        return;
      }
      const percent = Math.max(8, Math.min(82, (event.loaded / event.total) * 80));
      setProgress(percent);
      setStatusText("正在上传并分析文件...");
    };

    xhr.onload = () => {
      if (xhr.status === 200) {
        const response = JSON.parse(xhr.responseText);
        setProgress(100);
        setStatusText("识别完成");
        setReport(response.report);
        setSessionId(response.session_id);
        setSelectedSource(response.report.source_formats?.[0] || "");
        setSelectedTarget(response.report.available_targets?.[0] || "");
        setStage("ready");
        return;
      }

      let message = "文件分析失败。";
      try {
        message = JSON.parse(xhr.responseText).error || message;
      } catch {}
      setError(message);
      setStage("error");
    };

    xhr.onerror = () => {
      setError("网络连接失败，无法上传文件。");
      setStage("error");
    };

    xhr.send(formData);
  }

  function startConversion() {
    if (!sessionId || !selectedTarget) {
      return;
    }

    setStage("converting");
    setProgress(0);
    setStatusText("正在生成转换结果...");
    setError("");

    const formData = new FormData();
    formData.append("session_id", sessionId);
    if (selectedSource) {
      formData.append("source_format", selectedSource);
    }
    formData.append("target_format", selectedTarget);
    if (namespace.trim()) {
      formData.append("namespace", namespace.trim());
    }

    const xhr = new XMLHttpRequest();
    xhr.open("POST", "/api/convert", true);

    let pseudoProgress = 0;
    const timer = window.setInterval(() => {
      if (xhr.readyState === 4) {
        window.clearInterval(timer);
        return;
      }
      pseudoProgress = Math.min(90, pseudoProgress + 4);
      setProgress(pseudoProgress);
      setStatusText("正在生成转换结果...");
    }, 180);

    xhr.onload = () => {
      window.clearInterval(timer);
      if (xhr.status === 200) {
        const response = JSON.parse(xhr.responseText);
        setProgress(100);
        setStatusText("转换完成");
        setDownloadUrl(response.download_url);
        setStage("done");
        return;
      }

      let message = "生成结果失败。";
      try {
        message = JSON.parse(xhr.responseText).error || message;
      } catch {}
      setError(message);
      setStage("error");
    };

    xhr.onerror = () => {
      window.clearInterval(timer);
      setError("网络连接失败，无法生成结果。");
      setStage("error");
    };

    xhr.send(formData);
  }

  return (
    <Box
      sx={{
        minHeight: "100vh",
        bgcolor: "background.default",
        color: "text.primary",
        p: 1.5
      }}
    >
      <Box
        sx={{
        minHeight: "calc(100vh - 12px)",
        border: "1px solid",
        borderColor: "divider",
        bgcolor: (theme) => theme.palette.window,
        boxShadow: "0 16px 40px rgba(0, 0, 0, 0.12)",
        display: "grid",
        gridTemplateRows: "40px minmax(0, 1fr) 28px"
        }}
      >
        <Box
          sx={{
            display: "grid",
            gridTemplateColumns: { xs: "1fr", md: "220px minmax(0, 1fr) auto" },
            gap: 1,
            alignItems: "center",
            px: 1.5,
            borderBottom: "1px solid",
            borderColor: "divider",
            bgcolor: (theme) => theme.palette.panel.header
          }}
        >
          <Stack direction="row" spacing={1} alignItems="center" sx={{ minWidth: 0 }}>
            <Avatar variant="rounded" sx={{ width: 18, height: 18, bgcolor: "primary.main", fontSize: 11 }}>
              M
            </Avatar>
            <Typography variant="body2" fontWeight={700} noWrap>
              Minecraft Config Converter
            </Typography>
          </Stack>
          <Typography variant="caption" color="text.secondary" sx={{ display: { xs: "none", md: "block" } }}>
            本地转换会话
          </Typography>
          <Stack direction="row" spacing={1} justifyContent="flex-end">
            <Button size="small" variant="outlined" onClick={resetState}>
              重置
            </Button>
            <Button size="small" variant="contained" color="inherit" onClick={shutdownApp}>
              退出
            </Button>
          </Stack>
        </Box>

        <Box
          sx={{
            minHeight: 0,
            display: "grid",
            gridTemplateColumns: { xs: "1fr", lg: "220px minmax(0, 1fr) 300px" }
          }}
        >
          <Box
            sx={{
              borderRight: { lg: "1px solid" },
              borderColor: "divider",
              bgcolor: (theme) => theme.palette.sidebar,
              p: 1.5,
              display: "flex",
              flexDirection: "column",
              gap: 1.5
            }}
          >
            <Panel title="工作流" subtitle="当前任务">
              <Stack spacing={1}>
                {workflowItems.map((item) => (
                  <SidebarItem
                    key={item.id}
                    label={item.label}
                    helper={item.helper}
                    active={activeNav === item.id}
                  />
                ))}
              </Stack>
            </Panel>

            <Panel title="会话" subtitle="状态概览" grow>
              <Stack spacing={1.25}>
                <DataRow label="状态" value={stageLabels[stage]} />
                <DataRow label="会话 ID" value={sessionId || "未创建"} mono />
                <DataRow label="转换方向" value={`${selectedSource || "未选择"} -> ${selectedTarget || "未选择"}`} />
                <DataRow label="命名空间" value={namespace || "自动"} mono />
              </Stack>
            </Panel>
          </Box>

          <Box sx={{ minHeight: 0, p: 1.5, display: "flex", flexDirection: "column", gap: 1.5 }}>
            <Panel title="输入" subtitle="导入并检查待处理资源包">
              <Stack spacing={2}>
                {error ? <Alert severity="error">{error}</Alert> : null}
                {warnings.map((warning) => (
                  <Alert key={warning} severity="warning">
                    {warning}
                  </Alert>
                ))}

                <Box
                  onDragOver={(event) => {
                    event.preventDefault();
                    setDragActive(true);
                  }}
                  onDragLeave={() => setDragActive(false)}
                  onDrop={(event) => {
                    event.preventDefault();
                    setDragActive(false);
                    handleFile(event.dataTransfer.files?.[0]);
                  }}
                  sx={{
                    border: "1px dashed",
                    borderColor: dragActive ? "primary.main" : "divider",
                    bgcolor: dragActive ? "action.selected" : "background.paper",
                    p: 2
                  }}
                >
                  <Stack direction={{ xs: "column", md: "row" }} spacing={2} alignItems={{ md: "center" }}>
                    <Avatar variant="rounded" sx={{ width: 42, height: 42, bgcolor: "action.selected", color: "primary.main" }}>
                      <FolderZipRoundedIcon />
                    </Avatar>
                    <Box sx={{ flex: 1, minWidth: 0 }}>
                      <Typography variant="body2" fontWeight={700}>
                        ZIP 输入包
                      </Typography>
                      <Typography variant="body2" color="text.secondary">
                        拖入文件，或点击按钮选择本地 `.zip` 压缩包。最大 500MB。
                      </Typography>
                    </Box>
                    <Button component="label" variant="contained" startIcon={<CloudUploadRoundedIcon />}>
                      选择文件
                      <input hidden type="file" accept=".zip" onChange={(event) => handleFile(event.target.files?.[0])} />
                    </Button>
                  </Stack>
                </Box>

                {(stage === "analyzing" || stage === "converting") ? (
                  <Box sx={{ border: "1px solid", borderColor: "divider", p: 1.5, bgcolor: "background.paper" }}>
                    <Stack spacing={1.25}>
                      <Stack direction="row" spacing={1} alignItems="center">
                        <CircularProgress size={16} />
                        <Typography variant="body2">{statusText}</Typography>
                      </Stack>
                      <LinearProgress variant="determinate" value={progress} />
                      <Typography variant="caption" color="text.secondary">
                        {Math.round(progress)}%
                      </Typography>
                    </Stack>
                  </Box>
                ) : null}
              </Stack>
            </Panel>

            <Box
              sx={{
                display: "grid",
                gap: 1.5,
                gridTemplateColumns: { xs: "1fr", xl: "minmax(0, 1fr) minmax(320px, 420px)" },
                minHeight: 0,
                flex: 1
              }}
            >
              <Panel title="转换配置" subtitle="来源、目标与输出参数" grow>
                <Stack spacing={2}>
                  <Box>
                    <Typography variant="subtitle2" sx={{ mb: 1, fontWeight: 700 }}>
                      来源格式
                    </Typography>
                    <Stack spacing={1}>
                      {supportedPlugins.map((plugin) => (
                        <PluginListItem
                          key={`source-${plugin.id}`}
                          plugin={plugin}
                          selected={selectedSource === plugin.id}
                          disabled={!sourceFormats.includes(plugin.id)}
                          onClick={() => setSelectedSource(plugin.id)}
                        />
                      ))}
                    </Stack>
                  </Box>

                  <Box sx={{ display: "flex", justifyContent: "center", py: 0.5 }}>
                    <Chip icon={<ArrowForwardRoundedIcon />} label="目标输出" size="small" variant="outlined" />
                  </Box>

                  <Box>
                    <Typography variant="subtitle2" sx={{ mb: 1, fontWeight: 700 }}>
                      目标格式
                    </Typography>
                    <Stack spacing={1}>
                      {supportedPlugins.map((plugin) => (
                        <PluginListItem
                          key={`target-${plugin.id}`}
                          plugin={plugin}
                          selected={selectedTarget === plugin.id}
                          disabled={!availableTargets.includes(plugin.id)}
                          onClick={() => setSelectedTarget(plugin.id)}
                        />
                      ))}
                    </Stack>
                  </Box>

                  <Divider />

                  <TextField
                    size="small"
                    label="命名空间"
                    value={namespace}
                    onChange={(event) => setNamespace(event.target.value)}
                    placeholder="可选，不填则自动推断"
                  />

                  <Stack direction={{ xs: "column", sm: "row" }} spacing={1}>
                    <Button variant="contained" onClick={startConversion} disabled={!sessionId || !selectedTarget}>
                      开始转换
                    </Button>
                    <Button variant="outlined" onClick={resetState}>
                      清空当前任务
                    </Button>
                    {stage === "done" ? (
                      <Button href={downloadUrl} variant="contained" color="success">
                        下载结果
                      </Button>
                    ) : null}
                  </Stack>
                </Stack>
              </Panel>

              <Stack spacing={1.5}>
                <Panel title="检查结果" subtitle="分析器输出">
                  <Stack spacing={0}>
                    <DataRow label="文件名" value={report?.filename || "未载入"} mono />
                    <DataRow label="识别结果" value={summaryText} />
                    <DataRow label="内容类型" value={(report?.content_types || []).join(" / ") || "无"} />
                    <DataRow
                      label="完整性"
                      value={
                        [
                          completeness.items_config ? "物品配置" : null,
                          completeness.categories_config ? "分类配置" : null,
                          completeness.resource_files ? "资源文件" : null
                        ].filter(Boolean).join(" / ") || "存在缺失项"
                      }
                    />
                  </Stack>
                </Panel>

                <Panel title="统计" subtitle="包内容数量">
                  <Stack spacing={1.25}>
                    <MetricRow label="物品" value={details.item_count ?? 0} />
                    <MetricRow label="贴图" value={details.texture_count ?? 0} />
                    <MetricRow label="模型" value={details.model_count ?? 0} />
                  </Stack>
                </Panel>

                <Panel title="运行状态" subtitle="当前执行阶段" grow>
                  <Stack spacing={1.25}>
                    <Stack direction="row" spacing={1} alignItems="center">
                      <Avatar variant="rounded" sx={{ width: 28, height: 28, bgcolor: "action.selected", color: "primary.main" }}>
                        {stage === "done" ? <TaskAltRoundedIcon fontSize="small" /> : stage === "ready" ? <SettingsEthernetRoundedIcon fontSize="small" /> : <TopicRoundedIcon fontSize="small" />}
                      </Avatar>
                      <Box>
                        <Typography variant="body2" fontWeight={700}>
                          {stageLabels[stage]}
                        </Typography>
                        <Typography variant="caption" color="text.secondary">
                          {statusText}
                        </Typography>
                      </Box>
                    </Stack>
                    {stage === "done" ? (
                      <Alert severity="success">输出文件已经准备完成，可以直接下载。</Alert>
                    ) : (
                      <Typography variant="body2" color="text.secondary">
                        等待用户确认或服务端处理完成。
                      </Typography>
                    )}
                    <Box sx={{ pt: 0.5 }}>
                      <MetricRow label="支持插件" value={supportedPlugins.length} />
                    </Box>
                  </Stack>
                </Panel>
              </Stack>
            </Box>
          </Box>

          <Box
            sx={{
              display: { xs: "none", lg: "flex" },
              flexDirection: "column",
              gap: 1.5,
              p: 1.5,
              borderLeft: "1px solid",
              borderColor: "divider",
              bgcolor: (theme) => theme.palette.sidebar
            }}
          >
            <Panel title="来源包" subtitle="输入摘要">
              <Stack spacing={1.25}>
                <MetricRow label="来源格式" value={sourceFormats.join(" / ") || "未识别"} />
                <MetricRow label="可转目标" value={availableTargets.join(" / ") || "无"} />
                <MetricRow label="命名空间" value={namespace || "自动"} />
              </Stack>
            </Panel>

            <Panel title="说明" subtitle="当前应用行为" grow>
              <Typography variant="body2" color="text.secondary" sx={{ lineHeight: 1.65 }}>
                该界面按桌面工具风格组织，不展示网页宣传区块。左侧是流程导航，中间是主工作区，右侧是当前任务的检查信息与输出摘要。
              </Typography>
            </Panel>
          </Box>
        </Box>

        <Box
          sx={{
            px: 1.5,
            display: "grid",
            gridTemplateColumns: { xs: "1fr", md: "1fr auto auto" },
            gap: 1.5,
            alignItems: "center",
            borderTop: "1px solid",
            borderColor: "divider",
            bgcolor: (theme) => theme.palette.panel.header
          }}
        >
          <Typography variant="caption" color="text.secondary" noWrap>
            MCC Desktop Shell
          </Typography>
          <Typography variant="caption" color="text.secondary" noWrap>
            {statusText}
          </Typography>
          <Typography variant="caption" color="text.secondary" noWrap>
            {report?.filename || "无活动文件"}
          </Typography>
        </Box>
      </Box>
    </Box>
  );
}
