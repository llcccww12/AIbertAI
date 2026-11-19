const __vite__mapDeps = (
  i,
  m = __vite__mapDeps,
  d = m.f || (m.f = ["assets/index-zDHqw1Zn.js", "assets/index-BvV-3XZ5.css"]),
) => i.map((i) => d[i]);
import { _ as N, M, m as P } from "./index-zDHqw1Zn.js";
var s;
(function (a) {
  ((a.LOAD = "LOAD"),
    (a.EXEC = "EXEC"),
    (a.FFPROBE = "FFPROBE"),
    (a.WRITE_FILE = "WRITE_FILE"),
    (a.READ_FILE = "READ_FILE"),
    (a.DELETE_FILE = "DELETE_FILE"),
    (a.RENAME = "RENAME"),
    (a.CREATE_DIR = "CREATE_DIR"),
    (a.LIST_DIR = "LIST_DIR"),
    (a.DELETE_DIR = "DELETE_DIR"),
    (a.ERROR = "ERROR"),
    (a.DOWNLOAD = "DOWNLOAD"),
    (a.PROGRESS = "PROGRESS"),
    (a.LOG = "LOG"),
    (a.MOUNT = "MOUNT"),
    (a.UNMOUNT = "UNMOUNT"));
})(s || (s = {}));
const T = (() => {
    let a = 0;
    return () => a++;
  })(),
  x = new Error("ffmpeg is not loaded, call `await ffmpeg.load()` first"),
  k = new Error("called FFmpeg.terminate()");
