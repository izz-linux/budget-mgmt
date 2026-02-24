/**
 * Parse a date string as a local date.
 * Expects a "YYYY-MM-DD" date, but will also accept full ISO strings
 * like "YYYY-MM-DDTHH:mm:ssZ" and use only the date portion.
 * Using new Date('YYYY-MM-DD') interprets the string as UTC midnight,
 * which can cause off-by-one-day errors when converted to local time.
 */
export function parseLocalDate(dateStr: string): Date {
  const [datePart] = dateStr.split('T');
  const [year, month, day] = datePart.split('-').map(Number);
  return new Date(year, month - 1, day);
}

/**
 * Format a date string (YYYY-MM-DD or full ISO) for display, e.g., "Mar 15".
 */
export function formatShortDate(dateStr: string): string {
  const d = parseLocalDate(dateStr);
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
}

/**
 * Format a date string (YYYY-MM-DD or full ISO) as month/year, e.g., "Mar 2026".
 */
export function formatMonthYear(dateStr: string): string {
  const d = parseLocalDate(dateStr);
  return d.toLocaleDateString('en-US', { month: 'short', year: 'numeric' });
}
