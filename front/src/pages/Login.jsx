import React, { useState } from "react";
import { useNavigate } from "react-router-dom";
import { login } from "../api";
import { Button, TextField, Typography, Paper, Box, Alert, Link as MuiLink } from "@mui/material";

export default function Login() {
    const [username, setUsername] = useState("");
    const [password, setPassword] = useState("");
    const [msg, setMsg] = useState("");
    const navigate = useNavigate();

// === 修改开始 ===
    const handleLogin = async () => {
        try {
            const res = await login(username, password);
            // ======= 高亮：打印API返回内容 =======
            console.log('API返回', res);
            localStorage.setItem("token", res.data.token || "dummy");
            localStorage.setItem("username", username); // ======= 新增: 保存用户名 =======
            // ======= 高亮：打印token写入效果 =======
            console.log('token after set:', localStorage.getItem("token"));
            navigate("/dashboard");
        } catch (e) {
            console.log(e);
            setMsg("登录失败，请重试");
        }
    };
    // === 修改结束 ===

    return (
        <Box sx={{ display: "flex", minHeight: "80vh", alignItems: "center", justifyContent: "center" }}>
            <Paper elevation={3} sx={{ p: 4, minWidth: 320 }}>
                <Typography variant="h5" mb={2} align="center">登录</Typography>
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
                <Button variant="contained" color="primary" fullWidth sx={{ mt: 2 }} onClick={handleLogin}>登录</Button>
                {msg && <Alert severity="error" sx={{ mt: 2 }}>{msg}</Alert>}
                <Typography sx={{ mt: 2 }} align="center">
                    <MuiLink href="/register" underline="hover">还没有账号？注册</MuiLink>
                </Typography>
            </Paper>
        </Box>
    );
}