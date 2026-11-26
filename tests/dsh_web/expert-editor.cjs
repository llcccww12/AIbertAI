const assert = require('node:assert/strict');
const { chromium } = require(process.env.DSH_PLAYWRIGHT_CORE || '../../AIbertAI/code/app/node_modules/playwright-core');

// Run with TestDSHWebFixture and DSH_WEB_EXPERT_EDITOR=1.
async function run() {
  const base = process.env.DSH_WEB_BASE || 'http://127.0.0.1:18767';
  const browser = await chromium.launch({ executablePath: process.env.DSH_CHROME, headless: true });
  const context = await browser.newContext({ locale: 'zh-CN', viewport: { width: 1280, height: 850 } });
  const page = await context.newPage();
  const writes = [];
  page.on('websocket', socket => socket.on('framesent', ({ payload }) => {
    const frame = JSON.parse(payload);
    if (frame.method === '_monkeyai/resources/save') writes.push(frame.params);
  }));
  const expertName = '画布插件验收专家';
  const editor = page.locator('.expert-dialog');
  const pluginsDefault = editor.getByRole('checkbox', { name: '使用默认组合' });
  const canvasDefault = editor.getByRole('checkbox', { name: '沿用专家默认画布工具' });
  const pluginsTab = editor.getByRole('tab', { name: /^插件/ });
  const canvasTab = editor.getByRole('tab', { name: /^画布/ });
  const skillTab = editor.getByRole('tab', { name: /^技能/ });
  const save = async () => {
    await editor.getByRole('button', { name: '保存', exact: true }).click();
    await editor.waitFor({ state: 'hidden' });
    return writes.at(-1).input;
  };
  const reopen = async () => {
    await page.getByRole('button', { name: expertName, exact: true }).click();
    await page.waitForFunction(name => document.querySelector('#server-expert-name')?.value === name, expertName);
  };
  try {
    await page.goto(base + '/settings/experts');
    const create = page.getByRole('button', { name: '新建专家', exact: true });
    const login = page.getByRole('button', { name: '登录', exact: true });
    await login.or(create).first().waitFor();
    if (await login.isVisible()) {
      await page.locator('input').first().fill(base);
      await login.click();
      await page.waitForURL(url => url.pathname !== '/login');
      await page.goto(base + '/settings/experts');
    }
    await create.click();
    await page.getByRole('button', { name: /手动创建专家/ }).click();
    await editor.getByLabel('专家名称').fill(expertName);
    await editor.getByLabel('角色设定').fill('制作配图并交付画布。');
    assert.equal(await pluginsDefault.getAttribute('aria-checked'), 'true');
    let input = await save();
    assert.equal(input.dsh_plugins, null);
    assert.equal(input.canvas_tools, null);

    await reopen();
    assert.equal(await editor.getByRole('group', { name: '会话插件' }).getByRole('checkbox').count(), 20);
    const fs = editor.getByRole('checkbox', { name: /文件读写/ });
    await fs.click();
    assert.equal(await pluginsDefault.getAttribute('aria-checked'), 'false');
    await canvasTab.click();
    assert.equal(await canvasDefault.getAttribute('aria-checked'), 'true');
    assert.equal(await editor.getByRole('group', { name: '画布工具' }).getByRole('checkbox').count(), 12);
    await editor.getByRole('checkbox', { name: /生成图片/ }).click();
    assert.equal(await canvasDefault.getAttribute('aria-checked'), 'false');
    await skillTab.click();
    await editor.getByRole('checkbox', { name: /fixture-drawing/ }).click();
    await editor.getByRole('button', { name: '启用技能加载工具' }).click();
    input = await save();
    assert.deepEqual(input.dsh_plugins, ['tool-fs', 'tool-skill']);
    assert.deepEqual(input.canvas_tools, ['media_generate_image']);
    assert.deepEqual(input.skill_ids, ['88888888-8888-4888-8888-888888888888']);

    await reopen();
    await canvasTab.click();
    const image = editor.getByRole('checkbox', { name: /生成图片/ });
    const placeImage = editor.getByRole('checkbox', { name: /将生成的图片放上画布/ });
    assert.equal(await image.getAttribute('aria-checked'), 'true');
    assert.equal(await placeImage.getAttribute('aria-checked'), 'false');
    await image.click();
    await placeImage.click();
    input = await save();
    assert.deepEqual(input.canvas_tools, ['canvas_create_image']);
    assert.equal(Object.hasOwn(input, 'dsh_plugins'), false);

    await reopen();
    assert.equal(await fs.getAttribute('aria-checked'), 'true');
    await fs.click();
    await editor.getByRole('checkbox', { name: /按需加载技能/ }).click();
    await canvasTab.click();
    assert.equal(await image.getAttribute('aria-checked'), 'false');
    assert.equal(await placeImage.getAttribute('aria-checked'), 'true');
    await placeImage.click();
    input = await save();
    assert.deepEqual(input.dsh_plugins, []);
    assert.deepEqual(input.canvas_tools, []);

    await reopen();
    assert.equal(await pluginsDefault.getAttribute('aria-checked'), 'false');
    assert.equal(await fs.getAttribute('aria-checked'), 'false');
    await editor.getByLabel('简介（选填）').fill('仅修改简介');
    input = await save();
    assert.equal(Object.hasOwn(input, 'dsh_plugins'), false);
    assert.equal(Object.hasOwn(input, 'canvas_tools'), false);

    await reopen();
    await canvasTab.click();
    assert.equal(await canvasDefault.getAttribute('aria-checked'), 'false');
    await canvasDefault.click();
    await pluginsTab.click();
    await pluginsDefault.click();
    input = await save();
    assert.equal(input.dsh_plugins, null);
    assert.equal(input.canvas_tools, null);
    assert.equal(writes.length, 6);
    assert(writes.slice(1).every(request => request.id && request.etag));
    console.log('Employee expert editor passed: default, custom, empty and restored presets; linked skill; independent image generation and placement; conditional updates.');
  } finally {
    await context.request.post(base + '/fixture/finish').catch(() => {});
    await browser.close();
  }
}

run().catch(error => { console.error(error); process.exitCode = 1; });

