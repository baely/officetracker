import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  ActivityIndicator,
  Alert,
  Linking,
  Pressable,
  RefreshControl,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  View,
} from 'react-native';
import { Api, isUnauthorized, Settings, TokenInfo, Weekday } from '../api';
import Header from '../components/Header';
import Legend from '../components/Legend';
import ScheduleEditor from '../components/ScheduleEditor';
import { MONTH_NAMES } from '../dates';
import { AttendanceState } from '../states';
import { clearConnection, Connection } from '../storage';
import { colors, radius, spacing } from '../theme';

interface Props {
  conn: Connection;
  onClose: () => void;
  onUnauthorized: () => void;
  onDisconnect: () => void;
}

function formatDate(iso: string): string {
  const d = new Date(iso);
  return isNaN(d.getTime()) ? iso : d.toLocaleDateString();
}

export default function SettingsScreen({
  conn,
  onClose,
  onUnauthorized,
  onDisconnect,
}: Props) {
  const api = useMemo(
    () => new Api(conn, onUnauthorized),
    [conn, onUnauthorized],
  );

  const [settings, setSettings] = useState<Settings | null>(null);
  const [tokens, setTokens] = useState<TokenInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [tokenName, setTokenName] = useState('');
  const [creating, setCreating] = useState(false);
  const [newSecret, setNewSecret] = useState<string | null>(null);
  const [addingAccount, setAddingAccount] = useState(false);

  const load = useCallback(
    async (isRefresh = false) => {
      if (isRefresh) setRefreshing(true);
      else setLoading(true);
      setError(null);
      try {
        const [s, t] = await Promise.all([api.getSettings(), api.listTokens()]);
        setSettings(s);
        setTokens(t);
      } catch (e: any) {
        setError(e?.message ?? 'Failed to load settings.');
      } finally {
        setLoading(false);
        setRefreshing(false);
      }
    },
    [api],
  );

  useEffect(() => {
    load();
  }, [load]);

  function cycleSchedule(day: Weekday, next: AttendanceState) {
    if (!settings) return;
    const schedule = { ...settings.schedule, [day]: next };
    setSettings({ ...settings, schedule });
    api.updateSchedule(schedule).catch((e: any) => {
      if (isUnauthorized(e)) return;
      Alert.alert('Could not save schedule', e?.message ?? 'Please try again.');
      load(true);
    });
  }

  function setTrackingStart(month: number) {
    if (!settings || month === settings.trackingYearStartMonth) return;
    const prev = settings.trackingYearStartMonth;
    setSettings({ ...settings, trackingYearStartMonth: month });
    api.updateTrackingYearStartMonth(month).catch((e: any) => {
      if (isUnauthorized(e)) return;
      setSettings((s) => (s ? { ...s, trackingYearStartMonth: prev } : s));
      Alert.alert('Could not save', e?.message ?? 'Please try again.');
    });
  }

  async function addAccount() {
    setAddingAccount(true);
    try {
      const url = await api.getAccountLinkUrl();
      await Linking.openURL(url);
      Alert.alert(
        'Finish in your browser',
        'Complete the sign-in to link the account, then pull down here to refresh.',
      );
    } catch (e: any) {
      if (!isUnauthorized(e)) {
        Alert.alert('Could not start linking', e?.message ?? 'Please try again.');
      }
    } finally {
      setAddingAccount(false);
    }
  }

  async function createToken() {
    const name = tokenName.trim();
    if (!name) {
      Alert.alert('Name required', 'Enter a name for the token.');
      return;
    }
    setCreating(true);
    try {
      const secret = await api.createToken(name);
      setNewSecret(secret);
      setTokenName('');
      setTokens(await api.listTokens());
    } catch (e: any) {
      if (!isUnauthorized(e)) {
        Alert.alert('Could not create token', e?.message ?? 'Please try again.');
      }
    } finally {
      setCreating(false);
    }
  }

  function revoke(token: TokenInfo) {
    Alert.alert(
      'Revoke token',
      `Revoke "${token.name}"? Anything using it will stop working.`,
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Revoke',
          style: 'destructive',
          onPress: async () => {
            try {
              await api.revokeToken(token.tokenId);
              setTokens((ts) => ts.filter((t) => t.tokenId !== token.tokenId));
            } catch (e: any) {
              if (!isUnauthorized(e)) {
                Alert.alert('Could not revoke', e?.message ?? 'Please try again.');
              }
            }
          },
        },
      ],
    );
  }

  function signOut() {
    Alert.alert('Sign out', 'Sign out of Officetracker on this device?', [
      { text: 'Cancel', style: 'cancel' },
      {
        text: 'Sign out',
        style: 'destructive',
        onPress: async () => {
          await api.logout(); // best-effort server-side revoke
          await clearConnection();
          onDisconnect();
        },
      },
    ]);
  }

  return (
    <ScrollView
      style={styles.flex}
      contentContainerStyle={styles.content}
      keyboardShouldPersistTaps="handled"
      keyboardDismissMode="interactive"
      automaticallyAdjustKeyboardInsets
      refreshControl={
        <RefreshControl
          refreshing={refreshing}
          onRefresh={() => load(true)}
          tintColor={colors.textMuted}
        />
      }
    >
      <Header rightLabel="Done" onRightPress={onClose} />

      <View style={styles.body}>
      <Text style={styles.screenTitle}>Settings</Text>

      {loading ? (
        <View style={styles.loading}>
          <ActivityIndicator color={colors.textMuted} />
        </View>
      ) : error ? (
        <View style={styles.card}>
          <Text style={styles.errorText}>{error}</Text>
          <Pressable style={styles.button} onPress={() => load()}>
            <Text style={styles.buttonText}>Retry</Text>
          </Pressable>
        </View>
      ) : (
        <>
          {/* Accounts */}
          <Text style={styles.sectionLabel}>Accounts</Text>
          <View style={styles.card}>
            {settings && settings.linkedAccounts.length > 0 ? (
              settings.linkedAccounts.map((a, i) => (
                <View key={`${a.provider}-${i}`}>
                  {i > 0 && <View style={styles.hr} />}
                  <Text style={styles.fieldLabel}>{a.providerDisplay}</Text>
                  <Text style={styles.fieldValue}>{a.nickname || '—'}</Text>
                </View>
              ))
            ) : (
              <Text style={styles.muted}>No linked accounts.</Text>
            )}
          </View>
          <Pressable
            style={styles.button}
            onPress={addAccount}
            disabled={addingAccount}
          >
            {addingAccount ? (
              <ActivityIndicator color={colors.text} />
            ) : (
              <Text style={styles.buttonText}>Add account</Text>
            )}
          </Pressable>

          {/* Planned days */}
          <Text style={styles.sectionLabel}>Planned days</Text>
          <Text style={styles.hint}>
            Set the days you usually attend. They show as faded "planned" days on
            the calendar. Tap to cycle, long-press to go back.
          </Text>
          <View style={styles.card}>
            {settings && (
              <ScheduleEditor
                schedule={settings.schedule}
                onChange={cycleSchedule}
              />
            )}
            <View style={styles.legendWrap}>
              <Legend showPlanned={false} />
            </View>
          </View>

          {/* Tracking year */}
          <Text style={styles.sectionLabel}>Tracking year</Text>
          <Text style={styles.hint}>
            The month your reporting year starts on. Attendance is grouped into a
            12-month cycle from here.
          </Text>
          <View style={styles.card}>
            <ScrollView
              horizontal
              showsHorizontalScrollIndicator={false}
              contentContainerStyle={styles.monthRow}
              keyboardShouldPersistTaps="handled"
            >
              {MONTH_NAMES.map((name, i) => {
                const month = i + 1;
                const selected = settings?.trackingYearStartMonth === month;
                return (
                  <Pressable
                    key={month}
                    onPress={() => setTrackingStart(month)}
                    style={[styles.monthChip, selected && styles.monthChipSelected]}
                  >
                    <Text
                      style={[
                        styles.monthChipText,
                        selected && styles.monthChipTextSelected,
                      ]}
                    >
                      {name.slice(0, 3)}
                    </Text>
                  </Pressable>
                );
              })}
            </ScrollView>
          </View>

          {/* Developer tokens */}
          <Text style={styles.sectionLabel}>Developer tokens</Text>
          <Text style={styles.hint}>
            API tokens for scripts or the MCP server. Sent as a Bearer token.
          </Text>
          <View style={styles.card}>
            {tokens.length === 0 ? (
              <Text style={styles.muted}>No tokens yet.</Text>
            ) : (
              tokens.map((t, i) => (
                <View key={t.tokenId}>
                  {i > 0 && <View style={styles.hr} />}
                  <View style={styles.tokenRow}>
                    <View style={styles.tokenInfo}>
                      <Text style={styles.fieldValue}>{t.name}</Text>
                      <Text style={styles.muted}>
                        Created {formatDate(t.createdAt)}
                      </Text>
                    </View>
                    <Pressable onPress={() => revoke(t)} hitSlop={8}>
                      <Text style={styles.revoke}>Revoke</Text>
                    </Pressable>
                  </View>
                </View>
              ))
            )}
          </View>

          {newSecret ? (
            <View style={[styles.card, styles.secretCard]}>
              <Text style={styles.fieldLabel}>New token — copy it now</Text>
              <Text style={styles.secretValue} selectable>
                {newSecret}
              </Text>
              <Text style={styles.muted}>
                Long-press to copy. It won't be shown again.
              </Text>
              <Pressable style={styles.button} onPress={() => setNewSecret(null)}>
                <Text style={styles.buttonText}>Done</Text>
              </Pressable>
            </View>
          ) : (
            <View style={styles.createRow}>
              <TextInput
                style={styles.input}
                value={tokenName}
                onChangeText={setTokenName}
                placeholder="Token name"
                placeholderTextColor={colors.textFaint}
                autoCapitalize="none"
                editable={!creating}
              />
              <Pressable
                style={styles.createBtn}
                onPress={createToken}
                disabled={creating}
              >
                {creating ? (
                  <ActivityIndicator color="#ffffff" />
                ) : (
                  <Text style={styles.createBtnText}>Create</Text>
                )}
              </Pressable>
            </View>
          )}

          <Pressable style={[styles.button, styles.danger]} onPress={signOut}>
            <Text style={[styles.buttonText, styles.dangerText]}>Sign out</Text>
          </Pressable>
        </>
      )}
      </View>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1, backgroundColor: colors.surface },
  content: { paddingBottom: spacing.xl * 2 },
  body: { padding: spacing.lg },
  screenTitle: {
    fontSize: 24,
    fontWeight: '700',
    color: colors.text,
    marginTop: spacing.sm,
  },
  monthRow: { gap: spacing.sm, paddingVertical: 2 },
  monthChip: {
    paddingVertical: spacing.sm,
    paddingHorizontal: spacing.md,
    borderRadius: radius.md,
    borderWidth: 1,
    borderColor: colors.border,
  },
  monthChipSelected: {
    backgroundColor: colors.accent,
    borderColor: colors.accent,
  },
  monthChipText: { fontSize: 14, fontWeight: '600', color: colors.textMuted },
  monthChipTextSelected: { color: '#ffffff' },
  loading: { paddingVertical: spacing.xl * 2 },
  errorText: { color: colors.danger, marginBottom: spacing.md },
  sectionLabel: {
    fontSize: 17,
    fontWeight: '700',
    color: colors.text,
    marginTop: spacing.xl,
    marginBottom: spacing.sm,
  },
  hint: {
    fontSize: 13,
    color: colors.textFaint,
    lineHeight: 18,
    marginBottom: spacing.sm,
  },
  card: {
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: 10,
    padding: spacing.lg,
    backgroundColor: colors.surface,
  },
  fieldLabel: { fontSize: 12, color: colors.textMuted, marginBottom: 2 },
  fieldValue: { fontSize: 15, color: colors.text },
  muted: { fontSize: 13, color: colors.textFaint },
  hr: { height: 1, backgroundColor: colors.border, marginVertical: spacing.md },
  button: {
    marginTop: spacing.md,
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: radius.md,
    paddingVertical: spacing.md,
    alignItems: 'center',
  },
  buttonText: { fontSize: 15, fontWeight: '600', color: colors.text },
  legendWrap: { marginTop: spacing.lg },
  tokenRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
  },
  tokenInfo: { flex: 1, paddingRight: spacing.md },
  revoke: { fontSize: 14, fontWeight: '600', color: colors.danger },
  secretCard: { marginTop: spacing.md, backgroundColor: colors.fieldBg },
  secretValue: {
    fontSize: 13,
    color: colors.text,
    fontFamily: 'Courier',
    marginVertical: spacing.sm,
  },
  createRow: {
    flexDirection: 'row',
    gap: spacing.sm,
    marginTop: spacing.md,
  },
  input: {
    flex: 1,
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: radius.md,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.md,
    fontSize: 15,
    color: colors.text,
    backgroundColor: colors.fieldBg,
  },
  createBtn: {
    backgroundColor: colors.accent,
    borderRadius: radius.md,
    paddingHorizontal: spacing.lg,
    alignItems: 'center',
    justifyContent: 'center',
  },
  createBtnText: { color: '#ffffff', fontSize: 15, fontWeight: '600' },
  danger: { marginTop: spacing.xl, borderColor: '#fecaca' },
  dangerText: { color: colors.danger },
});
