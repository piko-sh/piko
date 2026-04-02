/**
 * Simple string utility module for E2E testing.
 * Tests that PPC components can import multi-line imports with @/ alias.
 */

/**
 * Converts a string to uppercase.
 * @param {string} str - String to convert
 * @returns {string} Uppercase string
 */
export function toUpperCase(str) {
  return str.toUpperCase();
}

/**
 * Converts a string to lowercase.
 * @param {string} str - String to convert
 * @returns {string} Lowercase string
 */
export function toLowerCase(str) {
  return str.toLowerCase();
}

/**
 * Capitalizes the first letter of a string.
 * @param {string} str - String to capitalize
 * @returns {string} Capitalized string
 */
export function capitalize(str) {
  if (!str) return str;
  return str.charAt(0).toUpperCase() + str.slice(1).toLowerCase();
}
