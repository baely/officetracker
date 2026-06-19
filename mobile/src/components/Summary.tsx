import React from 'react';
import { StyleSheet, Text, View } from 'react-native';
import { formatPercent, Stats } from '../stats';
import { colors, spacing } from '../theme';

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
        <View
          key={i}
          style={[
            styles.cell,
            { flex: COLS[i] },
            i < cells.length - 1 && styles.cellDivider,
            header && styles.headerCell,
          ]}
        >
          <Text style={[styles.cellText, header && styles.headerText]}>{c}</Text>
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
    borderWidth: 1,
    borderColor: colors.border,
    overflow: 'hidden',
  },
  row: {
    flexDirection: 'row',
    borderBottomWidth: 1,
    borderBottomColor: colors.border,
  },
  rowLast: {
    borderBottomWidth: 0,
  },
  cell: {
    paddingVertical: spacing.sm,
    paddingHorizontal: spacing.sm,
    justifyContent: 'center',
  },
  cellDivider: {
    borderRightWidth: 1,
    borderRightColor: colors.border,
  },
  headerCell: {
    backgroundColor: colors.tableHeaderBg,
  },
  cellText: {
    fontSize: 13,
    color: colors.textMuted,
  },
  headerText: {
    fontWeight: '700',
    color: colors.text,
  },
});

export default React.memo(Summary);
