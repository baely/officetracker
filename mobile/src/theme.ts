// Flat design tokens, using the Officetracker brand palette
// (see internal/embed/html/hero.html + bases/base.html).
export const colors = {
  bg: '#ffffff',
  surface: '#ffffff',
  border: '#e5e7eb',
  borderStrong: '#d1d5db',
  text: '#384346', // brand slate (hero body text)
  textMuted: '#526064', // brand muted slate
  textFaint: '#9ca3af',
  accent: '#24292e', // brand charcoal (primary buttons / wordmark)
  brandTint: '#eef8f6', // brand mint (hero background)
  danger: '#dc2626',
  fieldBg: '#f9fafb',
  todayRing: '#FFC107', // amber, matches the web app's "today" border
};

// "Officetracker" wordmark font (Google Calistoga), loaded at runtime in App.tsx.
export const fonts = {
  wordmark: 'Calistoga_400Regular',
};

export const spacing = {
  xs: 4,
  sm: 8,
  md: 12,
  lg: 16,
  xl: 24,
};

export const radius = {
  sm: 6,
  md: 8,
  lg: 12,
};
