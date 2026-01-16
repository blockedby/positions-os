import { test, expect } from '@playwright/test'

test.describe('API Responses', () => {
  // API-01: Targets endpoint returns empty array, not null
  test('API-01: should return empty array for empty targets list', async ({ request, baseURL }) => {
    const response = await request.get(`${baseURL}/api/v1/targets`)

    expect(response.ok()).toBeTruthy()
    expect(response.status()).toBe(200)

    const body = await response.json()

    // Critical: Should be an array, not null
    expect(Array.isArray(body)).toBe(true)

    // If empty, should be empty array
    if (body.length === 0) {
      expect(body).toEqual([])
    }
  })

  // API-02: Create target via JSON API
  test('API-02: should create target via JSON API', async ({ request, baseURL }) => {
    const newTarget = {
      name: `E2E Test Target ${Date.now()}`,
      type: 'TG_CHANNEL',
      url: '@e2e_test_channel',
      is_active: true,
    }

    const response = await request.post(`${baseURL}/api/v1/targets`, {
      data: newTarget,
    })

    expect(response.status()).toBe(201)

    const body = await response.json()
    expect(body.name).toBe(newTarget.name)
    expect(body.type).toBe(newTarget.type)
    expect(body.url).toBe(newTarget.url)
    expect(body.id).toBeDefined()
    expect(body.created_at).toBeDefined()

    // Cleanup: Delete the created target
    if (body.id) {
      await request.delete(`${baseURL}/api/v1/targets/${body.id}`)
    }
  })

  // API-03: Update target via JSON API
  test('API-03: should update target via JSON API', async ({ request, baseURL }) => {
    // First, create a target
    const createResponse = await request.post(`${baseURL}/api/v1/targets`, {
      data: {
        name: `E2E Update Test ${Date.now()}`,
        type: 'TG_CHANNEL',
        url: '@e2e_update_test',
        is_active: true,
      },
    })

    expect(createResponse.status()).toBe(201)
    const created = await createResponse.json()

    // Update the target
    const updateResponse = await request.put(`${baseURL}/api/v1/targets/${created.id}`, {
      data: {
        name: 'Updated E2E Target',
        type: 'TG_CHANNEL',
        url: '@e2e_update_test',
        is_active: false,
      },
    })

    expect(updateResponse.ok()).toBeTruthy()
    const updated = await updateResponse.json()
    expect(updated.name).toBe('Updated E2E Target')
    expect(updated.is_active).toBe(false)

    // Cleanup
    await request.delete(`${baseURL}/api/v1/targets/${created.id}`)
  })

  // API-04: Get single target by ID
  test('API-04: should get target by ID', async ({ request, baseURL }) => {
    // First, create a target
    const createResponse = await request.post(`${baseURL}/api/v1/targets`, {
      data: {
        name: `E2E GetById Test ${Date.now()}`,
        type: 'TG_CHANNEL',
        url: '@e2e_getbyid_test',
        is_active: true,
      },
    })

    expect(createResponse.status()).toBe(201)
    const created = await createResponse.json()

    // Get by ID
    const getResponse = await request.get(`${baseURL}/api/v1/targets/${created.id}`)
    expect(getResponse.ok()).toBeTruthy()

    const target = await getResponse.json()
    expect(target.id).toBe(created.id)
    expect(target.name).toBe(created.name)

    // Cleanup
    await request.delete(`${baseURL}/api/v1/targets/${created.id}`)
  })

  // API-05: Delete target returns 204
  test('API-05: should delete target and return 204', async ({ request, baseURL }) => {
    // First, create a target
    const createResponse = await request.post(`${baseURL}/api/v1/targets`, {
      data: {
        name: `E2E Delete Test ${Date.now()}`,
        type: 'TG_CHANNEL',
        url: '@e2e_delete_test',
        is_active: true,
      },
    })

    expect(createResponse.status()).toBe(201)
    const created = await createResponse.json()

    // Delete the target
    const deleteResponse = await request.delete(`${baseURL}/api/v1/targets/${created.id}`)
    expect(deleteResponse.status()).toBe(204)

    // Verify it's gone
    const getResponse = await request.get(`${baseURL}/api/v1/targets/${created.id}`)
    expect(getResponse.status()).toBe(404)
  })

  // API-06: Stats endpoint returns valid response
  test('API-06: should return stats with correct structure', async ({ request, baseURL }) => {
    const response = await request.get(`${baseURL}/api/v1/stats`)

    expect(response.ok()).toBeTruthy()

    const stats = await response.json()
    expect(stats).toHaveProperty('total_jobs')
    expect(stats).toHaveProperty('by_status')
    expect(typeof stats.total_jobs).toBe('number')
  })

  // API-07: Jobs endpoint returns paginated response
  test('API-07: should return jobs with pagination', async ({ request, baseURL }) => {
    const response = await request.get(`${baseURL}/api/v1/jobs`)

    expect(response.ok()).toBeTruthy()

    const body = await response.json()

    // Should have pagination metadata or be an array
    if (Array.isArray(body)) {
      // Simple array response
      expect(Array.isArray(body)).toBe(true)
    } else {
      // Paginated response
      expect(body).toHaveProperty('jobs')
      expect(Array.isArray(body.jobs)).toBe(true)
    }
  })

  // API-08: Content-Type headers are correct
  test('API-08: should return correct Content-Type header', async ({ request, baseURL }) => {
    const response = await request.get(`${baseURL}/api/v1/targets`)

    const contentType = response.headers()['content-type']
    expect(contentType).toContain('application/json')
  })

  // API-09: Invalid JSON returns appropriate error
  test('API-09: should handle invalid request body gracefully', async ({ request, baseURL }) => {
    const response = await request.post(`${baseURL}/api/v1/targets`, {
      headers: {
        'Content-Type': 'application/json',
      },
      data: 'not valid json{{{',
    })

    // Should return 400 Bad Request
    expect(response.status()).toBeGreaterThanOrEqual(400)
    expect(response.status()).toBeLessThan(500)
  })

  // API-10: Non-existent endpoint returns 404
  test('API-10: should return 404 for non-existent endpoint', async ({ request, baseURL }) => {
    const response = await request.get(`${baseURL}/api/v1/nonexistent`)

    expect(response.status()).toBe(404)
  })
})
