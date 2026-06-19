import { forwardRef } from 'react';

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

// Android and web don't use react-native-maps — they render the Leaflet WebView
// in LocationPicker instead — so this never mounts. It exists only so the import
// resolves (and react-native-maps stays out of the web/Android bundles). The
// matching iOS implementation lives in NativeMap.ios.tsx.
const NativeMap = forwardRef<NativeMapHandle, Props>(function NativeMap() {
  return null;
});

export default NativeMap;
