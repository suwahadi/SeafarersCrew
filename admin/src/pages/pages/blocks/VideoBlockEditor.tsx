import { useFormContext, useWatch } from 'react-hook-form'
import type { PageFormValues } from '@/lib/pagePayload'
import { parseVideoUrl, youtubeThumbnail } from '@/lib/videoUrl'
import { BlockText, type BlockEditorProps } from './fields'

export function VideoBlockEditor({ index }: BlockEditorProps) {
  const base = `blocks.${index}.settings` as const
  const { control } = useFormContext<PageFormValues>()
  const url = useWatch({ control, name: `${base}.url` })
  const parsed = url.trim() === '' ? null : parseVideoUrl(url)

  return (
    <div className="space-y-4">
      <BlockText name={`${base}.url`} label="Video link" required type="url" placeholder="https://www.youtube.com/watch?v=…" hint="A YouTube or Vimeo address." />
      {parsed ? (
        <div className="flex items-center gap-3">
          <div className="flex w-48 items-center justify-center overflow-hidden rounded-md bg-slate-100 ring-1 ring-inset ring-slate-200" style={{ aspectRatio: '16 / 9' }}>
            {parsed.provider === 'youtube' ? (
              <img src={youtubeThumbnail(parsed.id)} alt="" width={480} height={360} className="h-full w-full object-cover" />
            ) : (
              <span className="text-xs text-slate-500">No preview</span>
            )}
          </div>
          <p className="text-xs text-slate-600">{parsed.provider === 'youtube' ? 'YouTube' : 'Vimeo'} video detected.</p>
        </div>
      ) : url.trim() === '' ? null : (
        <p className="text-xs text-orange-700">This link is not a recognised YouTube or Vimeo address.</p>
      )}
      <BlockText name={`${base}.title`} label="Title" hint="Used as the accessible name of the video." />
    </div>
  )
}
