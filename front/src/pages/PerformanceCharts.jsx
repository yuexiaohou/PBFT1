import React, { useEffect, useState } from "react";
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

export default function PerformanceCharts() {
    const [algo, setAlgo] = useState('all');
    const [chartData, setChartData] = useState([]);    // [{algo, rounds:[{round,successRate}]}]
    const [singleRounds, setSingleRounds] = useState([]);
    const [loading, setLoading] = useState(true);

    useEffect(()=>{
        async function fetchStats() {
            setLoading(true);
            let url = "/api/performance" + (algo !== 'all' ? `?algo=${algo}` : '');
            const res = await fetch(url);
            const data = await res.json();
            if (algo === "all") {
                setChartData(data.algos || []);
                setSingleRounds([]);
            } else {
                setChartData([]);
                setSingleRounds(data.rounds||[]);
            }
            setLoading(false);
        }
        fetchStats();
    }, [algo]);

    return (
        <Box sx={{ my: 4, mx: "auto", maxWidth: 700 }}>
            <Paper sx={{ p: 3 }}>
                <Typography variant="h5" mb={2}>算法撮合性能特性对比</Typography>
                <FormControl fullWidth sx={{ maxWidth: 200, mb:2 }}>
                    <InputLabel>选择算法</InputLabel>
                    <Select
                        label="选择算法"
                        value={algo}
                        onChange={e => setAlgo(e.target.value)}
                        size="small"
                    >
                        {algoNames.map(a => (
                            <MenuItem key={a.value} value={a.value}>{a.label}</MenuItem>
                        ))}
                    </Select>
                </FormControl>
                {loading ? <Typography>数据加载中...</Typography> :
                    <>
                        <Typography variant="subtitle1" mt={2} gutterBottom>各轮次挂单成功率（%）</Typography>
                        {algo === "all" && chartData.length > 0 &&
                            <LineChart
                                series={chartData.map(as => ({
                                    data: as.rounds.map(r => Number((r.successRate*100).toFixed(2))),
                                    label: algoNames.find(a => a.value===as.algo)?.label || as.algo,
                                    color: colors[as.algo] || undefined,
                                }))}
                                xAxis={[{ label: "共识轮数", data: chartData[0]?.rounds.map(r => r.round) || [] }]}
                                yAxis={[{ label: "挂单成功率(%)" }]}
                                width={600} height={300}
                            />
                        }
                        {algo !== "all" && singleRounds.length > 0 &&
                            <LineChart
                                series={[
                                    {
                                        data: singleRounds.map(r => Number((r.successRate*100).toFixed(2))),
                                        label: algoNames.find(a => a.value===algo)?.label || algo,
                                        color: colors[algo] || undefined
                                    }
                                ]}
                                xAxis={[{ label: "共识轮数", data: singleRounds.map(r => r.round) }]}
                                yAxis={[{ label: "挂单成功率(%)" }]}
                                width={600} height={300}
                            />
                        }
                        {( (algo === "all" && chartData.length===0) || (algo!=="all" && singleRounds.length===0) ) &&
                            <Typography color="text.secondary" sx={{ py: 4 }}>暂无统计数据</Typography>
                        }
                    </>
                }
            </Paper>
        </Box>
    );
}