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
 * Format a Date as "YYYY-MM-DD" using its LOCAL calendar fields.
 * Date.toISOString() converts to UTC first, which shifts the date by a day for
 * anyone west of Greenwich.
 */
export function toISODateString(d: Date): string {
  const year = String(d.getFullYear()).padStart(4, '0');
  const month = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

/**
 * Shift a "YYYY-MM-DD" date by a whole number of months, clamping the day to
 * the target month's length.
 *
 * Date.setMonth() overflows rather than clamping: from Jan 31, +1 month gives
 * Mar 3 (Feb 31 normalised), skipping February entirely — and from Mar 31,
 * -1 month gives that same Mar 3, moving the date *forward*. Shifting on the
 * 1st and then clamping avoids both.
 */
export function addMonths(dateStr: string, months: number): string {
  const d = parseLocalDate(dateStr);
  const day = d.getDate();

  // Shift the month on a safe day-of-month, so no overflow can occur.
  const shifted = new Date(d.getFullYear(), d.getMonth() + months, 1);

  // Day 0 of the following month == last day of the target month.
  const lastDay = new Date(shifted.getFullYear(), shifted.getMonth() + 1, 0).getDate();
  shifted.setDate(Math.min(day, lastDay));

  return toISODateString(shifted);
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
