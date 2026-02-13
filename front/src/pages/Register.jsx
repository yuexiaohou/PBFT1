import React, { useState } from "react";
import { useNavigate } from "react-router-dom";
import { register } from "../api";
import { Button, TextField, Typography, Paper, Box, Alert, Link as MuiLink } from "@mui/material";

export default function Register() {
    const [username, setUsername] = useState("");
    const [password, setPassword] = useState("");
    const [msg, setMsg] = useState("");
    const navigate = useNavigate();

    const handleRegister = async () => {
        try {
            await register(username, password);
            setMsg("注册成功");
            setTimeout(() => navigate("/login"), 1000);
        } catch {
            setMsg("注册失败");
        }
    };

    return (
        <Box sx={{ display: "flex", minHeight: "80vh", alignItems: "center", justifyContent: "center" }}>
            <Paper elevation={3} sx={{ p: 4, minWidth: 320 }}>
                <Typography variant="h5" mb={2} align="center">注册</Typography>
                <TextField
                    label="用户名"
                    value={username}
                    onChange={e => setUsername(e.target.value)}
                    fullWidth
                    margin="normal"
                />
                <TextField
                    label="密码"
                    type="password"
                    value={password}
                    onChange={e => setPassword(e.target.value)}
                    fullWidth
                    margin="normal"
                />
                <Button variant="contained" color="primary" fullWidth sx={{ mt: 2 }} onClick={handleRegister}>注册</Button>
                {msg && <Alert severity={msg === "注册成功" ? "success" : "error"} sx={{ mt: 2 }}>{msg}</Alert>}
                <Typography sx={{ mt: 2 }} align="center">
                    <MuiLink href="/login" underline="hover">已有账号？登录</MuiLink>
                </Typography>
            </Paper>
        </Box>
    );
}