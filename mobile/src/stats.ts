import { MonthDays } from './api';
import { AttendanceState, isOffice, isWorkDay } from './states';

export interface Stats {
  office: number; // days present in office (real + scheduled)
  total: number; // tracked work days (WFH + office, real + scheduled)
  percent: number; // office / total * 100
}

function empty(): Stats {
  return { office: 0, total: 0, percent: 0 };
}

function withPercent(office: number, total: number): Stats {
  return {
    office,
    total,
    percent: total > 0 ? (office / total) * 100 : 0,
  };
}

export function monthStats(days: MonthDays): Stats {
  let office = 0;
  let total = 0;
  for (const state of Object.values(days)) {
    if (isOffice(state)) office++;
    if (isWorkDay(state)) total++;
  }
  return withPercent(office, total);
}

// Aggregates every month in a fetched fiscal year.
export function yearStats(year: Record<number, MonthDays>): Stats {
  let office = 0;
  let total = 0;
  for (const days of Object.values(year)) {
    const s = monthStats(days);
    office += s.office;
    total += s.total;
  }
  return total === 0 ? empty() : withPercent(office, total);
}

export function formatPercent(percent: number): string {
  // Two decimals to match the web summary table.
  return `${percent.toFixed(2)}%`;
}

export interface TargetProgress extends Stats {
  // More office days needed this month to meet the target, projected over the
  // remaining untracked weekdays. 0 once the target is met.
  needed: number;
}

// Progress for the viewed month against the attendance target, mirroring the
// web form page: the month-end work-day count is what's tracked so far plus
// remaining untracked weekdays (today included), assumed to become work days.
export function targetProgress(
  days: MonthDays,
  targetPercent: number,
  year: number,
  month: number, // 1-12
  now: Date = new Date(),
): TargetProgress {
  const tracked = monthStats(days);

  let projectedTotal = tracked.total;
  const startOfToday = new Date(now.getFullYear(), now.getMonth(), now.getDate());
  const daysInMonth = new Date(year, month, 0).getDate();
  for (let day = 1; day <= daysInMonth; day++) {
    if ((days[day] ?? AttendanceState.Untracked) !== AttendanceState.Untracked) {
      continue; // already tracked
    }
    const date = new Date(year, month - 1, day);
    if (date < startOfToday) continue; // past untracked days don't count
    const dow = date.getDay();
    if (dow === 0 || dow === 6) continue; // weekends
    projectedTotal++;
  }

  const needed = Math.max(
    0,
    Math.ceil((targetPercent / 100) * projectedTotal) - tracked.office,
  );
  return { ...tracked, needed };
}
