import React, { useState } from "react";
import { useNavigate } from "react-router-dom";
import { login } from "../api";
import { Button, TextField, Typography, Paper, Box, Alert, Link as MuiLink } from "@mui/material";

export default function Login() {
    const [username, setUsername] = useState("");
    const [password, setPassword] = useState("");
    const [msg, setMsg] = useState("");
    const navigate = useNavigate();

    // ======================= 【UI 优化-2026-03-15】修改：强制跳转以刷新侧边栏状态 =======================
    const handleLogin = async () => {
        try {
            const res = await login(username, password);
            // ======= 高亮-2026-03-15：规范化 API 返回校验逻辑 =======
            // 兼容 token 直接返回或包含在 res.data.token 中的情况
            const token = res.data?.token || res.token;
            const status = res.data?.status || res.status;

            if (token || status === "success") {
                // 写入凭证
                localStorage.setItem("token", token || "dummy_token");
                localStorage.setItem("username", username);

                // ======= 高亮-2026-03-15：核心跳转逻辑优化 =======
                // 使用 window.location.href 代替 navigate("/dashboard")
                // 这样可以确保 App.js 重新挂载，Navigation 组件能立即看到 token 并显示侧边栏
                window.location.href = "/dashboard";
            } else {
                setMsg("用户名或密码错误");
            }
        } catch (e) {
            console.error("Login Error:", e);
            setMsg("登录失败，请检查网络或重试");
        }
    };
    // ======================= 【UI 优化-2026-03-15】修改结束 =======================

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
                <Button
                    variant="contained"
                    color="primary"
                    fullWidth
                    sx={{ mt: 2 }}
                    onClick={handleLogin}
                >
                    登录
                </Button>
                {msg && <Alert severity="error" sx={{ mt: 2 }}>{msg}</Alert>}
                <Typography sx={{ mt: 2 }} align="center">
                    <MuiLink href="/register" underline="hover">还没有账号？注册</MuiLink>
                </Typography>
            </Paper>
        </Box>
    );
}