class B {
  #e = null;
  #r = {};
  #a = {};
  #o = [];
  #n = [];
  #threaded;
  loaded = !1;
  constructor(threaded) {
    this.#threaded = threaded;
  }
  #s = () => {
    this.#e &&
      (this.#e.onmessage = ({ data: { id: t, type: e, data: r } }) => {
        switch (e) {
          case s.LOAD:
            ((this.loaded = !0), this.#r[t](r));
            break;
          case s.MOUNT:
          case s.UNMOUNT:
          case s.EXEC:
          case s.FFPROBE:
          case s.WRITE_FILE:
          case s.READ_FILE:
          case s.DELETE_FILE:
          case s.RENAME:
          case s.CREATE_DIR:
          case s.LIST_DIR:
          case s.DELETE_DIR:
            this.#r[t](r);
            break;
          case s.LOG:
            this.#o.forEach((o) => o(r));
            break;
          case s.PROGRESS:
            this.#n.forEach((o) => o(r));
            break;
          case s.ERROR:
            this.#a[t](r);
            break;
        }
        (delete this.#r[t], delete this.#a[t]);
      });
  };
  #t = ({ type: t, data: e }, r = [], o) =>
    this.#e
      ? new Promise((i, c) => {
          const n = T();
          (this.#e && this.#e.postMessage({ id: n, type: t, data: e }, r),
            (this.#r[n] = i),
            (this.#a[n] = c),
            o?.addEventListener(
              "abort",
              () => {
                c(new DOMException(`Message # ${n} was aborted`, "AbortError"));
              },
              { once: !0 },
            ));
        })
      : Promise.reject(x);
  on(t, e) {
    t === "log" ? this.#o.push(e) : t === "progress" && this.#n.push(e);
  }
  off(t, e) {
    t === "log"
      ? (this.#o = this.#o.filter((r) => r !== e))
      : t === "progress" && (this.#n = this.#n.filter((r) => r !== e));
  }
  load = ({ classWorkerURL: t, ...e } = {}, { signal: r } = {}) => (
    this.#e ||
      ((this.#e = t
        ? new Worker(new URL(t, import.meta.url), { type: "module" })
        : new Worker(new URL("/assets/worker-BAOIWoxA.js", import.meta.url), {
            type: "module",
          })),
      this.#s()),
    this.#t({ type: s.LOAD, data: e }, void 0, r)
  );
  exec = (args, timeout = -1, { signal } = {}) => {
    if (this.#threaded) {
      // Cap decoding, filters and encoding so they can share the WASM worker pool.
      args = [
        "-filter_threads", "2", "-filter_complex_threads", "2", "-threads", "2",
        ...args.slice(0, -1), "-threads", "2", args.at(-1),
      ];
    }
    return this.#t({ type: s.EXEC, data: { args, timeout } }, void 0, signal);
  };
  ffprobe = (t, e = -1, { signal: r } = {}) =>
    this.#t({ type: s.FFPROBE, data: { args: t, timeout: e } }, void 0, r);
  terminate = () => {
    const t = Object.keys(this.#a);
    for (const e of t) (this.#a[e](k), delete this.#a[e], delete this.#r[e]);
    this.#e && (this.#e.terminate(), (this.#e = null), (this.loaded = !1));
  };
  writeFile = (t, e, { signal: r } = {}) => {
    const o = [];
    return (
      e instanceof Uint8Array && o.push(e.buffer),
      this.#t({ type: s.WRITE_FILE, data: { path: t, data: e } }, o, r)
    );
  };
  mount = (t, e, r) => {
    const o = [];
    return this.#t(
      { type: s.MOUNT, data: { fsType: t, options: e, mountPoint: r } },
      o,
    );
  };
  unmount = (t) => {
    const e = [];
    return this.#t({ type: s.UNMOUNT, data: { mountPoint: t } }, e);
  };
  readFile = (t, e = "binary", { signal: r } = {}) =>
    this.#t({ type: s.READ_FILE, data: { path: t, encoding: e } }, void 0, r);
  deleteFile = (t, { signal: e } = {}) =>
    this.#t({ type: s.DELETE_FILE, data: { path: t } }, void 0, e);
  rename = (t, e, { signal: r } = {}) =>
    this.#t({ type: s.RENAME, data: { oldPath: t, newPath: e } }, void 0, r);
  createDir = (t, { signal: e } = {}) =>
    this.#t({ type: s.CREATE_DIR, data: { path: t } }, void 0, e);
  listDir = (t, { signal: e } = {}) =>
    this.#t({ type: s.LIST_DIR, data: { path: t } }, void 0, e);
  deleteDir = (t, { signal: e } = {}) =>
    this.#t({ type: s.DELETE_DIR, data: { path: t } }, void 0, e);
}
var U;
(function (a) {
  ((a.MEMFS = "MEMFS"),
    (a.NODEFS = "NODEFS"),
    (a.NODERAWFS = "NODERAWFS"),
    (a.IDBFS = "IDBFS"),
    (a.WORKERFS = "WORKERFS"),
    (a.PROXYFS = "PROXYFS"));
})(U || (U = {}));
const j = "/assets/worker-BAOIWoxA.js";
async function loadLocalCore(threaded) {
  const engine = new B(threaded);
  const folder = new URL(`${M}/ffmpeg/core-${threaded ? "mt" : "st"}/`, import.meta.url);
  const asset = (name) => new URL(name, folder).href;
  try {
    await engine.load({
      classWorkerURL: j,
      coreURL: asset("ffmpeg-core.js"),
      wasmURL: asset("ffmpeg-core.wasm"),
      ...(threaded ? { workerURL: asset("ffmpeg-core.worker.js") } : {}),
    }, { signal: AbortSignal.timeout(20000) });
    return engine;
  } catch (error) {
    engine.terminate();
    throw error;
  }
}
let w,
  b = Promise.resolve();
async function O(a) {
  w ??= (async () => {
    a?.({ stage: "downloading", progress: 0 });
    const engine = globalThis.crossOriginIsolated
      ? await loadLocalCore(true).catch(() => loadLocalCore(false))
      : await loadLocalCore(false);
    a?.({ stage: "downloading", progress: 1 });
    return engine;
  })().catch((error) => {
    w = undefined;
    throw error;
  });
  return w;
}
function L(a) {
  const t = b.then(a);
  return ((b = t.catch(() => {})), t);
}
async function y(a, t) {
  for (const e of t)
    try {
      await a.deleteFile(e);
    } catch {}
}
const A = (a, t = !1) => [
    ...(!t && a.sourceStartMs ? ["-ss", String(a.sourceStartMs / 1e3)] : []),
    "-t",
    String(P(a).originalDuration),
    "-i",
    "input.mp4",
  ],
  D = (a, t = 7) => {
    const e = [];
    let r = a;
    for (let o = 0; o < t; o++) {
      const i = Math.max(0.5, Math.min(2, r));
      (e.push(i), (r /= i));
    }
    return e;
  };
function X(a, t, e) {
  return L(async () => {
    const r = await O(),
      o = [],
      i = ({ message: c }) => {
        (o.push(c.slice(0, 500)), o.length > 8 && o.shift());
      };
    r.on("log", i);
    try {
      await r.writeFile("input.mp4", new Uint8Array(await a.arrayBuffer()));
      const { MediaProjectSpeedEngine: c } = await N(
          async () => {
            const { MediaProjectSpeedEngine: E } =
              await import("./index-zDHqw1Zn.js").then((p) => p.f);
            return { MediaProjectSpeedEngine: E };
          },
          __vite__mapDeps([0, 1]),
        ),
        n = new c(t),
        l = n.data.keyframes.length
          ? n.data.keyframes.map((E) => E.speed)
          : [n.baseSpeed],
        d = Math.max(
          1,
          Math.ceil(Math.log2(Math.max(...l))),
          Math.ceil(Math.log2(1 / Math.min(...l))),
        ),
        f = D(n.getSpeedAtSourceTime(0), d),
        u = f.map((E, p) => `atempo@s${p}=${E}`);
      if (n.data.keyframes.length) {
        const E = [];
        let p = f;
        for (let R = 1; R < 1e3; R++) {
          const I = (R / 1e3) * n.data.originalDuration,
            _ = D(n.getSpeedAtSourceTime(I), d);
          (_.forEach((v, S) => {
            Math.abs(v - p[S]) > 1e-8 &&
              E.push(`${I.toFixed(6)} atempo@s${S} tempo ${v.toFixed(8)}`);
          }),
            (p = _));
        }
        E.length &&
          (await r.writeFile(
            "speed-commands.txt",
            new TextEncoder().encode(
              E.join(`;
`) + ";",
            ),
          ),
          u.unshift("asendcmd=f=speed-commands.txt"));
      }
      const h = n.getEffectiveDuration();
      if (
        (u.push(`apad=whole_dur=${h}`, `atrim=duration=${h}`),
        (await r.exec([
          ...A(t, e),
          "-vn",
          "-af",
          ["asetpts=PTS-STARTPTS", ...u].join(","),
          "-ar",
          "48000",
          "-ac",
          "2",
          "-y",
          "output.wav",
        ])) !== 0)
      )
        throw new Error(`本地音高修正处理失败：${o.join(" | ")}`);
      const m = await r.readFile("output.wav");
      if (typeof m == "string" || !m.length)
        throw new Error("音高修正未生成有效音频");
      return new Blob([new Uint8Array(m)], { type: "audio/wav" });
    } finally {
      (r.off("log", i),
        await y(r, ["input.mp4", "output.wav", "speed-commands.txt"]));
    }
  });
}
function V(a, t, e, r) {
  return L(async () => {
    const o = await O(r),
      i = t.kind === "audio",
      c = i ? "output.wav" : "output.mp4",
      n = t.speed ?? 1,
      l =
        t.speedRamp?.pitchCorrection === !1
          ? `aresample=48000,asetrate=${48e3 * n},aresample=48000`
          : D(n)
              .map((f) => `atempo=${f}`)
              .join(","),
      d = ({ progress: f }) =>
        r?.({ stage: "retiming", progress: Math.max(0, Math.min(1, f)) });
    o.on("progress", d);
    try {
      await o.writeFile("input.mp4", new Uint8Array(await a.arrayBuffer()));
      const f = i
        ? ["-vn"]
        : [
            "-vf",
            `settb=AVTB,setpts=(PTS-STARTPTS)/${n},fps=60:round=up:eof_action=pass`,
            "-c:v",
            "libx264",
            "-preset",
            "ultrafast",
            "-crf",
            "18",
            "-c:a",
            "aac",
          ];
      if ((await o.exec([...A(t, e), ...f, "-af", l, "-y", c])) !== 0)
        throw new Error("本地高速素材处理失败");
      const u = await o.readFile(c);
      if (typeof u == "string" || !u.length)
        throw new Error("高速处理未生成有效素材");
      return new Blob([new Uint8Array(u)], {
        type: i ? "audio/wav" : "video/mp4",
      });
    } finally {
      (o.off("progress", d), await y(o, ["input.mp4", c]));
    }
  });
}
function Y(a, t, e, r) {
  return L(async () => {
    const o = await O(r),
      i = t.kind === "audio",
      c = i ? "output.wav" : "output.mp4",
      n = ({ progress: l }) =>
        r?.({ stage: "reversing", progress: Math.max(0, Math.min(1, l)) });
    o.on("progress", n);
    try {
      await o.writeFile("input.mp4", new Uint8Array(await a.arrayBuffer()));
      const l = i
        ? ["-vn", "-af", "areverse"]
        : [
            "-vf",
            "reverse",
            "-af",
            "areverse",
            "-c:v",
            "libx264",
            "-preset",
            "ultrafast",
            "-crf",
            "18",
            "-c:a",
            "aac",
          ];
      if ((await o.exec([...A(t, e), ...l, "-y", c])) !== 0)
        throw new Error("本地倒放处理失败");
      const d = await o.readFile(c);
      if (typeof d == "string" || !d.length)
        throw new Error("倒放未生成有效素材");
      return new Blob([new Uint8Array(d)], {
        type: i ? "audio/wav" : "video/mp4",
      });
    } finally {
      (o.off("progress", n), await y(o, ["input.mp4", c]));
    }
  });
}
export { V as fastForward, Y as reverse, X as timeStretch };
