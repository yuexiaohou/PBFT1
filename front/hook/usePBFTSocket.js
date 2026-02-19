import { useEffect } from "react";

export function usePBFTSocket(onPbftMsg) {
    useEffect(() => {
        const ws = new WebSocket("ws://localhost:5000/pbft/ws");
        ws.onmessage = (event) => {
            const data = JSON.parse(event.data);
            onPbftMsg && onPbftMsg(data);
        };
        ws.onerror = () => {};
        return () => ws.close();
    }, [onPbftMsg]);
}