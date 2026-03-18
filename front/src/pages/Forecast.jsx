import React, { useEffect, useState } from "react";
import { getForecast } from "../api";
import { Typography, Paper, Box, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, CircularProgress } from "@mui/material";
import { LineChart } from "@mui/x-charts";

export default function Forecast() {
    const [forecastData, setForecastData] = useState([]);
    const [loading, setLoading] = useState(true);
    const [rmse, setRmse] = useState(0);

    useEffect(() => {
        async function fetchData() {
            try {
                const res = await getForecast();
                if (res.data && res.data.forecast) {
                    setForecastData(res.data.forecast);
                    setRmse(res.data.ml_train_rmse);
                }
            } catch (e) {
                console.error("获取预测数据失败", e);
            } finally {
                setLoading(false);
            }
        }
        fetchData();
    }, []);

    const chartData = forecastData.map((d, i) => ({
        id: i,
        x: new Date(d.ds).getTime(),
        pred: d.hybrid_pred,
        lower: d.hybrid_lower_95,
        upper: d.hybrid_upper_95
    }));

    return (
        <Box sx={{ p: 3, maxWidth: 1000, mx: "auto" }}>
            <Typography variant="h5" mb={3}>📈 SARIMAX+ML 混合价格预测 (RMSE: {rmse?.toFixed(4)})</Typography>

            <Paper sx={{ p: 3, mb: 4, display: "flex", justifyContent: "center" }}>
                {loading ? (
                    <CircularProgress />
                ) : chartData.length > 0 ? (
                    <LineChart
                        width={900}
                        height={400}
                        series={[
                            { data: chartData.map(d => d.pred), label: '混合预测值', color: '#1976d2' },
                            { data: chartData.map(d => d.lower), label: '95% 下界', color: '#ff9800', showMark: false },
                            { data: chartData.map(d => d.upper), label: '95% 上界', color: '#ff9800', showMark: false }
                        ]}
                        xAxis={[{
                            data: chartData.map(d => d.x),
                            scaleType: 'time',
                            valueFormatter: (v) => new Date(v).toLocaleDateString(),
                            label: '日期'
                        }]}
                        yAxis={[{ label: '预测价格' }]}
                    />
                ) : (
                    <Typography color="text.secondary">暂无预测数据</Typography>
                )}
            </Paper>

            <TableContainer component={Paper}>
                <Table>
                    <TableHead>
                        <TableRow>
                            <TableCell>日期</TableCell>
                            <TableCell>Baseline</TableCell>
                            <TableCell>预测价</TableCell>
                            <TableCell>95%置信下界</TableCell>
                            <TableCell>95%置信上界</TableCell>
                        </TableRow>
                    </TableHead>
                    <TableBody>
                        {forecastData.map((r, i) => (
                            <TableRow key={i}>
                                <TableCell>{new Date(r.ds).toLocaleDateString()}</TableCell>
                                <TableCell>{r.baseline_mean?.toFixed(2)}</TableCell>
                                <TableCell style={{ fontWeight: "bold", color: "#1976d2" }}>{r.hybrid_pred?.toFixed(2)}</TableCell>
                                <TableCell>{r.hybrid_lower_95?.toFixed(2)}</TableCell>
                                <TableCell>{r.hybrid_upper_95?.toFixed(2)}</TableCell>
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>
            </TableContainer>
        </Box>
    );
}