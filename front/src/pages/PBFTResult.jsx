import React, { useEffect, useState } from "react";
import { getPBFTResult, getPBFTBlock } from "../api";
import { Paper, Box, Typography, Button, Alert } from "@mui/material";

export default function PBFTResult() {
    const [result, setResult] = useState(null);
    const [block, setBlock] = useState(null);
    const [msg, setMsg] = useState("");
    const [loading, setLoading] = useState(false);

    const fetchResult = async () => {
        setLoading(true);
        try {
            const { data } = await getPBFTResult();
            setResult(data);
            setMsg("");
        } catch {
            setMsg("获取PBFT结果失败");
        }
        setLoading(false);
    };

    const fetchBlock = async () => {
        try {
            const { data } = await getPBFTBlock();
            setBlock(data);
        } catch {
            setBlock(null);
        }
    };

    useEffect(() => {
        fetchResult();
        fetchBlock();
    }, []);

    return (
        <Box sx={{ minHeight: "40vh", mt: 3 }}>
            <Paper sx={{ p: 3 }}>
                <Typography variant="h6">PBFT共识结果</Typography>
                <Button onClick={fetchResult} disabled={loading}>刷新状态</Button>
                {msg && <Alert sx={{ mt: 2 }} severity="error">{msg}</Alert>}
                {result && (
                    <Box mt={2}>
                        <Typography>交易ID: {result.txId}</Typography>
                        <Typography>状态: {result.status}</Typography>
                        <Typography>节点投票:
                            {result.validators && result.validators.map(v =>
                                <span key={v.id}>{v.id}: {v.vote} | </span>
                            )}
                        </Typography>
                    </Box>
                )}
                <Typography variant="h6" mt={3}>最新区块</Typography>
                {block && (
                    <Box>
                        <Typography>区块高度: {block.height}</Typography>
                        <Typography>区块时间: {block.timestamp}</Typography>
                        <Typography>已确认交易: {block.confirmedTxs}</Typography>
                    </Box>
                )}
            </Paper>
        </Box>
    );
}