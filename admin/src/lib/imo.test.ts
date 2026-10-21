import { isValidIMO } from './imo'

describe('isValidIMO', () => {
  it.each([
    ['9074729', true],
    ['9074728', false],
    ['9176187', true],
    ['907472', false],
    ['90747290', false],
    ['abcdefg', false],
    ['', false],
  ])('%s → %s', (value, valid) => {
    expect(isValidIMO(value)).toBe(valid)
  })
})
