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

// ======= 2026-03-05 高亮新增：固定横轴轮数（两张新增图都用这个） =======
const fixedRounds = [100, 200, 300, 400,500, 600, 700, 800, 900, 1000];
// ======= 2026-03-05 高亮新增 END =======

export default function PerformanceCharts() {
    const [algo, setAlgo] = useState("all");
    const [chartData, setChartData] = useState([]); // [{algo, rounds:[{round,successRate}]}]
    const [singleRounds, setSingleRounds] = useState([]);
    const [loading, setLoading] = useState(true);

    // ======= 2026-03-05 高亮新增：三张图的数据 =======
    const [errorRateData, setErrorRateData] = useState([]); // [{algo, points:[{round,errorRate}]}]
    const [leaderChangeData, setLeaderChangeData] = useState([]); // [{algo, points:[{round,leaderChanges}]}]
    // ======= 2026-03-05 高亮新增 END =======

    // ======= 2026-03-05 高亮新增：简单错误提示（避免 JSON 解析报错导致白屏） =======
    const [errMsg, setErrMsg] = useState("");
    // ======= 2026-03-05 高亮新增 END =======

    useEffect(() => {
        async function fetchStats() {
            setLoading(true);
            setErrMsg("");
            try {
                const url = "/api/performance" + (algo !== "all" ? `?algo=${algo}` : "");
                const res = await fetch(url);
                const data = await res.json();
                if (algo === "all") {
                    setChartData(data.algos || []);
                    setSingleRounds([]);
                } else {
                    setChartData([]);
                    setSingleRounds(data.rounds || []);
                }

                // ======= 高亮-2026-03-15 22:00:00：错误节点使用率与主节点转换次数支持切换算法 =======
                const res2 = await fetch("/api/performance/errorrate" + (algo !== "all" ? `?algo=${algo}` : ""));
                const data2 = await res2.json();
                setErrorRateData(data2.algos || []);

                const res3 = await fetch("/api/performance/leaderchanges" + (algo !== "all" ? `?algo=${algo}` : ""));
                const data3 = await res3.json();
                setLeaderChangeData(data3.algos || []);
                // ======= 高亮-2026-03-15 22:00:00 END =======
            } catch (e) {
                setErrMsg("数据获取失败");
            }
            setLoading(false);
        }
        fetchStats();
    }, [algo]);

    const pointsToAlignedArray = (points, valueGetter, fallback = 0) => {
        const map = new Map((points || []).map((p) => [p.round, valueGetter(p)]));
        return fixedRounds.map((r) => {
            const v = map.get(r);
            return v === undefined || v === null || Number.isNaN(v) ? fallback : v;
        });
    };

    // ======= 【高亮-2026-03-13】修改：映射逻辑，将 errorRate 转换为百分比共识概率 =======
    // ======= 高亮-2026-03-15 22:00:00：errorRateSeries 支持单算法切换 =======
    const errorRateSeries = useMemo(() => {
        const filtered = algo === "all" ? (errorRateData || []) : (errorRateData || []).filter((x) => x.algo === algo);
        return filtered.map((as) => ({
            algo: as.algo,
            data: pointsToAlignedArray(
                as.points,
                (p) => Number((p.errorRate * 100).toFixed(2)),
                0
            ),
        }));
    }, [algo, errorRateData]);
    // ======= 高亮-2026-03-15 22:00:00 END =======

    // ======= 2026-03-05 高亮新增：主节点转换次数 series（all=多条；单算法=一条） =======
    // ======= 高亮-2026-03-15 22:00:00：leaderChangeSeries 支持单算法切换 =======
    const leaderChangeSeries = useMemo(() => {
        const filtered = algo === "all" ? (leaderChangeData || []) : (leaderChangeData || []).filter((x) => x.algo === algo);

        return filtered.map((as) => ({
            algo: as.algo,
            data: pointsToAlignedArray(
                as.points,
                (p) => Number(p.leaderChanges ?? 0),
                0
            ),
        }));
    }, [algo, leaderChangeData]);
    // ======= 高亮-2026-03-15 22:00:00 END =======

    // ======= 2026-03-05 高亮新增：标题随选择算法变化 =======
    const selectedAlgoLabel = useMemo(() => {
        return algoNames.find((a) => a.value === algo)?.label || algo;
    }, [algo]);
    // ======= 2026-03-05 高亮新增 END =======

    return (
        <Box sx={{ my: 4, mx: "auto", maxWidth: 760 }}>
            <Paper sx={{ p: 3 }}>
                <Typography variant="h5" mb={2}>算法撮合性能特性对比</Typography>

                <FormControl fullWidth sx={{ maxWidth: 200, mb: 2 }}>
                    <InputLabel>选择算法</InputLabel>
                    <Select
                        label="选择算法"
                        value={algo}
                        onChange={(e) => setAlgo(e.target.value)}
                        size="small"
                    >
                        {algoNames.map((a) => (
                            <MenuItem key={a.value} value={a.value}>{a.label}</MenuItem>
                        ))}
                    </Select>
                </FormControl>

                {loading ? (
                    <Typography>数据加载中...</Typography>
                ) : (
                    <>
                        {errMsg && (
                            <Typography color="error" sx={{ mb: 2 }}>{errMsg}</Typography>
                        )}

                        <Typography variant="subtitle1" mt={2} gutterBottom>各轮次挂单成功率（%）</Typography>

                        {/* ======= 高亮-2026-03-15 22:00:00: 第1张图：支持全部和单算法模式 ======= */}
                        {algo === "all" && chartData.length > 0 && (
                            <LineChart
                                series={chartData.map((as) => ({
                                    data: (as.rounds || []).map((r) => Number((r.successRate * 100).toFixed(2))),
                                    label: algoNames.find((a) => a.value === as.algo)?.label || as.algo,
                                    color: colors[as.algo] || undefined,
                                }))}
                                xAxis={[{ label: "共识轮数", data: chartData[0]?.rounds?.map((r) => r.round) || [] }]}
                                yAxis={[{ label: "挂单成功率(%)" }]}
                                width={680}
                                height={300}
                            />
                        )}

                        {algo !== "all" && singleRounds.length > 0 && (
                            <LineChart
                                series={[
                                    {
                                        data: singleRounds.map((r) => Number((r.successRate * 100).toFixed(2))),
                                        label: selectedAlgoLabel,
                                        color: colors[algo] || undefined,
                                    },
                                ]}
                                xAxis={[{ label: "共识轮数", data: singleRounds.map((r) => r.round) }]}
                                yAxis={[{ label: "挂单成功率(%)" }]}
                                width={680}
                                height={300}
                            />
                        )}
                        {/* ======= 高亮-2026-03-15 22:00:00 END ======= */}

                        {((algo === "all" && chartData.length === 0) || (algo !== "all" && singleRounds.length === 0)) && (
                            <Typography color="text.secondary" sx={{ py: 2 }}>暂无统计数据</Typography>
                        )}

                        {/* ======================= 高亮-2026-03-15 22:00:00 图2：错误节点使用率（支持全部和单算法） BEGIN ======================= */}
                        <Typography variant="subtitle1" mt={4} gutterBottom>
                            错误节点使用率（%）{algo === "all" ? "对比" : `（${selectedAlgoLabel}）`}（共识轮数： 100 /200/300/400/500/600/700/800/900/1000）
                        </Typography>

                        {errorRateSeries.length > 0 ? (
                            <LineChart
                                series={errorRateSeries.map((s) => ({
                                    data: s.data,
                                    label: algoNames.find((a) => a.value === s.algo)?.label || s.algo,
                                    color: colors[s.algo] || undefined,
                                }))}
                                xAxis={[{ label: "共识轮数", data: fixedRounds }]}
                                yAxis={[{ label: "错误节点使用率(%)" }]}
                                width={680}
                                height={300}
                            />
                        ) : (
                            <Typography color="text.secondary" sx={{ pb: 2 }}>
                                暂无错误节点使用率数据
                            </Typography>
                        )}

                        {/* ======================= 高亮-2026-03-15 22:00:00 图3：主节点转换次数（支持全部和单算法） BEGIN ======================= */}
                        <Typography variant="subtitle1" mt={4} gutterBottom>
                            主节点转换次数{algo === "all" ? "对比" : `（${selectedAlgoLabel}）`}（共识轮数： 100 /200/300/400/500/600/700/800/900/1000）
                        </Typography>

                        {leaderChangeSeries.length > 0 ? (
                            <LineChart
                                series={leaderChangeSeries.map((s) => ({
                                    data: s.data,
                                    label: algoNames.find((a) => a.value === s.algo)?.label || s.algo,
                                    color: colors[s.algo] || undefined,
                                }))}
                                xAxis={[{ label: "共识轮数", data: fixedRounds }]}
                                yAxis={[{ label: "主节点转换次数" }]}
                                width={680}
                                height={300}
                            />
                        ) : (
                            <Typography color="text.secondary" sx={{ pb: 2 }}>
                                暂无主节点转换次数数据
                            </Typography>
                        )}
                        {/* ======================= 高亮-2026-03-15 22:00:00 图3 END ======================= */}
                    </>
                )}
            </Paper>
        </Box>
    );
}