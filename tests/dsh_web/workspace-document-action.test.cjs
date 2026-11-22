const test = require('node:test')
const assert = require('node:assert/strict')
const fs = require('node:fs/promises')
const path = require('node:path')
const os = require('node:os')
const { workspaceDocumentAction, defaultApplication } = require('../../AIbertAI/code/app/out/main/workspace-document-action.cjs')

test('workspace documents dispatch only authorized local files to the native shell', async t => {
  const base = await fs.mkdtemp(path.join(os.tmpdir(), 'employee-document-'))
  t.after(() => fs.rm(base, { recursive: true, force: true }))
  const cwd = path.join(base, 'session')
  await fs.mkdir(cwd)
  const written = path.join(cwd, '难忘的一天.md')
  await fs.writeFile(written, '# 难忘的一天\n正文\n')
  const document = Buffer.from([80, 75, 3, 4, 0, 1, 2])
  await fs.writeFile(path.join(cwd, '难忘的一天.docx'), document)
  const sent = [], checked = []
  let current = true
  const options = { cwd, assertCurrent: () => { if (!current) throw Error('device changed') },
    preferred: async () => ({ id: 'default', name: 'Fixture Reader', icon: null }),
    verifyOffice: async (bytes, ext) => { assert.deepEqual(bytes, document); checked.push(ext) },
    openPath: async file => { sent.push(['open', file]); return '' },
    showItemInFolder: file => sent.push(['reveal', file]) }
  const act = (relative, action, more = {}) => workspaceDocumentAction({ ...options, ...more, relative, action })
  assert.equal((await act('难忘的一天.md', 'inspect')).application.name, 'Fixture Reader')
  await act('难忘的一天.md', 'open')
  await act('难忘的一天.md', 'reveal')
  await act('难忘的一天.docx', 'open')
  assert.deepEqual(checked, ['docx'])
  assert.deepEqual(sent.map(call => call[0]), ['open', 'reveal', 'open'])
  assert.equal(sent[0][1], await fs.realpath(written))
  assert.equal(await defaultApplication(null, written), null)
  const count = sent.length
  for (const name of ['test.js','runner.sh','binary.md','script.md','coded.md','wrong.docx']) {
    await fs.writeFile(path.join(cwd, name), name === 'binary.md' ? Buffer.from([0, 15, 7]) : name === 'script.md' ? '#!/bin/false\n' : 'text')
  }
  if (process.platform !== 'win32') {
    await fs.chmod(path.join(cwd, 'coded.md'), 0o755)
    await assert.rejects(act('coded.md', 'open'))
  }
  await fs.mkdir(path.join(cwd, 'folder.md'))
  await fs.writeFile(path.join(base, 'foreign.md'), 'another session')
  await fs.symlink(path.join(base, 'foreign.md'), path.join(cwd, 'link.md'))
  for (const name of ['../foreign.md', path.join(base, 'foreign.md'), 'link.md', 'folder.md',
    'test.js', 'runner.sh', 'binary.md', 'script.md', 'wrong.docx', 'missing.md']) {
    await assert.rejects(act(name, 'open'), undefined, name)
  }
  await assert.rejects(act('难忘的一天.md', 'unsupported'))
  await assert.rejects(act('难忘的一天.docx','open',{verifyOffice:async()=>{throw Error('invalid document')}}))
  current = false
  await assert.rejects(act('难忘的一天.md','reveal'), /device changed/)
  assert.equal(sent.length, count)
})
