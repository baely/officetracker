import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  ActivityIndicator,
  Alert,
  Image,
  Pressable,
  RefreshControl,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  View,
} from 'react-native';
import { Api, MonthDays } from '../api';
import Calendar from '../components/Calendar';
import Legend from '../components/Legend';
import Summary from '../components/Summary';
import {
  addMonths,
  fiscalYear,
  formatMonthYear,
  MONTH_NAMES,
  thisMonth,
  ViewMonth,
} from '../dates';
import { monthStats, yearStats } from '../stats';
import { AttendanceState, cycleState } from '../states';
import { Connection } from '../storage';
import { colors, fonts, radius, spacing } from '../theme';

interface Props {
  conn: Connection;
  onOpenSettings: () => void;
}

export default function CalendarScreen({ conn, onOpenSettings }: Props) {
  const api = useMemo(() => new Api(conn), [conn]);

  const [view, setView] = useState<ViewMonth>(thisMonth());
  const fy = fiscalYear(view.year, view.month);

  // Attendance for the loaded fiscal year, keyed month (1-12) -> day -> state.
  const [yearData, setYearData] = useState<Record<number, MonthDays>>({});
  const [notes, setNotes] = useState<Record<number, string>>({});
  const [loadedFy, setLoadedFy] = useState<number | null>(null);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Local, possibly-unsaved note text for the current month.
  const [noteText, setNoteText] = useState('');

  const load = useCallback(
    async (targetFy: number, isRefresh = false) => {
      if (isRefresh) setRefreshing(true);
      else setLoading(true);
      setError(null);
      try {
        const [year, yearNotes] = await Promise.all([
          api.getYear(targetFy),
          api.getNotes(targetFy),
        ]);
        setYearData(year);
        setNotes(yearNotes);
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

  // Load whenever we cross into a different fiscal year.
  useEffect(() => {
    if (loadedFy !== fy) {
      load(fy);
    }
  }, [fy, loadedFy, load]);

  // Keep the note editor in sync with the selected month.
  useEffect(() => {
    setNoteText(notes[view.month] ?? '');
  }, [notes, view.month]);

  const days: MonthDays = yearData[view.month] ?? {};

  const onCycle = useCallback(
    (day: number, direction: 1 | -1) => {
      const current = days[day] ?? AttendanceState.Untracked;
      const next = cycleState(current, direction);

      // Optimistic update.
      setYearData((prev) => {
        const month = { ...(prev[view.month] ?? {}) };
        month[day] = next;
        return { ...prev, [view.month]: month };
      });

      api
        .putDay(view.year, view.month, day, next)
        .then(() => {
          // Clearing a day can reveal a scheduled (planned) state computed by
          // the server, so refetch the year to pick it up.
          if (next === AttendanceState.Untracked) {
            return api.getYear(fy).then(setYearData);
          }
        })
        .catch((e: any) => {
          // Revert on failure.
          setYearData((prev) => {
            const month = { ...(prev[view.month] ?? {}) };
            month[day] = current;
            return { ...prev, [view.month]: month };
          });
          Alert.alert('Could not save', e?.message ?? 'Please try again.');
        });
    },
    [api, days, view, fy],
  );

  const saveNote = useCallback(() => {
    const trimmed = noteText;
    if ((notes[view.month] ?? '') === trimmed) return;
    setNotes((prev) => ({ ...prev, [view.month]: trimmed }));
    api.putNote(view.year, view.month, trimmed).catch((e: any) => {
      Alert.alert('Could not save note', e?.message ?? 'Please try again.');
    });
  }, [api, noteText, notes, view]);

  const go = (delta: number) => setView((v) => addMonths(v, delta));
  const goToday = () => setView(thisMonth());

  const month = monthStats(days);
  const year = yearStats(yearData);
  const isThisMonth =
    view.year === thisMonth().year && view.month === thisMonth().month;

  return (
    <ScrollView
      style={styles.flex}
      contentContainerStyle={styles.content}
      keyboardShouldPersistTaps="handled"
      refreshControl={
        <RefreshControl
          refreshing={refreshing}
          onRefresh={() => load(fy, true)}
          tintColor={colors.textMuted}
        />
      }
    >
      <View style={styles.brandBar}>
        <View style={styles.brandLeft}>
          <Image
            source={require('../../assets/office-building.png')}
            style={styles.brandIcon}
            resizeMode="contain"
          />
          <Text style={styles.wordmark}>Officetracker</Text>
        </View>
        <Pressable
          onPress={onOpenSettings}
          hitSlop={10}
          style={({ pressed }) => [styles.gear, pressed && styles.pressed]}
        >
          <Text style={styles.gearText}>⚙</Text>
        </Pressable>
      </View>

      <View style={styles.titleBlock}>
        <Text style={styles.month}>{MONTH_NAMES[view.month - 1]}</Text>
        <Text style={styles.year}>{view.year}</Text>
      </View>

      <View style={styles.nav}>
        <Pressable
          onPress={() => go(-1)}
          style={({ pressed }) => [styles.navBtn, pressed && styles.pressed]}
          hitSlop={8}
        >
          <Text style={styles.navText}>‹</Text>
        </Pressable>
        <Pressable
          onPress={goToday}
          style={({ pressed }) => [styles.todayBtn, pressed && styles.pressed]}
        >
          <Text
            style={[styles.todayBtnText, isThisMonth && styles.todayBtnTextActive]}
          >
            Today
          </Text>
        </Pressable>
        <Pressable
          onPress={() => go(1)}
          style={({ pressed }) => [styles.navBtn, pressed && styles.pressed]}
          hitSlop={8}
        >
          <Text style={styles.navText}>›</Text>
        </Pressable>
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
        <>
          <Calendar
            year={view.year}
            month={view.month}
            days={days}
            onCycle={onCycle}
          />

          <Text style={styles.tip}>Tap to cycle · long-press to go back</Text>
          <View style={styles.legendWrap}>
            <Legend />
          </View>

          <View style={styles.section}>
            <Summary
              monthLabel={MONTH_NAMES[view.month - 1]}
              month={month}
              year={year}
            />
          </View>

          <View style={styles.section}>
            <Text style={styles.sectionLabel}>Notes</Text>
            <TextInput
              style={styles.notes}
              value={noteText}
              onChangeText={setNoteText}
              onBlur={saveNote}
              placeholder={`Notes for ${formatMonthYear(view)}…`}
              placeholderTextColor={colors.textFaint}
              multiline
              textAlignVertical="top"
            />
          </View>
        </>
      )}
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1, backgroundColor: colors.bg },
  content: {
    padding: spacing.lg,
    paddingBottom: spacing.xl * 2,
  },
  brandBar: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
  },
  brandLeft: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: spacing.sm,
  },
  brandIcon: {
    width: 26,
    height: 26,
  },
  wordmark: {
    fontSize: 20,
    fontFamily: fonts.wordmark,
    color: colors.accent,
  },
  titleBlock: {
    flexDirection: 'row',
    alignItems: 'baseline',
    gap: spacing.sm,
    marginTop: spacing.lg,
  },
  month: {
    fontSize: 26,
    fontWeight: '700',
    color: colors.text,
  },
  year: {
    fontSize: 26,
    fontWeight: '300',
    color: colors.textFaint,
  },
  gear: {
    padding: spacing.xs,
  },
  gearText: {
    fontSize: 22,
    color: colors.textMuted,
  },
  nav: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginTop: spacing.md,
    marginBottom: spacing.lg,
  },
  navBtn: {
    width: 44,
    height: 36,
    alignItems: 'center',
    justifyContent: 'center',
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: radius.md,
  },
  navText: {
    fontSize: 22,
    color: colors.text,
    lineHeight: 24,
  },
  todayBtn: {
    paddingVertical: spacing.sm,
    paddingHorizontal: spacing.lg,
  },
  todayBtnText: {
    fontSize: 14,
    fontWeight: '600',
    color: colors.textMuted,
  },
  todayBtnTextActive: {
    color: colors.text,
  },
  pressed: { opacity: 0.6 },
  loading: {
    paddingVertical: spacing.xl * 2,
  },
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
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: radius.md,
    paddingVertical: spacing.sm,
    paddingHorizontal: spacing.lg,
  },
  retryText: {
    color: colors.text,
    fontWeight: '600',
  },
  tip: {
    marginTop: spacing.md,
    textAlign: 'center',
    fontSize: 12,
    color: colors.textFaint,
  },
  legendWrap: {
    marginTop: spacing.md,
  },
  section: {
    marginTop: spacing.xl,
  },
  sectionLabel: {
    fontSize: 13,
    fontWeight: '600',
    color: colors.text,
    marginBottom: spacing.sm,
    textTransform: 'uppercase',
    letterSpacing: 0.5,
  },
  notes: {
    minHeight: 96,
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: radius.md,
    padding: spacing.md,
    fontSize: 15,
    color: colors.text,
    backgroundColor: colors.fieldBg,
    lineHeight: 21,
  },
});
