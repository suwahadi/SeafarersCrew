export function PreviewFrame({ html, title = 'Document preview' }: { html: string; title?: string }) {
  return <iframe title={title} sandbox="" srcDoc={html} className="h-[70vh] w-full rounded-md bg-white ring-1 ring-slate-300" />
}
