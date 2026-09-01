import { defineConfig } from '@playwright/test';
import { existsSync } from 'node:fs';
import { join, resolve } from 'node:path';

const defaultBinary = process.platform === 'win32' ? 'bin/gateway-e2e.exe' : 'bin/gateway-e2e';
const binary = resolve(process.env.GATEWAY_E2E_BINARY || defaultBinary);
if (!existsSync(binary)) {
  throw new Error(`E2E gateway binary not found: ${binary}. Build it in bin before running npm run test:e2e.`);
}

const dataDir = resolve('.tmp', `e2e-${process.pid}`);
const quote = (value) => `"${String(value).replaceAll('"', '\\"')}"`;

export default defineConfig({
  testDir: './tests/e2e',
  timeout: 45_000,
  expect: { timeout: 8_000 },
  fullyParallel: false,
  workers: 1,
  reporter: process.env.CI ? 'github' : 'list',
  use: {
    baseURL: 'http://127.0.0.1:8787',
    viewport: { width: 1440, height: 900 },
    colorScheme: 'light',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },
  webServer: {
    command: quote(binary),
    url: 'http://127.0.0.1:8787/health',
    reuseExistingServer: false,
    timeout: 60_000,
    env: {
      ...process.env,
      GATEWAY_PORT: '8787',
      GATEWAY_TRAY: 'false',
      GATEWAY_OPEN_BROWSER_ON_DUPLICATE: 'false',
      GATEWAY_ADMIN_TOKEN: 'e2e-admin-token',
      GATEWAY_DB: join(dataDir, 'gateway.db'),
      GATEWAY_SECRET_PATH: join(dataDir, 'secret.key'),
      GATEWAY_ADMIN_TOKEN_FILE: join(dataDir, 'admin.token'),
      GATEWAY_DRAIN_GRACE_SECONDS: '0',
      GATEWAY_SHUTDOWN_TIMEOUT_SECONDS: '5',
    },
  },
});
