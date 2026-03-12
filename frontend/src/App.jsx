import { useEffect, useMemo, useState } from "react";
import {
  Alert,
  Avatar,
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  CircularProgress,
  Divider,
  LinearProgress,
  Stack,
  TextField,
  Typography
} from "@mui/material";
import { alpha } from "@mui/material/styles";
import ArrowForwardRoundedIcon from "@mui/icons-material/ArrowForwardRounded";
import AutoAwesomeRoundedIcon from "@mui/icons-material/AutoAwesomeRounded";
import CloudUploadRoundedIcon from "@mui/icons-material/CloudUploadRounded";
import FolderZipRoundedIcon from "@mui/icons-material/FolderZipRounded";
import Inventory2RoundedIcon from "@mui/icons-material/Inventory2Rounded";
import TaskAltRoundedIcon from "@mui/icons-material/TaskAltRounded";

const defaultPlugins = [
  { id: "ItemsAdder", name: "ItemsAdder", icon: "/static/images/itemsadder.webp" },
  { id: "Nexo", name: "Nexo", icon: "/static/images/nexo.webp" },
  { id: "Oraxen", name: "Oraxen", icon: "/static/images/oraxen.webp" },
  { id: "CraftEngine", name: "CraftEngine", icon: "/static/images/craftengine.webp" },
  { id: "MythicCrucible", name: "MythicCrucible", icon: "/static/images/mythiccrucible.webp" }
];

const journeySteps = [
  {
    title: "上传压缩包",
    description: "拖拽或选择一个 `.zip` 文件，MCC 会先分析内容。",
    icon: <CloudUploadRoundedIcon fontSize="small" />
  },
  {
    title: "确认转换方向",
    description: "识别来源插件后，选择要输出的目标格式和命名空间。",
    icon: <AutoAwesomeRoundedIcon fontSize="small" />
  },
  {
    title: "导出结果",
    description: "转换完成后直接下载生成包，无需额外步骤。",
    icon: <TaskAltRoundedIcon fontSize="small" />
  }
];

const stageLabels = {
  idle: "等待上传",
  analyzing: "分析中",
  ready: "待确认",
  converting: "转换中",
  done: "已完成",
  error: "发生错误"
};

function PluginOption({ plugin, selected, disabled, onClick }) {
  return (
    <Button
      fullWidth
      variant={selected ? "contained" : "outlined"}
      disabled={disabled}
      onClick={onClick}
      sx={{
        justifyContent: "flex-start",
        p: 1.1,
        borderRadius: 3,
        textTransform: "none",
        minHeight: 72,
        opacity: disabled ? 0.4 : 1
      }}
    >
      <Stack direction="row" spacing={1.25} alignItems="center" sx={{ width: "100%" }}>
        <Avatar
          src={plugin.icon}
          alt={plugin.name}
          variant="rounded"
          sx={{ width: 42, height: 42, bgcolor: "transparent" }}
        />
        <Stack spacing={0.25} alignItems="flex-start">
          <Typography variant="body2" fontWeight={700}>
            {plugin.name}
          </Typography>
          <Typography variant="caption" color={selected ? "inherit" : "text.secondary"}>
            {disabled ? "当前不可选" : selected ? "已选择" : "点击选择"}
          </Typography>
        </Stack>
      </Stack>
    </Button>
  );
}

function InfoTile({ label, value, helper }) {
  return (
    <Card variant="outlined" sx={{ height: "100%" }}>
      <CardContent sx={{ p: 2.25 }}>
        <Typography variant="subtitle2" color="text.secondary" sx={{ mb: 1 }}>
          {label}
        </Typography>
        <Typography variant="body1" fontWeight={700}>
          {value}
        </Typography>
        {helper ? (
          <Typography variant="body2" color="text.secondary" sx={{ mt: 0.75 }}>
            {helper}
          </Typography>
        ) : null}
      </CardContent>
    </Card>
  );
}

