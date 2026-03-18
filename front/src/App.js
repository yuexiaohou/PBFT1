import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import Login from "./pages/Login";
import Register from "./pages/Register";
import Dashboard from "./pages/Dashboard";
import Trade from "./pages/Trade";
import History from "./pages/History";
import Navigation from "./components/Navigation";
import PBFTResult from "./pages/PBFTResult";
import BlockSearch from "./pages/BlockSearch";
import MatchCharts from "./pages/MatchCharts";
import PerformanceCharts from "./pages/PerformanceCharts";
import {Box, CssBaseline} from "@mui/material";
import Forecast from "./pages/Forecast";

function App() {
    const isAuthenticated = !!localStorage.getItem("token"); // 仅做简单示例

    return (
        <BrowserRouter>
            <CssBaseline />
            <Box sx={{ display: "flex" }}> {/* 【UI 优化-2026-03-15】外层 Flex 容器 */}
                <Navigation />
                {/* 【UI 优化-2026-03-15】主内容区自适应，避开 240px 侧边栏 */}
                <Box component="main" sx={{
                    flexGrow: 1,
                    p: 3,
                    marginLeft: isAuthenticated ? "240px" : "0",
                    transition: "margin 0.3s",
                    minHeight: "100vh",
                    bgcolor: "#f5f5f5"
                }}>
                <Routes>
                  <Route path="/" element={<Navigate to={isAuthenticated ? "/dashboard" : "/login"} />} />
                  <Route path="/login" element={<Login />} />
                  <Route path="/register" element={<Register />} />
                  <Route path="/dashboard" element={isAuthenticated ? <Dashboard /> : <Navigate to="/login" />} />
                  <Route path="/trade" element={isAuthenticated ? <Trade /> : <Navigate to="/login" />} />
                  <Route path="/history" element={isAuthenticated ? <History /> : <Navigate to="/login" />} />
                  <Route path="/pbft" element={isAuthenticated ? <PBFTResult /> : <Navigate to="/login" />} />
                  <Route path="/blocksearch" element={isAuthenticated ? <BlockSearch /> : <Navigate to="/login" />} />
                  <Route path="/matchcharts" element={isAuthenticated ? <MatchCharts /> : <Navigate to="/login" />} />
                  <Route path="/performancecharts" element={isAuthenticated ? <PerformanceCharts /> : <Navigate to="/login" />} />
                  <Route path="/forecast" element={isAuthenticated ? <Forecast /> : <Navigate to="/login" />} />
                </Routes>
                </Box>
            </Box>
        </BrowserRouter>
    );
}
export default App;