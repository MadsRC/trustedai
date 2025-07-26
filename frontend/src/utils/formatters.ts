// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

/**
 * Formats cost values from fractional cents to display dollars with appropriate precision.
 * For small amounts (< $0.01), shows up to 8 decimal places to avoid displaying $0.00.
 * For larger amounts, uses standard 2 decimal places.
 */
export const formatCost = (costCents: number | bigint): string => {
  const cost = typeof costCents === "bigint" ? Number(costCents) : costCents;
  const dollars = cost / 100;

  // If the cost is very small (less than $0.01), show up to 8 decimal places
  // to avoid displaying $0.00 for fractional cent costs, without premature rounding
  if (dollars < 0.01 && dollars > 0) {
    // Use toFixed(8) for higher precision, then remove trailing zeros
    const formatted = dollars.toFixed(8).replace(/\.?0+$/, "");
    return `$${formatted}`;
  }

  // For costs $0.01 and above, use standard 2 decimal places
  return `$${dollars.toFixed(2)}`;
};

/**
 * Formats large numbers with locale-specific thousand separators.
 */
export const formatNumber = (num: number | bigint): string => {
  const n = typeof num === "bigint" ? Number(num) : num;
  return n.toLocaleString();
};
