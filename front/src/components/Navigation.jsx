import { Link } from "react-router-dom";

export default function Navigation() {
    const isAuthenticated = !!localStorage.getItem("token");
    const logout = () => {
        localStorage.removeItem("token");
        localStorage.removeItem("username"); // ======= 新增: 登出时删除用户名 =======
        window.location.href = "/login";
    };
    return (
        <nav>
            {isAuthenticated ? (
                <>
                    <Link to="/dashboard">账户</Link> |
                    <Link to="/trade">交易</Link> |
                    <Link to="/history">历史记录</Link> |
                    <Link to="/pbft">PBFT结果</Link> |
                    <Link to="/blocksearch">区块/溯源</Link> |
                    <Link to="/matchcharts">撮合统计</Link> |
                    <Link to="/performance">性能特性</Link> {/* === 2026-03-04 高亮新增 === */}
                    <button onClick={logout}>退出登录</button>
                </>
            ) : (
                <>
                    <Link to="/login">登录</Link> | <Link to="/register">注册</Link>
                </>
            )}
        </nav>
    );
}
