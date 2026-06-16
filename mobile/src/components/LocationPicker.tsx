import * as Location from 'expo-location';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  ActivityIndicator,
  Alert,
  Modal,
  Platform,
  Pressable,
  StyleSheet,
  Text,
  TextInput,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import MapView, { MapPressEvent, Marker, MarkerDragStartEndEvent } from 'react-native-maps';
import { WebView } from 'react-native-webview';
import { colors, radius, spacing } from '../theme';

export interface Coord {
  latitude: number;
  longitude: number;
}

interface Props {
  visible: boolean;
  // Existing location to start from, if any.
  initial?: Coord | null;
  onClose: () => void;
  onSelect: (coord: Coord, label?: string) => void;
}

// Sensible fallback centre (Sydney) when we can't get a current position.
const FALLBACK: Coord = { latitude: -33.8688, longitude: 151.2093 };

// ~1km span — a comfortable zoom for picking a building.
const DELTA = 0.01;

// Android falls back to a self-contained Leaflet map (OpenStreetMap tiles, no
// API key). iOS uses a native MapView (Apple Maps), which needs no key either.
// Tapping the map or dragging the pin posts the coordinate back.
function leafletHtml(c: Coord): string {
  return `<!DOCTYPE html><html><head>
<meta charset="utf-8"/>
<meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no"/>
<link rel="stylesheet" href="https://unpkg.com/leaflet@1.9.4/dist/leaflet.css"/>
<style>html,body,#map{height:100%;margin:0;padding:0;}.leaflet-container{background:#e9eef0;}</style>
</head><body>
<div id="map"></div>
<script src="https://unpkg.com/leaflet@1.9.4/dist/leaflet.js"></script>
<script>
  var map = L.map('map').setView([${c.latitude}, ${c.longitude}], 16);
  L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
    maxZoom: 19, attribution: '&copy; OpenStreetMap'
  }).addTo(map);
  var marker = L.marker([${c.latitude}, ${c.longitude}], { draggable: true }).addTo(map);
  function post(ll) {
    if (window.ReactNativeWebView) {
      window.ReactNativeWebView.postMessage(JSON.stringify({ lat: ll.lat, lng: ll.lng }));
    }
  }
  map.on('click', function (e) { marker.setLatLng(e.latlng); post(e.latlng); });
  marker.on('dragend', function () { post(marker.getLatLng()); });
  window.setLocation = function (lat, lng) {
    var ll = L.latLng(lat, lng);
    marker.setLatLng(ll);
    map.setView(ll, 16);
    post(ll);
  };
  post(marker.getLatLng());
</script>
</body></html>`;
}

function formatPlace(p?: Location.LocationGeocodedAddress): string | undefined {
  if (!p) return undefined;
  const line = [p.name || p.street, p.city || p.subregion].filter(Boolean);
  const s = line.join(', ');
  return s || undefined;
}

