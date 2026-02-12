import React, { useEffect, useState } from "react";
import { getBalance, deposit } from "../api";

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
        <div>
            <h2>账户总览</h2>
            <p>当前余额：{balance}</p>
            <input type="number" value={amount} onChange={e => setAmount(e.target.value)} placeholder="充值金额" />
            <button onClick={handleDeposit}>充值</button>
            <p>{msg}</p>
        </div>
    );
}