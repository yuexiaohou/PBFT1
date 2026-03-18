// ========== 【高亮-2026-03-18】修复报错：引入 useMemo 钩子 ==========
import React, { useEffect, useState, useMemo } from "react";
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

// 当面对非黑即白的二元状态时，JavaScript 的三元运算符（条件 ? 成立时返回 : 不成立时返回）是业内最标准的最佳实践。
function statusColor(status) {
    return status === "成功" ? "success" : "error";
}

export default function History() {
    const [records, setRecords] = useState([]);
    //const [loading, setLoading] = useState(true); 主要是为了提升用户体验（UX）和代码逻辑的健壮性，这也是 React 开发中处理异步网络请求的“标准最佳实践”。
    const [loading, setLoading] = useState(true);

    //使用 finally 是 JavaScript 里的最佳实践，它能保证无论发生什么异常，最终一定会执行关闭 loading 的操作。如果不用 finally，一旦接口报错，页面可能会永远卡在“加载中”的状态。
    //if (res.data && res.data.records) { setRecords(...) } 相较于if (res.data && res.data.records) {}增加了一层安全校验（防御性编程）。只有明确确认 res.data 存在，且 res.data.records 也存在时，才去更新数据。
    useEffect(() => {
        async function fetchRecords() {
            try {
                const res = await getTradeHistory();
                if (res.data && res.data.records) {
                    setRecords(res.data.records);
                }
            } catch (e) {
                console.error("获取交易历史失败", e);
            } finally {
                setLoading(false);
            }
        }
        fetchRecords();
    }, []);

    // ========== 【高亮-2026-03-16 12:30:00】格式化图表数据：X轴为时间戳，Y轴为价格 ==========
    // 使用 useMemo 来缓存计算结果，只有当 records 发生变化时才重新计算 chartData，这也是 React 中优化性能的最佳实践。
    // 通过 map() 方法，我们遍历 records 数组，为每条记录创建一个新的对象，其中 x 是时间戳，y 是价格，label 是卖出节点信息（如果有的话）。这样就能确保 ScatterChart 能正确地识别和显示时间轴与价格轴的数据。
    const chartData = useMemo(() => {
        // 后端记录是按时间倒序的，为了图表从左到右显示时间正序，我们先 reverse()
        return [...records].reverse().map((r, i) => {
            // 将 "YYYY-MM-DD HH:mm:ss" 转换为合法的时间对象格式 "YYYY-MM-DDTHH:mm:ss"
            const safeTimeStr = r.time.replace(" ", "T");
            return {
                id: i,
                x: new Date(safeTimeStr).getTime(), // 转换为时间戳作为X轴
                y: r.price, // Y轴为价格
                label: r.sellerNode || r.node // 气泡的标签显示详细节点信息
            };
        });
    }, [records]);

    // ScatterChart 默认处理数字类型的 X 和 Y 轴数据。由于后端返回的时间是字符串格式，我们需要在前端将其转换为 JavaScript 的 Date 对象，并进一步转换为时间戳（毫秒数）来适配 ScatterChart 的要求。
    // 通过 valueFormatter，我们可以自定义 X 轴的显示格式，将时间戳转换回可读的时间字符串，这样用户在查看图表时就能直观地理解每个数据点对应的交易时间和价格信息。
    // 这种处理方式不仅符合 ScatterChart 的数据格式要求，还能提升用户体验，让图表更具可读性和交互性。
    // 通过在 ScatterChart 的 series 中添加 valueFormatter，我们可以为每个数据点提供一个详细的提示信息，当用户点击或悬停在数据点上时，能够看到对应的交易时间、价格以及卖出节点信息。这种交互设计能够帮助用户更深入地理解每笔交易的具体情况，从而提升图表的实用性和用户体验。
    // 通过在 Table 中展示卖出节点信息，我们可以让用户在查看交易记录时直接看到每笔交易的卖出节点（如果有的话）。这不仅增加了表格的可读性，还能帮助用户更好地理解每笔交易的背景和细节，从而提升整体的用户体验。
    // <TableCell>{r.sellerNode || r.node}</TableCell>。优先读取我们刚才在后端新存入的、带有 PBFT 详情的 sellerNode 字段。
    return (
        <Box sx={{ p: 3, maxWidth: 1000, mx: "auto" }}>
            <Typography variant="h5" mb={3}>🕒 历史交易记录</Typography>

            <Paper sx={{ p: 3, mb: 4 }}>
                <Typography variant="subtitle1" gutterBottom>
                    成交价格走势 (点击数据点看卖出节点)
                </Typography>
                {loading ? (
                    <Typography>数据加载中...</Typography>
                ) : chartData.length > 0 ? (
                    // ========== 【高亮-2026-03-16 12:30:00】配置散点图：时间轴与价格轴 ==========
                    <ScatterChart
                        width={900}
                        height={350}
                        series={[
                            {
                                label: '成交价',
                                data: chartData,
                                valueFormatter: (v) => `时间: ${new Date(v.x).toLocaleTimeString()}, 价格: ${v.y.toFixed(2)}, 节点: ${v.label}`,
                            }
                        ]}
                        xAxis={[
                            {
                                scaleType: 'time', // 指定横轴刻度类型为时间
                                valueFormatter: (v) => new Date(v).toLocaleTimeString(),
                                label: '交易时间' // X轴文字
                            }
                        ]}
                        yAxis={[
                            {
                                label: '最低成交价格' // Y轴文字
                            }
                        ]}
                    />
                ) : (
                    <Typography color="text.secondary">暂无数据</Typography>
                )}
            </Paper>

            <TableContainer component={Paper}>
                <Table>
                    <TableHead>
                        <TableRow>
                            <TableCell>类型</TableCell>
                            <TableCell>数量</TableCell>
                            <TableCell>成交价格</TableCell>
                            <TableCell>卖出节点</TableCell>
                            <TableCell>时间</TableCell>
                            <TableCell>状态</TableCell>
                        </TableRow>
                    </TableHead>
                    <TableBody>
                        {records.map((r, i) => (
                            <TableRow key={i}>
                                <TableCell>{r.type}</TableCell>
                                <TableCell>{r.amount}</TableCell>
                                <TableCell>{r.price > 0 ? r.price.toFixed(2) : "-"}</TableCell>
                                {/* ========== 【高亮-2026-03-16 12:30:00】展示详细属性的卖出节点 ========== */}
                                <TableCell>{r.sellerNode || r.node}</TableCell>
                                <TableCell>{r.time}</TableCell>
                                <TableCell>
                                    <Chip label={r.status} color={statusColor(r.status)} size="small" />
                                </TableCell>
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>
            </TableContainer>
        </Box>
    );
}