const path = require('node:path')
const { webcrypto } = require('node:crypto')
const { test, expect } = require('@playwright/test')

const decryptionScript = path.resolve(__dirname, '../../pkg/themes/default/static/js/decryption.js')
const testOrigin = 'http://localhost'
const textEncoder = new TextEncoder()

async function encrypt(plaintext, password) {
  const salt = webcrypto.getRandomValues(new Uint8Array(16))
  const nonce = webcrypto.getRandomValues(new Uint8Array(12))
  const passwordKey = await webcrypto.subtle.importKey(
    'raw',
    textEncoder.encode(password),
    'PBKDF2',
    false,
    ['deriveKey'],
  )
  const key = await webcrypto.subtle.deriveKey(
    {
      name: 'PBKDF2',
      salt,
      iterations: 100000,
      hash: 'SHA-256',
    },
    passwordKey,
    { name: 'AES-GCM', length: 256 },
    false,
    ['encrypt'],
  )
  const ciphertext = new Uint8Array(
    await webcrypto.subtle.encrypt(
      { name: 'AES-GCM', iv: nonce },
      key,
      textEncoder.encode(plaintext),
    ),
  )
  const combined = new Uint8Array(salt.length + nonce.length + ciphertext.length)
  combined.set(salt)
  combined.set(nonce, salt.length)
  combined.set(ciphertext, salt.length + nonce.length)
  return Buffer.from(combined).toString('base64')
}

function encryptedBlock({ id, ciphertext, keyName }) {
  const keyAttribute = keyName ? ` data-key-name="${keyName}"` : ''
  return `
    <div class="encrypted-content" data-test="${id}" data-encrypted="${ciphertext}"${keyAttribute}>
      <div class="encrypted-content__locked">
        <h2>Encrypted ${id}</h2>
        <form class="encrypted-content__form" action="#" method="post">
          <label for="${id}-input">Password</label>
          <input id="${id}-input" type="password" class="encrypted-content__input">
          <button type="button" class="encrypted-content__button">Decrypt</button>
        </form>
        <label><input type="checkbox" class="encrypted-content__remember"> Remember</label>
        <p class="encrypted-content__error" style="display: none"></p>
      </div>
      <div class="encrypted-content__decrypted" style="display: none"></div>
    </div>
  `
}

function testPage(content) {
  return `<!doctype html>
    <html>
      <head>
        <style>
          html { scroll-behavior: auto; }
          body { margin: 0; min-height: 3200px; font: 16px sans-serif; }
          .spacer { height: 900px; }
          .encrypted-content { margin: 24px; min-height: 180px; padding: 24px; border: 1px solid #888; }
          .encrypted-content__decrypted { min-height: 120px; }
          .sr-only { position: absolute; width: 1px; height: 1px; overflow: hidden; clip: rect(0, 0, 0, 0); }
        </style>
      </head>
      <body>${content}</body>
    </html>`
}

async function loadPage(page, html) {
  await page.route(`${testOrigin}/`, (route) =>
    route.fulfill({ status: 200, contentType: 'text/html', body: html }),
  )
  await page.goto(`${testOrigin}/`, { waitUntil: 'domcontentloaded' })
}

async function initializeDecryption(page) {
  await page.addScriptTag({ path: decryptionScript })
}

test('multiple encrypted blocks preserve load-time focus and scroll position', async ({ page }) => {
  const password = 'load-password' // pragma: allowlist secret
  const firstCiphertext = await encrypt('<p>First plaintext</p>', password)
  const secondCiphertext = await encrypt('<p>Second plaintext</p>', password)
  const html = testPage(`
    <div class="spacer"></div>
    ${encryptedBlock({ id: 'first', ciphertext: firstCiphertext, keyName: 'load' })}
    <div class="spacer"></div>
    ${encryptedBlock({ id: 'second', ciphertext: secondCiphertext, keyName: 'load' })}
  `)

  await loadPage(page, html)
  await page.evaluate(() => window.scrollTo(0, 500))
  const initialScroll = await page.evaluate(() => window.scrollY)

  await initializeDecryption(page)
  await page.waitForTimeout(50)

  expect(await page.evaluate(() => window.scrollY)).toBe(initialScroll)
  expect(
    await page.locator('.encrypted-content__input').evaluateAll((inputs) =>
      inputs.some((input) => input === document.activeElement),
    ),
  ).toBe(false)
  await expect(page.locator('.encrypted-content__input').last()).not.toBeFocused()
})

