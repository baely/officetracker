import React from 'react';
import { StyleSheet, Text, View } from 'react-native';
import { MonthDays } from '../api';
import { monthGrid, WEEKDAYS } from '../dates';
import { AttendanceState } from '../states';
import { colors, spacing } from '../theme';
import DayCell from './DayCell';

interface Props {
  year: number;
  month: number; // 1-12
  days: MonthDays;
  onCycle: (day: number, direction: 1 | -1) => void;
}

function Calendar({ year, month, days, onCycle }: Props) {
  const weeks = monthGrid(year, month);

  return (
    <View>
      <View style={styles.weekdayRow}>
        {WEEKDAYS.map((w) => (
          <Text key={w} style={styles.weekday}>
            {w}
          </Text>
        ))}
      </View>
      {weeks.map((week, wi) => (
        <View key={wi} style={styles.week}>
          {week.map((cell, ci) => (
            <DayCell
              key={ci}
              day={cell.day}
              isToday={cell.isToday}
              state={
                cell.day != null
                  ? days[cell.day] ?? AttendanceState.Untracked
                  : AttendanceState.Untracked
              }
              onPress={() => cell.day != null && onCycle(cell.day, 1)}
              onLongPress={() => cell.day != null && onCycle(cell.day, -1)}
            />
          ))}
        </View>
      ))}
    </View>
  );
}

const styles = StyleSheet.create({
  weekdayRow: {
    flexDirection: 'row',
    marginBottom: spacing.xs,
  },
  weekday: {
    flex: 1,
    textAlign: 'center',
    fontSize: 12,
    fontWeight: '600',
    color: colors.textFaint,
    textTransform: 'uppercase',
    letterSpacing: 0.5,
  },
  week: {
    flexDirection: 'row',
  },
});

export default React.memo(Calendar);
