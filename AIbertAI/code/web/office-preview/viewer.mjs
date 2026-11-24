import { getDocument, GlobalWorkerOptions } from "/vendor/pdf.js";

GlobalWorkerOptions.workerSrc = "/vendor/pdf.worker.js";
const container = document.querySelector("main");
const token = new URL(location.href).searchParams.get("t");
const events = new EventSource(`${location.pathname}/events?t=${encodeURIComponent(token)}`);
function unavailable() {
  events.close();
  container.textContent = "当前文档无法预览";
}
events.addEventListener("failed", unavailable);
events.addEventListener("error", unavailable);
events.addEventListener("ready", async (event) => {
  events.close();
  try {
    const result = JSON.parse(event.data);
    const url = new URL(result.docUrl, location.origin);
    if (url.origin !== location.origin || url.pathname !== location.pathname + "/doc") throw new Error("invalid_pdf");
    const pdf = await getDocument({ url: url.href, isEvalSupported: false, useWasm: false, maxImageSize: 16e6 }).promise;
    if (!pdf.numPages || pdf.numPages > 500) throw new Error("invalid_page_count");
    for (let index = 1; index <= pdf.numPages; index++) {
      const page = await pdf.getPage(index);
      const initial = page.getViewport({ scale: 1 });
      const viewport = page.getViewport({ scale: 960 / initial.width });
      if (viewport.height * viewport.width > 2e6) throw new Error("invalid_pdf_page_size");
      const canvas = document.createElement("canvas");
      canvas.width = Math.ceil(viewport.width);
      canvas.height = Math.ceil(viewport.height);
      canvas.setAttribute("aria-label", `第 ${index} 页`);
      await page.render({ canvas, viewport, background: "#ffffff" }).promise;
      // Keep the loading indicator until the first page is drawn, as DSH does.
      if (index === 1) container.textContent = "";
      container.append(canvas);
      page.cleanup();
    }
  } catch {
    unavailable();
  }
});
