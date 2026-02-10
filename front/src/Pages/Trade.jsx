import React, { useState } from "react";
import { trade } from "../api";

export default function Trade() {
    const [type, setType] = useState("buy");
    const [amount, setAmount] = useState(0);
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
        <div>
            <h2>买入/卖出</h2>
            <select value={type} onChange={e => setType(e.target.value)}>
                <option value="buy">买入</option>
                <option value="sell">卖出</option>
            </select>
            <input type="number" value={amount} onChange={e => setAmount(e.target.value)} placeholder="数量" />
            <button onClick={handleTrade}>提交交易</button>
            <p>{msg}</p>
        </div>
    );
}