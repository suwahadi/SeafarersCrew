import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { ImageField } from './ImageField'

const upload = vi.fn()

vi.mock('@/api/files', () => ({ filesApi: { upload: (...args: unknown[]) => upload(...args) as unknown } }))

describe('ImageField', () => {
  it('uploads the chosen image with the CMS context and reports the new file id', async () => {
    upload.mockResolvedValue({ id: 'file-2', original_name: 'hero.jpg' })
    URL.createObjectURL = vi.fn(() => 'blob:preview')
    URL.revokeObjectURL = vi.fn()
    const onChange = vi.fn()
    render(<ImageField label="Hero image" value="" onChange={onChange} />)

    await userEvent.upload(screen.getByLabelText('Hero image'), new File(['x'], 'hero.jpg', { type: 'image/jpeg' }))

    expect(upload).toHaveBeenCalledWith(expect.any(File), 'cms_image')
    expect(onChange).toHaveBeenCalledWith('file-2')
  })

  it('shows the stored thumbnail for the current file and lets optional images be removed', async () => {
    const onChange = vi.fn()
    const { container } = render(
      <ImageField label="OG image" value="f1" onChange={onChange} image={{ file_id: 'f1', url: 'https://x/a.webp', thumb_url: 'https://x/a-480.webp', mime_type: 'image/webp', width: 1600, height: 900 }} />,
    )
    expect(container.querySelector('img')?.getAttribute('src')).toBe('https://x/a-480.webp')
    await userEvent.click(screen.getByRole('button', { name: 'Remove' }))
    expect(onChange).toHaveBeenCalledWith('')
  })

  it('shows the upload error from the server', async () => {
    upload.mockRejectedValue(new Error('The image could not be processed.'))
    render(<ImageField label="Logo" value="" onChange={vi.fn()} context="logo" />)
    await userEvent.upload(screen.getByLabelText('Logo'), new File(['x'], 'l.png', { type: 'image/png' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('The image could not be processed.')
  })
})
