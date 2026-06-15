// Attendance states. These mirror the server enum in pkg/model/common.go.
export enum AttendanceState {
  Untracked = 0,
  WorkFromHome = 1,
  Office = 2,
  Other = 3,
  // "Scheduled" variants are projected from the user's weekly schedule by the
  // server (see mergeScheduleWithYear). They are shown lighter and are treated
  // as Untracked when the user starts tapping a day.
  ScheduledWorkFromHome = 4,
  ScheduledOffice = 5,
  ScheduledOther = 6,
}

// The four states a user can cycle a day through by tapping.
const CYCLE = [
  AttendanceState.Untracked,
  AttendanceState.WorkFromHome,
  AttendanceState.Office,
  AttendanceState.Other,
];

// Reduce a (possibly scheduled) state to its concrete base for cycling.
function baseState(state: AttendanceState): AttendanceState {
  switch (state) {
    case AttendanceState.ScheduledWorkFromHome:
    case AttendanceState.ScheduledOffice:
    case AttendanceState.ScheduledOther:
      // Scheduled days behave like Untracked when the user taps them.
      return AttendanceState.Untracked;
    default:
      return state;
  }
}

// Cycle forward (+1) or backward (-1) through the four concrete states.
export function cycleState(
  state: AttendanceState,
  direction: 1 | -1,
): AttendanceState {
  const current = baseState(state);
  const idx = CYCLE.indexOf(current);
  const next = (idx + direction + CYCLE.length) % CYCLE.length;
  return CYCLE[next];
}

export function isScheduled(state: AttendanceState): boolean {
  return state >= AttendanceState.ScheduledWorkFromHome;
}

// Whether a state counts as "present in the office" for compliance reporting.
export function isOffice(state: AttendanceState): boolean {
  return (
    state === AttendanceState.Office || state === AttendanceState.ScheduledOffice
  );
}

// Whether a state counts as a tracked work day (WFH or office, real or scheduled).
export function isWorkDay(state: AttendanceState): boolean {
  return (
    state === AttendanceState.WorkFromHome ||
    state === AttendanceState.Office ||
    state === AttendanceState.ScheduledWorkFromHome ||
    state === AttendanceState.ScheduledOffice
  );
}

export interface StateAppearance {
  label: string;
  short: string;
  // Background fill for a day cell.
  bg: string;
  // Text colour for the day number drawn on top of the fill.
  fg: string;
  // True for "scheduled" (projected) states, which render with a dashed border.
  scheduled: boolean;
}

// Exact colours from the web app (internal/embed/html/bases/base.html).
// Note the web's mapping: Work-from-home is green, Office is red, Other is blue.
const HOME = '#4CAF50'; // .present
const OFFICE = '#F44336'; // .not-present
const OTHER = '#2196F3'; // .other
const HOME_PLANNED = '#C8E6C9'; // .scheduled-home
const OFFICE_PLANNED = '#FFCDD2'; // .scheduled-office
const OTHER_PLANNED = '#BBDEFB'; // .scheduled-other

// The web renders dark day numbers on the coloured cells (no colour override).
const DAY_TEXT = '#1f2937';

export function appearance(state: AttendanceState): StateAppearance {
  switch (state) {
    case AttendanceState.WorkFromHome:
      return { label: 'Home', short: 'WFH', bg: HOME, fg: DAY_TEXT, scheduled: false };
    case AttendanceState.Office:
      return { label: 'Office', short: 'Office', bg: OFFICE, fg: DAY_TEXT, scheduled: false };
    case AttendanceState.Other:
      return { label: 'Other', short: 'Other', bg: OTHER, fg: DAY_TEXT, scheduled: false };
    case AttendanceState.ScheduledWorkFromHome:
      return { label: 'Home (planned)', short: 'WFH', bg: HOME_PLANNED, fg: DAY_TEXT, scheduled: true };
    case AttendanceState.ScheduledOffice:
      return { label: 'Office (planned)', short: 'Office', bg: OFFICE_PLANNED, fg: DAY_TEXT, scheduled: true };
    case AttendanceState.ScheduledOther:
      return { label: 'Other (planned)', short: 'Other', bg: OTHER_PLANNED, fg: DAY_TEXT, scheduled: true };
    case AttendanceState.Untracked:
    default:
      return { label: 'Untracked', short: '—', bg: 'transparent', fg: DAY_TEXT, scheduled: false };
  }
}
