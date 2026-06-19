// Design tokens mirroring the Officetracker web app so the two clients feel
// like one product. Values are taken straight from the web stylesheet
// (internal/embed/html/bases/base.html + static/themes.css): an off-white page,
// a white content card, a pale-cyan nav bar, square-ish borders and a charcoal
// brand accent.
export const colors = {
  // Page + surfaces
  bg: '#f7f8f8', // web body background
  surface: '#ffffff', // white content card (web <main>)
  navBg: '#ddeeee', // web nav bar (`#dee`)
  border: '#dee2e6', // web border colour, used everywhere
  borderStrong: '#adb5bd', // dashed "planned" outlines
  cellBg: '#f8f9fa', // web weekday-header / day background
  fieldBg: '#f8f9fa', // inputs + notes textarea
  tableHeaderBg: '#f4f4f4', // web summary table header

  // Text
  text: '#212529', // headings / primary text
  textMuted: '#495057', // web secondary text colour
  textFaint: '#6c757d', // faint labels, hints, footer

  // Brand + status
  accent: '#24292e', // brand charcoal (buttons / wordmark)
  brandTint: '#eef8f6', // hero mint (login background)
  danger: '#F44336', // web red (.not-present)
  todayRing: '#FFC107', // web "today" amber border
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

// The web app uses a consistent 4px corner radius; keep it subtle and flat.
export const radius = {
  sm: 4,
  md: 4,
  lg: 6,
};
