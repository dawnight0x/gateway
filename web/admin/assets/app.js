const { createApp, computed, nextTick, onMounted, onUnmounted, reactive, ref } = Vue;

const messages = window.gatewayMessages;

const sectionIDs = ['dashboard', 'providers', 'model-routing', 'keys', 'routing', 'setup', 'logs'];

createApp({
  render: window.gatewayRender,
  setup() {
    const lang = ref(localStorage.getItem('gateway.lang') || 'zh');
    const theme = ref(localStorage.getItem('gateway.theme') || 'light');
    const sectionFromHash = () => {
      const raw = window.location.hash.replace(/^#/, '');
      if (raw.includes('=')) {
        const params = new URLSearchParams(raw);
        const section = params.get('section');
        return sectionIDs.includes(section) ? section : 'dashboard';
      }
      const id = raw;
      return sectionIDs.includes(id) ? id : 'dashboard';
    };
    const adminTokenFromHash = () => {
      const raw = window.location.hash.replace(/^#/, '');
      if (!raw.includes('=')) return '';
      const params = new URLSearchParams(raw);
      return (params.get('token') || params.get('admin_token') || '').trim();
    };
    const clearAdminTokenHash = () => {
      const raw = window.location.hash.replace(/^#/, '');
      if (!raw.includes('=')) return;
      const params = new URLSearchParams(raw);
      if (!params.has('token') && !params.has('admin_token')) return;
      const section = params.get('section');
      const cleanSection = sectionIDs.includes(section) ? section : 'dashboard';
      window.history.replaceState(null, '', `${window.location.pathname}${window.location.search}#${cleanSection}`);
    };
    const bootAdminToken = adminTokenFromHash();
    const activeSection = ref('dashboard');
    const dashboard = ref(null);
    const providers = ref([]);
    const keys = ref([]);
    const gatewayKeys = ref([]);
    const balances = ref([]);
    const balanceResults = ref([]);
    const balanceRefreshing = ref(false);
    const keyTestResults = ref({});
    const providerModels = ref({});
    const modelDiscovery = ref({});
    const modelRoutes = ref([]);
    const modelStates = ref([]);
    const refreshingProviderId = ref('');
    const modelPolicySavingProviderId = ref('');
    const modelPolicyDrafts = reactive({});
    const modelPolicySearch = reactive({});
    const testingKeyId = ref('');
    const testingAllKeys = ref(false);
    const bulkTestSummary = ref(null);
    const logs = ref([]);
    const logPage = reactive({ total: 0, limit: 50, offset: 0 });
    const logFilters = reactive({ q: '', status: '', providerId: '', keyId: '', model: '', errorType: '' });
    const logsLoading = ref(false);
    const logsAutoRefresh = ref(localStorage.getItem('gateway.logsAutoRefresh') !== 'false');
    const routing = ref({});
    const routingAdvanced = ref(false);
    const routingMessage = ref('');
    const maintenanceMessage = ref('');
    const backupPassphrase = ref('');
    const backupPassphraseConfirm = ref('');
    const snippets = ref({});
    const refreshing = ref(false);
    const actionBusy = ref(false);
    const refreshError = ref('');
    const formError = ref('');
    const legacyAdminToken = localStorage.getItem('gateway.adminToken') || '';
    localStorage.removeItem('gateway.adminToken');
    const adminToken = ref(bootAdminToken || sessionStorage.getItem('gateway.adminToken') || legacyAdminToken);
    const authRequired = ref(!adminToken.value);
    const authError = ref('');
    const authSubmitting = ref(false);
    const authForm = reactive({ token: '' });
    const notices = ref([]);
    const confirmDialog = reactive({ open: false, title: '', message: '', requiredId: '', typedId: '', destructive: false, resolve: null });
    const confirmDialogElement = ref(null);
    let confirmDialogReturnFocus = null;
    if (bootAdminToken) {
      sessionStorage.setItem('gateway.adminToken', bootAdminToken);
      clearAdminTokenHash();
    } else if (legacyAdminToken) {
      sessionStorage.setItem('gateway.adminToken', legacyAdminToken);
    }
    const providerFormOpen = ref(false);
    const keyFormOpen = ref(false);
    const modelRouteFormOpen = ref(false);
    const modelMapExample = '{\n  "gpt-4o": "gpt-4o-mini",\n  "claude-sonnet": "claude-3-5-sonnet-20241022"\n}';

    const providerForm = reactive({
      editingId: '',
      id: '',
      name: '',
      type: 'openai-compatible',
      baseUrl: '',
      balancePath: '',
      priority: 0,
      modelMap: '{}',
      defaultModel: '',
      firstKeyName: '',
      firstKeySecret: '',
      firstKeyPriority: 0,
    });

    const keyForm = reactive({
      editingId: '',
      providerId: '',
      name: '',
      secret: '',
      priority: 0,
      defaultModel: '',
      enabled: true,
    });
    const modelRouteForm = reactive({
      editingId: '',
      id: '',
      name: '',
      enabled: true,
      models: [],
    });
    const gatewayKeyForm = reactive({ name: '' });
    const createdGatewayKey = ref(null);
    const copyFeedback = ref({});
    const copyTimers = new Map();

    const navItems = [
      { id: 'dashboard', label: 'nav.dashboard', icon: '01' },
      { id: 'providers', label: 'nav.providers', icon: '02' },
      { id: 'model-routing', label: 'nav.modelRouting', icon: '03' },
      { id: 'keys', label: 'nav.keys', icon: '04' },
      { id: 'routing', label: 'nav.routing', icon: '05' },
      { id: 'setup', label: 'nav.setup', icon: '06' },
      { id: 'logs', label: 'nav.logs', icon: '07' },
    ];

    const providerTypes = [
      { value: 'openai-compatible', label: 'openai' },
      { value: 'anthropic-compatible', label: 'anthropic' },
      { value: 'gemini-compatible', label: 'gemini' },
      { value: 'new-api', label: 'newapi' },
      { value: 'sub2api', label: 'sub2api' },
      { value: 'custom', label: 'custom' },
    ];

    const providerTypeLabel = (type) => providerTypes.find((item) => item.value === type)?.label || type || 'custom';

    const t = (key, vars = {}) => {
      let text = messages[lang.value]?.[key] || messages.en[key] || key;
      Object.entries(vars).forEach(([name, value]) => { text = text.replace(`{${name}}`, value); });
      return text;
    };

    const localizeError = (error) => {
      const raw = String(error?.message || '').trim();
      const code = String(error?.code || '').trim();
      const knownCodes = {
        auth_required: 'error.authRequired',
        invalid_admin_token: 'error.authInvalid',
        provider_required: 'error.providerRequired',
        key_required: 'error.keyRequired',
      };
      if (knownCodes[code]) return t(knownCodes[code]);
      const matches = [
        [/invalid admin token|admin token is invalid|unauthorized/i, 'error.authInvalid'],
        [/provider .*required|provider name|base url/i, 'error.providerRequired'],
        [/key .*required|secret is required/i, 'error.keyRequired'],
        [/priority must be between/i, 'error.priorityRange'],
        [/failed to load|database|internal server/i, 'error.server'],
      ];
      const match = matches.find(([pattern]) => pattern.test(raw));
      return match ? t(match[1]) : (raw || t('error.action'));
    };

    const notify = (type, message) => {
      const id = `${Date.now()}-${Math.random()}`;
      notices.value = [...notices.value, { id, type, message }];
      window.setTimeout(() => {
        notices.value = notices.value.filter((item) => item.id !== id);
      }, 4200);
    };

    const openConfirmDialog = (message, title = t('confirm.title'), requiredId = '') => new Promise((resolve) => {
      confirmDialogReturnFocus = typeof document.activeElement?.focus === 'function' ? document.activeElement : null;
      Object.assign(confirmDialog, {
        open: true,
        title,
        message,
        requiredId,
        typedId: '',
        destructive: !!requiredId,
        resolve,
      });
      nextTick(() => {
        const target = confirmDialogElement.value?.querySelector('[autofocus], button:not([disabled]), input:not([disabled])');
        target?.focus();
      });
    });

    const finishConfirmDialog = (accepted) => {
      const resolve = confirmDialog.resolve;
      const matches = !confirmDialog.requiredId || confirmDialog.typedId === confirmDialog.requiredId;
      Object.assign(confirmDialog, { open: false, requiredId: '', typedId: '', resolve: null });
      if (resolve) resolve(!!accepted && matches);
      const returnFocus = confirmDialogReturnFocus;
      confirmDialogReturnFocus = null;
      nextTick(() => returnFocus?.focus());
    };

    const handleConfirmDialogKeydown = (event) => {
      if (event.key === 'Escape') {
        event.preventDefault();
        finishConfirmDialog(false);
        return;
      }
      if (event.key !== 'Tab') return;
      const focusable = Array.from(confirmDialogElement.value?.querySelectorAll(
        'button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [href], [tabindex]:not([tabindex="-1"])',
      ) || []).filter((element) => !element.hidden);
      if (!focusable.length) return;
      const first = focusable[0];
      const last = focusable[focusable.length - 1];
      if (event.shiftKey && document.activeElement === first) {
        event.preventDefault();
        last.focus();
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault();
        first.focus();
      }
    };

    const dismissNotice = (id) => {
      notices.value = notices.value.filter((item) => item.id !== id);
    };

    const api = async (path, options = {}, retryAuth = true) => {
      const headers = { 'content-type': 'application/json', ...(options.headers || {}) };
      if (adminToken.value) headers['x-admin-token'] = adminToken.value;
      const res = await fetch(`/admin/api/${path}`, {
        ...options,
        headers,
      });
      const text = await res.text();
      let data = null;
      if (text) {
        try {
          data = JSON.parse(text);
        } catch {
          data = { error: text };
        }
      }
      const errorValue = data?.error;
      const errorMessage = typeof errorValue === 'string'
        ? errorValue
        : errorValue?.message || res.statusText;
      if (res.status === 401 && retryAuth) {
        sessionStorage.removeItem('gateway.adminToken');
        adminToken.value = '';
        authRequired.value = true;
        authError.value = t('auth.invalid');
        const authErrorObject = new Error('auth_required');
        authErrorObject.code = 'auth_required';
        throw authErrorObject;
      }
      if (!res.ok) {
        const error = new Error(errorMessage);
        if (typeof errorValue === 'object') error.code = errorValue?.code || '';
        throw error;
      }
      return data;
    };

    const download = async (path, options = {}, fallbackName = 'download') => {
      const headers = { ...(options.headers || {}) };
      if (adminToken.value) headers['x-admin-token'] = adminToken.value;
      const res = await fetch(`/admin/api/${path}`, { ...options, headers });
      if (!res.ok) {
        const message = await res.text();
        if (res.status === 401) {
          sessionStorage.removeItem('gateway.adminToken');
          adminToken.value = '';
          authRequired.value = true;
          const error = new Error('auth_required');
          error.code = 'auth_required';
          throw error;
        }
        let parsed = null;
        try { parsed = message ? JSON.parse(message) : null; } catch { /* plain response */ }
        const error = new Error(parsed?.error?.message || parsed?.error || message || `${res.status} ${res.statusText}`);
        error.code = parsed?.error?.code || '';
        throw error;
      }
      const blob = await res.blob();
      const disposition = res.headers.get('content-disposition') || '';
      const match = disposition.match(/filename="?([^";]+)"?/i);
      const anchor = document.createElement('a');
      anchor.href = URL.createObjectURL(blob);
      anchor.download = match?.[1] || fallbackName;
      anchor.click();
      URL.revokeObjectURL(anchor.href);
    };

    const runAction = async (action, errorRef = refreshError, successMessage = '') => {
      if (actionBusy.value) return null;
      actionBusy.value = true;
      errorRef.value = '';
      try {
        const result = await action();
        if (successMessage) notify('success', successMessage);
        return result;
      } catch (error) {
        const message = localizeError(error);
        if (errorRef === refreshError) notify('error', message);
        errorRef.value = message;
        return null;
      } finally {
        actionBusy.value = false;
      }
    };

    let refreshPromise = null;
    const refresh = () => {
      if (refreshPromise) return refreshPromise;
      refreshing.value = true;
      refreshError.value = '';
      refreshPromise = (async () => {
        const nextDashboard = await api('dashboard');
        dashboard.value = {
          ...nextDashboard,
          providers: nextDashboard?.providers || [],
          keys: nextDashboard?.keys || [],
          stats: {
            ...(nextDashboard?.stats || {}),
            recent: nextDashboard?.stats?.recent || [],
          },
        };
        providers.value = dashboard.value.providers;
        keys.value = dashboard.value.keys;
        gatewayKeys.value = nextDashboard?.gatewayKeys || [];
        balances.value = nextDashboard?.balances || [];
        logs.value = nextDashboard?.logs || [];
        providerModels.value = nextDashboard?.providerModels || {};
        syncModelPolicyDrafts(providers.value);
        modelDiscovery.value = nextDashboard?.modelDiscovery || {};
        modelRoutes.value = nextDashboard?.modelRoutes || [];
        modelStates.value = nextDashboard?.modelStates || [];
        routing.value = nextDashboard?.routing || {};
        snippets.value = nextDashboard?.snippets || {};
        if (!keyForm.providerId && providers.value.length) keyForm.providerId = providers.value[0].id;
      })().then(() => {
        authRequired.value = false;
        return true;
      }).catch((error) => {
        if (error?.code !== 'auth_required') {
          refreshError.value = localizeError(error) || t('error.refresh');
          notify('error', refreshError.value);
        }
        return false;
      }).finally(() => {
        refreshing.value = false;
        refreshPromise = null;
      });
      return refreshPromise;
    };

    const logQuery = (extra = {}) => {
      const params = new URLSearchParams({ limit: String(logPage.limit), offset: String(logPage.offset) });
      Object.entries(logFilters).forEach(([name, value]) => {
        if (String(value || '').trim()) params.set(name === 'providerId' || name === 'keyId' || name === 'errorType' ? name : name, String(value).trim());
      });
      Object.entries(extra).forEach(([name, value]) => params.set(name, String(value)));
      return params.toString();
    };

    let logRequestSequence = 0;
    const loadLogs = async (offset = 0) => {
      if (!adminToken.value) return;
      const requestSequence = ++logRequestSequence;
      logsLoading.value = true;
      logPage.offset = Math.max(0, offset);
      try {
        const page = await api(`logs?${logQuery()}`);
        if (requestSequence !== logRequestSequence) return;
        logs.value = page?.items || [];
        logPage.total = Number(page?.total || 0);
        logPage.limit = Number(page?.limit || logPage.limit);
        logPage.offset = Number(page?.offset || 0);
      } catch (error) {
        if (requestSequence === logRequestSequence) {
          refreshError.value = error?.message || t('error.refresh');
        }
      } finally {
        if (requestSequence === logRequestSequence) {
          logsLoading.value = false;
        }
      }
    };

    let logRefreshTimer = null;
    const stopLogAutoRefresh = () => {
      if (logRefreshTimer !== null && typeof window.clearInterval === 'function') window.clearInterval(logRefreshTimer);
      logRefreshTimer = null;
    };
    const startLogAutoRefresh = () => {
      stopLogAutoRefresh();
      if (!logsAutoRefresh.value || activeSection.value !== 'logs' || typeof window.setInterval !== 'function') return;
      logRefreshTimer = window.setInterval(() => {
        if (document.visibilityState !== 'hidden' && !logsLoading.value) loadLogs(logPage.offset);
      }, 5000);
    };
    const syncLogsAutoRefresh = () => {
      localStorage.setItem('gateway.logsAutoRefresh', String(logsAutoRefresh.value));
      if (logsAutoRefresh.value) startLogAutoRefresh();
      else stopLogAutoRefresh();
    };

    const exportLogs = async () => runAction(async () => {
      await download(`logs?${logQuery({ format: 'csv', limit: 10000, offset: 0 })}`, {}, 'gateway-request-logs.csv');
    });

    const clearLogs = async () => {
      if (!await openConfirmDialog(t('confirm.clearLogs'), t('confirm.clearLogsTitle'))) return;
      await runAction(async () => {
        await api('logs', { method: 'DELETE' });
        await refresh();
        await loadLogs(0);
      }, refreshError, t('feedback.logsCleared'));
    };

    const saveRouting = async () => runAction(async () => {
      const result = await api('routing', { method: 'PATCH', body: JSON.stringify(routing.value) });
      routing.value = result?.routing || routing.value;
      routingMessage.value = result?.restartRequired ? t('routing.restartRequired') : t('routing.saved');
    });

    const checkIntegrity = async () => runAction(async () => {
      const result = await api('maintenance/integrity');
      maintenanceMessage.value = result?.status === 'ok' ? t('maintenance.integrityOk') : t('error.action');
    });

    const downloadBackup = async () => runAction(async () => {
      await download('maintenance/backup', { method: 'POST' }, 'gateway-backup.db');
      maintenanceMessage.value = t('maintenance.databaseBackupCreated');
    });

    const downloadPortableBackup = async () => runAction(async () => {
      if (backupPassphrase.value.length < 12) throw new Error(t('maintenance.passphraseTooShort'));
      if (backupPassphrase.value !== backupPassphraseConfirm.value) throw new Error(t('maintenance.passphraseMismatch'));
      await download('maintenance/portable-backup', {
        method: 'POST',
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ passphrase: backupPassphrase.value }),
      }, 'gateway-portable-backup.zip');
      backupPassphrase.value = '';
      backupPassphraseConfirm.value = '';
      maintenanceMessage.value = t('maintenance.portableBackupCreated');
    });

    const logoutAdmin = async () => {
      stopLogAutoRefresh();
      sessionStorage.removeItem('gateway.adminToken');
      localStorage.removeItem('gateway.adminToken');
      adminToken.value = '';
      authForm.token = '';
      authError.value = '';
      authRequired.value = true;
      dashboard.value = null;
      providers.value = [];
      keys.value = [];
      gatewayKeys.value = [];
      balances.value = [];
      balanceResults.value = [];
      providerModels.value = {};
      Object.keys(modelPolicyDrafts).forEach((providerId) => delete modelPolicyDrafts[providerId]);
      Object.keys(modelPolicySearch).forEach((providerId) => delete modelPolicySearch[providerId]);
      modelDiscovery.value = {};
      modelRoutes.value = [];
      modelStates.value = [];
      logRequestSequence += 1;
      logsLoading.value = false;
      logs.value = [];
      routing.value = {};
      snippets.value = {};
      refreshError.value = '';
      notify('success', t('auth.loggedOut'));
    };

    const submitAdminLogin = async () => {
      const token = authForm.token.trim();
      authError.value = '';
      if (!token) {
        authError.value = t('auth.required');
        return;
      }
      authSubmitting.value = true;
      adminToken.value = token;
      sessionStorage.setItem('gateway.adminToken', token);
      const ok = await refresh();
      authSubmitting.value = false;
      if (!ok) {
        sessionStorage.removeItem('gateway.adminToken');
        adminToken.value = '';
        authRequired.value = true;
        authError.value = t('auth.invalid');
      } else {
        authForm.token = '';
        authError.value = '';
      }
    };

    const setLang = (next) => {
      lang.value = next;
      localStorage.setItem('gateway.lang', next);
      document.documentElement.lang = next === 'zh' ? 'zh-CN' : 'en';
    };

    const setTheme = (next) => {
      theme.value = next;
      localStorage.setItem('gateway.theme', next);
      document.documentElement.dataset.theme = next;
      document.documentElement.style.colorScheme = next;
    };

    const setActiveSection = (id) => {
      const next = sectionIDs.includes(id) ? id : 'dashboard';
      activeSection.value = next;
      if (window.location.hash !== `#${next}`) {
        history.pushState(null, '', `#${next}`);
      }
      if (next === 'logs') {
        loadLogs(0);
        startLogAutoRefresh();
      } else {
        stopLogAutoRefresh();
      }
    };

    const syncSectionFromHash = () => {
      const next = sectionFromHash();
      if (next === activeSection.value) return;
      activeSection.value = next;
      if (next === 'logs') {
        loadLogs(0);
        startLogAutoRefresh();
      } else {
        stopLogAutoRefresh();
      }
    };

    const toggleTheme = () => {
      setTheme(theme.value === 'dark' ? 'light' : 'dark');
    };

    const currentTitle = computed(() => navItems.find((item) => item.id === activeSection.value)?.label || 'nav.dashboard');
    const port = computed(() => new URL(dashboard.value?.service?.proxyUrl || window.location.origin).port || (window.location.protocol === 'https:' ? '443' : '80'));
    const stats = computed(() => dashboard.value?.stats || {});
    const recentLogs = computed(() => stats.value.recent || []);
    const serviceReadiness = computed(() => dashboard.value?.service?.readiness || 'unknown');
    const serviceReadinessLabel = computed(() => t(`service.state.${serviceReadiness.value}`));
    const serviceReadinessReason = computed(() => t(`service.reason.${dashboard.value?.service?.readinessReason || 'unknown'}`));
    const serviceReadinessClass = computed(() => `is-${serviceReadiness.value}`);
    const setupSteps = computed(() => [
      { id: 'provider', label: 'setup.stepProvider', done: providers.value.some((item) => item.enabled) },
      { id: 'upstreamKey', label: 'setup.stepUpstreamKey', done: keys.value.some((item) => item.enabled && item.providerEnabled) },
      { id: 'gatewayKey', label: 'setup.stepGatewayKey', done: !!dashboard.value?.service?.gatewayAuthConfigured },
    ]);
    const setupComplete = computed(() => setupSteps.value.every((step) => step.done));
    const metricCards = computed(() => [
      { label: 'card.service', value: serviceReadinessLabel.value },
      { label: 'card.totalKeys', value: stats.value.totalKeys ?? 0 },
      { label: 'card.activeKeys', value: stats.value.activeKeys ?? 0 },
      { label: 'card.coolingKeys', value: stats.value.failedKeys ?? 0 },
      { label: 'card.todayRequests', value: stats.value.todayRequests ?? 0 },
      { label: 'card.todayTokens', value: stats.value.todayTokens ?? 0 },
    ]);

    const replaceSnippetKey = (value, gatewayKey) => {
      if (typeof value === 'string') {
        return gatewayKey ? value.replaceAll('CREATE_A_GATEWAY_KEY_IN_ADMIN', gatewayKey) : value;
      }
      if (Array.isArray(value)) return value.map((item) => replaceSnippetKey(item, gatewayKey));
      if (value && typeof value === 'object') {
        return Object.fromEntries(Object.entries(value).map(([name, item]) => [name, replaceSnippetKey(item, gatewayKey)]));
      }
      return value;
    };
    const setupSnippets = computed(() => replaceSnippetKey(snippets.value, createdGatewayKey.value?.plaintext || ''));
    const setupNeedsGatewayKey = computed(() => JSON.stringify(setupSnippets.value).includes('CREATE_A_GATEWAY_KEY_IN_ADMIN'));

    const pretty = (value) => JSON.stringify(value || {}, null, 2);

    const timeFormatters = new Map();
    const gatewayTimezone = () => dashboard.value?.service?.timezone || 'Asia/Singapore';
    const gatewayTimeFormatter = () => {
      const timezone = gatewayTimezone();
      if (!timeFormatters.has(timezone)) {
        timeFormatters.set(timezone, new Intl.DateTimeFormat('en-CA', {
          timeZone: timezone,
          year: 'numeric',
          month: '2-digit',
          day: '2-digit',
          hour: '2-digit',
          minute: '2-digit',
          second: '2-digit',
          hourCycle: 'h23',
        }));
      }
      return timeFormatters.get(timezone);
    };

    const formatGatewayTime = (value) => {
      if (!value) return '-';
      const date = value instanceof Date ? value : new Date(value);
      if (Number.isNaN(date.getTime())) return value;
      const parts = Object.fromEntries(
        gatewayTimeFormatter().formatToParts(date).map((part) => [part.type, part.value])
      );
      return `${parts.year}-${parts.month}-${parts.day} ${parts.hour}:${parts.minute}:${parts.second} ${gatewayTimezone()}`;
    };

    const formatShortTime = (value) => {
      if (!value) return '-';
      const date = value instanceof Date ? value : new Date(value);
      if (Number.isNaN(date.getTime())) return '-';
      const parts = Object.fromEntries(
        gatewayTimeFormatter().formatToParts(date).map((part) => [part.type, part.value])
      );
      return `${parts.hour}:${parts.minute}:${parts.second}`;
    };

    const logStatusClass = (status) => {
      const code = Number(status);
      if (code >= 200 && code < 300) return 'is-success';
      if (code >= 400 && code < 500) return 'is-warning';
      return 'is-error';
    };

    const logStatusLabel = (status) => {
      const state = logStatusClass(status);
      if (state === 'is-success') return t('dashboard.requestSuccess');
      if (state === 'is-warning') return t('dashboard.requestWarning');
      return t('dashboard.requestError');
    };

    const logProtocolLabel = (protocol) => {
      const labels = {
        openai: 'OpenAI',
        'openai-responses': 'OpenAI Responses',
        anthropic: 'Anthropic',
        gemini: 'Gemini',
      };
      const value = String(protocol || '').trim();
      return labels[value.toLowerCase()] || value || '-';
    };

    const logModelLabel = (model) => {
      const value = String(model || '').trim();
      return !value || value.toLowerCase() === 'auto' ? t('dashboard.autoRoute') : value;
    };

    const providerKeyHints = (providerId) => keys.value
      .filter((k) => k.providerId === providerId)
      .map((k) => k.keyHint || '***');

    const providerKeys = (providerId) => keys.value.filter((k) => k.providerId === providerId);

    const providerDisplayName = (providerId) => {
      const provider = providers.value.find((item) => item.id === providerId);
      return provider?.name || providerId || '-';
    };

    const keyDisplayName = (keyId) => {
      const key = keys.value.find((item) => item.id === keyId);
      return key?.name || keyId || '-';
    };

    const hasDistinctProviderName = (providerId) => {
      const name = providerDisplayName(providerId);
      return !!providerId && name !== providerId;
    };

    const hasDistinctKeyName = (keyId) => {
      const name = keyDisplayName(keyId);
      return !!keyId && name !== keyId;
    };

    const providerApiEndpoint = (provider) => {
      const base = (provider?.baseUrl || '').replace(/\/+$/, '');
      if (!base) return '-';
      if (['openai-compatible', 'new-api', 'sub2api'].includes(provider.type)) {
        return base.endsWith('/v1') ? `${base}/models` : `${base}/v1/models`;
      }
      if (provider.type === 'gemini-compatible') return `${base}/v1beta/models`;
      return `${base}/models`;
    };

    const balanceUnitSuffix = (currency) => {
      const unit = String(currency || '').trim();
      if (!unit) return '';
      if (unit === '$' || unit === '￥' || unit === '¥') return unit;
      return ` ${unit}`;
    };

    const formatBalanceValue = (value, currency = '') => {
      if (value === undefined || value === null || Number.isNaN(Number(value))) return '-';
      const n = Number(value);
      return `${n.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}${balanceUnitSuffix(currency)}`;
    };

    const balanceStatusText = (status) => {
      if (status === 'unlimited') return t('table.unlimited');
      return status || '-';
    };

    const displayBalanceValue = (item) => {
      if (!item) return '-';
      if (item.status === 'unlimited') return t('table.unlimited');
      return formatBalanceValue(item.balance, item.currency);
    };

    const balanceStatusClass = (item) => {
      if (item?.status === 'unlimited') return 'ok';
      if (item?.balance !== undefined && item?.balance !== null && Number(item.balance) < 0) return 'bad';
      return 'ok';
    };

    const balanceValueClass = (item) => {
      if (!item) return 'balance-value';
      if (item.status === 'unlimited') return 'balance-value ok';
      if (item.balance !== undefined && item.balance !== null && Number(item.balance) < 0) return 'balance-value bad';
      return 'balance-value';
    };

    const providerBalanceClass = (providerId) => balanceValueClass(balances.value.find((b) => b.providerId === providerId));

    const resultStatusClass = (status) => {
      if (status === 'ok' || status === 'unlimited') return 'ok';
      return 'bad';
    };

    const providerBalanceText = (providerId) => {
      const item = balances.value.find((b) => b.providerId === providerId);
      if (!item) return '-';
      if (item.status === 'unlimited') return balanceStatusText(item.status);
      if (item.status && item.status !== 'ok') return item.status;
      if (item.balance === undefined || item.balance === null) return item.status || '-';
      return formatBalanceValue(item.balance, item.currency);
    };

    const testResultText = (result) => {
      if (!result) return '';
      if (result.status === 'ok') {
        const models = result.modelCount === undefined || result.modelCount === null ? '' : ` · ${result.modelCount} models`;
        const tokens = result.totalTokens === undefined || result.totalTokens === null ? '' : ` · ${result.totalTokens} tokens`;
        return `${result.statusCode || 200} · ${result.latencyMs || 0}ms${models}${tokens}`;
      }
      return result.error || `${result.statusCode || '-'} · ${result.latencyMs || 0}ms`;
    };

    const testStatusClass = (result) => (result?.connectionStatus === 'ok' || result?.status === 'ok' ? 'ok' : 'bad');

    const testConnectionText = (result) => {
      const ok = result?.connectionStatus === 'ok' || result?.status === 'ok';
      return `${t('test.connection')}: ${ok ? t('test.ok') : t('test.failed')}`;
    };

    const testErrorText = (result) => {
      if (!result?.error) return '';
      const connectionOk = result?.connectionStatus === 'ok' || result?.status === 'ok';
      if (connectionOk && result?.tokenStatus && result.tokenStatus !== 'ok' && result.tokenStatus !== 'skipped') {
        return `${t('test.tokenWarning')}: ${result.error}`;
      }
      return result.error;
    };

    const tokenUsageText = (result) => {
      if (!result) return '-';
      if (result.totalTokens !== undefined && result.totalTokens !== null) {
        const prompt = result.promptTokens === undefined || result.promptTokens === null ? '-' : result.promptTokens;
        const completion = result.completionTokens === undefined || result.completionTokens === null ? '-' : result.completionTokens;
        return `${result.totalTokens} (${prompt}+${completion})`;
      }
      if (result.tokenStatus === 'skipped') return t('test.skipped');
      if (result.tokenStatus && result.tokenStatus !== 'ok') return t('test.tokenWarning');
      return result.tokenStatus || '-';
    };

    const capabilitiesText = (result) => Array.isArray(result?.capabilities) ? result.capabilities.join(', ') : '-';

    const modelMapDefault = (modelMap) => {
      const value = modelMap?.['*'] || modelMap?.default || '';
      return typeof value === 'string' ? value : '';
    };

    const rememberProviderModels = (providerId, models = []) => {
      if (!providerId || !Array.isArray(models) || !models.length) return;
      const merged = [...(providerModels.value[providerId] || []), ...models]
        .map((model) => String(model || '').trim())
        .filter(Boolean);
      providerModels.value = {
        ...providerModels.value,
        [providerId]: [...new Set(merged)].slice(0, 300),
      };
    };

    const providerModelOptions = (providerId) => {
      const provider = providers.value.find((item) => item.id === providerId);
      const fromMap = Object.values(provider?.modelMap || {}).filter(Boolean);
      const fromTests = providerModels.value[providerId] || [];
      return [...new Set([...fromMap, ...fromTests].map((model) => String(model || '').trim()).filter(Boolean))];
    };

    const providerDiscoveredModels = (providerId) => providerModels.value[providerId] || [];

    const providerModelPreview = (providerId) => providerDiscoveredModels(providerId).slice(0, 12);

    const isFlagshipModel = (modelId) => {
      const value = String(modelId || '').trim().toLowerCase();
      if (!value) return false;
      if (/(?:^|[/_.-])(embed(?:ding)?|rerank|moderation|guard|safety|tts|speech|whisper|transcri(?:be|ption)|ocr)(?:$|[/_.-])/.test(value)) return false;
      return [
        /(?:^|\/)step-3\.7-flash$/,
        /(?:^|\/)kat-coder-pro-v2$/,
        /(?:^|\/)gpt-image-2$/,
        /(?:^|[/_.-])gpt[/_.-]?(?:5|4(?:[/_.-]?1|o)?)(?:$|[/_.-])/,
        /(?:^|[/_.-])o(?:1|3|4)(?:$|[/_.-])/,
        /claude.*(?:opus|sonnet)/,
        /gemini.*(?:ultra|pro|flash)/,
        /deepseek.*(?:v3|v4|r1|reasoner)/,
        /(?:kimi|moonshot).*k(?:2|1[_.-]?5)/,
        /qwen.*(?:max|plus|72b|110b)/,
        /(?:^|[/_.-])glm[/_.-]?(?:4|5)/,
        /minimax.*m(?:2|3)/,
        /grok[/_.-]?(?:3|4)/,
        /mistral.*(?:large|medium)/,
        /llama.*(?:70b|405b)/,
      ].some((pattern) => pattern.test(value));
    };

    const modelOptionLabel = (modelId) => (
      isFlagshipModel(modelId) ? `${modelId} · ${t('model.recommended')}` : modelId
    );

    const syncModelPolicyDrafts = (nextProviders) => {
      const active = new Set();
      (nextProviders || []).forEach((provider) => {
        active.add(provider.id);
        if (modelPolicyDrafts[provider.id]?.dirty) return;
        const savedModels = [...new Set((provider.modelAllowlist || []).map((item) => String(item || '').trim()).filter(Boolean))];
        modelPolicyDrafts[provider.id] = {
          enabled: !!provider.modelAllowlistEnabled,
          models: savedModels.length || provider.modelAllowlistEnabled
            ? savedModels
            : [...new Set((providerModels.value[provider.id] || []).map((item) => String(item || '').trim()).filter(Boolean))],
          dirty: false,
        };
      });
      Object.keys(modelPolicyDrafts).forEach((providerId) => {
        if (!active.has(providerId)) delete modelPolicyDrafts[providerId];
      });
      Object.keys(modelPolicySearch).forEach((providerId) => {
        if (!active.has(providerId)) delete modelPolicySearch[providerId];
      });
    };

    const modelPolicyDraft = (providerId) => modelPolicyDrafts[providerId] || { enabled: false, models: [], dirty: false };

    const providerModelPolicyOptions = (providerId) => {
      const provider = providers.value.find((item) => item.id === providerId);
      const selected = provider?.modelAllowlist || [];
      return [...new Set([...providerDiscoveredModels(providerId), ...selected]
        .map((item) => String(item || '').trim()).filter(Boolean))]
        .sort((a, b) => a.localeCompare(b));
    };

    const filteredProviderModelPolicyOptions = (providerId) => {
      const query = String(modelPolicySearch[providerId] || '').trim().toLowerCase();
      const options = providerModelPolicyOptions(providerId);
      return (query ? options.filter((item) => item.toLowerCase().includes(query)) : options).slice(0, 300);
    };

    const modelPolicySelected = (providerId, modelId) => modelPolicyDraft(providerId).models.includes(modelId);

    const setModelPolicyEnabled = (providerId, enabled) => {
      const draft = modelPolicyDrafts[providerId];
      if (!draft) return;
      draft.enabled = !!enabled;
      draft.dirty = true;
    };

    const setProviderModelSelected = (providerId, modelId, selected) => {
      const draft = modelPolicyDrafts[providerId];
      if (!draft) return;
      const next = new Set(draft.models);
      if (selected) next.add(modelId);
      else next.delete(modelId);
      draft.models = [...next];
      draft.dirty = true;
    };

    const selectProviderModels = (providerId, mode) => {
      const draft = modelPolicyDrafts[providerId];
      if (!draft) return;
      const options = providerModelPolicyOptions(providerId);
      if (mode === 'all') draft.models = options;
      else if (mode === 'flagship') draft.models = options.filter(isFlagshipModel);
      else draft.models = [];
      draft.enabled = true;
      draft.dirty = true;
    };

    const saveProviderModelPolicy = async (provider) => {
      const draft = modelPolicyDrafts[provider?.id];
      if (!provider?.id || !draft || modelPolicySavingProviderId.value) return;
      modelPolicySavingProviderId.value = provider.id;
      try {
        await runAction(async () => {
          await api(`providers/${provider.id}`, {
            method: 'PATCH',
            body: JSON.stringify({
              modelAllowlistEnabled: !!draft.enabled,
              modelAllowlist: [...draft.models],
            }),
          });
          draft.dirty = false;
          await refresh();
        });
      } finally {
        modelPolicySavingProviderId.value = '';
      }
    };

    const discoveryForProvider = (providerId) => modelDiscovery.value[providerId] || {
      providerId,
      status: 'unknown',
      modelCount: providerModelOptions(providerId).length,
    };

    const discoveryStatusClass = (providerId) => {
      const status = discoveryForProvider(providerId).status;
      if (status === 'ok') return 'ok';
      if (status === 'unknown') return '';
      return 'bad';
    };

    const discoveryStatusText = (providerId) => {
      const status = discoveryForProvider(providerId).status;
      if (status === 'ok') return t('model.discovery.ok');
      if (status === 'partial') return t('model.discovery.partial');
      if (status === 'unknown') return t('model.discovery.unknown');
      if (status === 'empty') return t('model.discovery.empty');
      return t('model.discovery.error');
    };

    const refreshProviderModels = async (provider) => {
      if (!provider?.id || refreshingProviderId.value) return;
      refreshingProviderId.value = provider.id;
      try {
        await runAction(async () => {
          const result = await api(`providers/${provider.id}/models`, { method: 'POST', body: '{}' });
          rememberProviderModels(provider.id, result?.models || []);
          if (result?.keyId) keyTestResults.value = { ...keyTestResults.value, [result.keyId]: result };
          await refresh();
        });
      } finally {
        refreshingProviderId.value = '';
      }
    };

    const defaultRouteTarget = (providerId = '') => {
      const selectedProvider = providerId || providers.value.find((item) => item.enabled)?.id || providers.value[0]?.id || '';
      return {
        providerId: selectedProvider,
        upstreamModel: providerModelOptions(selectedProvider)[0] || '',
        enabled: true,
      };
    };

    const defaultRouteModel = (priority = 100) => {
      const target = defaultRouteTarget();
      return {
        name: target.upstreamModel || '',
        priority,
        enabled: true,
        targets: [target],
      };
    };

    const resetModelRouteForm = () => {
      Object.assign(modelRouteForm, {
        editingId: '',
        id: '',
        name: '',
        enabled: true,
        models: [defaultRouteModel()],
      });
    };

    const openModelRouteForm = (route = null) => {
      setActiveSection('model-routing');
      formError.value = '';
      if (!route) {
        resetModelRouteForm();
      } else {
        Object.assign(modelRouteForm, {
          editingId: route.id,
          id: route.id,
          name: route.name || route.id,
          enabled: !!route.enabled,
          models: (route.models || []).map((item) => ({
            name: item.name || '',
            priority: Number(item.priority || 0),
            enabled: !!item.enabled,
            targets: (item.targets || []).map((target) => ({
              providerId: target.providerId || '',
              upstreamModel: target.upstreamModel || '',
              enabled: !!target.enabled,
            })),
          })),
        });
      }
      modelRouteFormOpen.value = true;
      providerFormOpen.value = false;
      keyFormOpen.value = false;
    };

    const cancelModelRouteForm = () => {
      resetModelRouteForm();
      modelRouteFormOpen.value = false;
      formError.value = '';
    };

    const addRouteModel = () => {
      const priorities = modelRouteForm.models.map((item) => Number(item.priority || 0));
      const nextPriority = priorities.length ? Math.max(0, Math.min(...priorities) - 10) : 100;
      modelRouteForm.models.push(defaultRouteModel(nextPriority));
    };

    const removeRouteModel = (modelIndex) => {
      modelRouteForm.models.splice(modelIndex, 1);
    };

    const addRouteTarget = (modelIndex) => {
      const routeModel = modelRouteForm.models[modelIndex];
      if (!routeModel) return;
      const usedProviders = new Set(routeModel.targets.map((target) => target.providerId).filter(Boolean));
      const providerId = providers.value.find((provider) => provider.enabled && !usedProviders.has(provider.id))?.id
        || providers.value.find((provider) => !usedProviders.has(provider.id))?.id
        || '';
      routeModel.targets.push(providerId ? defaultRouteTarget(providerId) : { providerId: '', upstreamModel: '', enabled: true });
    };

    const removeRouteTarget = (modelIndex, targetIndex) => {
      modelRouteForm.models[modelIndex]?.targets.splice(targetIndex, 1);
    };

    const saveModelRoute = async () => {
      formError.value = '';
      const payload = {
        id: modelRouteForm.id.trim(),
        name: modelRouteForm.name.trim() || modelRouteForm.id.trim(),
        enabled: !!modelRouteForm.enabled,
        models: modelRouteForm.models.map((item) => ({
          name: String(item.name || '').trim(),
          priority: Number(item.priority || 0),
          enabled: !!item.enabled,
          targets: (item.targets || []).map((target) => ({
            providerId: String(target.providerId || '').trim(),
            upstreamModel: String(target.upstreamModel || '').trim(),
            enabled: !!target.enabled,
          })),
        })),
      };
      if (!payload.id || !payload.models.length) {
        formError.value = t('error.modelRouteRequired');
        return;
      }
      await runAction(async () => {
        await api(modelRouteForm.editingId ? `model-routes/${modelRouteForm.editingId}` : 'model-routes', {
          method: modelRouteForm.editingId ? 'PUT' : 'POST',
          body: JSON.stringify(payload),
        });
        cancelModelRouteForm();
        await refresh();
      }, formError);
    };

    const toggleModelRoute = async (route) => runAction(async () => {
      await api(`model-routes/${route.id}`, {
        method: 'PUT',
        body: JSON.stringify({ ...route, enabled: !route.enabled }),
      });
      await refresh();
    });

    const deleteModelRoute = async (route) => {
      if (!confirmDelete(t('confirm.deleteModelRoute', { id: route.id }), route.id)) return;
      await runAction(async () => {
        await api(`model-routes/${route.id}`, { method: 'DELETE' });
        await refresh();
      });
    };

    const routeProviderSelected = (modelIndex, targetIndex, providerId) => modelRouteForm.models[modelIndex]?.targets.some(
      (target, index) => index !== targetIndex && target.providerId === providerId,
    );

    const modelStatesFor = (providerId, upstreamModel) => modelStates.value.filter(
      (item) => item.providerId === providerId && item.modelId === upstreamModel,
    );

    const modelCoolingCount = (providerId, upstreamModel) => {
      const now = Date.now();
      return modelStatesFor(providerId, upstreamModel).filter((item) => {
        if (item.scope === 'provider') return false;
        const until = Date.parse(item.cooldownUntil || '');
        return Number.isFinite(until) && until > now;
      }).length;
    };

    const modelProviderCooling = (providerId, upstreamModel) => {
      const now = Date.now();
      return modelStatesFor(providerId, upstreamModel).some((item) => {
        if (item.scope !== 'provider') return false;
        const until = Date.parse(item.cooldownUntil || '');
        return Number.isFinite(until) && until > now;
      });
    };

    const modelHasFailureState = (providerId, upstreamModel) => modelStatesFor(providerId, upstreamModel).some(
      (item) => item.cooldownUntil || item.lastError || Number(item.consecutiveFailures || 0) > 0,
    );

    const resetModelState = async (providerId, upstreamModel) => runAction(async () => {
      await api('model-states/reset', {
        method: 'POST',
        body: JSON.stringify({ providerId, modelId: upstreamModel }),
      });
      await refresh();
    });

    const applyDefaultModelToMap = (modelMap, defaultModel) => {
      const next = { ...(modelMap || {}) };
      const value = String(defaultModel || '').trim();
      if (value) {
        next['*'] = value;
      } else {
        delete next['*'];
        delete next.default;
      }
      return next;
    };

    const openProviderForm = () => {
      setActiveSection('providers');
      resetProviderForm();
      providerFormOpen.value = true;
      keyFormOpen.value = false;
      formError.value = '';
    };

    const openKeyForm = (provider = null) => {
      setActiveSection('providers');
      resetKeyForm();
      formError.value = '';
      if (provider?.id) {
        keyForm.providerId = provider.id;
        keyForm.name = `${provider.id}-key-1`;
      } else if (!keyForm.providerId && providers.value.length) {
        keyForm.providerId = providers.value[0].id;
      }
      keyFormOpen.value = true;
      providerFormOpen.value = false;
    };

    const editProvider = (provider) => {
      setActiveSection('providers');
      formError.value = '';
      Object.assign(providerForm, {
        editingId: provider.id,
        id: provider.id,
        name: provider.name || provider.id,
        type: provider.type || 'openai-compatible',
        baseUrl: provider.baseUrl || '',
        balancePath: provider.balancePath || '',
        priority: Number(provider.priority || 0),
        modelMap: JSON.stringify(provider.modelMap || {}, null, 2),
        defaultModel: modelMapDefault(provider.modelMap),
        firstKeyName: '',
        firstKeySecret: '',
        firstKeyPriority: 0,
      });
      providerFormOpen.value = true;
      keyFormOpen.value = false;
    };

    const editKey = (key) => {
      setActiveSection('providers');
      formError.value = '';
      Object.assign(keyForm, {
        editingId: key.id,
        providerId: key.providerId,
        name: key.name || key.id,
        secret: '',
        priority: Number(key.priority || 0),
        defaultModel: modelMapDefault(key.providerModelMap),
        enabled: !!key.enabled,
      });
      keyFormOpen.value = true;
      providerFormOpen.value = false;
    };

    const openSetup = () => {
      setActiveSection('setup');
      formError.value = '';
    };

    const openSetupStep = (step) => {
      if (step.id === 'provider') {
        openProviderForm();
        return;
      }
      if (step.id === 'upstreamKey') {
        if (!providers.value.length) {
          openProviderForm();
          return;
        }
        setActiveSection('providers');
        openKeyForm(providers.value[0]);
        return;
      }
      setActiveSection('keys');
    };

    const parseModelMap = () => {
      const raw = (providerForm.modelMap || '').trim();
      if (!raw) return {};
      const parsed = JSON.parse(raw);
      if (!parsed || Array.isArray(parsed) || typeof parsed !== 'object') throw new Error(t('error.modelMapJson'));
      return parsed;
    };

    const resetProviderForm = () => {
      Object.assign(providerForm, {
        editingId: '',
        id: '',
        name: '',
        type: 'openai-compatible',
        baseUrl: '',
        balancePath: '',
        priority: 0,
        modelMap: '{}',
        defaultModel: '',
        firstKeyName: '',
        firstKeySecret: '',
        firstKeyPriority: 0,
      });
    };

    const resetKeyForm = () => {
      Object.assign(keyForm, {
        editingId: '',
        providerId: providers.value[0]?.id || '',
        name: '',
        secret: '',
        priority: 0,
        defaultModel: '',
        enabled: true,
      });
    };

    const cancelProviderForm = () => {
      resetProviderForm();
      providerFormOpen.value = false;
      formError.value = '';
    };

    const cancelKeyForm = () => {
      resetKeyForm();
      keyFormOpen.value = false;
      formError.value = '';
    };

    const saveProvider = async () => {
      formError.value = '';
      if (!providerForm.name || !providerForm.type || !providerForm.baseUrl) {
        formError.value = t('error.providerRequired');
        return;
      }
      let modelMap = {};
      try {
        modelMap = parseModelMap();
      } catch (err) {
        formError.value = err.message || t('error.modelMapJson');
        return;
      }
      modelMap = applyDefaultModelToMap(modelMap, providerForm.defaultModel);
      const isEditing = !!providerForm.editingId;
      const providerPayload = {
        id: providerForm.id,
        name: providerForm.name,
        type: providerForm.type,
        baseUrl: providerForm.baseUrl,
        balancePath: providerForm.balancePath,
        priority: Number(providerForm.priority || 0),
        modelMap,
      };
      if (!isEditing) providerPayload.enabled = true;
      const firstKeySecret = providerForm.firstKeySecret.trim();
      if (!isEditing && firstKeySecret) {
        providerPayload.firstKey = {
          name: providerForm.firstKeyName || `${providerForm.name}-key-1`,
          secret: firstKeySecret,
          priority: Number(providerForm.firstKeyPriority || 0),
          enabled: true,
        };
      }
      await runAction(async () => {
        const savedResult = await api(isEditing ? `providers/${providerForm.editingId}` : 'providers', {
          method: isEditing ? 'PATCH' : 'POST',
          body: JSON.stringify(providerPayload),
        });
        const savedProvider = savedResult?.provider || savedResult;
        const savedProviderId = savedProvider?.id || providerForm.id;
        resetProviderForm();
        providerFormOpen.value = false;
        await refresh();
        const targetProvider = providers.value.find((p) => p.id === savedProviderId) || providers.value[0];
        if (!isEditing && targetProvider && !firstKeySecret) openKeyForm(targetProvider);
      }, formError, t('feedback.providerSaved'));
    };

    const saveKey = async () => {
      const isEditing = !!keyForm.editingId;
      if (!keyForm.providerId || !keyForm.name.trim() || (!isEditing && !keyForm.secret.trim())) {
        formError.value = t('error.keyRequired');
        return;
      }
      await runAction(async () => {
        await api(isEditing ? `keys/${keyForm.editingId}` : 'keys', {
          method: isEditing ? 'PATCH' : 'POST',
          body: JSON.stringify({
            providerId: keyForm.providerId,
            name: keyForm.name,
            secret: keyForm.secret,
            priority: Number(keyForm.priority || 0),
            enabled: keyForm.enabled,
          }),
        });
        const provider = providers.value.find((item) => item.id === keyForm.providerId);
        if (provider && keyForm.defaultModel !== modelMapDefault(provider.modelMap)) {
          await api(`providers/${provider.id}`, {
            method: 'PATCH',
            body: JSON.stringify({
              name: provider.name,
              type: provider.type,
              baseUrl: provider.baseUrl,
              balancePath: provider.balancePath,
              priority: Number(provider.priority || 0),
              enabled: provider.enabled,
              modelMap: applyDefaultModelToMap(provider.modelMap || {}, keyForm.defaultModel),
            }),
          });
        }
        resetKeyForm();
        keyFormOpen.value = false;
        formError.value = '';
        await refresh();
      }, formError, t('feedback.keySaved'));
    };

    const createGatewayKey = async () => {
      await runAction(async () => {
        const item = await api('gateway-keys', {
          method: 'POST',
          body: JSON.stringify({ name: gatewayKeyForm.name || 'Gateway Key' }),
        });
        createdGatewayKey.value = { ...item, event: 'created' };
        gatewayKeyForm.name = '';
        await refresh();
      }, refreshError, t('feedback.gatewayKeyCreated'));
    };

    const refreshBalances = async () => {
      if (balanceRefreshing.value) return;
      balanceRefreshing.value = true;
      try {
        await runAction(async () => {
          const result = await api('balance/refresh', { method: 'POST', body: '{}' });
          balanceResults.value = result?.results || [];
          await refresh();
        });
      } finally {
        balanceRefreshing.value = false;
      }
    };

    const testKey = async (k) => {
      testingKeyId.value = k.id;
      try {
        await runAction(async () => {
          const result = await api(`keys/${k.id}/test`, { method: 'POST', body: '{}' });
          keyTestResults.value = { ...keyTestResults.value, [k.id]: result };
          rememberProviderModels(k.providerId, result?.models || []);
        });
      } finally {
        testingKeyId.value = '';
      }
    };

    const testAllKeys = async () => {
      if (testingAllKeys.value) return;
      testingAllKeys.value = true;
      try {
        await runAction(async () => {
          const result = await api('keys/test-all', { method: 'POST', body: '{}' });
          const items = result?.results || [];
          const next = { ...keyTestResults.value };
          items.forEach((item) => { next[item.keyId] = item; });
          keyTestResults.value = next;
          bulkTestSummary.value = {
            total: items.length,
            ok: items.filter((item) => item.status === 'ok').length,
            failed: items.filter((item) => item.status !== 'ok').length,
          };
        }, refreshError, t('feedback.keysTested'));
      } finally {
        testingAllKeys.value = false;
      }
    };

    const setCopyFeedback = (id, state) => {
      const key = id || 'default';
      if (copyTimers.has(key)) window.clearTimeout(copyTimers.get(key));
      copyFeedback.value = { ...copyFeedback.value, [key]: state };
      copyTimers.set(key, window.setTimeout(() => {
        const next = { ...copyFeedback.value };
        delete next[key];
        copyFeedback.value = next;
        copyTimers.delete(key);
      }, 1800));
    };

    const copyText = async (text, id = 'default') => {
      if (!text) return;
      try {
        await navigator.clipboard.writeText(text);
        setCopyFeedback(id, 'ok');
      } catch {
        setCopyFeedback(id, 'bad');
      }
    };

    const copyButtonText = (id = 'default') => {
      const state = copyFeedback.value[id];
      if (state === 'ok') return t('action.copied');
      if (state === 'bad') return t('action.copyFailed');
      return t('action.copy');
    };

    const copyButtonClass = (id = 'default') => {
      const state = copyFeedback.value[id];
      return {
        'copy-success': state === 'ok',
        'copy-error': state === 'bad',
      };
    };

    const toggleProvider = async (p) => {
      await runAction(async () => {
        await api(`providers/${p.id}`, { method: 'PATCH', body: JSON.stringify({ enabled: !p.enabled }) });
        await refresh();
      });
    };

    const deleteProvider = async (p) => {
      if (!await confirmDelete(t('confirm.deleteProvider', { id: p.id }), p.id)) return;
      await runAction(async () => {
        await api(`providers/${p.id}`, { method: 'DELETE' });
        await refresh();
      });
    };

    const toggleKey = async (k) => {
      await runAction(async () => {
        await api(`keys/${k.id}`, { method: 'PATCH', body: JSON.stringify({ enabled: !k.enabled }) });
        await refresh();
      });
    };

    const deleteKey = async (k) => {
      if (!await confirmDelete(t('confirm.deleteKey', { id: k.id }), k.id)) return;
      await runAction(async () => {
        await api(`keys/${k.id}`, { method: 'DELETE' });
        await refresh();
      });
    };

    const toggleGatewayKey = async (k) => {
      await runAction(async () => {
        await api(`gateway-keys/${k.id}`, { method: 'PATCH', body: JSON.stringify({ enabled: !k.enabled }) });
        await refresh();
      });
    };

    const rotateGatewayKey = async (k) => {
      if (!await openConfirmDialog(t('confirm.rotateGatewayKey', { id: k.id }), t('confirm.rotateTitle'))) return;
      await runAction(async () => {
        const item = await api(`gateway-keys/${k.id}/rotate`, { method: 'POST', body: '{}' });
        createdGatewayKey.value = { ...item, event: 'rotated' };
        await refresh();
      }, refreshError, t('feedback.gatewayKeyRotated'));
    };

    const deleteGatewayKey = async (k) => {
      if (!await confirmDelete(t('confirm.deleteGatewayKey', { id: k.id }), k.id)) return;
      await runAction(async () => {
        await api(`gateway-keys/${k.id}`, { method: 'DELETE' });
        await refresh();
      });
    };

    const confirmDelete = (message, id) => openConfirmDialog(message, t('confirm.deleteTitle'), id);

    const preferKey = async (k) => {
      await runAction(async () => {
        await api(`keys/${k.id}/prefer`, { method: 'POST', body: '{}' });
        await refresh();
      });
    };

    const resetKey = async (k) => {
      await runAction(async () => {
        await api(`keys/${k.id}/reset`, { method: 'POST', body: '{}' });
        await refresh();
      });
    };

    onMounted(async () => {
      setLang(lang.value);
      setTheme(theme.value);
      activeSection.value = sectionFromHash();
      window.addEventListener('hashchange', syncSectionFromHash);
      window.addEventListener('popstate', syncSectionFromHash);
      if (adminToken.value) await refresh();
      if (adminToken.value && activeSection.value === 'logs') {
        await loadLogs(0);
        startLogAutoRefresh();
      }
    });

    onUnmounted(() => {
      window.removeEventListener('hashchange', syncSectionFromHash);
      window.removeEventListener('popstate', syncSectionFromHash);
      copyTimers.forEach((timer) => window.clearTimeout(timer));
      copyTimers.clear();
      stopLogAutoRefresh();
    });

    return {
      activeSection,
      actionBusy,
      authError,
      authForm,
      authRequired,
      authSubmitting,
      currentTitle,
      dashboard,
      adminToken,
      balances,
      balanceRefreshing,
      balanceResults,
      cancelKeyForm,
      cancelModelRouteForm,
      cancelProviderForm,
      copyButtonClass,
      copyButtonText,
      copyText,
      createGatewayKey,
      createdGatewayKey,
      formError,
      formatGatewayTime,
      formatShortTime,
      logModelLabel,
      logProtocolLabel,
      logStatusClass,
      logStatusLabel,
      gatewayKeyForm,
      gatewayKeys,
      keyForm,
      keyFormOpen,
      keys,
      lang,
      logs,
      logPage,
      logFilters,
      logsLoading,
      logsAutoRefresh,
      syncLogsAutoRefresh,
      loadLogs,
      exportLogs,
      clearLogs,
      confirmDialog,
      confirmDialogElement,
      dismissNotice,
      finishConfirmDialog,
      handleConfirmDialogKeydown,
      logoutAdmin,
      metricCards,
      navItems,
      notices,
      openKeyForm,
      openProviderForm,
      openSetup,
      openSetupStep,
      port,
      pretty,
      providerKeyHints,
      providerKeys,
      providerDisplayName,
      keyDisplayName,
      hasDistinctProviderName,
      hasDistinctKeyName,
      providerModelOptions,
      providerTypeLabel,
      modelMapDefault,
      modelDiscovery,
      modelRouteForm,
      modelRouteFormOpen,
      modelRoutes,
      routeProviderSelected,
      modelCoolingCount,
      modelProviderCooling,
      modelHasFailureState,
      resetModelState,
      modelStates,
      openModelRouteForm,
      addRouteModel,
      removeRouteModel,
      addRouteTarget,
      removeRouteTarget,
      saveModelRoute,
      toggleModelRoute,
      deleteModelRoute,
      providerModelPreview,
      providerDiscoveredModels,
      providerModelPolicyOptions,
      filteredProviderModelPolicyOptions,
      modelPolicyDraft,
      modelPolicyDrafts,
      modelPolicySearch,
      modelPolicySelected,
      modelPolicySavingProviderId,
      isFlagshipModel,
      modelOptionLabel,
      setModelPolicyEnabled,
      setProviderModelSelected,
      selectProviderModels,
      saveProviderModelPolicy,
      discoveryForProvider,
      discoveryStatusClass,
      discoveryStatusText,
      refreshProviderModels,
      refreshingProviderId,
      providerApiEndpoint,
      providerBalanceText,
      balanceStatusClass,
      balanceStatusText,
      balanceValueClass,
      displayBalanceValue,
      providerBalanceClass,
      resultStatusClass,
      formatBalanceValue,
      providerForm,
      providerFormOpen,
      providerTypes,
      providers,
      recentLogs,
      refresh,
      refreshError,
      refreshing,
      refreshBalances,
      routing,
      routingAdvanced,
      routingMessage,
      saveRouting,
      maintenanceMessage,
      backupPassphrase,
      backupPassphraseConfirm,
      checkIntegrity,
      downloadBackup,
      downloadPortableBackup,
      saveKey,
      saveProvider,
      setActiveSection,
      setLang,
      serviceReadinessClass,
      serviceReadinessLabel,
      serviceReadinessReason,
      setupComplete,
      setupSteps,
      setupNeedsGatewayKey,
      setupSnippets,
      snippets,
      submitAdminLogin,
      t,
      theme,
      toggleTheme,
      testKey,
      testAllKeys,
      testingAllKeys,
      bulkTestSummary,
      testResultText,
      testStatusClass,
      testConnectionText,
      testErrorText,
      tokenUsageText,
      capabilitiesText,
      keyTestResults,
      testingKeyId,
      toggleProvider,
      deleteProvider,
      editProvider,
      editKey,
      toggleKey,
      deleteKey,
      toggleGatewayKey,
      rotateGatewayKey,
      deleteGatewayKey,
      preferKey,
      resetKey,
      modelMapExample,
    };
  },
}).mount('#app');
