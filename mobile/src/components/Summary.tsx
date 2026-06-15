import React from 'react';
import { StyleSheet, Text, View } from 'react-native';
import { formatPercent, Stats } from '../stats';
import { colors, radius, spacing } from '../theme';

interface Props {
  monthLabel: string;
  month: Stats;
  year: Stats;
}

function Cell({ value, label }: { value: string; label: string }) {
  return (
    <View style={styles.statCell}>
      <Text style={styles.statValue}>{value}</Text>
      <Text style={styles.statLabel}>{label}</Text>
    </View>
  );
}

function Summary({ monthLabel, month, year }: Props) {
  return (
    <View style={styles.card}>
      <View style={styles.row}>
        <Cell value={`${month.office}/${month.total}`} label={`${monthLabel} office days`} />
        <View style={styles.divider} />
        <Cell value={formatPercent(month.percent)} label="this month" />
      </View>
      <View style={styles.hr} />
      <View style={styles.row}>
        <Cell value={`${year.office}/${year.total}`} label="office days this year" />
        <View style={styles.divider} />
        <Cell value={formatPercent(year.percent)} label="year to date" />
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  card: {
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: radius.lg,
    padding: spacing.lg,
    backgroundColor: colors.surface,
  },
  row: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  statCell: {
    flex: 1,
    alignItems: 'center',
  },
  statValue: {
    fontSize: 24,
    fontWeight: '700',
    color: colors.text,
  },
  statLabel: {
    marginTop: 2,
    fontSize: 12,
    color: colors.textMuted,
    textAlign: 'center',
  },
  divider: {
    width: 1,
    alignSelf: 'stretch',
    backgroundColor: colors.border,
  },
  hr: {
    height: 1,
    backgroundColor: colors.border,
    marginVertical: spacing.md,
  },
});

export default React.memo(Summary);
