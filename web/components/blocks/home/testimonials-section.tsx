import { CardCarousel } from '@/components/ui/card-carousel'
import { Container } from '@/components/ui/container'
import { Icon } from '@/components/ui/icons'
import { SectionHeading } from '@/components/ui/section-heading'
import { SiteImage } from '@/components/ui/site-image'
import type { Testimonial } from '@/lib/types'

const MAX_RATING = 5

function Stars({ rating }: { rating: number }) {
  const value = Math.min(MAX_RATING, Math.max(0, Math.round(rating)))
  if (value === 0) {
    return null
  }
  return (
    <p role="img" aria-label={`Rated ${String(value)} out of ${String(MAX_RATING)}`} className="flex gap-0.5 text-amber-500">
      {Array.from({ length: MAX_RATING }, (_, position) => (
        <Icon
          key={position}
          name="star"
          className={`size-5 ${position < value ? 'fill-current' : 'text-slate-300'}`}
        />
      ))}
    </p>
  )
}

export function TestimonialsSection({ testimonials }: { testimonials: readonly Testimonial[] }) {
  return (
    <section aria-labelledby="home-testimonials" className="bg-surface py-16 sm:py-20">
      <Container>
        <SectionHeading id="home-testimonials" eyebrow="Testimonials" title="What our crew say" />
        <CardCarousel label="Testimonials">
          {testimonials.map((testimonial) => (
            <div key={testimonial.id} role="group" aria-roledescription="slide">
            <figure className="flex h-full flex-col rounded-lg border border-line bg-white p-6">
              <Stars rating={testimonial.rating} />
              <blockquote className="mt-4 flex-1 text-base leading-7 text-ink">{testimonial.quote}</blockquote>
              <figcaption className="mt-6 flex items-center gap-3">
                {testimonial.photo ? (
                  <SiteImage
                    image={testimonial.photo}
                    sizes="48px"
                    alt=""
                    className="size-12 rounded-full object-cover"
                  />
                ) : null}
                <span>
                  <span className="block font-semibold text-primary">{testimonial.name}</span>
                  {testimonial.position ? <span className="block text-sm text-muted">{testimonial.position}</span> : null}
                </span>
              </figcaption>
            </figure>
            </div>
          ))}
        </CardCarousel>
      </Container>
    </section>
  )
}
