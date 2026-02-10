import React, { useEffect, useState } from "react";
import { getTradeHistory } from "../api";

export default function History() {
    const [records, setRecords] = useState([]);

    useEffect(() => {
        async function fetchRecords() {
            try {
                const { data } = await getTradeHistory();
                setRecords(data.records || []);
            } catch {
                setRecords([]);
            }
        }
        fetchRecords();
    }, []);

    return (
        <div>
            <h2>历史交易记录</h2>
            <table border="1">
                <thead>
                <tr>
                    <th>类型</th>
                    <th>数量</th>
                    <th>时间</th>
                    <th>状态</th>
                </tr>
                </thead>
                <tbody>
                {records.map((r, i) => (
                    <tr key={i}>
                        <td>{r.type}</td>
                        <td>{r.amount}</td>
                        <td>{r.time}</td>
                        <td>{r.status}</td>
                    </tr>
                ))}
                </tbody>
            </table>
        </div>
    );
}
