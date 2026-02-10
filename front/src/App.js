import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import Login from "./pages/Login";
import Register from "./pages/Register";
import Dashboard from "./pages/Dashboard";
import Trade from "./pages/Trade";
import History from "./pages/History";
import Navigation from "../components/Navigation";

function App() {
    const isAuthenticated = !!localStorage.getItem("token"); // 仅做简单示例

    return (
        <BrowserRouter>
            <Navigation />
            <Routes>
                <Route path="/" element={<Navigate to={isAuthenticated ? "/dashboard" : "/login"} />} />
                <Route path="/login" element={<Login />} />
                <Route path="/register" element={<Register />} />
                <Route path="/dashboard" element={isAuthenticated ? <Dashboard /> : <Navigate to="/login" />} />
                <Route path="/trade" element={isAuthenticated ? <Trade /> : <Navigate to="/login" />} />
                <Route path="/history" element={isAuthenticated ? <History /> : <Navigate to="/login" />} />
            </Routes>
        </BrowserRouter>
    );
}
export default App;