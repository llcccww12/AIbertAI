"use strict";
var __create = Object.create;
var __defProp = Object.defineProperty;
var __getOwnPropDesc = Object.getOwnPropertyDescriptor;
var __getOwnPropNames = Object.getOwnPropertyNames;
var __getProtoOf = Object.getPrototypeOf;
var __hasOwnProp = Object.prototype.hasOwnProperty;
var __copyProps = (to, from, except, desc) => {
  if (from && typeof from === "object" || typeof from === "function") {
    for (let key of __getOwnPropNames(from))
      if (!__hasOwnProp.call(to, key) && key !== except)
        __defProp(to, key, { get: () => from[key], enumerable: !(desc = __getOwnPropDesc(from, key)) || desc.enumerable });
  }
  return to;
};
var __toESM = (mod, isNodeMode, target) => (target = mod != null ? __create(__getProtoOf(mod)) : {}, __copyProps(
  // If the importer is in node compatibility mode or this is not an ESM
  // file that has been converted to a CommonJS file using a Babel-
  // compatible transform (i.e. "__esModule" has not been set), then set
  // "default" to the CommonJS "module.exports" for node compatibility.
  isNodeMode || !mod || !mod.__esModule ? __defProp(target, "default", { value: mod, enumerable: true }) : target,
  mod
));
Object.defineProperty(exports, Symbol.toStringTag, { value: "Module" });
const node_fs = require("node:fs");
const node_os = require("node:os");
const node_path = require("node:path");
const node_crypto = require("node:crypto");
const index = require("../index.js");
require("node:http");
require("@modelcontextprotocol/sdk/server/index.js");
require("@modelcontextprotocol/sdk/server/streamableHttp.js");
require("@modelcontextprotocol/sdk/types.js");
require("path");
require("electron");
require("node:fs/promises");
require("node:url");
require("node:child_process");
require("node:util");
require("node:events");
require("node:net");
require("node:stream");
require("node:stream/promises");
require("node:timers/promises");
require("node:https");
require("electron-updater");
require("js-yaml");
require("semver");
require("electron-updater/out/providers/Provider");
function previewBrowserExecutable(platform = process.platform, env = process.env, home = node_os.homedir()) {
  if (env.MONKEYAI_PREVIEW_BROWSER) {
    if (!node_fs.existsSync(env.MONKEYAI_PREVIEW_BROWSER)) throw index.previewError("canvas_preview_browser_not_found");
    return env.MONKEYAI_PREVIEW_BROWSER;
  }
  const candidates = platform === "win32" ? [env.PROGRAMFILES, env["PROGRAMFILES(X86)"], env.LOCALAPPDATA].filter((root) => !!root).flatMap((root) => [node_path.join(root, "Microsoft/Edge/Application/msedge.exe"), node_path.join(root, "Google/Chrome/Application/chrome.exe")]) : platform === "darwin" ? ["/Applications/Google Chrome.app/Contents/MacOS/Google Chrome", "/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge", node_path.join(home, "Applications/Google Chrome.app/Contents/MacOS/Google Chrome")] : ["/usr/bin/google-chrome", "/usr/bin/google-chrome-stable", "/usr/bin/microsoft-edge", "/usr/bin/chromium", "/usr/bin/chromium-browser", "/snap/bin/chromium"];
  const executable = candidates.find((path) => node_fs.existsSync(path));
  if (!executable) throw index.previewError("canvas_preview_browser_not_found");
  return executable;
}
async function renderBrowserPreview(input) {
  if (Number(process.versions.node.split(".")[0]) < 20) throw index.previewError("canvas_preview_browser_runtime_unsupported");
  if (Buffer.byteLength(input.content) > index.MAX_RENDER_BYTES) throw index.previewError(index.ErrorCode.invalidHtmlPreview);
  const task = index.createPreviewTask(5e4, input.signal);
  const base = index.createPreviewTrace(input.requestId, "independent-browser", input.log);
  const trace = { ...base, step: (stage, run) => base.step(stage, () => task.run(run)) };
  try {
    await index.desktopImages().ensure({ signal: task.signal });
    const executable = previewBrowserExecutable();
    const { chromium } = await trace.step("browser-driver", () => import("playwright-core"));
    for (let attempt = 0; attempt < 2; attempt++) {
      task.check();
      let retryable = true;
      try {
        return await browserAttempt(input, chromium, executable, attempt === 1, task.signal, trace, (stage) => {
          retryable = stage !== "fonts";
        });
      } catch (error) {
        task.check();
        const code = error?.code;
        if (attempt || !retryable || index.imageCapabilityFailure(error) || code?.startsWith("image_") || code === "canvas_preview_browser_cleanup_failed") throw error;
        trace.event("browser-software-retry", "selected");
      }
    }
    throw index.previewError(index.ErrorCode.htmlPreviewUnavailable);
  } finally {
    task.close();
    trace.close();
  }
}
async function browserAttempt(input, chromium, executable, software, signal, trace, stage) {
  const task = index.createPreviewTask(24e3, signal, "canvas_preview_browser_attempt_timeout");
  const step = (name, run) => trace.step(name, () => task.run(run));
  let server;
  let killing;
  let launch;
  const kill = () => {
    if (server && !killing) {
      killing = server.kill();
      void killing.catch(() => {
        trace.event("browser-process-cleanup", "error");
      });
    }
  };
  task.signal.addEventListener("abort", kill, { once: true });
  try {
    task.check();
    stage("launch");
    trace.event("browser-attempt", "start", { software });
    const env = { ...process.env };
    delete env.NODE_OPTIONS;
    delete env.ELECTRON_RUN_AS_NODE;
    launch = chromium.launchServer({ executablePath: executable, headless: true, chromiumSandbox: true, host: "127.0.0.1", timeout: 12e3, env, args: software ? ["--disable-gpu"] : [] });
    launch.then((value) => {
      server = value;
      if (task.signal.aborted) kill();
    }, () => {
    });
    server = await step("browser-launch", () => launch);
    const pid = server.process().pid;
    trace.event("browser-process", "started", { pid, software });
    server.once("close", () => {
      trace.event("browser-process", "exited", { pid });
      task.cancel(index.previewError(index.ErrorCode.htmlPreviewUnavailable));
    });
    const browser = await step("browser-connect", () => chromium.connect(server.wsEndpoint(), { timeout: 8e3 }));
    const context = await step("browser-context", () => browser.newContext({ viewport: { width: input.width, height: input.height }, deviceScaleFactor: 1, serviceWorkers: "block", acceptDownloads: false }));
    const page = await step("browser-page", () => context.newPage());
    page.on("crash", () => task.cancel(index.previewError(index.ErrorCode.htmlPreviewUnavailable)));
    page.on("popup", (popup) => {
      void popup.close().catch(() => {
      });
    });
    const url = "https://" + node_crypto.randomUUID() + ".monkeyai-preview.invalid/index.html";
    const policy = index.HTML_PREVIEW_POLICY.replace("frame-src 'none'", "frame-src about:") + "; object-src 'none'";
    let served = false;
    await step("browser-resource-policy", () => context.route("**/*", async (route) => {
      const request = route.request();
      if (!served && request.url() === url && request.isNavigationRequest() && request.frame() === page.mainFrame()) {
        served = true;
        await route.fulfill({ status: 200, headers: { "Content-Type": "text/html; charset=utf-8", "Content-Security-Policy": policy, "Referrer-Policy": "no-referrer", "Cache-Control": "no-store" }, body: input.content });
      } else if (request.url().startsWith("https:") && ["image", "media"].includes(request.resourceType())) await route.continue();
      else await route.abort("blockedbyclient");
    }));
    stage("load");
    await step("browser-load", () => page.goto(url, { waitUntil: "domcontentloaded", timeout: 18e3 }));
    if (input.content.includes("<body data-monkeyai-poster-preview")) {
      stage("fonts");
      const cdp = await step("browser-font-session", () => context.newCDPSession(page));
      let attached = false;
      await step("browser-poster-fonts", () => index.preparePosterFonts({
        executeJavaScript: (code) => page.evaluate(code),
        debugger: {
          isAttached: () => attached,
          attach: () => {
            attached = true;
          },
          detach: () => {
            attached = false;
          },
          sendCommand: (method, params) => cdp.send(method, params)
        }
      }, input.fontFamilies, input.fontCss, { ...trace, step }));
    }
    stage("capture");
    await step("browser-assets", () => page.evaluate(async () => {
      const ready = async (doc) => {
        await doc.fonts.ready;
        await Promise.all([...doc.images].map((image) => image.decode().catch(() => {
        })));
      };
      await ready(document);
      await Promise.all([...document.querySelectorAll("iframe")].map(async (frame) => {
        if (frame.contentDocument) await ready(frame.contentDocument);
      }));
      await new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve)));
    }));
    const png = await step("browser-screenshot", () => page.screenshot({ type: "png", omitBackground: true, timeout: 18e3 }));
    const encoded = await step("browser-image-check", () => index.desktopImages().request({ type: "prepareBrowserPreview", png, width: input.width, height: input.height }, { signal: task.signal }));
    return `data:${encoded.mimeType};base64,${Buffer.from(encoded.data).toString("base64")}`;
  } finally {
    task.signal.removeEventListener("abort", kill);
    kill();
    try {
      if (launch && !server) {
        await launch.then((value) => {
          server = value;
          kill();
        }, () => {
        });
      }
      if (killing) await killing;
      trace.event("browser-process-cleanup", "done");
    } catch {
      throw index.previewError("canvas_preview_browser_cleanup_failed");
    } finally {
      task.close();
    }
  }
}
exports.previewBrowserExecutable = previewBrowserExecutable;
exports.renderBrowserPreview = renderBrowserPreview;
