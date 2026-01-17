import { test, expect } from '@playwright/test'
import { mockJobs } from './fixtures/mock-data'

test.describe('Prepare Application Flow', () => {
  const interestedJob = {
    ...mockJobs.interested,
    id: '660e8400-e29b-41d4-a716-446655440003',
    status: 'INTERESTED',
    structured_data: {
      title: 'Rust Developer',
      company: 'FinTech Startup',
      description: 'Rust developer for fintech startup...',
      salary_min: 200000,
      salary_max: 350000,
      currency: 'RUB',
      is_remote: true,
      language: 'RU',
      technologies: ['Rust', 'Tokio', 'PostgreSQL'],
      experience_years: 3,
      contacts: ['@hr_contact'],
    },
    raw_content: 'Rust developer for fintech startup...',
    created_at: '2025-01-14T08:00:00Z',
    updated_at: '2025-01-14T12:00:00Z',
  }

  test.beforeEach(async ({ page }) => {
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

    // Mock jobs list API
    await page.route('**/api/v1/jobs', (route) => {
      if (route.request().method() === 'GET') {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            jobs: [interestedJob],
            total: 1,
            page: 1,
            limit: 10,
            pages: 1,
          }),
        })
      }
    })

    // Mock single job API
    await page.route(`**/api/v1/jobs/${interestedJob.id}`, (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(interestedJob),
      })
    })
  })

  test('PAP-01: should show Prepare Application button for INTERESTED job', async ({ page }) => {
    // Navigate directly to job detail via URL
    await page.goto(`/jobs?id=${interestedJob.id}`)

    // Should see the Prepare Application button in the detail panel
    await expect(page.getByRole('button', { name: /prepare application/i })).toBeVisible({ timeout: 10000 })
  })

  test('PAP-02: should call prepareJob API when button clicked', async ({ page }) => {
    let prepareJobCalled = false
    let prepareJobId = ''

    // Mock prepareJob API
    await page.route(`**/api/v1/jobs/*/prepare`, (route) => {
      prepareJobCalled = true
      prepareJobId = route.request().url().split('/jobs/')[1].split('/prepare')[0]

      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          job_id: prepareJobId,
          status: 'TAILORED_APPROVED',
          resume_path: `/storage/jobs/${prepareJobId}/resume.pdf`,
          cover_letter_path: `/storage/jobs/${prepareJobId}/cover_letter.md`,
        }),
      })
    })

    // Navigate directly to job detail via URL
    await page.goto(`/jobs?id=${interestedJob.id}`)

    // Wait for detail to load
    await expect(page.getByRole('button', { name: /prepare application/i })).toBeVisible({ timeout: 10000 })

    // Click Prepare Application
    await page.getByRole('button', { name: /prepare application/i }).click()

    // Wait for API call
    await page.waitForTimeout(1000)

    // Verify API was called
    expect(prepareJobCalled).toBe(true)
    expect(prepareJobId).toBe(interestedJob.id)
  })

  test('PAP-03: should not show Prepare Application button for ANALYZED job', async ({ page }) => {
    const analyzedJob = {
      ...interestedJob,
      id: '660e8400-e29b-41d4-a716-446655440004',
      status: 'ANALYZED',
    }

    // Override routes for this test
    await page.route('**/api/v1/jobs', (route) => {
      if (route.request().method() === 'GET') {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            jobs: [analyzedJob],
            total: 1,
            page: 1,
            limit: 10,
            pages: 1,
          }),
        })
      }
    })

    await page.route(`**/api/v1/jobs/${analyzedJob.id}`, (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(analyzedJob),
      })
    })

    // Navigate directly to job detail via URL
    await page.goto(`/jobs?id=${analyzedJob.id}`)

    // Wait for detail panel to load (check for job-detail class)
    const detailPanel = page.locator('.job-detail')
    await expect(detailPanel).toBeVisible({ timeout: 10000 })

    // Should NOT see the Prepare Application button
    await expect(page.getByRole('button', { name: /prepare application/i })).not.toBeVisible()
  })

  test('PAP-04: should not show Prepare Application button for TAILORED_APPROVED job', async ({ page }) => {
    const tailoredJob = {
      ...interestedJob,
      id: '660e8400-e29b-41d4-a716-446655440005',
      status: 'TAILORED_APPROVED',
    }

    // Override routes for this test
    await page.route('**/api/v1/jobs', (route) => {
      if (route.request().method() === 'GET') {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            jobs: [tailoredJob],
            total: 1,
            page: 1,
            limit: 10,
            pages: 1,
          }),
        })
      }
    })

    await page.route(`**/api/v1/jobs/${tailoredJob.id}`, (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(tailoredJob),
      })
    })

    // Navigate directly to job detail via URL
    await page.goto(`/jobs?id=${tailoredJob.id}`)

    // Wait for detail panel to load (check for job-detail class)
    const detailPanel = page.locator('.job-detail')
    await expect(detailPanel).toBeVisible({ timeout: 10000 })

    // Should NOT see the Prepare Application button
    await expect(page.getByRole('button', { name: /prepare application/i })).not.toBeVisible()
  })
})
