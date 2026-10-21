import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { OptionsReady } from './OptionsReady'

const all = vi.fn()

vi.mock('@/api/options', () => ({
  OPTIONS_KEY: ['options'],
  optionsApi: { all: () => all() as unknown },
}))

function renderGate() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={client}>
      <OptionsReady>
        <p>form body</p>
      </OptionsReady>
    </QueryClientProvider>,
  )
}

describe('OptionsReady', () => {
  it('holds the form back until master options have loaded', async () => {
    let resolve: (v: unknown) => void = () => undefined
    all.mockReturnValue(new Promise((r) => (resolve = r)))
    renderGate()
    expect(screen.queryByText('form body')).not.toBeInTheDocument()
    resolve({ companies: [], vessels: [], ranks: [], vessel_types: [], currencies: [], crew_statuses: [], allowance_types: [], document_types: [], certificate_types: [] })
    expect(await screen.findByText('form body')).toBeInTheDocument()
  })

  it('shows the load error instead of an empty form', async () => {
    all.mockRejectedValue(new Error('options unavailable'))
    renderGate()
    expect(await screen.findByRole('alert')).toHaveTextContent('options unavailable')
    expect(screen.queryByText('form body')).not.toBeInTheDocument()
  })
})
