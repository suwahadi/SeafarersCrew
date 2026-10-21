import { BlockColor, BlockImage, BlockText, BlockTextArea, type BlockEditorProps } from './fields'

export function CtaBlockEditor({ index, media }: BlockEditorProps) {
  const base = `blocks.${index}.settings` as const
  return (
    <div className="space-y-4">
      <BlockText name={`${base}.heading`} label="Heading" required />
      <BlockTextArea name={`${base}.text`} label="Text" />
      <div className="grid gap-4 md:grid-cols-2">
        <BlockText name={`${base}.button_label`} label="Button label" />
        <BlockText name={`${base}.button_link`} label="Button link" hint="A path like /contact, or an https:// address." />
      </div>
      <div className="grid gap-4 md:grid-cols-2">
        <BlockImage name={`${base}.background_image_file_id`} label="Background image" media={media} hint="Optional. Shown instead of the color." />
        <BlockColor name={`${base}.background_color`} label="Background color" />
      </div>
    </div>
  )
}
