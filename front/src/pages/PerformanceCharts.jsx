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
const fixedRounds = [10, 100, 1000, 10000];
// ======= 2026-03-05 高亮新增 END =======

export default function PerformanceCharts() {
    const [algo, setAlgo] = useState("all");
    const [chartData, setChartData] = useState([]); // [{algo, rounds:[{round,successRate}]}]
    const [singleRounds, setSingleRounds] = useState([]);
    const [loading, setLoading] = useState(true);

    // ======= 2026-03-05 高亮新增：两张新增折线图的数据（后端返回全量 algos） =======
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

            // 1) 原有：成功率接口
            try {
                const url = "/api/performance" + (algo !== "all" ? `?algo=${algo}` : "");
                const res = await fetch(url);
                if (!res.ok) throw new Error(`performance http ${res.status}`);
                const data = await res.json();

                if (algo === "all") {
                    setChartData(data.algos || []);
                    setSingleRounds([]);
                } else {
                    setChartData([]);
                    setSingleRounds(data.rounds || []);
                }
            } catch (e) {
                setChartData([]);
                setSingleRounds([]);
                setErrMsg("性能特性（成功率）数据获取失败");
            }

            // 2) 错误节点使用率（全量）
            try {
                const res2 = await fetch("/api/performance/errorrate");
                if (!res2.ok) throw new Error(`errorrate http ${res2.status}`);
                const data2 = await res2.json();
                setErrorRateData(data2.algos || []);
            } catch (e) {
                setErrorRateData([]);
            }

            // 3) 主节点转换次数（全量）
            try {
                const res3 = await fetch("/api/performance/leaderchanges");
                if (!res3.ok) throw new Error(`leaderchanges http ${res3.status}`);
                const data3 = await res3.json();
                setLeaderChangeData(data3.algos || []);
            } catch (e) {
                setLeaderChangeData([]);
            }

            setLoading(false);
        }

        fetchStats();
    }, [algo]);

    // ======= 2026-03-05 高亮新增：工具函数：从后端 points 映射到固定 rounds 数组 =======
    const pointsToAlignedArray = (points, valueGetter, fallback = 0) => {
        const map = new Map((points || []).map((p) => [p.round, valueGetter(p)]));
        return fixedRounds.map((r) => {
            const v = map.get(r);
            return v === undefined || v === null || Number.isNaN(v) ? fallback : v;
        });
    };
    // ======= 2026-03-05 高亮新增 END =======

    // ======= 2026-03-05 高亮新增：错误节点使用率 series（all=多条；单算法=一条） =======
    const errorRateSeries = useMemo(() => {
        const filtered = algo === "all" ? (errorRateData || []) : (errorRateData || []).filter((x) => x.algo === algo);

        return filtered.map((as) => ({
            algo: as.algo,
            data: pointsToAlignedArray(
                as.points,
                (p) => Number((((p.errorRate ?? 0) * 100)).toFixed(2)),
                0
            ),
        }));
    }, [algo, errorRateData]);
    // ======= 2026-03-05 高亮新增 END =======

    // ======= 2026-03-05 高亮新增：主节点转换次数 series（all=多条；单算法=一条） =======
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
    // ======= 2026-03-05 高亮新增 END =======

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

                        {((algo === "all" && chartData.length === 0) || (algo !== "all" && singleRounds.length === 0)) && (
                            <Typography color="text.secondary" sx={{ py: 2 }}>暂无统计数据</Typography>
                        )}

                        {/* ======================= 2026-03-05 高亮新增：图2：错误节点使用率对比/单算法 BEGIN ======================= */}
                        <Typography variant="subtitle1" mt={4} gutterBottom>
                            错误节点使用率（%）{algo === "all" ? "对比" : `（${selectedAlgoLabel}）`}（共识轮数：10 / 100 / 1000 / 10000）
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
                        {/* ======================= 2026-03-05 高亮新增：图2 END ======================= */}

                        {/* ======================= 2026-03-05 高亮新增：图3：主节点转换次数对比/单算法 BEGIN ======================= */}
                        <Typography variant="subtitle1" mt={4} gutterBottom>
                            主节点转换次数{algo === "all" ? "对比" : `（${selectedAlgoLabel}）`}（共识轮数：10 / 100 / 1000 / 10000）
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
                        {/* ======================= 2026-03-05 高亮新增：图3 END ======================= */}
                    </>
                )}
            </Paper>
        </Box>
    );
}