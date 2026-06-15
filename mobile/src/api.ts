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

export class ApiError extends Error {
  constructor(
    message: string,
    public status?: number,
  ) {
    super(message);
    this.name = 'ApiError';
  }
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
  constructor(private conn: Connection) {}

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
      throw new ApiError('Unauthorized. Check your API token.', res.status);
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
}
