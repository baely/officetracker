import AsyncStorage from '@react-native-async-storage/async-storage';

const KEY = 'officetracker.connection';

export interface Connection {
  baseUrl: string;
  // API token sent as `Authorization: Bearer <token>` (empty if the server needs no auth).
  token: string;
}

export async function loadConnection(): Promise<Connection | null> {
  try {
    const raw = await AsyncStorage.getItem(KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as Connection;
    if (!parsed.baseUrl) return null;
    return { baseUrl: parsed.baseUrl, token: parsed.token ?? '' };
  } catch {
    return null;
  }
}

export async function saveConnection(conn: Connection): Promise<void> {
  await AsyncStorage.setItem(KEY, JSON.stringify(conn));
}

export async function clearConnection(): Promise<void> {
  await AsyncStorage.removeItem(KEY);
}

// ---- Work location (auto office detection) ----
//
// Stored only on the device: the geofencing/auto-marking feature is entirely
// client-side, so the server never needs to know where "work" is.

const WORK_LOCATION_KEY = 'officetracker.workLocation';
const AUTO_OFFICE_KEY = 'officetracker.autoOffice';
const START_MONTH_KEY = 'officetracker.startMonth';
const BANNER_DISMISSED_KEY = 'officetracker.workBannerDismissed';

// Default geofence radius in metres. Generous because low-accuracy GPS is fine —
// we only care that the user got "near" work, not their exact desk.
export const DEFAULT_WORK_RADIUS = 200;

export interface WorkLocation {
  latitude: number;
  longitude: number;
  // Geofence radius in metres.
  radius: number;
  // Human-readable label (reverse-geocoded address), for display only.
  label?: string;
}

export async function loadWorkLocation(): Promise<WorkLocation | null> {
  try {
    const raw = await AsyncStorage.getItem(WORK_LOCATION_KEY);
    if (!raw) return null;
    const p = JSON.parse(raw) as WorkLocation;
    if (typeof p.latitude !== 'number' || typeof p.longitude !== 'number') {
      return null;
    }
    return {
      latitude: p.latitude,
      longitude: p.longitude,
      radius: p.radius || DEFAULT_WORK_RADIUS,
      label: p.label,
    };
  } catch {
    return null;
  }
}

export async function saveWorkLocation(loc: WorkLocation): Promise<void> {
  await AsyncStorage.setItem(WORK_LOCATION_KEY, JSON.stringify(loc));
}

export async function clearWorkLocation(): Promise<void> {
  await AsyncStorage.removeItem(WORK_LOCATION_KEY);
}

// The last calendar day (YYYY-MM-DD, device-local) we resolved auto-marking for,
// so a geofence that fires repeatedly only hits the server once per day.
export async function getAutoOfficeHandledDate(): Promise<string | null> {
  try {
    return await AsyncStorage.getItem(AUTO_OFFICE_KEY);
  } catch {
    return null;
  }
}

export async function setAutoOfficeHandledDate(date: string): Promise<void> {
  await AsyncStorage.setItem(AUTO_OFFICE_KEY, date);
}

// The background task can't fetch settings cheaply, so the app caches the
// tracking-year start month here whenever it loads settings.
export async function cacheStartMonth(month: number): Promise<void> {
  await AsyncStorage.setItem(START_MONTH_KEY, String(month));
}

export async function getCachedStartMonth(): Promise<number | null> {
  try {
    const raw = await AsyncStorage.getItem(START_MONTH_KEY);
    if (!raw) return null;
    const n = Number(raw);
    return n >= 1 && n <= 12 ? n : null;
  } catch {
    return null;
  }
}

export async function isWorkBannerDismissed(): Promise<boolean> {
  try {
    return (await AsyncStorage.getItem(BANNER_DISMISSED_KEY)) === '1';
  } catch {
    return false;
  }
}

export async function dismissWorkBanner(): Promise<void> {
  await AsyncStorage.setItem(BANNER_DISMISSED_KEY, '1');
}
