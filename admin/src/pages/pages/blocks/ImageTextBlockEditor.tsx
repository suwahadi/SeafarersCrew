import { BlockImage, BlockRich, BlockSelect, BlockText, type BlockEditorProps } from './fields'

const POSITIONS = [
  { value: 'left', label: 'Image on the left' },
  { value: 'right', label: 'Image on the right' },
]

export function ImageTextBlockEditor({ index, media }: BlockEditorProps) {
  const base = `blocks.${index}.settings` as const
  return (
    <div className="space-y-4">
      <div className="grid gap-4 md:grid-cols-2">
        <BlockImage name={`${base}.image_file_id`} label="Image" media={media} required />
        <BlockSelect name={`${base}.image_position`} label="Image position" options={POSITIONS} />
      </div>
      <BlockText name={`${base}.heading`} label="Heading" />
      <BlockRich name={`${base}.text_html`} label="Text" />
      <div className="grid gap-4 md:grid-cols-2">
        <BlockText name={`${base}.button_label`} label="Button label" />
        <BlockText name={`${base}.button_link`} label="Button link" hint="A path like /contact, or an https:// address." />
      </div>
    </div>
  )
}
