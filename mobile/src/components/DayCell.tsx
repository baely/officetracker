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
}

function DayCell({ day, state, isToday, onPress, onLongPress }: Props) {
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
        style={({ pressed }) => [
          styles.day,
          filled && { backgroundColor: look.bg },
          look.scheduled && styles.scheduled,
          isToday && styles.today,
          pressed && styles.pressed,
        ]}
      >
        <Text style={[styles.dayNum, { color: look.fg }]}>{day}</Text>
      </Pressable>
    </View>
  );
}

const styles = StyleSheet.create({
  cell: {
    flex: 1,
    aspectRatio: 1,
    padding: 2,
  },
  day: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    borderRadius: radius.md,
    borderWidth: 1,
    borderColor: colors.border,
  },
  scheduled: {
    borderStyle: 'dashed',
    borderColor: colors.borderStrong,
  },
  today: {
    borderWidth: 2,
    borderColor: colors.todayRing,
  },
  pressed: {
    opacity: 0.6,
  },
  dayNum: {
    fontSize: 15,
    fontWeight: '500',
  },
});

export default React.memo(DayCell);
