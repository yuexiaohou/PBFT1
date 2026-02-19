import axios from "axios";

// ====== 高亮修改区域开始 ======
// 新增：自动获取localStorage中的用户名
const username = () => localStorage.getItem("username") || "";

// 新增：接口url自动拼接username参数
const withUsername = (url) => {
    const u = username();
    if (!u) return url;
    return url + (url.includes("?") ? "&" : "?") + "username=" + encodeURIComponent(u);
};
// ====== 高亮修改区域结束 ======

// 登录与注册接口，无需更动
export const login = (username, password) =>
    axios.post("/api/login", { username, password });

export const register = (username, password) =>
    axios.post("/api/register", { username, password });

// ====== 高亮修改区域开始 ======
// 账户相关接口都要带上username参数
export const getBalance = () => axios.get(withUsername("/api/account/balance"));

export const deposit = (amount) => axios.post(withUsername("/api/account/deposit"), { amount });

export const trade = (type, amount) => axios.post(withUsername("/api/trade"), { type, amount });

export const getTradeHistory = () => axios.get(withUsername("/api/trade/history"));
// ====== 高亮修改区域结束 ======