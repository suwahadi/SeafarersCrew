import type { SVGProps } from 'react'

const paths = {
  menu: 'M4 6h16M4 12h16M4 18h16',
  close: 'M6 6l12 12M18 6L6 18',
  columns: 'M4 5h16v14H4zM9.5 5v14M14.5 5v14',
  filter: 'M3 5h18l-7 8v5l-4 2v-7z',
  download: 'M12 4v11m0 0l-4-4m4 4l4-4M5 20h14',
  upload: 'M12 16V5m0 0l-4 4m4-4l4 4M5 20h14',
  plus: 'M12 5v14M5 12h14',
  edit: 'M4 20h4L19 9a2.1 2.1 0 00-3-3L5 17zM14 8l3 3',
  trash: 'M4 7h16M10 11v6M14 11v6M6 7l1 13h10l1-13M9 7V4h6v3',
  eye: 'M2 12s4-7 10-7 10 7 10 7-4 7-10 7S2 12 2 12zM12 15a3 3 0 100-6 3 3 0 000 6z',
  mail: 'M3 6h18v12H3zM3 7l9 7 9-7',
  grip: 'M9 5h.01M15 5h.01M9 12h.01M15 12h.01M9 19h.01M15 19h.01',
  search: 'M11 4a7 7 0 100 14 7 7 0 000-14zM20 20l-4-4',
  check: 'M5 12l5 5L20 7',
  copy: 'M9 9h11v11H9zM5 15V4h11',
  eyeOff: 'M3 3l18 18M10.6 6.2A9.7 9.7 0 0112 6c6 0 10 6 10 6a17 17 0 01-3.2 3.9M6.5 7.5A17 17 0 002 12s4 7 10 7a9.6 9.6 0 004.1-.9M9.9 9.9a3 3 0 004.2 4.2',
  refresh: 'M20 11a8 8 0 00-14-4L4 9m0-5v5h5M4 13a8 8 0 0014 4l2-2m0 5v-5h-5',
  archive: 'M3 5h18v4H3zM5 9v10h14V9M10 13h4',
  chevronLeft: 'M15 5l-7 7 7 7',
  chevronRight: 'M9 5l7 7-7 7',
  chevronDown: 'M5 9l7 7 7-7',
  external: 'M14 4h6v6M20 4l-9 9M18 14v6H4V6h6',
  send: 'M21 3L10 14M21 3l-7 18-4-7-7-4z',
  file: 'M7 3h7l5 5v13H7zM14 3v5h5',
  star: 'M12 3l2.7 5.6 6.1.9-4.4 4.3 1 6.1L12 17l-5.4 2.9 1-6.1-4.4-4.3 6.1-.9z',
  more: 'M5 12h.01M12 12h.01M19 12h.01',
  indent: 'M4 6h16M10 12h10M4 18h16M4 9l3 3-3 3',
  outdent: 'M4 6h16M10 12h10M4 18h16M7 9l-3 3 3 3',
} as const

export type IconName = keyof typeof paths

export function Icon({ name, className = 'size-4', ...rest }: { name: IconName; className?: string } & Omit<SVGProps<SVGSVGElement>, 'name'>) {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true" focusable="false" className={className} {...rest}>
      <path d={paths[name]} />
    </svg>
  )
}
