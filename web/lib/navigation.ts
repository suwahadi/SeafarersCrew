interface ClickLike {
  defaultPrevented: boolean
  button: number
  metaKey: boolean
  ctrlKey: boolean
  shiftKey: boolean
  altKey: boolean
}

interface AnchorLike {
  href: string
  target: string
  hasAttribute: (name: string) => boolean
}

interface LocationLike {
  origin: string
  pathname: string
  search: string
}

export function isInternalNavigation(event: ClickLike, anchor: AnchorLike, current: LocationLike): boolean {
  if (event.defaultPrevented || event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) {
    return false
  }
  if ((anchor.target !== '' && anchor.target !== '_self') || anchor.hasAttribute('download')) {
    return false
  }
  let url: URL
  try {
    url = new URL(anchor.href)
  } catch {
    return false
  }
  if (url.origin !== current.origin) {
    return false
  }
  return url.pathname !== current.pathname || url.search !== current.search
}
