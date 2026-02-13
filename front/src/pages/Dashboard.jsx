import React, { useEffect, useState } from "react";
import { getBalance, deposit } from "../api";
import { Typography, Paper, Box, Button, TextField, Alert } from "@mui/material";
import AccountBalanceWalletIcon from "@mui/icons-material/AccountBalanceWallet";

export default function Dashboard() {
    const [balance, setBalance] = useState(0);
    const [msg, setMsg] = useState("");
    const [amount, setAmount] = useState(100);

    useEffect(() => {
        fetchBalance();
    }, []);

    const fetchBalance = async () => {
        try {
            const { data } = await getBalance();
            setBalance(data.balance);
        } catch {
            setMsg("获取余额失败");
        }
    };
    const handleDeposit = async () => {
        try {
            await deposit(Number(amount));
            setMsg("充值成功");
            fetchBalance();
        } catch {
            setMsg("充值失败");
        }
    };

    return (
        <Box sx={{ display: "flex", minHeight: "60vh", alignItems: "center", justifyContent: "center" }}>
            <Paper elevation={3} sx={{ p: 4, minWidth: 350 }}>
                <Box display="flex" alignItems="center" mb={2}>
                    <AccountBalanceWalletIcon fontSize="large" color="primary" sx={{ mr: 1 }} />
                    <Typography variant="h5">账户总览</Typography>
                </Box>
                <Typography variant="body1" mb={2}>
                    当前余额：<strong>{balance}</strong>
                </Typography>
                <Box display="flex" alignItems="center" gap={1}>
                    <TextField
                        type="number"
                        label="充值金额"
                        size="small"
                        value={amount}
                        onChange={e => setAmount(e.target.value)}
                    />
                    <Button variant="contained" onClick={handleDeposit}>充值</Button>
                </Box>
                {msg && <Alert sx={{ mt: 2 }} severity={msg === "充值成功" ? "success" : "error"}>{msg}</Alert>}
            </Paper>
        </Box>
    );
}