import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';
import vm from 'node:vm';

const appSource = await readFile(new URL('../web/admin/assets/app.js', import.meta.url), 'utf8');
const templateSource = await readFile(new URL('../web/admin/index.template.html', import.meta.url), 'utf8');
const stylesSource = await readFile(new URL('../web/admin/assets/styles.css', import.meta.url), 'utf8');
const i18nSource = await readFile(new URL('../web/admin/assets/i18n.js', import.meta.url), 'utf8');

const response = (data, status = 200) => ({
  ok: status >= 200 && status < 300,
  status,
  statusText: status === 200 ? 'OK' : 'Error',
  headers: { get: () => null },
  blob: async () => new Blob([JSON.stringify(data)]),
  text: async () => JSON.stringify(data),
});

const deferred = () => {
  let resolve;
  const promise = new Promise((next) => { resolve = next; });
  return { promise, resolve };
};

function createStorage(initial = {}) {
  const values = new Map(Object.entries(initial));
  return {
    getItem: (key) => values.get(key) ?? null,
    setItem: (key, value) => values.set(key, String(value)),
    removeItem: (key) => values.delete(key),
  };
}

function createHarness({ hash = '', fetchImpl = async () => response({}) } = {}) {
  const mounted = [];
  const unmounted = [];
  const listeners = new Map();
  const downloads = [];
  const location = { hash, pathname: '/admin/', search: '', origin: 'http://localhost:18787', protocol: 'http:' };
  const updateLocation = (url) => {
    const parsed = new URL(url, location.origin);
    location.hash = parsed.hash;
    location.pathname = parsed.pathname;
    location.search = parsed.search;
  };
  const localStorage = createStorage({ 'gateway.lang': 'en' });
  const sessionStorage = createStorage({ 'gateway.adminToken': 'test-admin-token' });
  const messages = new Proxy({}, { get: (_, key) => String(key) });
  let exposed;
  const window = {
    gatewayMessages: { en: messages, zh: messages },
    gatewayRender() {},
    location,
    history: {
      pushState: (_state, _title, url) => updateLocation(url),
      replaceState: (_state, _title, url) => updateLocation(url),
    },
    addEventListener: (name, listener) => listeners.set(name, listener),
    removeEventListener: (name) => listeners.delete(name),
    clearTimeout,
    setTimeout,
    prompt: () => null,
  };
  const Vue = {
    createApp: (options) => ({
      mount: () => {
        exposed = options.setup();
        return exposed;
      },
    }),
    computed: (getter) => ({ get value() { return getter(); } }),
    onMounted: (callback) => mounted.push(callback),
    onUnmounted: (callback) => unmounted.push(callback),
    reactive: (value) => value,
    ref: (value) => ({ value }),
  };
  const context = vm.createContext({
    AbortController,
    Blob,
    URL,
    URLSearchParams,
    Vue,
    clearTimeout,
    confirm: () => true,
    console,
    document: {
      documentElement: { dataset: {}, style: {} },
      createElement: () => {
        const anchor = { href: '', download: '', click: () => downloads.push({ href: anchor.href, download: anchor.download }) };
        return anchor;
      },
    },
    fetch: fetchImpl,
    history: window.history,
    Intl,
    localStorage,
    sessionStorage,
    setTimeout,
    window,
  });
  vm.runInContext(appSource, context, { filename: 'web/admin/assets/app.js' });
  return { downloads, exposed, location, listeners, mounted, unmounted };
}

test('opening the logs hash loads the paginated log endpoint', async () => {
  const requests = [];
  const harness = createHarness({
    hash: '#logs',
    fetchImpl: async (url) => {
      requests.push(url);
      if (url.startsWith('/admin/api/logs?')) {
        return response({ items: [], total: 0, limit: 50, offset: 0 });
      }
      return response({ service: { timezone: 'Asia/Singapore' } });
    },
  });
  await harness.mounted[0]();
  assert.equal(harness.exposed.activeSection.value, 'logs');
  assert.ok(requests.some((url) => url === '/admin/api/logs?limit=50&offset=0'));
});

