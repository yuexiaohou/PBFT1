import React, { useState } from "react";
import { useNavigate } from "react-router-dom";
import { login } from "../api";

export default function Login() {
    const [username, setUsername] = useState("");
    const [password, setPassword] = useState("");
    const [msg, setMsg] = useState("");
    const navigate = useNavigate();

    const handleLogin = async () => {
        try {
            const res = await login(username, password);
            localStorage.setItem("token", res.data.token || "dummy");
            navigate("/dashboard");
        } catch {
            setMsg("登录失败，请重试");
        }
    };
    return (
        <div>
            <h2>登录</h2>
            <input value={username} onChange={e => setUsername(e.target.value)} placeholder="用户名" />
            <input value={password} type="password" onChange={e => setPassword(e.target.value)} placeholder="密码" />
            <button onClick={handleLogin}>登录</button>
            <p>{msg}</p>
            <p><a href="/register">还没有账号？注册</a></p>
        </div>
    );
}