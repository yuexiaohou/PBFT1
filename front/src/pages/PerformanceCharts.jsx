import React, { useEffect, useMemo, useState } from "react";
import { Paper, Typography, Box, FormControl, MenuItem, Select, InputLabel } from "@mui/material";
import { LineChart } from "@mui/x-charts";

const algoNames = [
    { value: "all", label: "全部对比" },
    { value: "pbft", label: "PBFT" },
    { value: "pos", label: "POS" },
    { value: "raft", label: "RAFT" },
    { value: "custom", label: "自定义" }
];

const colors = { pbft: "blue", pos: "orange", raft: "green", custom: "purple" };

// 固定横轴轮数
const fixedRounds = [100, 200, 300, 400, 500, 600, 700, 800, 900, 1000];

export default function PerformanceCharts() {
    // ========== 【高亮-2026-03-15 22:27:00】每个图表独立算法选择 ==========
    const [algoSuccess, setAlgoSuccess] = useState("all");
    const [algoError, setAlgoError] = useState("all");
    const [algoLeader, setAlgoLeader] = useState("all");
    // ========== 【高亮-2026-03-15 22:27:00】END ==========

    // 成功率图数据
    const [chartData, setChartData] = useState([]);
    const [singleRounds, setSingleRounds] = useState([]);
    // 错误节点数据
    const [errorRateData, setErrorRateData] = useState([]);
    // 主节点转换次数数据
    const [leaderChangeData, setLeaderChangeData] = useState([]);

    // loading/error
    const [loadingSuccess, setLoadingSuccess] = useState(true);
    const [errMsgSuccess, setErrMsgSuccess] = useState("");
    const [loadingError, setLoadingError] = useState(true);
    const [errMsgError, setErrMsgError] = useState("");
    const [loadingLeader, setLoadingLeader] = useState(true);
    const [errMsgLeader, setErrMsgLeader] = useState("");

    // ========== 【高亮-2026-03-15 22:27:00】挂单成功率数据请求 ==========
    useEffect(() => {
        async function fetchStats() {
            setLoadingSuccess(true);
            setErrMsgSuccess("");
            try {
                const url = "/api/performance" + (algoSuccess !== "all" ? `?algo=${algoSuccess}` : "");
                const res = await fetch(url);
                const data = await res.json();
                if (algoSuccess === "all") {
                    setChartData(data.algos || []);
                    setSingleRounds([]);
                } else {
                    setChartData([]);
                    setSingleRounds(data.rounds || []);
                }
            } catch (e) {
                setErrMsgSuccess("数据获取失败");
            }
            setLoadingSuccess(false);
        }
        fetchStats();
    }, [algoSuccess]);
    // ========== 【高亮-2026-03-15 22:27:00】END ==========

    // ========== 【高亮-2026-03-15 22:27:00】错误节点使用率数据请求 ==========
    useEffect(() => {
        async function fetchErrorRate() {
            setLoadingError(true);
            setErrMsgError("");
            try {
                const url = "/api/performance/errorrate" + (algoError !== "all" ? `?algo=${algoError}` : "");
                const res = await fetch(url);
                const data = await res.json();
                setErrorRateData(data.algos || []);
            } catch (e) {
                setErrMsgError("数据获取失败");
            }
            setLoadingError(false);
        }
        fetchErrorRate();
    }, [algoError]);
    // ========== 【高亮-2026-03-15 22:27:00】END ==========

    // ========== 【高亮-2026-03-15 22:27:00】主节点转换次数数据请求 ==========
    useEffect(() => {
        async function fetchLeader() {
            setLoadingLeader(true);
            setErrMsgLeader("");
            try {
                const url = "/api/performance/leaderchanges" + (algoLeader !== "all" ? `?algo=${algoLeader}` : "");
                const res = await fetch(url);
                const data = await res.json();
                setLeaderChangeData(data.algos || []);
            } catch (e) {
                setErrMsgLeader("数据获取失败");
            }
            setLoadingLeader(false);
        }
        fetchLeader();
    }, [algoLeader]);
    // ========== 【高亮-2026-03-15 22:27:00】END ==========

    const pointsToAlignedArray = (points, valueGetter, fallback = 0) => {
        const map = new Map((points || []).map((p) => [p.round, valueGetter(p)]));
        return fixedRounds.map((r) => {
            const v = map.get(r);
            return v === undefined || v === null || Number.isNaN(v) ? fallback : v;
        });
    };

    // 【高亮-2026-03-15 22:27:00】errorRateSeries 按错误节点选择算法
    const errorRateSeries = useMemo(() => {
        const filtered = algoError === "all" ? (errorRateData || []) : (errorRateData || []).filter((x) => x.algo === algoError);
        return filtered.map((as) => ({
            algo: as.algo,
            data: pointsToAlignedArray(
                as.points,
                (p) => Number((p.errorRate * 100).toFixed(2)),
                0
            ),
        }));
    }, [algoError, errorRateData]);

    // 【高亮-2026-03-15 22:27:00】leaderChangeSeries 按主节点选择算法
    const leaderChangeSeries = useMemo(() => {
        const filtered = algoLeader === "all" ? (leaderChangeData || []) : (leaderChangeData || []).filter((x) => x.algo === algoLeader);
        return filtered.map((as) => ({
            algo: as.algo,
            data: pointsToAlignedArray(
                as.points,
                (p) => Number(p.leaderChanges ?? 0),
                0
            ),
        }));
    }, [algoLeader, leaderChangeData]);

    const selectedLabel = (a) => algoNames.find(x => x.value === a)?.label || a;

    return (
        <Box sx={{ my: 4, mx: "auto", maxWidth: 760 }}>
            <Paper sx={{ p: 3 }}>
                <Typography variant="h5" mb={2}>算法撮合性能特性对比</Typography>

                {/* ========== 【高亮-2026-03-15 22:27:00】挂单成功率选择器及图表 ========== */}
                <Box sx={{ mb: 2 }}>
                    <FormControl fullWidth sx={{ maxWidth: 200, mb: 1 }}>
                        <InputLabel>选择算法（挂单成功率）</InputLabel>
                        <Select
                            label="选择算法（挂单成功率）"
                            value={algoSuccess}
                            onChange={e => setAlgoSuccess(e.target.value)}
                            size="small"
                        >
                            {algoNames.map((a) => (
                                <MenuItem key={a.value} value={a.value}>{a.label}</MenuItem>
                            ))}
                        </Select>
                    </FormControl>
                    {loadingSuccess ? (
                        <Typography>数据加载中...</Typography>
                    ) : (
                        <>
                            {errMsgSuccess && <Typography color="error">{errMsgSuccess}</Typography>}
                            <Typography variant="subtitle1" mt={1} gutterBottom>各轮次挂单成功率（%）</Typography>
                            {algoSuccess === "all" && chartData.length > 0 && (
                                <LineChart
                                    series={chartData.map((as) => ({
                                        data: (as.rounds || []).map((r) => Number((r.successRate * 100).toFixed(2))),
                                        label: selectedLabel(as.algo),
                                        color: colors[as.algo] || undefined,
                                    }))}
                                    xAxis={[{ label: "共识轮数", data: chartData[0]?.rounds?.map((r) => r.round) || [] }]}
                                    yAxis={[{ label: "挂单成功率(%)" }]}
                                    width={680}
                                    height={300}
                                />
                            )}
                            {algoSuccess !== "all" && singleRounds.length > 0 && (
                                <LineChart
                                    series={[
                                        {
                                            data: singleRounds.map((r) => Number((r.successRate * 100).toFixed(2))),
                                            label: selectedLabel(algoSuccess),
                                            color: colors[algoSuccess] || undefined,
                                        },
                                    ]}
                                    xAxis={[{ label: "共识轮数", data: singleRounds.map((r) => r.round) }]}
                                    yAxis={[{ label: "挂单成功率(%)" }]}
                                    width={680}
                                    height={300}
                                />
                            )}
                            {(algoSuccess === "all" && chartData.length === 0) || (algoSuccess !== "all" && singleRounds.length === 0) ?
                                <Typography color="text.secondary" sx={{ py: 2 }}>暂无统计数据</Typography>
                                : null
                            }
                        </>
                    )}
                </Box>
                {/* ========== 【高亮-2026-03-15 22:27:00】挂单成功率END ========== */}

                {/* ========== 【高亮-2026-03-15 22:27:00】错误节点使用率选择器及图表 ========== */}
                <Box sx={{ mb: 2 }}>
                    <FormControl fullWidth sx={{ maxWidth: 200, mb: 1 }}>
                        <InputLabel>选择算法（错误节点使用率）</InputLabel>
                        <Select
                            label="选择算法（错误节点使用率）"
                            value={algoError}
                            onChange={e => setAlgoError(e.target.value)}
                            size="small"
                        >
                            {algoNames.map((a) => (
                                <MenuItem key={a.value} value={a.value}>{a.label}</MenuItem>
                            ))}
                        </Select>
                    </FormControl>
                    {loadingError ? (
                        <Typography>数据加载中...</Typography>
                    ) : (
                        <>
                            {errMsgError && <Typography color="error">{errMsgError}</Typography>}
                            <Typography variant="subtitle1" mt={1} gutterBottom>
                                错误节点使用率（%）{algoError === "all" ? "对比" : `（${selectedLabel(algoError)}）`}
                            </Typography>
                            {errorRateSeries.length > 0 ? (
                                <LineChart
                                    series={errorRateSeries.map((s) => ({
                                        data: s.data,
                                        label: selectedLabel(s.algo),
                                        color: colors[s.algo] || undefined,
                                    }))}
                                    xAxis={[{ label: "共识轮数", data: fixedRounds }]}
                                    yAxis={[{ label: "错误节点使用率(%)" }]}
                                    width={680}
                                    height={300}
                                />
                            ) : (
                                <Typography color="text.secondary" sx={{ py: 2 }}>暂无错误节点使用率数据</Typography>
                            )}
                        </>
                    )}
                </Box>
                {/* ========== 【高亮-2026-03-15 22:27:00】错误节点使用率END ========== */}

                {/* ========== 【高亮-2026-03-15 22:27:00】主节点转换次数选择器及图表 ========== */}
                <Box sx={{ mb: 2 }}>
                    <FormControl fullWidth sx={{ maxWidth: 200, mb: 1 }}>
                        <InputLabel>选择算法（主节点转换次数）</InputLabel>
                        <Select
                            label="选择算法（主节点转换次数）"
                            value={algoLeader}
                            onChange={e => setAlgoLeader(e.target.value)}
                            size="small"
                        >
                            {algoNames.map((a) => (
                                <MenuItem key={a.value} value={a.value}>{a.label}</MenuItem>
                            ))}
                        </Select>
                    </FormControl>
                    {loadingLeader ? (
                        <Typography>数据加载中...</Typography>
                    ) : (
                        <>
                            {errMsgLeader && <Typography color="error">{errMsgLeader}</Typography>}
                            <Typography variant="subtitle1" mt={1} gutterBottom>
                                主节点转换次数{algoLeader === "all" ? "对比" : `（${selectedLabel(algoLeader)}）`}
                            </Typography>
                            {leaderChangeSeries.length > 0 ? (
                                <LineChart
                                    series={leaderChangeSeries.map((s) => ({
                                        data: s.data,
                                        label: selectedLabel(s.algo),
                                        color: colors[s.algo] || undefined,
                                    }))}
                                    xAxis={[{ label: "共识轮数", data: fixedRounds }]}
                                    yAxis={[{ label: "主节点转换次数" }]}
                                    width={680}
                                    height={300}
                                />
                            ) : (
                                <Typography color="text.secondary" sx={{ py: 2 }}>暂无主节点转换次数数据</Typography>
                            )}
                        </>
                    )}
                </Box>
                {/* ========== 【高亮-2026-03-15 22:27:00】主节点END ========== */}
            </Paper>
        </Box>
    );
}