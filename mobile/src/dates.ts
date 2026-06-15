// Date helpers shared with the web client (see internal/embed/html/bases/form.js
// and internal/implementation/v1/state.go).
//
// The server groups data into a fiscal year: calendar months Jan–Sep belong to
// fiscal year Y, while Oct–Dec belong to fiscal year Y+1. The /state/{year} and
// /note/{year} endpoints are keyed by that fiscal year.

export const MONTH_NAMES = [
  'January', 'February', 'March', 'April', 'May', 'June',
  'July', 'August', 'September', 'October', 'November', 'December',
];

export const WEEKDAYS = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'];

// month is 1-12. Returns the fiscal year that contains this calendar month.
export function fiscalYear(year: number, month: number): number {
  return month <= 9 ? year : year + 1;
}

export interface ViewMonth {
  year: number; // calendar year
  month: number; // 1-12
}

export function thisMonth(): ViewMonth {
  const now = new Date();
  return { year: now.getFullYear(), month: now.getMonth() + 1 };
}

export function addMonths(view: ViewMonth, delta: number): ViewMonth {
  // month is 1-12; convert to 0-based, shift, normalise.
  let m = view.month - 1 + delta;
  let y = view.year;
  while (m < 0) {
    m += 12;
    y -= 1;
  }
  while (m > 11) {
    m -= 12;
    y += 1;
  }
  return { year: y, month: m + 1 };
}

export function daysInMonth(year: number, month: number): number {
  // month is 1-12. Day 0 of the next month is the last day of this month.
  return new Date(year, month, 0).getDate();
}

// A single slot in the 7-column calendar grid. `day` is null for padding cells
// that fall outside the current month.
export interface GridCell {
  day: number | null;
  isToday: boolean;
}

// Build a Monday-first grid of weeks for the given month.
export function monthGrid(year: number, month: number): GridCell[][] {
  const total = daysInMonth(year, month);
  // JS getDay(): 0=Sun..6=Sat. Convert to Monday-first index 0=Mon..6=Sun.
  const firstWeekday = (new Date(year, month - 1, 1).getDay() + 6) % 7;

  const now = new Date();
  const isCurrentMonth =
    now.getFullYear() === year && now.getMonth() + 1 === month;
  const todayDate = now.getDate();

  const cells: GridCell[] = [];
  for (let i = 0; i < firstWeekday; i++) {
    cells.push({ day: null, isToday: false });
  }
  for (let d = 1; d <= total; d++) {
    cells.push({ day: d, isToday: isCurrentMonth && d === todayDate });
  }
  while (cells.length % 7 !== 0) {
    cells.push({ day: null, isToday: false });
  }

  const weeks: GridCell[][] = [];
  for (let i = 0; i < cells.length; i += 7) {
    weeks.push(cells.slice(i, i + 7));
  }
  return weeks;
}

export function formatMonthYear(view: ViewMonth): string {
  return `${MONTH_NAMES[view.month - 1]} ${view.year}`;
}
