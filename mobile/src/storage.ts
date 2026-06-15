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