test('only the latest log response updates the page', async () => {
  const first = deferred();
  const second = deferred();
  let requestCount = 0;
  const harness = createHarness({
    fetchImpl: (url) => {
      assert.match(url, /^\/admin\/api\/logs\?/);
      requestCount += 1;
      return requestCount === 1 ? first.promise : second.promise;
    },
  });

  harness.exposed.logFilters.q = 'first';
  const firstLoad = harness.exposed.loadLogs(0);
  harness.exposed.logFilters.q = 'second';
  const secondLoad = harness.exposed.loadLogs(50);
  second.resolve(response({ items: [{ id: 'new' }], total: 51, limit: 50, offset: 50 }));
  await secondLoad;
  first.resolve(response({ items: [{ id: 'old' }], total: 1, limit: 50, offset: 0 }));
  await firstLoad;

  assert.equal(harness.exposed.logs.value.length, 1);
  assert.equal(harness.exposed.logs.value[0].id, 'new');
  assert.equal(harness.exposed.logPage.offset, 50);
  assert.equal(harness.exposed.logsLoading.value, false);
});

test('timestamps use the storage timezone returned by the backend', () => {
  const harness = createHarness();
  harness.exposed.dashboard.value = { service: { timezone: 'America/New_York' } };
  assert.equal(
    harness.exposed.formatGatewayTime('2026-01-15T12:00:00Z'),
    '2026-01-15 07:00:00 America/New_York',
  );
});

test('section navigation updates the URL hash', () => {
  const harness = createHarness({ hash: '#setup' });
  harness.exposed.setActiveSection('providers');
  assert.equal(harness.location.hash, '#providers');
  assert.equal(harness.exposed.activeSection.value, 'providers');
});

test('provider form saves the provider and its first key before refreshing', async () => {
  const requests = [];
  const provider = {
    id: 'acme', name: 'Acme', type: 'openai-compatible', baseUrl: 'https://api.example/v1',
    balancePath: '', priority: 2, enabled: true, modelMap: {},
  };
  const harness = createHarness({
    fetchImpl: async (url, options = {}) => {
      requests.push({ url, options });
      if (url === '/admin/api/providers') return response(provider);
      if (url === '/admin/api/keys') return response({ id: 'acme-primary' });
      if (url === '/admin/api/dashboard') return response({ providers: [provider], keys: [] });
      return response({});
    },
  });
  harness.exposed.openProviderForm();
  Object.assign(harness.exposed.providerForm, {
    id: 'acme', name: 'Acme', type: 'openai-compatible', baseUrl: 'https://api.example/v1',
    priority: 2, firstKeyName: 'Primary', firstKeySecret: 'sk-acme', firstKeyPriority: 3,
  });
  await harness.exposed.saveProvider();

  const providerRequest = requests.find((item) => item.url === '/admin/api/providers');
  const keyRequest = requests.find((item) => item.url === '/admin/api/keys');
  assert.equal(providerRequest.options.method, 'POST');
  assert.equal(JSON.parse(providerRequest.options.body).name, 'Acme');
  assert.equal(keyRequest.options.method, 'POST');
  assert.deepEqual(JSON.parse(keyRequest.options.body), {
    providerId: 'acme', name: 'Primary', secret: 'sk-acme', priority: 3, enabled: true,
  });
  assert.equal(harness.exposed.providerFormOpen.value, false);
});

