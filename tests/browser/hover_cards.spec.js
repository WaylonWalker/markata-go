const { test, expect } = require('@playwright/test')

// Point HOVER_CARDS_URL at a built page containing a wikilink, a glossary
// term, a [data-hover-title] element, a [data-hover-card] element, and an
// external link annotated by the external_link_hover plugin.
const pageURL = process.env.HOVER_CARDS_URL

async function hoverAndGetCard(page, selector) {
  const target = page.locator(selector).first()
  await target.evaluate((el) => el.scrollIntoView({ block: 'center', behavior: 'instant' }))
  await page.mouse.move(0, 0)
  await target.hover()
  const card = page.locator('.hover-card')
  await expect(card).toHaveCount(1)
  return { target, card }
}

async function expectCardClearOfTarget(target, card) {
  const lines = await target.evaluate((el) => [...el.getClientRects()].map((r) => ({ top: r.top, bottom: r.bottom })))
  const box = await card.boundingBox()
  for (const line of lines) {
    const overlaps = box.y < line.bottom - 1 && box.y + box.height > line.top + 1
    expect(overlaps, 'hover card must not cover its target').toBeFalsy()
  }
  const viewport = target.page().viewportSize()
  expect(box.x).toBeGreaterThanOrEqual(0)
  expect(box.x + box.width).toBeLessThanOrEqual(viewport.width)
}

test.describe('hover cards', () => {
  test.beforeEach(async ({ page }) => {
    expect(pageURL, 'HOVER_CARDS_URL must be set').toBeTruthy()
    await page.goto(pageURL, { waitUntil: 'networkidle' })
  })

  test('wikilinks show title, description, and path', async ({ page }) => {
    const { target, card } = await hoverAndGetCard(page, 'a.wikilink[data-title]')
    await expect(card.locator('.tooltip-title')).toHaveText(await target.getAttribute('data-title'))
    await expect(card.locator('.tooltip-path')).not.toBeEmpty()
    await expectCardClearOfTarget(target, card)
  })

  test('glossary terms use the card instead of the native title', async ({ page }) => {
    const { target, card } = await hoverAndGetCard(page, 'a.glossary-term')
    await expect(card).toHaveClass(/hover-card--glossary/)
    await expect(card.locator('.tooltip-desc')).not.toBeEmpty()
    await expect(target).not.toHaveAttribute('title', /.+/)
  })

  test('custom data-hover-title and template cards render', async ({ page }) => {
    let { card } = await hoverAndGetCard(page, '[data-hover-title]:not(a)')
    await expect(card.locator('.tooltip-title')).not.toBeEmpty()

    await page.mouse.move(0, 0)
    await expect(page.locator('.hover-card')).toHaveCount(0)

    const rich = await hoverAndGetCard(page, '[data-hover-card]')
    await expect(rich.card.locator('.hover-card-body')).not.toBeEmpty()
    await expectCardClearOfTarget(rich.target, rich.card)
  })

  test('external links show site metadata and never a blank image', async ({ page }) => {
    const { target, card } = await hoverAndGetCard(page, 'a[data-link-preview]')
    await expect(card).toHaveClass(/hover-card--external/)
    const title = (await target.getAttribute('data-link-title')) || (await target.getAttribute('data-link-host'))
    await expect(card.locator('.tooltip-title')).toHaveText(title)
    await expect(card.locator('.tooltip-site')).toContainText('\u2197')
    await expect(target).not.toHaveAttribute('title', /.+/)
    // Images stay hidden until they load, so a slow or blocked image takes no space.
    const blank = await card.locator('.hover-card-image').evaluateAll((imgs) =>
      imgs.some((img) => !img.hidden && !(img.complete && img.naturalWidth > 0))
    )
    expect(blank, 'unloaded preview images must stay hidden').toBeFalsy()
    await expectCardClearOfTarget(target, card)

    await page.mouse.move(0, 0)
    await expect(page.locator('.hover-card')).toHaveCount(0)
    const optOut = page.locator('a[data-no-preview]').first()
    if (await optOut.count()) {
      await expect(optOut).not.toHaveAttribute('data-link-preview', /.*/)
    }
  })

  test('the pointer can move into a rich card and Escape closes it', async ({ page }) => {
    const { card } = await hoverAndGetCard(page, '[data-hover-card]')
    const box = await card.boundingBox()
    await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2, { steps: 5 })
    await page.waitForTimeout(300)
    await expect(card).toBeVisible()
    await page.keyboard.press('Escape')
    await expect(page.locator('.hover-card')).toHaveCount(0)
  })

  for (const [line, expected] of [['first', 'above'], ['last', 'below']]) {
    test(`wrapped wikilinks open ${expected} when hovering the ${line} line`, async ({ page }) => {
      await page.setViewportSize({ width: 420, height: 800 })
      const multiLine = await page.locator('a.wikilink[data-title]').evaluateAll((links) =>
        links.findIndex((a) => a.getClientRects().length > 1))
      test.skip(multiLine < 0, 'no wrapped wikilink on this page')
      const target = page.locator('a.wikilink[data-title]').nth(multiLine)
      await target.evaluate((el) => el.scrollIntoView({ block: 'center', behavior: 'instant' }))
      const rects = await target.evaluate((el) => [...el.getClientRects()].map((r) => r.toJSON()))
      const r = line === 'first' ? rects[0] : rects[rects.length - 1]
      await page.mouse.move(r.left + r.width / 2, r.top + r.height / 2, { steps: 3 })
      const card = page.locator('.hover-card')
      await expect(card).toHaveCount(1)
      await expect(card).toHaveAttribute('data-placement', expected)
      await expectCardClearOfTarget(target, card)
    })
  }

  test('keyboard focus opens a card on custom targets', async ({ page }) => {
    const target = page.locator('[data-hover-title]:not(a)').first()
    await expect(target).toHaveAttribute('tabindex', '0')
    await target.evaluate((el) => el.scrollIntoView({ block: 'center', behavior: 'instant' }))
    // A key press first makes programmatic focus count as :focus-visible.
    await page.keyboard.press('Shift')
    await target.focus()
    await expect(page.locator('.hover-card')).toHaveCount(1)
  })
})
