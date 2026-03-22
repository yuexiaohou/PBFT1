import React, { useEffect, useMemo, useState } from "react";
import {Paper, Typography, Box, FormControl, MenuItem, Select, InputLabel, Checkbox, ListItemText} from "@mui/material";
import { LineChart } from "@mui/x-charts";

// ========== 【高亮-2026-03-16 09:00:00】算法选择数组 ==========
const algoNames = [
    { value: "pbft", label: "PBFT" },
    { value: "pos", label: "POS" },
    { value: "raft", label: "RAFT" },
    { value: "apbft", label: "APBFT" }
];

const colors = { pbft: "blue", pos: "orange", raft: "green", apbft: "purple" };

// 【高亮-2026-03-15 23:10:00】三个图独立的横轴采样
const roundsChart1 = Array.from({length: 20}, (_, i) => i+1);  // 1~20
const roundsChart234 = [100,200,300,400,500,600,700,800,900,1000]; // 图2~图4

// 算法多选下拉框组件【高亮-2026-03-15 23:10:00】
// 在MUI的Select组件中，默认就是单选， 因此要实现多选，需要设置multiple属性，并且value必须是一个数组
function AlgoMultiSelect({ label, value, onChange }) {
    return (
        <FormControl fullWidth sx={{ maxWidth: 350 }}>
            <InputLabel>{label}</InputLabel>
            <Select
                multiple
                value={value}// 数组类型
                onChange={onChange}
                renderValue={(selected) => selected.map(v => algoNames.find(a => a.value===v)?.label).join(", ")}
                label={label}
                size="small"
             variant="outlined">
                {algoNames.map((a) => (
                    <MenuItem key={a.value} value={a.value}>
                        <Checkbox checked={value.indexOf(a.value) > -1} />
                        <ListItemText primary={a.label} />
                    </MenuItem>
                ))}
            </Select>
        </FormControl>
    );
}

