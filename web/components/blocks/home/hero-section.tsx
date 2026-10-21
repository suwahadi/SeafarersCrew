import type { CSSProperties } from 'react'
import { HeroSlider } from '@/components/blocks/home/hero-slider'
import { ButtonLink } from '@/components/ui/button'
import { Container } from '@/components/ui/container'
import { buildSrcSet } from '@/lib/images'
import type { HeroSlide, MaybeImage } from '@/lib/types'

const FALLBACK_DESKTOP_RATIO = '16 / 7'
const FALLBACK_MOBILE_RATIO = '4 / 5'

function ratio(image: MaybeImage, fallback: string): string {
  return image && image.width > 0 && image.height > 0 ? `${String(image.width)} / ${String(image.height)}` : fallback
}

function HeroPicture({ desktop, mobile, first }: { desktop: MaybeImage; mobile: MaybeImage; first: boolean }) {
  if (!desktop) {
    return null
  }
  const mobileSrcSet = mobile ? (buildSrcSet(mobile) ?? mobile.url) : undefined
  return (
    <picture>
      {mobile && mobileSrcSet ? (
        <source media="(max-width: 767px)" srcSet={mobileSrcSet} sizes="100vw" width={mobile.width} height={mobile.height} />
      ) : null}
      <img
        src={desktop.url}
        srcSet={buildSrcSet(desktop)}
        sizes="100vw"
        width={desktop.width}
        height={desktop.height}
        alt={desktop.alt}
        loading={first ? 'eager' : 'lazy'}
        fetchPriority={first ? 'high' : undefined}
        decoding={first ? 'sync' : 'async'}
        className="absolute inset-0 size-full object-cover"
      />
    </picture>
  )
}

interface HeroSlideViewProps {
  slide: HeroSlide
  position: number
  total: number
}

function HeroSlideView({ slide, position, total }: HeroSlideViewProps) {
  const first = position === 0
  const Heading = first ? 'h1' : 'h2'
  const mobile = slide.mobile_image ?? slide.image
  const style = {
    '--ar-desktop': ratio(slide.image, FALLBACK_DESKTOP_RATIO),
    '--ar-mobile': ratio(mobile, FALLBACK_MOBILE_RATIO),
  } as CSSProperties
  return (
    <div
      role="group"
      aria-roledescription="slide"
      aria-label={`${String(position + 1)} of ${String(total)}`}
      className="hero-slide overflow-hidden bg-primary"
      style={style}
    >
      <HeroPicture desktop={slide.image} mobile={mobile} first={first} />
      <div aria-hidden="true" className="absolute inset-0 bg-gradient-to-r from-primary/90 via-primary/55 to-primary/10" />
      <div className="absolute inset-0 flex items-center">
        <Container className="pb-14 text-on-primary">
          <div className="max-w-2xl">
            <Heading className="text-3xl font-extrabold leading-tight tracking-tight sm:text-4xl lg:text-5xl">
              {slide.heading}
            </Heading>
            {slide.subheading ? <p className="mt-4 text-base opacity-95 sm:text-lg">{slide.subheading}</p> : null}
            {slide.button_label && slide.button_link ? (
              <ButtonLink href={slide.button_link} variant="accent" className="mt-6">
                {slide.button_label}
              </ButtonLink>
            ) : null}
          </div>
        </Container>
      </div>
    </div>
  )
}

interface HeroSectionProps {
  slides: readonly HeroSlide[]
  autoplaySeconds: number
}

export function HeroSection({ slides, autoplaySeconds }: HeroSectionProps) {
  return (
    <HeroSlider count={slides.length} autoplaySeconds={autoplaySeconds}>
      {slides.map((slide, position) => (
        <HeroSlideView key={slide.id} slide={slide} position={position} total={slides.length} />
      ))}
    </HeroSlider>
  )
}
