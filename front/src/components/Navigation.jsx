import { Link } from "react-router-dom";

export default function Navigation() {
    const isAuthenticated = !!localStorage.getItem("token");
    const logout = () => {
        localStorage.removeItem("token");
        window.location.href = "/login";
    };
    return (
        <nav>
            {isAuthenticated ? (
                <>
                    <Link to="/dashboard">账户</Link> |
                    <Link to="/trade">交易</Link> |
                    <Link to="/history">历史记录</Link> |
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
