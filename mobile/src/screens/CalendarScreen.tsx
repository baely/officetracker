import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  ActivityIndicator,
  Alert,
  PanResponder,
  Pressable,
  RefreshControl,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  View,
} from 'react-native';
import { Api, isUnauthorized, MonthDays } from '../api';
import Calendar from '../components/Calendar';
import Header from '../components/Header';
import Legend from '../components/Legend';
import LocationPicker, { Coord } from '../components/LocationPicker';
import Summary, { SummaryRow } from '../components/Summary';
import WorkLocationBanner from '../components/WorkLocationBanner';
import {
  addMonths,
  calendarYearForMonth,
  DEFAULT_TRACKING_YEAR_START_MONTH,
  formatMonthYear,
  MONTH_NAMES,
  thisMonth,
  trackingMonthOrder,
  trackingYear,
  ViewMonth,
} from '../dates';
import { enableWorkTracking } from '../location';
import { monthStats, yearStats } from '../stats';
import { AttendanceState, cycleState } from '../states';
import {
  cacheStartMonth,
  Connection,
  DEFAULT_WORK_RADIUS,
  dismissWorkBanner,
  isWorkBannerDismissed,
  loadWorkLocation,
} from '../storage';
import { colors, radius, spacing } from '../theme';

interface Props {
  conn: Connection;
  onOpenSettings: () => void;
  onUnauthorized: () => void;
}

