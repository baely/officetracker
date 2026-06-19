import React, { forwardRef, useImperativeHandle, useRef } from 'react';
import { StyleSheet } from 'react-native';
import MapView, {
  MapPressEvent,
  Marker,
  MarkerDragStartEndEvent,
} from 'react-native-maps';

export interface Coord {
  latitude: number;
  longitude: number;
}

export interface NativeMapHandle {
  recenter: (c: Coord) => void;
}

interface Props {
  center: Coord;
  coord: Coord | null;
  delta: number;
  onChange: (c: Coord) => void;
}

// iOS-only: a native MapView (Apple Maps, no API key). Kept in its own .ios file
// so react-native-maps — which calls codegenNativeComponent at import time and
// is unavailable on web/Android — never enters those bundles. Android and web
// resolve NativeMap.tsx (a no-op) and use the Leaflet WebView instead.
const NativeMap = forwardRef<NativeMapHandle, Props>(function NativeMap(
  { center, coord, delta, onChange },
  ref,
) {
  const mapRef = useRef<MapView>(null);

  useImperativeHandle(
    ref,
    () => ({
      recenter: (c: Coord) =>
        mapRef.current?.animateToRegion(
          { ...c, latitudeDelta: delta, longitudeDelta: delta },
          350,
        ),
    }),
    [delta],
  );

  return (
    <MapView
      ref={mapRef}
      style={styles.map}
      initialRegion={{
        latitude: center.latitude,
        longitude: center.longitude,
        latitudeDelta: delta,
        longitudeDelta: delta,
      }}
      onPress={(e: MapPressEvent) => onChange(e.nativeEvent.coordinate)}
    >
      {coord && (
        <Marker
          draggable
          coordinate={coord}
          onDragEnd={(e: MarkerDragStartEndEvent) =>
            onChange(e.nativeEvent.coordinate)
          }
        />
      )}
    </MapView>
  );
});

const styles = StyleSheet.create({
  map: { flex: 1 },
});

export default NativeMap;
