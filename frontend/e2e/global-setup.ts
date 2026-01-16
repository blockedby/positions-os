import { chromium, FullConfig } from '@playwright/test'

async function globalSetup(config: FullConfig) {
  const baseURL = config.projects[0].use.baseURL || 'http://localhost:3100'

  console.log('Global Setup: Waiting for backend...')

  // Wait for backend to be ready
  const browser = await chromium.launch()
  const page = await browser.newPage()

  let retries = 30
  while (retries > 0) {
    try {
      const response = await page.goto(`${baseURL}/api/v1/stats`, {
        timeout: 5000,
      })
      if (response?.ok()) {
        console.log('Backend is ready!')
        break
      }
    } catch {
      retries--
      if (retries === 0) {
        await browser.close()
        throw new Error('Backend did not start in time')
      }
      console.log(`Waiting for backend... (${retries} retries left)`)
      await page.waitForTimeout(1000)
    }
  }

  await browser.close()
}

export default globalSetup
