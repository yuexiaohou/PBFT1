import React, { useState } from "react";
import { getPBFTBlockById } from "../api";
import { TextField, Button, Paper, Alert } from "@mui/material";

export default function BlockSearch() {
    const [blockId, setBlockId] = useState("");
    const [block, setBlock] = useState(null);
    const [msg, setMsg] = useState("");
    const handleSearch = async () => {
        try {
            const { data } = await getPBFTBlockById(blockId);
            setBlock(data);
            setMsg("");
        } catch {
            setMsg("区块或交易未找到");
        }
    };
    return (
        <Paper sx={{ p: 3 }}>
            <TextField label="区块高度或TxId" value={blockId} onChange={e => setBlockId(e.target.value)} />
            <Button onClick={handleSearch}>查询</Button>
            {msg && <Alert severity="error">{msg}</Alert>}
            {block && <pre>{JSON.stringify(block, null, 2)}</pre>}
        </Paper>
    );
}