export default function LocationPicker({
  visible,
  initial,
  onClose,
  onSelect,
}: Props) {
  const webRef = useRef<WebView>(null);
  const mapRef = useRef<MapView>(null);
  const [center, setCenter] = useState<Coord | null>(null);
  const [coord, setCoord] = useState<Coord | null>(null);
  const [address, setAddress] = useState('');
  const [searching, setSearching] = useState(false);
  const [saving, setSaving] = useState(false);

  // Resolve an initial centre each time the modal opens: the existing location,
  // else the device's current position, else a fallback.
  useEffect(() => {
    if (!visible) return;
    let cancelled = false;
    setCenter(null);
    setCoord(null);
    setAddress('');
    (async () => {
      if (initial) {
        if (!cancelled) {
          setCenter(initial);
          setCoord(initial);
        }
        return;
      }
      try {
        const perm = await Location.requestForegroundPermissionsAsync();
        if (perm.status === 'granted') {
          const pos = await Location.getCurrentPositionAsync({
            accuracy: Location.Accuracy.Balanced,
          });
          if (!cancelled) {
            const c = {
              latitude: pos.coords.latitude,
              longitude: pos.coords.longitude,
            };
            setCenter(c);
            setCoord(c);
            return;
          }
        }
      } catch {
        // fall through to fallback
      }
      if (!cancelled) {
        setCenter(FALLBACK);
        setCoord(FALLBACK);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [visible, initial]);

  // Android-only: Leaflet HTML for the WebView.
  const html = useMemo(
    () => (center && Platform.OS !== 'ios' ? leafletHtml(center) : ''),
    [center],
  );

  const onWebMessage = useCallback((e: { nativeEvent: { data: string } }) => {
    try {
      const m = JSON.parse(e.nativeEvent.data) as { lat: number; lng: number };
      if (typeof m.lat === 'number' && typeof m.lng === 'number') {
        setCoord({ latitude: m.lat, longitude: m.lng });
      }
    } catch {
      // ignore malformed messages
    }
  }, []);

  // Move the visible map to a coordinate (after an address search).
  function recenter(c: Coord) {
    if (Platform.OS === 'ios') {
      mapRef.current?.animateToRegion(
        { ...c, latitudeDelta: DELTA, longitudeDelta: DELTA },
        350,
      );
    } else {
      webRef.current?.injectJavaScript(
        `window.setLocation(${c.latitude}, ${c.longitude}); true;`,
      );
    }
  }

  async function searchAddress() {
    const q = address.trim();
    if (!q) return;
    setSearching(true);
    try {
      const results = await Location.geocodeAsync(q);
      if (!results.length) {
        Alert.alert('Not found', 'Could not find that address. Try the map.');
        return;
      }
      const r = results[0];
      const c = { latitude: r.latitude, longitude: r.longitude };
      setCoord(c);
      recenter(c);
    } catch {
      Alert.alert('Search failed', 'Could not look up that address.');
    } finally {
      setSearching(false);
    }
  }

  async function confirm() {
    if (!coord) return;
    setSaving(true);
    let label: string | undefined;
    try {
      const places = await Location.reverseGeocodeAsync(coord);
      label = formatPlace(places[0]);
    } catch {
      // label is optional
    }
    setSaving(false);
    onSelect(coord, label);
  }

  return (
    <Modal
      visible={visible}
      animationType="slide"
      onRequestClose={onClose}
      presentationStyle="fullScreen"
    >
      <SafeAreaView style={styles.flex} edges={['top', 'left', 'right']}>
        <View style={styles.header}>
          <Pressable onPress={onClose} hitSlop={10}>
            <Text style={styles.cancel}>Cancel</Text>
          </Pressable>
          <Text style={styles.title}>Work location</Text>
          <View style={styles.headerSpacer} />
        </View>

        <View style={styles.searchRow}>
          <TextInput
            style={styles.input}
            value={address}
            onChangeText={setAddress}
            placeholder="Search an address"
            placeholderTextColor={colors.textFaint}
            autoCapitalize="words"
            returnKeyType="search"
            onSubmitEditing={searchAddress}
          />
          <Pressable
            style={styles.searchBtn}
            onPress={searchAddress}
            disabled={searching}
          >
            {searching ? (
              <ActivityIndicator color="#ffffff" />
            ) : (
              <Text style={styles.searchBtnText}>Search</Text>
            )}
          </Pressable>
        </View>

        <View style={styles.mapWrap}>
          {!center ? (
            <View style={styles.mapLoading}>
              <ActivityIndicator color={colors.textMuted} />
              <Text style={styles.hint}>Finding your location…</Text>
            </View>
          ) : Platform.OS === 'ios' ? (
            <MapView
              ref={mapRef}
              style={styles.map}
              initialRegion={{
                latitude: center.latitude,
                longitude: center.longitude,
                latitudeDelta: DELTA,
                longitudeDelta: DELTA,
              }}
              onPress={(e: MapPressEvent) =>
                setCoord(e.nativeEvent.coordinate)
              }
            >
              {coord && (
                <Marker
                  draggable
                  coordinate={coord}
                  onDragEnd={(e: MarkerDragStartEndEvent) =>
                    setCoord(e.nativeEvent.coordinate)
                  }
                />
              )}
            </MapView>
          ) : (
            <WebView
              ref={webRef}
              originWhitelist={['*']}
              source={{ html }}
              onMessage={onWebMessage}
              style={styles.map}
              scrollEnabled={false}
            />
          )}
        </View>

        <View style={styles.footer}>
          <Text style={styles.hint}>
            Tap the map or drag the pin to set where "work" is.
          </Text>
          <Pressable
            style={[styles.confirm, !coord && styles.confirmDisabled]}
            onPress={confirm}
            disabled={!coord || saving}
          >
            {saving ? (
              <ActivityIndicator color="#ffffff" />
            ) : (
              <Text style={styles.confirmText}>Use this location</Text>
            )}
          </Pressable>
        </View>
      </SafeAreaView>
    </Modal>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1, backgroundColor: colors.surface },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md,
    borderBottomWidth: 1,
    borderBottomColor: colors.border,
  },
  cancel: { fontSize: 16, color: colors.textMuted },
  title: { fontSize: 16, fontWeight: '700', color: colors.text },
  headerSpacer: { width: 52 },
  searchRow: {
    flexDirection: 'row',
    gap: spacing.sm,
    padding: spacing.lg,
  },
  input: {
    flex: 1,
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: radius.md,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.md,
    fontSize: 15,
    color: colors.text,
    backgroundColor: colors.fieldBg,
  },
  searchBtn: {
    backgroundColor: colors.accent,
    borderRadius: radius.md,
    paddingHorizontal: spacing.lg,
    alignItems: 'center',
    justifyContent: 'center',
  },
  searchBtnText: { color: '#ffffff', fontSize: 15, fontWeight: '600' },
  mapWrap: { flex: 1, overflow: 'hidden' },
  map: { flex: 1 },
  mapLoading: { flex: 1, alignItems: 'center', justifyContent: 'center' },
  footer: {
    padding: spacing.lg,
    borderTopWidth: 1,
    borderTopColor: colors.border,
  },
  hint: {
    fontSize: 13,
    color: colors.textFaint,
    lineHeight: 18,
    textAlign: 'center',
    marginBottom: spacing.sm,
  },
  confirm: {
    backgroundColor: colors.accent,
    borderRadius: radius.md,
    paddingVertical: spacing.md,
    alignItems: 'center',
  },
  confirmDisabled: { opacity: 0.5 },
  confirmText: { color: '#ffffff', fontSize: 15, fontWeight: '600' },
});
