import { FullConfig } from '@playwright/test'

async function globalTeardown(_config: FullConfig) {
  console.log('Global Teardown: Cleaning up...')
  // Add cleanup logic if needed (e.g., reset database state)
}

export default globalTeardown
