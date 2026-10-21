import type { ReactNode } from 'react'
import { CardGridBlock } from '@/components/blocks/card-grid-block'
import { CtaBlock } from '@/components/blocks/cta-block'
import { FaqBlock } from '@/components/blocks/faq-block'
import { GalleryBlock } from '@/components/blocks/gallery-block'
import { ImageTextBlock } from '@/components/blocks/image-text-block'
import { RichTextBlock } from '@/components/blocks/rich-text-block'
import { StatsBlock } from '@/components/blocks/stats-block'
import { VideoBlock } from '@/components/blocks/video-block'
import { Container } from '@/components/ui/container'
import { isKnownBlock } from '@/lib/blocks'
import type { KnownBlock, RawBlock } from '@/lib/types'

function renderBlock(block: KnownBlock): ReactNode {
  switch (block.type) {
    case 'rich_text':
      return <RichTextBlock settings={block.settings} />
    case 'image_text':
      return <ImageTextBlock settings={block.settings} />
    case 'gallery':
      return <GalleryBlock settings={block.settings} />
    case 'video':
      return <VideoBlock settings={block.settings} />
    case 'cta':
      return <CtaBlock settings={block.settings} />
    case 'stats':
      return <StatsBlock settings={block.settings} />
    case 'faq':
      return <FaqBlock settings={block.settings} />
    case 'card_grid':
      return <CardGridBlock settings={block.settings} />
  }
}

export function BlockRenderer({ blocks }: { blocks: readonly RawBlock[] | null }) {
  const known = (blocks ?? []).filter(isKnownBlock)
  return (
    <>
      {known.map((block) => (
        <section key={block.id} data-block={block.type} className="py-8 first:pt-12 last:pb-16">
          <Container>{renderBlock(block)}</Container>
        </section>
      ))}
    </>
  )
}
