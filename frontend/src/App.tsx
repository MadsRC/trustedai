// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

import React from "react";
import {
  BrowserRouter as Router,
  Routes,
  Route,
  Navigate,
} from "react-router-dom";
import Layout from "./components/Layout";
import Dashboard from "./pages/Dashboard";
import Users from "./pages/Users";
import Organizations from "./pages/Organizations";
import Tokens from "./pages/Tokens";
import Profile from "./pages/Profile";
import Login from "./pages/Login";
import SSOCallback from "./pages/SSOCallback";
import { setAuthData } from "./utils/auth";
import { AuthProvider, useAuth } from "./contexts/AuthContext";
import "./App.css";

const AppRoutes: React.FC = () => {
  const { isAuthenticated, isLoading, refreshAuth } = useAuth();

  // Show loading spinner while checking authentication
  if (isLoading) {
    return (
      <div
        style={{
          display: "flex",
          justifyContent: "center",
          alignItems: "center",
          height: "100vh",
          fontSize: "18px",
          backgroundColor: "#000",
          color: "#00ff00",
          fontFamily: "monospace",
        }}
      >
        Loading...
      </div>
    );
  }

  const handleLogin = (apiKey: string, username: string) => {
    setAuthData(apiKey, username);
    refreshAuth(); // Trigger auth state refresh after login
  };

  return (
    <Routes>
      <Route
        path="/login"
        element={
          isAuthenticated ? (
            <Navigate to="/" replace />
          ) : (
            <Login onLogin={handleLogin} />
          )
        }
      />
      <Route
        path="/sso/callback"
        element={<SSOCallback onLogin={handleLogin} />}
      />
      <Route
        path="/"
        element={
          isAuthenticated ? <Layout /> : <Navigate to="/login" replace />
        }
      >
        <Route index element={<Dashboard />} />
        <Route path="users" element={<Users />} />
        <Route path="organizations" element={<Organizations />} />
        <Route path="tokens" element={<Tokens />} />
        <Route path="profile" element={<Profile />} />
      </Route>
    </Routes>
  );
};

function App() {
  return (
    <AuthProvider>
      <Router>
        <AppRoutes />
      </Router>
    </AuthProvider>
  );
}

export default App;
