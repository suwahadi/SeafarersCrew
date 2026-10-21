import { z } from 'zod'
import { MAX_BLOCKS, validateBlock, type FormBlock } from '@/lib/pageBlocks'
import { META_DESCRIPTION_MAX, SEO_TITLE_MAX } from '@/components/cms/SeoFields'

const blockSchema = z.custom<FormBlock>((value) => typeof value === 'object' && value !== null).superRefine((block, ctx) => {
  for (const issue of validateBlock(block)) {
    ctx.addIssue({ code: 'custom', path: ['settings', ...issue.path], message: issue.message })
  }
})

export const pageSchema = z.object({
  title: z.string().trim().min(1, 'The title is required.').max(200, 'The title must be at most 200 characters.'),
  slug: z.string().trim().max(120, 'The slug must be at most 120 characters.'),
  status: z.enum(['draft', 'published']),
  seo_title: z.string().max(SEO_TITLE_MAX, `The SEO title must be at most ${SEO_TITLE_MAX} characters.`),
  meta_description: z.string().max(META_DESCRIPTION_MAX, `The meta description must be at most ${META_DESCRIPTION_MAX} characters.`),
  og_image_file_id: z.string(),
  in_header_menu: z.boolean(),
  in_footer_menu: z.boolean(),
  menu_order: z.number({ error: 'Enter a number from 0 to 10000.' }).int('Enter a whole number.').min(0, 'Enter a number from 0 to 10000.').max(10000, 'Enter a number from 0 to 10000.'),
  intro_html: z.string(),
  blocks: z.array(blockSchema).max(MAX_BLOCKS, `A page can have at most ${MAX_BLOCKS} blocks.`),
})

export const pageTopFields = [
  'title',
  'slug',
  'status',
  'seo_title',
  'meta_description',
  'og_image_file_id',
  'in_header_menu',
  'in_footer_menu',
  'menu_order',
  'intro_html',
  'blocks',
] as const
