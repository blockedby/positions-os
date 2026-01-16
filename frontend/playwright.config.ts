import { defineConfig, devices } from '@playwright/test'

export default defineConfig({
  // Test directory
  testDir: './e2e',

  // Run tests in parallel
  fullyParallel: true,

  // Fail build on CI if test.only left in code
  forbidOnly: !!process.env.CI,

  // Retry failed tests (more on CI)
  retries: process.env.CI ? 2 : 0,

  // Limit workers for stability
  workers: process.env.CI ? 1 : 4,

  // Reporter configuration
  reporter: [['html', { outputFolder: 'playwright-report' }], ['list']],

  // Shared settings for all projects
  use: {
    // Base URL for navigation (React frontend on port 3000, API on 3100)
    baseURL: process.env.BASE_URL || 'http://localhost:3000',

    // Capture trace on first retry
    trace: 'on-first-retry',

    // Screenshot on failure
    screenshot: 'only-on-failure',

    // Video on failure
    video: 'on-first-retry',

    // Timeout for actions
    actionTimeout: 10000,

    // Timeout for navigation
    navigationTimeout: 30000,
  },

  // Test timeout (15 seconds per test)
  timeout: 15000,

  // Expect timeout
  expect: {
    timeout: 5000,
  },

  // Browser projects
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],

  // Output directory for test artifacts
  outputDir: 'test-results',

  // Global setup/teardown
  globalSetup: './e2e/global-setup.ts',
  globalTeardown: './e2e/global-teardown.ts',
})
