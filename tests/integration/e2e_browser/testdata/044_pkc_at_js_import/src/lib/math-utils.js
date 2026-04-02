/**
 * Simple math utility module for E2E testing.
 * Tests that PPC components can import and use @/ JavaScript files.
 */

/**
 * Adds two numbers together.
 * @param {number} a - First number
 * @param {number} b - Second number
 * @returns {number} Sum of a and b
 */
export function add(a, b) {
  return a + b;
}

/**
 * Multiplies two numbers together.
 * @param {number} a - First number
 * @param {number} b - Second number
 * @returns {number} Product of a and b
 */
export function multiply(a, b) {
  return a * b;
}

/**
 * Formats a number with a prefix.
 * @param {string} prefix - Prefix to prepend
 * @param {number} value - Number to format
 * @returns {string} Formatted string
 */
export function formatWithPrefix(prefix, value) {
  return `${prefix}: ${value}`;
}
