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

    return (
        <Box sx={{ display: "flex", minHeight: "60vh", alignItems: "center", justifyContent: "center" }}>
            <Paper elevation={3} sx={{ p: 4, minWidth: 500 }}>
                <Box display="flex" alignItems="center" mb={2}>
                    <HistoryIcon fontSize="large" color="primary" sx={{ mr: 1 }} />
                    <Typography variant="h5">历史交易记录</Typography>
                </Box>
                <TableContainer>
                    <Table>
                        <TableHead>
                            <TableRow>
                                <TableCell>类型</TableCell>
                                <TableCell>数量</TableCell>
                                <TableCell>时间</TableCell>
                                <TableCell>状态</TableCell>
                            </TableRow>
                        </TableHead>
                        <TableBody>
                            {records.map((r, i) => (
                                <TableRow key={i}>
                                    <TableCell>{r.type}</TableCell>
                                    <TableCell>{r.amount}</TableCell>
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