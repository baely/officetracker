import { MonthDays } from './api';
import { isOffice, isWorkDay } from './states';

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
