import React, { useEffect, useState } from "react";
import { getTradeHistory } from "../api";
import {
    Typography,
    Paper,
    Box,
    Table,
    TableBody,
    TableCell,
    TableContainer,
    TableHead,
    TableRow,
    Chip
} from "@mui/material";
import HistoryIcon from "@mui/icons-material/History";
// ========== 【高亮】引入 LineChart 图表组件 ==========
import {LineChart, ScatterChart} from '@mui/x-charts';
// ========== 【高亮】END ==========

function statusColor(status) {
    if (status === "成功") return "success";
    if (status === "失败") return "error";
    return "default";
}

export default function History() {
    const [records, setRecords] = useState([]);

    useEffect(() => {
        async function fetchRecords() {
            try {
                const { data } = await getTradeHistory();
                setRecords(data.records || []);
            } catch {
                setRecords([]);
            }
        }
        fetchRecords();
    }, []);

    // ========== 【高亮】生成价格趋势数据 ==========
    const priceTrend = records
        .filter(r => r.price && !isNaN(r.price))
        .map((r, i) => ({ x: i + 1, y: r.price }));

    // ========== 【高亮】END ==========

    return (
        <Box sx={{ display: "flex", minHeight: "60vh", alignItems: "center", justifyContent: "center" }}>
            <Paper elevation={3} sx={{ p: 4, minWidth: 500 }}>
                <Box display="flex" alignItems="center" mb={2}>
                    <HistoryIcon fontSize="large" color="primary" sx={{ mr: 1 }} />
                    <Typography variant="h5">历史交易记录</Typography>
                </Box>
                {/* ========== 【高亮】价格趋势点图 ========== */}
                {pricePoints.length > 0 &&
                    <Box sx={{mb:3}}>
                        <Typography variant="body1" mb={1}>成交价格走势 (点击数据点看卖出节点)</Typography>
                        <ScatterChart
                            width={500}
                            height={220}
                            series={[
                                {
                                    data: pricePoints,
                                    label: "成交价",
                                    markerLabel: d => `卖出节点: ${d.label}`,
                                    color: "#1976d2",
                                }
                            ]}
                            xAxis={[{label: "成交序号"}]}
                            yAxis={[{ label: '成交价格' }]}
                        />
                    </Box>
                }
                {/* ========== 【高亮】END ========== */}
                <TableContainer>
                    <Table>
                        <TableHead>
                            <TableRow>
                                <TableCell>类型</TableCell>
                                <TableCell>数量</TableCell>
                                {/* ========== 【高亮】新增表头 ========= */}
                                <TableCell>成交价格</TableCell>
                                <TableCell>卖出节点</TableCell>
                                {/* ========== 【高亮】END ========= */}
                                <TableCell>时间</TableCell>
                                <TableCell>状态</TableCell>
                            </TableRow>
                        </TableHead>
                        <TableBody>
                            {records.map((r, i) => (
                                <TableRow key={i}>
                                    <TableCell>{r.type}</TableCell>
                                    <TableCell>{r.amount}</TableCell>
                                    {/* ========== 【高亮】数据字段 ========= */}
                                    <TableCell>{r.price ? Number(r.price).toFixed(2) : "-"}</TableCell>
                                    <TableCell>{r.sellNode || "-"}</TableCell>
                                    {/* ========== 【高亮】END ========= */}
                                    <TableCell>{r.time}</TableCell>
                                    <TableCell>
                                        <Chip label={r.status} color={statusColor(r.status)} size="small" />
                                    </TableCell>
                                </TableRow>
                            ))}
                        </TableBody>
                    </Table>
                </TableContainer>
                {!records.length && <Typography sx={{ mt: 2 }}>暂无记录</Typography>}
            </Paper>
        </Box>
    );
}