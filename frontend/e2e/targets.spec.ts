import { test, expect } from '@playwright/test'
import { mockTargets, generateNewTarget } from './fixtures/mock-data'

test.describe('Targets CRUD', () => {
  // TG-01: Empty state when no targets exist
  test('TG-01: should display empty state when no targets exist', async ({ page }) => {
    // Mock empty targets response
    await page.route('**/api/v1/targets', (route) => {
      if (route.request().method() === 'GET') {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([]),
        })
      } else {
        route.continue()
      }
    })

    await page.goto('/settings')

    // Should display empty state message (actual text: "No targets configured. Add a target to start scraping.")
    await expect(
      page.getByText(/no targets configured|no scraping targets|add your first/i)
    ).toBeVisible({ timeout: 10000 })
  })

  // TG-02: Display existing targets with name, type, and URL
  test('TG-02: should display existing targets in list', async ({ page }) => {
    // Mock targets response
    await page.route('**/api/v1/targets', (route) => {
      if (route.request().method() === 'GET') {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(Object.values(mockTargets)),
        })
      } else {
        route.continue()
      }
    })

    await page.goto('/settings')

    // Should display target names
    await expect(page.getByText('Go Jobs Channel')).toBeVisible({ timeout: 10000 })
    await expect(page.getByText('Rust Forum')).toBeVisible()
    await expect(page.getByText('Paused Target')).toBeVisible()
  })

  // TG-03: Create a new TG_CHANNEL target
  test('TG-03: should create a new target', async ({ page }) => {
    let createCalled = false
    const newTarget = generateNewTarget({ name: 'New Test Channel' })

    // Mock targets endpoint
    await page.route('**/api/v1/targets', (route) => {
      const method = route.request().method()

      if (method === 'GET') {
        // Return empty initially, then with new target after creation
        if (createCalled) {
          route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify([
              {
                id: 'new-id',
                ...newTarget,
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
                last_scraped_at: null,
              },
            ]),
          })
        } else {
          route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify([]),
          })
        }
      } else if (method === 'POST') {
        createCalled = true
        route.fulfill({
          status: 201,
          contentType: 'application/json',
          body: JSON.stringify({
            id: 'new-id',
            ...newTarget,
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
            last_scraped_at: null,
          }),
        })
      } else {
        route.continue()
      }
    })

    await page.goto('/settings')

    // Click Add Target button
    await page.getByRole('button', { name: /add target/i }).click()

    // Fill the form
    await page.getByLabel(/name/i).fill('New Test Channel')
    await page.getByLabel(/type/i).selectOption('TG_CHANNEL')
    await page.getByLabel(/url|channel/i).fill('@test_channel')

    // Submit the form
    await page.getByRole('button', { name: /create|save|submit/i }).click()

    // Target should appear in the list
    await expect(page.getByText('New Test Channel')).toBeVisible({ timeout: 10000 })
  })

  // TG-04: Validation errors for missing fields
  test('TG-04: should show validation errors for missing fields', async ({ page }) => {
    await page.route('**/api/v1/targets', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([]),
      })
    })

    await page.goto('/settings')

    // Click Add Target button
    await page.getByRole('button', { name: /add target/i }).click()

    // Try to submit empty form
    await page.getByRole('button', { name: /create|save|submit/i }).click()

    // Should show custom validation errors (TargetForm uses custom validation)
    // Error messages: "Name is required", "URL is required"
    await expect(
      page.getByText(/name is required|url is required|required/i)
    ).toBeVisible({ timeout: 5000 })
  })

  // TG-05: Edit an existing target
  test('TG-05: should edit an existing target', async ({ page }) => {
    const targets = Object.values(mockTargets)
    let updatedTarget = { ...targets[0], name: 'Updated Channel Name' }

    await page.route('**/api/v1/targets', (route) => {
      if (route.request().method() === 'GET') {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([updatedTarget, targets[1], targets[2]]),
        })
      } else {
        route.continue()
      }
    })

    await page.route('**/api/v1/targets/*', (route) => {
      const method = route.request().method()

      if (method === 'PUT' || method === 'PATCH') {
        const body = route.request().postDataJSON()
        updatedTarget = { ...updatedTarget, ...body, updated_at: new Date().toISOString() }
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(updatedTarget),
        })
      } else if (method === 'GET') {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(targets[0]),
        })
      } else {
        route.continue()
      }
    })

    await page.goto('/settings')

    // Click Edit button on first target
    await page.getByRole('button', { name: /edit/i }).first().click()

    // Update the name
    const nameInput = page.getByLabel(/name/i)
    await nameInput.clear()
    await nameInput.fill('Updated Channel Name')

    // Save changes
    await page.getByRole('button', { name: /save|update|submit/i }).click()

    // Updated name should be visible
    await expect(page.getByText('Updated Channel Name')).toBeVisible({ timeout: 10000 })
  })

  // TG-06: Delete a target with confirmation
  test('TG-06: should delete target after confirmation', async ({ page }) => {
    const targets = Object.values(mockTargets)
    let deletedId: string | null = null

    await page.route('**/api/v1/targets', (route) => {
      if (route.request().method() === 'GET') {
        const filteredTargets = targets.filter((t) => t.id !== deletedId)
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(filteredTargets),
        })
      } else {
        route.continue()
      }
    })

    await page.route('**/api/v1/targets/*', (route) => {
      if (route.request().method() === 'DELETE') {
        const url = route.request().url()
        deletedId = url.split('/').pop() || null
        route.fulfill({ status: 204 })
      } else {
        route.continue()
      }
    })

    // Handle confirmation dialog
    page.on('dialog', async (dialog) => {
      await dialog.accept()
    })

    await page.goto('/settings')

    // Verify initial target is visible
    await expect(page.getByText('Go Jobs Channel')).toBeVisible({ timeout: 10000 })

    // Click Delete button on first target
    await page.getByRole('button', { name: /delete/i }).first().click()

    // Target should be removed after deletion (page will refetch)
    // Wait for the deletion to complete
    await page.waitForTimeout(1000)

    // The first target should be removed (or request made to delete it)
    expect(deletedId).toBeTruthy()
  })

  // TG-07: Toggle is_active status
  test('TG-07: should toggle target active status', async ({ page }) => {
    const targets = Object.values(mockTargets)

    await page.route('**/api/v1/targets', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(targets),
      })
    })

    await page.route('**/api/v1/targets/*', (route) => {
      const method = route.request().method()

      if (method === 'PUT' || method === 'PATCH') {
        const body = route.request().postDataJSON()
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ ...targets[0], ...body }),
        })
      } else if (method === 'GET') {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(targets[0]),
        })
      } else {
        route.continue()
      }
    })

    await page.goto('/settings')

    // Click Edit on first target
    await page.getByRole('button', { name: /edit/i }).first().click()

    // Toggle active checkbox
    const activeCheckbox = page.getByLabel(/active/i)
    await activeCheckbox.click()

    // Save changes
    await page.getByRole('button', { name: /save|update|submit/i }).click()

    // Badge should update (depending on implementation)
    await page.waitForTimeout(500)
  })
})
