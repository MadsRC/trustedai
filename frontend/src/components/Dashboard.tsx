// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

import { Routes, Route } from "react-router-dom";
import Sidebar from "./Sidebar";
import TopPanel from "./TopPanel";
import BottomPanel from "./BottomPanel";
import MainArea from "./MainArea";
import Models from "./Models";
import Credentials from "./Credentials";
import Organizations from "./Organizations";
import Users from "./Users";
import APIKeys from "./APIKeys";
import Profile from "./Profile";
import Analytics from "./Analytics";
import Settings from "./Settings";

function Dashboard() {
  return (
    <div className="min-h-screen flex flex-col">
      <TopPanel />

      <div className="flex flex-1">
        <Sidebar />
        <Routes>
          <Route path="/" element={<MainArea />} />
          <Route path="/models" element={<Models />} />
          <Route path="/credentials" element={<Credentials />} />
          <Route path="/organizations" element={<Organizations />} />
          <Route path="/users" element={<Users />} />
          <Route path="/api-keys" element={<APIKeys />} />
          <Route path="/analytics" element={<Analytics />} />
          <Route path="/profile" element={<Profile />} />
          <Route path="/settings" element={<Settings />} />
        </Routes>
      </div>

      <BottomPanel />
    </div>
  );
}

export default Dashboard;