test('key edit and routing save send the current form state', async () => {
  const requests = [];
  const provider = {
    id: 'acme', name: 'Acme', type: 'openai-compatible', baseUrl: 'https://api.example/v1',
    balancePath: '', priority: 0, enabled: true, modelMap: {},
  };
  const harness = createHarness({
    fetchImpl: async (url, options = {}) => {
      requests.push({ url, options });
      if (url === '/admin/api/routing') return response({ routing: { retryPerRequest: 4, fallbackOnBusy: false }, restartRequired: true });
      if (url === '/admin/api/dashboard') return response({ providers: [provider], keys: [] });
      return response({});
    },
  });
  harness.exposed.providers.value = [provider];
  harness.exposed.editKey({ id: 'key-1', providerId: 'acme', name: 'Primary', priority: 1, enabled: true, providerModelMap: {} });
  harness.exposed.keyForm.name = 'Primary updated';
  harness.exposed.keyForm.priority = 5;
  await harness.exposed.saveKey();
  harness.exposed.routing.value = { retryPerRequest: 4, fallbackOnBusy: false };
  await harness.exposed.saveRouting();

  const keyRequest = requests.find((item) => item.url === '/admin/api/keys/key-1');
  const routingRequest = requests.find((item) => item.url === '/admin/api/routing');
  assert.equal(keyRequest.options.method, 'PATCH');
  assert.equal(JSON.parse(keyRequest.options.body).name, 'Primary updated');
  assert.equal(JSON.parse(keyRequest.options.body).priority, 5);
  assert.equal(routingRequest.options.method, 'PATCH');
  assert.deepEqual(JSON.parse(routingRequest.options.body), { retryPerRequest: 4, fallbackOnBusy: false });
  assert.equal(harness.exposed.routing.value.retryPerRequest, 4);
  assert.equal(harness.exposed.routingMessage.value, 'routing.restartRequired');
});

test('dashboard refresh restores persisted model inventories and routes', async () => {
  const route = {
    id: 'coding-auto', name: 'Coding', enabled: true,
    models: [{
      name: 'primary', priority: 100, enabled: true,
      targets: [{ providerId: 'acme', upstreamModel: 'gpt-primary', enabled: true }],
    }],
  };
  const harness = createHarness({
    fetchImpl: async (url) => {
      assert.equal(url, '/admin/api/dashboard');
      return response({
        providers: [{ id: 'acme', name: 'Acme', enabled: true, modelMap: {} }],
        keys: [],
        providerModels: { acme: ['gpt-primary', 'gpt-fallback'] },
        modelDiscovery: { acme: { providerId: 'acme', status: 'ok', modelCount: 2 } },
        modelRoutes: [route],
        modelStates: [
          { providerId: 'acme', keyId: '', modelId: 'gpt-primary', scope: 'provider', failureCount: 1, consecutiveFailures: 1, cooldownUntil: '2999-01-01T00:00:00Z' },
          { providerId: 'acme', keyId: 'key-a', modelId: 'gpt-primary', scope: 'key', failureCount: 1, consecutiveFailures: 1, cooldownUntil: '2999-01-01T00:00:00Z' },
          { providerId: 'acme', keyId: 'key-b', modelId: 'gpt-primary', scope: 'key', failureCount: 1, consecutiveFailures: 1, cooldownUntil: '2000-01-01T00:00:00Z' },
        ],
      });
    },
  });

  await harness.mounted[0]();

  assert.deepEqual([...harness.exposed.providerDiscoveredModels('acme')], ['gpt-primary', 'gpt-fallback']);
  assert.equal(harness.exposed.discoveryForProvider('acme').status, 'ok');
  assert.equal(harness.exposed.modelRoutes.value[0].id, 'coding-auto');
  assert.equal(harness.exposed.modelCoolingCount('acme', 'gpt-primary'), 1);
  assert.equal(harness.exposed.modelProviderCooling('acme', 'gpt-primary'), true);
  assert.equal(harness.exposed.modelHasFailureState('acme', 'gpt-primary'), true);
  assert.match(templateSource, /modelProviderCooling\(target\.providerId, target\.upstreamModel\)/);
});

test('model health reset targets all health state for a provider model', async () => {
  const requests = [];
  const harness = createHarness({
    fetchImpl: async (url, options = {}) => {
      requests.push({ url, options });
      if (url === '/admin/api/model-states/reset') return response({ ok: true });
      if (url === '/admin/api/dashboard') return response({ modelStates: [] });
      return response({});
    },
  });
  await harness.mounted[0]();

  await harness.exposed.resetModelState('acme', 'gpt-primary');

  const request = requests.find((item) => item.url === '/admin/api/model-states/reset');
  assert.equal(request.options.method, 'POST');
  assert.deepEqual(JSON.parse(request.options.body), { providerId: 'acme', modelId: 'gpt-primary' });
});

