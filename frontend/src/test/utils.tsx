// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

import {
  render as rtlRender,
  type RenderOptions,
} from "@testing-library/react";
import { BrowserRouter } from "react-router-dom";
import { type ReactElement } from "react";

// Custom render function that includes common providers
const customRender = (
  ui: ReactElement,
  options?: Omit<RenderOptions, "wrapper">,
) => {
  const AllTheProviders = ({ children }: { children: React.ReactNode }) => {
    return <BrowserRouter>{children}</BrowserRouter>;
  };

  return rtlRender(ui, { wrapper: AllTheProviders, ...options });
};

// Re-export everything from React Testing Library
export * from "@testing-library/react";
// Export our custom render function
export { customRender as render };
