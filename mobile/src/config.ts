// Default server; users can point elsewhere from the login screen.
export const DEFAULT_BASE_URL = 'https://officetracker.com.au';

export interface KnownServer {
  // Shown in the login server picker.
  label: string;
  // Base URL connected to (no trailing slash).
  url: string;
}

// Servers offered on the login screen. The first entry is the default and must
// match DEFAULT_BASE_URL. Users can still enter any other instance manually.
export const KNOWN_SERVERS: KnownServer[] = [
  { label: 'officetracker.com.au', url: 'https://officetracker.com.au' },
  { label: 'beta.officetracker.com.au', url: 'https://beta.officetracker.com.au' },
  { label: 'officetracker.baileys.app', url: 'https://officetracker.baileys.app' },
];

// Native Auth0 application (react-native-auth0). Client ID is public.
export const AUTH0_DOMAIN = 'auth.officetracker.com.au';
export const AUTH0_CLIENT_ID = 'i7HP3uXeIm0ItFiqXRprnRYmU6bJCI6S';

// Must match app.json's react-native-auth0 customScheme + the Auth0 callback URLs.
export const AUTH0_SCHEME = 'officetracker';
