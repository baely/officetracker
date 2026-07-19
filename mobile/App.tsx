import { Calistoga_400Regular, useFonts } from '@expo-google-fonts/calistoga';
import { StatusBar } from 'expo-status-bar';
import React, { useCallback, useEffect, useRef, useState } from 'react';
import { ActivityIndicator, Alert, AppState, StyleSheet, View } from 'react-native';
import { Auth0Provider } from 'react-native-auth0';
import { SafeAreaProvider, SafeAreaView } from 'react-native-safe-area-context';
import { AUTH0_CLIENT_ID, AUTH0_DOMAIN } from './src/config';
// Importing this registers the background geofence task (TaskManager.defineTask),
// which must happen at module load so it's available when iOS/Android relaunch
// the app for a location event.
import { checkProximityNow, syncWorkGeofence } from './src/location';
import TabBar, { Tab } from './src/components/TabBar';
import CalendarScreen from './src/screens/CalendarScreen';
import LoginScreen from './src/screens/LoginScreen';
import ReportScreen from './src/screens/ReportScreen';
import SettingsScreen from './src/screens/SettingsScreen';
import { clearConnection, Connection, loadConnection } from './src/storage';
import { colors } from './src/theme';

type Screen = 'loading' | 'login' | 'app';

export default function App() {
  const [screen, setScreen] = useState<Screen>('loading');
  // Which of the three pages the bottom tab bar shows when connected.
  const [tab, setTab] = useState<Tab>('calendar');
  const [conn, setConn] = useState<Connection | null>(null);
  const [fontsLoaded] = useFonts({ Calistoga_400Regular });
  // Dedupes concurrent 401s so we only sign out / alert once.
  const signingOut = useRef(false);

  useEffect(() => {
    loadConnection().then((c) => {
      setConn(c);
      setScreen(c ? 'app' : 'login');
    });
  }, []);

  // Resume work-location tracking after a restart, and catch the case where the
  // app opens while already at work (a geofence only fires on a crossing). Both
  // only read existing permissions — they never prompt — so they're safe to run
  // every launch. The work logic no-ops when no location/connection is stored.
  useEffect(() => {
    syncWorkGeofence();
    checkProximityNow();
    const sub = AppState.addEventListener('change', (state) => {
      if (state === 'active') checkProximityNow();
    });
    return () => sub.remove();
  }, []);

  // On a rejected token (401/403): drop the session and return to login.
  const handleUnauthorized = useCallback(() => {
    if (signingOut.current) return;
    signingOut.current = true;
    clearConnection();
    setConn(null);
    setScreen('login');
    Alert.alert(
      'Signed out',
      'Your session has expired or was revoked. Please sign in again.',
    );
  }, []);

  const onConnected = useCallback((c: Connection) => {
    signingOut.current = false;
    setConn(c);
    setTab('calendar');
    setScreen('app');
  }, []);

  // Hold on the spinner until both the saved session and the brand font are ready.
  let body: React.ReactNode = (
    <View style={styles.center}>
      <ActivityIndicator color={colors.textMuted} />
    </View>
  );
  switch (fontsLoaded ? screen : 'loading') {
    case 'loading':
      break;
    case 'login':
      body = (
        <LoginScreen initialBaseUrl={conn?.baseUrl} onConnected={onConnected} />
      );
      break;
    case 'app':
      body = conn ? (
        <View style={styles.app}>
          <View style={styles.page}>
            {tab === 'calendar' && (
              <CalendarScreen conn={conn} onUnauthorized={handleUnauthorized} />
            )}
            {tab === 'report' && (
              <ReportScreen conn={conn} onUnauthorized={handleUnauthorized} />
            )}
            {tab === 'settings' && (
              <SettingsScreen
                conn={conn}
                onUnauthorized={handleUnauthorized}
                onDisconnect={() => {
                  setConn(null);
                  setScreen('login');
                }}
              />
            )}
          </View>
          <TabBar active={tab} onSelect={setTab} />
        </View>
      ) : null;
      break;
  }

  return (
    <Auth0Provider domain={AUTH0_DOMAIN} clientId={AUTH0_CLIENT_ID}>
      <SafeAreaProvider>
        <SafeAreaView style={styles.safe} edges={['top', 'left', 'right']}>
          <StatusBar style="dark" />
          {body}
        </SafeAreaView>
      </SafeAreaProvider>
    </Auth0Provider>
  );
}

const styles = StyleSheet.create({
  safe: { flex: 1, backgroundColor: colors.bg },
  center: { flex: 1, alignItems: 'center', justifyContent: 'center' },
  app: { flex: 1 },
  page: { flex: 1 },
});
