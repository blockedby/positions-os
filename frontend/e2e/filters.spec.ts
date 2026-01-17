import { test, expect } from '@playwright/test'
import { mockJobs } from './fixtures/mock-data'

test.describe('Job Filters', () => {
  test.beforeEach(async ({ page }) => {
    // Mock jobs API
    await page.route('**/api/v1/jobs*', (route) => {
      const url = new URL(route.request().url())
      const technologies = url.searchParams.get('technologies')
      const salaryMin = url.searchParams.get('salary_min')
      const salaryMax = url.searchParams.get('salary_max')

      // Filter mock data based on query params
      let filteredJobs = Object.values(mockJobs)

      if (technologies) {
        const techList = technologies.split(',').map((t) => t.toLowerCase())
        filteredJobs = filteredJobs.filter((job) =>
          job.structured_data?.technologies?.some((t: string) =>
            techList.includes(t.toLowerCase())
          )
        )
      }

      if (salaryMin) {
        const min = parseInt(salaryMin, 10)
        filteredJobs = filteredJobs.filter(
          (job) => job.structured_data?.salary_min && job.structured_data.salary_min >= min
        )
      }

      if (salaryMax) {
        const max = parseInt(salaryMax, 10)
        filteredJobs = filteredJobs.filter(
          (job) => job.structured_data?.salary_max && job.structured_data.salary_max <= max
        )
      }

      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          jobs: filteredJobs,
          total: filteredJobs.length,
          page: 1,
          limit: 10,
          pages: 1,
        }),
      })
    })

    // Mock stats API
    await page.route('**/api/v1/stats', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          total_jobs: 100,
          analyzed_jobs: 80,
          interested_jobs: 20,
          rejected_jobs: 10,
          today_jobs: 5,
          active_targets: 2,
        }),
      })
    })
  })

  // FLT-01: Technology filter input exists
  test('FLT-01: should display technology filter input', async ({ page }) => {
    await page.goto('/jobs')

    const techInput = page.getByPlaceholder(/technologies/i)
    await expect(techInput).toBeVisible({ timeout: 10000 })
  })

  // FLT-02: Salary min/max inputs exist
  test('FLT-02: should display salary min and max inputs', async ({ page }) => {
    await page.goto('/jobs')

    const minInput = page.getByPlaceholder(/min salary/i)
    const maxInput = page.getByPlaceholder(/max salary/i)

    await expect(minInput).toBeVisible({ timeout: 10000 })
    await expect(maxInput).toBeVisible()
  })

  // FLT-03: Applying technology filter
  test('FLT-03: should apply technology filter', async ({ page }) => {
    let requestUrl = ''

    // Set up route to capture the request URL
    await page.route('**/api/v1/jobs*', (route) => {
      requestUrl = route.request().url()
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          jobs: [],
          total: 0,
          page: 1,
          limit: 10,
          pages: 0,
        }),
      })
    })

    await page.goto('/jobs')
    await page.waitForTimeout(500) // Wait for initial load

    // Enter technologies
    const techInput = page.getByPlaceholder(/technologies/i)
    await techInput.fill('Go, PostgreSQL')

    // Reset requestUrl to capture the filter request
    requestUrl = ''

    // Click Apply
    await page.getByRole('button', { name: /apply/i }).click()

    // Wait for the request to be made
    await page.waitForTimeout(1000)

    expect(requestUrl).toContain('technologies=')
  })

  // FLT-04: Applying salary range filter
  test('FLT-04: should apply salary range filter', async ({ page }) => {
    let requestUrl = ''

    // Set up route to capture the request URL
    await page.route('**/api/v1/jobs*', (route) => {
      requestUrl = route.request().url()
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          jobs: [],
          total: 0,
          page: 1,
          limit: 10,
          pages: 0,
        }),
      })
    })

    await page.goto('/jobs')
    await page.waitForTimeout(500) // Wait for initial load

    // Enter salary range
    const minInput = page.getByPlaceholder(/min salary/i)
    const maxInput = page.getByPlaceholder(/max salary/i)

    await minInput.fill('150000')
    await maxInput.fill('300000')

    // Reset requestUrl to capture the filter request
    requestUrl = ''

    // Click Apply
    await page.getByRole('button', { name: /apply/i }).click()

    // Wait for the request to be made
    await page.waitForTimeout(1000)

    expect(requestUrl).toContain('salary_min=150000')
    expect(requestUrl).toContain('salary_max=300000')
  })

  // FLT-05: Clearing filters
  test('FLT-05: should clear all filters', async ({ page }) => {
    // Set up route
    await page.route('**/api/v1/jobs*', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          jobs: [],
          total: 0,
          page: 1,
          limit: 10,
          pages: 0,
        }),
      })
    })

    await page.goto('/jobs')
    await page.waitForTimeout(500)

    // Enter some filter values
    const techInput = page.getByPlaceholder(/technologies/i)
    const minInput = page.getByPlaceholder(/min salary/i)

    await techInput.fill('Go')
    await minInput.fill('100000')

    // Click Clear
    await page.getByRole('button', { name: /clear/i }).click()

    // Inputs should be empty
    await expect(techInput).toHaveValue('')
    await expect(minInput).toHaveValue('')
  })

  // FLT-06: Combining multiple filters
  test('FLT-06: should apply multiple filters together', async ({ page }) => {
    let requestUrl = ''

    // Set up route to capture the request URL
    await page.route('**/api/v1/jobs*', (route) => {
      requestUrl = route.request().url()
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          jobs: [],
          total: 0,
          page: 1,
          limit: 10,
          pages: 0,
        }),
      })
    })

    await page.goto('/jobs')
    await page.waitForTimeout(500) // Wait for initial load

    // Enter all filter values
    const techInput = page.getByPlaceholder(/technologies/i)
    const minInput = page.getByPlaceholder(/min salary/i)
    const maxInput = page.getByPlaceholder(/max salary/i)

    await techInput.fill('Rust')
    await minInput.fill('200000')
    await maxInput.fill('400000')

    // Reset requestUrl to capture the filter request
    requestUrl = ''

    // Click Apply
    await page.getByRole('button', { name: /apply/i }).click()

    // Wait for the request to be made
    await page.waitForTimeout(1000)

    expect(requestUrl).toContain('technologies=Rust')
    expect(requestUrl).toContain('salary_min=200000')
    expect(requestUrl).toContain('salary_max=400000')
  })
})