test('nested encrypted content initializes after outer content decrypts', async ({ page }) => {
  const outerPassword = 'outer-password' // pragma: allowlist secret
  const nestedPassword = 'nested-password' // pragma: allowlist secret
  const nestedCiphertext = await encrypt('<p>Nested plaintext</p>', nestedPassword)
  const outerCiphertext = await encrypt(
    `<p>Outer plaintext</p>${encryptedBlock({ id: 'nested', ciphertext: nestedCiphertext, keyName: 'nested' })}`,
    outerPassword,
  )
  const html = testPage(`
    <div class="spacer"></div>
    ${encryptedBlock({ id: 'outer', ciphertext: outerCiphertext, keyName: 'outer' })}
  `)

  await loadPage(page, html)
  await initializeDecryption(page)

  const outer = page.locator('[data-test="outer"]')
  await outer.locator('.encrypted-content__input').fill(outerPassword)
  await page.evaluate(() => window.scrollTo(0, 500))
  const scrollBeforeDecrypt = await page.evaluate(() => window.scrollY)
  await outer.locator('.encrypted-content__button').evaluate((button) => button.click())

  await expect(outer.locator('.encrypted-content__decrypted').first()).toContainText('Outer plaintext')
  await expect(outer.locator('.encrypted-content__decrypted').first()).toBeFocused()
  expect(await page.evaluate(() => window.scrollY)).toBe(scrollBeforeDecrypt)

  const nested = page.locator('[data-test="nested"]')
  await expect(nested).toBeVisible()
  await expect(nested.locator('.encrypted-content__input')).toBeVisible()
  await nested.locator('.encrypted-content__input').fill(nestedPassword)
  await nested.locator('.encrypted-content__button').click()
  await expect(nested.locator('.encrypted-content__decrypted')).toContainText('Nested plaintext')
})

test('remembered-password decryption preserves focus and scroll position', async ({ page }) => {
  const password = 'remembered-password' // pragma: allowlist secret
  const firstCiphertext = await encrypt('<p>Remembered first</p>', password)
  const secondCiphertext = await encrypt('<p>Remembered second</p>', password)
  const html = testPage(`
    <div class="spacer"></div>
    ${encryptedBlock({ id: 'remembered-first', ciphertext: firstCiphertext, keyName: 'remembered' })}
    <div class="spacer"></div>
    ${encryptedBlock({ id: 'remembered-second', ciphertext: secondCiphertext, keyName: 'remembered' })}
  `)

  await loadPage(page, html)
  await page.evaluate((savedPassword) => {
    sessionStorage.setItem('markata_decrypt_remembered', savedPassword)
    window.scrollTo(0, 500)
  }, password)
  const initialScroll = await page.evaluate(() => window.scrollY)

  await initializeDecryption(page)
  await expect(page.locator('.encrypted-content__decrypted').first()).toContainText('Remembered first')
  await expect(page.locator('.encrypted-content__decrypted').last()).toContainText('Remembered second')

  expect(await page.evaluate(() => window.scrollY)).toBe(initialScroll)
  expect(
    await page.evaluate(() => document.activeElement?.classList.contains('encrypted-content__input')),
  ).toBe(false)
  expect(
    await page.locator('.encrypted-content__decrypted').evaluateAll((elements) =>
      elements.some((element) => element === document.activeElement),
    ),
  ).toBe(false)
})

test('same-key encrypted content revealed during decryption unlocks with the outer block', async ({ page }) => {
  const password = 'shared-password' // pragma: allowlist secret
  const nestedCiphertext = await encrypt('<p>Same-key nested plaintext</p>', password)
  const outerCiphertext = await encrypt(
    `<p>Shared outer plaintext</p>${encryptedBlock({ id: 'same-key-nested', ciphertext: nestedCiphertext, keyName: 'shared' })}`,
    password,
  )
  const html = testPage(`
    <div class="spacer"></div>
    ${encryptedBlock({ id: 'same-key-outer', ciphertext: outerCiphertext, keyName: 'shared' })}
  `)

  await loadPage(page, html)
  await initializeDecryption(page)

  const outer = page.locator('[data-test="same-key-outer"]')
  await outer.locator('.encrypted-content__input').fill(password)
  await outer.locator('.encrypted-content__button').click()

  const nested = page.locator('[data-test="same-key-nested"]')
  await expect(nested.locator('.encrypted-content__decrypted')).toContainText('Same-key nested plaintext')
  await expect(nested.locator('.encrypted-content__locked')).toBeHidden()
})
