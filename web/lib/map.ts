const LNG_SPAN = 0.012
const LAT_SPAN = 0.007

export interface Coordinates {
  lat: number
  lng: number
}

export function isValidCoordinates(value: Coordinates | null | undefined): value is Coordinates {
  return (
    !!value &&
    Number.isFinite(value.lat) &&
    Number.isFinite(value.lng) &&
    Math.abs(value.lat) <= 90 &&
    Math.abs(value.lng) <= 180 &&
    !(value.lat === 0 && value.lng === 0)
  )
}

export function mapEmbedUrl({ lat, lng }: Coordinates): string {
  const bbox = [lng - LNG_SPAN, lat - LAT_SPAN, lng + LNG_SPAN, lat + LAT_SPAN].map((n) => n.toFixed(6)).join(',')
  return `https://www.openstreetmap.org/export/embed.html?bbox=${encodeURIComponent(bbox)}&layer=mapnik&marker=${lat.toFixed(6)}%2C${lng.toFixed(6)}`
}

export function mapViewUrl({ lat, lng }: Coordinates): string {
  return `https://www.openstreetmap.org/?mlat=${lat.toFixed(6)}&mlon=${lng.toFixed(6)}#map=16/${lat.toFixed(6)}/${lng.toFixed(6)}`
}
