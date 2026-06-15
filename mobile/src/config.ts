// The Office Tracker server the app talks to by default. Advanced users can
// point at another instance from the login screen.
// NOTE: temporarily pointed at beta for testing the native-auth backend.
// Switch back to https://officetracker.com.au before a production release.
export const DEFAULT_BASE_URL = 'https://beta.officetracker.com.au';

// Auth0 configuration for the Native application (react-native-auth0). The
// domain is the tenant's custom domain; the client ID is public.
export const AUTH0_DOMAIN = 'auth.officetracker.com.au';
export const AUTH0_CLIENT_ID = 'i7HP3uXeIm0ItFiqXRprnRYmU6bJCI6S';

// Must match the `customScheme` in app.json's react-native-auth0 plugin and the
// callback URLs registered in the Auth0 dashboard.
export const AUTH0_SCHEME = 'officetracker';
