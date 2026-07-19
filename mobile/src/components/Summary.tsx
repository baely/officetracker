import React from 'react';
import { StyleSheet, Text, View } from 'react-native';
import { formatPercent, Stats } from '../stats';
import { colors, radius, spacing } from '../theme';

// One row per tracking-year month, mirroring the web summary table.
export interface SummaryRow {
  label: string; // e.g. "October 2025"
  office: number;
  total: number;
  percent: number;
}

interface Props {
  rows: SummaryRow[];
  total: Stats; // aggregate, for the headline
}

// Column flex weights: a wide month column, three equal numeric columns.
const COLS = [2, 1, 1, 1];

function Row({
  cells,
  header,
  last,
}: {
  cells: string[];
  header?: boolean;
  last?: boolean;
}) {
  return (
    <View style={[styles.row, last && styles.rowLast]}>
      {cells.map((c, i) => (
        <View key={i} style={[styles.cell, { flex: COLS[i] }]}>
          <Text
            style={[
              styles.cellText,
              i > 0 && styles.numText,
              header && styles.headerText,
            ]}
          >
            {c}
          </Text>
        </View>
      ))}
    </View>
  );
}

function Summary({ rows, total }: Props) {
  return (
    <View>
      <Text style={styles.headline}>
        Present in office for {total.office} out of {total.total} days. (
        {formatPercent(total.percent)})
      </Text>
      <View style={styles.table}>
        <Row header cells={['Month', 'Present', 'Total', 'Percent']} />
        {rows.length === 0 ? (
          <Row last cells={['No tracked days yet.', '', '', '']} />
        ) : (
          rows.map((r, i) => (
            <Row
              key={r.label}
              last={i === rows.length - 1}
              cells={[
                r.label,
                String(r.office),
                String(r.total),
                formatPercent(r.percent),
              ]}
            />
          ))
        )}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  headline: {
    fontSize: 15,
    color: colors.text,
    marginBottom: spacing.md,
    lineHeight: 21,
  },
  table: {
    backgroundColor: colors.cellBg,
    borderRadius: radius.lg,
    paddingHorizontal: spacing.xs,
    overflow: 'hidden',
  },
  row: {
    flexDirection: 'row',
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: colors.border,
  },
  rowLast: {
    borderBottomWidth: 0,
  },
  cell: {
    paddingVertical: spacing.md,
    paddingHorizontal: spacing.sm,
    justifyContent: 'center',
  },
  cellText: {
    fontSize: 14,
    color: colors.textMuted,
  },
  numText: {
    textAlign: 'right',
    fontVariant: ['tabular-nums'],
  },
  headerText: {
    fontWeight: '600',
    color: colors.text,
  },
});

export default React.memo(Summary);
