import React from 'react';
import { Pressable, StyleSheet, Text, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { colors, spacing } from '../theme';

export type Tab = 'calendar' | 'report' | 'settings';

const TABS: { key: Tab; label: string }[] = [
  { key: 'calendar', label: 'Calendar' },
  { key: 'report', label: 'Report' },
  { key: 'settings', label: 'Settings' },
];

interface Props {
  active: Tab;
  onSelect: (tab: Tab) => void;
}

// Flat bottom bar in the brand block colour, mirroring the web navbar's
// plain-language links; the active tab is accented.
export default function TabBar({ active, onSelect }: Props) {
  const insets = useSafeAreaInsets();
  return (
    <View style={[styles.bar, { paddingBottom: insets.bottom }]}>
      {TABS.map((t) => (
        <Pressable
          key={t.key}
          style={({ pressed }) => [styles.tab, pressed && styles.pressed]}
          onPress={() => onSelect(t.key)}
        >
          <Text style={[styles.label, active === t.key && styles.labelActive]}>
            {t.label}
          </Text>
        </Pressable>
      ))}
    </View>
  );
}

const styles = StyleSheet.create({
  bar: {
    flexDirection: 'row',
    backgroundColor: colors.navBg,
  },
  tab: {
    flex: 1,
    alignItems: 'center',
    paddingVertical: spacing.md,
  },
  pressed: { opacity: 0.6 },
  label: {
    fontSize: 15,
    color: colors.textMuted,
  },
  labelActive: {
    color: colors.accent,
    fontWeight: '700',
  },
});
