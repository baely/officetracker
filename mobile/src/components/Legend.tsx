import React from 'react';
import { StyleSheet, Text, View } from 'react-native';
import { appearance, AttendanceState } from '../states';
import { colors, spacing } from '../theme';

const ITEMS = [
  AttendanceState.WorkFromHome,
  AttendanceState.Office,
  AttendanceState.Other,
];

function Legend({ showPlanned = true }: { showPlanned?: boolean }) {
  return (
    <View style={styles.row}>
      {ITEMS.map((s) => {
        const look = appearance(s);
        return (
          <View key={s} style={styles.item}>
            <View style={[styles.swatch, { backgroundColor: look.bg }]} />
            <Text style={styles.label}>{look.label}</Text>
          </View>
        );
      })}
      {showPlanned && (
        <View style={styles.item}>
          <View style={[styles.swatch, styles.scheduledSwatch]} />
          <Text style={styles.label}>Planned</Text>
        </View>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  row: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    justifyContent: 'center',
    rowGap: spacing.sm,
    columnGap: spacing.lg,
  },
  item: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: spacing.sm,
  },
  // Square swatch with a border, matching the web .legend-color.
  swatch: {
    width: 16,
    height: 16,
    borderWidth: 1,
    borderColor: colors.border,
  },
  scheduledSwatch: {
    backgroundColor: 'transparent',
    borderStyle: 'dashed',
    borderColor: colors.borderStrong,
  },
  label: {
    fontSize: 13,
    color: colors.textMuted,
  },
});

export default React.memo(Legend);
