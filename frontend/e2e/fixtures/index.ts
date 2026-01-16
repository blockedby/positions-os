import { test as base } from '@playwright/test'
import { mockTargets, mockJobs, mockStats } from './mock-data'

// Re-export from @playwright/test
export { expect } from '@playwright/test'

// Extended test with mock API support
export const test = base.extend<{
  mockApi: void
}>({
  mockApi: async ({ page }, use) => {
    // Mock all API endpoints
    await page.route('**/api/v1/targets', (route) => {
      const method = route.request().method()

      if (method === 'GET') {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(Object.values(mockTargets)),
        })
      } else if (method === 'POST') {
        const body = route.request().postDataJSON()
        const newTarget = {
          id: 'new-target-id',
          ...body,
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
          last_scraped_at: null,
        }
        route.fulfill({
          status: 201,
          contentType: 'application/json',
          body: JSON.stringify(newTarget),
        })
      } else {
        route.continue()
      }
    })

    await page.route('**/api/v1/targets/*', (route) => {
      const method = route.request().method()
      const url = route.request().url()
      const id = url.split('/').pop()

      if (method === 'GET') {
        const target = Object.values(mockTargets).find((t) => t.id === id)
        if (target) {
          route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify(target),
          })
        } else {
          route.fulfill({ status: 404, body: JSON.stringify({ error: 'Not found' }) })
        }
      } else if (method === 'PUT' || method === 'PATCH') {
        const body = route.request().postDataJSON()
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ id, ...body, updated_at: new Date().toISOString() }),
        })
      } else if (method === 'DELETE') {
        route.fulfill({ status: 204 })
      } else {
        route.continue()
      }
    })

    await page.route('**/api/v1/jobs*', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          jobs: Object.values(mockJobs),
          total: 3,
          page: 1,
          per_page: 20,
        }),
      })
    })

    await page.route('**/api/v1/stats', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(mockStats),
      })
    })

    await use()
  },
})
