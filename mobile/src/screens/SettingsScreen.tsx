import React from 'react';
import {
  Alert,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { clearConnection, Connection } from '../storage';
import { colors, radius, spacing } from '../theme';

interface Props {
  conn: Connection;
  onClose: () => void;
  onSignInAgain: () => void;
  onDisconnect: () => void;
}

export default function SettingsScreen({
  conn,
  onClose,
  onSignInAgain,
  onDisconnect,
}: Props) {
  function disconnect() {
    Alert.alert('Sign out', 'Sign out of Office Tracker on this device?', [
      { text: 'Cancel', style: 'cancel' },
      {
        text: 'Sign out',
        style: 'destructive',
        onPress: async () => {
          await clearConnection();
          onDisconnect();
        },
      },
    ]);
  }

  return (
    <ScrollView style={styles.flex} contentContainerStyle={styles.content}>
      <View style={styles.header}>
        <Text style={styles.title}>Settings</Text>
        <Pressable onPress={onClose} hitSlop={10}>
          <Text style={styles.done}>Done</Text>
        </Pressable>
      </View>

      <Text style={styles.sectionLabel}>Account</Text>
      <View style={styles.card}>
        <Text style={styles.fieldLabel}>Server</Text>
        <Text style={styles.fieldValue}>{conn.baseUrl}</Text>
        <View style={styles.hr} />
        <Text style={styles.fieldLabel}>Status</Text>
        <Text style={styles.fieldValue}>
          {conn.token ? 'Signed in' : 'Not signed in'}
        </Text>
      </View>
      <Pressable style={styles.button} onPress={onSignInAgain}>
        <Text style={styles.buttonText}>Sign in again</Text>
      </Pressable>

      <Pressable
        style={[styles.button, styles.danger]}
        onPress={disconnect}
      >
        <Text style={[styles.buttonText, styles.dangerText]}>Sign out</Text>
      </Pressable>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1, backgroundColor: colors.bg },
  content: { padding: spacing.lg, paddingBottom: spacing.xl * 2 },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginBottom: spacing.lg,
  },
  title: { fontSize: 24, fontWeight: '700', color: colors.text },
  done: { fontSize: 16, fontWeight: '600', color: colors.text },
  sectionLabel: {
    fontSize: 12,
    fontWeight: '600',
    color: colors.textMuted,
    textTransform: 'uppercase',
    letterSpacing: 0.5,
    marginTop: spacing.lg,
    marginBottom: spacing.sm,
  },
  card: {
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: radius.lg,
    padding: spacing.lg,
  },
  fieldLabel: { fontSize: 12, color: colors.textMuted, marginBottom: 2 },
  fieldValue: { fontSize: 15, color: colors.text },
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
  danger: { marginTop: spacing.xl, borderColor: '#fecaca' },
  dangerText: { color: colors.danger },
});
