// Date helpers. The server groups data into a per-user "tracking year": 12 months
// from a configurable start month (default October), labelled by the year it ends.
// /state/{year} and /note/{year} are keyed by that label (mirrors util.TrackingYear).

export const MONTH_NAMES = [
  'January', 'February', 'March', 'April', 'May', 'June',
  'July', 'August', 'September', 'October', 'November', 'December',
];

export const WEEKDAYS = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'];

// Matches model.DefaultTrackingYearStartMonth (October).
export const DEFAULT_TRACKING_YEAR_START_MONTH = 10;

export function normaliseStartMonth(startMonth: number): number {
  return startMonth >= 1 && startMonth <= 12
    ? startMonth
    : DEFAULT_TRACKING_YEAR_START_MONTH;
}

// month is 1-12. Returns the tracking-year label that contains this calendar
// month, given the user's start month. Mirrors util.TrackingYear in Go.
export function trackingYear(
  year: number,
  month: number,
  startMonth: number = DEFAULT_TRACKING_YEAR_START_MONTH,
): number {
  const sm = normaliseStartMonth(startMonth);
  if (sm === 1) return year;
  return month >= sm ? year + 1 : year;
}

// calendarYearForMonth maps a 1-indexed calendar month to the calendar year it
// falls in within the tracking year labelled fy (mirrors form.js / util.Go).
export function calendarYearForMonth(
  month: number,
  fy: number,
  startMonth: number = DEFAULT_TRACKING_YEAR_START_MONTH,
): number {
  const sm = normaliseStartMonth(startMonth);
  if (sm === 1) return fy;
  return month >= sm ? fy - 1 : fy;
}

// trackingMonthOrder returns a month's position (0-11) within the tracking year,
// so summary rows read start-month-first (mirrors form.js).
export function trackingMonthOrder(
  month: number,
  startMonth: number = DEFAULT_TRACKING_YEAR_START_MONTH,
): number {
  const sm = normaliseStartMonth(startMonth);
  return (month - sm + 12) % 12;
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
