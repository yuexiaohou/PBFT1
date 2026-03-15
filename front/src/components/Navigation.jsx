import { Link, useLocation } from "react-router-dom";
import { Box, Typography, Button, Divider, List, ListItem, ListItemIcon, ListItemText } from "@mui/material";
import DashboardIcon from "@mui/icons-material/Dashboard";
import SwapHorizIcon from "@mui/icons-material/SwapHoriz";
import HistoryIcon from "@mui/icons-material/History";
import InsightsIcon from "@mui/icons-material/Insights";
import SecurityIcon from "@mui/icons-material/Security";
import StorageIcon from "@mui/icons-material/Storage";
import LogoutIcon from "@mui/icons-material/Logout";

// ======================= 【UI 优化-2026-03-15】修改：改为固定侧边导航结构 =======================
export default function Navigation() {
    const location = useLocation();
    const isAuthenticated = !!localStorage.getItem("token");
    const username = localStorage.getItem("username") || "Guest";

    const logout = () => {
        localStorage.clear();
        window.location.href = "/login";
    };

    if (!isAuthenticated) return null; // 未登录时不显示侧边栏

    const menuItems = [
        { label: "账户总览", path: "/dashboard", icon: <DashboardIcon /> },
        { label: "实时交易", path: "/trade", icon: <SwapHorizIcon /> },
        { label: "历史记录", path: "/history", icon: <HistoryIcon /> },
        { label: "PBFT共识", path: "/pbft", icon: <SecurityIcon /> },
        { label: "区块溯源", path: "/blocksearch", icon: <StorageIcon /> },
        { label: "撮合统计", path: "/matchcharts", icon: <InsightsIcon /> },
        { label: "性能特性", path: "/performancecharts", icon: <InsightsIcon /> },
    ];

    return (
        <Box sx={{
            width: 240,
            height: "100vh",
            backgroundColor: "#1a2a3a",
            color: "white",
            display: "flex",
            flexDirection: "column",
            position: "fixed",
            left: 0,
            top: 0
        }}>
            <Box sx={{ p: 3, textAlign: "center" }}>
                <Typography variant="h6" fontWeight="bold" color="#3498db">PBFT System</Typography>
                <Typography variant="caption" color="gray">User: {username}</Typography>
            </Box>
            <Divider sx={{ bgcolor: "#2c3e50" }} />
            <List sx={{ flex: 1 }}>
                {menuItems.map((item) => (
                    <ListItem
                        button
                        key={item.path}
                        component={Link}
                        to={item.path}
                        selected={location.pathname === item.path}
                        sx={{
                            "&.Mui-selected": { bgcolor: "#34495e", borderLeft: "4px solid #3498db" },
                            "&:hover": { bgcolor: "#2c3e50" },
                            color: "white"
                        }}
                    >
                        <ListItemIcon sx={{ color: "inherit" }}>{item.icon}</ListItemIcon>
                        <ListItemText primary={item.label} />
                    </ListItem>
                ))}
            </List>
            <Box sx={{ p: 2 }}>
                <Button
                    fullWidth
                    variant="contained"
                    color="error"
                    startIcon={<LogoutIcon />}
                    onClick={logout}
                >
                    退出登录
                </Button>
            </Box>
        </Box>
    );
}