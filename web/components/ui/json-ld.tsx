import { serializeJsonLd, type JsonLdNode } from '@/lib/jsonld'

export function JsonLd({ node }: { node: JsonLdNode }) {
  return <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: serializeJsonLd(node) }} />
}
