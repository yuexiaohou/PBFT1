import axios from "axios";

// 新增：获取用户名+带用户名拼接函数
const username = () => localStorage.getItem("username") || "";
const withUsername = (url) => {
    const u = username();
    if (!u) return url;
    return url + (url.includes("?") ? "&" : "?") + "username=" + encodeURIComponent(u);
};

// 登录、注册
export const login = (username, password) =>
    axios.post("/api/login", { username, password });

export const register = (username, password) =>
    axios.post("/api/register", { username, password });

// 账户相关，只保留一份新版！
export const getBalance = () => axios.get(withUsername("/api/account/balance"));
export const deposit = (amount) => axios.post(withUsername("/api/account/deposit"), { amount });
export const trade = (type, amount) => axios.post(withUsername("/api/trade"), { type, amount });
export const getTradeHistory = () => axios.get(withUsername("/api/trade/history"));