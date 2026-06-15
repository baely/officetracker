# Office Tracker — Mobile

Native iOS and Android client for [Office Tracker](../README.md), built with
[Expo](https://expo.dev) (React Native + TypeScript). One codebase compiles to
both platforms.

It talks to an existing Office Tracker server over its REST API
(`/api/v1/...`) — log office attendance on a monthly calendar, see compliance
stats, and keep per-month notes.

## Features

- 📅 Monthly calendar — tap a day to cycle Untracked → Home → Office → Other;
  long-press to go back.
- 🗓️ "Planned" days projected from your weekly schedule render with a dashed
  outline (driven by the server's schedule preferences).
- 📊 Live month + year compliance summary (office days / work days, percentage).
- 📝 Per-month notes.
- 🔐 Native **Auth0** login via [`react-native-auth0`](https://github.com/auth0/react-native-auth0)
  (PKCE, system browser). No URLs or tokens to copy.

## Running in development

You need [Node.js](https://nodejs.org) (18+). Because `react-native-auth0` ships
native code, **this app does not run in Expo Go** — you need a development build.

```shell
cd mobile
npm install
npx expo run:ios       # or: npx expo run:android
```

`expo run:*` builds and installs a dev client on a simulator/emulator (or
connected device), then starts Metro. iOS requires Xcode + CocoaPods; Android
requires Android Studio. Alternatively build a dev client in the cloud with
`eas build --profile development`.

After the first dev build, you can iterate with just `npx expo start` (it loads
into the installed dev client). The `w` web preview won't work — the Auth0
native module isn't available there.

### Signing in

Tap **Sign in**. The system browser opens the Auth0 Universal Login (the same
tenant as the web app). After you authenticate, the app receives an Auth0 ID
token, exchanges it at `POST /auth/native` on the server for a long-lived API
token, and uses that as an `Authorization: Bearer` credential for all API
calls. The token shows up on the server's Developer page as
"Office Tracker mobile app". Nothing to copy or paste.

By default it connects to `https://officetracker.com.au`. Tap **Use a different
server** on the login screen to point at another instance (e.g.
`https://beta.officetracker.com.au`) — note it still authenticates against the
same Auth0 tenant.

## Auth0 configuration

The app uses a **Native** Auth0 application (client ID in `src/config.ts`,
domain `auth.officetracker.com.au`). In the Auth0 dashboard, that application's
**Allowed Callback URLs** and **Allowed Logout URLs** must include:

```text
officetracker://auth.officetracker.com.au/ios/com.baely.officetracker/callback
officetracker://auth.officetracker.com.au/android/com.baely.officetracker/callback
```

The server verifies the app's ID token against this client ID via the
`AUTH0_NATIVE_CLIENT_ID` env var (see `config/cloud.env`).

## Building native binaries

Local builds need the platform toolchains (Xcode for iOS, Android Studio for
Android). The simplest path is Expo's cloud build service, [EAS](https://docs.expo.dev/build/introduction/):

```shell
npm install -g eas-cli
eas login
eas build --platform ios       # or android, or all
```

`eas.json` defines `development` (dev client), `preview` (internal/APK), and
`production` profiles.

To generate the native `ios/` and `android/` project folders locally instead:

```shell
npx expo prebuild
```

## Project layout

```
App.tsx                 Root: Auth0Provider, loads saved session, routes screens
src/
  api.ts                REST client + /auth/native token exchange
  storage.ts            Persists the server URL + exchanged token (AsyncStorage)
  config.ts             Default server URL + Auth0 domain / client ID / scheme
  states.ts             Attendance state enum, cycling, colours
  dates.ts              Fiscal-year + calendar-grid helpers
  stats.ts              Month / year compliance aggregation
  theme.ts              Flat design tokens
  components/           Calendar, DayCell, Summary, Legend
  screens/              LoginScreen, CalendarScreen, SettingsScreen
```

### How sign-in works

`react-native-auth0` runs the OAuth Authorization Code + PKCE flow natively and
returns an Auth0 ID token. The app can't use that token against the API
directly — the server validates its own credentials — so it posts the ID token
to `POST /auth/native`. The server (`internal/auth/native.go`) verifies the
token against the Native app's client ID (`AUTH0_NATIVE_CLIENT_ID`), maps it to
a user (the same `subjectToUserID` mapping as the web callback), mints an API
token, and returns it. The app stores that token and sends it as
`Authorization: Bearer …` (the server's `MethodSecret`).

### A note on the "fiscal year"

The server groups data into a fiscal year: calendar months Jan–Sep belong to
fiscal year `Y`, and Oct–Dec belong to `Y+1`. The `/state/{year}` and
`/note/{year}` endpoints are keyed by that fiscal year, while day writes
(`PUT /state/{year}/{month}/{day}`) use the real calendar date. See
`src/dates.ts` and `src/api.ts`.
