import { BlockRich, type BlockEditorProps } from './fields'

export function RichTextBlockEditor({ index }: BlockEditorProps) {
  return <BlockRich name={`blocks.${index}.settings.html`} label="Content" minHeight="10rem" />
}
