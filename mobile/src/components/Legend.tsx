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
    gap: spacing.md,
  },
  item: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
  },
  swatch: {
    width: 12,
    height: 12,
    borderRadius: 3,
  },
  scheduledSwatch: {
    backgroundColor: 'transparent',
    borderWidth: 1,
    borderStyle: 'dashed',
    borderColor: colors.borderStrong,
  },
  label: {
    fontSize: 12,
    color: colors.textMuted,
  },
});

export default React.memo(Legend);