test('model route form sends nested model and provider priorities', async () => {
  const requests = [];
  const provider = { id: 'acme', name: 'Acme', enabled: true, modelMap: {} };
  const harness = createHarness({
    fetchImpl: async (url, options = {}) => {
      requests.push({ url, options });
      if (url === '/admin/api/model-routes') return response({ id: 'coding-auto' });
      if (url === '/admin/api/dashboard') {
        return response({ providers: [provider], keys: [], providerModels: { acme: ['gpt-primary', 'gpt-fallback'] } });
      }
      return response({});
    },
  });
  await harness.mounted[0]();
  harness.exposed.openModelRouteForm();
  Object.assign(harness.exposed.modelRouteForm, {
    id: 'coding-auto',
    name: 'Coding',
    enabled: true,
    models: [
      {
        name: 'gpt-primary', priority: 100, enabled: true,
        targets: [{ providerId: 'acme', upstreamModel: 'gpt-primary', enabled: true }],
      },
      {
        name: 'gpt-fallback', priority: 10, enabled: true,
        targets: [{ providerId: 'acme', upstreamModel: 'gpt-fallback', enabled: true }],
      },
    ],
  });

  await harness.exposed.saveModelRoute();

  const request = requests.find((item) => item.url === '/admin/api/model-routes');
  assert.equal(request.options.method, 'POST');
  assert.deepEqual(JSON.parse(request.options.body), {
    id: 'coding-auto',
    name: 'Coding',
    enabled: true,
    models: [
      {
        name: 'gpt-primary', priority: 100, enabled: true,
        targets: [{ providerId: 'acme', upstreamModel: 'gpt-primary', enabled: true }],
      },
      {
        name: 'gpt-fallback', priority: 10, enabled: true,
        targets: [{ providerId: 'acme', upstreamModel: 'gpt-fallback', enabled: true }],
      },
    ],
  });
  assert.equal(harness.exposed.modelRouteFormOpen.value, false);
});

test('adding a fallback model never creates a negative priority', () => {
  const harness = createHarness();
  harness.exposed.modelRouteForm.models = [{ priority: 0 }];

  harness.exposed.addRouteModel();

  assert.equal(harness.exposed.modelRouteForm.models[1].priority, 0);
});

test('adding a route target selects an unused provider in the same model tier', async () => {
  const harness = createHarness({
    fetchImpl: async () => response({
      providers: [
        { id: 'first', name: 'First', enabled: true, modelMap: {} },
        { id: 'second', name: 'Second', enabled: true, modelMap: {} },
      ],
      providerModels: { first: ['model-a'], second: ['model-b'] },
    }),
  });
  await harness.mounted[0]();
  harness.exposed.openModelRouteForm();

  harness.exposed.addRouteTarget(0);

  assert.equal(harness.exposed.modelRouteForm.models[0].targets[0].providerId, 'first');
  assert.equal(harness.exposed.modelRouteForm.models[0].targets[1].providerId, 'second');
  assert.equal(harness.exposed.routeProviderSelected(0, 1, 'first'), true);
});

test('provider model refresh calls the discovery-only endpoint', async () => {
  const requests = [];
  const provider = { id: 'acme', name: 'Acme', enabled: true, modelMap: {} };
  const harness = createHarness({
    fetchImpl: async (url, options = {}) => {
      requests.push({ url, options });
      if (url === '/admin/api/providers/acme/models') {
        return response({ providerId: 'acme', status: 'ok', models: ['gpt-primary'], modelCount: 1 });
      }
      if (url === '/admin/api/dashboard') {
        return response({ providers: [provider], keys: [{ id: 'key', providerId: 'acme' }], providerModels: { acme: ['gpt-primary'] } });
      }
      return response({});
    },
  });
  await harness.mounted[0]();

  await harness.exposed.refreshProviderModels(provider);

  const request = requests.find((item) => item.url === '/admin/api/providers/acme/models');
  assert.equal(request.options.method, 'POST');
  assert.equal(request.options.body, '{}');
  assert.deepEqual([...harness.exposed.providerDiscoveredModels('acme')], ['gpt-primary']);
  assert.equal(harness.exposed.refreshingProviderId.value, '');
});

