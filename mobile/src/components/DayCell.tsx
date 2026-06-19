import React from 'react';
import { Pressable, StyleSheet, Text, View } from 'react-native';
import { appearance, AttendanceState } from '../states';
import { colors, radius } from '../theme';

interface Props {
  day: number | null;
  state: AttendanceState;
  isToday: boolean;
  onPress: () => void;
  onLongPress: () => void;
  // Display-only: render the day without responding to taps (read-only server).
  disabled?: boolean;
}

function DayCell({ day, state, isToday, onPress, onLongPress, disabled }: Props) {
  if (day === null) {
    return <View style={styles.cell} />;
  }

  const look = appearance(state);
  const filled = state !== AttendanceState.Untracked;

  return (
    <View style={styles.cell}>
      <Pressable
        onPress={onPress}
        onLongPress={onLongPress}
        delayLongPress={250}
        disabled={disabled}
        style={({ pressed }) => [
          styles.day,
          filled && { backgroundColor: look.bg },
          look.scheduled && styles.scheduled,
          isToday && styles.today,
          pressed && !disabled && styles.pressed,
        ]}
      >
        <Text
          style={[styles.dayNum, { color: look.fg }, isToday && styles.todayNum]}
        >
          {day}
        </Text>
      </Pressable>
    </View>
  );
}

const styles = StyleSheet.create({
  cell: {
    flex: 1,
    aspectRatio: 1,
    padding: 3,
  },
  day: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    borderRadius: radius.md,
    borderWidth: 1,
    borderColor: colors.border,
    backgroundColor: colors.surface, // untracked days are white, like the web
  },
  scheduled: {
    borderStyle: 'dashed',
    borderColor: colors.borderStrong,
  },
  today: {
    borderWidth: 3,
    borderColor: colors.todayRing,
  },
  pressed: {
    opacity: 0.6,
  },
  dayNum: {
    fontSize: 15,
    fontWeight: '500',
  },
  todayNum: {
    fontWeight: '700',
  },
});

export default React.memo(DayCell);
