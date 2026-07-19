import { Ionicons } from '@expo/vector-icons';
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  ActivityIndicator,
  Pressable,
  RefreshControl,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { Api, MonthDays } from '../api';
import Summary, { SummaryRow } from '../components/Summary';
import {
  calendarYearForMonth,
  DEFAULT_TRACKING_YEAR_START_MONTH,
  MONTH_NAMES,
  thisMonth,
  trackingMonthOrder,
  trackingYear,
} from '../dates';
import { monthStats, yearStats } from '../stats';
import { Connection } from '../storage';
import { colors, radius, spacing } from '../theme';

interface Props {
  conn: Connection;
  onUnauthorized: () => void;
}

// The yearly report, mirroring the web /report page: prev/next tracking-year
// navigation above the summary table.
export default function ReportScreen({ conn, onUnauthorized }: Props) {
  const api = useMemo(
    () => new Api(conn, onUnauthorized),
    [conn, onUnauthorized],
  );

  const [startMonth, setStartMonth] = useState(DEFAULT_TRACKING_YEAR_START_MONTH);
  // null = the current tracking year, which can shift once the user's start
  // month loads; set explicitly when the user navigates.
  const [fySel, setFySel] = useState<number | null>(null);
  const now = thisMonth();
  const fy = fySel ?? trackingYear(now.year, now.month, startMonth);

  const [yearData, setYearData] = useState<Record<number, MonthDays>>({});
  const [loadedFy, setLoadedFy] = useState<number | null>(null);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Best-effort fetch of the start month. Plain Api (no onUnauthorized) so an
  // older server that 401s /settings just keeps the default instead of logging out.
  useEffect(() => {
    let cancelled = false;
    new Api(conn)
      .getSettings()
      .then((s) => {
        if (!cancelled) setStartMonth(s.trackingYearStartMonth);
      })
      .catch(() => {});
    return () => {
      cancelled = true;
    };
  }, [conn]);

  const load = useCallback(
    async (targetFy: number, isRefresh = false) => {
      if (isRefresh) setRefreshing(true);
      else setLoading(true);
      setError(null);
      try {
        setYearData(await api.getYear(targetFy));
        setLoadedFy(targetFy);
      } catch (e: any) {
        setError(e?.message ?? 'Failed to load data.');
      } finally {
        setLoading(false);
        setRefreshing(false);
      }
    },
    [api],
  );

  useEffect(() => {
    if (loadedFy !== fy) {
      load(fy);
    }
  }, [fy, loadedFy, load]);

  const total = useMemo(() => yearStats(yearData), [yearData]);

  // One row per tracked month, ordered start-month-first, mirroring the web
  // report table.
  const rows = useMemo<SummaryRow[]>(() => {
    return Object.keys(yearData)
      .map(Number)
      .map((m) => {
        const s = monthStats(yearData[m] ?? {});
        const calYear = calendarYearForMonth(m, fy, startMonth);
        return { label: `${MONTH_NAMES[m - 1]} ${calYear}`, ...s, month: m };
      })
      .filter((r) => r.total > 0)
      .sort(
        (a, b) =>
          trackingMonthOrder(a.month, startMonth) -
          trackingMonthOrder(b.month, startMonth),
      );
  }, [yearData, fy, startMonth]);

  return (
    <View style={styles.screen}>
      <ScrollView
        style={styles.flex}
        contentContainerStyle={styles.content}
        refreshControl={
          <RefreshControl
            refreshing={refreshing}
            onRefresh={() => load(fy, true)}
            tintColor={colors.textMuted}
          />
        }
      >
        <View style={styles.body}>
          <View style={styles.titleRow}>
            <Text style={styles.title}>Report</Text>
            <View style={styles.yearNav}>
              <Pressable
                onPress={() => setFySel(fy - 1)}
                style={({ pressed }) => [styles.navBtn, pressed && styles.pressed]}
                hitSlop={8}
              >
                <Ionicons name="chevron-back" size={20} color={colors.text} />
              </Pressable>
              {/* Tap the year to jump back to the current tracking year. */}
              <Pressable onPress={() => setFySel(null)} hitSlop={8}>
                <Text style={styles.year}>{fy}</Text>
              </Pressable>
              <Pressable
                onPress={() => setFySel(fy + 1)}
                style={({ pressed }) => [styles.navBtn, pressed && styles.pressed]}
                hitSlop={8}
              >
                <Ionicons name="chevron-forward" size={20} color={colors.text} />
              </Pressable>
            </View>
          </View>

          {loading ? (
            <View style={styles.loading}>
              <ActivityIndicator color={colors.textMuted} />
            </View>
          ) : error ? (
            <View style={styles.errorBox}>
              <Text style={styles.errorText}>{error}</Text>
              <Pressable onPress={() => load(fy)} style={styles.retry}>
                <Text style={styles.retryText}>Retry</Text>
              </Pressable>
            </View>
          ) : (
            <View style={styles.section}>
              <Summary rows={rows} total={total} />
            </View>
          )}
        </View>
      </ScrollView>
    </View>
  );
}

const styles = StyleSheet.create({
  screen: { flex: 1, backgroundColor: colors.surface },
  flex: { flex: 1, backgroundColor: colors.surface },
  content: { paddingBottom: spacing.xl * 2 },
  body: { padding: spacing.lg },
  titleRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginTop: spacing.sm,
    marginBottom: spacing.lg,
  },
  title: {
    fontSize: 28,
    fontWeight: '700',
    color: colors.text,
  },
  yearNav: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: spacing.sm,
  },
  navBtn: {
    width: 36,
    height: 32,
    alignItems: 'center',
    justifyContent: 'center',
    borderRadius: radius.md,
    backgroundColor: colors.cellBg,
  },
  year: {
    fontSize: 17,
    fontWeight: '600',
    color: colors.text,
    fontVariant: ['tabular-nums'],
  },
  pressed: { opacity: 0.6 },
  loading: { paddingVertical: spacing.xl * 2 },
  errorBox: {
    paddingVertical: spacing.xl,
    alignItems: 'center',
  },
  errorText: {
    color: colors.danger,
    textAlign: 'center',
    marginBottom: spacing.md,
  },
  retry: {
    borderRadius: radius.md,
    backgroundColor: colors.cellBg,
    paddingVertical: spacing.sm,
    paddingHorizontal: spacing.lg,
  },
  retryText: {
    color: colors.text,
    fontWeight: '600',
  },
  section: { marginTop: spacing.xs },
});
