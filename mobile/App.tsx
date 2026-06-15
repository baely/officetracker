import { StatusBar } from 'expo-status-bar';
import React, { useEffect, useState } from 'react';
import { ActivityIndicator, StyleSheet, View } from 'react-native';
import { Auth0Provider } from 'react-native-auth0';
import { SafeAreaProvider, SafeAreaView } from 'react-native-safe-area-context';
import { AUTH0_CLIENT_ID, AUTH0_DOMAIN } from './src/config';
import CalendarScreen from './src/screens/CalendarScreen';
import LoginScreen from './src/screens/LoginScreen';
import SettingsScreen from './src/screens/SettingsScreen';
import { Connection, loadConnection } from './src/storage';
import { colors } from './src/theme';

type Screen = 'loading' | 'login' | 'relogin' | 'calendar' | 'settings';

export default function App() {
  const [screen, setScreen] = useState<Screen>('loading');
  const [conn, setConn] = useState<Connection | null>(null);

  useEffect(() => {
    loadConnection().then((c) => {
      setConn(c);
      setScreen(c ? 'calendar' : 'login');
    });
  }, []);

  let body: React.ReactNode = null;
  switch (screen) {
    case 'loading':
      body = (
        <View style={styles.center}>
          <ActivityIndicator color={colors.textMuted} />
        </View>
      );
      break;
    case 'login':
    case 'relogin':
      body = (
        <LoginScreen
          initialBaseUrl={conn?.baseUrl}
          onConnected={(c) => {
            setConn(c);
            setScreen('calendar');
          }}
          onCancel={
            screen === 'relogin' ? () => setScreen('settings') : undefined
          }
        />
      );
      break;
    case 'calendar':
      body = conn ? (
        <CalendarScreen conn={conn} onOpenSettings={() => setScreen('settings')} />
      ) : null;
      break;
    case 'settings':
      body = conn ? (
        <SettingsScreen
          conn={conn}
          onClose={() => setScreen('calendar')}
          onSignInAgain={() => setScreen('relogin')}
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
