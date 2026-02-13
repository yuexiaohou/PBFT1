import React, { useState } from "react";
import { trade } from "../api";
import { Button, TextField, Typography, Paper, Box, Alert, MenuItem, Select, InputLabel, FormControl } from "@mui/material";
import SwapHorizIcon from "@mui/icons-material/SwapHoriz";

export default function Trade() {
    const [type, setType] = useState("buy");
    const [amount, setAmount] = useState("");
    const [msg, setMsg] = useState("");

    const handleTrade = async () => {
        try {
            await trade(type, Number(amount));
            setMsg("提交成功");
        } catch {
            setMsg("操作失败");
        }
    };

    return (
        <Box sx={{ display: "flex", minHeight: "60vh", alignItems: "center", justifyContent: "center" }}>
            <Paper elevation={3} sx={{ p: 4, minWidth: 350 }}>
                <Box display="flex" alignItems="center" mb={2}>
                    <SwapHorizIcon fontSize="large" color="primary" sx={{ mr: 1 }} />
                    <Typography variant="h5">买入/卖出</Typography>
                </Box>
                <FormControl fullWidth sx={{ mb: 2 }}>
                    <InputLabel id="type-label">类型</InputLabel>
                    <Select
                        labelId="type-label"
                        value={type}
                        label="类型"
                        onChange={e => setType(e.target.value)}
                    >
                        <MenuItem value="buy">买入</MenuItem>
                        <MenuItem value="sell">卖出</MenuItem>
                    </Select>
                </FormControl>
                <TextField
                    label="数量"
                    type="number"
                    value={amount}
                    onChange={e => setAmount(e.target.value)}
                    fullWidth
                    sx={{ mb: 2 }}
                />
                <Button variant="contained" color="primary" fullWidth onClick={handleTrade}>提交交易</Button>
                {msg && <Alert severity={msg === "提交成功" ? "success" : "error"} sx={{ mt: 2 }}>{msg}</Alert>}
            </Paper>
        </Box>
    );
}