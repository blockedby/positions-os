import { FullConfig } from '@playwright/test'

async function globalSetup(config: FullConfig) {
  const baseURL = process.env.BASE_URL || config.projects[0].use.baseURL || 'http://localhost:3100'

  // Skip backend check if SKIP_BACKEND_CHECK is set
  if (process.env.SKIP_BACKEND_CHECK) {
    console.log('Global Setup: Skipping backend check (SKIP_BACKEND_CHECK=1)')
    return
  }

  console.log(`Global Setup: Checking backend at ${baseURL}...`)

  // Quick check - fail fast if backend not available (3 retries, 1s each)
  const maxRetries = process.env.CI ? 10 : 3
  const retryDelay = 1000

  for (let i = 0; i < maxRetries; i++) {
    try {
      const response = await fetch(`${baseURL}/api/v1/stats`, {
        signal: AbortSignal.timeout(2000)
      })
      if (response.ok) {
        console.log('Global Setup: Backend is ready!')
        return
      }
    } catch {
      if (i < maxRetries - 1) {
        console.log(`Global Setup: Backend not ready, retry ${i + 1}/${maxRetries}...`)
        await new Promise(r => setTimeout(r, retryDelay))
      }
    }
  }

  throw new Error(`Backend not available at ${baseURL}/api/v1/stats after ${maxRetries} attempts. Start backend first: go run ./cmd/collector/main.go`)
}

export default globalSetup
