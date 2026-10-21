interface RichHtmlProps {
  html: string
  className?: string
}

export function RichHtml({ html, className = '' }: RichHtmlProps) {
  if (html.trim() === '') {
    return null
  }
  return <div className={`prose-site ${className}`.trim()} dangerouslySetInnerHTML={{ __html: html }} />
}
