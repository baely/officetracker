import { AttendanceState } from './states';
import { Connection } from './storage';

// Wire formats returned by the Go server (pkg/model).
interface YearStatePayload {
  data: {
    months: Record<string, { days: Record<string, { state: number }> }>;
  };
}

interface NotesPayload {
  data: Record<string, { note: string }>;
}

// month (1-12) -> day (1-31) -> state
export type MonthDays = Record<number, AttendanceState>;

export const WEEKDAYS_LOWER = [
  'monday',
  'tuesday',
  'wednesday',
  'thursday',
  'friday',
  'saturday',
  'sunday',
] as const;
export type Weekday = (typeof WEEKDAYS_LOWER)[number];

// A recurring weekly plan: each weekday maps to a base state (0-3).
export type SchedulePreferences = Record<Weekday, AttendanceState>;

export interface LinkedAccount {
  provider: string;
  providerDisplay: string;
  nickname: string;
}

export interface Settings {
  linkedAccounts: LinkedAccount[];
  schedule: SchedulePreferences;
}

export interface TokenInfo {
  tokenId: number;
  name: string;
  createdAt: string;
}

function emptySchedule(): SchedulePreferences {
  return {
    monday: AttendanceState.Untracked,
    tuesday: AttendanceState.Untracked,
    wednesday: AttendanceState.Untracked,
    thursday: AttendanceState.Untracked,
    friday: AttendanceState.Untracked,
    saturday: AttendanceState.Untracked,
    sunday: AttendanceState.Untracked,
  };
}

export class ApiError extends Error {
  constructor(
    message: string,
    public status?: number,
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

// True when an error means the stored token is no longer valid (expired or
// revoked) — the app should sign out rather than keep retrying.
export function isUnauthorized(e: unknown): boolean {
  return e instanceof ApiError && (e.status === 401 || e.status === 403);
}

function normaliseBase(baseUrl: string): string {
  return baseUrl.replace(/\/+$/, '');
}

// Exchanges an Auth0 ID token (obtained natively via react-native-auth0) for a
// long-lived Office Tracker API token. See the server's /auth/native handler.
export async function exchangeNativeToken(
  baseUrl: string,
  idToken: string,
): Promise<string> {
  let res: Response;
  try {
    res = await fetch(`${normaliseBase(baseUrl)}/auth/native`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ id_token: idToken }),
    });
  } catch {
    throw new ApiError('Could not reach the server. Check your connection.');
  }
  if (res.status === 401 || res.status === 403) {
    throw new ApiError('Sign in was rejected by the server.', res.status);
  }
  if (!res.ok) {
    throw new ApiError(`Sign in failed (server returned ${res.status}).`, res.status);
  }
  const body = (await res.json()) as { token?: string };
  if (!body.token) {
    throw new ApiError('Server did not return a token.');
  }
  return body.token;
}

export class Api {
  // onUnauthorized fires when the server rejects our token (401/403) so the app
  // can sign out and return to login instead of getting stuck.
  constructor(
    private conn: Connection,
    private onUnauthorized?: () => void,
  ) {}

  private headers(json = false): Record<string, string> {
    const h: Record<string, string> = {};
    if (json) h['Content-Type'] = 'application/json';
    if (this.conn.token) h['Authorization'] = `Bearer ${this.conn.token}`;
    return h;
  }

  private url(path: string): string {
    return `${normaliseBase(this.conn.baseUrl)}${path}`;
  }

  private async request(path: string, init?: RequestInit): Promise<Response> {
    let res: Response;
    try {
      res = await fetch(this.url(path), init);
    } catch (e) {
      throw new ApiError(
        'Could not reach the server. Check the URL and your connection.',
      );
    }
    if (res.status === 401 || res.status === 403) {
      // Token expired or revoked — trigger a sign-out.
      this.onUnauthorized?.();
      throw new ApiError('Your session has expired. Please sign in again.', res.status);
    }
    if (!res.ok) {
      throw new ApiError(`Server returned ${res.status}.`, res.status);
    }
    return res;
  }

  // Verifies the URL is an Office Tracker server. Does not require auth.
  async health(): Promise<void> {
    await this.request('/api/v1/health/check');
  }

