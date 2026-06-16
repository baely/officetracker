import React from 'react';
import { Image, Pressable, StyleSheet, Text, View } from 'react-native';
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
  return (
    <View style={styles.nav}>
      <View style={styles.brand}>
        <Image
          source={require('../../assets/office-building.png')}
          style={styles.icon}
          resizeMode="contain"
        />
        <Text style={styles.wordmark}>Officetracker</Text>
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