export default function PerformanceCharts() {
    // ========== 【高亮-2026-03-15 22:27:00】每个图表独立算法选择 ==========
    // 每个图表的算法选择状态均为多选数组，并独立声明，兼容多算法对比需求
    const [algosSuccess, setAlgosSuccess] = useState(algoNames.map(a=>a.value)); // 图1
    const [algosError, setAlgosError] = useState(algoNames.map(a=>a.value)); // 图2
    const [algosLeader, setAlgosLeader] = useState(algoNames.map(a=>a.value)); // 图3
    const [algosCost, setAlgosCost] = useState(algoNames.map(a=>a.value)); // 图4
    // ======================= 【高亮-2026-03-22】新增：时延图表独立的算法选择状态 =======================
    const [algosLatency, setAlgosLatency] = useState(algoNames.map(a=>a.value)); // 图5 (时延)

    // 图数据
    // 之前的const [chartData, setChartData] = useState([])只能存储单算法数据
    const [chart1Data, setChart1Data] = useState([]);
    const [chart2ErrorData, setChart2ErrorData] = useState([]);
    const [chart3LeaderData, setChart3LeaderData] = useState([]);
    const [chart4CostData, setChart4CostData] = useState([]); // 【高亮-2026-03-15 23:40:00】
    // ======================= 【高亮-2026-03-22】新增：时延图表独立数据状态 =======================
    const [chart5LatencyData, setChart5LatencyData] = useState([]);

    const [loading1, setLoading1] = useState(true), [errMsg1, setErrMsg1] = useState("");
    const [loading2, setLoading2] = useState(true), [errMsg2, setErrMsg2] = useState("");
    const [loading3, setLoading3] = useState(true), [errMsg3, setErrMsg3] = useState("");
    const [loading4, setLoading4] = useState(true), [errMsg4, setErrMsg4] = useState(""); // 【高亮-2026-03-15 23:40:00】
    // ======================= 【高亮-2026-03-22】新增：时延图表加载与报错状态 =======================
    const [loading5, setLoading5] = useState(true), [errMsg5, setErrMsg5] = useState("");

    // 图1：挂单成功率
    // 旧写法是要实现可以在图中呈现单个算法和全部算法，因此通过采用algosSuccess!=="all"与?algo=${algoSuccess}` : "",由于要实现多选，因此通过algosSuccess.join(",")实现数组应用
    useEffect(() => {
        async function fetchStats() {
            setLoading1(true); setErrMsg1("");
            try {
                const url = `/api/performance?algo=${algosSuccess.join(",")}&rounds=1-20`;
                const res = await fetch(url);
                const data = await res.json();
                setChart1Data(data.algos || []);
            } catch (e) { setErrMsg1("数据获取失败"); }
            setLoading1(false);
        }
        fetchStats();
    }, [algosSuccess]);

    // 图2：错误节点参与率
    useEffect(() => {
        async function fetchErrorRate() {
            setLoading2(true); setErrMsg2("");
            try {
                const url = `/api/performance/errorrate?algo=${algosError.join(",")}`;
                const res = await fetch(url);
                const data = await res.json();
                setChart2ErrorData(data.algos || []);
            } catch (e) { setErrMsg2("数据获取失败"); }
            setLoading2(false);
        }
        fetchErrorRate();
    }, [algosError]);

    // 图3：主节点切换次数
    useEffect(() => {
        async function fetchLeader() {
            setLoading3(true); setErrMsg3("");
            try {
                const url = `/api/performance/leaderchanges?algo=${algosLeader.join(",")}`;
                const res = await fetch(url);
                const data = await res.json();
                setChart3LeaderData(data.algos || []);
            } catch (e) { setErrMsg3("数据获取失败"); }
            setLoading3(false);
        }
        fetchLeader();
    }, [algosLeader]);

    // 图4：平均节点开销【高亮-2026-03-15 23:40:00】
    useEffect(() => {
        async function fetchNodeCost() {
            setLoading4(true); setErrMsg4("");
            try {
                const url = `/api/performance/nodecost?algo=${algosCost.join(",")}`;
                const res = await fetch(url);
                const data = await res.json();
                setChart4CostData(data.algos || []);
            } catch (e) { setErrMsg4("数据获取失败"); }
            setLoading4(false);
        }
        fetchNodeCost();
    }, [algosCost]);

    // ======================= 【高亮-2026-03-22】新增：获取交易平均时延数据 =======================
    useEffect(() => {
        async function fetchLatency() {
            setLoading5(true); setErrMsg5("");
            try {
                // 修改了与后端对齐的路由和带上了参数
                const url = `/api/performance/latency?algo=${algosLatency.join(",")}`;
                const res = await fetch(url);
                if (!res.ok) throw new Error("HTTP error");
                const data = await res.json();

                // 此时必定返回的是 { algos: [...] } ���式了
                setChart5LatencyData(data.algos || []);
            } catch (e) {
                console.error(e);
                setErrMsg5("时延数据获取失败");
            }
            setLoading5(false);
        }
        fetchLatency();
    }, [algosLatency]);

    // 工具：对齐采样点
    // 将axis作为参数传进去
    const alignPoints = (allPoints, axis, getter) => {
        const map = new Map((allPoints || []).map((p) => [p.round, getter(p)]));
        return axis.map(r => {
            const v = map.get(r); return v==null||isNaN(v)?0:v;
        });
    };

    // 图1
    // 采用多选模式的 filter+map+alignPoints，首先通过filter筛选出用户选择的算法数据，然后通过map对每个算法的数据进行处理，使用alignPoints函数将原始数据对齐到预设的roundsChart1上，并且通过getter函数提取出需要展示的数值（挂单成功率转百分比）。最终返回一个包含多个算法系列数据的数组，每个系列包含算法名称和对应的数据点。
    const chart1Series = useMemo(() => {
        return (chart1Data || [])
            .filter(as => algosSuccess.includes(as.algo))
            .map(as => ({
                algo: as.algo,
                data: alignPoints(as.rounds, roundsChart1, r => Number((r.successRate * 100).toFixed(2))),
            }));
    }, [chart1Data, algosSuccess]);
    // 图2
    const chart2Series = useMemo(() => {
        return (chart2ErrorData || [])
            .filter(as => algosError.includes(as.algo))
            .map(as => ({
                algo: as.algo,
                data: alignPoints(as.points, roundsChart234, r=>Number((r.errorRate*100).toFixed(2))),
            }));
    }, [chart2ErrorData, algosError]);
    // 图3
    const chart3Series = useMemo(() => {
        return (chart3LeaderData || [])
            .filter(as => algosLeader.includes(as.algo))
            .map(as => ({
                algo: as.algo,
                data: alignPoints(as.points, roundsChart234, r=>Number(r.leaderChanges??0)),
            }));
    }, [chart3LeaderData, algosLeader]);

    // 图4数据格式化【高亮-2026-03-15 23:40:00】
    const chart4Series = useMemo(() => {
        return (chart4CostData || [])
            .filter(as => algosCost.includes(as.algo))
            .map(as => ({
                algo: as.algo,
                data: alignPoints(as.points, roundsChart234, r=>Number(r.nodeCost??0)),
            }));
    }, [chart4CostData, algosCost]);

    // ======================= 【高亮-2026-03-22】新增：图5 时延数据格式化 =======================
    const chart5Series = useMemo(() => {
        return (chart5LatencyData || [])
            .filter(as => algosLatency.includes(as.algo))
            .map(as => ({
                algo: as.algo,
                // 时延在后端模拟中是按照 1~20 轮生成的，所以使用 roundsChart1 (1-20) 轴
                data: alignPoints(as.points, roundsChart1, r=>Number(r.latency??0)),
            }));
    }, [chart5LatencyData, algosLatency]);

    const selectedLabel = (a) => algoNames.find(x => x.value === a)?.label || a;

    return (
        <Box sx={{ my: 4, mx: "auto", maxWidth: 800 }}>
            <Paper sx={{ p: 3 }}>
                <Typography variant="h5" mb={2}>算法撮合性能特性对比</Typography>

                {/* 图1：挂单成功率 */}
                <Box sx={{ mb: 4 }}>
                    <AlgoMultiSelect
                        label="选择算法（挂单成功率）"
                        value={algosSuccess}
                        onChange={e => setAlgosSuccess(typeof e.target.value === "string" ? e.target.value.split(',') : e.target.value)}
                    />
                    {loading1 ? <Typography>数据加载中...</Typography> : (
                        <>
                            {errMsg1 && <Typography color="error">{errMsg1}</Typography>}
                            <Typography variant="subtitle1" mt={1} gutterBottom>各轮次挂单成功率（%）（共识轮数1-20）</Typography>
                            {chart1Series.length > 0 ? (
                                <LineChart
                                    series={chart1Series.map(s => ({
                                        data: s.data,
                                        label: selectedLabel(s.algo),
                                        color: colors[s.algo] || undefined,
                                    }))}
                                    xAxis={[{label:"共识轮数",data:roundsChart1}]}
                                    yAxis={[{label:"挂单成功率(%)"}]}
                                    width={680}
                                    height={300}
                                />
                            ):<Typography color="text.secondary" sx={{ py: 2 }}>暂无统计数据</Typography>}
                        </>
                    )}
                </Box>

                {/* 图2：错误节点率 */}
                <Box sx={{ mb: 4 }}>
                    <AlgoMultiSelect
                        label="选择算法（错误节点使用率）"
                        value={algosError}
                        onChange={e => setAlgosError(typeof e.target.value === "string" ? e.target.value.split(',') : e.target.value)}
                    />
                    {loading2 ? <Typography>数据加载中...</Typography> : (
                        <>
                            {errMsg2 && <Typography color="error">{errMsg2}</Typography>}
                            <Typography variant="subtitle1" mt={1} gutterBottom>
                                错误节点参与共识率（%）（共识轮数100-1000）
                            </Typography>
                            {chart2Series.length > 0 ? (
                                <LineChart
                                    series={chart2Series.map(s=>({
                                        data: s.data,
                                        label: selectedLabel(s.algo),
                                        color: colors[s.algo] || undefined,
                                    }))}
                                    xAxis={[{label:"共识轮数",data:roundsChart234}]}
                                    yAxis={[{label:"错误节点参与共识率(%)"}]}
                                    width={680}
                                    height={300}
                                />
                            ):<Typography color="text.secondary" sx={{ py: 2 }}>暂无错误节点参与率数据</Typography>}
                        </>
                    )}
                </Box>

                {/* 图3：主节点切换次数 */}
                <Box sx={{ mb: 4 }}>
                    <AlgoMultiSelect
                        label="选择算法（主节点切换次数）"
                        value={algosLeader}
                        onChange={e => setAlgosLeader(typeof e.target.value === "string" ? e.target.value.split(',') : e.target.value)}
                    />
                    {loading3 ? <Typography>数据加载中...</Typography> : (
                        <>
                            {errMsg3 && <Typography color="error">{errMsg3}</Typography>}
                            <Typography variant="subtitle1" mt={1} gutterBottom>
                                主节点切换次数（共识轮数100-1000）
                            </Typography>
                            {chart3Series.length > 0 ? (
                                <LineChart
                                    series={chart3Series.map(s=>({
                                        data: s.data,
                                        label: selectedLabel(s.algo),
                                        color: colors[s.algo] || undefined,
                                    }))}
                                    xAxis={[{label:"共识轮数",data:roundsChart234}]}
                                    yAxis={[{label:"主节点切换次数"}]}
                                    width={680}
                                    height={300}
                                />
                            ):<Typography color="text.secondary" sx={{ py: 2 }}>暂无主节点切换次数数据</Typography>}
                        </>
                    )}
                </Box>

                {/* 图4：平均节点开销【高亮-2026-03-15 23:40:00】 */}
                <Box sx={{ mb: 4 }}>
                    <AlgoMultiSelect
                        label="选择算法（平均节点开销）"
                        value={algosCost}
                        onChange={e => setAlgosCost(typeof e.target.value === "string" ? e.target.value.split(',') : e.target.value)}
                    />
                    {loading4 ? <Typography>数据加载中...</Typography> : (
                        <>
                            {errMsg4 && <Typography color="error">{errMsg4}</Typography>}
                            <Typography variant="subtitle1" mt={1} gutterBottom>
                                平均节点开销（共识轮数100-1000）
                            </Typography>
                            {chart4Series.length > 0 ? (
                                <LineChart
                                    series={chart4Series.map(s=>({
                                        data: s.data,
                                        label: selectedLabel(s.algo),
                                        color: colors[s.algo] || undefined,
                                    }))}
                                    xAxis={[{label:"共识轮数",data:roundsChart234}]}
                                    yAxis={[{label:"平均节点开销"}]}
                                    width={680}
                                    height={300}
                                />
                            ):<Typography color="text.secondary" sx={{ py: 2 }}>暂无平均节点开销数据</Typography>}
                        </>
                    )}
                </Box>

                {/* ======================= 【高亮-2026-03-22】新增：图5 交易平均时延 (置于最上方以突出优势) ======================= */}
                {/* ======================= 【高亮-2026-03-22 10:15】修改：图5 交易平均时延，去除原有的蓝色背景、内边距与边框，使底色恢复白色以对齐其他图表 ======================= */}
                <Box sx={{ mb: 6 }}>
                    <Typography variant="h6" color="primary" gutterBottom>⏱️ 核心优势: 交易平均时延对比 (Consensus Latency)</Typography>
                    <Typography variant="body2" color="text.secondary" gutterBottom sx={{ mb: 2 }}>
                    </Typography>
                    <AlgoMultiSelect
                        label="选择算法（交易平均时延）"
                        value={algosLatency}
                        onChange={e => setAlgosLatency(typeof e.target.value === "string" ? e.target.value.split(',') : e.target.value)}
                    />
                    {loading5 ? <Typography>数据加载中...</Typography> : (
                        <>
                            {errMsg5 && <Typography color="error">{errMsg5}</Typography>}
                            <Typography variant="subtitle1" mt={1} gutterBottom>各轮次平均共识时延（单位：毫秒 ms）（共识轮数1-20）</Typography>
                            {chart5Series.length > 0 ? (
                                <LineChart
                                    series={chart5Series.map(s => ({
                                        data: s.data,
                                        label: selectedLabel(s.algo),
                                        color: colors[s.algo] || undefined,
                                        curve: "linear"
                                    }))}
                                    xAxis={[{label:"共识轮数", data:roundsChart1}]}
                                    yAxis={[{label:"平均时延(ms)"}]}
                                    width={680}
                                    height={300}
                                />
                            ):<Typography color="text.secondary" sx={{ py: 2 }}>暂无时延统计数据</Typography>}
                        </>
                    )}
                </Box>
            </Paper>
        </Box>
    );
}