  // Fetches a whole fiscal year of attendance (with scheduled days merged in by
  // the server) and flattens it to month -> day -> state.
  async getYear(fiscalYear: number): Promise<Record<number, MonthDays>> {
    const res = await this.request(`/api/v1/state/${fiscalYear}`, {
      headers: this.headers(),
    });
    const payload = (await res.json()) as YearStatePayload;
    const out: Record<number, MonthDays> = {};
    const months = payload?.data?.months ?? {};
    for (const [monthStr, monthVal] of Object.entries(months)) {
      const month = Number(monthStr);
      const days: MonthDays = {};
      for (const [dayStr, dayVal] of Object.entries(monthVal.days ?? {})) {
        days[Number(dayStr)] = dayVal.state as AttendanceState;
      }
      out[month] = days;
    }
    return out;
  }

  // Saves a single day. year/month/day are calendar values (not fiscal).
  async putDay(
    year: number,
    month: number,
    day: number,
    state: AttendanceState,
  ): Promise<void> {
    await this.request(`/api/v1/state/${year}/${month}/${day}`, {
      method: 'PUT',
      headers: this.headers(true),
      body: JSON.stringify({ data: { state } }),
    });
  }

  async getNotes(fiscalYear: number): Promise<Record<number, string>> {
    const res = await this.request(`/api/v1/note/${fiscalYear}`, {
      headers: this.headers(),
    });
    const payload = (await res.json()) as NotesPayload;
    const out: Record<number, string> = {};
    for (const [monthStr, val] of Object.entries(payload?.data ?? {})) {
      out[Number(monthStr)] = val.note ?? '';
    }
    return out;
  }

  async putNote(year: number, month: number, note: string): Promise<void> {
    await this.request(`/api/v1/note/${year}/${month}`, {
      method: 'PUT',
      headers: this.headers(true),
      body: JSON.stringify({ data: { note } }),
    });
  }

  // ---- Settings: linked accounts + recurring schedule ----

  async getSettings(): Promise<Settings> {
    const res = await this.request('/api/v1/settings/', {
      headers: this.headers(),
    });
    const p = (await res.json()) as {
      linked_accounts?: { provider: string; provider_display: string; nickname: string }[];
      schedule_preferences?: Partial<Record<Weekday, number>>;
    };
    const sp = p.schedule_preferences ?? {};
    const schedule = emptySchedule();
    for (const day of WEEKDAYS_LOWER) {
      schedule[day] = (sp[day] ?? 0) as AttendanceState;
    }
    const linkedAccounts = (p.linked_accounts ?? []).map((a) => ({
      provider: a.provider,
      providerDisplay: a.provider_display,
      nickname: a.nickname,
    }));
    return { linkedAccounts, schedule };
  }

  async updateSchedule(schedule: SchedulePreferences): Promise<void> {
    await this.request('/api/v1/settings/schedule', {
      method: 'PUT',
      headers: this.headers(true),
      body: JSON.stringify({ data: schedule }),
    });
  }

  // Returns an Auth0 URL to link an additional social account (expires in 10m).
  async getAccountLinkUrl(): Promise<string> {
    const res = await this.request('/api/v1/account/link', {
      headers: this.headers(),
    });
    const p = (await res.json()) as { url?: string };
    if (!p.url) throw new ApiError('Server did not return a link URL.');
    return p.url;
  }

  // ---- Developer tokens ----

  async listTokens(): Promise<TokenInfo[]> {
    const res = await this.request('/api/v1/developer/tokens', {
      headers: this.headers(),
    });
    const p = (await res.json()) as {
      tokens?: { token_id: number; name: string; created_at: string }[];
    };
    return (p.tokens ?? []).map((t) => ({
      tokenId: t.token_id,
      name: t.name,
      createdAt: t.created_at,
    }));
  }

  async createToken(name: string): Promise<string> {
    const res = await this.request('/api/v1/developer/secret', {
      method: 'POST',
      headers: this.headers(true),
      body: JSON.stringify({ data: { name } }),
    });
    const p = (await res.json()) as { secret?: string };
    if (!p.secret) throw new ApiError('Server did not return a token.');
    return p.secret;
  }

  async revokeToken(tokenId: number): Promise<void> {
    await this.request(`/api/v1/developer/tokens/${tokenId}`, {
      method: 'DELETE',
      headers: this.headers(),
    });
  }

  // Best-effort: asks the server to revoke the token we're authenticated with,
  // so signing out doesn't leave it active. Errors are ignored (we're logging
  // out regardless) and it deliberately doesn't trigger onUnauthorized.
  async logout(): Promise<void> {
    try {
      await fetch(this.url('/api/v1/auth/logout'), {
        method: 'POST',
        headers: this.headers(),
      });
    } catch {
      // ignore — sign-out proceeds locally either way
    }
  }
}