function StepCard({ title, description, icon }) {
  return (
    <Card
      variant="outlined"
      sx={{
        background: (theme) =>
          `linear-gradient(180deg, ${alpha(theme.palette.primary.main, 0.09)} 0%, ${alpha(
            theme.palette.background.paper,
            0.95
          )} 100%)`
      }}
    >
      <CardContent sx={{ p: 2.25 }}>
        <Stack direction="row" spacing={1.5} alignItems="flex-start">
          <Avatar
            sx={{
              width: 38,
              height: 38,
              bgcolor: (theme) => alpha(theme.palette.primary.main, 0.12),
              color: "primary.main"
            }}
          >
            {icon}
          </Avatar>
          <Box>
            <Typography variant="body2" fontWeight={700} sx={{ mb: 0.5 }}>
              {title}
            </Typography>
            <Typography variant="body2" color="text.secondary">
              {description}
            </Typography>
          </Box>
        </Stack>
      </CardContent>
    </Card>
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
  const showUploader = stage === "idle" || stage === "error";

  useEffect(() => {
    const timer = window.setInterval(() => {
      fetch("/api/heartbeat", { method: "POST" }).catch(() => {});
    }, 2000);
    return () => window.clearInterval(timer);
  }, []);

  const summaryText = useMemo(() => {
    if (!report) {
      return "上传资源包后，这里会显示识别出的格式、内容类型与可用目标。";
    }

    const fromText = sourceFormats.length ? sourceFormats.join(" / ") : "未知来源";
    const toText = availableTargets.length ? availableTargets.join(" / ") : "暂无可用目标";
    return `已识别为 ${fromText}，当前可导出为 ${toText}。`;
  }, [availableTargets, report, sourceFormats]);

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
    setStatusText("正在上传并分析文件…");
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
      setStatusText("正在上传并分析文件…");
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
    setStatusText("正在生成转换结果…");
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
      setStatusText("正在生成转换结果…");
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
        background:
          "radial-gradient(circle at top left, rgba(14,116,144,0.18), transparent 28%), linear-gradient(180deg, #f7f3eb 0%, #eef4f8 48%, #f6f8fb 100%)",
        p: { xs: 2, md: 3 }
      }}
    >
      <Stack spacing={2.5}>
        <Card
          sx={{
            overflow: "hidden",
            color: "#f8fafc",
            background: "linear-gradient(135deg, #0f172a 0%, #102a43 52%, #0e7490 100%)"
          }}
        >
          <CardContent sx={{ p: { xs: 3, md: 4 } }}>
            <Box
              sx={{
                display: "grid",
                gap: 2,
                alignItems: "end",
                gridTemplateColumns: { xs: "1fr", lg: "minmax(0, 1fr) auto" }
              }}
            >
              <Stack spacing={1.5}>
                <Stack direction="row" spacing={1} alignItems="center" flexWrap="wrap" useFlexGap>
                  <Chip
                    label="WebView2"
                    size="small"
                    sx={{ bgcolor: "rgba(255,255,255,0.12)", color: "#e0f2fe" }}
                  />
                  <Chip
                    label="Local Converter"
                    size="small"
                    sx={{ bgcolor: "rgba(255,255,255,0.1)", color: "#e2e8f0" }}
                  />
                  <Chip
                    label={stageLabels[stage]}
                    size="small"
                    sx={{ bgcolor: "rgba(255,255,255,0.16)", color: "#f8fafc" }}
                  />
                </Stack>
                <Box>
                  <Typography variant="h3" sx={{ mb: 1 }}>
                    Minecraft Config Converter
                  </Typography>
                  <Typography sx={{ color: "rgba(226,232,240,0.88)", maxWidth: 720 }}>
                    在本地分析资源包、确认来源插件、转换为 CraftEngine 结构，并直接在桌面壳内下载结果。
                  </Typography>
                </Box>
              </Stack>

              <Stack
                direction={{ xs: "column", sm: "row" }}
                spacing={1.25}
                justifyContent={{ lg: "flex-end" }}
              >
                <Button
                  variant="outlined"
                  onClick={resetState}
                  sx={{
                    color: "#f8fafc",
                    borderColor: "rgba(226,232,240,0.35)",
                    "&:hover": { borderColor: "rgba(226,232,240,0.6)" }
                  }}
                >
                  重置流程
                </Button>
                <Button
                  variant="contained"
                  onClick={shutdownApp}
                  sx={{
                    bgcolor: "#f8fafc",
                    color: "#0f172a",
                    "&:hover": { bgcolor: "#e2e8f0" }
                  }}
                >
                  退出 MCC
                </Button>
              </Stack>
            </Box>
          </CardContent>
        </Card>

        <Box
          sx={{
            display: "grid",
            gap: 2,
            gridTemplateColumns: { xs: "1fr", xl: "minmax(0, 1.55fr) 360px" }
          }}
        >
          <Card sx={{ minWidth: 0 }}>
            <CardContent sx={{ p: { xs: 2.5, md: 3 } }}>
              <Stack spacing={3}>
                <Box>
                  <Typography variant="subtitle2" color="text.secondary" sx={{ mb: 0.75 }}>
                    Workflow
                  </Typography>
                  <Typography variant="h4" sx={{ mb: 0.75 }}>
                    资源转换面板
                  </Typography>
                  <Typography color="text.secondary">{summaryText}</Typography>
                </Box>

                {error ? (
                  <Alert
                    severity="error"
                    action={
                      <Button color="inherit" size="small" onClick={resetState}>
                        重新开始
                      </Button>
                    }
                  >
                    {error}
                  </Alert>
                ) : null}

                {showUploader ? (
                  <Card
                    variant="outlined"
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
                      borderRadius: 4,
                      borderStyle: "dashed",
                      borderWidth: 2,
                      borderColor: dragActive ? "primary.main" : "divider",
                      background: (theme) =>
                        dragActive
                          ? alpha(theme.palette.primary.main, 0.08)
                          : "linear-gradient(180deg, rgba(255,255,255,0.8) 0%, rgba(247,243,235,0.75) 100%)"
                    }}
                  >
                    <CardContent sx={{ p: { xs: 3, md: 4 } }}>
                      <Stack spacing={2.5} alignItems="flex-start">
                        <Avatar
                          variant="rounded"
                          sx={{
                            width: 56,
                            height: 56,
                            bgcolor: (theme) => alpha(theme.palette.primary.main, 0.12),
                            color: "primary.main"
                          }}
                        >
                          <FolderZipRoundedIcon />
                        </Avatar>
                        <Box>
                          <Typography variant="h4" sx={{ mb: 1 }}>
                            导入待处理压缩包
                          </Typography>
                          <Typography color="text.secondary">
                            支持拖拽上传，或从本地选择一个 `.zip` 文件。当前上限为 500MB。
                          </Typography>
                        </Box>
                        <Stack direction={{ xs: "column", sm: "row" }} spacing={1.25}>
                          <Button
                            component="label"
                            variant="contained"
                            startIcon={<CloudUploadRoundedIcon />}
                          >
                            选择文件
                            <input
                              hidden
                              type="file"
                              accept=".zip"
                              onChange={(event) => handleFile(event.target.files?.[0])}
                            />
                          </Button>
                          <Chip label="仅接受 .zip" color="primary" variant="outlined" />
                        </Stack>
                      </Stack>
                    </CardContent>
                  </Card>
                ) : null}

                {(stage === "analyzing" || stage === "converting") && (
                  <Card variant="outlined" sx={{ borderRadius: 4 }}>
                    <CardContent sx={{ p: 3 }}>
                      <Stack spacing={2}>
                        <Stack direction="row" spacing={1.25} alignItems="center">
                          <CircularProgress size={20} />
                          <Typography variant="body1" fontWeight={700}>
                            {statusText}
                          </Typography>
                        </Stack>
                        <LinearProgress variant="determinate" value={progress} />
                        <Typography variant="body2" color="text.secondary">
                          {Math.round(progress)}%
                        </Typography>
                      </Stack>
                    </CardContent>
                  </Card>
                )}

                {stage === "ready" && report ? (
                  <Stack spacing={3}>
                    <Box
                      sx={{
                        display: "grid",
                        gap: 2,
                        alignItems: "start",
                        gridTemplateColumns: { xs: "1fr", md: "minmax(0, 1fr) auto" }
                      }}
                    >
                      <Box>
                        <Chip label="已识别" color="primary" variant="outlined" sx={{ mb: 1.25 }} />
                        <Typography variant="h4" sx={{ mb: 0.75 }}>
                          确认来源与目标
                        </Typography>
                        <Typography color="text.secondary">
                          识别结果已经准备好。选择转换方向后即可开始处理。
                        </Typography>
                      </Box>
                      <TextField
                        size="small"
                        label="命名空间"
                        value={namespace}
                        onChange={(event) => setNamespace(event.target.value)}
                        placeholder="可选，不填则自动推断"
                        sx={{ minWidth: { md: 260 } }}
                      />
                    </Box>

                    {warnings.map((warning) => (
                      <Alert key={warning} severity="warning">
                        {warning}
                      </Alert>
                    ))}

                    <Box
                      sx={{
                        display: "grid",
                        gap: 2,
                        alignItems: "center",
                        gridTemplateColumns: { xs: "1fr", lg: "minmax(0, 1fr) 72px minmax(0, 1fr)" }
                      }}
                    >
                      <Stack spacing={1.25}>
                        <Typography variant="subtitle2" color="text.secondary">
                          来源格式
                        </Typography>
                        <Box
                          sx={{
                            display: "grid",
                            gap: 1.25,
                            gridTemplateColumns: "repeat(auto-fit, minmax(180px, 1fr))"
                          }}
                        >
                          {supportedPlugins.map((plugin) => (
                            <PluginOption
                              key={`source-${plugin.id}`}
                              plugin={plugin}
                              selected={selectedSource === plugin.id}
                              disabled={!sourceFormats.includes(plugin.id)}
                              onClick={() => setSelectedSource(plugin.id)}
                            />
                          ))}
                        </Box>
                      </Stack>

                      <Stack alignItems="center" justifyContent="center" sx={{ py: 2 }}>
                        <Avatar
                          sx={{
                            width: 48,
                            height: 48,
                            bgcolor: (theme) => alpha(theme.palette.primary.main, 0.12),
                            color: "primary.main"
                          }}
                        >
                          <ArrowForwardRoundedIcon />
                        </Avatar>
                      </Stack>

                      <Stack spacing={1.25}>
                        <Typography variant="subtitle2" color="text.secondary">
                          目标格式
                        </Typography>
                        <Box
                          sx={{
                            display: "grid",
                            gap: 1.25,
                            gridTemplateColumns: "repeat(auto-fit, minmax(180px, 1fr))"
                          }}
                        >
                          {supportedPlugins.map((plugin) => (
                            <PluginOption
                              key={`target-${plugin.id}`}
                              plugin={plugin}
                              selected={selectedTarget === plugin.id}
                              disabled={!availableTargets.includes(plugin.id)}
                              onClick={() => setSelectedTarget(plugin.id)}
                            />
                          ))}
                        </Box>
                      </Stack>
                    </Box>

                    <Divider />

                    <Box
                      sx={{
                        display: "grid",
                        gap: 1.5,
                        gridTemplateColumns: { xs: "1fr", md: "repeat(2, minmax(0, 1fr))" }
                      }}
                    >
                      <InfoTile label="文件名" value={report.filename || "未知文件"} />
                      <InfoTile
                        label="内容类型"
                        value={(report.content_types || []).join(" / ") || "暂未识别"}
                      />
                      <InfoTile
                        label="完整性"
                        value={
                          [
                            completeness.items_config ? "物品配置" : null,
                            completeness.categories_config ? "分类配置" : null,
                            completeness.resource_files ? "资源文件" : null
                          ]
                            .filter(Boolean)
                            .join(" / ") || "存在缺失项"
                        }
                        helper="识别结果会影响可转换范围。"
                      />
                      <InfoTile
                        label="统计"
                        value={`物品 ${details.item_count ?? 0} · 贴图 ${details.texture_count ?? 0} · 模型 ${details.model_count ?? 0}`}
                      />
                    </Box>

                    <Stack direction={{ xs: "column", sm: "row" }} spacing={1.25}>
                      <Button
                        variant="contained"
                        onClick={startConversion}
                        disabled={!sessionId || !selectedTarget}
                      >
                        开始转换
                      </Button>
                      <Button variant="outlined" onClick={resetState}>
                        重新选择文件
                      </Button>
                    </Stack>
                  </Stack>
                ) : null}

                {stage === "done" && (
                  <Card
                    variant="outlined"
                    sx={{
                      borderRadius: 4,
                      background: (theme) =>
                        `linear-gradient(180deg, ${alpha(theme.palette.success.main, 0.08)} 0%, ${alpha(
                          theme.palette.background.paper,
                          1
                        )} 100%)`
                    }}
                  >
                    <CardContent sx={{ p: 3 }}>
                      <Stack spacing={2}>
                        <Chip label="Completed" color="success" variant="outlined" sx={{ width: "fit-content" }} />
                        <Typography variant="h4">转换完成</Typography>
                        <Typography color="text.secondary">
                          输出文件已经准备好，可以直接下载使用。
                        </Typography>
                        <Stack direction={{ xs: "column", sm: "row" }} spacing={1.25}>
                          <Button href={downloadUrl} variant="contained" color="success">
                            下载结果
                          </Button>
                          <Button variant="outlined" onClick={resetState}>
                            继续处理其他文件
                          </Button>
                        </Stack>
                      </Stack>
                    </CardContent>
                  </Card>
                )}
              </Stack>
            </CardContent>
          </Card>

          <Stack spacing={2}>
            <Card>
              <CardContent sx={{ p: 2.5 }}>
                <Stack spacing={2}>
                  <Box>
                    <Typography variant="subtitle2" color="text.secondary" sx={{ mb: 0.75 }}>
                      Journey
                    </Typography>
                    <Typography variant="h5">三步完成转换</Typography>
                  </Box>
                  {journeySteps.map((step) => (
                    <StepCard
                      key={step.title}
                      title={step.title}
                      description={step.description}
                      icon={step.icon}
                    />
                  ))}
                </Stack>
              </CardContent>
            </Card>

            <Card>
              <CardContent sx={{ p: 2.5 }}>
                <Stack spacing={2}>
                  <Box>
                    <Typography variant="subtitle2" color="text.secondary" sx={{ mb: 0.75 }}>
                      Support
                    </Typography>
                    <Typography variant="h5">支持格式</Typography>
                  </Box>
                  <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap>
                    {supportedPlugins.map((plugin) => (
                      <Chip key={plugin.id} label={plugin.name} variant="outlined" />
                    ))}
                  </Stack>
                  <Typography variant="body2" color="text.secondary">
                    当前 Go 重写版已覆盖 ItemsAdder 与 Nexo 到 CraftEngine 的转换流程，其余格式会展示但不会误报为可转换。
                  </Typography>
                </Stack>
              </CardContent>
            </Card>

            <Card>
              <CardContent sx={{ p: 2.5 }}>
                <Stack spacing={1.5}>
                  <Stack direction="row" spacing={1} alignItems="center">
                    <Avatar
                      variant="rounded"
                      sx={{
                        width: 36,
                        height: 36,
                        bgcolor: (theme) => alpha(theme.palette.primary.main, 0.12),
                        color: "primary.main"
                      }}
                    >
                      <Inventory2RoundedIcon fontSize="small" />
                    </Avatar>
                    <Box>
                      <Typography variant="subtitle2" color="text.secondary">
                        Session
                      </Typography>
                      <Typography variant="body1" fontWeight={700}>
                        当前会话概览
                      </Typography>
                    </Box>
                  </Stack>
                  <InfoTile
                    label="当前状态"
                    value={stageLabels[stage]}
                    helper={statusText}
                  />
                  <InfoTile
                    label="来源 / 目标"
                    value={`${selectedSource || "未选择"} -> ${selectedTarget || "未选择"}`}
                    helper={summaryText}
                  />
                </Stack>
              </CardContent>
            </Card>
          </Stack>
        </Box>
      </Stack>
    </Box>
  );
}
