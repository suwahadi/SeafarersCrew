import { Container } from '@/components/ui/container'
import { RichHtml } from '@/components/ui/rich-html'

export function IntroSection({ html }: { html: string }) {
  return (
    <section className="py-16 sm:py-20">
      <Container>
        <RichHtml html={html} className="prose-wide mx-auto max-w-3xl text-center" />
      </Container>
    </section>
  )
}
