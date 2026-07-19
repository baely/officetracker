import React from 'react';
import { StyleSheet, Text, View } from 'react-native';
import { MonthDays } from '../api';
import { targetProgress } from '../stats';
import { colors, radius, spacing } from '../theme';

interface Props {
  days: MonthDays;
  targetPercent: number; // 0 = no target set
  year: number;
  month: number; // 1-12
}

// Monthly attendance target progress for the viewed month, mirroring the web
// form page's target box: days-needed first, then tracked progress.
function TargetBox({ days, targetPercent, year, month }: Props) {
  if (targetPercent <= 0) {
    return (
      <View style={styles.box}>
        <Text style={styles.line}>
          No monthly attendance target set. You can set one in Settings.
        </Text>
      </View>
    );
  }

  const p = targetProgress(days, targetPercent, year, month);
  return (
    <View style={styles.box}>
      <Text style={styles.line}>
        {p.needed > 0 ? (
          <>
            <Text style={styles.num}>{p.needed}</Text> more office{' '}
            {p.needed === 1 ? 'day' : 'days'} needed this month.
          </>
        ) : (
          'Target met for this month.'
        )}
      </Text>
      <Text style={styles.line}>
        In office <Text style={styles.num}>{p.office}</Text> of{' '}
        <Text style={styles.num}>{p.total}</Text> tracked days (
        <Text style={styles.num}>{p.percent.toFixed(1)}%</Text>).
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  box: {
    backgroundColor: colors.cellBg,
    borderRadius: radius.lg,
    padding: spacing.lg,
    gap: spacing.xs,
  },
  line: {
    fontSize: 14,
    color: colors.text,
    lineHeight: 20,
  },
  num: {
    fontWeight: '600',
  },
});

export default React.memo(TargetBox);