test('model recommendation recognizes configured and leading model families while excluding utility models', () => {
  const harness = createHarness();
  for (const model of [
    'newapi/step-3.7-flash',
    'newapi/kat-coder-pro-V2',
    'moonshotai/kimi-k2.6',
    'deepseek-ai/deepseek-v4-flash',
    'deepseek-ai/deepseek-v4-pro',
    'minimaxai/minimax-m3',
    'z-ai/glm-5.2',
    'grok-4.5',
    'openai/gpt-5.2',
    'openai/gpt-5.5',
    'openai/gpt-5.5-openai-compact',
    'openai/gpt-5.6',
    'openai/gpt-5.6-luna',
    'openai/gpt-5.6-sol',
    'openai/gpt-5.6-sol-openai-compact',
    'openai/gpt-5.6-terra',
    'openai/gpt-5.6-terra-openai-compact',
    'openai/gpt-image-2',
    'anthropic/claude-opus-4',
  ]) {
    assert.equal(harness.exposed.isFlagshipModel(model), true, model);
  }
  assert.equal(harness.exposed.isFlagshipModel('text-embedding-3-large'), false);
  assert.equal(harness.exposed.isFlagshipModel('vendor/legacy-chat-small'), false);
});

test('recommendation labels appear in upstream key model selects', () => {
  const harness = createHarness();
  assert.match(templateSource, /<details class="model-policy-picker">/);
  const keyOptions = templateSource.match(/<option v-for="model in providerModelOptions\(keyForm\.providerId\)"[^>]*>\{\{ modelOptionLabel\(model\) \}\}<\/option>/g) || [];
  assert.equal(keyOptions.length, 2);
  assert.doesNotMatch(templateSource, /class="flagship-tag"/);
  assert.equal(harness.exposed.modelOptionLabel('openai/gpt-5.2'), 'openai/gpt-5.2 · model.recommended');
  assert.equal(harness.exposed.modelOptionLabel('vendor/legacy-chat-small'), 'vendor/legacy-chat-small');
  assert.match(stylesSource, /\.model-policy-picker:not\(\[open\]\)\s*>\s*\.model-policy-picker-panel\s*\{\s*display:\s*none;/);
});

test('provider model routing and policy controls live inside the edit-key form', () => {
  const editFormStart = templateSource.indexOf('<form v-if="keyFormOpen && keyForm.editingId === k.id"');
  const modelManagement = templateSource.indexOf('<section class="model-inventory key-model-management">');
  const editFormEnd = templateSource.indexOf('</form>', editFormStart);
  assert.ok(editFormStart >= 0);
  assert.ok(modelManagement > editFormStart && modelManagement < editFormEnd);
  assert.equal(templateSource.match(/class="model-inventory key-model-management"/g)?.length, 1);
  assert.match(stylesSource, /\.key-model-management\s*\{[^}]*grid-column:\s*1\s*\/\s*-1;/s);
  assert.match(i18nSource, /'model\.inventory': '模型路由'/);
  assert.match(i18nSource, /'model\.inventory': 'Model routing'/);
});

test('model routing picker marks recommendations while preserving raw checkbox values', () => {
  const harness = createHarness();
  assert.match(templateSource, /<code :class="\{ 'flagship-option': isFlagshipModel\(model\) \}">\{\{ modelOptionLabel\(model\) \}\}<\/code>/);
  assert.match(templateSource, /setProviderModelSelected\(p\.id, model, \$event\.target\.checked\)/);
  assert.match(stylesSource, /\.model-policy-item code\.flagship-option\s*\{[^}]*color:\s*var\(--amber\);[^}]*font-weight:\s*700;/s);
  assert.equal(harness.exposed.modelOptionLabel('newapi/step-3.7-flash'), 'newapi/step-3.7-flash · model.recommended');
  assert.equal(harness.exposed.modelOptionLabel('legacy-chat-small'), 'legacy-chat-small');
});

