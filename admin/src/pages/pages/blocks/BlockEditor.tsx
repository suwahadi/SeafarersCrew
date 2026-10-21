import type { BlockType } from '@/api/pages'
import { CardGridBlockEditor } from './CardGridBlockEditor'
import { CtaBlockEditor } from './CtaBlockEditor'
import { FaqBlockEditor } from './FaqBlockEditor'
import type { BlockEditorProps } from './fields'
import { GalleryBlockEditor } from './GalleryBlockEditor'
import { ImageTextBlockEditor } from './ImageTextBlockEditor'
import { RichTextBlockEditor } from './RichTextBlockEditor'
import { StatsBlockEditor } from './StatsBlockEditor'
import { VideoBlockEditor } from './VideoBlockEditor'

export function BlockEditor({ type, ...props }: BlockEditorProps & { type: BlockType }) {
  switch (type) {
    case 'rich_text':
      return <RichTextBlockEditor {...props} />
    case 'image_text':
      return <ImageTextBlockEditor {...props} />
    case 'gallery':
      return <GalleryBlockEditor {...props} />
    case 'video':
      return <VideoBlockEditor {...props} />
    case 'cta':
      return <CtaBlockEditor {...props} />
    case 'stats':
      return <StatsBlockEditor {...props} />
    case 'faq':
      return <FaqBlockEditor {...props} />
    case 'card_grid':
      return <CardGridBlockEditor {...props} />
  }
}
