import React, { useState } from "react";
import { useNavigate } from "react-router-dom";
import { register } from "../api";

export default function Register() {
    const [username, setUsername] = useState("");
    const [password, setPassword] = useState("");
    const [msg, setMsg] = useState("");
    const navigate = useNavigate();

    const handleRegister = async () => {
        try {
            await register(username, password);
            setMsg("注册成功");
            navigate("/login");
        } catch {
            setMsg("注册失败");
        }
    };
    return (
        <div>
            <h2>注册</h2>
            <input value={username} onChange={e => setUsername(e.target.value)} placeholder="用户名" />
            <input value={password} type="password" onChange={e => setPassword(e.target.value)} placeholder="密码" />
            <button onClick={handleRegister}>注册</button>
            <p>{msg}</p>
            <p><a href="/login">已有账号？登录</a></p>
        </div>
    );
}