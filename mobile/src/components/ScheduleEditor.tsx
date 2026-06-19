import React from 'react';
import { Pressable, StyleSheet, Text, View } from 'react-native';
import { SchedulePreferences, Weekday, WEEKDAYS_LOWER } from '../api';
import { appearance, AttendanceState, cycleState } from '../states';
import { colors, radius, spacing } from '../theme';

const LABELS: Record<Weekday, string> = {
  monday: 'Mon',
  tuesday: 'Tue',
  wednesday: 'Wed',
  thursday: 'Thu',
  friday: 'Fri',
  saturday: 'Sat',
  sunday: 'Sun',
};

interface Props {
  schedule: SchedulePreferences;
  onChange: (day: Weekday, next: AttendanceState) => void;
  // Display-only: render the schedule without responding to taps (read-only server).
  disabled?: boolean;
}

export default function ScheduleEditor({ schedule, onChange, disabled }: Props) {
  return (
    <View style={styles.row}>
      {WEEKDAYS_LOWER.map((day) => {
        const state = schedule[day];
        const look = appearance(state);
        const filled = state !== AttendanceState.Untracked;
        return (
          <View key={day} style={styles.col}>
            <Text style={styles.label}>{LABELS[day]}</Text>
            <Pressable
              onPress={() => onChange(day, cycleState(state, 1))}
              onLongPress={() => onChange(day, cycleState(state, -1))}
              delayLongPress={250}
              disabled={disabled}
              style={({ pressed }) => [
                styles.cell,
                filled && { backgroundColor: look.bg },
                pressed && !disabled && styles.pressed,
              ]}
            />
          </View>
        );
      })}
    </View>
  );
}

const styles = StyleSheet.create({
  row: {
    flexDirection: 'row',
  },
  col: {
    flex: 1,
    alignItems: 'center',
    paddingHorizontal: 2,
  },
  label: {
    fontSize: 11,
    fontWeight: '600',
    color: colors.textFaint,
    textTransform: 'uppercase',
    marginBottom: spacing.xs,
  },
  cell: {
    width: '100%',
    aspectRatio: 1,
    borderRadius: radius.md,
    borderWidth: 1,
    borderColor: colors.border,
  },
  pressed: { opacity: 0.6 },
});
