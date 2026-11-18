import { p, r as u, a as m, e as l } from "./index-zDHqw1Zn.js";
import { d as f } from "./media-container-CTeLWKx6.js";
async function g(w, n, r, e, a = new AbortController().signal) {
  const o = p(r);
  if (!o || !r.source || r.clips?.length)
    throw new Error("视频没有可导出的内容");
  const c = f(new Uint8Array(await e.slice(0, 12).arrayBuffer()));
  if (!c) throw new Error("视频导出仅支持 MP4 或 WebM");
  const s = URL.createObjectURL(e);
  try {
    const t = await u(s, "video", a);
    if (!t.width || !t.height)
      throw new Error("无法读取原视频尺寸，未导出缩小的视频");
    const d = m([], o),
      i = d[0]?.clips[0];
    if (!i || i.kind !== "video") throw new Error("无法准备视频音轨");
    r.colorGrading && (i.colorGrading = structuredClone(r.colorGrading));
    const h = {
      id: r.id,
      type: "media-project",
      projectKind: "video",
      x: 0,
      y: 0,
      zIndex: 0,
      width: t.width,
      height: t.height,
      durationMs: o.durationMs,
      tracks: d,
    };
    return await l(w, n, h, c === "video/webm" ? "webm" : "mp4", {
      signal: a,
      sourceBlobs: new Map([[r.source, e]]),
      ...(r.watermark
        ? {
            watermark: { value: r.watermark, width: r.width, height: r.height },
          }
        : {}),
    });
  } finally {
    URL.revokeObjectURL(s);
  }
}
export { g as exportVideoWithAudio };
