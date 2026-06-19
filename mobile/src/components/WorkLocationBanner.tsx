import React from 'react';
import { Pressable, StyleSheet, Text, View } from 'react-native';
import { colors, radius, spacing } from '../theme';

interface Props {
  onPress: () => void;
  onDismiss: () => void;
}

// Home-screen prompt to set a work location so office days get marked
// automatically. Shown only while no location is configured.
export default function WorkLocationBanner({ onPress, onDismiss }: Props) {
  return (
    <View style={styles.banner}>
      <View style={styles.text}>
        <Text style={styles.title}>Mark office days automatically</Text>
        <Text style={styles.body}>
          Set your work location and Officetracker will tick off office days when
          you arrive.
        </Text>
        <Pressable
          onPress={onPress}
          style={({ pressed }) => [styles.cta, pressed && styles.pressed]}
        >
          <Text style={styles.ctaText}>Set work location</Text>
        </Pressable>
      </View>
      <Pressable onPress={onDismiss} hitSlop={10} style={styles.dismiss}>
        <Text style={styles.dismissText}>×</Text>
      </Pressable>
    </View>
  );
}

const styles = StyleSheet.create({
  banner: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    backgroundColor: colors.brandTint,
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: 10,
    padding: spacing.lg,
    marginBottom: spacing.md,
  },
  text: { flex: 1 },
  title: { fontSize: 15, fontWeight: '700', color: colors.text },
  body: {
    fontSize: 13,
    color: colors.textMuted,
    lineHeight: 18,
    marginTop: 2,
  },
  cta: {
    alignSelf: 'flex-start',
    marginTop: spacing.md,
    backgroundColor: colors.accent,
    borderRadius: radius.md,
    paddingVertical: spacing.sm,
    paddingHorizontal: spacing.md,
  },
  ctaText: { color: '#ffffff', fontSize: 14, fontWeight: '600' },
  pressed: { opacity: 0.7 },
  dismiss: {
    paddingLeft: spacing.md,
    marginTop: -2,
  },
  dismissText: { fontSize: 22, color: colors.textFaint, lineHeight: 22 },
});
