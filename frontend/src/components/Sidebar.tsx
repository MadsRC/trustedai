// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

import { Link, useLocation } from "react-router-dom";
import { useAuth } from "../hooks/useAuth";
import { User } from "lucide-react";

interface MenuItem {
  name: string;
  href: string;
}

interface MenuSection {
  title: string;
  items: MenuItem[];
}

function Sidebar() {
  const location = useLocation();
  const { user } = useAuth();

  const menuSections: MenuSection[] = [
    {
      title: "Overview",
      items: [{ name: "Dashboard", href: "/" }],
    },
    {
      title: "Inference Gateway",
      items: [
        { name: "Models", href: "/models" },
        { name: "Credentials", href: "/credentials" },
      ],
    },
    {
      title: "Identity & Access Management",
      items: [
        { name: "Organizations", href: "/organizations" },
        { name: "Users", href: "/users" },
        { name: "API Keys", href: "/api-keys" },
      ],
    },
    {
      title: "System",
      items: [
        { name: "Analytics", href: "/analytics" },
        { name: "Settings", href: "/settings" },
      ],
    },
  ];

  return (
    <div className="w-64 bg-gray-800 text-white flex flex-col">
      <nav className="mt-4 flex-1">
        {menuSections.map((section, sectionIndex) => (
          <div key={section.title} className={sectionIndex > 0 ? "mt-6" : ""}>
            <div className="px-4 py-2">
              <h3 className="text-xs font-semibold text-gray-400 uppercase tracking-wider">
                {section.title}
              </h3>
            </div>
            <ul className="space-y-1">
              {section.items.map((item) => {
                const isActive = location.pathname === item.href;
                return (
                  <li key={item.name}>
                    <Link
                      to={item.href}
                      className={`block px-4 py-2 text-sm transition-colors ${
                        isActive
                          ? "bg-gray-700 text-white"
                          : "hover:bg-gray-700"
                      }`}
                    >
                      {item.name}
                    </Link>
                  </li>
                );
              })}
            </ul>
          </div>
        ))}
      </nav>

      <div className="border-t border-gray-700 p-4">
        <Link
          to="/profile"
          className={`flex items-center space-x-3 px-3 py-2 rounded-md text-sm transition-colors ${
            location.pathname === "/profile"
              ? "bg-gray-700 text-white"
              : "hover:bg-gray-700"
          }`}
        >
          <User size={16} />
          <div className="flex-1 min-w-0">
            <div className="truncate font-medium">
              {user?.name || "Profile"}
            </div>
            <div className="truncate text-xs text-gray-400">{user?.email}</div>
          </div>
        </Link>
      </div>
    </div>
  );
}

export default Sidebar;
