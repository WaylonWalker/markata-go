const { test, expect } = require('@playwright/test')

const readerURL = process.env.READER_OVERFLOW_URL

test('DuckDB reader cards do not overflow horizontally', async ({ page }) => {
  expect(readerURL, 'READER_OVERFLOW_URL must be set').toBeTruthy()

  await page.goto(readerURL, { waitUntil: 'networkidle' })

  const issues = await page.locator('.reader-entry').evaluateAll((cards) => {
    const epsilon = 1

    function overflowsCard(card, element) {
      if (!card || !element) return false
      const cardRect = card.getBoundingClientRect()
      const rect = element.getBoundingClientRect()
      return rect.left < cardRect.left - epsilon || rect.right > cardRect.right + epsilon
    }

    return cards.flatMap((card) => {
      const source = card.querySelector('.reader-entry-source')
      const title = card.querySelector('.reader-entry-title a')
      const text = card.textContent || ''
      if (!text.includes('DuckDB') && !text.includes('DuckLake') && !text.includes('OLAP')) {
        return []
      }

      const problems = []
      if (card.scrollWidth - card.clientWidth > epsilon) {
        problems.push({ type: 'card-scroll', text: text.trim().slice(0, 140) })
      }
      if (overflowsCard(card, source)) {
        problems.push({ type: 'source-overflow', text: source?.textContent?.trim() || '' })
      }
      if (overflowsCard(card, title)) {
        problems.push({ type: 'title-overflow', text: title?.textContent?.trim() || '' })
      }
      return problems
    })
  })

  expect(issues).toEqual([])
})