test('model routing search cannot submit the surrounding key form', () => {
  assert.match(
    templateSource,
    /type="search"[^>]*@input="modelPolicySearch\[p\.id\] = \$event\.target\.value"[^>]*@keydown\.enter\.prevent/,
  );
});

test('flagship quick selection persists a provider model allowlist', async () => {
  const requests = [];
  let provider = {
    id: 'acme', name: 'Acme', type: 'openai-compatible', baseUrl: 'https://api.example/v1',
    enabled: true, modelMap: {}, modelAllowlistEnabled: false, modelAllowlist: [],
  };
  const models = ['moonshotai/kimi-k2.6', 'deepseek-ai/deepseek-v4-flash', 'text-embedding-3-large', 'legacy-chat-small'];
  const harness = createHarness({
    fetchImpl: async (url, options = {}) => {
      requests.push({ url, options });
      if (url === '/admin/api/providers/acme') {
        const patch = JSON.parse(options.body);
        provider = { ...provider, ...patch };
        return response(provider);
      }
      if (url === '/admin/api/dashboard') {
        return response({ providers: [provider], keys: [], providerModels: { acme: models } });
      }
      return response({});
    },
  });
  await harness.mounted[0]();

  assert.deepEqual([...harness.exposed.modelPolicyDraft('acme').models].sort(), [...models].sort());
  assert.equal(harness.exposed.modelPolicyDraft('acme').enabled, false);

  harness.exposed.selectProviderModels('acme', 'flagship');
  assert.equal(harness.exposed.modelPolicyDraft('acme').enabled, true);
  assert.deepEqual(
    [...harness.exposed.modelPolicyDraft('acme').models].sort(),
    ['deepseek-ai/deepseek-v4-flash', 'moonshotai/kimi-k2.6'].sort(),
  );
  await harness.exposed.saveProviderModelPolicy(provider);

  const request = requests.find((item) => item.url === '/admin/api/providers/acme');
  assert.equal(request.options.method, 'PATCH');
  assert.deepEqual(JSON.parse(request.options.body), {
    modelAllowlistEnabled: true,
    modelAllowlist: ['deepseek-ai/deepseek-v4-flash', 'moonshotai/kimi-k2.6'],
  });
  assert.equal(harness.exposed.modelPolicyDraft('acme').dirty, false);
  assert.equal(harness.exposed.modelPolicyDraft('acme').enabled, true);
});

test('database and portable backup actions download files and clear passphrases', async () => {
  const requests = [];
  const harness = createHarness({
    fetchImpl: async (url, options = {}) => {
      requests.push({ url, options });
      return response({ backup: true });
    },
  });
  await harness.exposed.downloadBackup();
  harness.exposed.backupPassphrase.value = 'correct horse battery staple';
  harness.exposed.backupPassphraseConfirm.value = 'correct horse battery staple';
  await harness.exposed.downloadPortableBackup();

  const portableRequest = requests.find((item) => item.url === '/admin/api/maintenance/portable-backup');
  assert.equal(requests.find((item) => item.url === '/admin/api/maintenance/backup').options.method, 'POST');
  assert.equal(portableRequest.options.method, 'POST');
  assert.deepEqual(JSON.parse(portableRequest.options.body), { passphrase: 'correct horse battery staple' });
  assert.deepEqual(harness.downloads.map((item) => item.download), ['gateway-backup.db', 'gateway-portable-backup.zip']);
  assert.equal(harness.exposed.backupPassphrase.value, '');
  assert.equal(harness.exposed.backupPassphraseConfirm.value, '');
});
