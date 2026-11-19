"use strict";

const { contextBridge, ipcRenderer } = require("electron");
contextBridge.exposeInMainWorld("acceptance", {
  deployment: () => ipcRenderer.invoke("acceptance:deployment"),
  run: (prompt) => ipcRenderer.invoke("acceptance:run", prompt),
  authorize: () => ipcRenderer.invoke("acceptance:authorize"),
  onEvent: (handler) => {
    ipcRenderer.on("acceptance:event", (_event, value) => handler(value));
  },
});
