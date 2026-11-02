import { expect, test } from '@playwright/test';
import { readFile } from 'node:fs/promises';

test.beforeEach(async ({ page }) => {
  await page.addInitScript(() => localStorage.setItem('gateway.lang', 'en'));
});

test('routing changes persist after reload and still require restart', async ({ page }) => {
  await page.goto('/admin/#token=e2e-admin-token');
  await page.getByRole('link', { name: 'Routing', exact: true }).click();
  const timeout = page.getByLabel('Request timeout (seconds)', { exact: true });
  await timeout.fill('123');
  await page.getByRole('button', { name: 'Show advanced configuration' }).click();
  await page.getByLabel('Stream idle timeout (seconds)', { exact: true }).fill('246');
  await page.getByRole('button', { name: 'Save settings', exact: true }).click();
  await expect(page.getByRole('status')).toContainText('Restart the gateway to apply them.');
  await page.reload();
  await expect(timeout).toHaveValue('123');
  await page.getByRole('button', { name: 'Show advanced configuration' }).click();
  await expect(page.getByLabel('Stream idle timeout (seconds)', { exact: true })).toHaveValue('246');
  await expect(page.getByLabel('Allow ambiguous retries that may duplicate billing')).not.toBeChecked();
});

test('maintenance checks integrity and exports encrypted portable backup', async ({ page }, testInfo) => {
  const fixture = await page.request.post('/admin/api/providers', {
    headers: { 'x-admin-token': 'e2e-admin-token' },
    data: {
      id: 'e2e-backup-fixture', name: 'Backup fixture', type: 'openai-compatible',
      baseUrl: 'https://example.invalid/v1', enabled: false,
      firstKey: { name: 'Backup key', secret: 'sk-e2e-secret' },
    },
  });
  expect(fixture.ok()).toBe(true);
  await page.goto('/admin/#token=e2e-admin-token');
  await page.getByRole('link', { name: 'Routing', exact: true }).click();
  await page.getByRole('button', { name: 'Check database', exact: true }).click();
  await expect(page.getByRole('status')).toContainText('Database integrity check passed.');
  const passphrase = page.getByLabel('Backup passphrase (12+ characters)', { exact: true });
  const confirmation = page.getByLabel('Confirm backup passphrase', { exact: true });
  await passphrase.fill('synthetic-e2e-passphrase');
  await confirmation.fill('different-e2e-passphrase');
  await page.getByRole('button', { name: 'Create encrypted portable backup' }).click();
  await expect(page.getByRole('alert')).toContainText('The backup passphrases do not match.');
  await confirmation.fill('synthetic-e2e-passphrase');
  const downloadPromise = page.waitForEvent('download');
  await page.getByRole('button', { name: 'Create encrypted portable backup' }).click();
  const download = await downloadPromise;
  expect(download.suggestedFilename()).toMatch(/^gateway-portable-.*\.zip$/);
  const backupPath = testInfo.outputPath('portable-backup.zip');
  await download.saveAs(backupPath);
  const bytes = await readFile(backupPath);
  expect(bytes.subarray(0, 2).toString()).toBe('PK');
  expect(bytes.includes(Buffer.from('sk-e2e-secret'))).toBe(false);
  await expect(passphrase).toHaveValue('');
  await expect(confirmation).toHaveValue('');
  await expect(page.getByRole('status')).toContainText('An encrypted portable backup was created and downloaded.');
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
