// Run with the desktop's Electron executable and the packaged canvas worker path.
const { app, utilityProcess } = require("electron");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const os = require("node:os");
const { randomUUID, createHash } = require("node:crypto");
const temp = fs.mkdtempSync(path.join(os.tmpdir(), "dsh-guarded-image-"));
app.setPath("userData", temp);
const workerPath = path.resolve(process.argv[2]);
const hash = data => createHash("sha256").update(data).digest("hex");

async function run(child) {
  const pending = new Map();
  child.on("message", message => {
    if (message.type !== "private-catalog-result") return;
    const finish = pending.get(message.id);
    pending.delete(message.id);
    finish?.(message);
  });
  function invoke(tool, args) {
    const id = randomUUID();
    return new Promise((resolve, reject) => {
      const timeout = setTimeout(() => reject(Error("Worker timed out: " + tool)), 12000);
      pending.set(id, message => { clearTimeout(timeout); resolve(message); });
      child.postMessage({ type: "private-catalog", id, tool, args });
    });
  }
  const cwd = path.join(temp, "workspace"), sessionId = randomUUID();
  fs.mkdirSync(cwd);
  child.postMessage({ type: "image-preflight", result: {state: "passed"} });
  child.postMessage({ type: "init-private-catalog", origin: "http://127.0.0.1:9" });
  const capabilities = await invoke("design_guard_capabilities", {});
  assert.ok(capabilities.data.tools.includes("canvas_create_image"));
  const opened = await invoke("design_session_open", {cwd, sessionId, selection: {
    designType: "image", designTypeSource: "user", templateRef: null, designSystemRef: null, mediaModelId: null,
  }});
  assert.ok(opened.data, opened.error);
  const design = opened.data, root = fs.realpathSync(path.join(cwd, design.localDesignPath));
  const file = path.join(root, "canvas.json"), assets = path.join(root, "assets/images");
  const contextSha = hash(fs.readFileSync(path.join(root, "design-context.json")));
  const read = () => JSON.parse(fs.readFileSync(file));
  const guard = () => ({ expectedRevision: read().revision ?? 0, expectedCanvasDigest: hash(fs.readFileSync(file)),
    expectedArtifactsDigest: hash(""), expectedContextDigest: contextSha });
  const directory = path.join(cwd, ".aibertai/media/generated");
  fs.mkdirSync(directory, {recursive: true});
  const sharp = require(require.resolve("sharp", {paths:[path.dirname(workerPath)]}));
  async function fixture(format) {
    const source_path = ".aibertai/media/generated/" + randomUUID() + "." + format;
    const bytes = await sharp({create:{width:4, height:2, channels:3, background:"#f03800"}}).toFormat(format).toBuffer();
    fs.writeFileSync(path.join(cwd,source_path),bytes);
    return {source_path, bytes};
  }
  const first = await fixture("png");
  function submit(image, input = {}, expected = guard(), digest = hash(image.bytes)) {
    return invoke("design_model_tool", { cwd, sessionId, design, name:"canvas_create_image",
      args:{source_path:image.source_path,...input}, ...expected, expectedSourceDigest:digest });
  }
  const helpers = require(path.join(path.dirname(workerPath), "private-guarded-image.cjs"));
  const snapshot = helpers.readGeneratedImage(cwd, {source_path:first.source_path}, hash(first.bytes), 30 * 1024 * 1024);
  const firstPath = path.join(cwd, first.source_path);
  fs.renameSync(firstPath, firstPath + ".previous");
  fs.writeFileSync(firstPath, first.bytes);
  assert.throws(snapshot.verify, /canvas_image_source_changed/);
  const verified = helpers.readGeneratedImage(cwd, {source_path:first.source_path}, hash(first.bytes), 30 * 1024 * 1024);
  assert.throws(() => helpers.withCanvasImageAsset(root, verified, (_, verifyAsset) => {
    fs.writeFileSync(firstPath, "changed while preparing the node");
    verifyAsset();
  }), /canvas_image_source_changed/);
  fs.unlinkSync(firstPath);
  fs.renameSync(firstPath + ".previous", firstPath);
  assert.deepEqual(fs.readdirSync(assets),[]);
  const start = guard();
  const refused = async (reply) => { const {error} = await reply; assert.ok(error, "Worker accepted invalid image"); };
  for (const extra of [{anchor_node_id:"fake"}, {placement_node_id:"fake"}, {x:12}, {title:"   "},
    {source_path:"../outside.png"}, {source_path:path.join(cwd,first.source_path)}]) await refused(submit(first,extra));
  await refused(submit(first,{},start,"0".repeat(64)));
  const sourceFile = path.join(cwd,first.source_path);
  fs.renameSync(sourceFile,sourceFile+".original");
  fs.symlinkSync(sourceFile+".original",sourceFile);
  await refused(submit(first)); fs.unlinkSync(sourceFile); fs.renameSync(sourceFile+".original",sourceFile);
  fs.renameSync(directory,directory+".original"); fs.symlinkSync(directory+".original",directory,"dir");
  await refused(submit(first)); fs.unlinkSync(directory); fs.renameSync(directory+".original",directory);
  fs.renameSync(assets,assets+"-original"); fs.symlinkSync(assets+"-original",assets,"dir");
  await refused(submit(first)); fs.unlinkSync(assets); fs.renameSync(assets+"-original",assets);
  assert.deepEqual(guard(),start);
  assert.deepEqual(fs.readdirSync(assets),[]);
  const keeper = await submit(first,{title:"校对图"});
  assert.ok(keeper.data,keeper.error);
  const saved = read().nodes.find(node => node.id === keeper.data.node.id);
  assert.equal(saved.type,"image"); assert.equal(saved.title,"校对图");
  assert.equal(saved.width,4); assert.equal(saved.height,2);
  assert.deepEqual(fs.readFileSync(path.join(root,saved.source)),first.bytes);
  await refused(submit(first,{},start));
  const second = await fixture("webp");
  const same = guard(), calls = await Promise.all([submit(second,{},same), submit(second,{},same)]);
  assert.equal(calls.filter(row => row.data).length,1);
  assert.equal(calls.filter(row => row.error === "canvas_revision_conflict").length,1);
  const last = await fixture("jpeg");
  assert.ok((await submit(last)).data);
  assert.deepEqual(read().nodes.find(node => node.id === saved.id),saved);
  assert.equal(fs.readdirSync(assets).length,3);
  assert.equal(read().revision, start.expectedRevision+3);
  console.log("Guarded canvas worker: PNG/JPEG/WebP nodes, original bytes, scope, symlinks, stale inputs and concurrent revision passed.");
}

app.whenReady().then(async () => {
  const child = utilityProcess.fork(workerPath, [], {stdio:"pipe"});
  child.stderr.on("data", data => process.stderr.write(data));
  try {
    await run(child);
    app.exit(0);
  } catch (error) {
    console.error(error);
    app.exit(1);
  } finally {
    child.kill();
    fs.rmSync(temp,{recursive:true,force:true});
  }
});
