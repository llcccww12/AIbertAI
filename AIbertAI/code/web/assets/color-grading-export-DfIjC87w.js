const __vite__mapDeps = (
  i,
  m = __vite__mapDeps,
  d = m.f ||
    (m.f = [
      "assets/index-DwJ5Es5U.js",
      "assets/__vite-browser-external-D7Ct-6yo.js",
    ]),
) => i.map((i) => d[i]);
import { _ as S, h as C, C as I, s as P, d as x } from "./index-zDHqw1Zn.js";
import { d as G } from "./media-container-CTeLWKx6.js";
async function W(p, e, g) {
  const {
      ALL_FORMATS: v,
      BlobSource: M,
      BufferTarget: y,
      Conversion: T,
      Input: b,
      Mp4OutputFormat: E,
      Output: _,
      QUALITY_HIGH: A,
      WebMOutputFormat: F,
    } = await S(
      async () => {
        const {
          ALL_FORMATS: c,
          BlobSource: h,
          BufferTarget: f,
          Conversion: u,
          Input: i,
          Mp4OutputFormat: r,
          Output: l,
          QUALITY_HIGH: m,
          WebMOutputFormat: O,
        } = await import("./index-DwJ5Es5U.js");
        return {
          ALL_FORMATS: c,
          BlobSource: h,
          BufferTarget: f,
          Conversion: u,
          Input: i,
          Mp4OutputFormat: r,
          Output: l,
          QUALITY_HIGH: m,
          WebMOutputFormat: O,
        };
      },
      __vite__mapDeps([0, 1]),
    ),
    s = G(new Uint8Array(await p.slice(0, 12).arrayBuffer()));
  if (!s) throw new Error("视频导出仅支持 MP4 或 WebM");
  const o = new b({ source: new M(p), formats: v }),
    w = new y(),
    t = document.createElement("canvas"),
    a = document.createElement("canvas"),
    n = t.getContext("2d");
  if (!n) throw (o.dispose(), new Error("无法初始化视频导出画布"));
  let d;
  try {
    (C(e.colorGrading) && (d = new I(a)),
      document.fonts &&
        (await Promise.all(
          (e.subtitles ?? []).map((r) =>
            document.fonts.load(`${r.fontWeight} 48px ${r.fontFamily}`),
          ),
        )));
    const c = await o.getPrimaryVideoTrack();
    if (!c) throw new Error("视频文件中没有可导出的视频轨道");
    const h = await o.getPrimaryAudioTrack(),
      f = await Promise.all([c.getFirstTimestamp(), h?.getFirstTimestamp()]),
      u = Math.max(0, Math.min(...f.filter((r) => r !== void 0))),
      i = await T.init({
        input: o,
        output: new _({
          format: s === "video/webm" ? new F() : new E(),
          target: w,
        }),
        tracks: "primary",
        video: {
          codec: s === "video/webm" ? "vp9" : "avc",
          quality: A,
          forceTranscode: !0,
          process: (r) => {
            if (
              ((t.width !== r.displayWidth || t.height !== r.displayHeight) &&
                ((t.width = r.displayWidth), (t.height = r.displayHeight)),
              r.draw(n, 0, 0),
              d)
            ) {
              if (
                (d.render(t, t.width, t.height, e.colorGrading, r.timestamp),
                !e.subtitles?.length)
              )
                return a;
              n.drawImage(a, 0, 0, t.width, t.height);
            }
            const l = Math.round(r.timestamp * 1e3) + (e.trimStartMs ?? 0);
            for (const m of e.subtitles ?? [])
              P(m, l) && x(n, m, t.width, t.height);
            return t;
          },
        },
        trim:
          e.trimStartMs || e.trimEndMs !== void 0
            ? {
                start: u + (e.trimStartMs ?? 0) / 1e3,
                ...(e.trimEndMs === void 0
                  ? {}
                  : { end: u + e.trimEndMs / 1e3 }),
              }
            : void 0,
        showWarnings: !1,
      });
    if (!i.isValid || i.discardedTracks.length)
      throw new Error("当前平台不支持该视频或音轨的编码，未导出不完整视频");
    if (((i.onProgress = g), await i.execute(), !w.buffer))
      throw new Error("视频导出未生成结果");
    return new Blob([w.buffer], { type: s });
  } finally {
    (d?.dispose(),
      o.dispose(),
      (t.width = a.width = 1),
      (t.height = a.height = 1));
  }
}
export { W as exportColorGradedVideo };
