import React, { useRef } from 'react';
import { Alert, Image, Pressable, StyleSheet, Text, View } from 'react-native';
import { colors, fonts, spacing } from '../theme';

interface Props {
  // Optional web-style nav link shown on the right (e.g. "/settings").
  rightLabel?: string;
  onRightPress?: () => void;
}

// The pale-cyan brand bar that mirrors the web app's <nav>
// (internal/embed/html/bases/base.html): the office-building icon, the
// "Officetracker" wordmark in Calistoga, and a slash-prefixed nav link.
export default function Header({ rightLabel, onRightPress }: Props) {
  // iykyk: seven quick taps on the wordmark asks the source of truth.
  const taps = useRef(0);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const askTheSourceOfTruth = async () => {
    taps.current += 1;
    if (timer.current) clearTimeout(timer.current);
    timer.current = setTimeout(() => (taps.current = 0), 1500);
    if (taps.current < 7) return;
    taps.current = 0;

    const title = 'Is Bailey Butler in the office today?';
    try {
      const res = await fetch('https://isbaileybutlerintheoffice.today/raw');
      const answer = (await res.text()).trim().toLowerCase();
      Alert.alert(title, answer === 'yes' ? 'Yes.' : 'No.');
    } catch {
      Alert.alert(title, 'Unreachable. Assume no.');
    }
  };

  return (
    <View style={styles.nav}>
      <View style={styles.brand}>
        <Image
          source={require('../../assets/office-building.png')}
          style={styles.icon}
          resizeMode="contain"
        />
        <Pressable onPress={askTheSourceOfTruth} hitSlop={6}>
          <Text style={styles.wordmark}>Officetracker</Text>
        </Pressable>
      </View>
      {rightLabel && (
        <Pressable
          onPress={onRightPress}
          hitSlop={10}
          style={({ pressed }) => pressed && styles.pressed}
        >
          <Text style={styles.link}>{rightLabel}</Text>
        </Pressable>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  nav: {
    backgroundColor: colors.navBg,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: spacing.lg,
    paddingTop: spacing.lg,
    paddingBottom: spacing.lg,
  },
  brand: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: spacing.sm,
  },
  icon: {
    width: 28,
    height: 28,
  },
  wordmark: {
    fontSize: 24,
    fontFamily: fonts.wordmark,
    color: colors.text,
  },
  link: {
    fontSize: 16,
    color: colors.textMuted,
  },
  pressed: { opacity: 0.6 },
});
