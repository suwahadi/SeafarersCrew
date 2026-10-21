import { render, screen } from '@testing-library/react'
import { PreviewFrame } from './PreviewFrame'

describe('PreviewFrame', () => {
  it('renders the HTML in a fully sandboxed iframe', () => {
    render(<PreviewFrame html="<p>Hello</p><script>alert(1)</script>" />)
    const frame = screen.getByTitle('Document preview')
    expect(frame.tagName).toBe('IFRAME')
    expect(frame).toHaveAttribute('sandbox', '')
    expect(frame).toHaveAttribute('srcdoc', '<p>Hello</p><script>alert(1)</script>')
  })
})
