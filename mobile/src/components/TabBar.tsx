import { Ionicons } from '@expo/vector-icons';
import React from 'react';
import { Pressable, StyleSheet, Text, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { colors, spacing } from '../theme';

export type Tab = 'calendar' | 'report' | 'settings';

type IconName = keyof typeof Ionicons.glyphMap;

const TABS: { key: Tab; label: string; icon: IconName; iconActive: IconName }[] = [
  { key: 'calendar', label: 'Calendar', icon: 'calendar-outline', iconActive: 'calendar' },
  { key: 'report', label: 'Report', icon: 'stats-chart-outline', iconActive: 'stats-chart' },
  { key: 'settings', label: 'Settings', icon: 'settings-outline', iconActive: 'settings' },
];

interface Props {
  active: Tab;
  onSelect: (tab: Tab) => void;
}

// Native-style bottom tab bar in the brand block colour: icon over a small
// label, the active tab accented.
export default function TabBar({ active, onSelect }: Props) {
  const insets = useSafeAreaInsets();
  return (
    <View style={[styles.bar, { paddingBottom: insets.bottom }]}>
      {TABS.map((t) => {
        const selected = active === t.key;
        return (
          <Pressable
            key={t.key}
            style={({ pressed }) => [styles.tab, pressed && styles.pressed]}
            onPress={() => onSelect(t.key)}
          >
            <Ionicons
              name={selected ? t.iconActive : t.icon}
              size={24}
              color={selected ? colors.accent : colors.textFaint}
            />
            <Text style={[styles.label, selected && styles.labelActive]}>
              {t.label}
            </Text>
          </Pressable>
        );
      })}
    </View>
  );
}

const styles = StyleSheet.create({
  bar: {
    flexDirection: 'row',
    backgroundColor: colors.navBg,
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: colors.border,
  },
  tab: {
    flex: 1,
    alignItems: 'center',
    paddingTop: spacing.sm,
    paddingBottom: spacing.xs,
    gap: 2,
  },
  pressed: { opacity: 0.6 },
  label: {
    fontSize: 11,
    color: colors.textFaint,
  },
  labelActive: {
    color: colors.accent,
    fontWeight: '600',
  },
});
