import axios from "axios";

// ========== 【高亮】新增 begin ==========
const username = () => localStorage.getItem("username") || "";
const withUsername = (url) => {
    const u = username();
    if (!u) return url; // 没用户直接不拼接
    return url + (url.includes("?") ? "&" : "?") + "username=" + encodeURIComponent(u);
};
// ========== 【高亮】新增 end ==========

export const login = (username, password) =>
    axios.post("/api/login", { username, password });

export const register = (username, password) =>
    axios.post("/api/register", { username, password });

// ========== 【高亮】必须替换为如下 ==========
export const getBalance = () => axios.get(withUsername("/api/account/balance"));

export const deposit = (amount) => axios.post(withUsername("/api/account/deposit"), { amount });

export const trade = (type, amount) => axios.post(withUsername("/api/trade"), { type, amount });

export const getTradeHistory = () => axios.get(withUsername("/api/trade/history"));
// ========== 【高亮】必须替换为如下 ==========

export const getPBFTResult = () => axios.get("/api/pbft/result");
export const getPBFTBlock = () => axios.get("/api/pbft/block");
export const getPBFTBlockById = (id) => axios.get(`/api/pbft/block?id=${id}`);