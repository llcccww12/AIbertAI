"use strict";

const { app, BrowserWindow, ipcMain, shell } = require("electron");
const { spawn } = require("node:child_process");
const path = require("node:path");
const readline = require("node:readline");
const { backendOrigin, modelName } = require("./deployment.json");

const executable = path.join(process.resourcesPath, process.platform === "win32"
  ? "aibertai-native.exe"
  : "aibertai-native");
let mainWindow;
let active;

app.commandLine.appendSwitch("disable-background-networking");

app.whenReady().then(() => {
  mainWindow = new BrowserWindow({
    width: 760,
    height: 620,
    minWidth: 540,
    minHeight: 490,
    title: "AIbertAI 私有部署验收",
    webPreferences: {
      preload: path.join(__dirname, "preload.cjs"),
      sandbox: true,
      contextIsolation: true,
      nodeIntegration: false,
      webviewTag: false,
    },
  });

  const { session } = mainWindow.webContents;
  session.webRequest.onBeforeRequest(({ url }, next) => {
    next({ cancel: !url.startsWith("file:") });
  });
  session.setPermissionRequestHandler((_contents, _permission, next) => next(false));
  mainWindow.webContents.setWindowOpenHandler(() => ({ action: "deny" }));
  mainWindow.webContents.on("will-navigate", (event) => event.preventDefault());
  mainWindow.on("closed", () => { mainWindow = null; });
  void mainWindow.loadFile("index.html");
});

function isTrusted(event) {
  return mainWindow?.webContents === event.sender &&
    event.senderFrame === mainWindow.webContents.mainFrame;
}

ipcMain.handle("acceptance:deployment", (event) => {
  if (!isTrusted(event)) throw new Error("Invalid sender");
  return { backendOrigin, modelName };
});

ipcMain.handle("acceptance:run", (event, prompt) => {
  if (!isTrusted(event) || active || typeof prompt !== "string" ||
      !prompt.trim() || Buffer.byteLength(prompt, "utf8") > 16384) {
    throw new Error("Invalid request");
  }
  const child = spawn(executable, ["verify"], {
    windowsHide: true,
    stdio: ["pipe", "pipe", "ignore"],
  });
  active = { child, authorize: null };
  const send = (type, value) => {
    if (!event.sender.isDestroyed()) event.sender.send("acceptance:event", { type, value });
  };
  child.stdin.on("error", () => {});
  child.stdin.end(prompt);
  const events = readline.createInterface({ input: child.stdout });
  events.on("line", (line) => {
    try {
      const { event: type, value } = JSON.parse(line);
      if (typeof value !== "string" || !["authorize", "login", "delta", "reported", "error"].includes(type)) return;
      if (type === "authorize") {
        const requested = new URL(value);
        if (requested.origin !== backendOrigin || requested.pathname !== "/oauth/authorize" ||
            requested.username || requested.password || requested.hash) {
          throw new Error("OAuth 地址不符合私有部署配置");
        }
        active.authorize = requested.href;
        send("authorize", "");
        return;
      }
      send(type, value);
    } catch {
      send("error", "无法验证运行时返回的数据");
      child.kill();
    }
  });
  child.on("error", () => send("error", "无法启动验收运行时"));
  child.once("close", (code) => {
    active = null;
    send("finished", code === 0 ? "passed" : "failed");
  });
});

ipcMain.handle("acceptance:authorize", async (event) => {
  if (!isTrusted(event) || !active?.authorize) throw new Error("No authorization pending");
  await shell.openExternal(active.authorize);
});

app.on("window-all-closed", () => app.quit());
app.on("before-quit", () => active?.child.kill());
