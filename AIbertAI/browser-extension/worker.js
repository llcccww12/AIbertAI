const host = 'com.aibertai.browser';
const tools = ['browser_tabs', 'browser_read', 'browser_focus_tab'];
const grants = new Map();
let port;
let connected = false;
let shared;

async function readPage(tabId) {
  const [frame] = await chrome.scripting.executeScript({
    target: { tabId },
    func: () => ({
      url: location.href,
      title: document.title.slice(0, 1024),
      text: (document.body?.innerText ?? '').slice(0, 32000),
    }),
  });
  const page = frame?.result;
  const url = new URL(page?.url);
  if (!['https:', 'http:'].includes(url.protocol) || url.username || url.password ||
      page.url.length > 8192 || !frame.documentId) throw Error('invalid_browser_url');
  return { ...page, documentVersion: frame.documentId };
}

function forget(reason) {
  for (const grantId of grants.keys()) {
    port?.postMessage({ type: 'event', event: { kind: 'tab', grantId, reason } });
  }
  grants.clear();
  shared = undefined;
  port?.postMessage({ type: 'event', event: { kind: 'shared_changed' } });
}

function stop() {
  const previous = port;
  port = undefined;
  connected = false;
  grants.clear();
  shared = undefined;
  previous?.disconnect();
}

async function connect() {
  if (port) return;
  const installationId = (await chrome.storage.local.get('installationId')).installationId ?? crypto.randomUUID();
  await chrome.storage.local.set({ installationId });
  if (port) return;
  const connection = chrome.runtime.connectNative(host);
  port = connection;
  connection.onDisconnect.addListener(() => {
    void chrome.runtime.lastError;
    if (port === connection) stop();
  });
  connection.onMessage.addListener((message) => {
    if (port !== connection) return;
    if (message.type === 'welcome' && message.protocol === 4) connected = true;
    else if (message.type === 'rejected') stop();
    else if (connected && message.type === 'request' && typeof message.id === 'string') {
      void reply(connection, message);
    }
  });
  connection.postMessage({ type: 'event', event: {
    kind: 'hello', browser: 'Chrome / Edge', version: chrome.runtime.getManifest().version,
    protocol: 4, minProtocol: 4, installationId, tools,
  } });
}

async function sharedPage() {
  const selection = shared;
  if (!selection) throw Error('tab_not_granted');
  const current = await readPage(selection.tabId);
  if (shared !== selection || current.documentVersion !== selection.page.documentVersion ||
      current.url !== selection.page.url) throw Error('tab_not_granted');
  const { text, ...page } = current;
  return { text, page };
}

async function focusSharedPage() {
  const selection = shared;
  if (!selection) throw Error('tab_not_granted');
  const tab = await chrome.tabs.get(selection.tabId);
  if (shared !== selection) throw Error('tab_not_granted');
  await chrome.windows.update(tab.windowId, { focused: true });
  await chrome.tabs.update(tab.id, { active: true });
}

async function request(input) {
  if (input.kind === 'clear_shared') {
    forget('user_revoked');
    return {};
  }
  if (input.kind === 'shared_page') {
    const result = await sharedPage();
    if (input.action === 'focus') {
      await focusSharedPage();
      return { page: result.page };
    }
    if (input.action === 'read') return result;
    throw Error('unsupported_capability');
  }
  if (input.kind === 'list') {
    return { data: shared ? [{ tabId: shared.tabId, url: shared.page.url, title: shared.page.title }] : [] };
  }
  if (input.kind === 'revoke') {
    grants.delete(input.grantId);
    return {};
  }
  if (input.kind === 'grant' || input.kind === 'resume') {
    const grant = input.grant;
    if (!shared || grant?.tabId !== shared.tabId || typeof grant.id !== 'string' ||
        !Number.isSafeInteger(grant.revision) || (grants.size && !grants.has(grant.id))) {
      throw Error('tab_not_granted');
    }
    const { page } = await sharedPage();
    grants.set(grant.id, { revision: grant.revision, paused: false });
    return { page };
  }
  const grant = grants.get(input.grantId);
  if (!grant || !shared) throw Error('tab_not_granted');
  if (input.kind === 'pause') {
    grant.paused = true;
    return {};
  }
  if (input.kind === 'focus') {
    await sharedPage();
    await focusSharedPage();
    return {};
  }
  if (input.kind === 'cancel') return {};
  if (input.kind !== 'execute' || input.tool !== 'browser_read') throw Error('unsupported_capability');
  if (grant.paused || input.revision !== grant.revision || Date.now() >= input.deadline ||
      (input.expiresAt && Date.now() >= input.expiresAt)) throw Error('stale_grant');
  if (input.args?.frameId) throw Error('unsupported_capability');
  const { text, page } = await sharedPage();
  if (grants.get(input.grantId) !== grant || grant.paused || Date.now() >= input.deadline) {
    throw Error('stale_grant');
  }
  const offset = Math.min(text.length, Math.max(0, input.args?.offset ?? 0));
  const end = Math.min(text.length, offset + Math.min(32000, Math.max(1, input.args?.limit ?? 12000)));
  return { page, data: { text: text.slice(offset, end), truncated: end < text.length,
    ...(end < text.length ? { nextOffset: end } : {}) } };
}

async function reply(connection, message) {
  let result;
  try {
    result = { result: await request(message.request) };
  } catch (err) {
    const code = ['invalid_browser_url', 'tab_not_granted', 'unsupported_capability', 'stale_grant'].includes(err.message)
      ? err.message : 'browser_operation_failed';
    result = { error: { code } };
  }
  if (port === connection) connection.postMessage({ type: 'response', id: message.id, ...result });
}

chrome.tabs.onRemoved.addListener((id) => {
  if (shared?.tabId === id) forget('tab_closed');
});
chrome.tabs.onUpdated.addListener((id, change) => {
  if (shared?.tabId === id && (change.status === 'loading' || change.url)) forget('tab_navigated');
});

chrome.runtime.onMessage.addListener((message, sender, respond) => {
  if (sender.url !== chrome.runtime.getURL('popup.html')) return false;
  void (async () => {
    if (message.action === 'connect') await connect();
    else if (message.action === 'disconnect') stop();
    else if (message.action === 'clear') forget('user_revoked');
    else if (message.action === 'share') {
      if (!connected) throw Error('请先连接桌面端');
      const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
      if (!tab?.id) throw Error('找不到当前标签页');
      const connection = port;
      const { documentVersion, url, title } = await readPage(tab.id);
      if (!connected || connection !== port) throw Error('桌面端已断开');
      forget('user_revoked');
      shared = { tabId: tab.id, page: { documentVersion, url, title } };
    }
    return { connected, connecting: !!port && !connected, title: shared ? shared.page.title || shared.page.url : undefined };
  })().then(respond, () => respond({ error: '无法分享此页面。请确认已连接，并打开普通网页后重试。' }));
  return true;
});
