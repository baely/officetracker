import React, { useState } from 'react';
import {
  ActivityIndicator,
  Image,
  KeyboardAvoidingView,
  Platform,
  Pressable,
  StyleSheet,
  Text,
  TextInput,
  View,
} from 'react-native';
import { useAuth0 } from 'react-native-auth0';
import { exchangeNativeToken, fetchServerMeta, ServerMeta } from '../api';
import { AUTH0_SCHEME, DEFAULT_BASE_URL } from '../config';
import { Connection, saveConnection } from '../storage';
import { colors, fonts, radius, spacing } from '../theme';

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
  const { authorize } = useAuth0();
  const [baseUrl, setBaseUrl] = useState(initialBaseUrl ?? DEFAULT_BASE_URL);
  const [advanced, setAdvanced] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  // Cached capabilities of the entered server, probed when the field changes so
  // we can relabel the button and hint that no sign-in is needed.
  const [serverMeta, setServerMeta] = useState<ServerMeta | null>(null);

  const normalised = normaliseUrl(baseUrl);
  const anonymous = serverMeta?.auth === 'none';

  // Probe the server's /api/v1/meta so the UI reflects an anonymous server
  // before the user commits. Runs on blur of the server field.
  async function probeServer() {
    setServerMeta(await fetchServerMeta(normaliseUrl(baseUrl)));
  }

  async function connect() {
    setError(null);
    setBusy(true);
    try {
      // Ask the server how to authenticate (cheap, unauthenticated probe).
      const meta = await fetchServerMeta(normalised);
      setServerMeta(meta);

      // Anonymous server: no Auth0 prompt and no token exchange — connect with
      // an empty token straight away.
      if (meta.auth === 'none') {
        const conn: Connection = {
          baseUrl: normalised,
          token: '',
          readOnly: meta.readOnly,
        };
        await saveConnection(conn);
        onConnected(conn);
        return;
      }

      const credentials = await authorize(
        // Always show a fresh login screen instead of silently reusing a cached
        // Auth0 session. prompt=login asks the server to re-prompt, and
        // ephemeralSession runs the iOS web flow in a private session that
        // doesn't share Safari's cookies — so no SSO session lingers after a
        // sign-out and signing back in actually asks for credentials.
        {
          scope: 'openid profile email',
          additionalParameters: { prompt: 'login' },
        },
        { customScheme: AUTH0_SCHEME, ephemeralSession: true },
      );
      if (!credentials?.idToken) {
        // User cancelled the Auth0 prompt.
        setBusy(false);
        return;
      }

      const token = await exchangeNativeToken(normalised, credentials.idToken);
      const conn: Connection = {
        baseUrl: normalised,
        token,
        readOnly: meta.readOnly,
      };
      await saveConnection(conn);
      onConnected(conn);
    } catch (e: any) {
      setError(e?.message ?? 'Sign in failed. Please try again.');
      setBusy(false);
    }
  }

  return (
    <KeyboardAvoidingView
      style={styles.flex}
      behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
    >
      <View style={styles.content}>
        <View style={styles.brand}>
          <Image
            source={require('../../assets/office-building.png')}
            style={styles.logo}
            resizeMode="contain"
          />
          <Text style={styles.title}>Officetracker</Text>
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
            onPress={connect}
            disabled={busy}
          >
            {busy ? (
              <ActivityIndicator color="#ffffff" />
            ) : (
              <Text style={styles.buttonText}>
                {anonymous ? 'Continue' : 'Sign in'}
              </Text>
            )}
          </Pressable>

          {advanced ? (
            <View style={styles.advanced}>
              <Text style={styles.label}>Server</Text>
              <TextInput
                style={styles.input}
                value={baseUrl}
                onChangeText={(t) => {
                  setBaseUrl(t);
                  // A new URL invalidates the previous probe.
                  setServerMeta(null);
                }}
                onBlur={probeServer}
                autoCapitalize="none"
                autoCorrect={false}
                keyboardType="url"
                inputMode="url"
                placeholder={DEFAULT_BASE_URL}
                placeholderTextColor={colors.textFaint}
                editable={!busy}
              />
              <Text style={styles.hint}>
                {anonymous
                  ? "This server doesn't require sign in — it's a read-only demo."
                  : 'Change this only if you use a different Office Tracker instance.'}
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
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1, backgroundColor: colors.brandTint },
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
    width: 96,
    height: 96,
    marginBottom: spacing.md,
  },
  title: {
    fontSize: 32,
    fontFamily: fonts.wordmark,
    color: colors.accent,
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
