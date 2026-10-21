import { render, screen } from '@testing-library/react'
import { DocumentExpiry } from './DocumentsTab'

describe('DocumentExpiry', () => {
  it('shows the formatted date with the days remaining for an expiring document', () => {
    render(<DocumentExpiry row={{ has_expiry: true, expired_date: '2026-11-15', badge: 'red', days_left: 25 }} />)
    expect(screen.getByText('15-Nov-2026')).toBeInTheDocument()
    expect(screen.getByText('25 days')).toBeInTheDocument()
  })

  it('labels expired and valid documents', () => {
    const { rerender } = render(<DocumentExpiry row={{ has_expiry: true, expired_date: '2026-01-01', badge: 'expired', days_left: -10 }} />)
    expect(screen.getByText('Expired')).toBeInTheDocument()
    rerender(<DocumentExpiry row={{ has_expiry: true, expired_date: '2030-01-01', badge: 'green', days_left: 1200 }} />)
    expect(screen.getByText('Valid')).toBeInTheDocument()
  })

  it('says no expiry for documents that do not expire', () => {
    render(<DocumentExpiry row={{ has_expiry: false, expired_date: null, badge: '', days_left: null }} />)
    expect(screen.getByText('No expiry')).toBeInTheDocument()
  })
})
