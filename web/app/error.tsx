'use client'

import Link from 'next/link'
import { Container } from '@/components/ui/container'
import { buttonClass } from '@/components/ui/button'

interface ErrorPageProps {
  reset: () => void
}

export default function ErrorPage({ reset }: ErrorPageProps) {
  return (
    <Container className="py-20 text-center sm:py-28">
      <h1 className="text-3xl font-bold tracking-tight text-primary sm:text-4xl">Something went wrong</h1>
      <p className="mx-auto mt-4 max-w-lg text-muted">
        We could not load this page right now. Please try again in a moment.
      </p>
      <div className="mt-8 flex flex-wrap justify-center gap-3">
        <button type="button" onClick={reset} className={buttonClass('primary')}>
          Try again
        </button>
        <Link href="/" className={buttonClass('outline')}>
          Back to home
        </Link>
      </div>
    </Container>
  )
}
