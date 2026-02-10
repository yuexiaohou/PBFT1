import axios from "axios";
axios.defaults.baseURL = "/api"; // 代理到你的后端接口

// 登录
export const login = (username, password) => axios.post("/login", { username, password });

// 注册
export const register = (username, password) => axios.post("/register", { username, password });

// 获取账户余额
export const getBalance = () => axios.get("/account/balance");

// 账户充值
export const deposit = (amount) => axios.post("/account/deposit", { amount });

// 买入/卖出下单
export const trade = (type, amount) => axios.post("/trade", { type, amount });

// 查询历史交易
export const getTradeHistory = () => axios.get("/trade/history");