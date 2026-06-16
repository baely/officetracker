// Background work-location detection.
//
// When the user is near their saved "work" location we mark today as Office —
// unless they've already set the day themselves. This uses a geofence (cheap,
// survives app restarts and works when the app is backgrounded/terminated)
// rather than a continuous location stream, plus a one-shot foreground check to
// catch the case where they're already inside the region when it's configured.
import * as Location from 'expo-location';
import * as TaskManager from 'expo-task-manager';
import { Api } from './api';
import { DEFAULT_TRACKING_YEAR_START_MONTH, trackingYear } from './dates';
import { AttendanceState } from './states';
import {
  clearWorkLocation,
  getAutoOfficeHandledDate,
  getCachedStartMonth,
  loadConnection,
  loadWorkLocation,
  saveWorkLocation,
  setAutoOfficeHandledDate,
  WorkLocation,
} from './storage';

export const GEOFENCE_TASK = 'officetracker-work-geofence';
const REGION_ID = 'work';

// Device-local YYYY-MM-DD for the given date.
function localDateKey(d = new Date()): string {
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${y}-${m}-${day}`;
}

// Great-circle distance between two lat/lng points, in metres.
function distanceMeters(
  a: { latitude: number; longitude: number },
  b: { latitude: number; longitude: number },
): number {
  const R = 6371000; // earth radius (m)
  const toRad = (deg: number) => (deg * Math.PI) / 180;
  const dLat = toRad(b.latitude - a.latitude);
  const dLng = toRad(b.longitude - a.longitude);
  const lat1 = toRad(a.latitude);
  const lat2 = toRad(b.latitude);
  const h =
    Math.sin(dLat / 2) ** 2 +
    Math.cos(lat1) * Math.cos(lat2) * Math.sin(dLng / 2) ** 2;
  return 2 * R * Math.asin(Math.sqrt(h));
}

// A state the user set themselves — auto-marking must never overwrite it. The
// scheduled/planned variants (and Untracked) are fair game.
function isUserSet(state: AttendanceState): boolean {
  return (
    state === AttendanceState.WorkFromHome ||
    state === AttendanceState.Office ||
    state === AttendanceState.Other
  );
}

// If today hasn't been set by the user (untracked or only planned), mark it as
// Office. Guarded so it touches the server at most once per day. Safe to call
// from the background task or the foreground.
export async function markOfficeForToday(): Promise<void> {
  const now = new Date();
  const dateKey = localDateKey(now);
  if ((await getAutoOfficeHandledDate()) === dateKey) return;

  const conn = await loadConnection();
  if (!conn) return;

  const year = now.getFullYear();
  const month = now.getMonth() + 1; // 1-12
  const day = now.getDate();
  const startMonth =
    (await getCachedStartMonth()) ?? DEFAULT_TRACKING_YEAR_START_MONTH;
  const fy = trackingYear(year, month, startMonth);

  // No onUnauthorized handler: a background trigger must never sign the user out.
  const api = new Api(conn);

  try {
    const yearData = await api.getYear(fy);
    const current = yearData[month]?.[day] ?? AttendanceState.Untracked;
    if (isUserSet(current)) {
      // Respect a day the user set (including a manual WFH/Other).
      await setAutoOfficeHandledDate(dateKey);
      return;
    }
    await api.putDay(year, month, day, AttendanceState.Office);
    await setAutoOfficeHandledDate(dateKey);
  } catch {
    // Network/auth error — leave today unhandled so a later trigger retries.
  }
}

// Geofence handler. Fires on region enter even when the app is backgrounded.
TaskManager.defineTask(GEOFENCE_TASK, async ({ data, error }) => {
  if (error) return;
  const { eventType } = (data ?? {}) as {
    eventType?: Location.GeofencingEventType;
  };
  if (eventType === Location.GeofencingEventType.Enter) {
    await markOfficeForToday();
  }
});

// Requests foreground then background ("Always") permission. Background is what
// lets the geofence fire while the app isn't open.
export async function requestTrackingPermissions(): Promise<{
  foreground: boolean;
  background: boolean;
}> {
  const fg = await Location.requestForegroundPermissionsAsync();
  if (fg.status !== 'granted') return { foreground: false, background: false };
  const bg = await Location.requestBackgroundPermissionsAsync();
  return { foreground: true, background: bg.status === 'granted' };
}

async function startGeofence(loc: WorkLocation): Promise<void> {
  if (await TaskManager.isTaskRegisteredAsync(GEOFENCE_TASK)) {
    try {
      await Location.stopGeofencingAsync(GEOFENCE_TASK);
    } catch {
      // not running — ignore
    }
  }
  await Location.startGeofencingAsync(GEOFENCE_TASK, [
    {
      identifier: REGION_ID,
      latitude: loc.latitude,
      longitude: loc.longitude,
      radius: loc.radius,
      notifyOnEnter: true,
      notifyOnExit: false,
    },
  ]);
}

async function stopGeofence(): Promise<void> {
  if (await TaskManager.isTaskRegisteredAsync(GEOFENCE_TASK)) {
    try {
      await Location.stopGeofencingAsync(GEOFENCE_TASK);
    } catch {
      // ignore
    }
  }
}

// One-shot foreground check: a geofence only fires on a *crossing*, so this
// covers configuring the location (or opening the app) while already at work.
export async function checkProximityNow(): Promise<void> {
  const loc = await loadWorkLocation();
  if (!loc) return;
  // Already resolved today — skip the (battery-costly) location fix entirely.
  if ((await getAutoOfficeHandledDate()) === localDateKey()) return;
  const { status } = await Location.getForegroundPermissionsAsync();
  if (status !== 'granted') return;
  try {
    const pos = await Location.getCurrentPositionAsync({
      accuracy: Location.Accuracy.Balanced,
    });
    if (distanceMeters(pos.coords, loc) <= loc.radius) {
      await markOfficeForToday();
    }
  } catch {
    // best effort
  }
}

// Persist the location, request permissions, arm the geofence and do an
// immediate proximity check. Returns whether background tracking is active
// (false if the user declined the "Always" permission).
export async function enableWorkTracking(loc: WorkLocation): Promise<boolean> {
  await saveWorkLocation(loc);
  const perms = await requestTrackingPermissions();
  if (perms.background) {
    await startGeofence(loc);
  }
  if (perms.foreground) {
    await checkProximityNow();
  }
  return perms.background;
}

export async function disableWorkTracking(): Promise<void> {
  await stopGeofence();
  await clearWorkLocation();
}

// Reconciles the running geofence with stored state + current permissions.
// Called on launch so tracking resumes after the app (or device) restarts.
export async function syncWorkGeofence(): Promise<void> {
  const loc = await loadWorkLocation();
  if (!loc) {
    await stopGeofence();
    return;
  }
  const { status } = await Location.getBackgroundPermissionsAsync();
  if (status !== 'granted') {
    await stopGeofence();
    return;
  }
  await startGeofence(loc);
}
