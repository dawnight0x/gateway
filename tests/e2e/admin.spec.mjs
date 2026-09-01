import { expect, test } from '@playwright/test';

test.beforeEach(async ({ page }) => {
  await page.addInitScript(() => localStorage.setItem('gateway.lang', 'en'));
});

test('admin setup and destructive dialog are keyboard accessible', async ({ page }) => {
  await page.goto('/admin/#token=e2e-admin-token');
  await expect(page).toHaveURL(/#dashboard$/);
  await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible();

  await page.getByRole('link', { name: 'Upstream Connections' }).click();
  await page.getByRole('button', { name: 'Add Upstream', exact: true }).first().click();
  const createForm = page.getByTestId('provider-form');
  await createForm.getByLabel(/^Upstream ID/).fill('e2e-provider');
  await createForm.getByLabel(/^Display name/).fill('E2E Provider');
  await createForm.getByLabel(/^API root URL/).fill('https://example.invalid/v1');
  await createForm.getByLabel(/^Key display name/).fill('E2E Key');
  await createForm.getByLabel(/^Real upstream key/).fill('sk-e2e-secret');
  await createForm.getByRole('button', { name: 'Save Upstream' }).click();

  const provider = page.locator('.provider-card').filter({ hasText: 'E2E Provider' });
  await expect(provider).toBeVisible();
  const providerActions = provider.locator('.provider-actions');
  await providerActions.getByRole('button', { name: 'Edit Upstream' }).click();
  await expect(provider.getByLabel(/^Display name/)).toHaveValue('E2E Provider');
  await provider.getByRole('button', { name: 'Cancel' }).click();

  const deleteButton = providerActions.getByRole('button', { name: 'Delete' });
  await deleteButton.focus();
  await deleteButton.click();
  const dialog = page.getByRole('dialog');
  await expect(dialog).toBeVisible();
  await expect(dialog).toHaveAttribute('aria-labelledby', 'confirm-dialog-title');
  const confirmation = dialog.getByRole('textbox');
  await expect(confirmation).toBeFocused();
  await page.keyboard.press('Shift+Tab');
  await expect(dialog.getByRole('button', { name: 'Cancel' })).toBeFocused();
  await page.keyboard.press('Escape');
  await expect(dialog).toBeHidden();
  await expect(deleteButton).toBeFocused();
});
