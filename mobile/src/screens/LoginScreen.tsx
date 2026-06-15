import React, { useState } from 'react';
import {
  ActivityIndicator,
  Pressable,
  StyleSheet,
  Text,
  TextInput,
  View,
} from 'react-native';
import { useAuth0 } from 'react-native-auth0';
import { exchangeNativeToken } from '../api';
import { AUTH0_SCHEME, DEFAULT_BASE_URL } from '../config';
import { Connection, saveConnection } from '../storage';
import { colors, radius, spacing } from '../theme';

interface Props {
  initialBaseUrl?: string;
  onConnected: (conn: Connection) => void;
  onCancel?: () => void;
}

function normaliseUrl(raw: string): string {
  let url = raw.trim();
  if (!/^https?:\/\//i.test(url)) url = `https://${url}`;
  return url.replace(/\/+$/, '');
}

export default function LoginScreen({
  initialBaseUrl,
  onConnected,
  onCancel,
}: Props) {
  const { authorize, clearSession } = useAuth0();
  const [baseUrl, setBaseUrl] = useState(initialBaseUrl ?? DEFAULT_BASE_URL);
  const [advanced, setAdvanced] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const normalised = normaliseUrl(baseUrl);

  async function signIn() {
    setError(null);
    setBusy(true);
    try {
      const credentials = await authorize(
        { scope: 'openid profile email' },
        { customScheme: AUTH0_SCHEME },
      );
      if (!credentials?.idToken) {
        // User cancelled the Auth0 prompt.
        setBusy(false);
        return;
      }

      const token = await exchangeNativeToken(normalised, credentials.idToken);
      const conn: Connection = { baseUrl: normalised, token };
      await saveConnection(conn);

      // The Auth0 session itself isn't needed after exchange; drop it so the
      // next sign-in always re-prompts cleanly.
      clearSession({}, { customScheme: AUTH0_SCHEME }).catch(() => {});

      onConnected(conn);
    } catch (e: any) {
      setError(e?.message ?? 'Sign in failed. Please try again.');
      setBusy(false);
    }
  }

  return (
    <View style={styles.flex}>
      <View style={styles.content}>
        <View style={styles.brand}>
          <View style={styles.logo}>
            <Text style={styles.logoMark}>OT</Text>
          </View>
          <Text style={styles.title}>Office Tracker</Text>
          <Text style={styles.subtitle}>
            Log your office attendance and track RTO compliance.
          </Text>
        </View>

        <View style={styles.actions}>
          {error && <Text style={styles.error}>{error}</Text>}

          <Pressable
            style={({ pressed }) => [
              styles.button,
              pressed && styles.buttonPressed,
              busy && styles.buttonDisabled,
            ]}
            onPress={signIn}
            disabled={busy}
          >
            {busy ? (
              <ActivityIndicator color="#ffffff" />
            ) : (
              <Text style={styles.buttonText}>Sign in</Text>
            )}
          </Pressable>

          {advanced ? (
            <View style={styles.advanced}>
              <Text style={styles.label}>Server</Text>
              <TextInput
                style={styles.input}
                value={baseUrl}
                onChangeText={setBaseUrl}
                autoCapitalize="none"
                autoCorrect={false}
                keyboardType="url"
                inputMode="url"
                placeholder={DEFAULT_BASE_URL}
                placeholderTextColor={colors.textFaint}
                editable={!busy}
              />
              <Text style={styles.hint}>
                Change this only if you use a different Office Tracker instance.
              </Text>
            </View>
          ) : (
            <Pressable onPress={() => setAdvanced(true)} hitSlop={8} disabled={busy}>
              <Text style={styles.advancedLink}>Use a different server</Text>
            </Pressable>
          )}

          {onCancel && (
            <Pressable style={styles.cancel} onPress={onCancel} disabled={busy}>
              <Text style={styles.cancelText}>Cancel</Text>
            </Pressable>
          )}
        </View>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1, backgroundColor: colors.bg },
  content: {
    flex: 1,
    padding: spacing.xl,
    justifyContent: 'space-between',
  },
  brand: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
  },
  logo: {
    width: 72,
    height: 72,
    borderRadius: radius.lg,
    backgroundColor: colors.accent,
    alignItems: 'center',
    justifyContent: 'center',
    marginBottom: spacing.lg,
  },
  logoMark: {
    color: '#ffffff',
    fontSize: 28,
    fontWeight: '700',
    letterSpacing: 1,
  },
  title: {
    fontSize: 28,
    fontWeight: '700',
    color: colors.text,
  },
  subtitle: {
    marginTop: spacing.sm,
    fontSize: 15,
    color: colors.textMuted,
    textAlign: 'center',
    lineHeight: 21,
    paddingHorizontal: spacing.lg,
  },
  actions: {
    paddingBottom: spacing.lg,
  },
  error: {
    marginBottom: spacing.md,
    color: colors.danger,
    fontSize: 14,
    textAlign: 'center',
  },
  button: {
    backgroundColor: colors.accent,
    borderRadius: radius.md,
    paddingVertical: spacing.md + 2,
    alignItems: 'center',
  },
  buttonPressed: { opacity: 0.8 },
  buttonDisabled: { opacity: 0.6 },
  buttonText: { color: '#ffffff', fontSize: 16, fontWeight: '600' },
  advancedLink: {
    marginTop: spacing.lg,
    textAlign: 'center',
    color: colors.textMuted,
    fontSize: 14,
  },
  advanced: { marginTop: spacing.lg },
  label: {
    fontSize: 13,
    fontWeight: '600',
    color: colors.text,
    marginBottom: spacing.xs,
  },
  input: {
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: radius.md,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.md,
    fontSize: 15,
    color: colors.text,
    backgroundColor: colors.fieldBg,
  },
  hint: {
    marginTop: spacing.xs,
    fontSize: 12,
    color: colors.textFaint,
    lineHeight: 17,
  },
  cancel: {
    marginTop: spacing.md,
    alignItems: 'center',
    paddingVertical: spacing.sm,
  },
  cancelText: { color: colors.textMuted, fontSize: 15 },
});
