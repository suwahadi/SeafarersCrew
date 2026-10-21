const MIN_AUTOPLAY_SECONDS = 3

export function nextIndex(current: number, count: number, direction: 1 | -1): number {
  if (count <= 0) {
    return 0
  }
  return (current + direction + count) % count
}

export interface AutoplayState {
  count: number
  seconds: number
  reducedMotion: boolean
  hovered: boolean
  focused: boolean
  hidden: boolean
}

export function shouldAutoplay(state: AutoplayState): boolean {
  return state.count > 1 && state.seconds > 0 && !state.reducedMotion && !state.hovered && !state.focused && !state.hidden
}

export function autoplayDelayMs(seconds: number): number {
  return Math.max(MIN_AUTOPLAY_SECONDS, seconds) * 1000
}

export function indexFromScroll(scrollLeft: number, width: number, count: number): number {
  if (width <= 0 || count <= 0) {
    return 0
  }
  return Math.min(count - 1, Math.max(0, Math.round(scrollLeft / width)))
}