export default function CalendarScreen({
  conn,
  onOpenSettings,
  onUnauthorized,
}: Props) {
  const api = useMemo(
    () => new Api(conn, onUnauthorized),
    [conn, onUnauthorized],
  );

  // Read-only servers can be browsed but never written to.
  const readOnly = conn.readOnly;

  const [view, setView] = useState<ViewMonth>(thisMonth());
  const [startMonth, setStartMonth] = useState(DEFAULT_TRACKING_YEAR_START_MONTH);
  const fy = trackingYear(view.year, view.month, startMonth);

  // Loaded tracking year, keyed month (1-12) -> day -> state.
  const [yearData, setYearData] = useState<Record<number, MonthDays>>({});
  const [notes, setNotes] = useState<Record<number, string>>({});
  const [loadedFy, setLoadedFy] = useState<number | null>(null);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Local, possibly-unsaved note text for the current month.
  const [noteText, setNoteText] = useState('');

  // Work-location prompt + picker.
  const [showBanner, setShowBanner] = useState(false);
  const [pickerVisible, setPickerVisible] = useState(false);

  // Show the banner only when no work location is set and it wasn't dismissed.
  useEffect(() => {
    let cancelled = false;
    Promise.all([loadWorkLocation(), isWorkBannerDismissed()]).then(
      ([loc, dismissed]) => {
        if (!cancelled) setShowBanner(!loc && !dismissed);
      },
    );
    return () => {
      cancelled = true;
    };
  }, []);

  const onPickLocation = useCallback(
    async (coord: Coord, label?: string) => {
      setPickerVisible(false);
      setShowBanner(false);
      const background = await enableWorkTracking({
        latitude: coord.latitude,
        longitude: coord.longitude,
        radius: DEFAULT_WORK_RADIUS,
        label,
      });
      if (!background) {
        Alert.alert(
          'Work location saved',
          'To mark office days while the app is closed, allow "Always" location access for Officetracker in your device settings.',
        );
      }
    },
    [],
  );

  const dismissBanner = useCallback(() => {
    setShowBanner(false);
    dismissWorkBanner();
  }, []);

  // Live refs so optimistic callbacks read current state (not a stale closure)
  // and a late refetch can confirm we're still on the same tracking year.
  const fyRef = useRef(fy);
  fyRef.current = fy;
  const yearDataRef = useRef(yearData);
  yearDataRef.current = yearData;

  // Best-effort fetch of the start month. Plain Api (no onUnauthorized) so an
  // older server that 401s /settings just keeps the default instead of logging out.
  useEffect(() => {
    let cancelled = false;
    new Api(conn)
      .getSettings()
      .then((s) => {
        if (!cancelled) setStartMonth(s.trackingYearStartMonth);
        // Cache for the background geofence task, which can't fetch settings.
        cacheStartMonth(s.trackingYearStartMonth);
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
      if (readOnly) return;
      // Read the live value so rapid taps don't cycle/revert from a stale closure.
      const current =
        yearDataRef.current[view.month]?.[day] ?? AttendanceState.Untracked;
      const next = cycleState(current, direction);
      const targetFy = fy;

      // Optimistic update.
      setYearData((prev) => {
        const month = { ...(prev[view.month] ?? {}) };
        month[day] = next;
        return { ...prev, [view.month]: month };
      });

      api
        .putDay(view.year, view.month, day, next)
        .then(() => {
          // Clearing a day may reveal a server-computed scheduled state; refetch,
          // but only apply it if we haven't since navigated to another year.
          if (next === AttendanceState.Untracked) {
            return api.getYear(targetFy).then((data) => {
              if (fyRef.current === targetFy) setYearData(data);
            });
          }
        })
        .catch((e: any) => {
          // Revert just this day on failure.
          setYearData((prev) => {
            const month = { ...(prev[view.month] ?? {}) };
            month[day] = current;
            return { ...prev, [view.month]: month };
          });
          // On 401/403 the app is already signing out; don't also alert.
          if (!isUnauthorized(e)) {
            Alert.alert('Could not save', e?.message ?? 'Please try again.');
          }
        });
    },
    [api, view, fy, readOnly],
  );

  const saveNote = useCallback(() => {
    if (readOnly) return;
    const text = noteText;
    const prev = notes[view.month] ?? '';
    if (prev === text) return;
    setNotes((n) => ({ ...n, [view.month]: text }));
    api.putNote(view.year, view.month, text).catch((e: any) => {
      if (isUnauthorized(e)) return;
      setNotes((n) => ({ ...n, [view.month]: prev })); // revert on failure
      Alert.alert('Could not save note', e?.message ?? 'Please try again.');
    });
  }, [api, noteText, notes, view, readOnly]);

  // Save any pending note before leaving the current month/screen.
  const go = (delta: number) => {
    saveNote();
    setView((v) => addMonths(v, delta));
  };
  const goToday = () => {
    saveNote();
    setView(thisMonth());
  };
  const openSettings = () => {
    saveNote();
    onOpenSettings();
  };

  const year = useMemo(() => yearStats(yearData), [yearData]);

  // One row per tracked month, ordered start-month-first, mirroring the web
  // summary table.
  const summaryRows = useMemo<SummaryRow[]>(() => {
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

  // Swipe the calendar left/right to change months. Built once; reads the
  // latest navigation handler through a ref to avoid a stale closure.
  const goRef = useRef(go);
  goRef.current = go;
  const monthSwipe = useRef(
    PanResponder.create({
      // Only claim clearly-horizontal drags so day taps and vertical scroll
      // keep working.
      onMoveShouldSetPanResponder: (_e, g) =>
        Math.abs(g.dx) > 20 && Math.abs(g.dx) > Math.abs(g.dy) * 1.5,
      onPanResponderRelease: (_e, g) => {
        if (g.dx <= -50) goRef.current(1); // swipe left → next month
        else if (g.dx >= 50) goRef.current(-1); // swipe right → previous month
      },
    }),
  ).current;

  return (
    <View style={styles.screen}>
      {/* Fixed nav bar — pull-to-refresh only scrolls the content below it. */}
      <Header rightLabel="Settings" onRightPress={openSettings} />

      <ScrollView
        style={styles.flex}
        contentContainerStyle={styles.content}
        keyboardShouldPersistTaps="handled"
        keyboardDismissMode="interactive"
        automaticallyAdjustKeyboardInsets
        refreshControl={
          <RefreshControl
            refreshing={refreshing}
            onRefresh={() => load(fy, true)}
            tintColor={colors.textMuted}
          />
        }
      >
        <View style={styles.body}>
        {showBanner && !readOnly && (
          <WorkLocationBanner
            onPress={() => setPickerVisible(true)}
            onDismiss={dismissBanner}
          />
        )}
      <View style={styles.calendarNav}>
        <Pressable
          onPress={() => go(-1)}
          style={({ pressed }) => [styles.navBtn, pressed && styles.pressed]}
          hitSlop={8}
        >
          <Text style={styles.navText}>‹</Text>
        </Pressable>
        <Pressable onPress={goToday} hitSlop={8}>
          <Text style={styles.monthYear}>
            {MONTH_NAMES[view.month - 1]} {view.year}
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
          <View {...monthSwipe.panHandlers}>
            <Calendar
              year={view.year}
              month={view.month}
              days={days}
              onCycle={onCycle}
              readOnly={readOnly}
            />
          </View>

          <Text style={styles.tip}>
            Tap a day to cycle through home, office and other; long-press to go
            back.
          </Text>
          <View style={styles.legendWrap}>
            <Legend />
          </View>

          <View style={styles.section}>
            <Text style={styles.heading}>Notes</Text>
            <TextInput
              style={styles.notes}
              value={noteText}
              onChangeText={setNoteText}
              onBlur={saveNote}
              editable={!readOnly}
              placeholder={`Notes for ${formatMonthYear(view)}…`}
              placeholderTextColor={colors.textFaint}
              multiline
              textAlignVertical="top"
            />
          </View>

          <View style={styles.section}>
            <Text style={styles.heading}>Summary</Text>
            <Summary rows={summaryRows} total={year} />
          </View>
        </>
      )}
        </View>
      </ScrollView>

      <LocationPicker
        visible={pickerVisible}
        onClose={() => setPickerVisible(false)}
        onSelect={onPickLocation}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  screen: { flex: 1, backgroundColor: colors.surface },
  flex: { flex: 1, backgroundColor: colors.surface },
  content: {
    paddingBottom: spacing.xl * 2,
  },
  body: {
    padding: spacing.lg,
  },
  calendarNav: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginTop: spacing.lg,
    marginBottom: spacing.md,
    // Match the 3px padding inside each day cell so the arrows line up with the
    // Monday and Sunday columns below.
    paddingHorizontal: 3,
  },
  navBtn: {
    width: 44,
    height: 36,
    alignItems: 'center',
    justifyContent: 'center',
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: radius.md,
    backgroundColor: colors.cellBg,
  },
  navText: {
    fontSize: 22,
    color: colors.text,
    lineHeight: 24,
  },
  monthYear: {
    fontSize: 22,
    fontWeight: '700',
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
    lineHeight: 17,
  },
  legendWrap: {
    marginTop: spacing.md,
  },
  section: {
    marginTop: spacing.xl,
  },
  heading: {
    fontSize: 18,
    fontWeight: '700',
    color: colors.text,
    marginBottom: spacing.sm,
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
