const status = document.getElementById('status');
const buttons = Object.fromEntries(['connect', 'share', 'clear', 'disconnect'].map(id => [id, document.getElementById(id)]));
let updateTimer;
async function run(action) {
  clearTimeout(updateTimer);
  try {
    const state = await chrome.runtime.sendMessage({ action });
    if (state.error) {
      status.textContent = state.error;
      return;
    }
    buttons.connect.disabled = state.connected || state.connecting;
    buttons.disconnect.disabled = !state.connected && !state.connecting;
    buttons.share.disabled = !state.connected;
    buttons.clear.disabled = !state.title;
    status.textContent = state.connecting ? '正在连接…' : !state.connected
      ? '请在桌面端的「设置 → 浏览器」启用联动。'
      : state.title ? '已分享：' + state.title : '已连接。请选择要分享的标签页。';
    if (state.connecting) updateTimer = setTimeout(() => void run('status'), 200);
  } catch {
    status.textContent = '暂时无法连接 AIbertAI。';
  }
}
for (const [action, button] of Object.entries(buttons)) button.onclick = () => void run(action);
void run('status');
