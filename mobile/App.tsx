import { Calistoga_400Regular, useFonts } from '@expo-google-fonts/calistoga';
import { StatusBar } from 'expo-status-bar';
import React, { useCallback, useEffect, useRef, useState } from 'react';
import { ActivityIndicator, Alert, StyleSheet, View } from 'react-native';
import { Auth0Provider } from 'react-native-auth0';
import { SafeAreaProvider, SafeAreaView } from 'react-native-safe-area-context';
import { AUTH0_CLIENT_ID, AUTH0_DOMAIN } from './src/config';
import CalendarScreen from './src/screens/CalendarScreen';
import LoginScreen from './src/screens/LoginScreen';
import SettingsScreen from './src/screens/SettingsScreen';
import { clearConnection, Connection, loadConnection } from './src/storage';
import { colors } from './src/theme';

type Screen = 'loading' | 'login' | 'relogin' | 'calendar' | 'settings';

export default function App() {
  const [screen, setScreen] = useState<Screen>('loading');
  const [conn, setConn] = useState<Connection | null>(null);
  const [fontsLoaded] = useFonts({ Calistoga_400Regular });
  // Dedupes concurrent 401s so we only sign out / alert once.
  const signingOut = useRef(false);

  useEffect(() => {
    loadConnection().then((c) => {
      setConn(c);
      setScreen(c ? 'calendar' : 'login');
    });
  }, []);

  // Called when the server rejects our token (expired or revoked): wipe the
  // saved session and return to login instead of leaving the app stuck.
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
    setScreen('calendar');
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
    case 'relogin':
      body = (
        <LoginScreen
          initialBaseUrl={conn?.baseUrl}
          onConnected={onConnected}
          onCancel={
            screen === 'relogin' ? () => setScreen('settings') : undefined
          }
        />
      );
      break;
    case 'calendar':
      body = conn ? (
        <CalendarScreen
          conn={conn}
          onOpenSettings={() => setScreen('settings')}
          onUnauthorized={handleUnauthorized}
        />
      ) : null;
      break;
    case 'settings':
      body = conn ? (
        <SettingsScreen
          conn={conn}
          onClose={() => setScreen('calendar')}
          onSignInAgain={() => setScreen('relogin')}
          onUnauthorized={handleUnauthorized}
          onDisconnect={() => {
            setConn(null);
            setScreen('login');
          }}
        />
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
});
