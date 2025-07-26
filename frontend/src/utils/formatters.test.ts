// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

import { describe, it, expect } from "vitest";
import { formatCost, formatNumber } from "./formatters";

describe("formatCost", () => {
  describe("when cost is very small (< $0.01)", () => {
    it("should format 0.1 cents as $0.001", () => {
      expect(formatCost(0.1)).toBe("$0.001");
    });

    it("should format 0.5 cents as $0.005", () => {
      expect(formatCost(0.5)).toBe("$0.005");
    });

    it("should format 0.3 cents as $0.003", () => {
      expect(formatCost(0.3)).toBe("$0.003");
    });

    it("should format 0.25 cents as $0.0025", () => {
      expect(formatCost(0.25)).toBe("$0.0025");
    });

    it("should format 0.123456 cents as $0.00123456 (preserving precision)", () => {
      expect(formatCost(0.123456)).toBe("$0.00123456");
    });

    it("should format 0.00118 cents as $0.0000118 (the reported issue)", () => {
      expect(formatCost(0.00118)).toBe("$0.0000118");
    });

    it("should handle bigint values for small costs", () => {
      expect(formatCost(BigInt(0))).toBe("$0.00"); // BigInt(0) becomes 0, which is not > 0
    });
  });

  describe("when cost is $0.01 or above", () => {
    it("should format 1 cent as $0.01", () => {
      expect(formatCost(1)).toBe("$0.01");
    });

    it("should format 50 cents as $0.50", () => {
      expect(formatCost(50)).toBe("$0.50");
    });

    it("should format 150 cents as $1.50", () => {
      expect(formatCost(150)).toBe("$1.50");
    });

    it("should format 1000 cents as $10.00", () => {
      expect(formatCost(1000)).toBe("$10.00");
    });

    it("should format 999999 cents as $9999.99", () => {
      expect(formatCost(999999)).toBe("$9999.99");
    });

    it("should handle bigint values for larger costs", () => {
      expect(formatCost(BigInt(150))).toBe("$1.50");
    });
  });

  describe("edge cases", () => {
    it("should format 0 cents as $0.00", () => {
      expect(formatCost(0)).toBe("$0.00");
    });

    it("should format exactly 1 cent as $0.01", () => {
      expect(formatCost(1)).toBe("$0.01");
    });

    it("should format 0.99 cents as $0.0099", () => {
      expect(formatCost(0.99)).toBe("$0.0099");
    });

    it("should handle negative values (though unlikely in real usage)", () => {
      expect(formatCost(-50)).toBe("$-0.50");
    });
  });
});

describe("formatNumber", () => {
  it("should format regular numbers with commas", () => {
    expect(formatNumber(1000)).toBe("1,000");
    expect(formatNumber(1234567)).toBe("1,234,567");
  });

  it("should handle small numbers without commas", () => {
    expect(formatNumber(123)).toBe("123");
    expect(formatNumber(0)).toBe("0");
  });

  it("should handle bigint values", () => {
    expect(formatNumber(BigInt(1000000))).toBe("1,000,000");
  });

  it("should handle negative numbers", () => {
    expect(formatNumber(-1000)).toBe("-1,000");
  });
});
