const fs = require("node:fs");
const path = require("node:path");
const { createHash, randomUUID } = require("node:crypto");

const generatedPath = /^\.aibertai\/media\/generated\/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\.(png|jpe?g|webp)$/;
const sha256 = (data) => createHash("sha256").update(data).digest("hex");
const sameFile = (a, b) => a.dev === b.dev && a.ino === b.ino;
const stat = (file) => fs.lstatSync(file, { bigint: true });

function validateInput(input, expectedDigest) {
  if (!input || typeof input !== "object" || Array.isArray(input) ||
      Object.keys(input).some(key => key !== "source_path" && key !== "title") ||
      typeof input.source_path !== "string" || !generatedPath.test(input.source_path) ||
      (input.title !== undefined && (typeof input.title !== "string" || !input.title.trim() || [...input.title].length > 160)) ||
      typeof expectedDigest !== "string" || !/^[a-f0-9]{64}$/.test(expectedDigest)) {
    throw new Error("invalid_canvas_image_source");
  }
}

function readGeneratedImage(workspace, input, expectedDigest, maxBytes) {
  validateInput(input, expectedDigest);
  const root = fs.realpathSync(workspace);
  const file = path.join(root, input.source_path);
  function checkPath() {
    if (fs.realpathSync(workspace) !== root || fs.realpathSync(file) !== file) throw new Error("invalid_canvas_image_source");
    const source = stat(file);
    if (!source.isFile() || source.nlink !== 1n || source.size < 1n || source.size > maxBytes) throw new Error("invalid_canvas_image_source");
    return source;
  }
  const initial = checkPath();
  function verify(current) {
    if (!sameFile(initial, current) || current.size !== initial.size || initial.ctimeNs !== current.ctimeNs) {
      throw new Error("canvas_image_source_changed");
    }
  }
  const fd = fs.openSync(file, fs.constants.O_RDONLY | fs.constants.O_NOFOLLOW | fs.constants.O_NONBLOCK);
  try {
    verify(fs.fstatSync(fd, { bigint: true }));
    const bytes = Buffer.alloc(Number(initial.size));
    for (let offset = 0; offset < bytes.length;) {
      const count = fs.readSync(fd, bytes, offset, bytes.length - offset);
      if (!count) throw new Error("canvas_image_source_changed");
      offset += count;
    }
    if (sha256(bytes) !== expectedDigest) throw new Error("canvas_image_source_changed");
    verify(checkPath());
    return { bytes, extension: path.extname(file), verify: () => verify(checkPath()) };
  } finally {
    fs.closeSync(fd);
  }
}

// Run the existing synchronous canvas transaction with an exclusive new asset.
// A pending journal may still need the bytes if a durable commit is interrupted.
function withCanvasImageAsset(root, image, commit) {
  const images = path.join(root, "assets", "images");
  const history = path.join(root, "history");
  for (const dir of [images, history]) {
    if (fs.realpathSync(dir) !== dir || !stat(dir).isDirectory()) throw new Error("canvas_storage_unsafe");
  }
  image.verify();
  const source = "assets/images/" + randomUUID() + image.extension;
  const file = path.join(root, source);
  const fd = fs.openSync(file, "wx", 0o600);
  const owned = fs.fstatSync(fd, { bigint: true });
  try {
    fs.writeFileSync(fd, image.bytes);
    fs.fsyncSync(fd);
    const written = fs.fstatSync(fd, { bigint: true });
    function verifyAsset() {
      const current = stat(file);
      if (fs.realpathSync(file) !== file || !sameFile(owned, current) || current.ctimeNs !== written.ctimeNs ||
          fs.realpathSync(history) !== history) throw new Error("canvas_storage_unsafe");
      image.verify();
    }
    verifyAsset();
    return commit(source, verifyAsset);
  } catch (error) {
    if (fs.existsSync(file) && fs.realpathSync(file) === file && sameFile(owned, stat(file)) &&
        !fs.existsSync(path.join(history, "pending.json"))) {
      // A directory-sync failure after commit can also leave a durable reference.
      const canvas = JSON.parse(fs.readFileSync(path.join(root, "canvas.json"), "utf8"));
      if (!canvas.nodes.some(node => node.source === source)) fs.unlinkSync(file);
    }
    throw error;
  } finally {
    fs.closeSync(fd);
  }
}

module.exports = { validateInput, readGeneratedImage, withCanvasImageAsset };
