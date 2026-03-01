import React, { useEffect, useState } from "react";
import { Box, Typography, Paper } from "@mui/material";
import { LineChart, ScatterChart } from "@mui/x-charts";
import axios from "axios";

// ========== 高亮：撮合轮次最低价格及节点分布图 ==========
function PriceMinChart({ rounds }) {
    // 每轮 {Round, MinPrice, Buyer, Seller}
    const pricePoints = rounds.map(r => ({
        x: r.Round,
        y: r.MinPrice,
        buyer: r.Buyer,
        seller: r.Seller
    }));

    return (
        <Box sx={{mb:4}}>
            <Typography variant="h6">各轮次最低成交价格与节点</Typography>
            <ScatterChart
                width={600}
                height={270}
                series={[
                    {
                        data: pricePoints.map(p => ({ x: p.x, y: p.y, label: `${p.buyer}->${p.seller}` })),
                        label: "最低成交价",
                        markerLabel: d => `第${d.x}轮: ${d.y.toFixed(2)} (${d.label})`,
                        color: "#1976d2"
                    }
                ]}
                xAxis={[{ label: "共识轮数" }]}
                yAxis={[{ label: "最低成交价格" }]}
            />
        </Box>
    );
}

// ========== 高亮：挂单成功率折线图 ==========
function SuccessRateChart({ rounds }) {
    const roundX = rounds.map(r => r.Round);
    const successY = rounds.map(r => r.SuccessRate * 100);

    return (
        <Box>
            <Typography variant="h6">每轮撮合成功率（%）</Typography>
            <LineChart
                width={600}
                height={240}
                series={[{
                    data: successY,
                    label: "挂单成功率(%)",
                    color: "#00695f",
                }]}
                xAxis={[{ data: roundX, label: "共识轮数" }]}
                yAxis={[{ label: "成功率（%）" }]}
            />
        </Box>
    );
}

// ========== 高亮：页面聚合 ==========
export default function MatchCharts() {
    const [rounds, setRounds] = useState([]);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        async function fetchCharts() {
            setLoading(true);
            const { data } = await axios.get("/api/trade/pricechart");
            setRounds(data.rounds || []);
            setLoading(false);
        }
        fetchCharts();
    }, []);

    return (
        <Box sx={{my:4, mx:"auto", maxWidth: 700}}>
            <Paper sx={{p:3}}>
                <Typography variant="h5" mb={2}>撮合交易统计图表</Typography>
                {loading ? <Typography>数据加载中...</Typography> : (
                    <>
                        <PriceMinChart rounds={rounds} />
                        <SuccessRateChart rounds={rounds} />
                    </>
                )}
            </Paper>
        </Box>
    );
}