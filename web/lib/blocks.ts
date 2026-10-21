import type { BlockType, KnownBlock, RawBlock } from '@/lib/types'

export const BLOCK_TYPES: readonly BlockType[] = [
  'rich_text',
  'image_text',
  'gallery',
  'video',
  'cta',
  'stats',
  'faq',
  'card_grid',
]

export function isKnownBlock(block: RawBlock): block is KnownBlock {
  return (BLOCK_TYPES as readonly string[]).includes(block.type) && typeof block.settings === 'object' && block.settings !== null
}

export function findBlock<K extends BlockType>(
  blocks: readonly RawBlock[] | null,
  type: K,
): Extract<KnownBlock, { type: K }> | undefined {
  for (const block of blocks ?? []) {
    if (isKnownBlock(block) && block.type === type) {
      return block as Extract<KnownBlock, { type: K }>
    }
  }
  return undefined
}