test.describe('Dashboard ScrapeStatus', () => {
  // DSH-01: ScrapeStatus shows idle when not scraping
  test('DSH-01: should display idle status when not scraping', async ({ page }) => {
    // Mock scrape status API
    await page.route('**/api/v1/scrape/status', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ is_scraping: false }),
      })
    })

    // Mock stats API
    await page.route('**/api/v1/stats', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          total_jobs: 100,
          analyzed_jobs: 80,
          interested_jobs: 20,
          rejected_jobs: 10,
          today_jobs: 5,
          active_targets: 2,
        }),
      })
    })

    // Mock jobs API for recent jobs
    await page.route('**/api/v1/jobs*', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          jobs: [],
          total: 0,
          page: 1,
          limit: 8,
          pages: 0,
        }),
      })
    })

    await page.goto('/')

    // Should show "Idle" status
    await expect(page.getByText(/idle/i)).toBeVisible({ timeout: 10000 })
  })

  // DSH-02: ScrapeStatus shows progress when scraping
  test('DSH-02: should display progress when actively scraping', async ({ page }) => {
    // Mock scrape status API with active scraping
    await page.route('**/api/v1/scrape/status', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          is_scraping: true,
          target: '@golang_jobs',
          processed: 50,
          new_jobs: 12,
        }),
      })
    })

    // Mock stats API
    await page.route('**/api/v1/stats', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          total_jobs: 100,
          analyzed_jobs: 80,
          interested_jobs: 20,
          rejected_jobs: 10,
          today_jobs: 5,
          active_targets: 2,
        }),
      })
    })

    // Mock jobs API for recent jobs
    await page.route('**/api/v1/jobs*', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          jobs: [],
          total: 0,
          page: 1,
          limit: 8,
          pages: 0,
        }),
      })
    })

    await page.goto('/')

    // Should show scraping progress
    await expect(page.getByText('@golang_jobs')).toBeVisible({ timeout: 10000 })
    await expect(page.getByText(/50 processed/i)).toBeVisible()
    await expect(page.getByText(/12 new jobs/i)).toBeVisible()
  })
})
