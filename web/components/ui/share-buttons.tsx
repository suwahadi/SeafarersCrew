import { CopyLinkButton } from '@/components/ui/copy-link-button'
import { Icon } from '@/components/ui/icons'
import { shareTargets } from '@/lib/share'

const BUTTON =
  'inline-flex items-center justify-center gap-2 rounded-md border border-line px-3 py-2 text-sm font-medium text-primary transition-colors duration-150 ease-out hover:bg-surface max-sm:size-10 max-sm:px-0'

export function ShareButtons({ url, title }: { url: string; title: string }) {
  return (
    <div className="flex items-center gap-2 sm:flex-wrap">
      <p className="mr-1 text-sm font-semibold text-muted">Share</p>
      {shareTargets(url, title).map((target) => (
        <a key={target.key} href={target.href} target="_blank" rel="noopener noreferrer" className={BUTTON}>
          <Icon name={target.key} className={`size-4 ${target.key === 'whatsapp' ? 'text-[#25D366]' : ''}`} />
          <span className="max-sm:sr-only">
            {target.label}
            <span className="sr-only"> (opens in a new tab)</span>
          </span>
        </a>
      ))}
      <CopyLinkButton url={url} className={BUTTON} />
    </div>
  )
}
