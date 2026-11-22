// Inject the employee webview's React singleton before loading DSH components.
export const framework = {};
export function bindChatFramework(react, jsx, dom) {
  Object.assign(framework, { react, jsx, dom });
}
