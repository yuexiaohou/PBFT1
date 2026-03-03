import React, { useEffect, useState } from "react";
import { Paper, Typography, Box } from "@mui/material";
import { LineChart, ScatterChart } from "@mui/x-charts"; // 保证已安装
// ======== 2026-03-04 高亮：引入 PieChart，如需饼图 ========
// import PBFTPieChart from '../components/PBFTPieChart';
// ======== 高亮 END ========

// ======== 2026-03-04 高亮：统计最低价格散点图 ========
function PriceMinChart({ rounds }) {
    if (!rounds || rounds.length === 0) return <Typography color="text.secondary" sx={{ py:5 }}>暂无撮合统计数据</Typography>;
    const data = rounds.map(r => ({
        x: r.round,
        y: r.minPrice,
        label: `${r.buyerNode}→${r.sellerNode}`
    }));
    return (
        <ScatterChart
            series={[
                {
                    data,
                    label: "最低成交价",
                    markerLabel: d => `第${d.x}轮: ${d.y.toFixed(2)} (${d.label})`,
                }
            ]}
            xAxis={[{ label: "共识轮数", data: data.map(d => d.x) }]}
            yAxis={[{ label: "最低成交价格" }]}
            width={600}
            height={260}
        />
    );
}

// ======== 2026-03-04 高亮：撮合成功率折线图 ========
function SuccessRateChart({ rounds }) {
    if (!rounds || rounds.length === 0) return null;
    return (
        <LineChart
            series={[
                {
                    data: rounds.map(r => ({ x: r.round, y: (r.successRate * 100).toFixed(2), label: `${r.buyerNode}→${r.sellerNode}` }),
                    ),
                    label: "挂单成功率(%)",
                }
            ]}
            xAxis={[{ label: "共识轮数", data: rounds.map(r => r.round) }]}
            yAxis={[{ label: "成功率 (%)" }]}
            width={600}
            height={260}
        />
    );
}

// ======== 2026-03-04 高亮：撮合统计主页面 ========
export default function MatchCharts() {
    const [rounds, setRounds] = useState([]);
    const [loading, setLoading] = useState(true);

    // ======== 2026-03-04 高亮：拉取数据，并调试打印 ========
    useEffect(() => {
        async function fetchCharts() {
            setLoading(true);
            try {
                const res = await fetch("/api/trade/pricechart"); // 直接API路径
                const data = await res.json();
                setRounds(data.rounds || []);
                // ======== 2026-03-04 高亮：调试输出 ========
                console.log("撮合统计 rounds 数据:", data.rounds);
            } catch (err) {
                setRounds([]);
            }
            setLoading(false);
        }
        fetchCharts();
    }, []);

    // ======== 2026-03-04 高亮：页面渲染 ========
    return (
        <Box sx={{ my: 4, mx: "auto", maxWidth: 700 }}>
            <Paper sx={{ p: 3 }}>
                <Typography variant="h5" mb={2}>撮合交易统计图表</Typography>
                {loading ? <Typography>数据加载中...</Typography>
                    : (rounds.length === 0
                            ? <Typography color="text.secondary" sx={{ py: 5 }}>暂无撮合统计数据</Typography>
                            : <>
                                <Typography variant="subtitle1" gutterBottom>各轮次最低成交价格与节点</Typography>
                                <PriceMinChart rounds={rounds} />
                                <Typography variant="subtitle1" mt={3} gutterBottom>每轮撮合成功率（%）</Typography>
                                <SuccessRateChart rounds={rounds} />
                            </>
                    )
                }
            </Paper>
        </Box>
    );
}