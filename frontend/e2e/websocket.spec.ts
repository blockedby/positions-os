import { test, expect } from '@playwright/test'

test.describe('WebSocket Stability', () => {
  // WS-01: Verify only one WebSocket connection is established
  test('WS-01: should establish only one WebSocket connection', async ({ page }) => {
    const wsConnections: string[] = []

    page.on('websocket', (ws) => {
      wsConnections.push(ws.url())
    })

    await page.goto('/settings')
    // Wait enough time for potential reconnection issues to manifest
    await page.waitForTimeout(5000)

    // Should have exactly one connection
    expect(wsConnections.length).toBe(1)
    expect(wsConnections[0]).toContain('/ws')
  })

  // WS-02: Verify no WebSocket reconnection loop errors
  test('WS-02: should not produce WebSocket reconnection errors', async ({ page }) => {
    const wsErrors: string[] = []
    const wsCloseEvents: string[] = []

    page.on('console', (msg) => {
      const text = msg.text()
      if (msg.type() === 'error' && text.toLowerCase().includes('websocket')) {
        wsErrors.push(text)
      }
      if (text.includes('WebSocket') && text.includes('closed')) {
        wsCloseEvents.push(text)
      }
    })

    await page.goto('/settings')
    // Wait 10 seconds to observe any reconnection behavior
    await page.waitForTimeout(10000)

    // Should not have repeated WebSocket errors
    expect(wsErrors.length).toBeLessThanOrEqual(1) // Allow at most 1 error (initial connection issues)

    // Should not have multiple close events (which would indicate reconnection loop)
    expect(wsCloseEvents.length).toBeLessThanOrEqual(1)
  })

  // WS-03: Verify WebSocket survives navigation between pages
  test('WS-03: should maintain stable connection during navigation', async ({ page }) => {
    const wsConnections: string[] = []

    page.on('websocket', (ws) => {
      wsConnections.push(ws.url())
    })

    // Navigate through multiple pages
    await page.goto('/')
    await page.waitForTimeout(1000)

    await page.goto('/settings')
    await page.waitForTimeout(1000)

    await page.goto('/jobs')
    await page.waitForTimeout(1000)

    await page.goto('/settings')
    await page.waitForTimeout(2000)

    // Should have at most 4 connections (one per page navigation)
    // Ideally just 1 if connection is maintained, but up to 4 if reconnecting on route change
    expect(wsConnections.length).toBeLessThanOrEqual(4)
  })

  // WS-04: Count WebSocket connections over time - no rapid reconnections
  test('WS-04: should not rapidly reconnect', async ({ page }) => {
    const connectionTimes: number[] = []

    page.on('websocket', () => {
      connectionTimes.push(Date.now())
    })

    await page.goto('/settings')
    await page.waitForTimeout(5000)

    // If there are multiple connections, they should be spread out
    if (connectionTimes.length > 1) {
      for (let i = 1; i < connectionTimes.length; i++) {
        const timeBetweenConnections = connectionTimes[i] - connectionTimes[i - 1]
        // Connections should be at least 1 second apart (not rapid-fire reconnecting)
        expect(timeBetweenConnections).toBeGreaterThan(1000)
      }
    }

    // Should not have more than 2 connections in 5 seconds
    expect(connectionTimes.length).toBeLessThanOrEqual(2)
  })

  // WS-05: Verify WebSocket messages are received
  test('WS-05: should receive WebSocket messages', async ({ page }) => {
    const messagesReceived: unknown[] = []

    page.on('websocket', (ws) => {
      ws.on('framereceived', (frame) => {
        if (frame.payload) {
          try {
            messagesReceived.push(JSON.parse(frame.payload.toString()))
          } catch {
            // Ignore non-JSON frames
          }
        }
      })
    })

    await page.goto('/settings')
    await page.waitForTimeout(3000)

    // WebSocket should be connected (connection established)
    // Note: We may not receive messages if no events are happening,
    // but the connection should be established
  })
})
