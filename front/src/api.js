import axios from "axios";

export const login = (username, password) =>
    axios.post("/api/login", { username, password });

export const register = (username, password) =>
    axios.post("/api/register", { username, password });

export const getBalance = () => axios.get("/api/account/balance");

export const deposit = (amount) => axios.post("/api/account/deposit", { amount });

export const trade = (type, amount) => axios.post("/api/trade", { type, amount });

export const getTradeHistory = () => axios.get("/api/trade/history");