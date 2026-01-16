import { Page, Locator, expect } from '@playwright/test'

export class SettingsPage {
  readonly page: Page
  readonly heading: Locator
  readonly targetsList: Locator
  readonly addTargetButton: Locator
  readonly targetForm: Locator
  readonly nameInput: Locator
  readonly typeSelect: Locator
  readonly urlInput: Locator
  readonly isActiveCheckbox: Locator
  readonly submitButton: Locator
  readonly cancelButton: Locator

  constructor(page: Page) {
    this.page = page
    this.heading = page.getByRole('heading', { name: /settings/i })
    this.targetsList = page.locator('[data-testid="targets-list"]')
    this.addTargetButton = page.getByRole('button', { name: /add target/i })
    this.targetForm = page.locator('[data-testid="target-form"]')
    this.nameInput = page.getByLabel(/name/i)
    this.typeSelect = page.getByLabel(/type/i)
    this.urlInput = page.getByLabel(/url|channel/i)
    this.isActiveCheckbox = page.getByLabel(/active/i)
    this.submitButton = page.getByRole('button', { name: /create|save|submit/i })
    this.cancelButton = page.getByRole('button', { name: /cancel/i })
  }

  async goto() {
    await this.page.goto('/settings')
    await expect(this.heading).toBeVisible()
  }

  async openAddTargetForm() {
    await this.addTargetButton.click()
    await expect(this.targetForm).toBeVisible()
  }

  async fillTargetForm(data: { name: string; type: string; url: string; isActive?: boolean }) {
    await this.nameInput.fill(data.name)
    if (data.type) {
      await this.typeSelect.selectOption(data.type)
    }
    await this.urlInput.fill(data.url)
    if (data.isActive !== undefined) {
      const isChecked = await this.isActiveCheckbox.isChecked()
      if (isChecked !== data.isActive) {
        await this.isActiveCheckbox.click()
      }
    }
  }

  async submitForm() {
    await this.submitButton.click()
  }

  getTargetCard(name: string): Locator {
    return this.page.locator(`[data-testid="target-card"]:has-text("${name}")`)
  }

  getDeleteButton(targetName: string): Locator {
    return this.getTargetCard(targetName).getByRole('button', { name: /delete/i })
  }

  getEditButton(targetName: string): Locator {
    return this.getTargetCard(targetName).getByRole('button', { name: /edit/i })
  }
}
