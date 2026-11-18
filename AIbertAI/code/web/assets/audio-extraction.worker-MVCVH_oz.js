(function () {
  "use strict";
  function g(i) {
    if (!i) throw new Error("Assertion failed.");
  }
  const on = Math.PI / 180,
    cn = 180 / Math.PI,
    Lt = (i) => {
      const e = ((i % 360) + 360) % 360;
      if (e === 0 || e === 90 || e === 180 || e === 270) return e;
      throw new Error(`Invalid rotation ${i}.`);
    },
    Fi = [1, 0, 0, 0, 1, 0, 0, 0, 1],
    qs = (i, e, t) => {
      const [r, n, , s, a] = i,
        o = Math.abs(r) * e + Math.abs(s) * t,
        c = Math.abs(n) * e + Math.abs(a) * t;
      return ai(ai($s(-e / 2, -t / 2), i), $s(o / 2, c / 2));
    },
    Ri = (i) => {
      const [e, t] = i,
        r = Math.atan2(t, js(i) ? -e : e);
      return Lt(pn(r * cn, 90));
    },
    js = (i) => {
      const [e, t, , r, n] = i;
      return e * n - t * r < 0;
    },
    ln = (i) => {
      const e = i * on,
        t = Math.round(Math.cos(e)),
        r = Math.round(Math.sin(e));
      return [t, r, 0, -r, t, 0, 0, 0, 1];
    },
    $s = (i, e) => [1, 0, 0, 0, 1, 0, i, e, 1],
    Ks = (i, e) => [i, 0, 0, 0, e, 0, 0, 0, 1],
    ai = (i, e) => {
      const t = new Array(9);
      for (let r = 0; r < 3; r++)
        for (let n = 0; n < 3; n++)
          t[3 * r + n] =
            i[3 * r] * e[n] + i[3 * r + 1] * e[3 + n] + i[3 * r + 2] * e[6 + n];
      return t;
    },
    Gs = (i, e, t, r) => ({ rotation: Lt(i + (e ? -t : t)), flip: e !== r }),
    ne = (i) => i && i[i.length - 1],
    ut = (i) => i >= 0 && i < 2 ** 32,
    rl = (i) => i >= -2147483648 && i < 2 ** 31,
    M = (i) => {
      let e = 0;
      for (; i.readBits(1) === 0 && e < 32;) e++;
      if (e >= 32) throw new Error("Invalid exponential-Golomb code.");
      return (1 << e) - 1 + i.readBits(e);
    },
    oi = (i, e) => {
      const t = e + 1,
        r = Math.floor(Math.log2(t));
      (i.writeBits(r, 0), i.writeBits(1, 1), i.writeBits(r, t - 2 ** r));
    },
    dt = (i) => {
      const e = M(i);
      return (e & 1) === 0 ? -(e >> 1) : (e + 1) >> 1;
    },
    nl = (i, e, t, r) => {
      for (let n = e; n < t; n++) {
        const s = Math.floor(n / 8);
        let a = i[s];
        const o = 7 - (n & 7);
        ((a &= ~(1 << o)),
          (a |= ((r & (1 << (t - n - 1))) >> (t - n - 1)) << o),
          (i[s] = a));
      }
    },
    me = (i) =>
      i.constructor === Uint8Array
        ? i
        : ArrayBuffer.isView(i)
          ? new Uint8Array(i.buffer, i.byteOffset, i.byteLength)
          : new Uint8Array(i),
    j = (i) =>
      i.constructor === DataView
        ? i
        : ArrayBuffer.isView(i)
          ? new DataView(i.buffer, i.byteOffset, i.byteLength)
          : new DataView(i),
    sl =
      typeof globalThis.TextEncoder < "u"
        ? globalThis.TextEncoder
        : class {
            constructor() {
              this.encoding = "utf-8";
            }
            encode(e = "") {
              const t = new Uint8Array(3 * e.length);
              let r = 0;
              for (let n = 0; n < e.length; n++) {
                let s = e.charCodeAt(n);
                if (s < 128) t[r++] = s;
                else if (s < 2048)
                  ((t[r++] = 192 | (s >> 6)), (t[r++] = 128 | (s & 63)));
                else if (s < 55296 || s > 57343)
                  ((t[r++] = 224 | (s >> 12)),
                    (t[r++] = 128 | ((s >> 6) & 63)),
                    (t[r++] = 128 | (s & 63)));
                else {
                  const a = n + 1 < e.length ? e.charCodeAt(n + 1) : 0;
                  s < 56320 && a >= 56320 && a <= 57343
                    ? ((s = 65536 + ((s - 55296) << 10) + (a - 56320)),
                      n++,
                      (t[r++] = 240 | (s >> 18)),
                      (t[r++] = 128 | ((s >> 12) & 63)),
                      (t[r++] = 128 | ((s >> 6) & 63)),
                      (t[r++] = 128 | (s & 63)))
                    : ((t[r++] = 239), (t[r++] = 191), (t[r++] = 189));
                }
              }
              return t.slice(0, r);
            }
          },
    Mi =
      typeof globalThis.TextDecoder < "u"
        ? globalThis.TextDecoder
        : class {
            constructor(e = "utf-8") {
              const t = e.trim().toLowerCase();
              if (t === "utf-8" || t === "utf8" || t === "unicode-1-1-utf-8")
                this.encoding = "utf-8";
              else if (t === "utf-16le" || t === "utf-16")
                this.encoding = "utf-16le";
              else if (t === "utf-16be") this.encoding = "utf-16be";
              else
                throw new RangeError(
                  `The encoding label provided ('${e}') is invalid.`,
                );
            }
            decode(e) {
              const t = e ? me(e) : new Uint8Array(0),
                r = [];
              if (this.encoding === "utf-8") {
                let s =
                  t.length >= 3 && t[0] === 239 && t[1] === 187 && t[2] === 191
                    ? 3
                    : 0;
                for (; s < t.length;) {
                  const a = t[s];
                  if (a < 128) {
                    (r.push(a), s++);
                    continue;
                  }
                  let o, c;
                  if (a >= 194 && a < 224) ((o = 1), (c = a & 31));
                  else if (a >= 224 && a < 240) ((o = 2), (c = a & 15));
                  else if (a >= 240 && a < 245) ((o = 3), (c = a & 7));
                  else {
                    (r.push(65533), s++);
                    continue;
                  }
                  let l = a === 224 ? 160 : a === 240 ? 144 : 128,
                    u = a === 237 ? 159 : a === 244 ? 143 : 191,
                    d = 1;
                  for (; d <= o; d++) {
                    const f = s + d < t.length ? t[s + d] : 0;
                    if (f < l || f > u) break;
                    ((c = (c << 6) | (f & 63)), (l = 128), (u = 191));
                  }
                  ((s += d),
                    d <= o
                      ? r.push(65533)
                      : c >= 65536
                        ? ((c -= 65536),
                          r.push(55296 | (c >> 10), 56320 | (c & 1023)))
                        : r.push(c));
                }
              } else {
                const s = this.encoding === "utf-16le",
                  a = s ? 255 : 254,
                  o = s ? 254 : 255;
                let c = t.length >= 2 && t[0] === a && t[1] === o ? 2 : 0;
                for (; c + 1 < t.length;) {
                  const l = s ? t[c] | (t[c + 1] << 8) : (t[c] << 8) | t[c + 1];
                  if (((c += 2), l >= 55296 && l <= 56319)) {
                    const u =
                      c + 1 < t.length
                        ? s
                          ? t[c] | (t[c + 1] << 8)
                          : (t[c] << 8) | t[c + 1]
                        : -1;
                    u >= 56320 && u <= 57343
                      ? (r.push(l, u), (c += 2))
                      : r.push(65533);
                  } else l >= 56320 && l <= 57343 ? r.push(65533) : r.push(l);
                }
                c < t.length && r.push(65533);
              }
              let n = "";
              for (let s = 0; s < r.length; s += 8192)
                n += String.fromCharCode(...r.slice(s, s + 8192));
              return n;
            }
          },
    Oe = new Mi(),
    oe = new sl(),
    ft = (i) => {
      for (let e = 0; e < i.length; e++) if (i.charCodeAt(e) > 255) return !1;
      return !0;
    },
    un = (i) => Object.fromEntries(Object.entries(i).map(([e, t]) => [t, e])),
    Ht = { bt709: 1, bt470bg: 5, smpte170m: 6, bt2020: 9, smpte432: 12 },
    ht = un(Ht),
    qt = {
      bt709: 1,
      smpte170m: 6,
      linear: 8,
      "iec61966-2-1": 13,
      pq: 16,
      hlg: 18,
    },
    mt = un(qt),
    jt = { rgb: 0, bt709: 1, bt470bg: 5, smpte170m: 6, "bt2020-ncl": 9 },
    pt = un(jt),
    dn = (i) =>
      !!i &&
      !!i.primaries &&
      !!i.transfer &&
      !!i.matrix &&
      i.fullRange !== void 0,
    Xs = (i) =>
      !i ||
      (i.primaries == null &&
        i.transfer == null &&
        i.matrix == null &&
        i.fullRange == null),
    Qs = {
      primaries: void 0,
      transfer: void 0,
      matrix: void 0,
      fullRange: void 0,
    },
    kr = (i) =>
      i instanceof ArrayBuffer ||
      (typeof SharedArrayBuffer < "u" && i instanceof SharedArrayBuffer) ||
      ArrayBuffer.isView(i);
  class ci {
    constructor() {
      ((this.currentPromise = Promise.resolve()), (this.pending = 0));
    }
    async acquire() {
      let e;
      const t = new Promise((n) => {
          let s = !1;
          e = () => {
            s || (n(), this.pending--, (s = !0));
          };
        }),
        r = this.currentPromise;
      return ((this.currentPromise = t), this.pending++, await r, e);
    }
  }
  const al = /^[0-9a-fA-F]+$/,
    Di = (i) => [...i].map((e) => e.toString(16).padStart(2, "0")).join(""),
    ol = (i) => {
      g(i.length % 2 === 0);
      const e = new Uint8Array(i.length / 2);
      for (let t = 0; t < i.length; t += 2)
        e[t / 2] = parseInt(i.slice(t, t + 2), 16);
      return e;
    },
    Ys = (i) => (
      (i = ((i >> 1) & 1431655765) | ((i & 1431655765) << 1)),
      (i = ((i >> 2) & 858993459) | ((i & 858993459) << 2)),
      (i = ((i >> 4) & 252645135) | ((i & 252645135) << 4)),
      (i = ((i >> 8) & 16711935) | ((i & 16711935) << 8)),
      (i = ((i >> 16) & 65535) | ((i & 65535) << 16)),
      i >>> 0
    ),
    Oi = (i, e, t) => {
      let r = 0,
        n = i.length - 1,
        s = -1;
      for (; r <= n;) {
        const a = (r + n) >> 1,
          o = t(i[a]);
        o === e ? ((s = a), (n = a - 1)) : o < e ? (r = a + 1) : (n = a - 1);
      }
      return s;
    },
    X = (i, e, t) => {
      let r = 0,
        n = i.length - 1,
        s = -1;
      for (; r <= n;) {
        const a = (r + (n - r + 1) / 2) | 0;
        t(i[a]) <= e ? ((s = a), (r = a + 1)) : (n = a - 1);
      }
      return s;
    },
    Zs = (i, e, t) => {
      const r = X(i, t(e), t);
      i.splice(r + 1, 0, e);
    },
    ce = () => {
      let i, e;
      return {
        promise: new Promise((r, n) => {
          ((i = r), (e = n));
        }),
        resolve: i,
        reject: e,
      };
    },
    fn = (i, e) => {
      const t = i.indexOf(e);
      t !== -1 && i.splice(t, 1);
    },
    Js = (i, e) => {
      for (let t = i.length - 1; t >= 0; t--) if (e(i[t])) return i[t];
    },
    hn = (i, e) => {
      for (let t = i.length - 1; t >= 0; t--) if (e(i[t])) return t;
      return -1;
    },
    cl = async function* (i) {
      Symbol.iterator in i
        ? yield* i[Symbol.iterator]()
        : yield* i[Symbol.asyncIterator]();
    },
    ll = (i) => {
      if (!(Symbol.iterator in i) && !(Symbol.asyncIterator in i))
        throw new TypeError("Argument must be an iterable or async iterable.");
    },
    xe = (i) => {
      throw new Error(`Unexpected value: ${i}`);
    },
    Tr = (i, e, t) => {
      const r = i.getUint8(e),
        n = i.getUint8(e + 1),
        s = i.getUint8(e + 2);
      return t ? r | (n << 8) | (s << 16) : (r << 16) | (n << 8) | s;
    },
    ul = (i, e, t) => (Tr(i, e, t) << 8) >> 8,
    mn = (i, e, t, r) => {
      ((t = t >>> 0),
        (t = t & 16777215),
        r
          ? (i.setUint8(e, t & 255),
            i.setUint8(e + 1, (t >>> 8) & 255),
            i.setUint8(e + 2, (t >>> 16) & 255))
          : (i.setUint8(e, (t >>> 16) & 255),
            i.setUint8(e + 1, (t >>> 8) & 255),
            i.setUint8(e + 2, t & 255)));
    },
    dl = (i, e, t, r) => {
      ((t = se(t, -8388608, 8388607)),
        t < 0 && (t = (t + 16777216) & 16777215),
        mn(i, e, t, r));
    },
    se = (i, e, t) => Math.max(e, Math.min(t, i)),
    fl = (i, e, t) => i + (e - i) * t,
    Pe = "und",
    zi = (i) => {
      const e = Math.round(i);
      return Math.abs(i / e - 1) < 10 * Number.EPSILON ? e : i;
    },
    pn = (i, e) => Math.round(i / e) * e,
    li = (i, e) => Math.round(i * e) / e,
    gn = (i, e) => Math.floor(i / e) * e,
    ea = (i, e) => Math.floor(i * e) / e,
    hl = (i) => {
      let e = 0;
      for (; i;) (e++, (i >>= 1));
      return e;
    },
    wn = (i) => {
      let e = 0;
      for (; i !== 0;) ((i &= i - 1), e++);
      return e;
    },
    ml = /^[a-z]{3}$/,
    Ni = (i) => ml.test(i),
    _t = 1e6 * (1 + Number.EPSILON),
    pl = (i, e) => {
      const t = i < 0 ? -1 : 1;
      i = Math.abs(i);
      let r = 0,
        n = 1,
        s = 1,
        a = 0,
        o = i;
      for (;;) {
        const c = Math.floor(o),
          l = c * s + r,
          u = c * a + n;
        if (u > e) return { num: t * s, den: a };
        if (
          ((r = s), (n = a), (s = l), (a = u), (o = 1 / (o - c)), !isFinite(o))
        )
          break;
      }
      return { num: t * s, den: a };
    };
  class Sr {
    constructor() {
      this.currentPromise = Promise.resolve();
    }
    call(e) {
      return (this.currentPromise = this.currentPromise.then(e));
    }
  }
  let yn = null;
  const Ui = () =>
    yn !== null
      ? yn
      : (yn = !!(
          typeof navigator < "u" &&
          (navigator.vendor?.match(/apple/i) ||
            (/AppleWebKit/.test(navigator.userAgent) &&
              !/Chrome/.test(navigator.userAgent)) ||
            /\b(iPad|iPhone|iPod)\b/.test(navigator.userAgent))
        ));
  let bn = null;
  const ta = () =>
    bn !== null
      ? bn
      : (bn =
          typeof navigator < "u" && navigator.userAgent?.includes("Firefox"));
  let kn = null;
  const Tn = () =>
      kn !== null
        ? kn
        : (kn = !!(
            typeof navigator < "u" &&
            (navigator.vendor?.includes("Google Inc") ||
              /Chrome/.test(navigator.userAgent))
          )),
    xr = (i) =>
      typeof globalThis.isSecureContext < "u" && !globalThis.isSecureContext
        ? `${i} is not available in this environment; this may be because this page is running in an insecure context. Try serving your page over HTTPS or use localhost.`
        : `${i} is not available in this environment.`,
    gl = (i) =>
      i instanceof DOMException &&
      i.name === "QuotaExceededError" &&
      /reclaimed/i.test(i.message),
    wl = (async () => {})().constructor,
    A = (i) =>
      i instanceof wl || i instanceof Promise
        ? !0
        : typeof i?.then == "function",
    ui = (i, e) => (i !== -1 ? i : e),
    ia = (i, e, t, r) => i <= r && t <= e,
    Wi = function* (i) {
      for (const e in i) {
        const t = i[e];
        t !== void 0 && (yield { key: e, value: t });
      }
    },
    yl = (i) => {
      switch (i.toLowerCase()) {
        case "image/jpeg":
        case "image/jpg":
          return ".jpg";
        case "image/png":
          return ".png";
        case "image/gif":
          return ".gif";
        case "image/webp":
          return ".webp";
        case "image/bmp":
          return ".bmp";
        case "image/svg+xml":
          return ".svg";
        case "image/tiff":
          return ".tiff";
        case "image/avif":
          return ".avif";
        case "image/x-icon":
        case "image/vnd.microsoft.icon":
          return ".ico";
        default:
          return null;
      }
    },
    ra = (i) => {
      const e = atob(i),
        t = new Uint8Array(e.length);
      for (let r = 0; r < e.length; r++) t[r] = e.charCodeAt(r);
      return t;
    },
    na = (i, e) => {
      if (i.length !== e.length) return !1;
      for (let t = 0; t < i.length; t++) if (i[t] !== e[t]) return !1;
      return !0;
    },
    Sn = () => {
      Symbol.dispose ??= Symbol("Symbol.dispose");
    },
    gt = (i) => typeof i == "number" && !Number.isNaN(i),
    vt = (i, e) => {
      if (e.includes("://")) return e;
      if (i.includes("://")) {
        const o = i.indexOf("?");
        o !== -1 && (i = i.slice(0, o));
      }
      let t;
      if (e.startsWith("/")) {
        const o = i.indexOf("://");
        if (o === -1) t = e;
        else {
          const c = i.indexOf("/", o + 3);
          c === -1 ? (t = i + e) : (t = i.slice(0, c) + e);
        }
      } else {
        const o = i.lastIndexOf("/");
        o === -1 ? (t = e) : (t = i.slice(0, o + 1) + e);
      }
      let r = "";
      const n = t.indexOf("://");
      if (n !== -1) {
        const o = t.indexOf("/", n + 3);
        o !== -1 && ((r = t.slice(0, o)), (t = t.slice(o)));
      }
      const s = t.split("/"),
        a = [];
      for (const o of s) o === ".." ? a.pop() : o !== "." && a.push(o);
      return r + a.join("/");
    },
    bl = (i, e) => {
      const t = i.indexOf("?");
      if (t === -1) return null;
      const r = i.indexOf("#", t),
        n = i.slice(t + 1, r === -1 ? void 0 : r);
      for (const s of n.split("&")) {
        const a = s.indexOf("="),
          o = a === -1 ? s : s.slice(0, a);
        if (decodeURIComponent(o) !== e) continue;
        const c = a === -1 ? "" : s.slice(a + 1);
        return decodeURIComponent(c);
      }
      return null;
    },
    Vi = (i, e) => {
      let t = 0;
      for (let r = 0; r < i.length; r++) e(i[r]) && t++;
      return t;
    },
    xn = (i, e) => {
      let t = -1,
        r = 1 / 0;
      for (let n = 0; n < i.length; n++) {
        const s = e(i[n]);
        s < r && ((r = s), (t = n));
      }
      return t;
    },
    di = (i) => {
      (g(Number.isInteger(i.num)), g(Number.isInteger(i.den)), g(i.den !== 0));
      let e = Math.abs(i.num),
        t = Math.abs(i.den);
      for (; t !== 0;) {
        const n = e % t;
        ((e = t), (t = n));
      }
      const r = e || 1;
      return { num: i.num / r, den: i.den / r };
    },
    Pn = (i, e) => {
      if (typeof i != "object" || !i)
        throw new TypeError(`${e} must be an object.`);
      if (!Number.isInteger(i.left) || i.left < 0)
        throw new TypeError(`${e}.left must be a non-negative integer.`);
      if (!Number.isInteger(i.top) || i.top < 0)
        throw new TypeError(`${e}.top must be a non-negative integer.`);
      if (!Number.isInteger(i.width) || i.width < 0)
        throw new TypeError(`${e}.width must be a non-negative integer.`);
      if (!Number.isInteger(i.height) || i.height < 0)
        throw new TypeError(`${e}.height must be a non-negative integer.`);
    },
    sa = (i) => new Promise((e) => setTimeout(e, i)),
    aa = (i) => (Array.isArray(i) ? i : [i]);
  class Li {
    constructor() {
      this._listeners = new Map();
    }
    on(e, t, r) {
      this._listeners.has(e) || this._listeners.set(e, new Set());
      const n = { fn: t, once: r?.once ?? !1 };
      return (
        this._listeners.get(e).add(n),
        () => {
          this._listeners.get(e)?.delete(n);
        }
      );
    }
    _emit(...e) {
      const [t, r] = e,
        n = this._listeners.get(t);
      if (n)
        for (const s of n) {
          try {
            s.fn(r);
          } catch (a) {
            console.error(a);
          }
          s.once && n.delete(s);
        }
    }
  }
  const fi = (i) => Math.ceil(i / 2) * 2,
    oa = (i) =>
      i !== null &&
      typeof i == "object" &&
      Object.getPrototypeOf(i) === Object.prototype &&
      Object.values(i).every((e) => typeof e == "string");
  var Je;
  (function (i) {
    ((i[(i.Silent = 0)] = "Silent"),
      (i[(i.Errors = 1)] = "Errors"),
      (i[(i.Warnings = 2)] = "Warnings"),
      (i[(i.Info = 3)] = "Info"));
  })(Je || (Je = {}));
  class L {
    constructor() {}
    static get level() {
      return L._level;
    }
    static set level(e) {
      if (
        e !== Je.Silent &&
        e !== Je.Errors &&
        e !== Je.Warnings &&
        e !== Je.Info
      )
        throw new TypeError(
          "Invalid log level. Use one of the values of the LogLevel enum.",
        );
      L._level = e;
    }
    static get _emitter() {
      return (L._emitterInstance ??= new Li());
    }
    static on(e, t, r) {
      return L._emitter.on(e, t, r);
    }
    static _error(...e) {
      (L._emitter._emit("error", e),
        L._level >= Je.Errors && console.error(...e));
    }
    static _warn(...e) {
      (L._emitter._emit("warn", e),
        L._level >= Je.Warnings && console.warn(...e));
    }
    static _info(...e) {
      (L._emitter._emit("info", e), L._level >= Je.Info && console.info(...e));
    }
  }
  ((L._level = Je.Info), (L._emitterInstance = null));
  class hi {
    constructor(e, t) {
      if (((this.data = e), (this.mimeType = t), !(e instanceof Uint8Array)))
        throw new TypeError("data must be a Uint8Array.");
      if (typeof t != "string")
        throw new TypeError("mimeType must be a string.");
    }
  }
  class Cn {
    constructor(e, t, r, n) {
      if (
        ((this.data = e),
        (this.mimeType = t),
        (this.name = r),
        (this.description = n),
        !(e instanceof Uint8Array))
      )
        throw new TypeError("data must be a Uint8Array.");
      if (t !== void 0 && typeof t != "string")
        throw new TypeError("mimeType, when provided, must be a string.");
      if (r !== void 0 && typeof r != "string")
        throw new TypeError("name, when provided, must be a string.");
      if (n !== void 0 && typeof n != "string")
        throw new TypeError("description, when provided, must be a string.");
    }
  }
  const In = (i) => {
      if (!i || typeof i != "object")
        throw new TypeError("tags must be an object.");
      if (i.title !== void 0 && typeof i.title != "string")
        throw new TypeError("tags.title, when provided, must be a string.");
      if (i.description !== void 0 && typeof i.description != "string")
        throw new TypeError(
          "tags.description, when provided, must be a string.",
        );
      if (i.artist !== void 0 && typeof i.artist != "string")
        throw new TypeError("tags.artist, when provided, must be a string.");
      if (i.album !== void 0 && typeof i.album != "string")
        throw new TypeError("tags.album, when provided, must be a string.");
      if (i.albumArtist !== void 0 && typeof i.albumArtist != "string")
        throw new TypeError(
          "tags.albumArtist, when provided, must be a string.",
        );
      if (
        i.trackNumber !== void 0 &&
        (!Number.isInteger(i.trackNumber) || i.trackNumber <= 0)
      )
        throw new TypeError(
          "tags.trackNumber, when provided, must be a positive integer.",
        );
      if (
        i.tracksTotal !== void 0 &&
        (!Number.isInteger(i.tracksTotal) || i.tracksTotal <= 0)
      )
        throw new TypeError(
          "tags.tracksTotal, when provided, must be a positive integer.",
        );
      if (
        i.discNumber !== void 0 &&
        (!Number.isInteger(i.discNumber) || i.discNumber <= 0)
      )
        throw new TypeError(
          "tags.discNumber, when provided, must be a positive integer.",
        );
      if (
        i.discsTotal !== void 0 &&
        (!Number.isInteger(i.discsTotal) || i.discsTotal <= 0)
      )
        throw new TypeError(
          "tags.discsTotal, when provided, must be a positive integer.",
        );
      if (i.genre !== void 0 && typeof i.genre != "string")
        throw new TypeError("tags.genre, when provided, must be a string.");
      if (
        i.date !== void 0 &&
        (!(i.date instanceof Date) || Number.isNaN(i.date.getTime()))
      )
        throw new TypeError("tags.date, when provided, must be a valid Date.");
      if (
        i.beatsPerMinute !== void 0 &&
        (!Number.isInteger(i.beatsPerMinute) || i.beatsPerMinute <= 0)
      )
        throw new TypeError(
          "tags.beatsPerMinute, when provided, must be a positive integer.",
        );
      if (i.lyrics !== void 0 && typeof i.lyrics != "string")
        throw new TypeError("tags.lyrics, when provided, must be a string.");
      if (i.images !== void 0) {
        if (!Array.isArray(i.images))
          throw new TypeError("tags.images, when provided, must be an array.");
        for (const e of i.images) {
          if (!e || typeof e != "object")
            throw new TypeError("Each image in tags.images must be an object.");
          if (!(e.data instanceof Uint8Array))
            throw new TypeError("Each image.data must be a Uint8Array.");
          if (typeof e.mimeType != "string")
            throw new TypeError("Each image.mimeType must be a string.");
          if (!["coverFront", "coverBack", "unknown"].includes(e.kind))
            throw new TypeError(
              "Each image.kind must be 'coverFront', 'coverBack', or 'unknown'.",
            );
        }
      }
      if (i.comment !== void 0 && typeof i.comment != "string")
        throw new TypeError("tags.comment, when provided, must be a string.");
      if (i.raw !== void 0) {
        if (!i.raw || typeof i.raw != "object")
          throw new TypeError("tags.raw, when provided, must be an object.");
        for (const e of Object.values(i.raw))
          if (
            e !== null &&
            typeof e != "string" &&
            !(Array.isArray(e) && e.every((t) => typeof t == "string")) &&
            !(e instanceof Uint8Array) &&
            !(e instanceof hi) &&
            !(e instanceof Cn) &&
            !oa(e)
          )
            throw new TypeError(
              "Each value in tags.raw must be a string, string array, Uint8Array, RichImageData, AttachedFile, Record<string, string>, or null.",
            );
      }
    },
    kl = (i) =>
      i.title === void 0 &&
      i.description === void 0 &&
      i.artist === void 0 &&
      i.album === void 0 &&
      i.albumArtist === void 0 &&
      i.trackNumber === void 0 &&
      i.tracksTotal === void 0 &&
      i.discNumber === void 0 &&
      i.discsTotal === void 0 &&
      i.genre === void 0 &&
      i.beatsPerMinute === void 0 &&
      i.date === void 0 &&
      i.lyrics === void 0 &&
      (!i.images || i.images.length === 0) &&
      i.comment === void 0 &&
      (i.raw === void 0 || Object.keys(i.raw).length === 0),
    wt = {
      default: !0,
      primary: !0,
      forced: !1,
      original: !1,
      commentary: !1,
      hearingImpaired: !1,
      visuallyImpaired: !1,
    },
    Tl = (i) => {
      if (!i || typeof i != "object")
        throw new TypeError("disposition must be an object.");
      if (i.default !== void 0 && typeof i.default != "boolean")
        throw new TypeError("disposition.default must be a boolean.");
      if (i.primary !== void 0 && typeof i.primary != "boolean")
        throw new TypeError("disposition.primary must be a boolean.");
      if (i.forced !== void 0 && typeof i.forced != "boolean")
        throw new TypeError("disposition.forced must be a boolean.");
      if (i.original !== void 0 && typeof i.original != "boolean")
        throw new TypeError("disposition.original must be a boolean.");
      if (i.commentary !== void 0 && typeof i.commentary != "boolean")
        throw new TypeError("disposition.commentary must be a boolean.");
      if (i.hearingImpaired !== void 0 && typeof i.hearingImpaired != "boolean")
        throw new TypeError("disposition.hearingImpaired must be a boolean.");
      if (
        i.visuallyImpaired !== void 0 &&
        typeof i.visuallyImpaired != "boolean"
      )
        throw new TypeError("disposition.visuallyImpaired must be a boolean.");
    };
  class $ {
    constructor(e) {
      ((this.bytes = e), (this.pos = 0));
    }
    seekToByte(e) {
      this.pos = 8 * e;
    }
    readBit() {
      const e = Math.floor(this.pos / 8),
        t = this.bytes[e] ?? 0,
        r = 7 - (this.pos & 7),
        n = (t & (1 << r)) >> r;
      return (this.pos++, n);
    }
    readBits(e) {
      if (e === 1) return this.readBit();
      let t = 0;
      for (let r = 0; r < e; r++) ((t <<= 1), (t |= this.readBit()));
      return t;
    }
    writeBits(e, t) {
      const r = this.pos + e;
      for (let n = this.pos; n < r; n++) {
        const s = Math.floor(n / 8);
        let a = this.bytes[s];
        const o = 7 - (n & 7);
        ((a &= ~(1 << o)),
          (a |= ((t & (1 << (r - n - 1))) >> (r - n - 1)) << o),
          (this.bytes[s] = a));
      }
      this.pos = r;
    }
    copyBits(e, t) {
      let r = 0;
      for (r; r < e - 7; r += 8) this.writeBits(8, t.readBits(8));
      const n = e - r;
      n > 0 && this.writeBits(n, t.readBits(n));
    }
    readAlignedByte() {
      if (this.pos % 8 !== 0) throw new Error("Bitstream is not byte-aligned.");
      const e = this.pos / 8,
        t = this.bytes[e] ?? 0;
      return ((this.pos += 8), t);
    }
    skipBits(e) {
      this.pos += e;
    }
    getBitsLeft() {
      return this.bytes.length * 8 - this.pos;
    }
    clone() {
      const e = new $(this.bytes);
      return ((e.pos = this.pos), e);
    }
  }
  const At = [
      96e3, 88200, 64e3, 48e3, 44100, 32e3, 24e3, 22050, 16e3, 12e3, 11025, 8e3,
      7350,
    ],
    mi = [-1, 1, 2, 3, 4, 5, 6, 8],
    En = (i) => {
      if (!i || i.byteLength < 2)
        throw new TypeError("AAC description must be at least 2 bytes long.");
      const e = new $(i),
        t = _n(e),
        { frequencyIndex: r, sampleRate: n } = vn(e),
        s = e.readBits(4);
      let a = null;
      s >= 1 && s <= 7 && (a = mi[s]);
      let o = t,
        c = !1,
        l = n;
      if (t === 5 || t === 29)
        ((c = t === 29),
          (l = vn(e).sampleRate),
          (o = _n(e)),
          o === 22 && e.skipBits(4));
      else
        for (; e.getBitsLeft() > 15;) {
          const u = e.pos;
          if (e.readBits(11) !== 695) {
            e.pos = u + 1;
            continue;
          }
          _n(e) === 5 &&
            e.readBits(1) &&
            ((l = vn(e).sampleRate),
            e.getBitsLeft() > 11 &&
              e.readBits(11) === 1352 &&
              (c = !!e.readBits(1)));
          break;
        }
      return (
        a !== null && a > 1 && (c = !1),
        {
          objectType: t,
          coreObjectType: o,
          frequencyIndex: r,
          channelConfiguration: s,
          outputSampleRate: l,
          outputNumberOfChannels: c && a === 1 ? 2 : a,
        }
      );
    },
    _n = (i) => {
      const e = i.readBits(5);
      return e === 31 ? 32 + i.readBits(6) : e;
    },
    vn = (i) => {
      const e = i.readBits(4);
      return e === 15
        ? { frequencyIndex: e, sampleRate: i.readBits(24) }
        : { frequencyIndex: e, sampleRate: e < At.length ? At[e] : null };
    },
    An = (i) => {
      const e = i.objectType === 5 || i.objectType === 29,
        t = i.objectType === 29,
        r = e ? i.outputSampleRate / 2 : i.outputSampleRate,
        n = t ? 1 : i.outputNumberOfChannels,
        s = mi.indexOf(n);
      if (s === -1)
        throw new TypeError(
          `Unsupported number of channels: ${i.outputNumberOfChannels}`,
        );
      let a = 16;
      (i.objectType >= 32 && (a += 6),
        Bn(r) === 15 && (a += 24),
        e && ((a += 9), Bn(i.outputSampleRate) === 15 && (a += 24)));
      const o = Math.ceil(a / 8),
        c = new Uint8Array(o),
        l = new $(c);
      return (
        ca(l, i.objectType),
        la(l, r),
        l.writeBits(4, s),
        e && (la(l, i.outputSampleRate), ca(l, 2)),
        l.writeBits(3, 0),
        c
      );
    },
    ca = (i, e) => {
      e < 32 ? i.writeBits(5, e) : (i.writeBits(5, 31), i.writeBits(6, e - 32));
    },
    la = (i, e) => {
      const t = Bn(e);
      (i.writeBits(4, t), t === 15 && i.writeBits(24, e));
    },
    Bn = (i) => {
      const e = At.indexOf(i);
      return e === -1 ? 15 : e;
    };
  const Pr = [48e3, 44100, 32e3],
    ua = [24e3, 22050, 16e3];
  const Hi = "Mediabunny v1.61.3";
  var be;
  (function (i) {
    ((i[(i.NON_IDR_SLICE = 1)] = "NON_IDR_SLICE"),
      (i[(i.SLICE_DPA = 2)] = "SLICE_DPA"),
      (i[(i.SLICE_DPB = 3)] = "SLICE_DPB"),
      (i[(i.SLICE_DPC = 4)] = "SLICE_DPC"),
      (i[(i.IDR = 5)] = "IDR"),
      (i[(i.SEI = 6)] = "SEI"),
      (i[(i.SPS = 7)] = "SPS"),
      (i[(i.PPS = 8)] = "PPS"),
      (i[(i.AUD = 9)] = "AUD"),
      (i[(i.SPS_EXT = 13)] = "SPS_EXT"));
  })(be || (be = {}));
  var de;
  (function (i) {
    ((i[(i.RASL_N = 8)] = "RASL_N"),
      (i[(i.RASL_R = 9)] = "RASL_R"),
      (i[(i.BLA_W_LP = 16)] = "BLA_W_LP"),
      (i[(i.RSV_IRAP_VCL23 = 23)] = "RSV_IRAP_VCL23"),
      (i[(i.VPS_NUT = 32)] = "VPS_NUT"),
      (i[(i.SPS_NUT = 33)] = "SPS_NUT"),
      (i[(i.PPS_NUT = 34)] = "PPS_NUT"),
      (i[(i.AUD_NUT = 35)] = "AUD_NUT"),
      (i[(i.PREFIX_SEI_NUT = 39)] = "PREFIX_SEI_NUT"),
      (i[(i.SUFFIX_SEI_NUT = 40)] = "SUFFIX_SEI_NUT"));
  })(de || (de = {}));
  const qi = function* (i) {
      let e = 0,
        t = -1;
      for (; e < i.length - 2;) {
        const r = i.indexOf(0, e);
        if (r === -1 || r >= i.length - 2) break;
        e = r;
        let n = 0;
        if (
          (e + 3 < i.length &&
          i[e + 1] === 0 &&
          i[e + 2] === 0 &&
          i[e + 3] === 1
            ? (n = 4)
            : i[e + 1] === 0 && i[e + 2] === 1 && (n = 3),
          n === 0)
        ) {
          e++;
          continue;
        }
        (t !== -1 && e > t && (yield { offset: t, length: e - t }),
          (t = e + n),
          (e = t));
      }
      t !== -1 && t < i.length && (yield { offset: t, length: i.length - t });
    },
    da = function* (i, e) {
      let t = 0;
      const r = new DataView(i.buffer, i.byteOffset, i.byteLength);
      for (; t + e <= i.length;) {
        let n;
        (e === 1
          ? (n = r.getUint8(t))
          : e === 2
            ? (n = r.getUint16(t, !1))
            : e === 3
              ? (n = Tr(r, t, !1))
              : (g(e === 4), (n = r.getUint32(t, !1))),
          (t += e),
          yield { offset: t, length: n },
          (t += n));
      }
    },
    fa = (i, e) => {
      if (e.description) {
        const n = (me(e.description)[4] & 3) + 1;
        return da(i, n);
      } else return qi(i);
    },
    ji = (i) => i & 31,
    Cr = (i) => {
      const e = [],
        t = i.length;
      for (let r = 0; r < t; r++)
        r + 2 < t && i[r] === 0 && i[r + 1] === 0 && i[r + 2] === 3
          ? (e.push(0, 0), (r += 2))
          : e.push(i[r]);
      return new Uint8Array(e);
    },
    Sl = (i) => {
      const e = [];
      let t = 0;
      for (const r of i)
        (t === 2 && r <= 3 && (e.push(3), (t = 0)),
          e.push(r),
          (t = r === 0 ? t + 1 : 0));
      return new Uint8Array(e);
    },
    Fn = new Uint8Array([0, 0, 0, 1]),
    ha = (i) => {
      const e = i.reduce((n, s) => n + Fn.byteLength + s.byteLength, 0),
        t = new Uint8Array(e);
      let r = 0;
      for (const n of i)
        (t.set(Fn, r), (r += Fn.byteLength), t.set(n, r), (r += n.byteLength));
      return t;
    },
    Rn = (i, e) => {
      const t = i.reduce((s, a) => s + e + a.byteLength, 0),
        r = new Uint8Array(t);
      let n = 0;
      for (const s of i) {
        const a = new DataView(r.buffer, r.byteOffset, r.byteLength);
        switch (e) {
          case 1:
            a.setUint8(n, s.byteLength);
            break;
          case 2:
            a.setUint16(n, s.byteLength, !1);
            break;
          case 3:
            mn(a, n, s.byteLength, !1);
            break;
          case 4:
            a.setUint32(n, s.byteLength, !1);
            break;
        }
        ((n += e), r.set(s, n), (n += s.byteLength));
      }
      return r;
    },
    xl = (i, e) => {
      if (e.description) {
        const n = (me(e.description)[4] & 3) + 1;
        return Rn(i, n);
      } else return ha(i);
    },
    $i = (i) => {
      try {
        const e = [],
          t = [],
          r = [];
        for (const o of qi(i)) {
          const c = i.subarray(o.offset, o.offset + o.length),
            l = ji(c[0]);
          l === be.SPS
            ? e.push(c)
            : l === be.PPS
              ? t.push(c)
              : l === be.SPS_EXT && r.push(c);
        }
        if (e.length === 0 || t.length === 0) return null;
        const n = e[0],
          s = pi(n);
        g(s !== null);
        const a =
          s.profileIdc === 100 ||
          s.profileIdc === 110 ||
          s.profileIdc === 122 ||
          s.profileIdc === 144;
        return {
          configurationVersion: 1,
          avcProfileIndication: s.profileIdc,
          profileCompatibility: s.constraintFlags,
          avcLevelIndication: s.levelIdc,
          lengthSizeMinusOne: 3,
          sequenceParameterSets: e,
          pictureParameterSets: t,
          chromaFormat: a ? s.chromaFormatIdc : null,
          bitDepthLumaMinus8: a ? s.bitDepthLumaMinus8 : null,
          bitDepthChromaMinus8: a ? s.bitDepthChromaMinus8 : null,
          sequenceParameterSetExt: a ? r : null,
        };
      } catch (e) {
        return (
          L._error("Error building AVC Decoder Configuration Record:", e),
          null
        );
      }
    },
    ma = (i) => {
      const e = [];
      (e.push(i.configurationVersion),
        e.push(i.avcProfileIndication),
        e.push(i.profileCompatibility),
        e.push(i.avcLevelIndication),
        e.push(252 | (i.lengthSizeMinusOne & 3)),
        e.push(224 | (i.sequenceParameterSets.length & 31)));
      for (const t of i.sequenceParameterSets) {
        const r = t.byteLength;
        (e.push(r >> 8), e.push(r & 255));
        for (let n = 0; n < r; n++) e.push(t[n]);
      }
      e.push(i.pictureParameterSets.length);
      for (const t of i.pictureParameterSets) {
        const r = t.byteLength;
        (e.push(r >> 8), e.push(r & 255));
        for (let n = 0; n < r; n++) e.push(t[n]);
      }
      if (
        (i.avcProfileIndication === 100 ||
          i.avcProfileIndication === 110 ||
          i.avcProfileIndication === 122 ||
          i.avcProfileIndication === 144) &&
        i.chromaFormat !== null
      ) {
        (g(i.bitDepthLumaMinus8 !== null),
          g(i.bitDepthChromaMinus8 !== null),
          g(i.sequenceParameterSetExt !== null),
          e.push(252 | (i.chromaFormat & 3)),
          e.push(248 | (i.bitDepthLumaMinus8 & 7)),
          e.push(248 | (i.bitDepthChromaMinus8 & 7)),
          e.push(i.sequenceParameterSetExt.length));
        for (const t of i.sequenceParameterSetExt) {
          const r = t.byteLength;
          (e.push(r >> 8), e.push(r & 255));
          for (let n = 0; n < r; n++) e.push(t[n]);
        }
      }
      return new Uint8Array(e);
    },
    Mn = (i) => {
      try {
        const e = j(i);
        let t = 0;
        const r = e.getUint8(t++),
          n = e.getUint8(t++),
          s = e.getUint8(t++),
          a = e.getUint8(t++),
          o = e.getUint8(t++) & 3,
          c = e.getUint8(t++) & 31,
          l = [];
        for (let h = 0; h < c; h++) {
          const p = e.getUint16(t, !1);
          ((t += 2), l.push(i.subarray(t, t + p)), (t += p));
        }
        const u = e.getUint8(t++),
          d = [];
        for (let h = 0; h < u; h++) {
          const p = e.getUint16(t, !1);
          ((t += 2), d.push(i.subarray(t, t + p)), (t += p));
        }
        const f = {
          configurationVersion: r,
          avcProfileIndication: n,
          profileCompatibility: s,
          avcLevelIndication: a,
          lengthSizeMinusOne: o,
          sequenceParameterSets: l,
          pictureParameterSets: d,
          chromaFormat: null,
          bitDepthLumaMinus8: null,
          bitDepthChromaMinus8: null,
          sequenceParameterSetExt: null,
        };
        if (
          (n === 100 || n === 110 || n === 122 || n === 144) &&
          t + 4 <= i.length
        ) {
          const h = e.getUint8(t++) & 3,
            p = e.getUint8(t++) & 7,
            m = e.getUint8(t++) & 7,
            y = e.getUint8(t++);
          ((f.chromaFormat = h),
            (f.bitDepthLumaMinus8 = p),
            (f.bitDepthChromaMinus8 = m));
          const w = [];
          for (let b = 0; b < y; b++) {
            const T = e.getUint16(t, !1);
            ((t += 2), w.push(i.subarray(t, t + T)), (t += T));
          }
          f.sequenceParameterSetExt = w;
        }
        return f;
      } catch (e) {
        return (
          L._error("Error deserializing AVC Decoder Configuration Record:", e),
          null
        );
      }
    },
    pa = {
      1: { num: 1, den: 1 },
      2: { num: 12, den: 11 },
      3: { num: 10, den: 11 },
      4: { num: 16, den: 11 },
      5: { num: 40, den: 33 },
      6: { num: 24, den: 11 },
      7: { num: 20, den: 11 },
      8: { num: 32, den: 11 },
      9: { num: 80, den: 33 },
      10: { num: 18, den: 11 },
      11: { num: 15, den: 11 },
      12: { num: 64, den: 33 },
      13: { num: 160, den: 99 },
      14: { num: 4, den: 3 },
      15: { num: 3, den: 2 },
      16: { num: 2, den: 1 },
    },
    pi = (i) => {
      try {
        const e = Cr(i),
          t = new $(e);
        if ((t.skipBits(1), t.skipBits(2), t.readBits(5) !== 7)) return null;
        const n = t.readAlignedByte(),
          s = t.readAlignedByte(),
          a = t.readAlignedByte();
        M(t);
        let o = 1,
          c = 0,
          l = 0,
          u = 0;
        if (
          (n === 100 ||
            n === 110 ||
            n === 122 ||
            n === 244 ||
            n === 44 ||
            n === 83 ||
            n === 86 ||
            n === 118 ||
            n === 128) &&
          ((o = M(t)),
          o === 3 && (u = t.readBits(1)),
          (c = M(t)),
          (l = M(t)),
          t.skipBits(1),
          t.readBits(1))
        ) {
          for (let H = 0; H < (o !== 3 ? 8 : 12); H++)
            if (t.readBits(1)) {
              const he = H < 6 ? 16 : 64;
              let Y = 8,
                G = 8;
              for (let ie = 0; ie < he; ie++) {
                if (G !== 0) {
                  const re = dt(t);
                  G = (Y + re + 256) % 256;
                }
                Y = G === 0 ? Y : G;
              }
            }
        }
        M(t);
        const d = M(t);
        if (d === 0) M(t);
        else if (d === 1) {
          (t.skipBits(1), dt(t), dt(t));
          const O = M(t);
          for (let H = 0; H < O; H++) dt(t);
        }
        (M(t), t.skipBits(1));
        const f = M(t),
          h = M(t),
          p = 16 * (f + 1),
          m = 16 * (h + 1);
        let y = p,
          w = m;
        const b = t.readBits(1);
        if ((b || t.skipBits(1), t.skipBits(1), t.readBits(1))) {
          const O = M(t),
            H = M(t),
            ee = M(t),
            he = M(t);
          let Y, G;
          if ((u === 0 ? o : 0) === 0) ((Y = 1), (G = 2 - b));
          else {
            const re = o === 3 ? 1 : 2,
              we = o === 1 ? 2 : 1;
            ((Y = re), (G = we * (2 - b)));
          }
          ((y -= Y * (O + H)), (w -= G * (ee + he)));
        }
        let S = 2,
          k = 2,
          P = 2,
          I = 0,
          E = { num: 1, den: 1 },
          C = null,
          _ = null,
          v = null,
          F = null;
        const D = t.pos;
        if (t.readBits(1)) {
          if (t.readBits(1)) {
            const re = t.readBits(8);
            if (re === 255) E = { num: t.readBits(16), den: t.readBits(16) };
            else {
              const we = pa[re];
              we && (E = we);
            }
          }
          (t.readBits(1) && t.skipBits(1),
            t.readBits(1) &&
              (t.skipBits(3),
              (I = t.readBits(1)),
              t.readBits(1) &&
                ((S = t.readBits(8)),
                (k = t.readBits(8)),
                (P = t.readBits(8)))),
            t.readBits(1) && (M(t), M(t)),
            t.readBits(1) && (t.skipBits(32), t.skipBits(32), t.skipBits(1)));
          const G = t.readBits(1);
          G && ga(t);
          const ie = t.readBits(1);
          (ie && ga(t),
            (G || ie) && t.skipBits(1),
            t.skipBits(1),
            (v = t.pos),
            (F = t.readBits(1)),
            F &&
              (t.skipBits(1), M(t), M(t), M(t), M(t), (C = M(t)), (_ = M(t))));
        }
        if (C === null) {
          g(_ === null);
          const O = s & 16;
          if (
            (n === 44 ||
              n === 86 ||
              n === 100 ||
              n === 110 ||
              n === 122 ||
              n === 244) &&
            O
          )
            ((C = 0), (_ = 0));
          else {
            const H = f + 1,
              ee = h + 1,
              he = (2 - b) * ee,
              Y = Rr.find((ie) => ie.level >= a) ?? ne(Rr),
              G = Math.min(Math.floor(Y.maxDpbMbs / (H * he)), 16);
            ((C = G), (_ = G));
          }
        }
        return (
          g(_ !== null),
          {
            emulationUnpreventedBytes: e,
            profileIdc: n,
            constraintFlags: s,
            levelIdc: a,
            frameMbsOnlyFlag: b,
            chromaFormatIdc: o,
            bitDepthLumaMinus8: c,
            bitDepthChromaMinus8: l,
            codedWidth: p,
            codedHeight: m,
            displayWidth: y,
            displayHeight: w,
            pixelAspectRatio: E,
            colourPrimaries: S,
            matrixCoefficients: P,
            transferCharacteristics: k,
            fullRangeFlag: I,
            numReorderFrames: C,
            maxDecFrameBuffering: _,
            vuiParametersFlagBitOffset: D,
            bitstreamRestrictionFlagBitOffset: v,
            bitstreamRestrictionFlag: F,
          }
        );
      } catch (e) {
        return (L._error("Error parsing AVC SPS:", e), null);
      }
    },
    ga = (i) => {
      const e = M(i);
      (i.skipBits(4), i.skipBits(4));
      for (let t = 0; t <= e; t++) (M(i), M(i), i.skipBits(1));
      (i.skipBits(5), i.skipBits(5), i.skipBits(5), i.skipBits(5));
    },
    wa = (i) => {
      g(i.bitstreamRestrictionFlag !== 1);
      const e = new Uint8Array(i.emulationUnpreventedBytes.byteLength + 64),
        t = new $(i.emulationUnpreventedBytes),
        r = new $(e);
      (i.bitstreamRestrictionFlag === null
        ? (r.copyBits(i.vuiParametersFlagBitOffset, t),
          r.writeBits(1, 1),
          r.writeBits(1, 0),
          r.writeBits(1, 0),
          r.writeBits(1, 0),
          r.writeBits(1, 0),
          r.writeBits(1, 0),
          r.writeBits(1, 0),
          r.writeBits(1, 0),
          r.writeBits(1, 0))
        : (g(i.bitstreamRestrictionFlagBitOffset !== null),
          r.copyBits(i.bitstreamRestrictionFlagBitOffset, t)),
        r.writeBits(1, 1),
        r.writeBits(1, 1),
        oi(r, 2),
        oi(r, 1),
        oi(r, 16),
        oi(r, 16),
        oi(r, i.numReorderFrames),
        oi(r, i.maxDecFrameBuffering),
        r.writeBits(1, 1),
        r.writeBits((8 - (r.pos % 8)) % 8, 0));
      const n = r.pos / 8;
      return (g(Number.isInteger(n)), Sl(e.subarray(0, n)));
    },
    Pl = (i, e) => {
      if (e.description) {
        const n = (me(e.description)[21] & 3) + 1;
        return Rn(i, n);
      } else return ha(i);
    },
    Ir = (i, e) => {
      if (e.description) {
        const n = (me(e.description)[21] & 3) + 1;
        return da(i, n);
      } else return qi(i);
    },
    gi = (i) => (i >> 1) & 63,
    Er = (i) => {
      try {
        const e = new $(Cr(i));
        (e.skipBits(16), e.readBits(4));
        const t = e.readBits(3),
          r = e.readBits(1),
          {
            general_profile_space: n,
            general_tier_flag: s,
            general_profile_idc: a,
            general_profile_compatibility_flags: o,
            general_constraint_indicator_flags: c,
            general_level_idc: l,
          } = Cl(e, t);
        M(e);
        const u = M(e);
        let d = 0;
        u === 3 && (d = e.readBits(1));
        const f = M(e),
          h = M(e);
        let p = f,
          m = h;
        if (e.readBits(1)) {
          const F = M(e),
            D = M(e),
            z = M(e),
            O = M(e);
          let H = 1,
            ee = 1;
          const he = d === 0 ? u : 0;
          (he === 1 ? ((H = 2), (ee = 2)) : he === 2 && ((H = 2), (ee = 1)),
            (p -= (F + D) * H),
            (m -= (z + O) * ee));
        }
        const y = M(e),
          w = M(e);
        M(e);
        const T = e.readBits(1) ? 0 : t;
        let S = 0;
        for (let F = T; F <= t; F++) (M(e), (S = M(e)), M(e));
        (M(e),
          M(e),
          M(e),
          M(e),
          M(e),
          M(e),
          e.readBits(1) && e.readBits(1) && Il(e),
          e.skipBits(1),
          e.skipBits(1),
          e.readBits(1) &&
            (e.skipBits(4), e.skipBits(4), M(e), M(e), e.skipBits(1)));
        const k = M(e);
        if ((El(e, k), e.readBits(1))) {
          const F = M(e);
          for (let D = 0; D < F; D++) (M(e), e.skipBits(1));
        }
        (e.skipBits(1), e.skipBits(1));
        let P = 2,
          I = 2,
          E = 2,
          C = 0,
          _ = 0,
          v = { num: 1, den: 1 };
        if (e.readBits(1)) {
          const F = vl(e, t);
          ((v = F.pixelAspectRatio),
            (P = F.colourPrimaries),
            (I = F.transferCharacteristics),
            (E = F.matrixCoefficients),
            (C = F.fullRangeFlag),
            (_ = F.minSpatialSegmentationIdc));
        }
        return {
          displayWidth: p,
          displayHeight: m,
          pixelAspectRatio: v,
          colourPrimaries: P,
          transferCharacteristics: I,
          matrixCoefficients: E,
          fullRangeFlag: C,
          maxDecFrameBuffering: S + 1,
          spsMaxSubLayersMinus1: t,
          spsTemporalIdNestingFlag: r,
          generalProfileSpace: n,
          generalTierFlag: s,
          generalProfileIdc: a,
          generalProfileCompatibilityFlags: o,
          generalConstraintIndicatorFlags: c,
          generalLevelIdc: l,
          chromaFormatIdc: u,
          bitDepthLumaMinus8: y,
          bitDepthChromaMinus8: w,
          minSpatialSegmentationIdc: _,
        };
      } catch (e) {
        return (L._error("Error parsing HEVC SPS:", e), null);
      }
    },
    Ki = (i) => {
      try {
        const e = [],
          t = [],
          r = [],
          n = [];
        for (const l of qi(i)) {
          const u = i.subarray(l.offset, l.offset + l.length),
            d = gi(u[0]);
          d === de.VPS_NUT
            ? e.push(u)
            : d === de.SPS_NUT
              ? t.push(u)
              : d === de.PPS_NUT
                ? r.push(u)
                : (d === de.PREFIX_SEI_NUT || d === de.SUFFIX_SEI_NUT) &&
                  n.push(u);
        }
        if (t.length === 0 || r.length === 0) return null;
        const s = Er(t[0]);
        if (!s) return null;
        let a = 0;
        if (r.length > 0) {
          const l = r[0],
            u = new $(Cr(l));
          (u.skipBits(16),
            M(u),
            M(u),
            u.skipBits(1),
            u.skipBits(1),
            u.skipBits(3),
            u.skipBits(1),
            u.skipBits(1),
            M(u),
            M(u),
            dt(u),
            u.skipBits(1),
            u.skipBits(1),
            u.readBits(1) && M(u),
            dt(u),
            dt(u),
            u.skipBits(1),
            u.skipBits(1),
            u.skipBits(1),
            u.skipBits(1));
          const d = u.readBits(1),
            f = u.readBits(1);
          !d && !f ? (a = 0) : d && !f ? (a = 2) : !d && f ? (a = 3) : (a = 0);
        }
        const o = [
          ...(e.length
            ? [{ arrayCompleteness: 1, nalUnitType: de.VPS_NUT, nalUnits: e }]
            : []),
          ...(t.length
            ? [{ arrayCompleteness: 1, nalUnitType: de.SPS_NUT, nalUnits: t }]
            : []),
          ...(r.length
            ? [{ arrayCompleteness: 1, nalUnitType: de.PPS_NUT, nalUnits: r }]
            : []),
          ...(n.length
            ? [{ arrayCompleteness: 1, nalUnitType: gi(n[0][0]), nalUnits: n }]
            : []),
        ];
        return {
          configurationVersion: 1,
          generalProfileSpace: s.generalProfileSpace,
          generalTierFlag: s.generalTierFlag,
          generalProfileIdc: s.generalProfileIdc,
          generalProfileCompatibilityFlags: s.generalProfileCompatibilityFlags,
          generalConstraintIndicatorFlags: s.generalConstraintIndicatorFlags,
          generalLevelIdc: s.generalLevelIdc,
          minSpatialSegmentationIdc: s.minSpatialSegmentationIdc,
          parallelismType: a,
          chromaFormatIdc: s.chromaFormatIdc,
          bitDepthLumaMinus8: s.bitDepthLumaMinus8,
          bitDepthChromaMinus8: s.bitDepthChromaMinus8,
          avgFrameRate: 0,
          constantFrameRate: 0,
          numTemporalLayers: s.spsMaxSubLayersMinus1 + 1,
          temporalIdNested: s.spsTemporalIdNestingFlag,
          lengthSizeMinusOne: 3,
          arrays: o,
        };
      } catch (e) {
        return (
          L._error("Error building HEVC Decoder Configuration Record:", e),
          null
        );
      }
    },
    Cl = (i, e) => {
      const t = i.readBits(2),
        r = i.readBits(1),
        n = i.readBits(5);
      let s = 0;
      for (let u = 0; u < 32; u++) s = (s << 1) | i.readBits(1);
      const a = new Uint8Array(6);
      for (let u = 0; u < 6; u++) a[u] = i.readBits(8);
      const o = i.readBits(8),
        c = [],
        l = [];
      for (let u = 0; u < e; u++)
        (c.push(i.readBits(1)), l.push(i.readBits(1)));
      if (e > 0) for (let u = e; u < 8; u++) i.skipBits(2);
      for (let u = 0; u < e; u++)
        (c[u] && i.skipBits(88), l[u] && i.skipBits(8));
      return {
        general_profile_space: t,
        general_tier_flag: r,
        general_profile_idc: n,
        general_profile_compatibility_flags: s,
        general_constraint_indicator_flags: a,
        general_level_idc: o,
      };
    },
    Il = (i) => {
      for (let e = 0; e < 4; e++)
        for (let t = 0; t < (e === 3 ? 2 : 6); t++)
          if (!i.readBits(1)) M(i);
          else {
            const n = Math.min(64, 1 << (4 + (e << 1)));
            e > 1 && dt(i);
            for (let s = 0; s < n; s++) dt(i);
          }
    },
    El = (i, e) => {
      const t = [];
      for (let r = 0; r < e; r++) t[r] = _l(i, r, e, t);
    },
    _l = (i, e, t, r) => {
      let n = 0,
        s = 0,
        a = 0;
      if ((e !== 0 && (s = i.readBits(1)), s)) {
        if (e === t) {
          const c = M(i);
          a = e - (c + 1);
        } else a = e - 1;
        (i.readBits(1), M(i));
        const o = r[a] ?? 0;
        for (let c = 0; c <= o; c++) i.readBits(1) || i.readBits(1);
        n = r[a];
      } else {
        const o = M(i),
          c = M(i);
        for (let l = 0; l < o; l++) (M(i), i.readBits(1));
        for (let l = 0; l < c; l++) (M(i), i.readBits(1));
        n = o + c;
      }
      return n;
    },
    vl = (i, e) => {
      let t = 2,
        r = 2,
        n = 2,
        s = 0,
        a = 0,
        o = { num: 1, den: 1 };
      if (i.readBits(1)) {
        const c = i.readBits(8);
        if (c === 255) o = { num: i.readBits(16), den: i.readBits(16) };
        else {
          const l = pa[c];
          l && (o = l);
        }
      }
      return (
        i.readBits(1) && i.readBits(1),
        i.readBits(1) &&
          (i.readBits(3),
          (s = i.readBits(1)),
          i.readBits(1) &&
            ((t = i.readBits(8)), (r = i.readBits(8)), (n = i.readBits(8)))),
        i.readBits(1) && (M(i), M(i)),
        i.readBits(1),
        i.readBits(1),
        i.readBits(1),
        i.readBits(1) && (M(i), M(i), M(i), M(i)),
        i.readBits(1) &&
          (i.readBits(32),
          i.readBits(32),
          i.readBits(1) && M(i),
          i.readBits(1) && Al(i, !0, e)),
        i.readBits(1) &&
          (i.readBits(1),
          i.readBits(1),
          i.readBits(1),
          (a = M(i)),
          M(i),
          M(i),
          M(i),
          M(i)),
        {
          pixelAspectRatio: o,
          colourPrimaries: t,
          transferCharacteristics: r,
          matrixCoefficients: n,
          fullRangeFlag: s,
          minSpatialSegmentationIdc: a,
        }
      );
    },
    Al = (i, e, t) => {
      let r = !1,
        n = !1,
        s = !1;
      ((r = i.readBits(1) === 1),
        (n = i.readBits(1) === 1),
        (r || n) &&
          ((s = i.readBits(1) === 1),
          s && (i.readBits(8), i.readBits(5), i.readBits(1), i.readBits(5)),
          i.readBits(4),
          i.readBits(4),
          s && i.readBits(4),
          i.readBits(5),
          i.readBits(5),
          i.readBits(5)));
      for (let a = 0; a <= t; a++) {
        const o = i.readBits(1) === 1;
        let c = !0;
        o || (c = i.readBits(1) === 1);
        let l = !1;
        c ? M(i) : (l = i.readBits(1) === 1);
        let u = 1;
        (l || (u = M(i) + 1), r && ya(i, u, s), n && ya(i, u, s));
      }
    },
    ya = (i, e, t) => {
      for (let r = 0; r < e; r++)
        (M(i), M(i), t && (M(i), M(i)), i.readBits(1));
    },
    Bl = (i) => {
      const e = [];
      (e.push(i.configurationVersion),
        e.push(
          ((i.generalProfileSpace & 3) << 6) |
            ((i.generalTierFlag & 1) << 5) |
            (i.generalProfileIdc & 31),
        ),
        e.push((i.generalProfileCompatibilityFlags >>> 24) & 255),
        e.push((i.generalProfileCompatibilityFlags >>> 16) & 255),
        e.push((i.generalProfileCompatibilityFlags >>> 8) & 255),
        e.push(i.generalProfileCompatibilityFlags & 255),
        e.push(...i.generalConstraintIndicatorFlags),
        e.push(i.generalLevelIdc & 255),
        e.push(240 | ((i.minSpatialSegmentationIdc >> 8) & 15)),
        e.push(i.minSpatialSegmentationIdc & 255),
        e.push(252 | (i.parallelismType & 3)),
        e.push(252 | (i.chromaFormatIdc & 3)),
        e.push(248 | (i.bitDepthLumaMinus8 & 7)),
        e.push(248 | (i.bitDepthChromaMinus8 & 7)),
        e.push((i.avgFrameRate >> 8) & 255),
        e.push(i.avgFrameRate & 255),
        e.push(
          ((i.constantFrameRate & 3) << 6) |
            ((i.numTemporalLayers & 7) << 3) |
            ((i.temporalIdNested & 1) << 2) |
            (i.lengthSizeMinusOne & 3),
        ),
        e.push(i.arrays.length & 255));
      for (const t of i.arrays) {
        (e.push(((t.arrayCompleteness & 1) << 7) | 0 | (t.nalUnitType & 63)),
          e.push((t.nalUnits.length >> 8) & 255),
          e.push(t.nalUnits.length & 255));
        for (const r of t.nalUnits) {
          (e.push((r.length >> 8) & 255), e.push(r.length & 255));
          for (let n = 0; n < r.length; n++) e.push(r[n]);
        }
      }
      return new Uint8Array(e);
    },
    ba = (i) => {
      try {
        const e = j(i);
        let t = 0;
        const r = e.getUint8(t++),
          n = e.getUint8(t++),
          s = (n >> 6) & 3,
          a = (n >> 5) & 1,
          o = n & 31,
          c = e.getUint32(t, !1);
        t += 4;
        const l = i.subarray(t, t + 6);
        t += 6;
        const u = e.getUint8(t++),
          d = ((e.getUint8(t++) & 15) << 8) | e.getUint8(t++),
          f = e.getUint8(t++) & 3,
          h = e.getUint8(t++) & 3,
          p = e.getUint8(t++) & 7,
          m = e.getUint8(t++) & 7,
          y = e.getUint16(t, !1);
        t += 2;
        const w = e.getUint8(t++),
          b = (w >> 6) & 3,
          T = (w >> 3) & 7,
          S = (w >> 2) & 1,
          k = w & 3,
          P = e.getUint8(t++),
          I = [];
        for (let E = 0; E < P; E++) {
          const C = e.getUint8(t++),
            _ = (C >> 7) & 1,
            v = C & 63,
            F = e.getUint16(t, !1);
          t += 2;
          const D = [];
          for (let z = 0; z < F; z++) {
            const O = e.getUint16(t, !1);
            ((t += 2), D.push(i.subarray(t, t + O)), (t += O));
          }
          I.push({ arrayCompleteness: _, nalUnitType: v, nalUnits: D });
        }
        return {
          configurationVersion: r,
          generalProfileSpace: s,
          generalTierFlag: a,
          generalProfileIdc: o,
          generalProfileCompatibilityFlags: c,
          generalConstraintIndicatorFlags: l,
          generalLevelIdc: u,
          minSpatialSegmentationIdc: d,
          parallelismType: f,
          chromaFormatIdc: h,
          bitDepthLumaMinus8: p,
          bitDepthChromaMinus8: m,
          avgFrameRate: y,
          constantFrameRate: b,
          numTemporalLayers: T,
          temporalIdNested: S,
          lengthSizeMinusOne: k,
          arrays: I,
        };
      } catch (e) {
        return (
          L._error("Error deserializing HEVC Decoder Configuration Record:", e),
          null
        );
      }
    };
  var Ee;
  (function (i) {
    ((i[(i.audAllowed = 0)] = "audAllowed"),
      (i[(i.beforeFirstVcl = 1)] = "beforeFirstVcl"),
      (i[(i.afterFirstVcl = 2)] = "afterFirstVcl"),
      (i[(i.eoBitstreamAllowed = 3)] = "eoBitstreamAllowed"),
      (i[(i.noMoreDataAllowed = 4)] = "noMoreDataAllowed"));
  })(Ee || (Ee = {}));
  const Fl = (i, e) => {
      const t = new Set();
      let r = Ee.audAllowed;
      for (const s of Ir(i, e)) {
        if (r === Ee.noMoreDataAllowed) {
          t.add(s.offset);
          continue;
        }
        const a = gi(i[s.offset]);
        if (r === Ee.eoBitstreamAllowed && a !== 37) {
          t.add(s.offset);
          continue;
        }
        let o = !1;
        (a === 35
          ? r > Ee.audAllowed
            ? (o = !0)
            : (r = Ee.beforeFirstVcl)
          : a <= 31
            ? r > Ee.afterFirstVcl
              ? (o = !0)
              : (r = Ee.afterFirstVcl)
            : a === 36
              ? r !== Ee.afterFirstVcl
                ? (o = !0)
                : (r = Ee.eoBitstreamAllowed)
              : a === 37
                ? r < Ee.afterFirstVcl
                  ? (o = !0)
                  : (r = Ee.noMoreDataAllowed)
                : a === 32 ||
                    a === 33 ||
                    a === 34 ||
                    a === 39 ||
                    (a >= 41 && a <= 44) ||
                    (a >= 48 && a <= 55)
                  ? r > Ee.beforeFirstVcl
                    ? (o = !0)
                    : (r = Ee.beforeFirstVcl)
                  : (a === 38 ||
                      a === 40 ||
                      (a >= 45 && a <= 47) ||
                      (a >= 56 && a <= 63)) &&
                    r < Ee.afterFirstVcl &&
                    (o = !0),
          o && t.add(s.offset));
      }
      if (t.size === 0) return null;
      const n = [];
      for (const s of Ir(i, e))
        t.has(s.offset) || n.push(i.subarray(s.offset, s.offset + s.length));
      return Pl(n, e);
    },
    Rl = {
      1: {
        colourPrimaries: 5,
        transferCharacteristics: 6,
        matrixCoefficients: 5,
      },
      2: {
        colourPrimaries: 1,
        transferCharacteristics: 1,
        matrixCoefficients: 1,
      },
      3: {
        colourPrimaries: 6,
        transferCharacteristics: 6,
        matrixCoefficients: 6,
      },
      4: {
        colourPrimaries: 7,
        transferCharacteristics: 7,
        matrixCoefficients: 7,
      },
      5: {
        colourPrimaries: 9,
        transferCharacteristics: 14,
        matrixCoefficients: 9,
      },
      7: {
        colourPrimaries: 1,
        transferCharacteristics: 13,
        matrixCoefficients: 0,
      },
    },
    Dn = (i) => {
      const e = new $(i);
      if (e.readBits(2) !== 2) return null;
      const r = e.readBits(1),
        s = (e.readBits(1) << 1) + r;
      if (
        (s === 3 && e.skipBits(1),
        e.readBits(1) === 1 ||
          e.readBits(1) !== 0 ||
          (e.skipBits(2), e.readBits(24) !== 4817730))
      )
        return null;
      let l = 8;
      s >= 2 && (l = e.readBits(1) ? 12 : 10);
      const u = e.readBits(3);
      let d = 0,
        f = 0;
      if (u !== 7)
        if (((f = e.readBits(1)), s === 1 || s === 3)) {
          const E = e.readBits(1),
            C = e.readBits(1);
          ((d = !E && !C ? 3 : E && !C ? 2 : 1), e.skipBits(1));
        } else d = 1;
      else ((d = 3), (f = 1));
      const h = e.readBits(16),
        p = e.readBits(16),
        m = h + 1,
        y = p + 1,
        w = m * y;
      let b = ne(Ft).level;
      for (const I of Ft)
        if (w <= I.maxPictureSize) {
          b = I.level;
          break;
        }
      const T = Rl[u],
        S = T?.colourPrimaries ?? 2,
        k = T?.transferCharacteristics ?? 2,
        P = T?.matrixCoefficients ?? 2;
      return {
        profile: s,
        level: b,
        bitDepth: l,
        chromaSubsampling: d,
        videoFullRangeFlag: f,
        colourPrimaries: S,
        transferCharacteristics: k,
        matrixCoefficients: P,
      };
    },
    Ml = (i) =>
      i.colourPrimaries !== 2 ||
      i.transferCharacteristics !== 2 ||
      i.matrixCoefficients !== 2,
    ka = function* (i) {
      const e = new $(i),
        t = () => {
          let r = 0;
          for (let n = 0; n < 8; n++) {
            const s = e.readAlignedByte();
            if (((r += (s & 127) * 2 ** (n * 7)), !(s & 128))) break;
            if (n === 7 && s & 128) return null;
          }
          return r > 2 ** 32 - 1 ? null : r;
        };
      for (; e.getBitsLeft() >= 8;) {
        e.skipBits(1);
        const r = e.readBits(4),
          n = e.readBits(1),
          s = e.readBits(1);
        (e.skipBits(1), n && e.skipBits(8));
        let a;
        if (s) {
          const o = t();
          if (o === null) return;
          a = o;
        } else a = Math.floor(e.getBitsLeft() / 8);
        (g(e.pos % 8 === 0),
          yield { type: r, data: i.subarray(e.pos / 8, e.pos / 8 + a) },
          e.skipBits(a * 8));
      }
    },
    _r = (i) => {
      for (const { type: e, data: t } of ka(i)) {
        if (e !== 1) continue;
        const r = new $(t),
          n = r.readBits(3);
        r.readBits(1);
        const s = r.readBits(1);
        let a = 0,
          o = 0,
          c = 0;
        if (s) a = r.readBits(5);
        else {
          const C = r.readBits(1);
          let _ = 0;
          if (C) {
            if ((r.skipBits(32), r.skipBits(32), r.readBits(1))) {
              let z = 0;
              for (; z < 32 && !r.readBits(1);) z++;
              z < 32 && r.skipBits(z);
            }
            ((_ = r.readBits(1)),
              _ &&
                ((c = r.readBits(5)),
                r.skipBits(32),
                r.skipBits(5),
                r.skipBits(5)));
          }
          const v = r.readBits(1),
            F = r.readBits(5);
          for (let D = 0; D <= F; D++) {
            r.skipBits(12);
            const z = r.readBits(5);
            if ((D === 0 && (a = z), z > 7)) {
              const O = r.readBits(1);
              D === 0 && (o = O);
            }
            if (_ && r.readBits(1)) {
              const H = c + 1;
              (r.skipBits(H), r.skipBits(H), r.skipBits(1));
            }
            v && r.readBits(1) && r.skipBits(4);
          }
        }
        const l = r.readBits(4),
          u = r.readBits(4),
          d = l + 1;
        r.skipBits(d);
        const f = u + 1;
        r.skipBits(f);
        let h = 0;
        if (
          (s ? (h = 0) : (h = r.readBits(1)),
          h && (r.skipBits(4), r.skipBits(3)),
          r.skipBits(1),
          r.skipBits(1),
          r.skipBits(1),
          !s)
        ) {
          (r.skipBits(1), r.skipBits(1), r.skipBits(1), r.skipBits(1));
          const C = r.readBits(1);
          C && (r.skipBits(1), r.skipBits(1));
          const _ = r.readBits(1);
          let v = 0;
          (_ ? (v = 2) : (v = r.readBits(1)),
            v > 0 && (r.readBits(1) || r.skipBits(1)),
            C && r.skipBits(3));
        }
        (r.skipBits(1), r.skipBits(1), r.skipBits(1));
        const p = r.readBits(1);
        let m = 8;
        n === 2 && p
          ? (m = r.readBits(1) ? 12 : 10)
          : n <= 2 && (m = p ? 10 : 8);
        let y = 0;
        n !== 1 && (y = r.readBits(1));
        let w = 2,
          b = 2,
          T = 2;
        r.readBits(1) &&
          ((w = r.readBits(8)), (b = r.readBits(8)), (T = r.readBits(8)));
        let k = 0,
          P = 1,
          I = 1,
          E = 0;
        return (
          y
            ? (k = r.readBits(1))
            : w === 1 && b === 13 && T === 0
              ? ((k = 1), (P = 0), (I = 0))
              : ((k = r.readBits(1)),
                n === 0
                  ? ((P = 1), (I = 1))
                  : n === 1
                    ? ((P = 0), (I = 0))
                    : m === 12
                      ? ((P = r.readBits(1)), (I = P ? r.readBits(1) : 0))
                      : ((P = 1), (I = 0)),
                P && I && (E = r.readBits(2))),
          {
            profile: n,
            level: a,
            tier: o,
            bitDepth: m,
            monochrome: y,
            chromaSubsamplingX: P,
            chromaSubsamplingY: I,
            chromaSamplePosition: E,
            videoFullRangeFlag: k,
            colourPrimaries: w,
            transferCharacteristics: b,
            matrixCoefficients: T,
          }
        );
      }
      return null;
    },
    Ta = (i) =>
      i.colourPrimaries !== 2 ||
      i.transferCharacteristics !== 2 ||
      i.matrixCoefficients !== 2,
    Sa = (i) => {
      if (i.length < 36) return null;
      const t = j(i);
      return t.getUint32(4) !== 1768124518 || t.getUint16(8) < 28
        ? null
        : {
            fullRange: !1,
            colourPrimaries: t.getUint8(22),
            transferCharacteristics: t.getUint8(23),
            matrixCoefficients: t.getUint8(24),
          };
    },
    On = (i) => {
      const e = j(i),
        t = e.getUint8(9),
        r = e.getUint16(10, !0),
        n = e.getUint32(12, !0),
        s = e.getInt16(16, !0),
        a = e.getUint8(18);
      let o = null;
      return (
        a && (o = i.subarray(19, 21 + t)),
        {
          outputChannelCount: t,
          preSkip: r,
          inputSampleRate: n,
          outputGain: s,
          channelMappingFamily: a,
          channelMappingTable: o,
        }
      );
    },
    Dl = [
      480, 960, 1920, 2880, 480, 960, 1920, 2880, 480, 960, 1920, 2880, 480,
      960, 480, 960, 120, 240, 480, 960, 120, 240, 480, 960, 120, 240, 480, 960,
      120, 240, 480, 960,
    ],
    Ol = (i) => {
      const e = i[0] >> 3,
        t = i[0] & 3;
      let r;
      return (
        t === 0 ? (r = 1) : t === 1 || t === 2 ? (r = 2) : (r = i[1] & 63),
        { durationInSamples: Dl[e] * r }
      );
    },
    zl = (i) => {
      if (i.length < 7) throw new Error("Setup header is too short.");
      if (i[0] !== 5) throw new Error("Wrong packet type in Setup header.");
      if (String.fromCharCode(...i.slice(1, 7)) !== "vorbis")
        throw new Error("Invalid packet signature in Setup header.");
      const t = i.length,
        r = new Uint8Array(t);
      for (let d = 0; d < t; d++) r[d] = i[t - 1 - d];
      const n = new $(r);
      let s = 0;
      for (; n.getBitsLeft() > 97;)
        if (n.readBits(1) === 1) {
          s = n.pos;
          break;
        }
      if (s === 0)
        throw new Error("Invalid Setup header: framing bit not found.");
      let a = 0,
        o = !1,
        c = 0;
      for (; n.getBitsLeft() >= 97;) {
        const d = n.pos,
          f = n.readBits(8),
          h = n.readBits(16),
          p = n.readBits(16);
        if (f > 63 || h !== 0 || p !== 0) {
          n.pos = d;
          break;
        }
        if ((n.skipBits(1), a++, a > 64)) break;
        n.clone().readBits(6) + 1 === a && ((o = !0), (c = a));
      }
      if (!o) throw new Error("Invalid Setup header: mode header not found.");
      if (c > 63) throw new Error(`Unsupported mode count: ${c}.`);
      const l = c;
      ((n.pos = 0), n.skipBits(s));
      const u = Array(l).fill(0);
      for (let d = l - 1; d >= 0; d--) (n.skipBits(40), (u[d] = n.readBits(1)));
      return { modeBlockflags: u };
    },
    vr = (i, e, t) => {
      switch (i) {
        case "avc": {
          for (const r of fa(t, e)) {
            const n = t[r.offset],
              s = ji(n);
            if (s >= be.NON_IDR_SLICE && s <= be.SLICE_DPC) return "delta";
            if (s === be.IDR) return "key";
            if (s === be.SEI && !Tn()) {
              const a = t.subarray(r.offset, r.offset + r.length),
                o = Cr(a);
              let c = 1;
              do {
                let l = 0;
                for (;;) {
                  const f = o[c++];
                  if (f === void 0 || ((l += f), f < 255)) break;
                }
                let u = 0;
                for (;;) {
                  const f = o[c++];
                  if (f === void 0 || ((u += f), f < 255)) break;
                }
                if (l === 6) {
                  const f = new $(o);
                  f.pos = 8 * c;
                  const h = M(f),
                    p = f.readBits(1);
                  if (h === 0 && p === 1) return "key";
                }
                c += u;
              } while (c < o.length - 1);
            }
          }
          return "delta";
        }
        case "hevc": {
          for (const r of Ir(t, e)) {
            const n = gi(t[r.offset]);
            if (n < de.BLA_W_LP) return "delta";
            if (n <= de.RSV_IRAP_VCL23) return "key";
          }
          return "delta";
        }
        case "vp8":
          return (t[0] & 1) === 0 ? "key" : "delta";
        case "vp9": {
          const r = new $(t);
          if (r.readBits(2) !== 2) return null;
          const n = r.readBits(1);
          return (
            (r.readBits(1) << 1) + n === 3 && r.skipBits(1),
            r.readBits(1) ? null : r.readBits(1) === 0 ? "key" : "delta"
          );
        }
        case "av1": {
          let r = !1;
          for (const { type: n, data: s } of ka(t))
            if (n === 1) {
              const a = new $(s);
              (a.skipBits(4), (r = !!a.readBits(1)));
            } else if (n === 3 || n === 6 || n === 7) {
              if (r) return "key";
              const a = new $(s);
              return a.readBits(1)
                ? null
                : a.readBits(2) === 0
                  ? "key"
                  : "delta";
            }
          return null;
        }
        case "prores":
          return "key";
        default:
          (xe(i), g(!1));
      }
    };
  var wi;
  (function (i) {
    ((i[(i.STREAMINFO = 0)] = "STREAMINFO"),
      (i[(i.VORBIS_COMMENT = 4)] = "VORBIS_COMMENT"),
      (i[(i.PICTURE = 6)] = "PICTURE"));
  })(wi || (wi = {}));
  const zn = (i, e) => {
      const t = j(i);
      let r = 0;
      const n = t.getUint32(r, !0);
      r += 4;
      const s = Oe.decode(i.subarray(r, r + n));
      ((r += n), n > 0 && ((e.raw ??= {}), (e.raw.vendor ??= s)));
      const a = t.getUint32(r, !0);
      r += 4;
      for (let o = 0; o < a; o++) {
        const c = t.getUint32(r, !0);
        r += 4;
        const l = Oe.decode(i.subarray(r, r + c));
        r += c;
        const u = l.indexOf("=");
        if (u === -1) continue;
        const d = l.slice(0, u).toUpperCase(),
          f = l.slice(u + 1);
        switch (
          ((e.raw ??= {}),
          Array.isArray(e.raw[d])
            ? (e.raw[d] = [...e.raw[d], f])
            : typeof e.raw[d] == "string"
              ? (e.raw[d] = [e.raw[d], f])
              : (e.raw[d] ??= f),
          d)
        ) {
          case "TITLE":
            e.title ??= f;
            break;
          case "DESCRIPTION":
            e.description ??= f;
            break;
          case "ARTIST":
            e.artist ??= f;
            break;
          case "ALBUM":
            e.album ??= f;
            break;
          case "ALBUMARTIST":
            e.albumArtist ??= f;
            break;
          case "COMMENT":
            e.comment ??= f;
            break;
          case "LYRICS":
            e.lyrics ??= f;
            break;
          case "TRACKNUMBER":
            {
              const h = f.split("/"),
                p = Number.parseInt(h[0], 10),
                m = h[1] && Number.parseInt(h[1], 10);
              (Number.isInteger(p) && p > 0 && (e.trackNumber ??= p),
                m && Number.isInteger(m) && m > 0 && (e.tracksTotal ??= m));
            }
            break;
          case "TRACKTOTAL":
            {
              const h = Number.parseInt(f, 10);
              Number.isInteger(h) && h > 0 && (e.tracksTotal ??= h);
            }
            break;
          case "DISCNUMBER":
            {
              const h = f.split("/"),
                p = Number.parseInt(h[0], 10),
                m = h[1] && Number.parseInt(h[1], 10);
              (Number.isInteger(p) && p > 0 && (e.discNumber ??= p),
                m && Number.isInteger(m) && m > 0 && (e.discsTotal ??= m));
            }
            break;
          case "DISCTOTAL":
            {
              const h = Number.parseInt(f, 10);
              Number.isInteger(h) && h > 0 && (e.discsTotal ??= h);
            }
            break;
          case "DATE":
            {
              const h = new Date(f);
              Number.isNaN(h.getTime()) || (e.date ??= h);
            }
            break;
          case "BPM":
            {
              const h = Number.parseInt(f, 10);
              Number.isInteger(h) && h > 0 && (e.beatsPerMinute ??= h);
            }
            break;
          case "GENRE":
            e.genre ??= f;
            break;
          case "METADATA_BLOCK_PICTURE":
            {
              const h = ra(f),
                p = j(h),
                m = p.getUint32(0, !1),
                y = p.getUint32(4, !1),
                w = String.fromCharCode(...h.subarray(8, 8 + y)),
                b = p.getUint32(8 + y, !1),
                T = Oe.decode(h.subarray(12 + y, 12 + y + b)),
                S = p.getUint32(y + b + 28),
                k = h.subarray(y + b + 32, y + b + 32 + S);
              ((e.images ??= []),
                e.images.push({
                  data: k,
                  mimeType: w,
                  kind:
                    m === 3 ? "coverFront" : m === 4 ? "coverBack" : "unknown",
                  name: void 0,
                  description: T || void 0,
                }));
            }
            break;
        }
      }
    },
    Nn = [2, 1, 2, 3, 3, 4, 4, 5],
    xa = (i) => {
      if (i.length < 7 || i[0] !== 11 || i[1] !== 119) return null;
      const e = new $(i);
      (e.skipBits(16), e.skipBits(16));
      const t = e.readBits(2);
      if (t === 3) return null;
      const r = e.readBits(6),
        n = e.readBits(5);
      if (n > 8) return null;
      const s = e.readBits(3),
        a = e.readBits(3);
      ((a & 1) !== 0 && a !== 1 && e.skipBits(2),
        (a & 4) !== 0 && e.skipBits(2),
        a === 2 && e.skipBits(2));
      const o = e.readBits(1),
        c = Math.floor(r / 2);
      return {
        fscod: t,
        bsid: n,
        bsmod: s,
        acmod: a,
        lfeon: o,
        bitRateCode: c,
      };
    },
    Nl = [
      128,
      138,
      192,
      128,
      140,
      192,
      160,
      174,
      240,
      160,
      176,
      240,
      192,
      208,
      288,
      192,
      210,
      288,
      224,
      242,
      336,
      224,
      244,
      336,
      256,
      278,
      384,
      256,
      280,
      384,
      320,
      348,
      480,
      320,
      350,
      480,
      384,
      416,
      288 * 2,
      384,
      418,
      288 * 2,
      448,
      486,
      336 * 2,
      448,
      488,
      336 * 2,
      256 * 2,
      278 * 2,
      384 * 2,
      256 * 2,
      279 * 2,
      384 * 2,
      320 * 2,
      348 * 2,
      480 * 2,
      320 * 2,
      349 * 2,
      480 * 2,
      384 * 2,
      417 * 2,
      576 * 2,
      384 * 2,
      418 * 2,
      576 * 2,
      448 * 2,
      487 * 2,
      672 * 2,
      448 * 2,
      488 * 2,
      672 * 2,
      512 * 2,
      557 * 2,
      768 * 2,
      512 * 2,
      558 * 2,
      768 * 2,
      640 * 2,
      696 * 2,
      960 * 2,
      640 * 2,
      697 * 2,
      960 * 2,
      768 * 2,
      835 * 2,
      1152 * 2,
      768 * 2,
      836 * 2,
      1152 * 2,
      896 * 2,
      975 * 2,
      1344 * 2,
      896 * 2,
      976 * 2,
      1344 * 2,
      1024 * 2,
      1114 * 2,
      1536 * 2,
      1024 * 2,
      1115 * 2,
      1536 * 2,
      1152 * 2,
      1253 * 2,
      1728 * 2,
      1152 * 2,
      1254 * 2,
      1728 * 2,
      1280 * 2,
      1393 * 2,
      1920 * 2,
      1280 * 2,
      1394 * 2,
      1920 * 2,
    ],
    Ul = 1536,
    Pa = [1, 2, 3, 6],
    Ca = (i) => {
      if (i.length < 6 || i[0] !== 11 || i[1] !== 119) return null;
      const e = new $(i);
      e.skipBits(16);
      const t = e.readBits(2);
      if ((e.skipBits(3), t !== 0 && t !== 2)) return null;
      const r = e.readBits(11),
        n = e.readBits(2);
      let s = 0,
        a;
      n === 3 ? ((s = e.readBits(2)), (a = 3)) : (a = e.readBits(2));
      const o = e.readBits(3),
        c = e.readBits(1),
        l = e.readBits(5);
      if (l < 11 || l > 16) return null;
      const u = Pa[a];
      let d;
      return (
        n < 3 ? (d = Pr[n] / 1e3) : (d = ua[s] / 1e3),
        {
          dataRate: Math.round(((r + 1) * d) / (u * 16)),
          substreams: [
            {
              fscod: n,
              fscod2: s,
              bsid: l,
              bsmod: 0,
              acmod: o,
              lfeon: c,
              numDepSub: 0,
              chanLoc: 0,
            },
          ],
        }
      );
    },
    Wl = (i) => {
      if (i.length < 2) return null;
      const e = new $(i),
        t = e.readBits(13),
        r = e.readBits(3),
        n = [];
      for (let s = 0; s <= r && !(Math.ceil(e.pos / 8) + 3 > i.length); s++) {
        const a = e.readBits(2),
          o = e.readBits(5);
        (e.skipBits(1), e.skipBits(1));
        const c = e.readBits(3),
          l = e.readBits(3),
          u = e.readBits(1);
        e.skipBits(3);
        const d = e.readBits(4);
        let f = 0;
        (d > 0 ? (f = e.readBits(9)) : e.skipBits(1),
          n.push({
            fscod: a,
            fscod2: null,
            bsid: o,
            bsmod: c,
            acmod: l,
            lfeon: u,
            numDepSub: d,
            chanLoc: f,
          }));
      }
      return n.length === 0 ? null : { dataRate: t, substreams: n };
    },
    Ia = (i) => {
      const e = i.substreams[0];
      return (
        g(e),
        e.fscod < 3
          ? Pr[e.fscod]
          : e.fscod2 !== null && e.fscod2 < 3
            ? ua[e.fscod2]
            : null
      );
    },
    Ea = (i) => {
      const e = i.substreams[0];
      g(e);
      let t = Nn[e.acmod] + e.lfeon;
      if (e.numDepSub > 0) {
        const r = [2, 2, 1, 1, 2, 2, 2, 1, 1];
        for (let n = 0; n < 9; n++) e.chanLoc & (1 << (8 - n)) && (t += r[n]);
      }
      return t;
    },
    Vl = 1683496997,
    Ar = 18,
    Un = 10,
    Ll = 4096,
    _a = 32,
    Wn = 20,
    Hl = 8,
    ql = [
      0, 8e3, 16e3, 32e3, 0, 0, 11025, 22050, 44100, 0, 0, 12e3, 24e3, 48e3,
      96e3, 192e3,
    ],
    jl = [
      32e3, 56e3, 64e3, 96e3, 112e3, 128e3, 192e3, 224e3, 256e3, 32e4, 384e3,
      448e3, 512e3, 576e3, 64e4, 768e3, 96e4, 1024e3, 1152e3, 128e4, 1344e3,
      1408e3, 1411200, 1472e3, 1536e3, 192e4, 2048e3, 3072e3, 384e4, 0, 0, 0,
    ],
    $l = [16, 16, 20, 20, 0, 24, 24, 0],
    Br = [1, 2, 2, 2, 2, 3, 3, 4, 4, 5, 6, 6, 6, 7, 8, 8],
    Kl = [1, 2, 2, 2, 2, 3, 18, 19, 6, 7, 518, 323, 83, 519, 582, 535],
    Gl = 8,
    Xl = 44646,
    Ql = [32e3, 44100, 48e3, 0],
    Yl = [
      8e3, 16e3, 32e3, 64e3, 128e3, 22050, 44100, 88200, 176400, 352800, 12e3,
      24e3, 48e3, 96e3, 192e3, 384e3,
    ],
    va = [512, 1024, 2048, 4096],
    Vn = (i) => {
      const e = Ba(i),
        t = j(i);
      let r = e ? Math.ceil(e.frameSize / 4) * 4 : 0,
        n = null;
      for (; r + 4 <= i.length && t.getUint32(r) === Vl;) {
        const a = Fr(i.subarray(r));
        if (!a) break;
        ((n ??= a), (r += a.frameSize));
      }
      if (e)
        return {
          frameSize: n ? r : e.frameSize,
          sampleRate: e.sampleRate,
          numberOfChannels: e.numberOfChannels,
          sampleCount: e.sampleCount,
          channelLayout: e.channelLayout,
          pcmResolution: e.pcmResolution,
          bitRate: e.bitRate,
          core: e,
          hasExtensions: n !== null,
        };
      if (!n?.asset) return null;
      const { asset: s } = n;
      return {
        frameSize: r,
        sampleRate: s.sampleRate,
        numberOfChannels: s.numberOfChannels,
        sampleCount: s.sampleCount,
        channelLayout: s.channelLayout,
        pcmResolution: s.pcmResolution,
        bitRate: 0,
        core: null,
        hasExtensions: !0,
      };
    },
    Aa = (i) => {
      const e = Vn(i);
      return e?.core ? (e.hasExtensions ? "dtsh" : "dtsc") : null;
    },
    Ba = (i) => {
      if (
        i.length < Ar ||
        i[0] !== 127 ||
        i[1] !== 254 ||
        i[2] !== 128 ||
        i[3] !== 1
      )
        return null;
      const e = new $(i);
      if ((e.skipBits(32), e.skipBits(1), e.readBits(5) !== _a - 1))
        return null;
      const t = e.readBits(1),
        r = e.readBits(7) + 1;
      if (r % Hl !== 0) return null;
      const n = e.readBits(14) + 1;
      if (n < 96) return null;
      const s = e.readBits(6);
      if (s >= Br.length) return null;
      const a = ql[e.readBits(4)];
      if (a === 0) return null;
      const o = jl[e.readBits(5)];
      if (e.readBits(1) !== 0) return null;
      (e.skipBits(4), e.skipBits(5));
      const c = e.readBits(2);
      if (c === 3) return null;
      (e.skipBits(1), t && e.skipBits(16), e.skipBits(7));
      const l = $l[e.readBits(3)];
      if (l === 0) return null;
      const u = c !== 0;
      return {
        frameSize: n,
        sampleRate: a,
        numberOfChannels: Br[s] + (u ? 1 : 0),
        sampleCount: r * _a,
        channelLayout: Kl[s] | (u ? Gl : 0),
        amode: s,
        lfePresent: u,
        bitRate: o,
        pcmResolution: l,
      };
    },
    Fr = (i) => {
      if (
        i.length < Un ||
        i[0] !== 100 ||
        i[1] !== 88 ||
        i[2] !== 32 ||
        i[3] !== 37
      )
        return null;
      const e = new $(i);
      (e.skipBits(32), e.skipBits(8));
      const t = e.readBits(2),
        r = e.readBits(1),
        n = 8 + 4 * r,
        s = 16 + 4 * r;
      e.skipBits(n);
      const a = e.readBits(s) + 1,
        o = { frameSize: a, asset: null };
      if (!e.readBits(1)) return o;
      const c = Ql[e.readBits(2)],
        l = 512 * (e.readBits(3) + 1);
      e.readBits(1) && e.skipBits(36);
      const u = e.readBits(3) + 1,
        d = e.readBits(3) + 1,
        f = [];
      for (let w = 0; w < u; w++) f.push(e.readBits(t + 1));
      for (const w of f) e.skipBits(8 * wn(w));
      if (e.readBits(1)) {
        e.skipBits(2);
        const w = (e.readBits(2) + 1) << 2,
          b = e.readBits(2) + 1;
        e.skipBits(b * w);
      }
      for (let w = 0; w < d; w++) e.skipBits(s);
      (e.skipBits(9),
        e.skipBits(3),
        e.readBits(1) && e.skipBits(4),
        e.readBits(1) && e.skipBits(24),
        e.readBits(1) && e.skipBits(8 * (e.readBits(10) + 1)));
      const h = e.readBits(5) + 1,
        p = Yl[e.readBits(4)],
        m = e.readBits(8) + 1;
      let y = 0;
      if (
        e.readBits(1) &&
        (m > 2 && e.skipBits(1), m > 6 && e.skipBits(1), e.readBits(1))
      ) {
        const w = (e.readBits(2) + 1) << 2;
        y = e.readBits(w);
      }
      return c === 0 || e.getBitsLeft() < 0
        ? o
        : {
            frameSize: a,
            asset: {
              sampleRate: p,
              numberOfChannels: m,
              sampleCount: Math.round((l * p) / c),
              channelLayout: y,
              pcmResolution: h,
            },
          };
    },
    Zl = (i) => {
      if (i.length < Wn) return null;
      const e = j(i),
        t = e.getUint32(0);
      if (t === 0) return null;
      const r = new $(i);
      r.seekToByte(13);
      const n = r.readBits(2);
      r.skipBits(5);
      const s = r.readBits(1),
        a = r.readBits(6);
      (r.skipBits(14), r.skipBits(1), r.skipBits(3));
      const o = r.readBits(16);
      let c = null;
      return (
        o !== 0 ? (c = eu(o)) : a < Br.length && (c = Br[a] + s),
        {
          sampleRate: t,
          maxBitrate: e.getUint32(4),
          avgBitrate: e.getUint32(8),
          pcmSampleDepth: i[12],
          sampleCount: va[n],
          channelLayout: o,
          numberOfChannels: c,
        }
      );
    },
    Jl = (i) => {
      const e = new Uint8Array(Wn),
        t = j(e);
      (t.setUint32(0, i.sampleRate),
        t.setUint32(4, i.bitRate),
        t.setUint32(8, i.bitRate),
        (e[12] = i.pcmResolution));
      const r = i.core && !i.hasExtensions ? 1 : 0,
        n = new $(e);
      return (
        n.seekToByte(13),
        n.writeBits(2, Math.max(va.indexOf(i.sampleCount), 0)),
        n.writeBits(5, r),
        n.writeBits(1, i.core?.lfePresent ? 1 : 0),
        n.writeBits(6, i.core?.amode ?? 0),
        n.writeBits(14, i.core ? i.core.frameSize - 1 : 0),
        n.writeBits(1, 0),
        n.writeBits(3, 0),
        n.writeBits(16, i.channelLayout),
        n.writeBits(1, 0),
        n.writeBits(1, 0),
        n.writeBits(1, 0),
        n.writeBits(5, 0),
        e
      );
    },
    eu = (i) => wn(i) + wn(i & Xl);
  const Fe = ["avc", "hevc", "vp9", "av1", "vp8", "prores"],
    pe = [
      "pcm-s16",
      "pcm-s16be",
      "pcm-s24",
      "pcm-s24be",
      "pcm-s32",
      "pcm-s32be",
      "pcm-f32",
      "pcm-f32be",
      "pcm-f64",
      "pcm-f64be",
      "pcm-u8",
      "pcm-s8",
      "ulaw",
      "alaw",
    ],
    Bt = ["aac", "opus", "mp3", "vorbis", "flac", "ac3", "eac3", "dts"],
    Ne = [...Bt, ...pe],
    $t = ["webvtt"],
    Rr = [
      { maxMacroblocks: 99, maxBitrate: 64e3, maxDpbMbs: 396, level: 10 },
      { maxMacroblocks: 396, maxBitrate: 192e3, maxDpbMbs: 900, level: 11 },
      { maxMacroblocks: 396, maxBitrate: 384e3, maxDpbMbs: 2376, level: 12 },
      { maxMacroblocks: 396, maxBitrate: 768e3, maxDpbMbs: 2376, level: 13 },
      { maxMacroblocks: 396, maxBitrate: 2e6, maxDpbMbs: 2376, level: 20 },
      { maxMacroblocks: 792, maxBitrate: 4e6, maxDpbMbs: 4752, level: 21 },
      { maxMacroblocks: 1620, maxBitrate: 4e6, maxDpbMbs: 8100, level: 22 },
      { maxMacroblocks: 1620, maxBitrate: 1e7, maxDpbMbs: 8100, level: 30 },
      { maxMacroblocks: 3600, maxBitrate: 14e6, maxDpbMbs: 18e3, level: 31 },
      { maxMacroblocks: 5120, maxBitrate: 2e7, maxDpbMbs: 20480, level: 32 },
      { maxMacroblocks: 8192, maxBitrate: 2e7, maxDpbMbs: 32768, level: 40 },
      { maxMacroblocks: 8192, maxBitrate: 5e7, maxDpbMbs: 32768, level: 41 },
      { maxMacroblocks: 8704, maxBitrate: 5e7, maxDpbMbs: 34816, level: 42 },
      {
        maxMacroblocks: 22080,
        maxBitrate: 135e6,
        maxDpbMbs: 110400,
        level: 50,
      },
      { maxMacroblocks: 36864, maxBitrate: 24e7, maxDpbMbs: 184320, level: 51 },
      { maxMacroblocks: 36864, maxBitrate: 24e7, maxDpbMbs: 184320, level: 52 },
      {
        maxMacroblocks: 139264,
        maxBitrate: 24e7,
        maxDpbMbs: 696320,
        level: 60,
      },
      {
        maxMacroblocks: 139264,
        maxBitrate: 48e7,
        maxDpbMbs: 696320,
        level: 61,
      },
      { maxMacroblocks: 139264, maxBitrate: 8e8, maxDpbMbs: 696320, level: 62 },
    ],
    Fa = [
      { maxPictureSize: 36864, maxBitrate: 128e3, tier: "L", level: 30 },
      { maxPictureSize: 122880, maxBitrate: 15e5, tier: "L", level: 60 },
      { maxPictureSize: 245760, maxBitrate: 3e6, tier: "L", level: 63 },
      { maxPictureSize: 552960, maxBitrate: 6e6, tier: "L", level: 90 },
      { maxPictureSize: 983040, maxBitrate: 1e7, tier: "L", level: 93 },
      { maxPictureSize: 2228224, maxBitrate: 12e6, tier: "L", level: 120 },
      { maxPictureSize: 2228224, maxBitrate: 3e7, tier: "H", level: 120 },
      { maxPictureSize: 2228224, maxBitrate: 2e7, tier: "L", level: 123 },
      { maxPictureSize: 2228224, maxBitrate: 5e7, tier: "H", level: 123 },
      { maxPictureSize: 8912896, maxBitrate: 25e6, tier: "L", level: 150 },
      { maxPictureSize: 8912896, maxBitrate: 1e8, tier: "H", level: 150 },
      { maxPictureSize: 8912896, maxBitrate: 4e7, tier: "L", level: 153 },
      { maxPictureSize: 8912896, maxBitrate: 16e7, tier: "H", level: 153 },
      { maxPictureSize: 8912896, maxBitrate: 6e7, tier: "L", level: 156 },
      { maxPictureSize: 8912896, maxBitrate: 24e7, tier: "H", level: 156 },
      { maxPictureSize: 35651584, maxBitrate: 6e7, tier: "L", level: 180 },
      { maxPictureSize: 35651584, maxBitrate: 24e7, tier: "H", level: 180 },
      { maxPictureSize: 35651584, maxBitrate: 12e7, tier: "L", level: 183 },
      { maxPictureSize: 35651584, maxBitrate: 48e7, tier: "H", level: 183 },
      { maxPictureSize: 35651584, maxBitrate: 24e7, tier: "L", level: 186 },
      { maxPictureSize: 35651584, maxBitrate: 8e8, tier: "H", level: 186 },
    ],
    Ft = [
      { maxPictureSize: 36864, maxBitrate: 2e5, level: 10 },
      { maxPictureSize: 73728, maxBitrate: 8e5, level: 11 },
      { maxPictureSize: 122880, maxBitrate: 18e5, level: 20 },
      { maxPictureSize: 245760, maxBitrate: 36e5, level: 21 },
      { maxPictureSize: 552960, maxBitrate: 72e5, level: 30 },
      { maxPictureSize: 983040, maxBitrate: 12e6, level: 31 },
      { maxPictureSize: 2228224, maxBitrate: 18e6, level: 40 },
      { maxPictureSize: 2228224, maxBitrate: 3e7, level: 41 },
      { maxPictureSize: 8912896, maxBitrate: 6e7, level: 50 },
      { maxPictureSize: 8912896, maxBitrate: 12e7, level: 51 },
      { maxPictureSize: 8912896, maxBitrate: 18e7, level: 52 },
      { maxPictureSize: 35651584, maxBitrate: 18e7, level: 60 },
      { maxPictureSize: 35651584, maxBitrate: 24e7, level: 61 },
      { maxPictureSize: 35651584, maxBitrate: 48e7, level: 62 },
    ],
    Ra = [
      { maxPictureSize: 147456, maxBitrate: 15e5, tier: "M", level: 0 },
      { maxPictureSize: 278784, maxBitrate: 3e6, tier: "M", level: 1 },
      { maxPictureSize: 665856, maxBitrate: 6e6, tier: "M", level: 4 },
      { maxPictureSize: 1065024, maxBitrate: 1e7, tier: "M", level: 5 },
      { maxPictureSize: 2359296, maxBitrate: 12e6, tier: "M", level: 8 },
      { maxPictureSize: 2359296, maxBitrate: 3e7, tier: "H", level: 8 },
      { maxPictureSize: 2359296, maxBitrate: 2e7, tier: "M", level: 9 },
      { maxPictureSize: 2359296, maxBitrate: 5e7, tier: "H", level: 9 },
      { maxPictureSize: 8912896, maxBitrate: 3e7, tier: "M", level: 12 },
      { maxPictureSize: 8912896, maxBitrate: 1e8, tier: "H", level: 12 },
      { maxPictureSize: 8912896, maxBitrate: 4e7, tier: "M", level: 13 },
      { maxPictureSize: 8912896, maxBitrate: 16e7, tier: "H", level: 13 },
      { maxPictureSize: 8912896, maxBitrate: 6e7, tier: "M", level: 14 },
      { maxPictureSize: 8912896, maxBitrate: 24e7, tier: "H", level: 14 },
      { maxPictureSize: 35651584, maxBitrate: 6e7, tier: "M", level: 15 },
      { maxPictureSize: 35651584, maxBitrate: 24e7, tier: "H", level: 15 },
      { maxPictureSize: 35651584, maxBitrate: 6e7, tier: "M", level: 16 },
      { maxPictureSize: 35651584, maxBitrate: 24e7, tier: "H", level: 16 },
      { maxPictureSize: 35651584, maxBitrate: 1e8, tier: "M", level: 17 },
      { maxPictureSize: 35651584, maxBitrate: 48e7, tier: "H", level: 17 },
      { maxPictureSize: 35651584, maxBitrate: 16e7, tier: "M", level: 18 },
      { maxPictureSize: 35651584, maxBitrate: 8e8, tier: "H", level: 18 },
      { maxPictureSize: 35651584, maxBitrate: 16e7, tier: "M", level: 19 },
      { maxPictureSize: 35651584, maxBitrate: 8e8, tier: "H", level: 19 },
    ],
    Ma = ".01.01.01.01.00",
    Da = ".0.110.01.01.01.0",
    Kt = ["ap4x", "ap4h", "apch", "apcn", "apcs", "apco"],
    Mr = ["dtsc", "dtsh", "dtsl", "dtse"],
    tu = [
      { fourCc: "apco", bitrate: 45e6, alpha: !1 },
      { fourCc: "apcs", bitrate: 102e6, alpha: !1 },
      { fourCc: "apcn", bitrate: 147e6, alpha: !1 },
      { fourCc: "apch", bitrate: 22e7, alpha: !1 },
      { fourCc: "ap4h", bitrate: 33e7, alpha: !0 },
      { fourCc: "ap4x", bitrate: 5e8, alpha: !0 },
    ],
    iu = (i, e, t, r, n) => {
      if (i === "avc") {
        const a = Math.ceil(e / 16) * Math.ceil(t / 16),
          o =
            Rr.find((f) => a <= f.maxMacroblocks && r <= f.maxBitrate) ??
            ne(Rr),
          c = o ? o.level : 0,
          l = "64".padStart(2, "0"),
          u = "00",
          d = c.toString(16).padStart(2, "0");
        return `avc1.${l}${u}${d}`;
      } else if (i === "hevc") {
        const c = e * t,
          l =
            Fa.find((d) => c <= d.maxPictureSize && r <= d.maxBitrate) ??
            ne(Fa);
        return `hev1.1.6.${l.tier}${l.level}.B0`;
      } else {
        if (i === "vp8") return "vp8";
        if (i === "vp9") {
          const a = e * t;
          return `vp09.00.${(Ft.find((l) => a <= l.maxPictureSize && r <= l.maxBitrate) ?? ne(Ft)).level.toString().padStart(2, "0")}.08`;
        } else if (i === "av1") {
          const a = e * t,
            o =
              Ra.find((u) => a <= u.maxPictureSize && r <= u.maxBitrate) ??
              ne(Ra);
          return `av01.0.${o.level.toString().padStart(2, "0")}${o.tier}.08`;
        } else if (i === "prores") {
          const a = Math.pow((e * t) / 2073600, 0.95),
            o = tu.filter((u) => u.alpha === n);
          let c = o[0].fourCc,
            l = 1 / 0;
          for (const { fourCc: u, bitrate: d } of o) {
            const f = Math.abs(d * a - r);
            f < l && ((l = f), (c = u));
          }
          return c;
        } else xe(i);
      }
      throw new TypeError(`Unhandled codec '${String(i)}'.`);
    },
    ru = (i) => {
      const e = i.split("."),
        t = Number(e[1]),
        r = Number(e[2]),
        n = Number(e[3]),
        s = e[4] ? Number(e[4]) : 1;
      return [1, 1, t, 2, 1, r, 3, 1, n, 4, 1, s];
    },
    Oa = (i) => {
      const e = i.split("."),
        n = (1 << 7) + 1,
        s = Number(e[1]),
        a = e[2],
        o = Number(a.slice(0, -1)),
        c = (s << 5) + o,
        l = a.slice(-1) === "H" ? 1 : 0,
        u = Number(e[3]),
        d = u === 8 ? 0 : 1,
        f = u === 12 ? 1 : 0,
        h = e[4] ? Number(e[4]) : 0,
        p = e[5] ? Number(e[5][0]) : 1,
        m = e[5] ? Number(e[5][1]) : 1,
        y = e[5] ? Number(e[5][2]) : 0,
        w = (l << 7) + (d << 6) + (f << 5) + (h << 4) + (p << 3) + (m << 2) + y;
      return [n, c, w, 0];
    },
    Dr = (i) => {
      const {
        codec: e,
        codecDescription: t,
        colorSpace: r,
        avcCodecInfo: n,
        hevcCodecInfo: s,
        vp9CodecInfo: a,
        av1CodecInfo: o,
        proresFormat: c,
      } = i;
      if (e === "avc") {
        if ((g(i.avcType !== null), n)) {
          const l = new Uint8Array([
            n.avcProfileIndication,
            n.profileCompatibility,
            n.avcLevelIndication,
          ]);
          return `avc${i.avcType}.${Di(l)}`;
        }
        if (!t || t.byteLength < 4)
          throw new TypeError(
            "AVC decoder description is not provided or is not at least 4 bytes long.",
          );
        return `avc${i.avcType}.${Di(t.subarray(1, 4))}`;
      } else if (e === "hevc") {
        let l, u, d, f, h, p;
        if (s)
          ((l = s.generalProfileSpace),
            (u = s.generalProfileIdc),
            (d = Ys(s.generalProfileCompatibilityFlags)),
            (f = s.generalTierFlag),
            (h = s.generalLevelIdc),
            (p = [...s.generalConstraintIndicatorFlags]));
        else {
          if (!t || t.byteLength < 23)
            throw new TypeError(
              "HEVC decoder description is not provided or is not at least 23 bytes long.",
            );
          const y = j(t),
            w = y.getUint8(1);
          ((l = (w >> 6) & 3),
            (u = w & 31),
            (d = Ys(y.getUint32(2))),
            (f = (w >> 5) & 1),
            (h = y.getUint8(12)),
            (p = []));
          for (let b = 0; b < 6; b++) p.push(y.getUint8(6 + b));
        }
        let m = "hev1.";
        for (
          m += ["", "A", "B", "C"][l] + u,
            m += ".",
            m += d.toString(16).toUpperCase(),
            m += ".",
            m += f === 0 ? "L" : "H",
            m += h;
          p.length > 0 && p[p.length - 1] === 0;
        )
          p.pop();
        return (
          p.length > 0 &&
            ((m += "."),
            (m += p.map((y) => y.toString(16).toUpperCase()).join("."))),
          m
        );
      } else {
        if (e === "vp8") return "vp8";
        if (e === "vp9") {
          if (!a) {
            const b = i.width * i.height;
            let T = ne(Ft).level;
            for (const S of Ft)
              if (b <= S.maxPictureSize) {
                T = S.level;
                break;
              }
            return `vp09.00.${T.toString().padStart(2, "0")}.08`;
          }
          const l = a.profile.toString().padStart(2, "0"),
            u = a.level.toString().padStart(2, "0"),
            d = a.bitDepth.toString().padStart(2, "0"),
            f = a.chromaSubsampling.toString().padStart(2, "0"),
            h = a.colourPrimaries.toString().padStart(2, "0"),
            p = a.transferCharacteristics.toString().padStart(2, "0"),
            m = a.matrixCoefficients.toString().padStart(2, "0"),
            y = a.videoFullRangeFlag.toString().padStart(2, "0");
          let w = `vp09.${l}.${u}.${d}.${f}`;
          return (
            (w += `.${h}.${p}.${m}.${y}`),
            w.endsWith(Ma) && (w = w.slice(0, -Ma.length)),
            w
          );
        } else if (e === "av1") {
          if (!o) {
            const S = i.width * i.height;
            let k = ne(Ft).level;
            for (const P of Ft)
              if (S <= P.maxPictureSize) {
                k = P.level;
                break;
              }
            return `av01.0.${k.toString().padStart(2, "0")}M.08`;
          }
          const l = o.profile,
            u = o.level.toString().padStart(2, "0"),
            d = o.tier ? "H" : "M",
            f = o.bitDepth.toString().padStart(2, "0"),
            h = o.monochrome ? "1" : "0",
            p =
              100 * o.chromaSubsamplingX +
              10 * o.chromaSubsamplingY +
              1 *
                (o.chromaSubsamplingX && o.chromaSubsamplingY
                  ? o.chromaSamplePosition
                  : 0);
          let m, y, w, b;
          Ta(o)
            ? ((m = o.colourPrimaries),
              (y = o.transferCharacteristics),
              (w = o.matrixCoefficients),
              (b = o.videoFullRangeFlag))
            : ((m = r?.primaries ? Ht[r.primaries] : 1),
              (y = r?.transfer ? qt[r.transfer] : 1),
              (w = r?.matrix ? jt[r.matrix] : 1),
              (b = r?.fullRange ? 1 : 0));
          let T = `av01.${l}.${u}${d}.${f}`;
          return (
            (T += `.${h}.${p.toString().padStart(3, "0")}`),
            (T += `.${m.toString().padStart(2, "0")}`),
            (T += `.${y.toString().padStart(2, "0")}`),
            (T += `.${w.toString().padStart(2, "0")}`),
            (T += `.${b}`),
            T.endsWith(Da) && (T = T.slice(0, -Da.length)),
            T
          );
        } else {
          if (e === "prores") return c ?? "apch";
          e !== null && xe(e);
        }
      }
      throw new TypeError(`Unhandled codec '${e}'.`);
    },
    za = (i) => {
      switch (i.codec) {
        case "avc":
          {
            let e = i.avcCodecInfo?.sequenceParameterSets[0];
            if (
              (!e &&
                i.codecDescription &&
                (e = Mn(i.codecDescription)?.sequenceParameterSets[0]),
              e)
            ) {
              const t = pi(e);
              if (t)
                return {
                  primaries: ht[t.colourPrimaries],
                  transfer: mt[t.transferCharacteristics],
                  matrix: pt[t.matrixCoefficients],
                  fullRange: !!t.fullRangeFlag,
                };
            }
          }
          break;
        case "hevc":
          {
            let e = i.hevcCodecInfo?.arrays.find(
              (t) => t.nalUnitType === de.SPS_NUT,
            )?.nalUnits[0];
            if (
              (!e &&
                i.codecDescription &&
                (e = ba(i.codecDescription)?.arrays.find(
                  (t) => t.nalUnitType === de.SPS_NUT,
                )?.nalUnits[0]),
              e)
            ) {
              const t = Er(e);
              if (t)
                return {
                  primaries: ht[t.colourPrimaries],
                  transfer: mt[t.transferCharacteristics],
                  matrix: pt[t.matrixCoefficients],
                  fullRange: !!t.fullRangeFlag,
                };
            }
          }
          break;
        case "vp8":
          break;
        case "vp9":
          if (i.vp9CodecInfo)
            return {
              primaries: ht[i.vp9CodecInfo.colourPrimaries],
              transfer: mt[i.vp9CodecInfo.transferCharacteristics],
              matrix: pt[i.vp9CodecInfo.matrixCoefficients],
              fullRange: !!i.vp9CodecInfo.videoFullRangeFlag,
            };
          break;
        case "av1":
          if (i.av1CodecInfo)
            return {
              primaries: ht[i.av1CodecInfo.colourPrimaries],
              transfer: mt[i.av1CodecInfo.transferCharacteristics],
              matrix: pt[i.av1CodecInfo.matrixCoefficients],
              fullRange: !!i.av1CodecInfo.videoFullRangeFlag,
            };
          break;
        case "prores":
          if (i.proresCodecInfo)
            return {
              primaries: ht[i.proresCodecInfo.colourPrimaries],
              transfer: mt[i.proresCodecInfo.transferCharacteristics],
              matrix: pt[i.proresCodecInfo.matrixCoefficients],
              fullRange: i.proresCodecInfo.fullRange,
            };
          break;
      }
      return {
        primaries: void 0,
        transfer: void 0,
        matrix: void 0,
        fullRange: void 0,
      };
    },
    nu = (i, e, t) => {
      if (i === "aac")
        return e >= 2 && t <= 24e3
          ? "mp4a.40.29"
          : t <= 24e3
            ? "mp4a.40.5"
            : "mp4a.40.2";
      if (i === "mp3") return "mp3";
      if (i === "opus") return "opus";
      if (i === "vorbis") return "vorbis";
      if (i === "flac") return "flac";
      if (i === "ac3") return "ac-3";
      if (i === "eac3") return "ec-3";
      if (i === "dts") return "dtsc";
      if (pe.includes(i)) return i;
      throw new TypeError(`Unhandled codec '${i}'.`);
    },
    Ln = (i) => {
      const {
        codec: e,
        codecDescription: t,
        aacCodecInfo: r,
        dtsFormat: n,
      } = i;
      if (e === "aac") {
        if (!r) throw new TypeError("AAC codec info must be provided.");
        if (r.isMpeg2) return "mp4a.67";
        {
          let s;
          return (
            r.objectType !== null ? (s = r.objectType) : (s = En(t).objectType),
            `mp4a.40.${s}`
          );
        }
      } else {
        if (e === "mp3") return "mp3";
        if (e === "opus") return "opus";
        if (e === "vorbis") return "vorbis";
        if (e === "flac") return "flac";
        if (e === "ac3") return "ac-3";
        if (e === "eac3") return "ec-3";
        if (e === "dts") return n ?? "dtsc";
        if (e && pe.includes(e)) return e;
      }
      throw new TypeError(`Unhandled codec '${e}'.`);
    },
    Gi = 48e3,
    Na = /^pcm-([usf])(\d+)(be)?$/,
    Ge = (i) => {
      if ((g(pe.includes(i)), i === "ulaw"))
        return {
          dataType: "ulaw",
          sampleSize: 1,
          littleEndian: !0,
          silentValue: 255,
        };
      if (i === "alaw")
        return {
          dataType: "alaw",
          sampleSize: 1,
          littleEndian: !0,
          silentValue: 213,
        };
      const e = Na.exec(i);
      g(e);
      let t;
      e[1] === "u"
        ? (t = "unsigned")
        : e[1] === "s"
          ? (t = "signed")
          : (t = "float");
      const r = Number(e[2]) / 8,
        n = e[3] !== "be",
        s = i === "pcm-u8" ? 2 ** 7 : 0;
      return { dataType: t, sampleSize: r, littleEndian: n, silentValue: s };
    },
    yt = (i) =>
      i.startsWith("avc1") || i.startsWith("avc3")
        ? "avc"
        : i.startsWith("hev1") || i.startsWith("hvc1")
          ? "hevc"
          : i === "vp8"
            ? "vp8"
            : i.startsWith("vp09")
              ? "vp9"
              : i.startsWith("av01")
                ? "av1"
                : Kt.includes(i)
                  ? "prores"
                  : i === "mp3" ||
                      i === "mp4a.69" ||
                      i === "mp4a.6B" ||
                      i === "mp4a.6b" ||
                      i === "mp4a.40.34"
                    ? "mp3"
                    : i.startsWith("mp4a.40.") || i === "mp4a.67"
                      ? "aac"
                      : i === "opus" || i === "Opus"
                        ? "opus"
                        : i === "vorbis"
                          ? "vorbis"
                          : i === "flac"
                            ? "flac"
                            : i === "ac-3" || i === "ac3"
                              ? "ac3"
                              : i === "ec-3" || i === "eac3"
                                ? "eac3"
                                : Mr.includes(i)
                                  ? "dts"
                                  : i === "ulaw"
                                    ? "ulaw"
                                    : i === "alaw"
                                      ? "alaw"
                                      : Na.test(i)
                                        ? i
                                        : i === "webvtt"
                                          ? "webvtt"
                                          : null,
    su = (i) =>
      i === "avc"
        ? { avc: { format: "avc" } }
        : i === "hevc"
          ? { hevc: { format: "hevc" } }
          : {},
    au = (i) =>
      i === "aac"
        ? { aac: { format: "aac" } }
        : i === "opus"
          ? { opus: { format: "opus" } }
          : {},
    ou = ["avc1", "avc3", "hev1", "hvc1", "vp8", "vp09", "av01", ...Kt],
    cu = /^(avc1|avc3)\.[0-9a-fA-F]{6}$/,
    lu =
      /^(hev1|hvc1)\.(?:[ABC]?\d+)\.[0-9a-fA-F]{1,8}\.[LH]\d+(?:\.[0-9a-fA-F]{1,2}){0,6}$/,
    uu = /^vp09(?:\.\d{2}){3}(?:(?:\.\d{2}){5})?$/,
    du =
      /^av01\.\d\.\d{2}[MH]\.\d{2}(?:\.\d\.\d{3}\.\d{2}\.\d{2}\.\d{2}\.\d)?$/,
    Hn = (i, e) => {
      if (!i) throw new TypeError("Video chunk metadata must be provided.");
      if (typeof i != "object")
        throw new TypeError("Video chunk metadata must be an object.");
      if (!i.decoderConfig)
        throw new TypeError(
          "Video chunk metadata must include a decoder configuration.",
        );
      if (typeof i.decoderConfig != "object")
        throw new TypeError(
          "Video chunk metadata decoder configuration must be an object.",
        );
      if (typeof i.decoderConfig.codec != "string")
        throw new TypeError(
          "Video chunk metadata decoder configuration must specify a codec string.",
        );
      if (!ou.some((t) => i.decoderConfig.codec.startsWith(t)))
        throw new TypeError(
          "Video chunk metadata decoder configuration codec string must be a valid video codec string as specified in the Mediabunny Codec Registry.",
        );
      if (
        !Number.isInteger(i.decoderConfig.codedWidth) ||
        i.decoderConfig.codedWidth <= 0
      )
        throw new TypeError(
          "Video chunk metadata decoder configuration must specify a valid codedWidth (positive integer).",
        );
      if (
        !Number.isInteger(i.decoderConfig.codedHeight) ||
        i.decoderConfig.codedHeight <= 0
      )
        throw new TypeError(
          "Video chunk metadata decoder configuration must specify a valid codedHeight (positive integer).",
        );
      if (
        i.decoderConfig.displayAspectWidth !== void 0 &&
        (!Number.isInteger(i.decoderConfig.displayAspectWidth) ||
          i.decoderConfig.displayAspectWidth <= 0)
      )
        throw new TypeError(
          "Video chunk metadata decoder configuration displayAspectWidth, when defined, must be a positive integer.",
        );
      if (
        i.decoderConfig.displayAspectHeight !== void 0 &&
        (!Number.isInteger(i.decoderConfig.displayAspectHeight) ||
          i.decoderConfig.displayAspectHeight <= 0)
      )
        throw new TypeError(
          "Video chunk metadata decoder configuration displayAspectHeight, when defined, must be a positive integer.",
        );
      if (
        (i.decoderConfig.displayAspectWidth !== void 0) !=
        (i.decoderConfig.displayAspectHeight !== void 0)
      )
        throw new TypeError(
          "Video chunk metadata decoder configuration must specify both displayAspectWidth and displayAspectHeight, or neither.",
        );
      if (
        i.decoderConfig.description !== void 0 &&
        !kr(i.decoderConfig.description)
      )
        throw new TypeError(
          "Video chunk metadata decoder configuration description, when defined, must be an ArrayBuffer or an ArrayBuffer view.",
        );
      if (i.decoderConfig.colorSpace !== void 0) {
        const { colorSpace: t } = i.decoderConfig;
        if (typeof t != "object")
          throw new TypeError(
            "Video chunk metadata decoder configuration colorSpace, when provided, must be an object.",
          );
        const r = Object.keys(Ht);
        if (t.primaries != null && !r.includes(t.primaries))
          throw new TypeError(
            `Video chunk metadata decoder configuration colorSpace primaries, when defined, must be one of ${r.join(", ")}.`,
          );
        const n = Object.keys(qt);
        if (t.transfer != null && !n.includes(t.transfer))
          throw new TypeError(
            `Video chunk metadata decoder configuration colorSpace transfer, when defined, must be one of ${n.join(", ")}.`,
          );
        const s = Object.keys(jt);
        if (t.matrix != null && !s.includes(t.matrix))
          throw new TypeError(
            `Video chunk metadata decoder configuration colorSpace matrix, when defined, must be one of ${s.join(", ")}.`,
          );
        if (t.fullRange != null && typeof t.fullRange != "boolean")
          throw new TypeError(
            "Video chunk metadata decoder configuration colorSpace fullRange, when defined, must be a boolean.",
          );
      }
      if (
        i.decoderConfig.codec.startsWith("avc1") ||
        i.decoderConfig.codec.startsWith("avc3")
      ) {
        if (!cu.test(i.decoderConfig.codec))
          throw new TypeError(
            "Video chunk metadata decoder configuration codec string for AVC must be a valid AVC codec string as specified in Section 3.4 of RFC 6381.",
          );
      } else if (
        i.decoderConfig.codec.startsWith("hev1") ||
        i.decoderConfig.codec.startsWith("hvc1")
      ) {
        if (!lu.test(i.decoderConfig.codec))
          throw new TypeError(
            "Video chunk metadata decoder configuration codec string for HEVC must be a valid HEVC codec string as specified in Section E.3 of ISO 14496-15.",
          );
      } else if (i.decoderConfig.codec.startsWith("vp8")) {
        if (i.decoderConfig.codec !== "vp8")
          throw new TypeError(
            'Video chunk metadata decoder configuration codec string for VP8 must be "vp8".',
          );
      } else if (i.decoderConfig.codec.startsWith("vp09")) {
        if (!uu.test(i.decoderConfig.codec))
          throw new TypeError(
            'Video chunk metadata decoder configuration codec string for VP9 must be a valid VP9 codec string as specified in Section "Codecs Parameter String" of https://www.webmproject.org/vp9/mp4/.',
          );
      } else if (i.decoderConfig.codec.startsWith("av01")) {
        if (!du.test(i.decoderConfig.codec))
          throw new TypeError(
            'Video chunk metadata decoder configuration codec string for AV1 must be a valid AV1 codec string as specified in Section "Codecs Parameter String" of https://aomediacodec.github.io/av1-isobmff/.',
          );
      } else if (
        Kt.some((t) => i.decoderConfig.codec.startsWith(t)) &&
        !Kt.some((t) => i.decoderConfig.codec === t)
      )
        throw new TypeError(
          `Video chunk metadata decoder configuration codec string for ProRes must be one of the valid ProRes four-character codes: ${Kt.join(", ")}.`,
        );
      if (e !== null && yt(i.decoderConfig.codec) !== e)
        throw new TypeError(
          `Video chunk metadata decoder configuration codec string '${i.decoderConfig.codec}' does not fit to the track codec '${e}'.`,
        );
    },
    fu = [
      "mp4a",
      "mp3",
      "opus",
      "vorbis",
      "flac",
      "ulaw",
      "alaw",
      "pcm",
      "ac-3",
      "ec-3",
      "dts",
    ],
    Xi = (i, e) => {
      if (!i) throw new TypeError("Audio chunk metadata must be provided.");
      if (typeof i != "object")
        throw new TypeError("Audio chunk metadata must be an object.");
      if (!i.decoderConfig)
        throw new TypeError(
          "Audio chunk metadata must include a decoder configuration.",
        );
      if (typeof i.decoderConfig != "object")
        throw new TypeError(
          "Audio chunk metadata decoder configuration must be an object.",
        );
      if (typeof i.decoderConfig.codec != "string")
        throw new TypeError(
          "Audio chunk metadata decoder configuration must specify a codec string.",
        );
      if (!fu.some((t) => i.decoderConfig.codec.startsWith(t)))
        throw new TypeError(
          "Audio chunk metadata decoder configuration codec string must be a valid audio codec string as specified in the Mediabunny Codec Registry.",
        );
      if (
        !Number.isInteger(i.decoderConfig.sampleRate) ||
        i.decoderConfig.sampleRate <= 0
      )
        throw new TypeError(
          "Audio chunk metadata decoder configuration must specify a valid sampleRate (positive integer).",
        );
      if (
        !Number.isInteger(i.decoderConfig.numberOfChannels) ||
        i.decoderConfig.numberOfChannels <= 0
      )
        throw new TypeError(
          "Audio chunk metadata decoder configuration must specify a valid numberOfChannels (positive integer).",
        );
      if (
        i.decoderConfig.description !== void 0 &&
        !kr(i.decoderConfig.description)
      )
        throw new TypeError(
          "Audio chunk metadata decoder configuration description, when defined, must be an ArrayBuffer or an ArrayBuffer view.",
        );
      if (
        i.decoderConfig.codec.startsWith("mp4a") &&
        i.decoderConfig.codec !== "mp4a.69" &&
        i.decoderConfig.codec !== "mp4a.6B" &&
        i.decoderConfig.codec !== "mp4a.6b"
      ) {
        if (
          ![
            "mp4a.40.2",
            "mp4a.40.02",
            "mp4a.40.5",
            "mp4a.40.05",
            "mp4a.40.29",
            "mp4a.67",
          ].includes(i.decoderConfig.codec)
        )
          throw new TypeError(
            "Audio chunk metadata decoder configuration codec string for AAC must be a valid AAC codec string as specified in https://www.w3.org/TR/webcodecs-aac-codec-registration/.",
          );
      } else if (
        i.decoderConfig.codec.startsWith("mp3") ||
        i.decoderConfig.codec.startsWith("mp4a")
      ) {
        if (
          i.decoderConfig.codec !== "mp3" &&
          i.decoderConfig.codec !== "mp4a.69" &&
          i.decoderConfig.codec !== "mp4a.6B" &&
          i.decoderConfig.codec !== "mp4a.6b"
        )
          throw new TypeError(
            'Audio chunk metadata decoder configuration codec string for MP3 must be "mp3", "mp4a.69" or "mp4a.6B".',
          );
      } else if (i.decoderConfig.codec.startsWith("opus")) {
        if (i.decoderConfig.codec !== "opus")
          throw new TypeError(
            'Audio chunk metadata decoder configuration codec string for Opus must be "opus".',
          );
        if (
          i.decoderConfig.description &&
          i.decoderConfig.description.byteLength < 18
        )
          throw new TypeError(
            "Audio chunk metadata decoder configuration description, when specified, is expected to be an Identification Header as specified in Section 5.1 of RFC 7845.",
          );
      } else if (i.decoderConfig.codec.startsWith("vorbis")) {
        if (i.decoderConfig.codec !== "vorbis")
          throw new TypeError(
            'Audio chunk metadata decoder configuration codec string for Vorbis must be "vorbis".',
          );
        if (!i.decoderConfig.description)
          throw new TypeError(
            "Audio chunk metadata decoder configuration for Vorbis must include a description, which is expected to adhere to the format described in https://www.w3.org/TR/webcodecs-vorbis-codec-registration/.",
          );
      } else if (i.decoderConfig.codec.startsWith("flac")) {
        if (i.decoderConfig.codec !== "flac")
          throw new TypeError(
            'Audio chunk metadata decoder configuration codec string for FLAC must be "flac".',
          );
        if (
          !i.decoderConfig.description ||
          i.decoderConfig.description.byteLength < 42
        )
          throw new TypeError(
            "Audio chunk metadata decoder configuration for FLAC must include a description, which is expected to adhere to the format described in https://www.w3.org/TR/webcodecs-flac-codec-registration/.",
          );
      } else if (
        i.decoderConfig.codec.startsWith("ac-3") ||
        i.decoderConfig.codec.startsWith("ac3")
      ) {
        if (i.decoderConfig.codec !== "ac-3")
          throw new TypeError(
            'Audio chunk metadata decoder configuration codec string for AC-3 must be "ac-3".',
          );
      } else if (
        i.decoderConfig.codec.startsWith("ec-3") ||
        i.decoderConfig.codec.startsWith("eac3")
      ) {
        if (i.decoderConfig.codec !== "ec-3")
          throw new TypeError(
            'Audio chunk metadata decoder configuration codec string for EC-3 must be "ec-3".',
          );
      } else if (i.decoderConfig.codec.startsWith("dts")) {
        if (!Mr.includes(i.decoderConfig.codec))
          throw new TypeError(
            `Audio chunk metadata decoder configuration codec string for DTS must be one of the following four-character codes: ${Mr.join(", ")}.`,
          );
      } else if (
        (i.decoderConfig.codec.startsWith("pcm") ||
          i.decoderConfig.codec.startsWith("ulaw") ||
          i.decoderConfig.codec.startsWith("alaw")) &&
        !pe.includes(i.decoderConfig.codec)
      )
        throw new TypeError(
          `Audio chunk metadata decoder configuration codec string for PCM must be one of the supported PCM codecs (${pe.join(", ")}).`,
        );
      if (e !== null && yt(i.decoderConfig.codec) !== e)
        throw new TypeError(
          `Audio chunk metadata decoder configuration codec string '${i.decoderConfig.codec}' does not fit to the track codec '${e}'.`,
        );
    },
    Ua = (i) => {
      if (!i) throw new TypeError("Subtitle metadata must be provided.");
      if (typeof i != "object")
        throw new TypeError("Subtitle metadata must be an object.");
      if (!i.config)
        throw new TypeError("Subtitle metadata must include a config object.");
      if (typeof i.config != "object")
        throw new TypeError("Subtitle metadata config must be an object.");
      if (typeof i.config.description != "string")
        throw new TypeError(
          "Subtitle metadata config description must be a string.",
        );
    };
  const Gt = 4,
    hu = [44100, 48e3, 32e3],
    mu = [
      -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, 32,
      40, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320, -1, -1, 32, 48,
      56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320, 384, -1, -1, 32, 64,
      96, 128, 160, 192, 224, 256, 288, 320, 352, 384, 416, 448, -1, -1, -1, -1,
      -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, 8, 16, 24, 32, 40,
      48, 56, 64, 80, 96, 112, 128, 144, 160, -1, -1, 8, 16, 24, 32, 40, 48, 56,
      64, 80, 96, 112, 128, 144, 160, -1, -1, 32, 48, 56, 64, 80, 96, 112, 128,
      144, 160, 176, 192, 224, 256, -1,
    ],
    Wa = 1483304551,
    Va = 1231971951,
    pu = (i, e, t, r, n) =>
      e === 0
        ? 0
        : e === 1
          ? Math.floor((144 * t) / (r << i)) + n
          : e === 2
            ? Math.floor((144 * t) / r) + n
            : (Math.floor((12 * t) / r) + n) * 4,
    gu = (i, e, t, r) =>
      e === 0
        ? 0
        : e === 1
          ? (144 * t) / (r << i)
          : e === 2
            ? (144 * t) / r
            : ((12 * t) / r) * 4,
    La = (i, e) => (i === 3 ? (e === 3 ? 21 : 36) : e === 3 ? 13 : 21),
    qn = (i, e) => {
      const t = i >>> 24,
        r = (i >>> 16) & 255,
        n = (i >>> 8) & 255,
        s = i & 255;
      if (t !== 255 && r !== 255 && n !== 255 && s !== 255)
        return { header: null, bytesAdvanced: 4 };
      if (t !== 255) return { header: null, bytesAdvanced: 1 };
      if ((r & 224) !== 224) return { header: null, bytesAdvanced: 1 };
      let a = 0,
        o = 0;
      r & 16 ? (a = r & 8 ? 0 : 1) : ((a = 1), (o = 1));
      const c = (r >> 3) & 3,
        l = (r >> 1) & 3,
        u = (n >> 4) & 15,
        d = ((n >> 2) & 3) % 3,
        f = (n >> 1) & 1,
        h = (s >> 6) & 3,
        p = (s >> 4) & 3,
        m = (s >> 3) & 1,
        y = (s >> 2) & 1,
        w = s & 3,
        b = mu[a * 16 * 4 + l * 16 + u];
      if (b === -1) return { header: null, bytesAdvanced: 1 };
      const T = b * 1e3,
        S = hu[d] >> (a + o),
        k = pu(a, l, T, S, f);
      if (e !== null && e < k) return { header: null, bytesAdvanced: 1 };
      let P;
      return (
        c === 3
          ? (P = l === 3 ? 384 : 1152)
          : l === 3
            ? (P = 384)
            : l === 2
              ? (P = 1152)
              : (P = 576),
        {
          header: {
            totalSize: k,
            mpegVersionId: c,
            lowSamplingFrequency: a,
            layer: l,
            bitrate: T,
            frequencyIndex: d,
            sampleRate: S,
            channel: h,
            modeExtension: p,
            copyright: m,
            original: y,
            emphasis: w,
            audioSamplesInFrame: P,
          },
          bytesAdvanced: 1,
        }
      );
    },
    wu = (i) => {
      let e = 127,
        t = 0,
        r = i;
      for (; (e ^ 2147483647) !== 0;)
        ((t = r & ~e),
          (t <<= 1),
          (t |= r & e),
          (e = ((e + 1) << 8) - 1),
          (r = t));
      return t;
    },
    jn = (i) => {
      let e = 2130706432,
        t = 0;
      for (; e !== 0;) ((t >>= 1), (t |= i & e), (e >>= 8));
      return t;
    };
  var Or;
  (function (i) {
    ((i[(i.FrameCount = 1)] = "FrameCount"),
      (i[(i.FileSize = 2)] = "FileSize"),
      (i[(i.Toc = 4)] = "Toc"));
  })(Or || (Or = {}));
  const Qi = (i) => (i === 3 ? 1 : 2);
  class bt {
    constructor(e) {
      this.input = e;
    }
    dispose() {}
  }
  const Ue = new Uint8Array(0);
  class Z {
    constructor(e, t, r, n, s = -1, a, o) {
      if (
        ((this.data = e),
        (this.type = t),
        (this.timestamp = r),
        (this.duration = n),
        (this.sequenceNumber = s),
        e === Ue && a === void 0)
      )
        throw new Error(
          "Internal error: byteLength must be explicitly provided when constructing metadata-only packets.",
        );
      if ((a === void 0 && (a = e.byteLength), !(e instanceof Uint8Array)))
        throw new TypeError("data must be a Uint8Array.");
      if (t !== "key" && t !== "delta")
        throw new TypeError('type must be either "key" or "delta".');
      if (!Number.isFinite(r))
        throw new TypeError("timestamp must be a number.");
      if (!Number.isFinite(n) || n < 0)
        throw new TypeError("duration must be a non-negative number.");
      if (!Number.isFinite(s))
        throw new TypeError("sequenceNumber must be a number.");
      if (!Number.isInteger(a) || a < 0)
        throw new TypeError("byteLength must be a non-negative integer.");
      if (o !== void 0 && (typeof o != "object" || !o))
        throw new TypeError("sideData, when provided, must be an object.");
      if (o?.alpha !== void 0 && !(o.alpha instanceof Uint8Array))
        throw new TypeError(
          "sideData.alpha, when provided, must be a Uint8Array.",
        );
      if (
        o?.alphaByteLength !== void 0 &&
        (!Number.isInteger(o.alphaByteLength) || o.alphaByteLength < 0)
      )
        throw new TypeError(
          "sideData.alphaByteLength, when provided, must be a non-negative integer.",
        );
      ((this.byteLength = a),
        (this.sideData = o ?? {}),
        this.sideData.alpha &&
          this.sideData.alphaByteLength === void 0 &&
          (this.sideData.alphaByteLength = this.sideData.alpha.byteLength));
    }
    get isMetadataOnly() {
      return this.data === Ue;
    }
    get microsecondTimestamp() {
      return Math.trunc(_t * this.timestamp);
    }
    get microsecondDuration() {
      return Math.trunc(_t * this.duration);
    }
    toEncodedVideoChunk() {
      if (this.isMetadataOnly)
        throw new TypeError(
          "Metadata-only packets cannot be converted to a video chunk.",
        );
      if (typeof EncodedVideoChunk > "u")
        throw new Error(
          "EncodedVideoChunk is not available in this environment.",
        );
      return new EncodedVideoChunk({
        data: this.data,
        type: this.type,
        timestamp: this.microsecondTimestamp,
        duration: this.microsecondDuration,
      });
    }
    alphaToEncodedVideoChunk(e = this.type) {
      if (!this.sideData.alpha)
        throw new TypeError("This packet does not contain alpha side data.");
      if (this.isMetadataOnly)
        throw new TypeError(
          "Metadata-only packets cannot be converted to a video chunk.",
        );
      if (typeof EncodedVideoChunk > "u")
        throw new Error(
          "EncodedVideoChunk is not available in this environment.",
        );
      return new EncodedVideoChunk({
        data: this.sideData.alpha,
        type: e,
        timestamp: this.microsecondTimestamp,
        duration: this.microsecondDuration,
      });
    }
    toEncodedAudioChunk() {
      if (this.isMetadataOnly)
        throw new TypeError(
          "Metadata-only packets cannot be converted to an audio chunk.",
        );
      if (typeof EncodedAudioChunk > "u")
        throw new Error(
          "EncodedAudioChunk is not available in this environment.",
        );
      return new EncodedAudioChunk({
        data: this.data,
        type: this.type,
        timestamp: this.microsecondTimestamp,
        duration: this.microsecondDuration,
      });
    }
    static fromEncodedChunk(e, t) {
      if (!(e instanceof EncodedVideoChunk || e instanceof EncodedAudioChunk))
        throw new TypeError(
          "chunk must be an EncodedVideoChunk or EncodedAudioChunk.",
        );
      const r = new Uint8Array(e.byteLength);
      return (
        e.copyTo(r),
        new Z(
          r,
          e.type,
          e.timestamp / 1e6,
          (e.duration ?? 0) / 1e6,
          void 0,
          void 0,
          t,
        )
      );
    }
    clone(e) {
      if (e !== void 0 && (typeof e != "object" || e === null))
        throw new TypeError("options, when provided, must be an object.");
      if (e?.data !== void 0 && !(e.data instanceof Uint8Array))
        throw new TypeError(
          "options.data, when provided, must be a Uint8Array.",
        );
      if (e?.type !== void 0 && e.type !== "key" && e.type !== "delta")
        throw new TypeError(
          'options.type, when provided, must be either "key" or "delta".',
        );
      if (e?.timestamp !== void 0 && !Number.isFinite(e.timestamp))
        throw new TypeError(
          "options.timestamp, when provided, must be a number.",
        );
      if (e?.duration !== void 0 && !Number.isFinite(e.duration))
        throw new TypeError(
          "options.duration, when provided, must be a number.",
        );
      if (e?.sequenceNumber !== void 0 && !Number.isFinite(e.sequenceNumber))
        throw new TypeError(
          "options.sequenceNumber, when provided, must be a number.",
        );
      if (
        e?.sideData !== void 0 &&
        (typeof e.sideData != "object" || e.sideData === null)
      )
        throw new TypeError(
          "options.sideData, when provided, must be an object.",
        );
      return new Z(
        e?.data ?? this.data,
        e?.type ?? this.type,
        e?.timestamp ?? this.timestamp,
        e?.duration ?? this.duration,
        e?.sequenceNumber ?? this.sequenceNumber,
        this.byteLength,
        e?.sideData ?? this.sideData,
      );
    }
  }
  const Ha = (i) => {
      let t =
        (i.hasVideo ? "video/" : i.hasAudio ? "audio/" : "application/") +
        (i.isQuickTime ? "quicktime" : "mp4");
      if (i.codecStrings.length > 0) {
        const r = [...new Set(i.codecStrings)];
        t += `; codecs="${r.join(", ")}"`;
      }
      return t;
    },
    qa = (i) => {
      const e = j(i);
      let t = 0;
      const r = e.getUint8(t);
      ((t += 1), (t += 3));
      const n = Di(i.subarray(t, t + 16));
      t += 16;
      let s = null;
      if (r > 0) {
        const o = e.getUint32(t);
        if (((t += 4), o > 0)) {
          s = [];
          for (let c = 0; c < o; c++)
            (s.push(Di(i.subarray(t, t + 16))), (t += 16));
        }
      }
      const a = e.getUint32(t);
      return ((t += 4), { systemId: n, keyIds: s, data: i.slice(t, t + a) });
    },
    ja = (i, e) => i.systemId === e.systemId && na(i.data, e.data);
  const kt = 8,
    Xt = 16,
    Rt = (i) => {
      let e = R(i);
      const t = le(i, 4);
      let r = 8;
      e === 1 && ((e = Le(i)), (r = 16));
      const s = e - r;
      return s < 0
        ? null
        : { name: t, totalSize: e, headerSize: r, contentSize: s };
    },
    Qt = (i) => ni(i) / 65536,
    $n = (i) => ni(i) / 1073741824,
    Kn = (i) => {
      let e = 0;
      for (let t = 0; t < 4; t++) {
        e <<= 7;
        const r = U(i);
        if (((e |= r & 127), (r & 128) === 0)) break;
      }
      return e;
    },
    et = (i) => {
      let e = fe(i);
      return (
        i.skip(2),
        (e = Math.min(e, i.remainingLength)),
        Oe.decode(N(i, e))
      );
    },
    yu = (i) => {
      const e = Rt(i);
      if (!e || e.name !== "data" || i.remainingLength < 8) return null;
      const t = R(i);
      i.skip(4);
      const r = N(i, e.contentSize - 8);
      switch (t) {
        case 1:
          return Oe.decode(r);
        case 2:
          return new Mi("utf-16be").decode(r);
        case 13:
          return new hi(r, "image/jpeg");
        case 14:
          return new hi(r, "image/png");
        case 27:
          return new hi(r, "image/bmp");
        default:
          return r;
      }
    };
  const tt = 16,
    Tt = new Uint32Array(256),
    yi = new Uint32Array(256),
    bi = new Uint32Array(256),
    ki = new Uint32Array(256),
    Ti = new Uint32Array(256),
    Ce = new Uint32Array(256),
    $a = new Uint32Array(10);
  let Ka = !1;
  const bu = () => {
    const i = new Uint8Array(256),
      e = new Uint8Array(256),
      t = new Uint8Array(256);
    for (let s = 0, a = 1; s < 256; s++)
      ((t[s] = a), (e[a] = s), (a = a ^ (a << 1) ^ (a & 128 ? 283 : 0)));
    const r = (s, a) => (s && a ? t[(e[s] + e[a]) % 255] : 0);
    i[0] = 99;
    for (let s = 1; s < 256; s++) {
      const a = t[255 - e[s]];
      let o = a ^ (a << 1) ^ (a << 2) ^ (a << 3) ^ (a << 4);
      ((o = (o >>> 8) ^ (o & 255) ^ 99), (i[s] = o));
    }
    for (let s = 0; s < 256; s++) {
      const a = i[s],
        o = i.indexOf(s);
      ((Tt[s] = (a << 24) | (a << 16) | (a << 8) | a),
        (Ce[s] = (o << 24) | (o << 16) | (o << 8) | o));
      const c = r(o, 14),
        l = r(o, 9),
        u = r(o, 13),
        d = r(o, 11),
        f = (c << 24) | (l << 16) | (u << 8) | d;
      ((yi[s] = f),
        (bi[s] = (f >>> 8) | (f << 24)),
        (ki[s] = (f >>> 16) | (f << 16)),
        (Ti[s] = (f >>> 24) | (f << 8)));
    }
    let n = 1;
    for (let s = 0; s < 10; s++)
      (($a[s] = n << 24), (n = (n << 1) ^ (n & 128 ? 283 : 0)));
    Ka = !0;
  };
  class Ga {
    constructor() {
      ((this.roundkey = new Uint32Array(44)),
        (this.iv = new Uint32Array(tt / Uint32Array.BYTES_PER_ELEMENT)),
        (this.in = new Uint8Array(tt)),
        (this.out = new Uint8Array(tt)),
        (this.inView = new DataView(this.in.buffer)),
        (this.outView = new DataView(this.out.buffer)));
    }
    init({ key: e, iv: t }) {
      (g(e.byteLength === 16), g(t.byteLength === 16), Ka || bu());
      const r = new DataView(e.buffer, e.byteOffset, e.byteLength),
        n = new DataView(t.buffer, t.byteOffset, t.byteLength);
      ((this.roundkey[0] = r.getUint32(0, !1)),
        (this.roundkey[1] = r.getUint32(4, !1)),
        (this.roundkey[2] = r.getUint32(8, !1)),
        (this.roundkey[3] = r.getUint32(12, !1)),
        (this.iv[0] = n.getUint32(0, !1)),
        (this.iv[1] = n.getUint32(4, !1)),
        (this.iv[2] = n.getUint32(8, !1)),
        (this.iv[3] = n.getUint32(12, !1)));
      for (let s = 4; s < 44; s += 4) {
        const a = this.roundkey[s - 1];
        ((this.roundkey[s] =
          this.roundkey[s - 4] ^
          (Tt[(a >>> 16) & 255] & 4278190080) ^
          (Tt[(a >>> 8) & 255] & 16711680) ^
          (Tt[(a >>> 0) & 255] & 65280) ^
          (Tt[(a >>> 24) & 255] & 255) ^
          $a[s / 4 - 1]),
          (this.roundkey[s + 1] = this.roundkey[s - 3] ^ this.roundkey[s]),
          (this.roundkey[s + 2] = this.roundkey[s - 2] ^ this.roundkey[s + 1]),
          (this.roundkey[s + 3] = this.roundkey[s - 1] ^ this.roundkey[s + 2]));
      }
      for (let s = 0, a = 40; s < a; s += 4, a -= 4)
        for (let o = 0; o < 4; o++) {
          const c = this.roundkey[s + o];
          ((this.roundkey[s + o] = this.roundkey[a + o]),
            (this.roundkey[a + o] = c));
        }
      for (let s = 4; s < 40; s += 4)
        for (let a = 0; a < 4; a++) {
          const o = this.roundkey[s + a];
          this.roundkey[s + a] =
            yi[Tt[(o >>> 24) & 255] & 255] ^
            bi[Tt[(o >>> 16) & 255] & 255] ^
            ki[Tt[(o >>> 8) & 255] & 255] ^
            Ti[Tt[(o >>> 0) & 255] & 255];
        }
    }
    decrypt() {
      let e = this.inView.getUint32(0, !1) ^ this.roundkey[0],
        t = this.inView.getUint32(4, !1) ^ this.roundkey[1],
        r = this.inView.getUint32(8, !1) ^ this.roundkey[2],
        n = this.inView.getUint32(12, !1) ^ this.roundkey[3];
      const s = this.inView.getUint32(0, !1),
        a = this.inView.getUint32(4, !1),
        o = this.inView.getUint32(8, !1),
        c = this.inView.getUint32(12, !1);
      let l, u, d, f;
      for (let w = 1; w < 10; w++) {
        const b = w * 4;
        ((l =
          yi[e >>> 24] ^
          bi[(n >>> 16) & 255] ^
          ki[(r >>> 8) & 255] ^
          Ti[t & 255] ^
          this.roundkey[b]),
          (u =
            yi[t >>> 24] ^
            bi[(e >>> 16) & 255] ^
            ki[(n >>> 8) & 255] ^
            Ti[r & 255] ^
            this.roundkey[b + 1]),
          (d =
            yi[r >>> 24] ^
            bi[(t >>> 16) & 255] ^
            ki[(e >>> 8) & 255] ^
            Ti[n & 255] ^
            this.roundkey[b + 2]),
          (f =
            yi[n >>> 24] ^
            bi[(r >>> 16) & 255] ^
            ki[(t >>> 8) & 255] ^
            Ti[e & 255] ^
            this.roundkey[b + 3]),
          (e = l),
          (t = u),
          (r = d),
          (n = f));
      }
      const h =
          (Ce[(e >>> 24) & 255] & 4278190080) ^
          (Ce[(n >>> 16) & 255] & 16711680) ^
          (Ce[(r >>> 8) & 255] & 65280) ^
          (Ce[(t >>> 0) & 255] & 255) ^
          this.roundkey[40],
        p =
          (Ce[(t >>> 24) & 255] & 4278190080) ^
          (Ce[(e >>> 16) & 255] & 16711680) ^
          (Ce[(n >>> 8) & 255] & 65280) ^
          (Ce[(r >>> 0) & 255] & 255) ^
          this.roundkey[41],
        m =
          (Ce[(r >>> 24) & 255] & 4278190080) ^
          (Ce[(t >>> 16) & 255] & 16711680) ^
          (Ce[(e >>> 8) & 255] & 65280) ^
          (Ce[(n >>> 0) & 255] & 255) ^
          this.roundkey[42],
        y =
          (Ce[(n >>> 24) & 255] & 4278190080) ^
          (Ce[(r >>> 16) & 255] & 16711680) ^
          (Ce[(t >>> 8) & 255] & 65280) ^
          (Ce[(e >>> 0) & 255] & 255) ^
          this.roundkey[43];
      (this.outView.setUint32(0, h ^ this.iv[0], !1),
        this.outView.setUint32(4, p ^ this.iv[1], !1),
        this.outView.setUint32(8, m ^ this.iv[2], !1),
        this.outView.setUint32(12, y ^ this.iv[3], !1),
        (this.iv[0] = s),
        (this.iv[1] = a),
        (this.iv[2] = o),
        (this.iv[3] = c));
    }
  }
  const ku = (i, e, t) => {
    let r = !1,
      n = 0;
    const s = 2 ** 16,
      a = 16,
      o = new Ga();
    return new ReadableStream({
      pull: async (c) => {
        r || (o.init(await e()), (r = !0));
        const l = s + a;
        let u = i.requestSliceRange(n, 0, l);
        if ((A(u) && (u = await u), !u || u.length === 0))
          throw new Error("Invalid ciphertext.");
        const d = u.length;
        if (d % 16 !== 0) throw new Error("Invalid ciphertext.");
        const f = d === l ? d - a : d,
          h = N(u, f),
          p = new Uint8Array(f);
        for (let m = 0; m < f; m += 16)
          (o.in.set(h.subarray(m, m + 16)), o.decrypt(), p.set(o.out, m));
        if (f < d) (c.enqueue(p), (n += f));
        else {
          const m = p[f - 1];
          if (m === 0 || m > 16)
            throw new Error(
              "Invalid PKCS#7 padding. Incorrect key or corrupted data.",
            );
          const y = p.subarray(0, f - m);
          (c.enqueue(y), c.close(), t());
        }
      },
      cancel: () => {
        t();
      },
    });
  };
  class Gn extends bt {
    constructor(e) {
      (super(e),
        (this.moovSlice = null),
        (this.currentTrack = null),
        (this.tracks = []),
        (this.metadataPromise = null),
        (this.movieTimescale = -1),
        (this.movieDurationInTimescale = -1),
        (this.movieMatrix = Fi),
        (this.isQuickTime = !1),
        (this.metadataTags = {}),
        (this.currentMetadataKeys = null),
        (this.isFragmented = !1),
        (this.fragmentTrackDefaults = []),
        (this.psshBoxes = []),
        (this.currentFragment = null),
        (this.lastReadFragment = null),
        (this.decryptionKeyCache = new Map()),
        (this.reader = e._reader));
    }
    async getTrackBackings() {
      return (
        await this.readMetadata(),
        this.tracks.map((e) => e.trackBacking)
      );
    }
    async getMimeType() {
      await this.readMetadata();
      const e = await this.getTrackBackings(),
        t = await Promise.all(
          e.map((r) => r.getDecoderConfig().then((n) => n?.codec ?? null)),
        );
      return Ha({
        isQuickTime: this.isQuickTime,
        hasVideo: this.tracks.some((r) => r.info?.type === "video"),
        hasAudio: this.tracks.some((r) => r.info?.type === "audio"),
        codecStrings: t.filter(Boolean),
      });
    }
    async getMetadataTags() {
      return (await this.readMetadata(), this.metadataTags);
    }
    readMetadata() {
      return (this.metadataPromise ??= (async () => {
        let e = 0,
          t = !1,
          r = !1;
        for (;;) {
          let n = this.reader.requestSliceRange(e, kt, Xt);
          if ((A(n) && (n = await n), !n)) break;
          const s = e,
            a = Rt(n);
          if (!a) break;
          if (a.name === "ftyp" || a.name === "styp") {
            const o = le(n, 4);
            this.isQuickTime = o === "qt  ";
          } else if (a.name === "moov") {
            let o = this.reader.requestSlice(n.filePos, a.contentSize);
            if ((A(o) && (o = await o), !o)) break;
            ((this.moovSlice = o), this.readContiguousBoxes(this.moovSlice));
            for (const c of this.tracks) {
              const l =
                c.editListPreviousSegmentDurations / this.movieTimescale;
              c.editListOffset -= Math.round(l * c.timescale);
            }
            ((t =
              this.isFragmented &&
              this.reader.fileSize !== null &&
              this.reader.fileSize > s + a.totalSize),
              (r = !0));
            break;
          } else if (a.name === "moof") {
            if (!this.input._initInput)
              throw new Error(
                '"moof" box encountered with no "moov" box present; this file is likely a Segment as described in ISO/IEC 14496-12 Section 8.16. A separate init file that contains a "moov" box is required to read this file, please provide it using InputOptions.initInput.',
              );
            (await this.copyMetadataFromInitInput(this.input._initInput),
              (t = !1),
              (r = !0));
            break;
          }
          e = s + a.totalSize;
        }
        if (
          (!r &&
            this.input._initInput &&
            (await this.copyMetadataFromInitInput(this.input._initInput)),
          t)
        ) {
          g(this.reader.fileSize !== null);
          let n = this.reader.requestSlice(this.reader.fileSize - 4, 4);
          (A(n) && (n = await n), g(n));
          const s = R(n),
            a = this.reader.fileSize - s;
          if (a >= 0 && a <= this.reader.fileSize - Xt) {
            let o = this.reader.requestSliceRange(a, kt, Xt);
            if ((A(o) && (o = await o), o)) {
              const c = Rt(o);
              if (c && c.name === "mfra") {
                let l = this.reader.requestSlice(o.filePos, c.contentSize);
                (A(l) && (l = await l), l && this.readContiguousBoxes(l));
              }
            }
          }
        }
      })());
    }
    async copyMetadataFromInitInput(e) {
      const t = await e._getDemuxer();
      if (t.constructor !== Gn)
        throw new Error("Init input must match the input's format.");
      (await t.readMetadata(),
        (this.movieTimescale = t.movieTimescale),
        (this.movieDurationInTimescale = t.movieDurationInTimescale),
        (this.movieMatrix = t.movieMatrix),
        (this.metadataTags = t.metadataTags),
        (this.isFragmented = !0),
        (this.fragmentTrackDefaults = t.fragmentTrackDefaults),
        (this.psshBoxes = t.psshBoxes));
      for (const r of t.tracks) {
        const n = {
          id: r.id,
          demuxer: this,
          trackBacking: null,
          disposition: r.disposition,
          timescale: r.timescale,
          durationInMediaTimescale: r.durationInMediaTimescale,
          durationInMovieTimescale: r.durationInMovieTimescale,
          matrix: r.matrix,
          internalCodecId: r.internalCodecId,
          name: r.name,
          languageCode: r.languageCode,
          sampleTableByteOffset: null,
          sampleTable: null,
          fragmentLookupTable: [],
          currentFragmentState: null,
          fragmentPositionCache: [],
          editListPreviousSegmentDurations: r.editListPreviousSegmentDurations,
          editListOffset: r.editListOffset,
          encryptionInfo: r.encryptionInfo,
          encryptionAuxInfo: null,
          frmaCodecString: null,
          maxBitrate: r.maxBitrate,
          avgBitrate: r.avgBitrate,
          info: r.info,
        };
        if (r.trackBacking) {
          if ((g(n.info), n.info.type === "video" && n.info.width !== -1)) {
            const s = n;
            ((n.trackBacking = new Qa(s)), this.tracks.push(n));
          } else if (
            n.info.type === "audio" &&
            n.info.numberOfChannels !== -1
          ) {
            const s = n;
            ((n.trackBacking = new Ya(s)), this.tracks.push(n));
          }
        }
      }
    }
    getSampleTableForTrack(e) {
      if (e.sampleTable) return e.sampleTable;
      const t = {
        sampleTimingEntries: [],
        sampleCompositionTimeOffsets: [],
        sampleSizes: [],
        keySampleIndices: null,
        chunkOffsets: [],
        sampleToChunk: [],
        presentationTimestamps: null,
        presentationTimestampIndexMap: null,
      };
      if (((e.sampleTable = t), e.sampleTableByteOffset === null)) return t;
      g(this.moovSlice);
      const r = this.moovSlice.slice(e.sampleTableByteOffset);
      if (
        ((this.currentTrack = e),
        this.traverseBox(r),
        (this.currentTrack = null),
        e.info?.type === "audio" &&
          e.info.codec &&
          pe.includes(e.info.codec) &&
          t.sampleCompositionTimeOffsets.length === 0)
      ) {
        g(e.info?.type === "audio");
        const s = Ge(e.info.codec),
          a = [],
          o = [];
        for (let c = 0; c < t.sampleToChunk.length; c++) {
          const l = t.sampleToChunk[c],
            u = t.sampleToChunk[c + 1],
            d =
              (u ? u.startChunkIndex : t.chunkOffsets.length) -
              l.startChunkIndex;
          for (let f = 0; f < d; f++) {
            const h = l.startSampleIndex + f * l.samplesPerChunk,
              p = h + l.samplesPerChunk,
              m = X(t.sampleTimingEntries, h, (E) => E.startIndex),
              y = t.sampleTimingEntries[m],
              w = X(t.sampleTimingEntries, p, (E) => E.startIndex),
              b = t.sampleTimingEntries[w],
              T = y.startDecodeTimestamp + (h - y.startIndex) * y.delta,
              k = b.startDecodeTimestamp + (p - b.startIndex) * b.delta - T,
              P = ne(a);
            P && P.delta === k
              ? P.count++
              : a.push({
                  startIndex: l.startChunkIndex + f,
                  startDecodeTimestamp: T,
                  count: 1,
                  delta: k,
                });
            const I =
              l.samplesPerChunk * s.sampleSize * e.info.numberOfChannels;
            o.push(I);
          }
          ((l.startSampleIndex = l.startChunkIndex), (l.samplesPerChunk = 1));
        }
        ((t.sampleTimingEntries = a), (t.sampleSizes = o));
      }
      if (t.sampleCompositionTimeOffsets.length > 0) {
        t.presentationTimestamps = [];
        for (const s of t.sampleTimingEntries)
          for (let a = 0; a < s.count; a++)
            t.presentationTimestamps.push({
              presentationTimestamp: s.startDecodeTimestamp + a * s.delta,
              sampleIndex: s.startIndex + a,
            });
        for (const s of t.sampleCompositionTimeOffsets)
          for (let a = 0; a < s.count; a++) {
            const o = s.startIndex + a,
              c = t.presentationTimestamps[o];
            c && (c.presentationTimestamp += s.offset);
          }
        (t.presentationTimestamps.sort(
          (s, a) => s.presentationTimestamp - a.presentationTimestamp,
        ),
          (t.presentationTimestampIndexMap = Array(
            t.presentationTimestamps.length,
          ).fill(-1)));
        for (let s = 0; s < t.presentationTimestamps.length; s++)
          t.presentationTimestampIndexMap[
            t.presentationTimestamps[s].sampleIndex
          ] = s;
      }
      return t;
    }
    async readFragment(e) {
      if (this.lastReadFragment?.moofOffset === e) return this.lastReadFragment;
      let t = this.reader.requestSliceRange(e, kt, Xt);
      (A(t) && (t = await t), g(t));
      const r = Rt(t);
      g(r?.name === "moof");
      let n = this.reader.requestSlice(e, r.totalSize);
      (A(n) && (n = await n), g(n), this.traverseBox(n));
      const s = this.lastReadFragment;
      g(s && s.moofOffset === e);
      for (const [, a] of s.trackData) {
        const o = a.track,
          { fragmentPositionCache: c } = o;
        if (!a.startTimestampIsFinal) {
          const u = o.fragmentLookupTable.find(
            (d) => d.moofOffset === s.moofOffset,
          );
          if (u) Qn(a, u.timestamp);
          else {
            const d = X(c, s.moofOffset - 1, (f) => f.moofOffset);
            if (d !== -1) {
              const f = c[d];
              Qn(a, f.endTimestamp);
            }
          }
          a.startTimestampIsFinal = !0;
        }
        const l = X(c, a.startTimestamp, (u) => u.startTimestamp);
        if (
          ((l === -1 || c[l].moofOffset !== s.moofOffset) &&
            c.splice(l + 1, 0, {
              moofOffset: s.moofOffset,
              startTimestamp: a.startTimestamp,
              endTimestamp: a.endTimestamp,
            }),
          a.encryptionAuxInfo && o.encryptionInfo)
        ) {
          const u = await io(
            this.reader,
            o.encryptionInfo,
            a.encryptionAuxInfo,
          );
          for (let d = 0; d < Math.min(a.samples.length, u.length); d++) {
            const f = u[d];
            a.samples[d].encryption = f;
          }
        }
      }
      return s;
    }
    readContiguousBoxes(e) {
      const t = e.filePos;
      for (; e.filePos - t <= e.length - kt && this.traverseBox(e););
    }
    *iterateContiguousBoxes(e) {
      const t = e.filePos;
      for (; e.filePos - t <= e.length - kt;) {
        const r = e.filePos,
          n = Rt(e);
        if (!n) break;
        (yield { boxInfo: n, slice: e }, (e.filePos = r + n.totalSize));
      }
    }
    traverseBox(e) {
      const t = e.filePos,
        r = Rt(e);
      if (!r) return !1;
      const n = e.filePos,
        s = t + r.totalSize;
      switch (r.name) {
        case "mdia":
        case "minf":
        case "dinf":
        case "mfra":
        case "edts":
        case "sinf":
        case "schi":
          this.readContiguousBoxes(e.slice(n, r.contentSize));
          break;
        case "mvhd":
          {
            const a = U(e);
            (e.skip(3),
              a === 1
                ? (e.skip(16),
                  (this.movieTimescale = R(e)),
                  (this.movieDurationInTimescale = Le(e)))
                : (e.skip(8),
                  (this.movieTimescale = R(e)),
                  (this.movieDurationInTimescale = R(e))),
              e.skip(16),
              (this.movieMatrix = Za(e)));
          }
          break;
        case "trak":
          {
            const a = {
              id: -1,
              demuxer: this,
              trackBacking: null,
              disposition: { ...wt, primary: !1 },
              info: null,
              timescale: -1,
              durationInMovieTimescale: -1,
              durationInMediaTimescale: -1,
              matrix: Fi,
              internalCodecId: null,
              name: null,
              languageCode: Pe,
              sampleTableByteOffset: -1,
              sampleTable: null,
              fragmentLookupTable: [],
              currentFragmentState: null,
              fragmentPositionCache: [],
              editListPreviousSegmentDurations: 0,
              editListOffset: 0,
              encryptionInfo: null,
              encryptionAuxInfo: null,
              frmaCodecString: null,
              maxBitrate: null,
              avgBitrate: null,
            };
            if (
              ((this.currentTrack = a),
              this.readContiguousBoxes(e.slice(n, r.contentSize)),
              a.id !== -1 && a.timescale !== -1 && a.info !== null)
            ) {
              if (a.info.type === "video" && a.info.width !== -1) {
                const o = a;
                ((a.trackBacking = new Qa(o)), this.tracks.push(a));
              } else if (
                a.info.type === "audio" &&
                a.info.numberOfChannels !== -1
              ) {
                const o = a;
                ((a.trackBacking = new Ya(o)), this.tracks.push(a));
              }
            }
            this.currentTrack = null;
          }
          break;
        case "tkhd":
          {
            const a = this.currentTrack;
            if (!a) break;
            const o = U(e),
              l = !!(It(e) & 1);
            if (((a.disposition.default = l), o === 0))
              (e.skip(8),
                (a.id = R(e)),
                e.skip(4),
                (a.durationInMovieTimescale = R(e)));
            else if (o === 1)
              (e.skip(16),
                (a.id = R(e)),
                e.skip(4),
                (a.durationInMovieTimescale = Le(e)));
            else throw new Error(`Incorrect track header version ${o}.`);
            e.skip(16);
            const u = Za(e);
            a.matrix = ai(u, this.movieMatrix);
          }
          break;
        case "elst":
          {
            const a = this.currentTrack;
            if (!a) break;
            const o = U(e);
            e.skip(3);
            let c = !1,
              l = 0;
            const u = R(e);
            for (let d = 0; d < u; d++) {
              const f = o === 1 ? Le(e) : R(e),
                h = o === 1 ? hf(e) : ni(e),
                p = Qt(e);
              if (c) {
                L._warn(
                  "Unsupported edit list: multiple edits are not currently supported. Only using first edit.",
                );
                break;
              }
              if (h === -1) {
                l += f;
                continue;
              }
              if (p !== 1) {
                L._warn("Unsupported edit list entry: media rate must be 1.");
                break;
              }
              ((a.editListPreviousSegmentDurations = l),
                (a.editListOffset = h),
                (c = !0));
            }
          }
          break;
        case "mdhd":
          {
            const a = this.currentTrack;
            if (!a) break;
            const o = U(e);
            (e.skip(3),
              o === 0
                ? (e.skip(8),
                  (a.timescale = R(e)),
                  (a.durationInMediaTimescale = R(e)))
                : o === 1 &&
                  (e.skip(16),
                  (a.timescale = R(e)),
                  (a.durationInMediaTimescale = Le(e))));
            let c = fe(e);
            if (c > 0) {
              a.languageCode = "";
              for (let l = 0; l < 3; l++)
                ((a.languageCode =
                  String.fromCharCode(96 + (c & 31)) + a.languageCode),
                  (c >>= 5));
              Ni(a.languageCode) || (a.languageCode = Pe);
            }
          }
          break;
        case "hdlr":
          {
            const a = this.currentTrack;
            if (!a) break;
            e.skip(8);
            const o = le(e, 4);
            o === "vide"
              ? (a.info = {
                  type: "video",
                  width: -1,
                  height: -1,
                  squarePixelWidth: -1,
                  squarePixelHeight: -1,
                  codec: null,
                  codecDescription: null,
                  colorSpace: { ...Qs },
                  avcType: null,
                  avcCodecInfo: null,
                  hevcCodecInfo: null,
                  vp9CodecInfo: null,
                  av1CodecInfo: null,
                  proresCodecInfo: null,
                  proresFormat: null,
                })
              : o === "soun" &&
                (a.info = {
                  type: "audio",
                  numberOfChannels: -1,
                  sampleRate: -1,
                  codec: null,
                  codecDescription: null,
                  aacCodecInfo: null,
                  dtsFormat: null,
                  pcmLittleEndian: !1,
                  pcmSampleSize: null,
                });
          }
          break;
        case "stbl":
          {
            const a = this.currentTrack;
            if (!a) break;
            ((a.sampleTableByteOffset = t),
              this.readContiguousBoxes(e.slice(n, r.contentSize)));
          }
          break;
        case "stsd":
          {
            const a = this.currentTrack;
            if (!a || a.info === null || a.sampleTable) break;
            const o = U(e);
            e.skip(3);
            const c = R(e);
            for (let l = 0; l < c; l++) {
              const u = e.filePos,
                d = Rt(e);
              if (!d) break;
              a.internalCodecId = d.name;
              const f = d.name.toLowerCase();
              if (a.info.type === "video") {
                (e.skip(24),
                  (a.info.width = fe(e)),
                  (a.info.height = fe(e)),
                  (a.info.squarePixelWidth = a.info.width),
                  (a.info.squarePixelHeight = a.info.height),
                  e.skip(50),
                  (a.frmaCodecString = null),
                  this.readContiguousBoxes(
                    e.slice(e.filePos, u + d.totalSize - e.filePos),
                  ));
                const h = f === "encv" ? a.frmaCodecString : f;
                ((a.frmaCodecString = null),
                  h === "avc1" || h === "avc3"
                    ? ((a.info.codec = "avc"),
                      (a.info.avcType = h === "avc1" ? 1 : 3))
                    : h === "hvc1" || h === "hev1"
                      ? (a.info.codec = "hevc")
                      : h === "vp08"
                        ? (a.info.codec = "vp8")
                        : h === "vp09"
                          ? (a.info.codec = "vp9")
                          : h === "av01"
                            ? (a.info.codec = "av1")
                            : Kt.includes(f)
                              ? ((a.info.codec = "prores"),
                                (a.info.proresFormat = f))
                              : h === null
                                ? L._warn(
                                    "Unknown encrypted video codec due to missing frma box.",
                                  )
                                : L._warn(
                                    `Unsupported video codec (sample entry type '${d.name}').`,
                                  ));
              } else {
                e.skip(8);
                const h = fe(e);
                e.skip(6);
                let p = fe(e),
                  m = fe(e);
                e.skip(4);
                let y = R(e) / 65536,
                  w = null;
                (o === 0 &&
                  h > 0 &&
                  (h === 1
                    ? (e.skip(4), (m = 8 * R(e)), e.skip(8))
                    : h === 2 &&
                      (e.skip(4),
                      (y = Bc(e)),
                      (p = R(e)),
                      e.skip(4),
                      (m = R(e)),
                      (w = R(e)),
                      e.skip(8))),
                  (a.info.numberOfChannels = p),
                  (a.info.sampleRate = y),
                  (a.frmaCodecString = null),
                  this.readContiguousBoxes(
                    e.slice(e.filePos, u + d.totalSize - e.filePos),
                  ));
                const b = f === "enca" ? a.frmaCodecString : f;
                if (((a.frmaCodecString = null), b !== "mp4a"))
                  if (b === "opus")
                    ((a.info.codec = "opus"), (a.info.sampleRate = Gi));
                  else if (b === "flac") a.info.codec = "flac";
                  else if (b === "ulaw") a.info.codec = "ulaw";
                  else if (b === "alaw") a.info.codec = "alaw";
                  else if (b === "ac-3") a.info.codec = "ac3";
                  else if (b === "ec-3") a.info.codec = "eac3";
                  else if (Mr.includes(b))
                    ((a.info.codec = "dts"), (a.info.dtsFormat = b));
                  else if (b === "twos")
                    m === 8
                      ? (a.info.codec = "pcm-s8")
                      : m === 16
                        ? (a.info.codec = a.info.pcmLittleEndian
                            ? "pcm-s16"
                            : "pcm-s16be")
                        : (L._warn(
                            `Unsupported sample size ${m} for codec 'twos'.`,
                          ),
                          (a.info.codec = null));
                  else if (b === "sowt")
                    m === 8
                      ? (a.info.codec = "pcm-s8")
                      : m === 16
                        ? (a.info.codec = "pcm-s16")
                        : (L._warn(
                            `Unsupported sample size ${m} for codec 'sowt'.`,
                          ),
                          (a.info.codec = null));
                  else if (b === "raw ") a.info.codec = "pcm-u8";
                  else if (b === "in24")
                    a.info.codec = a.info.pcmLittleEndian
                      ? "pcm-s24"
                      : "pcm-s24be";
                  else if (b === "in32")
                    a.info.codec = a.info.pcmLittleEndian
                      ? "pcm-s32"
                      : "pcm-s32be";
                  else if (b === "fl32")
                    a.info.codec = a.info.pcmLittleEndian
                      ? "pcm-f32"
                      : "pcm-f32be";
                  else if (b === "fl64")
                    a.info.codec = a.info.pcmLittleEndian
                      ? "pcm-f64"
                      : "pcm-f64be";
                  else if (b === "ipcm") {
                    const T = a.info.pcmSampleSize;
                    a.info.pcmLittleEndian
                      ? T === 16
                        ? (a.info.codec = "pcm-s16")
                        : T === 24
                          ? (a.info.codec = "pcm-s24")
                          : T === 32
                            ? (a.info.codec = "pcm-s32")
                            : (L._warn(`Invalid ipcm sample size ${T}.`),
                              (a.info.codec = null))
                      : T === 16
                        ? (a.info.codec = "pcm-s16be")
                        : T === 24
                          ? (a.info.codec = "pcm-s24be")
                          : T === 32
                            ? (a.info.codec = "pcm-s32be")
                            : (L._warn(`Invalid ipcm sample size ${T}.`),
                              (a.info.codec = null));
                  } else if (b === "fpcm") {
                    const T = a.info.pcmSampleSize;
                    a.info.pcmLittleEndian
                      ? T === 32
                        ? (a.info.codec = "pcm-f32")
                        : T === 64
                          ? (a.info.codec = "pcm-f64")
                          : (L._warn(`Invalid fpcm sample size ${T}.`),
                            (a.info.codec = null))
                      : T === 32
                        ? (a.info.codec = "pcm-f32be")
                        : T === 64
                          ? (a.info.codec = "pcm-f64be")
                          : (L._warn(`Invalid fpcm sample size ${T}.`),
                            (a.info.codec = null));
                  } else if (b === "lpcm" && w !== null) {
                    const T = (m + 7) >> 3,
                      S = !!(w & 1),
                      k = !!(w & 2),
                      P = w & 4 ? -1 : 0;
                    (m > 0 &&
                      m <= 64 &&
                      (S
                        ? m === 32 &&
                          (a.info.codec = k ? "pcm-f32be" : "pcm-f32")
                        : P & (1 << (T - 1))
                          ? T === 1
                            ? (a.info.codec = "pcm-s8")
                            : T === 2
                              ? (a.info.codec = k ? "pcm-s16be" : "pcm-s16")
                              : T === 3
                                ? (a.info.codec = k ? "pcm-s24be" : "pcm-s24")
                                : T === 4 &&
                                  (a.info.codec = k ? "pcm-s32be" : "pcm-s32")
                          : T === 1 && (a.info.codec = "pcm-u8")),
                      a.info.codec === null &&
                        L._warn("Unsupported PCM format."));
                  } else
                    b === null
                      ? L._warn(
                          "Unknown encrypted audio codec due to missing frma box.",
                        )
                      : L._warn(
                          `Unsupported audio codec (sample entry type '${d.name}').`,
                        );
              }
              e.filePos = u + d.totalSize;
            }
          }
          break;
        case "frma":
          {
            const a = this.currentTrack;
            if (!a) break;
            const c = le(e, 4).toLowerCase();
            a.frmaCodecString = c;
          }
          break;
        case "schm":
          {
            const a = this.currentTrack;
            if (!a) break;
            e.skip(4);
            const o = le(e, 4);
            o === "cenc" || o === "cens" || o === "cbcs"
              ? (a.encryptionInfo = {
                  scheme: o,
                  defaultKid: null,
                  defaultIsProtected: null,
                  defaultPerSampleIvSize: null,
                  defaultConstantIv: null,
                  defaultCryptByteBlock: null,
                  defaultSkipByteBlock: null,
                })
              : L._warn(`Unsupported encryption scheme '${o}'.`);
          }
          break;
        case "tenc":
          {
            const a = this.currentTrack;
            if (!a || !a.encryptionInfo) break;
            const o = U(e);
            (e.skip(3), e.skip(1));
            const c = U(e);
            if (
              (o > 0
                ? ((a.encryptionInfo.defaultCryptByteBlock = c >> 4),
                  (a.encryptionInfo.defaultSkipByteBlock = c & 15))
                : ((a.encryptionInfo.defaultCryptByteBlock = 0),
                  (a.encryptionInfo.defaultSkipByteBlock = 0)),
              (a.encryptionInfo.defaultIsProtected = U(e) !== 0),
              (a.encryptionInfo.defaultPerSampleIvSize = U(e)),
              (a.encryptionInfo.defaultKid = Di(N(e, 16))),
              a.encryptionInfo.defaultIsProtected &&
                a.encryptionInfo.defaultPerSampleIvSize === 0)
            ) {
              const l = U(e),
                u = new Uint8Array(16);
              (u.set(N(e, l), 0), (a.encryptionInfo.defaultConstantIv = u));
            }
          }
          break;
        case "avcC":
          {
            const a = this.currentTrack;
            if (!a || (g(a.info), r.contentSize === 0)) break;
            a.info.codecDescription = N(e, r.contentSize);
            const o = Mn(a.info.codecDescription)?.sequenceParameterSets[0],
              c = o && pi(o);
            c && Ja(a, c.pixelAspectRatio);
          }
          break;
        case "hvcC":
          {
            const a = this.currentTrack;
            if (!a || (g(a.info), r.contentSize === 0)) break;
            a.info.codecDescription = N(e, r.contentSize);
            const o = ba(a.info.codecDescription)?.arrays.find(
                (l) => l.nalUnitType === de.SPS_NUT,
              )?.nalUnits[0],
              c = o && Er(o);
            c && Ja(a, c.pixelAspectRatio);
          }
          break;
        case "vpcC":
          {
            const a = this.currentTrack;
            if (!a) break;
            (g(a.info?.type === "video"), e.skip(4));
            const o = U(e),
              c = U(e),
              l = U(e),
              u = l >> 4,
              d = (l >> 1) & 7,
              f = l & 1,
              h = U(e),
              p = U(e),
              m = U(e);
            a.info.vp9CodecInfo = {
              profile: o,
              level: c,
              bitDepth: u,
              chromaSubsampling: d,
              videoFullRangeFlag: f,
              colourPrimaries: h,
              transferCharacteristics: p,
              matrixCoefficients: m,
            };
          }
          break;
        case "av1C":
          {
            const a = this.currentTrack;
            if (!a) break;
            (g(a.info?.type === "video"), e.skip(1));
            const o = U(e),
              c = o >> 5,
              l = o & 31,
              u = U(e),
              d = u >> 7,
              f = (u >> 6) & 1,
              h = (u >> 5) & 1,
              p = (u >> 4) & 1,
              m = (u >> 3) & 1,
              y = (u >> 2) & 1,
              w = u & 3,
              b = c === 2 && f ? (h ? 12 : 10) : f ? 10 : 8;
            e.skip(1);
            const T = N(e, r.contentSize - 4),
              S = _r(T);
            a.info.av1CodecInfo = {
              profile: c,
              level: l,
              tier: d,
              bitDepth: b,
              monochrome: p,
              chromaSubsamplingX: m,
              chromaSubsamplingY: y,
              chromaSamplePosition: w,
              videoFullRangeFlag: S?.videoFullRangeFlag ?? 0,
              colourPrimaries: S?.colourPrimaries ?? 2,
              transferCharacteristics: S?.transferCharacteristics ?? 2,
              matrixCoefficients: S?.matrixCoefficients ?? 2,
            };
          }
          break;
        case "colr":
          {
            const a = this.currentTrack;
            if (!a) break;
            g(a.info?.type === "video");
            const o = le(e, 4);
            if (o !== "nclx" && o !== "nclc") break;
            const c = fe(e),
              l = fe(e),
              u = fe(e);
            let d;
            (o === "nclx" && (d = !!(U(e) & 128)),
              (a.info.colorSpace = {
                primaries: ht[c],
                transfer: mt[l],
                matrix: pt[u],
                fullRange: d,
              }));
          }
          break;
        case "pasp":
          {
            const a = this.currentTrack;
            if (!a) break;
            g(a.info?.type === "video");
            const o = R(e),
              c = R(e);
            o > 0 &&
              c > 0 &&
              ((a.info.squarePixelWidth = a.info.width),
              (a.info.squarePixelHeight = a.info.height),
              o > c
                ? (a.info.squarePixelWidth = Math.round((a.info.width * o) / c))
                : (a.info.squarePixelHeight = Math.round(
                    (a.info.height * c) / o,
                  )));
          }
          break;
        case "btrt":
          {
            const a = this.currentTrack;
            if (!a) break;
            e.skip(4);
            const o = R(e),
              c = R(e);
            ((a.maxBitrate = o > 0 ? o : null),
              (a.avgBitrate = c > 0 ? c : null));
          }
          break;
        case "wave":
          this.readContiguousBoxes(e.slice(n, r.contentSize));
          break;
        case "esds":
          {
            const a = this.currentTrack;
            if (!a || a.info?.type !== "audio") break;
            e.skip(4);
            const o = U(e);
            (g(o === 3), Kn(e), e.skip(2));
            const c = U(e),
              l = (c & 128) !== 0,
              u = (c & 64) !== 0,
              d = (c & 32) !== 0;
            if ((l && e.skip(2), u)) {
              const y = U(e);
              e.skip(y);
            }
            d && e.skip(2);
            const f = U(e);
            g(f === 4);
            const h = Kn(e),
              p = e.filePos,
              m = U(e);
            if (
              (m === 64 || m === 103
                ? ((a.info.codec = "aac"),
                  (a.info.aacCodecInfo = {
                    isMpeg2: m === 103,
                    objectType: null,
                  }))
                : m === 105 || m === 107
                  ? (a.info.codec = "mp3")
                  : m === 221
                    ? (a.info.codec = "vorbis")
                    : m === 169
                      ? (a.info.codec = "dts")
                      : L._warn(
                          `Unsupported audio codec (objectTypeIndication ${m}) - discarding track.`,
                        ),
              e.skip(12),
              h > e.filePos - p)
            ) {
              const y = U(e);
              g(y === 5);
              const w = Kn(e);
              if (
                ((a.info.codecDescription = N(e, w)), a.info.codec === "aac")
              ) {
                const b = En(a.info.codecDescription);
                (b.outputNumberOfChannels !== null &&
                  (a.info.numberOfChannels = b.outputNumberOfChannels),
                  b.outputSampleRate !== null &&
                    (a.info.sampleRate = b.outputSampleRate));
              }
            }
          }
          break;
        case "enda":
          {
            const a = this.currentTrack;
            if (!a) break;
            (g(a.info?.type === "audio"),
              (a.info.pcmLittleEndian = !!(fe(e) & 255)));
          }
          break;
        case "pcmC":
          {
            const a = this.currentTrack;
            if (!a) break;
            (g(a.info?.type === "audio"), e.skip(4));
            const o = U(e);
            ((a.info.pcmLittleEndian = !!(o & 1)),
              (a.info.pcmSampleSize = U(e)));
          }
          break;
        case "dOps":
          {
            const a = this.currentTrack;
            if (!a) break;
            (g(a.info?.type === "audio"), e.skip(1));
            const o = U(e),
              c = fe(e),
              l = R(e),
              u = Ts(e),
              d = U(e);
            let f;
            d !== 0 ? (f = N(e, 2 + o)) : (f = new Uint8Array(0));
            const h = new Uint8Array(19 + f.byteLength),
              p = new DataView(h.buffer);
            (p.setUint32(0, 1332770163, !1),
              p.setUint32(4, 1214603620, !1),
              p.setUint8(8, 1),
              p.setUint8(9, o),
              p.setUint16(10, c, !0),
              p.setUint32(12, l, !0),
              p.setInt16(16, u, !0),
              p.setUint8(18, d),
              h.set(f, 19),
              (a.info.codecDescription = h),
              (a.info.numberOfChannels = o));
          }
          break;
        case "dfLa":
          {
            const a = this.currentTrack;
            if (!a) break;
            (g(a.info?.type === "audio"), e.skip(4));
            const o = 127,
              c = 128,
              l = e.filePos;
            for (; e.filePos < s;) {
              const p = U(e),
                m = It(e);
              if ((p & o) === wi.STREAMINFO) {
                e.skip(10);
                const w = R(e),
                  b = w >>> 12,
                  T = ((w >> 9) & 7) + 1;
                ((a.info.sampleRate = b),
                  (a.info.numberOfChannels = T),
                  e.skip(20));
              } else e.skip(m);
              if (p & c) break;
            }
            const u = e.filePos;
            e.filePos = l;
            const d = N(e, u - l),
              f = new Uint8Array(4 + d.byteLength);
            (new DataView(f.buffer).setUint32(0, 1716281667, !1),
              f.set(d, 4),
              (a.info.codecDescription = f));
          }
          break;
        case "dac3":
          {
            const a = this.currentTrack;
            if (!a) break;
            g(a.info?.type === "audio");
            const o = N(e, 3),
              c = new $(o),
              l = c.readBits(2);
            c.skipBits(8);
            const u = c.readBits(3),
              d = c.readBits(1);
            (l < 3 && (a.info.sampleRate = Pr[l]),
              (a.info.numberOfChannels = Nn[u] + d));
          }
          break;
        case "dec3":
          {
            const a = this.currentTrack;
            if (!a) break;
            g(a.info?.type === "audio");
            const o = N(e, r.contentSize),
              c = Wl(o);
            if (!c) {
              L._warn("Invalid dec3 box contents, ignoring.");
              break;
            }
            const l = Ia(c);
            (l !== null && (a.info.sampleRate = l),
              (a.info.numberOfChannels = Ea(c)));
          }
          break;
        case "ddts":
          {
            const a = this.currentTrack;
            if (!a) break;
            g(a.info?.type === "audio");
            const o = N(e, Math.min(r.contentSize, Wn)),
              c = Zl(o);
            if (!c) {
              L._warn("Invalid ddts box contents, ignoring.");
              break;
            }
            ((a.info.sampleRate = c.sampleRate),
              c.numberOfChannels !== null &&
                (a.info.numberOfChannels = c.numberOfChannels));
          }
          break;
        case "stts":
          {
            const a = this.currentTrack;
            if (!a || !a.sampleTable) break;
            e.skip(4);
            const o = R(e);
            let c = 0,
              l = 0;
            for (let u = 0; u < o; u++) {
              const d = R(e),
                f = R(e);
              (a.sampleTable.sampleTimingEntries.push({
                startIndex: c,
                startDecodeTimestamp: l,
                count: d,
                delta: f,
              }),
                (c += d),
                (l += d * f));
            }
          }
          break;
        case "ctts":
          {
            const a = this.currentTrack;
            if (!a || !a.sampleTable) break;
            e.skip(4);
            const o = R(e);
            let c = 0;
            for (let l = 0; l < o; l++) {
              const u = R(e),
                d = ni(e);
              (a.sampleTable.sampleCompositionTimeOffsets.push({
                startIndex: c,
                count: u,
                offset: d,
              }),
                (c += u));
            }
          }
          break;
        case "stsz":
          {
            const a = this.currentTrack;
            if (!a || !a.sampleTable) break;
            e.skip(4);
            const o = R(e),
              c = R(e);
            if (o === 0)
              for (let l = 0; l < c; l++) {
                const u = R(e);
                a.sampleTable.sampleSizes.push(u);
              }
            else a.sampleTable.sampleSizes.push(o);
          }
          break;
        case "stz2":
          {
            const a = this.currentTrack;
            if (!a || !a.sampleTable) break;
            (e.skip(4), e.skip(3));
            const o = U(e),
              c = R(e),
              l = N(e, Math.ceil((c * o) / 8)),
              u = new $(l);
            for (let d = 0; d < c; d++) {
              const f = u.readBits(o);
              a.sampleTable.sampleSizes.push(f);
            }
          }
          break;
        case "stss":
          {
            const a = this.currentTrack;
            if (!a || !a.sampleTable) break;
            (e.skip(4), (a.sampleTable.keySampleIndices = []));
            const o = R(e);
            for (let c = 0; c < o; c++) {
              const l = R(e) - 1;
              a.sampleTable.keySampleIndices.push(l);
            }
            a.sampleTable.keySampleIndices[0] !== 0 &&
              a.sampleTable.keySampleIndices.unshift(0);
          }
          break;
        case "stsc":
          {
            const a = this.currentTrack;
            if (!a || !a.sampleTable) break;
            e.skip(4);
            const o = R(e);
            for (let l = 0; l < o; l++) {
              const u = R(e) - 1,
                d = R(e),
                f = R(e);
              a.sampleTable.sampleToChunk.push({
                startSampleIndex: -1,
                startChunkIndex: u,
                samplesPerChunk: d,
                sampleDescriptionIndex: f,
              });
            }
            let c = 0;
            for (let l = 0; l < a.sampleTable.sampleToChunk.length; l++)
              if (
                ((a.sampleTable.sampleToChunk[l].startSampleIndex = c),
                l < a.sampleTable.sampleToChunk.length - 1)
              ) {
                const d =
                  a.sampleTable.sampleToChunk[l + 1].startChunkIndex -
                  a.sampleTable.sampleToChunk[l].startChunkIndex;
                c += d * a.sampleTable.sampleToChunk[l].samplesPerChunk;
              }
          }
          break;
        case "stco":
          {
            const a = this.currentTrack;
            if (!a || !a.sampleTable) break;
            e.skip(4);
            const o = R(e);
            for (let c = 0; c < o; c++) {
              const l = R(e);
              a.sampleTable.chunkOffsets.push(l);
            }
          }
          break;
        case "co64":
          {
            const a = this.currentTrack;
            if (!a || !a.sampleTable) break;
            e.skip(4);
            const o = R(e);
            for (let c = 0; c < o; c++) {
              const l = Le(e);
              a.sampleTable.chunkOffsets.push(l);
            }
          }
          break;
        case "mvex":
          ((this.isFragmented = !0),
            this.readContiguousBoxes(e.slice(n, r.contentSize)));
          break;
        case "mehd":
          {
            const a = U(e);
            e.skip(3);
            const o = a === 1 ? Le(e) : R(e);
            this.movieDurationInTimescale = o;
          }
          break;
        case "trex":
          {
            e.skip(4);
            const a = R(e),
              o = R(e),
              c = R(e),
              l = R(e),
              u = R(e);
            this.fragmentTrackDefaults.push({
              trackId: a,
              defaultSampleDescriptionIndex: o,
              defaultSampleDuration: c,
              defaultSampleSize: l,
              defaultSampleFlags: u,
            });
          }
          break;
        case "tfra":
          {
            const a = U(e);
            e.skip(3);
            const o = R(e),
              c = this.tracks.find((b) => b.id === o);
            if (!c) break;
            const l = R(e),
              u = (l & 48) >> 4,
              d = (l & 12) >> 2,
              f = l & 3,
              h = [U, fe, It, R],
              p = h[u],
              m = h[d],
              y = h[f],
              w = R(e);
            for (let b = 0; b < w; b++) {
              const T = a === 1 ? Le(e) : R(e),
                S = a === 1 ? Le(e) : R(e);
              (p(e),
                m(e),
                y(e),
                c.fragmentLookupTable.push({ timestamp: T, moofOffset: S }));
            }
            c.fragmentLookupTable.sort((b, T) => b.timestamp - T.timestamp);
            for (let b = 0; b < c.fragmentLookupTable.length - 1; b++) {
              const T = c.fragmentLookupTable[b],
                S = c.fragmentLookupTable[b + 1];
              T.timestamp === S.timestamp &&
                (c.fragmentLookupTable.splice(b + 1, 1), b--);
            }
          }
          break;
        case "moof":
          ((this.currentFragment = {
            moofOffset: t,
            moofSize: r.totalSize,
            implicitBaseDataOffset: t,
            trackData: new Map(),
            psshBoxes: [],
          }),
            this.readContiguousBoxes(e.slice(n, r.contentSize)),
            (this.lastReadFragment = this.currentFragment),
            (this.currentFragment = null));
          break;
        case "traf":
          if (
            (g(this.currentFragment),
            this.readContiguousBoxes(e.slice(n, r.contentSize)),
            this.currentTrack)
          ) {
            const a = this.currentFragment.trackData.get(this.currentTrack.id);
            e: if (a) {
              if (a.samples.length === 0) {
                this.currentFragment.trackData.delete(this.currentTrack.id);
                break e;
              }
              a.presentationTimestamps = a.samples
                .map((u, d) => ({
                  presentationTimestamp: u.presentationTimestamp,
                  sampleIndex: d,
                }))
                .sort(
                  (u, d) => u.presentationTimestamp - d.presentationTimestamp,
                );
              for (let u = 0; u < a.presentationTimestamps.length; u++) {
                const d = a.presentationTimestamps[u],
                  f = a.samples[d.sampleIndex];
                if (
                  (a.firstKeyFrameTimestamp === null &&
                    f.isKeyFrame &&
                    (a.firstKeyFrameTimestamp = f.presentationTimestamp),
                  u < a.presentationTimestamps.length - 1)
                ) {
                  const p =
                    a.presentationTimestamps[u + 1].presentationTimestamp -
                    d.presentationTimestamp;
                  f.duration = p;
                }
              }
              const o = a.samples[a.presentationTimestamps[0].sampleIndex],
                c = a.samples[ne(a.presentationTimestamps).sampleIndex];
              ((a.startTimestamp = o.presentationTimestamp),
                (a.endTimestamp = c.presentationTimestamp + c.duration));
              const { currentFragmentState: l } = this.currentTrack;
              (g(l),
                l.startTimestamp !== null &&
                  (Qn(a, l.startTimestamp), (a.startTimestampIsFinal = !0)),
                l.encryptionAuxInfo &&
                  !a.samples[0].encryption &&
                  (a.encryptionAuxInfo = l.encryptionAuxInfo));
            }
            ((this.currentTrack.currentFragmentState = null),
              (this.currentTrack = null));
          }
          break;
        case "pssh":
          {
            if (this.input._formatOptions.isobmff?._suppressPsshParsing) break;
            const a = qa(N(e, r.contentSize));
            this.currentFragment
              ? this.currentFragment.psshBoxes.push(a)
              : this.currentTrack || this.psshBoxes.push(a);
          }
          break;
        case "tfhd":
          {
            (g(this.currentFragment), e.skip(1));
            const a = It(e),
              o = !!(a & 1),
              c = !!(a & 2),
              l = !!(a & 8),
              u = !!(a & 16),
              d = !!(a & 32),
              f = !!(a & 65536),
              h = !!(a & 131072),
              p = R(e),
              m = this.tracks.find((w) => w.id === p);
            if (!m) break;
            const y = this.fragmentTrackDefaults.find((w) => w.trackId === p);
            ((this.currentTrack = m),
              (m.currentFragmentState = {
                baseDataOffset: this.currentFragment.implicitBaseDataOffset,
                sampleDescriptionIndex:
                  y?.defaultSampleDescriptionIndex ?? null,
                defaultSampleDuration: y?.defaultSampleDuration ?? null,
                defaultSampleSize: y?.defaultSampleSize ?? null,
                defaultSampleFlags: y?.defaultSampleFlags ?? null,
                startTimestamp: null,
                encryptionAuxInfo: null,
              }),
              o
                ? (m.currentFragmentState.baseDataOffset = Le(e))
                : h &&
                  (m.currentFragmentState.baseDataOffset =
                    this.currentFragment.moofOffset),
              c && (m.currentFragmentState.sampleDescriptionIndex = R(e)),
              l && (m.currentFragmentState.defaultSampleDuration = R(e)),
              u && (m.currentFragmentState.defaultSampleSize = R(e)),
              d && (m.currentFragmentState.defaultSampleFlags = R(e)),
              f && (m.currentFragmentState.defaultSampleDuration = 0));
          }
          break;
        case "tfdt":
          {
            const a = this.currentTrack;
            if (!a) break;
            g(a.currentFragmentState);
            const o = U(e);
            e.skip(3);
            const c = o === 0 ? R(e) : Le(e);
            a.currentFragmentState.startTimestamp = c;
          }
          break;
        case "trun":
          {
            const a = this.currentTrack;
            if (!a) break;
            (g(this.currentFragment), g(a.currentFragmentState), e.skip(1));
            const o = It(e),
              c = !!(o & 1),
              l = !!(o & 4),
              u = !!(o & 256),
              d = !!(o & 512),
              f = !!(o & 1024),
              h = !!(o & 2048),
              p = R(e);
            let m = null;
            c && (m = ni(e));
            let y = null;
            l && (y = R(e));
            let w;
            this.currentFragment.trackData.has(a.id)
              ? ((w = this.currentFragment.trackData.get(a.id)),
                m !== null &&
                  (w.currentOffset = a.currentFragmentState.baseDataOffset + m))
              : ((w = {
                  track: a,
                  currentTimestamp: 0,
                  currentOffset:
                    a.currentFragmentState.baseDataOffset + (m ?? 0),
                  startTimestamp: 0,
                  endTimestamp: 0,
                  firstKeyFrameTimestamp: null,
                  samples: [],
                  presentationTimestamps: [],
                  startTimestampIsFinal: !1,
                  encryptionAuxInfo: null,
                }),
                this.currentFragment.trackData.set(a.id, w));
            for (let b = 0; b < p; b++) {
              let T;
              u
                ? (T = R(e))
                : (g(a.currentFragmentState.defaultSampleDuration !== null),
                  (T = a.currentFragmentState.defaultSampleDuration));
              let S;
              d
                ? (S = R(e))
                : (g(a.currentFragmentState.defaultSampleSize !== null),
                  (S = a.currentFragmentState.defaultSampleSize));
              let k;
              (f
                ? (k = R(e))
                : (g(a.currentFragmentState.defaultSampleFlags !== null),
                  (k = a.currentFragmentState.defaultSampleFlags)),
                b === 0 && y !== null && (k = y));
              let P = 0;
              h && (P = ni(e));
              const I = !(k & 65536);
              (w.samples.push({
                presentationTimestamp: w.currentTimestamp + P,
                duration: T,
                byteOffset: w.currentOffset,
                byteSize: S,
                isKeyFrame: I,
                encryption: null,
              }),
                (w.currentOffset += S),
                (w.currentTimestamp += T));
            }
            this.currentFragment.implicitBaseDataOffset = w.currentOffset;
          }
          break;
        case "saiz":
          {
            const a = this.currentTrack;
            if (!a || !a.encryptionInfo) break;
            if ((e.skip(1), It(e) & 1)) {
              const f = le(e, 4),
                h = R(e);
              if (f !== a.encryptionInfo.scheme || h !== 0) break;
            }
            const c = U(e),
              l = R(e);
            let u = null;
            c === 0 && l > 0 && (u = N(e, l));
            const d = to(a);
            ((d.defaultSampleInfoSize = c),
              (d.sampleSizes = u),
              (d.sampleCount = l));
          }
          break;
        case "saio":
          {
            const a = this.currentTrack;
            if (!a || !a.encryptionInfo) break;
            const o = U(e);
            if (It(e) & 1) {
              const f = le(e, 4),
                h = R(e);
              if (f !== a.encryptionInfo.scheme || h !== 0) break;
            }
            const l = R(e);
            if (l === 0) break;
            l > 1 &&
              L._warn(
                "Multiple saio entries are not supported; using the first offset only.",
              );
            let u = o === 0 ? R(e) : Number(Le(e));
            this.currentFragment && (u += this.currentFragment.moofOffset);
            const d = to(a);
            d.offset = u;
          }
          break;
        case "senc":
          {
            const a = this.currentTrack;
            if (!a || !a.encryptionInfo) break;
            g(this.currentFragment);
            const o = this.currentFragment.trackData.get(a.id);
            if (!o) break;
            e.skip(1);
            const l = !!(It(e) & 2),
              u = R(e),
              d = a.encryptionInfo.defaultPerSampleIvSize;
            g(d !== null);
            for (let f = 0; f < Math.min(u, o.samples.length); f++) {
              const h = new Uint8Array(16);
              d > 0
                ? h.set(N(e, d), 0)
                : h.set(a.encryptionInfo.defaultConstantIv, 0);
              let p = null;
              if (l) {
                const y = fe(e);
                p = [];
                for (let w = 0; w < y; w++) {
                  const b = fe(e),
                    T = R(e);
                  p.push({ clearLen: b, protectedLen: T });
                }
              }
              const m = o.samples[f];
              m.encryption = { iv: h, subsamples: p };
            }
          }
          break;
        case "udta":
          {
            const a = this.iterateContiguousBoxes(e.slice(n, r.contentSize));
            for (const { boxInfo: o, slice: c } of a) {
              if (o.name !== "meta" && !this.currentTrack) {
                const l = c.filePos;
                ((this.metadataTags.raw ??= {}),
                  o.name[0] === "©"
                    ? (this.metadataTags.raw[o.name] ??= et(c))
                    : (this.metadataTags.raw[o.name] ??= N(c, o.contentSize)),
                  (c.filePos = l));
              }
              switch (o.name) {
                case "meta":
                  (c.skip(-o.headerSize), this.traverseBox(c));
                  break;
                case "©nam":
                case "name":
                  this.currentTrack
                    ? (this.currentTrack.name = Oe.decode(N(c, o.contentSize)))
                    : (this.metadataTags.title ??= et(c));
                  break;
                case "©des":
                  this.currentTrack ||
                    (this.metadataTags.description ??= et(c));
                  break;
                case "©ART":
                  this.currentTrack || (this.metadataTags.artist ??= et(c));
                  break;
                case "©alb":
                  this.currentTrack || (this.metadataTags.album ??= et(c));
                  break;
                case "albr":
                  this.currentTrack ||
                    (this.metadataTags.albumArtist ??= et(c));
                  break;
                case "©gen":
                  this.currentTrack || (this.metadataTags.genre ??= et(c));
                  break;
                case "©day":
                  if (!this.currentTrack) {
                    const l = new Date(et(c));
                    Number.isNaN(l.getTime()) || (this.metadataTags.date ??= l);
                  }
                  break;
                case "©cmt":
                  this.currentTrack || (this.metadataTags.comment ??= et(c));
                  break;
                case "©lyr":
                  this.currentTrack || (this.metadataTags.lyrics ??= et(c));
                  break;
              }
            }
          }
          break;
        case "meta":
          {
            if (this.currentTrack) break;
            const o = R(e) !== 0;
            ((this.currentMetadataKeys = new Map()),
              o
                ? this.readContiguousBoxes(e.slice(n, r.contentSize))
                : this.readContiguousBoxes(e.slice(n + 4, r.contentSize - 4)),
              (this.currentMetadataKeys = null));
          }
          break;
        case "keys":
          {
            if (!this.currentMetadataKeys) break;
            e.skip(4);
            const a = R(e);
            for (let o = 0; o < a; o++) {
              const c = R(e);
              e.skip(4);
              const l = Oe.decode(N(e, c - 8));
              this.currentMetadataKeys.set(o + 1, l);
            }
          }
          break;
        case "ilst":
          {
            if (!this.currentMetadataKeys) break;
            const a = this.iterateContiguousBoxes(e.slice(n, r.contentSize));
            for (const { boxInfo: o, slice: c } of a) {
              let l = o.name;
              const u =
                (l.charCodeAt(0) << 24) +
                (l.charCodeAt(1) << 16) +
                (l.charCodeAt(2) << 8) +
                l.charCodeAt(3);
              this.currentMetadataKeys.has(u) &&
                (l = this.currentMetadataKeys.get(u));
              const d = yu(c);
              switch (
                ((this.metadataTags.raw ??= {}),
                (this.metadataTags.raw[l] ??= d),
                l)
              ) {
                case "©nam":
                case "titl":
                case "com.apple.quicktime.title":
                case "title":
                  typeof d == "string" && (this.metadataTags.title ??= d);
                  break;
                case "©des":
                case "desc":
                case "dscp":
                case "com.apple.quicktime.description":
                case "description":
                  typeof d == "string" && (this.metadataTags.description ??= d);
                  break;
                case "©ART":
                case "com.apple.quicktime.artist":
                case "artist":
                  typeof d == "string" && (this.metadataTags.artist ??= d);
                  break;
                case "©alb":
                case "albm":
                case "com.apple.quicktime.album":
                case "album":
                  typeof d == "string" && (this.metadataTags.album ??= d);
                  break;
                case "aART":
                case "album_artist":
                  typeof d == "string" && (this.metadataTags.albumArtist ??= d);
                  break;
                case "©cmt":
                case "com.apple.quicktime.comment":
                case "comment":
                  typeof d == "string" && (this.metadataTags.comment ??= d);
                  break;
                case "©gen":
                case "gnre":
                case "com.apple.quicktime.genre":
                case "genre":
                  typeof d == "string" && (this.metadataTags.genre ??= d);
                  break;
                case "©lyr":
                case "lyrics":
                  typeof d == "string" && (this.metadataTags.lyrics ??= d);
                  break;
                case "©day":
                case "rldt":
                case "com.apple.quicktime.creationdate":
                case "date":
                  if (typeof d == "string") {
                    const f = new Date(d);
                    Number.isNaN(f.getTime()) || (this.metadataTags.date ??= f);
                  }
                  break;
                case "tmpo":
                  if (d instanceof Uint8Array && d.length >= 2) {
                    const f = j(d).getInt16(0, !1);
                    f > 0 && (this.metadataTags.beatsPerMinute ??= f);
                  }
                  break;
                case "covr":
                case "com.apple.quicktime.artwork":
                  d instanceof hi
                    ? ((this.metadataTags.images ??= []),
                      this.metadataTags.images.push({
                        data: d.data,
                        kind: "coverFront",
                        mimeType: d.mimeType,
                      }))
                    : d instanceof Uint8Array &&
                      ((this.metadataTags.images ??= []),
                      this.metadataTags.images.push({
                        data: d,
                        kind: "coverFront",
                        mimeType: "image/*",
                      }));
                  break;
                case "track":
                  if (typeof d == "string") {
                    const f = d.split("/"),
                      h = Number.parseInt(f[0], 10),
                      p = f[1] && Number.parseInt(f[1], 10);
                    (Number.isInteger(h) &&
                      h > 0 &&
                      (this.metadataTags.trackNumber ??= h),
                      p &&
                        Number.isInteger(p) &&
                        p > 0 &&
                        (this.metadataTags.tracksTotal ??= p));
                  }
                  break;
                case "trkn":
                  if (d instanceof Uint8Array && d.length >= 6) {
                    const f = j(d),
                      h = f.getUint16(2, !1),
                      p = f.getUint16(4, !1);
                    (h > 0 && (this.metadataTags.trackNumber ??= h),
                      p > 0 && (this.metadataTags.tracksTotal ??= p));
                  }
                  break;
                case "disc":
                case "disk":
                  if (d instanceof Uint8Array && d.length >= 6) {
                    const f = j(d),
                      h = f.getUint16(2, !1),
                      p = f.getUint16(4, !1);
                    (h > 0 && (this.metadataTags.discNumber ??= h),
                      p > 0 && (this.metadataTags.discsTotal ??= p));
                  }
                  break;
              }
            }
          }
          break;
      }
      return ((e.filePos = s), !0);
    }
  }
  class Xa {
    constructor(e) {
      ((this.internalTrack = e),
        (this.packetToSampleIndex = new WeakMap()),
        (this.packetToFragmentLocation = new WeakMap()));
    }
    getId() {
      return this.internalTrack.id;
    }
    getNumber() {
      const e = this.internalTrack.demuxer,
        t = this.internalTrack.trackBacking.getType();
      let r = 0;
      for (const n of e.tracks)
        if ((n.trackBacking.getType() === t && r++, n === this.internalTrack))
          break;
      return r;
    }
    getCodec() {
      throw new Error("Not implemented on base class.");
    }
    getInternalCodecId() {
      return this.internalTrack.internalCodecId;
    }
    getName() {
      return this.internalTrack.name;
    }
    getLanguageCode() {
      return this.internalTrack.languageCode;
    }
    getTimeResolution() {
      return this.internalTrack.timescale;
    }
    isRelativeToUnixEpoch() {
      return !1;
    }
    getUnixTimeForTimestamp() {
      return null;
    }
    getDisposition() {
      return this.internalTrack.disposition;
    }
    getPairingMask() {
      return 1n;
    }
    getBitrate() {
      return this.internalTrack.maxBitrate;
    }
    getAverageBitrate() {
      return this.internalTrack.avgBitrate;
    }
    async getDurationFromMetadata() {
      const e = this.internalTrack;
      return e.durationInMediaTimescale <= 0
        ? null
        : (g(e.trackBacking),
          ((await e.trackBacking.getFirstPacket({ metadataOnly: !0 }))
            ?.timestamp ?? 0) +
            e.durationInMediaTimescale / e.timescale);
    }
    async getLiveRefreshInterval() {
      return null;
    }
    async getFirstPacket(e) {
      const t = await this.fetchPacketForSampleIndex(0, e);
      return t || !this.internalTrack.demuxer.isFragmented
        ? t
        : this.performFragmentedLookup(
            null,
            (r) =>
              r.trackData.get(this.internalTrack.id)
                ? { sampleIndex: 0, correctSampleFound: !0 }
                : { sampleIndex: -1, correctSampleFound: !1 },
            -1 / 0,
            1 / 0,
            e,
          );
    }
    mapTimestampIntoTimescale(e) {
      return (
        zi(e * this.internalTrack.timescale) + this.internalTrack.editListOffset
      );
    }
    async getPacket(e, t) {
      const r = this.mapTimestampIntoTimescale(e),
        n = this.internalTrack.demuxer.getSampleTableForTrack(
          this.internalTrack,
        ),
        s = Xn(n, r),
        a = await this.fetchPacketForSampleIndex(s, t);
      return !eo(n) || !this.internalTrack.demuxer.isFragmented
        ? a
        : this.performFragmentedLookup(
            null,
            (o) => {
              const c = o.trackData.get(this.internalTrack.id);
              if (!c) return { sampleIndex: -1, correctSampleFound: !1 };
              const l = X(
                  c.presentationTimestamps,
                  r,
                  (f) => f.presentationTimestamp,
                ),
                u = l !== -1 ? c.presentationTimestamps[l].sampleIndex : -1,
                d = l !== -1 && r < c.endTimestamp;
              return { sampleIndex: u, correctSampleFound: d };
            },
            r,
            r,
            t,
          );
    }
    async getNextPacket(e, t) {
      const r = this.packetToSampleIndex.get(e);
      if (r !== void 0) return this.fetchPacketForSampleIndex(r + 1, t);
      const n = this.packetToFragmentLocation.get(e);
      if (n === void 0)
        throw new Error("Packet was not created from this track.");
      return this.performFragmentedLookup(
        n.fragment,
        (s) => {
          if (s === n.fragment) {
            const a = s.trackData.get(this.internalTrack.id);
            if (n.sampleIndex + 1 < a.samples.length)
              return { sampleIndex: n.sampleIndex + 1, correctSampleFound: !0 };
          } else if (s.trackData.get(this.internalTrack.id))
            return { sampleIndex: 0, correctSampleFound: !0 };
          return { sampleIndex: -1, correctSampleFound: !1 };
        },
        -1 / 0,
        1 / 0,
        t,
      );
    }
    async getKeyPacket(e, t) {
      const r = this.mapTimestampIntoTimescale(e),
        n = this.internalTrack.demuxer.getSampleTableForTrack(
          this.internalTrack,
        ),
        s = Tu(n, r),
        a = await this.fetchPacketForSampleIndex(s, t);
      return !eo(n) || !this.internalTrack.demuxer.isFragmented
        ? a
        : this.performFragmentedLookup(
            null,
            (o) => {
              const c = o.trackData.get(this.internalTrack.id);
              if (!c) return { sampleIndex: -1, correctSampleFound: !1 };
              const l = hn(
                  c.presentationTimestamps,
                  (f) =>
                    c.samples[f.sampleIndex].isKeyFrame &&
                    f.presentationTimestamp <= r,
                ),
                u = l !== -1 ? c.presentationTimestamps[l].sampleIndex : -1,
                d = l !== -1 && r < c.endTimestamp;
              return { sampleIndex: u, correctSampleFound: d };
            },
            r,
            r,
            t,
          );
    }
    async getNextKeyPacket(e, t) {
      const r = this.packetToSampleIndex.get(e);
      if (r !== void 0) {
        const s = this.internalTrack.demuxer.getSampleTableForTrack(
            this.internalTrack,
          ),
          a = xu(s, r);
        return this.fetchPacketForSampleIndex(a, t);
      }
      const n = this.packetToFragmentLocation.get(e);
      if (n === void 0)
        throw new Error("Packet was not created from this track.");
      return this.performFragmentedLookup(
        n.fragment,
        (s) => {
          if (s === n.fragment) {
            const o = s.trackData
              .get(this.internalTrack.id)
              .samples.findIndex((c, l) => c.isKeyFrame && l > n.sampleIndex);
            if (o !== -1) return { sampleIndex: o, correctSampleFound: !0 };
          } else {
            const a = s.trackData.get(this.internalTrack.id);
            if (a && a.firstKeyFrameTimestamp !== null) {
              const o = a.samples.findIndex((c) => c.isKeyFrame);
              return (g(o !== -1), { sampleIndex: o, correctSampleFound: !0 });
            }
          }
          return { sampleIndex: -1, correctSampleFound: !1 };
        },
        -1 / 0,
        1 / 0,
        t,
      );
    }
    async fetchPacketForSampleIndex(e, t) {
      if (e === -1) return null;
      const r = this.internalTrack.demuxer.getSampleTableForTrack(
          this.internalTrack,
        ),
        n = Su(r, e);
      if (!n) return null;
      let s;
      if (t.metadataOnly) s = Ue;
      else {
        let l = this.internalTrack.demuxer.reader.requestSlice(
          n.sampleOffset,
          n.sampleSize,
        );
        if ((A(l) && (l = await l), !l)) return null;
        if (((s = N(l, n.sampleSize)), this.internalTrack.encryptionInfo)) {
          let u = null;
          if (this.internalTrack.encryptionAuxInfo) {
            const d = await io(
              this.internalTrack.demuxer.reader,
              this.internalTrack.encryptionInfo,
              this.internalTrack.encryptionAuxInfo,
            );
            e < d.length && (u = d[e]);
          }
          ((u ??= ro(this.internalTrack.encryptionInfo)),
            u && (s = await no(this.internalTrack, u, s, null)));
        }
      }
      const a =
          (n.presentationTimestamp - this.internalTrack.editListOffset) /
          this.internalTrack.timescale,
        o = n.duration / this.internalTrack.timescale,
        c = new Z(s, n.isKeyFrame ? "key" : "delta", a, o, e, n.sampleSize);
      return (this.packetToSampleIndex.set(c, e), c);
    }
    async fetchPacketInFragment(e, t, r) {
      if (t === -1) return null;
      const s = e.trackData.get(this.internalTrack.id).samples[t];
      g(s);
      let a;
      if (r.metadataOnly) a = Ue;
      else {
        let u = this.internalTrack.demuxer.reader.requestSlice(
          s.byteOffset,
          s.byteSize,
        );
        if ((A(u) && (u = await u), !u)) return null;
        if (((a = N(u, s.byteSize)), this.internalTrack.encryptionInfo)) {
          const d = s.encryption ?? ro(this.internalTrack.encryptionInfo);
          d && (a = await no(this.internalTrack, d, a, e));
        }
      }
      const o =
          (s.presentationTimestamp - this.internalTrack.editListOffset) /
          this.internalTrack.timescale,
        c = s.duration / this.internalTrack.timescale,
        l = new Z(
          a,
          s.isKeyFrame ? "key" : "delta",
          o,
          c,
          e.moofOffset + t,
          s.byteSize,
        );
      return (
        this.packetToFragmentLocation.set(l, { fragment: e, sampleIndex: t }),
        l
      );
    }
    async performFragmentedLookup(e, t, r, n, s) {
      const a = this.internalTrack.demuxer;
      let o = null,
        c = null,
        l = -1;
      if (e) {
        const { sampleIndex: y, correctSampleFound: w } = t(e);
        if (w) return this.fetchPacketInFragment(e, y, s);
        y !== -1 && ((c = e), (l = y));
      }
      const u = X(
          this.internalTrack.fragmentLookupTable,
          r,
          (y) => y.timestamp,
        ),
        d = u !== -1 ? this.internalTrack.fragmentLookupTable[u] : null,
        f = X(
          this.internalTrack.fragmentPositionCache,
          r,
          (y) => y.startTimestamp,
        ),
        h = f !== -1 ? this.internalTrack.fragmentPositionCache[f] : null,
        p = Math.max(d?.moofOffset ?? 0, h?.moofOffset ?? 0) || null;
      let m;
      for (
        e
          ? p === null || e.moofOffset >= p
            ? ((m = e.moofOffset + e.moofSize), (o = e))
            : (m = p)
          : (m = p ?? 0);
        ;
      ) {
        if (o) {
          const T = o.trackData.get(this.internalTrack.id);
          if (T && T.startTimestamp > n) break;
        }
        let y = a.reader.requestSliceRange(m, kt, Xt);
        if ((A(y) && (y = await y), !y)) break;
        const w = m,
          b = Rt(y);
        if (!b) break;
        if (b.name === "moof") {
          o = await a.readFragment(w);
          const { sampleIndex: T, correctSampleFound: S } = t(o);
          if (S) return this.fetchPacketInFragment(o, T, s);
          T !== -1 && ((c = o), (l = T));
        }
        m = w + b.totalSize;
      }
      if (d && (!c || c.moofOffset < d.moofOffset)) {
        const y = this.internalTrack.fragmentLookupTable[u - 1];
        g(!y || y.timestamp < d.timestamp);
        const w = y?.timestamp ?? -1 / 0;
        return this.performFragmentedLookup(null, t, w, n, s);
      }
      return c ? this.fetchPacketInFragment(c, l, s) : null;
    }
  }
  class Qa extends Xa {
    constructor(e) {
      (super(e), (this.decoderConfigPromise = null));
    }
    getType() {
      return "video";
    }
    getCodec() {
      return this.internalTrack.info.codec;
    }
    getCodedWidth() {
      return this.internalTrack.info.width;
    }
    getCodedHeight() {
      return this.internalTrack.info.height;
    }
    getSquarePixelWidth() {
      return this.internalTrack.info.squarePixelWidth;
    }
    getSquarePixelHeight() {
      return this.internalTrack.info.squarePixelHeight;
    }
    getTransformationMatrix() {
      return [...this.internalTrack.matrix];
    }
    async getColorSpace() {
      const e = await this.getDecoderConfig();
      return e
        ? {
            primaries: e.colorSpace?.primaries,
            transfer: e.colorSpace?.transfer,
            matrix: e.colorSpace?.matrix,
            fullRange: e.colorSpace?.fullRange,
          }
        : this.internalTrack.info.colorSpace;
    }
    async canBeTransparent() {
      return (
        this.internalTrack.info.codec === "prores" &&
        (this.internalTrack.info.proresFormat === "ap4h" ||
          this.internalTrack.info.proresFormat === "ap4x")
      );
    }
    async getDecoderConfig() {
      return this.internalTrack.info.codec
        ? (this.decoderConfigPromise ??= (async () => {
            if (
              this.internalTrack.info.codec === "avc" &&
              !this.internalTrack.info.codecDescription
            ) {
              const t = await this.getFirstPacket({});
              this.internalTrack.info.avcCodecInfo = t && $i(t.data);
            } else if (
              this.internalTrack.info.codec === "hevc" &&
              !this.internalTrack.info.codecDescription
            ) {
              const t = await this.getFirstPacket({});
              this.internalTrack.info.hevcCodecInfo = t && Ki(t.data);
            } else if (
              this.internalTrack.info.codec === "vp9" &&
              (!this.internalTrack.info.vp9CodecInfo ||
                !Ml(this.internalTrack.info.vp9CodecInfo))
            ) {
              const t = await this.getFirstPacket({}),
                r = t && Dn(t.data);
              r &&
                (this.internalTrack.info.vp9CodecInfo = {
                  ...(this.internalTrack.info.vp9CodecInfo ?? r),
                  videoFullRangeFlag: r.videoFullRangeFlag,
                  colourPrimaries: r.colourPrimaries,
                  transferCharacteristics: r.transferCharacteristics,
                  matrixCoefficients: r.matrixCoefficients,
                });
            } else if (
              this.internalTrack.info.codec === "av1" &&
              (!this.internalTrack.info.av1CodecInfo ||
                !Ta(this.internalTrack.info.av1CodecInfo))
            ) {
              const t = await this.getFirstPacket({}),
                r = t && _r(t.data);
              r && (this.internalTrack.info.av1CodecInfo = r);
            } else if (
              this.internalTrack.info.codec === "prores" &&
              !this.internalTrack.info.proresCodecInfo
            ) {
              const t = await this.getFirstPacket({});
              this.internalTrack.info.proresCodecInfo = t && Sa(t.data);
            }
            if (!dn(this.internalTrack.info.colorSpace)) {
              const t = za(this.internalTrack.info);
              ((this.internalTrack.info.colorSpace.primaries ??= t.primaries),
                (this.internalTrack.info.colorSpace.transfer ??= t.transfer),
                (this.internalTrack.info.colorSpace.matrix ??= t.matrix),
                (this.internalTrack.info.colorSpace.fullRange ??= t.fullRange));
            }
            const e = {
              codec: Dr(this.internalTrack.info),
              codedWidth: this.internalTrack.info.width,
              codedHeight: this.internalTrack.info.height,
              description: this.internalTrack.info.codecDescription ?? void 0,
              colorSpace: this.internalTrack.info.colorSpace,
            };
            return (
              (this.internalTrack.info.width !==
                this.internalTrack.info.squarePixelWidth ||
                this.internalTrack.info.height !==
                  this.internalTrack.info.squarePixelHeight) &&
                ((e.displayAspectWidth =
                  this.internalTrack.info.squarePixelWidth),
                (e.displayAspectHeight =
                  this.internalTrack.info.squarePixelHeight)),
              e
            );
          })())
        : null;
    }
  }
  class Ya extends Xa {
    constructor(e) {
      (super(e), (this.decoderConfigPromise = null));
    }
    getType() {
      return "audio";
    }
    getCodec() {
      return this.internalTrack.info.codec;
    }
    getNumberOfChannels() {
      return this.internalTrack.info.numberOfChannels;
    }
    getSampleRate() {
      return this.internalTrack.info.sampleRate;
    }
    async getDecoderConfig() {
      return this.internalTrack.info.codec
        ? (this.decoderConfigPromise ??= (async () => {
            if (
              this.internalTrack.info.codec === "dts" &&
              !this.internalTrack.info.dtsFormat
            ) {
              const e = await this.getFirstPacket({});
              this.internalTrack.info.dtsFormat = e && Aa(e.data);
            }
            return {
              codec: Ln(this.internalTrack.info),
              numberOfChannels: this.internalTrack.info.numberOfChannels,
              sampleRate: this.internalTrack.info.sampleRate,
              description: this.internalTrack.info.codecDescription ?? void 0,
            };
          })())
        : null;
    }
  }
  const Xn = (i, e) => {
      if (i.presentationTimestamps) {
        const t = X(
          i.presentationTimestamps,
          e,
          (r) => r.presentationTimestamp,
        );
        return t === -1 ? -1 : i.presentationTimestamps[t].sampleIndex;
      } else {
        const t = X(i.sampleTimingEntries, e, (n) => n.startDecodeTimestamp);
        if (t === -1) return -1;
        const r = i.sampleTimingEntries[t];
        return (
          r.startIndex +
          Math.min(
            Math.floor((e - r.startDecodeTimestamp) / r.delta),
            r.count - 1,
          )
        );
      }
    },
    Tu = (i, e) => {
      if (!i.keySampleIndices) return Xn(i, e);
      if (i.presentationTimestamps) {
        const t = X(
          i.presentationTimestamps,
          e,
          (r) => r.presentationTimestamp,
        );
        if (t === -1) return -1;
        for (let r = t; r >= 0; r--) {
          const n = i.presentationTimestamps[r].sampleIndex;
          if (Oi(i.keySampleIndices, n, (a) => a) !== -1) return n;
        }
        return -1;
      } else {
        const t = Xn(i, e),
          r = X(i.keySampleIndices, t, (n) => n);
        return i.keySampleIndices[r] ?? -1;
      }
    },
    Su = (i, e) => {
      const t = X(i.sampleTimingEntries, e, (w) => w.startIndex),
        r = i.sampleTimingEntries[t];
      if (!r || r.startIndex + r.count <= e) return null;
      let s = r.startDecodeTimestamp + (e - r.startIndex) * r.delta;
      const a = X(i.sampleCompositionTimeOffsets, e, (w) => w.startIndex),
        o = i.sampleCompositionTimeOffsets[a];
      o && e - o.startIndex < o.count && (s += o.offset);
      const c = i.sampleSizes[Math.min(e, i.sampleSizes.length - 1)],
        l = X(i.sampleToChunk, e, (w) => w.startSampleIndex),
        u = i.sampleToChunk[l];
      g(u);
      const d =
          u.startChunkIndex +
          Math.floor((e - u.startSampleIndex) / u.samplesPerChunk),
        f = i.chunkOffsets[d],
        h = u.startSampleIndex + (d - u.startChunkIndex) * u.samplesPerChunk;
      let p = 0,
        m = f;
      if (i.sampleSizes.length === 1)
        ((m += c * (e - h)), (p += c * u.samplesPerChunk));
      else
        for (let w = h; w < h + u.samplesPerChunk; w++) {
          const b = i.sampleSizes[w];
          (w < e && (m += b), (p += b));
        }
      let y = r.delta;
      if (i.presentationTimestamps) {
        const w = i.presentationTimestampIndexMap[e];
        (g(w !== void 0),
          w < i.presentationTimestamps.length - 1 &&
            (y = i.presentationTimestamps[w + 1].presentationTimestamp - s));
      }
      return {
        presentationTimestamp: s,
        duration: y,
        sampleOffset: m,
        sampleSize: c,
        chunkOffset: f,
        chunkSize: p,
        isKeyFrame: i.keySampleIndices
          ? Oi(i.keySampleIndices, e, (w) => w) !== -1
          : !0,
      };
    },
    xu = (i, e) => {
      if (!i.keySampleIndices) return e + 1;
      const t = X(i.keySampleIndices, e, (r) => r);
      return i.keySampleIndices[t + 1] ?? -1;
    },
    Qn = (i, e) => {
      ((i.startTimestamp += e), (i.endTimestamp += e));
      for (const t of i.samples) t.presentationTimestamp += e;
      for (const t of i.presentationTimestamps) t.presentationTimestamp += e;
    },
    Za = (i) => [Qt(i), Qt(i), $n(i), Qt(i), Qt(i), $n(i), Qt(i), Qt(i), $n(i)],
    Ja = (i, e) => {
      const t = i.info;
      if (t.type !== "video") return;
      const { num: r, den: n } = e;
      !(r > 0 && n > 0) ||
        r === n ||
        (r > n
          ? ((t.squarePixelWidth = Math.round((t.width * r) / n)),
            (t.squarePixelHeight = t.height))
          : ((t.squarePixelWidth = t.width),
            (t.squarePixelHeight = Math.round((t.height * n) / r))));
    },
    eo = (i) => i.sampleSizes.length === 0,
    to = (i) =>
      i.currentFragmentState
        ? (i.currentFragmentState.encryptionAuxInfo ??= {
            defaultSampleInfoSize: 0,
            sampleSizes: null,
            sampleCount: 0,
            offset: null,
            resolved: null,
          })
        : (i.encryptionAuxInfo ??= {
            defaultSampleInfoSize: 0,
            sampleSizes: null,
            sampleCount: 0,
            offset: null,
            resolved: null,
          }),
    io = async (i, e, t) => {
      if (t.resolved) return t.resolved;
      if (t.offset === null || t.sampleCount === 0)
        throw new Error(
          "Incomplete saiz/saio info; cannot resolve encryption data.",
        );
      let r = 0;
      if (t.defaultSampleInfoSize > 0)
        r = t.defaultSampleInfoSize * t.sampleCount;
      else {
        g(t.sampleSizes);
        for (let o = 0; o < t.sampleCount; o++) r += t.sampleSizes[o];
      }
      let n = i.requestSlice(t.offset, r);
      if ((A(n) && (n = await n), !n))
        throw new Error("Failed to read auxiliary encryption info.");
      const s = e.defaultPerSampleIvSize;
      g(s !== null);
      const a = [];
      for (let o = 0; o < t.sampleCount; o++) {
        const c =
            t.defaultSampleInfoSize > 0
              ? t.defaultSampleInfoSize
              : t.sampleSizes[o],
          l = new Uint8Array(16);
        s > 0 ? l.set(N(n, s), 0) : l.set(e.defaultConstantIv, 0);
        let u = null;
        if (c > s) {
          const d = fe(n);
          u = [];
          for (let f = 0; f < d; f++) {
            const h = fe(n),
              p = R(n);
            u.push({ clearLen: h, protectedLen: p });
          }
        }
        a.push({ iv: l, subsamples: u });
      }
      return ((t.resolved = a), a);
    },
    ro = (i) =>
      i.defaultConstantIv
        ? { iv: i.defaultConstantIv, subsamples: null }
        : null,
    no = async (i, e, t, r) => {
      g(i.encryptionInfo);
      const n = i.encryptionInfo;
      g(n.defaultKid !== null);
      const s = n.defaultKid;
      let a;
      const o = i.demuxer.decryptionKeyCache.get(s);
      if (o) a = await o;
      else {
        if (!i.demuxer.input._formatOptions.isobmff?.resolveKeyId)
          throw new Error(
            "Encrypted media samples encountered. To decrypt them, please provide a callback for InputOptions.formatOptions.isobmff.resolveKeyId.",
          );
        const c = (async () => {
          let l = i.demuxer.psshBoxes;
          if (r) {
            l = [...l, ...r.psshBoxes].filter(
              (d) => d.keyIds === null || d.keyIds.includes(s),
            );
            for (let d = 0; d < l.length - 1; d++)
              for (let f = d + 1; f < l.length; f++)
                ja(l[d], l[f]) && (l.splice(f, 1), f--);
          }
          const u = await i.demuxer.input._formatOptions.isobmff.resolveKeyId({
            keyId: s,
            psshBoxes: l,
          });
          if (!(
            (typeof u == "string" && u.length === 32 && al.test(u)) ||
            (u instanceof Uint8Array && u.byteLength === 16)
          ))
            throw new TypeError(
              "resolveKeyId must return a 32-character hex string or a 16-byte Uint8Array containing the decryption key.",
            );
          return u instanceof Uint8Array ? u : ol(u);
        })();
        (i.demuxer.decryptionKeyCache.set(s, c), (a = await c));
      }
      return n.scheme === "cenc" || n.scheme === "cens"
        ? Pu(a, n, e, t)
        : Cu(a, n, e, t);
    },
    Pu = async (i, e, t, r) => {
      const n = new Uint8Array(16);
      n.set(t.iv, 0);
      const s = await crypto.subtle.importKey(
          "raw",
          i,
          { name: "AES-CTR" },
          !1,
          ["decrypt"],
        ),
        a = async (p) => {
          const m = await crypto.subtle.decrypt(
            { name: "AES-CTR", counter: n, length: 64 },
            s,
            p,
          );
          return new Uint8Array(m);
        };
      if (!t.subsamples) return a(r);
      g(e.defaultCryptByteBlock !== null && e.defaultSkipByteBlock !== null);
      const o = so(
        t.subsamples,
        e.defaultCryptByteBlock,
        e.defaultSkipByteBlock,
      );
      let c = 0;
      for (const p of o) for (const m of p.perSubsample) c += m.length;
      const l = new Uint8Array(c);
      let u = 0;
      for (const p of o)
        for (const m of p.perSubsample)
          (l.set(r.subarray(m.offset, m.offset + m.length), u),
            (u += m.length));
      const d = await a(l),
        f = new Uint8Array(r);
      let h = 0;
      for (const p of o)
        for (const m of p.perSubsample)
          (f.set(d.subarray(h, h + m.length), m.offset), (h += m.length));
      return f;
    },
    Cu = (i, e, t, r) => {
      const n = new Ga();
      n.init({ key: i, iv: t.iv });
      const s = e.defaultCryptByteBlock,
        a = e.defaultSkipByteBlock;
      if ((g(s !== null && a !== null), !t.subsamples)) {
        const u = new Uint8Array(r),
          d = Math.floor(r.length / 16);
        for (let f = 0; f < d; f++) {
          const h = f * 16;
          (n.in.set(r.subarray(h, h + 16)), n.decrypt(), u.set(n.out, h));
        }
        return u;
      }
      if (s === 0 && a === 0)
        throw new Error("cbcs with subsamples requires pattern encryption.");
      const o = new Uint8Array(r),
        c = so(t.subsamples, s, a),
        l = new DataView(t.iv.buffer, t.iv.byteOffset, 16);
      for (const u of c) {
        ((n.iv[0] = l.getUint32(0, !1)),
          (n.iv[1] = l.getUint32(4, !1)),
          (n.iv[2] = l.getUint32(8, !1)),
          (n.iv[3] = l.getUint32(12, !1)));
        for (const d of u.perSubsample) {
          const f = d.length / 16;
          for (let h = 0; h < f; h++) {
            const p = d.offset + h * 16;
            (n.in.set(r.subarray(p, p + 16)), n.decrypt(), o.set(n.out, p));
          }
        }
      }
      return o;
    },
    so = (i, e, t) => {
      const r = [],
        n = e !== 0 || t !== 0;
      let s = 0;
      for (const a of i) {
        s += a.clearLen;
        const o = [];
        if (!n)
          (a.protectedLen > 0 && o.push({ offset: s, length: a.protectedLen }),
            (s += a.protectedLen));
        else {
          let c = a.protectedLen,
            l = s;
          for (; c > 0 && !(c < 16 * e);) {
            const u = 16 * e;
            (o.push({ offset: l, length: u }), (l += u), (c -= u));
            const d = Math.min(16 * t, c);
            ((l += d), (c -= d));
          }
          s += a.protectedLen;
        }
        r.push({ perSubsample: o });
      }
      return r;
    };
  class Yi {
    constructor(e) {
      this.value = e;
    }
  }
  class Yn {
    constructor(e) {
      this.value = e;
    }
  }
  class ao {
    constructor(e) {
      this.value = e;
    }
  }
  class Mt {
    constructor(e) {
      this.value = e;
    }
  }
  var x;
  (function (i) {
    ((i[(i.EBML = 440786851)] = "EBML"),
      (i[(i.EBMLVersion = 17030)] = "EBMLVersion"),
      (i[(i.EBMLReadVersion = 17143)] = "EBMLReadVersion"),
      (i[(i.EBMLMaxIDLength = 17138)] = "EBMLMaxIDLength"),
      (i[(i.EBMLMaxSizeLength = 17139)] = "EBMLMaxSizeLength"),
      (i[(i.DocType = 17026)] = "DocType"),
      (i[(i.DocTypeVersion = 17031)] = "DocTypeVersion"),
      (i[(i.DocTypeReadVersion = 17029)] = "DocTypeReadVersion"),
      (i[(i.Void = 236)] = "Void"),
      (i[(i.Segment = 408125543)] = "Segment"),
      (i[(i.SeekHead = 290298740)] = "SeekHead"),
      (i[(i.Seek = 19899)] = "Seek"),
      (i[(i.SeekID = 21419)] = "SeekID"),
      (i[(i.SeekPosition = 21420)] = "SeekPosition"),
      (i[(i.Duration = 17545)] = "Duration"),
      (i[(i.Info = 357149030)] = "Info"),
      (i[(i.TimestampScale = 2807729)] = "TimestampScale"),
      (i[(i.MuxingApp = 19840)] = "MuxingApp"),
      (i[(i.WritingApp = 22337)] = "WritingApp"),
      (i[(i.Tracks = 374648427)] = "Tracks"),
      (i[(i.TrackEntry = 174)] = "TrackEntry"),
      (i[(i.TrackNumber = 215)] = "TrackNumber"),
      (i[(i.TrackUID = 29637)] = "TrackUID"),
      (i[(i.TrackType = 131)] = "TrackType"),
      (i[(i.FlagEnabled = 185)] = "FlagEnabled"),
      (i[(i.FlagDefault = 136)] = "FlagDefault"),
      (i[(i.FlagForced = 21930)] = "FlagForced"),
      (i[(i.FlagOriginal = 21934)] = "FlagOriginal"),
      (i[(i.FlagHearingImpaired = 21931)] = "FlagHearingImpaired"),
      (i[(i.FlagVisualImpaired = 21932)] = "FlagVisualImpaired"),
      (i[(i.FlagCommentary = 21935)] = "FlagCommentary"),
      (i[(i.FlagLacing = 156)] = "FlagLacing"),
      (i[(i.Name = 21358)] = "Name"),
      (i[(i.Language = 2274716)] = "Language"),
      (i[(i.LanguageBCP47 = 2274717)] = "LanguageBCP47"),
      (i[(i.CodecID = 134)] = "CodecID"),
      (i[(i.CodecPrivate = 25506)] = "CodecPrivate"),
      (i[(i.CodecDelay = 22186)] = "CodecDelay"),
      (i[(i.SeekPreRoll = 22203)] = "SeekPreRoll"),
      (i[(i.DefaultDuration = 2352003)] = "DefaultDuration"),
      (i[(i.Video = 224)] = "Video"),
      (i[(i.PixelWidth = 176)] = "PixelWidth"),
      (i[(i.PixelHeight = 186)] = "PixelHeight"),
      (i[(i.DisplayWidth = 21680)] = "DisplayWidth"),
      (i[(i.DisplayHeight = 21690)] = "DisplayHeight"),
      (i[(i.DisplayUnit = 21682)] = "DisplayUnit"),
      (i[(i.AlphaMode = 21440)] = "AlphaMode"),
      (i[(i.Audio = 225)] = "Audio"),
      (i[(i.SamplingFrequency = 181)] = "SamplingFrequency"),
      (i[(i.Channels = 159)] = "Channels"),
      (i[(i.BitDepth = 25188)] = "BitDepth"),
      (i[(i.SimpleBlock = 163)] = "SimpleBlock"),
      (i[(i.BlockGroup = 160)] = "BlockGroup"),
      (i[(i.Block = 161)] = "Block"),
      (i[(i.BlockAdditions = 30113)] = "BlockAdditions"),
      (i[(i.BlockMore = 166)] = "BlockMore"),
      (i[(i.BlockAdditional = 165)] = "BlockAdditional"),
      (i[(i.BlockAddID = 238)] = "BlockAddID"),
      (i[(i.BlockDuration = 155)] = "BlockDuration"),
      (i[(i.ReferenceBlock = 251)] = "ReferenceBlock"),
      (i[(i.Cluster = 524531317)] = "Cluster"),
      (i[(i.Timestamp = 231)] = "Timestamp"),
      (i[(i.Cues = 475249515)] = "Cues"),
      (i[(i.CuePoint = 187)] = "CuePoint"),
      (i[(i.CueTime = 179)] = "CueTime"),
      (i[(i.CueTrackPositions = 183)] = "CueTrackPositions"),
      (i[(i.CueTrack = 247)] = "CueTrack"),
      (i[(i.CueClusterPosition = 241)] = "CueClusterPosition"),
      (i[(i.Colour = 21936)] = "Colour"),
      (i[(i.MatrixCoefficients = 21937)] = "MatrixCoefficients"),
      (i[(i.TransferCharacteristics = 21946)] = "TransferCharacteristics"),
      (i[(i.Primaries = 21947)] = "Primaries"),
      (i[(i.Range = 21945)] = "Range"),
      (i[(i.Projection = 30320)] = "Projection"),
      (i[(i.ProjectionType = 30321)] = "ProjectionType"),
      (i[(i.ProjectionPoseYaw = 30323)] = "ProjectionPoseYaw"),
      (i[(i.ProjectionPosePitch = 30324)] = "ProjectionPosePitch"),
      (i[(i.ProjectionPoseRoll = 30325)] = "ProjectionPoseRoll"),
      (i[(i.Attachments = 423732329)] = "Attachments"),
      (i[(i.AttachedFile = 24999)] = "AttachedFile"),
      (i[(i.FileDescription = 18046)] = "FileDescription"),
      (i[(i.FileName = 18030)] = "FileName"),
      (i[(i.FileMediaType = 18016)] = "FileMediaType"),
      (i[(i.FileData = 18012)] = "FileData"),
      (i[(i.FileUID = 18094)] = "FileUID"),
      (i[(i.Chapters = 272869232)] = "Chapters"),
      (i[(i.Tags = 307544935)] = "Tags"),
      (i[(i.Tag = 29555)] = "Tag"),
      (i[(i.Targets = 25536)] = "Targets"),
      (i[(i.TargetTypeValue = 26826)] = "TargetTypeValue"),
      (i[(i.TargetType = 25546)] = "TargetType"),
      (i[(i.TagTrackUID = 25541)] = "TagTrackUID"),
      (i[(i.TagEditionUID = 25545)] = "TagEditionUID"),
      (i[(i.TagChapterUID = 25540)] = "TagChapterUID"),
      (i[(i.TagAttachmentUID = 25542)] = "TagAttachmentUID"),
      (i[(i.SimpleTag = 26568)] = "SimpleTag"),
      (i[(i.TagName = 17827)] = "TagName"),
      (i[(i.TagLanguage = 17530)] = "TagLanguage"),
      (i[(i.TagString = 17543)] = "TagString"),
      (i[(i.TagBinary = 17541)] = "TagBinary"),
      (i[(i.ContentEncodings = 28032)] = "ContentEncodings"),
      (i[(i.ContentEncoding = 25152)] = "ContentEncoding"),
      (i[(i.ContentEncodingOrder = 20529)] = "ContentEncodingOrder"),
      (i[(i.ContentEncodingScope = 20530)] = "ContentEncodingScope"),
      (i[(i.ContentCompression = 20532)] = "ContentCompression"),
      (i[(i.ContentCompAlgo = 16980)] = "ContentCompAlgo"),
      (i[(i.ContentCompSettings = 16981)] = "ContentCompSettings"),
      (i[(i.ContentEncryption = 20533)] = "ContentEncryption"));
  })(x || (x = {}));
  const Iu = [x.EBML, x.Segment],
    Zi = [
      x.SeekHead,
      x.Info,
      x.Cluster,
      x.Tracks,
      x.Cues,
      x.Attachments,
      x.Chapters,
      x.Tags,
    ],
    zr = [...Iu, ...Zi],
    oo = (i) =>
      i < 256
        ? 1
        : i < 65536
          ? 2
          : i < 1 << 24
            ? 3
            : i < 2 ** 32
              ? 4
              : i < 2 ** 40
                ? 5
                : 6,
    co = (i) =>
      i < 1n << 8n
        ? 1
        : i < 1n << 16n
          ? 2
          : i < 1n << 24n
            ? 3
            : i < 1n << 32n
              ? 4
              : i < 1n << 40n
                ? 5
                : i < 1n << 48n
                  ? 6
                  : i < 1n << 56n
                    ? 7
                    : 8,
    lo = (i) =>
      i >= -64 && i < 64
        ? 1
        : i >= -8192 && i < 8192
          ? 2
          : i >= -1048576 && i < 1 << 20
            ? 3
            : i >= -134217728 && i < 1 << 27
              ? 4
              : i >= -17179869184 && i < 2 ** 34
                ? 5
                : 6,
    Eu = (i) => {
      if (i < 127) return 1;
      if (i < 16383) return 2;
      if (i < (1 << 21) - 1) return 3;
      if (i < (1 << 28) - 1) return 4;
      if (i < 2 ** 35 - 1) return 5;
      if (i < 2 ** 42 - 1) return 6;
      throw new Error("EBML varint size not supported " + i);
    };
  class _u {
    constructor(e) {
      ((this.writer = e),
        (this.helper = new Uint8Array(8)),
        (this.helperView = new DataView(this.helper.buffer)),
        (this.offsets = new WeakMap()),
        (this.dataOffsets = new WeakMap()));
    }
    writeByte(e) {
      (this.helperView.setUint8(0, e),
        this.writer.write(this.helper.subarray(0, 1)));
    }
    writeFloat32(e) {
      (this.helperView.setFloat32(0, e, !1),
        this.writer.write(this.helper.subarray(0, 4)));
    }
    writeFloat64(e) {
      (this.helperView.setFloat64(0, e, !1), this.writer.write(this.helper));
    }
    writeUnsignedInt(e, t = oo(e)) {
      let r = 0;
      switch (t) {
        case 6:
          this.helperView.setUint8(r++, (e / 2 ** 40) | 0);
        case 5:
          this.helperView.setUint8(r++, (e / 2 ** 32) | 0);
        case 4:
          this.helperView.setUint8(r++, e >> 24);
        case 3:
          this.helperView.setUint8(r++, e >> 16);
        case 2:
          this.helperView.setUint8(r++, e >> 8);
        case 1:
          this.helperView.setUint8(r++, e);
          break;
        default:
          throw new Error("Bad unsigned int size " + t);
      }
      this.writer.write(this.helper.subarray(0, r));
    }
    writeUnsignedBigInt(e, t = co(e)) {
      let r = 0;
      for (let n = t - 1; n >= 0; n--)
        this.helperView.setUint8(r++, Number((e >> BigInt(n * 8)) & 0xffn));
      this.writer.write(this.helper.subarray(0, r));
    }
    writeSignedInt(e, t = lo(e)) {
      (e < 0 && (e += 2 ** (t * 8)), this.writeUnsignedInt(e, t));
    }
    writeVarInt(e, t = Eu(e)) {
      let r = 0;
      switch (t) {
        case 1:
          this.helperView.setUint8(r++, 128 | e);
          break;
        case 2:
          (this.helperView.setUint8(r++, 64 | (e >> 8)),
            this.helperView.setUint8(r++, e));
          break;
        case 3:
          (this.helperView.setUint8(r++, 32 | (e >> 16)),
            this.helperView.setUint8(r++, e >> 8),
            this.helperView.setUint8(r++, e));
          break;
        case 4:
          (this.helperView.setUint8(r++, 16 | (e >> 24)),
            this.helperView.setUint8(r++, e >> 16),
            this.helperView.setUint8(r++, e >> 8),
            this.helperView.setUint8(r++, e));
          break;
        case 5:
          (this.helperView.setUint8(r++, 8 | ((e / 2 ** 32) & 7)),
            this.helperView.setUint8(r++, e >> 24),
            this.helperView.setUint8(r++, e >> 16),
            this.helperView.setUint8(r++, e >> 8),
            this.helperView.setUint8(r++, e));
          break;
        case 6:
          (this.helperView.setUint8(r++, 4 | ((e / 2 ** 40) & 3)),
            this.helperView.setUint8(r++, (e / 2 ** 32) | 0),
            this.helperView.setUint8(r++, e >> 24),
            this.helperView.setUint8(r++, e >> 16),
            this.helperView.setUint8(r++, e >> 8),
            this.helperView.setUint8(r++, e));
          break;
        default:
          throw new Error("Bad EBML varint size " + t);
      }
      this.writer.write(this.helper.subarray(0, r));
    }
    writeAsciiString(e) {
      this.writer.write(
        new Uint8Array(e.split("").map((t) => t.charCodeAt(0))),
      );
    }
    writeEBML(e) {
      if (e !== null)
        if (e instanceof Uint8Array) this.writer.write(e);
        else if (Array.isArray(e)) for (const t of e) this.writeEBML(t);
        else if (
          (this.offsets.set(e, this.writer.getPos()),
          this.writeUnsignedInt(e.id),
          Array.isArray(e.data))
        ) {
          const t = this.writer.getPos(),
            r = e.size === -1 ? 1 : (e.size ?? 4);
          e.size === -1
            ? this.writeByte(255)
            : this.writer.seek(this.writer.getPos() + r);
          const n = this.writer.getPos();
          if (
            (this.dataOffsets.set(e, n), this.writeEBML(e.data), e.size !== -1)
          ) {
            const s = this.writer.getPos() - n,
              a = this.writer.getPos();
            (this.writer.seek(t), this.writeVarInt(s, r), this.writer.seek(a));
          }
        } else if (typeof e.data == "number") {
          const t = e.size ?? oo(e.data);
          (this.writeVarInt(t), this.writeUnsignedInt(e.data, t));
        } else if (typeof e.data == "bigint") {
          const t = e.size ?? co(e.data);
          (this.writeVarInt(t), this.writeUnsignedBigInt(e.data, t));
        } else if (typeof e.data == "string")
          (this.writeVarInt(e.data.length), this.writeAsciiString(e.data));
        else if (e.data instanceof Uint8Array)
          (this.writeVarInt(e.data.byteLength, e.size),
            this.writer.write(e.data));
        else if (e.data instanceof Yi)
          (this.writeVarInt(4), this.writeFloat32(e.data.value));
        else if (e.data instanceof Yn)
          (this.writeVarInt(8), this.writeFloat64(e.data.value));
        else if (e.data instanceof ao) {
          const t = e.size ?? lo(e.data.value);
          (this.writeVarInt(t), this.writeSignedInt(e.data.value, t));
        } else if (e.data instanceof Mt) {
          const t = oe.encode(e.data.value);
          (this.writeVarInt(t.length), this.writer.write(t));
        } else xe(e.data);
    }
  }
  const Zn = 8,
    We = 2,
    it = 2 * Zn,
    uo = (i) => {
      if (i.remainingLength < 1) return null;
      const e = U(i);
      if ((i.skip(-1), e === 0)) return null;
      let t = 1,
        r = 128;
      for (; (e & r) === 0;) (t++, (r >>= 1));
      return i.remainingLength < t ? null : t;
    },
    Ji = (i) => {
      if (i.remainingLength < 1) return null;
      const e = U(i);
      if (e === 0) return null;
      let t = 1,
        r = 128;
      for (; (e & r) === 0;) (t++, (r >>= 1));
      if (i.remainingLength < t - 1) return null;
      let n = e & (r - 1);
      for (let s = 1; s < t; s++) ((n *= 256), (n += U(i)));
      return n;
    },
    K = (i, e) => {
      if (e < 1 || e > 8) throw new Error("Bad unsigned int size " + e);
      let t = 0;
      for (let r = 0; r < e; r++) ((t *= 256), (t += U(i)));
      return t;
    },
    vu = (i, e) => {
      if (e < 1) throw new Error("Bad unsigned int size " + e);
      let t = 0n;
      for (let r = 0; r < e; r++) ((t <<= 8n), (t += BigInt(U(i))));
      return t;
    },
    Jn = (i) => {
      const e = uo(i);
      return e === null || i.remainingLength < e ? null : K(i, e);
    },
    fo = (i) => {
      if (i.remainingLength < 1) return null;
      if (U(i) === 255) return;
      i.skip(-1);
      const t = Ji(i);
      if (t === null) return null;
      if (t !== 72057594037927940) return t;
    },
    rt = (i) => {
      g(i.remainingLength >= We);
      const e = Jn(i);
      if (e === null) return null;
      const t = fo(i);
      return t === null ? null : { id: e, size: t };
    },
    Si = (i, e) => {
      const t = N(i, e);
      let r = 0;
      for (; r < e && t[r] !== 0;) r += 1;
      return String.fromCharCode(...t.subarray(0, r));
    },
    er = (i, e) => {
      const t = N(i, e);
      let r = 0;
      for (; r < e && t[r] !== 0;) r += 1;
      return Oe.decode(t.subarray(0, r));
    },
    tr = (i, e) => {
      if (e === 0) return 0;
      if (e !== 4 && e !== 8) throw new Error("Bad float size " + e);
      return e === 4 ? pf(i) : Bc(i);
    },
    es = async (i, e, t, r) => {
      const n = new Set(t);
      let s = e;
      for (; r === null || s < r;) {
        let a = i.requestSliceRange(s, We, it);
        if ((A(a) && (a = await a), !a)) break;
        const o = rt(a);
        if (!o) break;
        if (n.has(o.id)) return { pos: s, found: !0 };
        (St(o.size), (s = a.filePos + o.size));
      }
      return { pos: r !== null && r > s ? r : s, found: !1 };
    },
    ho = async (i, e, t, r) => {
      const s = new Set(t);
      let a = e;
      for (; a < r;) {
        let o = i.requestSliceRange(a, 0, Math.min(65536, r - a));
        if ((A(o) && (o = await o), !o || o.length < Zn)) break;
        for (let c = 0; c < o.length - Zn; c++) {
          o.filePos = a;
          const l = Jn(o);
          if (l !== null && s.has(l)) return a;
          a++;
        }
      }
      return null;
    },
    Re = {
      avc: "V_MPEG4/ISO/AVC",
      hevc: "V_MPEGH/ISO/HEVC",
      vp8: "V_VP8",
      vp9: "V_VP9",
      av1: "V_AV1",
      prores: "V_PRORES",
      aac: "A_AAC",
      mp3: "A_MPEG/L3",
      opus: "A_OPUS",
      vorbis: "A_VORBIS",
      flac: "A_FLAC",
      ac3: "A_AC3",
      eac3: "A_EAC3",
      dts: "A_DTS",
      "pcm-u8": "A_PCM/INT/LIT",
      "pcm-s16": "A_PCM/INT/LIT",
      "pcm-s16be": "A_PCM/INT/BIG",
      "pcm-s24": "A_PCM/INT/LIT",
      "pcm-s24be": "A_PCM/INT/BIG",
      "pcm-s32": "A_PCM/INT/LIT",
      "pcm-s32be": "A_PCM/INT/BIG",
      "pcm-f32": "A_PCM/FLOAT/IEEE",
      "pcm-f64": "A_PCM/FLOAT/IEEE",
      webvtt: "S_TEXT/WEBVTT",
    };
  function St(i) {
    if (i === void 0)
      throw new Error(
        "Undefined element size is used in a place where it is not supported.",
      );
  }
  const mo = (i) => {
    let t =
      (i.hasVideo ? "video/" : i.hasAudio ? "audio/" : "application/") +
      (i.isWebM ? "webm" : "x-matroska");
    if (i.codecStrings.length > 0) {
      const r = [...new Set(i.codecStrings.filter(Boolean))];
      t += `; codecs="${r.join(", ")}"`;
    }
    return t;
  };
  var xt;
  (function (i) {
    ((i[(i.None = 0)] = "None"),
      (i[(i.Xiph = 1)] = "Xiph"),
      (i[(i.FixedSize = 2)] = "FixedSize"),
      (i[(i.Ebml = 3)] = "Ebml"));
  })(xt || (xt = {}));
  var Nr;
  (function (i) {
    ((i[(i.Block = 1)] = "Block"),
      (i[(i.Private = 2)] = "Private"),
      (i[(i.Next = 4)] = "Next"));
  })(Nr || (Nr = {}));
  var ir;
  (function (i) {
    ((i[(i.Zlib = 0)] = "Zlib"),
      (i[(i.Bzlib = 1)] = "Bzlib"),
      (i[(i.lzo1x = 2)] = "lzo1x"),
      (i[(i.HeaderStripping = 3)] = "HeaderStripping"));
  })(ir || (ir = {}));
  const ts = [
      { id: x.SeekHead, flag: "seekHeadSeen" },
      { id: x.Info, flag: "infoSeen" },
      { id: x.Tracks, flag: "tracksSeen" },
      { id: x.Cues, flag: "cuesSeen" },
    ],
    po = 10 * 2 ** 20;
  class Au extends bt {
    constructor(e) {
      (super(e),
        (this.readMetadataPromise = null),
        (this.segments = []),
        (this.currentSegment = null),
        (this.currentTrack = null),
        (this.currentCluster = null),
        (this.currentBlock = null),
        (this.currentBlockAdditional = null),
        (this.currentCueTime = null),
        (this.currentDecodingInstruction = null),
        (this.currentTagTargetIsMovie = !0),
        (this.currentSimpleTagName = null),
        (this.currentAttachedFile = null),
        (this.isWebM = !1),
        (this.reader = e._reader));
    }
    async getTrackBackings() {
      return (
        await this.readMetadata(),
        this.segments.flatMap((e) => e.tracks.map((t) => t.trackBacking))
      );
    }
    async getMimeType() {
      await this.readMetadata();
      const e = await this.getTrackBackings(),
        t = await Promise.all(
          e.map((r) => r.getDecoderConfig().then((n) => n?.codec ?? null)),
        );
      return mo({
        isWebM: this.isWebM,
        hasVideo: this.segments.some((r) =>
          r.tracks.some((n) => n.info?.type === "video"),
        ),
        hasAudio: this.segments.some((r) =>
          r.tracks.some((n) => n.info?.type === "audio"),
        ),
        codecStrings: t.filter(Boolean),
      });
    }
    async getMetadataTags() {
      await this.readMetadata();
      for (const t of this.segments)
        t.metadataTagsCollected ||
          (this.reader.fileSize !== null && (await this.loadSegmentMetadata(t)),
          (t.metadataTagsCollected = !0));
      let e = {};
      for (const t of this.segments) e = { ...e, ...t.metadataTags };
      return e;
    }
    readMetadata() {
      return (this.readMetadataPromise ??= (async () => {
        let e = 0;
        for (;;) {
          let t = this.reader.requestSliceRange(e, We, it);
          if ((A(t) && (t = await t), !t)) break;
          const r = rt(t);
          if (!r) break;
          const n = r.id;
          let s = r.size;
          const a = t.filePos;
          if (n === x.EBML) {
            St(s);
            let o = this.reader.requestSlice(a, s);
            if ((A(o) && (o = await o), !o)) break;
            this.readContiguousElements(o);
          } else if (n === x.Segment) {
            if (
              (await this.readSegment(a, s),
              s === void 0 || this.reader.fileSize === null)
            )
              break;
          } else if (n === x.Cluster) {
            if (this.reader.fileSize === null) break;
            s === void 0 &&
              (s =
                (await es(this.reader, a, zr, this.reader.fileSize)).pos - a);
            const o = ne(this.segments);
            o && (o.elementEndPos = a + s);
          }
          (St(s), (e = a + s));
        }
      })());
    }
    async readSegment(e, t) {
      ((this.currentSegment = {
        seekHeadSeen: !1,
        infoSeen: !1,
        tracksSeen: !1,
        cuesSeen: !1,
        tagsSeen: !1,
        attachmentsSeen: !1,
        timestampScale: -1,
        timestampFactor: -1,
        duration: -1,
        seekEntries: [],
        tracks: [],
        cuePoints: [],
        dataStartPos: e,
        elementEndPos: t === void 0 ? null : e + t,
        clusterSeekStartPos: e,
        lastReadCluster: null,
        metadataTags: {},
        metadataTagsCollected: !1,
      }),
        this.segments.push(this.currentSegment));
      let r = e;
      const n = new Set();
      for (
        ;
        this.currentSegment.elementEndPos === null ||
        r < this.currentSegment.elementEndPos;
      ) {
        let c = this.reader.requestSliceRange(r, We, it);
        if ((A(c) && (c = await c), !c)) break;
        const l = r,
          u = rt(c);
        if (!u || (!Zi.includes(u.id) && u.id !== x.Void)) {
          const m = await ho(
            this.reader,
            l,
            Zi,
            Math.min(this.currentSegment.elementEndPos ?? 1 / 0, l + po),
          );
          if (m) {
            r = m;
            continue;
          } else break;
        }
        const { id: d, size: f } = u,
          h = c.filePos,
          p = ts.findIndex((m) => m.id === d);
        if (p !== -1) {
          const m = ts[p].flag;
          ((this.currentSegment[m] = !0),
            d === x.SeekHead && n.add(l - e),
            St(f));
          let y = this.reader.requestSlice(h, f);
          (A(y) && (y = await y), y && this.readContiguousElements(y));
        } else if (d === x.Tags || d === x.Attachments) {
          (d === x.Tags
            ? (this.currentSegment.tagsSeen = !0)
            : (this.currentSegment.attachmentsSeen = !0),
            St(f));
          let m = this.reader.requestSlice(h, f);
          (A(m) && (m = await m), m && this.readContiguousElements(m));
        } else if (d === x.Cluster) {
          this.currentSegment.clusterSeekStartPos = l;
          break;
        }
        if (f === void 0) break;
        r = h + f;
      }
      if (this.reader.fileSize !== null)
        for (;;) {
          const c = this.currentSegment.seekEntries.find(
            (h) => h.id === x.SeekHead && !n.has(h.segmentPosition),
          );
          if (!c) break;
          n.add(c.segmentPosition);
          let l = this.reader.requestSliceRange(e + c.segmentPosition, We, it);
          if ((A(l) && (l = await l), !l)) continue;
          const u = rt(l);
          if (!u || u.id !== x.SeekHead) continue;
          const { size: d } = u;
          St(d);
          let f = this.reader.requestSlice(l.filePos, d);
          (A(f) && (f = await f), f && this.readContiguousElements(f));
        }
      if (
        (this.currentSegment.seekEntries.sort(
          (c, l) => c.segmentPosition - l.segmentPosition,
        ),
        this.reader.fileSize !== null)
      )
        for (const c of this.currentSegment.seekEntries) {
          const l = ts.find((m) => m.id === c.id);
          if (!l || this.currentSegment[l.flag]) continue;
          let u = this.reader.requestSliceRange(e + c.segmentPosition, We, it);
          if ((A(u) && (u = await u), !u)) continue;
          const d = rt(u);
          if (!d) continue;
          const { id: f, size: h } = d;
          if (f !== l.id) continue;
          (St(h), (this.currentSegment[l.flag] = !0));
          let p = this.reader.requestSlice(u.filePos, h);
          (A(p) && (p = await p), p && this.readContiguousElements(p));
        }
      this.currentSegment.timestampScale === -1 &&
        ((this.currentSegment.timestampScale = 1e6),
        (this.currentSegment.timestampFactor = 1e9 / 1e6));
      for (const c of this.currentSegment.tracks)
        c.defaultDurationNs !== null &&
          (c.defaultDuration =
            (this.currentSegment.timestampFactor * c.defaultDurationNs) / 1e9);
      const s = new Map(this.currentSegment.tracks.map((c) => [c.id, c]));
      for (const c of this.currentSegment.cuePoints) {
        const l = s.get(c.trackId);
        l && l.cuePoints.push(c);
      }
      for (const c of this.currentSegment.tracks) {
        c.cuePoints.sort((l, u) => l.time - u.time);
        for (let l = 0; l < c.cuePoints.length - 1; l++) {
          const u = c.cuePoints[l],
            d = c.cuePoints[l + 1];
          u.time === d.time && (c.cuePoints.splice(l + 1, 1), l--);
        }
      }
      let a = null,
        o = -1 / 0;
      for (const c of this.currentSegment.tracks)
        c.cuePoints.length > o && ((o = c.cuePoints.length), (a = c));
      for (const c of this.currentSegment.tracks)
        c.cuePoints.length === 0 && (c.cuePoints = a.cuePoints);
      this.currentSegment = null;
    }
    async readCluster(e, t) {
      if (t.lastReadCluster?.elementStartPos === e) return t.lastReadCluster;
      let r = this.reader.requestSliceRange(e, We, it);
      (A(r) && (r = await r), g(r));
      const n = e,
        s = rt(r);
      g(s);
      const a = s.id;
      g(a === x.Cluster);
      let o = s.size;
      const c = r.filePos;
      o === void 0 &&
        (o = (await es(this.reader, c, zr, t.elementEndPos)).pos - c);
      let l = this.reader.requestSlice(c, o);
      A(l) && (l = await l);
      const u = {
        segment: t,
        elementStartPos: n,
        elementEndPos: c + o,
        dataStartPos: c,
        timestamp: -1,
        trackData: new Map(),
      };
      if (((this.currentCluster = u), l)) {
        const d = this.readContiguousElements(l, zr);
        u.elementEndPos = d;
      }
      for (const [, d] of u.trackData) {
        const f = d.track;
        g(d.blocks.length > 0);
        let h = !1;
        for (let w = 0; w < d.blocks.length; w++) {
          const b = d.blocks[w];
          ((b.timestamp += u.timestamp), (h ||= b.lacing !== xt.None));
        }
        d.presentationTimestamps = d.blocks
          .map((w, b) => ({ timestamp: w.timestamp, blockIndex: b }))
          .sort((w, b) => w.timestamp - b.timestamp);
        for (let w = 0; w < d.presentationTimestamps.length; w++) {
          const b = d.presentationTimestamps[w],
            T = d.blocks[b.blockIndex];
          if (
            (d.firstKeyFrameTimestamp === null &&
              T.isKeyFrame &&
              (d.firstKeyFrameTimestamp = T.timestamp),
            w < d.presentationTimestamps.length - 1)
          ) {
            const S = d.presentationTimestamps[w + 1];
            T.duration = S.timestamp - T.timestamp;
          } else
            T.duration === 0 &&
              f.defaultDuration != null &&
              T.lacing === xt.None &&
              (T.duration = f.defaultDuration);
        }
        h &&
          (this.expandLacedBlocks(d.blocks, f),
          (d.presentationTimestamps = d.blocks
            .map((w, b) => ({ timestamp: w.timestamp, blockIndex: b }))
            .sort((w, b) => w.timestamp - b.timestamp)));
        const p = d.blocks[d.presentationTimestamps[0].blockIndex],
          m = d.blocks[ne(d.presentationTimestamps).blockIndex];
        ((d.startTimestamp = p.timestamp),
          (d.endTimestamp = m.timestamp + m.duration));
        const y = X(
          f.clusterPositionCache,
          d.startTimestamp,
          (w) => w.startTimestamp,
        );
        (y === -1 || f.clusterPositionCache[y].elementStartPos !== n) &&
          f.clusterPositionCache.splice(y + 1, 0, {
            elementStartPos: u.elementStartPos,
            startTimestamp: d.startTimestamp,
          });
      }
      return ((t.lastReadCluster = u), u);
    }
    getTrackDataInCluster(e, t) {
      let r = e.trackData.get(t);
      if (!r) {
        const n = e.segment.tracks.find((s) => s.id === t);
        if (!n) return null;
        ((r = {
          track: n,
          startTimestamp: 0,
          endTimestamp: 0,
          firstKeyFrameTimestamp: null,
          blocks: [],
          presentationTimestamps: [],
        }),
          e.trackData.set(t, r));
      }
      return r;
    }
    expandLacedBlocks(e, t) {
      for (let r = 0; r < e.length; r++) {
        const n = e[r];
        if (n.lacing === xt.None) continue;
        n.decoded ||
          ((n.data = this.decodeBlockData(t, n.data)), (n.decoded = !0));
        const s = Me.tempFromBytes(n.data),
          a = [],
          o = U(s) + 1;
        switch (n.lacing) {
          case xt.Xiph:
            {
              let l = 0;
              for (let u = 0; u < o - 1; u++) {
                let d = 0;
                for (; s.bufferPos < s.length;) {
                  const f = U(s);
                  if (((d += f), f < 255)) {
                    (a.push(d), (l += d));
                    break;
                  }
                }
              }
              a.push(s.length - (s.bufferPos + l));
            }
            break;
          case xt.FixedSize:
            {
              const l = s.length - 1,
                u = Math.floor(l / o);
              for (let d = 0; d < o; d++) a.push(u);
            }
            break;
          case xt.Ebml:
            {
              const l = Ji(s);
              g(l !== null);
              let u = l;
              a.push(u);
              let d = u;
              for (let f = 1; f < o - 1; f++) {
                const h = s.bufferPos,
                  p = Ji(s);
                g(p !== null);
                const m = p,
                  w = (1 << ((s.bufferPos - h) * 7 - 1)) - 1,
                  b = m - w;
                ((u += b), a.push(u), (d += u));
              }
              a.push(s.length - (s.bufferPos + d));
            }
            break;
          default:
            g(!1);
        }
        (g(a.length === o), e.splice(r, 1));
        const c = n.duration || o * (t.defaultDuration ?? 0);
        for (let l = 0; l < o; l++) {
          const u = a[l],
            d = N(s, u),
            f = n.timestamp + (c * l) / o,
            h = c / o;
          e.splice(r + l, 0, {
            timestamp: f,
            duration: h,
            isKeyFrame: n.isKeyFrame,
            data: d,
            lacing: xt.None,
            decoded: !0,
            postProcessed: !1,
            mainAdditional: n.mainAdditional,
          });
        }
        ((r += o), r--);
      }
    }
    async loadSegmentMetadata(e) {
      for (const t of e.seekEntries) {
        if (!(t.id === x.Tags && !e.tagsSeen)) {
          if (!(t.id === x.Attachments && !e.attachmentsSeen)) continue;
        }
        let r = this.reader.requestSliceRange(
          e.dataStartPos + t.segmentPosition,
          We,
          it,
        );
        if ((A(r) && (r = await r), !r)) continue;
        const n = rt(r);
        if (!n || n.id !== t.id) continue;
        const { size: s } = n;
        (St(s), g(!this.currentSegment), (this.currentSegment = e));
        let a = this.reader.requestSlice(r.filePos, s);
        (A(a) && (a = await a),
          a && this.readContiguousElements(a),
          (this.currentSegment = null),
          t.id === x.Tags
            ? (e.tagsSeen = !0)
            : t.id === x.Attachments && (e.attachmentsSeen = !0));
      }
    }
    readContiguousElements(e, t) {
      for (; e.remainingLength >= We;) {
        const r = e.filePos;
        if (!this.traverseElement(e, t)) return r;
      }
      return e.filePos;
    }
    traverseElement(e, t) {
      const r = rt(e);
      if (!r || (t && t.includes(r.id))) return !1;
      const { id: n, size: s } = r,
        a = e.filePos;
      switch ((St(s), n)) {
        case x.DocType:
          this.isWebM = Si(e, s) === "webm";
          break;
        case x.Seek:
          {
            if (!this.currentSegment) break;
            const o = { id: -1, segmentPosition: -1 };
            (this.currentSegment.seekEntries.push(o),
              this.readContiguousElements(e.slice(a, s)),
              (o.id === -1 || o.segmentPosition === -1) &&
                this.currentSegment.seekEntries.pop());
          }
          break;
        case x.SeekID:
          {
            const o =
              this.currentSegment?.seekEntries[
                this.currentSegment.seekEntries.length - 1
              ];
            if (!o) break;
            o.id = K(e, s);
          }
          break;
        case x.SeekPosition:
          {
            const o =
              this.currentSegment?.seekEntries[
                this.currentSegment.seekEntries.length - 1
              ];
            if (!o) break;
            o.segmentPosition = K(e, s);
          }
          break;
        case x.TimestampScale:
          {
            if (!this.currentSegment) break;
            ((this.currentSegment.timestampScale = K(e, s)),
              (this.currentSegment.timestampFactor =
                1e9 / this.currentSegment.timestampScale));
          }
          break;
        case x.Duration:
          {
            if (!this.currentSegment) break;
            this.currentSegment.duration = tr(e, s);
          }
          break;
        case x.TrackEntry:
          {
            if (
              !this.currentSegment ||
              ((this.currentTrack = {
                id: -1,
                segment: this.currentSegment,
                demuxer: this,
                clusterPositionCache: [],
                cuePoints: [],
                disposition: { ...wt, primary: !1 },
                trackBacking: null,
                codecId: null,
                codecPrivate: null,
                defaultDuration: null,
                defaultDurationNs: null,
                name: null,
                languageCode: "eng",
                hasLanguageBcp47: !1,
                decodingInstructions: [],
                info: null,
              }),
              this.readContiguousElements(e.slice(a, s)),
              !this.currentTrack)
            )
              break;
            if (
              (this.currentTrack.decodingInstructions.some(
                (o) =>
                  o.data?.type !== "decompress" ||
                  o.scope !== Nr.Block ||
                  o.data.algorithm !== ir.HeaderStripping,
              ) &&
                (L._warn(
                  `Track #${this.currentTrack.id} has an unsupported content encoding; dropping.`,
                ),
                (this.currentTrack = null)),
              this.currentTrack &&
                this.currentTrack.id !== -1 &&
                this.currentTrack.codecId &&
                this.currentTrack.info)
            ) {
              const o = this.currentTrack.codecId.indexOf("/"),
                c =
                  o === -1
                    ? this.currentTrack.codecId
                    : this.currentTrack.codecId.slice(0, o);
              if (
                this.currentTrack.info.type === "video" &&
                this.currentTrack.info.width !== -1 &&
                this.currentTrack.info.height !== -1
              ) {
                if (
                  ((this.currentTrack.info.squarePixelWidth =
                    this.currentTrack.info.width),
                  (this.currentTrack.info.squarePixelHeight =
                    this.currentTrack.info.height),
                  this.currentTrack.info.displayWidth !== null &&
                    this.currentTrack.info.displayHeight !== null)
                ) {
                  const u =
                      this.currentTrack.info.displayWidth *
                      this.currentTrack.info.height,
                    d =
                      this.currentTrack.info.displayHeight *
                      this.currentTrack.info.width;
                  u > 0 &&
                    d > 0 &&
                    (u > d
                      ? (this.currentTrack.info.squarePixelWidth = Math.round(
                          (this.currentTrack.info.width * u) / d,
                        ))
                      : (this.currentTrack.info.squarePixelHeight = Math.round(
                          (this.currentTrack.info.height * d) / u,
                        )));
                }
                if (this.currentTrack.codecId === Re.avc)
                  ((this.currentTrack.info.codec = "avc"),
                    (this.currentTrack.info.codecDescription =
                      this.currentTrack.codecPrivate));
                else if (this.currentTrack.codecId === Re.hevc)
                  ((this.currentTrack.info.codec = "hevc"),
                    (this.currentTrack.info.codecDescription =
                      this.currentTrack.codecPrivate));
                else if (c === Re.vp8) this.currentTrack.info.codec = "vp8";
                else if (c === Re.vp9) this.currentTrack.info.codec = "vp9";
                else if (c === Re.av1) this.currentTrack.info.codec = "av1";
                else if (c === Re.prores) {
                  const u = this.currentTrack.codecPrivate
                    ? Oe.decode(this.currentTrack.codecPrivate)
                    : "";
                  Kt.includes(u) &&
                    ((this.currentTrack.info.codec = "prores"),
                    (this.currentTrack.info.proresFormat = u));
                }
                const l = this.currentTrack;
                ((this.currentTrack.trackBacking = new Bu(l)),
                  this.currentSegment.tracks.push(this.currentTrack));
              } else if (this.currentTrack.info.type === "audio") {
                c === Re.aac
                  ? ((this.currentTrack.info.codec = "aac"),
                    (this.currentTrack.info.aacCodecInfo = {
                      isMpeg2: this.currentTrack.codecId.includes("MPEG2"),
                      objectType: null,
                    }),
                    (this.currentTrack.info.codecDescription =
                      this.currentTrack.codecPrivate))
                  : this.currentTrack.codecId === Re.mp3
                    ? (this.currentTrack.info.codec = "mp3")
                    : c === Re.opus
                      ? ((this.currentTrack.info.codec = "opus"),
                        (this.currentTrack.info.codecDescription =
                          this.currentTrack.codecPrivate),
                        (this.currentTrack.info.sampleRate = Gi))
                      : c === Re.vorbis
                        ? ((this.currentTrack.info.codec = "vorbis"),
                          (this.currentTrack.info.codecDescription =
                            this.currentTrack.codecPrivate))
                        : c === Re.flac
                          ? ((this.currentTrack.info.codec = "flac"),
                            (this.currentTrack.info.codecDescription =
                              this.currentTrack.codecPrivate))
                          : c === Re.ac3
                            ? ((this.currentTrack.info.codec = "ac3"),
                              (this.currentTrack.info.codecDescription =
                                this.currentTrack.codecPrivate))
                            : c === Re.eac3
                              ? ((this.currentTrack.info.codec = "eac3"),
                                (this.currentTrack.info.codecDescription =
                                  this.currentTrack.codecPrivate))
                              : c === Re.dts
                                ? ((this.currentTrack.info.codec = "dts"),
                                  this.currentTrack.codecId === "A_DTS/EXPRESS"
                                    ? (this.currentTrack.info.dtsFormat =
                                        "dtse")
                                    : this.currentTrack.codecId ===
                                        "A_DTS/LOSSLESS" &&
                                      (this.currentTrack.info.dtsFormat =
                                        "dtsl"))
                                : this.currentTrack.codecId === "A_PCM/INT/LIT"
                                  ? this.currentTrack.info.bitDepth === 8
                                    ? (this.currentTrack.info.codec = "pcm-u8")
                                    : this.currentTrack.info.bitDepth === 16
                                      ? (this.currentTrack.info.codec =
                                          "pcm-s16")
                                      : this.currentTrack.info.bitDepth === 24
                                        ? (this.currentTrack.info.codec =
                                            "pcm-s24")
                                        : this.currentTrack.info.bitDepth ===
                                            32 &&
                                          (this.currentTrack.info.codec =
                                            "pcm-s32")
                                  : this.currentTrack.codecId ===
                                      "A_PCM/INT/BIG"
                                    ? this.currentTrack.info.bitDepth === 8
                                      ? (this.currentTrack.info.codec =
                                          "pcm-u8")
                                      : this.currentTrack.info.bitDepth === 16
                                        ? (this.currentTrack.info.codec =
                                            "pcm-s16be")
                                        : this.currentTrack.info.bitDepth === 24
                                          ? (this.currentTrack.info.codec =
                                              "pcm-s24be")
                                          : this.currentTrack.info.bitDepth ===
                                              32 &&
                                            (this.currentTrack.info.codec =
                                              "pcm-s32be")
                                    : this.currentTrack.codecId ===
                                        "A_PCM/FLOAT/IEEE" &&
                                      (this.currentTrack.info.bitDepth === 32
                                        ? (this.currentTrack.info.codec =
                                            "pcm-f32")
                                        : this.currentTrack.info.bitDepth ===
                                            64 &&
                                          (this.currentTrack.info.codec =
                                            "pcm-f64"));
                const l = this.currentTrack;
                ((this.currentTrack.trackBacking = new Fu(l)),
                  this.currentSegment.tracks.push(this.currentTrack));
              }
            }
            this.currentTrack = null;
          }
          break;
        case x.TrackNumber:
          {
            if (!this.currentTrack) break;
            this.currentTrack.id = K(e, s);
          }
          break;
        case x.TrackType:
          {
            if (!this.currentTrack) break;
            const o = K(e, s);
            o === 1
              ? (this.currentTrack.info = {
                  type: "video",
                  width: -1,
                  height: -1,
                  displayWidth: null,
                  displayHeight: null,
                  displayUnit: null,
                  squarePixelWidth: -1,
                  squarePixelHeight: -1,
                  horizontalScale: 1,
                  verticalScale: 1,
                  rotation: 0,
                  codec: null,
                  codecDescription: null,
                  colorSpace: { ...Qs },
                  alphaMode: !1,
                  proresFormat: null,
                })
              : o === 2 &&
                (this.currentTrack.info = {
                  type: "audio",
                  numberOfChannels: 1,
                  sampleRate: 8e3,
                  bitDepth: -1,
                  codec: null,
                  codecDescription: null,
                  aacCodecInfo: null,
                  dtsFormat: null,
                });
          }
          break;
        case x.FlagEnabled:
          {
            if (!this.currentTrack) break;
            K(e, s) || (this.currentTrack = null);
          }
          break;
        case x.FlagDefault:
          {
            if (!this.currentTrack) break;
            this.currentTrack.disposition.default = !!K(e, s);
          }
          break;
        case x.FlagForced:
          {
            if (!this.currentTrack) break;
            this.currentTrack.disposition.forced = !!K(e, s);
          }
          break;
        case x.FlagOriginal:
          {
            if (!this.currentTrack) break;
            this.currentTrack.disposition.original = !!K(e, s);
          }
          break;
        case x.FlagHearingImpaired:
          {
            if (!this.currentTrack) break;
            this.currentTrack.disposition.hearingImpaired = !!K(e, s);
          }
          break;
        case x.FlagVisualImpaired:
          {
            if (!this.currentTrack) break;
            this.currentTrack.disposition.visuallyImpaired = !!K(e, s);
          }
          break;
        case x.FlagCommentary:
          {
            if (!this.currentTrack) break;
            this.currentTrack.disposition.commentary = !!K(e, s);
          }
          break;
        case x.CodecID:
          {
            if (!this.currentTrack) break;
            this.currentTrack.codecId = Si(e, s);
          }
          break;
        case x.CodecPrivate:
          {
            if (!this.currentTrack) break;
            this.currentTrack.codecPrivate = N(e, s);
          }
          break;
        case x.DefaultDuration:
          {
            if (!this.currentTrack) break;
            this.currentTrack.defaultDurationNs = K(e, s);
          }
          break;
        case x.Name:
          {
            if (!this.currentTrack) break;
            this.currentTrack.name = er(e, s);
          }
          break;
        case x.Language:
          {
            if (!this.currentTrack || this.currentTrack.hasLanguageBcp47) break;
            ((this.currentTrack.languageCode = Si(e, s)),
              Ni(this.currentTrack.languageCode) ||
                (this.currentTrack.languageCode = Pe));
          }
          break;
        case x.LanguageBCP47:
          {
            if (!this.currentTrack) break;
            const c = Si(e, s).split("-")[0];
            (c
              ? (this.currentTrack.languageCode = c)
              : (this.currentTrack.languageCode = Pe),
              (this.currentTrack.hasLanguageBcp47 = !0));
          }
          break;
        case x.Video:
          {
            if (this.currentTrack?.info?.type !== "video") break;
            this.readContiguousElements(e.slice(a, s));
          }
          break;
        case x.PixelWidth:
          {
            if (this.currentTrack?.info?.type !== "video") break;
            this.currentTrack.info.width = K(e, s);
          }
          break;
        case x.PixelHeight:
          {
            if (this.currentTrack?.info?.type !== "video") break;
            this.currentTrack.info.height = K(e, s);
          }
          break;
        case x.DisplayWidth:
          {
            if (this.currentTrack?.info?.type !== "video") break;
            this.currentTrack.info.displayWidth = K(e, s);
          }
          break;
        case x.DisplayHeight:
          {
            if (this.currentTrack?.info?.type !== "video") break;
            this.currentTrack.info.displayHeight = K(e, s);
          }
          break;
        case x.DisplayUnit:
          {
            if (this.currentTrack?.info?.type !== "video") break;
            this.currentTrack.info.displayUnit = K(e, s);
          }
          break;
        case x.AlphaMode:
          {
            if (this.currentTrack?.info?.type !== "video") break;
            this.currentTrack.info.alphaMode = K(e, s) === 1;
          }
          break;
        case x.Colour:
          {
            if (this.currentTrack?.info?.type !== "video") break;
            this.readContiguousElements(e.slice(a, s));
          }
          break;
        case x.MatrixCoefficients:
          {
            if (this.currentTrack?.info?.type !== "video") break;
            const o = K(e, s),
              c = pt[o];
            this.currentTrack.info.colorSpace.matrix = c;
          }
          break;
        case x.Range:
          {
            if (this.currentTrack?.info?.type !== "video") break;
            const o = K(e, s);
            this.currentTrack.info.colorSpace.fullRange =
              o === 1 || o === 2 ? o === 2 : void 0;
          }
          break;
        case x.TransferCharacteristics:
          {
            if (this.currentTrack?.info?.type !== "video") break;
            const o = K(e, s),
              c = mt[o];
            this.currentTrack.info.colorSpace.transfer = c;
          }
          break;
        case x.Primaries:
          {
            if (this.currentTrack?.info?.type !== "video") break;
            const o = K(e, s),
              c = ht[o];
            this.currentTrack.info.colorSpace.primaries = c;
          }
          break;
        case x.Projection:
          {
            if (this.currentTrack?.info?.type !== "video") break;
            this.readContiguousElements(e.slice(a, s));
          }
          break;
        case x.ProjectionPoseYaw:
          {
            if (this.currentTrack?.info?.type !== "video") break;
            const o = tr(e, s);
            try {
              const c = Lt(o);
              this.currentTrack.info.horizontalScale = Math.round(
                Math.cos(c * on),
              );
            } catch {}
          }
          break;
        case x.ProjectionPosePitch:
          {
            if (this.currentTrack?.info?.type !== "video") break;
            const o = tr(e, s);
            try {
              const c = Lt(o);
              this.currentTrack.info.verticalScale = Math.round(
                Math.cos(c * on),
              );
            } catch {}
          }
          break;
        case x.ProjectionPoseRoll:
          {
            if (this.currentTrack?.info?.type !== "video") break;
            const c = -tr(e, s);
            try {
              this.currentTrack.info.rotation = Lt(c);
            } catch {}
          }
          break;
        case x.Audio:
          {
            if (this.currentTrack?.info?.type !== "audio") break;
            this.readContiguousElements(e.slice(a, s));
          }
          break;
        case x.SamplingFrequency:
          {
            if (this.currentTrack?.info?.type !== "audio") break;
            this.currentTrack.info.sampleRate = tr(e, s);
          }
          break;
        case x.Channels:
          {
            if (this.currentTrack?.info?.type !== "audio") break;
            this.currentTrack.info.numberOfChannels = K(e, s);
          }
          break;
        case x.BitDepth:
          {
            if (this.currentTrack?.info?.type !== "audio") break;
            this.currentTrack.info.bitDepth = K(e, s);
          }
          break;
        case x.CuePoint:
          {
            if (!this.currentSegment) break;
            (this.readContiguousElements(e.slice(a, s)),
              (this.currentCueTime = null));
          }
          break;
        case x.CueTime:
          this.currentCueTime = K(e, s);
          break;
        case x.CueTrackPositions:
          {
            if (this.currentCueTime === null) break;
            g(this.currentSegment);
            const o = {
              time: this.currentCueTime,
              trackId: -1,
              clusterPosition: -1,
            };
            (this.currentSegment.cuePoints.push(o),
              this.readContiguousElements(e.slice(a, s)),
              (o.trackId === -1 || o.clusterPosition === -1) &&
                this.currentSegment.cuePoints.pop());
          }
          break;
        case x.CueTrack:
          {
            const o =
              this.currentSegment?.cuePoints[
                this.currentSegment.cuePoints.length - 1
              ];
            if (!o) break;
            o.trackId = K(e, s);
          }
          break;
        case x.CueClusterPosition:
          {
            const o =
              this.currentSegment?.cuePoints[
                this.currentSegment.cuePoints.length - 1
              ];
            if (!o) break;
            (g(this.currentSegment),
              (o.clusterPosition = this.currentSegment.dataStartPos + K(e, s)));
          }
          break;
        case x.Timestamp:
          {
            if (!this.currentCluster) break;
            this.currentCluster.timestamp = K(e, s);
          }
          break;
        case x.SimpleBlock:
          {
            if (!this.currentCluster) break;
            const o = Ji(e);
            if (o === null) break;
            const c = this.getTrackDataInCluster(this.currentCluster, o);
            if (!c) break;
            const l = Ts(e),
              u = U(e),
              d = (u >> 1) & 3;
            let f = !!(u & 128);
            c.track.info?.type === "audio" && c.track.info.codec && (f = !0);
            const h = N(e, s - (e.filePos - a)),
              p = c.track.decodingInstructions.length > 0;
            c.blocks.push({
              timestamp: l,
              duration: 0,
              isKeyFrame: f,
              data: h,
              lacing: d,
              decoded: !p,
              postProcessed: !1,
              mainAdditional: null,
            });
          }
          break;
        case x.BlockGroup:
          {
            if (!this.currentCluster) break;
            (this.readContiguousElements(e.slice(a, s)),
              (this.currentBlock = null));
          }
          break;
        case x.Block:
          {
            if (!this.currentCluster) break;
            const o = Ji(e);
            if (o === null) break;
            const c = this.getTrackDataInCluster(this.currentCluster, o);
            if (!c) break;
            const l = Ts(e),
              d = (U(e) >> 1) & 3,
              f = N(e, s - (e.filePos - a)),
              h = c.track.decodingInstructions.length > 0;
            ((this.currentBlock = {
              timestamp: l,
              duration: 0,
              isKeyFrame: !0,
              data: f,
              lacing: d,
              decoded: !h,
              postProcessed: !1,
              mainAdditional: null,
            }),
              c.blocks.push(this.currentBlock));
          }
          break;
        case x.BlockAdditions:
          this.readContiguousElements(e.slice(a, s));
          break;
        case x.BlockMore:
          {
            if (!this.currentBlock) break;
            ((this.currentBlockAdditional = { addId: 1, data: null }),
              this.readContiguousElements(e.slice(a, s)),
              this.currentBlockAdditional.data &&
                this.currentBlockAdditional.addId === 1 &&
                (this.currentBlock.mainAdditional =
                  this.currentBlockAdditional.data),
              (this.currentBlockAdditional = null));
          }
          break;
        case x.BlockAdditional:
          {
            if (!this.currentBlockAdditional) break;
            this.currentBlockAdditional.data = N(e, s);
          }
          break;
        case x.BlockAddID:
          {
            if (!this.currentBlockAdditional) break;
            this.currentBlockAdditional.addId = K(e, s);
          }
          break;
        case x.BlockDuration:
          {
            if (!this.currentBlock) break;
            this.currentBlock.duration = K(e, s);
          }
          break;
        case x.ReferenceBlock:
          {
            if (!this.currentBlock) break;
            this.currentBlock.isKeyFrame = !1;
          }
          break;
        case x.Tag:
          ((this.currentTagTargetIsMovie = !0),
            this.readContiguousElements(e.slice(a, s)));
          break;
        case x.Targets:
          this.readContiguousElements(e.slice(a, s));
          break;
        case x.TargetTypeValue:
          K(e, s) !== 50 && (this.currentTagTargetIsMovie = !1);
          break;
        case x.TagTrackUID:
        case x.TagEditionUID:
        case x.TagChapterUID:
        case x.TagAttachmentUID:
          this.currentTagTargetIsMovie = !1;
          break;
        case x.SimpleTag:
          {
            if (!this.currentTagTargetIsMovie) break;
            ((this.currentSimpleTagName = null),
              this.readContiguousElements(e.slice(a, s)));
          }
          break;
        case x.TagName:
          this.currentSimpleTagName = er(e, s);
          break;
        case x.TagString:
          {
            if (!this.currentSimpleTagName) break;
            const o = er(e, s);
            this.processTagValue(this.currentSimpleTagName, o);
          }
          break;
        case x.TagBinary:
          {
            if (!this.currentSimpleTagName) break;
            const o = N(e, s);
            this.processTagValue(this.currentSimpleTagName, o);
          }
          break;
        case x.AttachedFile:
          {
            if (!this.currentSegment) break;
            ((this.currentAttachedFile = {
              fileUid: null,
              fileName: null,
              fileMediaType: null,
              fileData: null,
              fileDescription: null,
            }),
              this.readContiguousElements(e.slice(a, s)));
            const o = this.currentSegment.metadataTags;
            if (
              (this.currentAttachedFile.fileUid &&
                this.currentAttachedFile.fileData &&
                ((o.raw ??= {}),
                (o.raw[this.currentAttachedFile.fileUid.toString()] = new Cn(
                  this.currentAttachedFile.fileData,
                  this.currentAttachedFile.fileMediaType ?? void 0,
                  this.currentAttachedFile.fileName ?? void 0,
                  this.currentAttachedFile.fileDescription ?? void 0,
                ))),
              this.currentAttachedFile.fileMediaType?.startsWith("image/") &&
                this.currentAttachedFile.fileData)
            ) {
              const c = this.currentAttachedFile.fileName;
              let l = "unknown";
              if (c) {
                const u = c.toLowerCase();
                u.startsWith("cover.")
                  ? (l = "coverFront")
                  : u.startsWith("back.") && (l = "coverBack");
              }
              ((o.images ??= []),
                o.images.push({
                  data: this.currentAttachedFile.fileData,
                  mimeType: this.currentAttachedFile.fileMediaType,
                  kind: l,
                  name: this.currentAttachedFile.fileName ?? void 0,
                  description:
                    this.currentAttachedFile.fileDescription ?? void 0,
                }));
            }
            this.currentAttachedFile = null;
          }
          break;
        case x.FileUID:
          {
            if (!this.currentAttachedFile) break;
            this.currentAttachedFile.fileUid = vu(e, s);
          }
          break;
        case x.FileName:
          {
            if (!this.currentAttachedFile) break;
            this.currentAttachedFile.fileName = er(e, s);
          }
          break;
        case x.FileMediaType:
          {
            if (!this.currentAttachedFile) break;
            this.currentAttachedFile.fileMediaType = Si(e, s);
          }
          break;
        case x.FileData:
          {
            if (!this.currentAttachedFile) break;
            this.currentAttachedFile.fileData = N(e, s);
          }
          break;
        case x.FileDescription:
          {
            if (!this.currentAttachedFile) break;
            this.currentAttachedFile.fileDescription = er(e, s);
          }
          break;
        case x.ContentEncodings:
          {
            if (!this.currentTrack) break;
            (this.readContiguousElements(e.slice(a, s)),
              this.currentTrack.decodingInstructions.sort(
                (o, c) => c.order - o.order,
              ));
          }
          break;
        case x.ContentEncoding:
          ((this.currentDecodingInstruction = {
            order: 0,
            scope: Nr.Block,
            data: null,
          }),
            this.readContiguousElements(e.slice(a, s)),
            this.currentDecodingInstruction.data &&
              this.currentTrack.decodingInstructions.push(
                this.currentDecodingInstruction,
              ),
            (this.currentDecodingInstruction = null));
          break;
        case x.ContentEncodingOrder:
          {
            if (!this.currentDecodingInstruction) break;
            this.currentDecodingInstruction.order = K(e, s);
          }
          break;
        case x.ContentEncodingScope:
          {
            if (!this.currentDecodingInstruction) break;
            this.currentDecodingInstruction.scope = K(e, s);
          }
          break;
        case x.ContentCompression:
          {
            if (!this.currentDecodingInstruction) break;
            ((this.currentDecodingInstruction.data = {
              type: "decompress",
              algorithm: ir.Zlib,
              settings: null,
            }),
              this.readContiguousElements(e.slice(a, s)));
          }
          break;
        case x.ContentCompAlgo:
          {
            if (this.currentDecodingInstruction?.data?.type !== "decompress")
              break;
            this.currentDecodingInstruction.data.algorithm = K(e, s);
          }
          break;
        case x.ContentCompSettings:
          {
            if (this.currentDecodingInstruction?.data?.type !== "decompress")
              break;
            this.currentDecodingInstruction.data.settings = N(e, s);
          }
          break;
        case x.ContentEncryption:
          {
            if (!this.currentDecodingInstruction) break;
            this.currentDecodingInstruction.data = { type: "decrypt" };
          }
          break;
      }
      return ((e.filePos = a + s), !0);
    }
    decodeBlockData(e, t) {
      g(e.decodingInstructions.length > 0);
      let r = t;
      for (const n of e.decodingInstructions)
        switch ((g(n.data), n.data.type)) {
          case "decompress":
            switch (n.data.algorithm) {
              case ir.HeaderStripping:
                if (n.data.settings && n.data.settings.length > 0) {
                  const s = n.data.settings,
                    a = new Uint8Array(s.length + r.length);
                  (a.set(s, 0), a.set(r, s.length), (r = a));
                }
                break;
            }
            break;
        }
      return r;
    }
    processTagValue(e, t) {
      if (!this.currentSegment?.metadataTags) return;
      const r = this.currentSegment.metadataTags;
      if (((r.raw ??= {}), (r.raw[e] ??= t), typeof t == "string"))
        switch (e.toLowerCase()) {
          case "title":
            r.title ??= t;
            break;
          case "description":
            r.description ??= t;
            break;
          case "artist":
            r.artist ??= t;
            break;
          case "album":
            r.album ??= t;
            break;
          case "album_artist":
            r.albumArtist ??= t;
            break;
          case "genre":
            r.genre ??= t;
            break;
          case "bpm":
            {
              const n = Number.parseInt(t, 10);
              Number.isInteger(n) && n > 0 && (r.beatsPerMinute ??= n);
            }
            break;
          case "comment":
            r.comment ??= t;
            break;
          case "lyrics":
            r.lyrics ??= t;
            break;
          case "date":
            {
              const n = new Date(t);
              Number.isNaN(n.getTime()) || (r.date ??= n);
            }
            break;
          case "track_number":
          case "part_number":
            {
              const n = t.split("/"),
                s = Number.parseInt(n[0], 10),
                a = n[1] && Number.parseInt(n[1], 10);
              (Number.isInteger(s) && s > 0 && (r.trackNumber ??= s),
                a && Number.isInteger(a) && a > 0 && (r.tracksTotal ??= a));
            }
            break;
          case "disc_number":
          case "disc":
            {
              const n = t.split("/"),
                s = Number.parseInt(n[0], 10),
                a = n[1] && Number.parseInt(n[1], 10);
              (Number.isInteger(s) && s > 0 && (r.discNumber ??= s),
                a && Number.isInteger(a) && a > 0 && (r.discsTotal ??= a));
            }
            break;
        }
    }
    getDurationFromMetadata(e) {
      return e.duration <= 0 ? null : e.duration / e.timestampFactor;
    }
  }
  class go {
    constructor(e) {
      ((this.internalTrack = e),
        (this.packetToClusterLocation = new WeakMap()));
    }
    getId() {
      return this.internalTrack.id;
    }
    getNumber() {
      const e = this.internalTrack.demuxer,
        t = this.internalTrack.trackBacking.getType();
      let r = 0;
      for (const n of e.segments)
        for (const s of n.tracks)
          if ((s.trackBacking.getType() === t && r++, s === this.internalTrack))
            break;
      return r;
    }
    getCodec() {
      throw new Error("Not implemented on base class.");
    }
    getInternalCodecId() {
      return this.internalTrack.codecId;
    }
    getName() {
      return this.internalTrack.name;
    }
    getLanguageCode() {
      return this.internalTrack.languageCode;
    }
    getTimeResolution() {
      return this.internalTrack.segment.timestampFactor;
    }
    isRelativeToUnixEpoch() {
      return !1;
    }
    getUnixTimeForTimestamp() {
      return null;
    }
    getDisposition() {
      return this.internalTrack.disposition;
    }
    getPairingMask() {
      return 1n;
    }
    getBitrate() {
      return null;
    }
    getAverageBitrate() {
      return null;
    }
    async getDurationFromMetadata() {
      return this.internalTrack.demuxer.getDurationFromMetadata(
        this.internalTrack.segment,
      );
    }
    async getLiveRefreshInterval() {
      return null;
    }
    async getFirstPacket(e) {
      return this.performClusterLookup(
        null,
        (t) =>
          t.trackData.get(this.internalTrack.id)
            ? { blockIndex: 0, correctBlockFound: !0 }
            : { blockIndex: -1, correctBlockFound: !1 },
        -1 / 0,
        1 / 0,
        e,
      );
    }
    intoTimescale(e) {
      return zi(e * this.internalTrack.segment.timestampFactor);
    }
    async getPacket(e, t) {
      const r = this.intoTimescale(e);
      return this.performClusterLookup(
        null,
        (n) => {
          const s = n.trackData.get(this.internalTrack.id);
          if (!s) return { blockIndex: -1, correctBlockFound: !1 };
          const a = X(s.presentationTimestamps, r, (l) => l.timestamp),
            o = a !== -1 ? s.presentationTimestamps[a].blockIndex : -1,
            c = a !== -1 && r < s.endTimestamp;
          return { blockIndex: o, correctBlockFound: c };
        },
        r,
        r,
        t,
      );
    }
    async getNextPacket(e, t) {
      const r = this.packetToClusterLocation.get(e);
      if (r === void 0)
        throw new Error("Packet was not created from this track.");
      return this.performClusterLookup(
        r.cluster,
        (n) => {
          if (n === r.cluster) {
            const s = n.trackData.get(this.internalTrack.id);
            if (r.blockIndex + 1 < s.blocks.length)
              return { blockIndex: r.blockIndex + 1, correctBlockFound: !0 };
          } else if (n.trackData.get(this.internalTrack.id))
            return { blockIndex: 0, correctBlockFound: !0 };
          return { blockIndex: -1, correctBlockFound: !1 };
        },
        -1 / 0,
        1 / 0,
        t,
      );
    }
    async getKeyPacket(e, t) {
      const r = this.intoTimescale(e);
      return this.performClusterLookup(
        null,
        (n) => {
          const s = n.trackData.get(this.internalTrack.id);
          if (!s) return { blockIndex: -1, correctBlockFound: !1 };
          const a = hn(
              s.presentationTimestamps,
              (l) => s.blocks[l.blockIndex].isKeyFrame && l.timestamp <= r,
            ),
            o = a !== -1 ? s.presentationTimestamps[a].blockIndex : -1,
            c = a !== -1 && r < s.endTimestamp;
          return { blockIndex: o, correctBlockFound: c };
        },
        r,
        r,
        t,
      );
    }
    async getNextKeyPacket(e, t) {
      const r = this.packetToClusterLocation.get(e);
      if (r === void 0)
        throw new Error("Packet was not created from this track.");
      return this.performClusterLookup(
        r.cluster,
        (n) => {
          if (n === r.cluster) {
            const a = n.trackData
              .get(this.internalTrack.id)
              .blocks.findIndex((o, c) => o.isKeyFrame && c > r.blockIndex);
            if (a !== -1) return { blockIndex: a, correctBlockFound: !0 };
          } else {
            const s = n.trackData.get(this.internalTrack.id);
            if (s && s.firstKeyFrameTimestamp !== null) {
              const a = s.blocks.findIndex((o) => o.isKeyFrame);
              return (g(a !== -1), { blockIndex: a, correctBlockFound: !0 });
            }
          }
          return { blockIndex: -1, correctBlockFound: !1 };
        },
        -1 / 0,
        1 / 0,
        t,
      );
    }
    async fetchPacketInCluster(e, t, r) {
      if (t === -1) return null;
      const s = e.trackData.get(this.internalTrack.id).blocks[t];
      if (
        (g(s),
        s.decoded ||
          ((s.data = this.internalTrack.demuxer.decodeBlockData(
            this.internalTrack,
            s.data,
          )),
          (s.decoded = !0)),
        !s.postProcessed)
      ) {
        if (
          this.internalTrack.info?.codec === "prores" &&
          !(
            s.data.length >= 8 &&
            s.data[4] === 105 &&
            s.data[5] === 99 &&
            s.data[6] === 112 &&
            s.data[7] === 102
          )
        ) {
          const f = new Uint8Array(s.data.length + 8);
          (j(f).setUint32(0, f.length, !1),
            (f[4] = 105),
            (f[5] = 99),
            (f[6] = 112),
            (f[7] = 102),
            f.set(s.data, 8),
            (s.data = f));
        }
        s.postProcessed = !0;
      }
      const a = r.metadataOnly ? Ue : s.data,
        o = s.timestamp / this.internalTrack.segment.timestampFactor,
        c = s.duration / this.internalTrack.segment.timestampFactor,
        l = {};
      s.mainAdditional &&
        this.internalTrack.info?.type === "video" &&
        this.internalTrack.info.alphaMode &&
        ((l.alpha = r.metadataOnly ? Ue : s.mainAdditional),
        (l.alphaByteLength = s.mainAdditional.byteLength));
      const u = new Z(
        a,
        s.isKeyFrame ? "key" : "delta",
        o,
        c,
        e.dataStartPos + t,
        s.data.byteLength,
        l,
      );
      return (
        this.packetToClusterLocation.set(u, { cluster: e, blockIndex: t }),
        u
      );
    }
    async performClusterLookup(e, t, r, n, s) {
      const { demuxer: a, segment: o } = this.internalTrack;
      let c = null,
        l = null,
        u = -1;
      if (e) {
        const { blockIndex: w, correctBlockFound: b } = t(e);
        if (b) return this.fetchPacketInCluster(e, w, s);
        w !== -1 && ((l = e), (u = w));
      }
      const d = X(this.internalTrack.cuePoints, r, (w) => w.time),
        f = d !== -1 ? this.internalTrack.cuePoints[d] : null,
        h = X(
          this.internalTrack.clusterPositionCache,
          r,
          (w) => w.startTimestamp,
        ),
        p = h !== -1 ? this.internalTrack.clusterPositionCache[h] : null,
        m = Math.max(f?.clusterPosition ?? 0, p?.elementStartPos ?? 0) || null;
      let y;
      for (
        e
          ? m === null || e.elementStartPos >= m
            ? ((y = e.elementEndPos), (c = e))
            : (y = m)
          : (y = m ?? o.clusterSeekStartPos);
        o.elementEndPos === null || y <= o.elementEndPos - We;
      ) {
        if (c) {
          const E = c.trackData.get(this.internalTrack.id);
          if (E && E.startTimestamp > n) break;
        }
        let w = a.reader.requestSliceRange(y, We, it);
        if ((A(w) && (w = await w), !w)) break;
        const b = y,
          T = rt(w);
        if (!T || (!Zi.includes(T.id) && T.id !== x.Void)) {
          const E = await ho(
            a.reader,
            b,
            Zi,
            Math.min(o.elementEndPos ?? 1 / 0, b + po),
          );
          if (E) {
            y = E;
            continue;
          } else break;
        }
        const S = T.id;
        let k = T.size;
        const P = w.filePos;
        if (S === x.Cluster) {
          ((c = await a.readCluster(b, o)), (k = c.elementEndPos - P));
          const { blockIndex: E, correctBlockFound: C } = t(c);
          if (C) return this.fetchPacketInCluster(c, E, s);
          E !== -1 && ((l = c), (u = E));
        }
        k === void 0 &&
          (g(S !== x.Cluster),
          (k = (await es(a.reader, P, zr, o.elementEndPos)).pos - P));
        const I = P + k;
        if (o.elementEndPos === null) {
          let E = a.reader.requestSliceRange(I, We, it);
          if ((A(E) && (E = await E), !E)) break;
          if (Jn(E) === x.Segment) {
            o.elementEndPos = I;
            break;
          }
        }
        y = I;
      }
      if (f && (!l || l.elementStartPos < f.clusterPosition)) {
        const w = this.internalTrack.cuePoints[d - 1];
        g(!w || w.time < f.time);
        const b = w?.time ?? -1 / 0;
        return this.performClusterLookup(null, t, b, n, s);
      }
      return l ? this.fetchPacketInCluster(l, u, s) : null;
    }
  }
  class Bu extends go {
    constructor(e) {
      (super(e), (this.decoderConfigPromise = null));
    }
    getType() {
      return "video";
    }
    getCodec() {
      return this.internalTrack.info.codec;
    }
    getCodedWidth() {
      return this.internalTrack.info.width;
    }
    getCodedHeight() {
      return this.internalTrack.info.height;
    }
    getSquarePixelWidth() {
      return this.internalTrack.info.squarePixelWidth;
    }
    getSquarePixelHeight() {
      return this.internalTrack.info.squarePixelHeight;
    }
    getTransformationMatrix() {
      const e = this.internalTrack.info,
        t = ai(Ks(e.horizontalScale, e.verticalScale), ln(e.rotation));
      return qs(t, e.width, e.height);
    }
    async getColorSpace() {
      const e = await this.getDecoderConfig();
      return e
        ? {
            primaries: e.colorSpace?.primaries,
            transfer: e.colorSpace?.transfer,
            matrix: e.colorSpace?.matrix,
            fullRange: e.colorSpace?.fullRange,
          }
        : this.internalTrack.info.colorSpace;
    }
    async canBeTransparent() {
      return (
        this.internalTrack.info.alphaMode ||
        (this.internalTrack.info.codec === "prores" &&
          (this.internalTrack.info.proresFormat === "ap4h" ||
            this.internalTrack.info.proresFormat === "ap4x"))
      );
    }
    async getDecoderConfig() {
      return this.internalTrack.info.codec
        ? (this.decoderConfigPromise ??= (async () => {
            let e = null;
            (this.internalTrack.info.codec === "vp9" ||
              this.internalTrack.info.codec === "av1" ||
              this.internalTrack.info.codec === "prores" ||
              (this.internalTrack.info.codec === "avc" &&
                !this.internalTrack.info.codecDescription) ||
              (this.internalTrack.info.codec === "hevc" &&
                !this.internalTrack.info.codecDescription)) &&
              (e = await this.getFirstPacket({}));
            const r = {
              width: this.internalTrack.info.width,
              height: this.internalTrack.info.height,
              codec: this.internalTrack.info.codec,
              codecDescription: this.internalTrack.info.codecDescription,
              colorSpace: this.internalTrack.info.colorSpace,
              avcType: 1,
              avcCodecInfo:
                this.internalTrack.info.codec === "avc" && e
                  ? $i(e.data)
                  : null,
              hevcCodecInfo:
                this.internalTrack.info.codec === "hevc" && e
                  ? Ki(e.data)
                  : null,
              vp9CodecInfo:
                this.internalTrack.info.codec === "vp9" && e
                  ? Dn(e.data)
                  : null,
              av1CodecInfo:
                this.internalTrack.info.codec === "av1" && e
                  ? _r(e.data)
                  : null,
              proresCodecInfo:
                this.internalTrack.info.codec === "prores" && e
                  ? Sa(e.data)
                  : null,
              proresFormat: this.internalTrack.info.proresFormat,
            };
            if (!dn(this.internalTrack.info.colorSpace)) {
              const s = za(r);
              ((this.internalTrack.info.colorSpace.primaries ??= s.primaries),
                (this.internalTrack.info.colorSpace.transfer ??= s.transfer),
                (this.internalTrack.info.colorSpace.matrix ??= s.matrix),
                (this.internalTrack.info.colorSpace.fullRange ??= s.fullRange));
            }
            const n = {
              codec: Dr(r),
              codedWidth: this.internalTrack.info.width,
              codedHeight: this.internalTrack.info.height,
              description: this.internalTrack.info.codecDescription ?? void 0,
              colorSpace: this.internalTrack.info.colorSpace,
            };
            return (
              (this.internalTrack.info.width !==
                this.internalTrack.info.squarePixelWidth ||
                this.internalTrack.info.height !==
                  this.internalTrack.info.squarePixelHeight) &&
                ((n.displayAspectWidth =
                  this.internalTrack.info.squarePixelWidth),
                (n.displayAspectHeight =
                  this.internalTrack.info.squarePixelHeight)),
              n
            );
          })())
        : null;
    }
  }
  class Fu extends go {
    constructor(e) {
      (super(e), (this.decoderConfigPromise = null));
    }
    getType() {
      return "audio";
    }
    getCodec() {
      return this.internalTrack.info.codec;
    }
    getNumberOfChannels() {
      return this.internalTrack.info.numberOfChannels;
    }
    getSampleRate() {
      return this.internalTrack.info.sampleRate;
    }
    async getDecoderConfig() {
      return this.internalTrack.info.codec
        ? (this.decoderConfigPromise ??= (async () => {
            if (
              this.internalTrack.info.codec === "dts" &&
              !this.internalTrack.info.dtsFormat
            ) {
              const e = await this.getFirstPacket({});
              this.internalTrack.info.dtsFormat = e && Aa(e.data);
            }
            return {
              codec: Ln({
                codec: this.internalTrack.info.codec,
                codecDescription: this.internalTrack.info.codecDescription,
                aacCodecInfo: this.internalTrack.info.aacCodecInfo,
                dtsFormat: this.internalTrack.info.dtsFormat,
              }),
              numberOfChannels: this.internalTrack.info.numberOfChannels,
              sampleRate: this.internalTrack.info.sampleRate,
              description: this.internalTrack.info.codecDescription ?? void 0,
            };
          })())
        : null;
    }
  }
  const is = async (i, e, t, r = null) => {
    let s = e;
    for (; t === null || s < t;) {
      const a = t !== null ? Math.min(65536, t - s) : 65536;
      let o = i.requestSliceRange(s, Gt, a);
      if ((A(o) && (o = await o), !o || o.length < Gt)) break;
      for (; o.remainingLength >= Gt;) {
        const c = o.filePos,
          l = R(o),
          u = i.fileSize !== null ? i.fileSize - s : null,
          d = qn(l, u);
        if (
          d.header &&
          (!r ||
            (d.header.sampleRate === r.sampleRate &&
              d.header.mpegVersionId === r.mpegVersionId &&
              d.header.layer === r.layer &&
              Qi(d.header.channel) === Qi(r.channel)))
        )
          return { header: d.header, startPos: s };
        ((o.filePos = c + d.bytesAdvanced), (s = o.filePos));
      }
    }
    return null;
  };
  class Ru extends bt {
    constructor(e) {
      (super(e),
        (this.metadataPromise = null),
        (this.firstFrameHeader = null),
        (this.firstFrameHeaderPos = null),
        (this.xingFrameHeader = null),
        (this.xingFrameHeaderPos = null),
        (this.loadedSamples = []),
        (this.metadataTags = null),
        (this.xingData = null),
        (this.trackBackings = []),
        (this.readingMutex = new ci()),
        (this.lastSampleLoaded = !1),
        (this.lastLoadedPos = 0),
        (this.nextTimestampInSamples = 0),
        (this.reader = e._reader));
    }
    async readMetadata() {
      return (this.metadataPromise ??= (async () => {
        for (; !this.firstFrameHeader && !this.lastSampleLoaded;)
          await this.advanceReader();
        if (
          (!this.firstFrameHeader &&
            this.xingFrameHeader &&
            ((this.firstFrameHeader = this.xingFrameHeader),
            (this.firstFrameHeaderPos = this.xingFrameHeaderPos)),
          !this.firstFrameHeader)
        )
          throw new Error("No valid MP3 frame found.");
        this.trackBackings = [new Mu(this)];
      })());
    }
    async advanceReader() {
      if (this.lastLoadedPos === 0)
        for (;;) {
          let o = this.reader.requestSlice(this.lastLoadedPos, je);
          if ((A(o) && (o = await o), !o)) {
            this.lastSampleLoaded = !0;
            return;
          }
          const c = Et(o);
          if (!c) break;
          this.lastLoadedPos = o.filePos + c.size;
        }
      const e = await is(
        this.reader,
        this.lastLoadedPos,
        this.reader.fileSize,
        this.firstFrameHeader,
      );
      if (!e) {
        this.lastSampleLoaded = !0;
        return;
      }
      const t = e.header;
      this.lastLoadedPos = e.startPos + t.totalSize - 1;
      const r = La(t.mpegVersionId, t.channel);
      let n = this.reader.requestSlice(e.startPos + r, 4);
      if ((A(n) && (n = await n), n)) {
        const o = R(n);
        if (o === Wa || o === Va) {
          if (
            (this.xingFrameHeader ||
              ((this.xingFrameHeader = t),
              (this.xingFrameHeaderPos = e.startPos)),
            !this.xingData)
          ) {
            let l = this.reader.requestSlice(e.startPos + r + 4, 12);
            if ((A(l) && (l = await l), l)) {
              const u = N(l, 12),
                d = j(u),
                f = d.getUint32(0, !1);
              this.xingData = {
                frameCount: f & Or.FrameCount ? d.getUint32(4, !1) : null,
                fileSize: f & Or.FileSize ? d.getUint32(8, !1) : null,
              };
            }
          }
          return;
        }
      }
      this.firstFrameHeader ||
        ((this.firstFrameHeader = t), (this.firstFrameHeaderPos = e.startPos));
      const s = t.audioSamplesInFrame / this.firstFrameHeader.sampleRate,
        a = {
          timestamp:
            this.nextTimestampInSamples / this.firstFrameHeader.sampleRate,
          duration: s,
          dataStart: e.startPos,
          dataSize: t.totalSize,
        };
      (this.loadedSamples.push(a),
        (this.nextTimestampInSamples += t.audioSamplesInFrame));
    }
    async getMimeType() {
      return "audio/mpeg";
    }
    async getTrackBackings() {
      return (await this.readMetadata(), this.trackBackings);
    }
    async getMetadataTags() {
      const e = await this.readingMutex.acquire();
      try {
        if ((await this.readMetadata(), this.metadataTags))
          return this.metadataTags;
        this.metadataTags = {};
        let t = 0,
          r = !1;
        for (;;) {
          let n = this.reader.requestSlice(t, je);
          if ((A(n) && (n = await n), !n)) break;
          const s = Et(n);
          if (!s) break;
          r = !0;
          let a = this.reader.requestSlice(n.filePos, s.size);
          if ((A(a) && (a = await a), !a)) break;
          (Gr(a, s, this.metadataTags), (t = n.filePos + s.size));
        }
        if (!r && this.reader.fileSize !== null && this.reader.fileSize >= Kr) {
          let n = this.reader.requestSlice(this.reader.fileSize - Kr, Kr);
          (A(n) && (n = await n),
            g(n),
            le(n, 3) === "TAG" && gf(n, this.metadataTags));
        }
        return this.metadataTags;
      } finally {
        e();
      }
    }
  }
  class Mu {
    constructor(e) {
      this.demuxer = e;
    }
    getType() {
      return "audio";
    }
    getId() {
      return 1;
    }
    getNumber() {
      return 1;
    }
    getTimeResolution() {
      return (
        g(this.demuxer.firstFrameHeader),
        this.demuxer.firstFrameHeader.sampleRate /
          this.demuxer.firstFrameHeader.audioSamplesInFrame
      );
    }
    isRelativeToUnixEpoch() {
      return !1;
    }
    getUnixTimeForTimestamp() {
      return null;
    }
    getPairingMask() {
      return 1n;
    }
    getBitrate() {
      return null;
    }
    getAverageBitrate() {
      return null;
    }
    async getDurationFromMetadata() {
      const e = this.demuxer;
      if (
        (g(e.firstFrameHeader !== null),
        g(e.firstFrameHeaderPos !== null),
        e.xingData)
      ) {
        if (e.xingData.frameCount !== null)
          return (
            (e.xingData.frameCount * e.firstFrameHeader.audioSamplesInFrame) /
            e.firstFrameHeader.sampleRate
          );
      } else if (e.reader.fileSize !== null) {
        const t = gu(
            e.firstFrameHeader.lowSamplingFrequency,
            e.firstFrameHeader.layer,
            e.firstFrameHeader.bitrate,
            e.firstFrameHeader.sampleRate,
          ),
          r = (e.reader.fileSize - e.firstFrameHeaderPos) / t;
        return (
          (Math.round(r) * e.firstFrameHeader.audioSamplesInFrame) /
          e.firstFrameHeader.sampleRate
        );
      }
      return null;
    }
    async getLiveRefreshInterval() {
      return null;
    }
    getName() {
      return null;
    }
    getLanguageCode() {
      return Pe;
    }
    getCodec() {
      return "mp3";
    }
    getInternalCodecId() {
      return null;
    }
    getNumberOfChannels() {
      return (
        g(this.demuxer.firstFrameHeader),
        Qi(this.demuxer.firstFrameHeader.channel)
      );
    }
    getSampleRate() {
      return (
        g(this.demuxer.firstFrameHeader),
        this.demuxer.firstFrameHeader.sampleRate
      );
    }
    getDisposition() {
      return { ...wt };
    }
    async getDecoderConfig() {
      return (
        g(this.demuxer.firstFrameHeader),
        {
          codec: "mp3",
          numberOfChannels: Qi(this.demuxer.firstFrameHeader.channel),
          sampleRate: this.demuxer.firstFrameHeader.sampleRate,
        }
      );
    }
    async getPacketAtIndex(e, t) {
      if (e === -1) return null;
      const r = this.demuxer.loadedSamples[e];
      if (!r) return null;
      let n;
      if (t.metadataOnly) n = Ue;
      else {
        let s = this.demuxer.reader.requestSlice(r.dataStart, r.dataSize);
        if ((A(s) && (s = await s), !s)) return null;
        n = N(s, r.dataSize);
      }
      return new Z(n, "key", r.timestamp, r.duration, e, r.dataSize);
    }
    getFirstPacket(e) {
      return this.getPacketAtIndex(0, e);
    }
    async getNextPacket(e, t) {
      const r = await this.demuxer.readingMutex.acquire();
      try {
        const n = Oi(
          this.demuxer.loadedSamples,
          e.timestamp,
          (a) => a.timestamp,
        );
        if (n === -1)
          throw new Error("Packet was not created from this track.");
        const s = n + 1;
        for (
          ;
          s >= this.demuxer.loadedSamples.length &&
          !this.demuxer.lastSampleLoaded;
        )
          await this.demuxer.advanceReader();
        return this.getPacketAtIndex(s, t);
      } finally {
        r();
      }
    }
    async getPacket(e, t) {
      const r = await this.demuxer.readingMutex.acquire();
      try {
        for (;;) {
          const n = X(this.demuxer.loadedSamples, e, (s) => s.timestamp);
          if (n === -1 && this.demuxer.loadedSamples.length > 0) return null;
          if (this.demuxer.lastSampleLoaded) return this.getPacketAtIndex(n, t);
          if (n >= 0 && n + 1 < this.demuxer.loadedSamples.length)
            return this.getPacketAtIndex(n, t);
          await this.demuxer.advanceReader();
        }
      } finally {
        r();
      }
    }
    getKeyPacket(e, t) {
      return this.getPacket(e, t);
    }
    getNextKeyPacket(e, t) {
      return this.getNextPacket(e, t);
    }
  }
  const wo = 1399285583,
    Du = 79764919,
    yo = new Uint32Array(256);
  for (let i = 0; i < 256; i++) {
    let e = i << 24;
    for (let t = 0; t < 8; t++) e = e & 2147483648 ? (e << 1) ^ Du : e << 1;
    yo[i] = (e >>> 0) & 4294967295;
  }
  const Ou = (i) => {
      const e = j(i),
        t = e.getUint32(22, !0);
      e.setUint32(22, 0, !0);
      let r = 0;
      for (let n = 0; n < i.length; n++) {
        const s = i[n];
        r = ((r << 8) ^ yo[(r >>> 24) ^ s]) >>> 0;
      }
      return (e.setUint32(22, t, !0), r);
    },
    bo = (i, e, t) => {
      let r = 0,
        n = null;
      if (i.length > 0)
        if (e.codec === "vorbis") {
          g(e.vorbisInfo);
          const s = e.vorbisInfo.modeBlockflags.length,
            o = ((1 << hl(s - 1)) - 1) << 1,
            c = (i[0] & o) >> 1;
          if (c >= e.vorbisInfo.modeBlockflags.length)
            throw new Error("Invalid mode number.");
          let l = t;
          const u = e.vorbisInfo.modeBlockflags[c];
          if (((n = e.vorbisInfo.blocksizes[u]), u === 1)) {
            const d = (o | 1) + 1,
              f = i[0] & d ? 1 : 0;
            l = e.vorbisInfo.blocksizes[f];
          }
          r = l !== null ? (l + n) >> 2 : 0;
        } else e.codec === "opus" && (r = Ol(i).durationInSamples);
      return { durationInSamples: r, vorbisBlockSize: n };
    },
    zu = (i) => {
      let e = "audio/ogg";
      if (i.codecStrings) {
        const t = [...new Set(i.codecStrings)];
        e += `; codecs="${t.join(", ")}"`;
      }
      return e;
    };
  const Dt = 27,
    xi = 282,
    Nu = xi + 65025,
    rr = (i) => {
      const e = i.filePos;
      if (_i(i) !== wo) return null;
      i.skip(1);
      const r = U(i),
        n = mf(i),
        s = _i(i),
        a = _i(i),
        o = _i(i),
        c = U(i),
        l = new Uint8Array(c);
      for (let h = 0; h < c; h++) l[h] = U(i);
      const u = 27 + c,
        d = l.reduce((h, p) => h + p, 0),
        f = u + d;
      return {
        headerStartPos: e,
        totalSize: f,
        dataStartPos: e + u,
        dataSize: d,
        headerType: r,
        granulePosition: n,
        serialNumber: s,
        sequenceNumber: a,
        checksum: o,
        lacingValues: l,
      };
    },
    Uu = (i, e) => {
      for (; i.filePos < e - 3;) {
        const t = _i(i),
          r = t & 255,
          n = (t >>> 8) & 255,
          s = (t >>> 16) & 255,
          a = (t >>> 24) & 255,
          o = 79;
        if (!(r !== o && n !== o && s !== o && a !== o)) {
          if ((i.skip(-4), t === wo)) return !0;
          i.skip(1);
        }
      }
      return !1;
    };
  class Wu extends bt {
    constructor(e) {
      (super(e),
        (this.metadataPromise = null),
        (this.bitstreams = []),
        (this.trackBackings = []),
        (this.metadataTags = {}),
        (this.reader = e._reader));
    }
    async readMetadata() {
      return (this.metadataPromise ??= (async () => {
        let e = 0;
        for (;;) {
          let t = this.reader.requestSliceRange(e, Dt, xi);
          if ((A(t) && (t = await t), !t)) break;
          const r = rr(t);
          if (!r || !!!(r.headerType & 2)) break;
          (this.bitstreams.push({
            serialNumber: r.serialNumber,
            bosPage: r,
            description: null,
            numberOfChannels: -1,
            sampleRate: -1,
            codecInfo: { codec: null, vorbisInfo: null, opusInfo: null },
            lastMetadataPacket: null,
          }),
            (e = r.headerStartPos + r.totalSize));
        }
        for (const t of this.bitstreams) {
          const r = await this.readPacket(t.bosPage, 0);
          r &&
            (r.data.byteLength >= 7 &&
            r.data[0] === 1 &&
            r.data[1] === 118 &&
            r.data[2] === 111 &&
            r.data[3] === 114 &&
            r.data[4] === 98 &&
            r.data[5] === 105 &&
            r.data[6] === 115
              ? await this.readVorbisMetadata(r, t)
              : r.data.byteLength >= 8 &&
                r.data[0] === 79 &&
                r.data[1] === 112 &&
                r.data[2] === 117 &&
                r.data[3] === 115 &&
                r.data[4] === 72 &&
                r.data[5] === 101 &&
                r.data[6] === 97 &&
                r.data[7] === 100 &&
                (await this.readOpusMetadata(r, t)),
            t.codecInfo.codec !== null &&
              this.trackBackings.push(new Vu(t, this)));
        }
      })());
    }
    async readVorbisMetadata(e, t) {
      let r = await this.findNextPacketStart(e);
      if (!r) return;
      const n = await this.readPacket(r.startPage, r.startSegmentIndex);
      if (!n || ((r = await this.findNextPacketStart(n)), !r)) return;
      const s = await this.readPacket(r.startPage, r.startSegmentIndex);
      if (!s || n.data[0] !== 3 || s.data[0] !== 5) return;
      const a = [],
        o = (d) => {
          for (; a.push(Math.min(255, d)), !(d < 255);) d -= 255;
        };
      (o(e.data.length), o(n.data.length));
      const c = new Uint8Array(
        1 + a.length + e.data.length + n.data.length + s.data.length,
      );
      ((c[0] = 2),
        c.set(a, 1),
        c.set(e.data, 1 + a.length),
        c.set(n.data, 1 + a.length + e.data.length),
        c.set(s.data, 1 + a.length + e.data.length + n.data.length),
        (t.codecInfo.codec = "vorbis"),
        (t.description = c),
        (t.lastMetadataPacket = s));
      const l = j(e.data);
      ((t.numberOfChannels = l.getUint8(11)),
        (t.sampleRate = l.getUint32(12, !0)));
      const u = l.getUint8(28);
      ((t.codecInfo.vorbisInfo = {
        blocksizes: [1 << (u & 15), 1 << (u >> 4)],
        modeBlockflags: zl(s.data).modeBlockflags,
      }),
        zn(n.data.subarray(7), this.metadataTags));
    }
    async readOpusMetadata(e, t) {
      const r = await this.findNextPacketStart(e);
      if (!r) return;
      const n = await this.readPacket(r.startPage, r.startSegmentIndex);
      if (!n) return;
      ((t.codecInfo.codec = "opus"),
        (t.description = e.data),
        (t.lastMetadataPacket = n));
      const s = On(e.data);
      ((t.numberOfChannels = s.outputChannelCount),
        (t.sampleRate = Gi),
        (t.codecInfo.opusInfo = { preSkip: s.preSkip }),
        zn(n.data.subarray(8), this.metadataTags));
    }
    async readPacket(e, t) {
      g(t < e.lacingValues.length);
      let r = 0;
      for (let d = 0; d < t; d++) r += e.lacingValues[d];
      let n = e,
        s = r,
        a = t;
      const o = [];
      e: for (;;) {
        let d = this.reader.requestSlice(n.dataStartPos, n.dataSize);
        (A(d) && (d = await d), g(d));
        const f = N(d, n.dataSize);
        for (;;) {
          if (a === n.lacingValues.length) {
            o.push(f.subarray(r, s));
            break;
          }
          const p = n.lacingValues[a];
          if (((s += p), p < 255)) {
            o.push(f.subarray(r, s));
            break e;
          }
          a++;
        }
        let h = n.headerStartPos + n.totalSize;
        for (;;) {
          let p = this.reader.requestSliceRange(h, Dt, xi);
          if ((A(p) && (p = await p), !p)) return null;
          const m = rr(p);
          if (!m) return null;
          if (((n = m), n.serialNumber === e.serialNumber)) break;
          h = n.headerStartPos + n.totalSize;
        }
        ((r = 0), (s = 0), (a = 0));
      }
      const c = o.reduce((d, f) => d + f.length, 0);
      if (c === 0) return null;
      const l = new Uint8Array(c);
      let u = 0;
      for (let d = 0; d < o.length; d++) {
        const f = o[d];
        (l.set(f, u), (u += f.length));
      }
      return { data: l, endPage: n, endSegmentIndex: a };
    }
    async findNextPacketStart(e) {
      if (e.endSegmentIndex < e.endPage.lacingValues.length - 1)
        return {
          startPage: e.endPage,
          startSegmentIndex: e.endSegmentIndex + 1,
        };
      if (!!(e.endPage.headerType & 4)) return null;
      let r = e.endPage.headerStartPos + e.endPage.totalSize;
      for (;;) {
        let n = this.reader.requestSliceRange(r, Dt, xi);
        if ((A(n) && (n = await n), !n)) return null;
        const s = rr(n);
        if (!s) return null;
        if (s.serialNumber === e.endPage.serialNumber)
          return { startPage: s, startSegmentIndex: 0 };
        r = s.headerStartPos + s.totalSize;
      }
    }
    async getMimeType() {
      await this.readMetadata();
      const e = await Promise.all(
        this.trackBackings.map((t) =>
          t.getDecoderConfig().then((r) => r?.codec ?? null),
        ),
      );
      return zu({ codecStrings: e.filter(Boolean) });
    }
    async getTrackBackings() {
      return (await this.readMetadata(), this.trackBackings);
    }
    async getMetadataTags() {
      return (await this.readMetadata(), this.metadataTags);
    }
  }
  class Vu {
    constructor(e, t) {
      ((this.bitstream = e),
        (this.demuxer = t),
        (this.encodedPacketToMetadata = new WeakMap()),
        (this.sequentialScanCache = []),
        (this.sequentialScanMutex = new ci()),
        (this.firstTimestampInSamplesPromise = null),
        (this.internalSampleRate =
          e.codecInfo.codec === "opus" ? Gi : e.sampleRate));
    }
    getType() {
      return "audio";
    }
    getId() {
      return this.bitstream.serialNumber;
    }
    getNumber() {
      const e = this.demuxer.trackBackings.findIndex(
        (t) => t.bitstream === this.bitstream,
      );
      return (g(e !== -1), e + 1);
    }
    getNumberOfChannels() {
      return this.bitstream.numberOfChannels;
    }
    getSampleRate() {
      return this.bitstream.sampleRate;
    }
    getTimeResolution() {
      return this.bitstream.sampleRate;
    }
    isRelativeToUnixEpoch() {
      return !1;
    }
    getUnixTimeForTimestamp() {
      return null;
    }
    getPairingMask() {
      return 1n;
    }
    getBitrate() {
      return null;
    }
    getAverageBitrate() {
      return null;
    }
    async getDurationFromMetadata() {
      return null;
    }
    async getLiveRefreshInterval() {
      return null;
    }
    getCodec() {
      return this.bitstream.codecInfo.codec;
    }
    getInternalCodecId() {
      return null;
    }
    async getDecoderConfig() {
      return (
        g(this.bitstream.codecInfo.codec),
        {
          codec: this.bitstream.codecInfo.codec,
          numberOfChannels: this.bitstream.numberOfChannels,
          sampleRate: this.bitstream.sampleRate,
          description: this.bitstream.description ?? void 0,
        }
      );
    }
    getName() {
      return null;
    }
    getLanguageCode() {
      return Pe;
    }
    getDisposition() {
      return { ...wt, primary: !1 };
    }
    granulePositionToTimestampInSamples(e) {
      return this.bitstream.codecInfo.codec === "opus"
        ? (g(this.bitstream.codecInfo.opusInfo),
          e - this.bitstream.codecInfo.opusInfo.preSkip)
        : e;
    }
    createEncodedPacketFromOggPacket(e, t, r) {
      if (!e) return null;
      const n = bo(e.data, this.bitstream.codecInfo, t.vorbisLastBlocksize),
        { vorbisBlockSize: s } = n;
      let { durationInSamples: a } = n;
      if (
        !!(e.endPage.headerType & 4) &&
        e.endSegmentIndex === e.endPage.lacingValues.length - 1
      ) {
        const d = this.granulePositionToTimestampInSamples(
          e.endPage.granulePosition,
        );
        a = se(d - t.timestampInSamples, 0, a);
      }
      const c = Math.max(t.timestampInSamples, 0),
        l = Math.max(t.timestampInSamples + a, 0),
        u = new Z(
          r.metadataOnly ? Ue : e.data,
          "key",
          c / this.internalSampleRate,
          (l - c) / this.internalSampleRate,
          e.endPage.headerStartPos + e.endSegmentIndex,
          e.data.byteLength,
        );
      return (
        this.encodedPacketToMetadata.set(u, {
          packet: e,
          timestampInSamples: t.timestampInSamples,
          durationInSamples: a,
          vorbisLastBlockSize: t.vorbisLastBlocksize,
          vorbisBlockSize: s,
        }),
        u
      );
    }
    getFirstTimestampInSamples() {
      return (this.firstTimestampInSamplesPromise ??= (async () => {
        const e = this.granulePositionToTimestampInSamples(0);
        g(this.bitstream.lastMetadataPacket);
        let t = await this.demuxer.findNextPacketStart(
          this.bitstream.lastMetadataPacket,
        );
        if (!t) return e;
        let r = null,
          n = 0,
          s = null;
        for (; t;) {
          const a = await this.demuxer.readPacket(
            t.startPage,
            t.startSegmentIndex,
          );
          if (
            !a ||
            ((r ??= a.endPage), a.endPage.headerStartPos !== r.headerStartPos)
          )
            break;
          const { durationInSamples: o, vorbisBlockSize: c } = bo(
            a.data,
            this.bitstream.codecInfo,
            s,
          );
          ((n += o), (s = c), (t = await this.demuxer.findNextPacketStart(a)));
        }
        return r ? e + Math.max(r.granulePosition - n, 0) : e;
      })());
    }
    async getFirstPacket(e) {
      g(this.bitstream.lastMetadataPacket);
      const t = await this.demuxer.findNextPacketStart(
        this.bitstream.lastMetadataPacket,
      );
      if (!t) return null;
      const r = await this.getFirstTimestampInSamples(),
        n = await this.demuxer.readPacket(t.startPage, t.startSegmentIndex);
      return this.createEncodedPacketFromOggPacket(
        n,
        { timestampInSamples: r, vorbisLastBlocksize: null },
        e,
      );
    }
    async getNextPacket(e, t) {
      const r = this.encodedPacketToMetadata.get(e);
      if (!r) throw new Error("Packet was not created from this track.");
      const n = await this.demuxer.findNextPacketStart(r.packet);
      if (!n) return null;
      const s = r.timestampInSamples + r.durationInSamples,
        a = await this.demuxer.readPacket(n.startPage, n.startSegmentIndex);
      return this.createEncodedPacketFromOggPacket(
        a,
        { timestampInSamples: s, vorbisLastBlocksize: r.vorbisBlockSize },
        t,
      );
    }
    async getPacket(e, t) {
      if (this.demuxer.reader.fileSize === null)
        return this.getPacketSequential(e, t);
      const r = zi(e * this.internalSampleRate),
        n = await this.getFirstTimestampInSamples();
      if (r === Math.max(n, 0)) return this.getFirstPacket(t);
      if (r < Math.max(n, 0)) return null;
      g(this.bitstream.lastMetadataPacket);
      const s = await this.demuxer.findNextPacketStart(
        this.bitstream.lastMetadataPacket,
      );
      if (!s) return null;
      let a = s.startPage,
        o = this.demuxer.reader.fileSize;
      const c = [a];
      e: for (; a.headerStartPos + a.totalSize < o;) {
        const T = a.headerStartPos,
          S = Math.floor((T + o) / 2);
        let k = S;
        for (;;) {
          const P = Math.min(k + Nu, o - Dt);
          let I = this.demuxer.reader.requestSlice(k, P - k);
          if ((A(I) && (I = await I), g(I), !Uu(I, P))) {
            if (o <= S + Dt) break e;
            o = S + Dt;
            continue e;
          }
          let C = this.demuxer.reader.requestSliceRange(I.filePos, Dt, xi);
          (A(C) && (C = await C), g(C));
          const _ = rr(C);
          g(_);
          let v = !1;
          if (_.serialNumber === this.bitstream.serialNumber) v = !0;
          else {
            let D = this.demuxer.reader.requestSlice(
              _.headerStartPos,
              _.totalSize,
            );
            (A(D) && (D = await D), g(D));
            const z = N(D, _.totalSize);
            v = Ou(z) === _.checksum;
          }
          if (!v) {
            k = _.headerStartPos + 4;
            continue;
          }
          if (v && _.serialNumber !== this.bitstream.serialNumber) {
            k = _.headerStartPos + _.totalSize;
            continue;
          }
          if (_.granulePosition === -1) {
            k = _.headerStartPos + _.totalSize;
            continue;
          }
          this.granulePositionToTimestampInSamples(_.granulePosition) > r
            ? (o = _.headerStartPos)
            : ((a = _), c.push(_));
          continue e;
        }
      }
      let l = s.startPage;
      for (const T of c) {
        if (T.granulePosition === a.granulePosition) break;
        (!l || T.headerStartPos > l.headerStartPos) && (l = T);
      }
      let u = l;
      const d = [u];
      for (
        ;
        !(
          u.serialNumber === this.bitstream.serialNumber &&
          u.granulePosition === a.granulePosition
        );
      ) {
        const T = u.headerStartPos + u.totalSize;
        let S = this.demuxer.reader.requestSliceRange(T, Dt, xi);
        (A(S) && (S = await S), g(S));
        const k = rr(S);
        (g(k),
          (u = k),
          u.serialNumber === this.bitstream.serialNumber && d.push(u));
      }
      g(u.granulePosition !== -1);
      let f = null,
        h,
        p,
        m = u,
        y = 0;
      if (u.headerStartPos === s.startPage.headerStartPos)
        ((h = n), (p = !0), (f = 0));
      else {
        ((h = 0), (p = !1));
        for (let k = u.lacingValues.length - 1; k >= 0; k--)
          if (u.lacingValues[k] < 255) {
            f = k + 1;
            break;
          }
        if (f === null)
          throw new Error(
            "Invalid page with granule position: no packets end on this page.",
          );
        y = f - 1;
        const T = { data: Ue, endPage: m, endSegmentIndex: y };
        if (await this.demuxer.findNextPacketStart(T)) {
          const k = To(d, u, f);
          g(k);
          const P = ko(d, k.page, k.segmentIndex);
          P && ((u = P.page), (f = P.segmentIndex));
        } else
          for (;;) {
            const k = To(d, u, f);
            if (!k) break;
            const P = ko(d, k.page, k.segmentIndex);
            if (!P) break;
            if (
              ((u = P.page),
              (f = P.segmentIndex),
              k.page.headerStartPos !== m.headerStartPos)
            ) {
              ((m = k.page), (y = k.segmentIndex));
              break;
            }
          }
      }
      let w = null,
        b = null;
      for (; u !== null;) {
        g(f !== null);
        const T = await this.demuxer.readPacket(u, f);
        if (!T) break;
        if (!(
          u.headerStartPos === s.startPage.headerStartPos &&
          f < s.startSegmentIndex
        )) {
          let P = this.createEncodedPacketFromOggPacket(
            T,
            {
              timestampInSamples: h,
              vorbisLastBlocksize: b?.vorbisBlockSize ?? null,
            },
            t,
          );
          g(P);
          let I = this.encodedPacketToMetadata.get(P);
          if (
            (g(I),
            !p &&
            T.endPage.headerStartPos === m.headerStartPos &&
            T.endSegmentIndex === y
              ? ((h = this.granulePositionToTimestampInSamples(
                  u.granulePosition,
                )),
                (p = !0),
                (P = this.createEncodedPacketFromOggPacket(
                  T,
                  {
                    timestampInSamples: h - I.durationInSamples,
                    vorbisLastBlocksize: b?.vorbisBlockSize ?? null,
                  },
                  t,
                )),
                g(P),
                (I = this.encodedPacketToMetadata.get(P)),
                g(I))
              : (h += I.durationInSamples),
            (w = P),
            (b = I),
            p &&
              (Math.max(h, 0) > r || Math.max(I.timestampInSamples, 0) === r))
          )
            break;
        }
        const k = await this.demuxer.findNextPacketStart(T);
        if (!k) break;
        ((u = k.startPage), (f = k.startSegmentIndex));
      }
      return w;
    }
    async getPacketSequential(e, t) {
      const r = await this.sequentialScanMutex.acquire();
      try {
        const n = zi(e * this.internalSampleRate);
        e = n / this.internalSampleRate;
        const s = X(this.sequentialScanCache, n, (c) => c.timestampInSamples);
        let a;
        if (s !== -1) {
          const c = this.sequentialScanCache[s];
          a = this.createEncodedPacketFromOggPacket(
            c.packet,
            {
              timestampInSamples: c.timestampInSamples,
              vorbisLastBlocksize: c.vorbisLastBlockSize,
            },
            t,
          );
        } else if (((a = await this.getFirstPacket(t)), a && a.timestamp > e))
          return null;
        let o = 0;
        for (; a && a.timestamp < e;) {
          const c = await this.getNextPacket(a, t);
          if (!c || c.timestamp > e) break;
          if (((a = c), o++, o === 100)) {
            o = 0;
            const l = this.encodedPacketToMetadata.get(a);
            (g(l),
              this.sequentialScanCache.length > 0 &&
                g(
                  ne(this.sequentialScanCache).timestampInSamples <=
                    l.timestampInSamples,
                ),
              this.sequentialScanCache.push(l));
          }
        }
        return a;
      } finally {
        r();
      }
    }
    getKeyPacket(e, t) {
      return this.getPacket(e, t);
    }
    getNextKeyPacket(e, t) {
      return this.getNextPacket(e, t);
    }
  }
  const ko = (i, e, t) => {
      let r = e,
        n = t;
      e: for (;;) {
        for (n--, n; n >= 0; n--)
          if (r.lacingValues[n] < 255) {
            n++;
            break e;
          }
        if ((g(n === -1), !(r.headerType & 1))) {
          n = 0;
          break;
        }
        const a = Js(i, (o) => o.headerStartPos < r.headerStartPos);
        if (!a) return null;
        ((r = a), (n = r.lacingValues.length));
      }
      if ((g(n !== -1), n === r.lacingValues.length)) {
        const s = i[i.indexOf(r) + 1];
        (g(s), (r = s), (n = 0));
      }
      return { page: r, segmentIndex: n };
    },
    To = (i, e, t) => {
      if (t > 0) return { page: e, segmentIndex: t - 1 };
      const r = Js(i, (n) => n.headerStartPos < e.headerStartPos);
      return r ? { page: r, segmentIndex: r.lacingValues.length - 1 } : null;
    };
  var ge;
  (function (i) {
    ((i[(i.PCM = 1)] = "PCM"),
      (i[(i.IEEE_FLOAT = 3)] = "IEEE_FLOAT"),
      (i[(i.ALAW = 6)] = "ALAW"),
      (i[(i.MULAW = 7)] = "MULAW"),
      (i[(i.EXTENSIBLE = 65534)] = "EXTENSIBLE"));
  })(ge || (ge = {}));
  class Lu extends bt {
    constructor(e) {
      (super(e),
        (this.metadataPromise = null),
        (this.dataStart = -1),
        (this.dataSize = -1),
        (this.audioInfo = null),
        (this.trackBackings = []),
        (this.lastKnownPacketIndex = 0),
        (this.metadataTags = {}),
        (this.reader = e._reader));
    }
    async readMetadata() {
      return (this.metadataPromise ??= (async () => {
        let e = this.reader.requestSlice(0, 12);
        (A(e) && (e = await e), g(e));
        const t = le(e, 4),
          r = t !== "RIFX",
          n = t === "RF64",
          s = Nt(e, r);
        let a = n
          ? this.reader.fileSize
          : Math.min(s + 8, this.reader.fileSize ?? 1 / 0);
        if (le(e, 4) !== "WAVE")
          throw new Error("Invalid WAVE file - wrong format");
        let c = 0,
          l = null,
          u = e.filePos;
        for (; a === null || u < a;) {
          let f = this.reader.requestSlice(u, 8);
          if ((A(f) && (f = await f), !f)) break;
          const h = le(f, 4),
            p = Nt(f, r),
            m = f.filePos;
          if (n && c === 0 && h !== "ds64")
            throw new Error('Invalid RF64 file: First chunk must be "ds64".');
          if (h === "fmt ") await this.parseFmtChunk(m, p, r);
          else if (h === "data") {
            if (
              ((l ??= p),
              (this.dataStart = f.filePos),
              (this.dataSize = Math.min(l, (a ?? 1 / 0) - this.dataStart)),
              this.reader.fileSize === null)
            )
              break;
          } else if (h === "ds64") {
            let y = this.reader.requestSlice(m, p);
            if ((A(y) && (y = await y), !y)) break;
            const w = Ac(y, r);
            ((l = Ac(y, r)),
              (a = Math.min(w + 8, this.reader.fileSize ?? 1 / 0)));
          } else
            h === "LIST"
              ? await this.parseListChunk(m, p, r)
              : (h === "ID3 " || h === "id3 ") &&
                (await this.parseId3Chunk(m, p));
          ((u = m + p + (p & 1)), c++);
        }
        if (!this.audioInfo)
          throw new Error('Invalid WAVE file - missing "fmt " chunk');
        if (this.dataStart === -1)
          throw new Error('Invalid WAVE file - missing "data" chunk');
        const d = this.audioInfo.blockSizeInBytes;
        ((this.dataSize = Math.floor(this.dataSize / d) * d),
          this.trackBackings.push(new Hu(this)));
      })());
    }
    async parseFmtChunk(e, t, r) {
      let n = this.reader.requestSlice(e, t);
      if ((A(n) && (n = await n), !n)) return;
      let s = pr(n, r);
      const a = pr(n, r),
        o = Nt(n, r);
      n.skip(4);
      const c = pr(n, r);
      let l;
      if ((t === 14 ? (l = 8) : (l = pr(n, r)), t >= 18 && s !== 357)) {
        const u = pr(n, r),
          d = t - 18;
        if (Math.min(d, u) >= 22 && s === ge.EXTENSIBLE) {
          n.skip(6);
          const h = N(n, 16);
          s = h[0] | (h[1] << 8);
        }
      }
      if (
        ((s === ge.MULAW || s === ge.ALAW) && (l = 8),
        s !== ge.PCM && s !== ge.IEEE_FLOAT && s !== ge.ALAW && s !== ge.MULAW)
      )
        throw new Error(
          `Unsupported WAVE codec (format tag ${s}). Only integer/float PCM, A-law, and μ-law are supported.`,
        );
      if (s === ge.PCM && ![8, 16, 24, 32].includes(l))
        throw new Error(
          `Unsupported WAVE PCM bit depth (${l}). Only 8, 16, 24, and 32 bits are supported.`,
        );
      if (s === ge.IEEE_FLOAT && ![32, 64].includes(l))
        throw new Error(
          `Unsupported WAVE float bit depth (${l}). Only 32 and 64 bits are supported.`,
        );
      this.audioInfo = {
        format: s,
        numberOfChannels: a,
        sampleRate: o,
        sampleSizeInBytes: Math.ceil(l / 8),
        blockSizeInBytes: c,
      };
    }
    async parseListChunk(e, t, r) {
      let n = this.reader.requestSlice(e, t);
      if ((A(n) && (n = await n), !n)) return;
      const s = le(n, 4);
      if (s !== "INFO" && s !== "INF0") return;
      let a = n.filePos;
      for (; a <= e + t - 8;) {
        n.filePos = a;
        const o = le(n, 4),
          c = Nt(n, r),
          l = N(n, c);
        let u = 0;
        for (let f = 0; f < l.length && l[f] !== 0; f++) u++;
        const d = String.fromCharCode(...l.subarray(0, u));
        switch (
          ((this.metadataTags.raw ??= {}), (this.metadataTags.raw[o] = d), o)
        ) {
          case "INAM":
          case "TITL":
            this.metadataTags.title ??= d;
            break;
          case "TIT3":
            this.metadataTags.description ??= d;
            break;
          case "IART":
            this.metadataTags.artist ??= d;
            break;
          case "IPRD":
            this.metadataTags.album ??= d;
            break;
          case "IPRT":
          case "ITRK":
          case "TRCK":
            {
              const f = d.split("/"),
                h = Number.parseInt(f[0], 10),
                p = f[1] && Number.parseInt(f[1], 10);
              (Number.isInteger(h) &&
                h > 0 &&
                (this.metadataTags.trackNumber ??= h),
                p &&
                  Number.isInteger(p) &&
                  p > 0 &&
                  (this.metadataTags.tracksTotal ??= p));
            }
            break;
          case "ICRD":
          case "IDIT":
            {
              const f = new Date(d);
              Number.isNaN(f.getTime()) || (this.metadataTags.date ??= f);
            }
            break;
          case "YEAR":
            {
              const f = Number.parseInt(d, 10);
              Number.isInteger(f) &&
                f > 0 &&
                (this.metadataTags.date ??= new Date(f, 0, 1));
            }
            break;
          case "IGNR":
          case "GENR":
            this.metadataTags.genre ??= d;
            break;
          case "ICMT":
          case "CMNT":
          case "COMM":
            this.metadataTags.comment ??= d;
            break;
        }
        a += 8 + c + (c & 1);
      }
    }
    async parseId3Chunk(e, t) {
      let r = this.reader.requestSlice(e, t);
      if ((A(r) && (r = await r), !r)) return;
      const n = Et(r);
      if (n) {
        const s = t - je;
        if (((n.size = Math.min(n.size, s)), n.size > 0)) {
          const a = r.slice(e + je, n.size);
          Gr(a, n, this.metadataTags);
        }
      }
    }
    getCodec() {
      if ((g(this.audioInfo), this.audioInfo.format === ge.MULAW))
        return "ulaw";
      if (this.audioInfo.format === ge.ALAW) return "alaw";
      if (this.audioInfo.format === ge.PCM) {
        if (this.audioInfo.sampleSizeInBytes === 1) return "pcm-u8";
        if (this.audioInfo.sampleSizeInBytes === 2) return "pcm-s16";
        if (this.audioInfo.sampleSizeInBytes === 3) return "pcm-s24";
        if (this.audioInfo.sampleSizeInBytes === 4) return "pcm-s32";
      }
      if (this.audioInfo.format === ge.IEEE_FLOAT) {
        if (this.audioInfo.sampleSizeInBytes === 4) return "pcm-f32";
        if (this.audioInfo.sampleSizeInBytes === 8) return "pcm-f64";
      }
      g(!1);
    }
    async getMimeType() {
      return "audio/wav";
    }
    async getTrackBackings() {
      return (await this.readMetadata(), this.trackBackings);
    }
    async getMetadataTags() {
      return (await this.readMetadata(), this.metadataTags);
    }
  }
  const Pi = 2048;
  class Hu {
    constructor(e) {
      this.demuxer = e;
    }
    getType() {
      return "audio";
    }
    getId() {
      return 1;
    }
    getNumber() {
      return 1;
    }
    getCodec() {
      return this.demuxer.getCodec();
    }
    getInternalCodecId() {
      return (g(this.demuxer.audioInfo), this.demuxer.audioInfo.format);
    }
    async getDecoderConfig() {
      const e = this.demuxer.getCodec();
      return e
        ? (g(this.demuxer.audioInfo),
          {
            codec: e,
            numberOfChannels: this.demuxer.audioInfo.numberOfChannels,
            sampleRate: this.demuxer.audioInfo.sampleRate,
          })
        : null;
    }
    getNumberOfChannels() {
      return (
        g(this.demuxer.audioInfo),
        this.demuxer.audioInfo.numberOfChannels
      );
    }
    getSampleRate() {
      return (g(this.demuxer.audioInfo), this.demuxer.audioInfo.sampleRate);
    }
    getTimeResolution() {
      return (g(this.demuxer.audioInfo), this.demuxer.audioInfo.sampleRate);
    }
    isRelativeToUnixEpoch() {
      return !1;
    }
    getUnixTimeForTimestamp() {
      return null;
    }
    getPairingMask() {
      return 1n;
    }
    getBitrate() {
      return null;
    }
    getAverageBitrate() {
      return null;
    }
    async getDurationFromMetadata() {
      return (
        g(this.demuxer.dataSize !== -1),
        this.demuxer.dataSize /
          this.demuxer.audioInfo.blockSizeInBytes /
          this.demuxer.audioInfo.sampleRate
      );
    }
    async getLiveRefreshInterval() {
      return null;
    }
    getName() {
      return null;
    }
    getLanguageCode() {
      return Pe;
    }
    getDisposition() {
      return { ...wt };
    }
    async getPacketAtIndex(e, t) {
      (g(e >= 0), g(this.demuxer.audioInfo));
      const r = e * Pi * this.demuxer.audioInfo.blockSizeInBytes;
      if (r >= this.demuxer.dataSize) return null;
      const n = Math.min(
        Pi * this.demuxer.audioInfo.blockSizeInBytes,
        this.demuxer.dataSize - r,
      );
      if (this.demuxer.reader.fileSize === null) {
        let c = this.demuxer.reader.requestSlice(this.demuxer.dataStart + r, n);
        if ((A(c) && (c = await c), !c)) return null;
      }
      let s;
      if (t.metadataOnly) s = Ue;
      else {
        let c = this.demuxer.reader.requestSlice(this.demuxer.dataStart + r, n);
        (A(c) && (c = await c), g(c), (s = N(c, n)));
      }
      const a = (e * Pi) / this.demuxer.audioInfo.sampleRate,
        o =
          n /
          this.demuxer.audioInfo.blockSizeInBytes /
          this.demuxer.audioInfo.sampleRate;
      return (
        (this.demuxer.lastKnownPacketIndex = Math.max(
          e,
          this.demuxer.lastKnownPacketIndex,
        )),
        new Z(s, "key", a, o, e, n)
      );
    }
    getFirstPacket(e) {
      return this.getPacketAtIndex(0, e);
    }
    async getPacket(e, t) {
      g(this.demuxer.audioInfo);
      const r = Math.floor(
        Math.min(
          (e * this.demuxer.audioInfo.sampleRate) / Pi,
          (this.demuxer.dataSize - 1) /
            (Pi * this.demuxer.audioInfo.blockSizeInBytes),
        ),
      );
      if (r < 0) return null;
      const n = await this.getPacketAtIndex(r, t);
      if (n) return n;
      if (r === 0) return null;
      g(this.demuxer.reader.fileSize === null);
      let s = await this.getPacketAtIndex(this.demuxer.lastKnownPacketIndex, t);
      for (; s;) {
        const a = await this.getNextPacket(s, t);
        if (!a) break;
        s = a;
      }
      return s;
    }
    getNextPacket(e, t) {
      g(this.demuxer.audioInfo);
      const r = Math.round(
        (e.timestamp * this.demuxer.audioInfo.sampleRate) / Pi,
      );
      return this.getPacketAtIndex(r + 1, t);
    }
    getKeyPacket(e, t) {
      return this.getPacket(e, t);
    }
    getNextKeyPacket(e, t) {
      return this.getNextPacket(e, t);
    }
  }
  const nr = 7,
    Ot = 9,
    Pt = (i) => {
      const e = i.filePos,
        t = N(i, 9),
        r = new $(t);
      if (r.readBits(12) !== 4095 || (r.skipBits(1), r.readBits(2) !== 0))
        return null;
      const a = r.readBits(1),
        o = r.readBits(2) + 1,
        c = r.readBits(4);
      if (c === 15) return null;
      r.skipBits(1);
      const l = r.readBits(3);
      if (l === 0)
        throw new Error(
          "ADTS frames with channel configuration 0 are not supported.",
        );
      (r.skipBits(1), r.skipBits(1), r.skipBits(1), r.skipBits(1));
      const u = r.readBits(13);
      r.skipBits(11);
      const d = r.readBits(2) + 1;
      if (d !== 1)
        throw new Error(
          "ADTS frames with more than one AAC frame are not supported.",
        );
      let f = null;
      return (
        a === 1 ? (i.filePos -= 2) : (f = r.readBits(16)),
        {
          objectType: o,
          samplingFrequencyIndex: c,
          channelConfiguration: l,
          frameLength: u,
          numberOfAacFrames: d,
          crcCheck: f,
          startPos: e,
        }
      );
    };
  const Ur = 1024;
  class qu extends bt {
    constructor(e) {
      (super(e),
        (this.metadataPromise = null),
        (this.firstFrameHeader = null),
        (this.loadedSamples = []),
        (this.metadataTags = null),
        (this.trackBackings = []),
        (this.readingMutex = new ci()),
        (this.lastSampleLoaded = !1),
        (this.lastLoadedPos = 0),
        (this.nextTimestampInSamples = 0),
        (this.reader = e._reader));
    }
    async readMetadata() {
      return (this.metadataPromise ??= (async () => {
        for (; !this.firstFrameHeader && !this.lastSampleLoaded;)
          await this.advanceReader();
        (g(this.firstFrameHeader), (this.trackBackings = [new ju(this)]));
      })());
    }
    async advanceReader() {
      if (this.lastLoadedPos === 0)
        for (;;) {
          let a = this.reader.requestSlice(this.lastLoadedPos, je);
          if ((A(a) && (a = await a), !a)) {
            this.lastSampleLoaded = !0;
            return;
          }
          const o = Et(a);
          if (!o) break;
          this.lastLoadedPos = a.filePos + o.size;
        }
      let e = this.reader.requestSliceRange(this.lastLoadedPos, nr, Ot);
      if ((A(e) && (e = await e), !e)) {
        this.lastSampleLoaded = !0;
        return;
      }
      const t = Pt(e);
      if (!t) {
        this.lastSampleLoaded = !0;
        return;
      }
      if (
        this.reader.fileSize !== null &&
        t.startPos + t.frameLength > this.reader.fileSize
      ) {
        this.lastSampleLoaded = !0;
        return;
      }
      this.firstFrameHeader || (this.firstFrameHeader = t);
      const r = At[t.samplingFrequencyIndex];
      g(r !== void 0);
      const n = Ur / r,
        s = {
          timestamp: this.nextTimestampInSamples / r,
          duration: n,
          dataStart: t.startPos,
          dataSize: t.frameLength,
        };
      (this.loadedSamples.push(s),
        (this.nextTimestampInSamples += Ur),
        (this.lastLoadedPos = t.startPos + t.frameLength));
    }
    async getMimeType() {
      return "audio/aac";
    }
    async getTrackBackings() {
      return (await this.readMetadata(), this.trackBackings);
    }
    async getMetadataTags() {
      const e = await this.readingMutex.acquire();
      try {
        if ((await this.readMetadata(), this.metadataTags))
          return this.metadataTags;
        this.metadataTags = {};
        let t = 0;
        for (;;) {
          let r = this.reader.requestSlice(t, je);
          if ((A(r) && (r = await r), !r)) break;
          const n = Et(r);
          if (!n) break;
          let s = this.reader.requestSlice(r.filePos, n.size);
          if ((A(s) && (s = await s), !s)) break;
          (Gr(s, n, this.metadataTags), (t = r.filePos + n.size));
        }
        return this.metadataTags;
      } finally {
        e();
      }
    }
  }
  class ju {
    constructor(e) {
      this.demuxer = e;
    }
    getType() {
      return "audio";
    }
    getId() {
      return 1;
    }
    getNumber() {
      return 1;
    }
    getTimeResolution() {
      return this.getSampleRate() / Ur;
    }
    isRelativeToUnixEpoch() {
      return !1;
    }
    getUnixTimeForTimestamp() {
      return null;
    }
    getPairingMask() {
      return 1n;
    }
    getBitrate() {
      return null;
    }
    getAverageBitrate() {
      return null;
    }
    async getDurationFromMetadata() {
      return null;
    }
    async getLiveRefreshInterval() {
      return null;
    }
    getName() {
      return null;
    }
    getLanguageCode() {
      return Pe;
    }
    getCodec() {
      return "aac";
    }
    getInternalCodecId() {
      return (
        g(this.demuxer.firstFrameHeader),
        this.demuxer.firstFrameHeader.objectType
      );
    }
    getNumberOfChannels() {
      g(this.demuxer.firstFrameHeader);
      const e = mi[this.demuxer.firstFrameHeader.channelConfiguration];
      return (g(e !== void 0), e);
    }
    getSampleRate() {
      g(this.demuxer.firstFrameHeader);
      const e = At[this.demuxer.firstFrameHeader.samplingFrequencyIndex];
      return (g(e !== void 0), e);
    }
    getDisposition() {
      return { ...wt };
    }
    async getDecoderConfig() {
      return (
        g(this.demuxer.firstFrameHeader),
        {
          codec: `mp4a.40.${this.demuxer.firstFrameHeader.objectType}`,
          numberOfChannels: this.getNumberOfChannels(),
          sampleRate: this.getSampleRate(),
        }
      );
    }
    async getPacketAtIndex(e, t) {
      if (e === -1) return null;
      const r = this.demuxer.loadedSamples[e];
      if (!r) return null;
      let n;
      if (t.metadataOnly) n = Ue;
      else {
        let s = this.demuxer.reader.requestSlice(r.dataStart, r.dataSize);
        if ((A(s) && (s = await s), !s)) return null;
        n = N(s, r.dataSize);
      }
      return new Z(n, "key", r.timestamp, r.duration, e, r.dataSize);
    }
    getFirstPacket(e) {
      return this.getPacketAtIndex(0, e);
    }
    async getNextPacket(e, t) {
      const r = await this.demuxer.readingMutex.acquire();
      try {
        const n = Oi(
          this.demuxer.loadedSamples,
          e.timestamp,
          (a) => a.timestamp,
        );
        if (n === -1)
          throw new Error("Packet was not created from this track.");
        const s = n + 1;
        for (
          ;
          s >= this.demuxer.loadedSamples.length &&
          !this.demuxer.lastSampleLoaded;
        )
          await this.demuxer.advanceReader();
        return this.getPacketAtIndex(s, t);
      } finally {
        r();
      }
    }
    async getPacket(e, t) {
      const r = await this.demuxer.readingMutex.acquire();
      try {
        for (;;) {
          const n = X(this.demuxer.loadedSamples, e, (s) => s.timestamp);
          if (n === -1 && this.demuxer.loadedSamples.length > 0) return null;
          if (this.demuxer.lastSampleLoaded) return this.getPacketAtIndex(n, t);
          if (n >= 0 && n + 1 < this.demuxer.loadedSamples.length)
            return this.getPacketAtIndex(n, t);
          await this.demuxer.advanceReader();
        }
      } finally {
        r();
      }
    }
    getKeyPacket(e, t) {
      return this.getPacket(e, t);
    }
    getNextKeyPacket(e, t) {
      return this.getNextPacket(e, t);
    }
  }
  const $u = (i) =>
      i === 0
        ? null
        : i === 1
          ? 192
          : i >= 2 && i <= 5
            ? 144 * 2 ** i
            : i === 6
              ? "uncommon-u8"
              : i === 7
                ? "uncommon-u16"
                : i >= 8 && i <= 15
                  ? 2 ** i
                  : null,
    Ku = (i, e) => {
      switch (i) {
        case 0:
          return e;
        case 1:
          return 88200;
        case 2:
          return 176400;
        case 3:
          return 192e3;
        case 4:
          return 8e3;
        case 5:
          return 16e3;
        case 6:
          return 22050;
        case 7:
          return 24e3;
        case 8:
          return 32e3;
        case 9:
          return 44100;
        case 10:
          return 48e3;
        case 11:
          return 96e3;
        case 12:
          return "uncommon-u8";
        case 13:
          return "uncommon-u16";
        case 14:
          return "uncommon-u16-10";
        default:
          return null;
      }
    },
    Gu = (i) => {
      let e = 0;
      const t = new $(N(i, 1));
      for (; t.readBits(1) === 1;) e++;
      if (e === 0) return t.readBits(7);
      const r = [],
        n = e - 1,
        s = new $(N(i, n)),
        a = 8 - e - 1;
      for (let c = 0; c < a; c++) r.unshift(t.readBits(1));
      for (let c = 0; c < n; c++)
        for (let l = 0; l < 8; l++) {
          const u = s.readBits(1);
          l < 2 || r.unshift(u);
        }
      return r.reduce((c, l, u) => c | (l << u), 0);
    },
    Xu = (i, e) => {
      if (e === "uncommon-u16") return fe(i) + 1;
      if (e === "uncommon-u8") return U(i) + 1;
      if (typeof e == "number") return e;
      (xe(e), g(!1));
    },
    Qu = (i, e) =>
      e === "uncommon-u16"
        ? fe(i)
        : e === "uncommon-u16-10"
          ? fe(i) * 10
          : e === "uncommon-u8"
            ? U(i)
            : typeof e == "number"
              ? e
              : null,
    Yu = (i) => {
      let t = 0;
      for (const r of i) {
        t ^= r;
        for (let n = 0; n < 8; n++)
          ((t & 128) !== 0 ? (t = (t << 1) ^ 7) : (t <<= 1), (t &= 255));
      }
      return t;
    };
  class Zu extends bt {
    constructor(e) {
      (super(e),
        (this.loadedSamples = []),
        (this.metadataPromise = null),
        (this.trackBacking = null),
        (this.metadataTags = {}),
        (this.audioInfo = null),
        (this.lastLoadedPos = null),
        (this.blockingBit = null),
        (this.readingMutex = new ci()),
        (this.lastSampleLoaded = !1),
        (this.reader = e._reader));
    }
    async getMetadataTags() {
      return (await this.readMetadata(), this.metadataTags);
    }
    async getTrackBackings() {
      return (
        await this.readMetadata(),
        g(this.trackBacking),
        [this.trackBacking]
      );
    }
    async getMimeType() {
      return "audio/flac";
    }
    async readMetadata() {
      return (this.metadataPromise ??= (async () => {
        let e = 0;
        for (;;) {
          let t = this.reader.requestSlice(e, je);
          if ((A(t) && (t = await t), !t)) {
            this.lastSampleLoaded = !0;
            return;
          }
          const r = Et(t);
          if (!r) break;
          let n = this.reader.requestSlice(t.filePos, r.size);
          (A(n) && (n = await n),
            g(n),
            Gr(n, r, this.metadataTags),
            (e = t.filePos + r.size));
        }
        for (
          e += 4;
          this.reader.fileSize === null || e < this.reader.fileSize;
        ) {
          let t = this.reader.requestSlice(e, 4);
          if ((A(t) && (t = await t), (e += 4), t === null))
            throw new Error(
              `Metadata block at position ${e} is too small! Corrupted file.`,
            );
          g(t);
          const r = U(t),
            n = It(t),
            s = (r & 128) !== 0;
          switch (r & 127) {
            case wi.STREAMINFO: {
              let o = this.reader.requestSlice(e, n);
              if ((A(o) && (o = await o), g(o), o === null))
                throw new Error(
                  `StreamInfo block at position ${e} is too small! Corrupted file.`,
                );
              const c = N(o, 34),
                l = new $(c),
                u = l.readBits(16),
                d = l.readBits(16),
                f = l.readBits(24),
                h = l.readBits(24),
                p = l.readBits(20),
                m = l.readBits(3) + 1;
              l.readBits(5);
              const y = l.readBits(36);
              l.skipBits(128);
              const w = new Uint8Array(42);
              (w.set(new Uint8Array([102, 76, 97, 67]), 0),
                w.set(new Uint8Array([128, 0, 0, 34]), 4),
                w.set(c, 8),
                (this.audioInfo = {
                  numberOfChannels: m,
                  sampleRate: p,
                  totalSamples: y,
                  minimumBlockSize: u,
                  maximumBlockSize: d,
                  minimumFrameSize: f,
                  maximumFrameSize: h,
                  description: w,
                }),
                (this.trackBacking = new Ju(this)));
              break;
            }
            case wi.VORBIS_COMMENT: {
              let o = this.reader.requestSlice(e, n);
              (A(o) && (o = await o), g(o), zn(N(o, n), this.metadataTags));
              break;
            }
            case wi.PICTURE: {
              let o = this.reader.requestSlice(e, n);
              (A(o) && (o = await o), g(o));
              const c = R(o),
                l = R(o),
                u = Oe.decode(N(o, l)),
                d = R(o),
                f = Oe.decode(N(o, d));
              o.skip(16);
              const h = R(o),
                p = N(o, h);
              ((this.metadataTags.images ??= []),
                this.metadataTags.images.push({
                  data: p,
                  mimeType: u,
                  kind:
                    c === 3 ? "coverFront" : c === 4 ? "coverBack" : "unknown",
                  description: f,
                }));
              break;
            }
          }
          if (((e += n), s)) {
            this.lastLoadedPos = e;
            break;
          }
        }
        if (!this.audioInfo)
          throw new Error(
            "Missing STREAMINFO metadata block! Corrupted FLAC file.",
          );
      })());
    }
    async readNextFlacFrame({ startPos: e, isFirstPacket: t }) {
      g(this.audioInfo);
      const r = 6,
        n = 16,
        s = 10,
        a =
          this.audioInfo.maximumBlockSize *
            this.audioInfo.numberOfChannels *
            4 +
          n +
          2,
        o = this.audioInfo.minimumFrameSize || s,
        l = (this.audioInfo.maximumFrameSize || a) + n,
        u = await this.reader.requestSliceRange(e, n, l);
      if (!u) return null;
      const d = this.readFlacFrameHeader({ slice: u, isFirstPacket: t });
      if (!d) return null;
      for (u.filePos = e + o; ;) {
        if (u.filePos > u.end - r)
          return {
            num: d.num,
            blockSize: d.blockSize,
            sampleRate: d.sampleRate,
            size: u.end - e,
            isLastFrame: !0,
          };
        if (U(u) === 255) {
          const h = u.filePos,
            p = U(u),
            m = this.blockingBit === 1 ? 249 : 248;
          if (p !== m) {
            u.filePos = h;
            continue;
          }
          u.skip(-2);
          const y = u.filePos - e,
            w = this.readFlacFrameHeader({ slice: u, isFirstPacket: !1 });
          if (!w) {
            u.filePos = h;
            continue;
          }
          if (this.blockingBit === 0) {
            if (w.num - d.num !== 1) {
              u.filePos = h;
              continue;
            }
          } else if (w.num - d.num !== d.blockSize) {
            u.filePos = h;
            continue;
          }
          return {
            num: d.num,
            blockSize: d.blockSize,
            sampleRate: d.sampleRate,
            size: y,
            isLastFrame: !1,
          };
        }
      }
    }
    readFlacFrameHeader({ slice: e, isFirstPacket: t }) {
      const r = e.filePos,
        n = N(e, 4),
        s = new $(n);
      if (s.readBits(15) !== 32764) return null;
      if (this.blockingBit === null) {
        g(t);
        const y = s.readBits(1);
        this.blockingBit = y;
      } else if (this.blockingBit === 1) {
        if ((g(!t), s.readBits(1) !== 1)) return null;
      } else if (this.blockingBit === 0) {
        if ((g(!t), s.readBits(1) !== 0)) return null;
      } else throw new Error("Invalid blocking bit");
      const o = $u(s.readBits(4));
      if (!o) return null;
      g(this.audioInfo);
      const c = Ku(s.readBits(4), this.audioInfo.sampleRate);
      if (!c || (s.readBits(4), s.readBits(3), s.readBits(1) !== 0))
        return null;
      const u = Gu(e),
        d = Xu(e, o),
        f = Qu(e, c);
      if (f === null || f !== this.audioInfo.sampleRate) return null;
      const h = e.filePos - r,
        p = U(e);
      (e.skip(-h), e.skip(-1));
      const m = Yu(N(e, h));
      return p !== m ? null : { num: u, blockSize: d, sampleRate: f };
    }
    async advanceReader() {
      (await this.readMetadata(),
        g(this.lastLoadedPos !== null),
        g(this.audioInfo));
      const e = this.lastLoadedPos,
        t = await this.readNextFlacFrame({
          startPos: e,
          isFirstPacket: this.loadedSamples.length === 0,
        });
      if (!t) {
        this.lastSampleLoaded = !0;
        return;
      }
      const r = this.loadedSamples[this.loadedSamples.length - 1],
        s = {
          blockOffset: r ? r.blockOffset + r.blockSize : 0,
          blockSize: t.blockSize,
          byteOffset: e,
          byteSize: t.size,
        };
      if (
        ((this.lastLoadedPos = this.lastLoadedPos + t.size),
        this.loadedSamples.push(s),
        t.isLastFrame)
      ) {
        this.lastSampleLoaded = !0;
        return;
      }
    }
  }
  class Ju {
    constructor(e) {
      this.demuxer = e;
    }
    getType() {
      return "audio";
    }
    getId() {
      return 1;
    }
    getNumber() {
      return 1;
    }
    getCodec() {
      return "flac";
    }
    getInternalCodecId() {
      return null;
    }
    getNumberOfChannels() {
      return (
        g(this.demuxer.audioInfo),
        this.demuxer.audioInfo.numberOfChannels
      );
    }
    getSampleRate() {
      return (g(this.demuxer.audioInfo), this.demuxer.audioInfo.sampleRate);
    }
    getName() {
      return null;
    }
    getLanguageCode() {
      return Pe;
    }
    getTimeResolution() {
      return (g(this.demuxer.audioInfo), this.demuxer.audioInfo.sampleRate);
    }
    isRelativeToUnixEpoch() {
      return !1;
    }
    getUnixTimeForTimestamp() {
      return null;
    }
    getPairingMask() {
      return 1n;
    }
    getBitrate() {
      return null;
    }
    getAverageBitrate() {
      return null;
    }
    async getDurationFromMetadata() {
      return (
        g(this.demuxer.audioInfo),
        this.demuxer.audioInfo.totalSamples === 0
          ? null
          : this.demuxer.audioInfo.totalSamples /
            this.demuxer.audioInfo.sampleRate
      );
    }
    async getLiveRefreshInterval() {
      return null;
    }
    getDisposition() {
      return { ...wt };
    }
    async getDecoderConfig() {
      return (
        g(this.demuxer.audioInfo),
        {
          codec: "flac",
          numberOfChannels: this.demuxer.audioInfo.numberOfChannels,
          sampleRate: this.demuxer.audioInfo.sampleRate,
          description: this.demuxer.audioInfo.description,
        }
      );
    }
    async getPacket(e, t) {
      if ((g(this.demuxer.audioInfo), e < 0)) return null;
      const r = await this.demuxer.readingMutex.acquire();
      try {
        for (;;) {
          const n = X(
            this.demuxer.loadedSamples,
            e,
            (c) => c.blockOffset / this.demuxer.audioInfo.sampleRate,
          );
          if (n === -1) {
            await this.demuxer.advanceReader();
            continue;
          }
          const s = this.demuxer.loadedSamples[n],
            a = s.blockOffset / this.demuxer.audioInfo.sampleRate,
            o = s.blockSize / this.demuxer.audioInfo.sampleRate;
          if (a + o <= e) {
            if (this.demuxer.lastSampleLoaded)
              return this.getPacketAtIndex(
                this.demuxer.loadedSamples.length - 1,
                t,
              );
            await this.demuxer.advanceReader();
            continue;
          }
          return this.getPacketAtIndex(n, t);
        }
      } finally {
        r();
      }
    }
    async getNextPacket(e, t) {
      const r = await this.demuxer.readingMutex.acquire();
      try {
        const n = e.sequenceNumber + 1;
        if (
          this.demuxer.lastSampleLoaded &&
          n >= this.demuxer.loadedSamples.length
        )
          return null;
        for (
          ;
          n >= this.demuxer.loadedSamples.length &&
          !this.demuxer.lastSampleLoaded;
        )
          await this.demuxer.advanceReader();
        return this.getPacketAtIndex(n, t);
      } finally {
        r();
      }
    }
    getKeyPacket(e, t) {
      return this.getPacket(e, t);
    }
    getNextKeyPacket(e, t) {
      return this.getNextPacket(e, t);
    }
    async getPacketAtIndex(e, t) {
      const r = this.demuxer.loadedSamples[e];
      if (!r) return null;
      let n;
      if (t.metadataOnly) n = Ue;
      else {
        let o = this.demuxer.reader.requestSlice(r.byteOffset, r.byteSize);
        if ((A(o) && (o = await o), !o)) return null;
        n = N(o, r.byteSize);
      }
      g(this.demuxer.audioInfo);
      const s = r.blockOffset / this.demuxer.audioInfo.sampleRate,
        a = r.blockSize / this.demuxer.audioInfo.sampleRate;
      return new Z(n, "key", s, a, e, r.byteSize);
    }
    async getFirstPacket(e) {
      for (
        ;
        this.demuxer.loadedSamples.length === 0 &&
        !this.demuxer.lastSampleLoaded;
      )
        await this.demuxer.advanceReader();
      return this.getPacketAtIndex(0, e);
    }
  }
  const Xe = 9e4,
    rs = 2 ** 33,
    Ve = 188,
    ed = (i) => {
      let e = "video/MP2T";
      const t = [...new Set(i.filter(Boolean))];
      return (t.length > 0 && (e += `; codecs="${t.join(", ")}"`), e);
    };
  const So =
      "PES packet is missing PTS where it was expected. PES packets without PTS are not currently supported. If you think this file should be supported, please report it.",
    xo = 5,
    td = 1212435798,
    id = 1212436562,
    rd = 1146376960,
    nd = new Set([133, 134, 162]),
    Po = new Set();
  class sd extends bt {
    constructor(e) {
      (super(e),
        (this.metadataPromise = null),
        (this.elementaryStreams = []),
        (this.trackBackingEntries = []),
        (this.packetOffset = 0),
        (this.packetStride = -1),
        (this.sectionEndPositions = []),
        (this.seekChunkSize = 5 * 1024 * 1024),
        (this.minReferencePointByteDistance = -1),
        (this.timestampWrapInfo = null),
        (this.reader = e._reader));
    }
    async readMetadata() {
      return (this.metadataPromise ??= (async () => {
        const e = Ve + 16 + 1;
        let t = this.reader.requestSlice(0, e);
        (A(t) && (t = await t), g(t));
        const r = N(t, e);
        if (r[0] === 71 && r[Ve] === 71)
          ((this.packetOffset = 0), (this.packetStride = Ve));
        else if (r[0] === 71 && r[Ve + 16] === 71)
          ((this.packetOffset = 0), (this.packetStride = Ve + 16));
        else if (r[4] === 71 && r[4 + Ve + 4] === 71)
          ((this.packetOffset = 4), (this.packetStride = Ve + 4));
        else throw new Error("Unreachable.");
        const n = 256;
        this.minReferencePointByteDistance = n * this.packetStride;
        let s = this.packetOffset,
          a = null,
          o = !1,
          c = !1;
        for (;;) {
          const l = await this.readPacketHeader(s);
          if (!l) break;
          if (l.payloadUnitStartIndicator === 0) {
            s += this.packetStride;
            continue;
          }
          if (c && !this.elementaryStreams.some((m) => m.pid === l.pid)) {
            s += this.packetStride;
            continue;
          }
          const u = await this.readSection(s, !0, !c);
          if (!u) break;
          const d = 3,
            f = 32;
          let h = !1;
          if (
            !c &&
            u.pid !== 0 &&
            !(u.payload[0] === 0 && u.payload[1] === 0 && u.payload[2] === 1)
          ) {
            const y = new $(u.payload),
              w = y.readAlignedByte();
            (y.skipBits(8 * w), (h = y.readBits(8) === 2));
          }
          if (u.pid === 0 && !o) {
            const m = new $(u.payload),
              y = m.readAlignedByte();
            (m.skipBits(8 * y), m.skipBits(14));
            const w = m.readBits(10);
            for (m.skipBits(40); 8 * (w + d) - m.pos > f;) {
              const b = m.readBits(16);
              m.skipBits(3);
              const T = m.readBits(13);
              if (b !== 0) {
                if (a !== null)
                  throw new Error(
                    "Only files with a single program are supported.",
                  );
                a = T;
              }
            }
            if (a === null)
              throw new Error(
                "Program Association Table must link to a Program Map Table.",
              );
            o = !0;
          } else if ((u.pid === a || h) && !c) {
            const m = new $(u.payload),
              y = m.readAlignedByte();
            (m.skipBits(8 * y), m.skipBits(12));
            const w = m.readBits(12);
            (m.skipBits(43), m.readBits(13), m.skipBits(6));
            const b = m.readBits(10),
              T = m.pos + 8 * b;
            let S = !1;
            for (; m.pos < T;) {
              const k = m.readBits(8),
                P = m.readBits(8),
                I = m.pos + 8 * P;
              if (k === xo && P >= 4) {
                const E = m.readBits(32);
                S ||= E === td || E === id;
              }
              m.pos = I;
            }
            for (m.pos = T; 8 * (w + d) - m.pos > f;) {
              const k = m.readBits(8);
              m.skipBits(3);
              const P = m.readBits(13);
              m.skipBits(6);
              const I = m.readBits(10),
                E = m.pos + 8 * I;
              let C = !1,
                _ = !1,
                v = !1;
              for (; m.pos < E;) {
                const z = m.readBits(8),
                  O = m.readBits(8),
                  H = m.pos + 8 * O;
                if (z === 106) C = !0;
                else if (z === 122 || z === 204) _ = !0;
                else if (z === 123) v = !0;
                else if (z === xo && O >= 4) {
                  const ee = m.readBits(32);
                  v ||= (ee & 4294967040) === rd;
                }
                m.pos = H;
              }
              let F = null;
              const D = S && nd.has(k) ? 130 : k;
              switch (D) {
                case 27:
                case 36:
                  F = {
                    type: "video",
                    codec: k === 27 ? "avc" : "hevc",
                    decoderConfig: null,
                    avcCodecInfo: null,
                    hevcCodecInfo: null,
                    colorSpace: {
                      primaries: null,
                      transfer: null,
                      matrix: null,
                      fullRange: null,
                    },
                    width: -1,
                    height: -1,
                    squarePixelWidth: -1,
                    squarePixelHeight: -1,
                    reorderSize: -1,
                  };
                  break;
                case 3:
                case 4:
                case 15:
                case 129:
                case 135:
                case 130:
                case 138:
                  {
                    let z;
                    (D === 3 || D === 4
                      ? (z = "mp3")
                      : D === 15
                        ? (z = "aac")
                        : D === 129
                          ? (z = "ac3")
                          : D === 135
                            ? (z = "eac3")
                            : (z = "dts"),
                      (F = {
                        type: "audio",
                        codec: z,
                        decoderConfig: null,
                        aacCodecInfo: null,
                        dtsFormat: null,
                        numberOfChannels: -1,
                        sampleRate: -1,
                      }));
                  }
                  break;
                case 6:
                  _
                    ? (F = {
                        type: "audio",
                        codec: "eac3",
                        decoderConfig: null,
                        aacCodecInfo: null,
                        dtsFormat: null,
                        numberOfChannels: -1,
                        sampleRate: -1,
                      })
                    : C
                      ? (F = {
                          type: "audio",
                          codec: "ac3",
                          decoderConfig: null,
                          aacCodecInfo: null,
                          dtsFormat: null,
                          numberOfChannels: -1,
                          sampleRate: -1,
                        })
                      : v &&
                        (F = {
                          type: "audio",
                          codec: "dts",
                          decoderConfig: null,
                          aacCodecInfo: null,
                          dtsFormat: null,
                          numberOfChannels: -1,
                          sampleRate: -1,
                        });
                  break;
                default:
                  Po.has(k) ||
                    (L._warn(
                      `Note: MPEG-TS streams with stream_type 0x${k.toString(16)} are not currently supported.`,
                    ),
                    Po.add(k));
              }
              F &&
                this.elementaryStreams.push({
                  demuxer: this,
                  pid: P,
                  streamType: k,
                  initialized: !1,
                  firstSection: null,
                  canBeTrustedWithKeyPackets: !1,
                  info: F,
                  referencePesPackets: [],
                });
            }
            c = !0;
          } else {
            const m = this.elementaryStreams.find((y) => y.pid === u.pid);
            e: if (m && !m.initialized) {
              const y = Ci(this, u, !0);
              if (!y)
                throw new Error(
                  `Couldn't read first PES packet for Elementary Stream with PID ${m.pid}`,
                );
              if (
                ((m.firstSection = u),
                (m.canBeTrustedWithKeyPackets = u.randomAccessIndicator === 1),
                this.input._initInput)
              ) {
                const T = (
                  await this.input._initInput._getDemuxer()
                ).elementaryStreams.find(
                  (S) => S.pid === u.pid && S.info.codec === m.info.codec,
                );
                if (T) {
                  ((m.info = T.info), (m.initialized = !0));
                  break e;
                }
              }
              const w = new sr(m, y);
              if (m.info.type === "video") {
                for (;;) {
                  const b = w;
                  if (
                    ((b.suppliedPacket = null),
                    await w.markNextPacket(),
                    m.info.codec === "avc")
                  ) {
                    if (!w.suppliedPacket)
                      throw new Error(
                        "Invalid AVC video stream; could not extract AVCDecoderConfigurationRecord from any packet.",
                      );
                    if (
                      ((m.info.avcCodecInfo = $i(w.suppliedPacket.data)),
                      !m.info.avcCodecInfo)
                    )
                      continue;
                    const T = m.info.avcCodecInfo.sequenceParameterSets[0];
                    g(T);
                    const S = pi(T);
                    ((m.info.width = S.displayWidth),
                      (m.info.height = S.displayHeight));
                    const k = S.pixelAspectRatio.num,
                      P = S.pixelAspectRatio.den;
                    ((m.info.squarePixelWidth = m.info.width),
                      (m.info.squarePixelHeight = m.info.height),
                      k > 0 &&
                        P > 0 &&
                        (k > P
                          ? (m.info.squarePixelWidth = Math.round(
                              (m.info.width * k) / P,
                            ))
                          : (m.info.squarePixelHeight = Math.round(
                              (m.info.height * P) / k,
                            ))),
                      (m.info.colorSpace = {
                        primaries: ht[S.colourPrimaries],
                        transfer: mt[S.transferCharacteristics],
                        matrix: pt[S.matrixCoefficients],
                        fullRange: !!S.fullRangeFlag,
                      }),
                      (m.info.reorderSize = S.maxDecFrameBuffering));
                    break;
                  } else if (m.info.codec === "hevc") {
                    if (!w.suppliedPacket)
                      throw new Error(
                        "Invalid HEVC video stream; could not extract HVCDecoderConfigurationRecord from first packet.",
                      );
                    if (
                      ((m.info.hevcCodecInfo = Ki(w.suppliedPacket.data)),
                      !m.info.hevcCodecInfo)
                    )
                      continue;
                    const S = m.info.hevcCodecInfo.arrays.find(
                      (E) => E.nalUnitType === de.SPS_NUT,
                    ).nalUnits[0];
                    g(S);
                    const k = Er(S);
                    ((m.info.width = k.displayWidth),
                      (m.info.height = k.displayHeight));
                    const P = k.pixelAspectRatio.num,
                      I = k.pixelAspectRatio.den;
                    ((m.info.squarePixelWidth = m.info.width),
                      (m.info.squarePixelHeight = m.info.height),
                      P > 0 &&
                        I > 0 &&
                        (P > I
                          ? (m.info.squarePixelWidth = Math.round(
                              (m.info.width * P) / I,
                            ))
                          : (m.info.squarePixelHeight = Math.round(
                              (m.info.height * I) / P,
                            ))),
                      (m.info.colorSpace = {
                        primaries: ht[k.colourPrimaries],
                        transfer: mt[k.transferCharacteristics],
                        matrix: pt[k.matrixCoefficients],
                        fullRange: !!k.fullRangeFlag,
                      }),
                      (m.info.reorderSize = k.maxDecFrameBuffering));
                    break;
                  } else throw new Error("Unhandled.");
                }
                ((m.info.decoderConfig = {
                  codec: Dr({
                    width: m.info.width,
                    height: m.info.height,
                    codec: m.info.codec,
                    codecDescription: null,
                    colorSpace: m.info.colorSpace,
                    avcType: 1,
                    avcCodecInfo: m.info.avcCodecInfo,
                    hevcCodecInfo: m.info.hevcCodecInfo,
                    vp9CodecInfo: null,
                    av1CodecInfo: null,
                    proresFormat: null,
                  }),
                  codedWidth: m.info.width,
                  codedHeight: m.info.height,
                  colorSpace: m.info.colorSpace,
                }),
                  (m.info.width !== m.info.squarePixelWidth ||
                    m.info.height !== m.info.squarePixelHeight) &&
                    ((m.info.decoderConfig.displayAspectWidth =
                      m.info.squarePixelWidth),
                    (m.info.decoderConfig.displayAspectHeight =
                      m.info.squarePixelHeight)),
                  (m.initialized = !0));
              } else {
                if ((await w.markNextPacket(), !w.suppliedPacket))
                  throw new Error(
                    `Couldn't parse first media packet for Elementary Stream with PID ${m.pid}`,
                  );
                if (m.info.codec === "aac") {
                  const b = Me.tempFromBytes(w.suppliedPacket.data),
                    T = Pt(b);
                  if (!T)
                    throw new Error(
                      "Invalid AAC audio stream; could not read ADTS frame header from first packet.",
                    );
                  ((m.info.aacCodecInfo = {
                    isMpeg2: !1,
                    objectType: T.objectType,
                  }),
                    (m.info.numberOfChannels = mi[T.channelConfiguration]),
                    (m.info.sampleRate = At[T.samplingFrequencyIndex]));
                } else if (m.info.codec === "mp3") {
                  const b = R(Me.tempFromBytes(w.suppliedPacket.data)),
                    T = qn(b, w.suppliedPacket.data.byteLength);
                  if (!T.header)
                    throw new Error(
                      "Invalid MP3 audio stream; could not read frame header from first packet.",
                    );
                  ((m.info.numberOfChannels = Qi(T.header.channel)),
                    (m.info.sampleRate = T.header.sampleRate));
                } else if (m.info.codec === "ac3") {
                  const b = xa(w.suppliedPacket.data);
                  if (!b)
                    throw new Error(
                      "Invalid AC-3 audio stream; could not read sync frame from first packet.",
                    );
                  if (b.fscod === 3)
                    throw new Error(
                      "Invalid AC-3 audio stream; reserved sample rate code found in first packet.",
                    );
                  ((m.info.numberOfChannels = Nn[b.acmod] + b.lfeon),
                    (m.info.sampleRate = Pr[b.fscod]));
                } else if (m.info.codec === "eac3") {
                  const b = Ca(w.suppliedPacket.data);
                  if (!b)
                    throw new Error(
                      "Invalid E-AC-3 audio stream; could not read sync frame from first packet.",
                    );
                  const T = Ia(b);
                  if (T === null)
                    throw new Error(
                      "Invalid E-AC-3 audio stream; reserved sample rate code found in first packet.",
                    );
                  ((m.info.numberOfChannels = Ea(b)), (m.info.sampleRate = T));
                } else if (m.info.codec === "dts") {
                  const b = Vn(w.suppliedPacket.data);
                  if (!b)
                    throw new Error(
                      "Invalid DTS audio stream; could not read frame header from first packet.",
                    );
                  ((m.info.numberOfChannels = b.numberOfChannels),
                    (m.info.sampleRate = b.sampleRate),
                    b.core &&
                      (m.info.dtsFormat = b.hasExtensions ? "dtsh" : "dtsc"));
                } else throw new Error("Unhandled.");
                ((m.info.decoderConfig = {
                  codec: Ln({
                    codec: m.info.codec,
                    codecDescription: null,
                    aacCodecInfo: m.info.aacCodecInfo,
                    dtsFormat: m.info.dtsFormat,
                  }),
                  numberOfChannels: m.info.numberOfChannels,
                  sampleRate: m.info.sampleRate,
                }),
                  (m.initialized = !0));
              }
            }
          }
          if (c && this.elementaryStreams.every((m) => m.initialized)) break;
          s += this.packetStride;
        }
        if (!c)
          throw o
            ? new Error("No Program Map Table found in the file.")
            : new Error("No Program Association Table found in the file.");
        for (const l of this.elementaryStreams)
          l.initialized &&
            (l.info.type === "video"
              ? this.trackBackingEntries.push(new ad(l))
              : this.trackBackingEntries.push(new od(l)));
      })());
    }
    async getTrackBackings() {
      return (await this.readMetadata(), this.trackBackingEntries);
    }
    async getMetadataTags() {
      return {};
    }
    async getMimeType() {
      await this.readMetadata();
      const e = await Promise.all(
        this.trackBackingEntries.map((t) =>
          t.getDecoderConfig().then((r) => r?.codec ?? null),
        ),
      );
      return ed(e);
    }
    async readSection(e, t, r = !1) {
      let n = e,
        s = e;
      const a = [];
      let o = 0,
        c = null,
        l = !0,
        u = 0;
      for (;;) {
        const f = await this.readPacket(s);
        if (((s += this.packetStride), !f)) break;
        if (c) {
          if (f.pid !== c.pid) {
            if (r) break;
            continue;
          }
          if (f.payloadUnitStartIndicator === 1) break;
        } else {
          if (f.payloadUnitStartIndicator === 0) break;
          c = f;
        }
        const h = !!(f.adaptationFieldControl & 2),
          p = !!(f.adaptationFieldControl & 1);
        let m = 0;
        if (
          (h &&
            ((m = 1 + f.body[0]),
            f === c && m > 1 && (u = (f.body[1] >> 6) & 1)),
          p &&
            (m === 0
              ? (a.push(f.body), (o += f.body.byteLength))
              : (a.push(f.body.subarray(m)), (o += f.body.byteLength - m))),
          (n = s),
          !t && o >= 64)
        ) {
          l = !1;
          break;
        }
        if (Oi(this.sectionEndPositions, n, (w) => w) !== -1) {
          l = !1;
          break;
        }
      }
      if (l) {
        const f = X(this.sectionEndPositions, n, (h) => h);
        this.sectionEndPositions.splice(f + 1, 0, n);
      }
      if (!c) return null;
      let d;
      if (a.length === 1) d = a[0];
      else {
        const f = a.reduce((p, m) => p + m.length, 0);
        d = new Uint8Array(f);
        let h = 0;
        for (const p of a) (d.set(p, h), (h += p.length));
      }
      return {
        startPos: e,
        endPos: t ? n : null,
        pid: c.pid,
        payload: d,
        randomAccessIndicator: u,
      };
    }
    async readPacketHeader(e) {
      let t = this.reader.requestSlice(e, 4);
      if ((A(t) && (t = await t), !t)) return null;
      if (U(t) !== 71)
        throw new Error(
          "Invalid TS packet sync byte. Likely an internal bug, please report this file.",
        );
      const n = fe(t),
        s = (n >> 14) & 1,
        a = n & 8191,
        c = (U(t) >> 4) & 3;
      return {
        payloadUnitStartIndicator: s,
        pid: a,
        adaptationFieldControl: c,
      };
    }
    async readPacket(e) {
      let t = this.reader.requestSlice(e, Ve);
      if ((A(t) && (t = await t), !t)) return null;
      const r = N(t, Ve);
      if (r[0] !== 71)
        throw new Error(
          "Invalid TS packet sync byte. Likely an internal bug, please report this file.",
        );
      const s = (r[1] << 8) + r[2],
        a = (s >> 14) & 1,
        o = s & 8191,
        l = (r[3] >> 4) & 3;
      return {
        payloadUnitStartIndicator: a,
        pid: o,
        adaptationFieldControl: l,
        body: r.subarray(4),
      };
    }
    normalizeTimestamp(e) {
      if (!this.timestampWrapInfo) {
        const n = 60 * Xe;
        this.timestampWrapInfo = {
          reference: e - n,
          offset: e >= rs - n ? -rs : rs,
        };
      }
      const { reference: t, offset: r } = this.timestampWrapInfo;
      return (r < 0 && e >= t) || (r > 0 && e < t) ? e + r : e;
    }
  }
  const Yt = (i, e, t) => {
      if (e.payload.byteLength < 3) return null;
      const r = new $(e.payload);
      if (r.readBits(24) !== 1) return null;
      const s = r.readBits(8);
      if (
        (r.skipBits(16),
        s === 188 ||
          s === 190 ||
          s === 191 ||
          s === 240 ||
          s === 241 ||
          s === 255 ||
          s === 242 ||
          s === 248)
      )
        return null;
      r.skipBits(8);
      const a = r.readBits(2);
      r.skipBits(14);
      let o = null;
      if (a === 2 || a === 3)
        ((o = 0),
          r.skipBits(4),
          (o += r.readBits(3) * (1 << 30)),
          r.skipBits(1),
          (o += r.readBits(15) * 32768),
          r.skipBits(1),
          (o += r.readBits(15)),
          (o = i.normalizeTimestamp(o)));
      else if (t) throw new Error(So);
      return {
        sectionStartPos: e.startPos,
        sectionEndPos: e.endPos,
        pts: o,
        randomAccessIndicator: e.randomAccessIndicator,
      };
    },
    Ci = (i, e, t) => {
      g(e.endPos !== null);
      const r = Yt(i, e, t);
      if (!r) return null;
      const n = new $(e.payload);
      n.skipBits(32);
      const s = n.readBits(16),
        a = 6;
      n.skipBits(16);
      const o = n.readBits(8),
        c = n.pos + 8 * o;
      n.pos = c;
      const l = c / 8;
      g(Number.isInteger(l));
      const u = e.payload.subarray(l, s > 0 ? a + s : e.payload.byteLength);
      return { ...r, data: u };
    };
  class Wr {
    constructor(e) {
      ((this.elementaryStream = e),
        (this.packetBuffers = new WeakMap()),
        (this.packetSectionStarts = new WeakMap()));
    }
    getId() {
      return this.elementaryStream.pid;
    }
    getNumber() {
      const e = this.elementaryStream.demuxer,
        t = this.elementaryStream.info.type;
      let r = 0;
      for (const n of e.trackBackingEntries)
        if (
          (n.getType() === t && r++,
          g(n instanceof Wr),
          n.elementaryStream === this.elementaryStream)
        )
          break;
      return r;
    }
    getCodec() {
      throw new Error("Not implemented on base class.");
    }
    getInternalCodecId() {
      return this.elementaryStream.streamType;
    }
    getName() {
      return null;
    }
    getLanguageCode() {
      return Pe;
    }
    getDisposition() {
      return { ...wt, primary: !1 };
    }
    getTimeResolution() {
      return Xe;
    }
    isRelativeToUnixEpoch() {
      return !1;
    }
    getUnixTimeForTimestamp() {
      return null;
    }
    getPairingMask() {
      return 1n;
    }
    getBitrate() {
      return null;
    }
    getAverageBitrate() {
      return null;
    }
    async getDurationFromMetadata() {
      return null;
    }
    async getLiveRefreshInterval() {
      return null;
    }
    createEncodedPacket(e, t, r) {
      let n;
      return (
        this.allPacketsAreKeyPackets()
          ? (n = "key")
          : (n = e.randomAccessIndicator === 1 ? "key" : "delta"),
        new Z(
          r.metadataOnly ? Ue : e.data,
          n,
          e.pts / Xe,
          Math.max(t / Xe, 0),
          e.sequenceNumber,
          e.data.byteLength,
        )
      );
    }
    async getFirstPacket(e) {
      const t = this.elementaryStream.firstSection;
      g(t);
      const r = Ci(this.elementaryStream.demuxer, t, !0);
      g(r);
      const n = new sr(this.elementaryStream, r),
        s = new ss(this, n),
        a = await s.readNext();
      if (!a) return null;
      const o = this.createEncodedPacket(a.packet, a.duration, e);
      return (
        this.packetBuffers.set(o, s),
        this.packetSectionStarts.set(o, a.packet.sectionStartPos),
        o
      );
    }
    async getNextPacket(e, t) {
      let r = this.packetBuffers.get(e);
      if (r) {
        const u = await r.readNext();
        if (!u) return null;
        this.packetBuffers.delete(e);
        const d = this.createEncodedPacket(u.packet, u.duration, t);
        return (
          this.packetBuffers.set(d, r),
          this.packetSectionStarts.set(d, u.packet.sectionStartPos),
          d
        );
      }
      const n = this.packetSectionStarts.get(e);
      if (n === void 0)
        throw new Error("Packet was not created from this track.");
      const s = this.elementaryStream.demuxer,
        a = await s.readSection(n, !0);
      g(a);
      const o = Ci(s, a, !0);
      g(o);
      const c = new sr(this.elementaryStream, o);
      r = new ss(this, c);
      const l = e.sequenceNumber;
      for (;;) {
        const u = await r.readNext();
        if (!u) return null;
        if (u.packet.sequenceNumber > l) {
          const d = this.createEncodedPacket(u.packet, u.duration, t);
          return (
            this.packetBuffers.set(d, r),
            this.packetSectionStarts.set(d, u.packet.sectionStartPos),
            d
          );
        }
      }
    }
    async getNextKeyPacket(e, t) {
      let r = e;
      for (;;) {
        if (((r = await this.getNextPacket(r, t)), !r)) return null;
        if (r.type === "key") return r;
      }
    }
    getPacket(e, t) {
      return this.doPacketLookup(e, !1, t);
    }
    getKeyPacket(e, t) {
      return this.doPacketLookup(e, !0, t);
    }
    async doPacketLookup(e, t, r) {
      const n = zi(e * Xe),
        s = this.elementaryStream.demuxer,
        { reader: a, seekChunkSize: o } = s,
        c = this.elementaryStream.pid,
        l = async (S, k, P) => {
          let I = S;
          for (; I < k;) {
            const E = await s.readPacketHeader(I);
            if (!E) return null;
            if (E.pid === c && E.payloadUnitStartIndicator === 1) {
              const C = await s.readSection(I, P);
              if (!C) return null;
              const _ = Yt(s, C, !1);
              if (_ && _.pts !== null)
                return { pesPacketHeader: _, section: C };
            }
            I += s.packetStride;
          }
          return null;
        },
        u = this.elementaryStream.firstSection;
      g(u);
      const d = Yt(s, u, !0);
      if ((g(d), n < d.pts)) return null;
      let f;
      const h = this.elementaryStream.referencePesPackets,
        p = X(h, n, (S) => S.pts),
        m = p !== -1 ? h[p] : null;
      if (m && n - m.pts < Xe / 2) f = m.sectionStartPos;
      else {
        let S = 0;
        if (a.fileSize !== null) {
          const k = Math.ceil(a.fileSize / o);
          if (k > 1) {
            let P = 0,
              I = k - 1;
            for (S = P; P <= I;) {
              const E = Math.floor((P + I) / 2),
                C = gn(E * o, s.packetStride) + d.sectionStartPos,
                _ = C + o,
                v = await l(C, _, !1);
              if (!v) {
                I = E - 1;
                continue;
              }
              v.pesPacketHeader.pts <= n ? ((S = E), (P = E + 1)) : (I = E - 1);
            }
          }
        }
        f = gn(S * o, s.packetStride) + d.sectionStartPos;
      }
      let w = (await l(f, a.fileSize ?? 1 / 0, !1))?.pesPacketHeader ?? null;
      w || (w = d);
      const b = this.getReorderSize(),
        T = async (S, k) => {
          const P = await s.readSection(S, !0);
          g(P);
          const I = Ci(s, P, !0);
          g(I);
          const E = new sr(this.elementaryStream, I),
            C = new ss(this, E);
          for (
            ;
            !(
              (ne(C.presentationOrderPackets)?.pts ?? -1 / 0) >= n ||
              !(await C.readNextPacket())
            );
          );
          const _ = hn(C.presentationOrderPackets, k);
          if (_ === -1) return null;
          const v = C.presentationOrderPackets[_],
            F = _ === 0 ? 0 : v.pts - C.presentationOrderPackets[_ - 1].pts;
          for (; C.decodeOrderPackets[0] !== v;) C.decodeOrderPackets.shift();
          C.lastDuration = F;
          const D = await C.readNext();
          g(D);
          const z = this.createEncodedPacket(D.packet, D.duration, r);
          return (
            this.packetBuffers.set(z, C),
            this.packetSectionStarts.set(z, D.packet.sectionStartPos),
            z
          );
        };
      if (!t || this.allPacketsAreKeyPackets()) {
        e: for (;;) {
          let S = w.sectionStartPos + s.packetStride;
          for (;;) {
            const k = await s.readPacketHeader(S);
            if (!k) break e;
            if (k.pid === c && k.payloadUnitStartIndicator === 1) {
              const P = await s.readSection(S, !1);
              if (P) {
                const I = Yt(s, P, !1);
                if (I && I.pts !== null) {
                  if (I.pts > n) break e;
                  ((w = I), ns(this.elementaryStream, w));
                  break;
                }
              }
            }
            S += s.packetStride;
          }
        }
        e: for (let S = 0; S < b + 1; S++) {
          let k = w.sectionStartPos - s.packetStride;
          for (; k >= s.packetOffset;) {
            const P = await s.readPacketHeader(k);
            if (!P) break e;
            if (P.pid === c && P.payloadUnitStartIndicator === 1) {
              const I = await s.readSection(k, !1);
              if (I) {
                const E = Yt(s, I, !1);
                if (E && E.pts !== null) {
                  w = E;
                  break;
                }
              }
            }
            k -= s.packetStride;
          }
        }
        return T(w.sectionStartPos, (S) => S.pts <= n);
      } else {
        let S = f,
          k = null;
        const P = !this.elementaryStream.canBeTrustedWithKeyPackets;
        for (;;) {
          let I = null;
          const E = S <= d.sectionStartPos;
          let C,
            _ = null;
          if (E) ((C = d), (_ = u));
          else {
            const D = await l(S, a.fileSize ?? 1 / 0, P);
            ((C = D?.pesPacketHeader ?? null), (_ = D?.section ?? null));
          }
          let v = !1,
            F = 0;
          e: for (; C && !(k !== null && C.sectionStartPos >= k);) {
            if (C.pts <= n) {
              let z;
              if (this.elementaryStream.canBeTrustedWithKeyPackets)
                z = C.randomAccessIndicator === 1;
              else {
                g(_);
                const O = Ci(s, _, !0);
                g(O);
                const H = new sr(this.elementaryStream, O);
                (await H.markNextPacket(),
                  (z = H.suppliedPacket?.randomAccessIndicator === 1));
              }
              z && (I = C);
            }
            if ((C.pts > n && (v = !0), v && (F++, F > b))) break;
            let D = C.sectionStartPos + s.packetStride;
            for (;;) {
              const z = await s.readPacketHeader(D);
              if (!z) break e;
              if (z.pid === c && z.payloadUnitStartIndicator === 1) {
                const O = await s.readSection(D, P);
                if (O) {
                  const H = Yt(s, O, !1);
                  if (H && H.pts !== null) {
                    ((C = H), (_ = O), ns(this.elementaryStream, C));
                    break;
                  }
                }
              }
              D += s.packetStride;
            }
          }
          if (I) {
            let D = I;
            if (F === 0)
              e: for (let O = 0; O < b; O++) {
                let H = D.sectionStartPos - s.packetStride;
                for (; H >= s.packetOffset;) {
                  const ee = await s.readPacketHeader(H);
                  if (!ee) break e;
                  if (ee.pid === c && ee.payloadUnitStartIndicator === 1) {
                    const he = await s.readSection(H, P);
                    if (he) {
                      const Y = Yt(s, he, !1);
                      if (Y && Y.pts !== null) {
                        D = Y;
                        break;
                      }
                    }
                  }
                  H -= s.packetStride;
                }
              }
            const z = await T(
              D.sectionStartPos,
              (O) => O.pts <= n && O.randomAccessIndicator === 1,
            );
            return (g(z), z);
          }
          if (E) return null;
          ((k = S),
            (S = Math.max(
              gn(S - d.sectionStartPos - o, s.packetStride) + d.sectionStartPos,
              d.sectionStartPos,
            )));
        }
      }
    }
  }
  class ad extends Wr {
    getType() {
      return "video";
    }
    getCodec() {
      return this.elementaryStream.info.codec;
    }
    getCodedWidth() {
      return this.elementaryStream.info.width;
    }
    getCodedHeight() {
      return this.elementaryStream.info.height;
    }
    getSquarePixelWidth() {
      return this.elementaryStream.info.squarePixelWidth;
    }
    getSquarePixelHeight() {
      return this.elementaryStream.info.squarePixelHeight;
    }
    getTransformationMatrix() {
      return [...Fi];
    }
    async getColorSpace() {
      return this.elementaryStream.info.colorSpace;
    }
    async canBeTransparent() {
      return !1;
    }
    async getDecoderConfig() {
      return (
        g(this.elementaryStream.info.decoderConfig),
        this.elementaryStream.info.decoderConfig
      );
    }
    allPacketsAreKeyPackets() {
      return !1;
    }
    getReorderSize() {
      return this.elementaryStream.info.reorderSize;
    }
  }
  class od extends Wr {
    getType() {
      return "audio";
    }
    getCodec() {
      return this.elementaryStream.info.codec;
    }
    getNumberOfChannels() {
      return this.elementaryStream.info.numberOfChannels;
    }
    getSampleRate() {
      return this.elementaryStream.info.sampleRate;
    }
    async getDecoderConfig() {
      return (
        g(this.elementaryStream.info.decoderConfig),
        this.elementaryStream.info.decoderConfig
      );
    }
    allPacketsAreKeyPackets() {
      return !0;
    }
    getReorderSize() {
      return 0;
    }
  }
  const ns = (i, e) => {
    const t = i.referencePesPackets,
      r = X(t, e.sectionStartPos, (n) => n.sectionStartPos);
    if (r >= 0) {
      const n = t[r];
      if (e.pts <= n.pts) return !1;
      const s = i.demuxer.minReferencePointByteDistance;
      if (e.sectionStartPos - n.sectionStartPos < s) return !1;
      if (r < t.length - 1) {
        const a = t[r + 1];
        if (a.pts < e.pts || a.sectionStartPos - e.sectionStartPos < s)
          return !1;
      }
    }
    return (t.splice(r + 1, 0, e), !0);
  };
  class sr {
    constructor(e, t) {
      ((this.currentPos = 0),
        (this.pesPackets = []),
        (this.currentPesPacketIndex = 0),
        (this.currentPesPacketPos = 0),
        (this.endPos = 0),
        (this.lastSuppliedPesPacket = null),
        (this.nextPts = null),
        (this.suppliedPacket = null),
        (this.elementaryStream = e),
        (this.pid = e.pid),
        (this.demuxer = e.demuxer),
        (this.startingPesPacket = t));
    }
    ensureBuffered(e) {
      const t = this.endPos - this.currentPos;
      return t >= e
        ? e
        : this.bufferData(e - t).then(() =>
            Math.min(this.endPos - this.currentPos, e),
          );
    }
    getCurrentPesPacket() {
      const e = this.pesPackets[this.currentPesPacketIndex];
      return (g(e), e);
    }
    async bufferData(e) {
      const t = this.endPos + e;
      for (; this.endPos < t;) {
        let r;
        if (this.pesPackets.length === 0) r = this.startingPesPacket;
        else {
          let n = ne(this.pesPackets).sectionEndPos;
          for (g(n !== null); ;) {
            const s = await this.demuxer.readPacketHeader(n);
            if (!s) return;
            if (s.pid === this.pid) {
              const a = await this.demuxer.readSection(n, !0);
              if (!a) return;
              const o = Ci(this.demuxer, a, !1);
              if (o) {
                r = o;
                break;
              }
            }
            n += this.demuxer.packetStride;
          }
        }
        (this.pesPackets.push(r), (this.endPos += r.data.byteLength));
      }
    }
    readBytes(e) {
      const t = this.getCurrentPesPacket(),
        r = this.currentPos - this.currentPesPacketPos,
        n = r + e;
      if (((this.currentPos += e), n <= t.data.byteLength))
        return t.data.subarray(r, n);
      const s = new Uint8Array(e);
      s.set(t.data.subarray(r));
      let a = t.data.byteLength - r;
      for (;;) {
        this.advanceCurrentPacket();
        const o = this.getCurrentPesPacket(),
          c = e - a;
        if (c <= o.data.byteLength) {
          s.set(o.data.subarray(0, c), a);
          break;
        }
        (s.set(o.data, a), (a += o.data.byteLength));
      }
      return s;
    }
    readU8() {
      let e = this.getCurrentPesPacket();
      const t = this.currentPos - this.currentPesPacketPos;
      return (
        this.currentPos++,
        t < e.data.byteLength
          ? e.data[t]
          : (this.advanceCurrentPacket(),
            (e = this.getCurrentPesPacket()),
            e.data[0])
      );
    }
    seekTo(e) {
      if (e !== this.currentPos) {
        if (e < this.currentPos)
          for (; e < this.currentPesPacketPos;) {
            this.currentPesPacketIndex--;
            const t = this.getCurrentPesPacket();
            this.currentPesPacketPos -= t.data.byteLength;
          }
        else
          for (;;) {
            const t = this.getCurrentPesPacket(),
              r = this.currentPesPacketPos + t.data.byteLength;
            if (e < r) break;
            ((this.currentPesPacketPos += t.data.byteLength),
              this.currentPesPacketIndex++);
          }
        this.currentPos = e;
      }
    }
    skip(e) {
      this.seekTo(this.currentPos + e);
    }
    advanceCurrentPacket() {
      ((this.currentPesPacketPos += this.getCurrentPesPacket().data.byteLength),
        this.currentPesPacketIndex++);
    }
    async markNextPacket() {
      g(!this.suppliedPacket);
      const e = this.elementaryStream;
      if (e.info.type === "video") {
        const t = e.info.codec,
          r = 1024;
        if (t !== "avc" && t !== "hevc") throw new Error("Unhandled.");
        const n = t === "avc" ? 1 : 2;
        let s = null,
          a = !1,
          o = 0;
        for (;;) {
          let c = this.ensureBuffered(r);
          if ((A(c) && (c = await c), c === 0)) break;
          const l = this.currentPos,
            u = this.readBytes(c),
            d = u.byteLength;
          let f = 0;
          for (; f < d;) {
            const h = u.indexOf(0, f);
            if (h === -1 || h >= d) break;
            f = h;
            const p = l + f;
            if (f + 3 >= d) {
              this.seekTo(p);
              break;
            }
            const m = u[f + 1],
              y = u[f + 2],
              w = u[f + 3];
            let b = 0;
            if (
              (m === 0 && y === 0 && w === 1
                ? (b = 4)
                : m === 0 && y === 1 && (b = 3),
              b === 0)
            ) {
              f++;
              continue;
            }
            const T = p;
            s ??= T;
            const S = f + b,
              k = S + n,
              P = 6;
            if (k + (t === "avc" ? P : 1) > d) {
              this.seekTo(p);
              break;
            }
            const E = u[S];
            let C, _, v;
            if (t === "avc")
              ((C = ji(E)),
                (_ =
                  C === be.NON_IDR_SLICE || C === be.SLICE_DPA || C === be.IDR),
                (v =
                  C === be.SEI ||
                  C === be.SPS ||
                  C === be.PPS ||
                  C === be.AUD));
            else {
              if (((C = gi(E)), (((E & 1) << 5) | (u[S + 1] >> 3)) > 0)) {
                f += b;
                continue;
              }
              ((_ = C <= de.RASL_R || (C >= de.BLA_W_LP && C <= 21)),
                (v =
                  (C >= de.VPS_NUT && C <= 37) ||
                  C === de.PREFIX_SEI_NUT ||
                  (C >= 41 && C <= 44) ||
                  (C >= 48 && C <= 55)));
            }
            let F = !1;
            if (_) {
              let D;
              if (t === "avc") {
                const z = u.subarray(k, k + P),
                  O = M(new $(z));
                ((D = !a || O <= o), (o = O));
              } else D = u[k] >> 7 === 1;
              D && (a ? (F = !0) : (a = !0));
            } else v && a && (F = !0);
            if (F) {
              const D = T - s;
              return (this.seekTo(s), this.supplyPacket(D, 0));
            }
            f += b;
          }
          if (c < r) break;
        }
        if (s !== null && this.endPos > s) {
          const c = this.endPos - s;
          return (this.seekTo(s), this.supplyPacket(c, 0));
        }
      } else {
        const t = e.info.codec,
          r = 128;
        for (;;) {
          let n = this.ensureBuffered(r);
          A(n) && (n = await n);
          const s = this.currentPos;
          for (; this.currentPos - s < n;) {
            const a = this.readU8();
            if (t === "aac") {
              if (a !== 255) continue;
              this.skip(-1);
              const o = this.currentPos;
              let c = this.ensureBuffered(Ot);
              if ((A(c) && (c = await c), c < Ot)) return;
              const l = this.readBytes(Ot),
                u = Pt(Me.tempFromBytes(l));
              if (u) {
                this.seekTo(o);
                let d = this.ensureBuffered(u.frameLength);
                return (
                  A(d) && (d = await d),
                  this.supplyPacket(
                    d,
                    Math.round((Ur * Xe) / e.info.sampleRate),
                  )
                );
              } else this.seekTo(o + 1);
            } else if (t === "mp3") {
              if (a !== 255) continue;
              this.skip(-1);
              const o = this.currentPos;
              let c = this.ensureBuffered(Gt);
              if ((A(c) && (c = await c), c < Gt)) return;
              const l = this.readBytes(Gt),
                u = j(l).getUint32(0),
                d = qn(u, null);
              if (d.header) {
                this.seekTo(o);
                let f = this.ensureBuffered(d.header.totalSize);
                A(f) && (f = await f);
                const h =
                  (d.header.audioSamplesInFrame * Xe) / e.info.sampleRate;
                return this.supplyPacket(f, Math.round(h));
              } else this.seekTo(o + 1);
            } else if (t === "ac3") {
              if (a !== 11) continue;
              this.skip(-1);
              const o = this.currentPos;
              let c = this.ensureBuffered(5);
              if ((A(c) && (c = await c), c < 5)) return;
              const l = this.readBytes(5);
              if (l[0] !== 11 || l[1] !== 119) {
                this.seekTo(o + 1);
                continue;
              }
              const u = l[4] >> 6,
                d = l[4] & 63;
              if (u === 3 || d > 37) {
                this.seekTo(o + 1);
                continue;
              }
              const f = Nl[3 * d + u];
              (g(f !== void 0),
                this.seekTo(o),
                (c = this.ensureBuffered(f)),
                A(c) && (c = await c));
              const h = Math.round((Ul * Xe) / e.info.sampleRate);
              return this.supplyPacket(c, h);
            } else if (t === "eac3") {
              if (a !== 11) continue;
              this.skip(-1);
              const o = this.currentPos;
              let c = this.ensureBuffered(5);
              if ((A(c) && (c = await c), c < 5)) return;
              const l = this.readBytes(5);
              if (l[0] !== 11 || l[1] !== 119) {
                this.seekTo(o + 1);
                continue;
              }
              const d = ((((l[2] & 7) << 8) | l[3]) + 1) * 2,
                h = l[4] >> 6 === 3 ? 3 : (l[4] >> 4) & 3,
                p = Pa[h];
              (this.seekTo(o),
                (c = this.ensureBuffered(d)),
                A(c) && (c = await c));
              const m = p * 256,
                y = Math.round((m * Xe) / e.info.sampleRate);
              return this.supplyPacket(c, y);
            } else if (t === "dts") {
              if (a !== 127 && a !== 100) continue;
              this.skip(-1);
              const o = this.currentPos;
              let c = this.ensureBuffered(Ar);
              if ((A(c) && (c = await c), c < Ar)) return;
              const l = this.readBytes(Ar),
                u = Ba(l);
              let d = u ? null : Fr(l);
              if (!u && !d) {
                this.seekTo(o + 1);
                continue;
              }
              if (d && !d.asset) {
                this.seekTo(o);
                const m = Math.min(d.frameSize, Ll);
                let y = this.ensureBuffered(m);
                (A(y) && (y = await y), (d = Fr(this.readBytes(y)) ?? d));
              }
              let f = u ? u.frameSize : d.frameSize;
              if (u) {
                let m = Math.ceil(u.frameSize / 4) * 4;
                for (;;) {
                  this.seekTo(o);
                  const y = m + Un;
                  let w = this.ensureBuffered(y);
                  if ((A(w) && (w = await w), w < y)) break;
                  this.seekTo(o + m);
                  const b = Fr(this.readBytes(Un));
                  if (!b) break;
                  ((m += b.frameSize), (f = m));
                }
              }
              const h = u?.sampleCount ?? d.asset?.sampleCount;
              if (h === void 0) {
                this.seekTo(o + 1);
                continue;
              }
              (this.seekTo(o),
                (c = this.ensureBuffered(f)),
                A(c) && (c = await c));
              const p = Math.round((h * Xe) / e.info.sampleRate);
              return this.supplyPacket(c, p);
            } else throw new Error("Unhandled.");
          }
          if (n < r) break;
        }
      }
    }
    supplyPacket(e, t) {
      const r = this.getCurrentPesPacket();
      let n;
      if (this.lastSuppliedPesPacket === r)
        (g(this.nextPts !== null), (n = this.nextPts));
      else {
        if (r.pts === null) throw new Error(So);
        ((n = r.pts), ns(this.elementaryStream, r));
      }
      ((this.lastSuppliedPesPacket = r), (this.nextPts = n + t));
      const s = r.sectionStartPos,
        a = s + (this.currentPos - this.currentPesPacketPos),
        o = this.readBytes(e);
      let c = r.randomAccessIndicator;
      if (c === 0 && !this.elementaryStream.canBeTrustedWithKeyPackets) {
        if (this.elementaryStream.info.type === "audio") c = 1;
        else if (this.elementaryStream.info.decoderConfig) {
          const l =
            vr(
              this.elementaryStream.info.codec,
              this.elementaryStream.info.decoderConfig,
              o,
            ) === "key";
          c = Number(l);
        }
      }
      ((this.suppliedPacket = {
        pts: n,
        data: o,
        sequenceNumber: a,
        sectionStartPos: s,
        randomAccessIndicator: c,
      }),
        this.pesPackets.splice(0, this.currentPesPacketIndex),
        (this.currentPesPacketIndex = 0));
    }
  }
  class ss {
    constructor(e, t) {
      ((this.decodeOrderPackets = []),
        (this.reorderBuffer = []),
        (this.presentationOrderPackets = []),
        (this.reachedEnd = !1),
        (this.lastDuration = 0),
        (this.backing = e),
        (this.context = t),
        (this.reorderSize = e.getReorderSize()),
        g(this.reorderSize >= 0));
    }
    async readNext() {
      if (
        this.decodeOrderPackets.length === 0 &&
        !(await this.readNextPacket())
      )
        return null;
      await this.ensureCurrentPacketHasNext();
      const e = this.decodeOrderPackets[0],
        t = this.presentationOrderPackets.indexOf(e);
      g(t !== -1);
      let r;
      for (
        t === this.presentationOrderPackets.length - 1
          ? (r = this.lastDuration)
          : ((r = this.presentationOrderPackets[t + 1].pts - e.pts),
            (this.lastDuration = r)),
          this.decodeOrderPackets.shift();
        this.presentationOrderPackets.length > 0;
      ) {
        const n = this.presentationOrderPackets[0];
        if (this.decodeOrderPackets.includes(n)) break;
        this.presentationOrderPackets.shift();
      }
      return { packet: e, duration: r };
    }
    async readNextPacket() {
      if (this.reachedEnd) return !1;
      let e;
      return (
        this.context.suppliedPacket
          ? (e = this.context.suppliedPacket)
          : (await this.context.markNextPacket(),
            (e = this.context.suppliedPacket)),
        (this.context.suppliedPacket = null),
        e
          ? (this.decodeOrderPackets.push(e),
            this.processPacketThroughReorderBuffer(e),
            !0)
          : ((this.reachedEnd = !0), this.flushReorderBuffer(), !1)
      );
    }
    async ensureCurrentPacketHasNext() {
      const e = this.decodeOrderPackets[0];
      for (g(e); ;) {
        const t = this.presentationOrderPackets.indexOf(e);
        if (
          (t !== -1 && t <= this.presentationOrderPackets.length - 2) ||
          !(await this.readNextPacket())
        )
          break;
      }
    }
    processPacketThroughReorderBuffer(e) {
      if (
        (this.reorderBuffer.push(e),
        this.reorderBuffer.length > this.reorderSize)
      ) {
        let t = 0;
        for (let n = 1; n < this.reorderBuffer.length; n++)
          this.reorderBuffer[n].pts < this.reorderBuffer[t].pts && (t = n);
        const r = this.reorderBuffer[t];
        (this.presentationOrderPackets.push(r),
          this.reorderBuffer.splice(t, 1));
      }
    }
    flushReorderBuffer() {
      (this.reorderBuffer.sort((e, t) => e.pts - t.pts),
        this.presentationOrderPackets.push(...this.reorderBuffer),
        (this.reorderBuffer.length = 0));
    }
  }
  const Co = "application/vnd.apple.mpegurl",
    Io = "#EXT-X-STREAM-INF:",
    Eo = "#EXT-X-I-FRAME-STREAM-INF:",
    _o = "#EXT-X-MEDIA:",
    Vr = "#EXT-X-DEFINE:",
    as = "#EXTINF:",
    vo = "#EXT-X-MAP:",
    Ao = "#EXT-X-KEY:",
    Bo = "#EXT-X-MEDIA-SEQUENCE:",
    Fo = "#EXT-X-BYTERANGE:",
    Ro = "#EXT-X-PROGRAM-DATE-TIME:",
    cd = "#EXT-X-DISCONTINUITY",
    Mo = "#EXT-X-TARGETDURATION:",
    ld = "#EXT-X-ENDLIST",
    Do = "#EXT-X-PLAYLIST-TYPE:",
    ud = "#EXT-X-I-FRAMES-ONLY",
    Oo = (i) => i.length === 0 || (i.startsWith("#") && !i.startsWith("#EXT"));
  class Ii {
    constructor(e, t) {
      this._attributes = {};
      let r = "",
        n = "",
        s = !1,
        a = !1,
        o = !1;
      const c = () => {
        if (!r) return;
        const l = r.trim().toLowerCase();
        Object.prototype.hasOwnProperty.call(this._attributes, l) &&
          L._warn(`Duplicate AttributeName "${l}"; using last value`);
        const u = o || n.startsWith("0x") || n.startsWith("0X");
        ((this._attributes[l] = u && t ? t.substitute(n) : n),
          (r = ""),
          (n = ""),
          (s = !1),
          (o = !1));
      };
      for (let l = 0; l < e.length; l++) {
        const u = e[l];
        u === '"'
          ? (s && !a && n.length === 0 && (o = !0), (a = !a))
          : u === "=" && !s && !a
            ? (s = !0)
            : u === "," && !a
              ? c()
              : s
                ? (n += u)
                : (r += u);
      }
      if (a)
        throw new Error(
          "Invalid M3U8 file; unterminated quoted-string AttributeValue.",
        );
      c();
    }
    get(e) {
      return this._attributes[e.toLowerCase()] ?? null;
    }
    getAsNumber(e) {
      const t = this.get(e);
      if (t === null) return null;
      const r = Number(t);
      return Number.isFinite(r) ? r : null;
    }
    merge(e) {
      Object.assign(this._attributes, e._attributes);
    }
  }
  const dd = /^[A-Za-z0-9_-]+$/,
    fd = /\{\$([A-Za-z0-9_-]+)\}/g;
  class zo {
    constructor(e, t) {
      ((this.playlistPath = e),
        (this.importedVariables = t),
        (this.variables = new Map()));
    }
    define(e) {
      const t = new Ii(e, this),
        r = t.get("name"),
        n = t.get("import"),
        s = t.get("queryparam");
      if ([r, n, s].filter((o) => o !== null).length !== 1)
        throw new Error(
          "Invalid #EXT-X-DEFINE tag; exactly one of NAME, IMPORT, or QUERYPARAM must be present.",
        );
      if (r !== null) {
        const o = t.get("value");
        if (o === null)
          throw new Error("Invalid #EXT-X-DEFINE tag; NAME requires VALUE.");
        this.set(r, o);
      } else if (n !== null) {
        const o = this.importedVariables?.get(n);
        if (o === void 0)
          throw new Error(
            `Invalid #EXT-X-DEFINE tag; cannot import undefined variable "${n}".`,
          );
        this.set(n, o);
      } else if (s !== null) {
        const o = bl(this.playlistPath, s);
        if (o === null)
          throw new Error(
            `Invalid #EXT-X-DEFINE tag; query parameter "${s}" is missing.`,
          );
        this.set(s, o);
      }
    }
    substitute(e) {
      return e.replaceAll(fd, (t, r) => {
        const n = this.variables.get(r);
        if (n === void 0)
          throw new Error(
            `Invalid M3U8 file; variable "${r}" is referenced before being defined.`,
          );
        return n;
      });
    }
    get(e) {
      return this.variables.get(e);
    }
    set(e, t) {
      if (!dd.test(e))
        throw new Error(
          `Invalid #EXT-X-DEFINE tag; invalid variable name "${e}".`,
        );
      if (this.variables.has(e))
        throw new Error(
          `Invalid #EXT-X-DEFINE tag; variable "${e}" is already defined.`,
        );
      this.variables.set(e, t);
    }
  }
  class hd {
    constructor(e, t, r) {
      ((this.nextInputCacheAge = 0),
        (this.inputCache = []),
        (this.trackBackingsPromise = null),
        (this.firstSegment = null),
        (this.firstSegmentFirstTimestamps = new WeakMap()),
        (this.firstTimestampCache = new WeakMap()),
        (this.input = e),
        (this.path = t),
        (this.trackDeclarations = r));
    }
    async getDurationFromMetadata(e) {
      const t = await this.getSegmentAt(1 / 0, {
        skipLiveWait: e.skipLiveWait,
      });
      return t ? t.timestamp + t.duration : null;
    }
    async getUnixTimeForTimestamp(e) {
      let t = await this.getSegmentAt(e, {});
      if (
        ((t ??= await this.getFirstSegment({})),
        !t || t.unixEpochTimestamp === null)
      )
        return null;
      const r = e - t.timestamp;
      return t.unixEpochTimestamp + r;
    }
    async getTrackBackings() {
      return (this.trackBackingsPromise ??= (async () => {
        const e = [];
        if (this.trackDeclarations) {
          for (const t of this.trackDeclarations)
            if (t.type === "video") {
              const r = Vi(e, (n) => n.getType() === "video") + 1;
              e.push(new Uo(this, t, r));
            } else if (t.type === "audio") {
              const r = Vi(e, (n) => n.getType() === "audio") + 1;
              e.push(new Wo(this, t, r));
            }
        } else {
          if (
            ((this.firstSegment = await this.getFirstSegment({})),
            !this.firstSegment)
          )
            return [];
          const r = await this.getInputForSegment(
            this.firstSegment,
          ).getTracks();
          for (const n of r)
            if (n.type === "video") {
              const s = Vi(e, (a) => a.getType() === "video") + 1;
              e.push(new Uo(this, { id: e.length + 1, type: "video" }, s));
            } else if (n.type === "audio") {
              const s = Vi(e, (a) => a.getType() === "audio") + 1;
              e.push(new Wo(this, { id: e.length + 1, type: "audio" }, s));
            }
        }
        return e;
      })());
    }
    async getFirstTimestampForInput(e) {
      const t = this.firstTimestampCache.get(e);
      if (t !== void 0) return t;
      const r = await e.getFirstTimestamp();
      return (this.firstTimestampCache.set(e, r), r);
    }
    async getMediaOffset(e, t) {
      const r = e.firstSegment ?? e;
      let n;
      if (this.firstSegmentFirstTimestamps.has(r))
        n = this.firstSegmentFirstTimestamps.get(r);
      else {
        const l = this.getInputForSegment(r);
        ((n = await this.getFirstTimestampForInput(l)),
          this.firstSegmentFirstTimestamps.set(r, n));
      }
      if (r === e) return r.timestamp - n;
      const s = await this.getFirstTimestampForInput(t),
        a = e.timestamp - r.timestamp,
        c = s - n - a;
      return Math.abs(c) <= Math.min(0.25, a)
        ? r.timestamp - n
        : e.timestamp - s;
    }
    dispose() {
      for (const e of this.inputCache) e.input.dispose();
      this.inputCache.length = 0;
    }
  }
  class No {
    constructor(e, t, r) {
      ((this.packetInfos = new WeakMap()),
        (this.hydrationPromise = null),
        (this.firstInputTrack = null),
        (this.firstSegment = null),
        (this.segmentedInput = e),
        (this.decl = t),
        (this.number = r));
    }
    hydrate() {
      return (this.hydrationPromise ??= (async () => {
        if (
          ((this.segmentedInput.firstSegment ??=
            await this.segmentedInput.getFirstSegment({})),
          !this.segmentedInput.firstSegment)
        )
          throw new Error("Missing first segment, can't retrieve track.");
        let e = this.segmentedInput.firstSegment,
          t = null;
        for (
          ;
          e &&
          ((t =
            (await this.segmentedInput.getInputForSegment(e).getTracks()).find(
              (s) => s.type === this.decl.type && s.number === this.number,
            ) ?? null),
          !t);
        )
          e = await this.segmentedInput.getNextSegment(e, {});
        if (!t)
          throw new Error("No matching track found in underlying media data.");
        ((this.firstInputTrack = t), (this.firstSegment = e));
      })());
    }
    getId() {
      return this.decl.id;
    }
    getType() {
      return this.decl.type;
    }
    getNumber() {
      return this.number;
    }
    delegate(e) {
      return this.firstInputTrack ? e() : this.hydrate().then(e);
    }
    async getDecoderConfig() {
      return this.delegate(() =>
        this.firstInputTrack._backing.getDecoderConfig(),
      );
    }
    getHasOnlyKeyPackets() {
      return this.delegate(
        () => this.firstInputTrack._backing.getHasOnlyKeyPackets?.() ?? null,
      );
    }
    getPairingMask() {
      return 1n;
    }
    getCodec() {
      return this.delegate(() => this.firstInputTrack._backing.getCodec());
    }
    getInternalCodecId() {
      return this.delegate(() =>
        this.firstInputTrack._backing.getInternalCodecId(),
      );
    }
    getDisposition() {
      return this.delegate(() =>
        this.firstInputTrack._backing.getDisposition(),
      );
    }
    getLanguageCode() {
      return this.delegate(() =>
        this.firstInputTrack._backing.getLanguageCode(),
      );
    }
    getName() {
      return this.delegate(() => this.firstInputTrack._backing.getName());
    }
    getTimeResolution() {
      return this.delegate(() =>
        this.firstInputTrack._backing.getTimeResolution(),
      );
    }
    async isRelativeToUnixEpoch() {
      return (
        await this.hydrate(),
        g(this.segmentedInput.firstSegment),
        this.segmentedInput.firstSegment.unixEpochTimestamp ===
          this.segmentedInput.firstSegment.timestamp
      );
    }
    getUnixTimeForTimestamp(e) {
      return this.segmentedInput.getUnixTimeForTimestamp(e);
    }
    getBitrate() {
      return this.delegate(() => this.firstInputTrack._backing.getBitrate());
    }
    getAverageBitrate() {
      return this.delegate(() =>
        this.firstInputTrack._backing.getAverageBitrate(),
      );
    }
    getDurationFromMetadata(e) {
      return this.segmentedInput.getDurationFromMetadata(e);
    }
    getLiveRefreshInterval() {
      return this.segmentedInput.getLiveRefreshInterval();
    }
    async createAdjustedPacket(e, t, r) {
      (g(e.sequenceNumber >= 0), g(this.segmentedInput.firstSegment));
      const n = await this.segmentedInput.getMediaOffset(t, r.input),
        s = t.timestamp - this.segmentedInput.firstSegment.timestamp,
        a = e.clone({
          timestamp: li(e.timestamp + n, await r.getTimeResolution()),
          sequenceNumber: Math.floor(1e8 * s) + e.sequenceNumber,
        });
      return (
        this.packetInfos.set(a, { segment: t, track: r, sourcePacket: e }),
        a
      );
    }
    async getFirstPacket(e) {
      (await this.hydrate(), g(this.firstInputTrack), g(this.firstSegment));
      let t = this.firstInputTrack,
        r = this.firstSegment;
      for (;;) {
        if (t) {
          const a = await t._backing.getFirstPacket(e);
          if (a) return this.createAdjustedPacket(a, r, t);
        }
        if (
          ((r = await this.segmentedInput.getNextSegment(r, {
            skipLiveWait: e.skipLiveWait,
          })),
          !r)
        )
          break;
        t =
          (await this.segmentedInput.getInputForSegment(r).getTracks()).find(
            (a) =>
              a.type === this.firstInputTrack.type &&
              a.number === this.firstInputTrack.number,
          ) ?? null;
      }
      return null;
    }
    getNextPacket(e, t) {
      return this._getNextInternal(e, t, !1);
    }
    getNextKeyPacket(e, t) {
      return this._getNextInternal(e, t, !0);
    }
    async _getNextInternal(e, t, r) {
      const n = this.packetInfos.get(e);
      if (!n) throw new Error("Packet was not created from this track.");
      const s = r
        ? await n.track._backing.getNextKeyPacket(n.sourcePacket, t)
        : await n.track._backing.getNextPacket(n.sourcePacket, t);
      if (s) return this.createAdjustedPacket(s, n.segment, n.track);
      let a = n.segment;
      for (;;) {
        const o = await this.segmentedInput.getNextSegment(a, {
          skipLiveWait: t.skipLiveWait,
        });
        if (!o) return null;
        const u = (
          await this.segmentedInput.getInputForSegment(o).getTracks()
        ).find((f) => f.type === n.track.type && f.number === n.track.number);
        if (!u) {
          a = o;
          continue;
        }
        const d = await u._backing.getFirstPacket(t);
        return d ? this.createAdjustedPacket(d, o, u) : null;
      }
    }
    getPacket(e, t) {
      return this._getPacketInternal(e, t, !1);
    }
    getKeyPacket(e, t) {
      return this._getPacketInternal(e, t, !0);
    }
    async _getPacketInternal(e, t, r) {
      let n = await this.segmentedInput.getSegmentAt(e, {
        skipLiveWait: t.skipLiveWait,
      });
      if (!n) return null;
      for (await this.hydrate(); n;) {
        const s = this.segmentedInput.getInputForSegment(n),
          o = (await s.getTracks()).find(
            (d) =>
              d.type === this.firstInputTrack.type &&
              d.number === this.firstInputTrack.number,
          );
        if (!o) {
          n = await this.segmentedInput.getPreviousSegment(n, {
            skipLiveWait: t.skipLiveWait,
          });
          continue;
        }
        const c = await this.segmentedInput.getMediaOffset(n, s),
          l = e - c,
          u = r
            ? await o._backing.getKeyPacket(l, t)
            : await o._backing.getPacket(l, t);
        if (!u) {
          n = await this.segmentedInput.getPreviousSegment(n, {
            skipLiveWait: t.skipLiveWait,
          });
          continue;
        }
        return this.createAdjustedPacket(u, n, o);
      }
      return null;
    }
  }
  class Uo extends No {
    getType() {
      return "video";
    }
    getCodec() {
      return this.delegate(() => this.firstInputTrack._backing.getCodec());
    }
    getCodedWidth() {
      return this.delegate(() => this.firstInputTrack._backing.getCodedWidth());
    }
    getCodedHeight() {
      return this.delegate(() =>
        this.firstInputTrack._backing.getCodedHeight(),
      );
    }
    getSquarePixelWidth() {
      return this.delegate(() =>
        this.firstInputTrack._backing.getSquarePixelWidth(),
      );
    }
    getSquarePixelHeight() {
      return this.delegate(() =>
        this.firstInputTrack._backing.getSquarePixelHeight(),
      );
    }
    getTransformationMatrix() {
      return this.delegate(() =>
        this.firstInputTrack._backing.getTransformationMatrix(),
      );
    }
    async getColorSpace() {
      return this.delegate(() => this.firstInputTrack._backing.getColorSpace());
    }
    async canBeTransparent() {
      return this.delegate(() =>
        this.firstInputTrack._backing.canBeTransparent(),
      );
    }
    async getDecoderConfig() {
      return this.delegate(() =>
        this.firstInputTrack._backing.getDecoderConfig(),
      );
    }
  }
  class Wo extends No {
    getType() {
      return "audio";
    }
    getCodec() {
      return this.delegate(() => this.firstInputTrack._backing.getCodec());
    }
    getNumberOfChannels() {
      return this.delegate(() =>
        this.firstInputTrack._backing.getNumberOfChannels(),
      );
    }
    getSampleRate() {
      return this.delegate(() => this.firstInputTrack._backing.getSampleRate());
    }
    async getDecoderConfig() {
      return this.delegate(() =>
        this.firstInputTrack._backing.getDecoderConfig(),
      );
    }
  }
  Sn();
  const Vo = 0,
    Lo = 1 / 0;
  typeof FinalizationRegistry < "u" &&
    new FinalizationRegistry((i) => {
      i();
    });
  class nt extends Li {
    constructor() {
      (super(),
        (this._disposed = !1),
        (this._refCount = 0),
        (this._usedForHls = !1),
        (this._refFinalizationRegistry = null),
        (this._sizePromise = null),
        (this.onread = null),
        typeof FinalizationRegistry < "u" &&
          (this._refFinalizationRegistry = new FinalizationRegistry((e) => {
            e._decrementRefCount();
          })));
    }
    async getSizeOrNull() {
      if (this._disposed) throw new ve();
      return (this._sizePromise ??= (async () => {
        let e = this._getFileSize();
        return (
          e !== void 0 ||
            (await this._read(0, 1, Vo, Lo),
            (e = this._getFileSize()),
            g(e !== void 0)),
          e
        );
      })());
    }
    async getSize() {
      if (this._disposed) throw new ve();
      const e = await this.getSizeOrNull();
      if (e === null)
        throw new Error("Cannot determine the size of an unsized source.");
      return e;
    }
    slice(e, t) {
      if (!Number.isInteger(e) || e < 0)
        throw new TypeError("offset must be a non-negative integer.");
      if (t !== void 0 && (!Number.isInteger(t) || t < 0))
        throw new TypeError(
          "length, when provided, must be a non-negative integer.",
        );
      return new yd(this, e, t);
    }
    _dispatchRead(e, t) {
      (this.onread?.(e, t), this._emit("read", { start: e, end: t }));
    }
    ref() {
      return new os(this);
    }
    _incrementRefCount() {
      this._refCount++;
    }
    _decrementRefCount() {
      (this._refCount--,
        this._refCount === 0 && (this._dispose(), (this._disposed = !0)));
    }
  }
  class os {
    constructor(e) {
      if (((this._freed = !1), e._disposed))
        throw new Error("Cannot ref a disposed source.");
      (e._incrementRefCount(),
        e._refFinalizationRegistry?.register(this, e, this),
        (this._source = e));
    }
    get source() {
      if (!this._source)
        throw new Error("Can't get source; ref has already been freed.");
      return this._source;
    }
    get freed() {
      return this._freed;
    }
    free() {
      if (this._freed)
        throw new Error("Illegal operation: double free on SourceRef.");
      const e = this.source;
      (g(e._refCount > 0),
        e._decrementRefCount(),
        e._refFinalizationRegistry?.unregister(this),
        (this._freed = !0),
        (this._source = null));
    }
    [Symbol.dispose]() {
      this.freed || this.free();
    }
  }
  class ar extends nt {
    constructor(e, t) {
      if (typeof e != "string")
        throw new TypeError("rootPath must be a string.");
      if (typeof t != "function")
        throw new TypeError("requestHandler must be a function.");
      (super(), (this.rootPath = e), (this.requestHandler = t));
    }
    _resolveRequest(e) {
      const t = this.requestHandler(e),
        r = (n) => {
          if (!(n instanceof nt || n instanceof os))
            throw new TypeError(
              "requestHandler must return or resolve to a Source or SourceRef.",
            );
          const s = n instanceof nt ? n.ref() : n;
          return ((s.source._usedForHls ||= this._usedForHls), s);
        };
      return A(t) ? t.then(r) : r(t);
    }
  }
  const Ho = (i, e) => i.path === e.path;
  class md extends ar {
    constructor() {
      (super(...arguments), (this._root = null), (this._rootRequest = null));
    }
    _read(e, t, r, n) {
      if (!this._root) {
        if (!this._rootRequest) {
          const s = this._resolveRequest({ path: this.rootPath, isRoot: !0 }),
            a = (o) => {
              const c = o instanceof nt ? o.ref() : o;
              return ((this._root = c), (this._rootRequest = null), c);
            };
          A(s) ? (this._rootRequest = s.then(a)) : (a(s), g(this._root));
        }
        if (this._rootRequest)
          return this._rootRequest.then((s) => s.source._read(e, t, r, n));
      }
      return this._root.source._read(e, t, r, n);
    }
    _getFileSize() {
      if (this._root) return this._root.source._getFileSize();
    }
    _dispose() {
      this._root
        ? this._root.free()
        : this._rootRequest && this._rootRequest.then((e) => e.free());
    }
  }
  const qo =
    typeof FinalizationRegistry < "u"
      ? new FinalizationRegistry((i) => {
          i.cancel().catch(() => {});
        })
      : null;
  class cs extends nt {
    constructor(e, t = {}) {
      if (!(e instanceof Blob)) throw new TypeError("blob must be a Blob.");
      if (!t || typeof t != "object")
        throw new TypeError("options must be an object.");
      if (
        t.maxCacheSize !== void 0 &&
        (!gt(t.maxCacheSize) || t.maxCacheSize < 0)
      )
        throw new TypeError(
          "options.maxCacheSize, when provided, must be a non-negative number.",
        );
      if (t.useStreamReader !== void 0 && typeof t.useStreamReader != "boolean")
        throw new TypeError(
          "options.useStreamReader, when provided, must be a boolean.",
        );
      if (
        t.handleUnhandledError !== void 0 &&
        typeof t.handleUnhandledError != "function"
      )
        throw new TypeError(
          "options.handleUnhandledError, when provided, must be a function.",
        );
      (super(),
        (this._readers = new WeakMap()),
        (this._blob = e),
        (this._options = t),
        (this._orchestrator = new wd({
          maxCacheSize: t.maxCacheSize ?? 8 * 2 ** 20,
          maxWorkerCount: 4,
          runWorker: this._runWorker.bind(this),
          onIdleWorkerRemoved: (r) => {
            const n = this._readers.get(r);
            n &&
              (this._readers.delete(r),
              qo?.unregister(r),
              n.cancel().catch(() => {}));
          },
          prefetchProfile: gd.fileSystem,
          handleUnhandledError: t.handleUnhandledError,
        })),
        (this._orchestrator.fileSize = e.size));
    }
    _getFileSize() {
      return this._orchestrator.fileSize;
    }
    _read(e, t, r, n) {
      return this._orchestrator.read(e, t, r, n);
    }
    async _runWorker(e) {
      g(e.strictTarget);
      let t = this._readers.get(e);
      for (
        t === void 0 &&
        (("stream" in this._blob) &&
        !Ui() &&
        this._options.useStreamReader !== !1
          ? ((t = this._blob.slice(e.currentPos).stream().getReader()),
            qo?.register(e, t, e))
          : (t = null),
        this._readers.set(e, t));
        e.currentPos < e.targetPos && !e.aborted;
      )
        if (t) {
          const { done: r, value: n } = await t.read();
          if (r)
            throw (
              this._orchestrator.onWorkerFinished(e),
              new Error(
                "Blob reader stopped unexpectedly before all requested data was read.",
              )
            );
          if (e.aborted) break;
          (this._dispatchRead(e.currentPos, e.currentPos + n.length),
            this._orchestrator.supplyWorkerData(e, n));
        } else {
          const r = await this._blob
            .slice(e.currentPos, e.targetPos)
            .arrayBuffer();
          if (e.aborted) break;
          (this._dispatchRead(e.currentPos, e.currentPos + r.byteLength),
            this._orchestrator.supplyWorkerData(e, new Uint8Array(r)));
        }
      (this._orchestrator.signalWorkerStoppedRunning(e),
        e.aborted && (await t?.cancel().catch(() => {})));
    }
    _dispose() {
      this._orchestrator.dispose();
    }
  }
  class pd extends nt {
    constructor(e, t = {}) {
      if (!(e instanceof ReadableStream))
        throw new TypeError("stream must be a ReadableStream.");
      if (!t || typeof t != "object")
        throw new TypeError("options must be an object.");
      if (
        t.handleUnhandledError !== void 0 &&
        typeof t.handleUnhandledError != "function"
      )
        throw new TypeError(
          "options.handleUnhandledError, when provided, must be a function.",
        );
      if (
        t.maxCacheSize !== void 0 &&
        (!gt(t.maxCacheSize) || t.maxCacheSize < 0)
      )
        throw new TypeError(
          "options.maxCacheSize, when provided, must be a non-negative number.",
        );
      (super(),
        (this._reader = null),
        (this._cache = []),
        (this._pendingSlices = []),
        (this._currentIndex = 0),
        (this._targetIndex = 0),
        (this._maxRequestedIndex = 0),
        (this._endIndex = null),
        (this._pulling = !1),
        (this._cacheMissErrorMessage =
          "Attempted to read data from an already-evicted part of the cache. With ReadableStreamSource, you must access the data more sequentially or increase the size of its cache."),
        (this._stream = e),
        (this._maxCacheSize = t.maxCacheSize ?? 32 * 2 ** 20),
        (this._handleUnhandledError = t.handleUnhandledError));
    }
    _getFileSize() {
      return this._endIndex;
    }
    _read(e, t) {
      if (this._endIndex !== null && t > this._endIndex) return null;
      this._maxRequestedIndex = Math.max(this._maxRequestedIndex, t);
      const r = X(this._cache, e, (u) => u.start),
        n = r !== -1 ? this._cache[r] : null;
      if (n && n.start <= e && t <= n.end)
        return { bytes: n.bytes, view: n.view, offset: n.start };
      let s = e;
      const a = new Uint8Array(t - e);
      if (r !== -1)
        for (let u = r; u < this._cache.length; u++) {
          const d = this._cache[u];
          if (d.start >= t) break;
          const f = Math.max(e, d.start);
          f > s && this._throwDueToCacheMiss();
          const h = Math.min(t, d.end);
          f < h &&
            (a.set(d.bytes.subarray(f - d.start, h - d.start), f - e), (s = h));
        }
      if (s === t) return { bytes: a, view: j(a), offset: e };
      this._currentIndex > s && this._throwDueToCacheMiss();
      const { promise: o, resolve: c, reject: l } = ce();
      return (
        this._pendingSlices.push({
          start: e,
          end: t,
          bytes: a,
          resolve: c,
          reject: l,
        }),
        (this._targetIndex = Math.max(this._targetIndex, t)),
        this._pulling ||
          ((this._pulling = !0),
          this._pull().catch((u) => {
            if (((this._pulling = !1), this._pendingSlices.length > 0))
              (this._pendingSlices.forEach((d) => d.reject(u)),
                (this._pendingSlices.length = 0));
            else if (this._handleUnhandledError) this._handleUnhandledError(u);
            else throw u;
          })),
        o
      );
    }
    _throwDueToCacheMiss() {
      throw new Error(this._cacheMissErrorMessage);
    }
    async _pull() {
      for (
        this._reader ??= this._stream.getReader();
        this._currentIndex < this._targetIndex && !this._disposed;
      ) {
        const { done: e, value: t } = await this._reader.read();
        if (e) {
          for (const s of this._pendingSlices) s.resolve(null);
          ((this._pendingSlices.length = 0),
            (this._endIndex = this._currentIndex));
          break;
        }
        const r = this._currentIndex,
          n = this._currentIndex + t.byteLength;
        this._dispatchRead(r, n);
        for (let s = 0; s < this._pendingSlices.length; s++) {
          const a = this._pendingSlices[s],
            o = Math.max(r, a.start),
            c = Math.min(n, a.end);
          o < c &&
            (a.bytes.set(t.subarray(o - r, c - r), o - a.start),
            c === a.end &&
              (a.resolve({ bytes: a.bytes, view: j(a.bytes), offset: a.start }),
              this._pendingSlices.splice(s, 1),
              s--));
        }
        for (
          this._cache.push({ start: r, end: n, bytes: t, view: j(t), age: 0 });
          this._cache.length > 0;
        ) {
          const s = this._cache[0];
          if (this._maxRequestedIndex - s.end <= this._maxCacheSize) break;
          this._cache.shift();
        }
        this._currentIndex += t.byteLength;
      }
      this._pulling = !1;
    }
    _dispose() {
      for (const e of this._pendingSlices) e.reject(new ve());
      ((this._pendingSlices.length = 0),
        (this._cache.length = 0),
        this._reader?.cancel().catch(() => {}));
    }
  }
  const gd = {
    fileSystem: (i, e) => (
      (i = Math.floor((i - 65536) / 65536) * 65536),
      (e = Math.ceil((e + 65536) / 65536) * 65536),
      { start: i, end: e }
    ),
  };
  class wd {
    constructor(e) {
      ((this.options = e),
        (this.fileSize = null),
        (this.nextAge = 0),
        (this.workers = []),
        (this.cache = []),
        (this.currentCacheSize = 0),
        (this.disposed = !1),
        (this.queuedReads = []));
    }
    read(e, t, r, n) {
      g(!this.disposed);
      const s = this.options.prefetchProfile(e, t, this.workers),
        a = Math.max(s.start, r),
        o = Math.min(s.end, this.fileSize ?? 1 / 0, n);
      g(a <= e && t <= o);
      let c = null;
      const l = X(this.cache, e, (k) => k.start),
        u = l !== -1 ? this.cache[l] : null;
      u &&
        u.start <= e &&
        t <= u.end &&
        ((u.age = this.nextAge++),
        (c = { bytes: u.bytes, view: u.view, offset: u.start }));
      const d = X(this.cache, a, (k) => k.start),
        f = c ? null : new Uint8Array(t - e);
      let h = 0,
        p = a;
      const m = [];
      if (d !== -1) {
        for (let k = d; k < this.cache.length; k++) {
          const P = this.cache[k];
          if (P.start >= o) break;
          if (P.end <= a) continue;
          const I = Math.max(a, P.start),
            E = Math.min(o, P.end);
          if ((g(I <= E), p < I && m.push({ start: p, end: I }), (p = E), f)) {
            const C = Math.max(e, P.start),
              _ = Math.min(t, P.end);
            if (C < _) {
              const v = C - e;
              (f.set(P.bytes.subarray(C - P.start, _ - P.start), v),
                v === h && (h = _ - e));
            }
          }
          P.age = this.nextAge++;
        }
        p < o && m.push({ start: p, end: o });
      } else m.push({ start: a, end: o });
      if (
        (f && h >= f.length && (c = { bytes: f, view: j(f), offset: e }),
        m.length === 0)
      )
        return (g(c), c);
      const { promise: y, resolve: w, reject: b } = ce(),
        T = [];
      for (const k of m) {
        const P = Math.max(e, k.start),
          I = Math.min(t, k.end);
        P === k.start && I === k.end
          ? T.push(k)
          : P < I && T.push({ start: P, end: I });
      }
      const S = f && { start: e, bytes: f, holes: T, resolve: w, reject: b };
      e: for (const k of m) {
        for (const E of this.workers)
          if (this.checkHoleAgainstWorker(E, k, S ? [S] : [])) {
            this.checkQueuedReadsAgainstWorker(E);
            continue e;
          }
        const P = k.end < o || this.fileSize !== null,
          I = this.createWorker(k.start, k.end, P);
        if (I) (S && (I.pendingSlices = [S]), this.runWorker(I));
        else {
          let E = X(this.queuedReads, k.start, (_) => _.hole.start),
            C = E !== -1 ? this.queuedReads[E] : null;
          for (
            C && k.start <= C.hole.end
              ? ((C.hole.end = Math.max(C.hole.end, k.end)),
                (C.strictTarget &&= P),
                S && C.pendingSlices.push(S))
              : (E++,
                (C = {
                  hole: { start: k.start, end: k.end },
                  strictTarget: P,
                  pendingSlices: S ? [S] : [],
                  age: this.nextAge++,
                }),
                this.queuedReads.splice(E, 0, C));
            E + 1 < this.queuedReads.length;
          ) {
            const _ = this.queuedReads[E + 1];
            if (_.hole.start > C.hole.end) break;
            ((C.hole.end = Math.max(C.hole.end, _.hole.end)),
              C.pendingSlices.push(..._.pendingSlices),
              (C.strictTarget &&= _.strictTarget),
              (C.age = Math.min(C.age, _.age)),
              this.queuedReads.splice(E + 1, 1));
          }
        }
      }
      return (
        c
          ? y.catch((k) => {
              if (!this.disposed)
                if (this.options.handleUnhandledError)
                  this.options.handleUnhandledError(k);
                else throw k;
            })
          : (g(f),
            (c = y.then((k) => k && { bytes: k, view: j(k), offset: e }))),
        c
      );
    }
    checkHoleAgainstWorker(e, t, r) {
      if (ia(t.start - 131072, t.start, e.currentPos, e.targetPos)) {
        e.targetPos = Math.max(e.targetPos, t.end);
        for (let s = 0; s < r.length; s++) {
          const a = r[s];
          e.pendingSlices.includes(a) || e.pendingSlices.push(a);
        }
        return (e.running || this.runWorker(e), !0);
      }
      return !1;
    }
    checkQueuedReadsAgainstWorker(e) {
      let t = !1;
      for (let r = 0; r < this.queuedReads.length; r++) {
        const n = this.queuedReads[r];
        if (this.checkHoleAgainstWorker(e, n.hole, n.pendingSlices))
          (this.queuedReads.splice(r, 1), r--, (t = !0));
        else if (t) break;
      }
    }
    createWorker(e, t, r) {
      if (this.workers.length >= this.options.maxWorkerCount) {
        let s = null,
          a = null;
        for (let o = 0; o < this.workers.length; o++) {
          const c = this.workers[o];
          !c.running &&
            c.pendingSlices.length === 0 &&
            (!s || c.age < s.age) &&
            ((a = o), (s = c));
        }
        if (s)
          (g(a !== null),
            g(s.pendingSlices.length === 0),
            this.workers.splice(a, 1),
            this.options.onIdleWorkerRemoved?.(s));
        else return null;
      }
      const n = {
        startPos: e,
        currentPos: e,
        targetPos: t,
        strictTarget: r,
        running: !1,
        aborted: this.disposed,
        pendingSlices: [],
        age: this.nextAge++,
      };
      return (this.workers.push(n), n);
    }
    runWorker(e) {
      (g(!e.running),
        g(e.currentPos < e.targetPos),
        (e.running = !0),
        (e.age = this.nextAge++),
        this.options
          .runWorker(e)
          .catch((t) => {
            if (((e.running = !1), e.pendingSlices.length > 0))
              (e.pendingSlices.forEach((r) => r.reject(t)),
                (e.pendingSlices.length = 0));
            else if (!e.aborted && !this.disposed)
              if (this.options.handleUnhandledError)
                this.options.handleUnhandledError(t);
              else throw t;
          })
          .finally(() => {
            if (!e.running && this.queuedReads.length > 0) {
              let t = 0;
              for (let s = 1; s < this.queuedReads.length; s++)
                this.queuedReads[s].age < this.queuedReads[t].age && (t = s);
              const r = this.queuedReads[t],
                n = this.createWorker(r.hole.start, r.hole.end, r.strictTarget);
              if (!n) return;
              (this.queuedReads.splice(t, 1),
                (n.pendingSlices = r.pendingSlices),
                this.runWorker(n));
            }
          }));
    }
    supplyWorkerData(e, t) {
      g(!e.aborted);
      const r = e.currentPos,
        n = r + t.length;
      (this.insertIntoCache({
        start: r,
        end: n,
        bytes: t,
        view: j(t),
        age: this.nextAge++,
      }),
        (e.currentPos += t.length),
        e.currentPos > e.targetPos &&
          ((e.targetPos = e.currentPos),
          this.checkQueuedReadsAgainstWorker(e)));
      for (let s = 0; s < e.pendingSlices.length; s++) {
        const a = e.pendingSlices[s],
          o = Math.max(r, a.start),
          c = Math.min(n, a.start + a.bytes.length);
        o < c && a.bytes.set(t.subarray(o - r, c - r), o - a.start);
        for (let l = 0; l < a.holes.length; l++) {
          const u = a.holes[l];
          (r <= u.start && n > u.start && (u.start = n),
            u.end <= u.start && (a.holes.splice(l, 1), l--));
        }
        a.holes.length === 0 &&
          (a.resolve(a.bytes), e.pendingSlices.splice(s, 1), s--);
      }
      for (let s = 0; s < this.workers.length; s++) {
        const a = this.workers[s];
        e === a ||
          a.running ||
          (ia(r, n, a.currentPos, a.targetPos) &&
            (this.workers.splice(s, 1),
            this.options.onIdleWorkerRemoved?.(a),
            s--));
      }
    }
    supplyFileSize(e) {
      (g(this.fileSize === null), (this.fileSize = e));
      for (const t of this.workers) {
        ((t.targetPos = Math.min(t.targetPos, e)), (t.strictTarget = !0));
        for (let r = 0; r < t.pendingSlices.length; r++) {
          const n = t.pendingSlices[r];
          for (const s of n.holes)
            if (s.end > e) {
              (n.resolve(null), t.pendingSlices.splice(r, 1), r--);
              break;
            }
        }
      }
      for (let t = 0; t < this.queuedReads.length; t++) {
        const r = this.queuedReads[t];
        if (r.hole.start >= e) {
          for (const n of r.pendingSlices) n.resolve(null);
          (this.queuedReads.splice(t, 1), t--);
        } else if (r.hole.end > e) {
          ((r.hole.end = e), (r.strictTarget = !0));
          for (let n = 0; n < r.pendingSlices.length; n++) {
            const s = r.pendingSlices[n];
            s.start >= e &&
              (s.resolve(null), r.pendingSlices.splice(n, 1), n--);
          }
        }
      }
    }
    signalWorkerStoppedRunning(e) {
      ((e.running = !1), e.aborted || (e.pendingSlices.length = 0));
    }
    onWorkerFinished(e) {
      const t = this.workers.indexOf(e);
      (g(t !== -1),
        (e.running = !1),
        this.workers.splice(t, 1),
        this.options.onIdleWorkerRemoved?.(e),
        this.fileSize === null && this.supplyFileSize(e.currentPos));
      for (const r of e.pendingSlices) r.resolve(null);
    }
    insertIntoCache(e) {
      if (this.options.maxCacheSize === 0) return;
      let t = X(this.cache, e.start, (r) => r.start) + 1;
      if (t > 0) {
        const r = this.cache[t - 1];
        if (r.end >= e.end) return;
        if (r.end > e.start) {
          const n = new Uint8Array(e.end - r.start);
          (n.set(r.bytes, 0),
            n.set(e.bytes, e.start - r.start),
            (this.currentCacheSize += e.end - r.end),
            (r.bytes = n),
            (r.view = j(n)),
            (r.end = e.end),
            t--,
            (e = r));
        } else
          (this.cache.splice(t, 0, e),
            (this.currentCacheSize += e.bytes.length));
      } else
        (this.cache.splice(t, 0, e), (this.currentCacheSize += e.bytes.length));
      for (let r = t + 1; r < this.cache.length; r++) {
        const n = this.cache[r];
        if (e.end <= n.start) break;
        if (e.end >= n.end) {
          (this.cache.splice(r, 1),
            (this.currentCacheSize -= n.bytes.length),
            r--);
          continue;
        }
        const s = new Uint8Array(n.end - e.start);
        (s.set(e.bytes, 0),
          s.set(n.bytes, n.start - e.start),
          (this.currentCacheSize -= e.end - n.start),
          (e.bytes = s),
          (e.view = j(s)),
          (e.end = n.end),
          this.cache.splice(r, 1));
        break;
      }
      for (; this.currentCacheSize > this.options.maxCacheSize;) {
        let r = 0,
          n = this.cache[0];
        for (let s = 1; s < this.cache.length; s++) {
          const a = this.cache[s];
          a.age < n.age && ((r = s), (n = a));
        }
        if (this.currentCacheSize - n.bytes.length <= this.options.maxCacheSize)
          break;
        (this.cache.splice(r, 1), (this.currentCacheSize -= n.bytes.length));
      }
    }
    dispose() {
      for (const e of this.workers) {
        for (const t of e.pendingSlices) t.reject(new ve());
        ((e.pendingSlices.length = 0),
          (e.aborted = !0),
          e.running || this.options.onIdleWorkerRemoved?.(e));
      }
      for (const e of this.queuedReads)
        for (const t of e.pendingSlices) t.reject(new ve());
      ((this.workers.length = 0),
        (this.cache.length = 0),
        (this.queuedReads.length = 0),
        (this.disposed = !0));
    }
  }
  class yd extends nt {
    constructor(e, t, r) {
      if ((super(), (this._ref = null), e._disposed))
        throw new Error("Cannot create a slice of a disposed source.");
      ((this._baseSource = e), (this._offset = t), (this._length = r ?? null));
    }
    _getFileSize() {
      const e = this._baseSource._getFileSize();
      return e === void 0
        ? this._length !== null
          ? this._length
          : void 0
        : e === null
          ? this._length !== null
            ? this._length
            : null
          : se(e - this._offset, 0, this._length ?? 1 / 0);
    }
    _read(e, t, r, n) {
      if (this._length !== null && t > this._length) return null;
      const s = this._baseSource._read(
          this._offset + e,
          this._offset + t,
          this._offset + r,
          this._offset + n,
        ),
        a = (o) => (o ? ((o.offset -= this._offset), o) : null);
      return A(s) ? s.then(a) : a(s);
    }
    _dispose() {
      this._ref?.free();
    }
    ref() {
      return ((this._ref ??= this._baseSource.ref()), super.ref());
    }
  }
  var jo = function (i, e, t) {
      if (e != null) {
        if (typeof e != "object" && typeof e != "function")
          throw new TypeError("Object expected.");
        var r, n;
        if (t) {
          if (!Symbol.asyncDispose)
            throw new TypeError("Symbol.asyncDispose is not defined.");
          r = e[Symbol.asyncDispose];
        }
        if (r === void 0) {
          if (!Symbol.dispose)
            throw new TypeError("Symbol.dispose is not defined.");
          ((r = e[Symbol.dispose]), t && (n = r));
        }
        if (typeof r != "function")
          throw new TypeError("Object not disposable.");
        (n &&
          (r = function () {
            try {
              n.call(this);
            } catch (s) {
              return Promise.reject(s);
            }
          }),
          i.stack.push({ value: e, dispose: r, async: t }));
      } else t && i.stack.push({ async: !0 });
      return e;
    },
    $o = (function (i) {
      return function (e) {
        function t(a) {
          ((e.error = e.hasError
            ? new i(a, e.error, "An error was suppressed during disposal.")
            : a),
            (e.hasError = !0));
        }
        var r,
          n = 0;
        function s() {
          for (; (r = e.stack.pop());)
            try {
              if (!r.async && n === 1)
                return ((n = 0), e.stack.push(r), Promise.resolve().then(s));
              if (r.dispose) {
                var a = r.dispose.call(r.value);
                if (r.async)
                  return (
                    (n |= 2),
                    Promise.resolve(a).then(s, function (o) {
                      return (t(o), s());
                    })
                  );
              } else n |= 1;
            } catch (o) {
              t(o);
            }
          if (n === 1)
            return e.hasError ? Promise.reject(e.error) : Promise.resolve();
          if (e.hasError) throw e.error;
        }
        return s();
      };
    })(
      typeof SuppressedError == "function"
        ? SuppressedError
        : function (i, e, t) {
            var r = new Error(t);
            return (
              (r.name = "SuppressedError"),
              (r.error = i),
              (r.suppressed = e),
              r
            );
          },
    );
  const bd = /^0[xX][0-9a-fA-F]+$/,
    kd = /^data:.*;base64,/i;
  class Ko extends hd {
    constructor(e, t, r, n) {
      (super(e.input, t, r),
        (this.segments = []),
        (this.nextLines = null),
        (this.currentUpdateSegmentsPromise = null),
        (this.streamHasEnded = !1),
        (this.lastSegmentUpdateTime = -1 / 0),
        (this.refreshInterval = 5),
        (this.rootPath = t),
        (this.demuxer = e),
        (this.nextLines = n));
    }
    runUpdateSegments() {
      return (this.currentUpdateSegmentsPromise ??= (async () => {
        try {
          const e = this.getRemainingWaitTimeMs();
          (e > 0 && (await sa(e)),
            (this.lastSegmentUpdateTime = performance.now()),
            await this.updateSegments());
        } finally {
          this.currentUpdateSegmentsPromise = null;
        }
      })());
    }
    getRemainingWaitTimeMs() {
      const e = performance.now() - this.lastSegmentUpdateTime,
        t = Math.max(0, 1e3 * this.refreshInterval - e);
      return t <= 50 ? 0 : t;
    }
    async updateSegments() {
      let e = this.nextLines;
      if (((this.nextLines = null), !e)) {
        const S = { stack: [], error: void 0, hasError: !1 };
        try {
          const k = jo(
              S,
              await this.demuxer.input._getSourceUncached({
                path: this.rootPath,
                isRoot: !1,
              }),
              !1,
            ),
            I = await new $r(k.source).requestEntireFile();
          (g(I),
            (e = Fc(I, I.length, { ignore: Oo })),
            k.source instanceof ar && (this.rootPath = k.source.rootPath));
        } catch (k) {
          ((S.error = k), (S.hasError = !0));
        } finally {
          $o(S);
        }
      }
      const t =
          this.input._formatOptions.hls?.offsetTimestampsByDateTime !== !1,
        r = new zo(
          this.rootPath,
          this.demuxer.hasMasterPlaylist
            ? this.demuxer.masterPlaylistVariables
            : null,
        );
      let n = !1,
        s = 0,
        a = null,
        o = null,
        c = null,
        l = 0,
        u = null,
        d = null,
        f = null,
        h = null,
        p = null,
        m = null,
        y = !1,
        w = ne(this.segments) ?? null;
      const b = (S) => {
          const k = S.indexOf("@"),
            P = Number(k === -1 ? S : S.slice(0, k));
          if (!Number.isInteger(P) || P < 0)
            throw new Error(`Invalid #EXT-X-BYTERANGE length '${S}'.`);
          let I = null;
          if (
            k !== -1 &&
            ((I = Number(S.slice(k + 1))), !Number.isInteger(I) || I < 0)
          )
            throw new Error(`Invalid #EXT-X-BYTERANGE offset '${S}'.`);
          return { length: P, offset: I };
        },
        T = (S) => {
          ((l = S),
            w &&
              (g(w.sequenceNumber !== null),
              w.sequenceNumber < S &&
                ((s = w.timestamp + w.duration),
                (u = w.firstSegment),
                (d = w.initSegment),
                (p = w.lastProgramDateTimeSeconds),
                (a =
                  w.unixEpochTimestamp !== null
                    ? w.unixEpochTimestamp + w.duration
                    : null),
                (w = null))));
        };
      for (let S = 0; S < e.length; S++) {
        const k = e[S];
        if (!n) {
          if (k !== "#EXTM3U")
            throw new Error(
              "Invalid M3U8 file; expected first line to be #EXTM3U.",
            );
          n = !0;
          continue;
        }
        if (!k.startsWith("#")) {
          if (!w) {
            if (o === null)
              throw new Error(
                "Invalid M3U8 file; a segment must be preceded by an #EXTINF tag.",
              );
            let P = c;
            if (P && P.method === "AES-128" && !P.iv) {
              const _ = new Uint8Array(tt),
                v = j(_);
              (v.setUint32(8, Math.floor(l / 2 ** 32)),
                v.setUint32(12, l),
                (P = { ...P, iv: _ }));
            }
            const E = {
                path: vt(this.rootPath, r.substitute(k)),
                offset: h?.offset ?? 0,
                length: h?.length ?? null,
              },
              C = {
                timestamp: s,
                unixEpochTimestamp: a,
                firstSegment: u,
                sequenceNumber: l,
                location: E,
                duration: o,
                encryption: P,
                initSegment: d,
                lastProgramDateTimeSeconds: p,
              };
            ((u ??= C),
              (s += o),
              a !== null && (a += o),
              this.segments.push(C));
          }
          ((o = null), h === null ? (f = null) : (h = null), T(l + 1));
        }
        if (k.startsWith(as)) {
          if (w) {
            y = !0;
            continue;
          }
          y || (p === null && l > 0 && m !== null && (s = l * m), (y = !0));
          const P = k.slice(as.length),
            I = P.indexOf(","),
            E = I === -1 ? P : P.slice(0, I),
            C = Number(E);
          if (!Number.isFinite(C) || C < 0)
            throw new Error(`Invalid #EXTINF tag duration '${E}'.`);
          o = C;
        } else if (k.startsWith(Vr)) r.define(k.slice(Vr.length));
        else if (k.startsWith(vo)) {
          const P = new Ii(k.slice(vo.length), r),
            I = P.get("uri");
          if (!I)
            throw new Error("Invalid #EXT-X-MAP tag; missing URI attribute.");
          const E = P.get("byterange");
          let C = null;
          if ((E !== null && (C = b(E)), C && C.offset === null))
            throw new Error(
              "Invalid #EXT-X-MAP tag; BYTERANGE attribute must have a specified offset.",
            );
          if (!w) {
            const v = {
              path: vt(this.rootPath, I),
              offset: C?.offset ?? 0,
              length: C?.length ?? null,
            };
            if (c?.method === "AES-128" && !c.iv)
              throw new Error(
                "IV attribute must be set on #EXT-X-KEY tag preceding the #EXT-X-MAP tag.",
              );
            d = {
              timestamp: s,
              unixEpochTimestamp: a,
              firstSegment: null,
              sequenceNumber: null,
              location: v,
              duration: 0,
              encryption: c,
              initSegment: null,
              lastProgramDateTimeSeconds: p,
            };
          }
          ((o = null), h === null ? (f = null) : (h = null));
        } else if (k.startsWith(Ao)) {
          const P = new Ii(k.slice(Ao.length), r),
            I = P.get("method");
          if (I === "NONE") c = null;
          else if (I === "AES-128") {
            const E = P.get("uri");
            if (!E)
              throw new Error(
                "Invalid #EXT-X-KEY: AES-128 requires a URI attribute.",
              );
            let C = null;
            const _ = P.get("iv");
            if (_) {
              if (!bd.test(_)) throw new Error(`Unsupported IV format '${_}'.`);
              let F = _.slice(2);
              ((F = F.padStart(2 * tt, "0")),
                (F = F.slice(-2 * tt)),
                (C = new Uint8Array(tt)));
              for (let D = 0; D < tt; D++) {
                const z = 2 * D;
                C[D] = parseInt(F.slice(z, z + 2), 16);
              }
            }
            const v = P.get("keyformat") ?? "identity";
            if (v !== "identity")
              throw new Error(
                "For AES-128 encryption, only the 'identity' KEYFORMAT is currently supported. If you think other formats should be supported, please raise an issue.",
              );
            c = {
              method: "AES-128",
              keyUri: vt(this.rootPath, E),
              iv: C,
              keyFormat: v,
            };
          } else if (I === "SAMPLE-AES" || I === "SAMPLE-AES-CTR") {
            const E = P.get("uri");
            if (!E)
              throw new Error(
                `Invalid #EXT-X-KEY: ${I} requires a URI attribute.`,
              );
            if ((P.get("keyformat") ?? "identity") === "identity")
              throw new Error(
                "For SAMPLE-AES and SAMPLE-AES-CTR encryption, the 'identity' KEYFORMAT is not supported. If you think this format should be supported, please raise an issue.",
              );
            let _ = null;
            if (kd.test(E)) {
              const v = E.indexOf(","),
                F = ra(E.slice(v + 1));
              if (
                F.length >= 8 &&
                F[4] === 112 &&
                F[5] === 115 &&
                F[6] === 115 &&
                F[7] === 104
              ) {
                const D = j(F).getUint32(0);
                _ = qa(F.subarray(8, Math.min(D, F.length)));
              }
            }
            c = { method: I, psshBox: _ };
          } else
            throw new Error(
              `Unsupported encryption method '${I}'. If you think this method should be supported, please raise an issue.`,
            );
        } else if (k.startsWith(Bo)) {
          const P = k.slice(Bo.length),
            I = Number(P);
          if (!Number.isInteger(I) || I < 0)
            throw new Error(`Invalid EXT-X-MEDIA-SEQUENCE value '${P}'.`);
          T(I);
        } else if (k.startsWith(Fo)) {
          const P = b(k.slice(Fo.length));
          if (P.offset === null) {
            if (f === null)
              throw new Error(
                "Invalid M3U8 file; #EXT-X-BYTERANGE without offset requires a previous byte range.",
              );
            P.offset = f;
          }
          ((h = P), (f = P.offset + P.length));
        } else if (k.startsWith(Ro)) {
          if (w) continue;
          const P = k.slice(Ro.length),
            I = Date.parse(P);
          if (!Number.isFinite(I)) continue;
          const E = I / 1e3;
          if (p === E) continue;
          if (p === null && this.segments.length > 0) {
            const C = ne(this.segments),
              _ = C.timestamp + C.duration,
              v = E - _;
            for (const F of this.segments)
              ((F.unixEpochTimestamp = F.timestamp + v),
                t && (F.timestamp = F.unixEpochTimestamp));
          }
          ((p = E), (a = E), t && (s = E));
        } else if (k === cd) u = null;
        else if (k.startsWith(Mo)) {
          const P = k.slice(Mo.length),
            I = Number(P);
          if (!Number.isFinite(I) || I < 0)
            throw new Error(`Invalid EXT-X-TARGETDURATION value '${P}'.`);
          ((this.refreshInterval = I), (m = I));
        } else if (k === ld) {
          this.streamHasEnded = !0;
          break;
        } else
          k.startsWith(Do) &&
            k.slice(Do.length).toLowerCase() === "vod" &&
            (this.streamHasEnded = !0);
      }
      if (!n) throw new Error("Invalid M3U8 file; no #EXTM3U header.");
    }
    async getFirstSegment() {
      return (
        this.segments.length === 0 && (await this.runUpdateSegments()),
        this.segments[0] ?? null
      );
    }
    async getSegmentAt(e, t) {
      this.segments.length === 0 && (await this.runUpdateSegments());
      let r = !!t.skipLiveWait && this.getRemainingWaitTimeMs() > 0;
      for (;;) {
        const n = X(this.segments, e, (a) => a.timestamp);
        if (n === -1) return null;
        if (n < this.segments.length - 1 || this.streamHasEnded || r)
          return this.segments[n];
        const s = this.segments[n];
        if (e < s.timestamp + s.duration) return s;
        (await this.runUpdateSegments(), t.skipLiveWait && (r = !0));
      }
    }
    async getNextSegment(e, t) {
      const r = this.segments.indexOf(e);
      g(r !== -1);
      const n = r + 1;
      let s = !!t.skipLiveWait && this.getRemainingWaitTimeMs() > 0;
      for (;;) {
        if (n < this.segments.length) return this.segments[n];
        if (this.streamHasEnded || s) return null;
        (await this.runUpdateSegments(), t.skipLiveWait && (s = !0));
      }
    }
    async getPreviousSegment(e) {
      const t = this.segments.indexOf(e);
      return (g(t !== -1), this.segments[t - 1] ?? null);
    }
    getInputForSegment(e) {
      const t = e,
        r = this.inputCache.find((c) => c.segment === t);
      if (r) return ((r.age = this.nextInputCacheAge++), r.input);
      let n = null;
      (t.initSegment || t.firstSegment) &&
        (n = this.getInputForSegment(t.initSegment ?? t.firstSegment));
      const s = {
          ...this.input._formatOptions,
          isobmff: {
            ...this.input._formatOptions.isobmff,
            resolveKeyId:
              this.input._formatOptions.isobmff?.resolveKeyId &&
              ((c) => {
                if (
                  !t.encryption ||
                  !(
                    t.encryption.method === "SAMPLE-AES" ||
                    t.encryption.method === "SAMPLE-AES-CTR"
                  ) ||
                  !t.encryption.psshBox
                )
                  return this.input._formatOptions.isobmff.resolveKeyId(c);
                let l = c.psshBoxes;
                const { psshBox: u } = t.encryption;
                return (
                  (u.keyIds === null || u.keyIds.includes(c.keyId)) &&
                    !l.some((d) => ja(d, u)) &&
                    (l = [...l, u]),
                  this.input._formatOptions.isobmff.resolveKeyId({
                    ...c,
                    psshBoxes: l,
                  })
                );
              }),
          },
        },
        a = new ri({
          source: new md(t.location.path, async (c) => {
            g(c.isRoot);
            const l = { ...c, isRoot: !1 };
            let u;
            const d = t.location.offset > 0 || t.location.length !== null;
            if (
              !t.encryption ||
              t.encryption.method === "SAMPLE-AES" ||
              t.encryption.method === "SAMPLE-AES-CTR"
            ) {
              if (((u = await this.input._getSourceCached(l)), d)) {
                const h = u.source
                  .slice(t.location.offset, t.location.length ?? void 0)
                  .ref();
                (u.free(), (u = h));
              }
            } else if (t.encryption.method === "AES-128") {
              const f = t.encryption;
              g(f.iv);
              let h = await this.input._getSourceCached(l);
              if (d) {
                const w = h.source
                  .slice(t.location.offset, t.location.length ?? void 0)
                  .ref();
                (h.free(), (h = w));
              }
              const p = new $r(h.source),
                m = ku(
                  p,
                  async () => {
                    const y = { stack: [], error: void 0, hasError: !1 };
                    try {
                      const w = jo(
                          y,
                          await this.input._getSourceCached(
                            { path: f.keyUri, isRoot: !1 },
                            df,
                          ),
                          !1,
                        ),
                        T = await new $r(w.source).requestSlice(0, tt);
                      if (!T)
                        throw new Error(
                          "Invalid AES-128 key; expected at least 16 bytes of data.",
                        );
                      return { key: N(T, tt), iv: f.iv };
                    } catch (w) {
                      ((y.error = w), (y.hasError = !0));
                    } finally {
                      $o(y);
                    }
                  },
                  () => {
                    h.free();
                  },
                );
              u = new pd(m).ref();
            } else g(!1);
            return u;
          }),
          formats: this.input._formats.filter((c) => !(c instanceof tc)),
          initInput: n ?? void 0,
          formatOptions: s,
        });
      if (
        ((a._onFormatDetermined = (c) => {
          if (
            (t.encryption?.method === "SAMPLE-AES" ||
              t.encryption?.method === "SAMPLE-AES-CTR") &&
            !c._isIsobmff
          )
            throw new Error(
              "The SAMPLE-AES and SAMPLE-AES-CTR encryption methods are currently only supported for ISOBMFF files.",
            );
        }),
        this.inputCache.push({
          segment: t,
          input: a,
          age: this.nextInputCacheAge++,
        }),
        this.inputCache.length > 4)
      ) {
        const c = xn(this.inputCache, (l) => l.age);
        (g(c !== -1), this.inputCache.splice(c, 1));
      }
      return a;
    }
    async getLiveRefreshInterval() {
      return (
        this.getRemainingWaitTimeMs() === 0 && (await this.runUpdateSegments()),
        this.streamHasEnded ? null : this.refreshInterval
      );
    }
  }
  class Td extends bt {
    constructor(e) {
      (super(e),
        (this.metadataPromise = null),
        (this.trackBackings = null),
        (this.internalTracks = null),
        (this.segmentedInputs = []),
        (this.hasMasterPlaylist = !0),
        (this.masterPlaylistVariables = null));
    }
    readMetadata() {
      return (this.metadataPromise ??= (async () => {
        g(this.input._rootSource instanceof ar);
        const e = await this.input._reader.requestEntireFile();
        g(e);
        const t = Fc(e, e.length, { ignore: Oo }),
          { rootPath: r } = this.input._rootSource,
          n = new zo(r, null),
          s = [],
          a = [];
        for (let u = 1; u < t.length; u++) {
          const d = t[u];
          if (d.startsWith(Vr)) n.define(d.slice(Vr.length));
          else if (d.startsWith(Io)) {
            const f = u;
            let h = t[++u];
            if (h === void 0)
              throw new Error(
                "Incorrect M3U8 file; a line must follow the #EXT-X-STREAM-INF tag.",
              );
            h = n.substitute(h);
            const p = vt(r, h),
              m = new Ii(d.slice(Io.length), n);
            if (m.getAsNumber("bandwidth") === null)
              throw new Error(
                "Invalid M3U8 file; #EXT-X-STREAM-INF tag requires a BANDWIDTH attribute with a valid numerical value.",
              );
            s.push({
              fullPath: p,
              attributes: m,
              lineNumber: f,
              hasOnlyKeyPackets: !1,
            });
          } else if (d.startsWith(Eo)) {
            const f = new Ii(d.slice(Eo.length), n),
              h = f.get("uri");
            if (h === null)
              throw new Error(
                "Invalid M3U8 file; #EXT-X-I-FRAME-STREAM-INF tag requires a URI attribute.",
              );
            if (f.getAsNumber("bandwidth") === null)
              throw new Error(
                "Invalid M3U8 file; #EXT-X-I-FRAME-STREAM-INF tag requires a BANDWIDTH attribute with a valid numerical value.",
              );
            const m = vt(r, h);
            s.push({
              fullPath: m,
              attributes: f,
              lineNumber: u,
              hasOnlyKeyPackets: !0,
            });
          } else if (d.startsWith(_o)) {
            const f = new Ii(d.slice(_o.length), n);
            if (f.get("type") === null)
              throw new Error(
                "Invalid M3U8 file; #EXT-X-MEDIA tag requires a TYPE attribute.",
              );
            if (f.get("group-id") === null)
              throw new Error(
                "Invalid M3U8 file; #EXT-X-MEDIA tag requires a GROUP-ID attribute.",
              );
            let m = null;
            const y = f.get("uri");
            (y !== null && (m = vt(r, y)),
              a.push({ fullPath: m, attributes: f, lineNumber: u }));
          } else if (d !== ud) {
            if (d.startsWith(as)) {
              const f = new Ko(this, r, null, t);
              ((this.segmentedInputs = [f]),
                (this.hasMasterPlaylist = !1),
                (this.trackBackings = await f.getTrackBackings()));
              return;
            }
          }
        }
        this.masterPlaylistVariables = n;
        const o = await Promise.all(
            s.map(async (u, d) => {
              const f = [],
                h = u.attributes.get("codecs");
              let p;
              if (h) p = h.split(",").map((E) => E.trim());
              else {
                const C = await this.getSegmentedInputForPath(
                    u.fullPath,
                  ).getTrackBackings(),
                  _ = await Promise.all(
                    C.map(async (v) => ({
                      track: v,
                      codec: await v.getCodec(),
                    })),
                  );
                p = await Promise.all(
                  _.filter((v) => v.codec !== null).map((v) =>
                    v.track.getDecoderConfig().then((F) => F.codec),
                  ),
                );
              }
              const m = u.attributes.get("video"),
                y = u.attributes.get("audio"),
                w = p.some((E) => Fe.includes(yt(E))),
                b = p.some((E) => Ne.includes(yt(E)));
              if (m !== null && !w) {
                const E = a.find((C) => {
                  const _ = C.attributes.get("group-id"),
                    v = C.attributes.get("type");
                  return _ === m && v.toLowerCase() === "video";
                });
                e: if (E) {
                  const C = E.attributes.get("uri");
                  if (C === null) break e;
                  const _ = vt(r, C),
                    D = (
                      await this.getSegmentedInputForPath(_).getTrackBackings()
                    ).find((O) => O.getType() === "video");
                  if (!D || (await D.getCodec()) === null) break e;
                  const z = await D.getDecoderConfig().then(
                    (O) => O?.codec ?? null,
                  );
                  (g(z !== null), p.push(z));
                }
              }
              if (y !== null && !b) {
                const E = a.find((C) => {
                  const _ = C.attributes.get("group-id"),
                    v = C.attributes.get("type");
                  return _ === y && v.toLowerCase() === "audio";
                });
                e: if (E) {
                  const C = E.attributes.get("uri");
                  if (C === null) break e;
                  const _ = vt(r, C),
                    D = (
                      await this.getSegmentedInputForPath(_).getTrackBackings()
                    ).find((O) => O.getType() === "audio");
                  if (!D || (await D.getCodec()) === null) break e;
                  const z = await D.getDecoderConfig().then(
                    (O) => O?.codec ?? null,
                  );
                  (g(z !== null), p.push(z));
                }
              }
              p = [...new Set(p)];
              let T = null,
                S = null;
              const k = u.attributes.getAsNumber("bandwidth");
              g(k !== null);
              const P = u.attributes.getAsNumber("average-bandwidth"),
                I = u.attributes.get("name");
              for (const E of p) {
                const C = yt(E);
                if (C !== null) {
                  if (Fe.includes(C)) {
                    if (T !== null)
                      throw new Error(
                        "Unsupported M3U8 file; multiple video codecs found in the CODECS attribute of a variant stream.",
                      );
                    T = E;
                    const _ = u.attributes.get("video");
                    if (_ === null) {
                      const v = u.attributes.get("resolution");
                      let F = null,
                        D = null;
                      if (v) {
                        const z = v.match(/^(\d+)x(\d+)$/);
                        z && ((F = Number(z[1])), (D = Number(z[2])));
                      }
                      f.push({
                        id: -1,
                        demuxer: this,
                        backingTrack: null,
                        default: !0,
                        autoselect: !0,
                        languageCode: Pe,
                        lineNumber: u.lineNumber,
                        fullPath: u.fullPath,
                        fullCodecString: T,
                        pairingMask: 1n << BigInt(d),
                        peakBitrate: k,
                        averageBitrate: P,
                        name: I,
                        hasOnlyKeyPackets: u.hasOnlyKeyPackets,
                        info: { type: "video", width: F, height: D },
                      });
                    } else
                      for (const v of a) {
                        const F = v.attributes.get("group-id"),
                          D = v.attributes.get("type");
                        if (F !== _ || D.toLowerCase() !== "video") continue;
                        const z =
                          v.attributes.get("resolution") ??
                          u.attributes.get("resolution");
                        let O = null,
                          H = null;
                        if (z) {
                          const ee = z.match(/^(\d+)x(\d+)$/);
                          ee && ((O = Number(ee[1])), (H = Number(ee[2])));
                        }
                        f.push({
                          id: -1,
                          demuxer: this,
                          backingTrack: null,
                          default: Lr(v.attributes),
                          autoselect: Lr(v.attributes) || Yo(v.attributes),
                          languageCode: Zo(v.attributes.get("language")),
                          lineNumber: v.lineNumber,
                          fullPath: v.fullPath ?? u.fullPath,
                          fullCodecString: T,
                          pairingMask: 1n << BigInt(d),
                          peakBitrate: v.fullPath === null ? k : null,
                          averageBitrate: v.fullPath === null ? P : null,
                          name: v.attributes.get("name"),
                          hasOnlyKeyPackets: u.hasOnlyKeyPackets,
                          info: { type: "video", width: O, height: H },
                        });
                      }
                  } else if (Ne.includes(C)) {
                    if (S !== null)
                      throw new Error(
                        "Unsupported M3U8 file; multiple audio codecs found in the CODECS attribute of a variant stream.",
                      );
                    S = E;
                    const _ = u.attributes.get("audio");
                    if (_ === null) {
                      const v = u.attributes.get("channels"),
                        F = v !== null ? Number(v.split("/")[0]) : null;
                      f.push({
                        id: -1,
                        demuxer: this,
                        backingTrack: null,
                        default: !0,
                        autoselect: !0,
                        languageCode: Pe,
                        lineNumber: u.lineNumber,
                        fullPath: u.fullPath,
                        fullCodecString: S,
                        pairingMask: 1n << BigInt(d),
                        peakBitrate: k,
                        averageBitrate: P,
                        name: I,
                        hasOnlyKeyPackets: u.hasOnlyKeyPackets,
                        info: {
                          type: "audio",
                          numberOfChannels:
                            F !== null && Number.isInteger(F) && F > 0
                              ? F
                              : null,
                        },
                      });
                    } else
                      for (const v of a) {
                        const F = v.attributes.get("group-id"),
                          D = v.attributes.get("type");
                        if (F !== _ || D.toLowerCase() !== "audio") continue;
                        const z =
                            v.attributes.get("channels") ??
                            u.attributes.get("channels"),
                          O = z !== null ? Number(z.split("/")[0]) : null;
                        f.push({
                          id: -1,
                          demuxer: this,
                          backingTrack: null,
                          default: Lr(v.attributes),
                          autoselect: Lr(v.attributes) || Yo(v.attributes),
                          languageCode: Zo(v.attributes.get("language")),
                          lineNumber: v.lineNumber,
                          fullPath: v.fullPath ?? u.fullPath,
                          fullCodecString: S,
                          pairingMask: 1n << BigInt(d),
                          peakBitrate: v.fullPath === null ? k : null,
                          averageBitrate: v.fullPath === null ? P : null,
                          name: v.attributes.get("name"),
                          hasOnlyKeyPackets: u.hasOnlyKeyPackets,
                          info: {
                            type: "audio",
                            numberOfChannels:
                              O !== null && Number.isInteger(O) && O > 0
                                ? O
                                : null,
                          },
                        });
                      }
                  }
                }
              }
              return f;
            }),
          ),
          c = [],
          l = (u) => {
            const d = c.find(
              (f) => f.fullPath === u.fullPath && f.info.type === u.info.type,
            );
            d
              ? ((d.pairingMask |= u.pairingMask),
                (d.default ||= u.default),
                (d.autoselect ||= u.autoselect),
                (d.lineNumber = Math.min(d.lineNumber, u.lineNumber)),
                u.peakBitrate !== null &&
                  (d.peakBitrate = Math.max(
                    d.peakBitrate ?? -1 / 0,
                    u.peakBitrate,
                  )),
                u.averageBitrate !== null &&
                  (d.averageBitrate = Math.max(
                    d.averageBitrate ?? -1 / 0,
                    u.averageBitrate,
                  )),
                d.languageCode === Pe && (d.languageCode = u.languageCode))
              : ((u.id = c.length + 1), c.push(u));
          };
        for (const u of o) for (const d of u) l(d);
        (c.sort((u, d) => u.lineNumber - d.lineNumber),
          (this.trackBackings = []));
        for (const u of c)
          u.info.type === "video"
            ? this.trackBackings.push(new Xo(u))
            : this.trackBackings.push(new Qo(u));
        this.internalTracks = c;
      })());
    }
    async getTrackBackings() {
      return (
        await this.readMetadata(),
        g(this.trackBackings),
        this.trackBackings
      );
    }
    getSegmentedInputForPath(e) {
      let t = this.segmentedInputs.find((n) => n.path === e);
      if (t) return t;
      let r = null;
      return (
        this.internalTracks &&
          (r = this.internalTracks
            .filter((s) => s.fullPath === e)
            .map((s) => ({ id: s.id, type: s.info.type }))),
        (t = new Ko(this, e, r, null)),
        this.segmentedInputs.push(t),
        t
      );
    }
    async getMetadataTags() {
      return {};
    }
    async getMimeType() {
      return Co;
    }
    dispose() {
      if (this.segmentedInputs) {
        for (const e of this.segmentedInputs) e.dispose();
        this.segmentedInputs.length = 0;
      }
    }
  }
  class Go {
    constructor(e) {
      ((this.internalTrack = e), (this.hydrationPromise = null));
    }
    hydrate() {
      return (this.hydrationPromise ??= (async () => {
        const e = this.internalTrack.demuxer.getSegmentedInputForPath(
          this.internalTrack.fullPath,
        );
        let t = null;
        const n = (await e.getTrackBackings()).filter(
          (s) => s.getType() === this.getType(),
        );
        if (n.length === 1) t = n[0];
        else if (this instanceof Xo) {
          for (const s of n)
            if ((await s.getCodec()) === this.getCodec()) {
              t = s;
              break;
            }
        } else {
          g(this instanceof Qo);
          for (const s of n)
            if ((await s.getCodec()) === this.getCodec()) {
              t = s;
              break;
            }
        }
        if (!t)
          throw new Error(
            "Could not find matching track in underlying media data.",
          );
        this.internalTrack.backingTrack = t;
      })());
    }
    delegate(e) {
      return this.internalTrack.backingTrack ? e() : this.hydrate().then(e);
    }
    getCodec() {
      throw new Error("Not implemented on base class.");
    }
    getDisposition() {
      return {
        ...wt,
        default: this.internalTrack.autoselect,
        primary: this.internalTrack.default,
      };
    }
    getId() {
      return this.internalTrack.id;
    }
    getPairingMask() {
      return this.internalTrack.pairingMask;
    }
    getInternalCodecId() {
      return null;
    }
    getLanguageCode() {
      return this.internalTrack.languageCode;
    }
    getName() {
      return this.internalTrack.name;
    }
    getNumber() {
      g(this.internalTrack.demuxer.internalTracks);
      const e = this.internalTrack.info.type;
      let t = 0;
      for (const r of this.internalTrack.demuxer.internalTracks)
        if ((r.info.type === e && t++, r === this.internalTrack)) break;
      return t;
    }
    getTimeResolution() {
      return this.delegate(() =>
        this.internalTrack.backingTrack.getTimeResolution(),
      );
    }
    isRelativeToUnixEpoch() {
      return this.delegate(() =>
        this.internalTrack.backingTrack.isRelativeToUnixEpoch(),
      );
    }
    getUnixTimeForTimestamp(e) {
      return this.delegate(() =>
        this.internalTrack.backingTrack.getUnixTimeForTimestamp(e),
      );
    }
    getBitrate() {
      return this.internalTrack.peakBitrate;
    }
    getAverageBitrate() {
      return this.internalTrack.averageBitrate;
    }
    async getDurationFromMetadata(e) {
      return (
        await this.hydrate(),
        this.internalTrack.backingTrack.getDurationFromMetadata(e)
      );
    }
    async getLiveRefreshInterval() {
      return (
        await this.hydrate(),
        this.internalTrack.backingTrack.getLiveRefreshInterval()
      );
    }
    getHasOnlyKeyPackets() {
      return this.internalTrack.hasOnlyKeyPackets || null;
    }
    async getFirstPacket(e) {
      return (
        await this.hydrate(),
        this.internalTrack.backingTrack.getFirstPacket(e)
      );
    }
    async getPacket(e, t) {
      return (
        await this.hydrate(),
        this.internalTrack.backingTrack.getPacket(e, t)
      );
    }
    async getKeyPacket(e, t) {
      return (
        await this.hydrate(),
        this.internalTrack.backingTrack.getKeyPacket(e, t)
      );
    }
    async getNextPacket(e, t) {
      return (
        await this.hydrate(),
        this.internalTrack.backingTrack.getNextPacket(e, t)
      );
    }
    async getNextKeyPacket(e, t) {
      return (
        await this.hydrate(),
        this.internalTrack.backingTrack.getNextKeyPacket(e, t)
      );
    }
  }
  class Xo extends Go {
    constructor(e) {
      super(e);
    }
    get backingVideoTrack() {
      return this.internalTrack.backingTrack;
    }
    getType() {
      return "video";
    }
    getCodec() {
      return yt(this.internalTrack.fullCodecString);
    }
    getCodedWidth() {
      return this.delegate(() => this.backingVideoTrack.getCodedWidth());
    }
    getCodedHeight() {
      return this.delegate(() => this.backingVideoTrack.getCodedHeight());
    }
    getSquarePixelWidth() {
      return this.delegate(() => this.backingVideoTrack.getSquarePixelWidth());
    }
    getSquarePixelHeight() {
      return this.delegate(() => this.backingVideoTrack.getSquarePixelHeight());
    }
    getMetadataDisplayWidth() {
      return this.backingVideoTrack ? null : this.internalTrack.info.width;
    }
    getMetadataDisplayHeight() {
      return this.backingVideoTrack ? null : this.internalTrack.info.height;
    }
    getTransformationMatrix() {
      return this.delegate(() =>
        this.backingVideoTrack.getTransformationMatrix(),
      );
    }
    async getColorSpace() {
      return (await this.hydrate(), this.backingVideoTrack.getColorSpace());
    }
    async canBeTransparent() {
      return (await this.hydrate(), this.backingVideoTrack.canBeTransparent());
    }
    getMetadataCodecParameterString() {
      return this.backingVideoTrack ? null : this.internalTrack.fullCodecString;
    }
    async getDecoderConfig() {
      return (await this.hydrate(), this.backingVideoTrack.getDecoderConfig());
    }
  }
  class Qo extends Go {
    constructor(e) {
      super(e);
    }
    get backingAudioTrack() {
      return this.internalTrack.backingTrack;
    }
    getType() {
      return "audio";
    }
    getCodec() {
      return yt(this.internalTrack.fullCodecString);
    }
    getNumberOfChannels() {
      return this.internalTrack.info.numberOfChannels !== null
        ? this.internalTrack.info.numberOfChannels
        : this.delegate(() => this.backingAudioTrack.getNumberOfChannels());
    }
    getSampleRate() {
      return this.delegate(() => this.backingAudioTrack.getSampleRate());
    }
    getMetadataCodecParameterString() {
      return this.backingAudioTrack ? null : this.internalTrack.fullCodecString;
    }
    async getDecoderConfig() {
      return (await this.hydrate(), this.backingAudioTrack.getDecoderConfig());
    }
  }
  const Lr = (i) => {
      const e = i.get("default");
      if (e === null) return !1;
      const t = e.toUpperCase();
      if (t === "YES") return !0;
      if (t === "NO") return !1;
      throw new Error(
        `Invalid M3U8 file; #EXT-X-MEDIA DEFAULT attribute must be YES or NO, got "${e}".`,
      );
    },
    Yo = (i) => {
      const e = i.get("autoselect");
      if (e === null) return !1;
      const t = e.toUpperCase();
      if (t === "YES") return !0;
      if (t === "NO") return !1;
      throw new Error(
        `Invalid M3U8 file; #EXT-X-MEDIA AUTOSELECT attribute must be YES or NO, got "${e}".`,
      );
    },
    Zo = (i) => {
      if (i === null) return Pe;
      const e = i.split("-")[0];
      return e || Pe;
    };
  class st {
    constructor() {
      this._isIsobmff = !1;
    }
  }
  class Jo extends st {
    constructor() {
      (super(...arguments), (this._isIsobmff = !0));
    }
    async _getMajorBrand(e) {
      let t = e._reader.requestSlice(0, 12);
      if ((A(t) && (t = await t), !t)) return null;
      t.skip(4);
      const r = le(t, 4);
      return r !== "ftyp" && r !== "styp" ? null : le(t, 4);
    }
    _createDemuxer(e) {
      return new Gn(e);
    }
  }
  class Sd extends Jo {
    async _canReadInput(e) {
      const t = await this._getMajorBrand(e);
      if (t !== null) return t !== "qt  ";
      let r = 0;
      for (let n = 0; n < 10; n++) {
        let s = e._reader.requestSlice(r, 8);
        if ((A(s) && (s = await s), !s)) return !1;
        let a = R(s),
          o = 8;
        if (a === 1) {
          let l = e._reader.requestSlice(r + 8, 8);
          if ((A(l) && (l = await l), !l)) return !1;
          ((a = Le(l)), (o = 16));
        }
        if (a < o) return !1;
        const c = le(s, 4);
        if (c === "moof" || c === "sidx") return !0;
        if (c === "emsg" || c === "prft" || c === "free") r += a;
        else return !1;
      }
      return !1;
    }
    get name() {
      return "MP4";
    }
    get mimeType() {
      return "video/mp4";
    }
  }
  class xd extends Jo {
    async _canReadInput(e) {
      return (await this._getMajorBrand(e)) === "qt  ";
    }
    get name() {
      return "QuickTime File Format";
    }
    get mimeType() {
      return "video/quicktime";
    }
  }
  class ec extends st {
    async isSupportedEBMLOfDocType(e, t) {
      let r = e._reader.requestSlice(0, it);
      if ((A(r) && (r = await r), !r)) return !1;
      const n = uo(r);
      if (n === null || n < 1 || n > 8 || K(r, n) !== x.EBML) return !1;
      const a = fo(r);
      if (typeof a != "number") return !1;
      let o = e._reader.requestSlice(r.filePos, a);
      if ((A(o) && (o = await o), !o)) return !1;
      const c = r.filePos;
      for (; o.filePos <= c + a - We;) {
        const l = rt(o);
        if (!l) break;
        const { id: u, size: d } = l,
          f = o.filePos;
        if (d === void 0) return !1;
        switch (u) {
          case x.EBMLVersion:
            if (K(o, d) !== 1) return !1;
            break;
          case x.EBMLReadVersion:
            if (K(o, d) !== 1) return !1;
            break;
          case x.DocType:
            if (Si(o, d) !== t) return !1;
            break;
          case x.DocTypeVersion:
            if (K(o, d) > 4) return !1;
            break;
        }
        o.filePos = f + d;
      }
      return !0;
    }
    _canReadInput(e) {
      return this.isSupportedEBMLOfDocType(e, "matroska");
    }
    _createDemuxer(e) {
      return new Au(e);
    }
    get name() {
      return "Matroska";
    }
    get mimeType() {
      return "video/x-matroska";
    }
  }
  class Pd extends ec {
    _canReadInput(e) {
      return this.isSupportedEBMLOfDocType(e, "webm");
    }
    get name() {
      return "WebM";
    }
    get mimeType() {
      return "video/webm";
    }
  }
  class Cd extends st {
    async _canReadInput(e) {
      let t = 0;
      for (;;) {
        let d = e._reader.requestSlice(t, je);
        if ((A(d) && (d = await d), !d)) break;
        const f = Et(d);
        if (!f) break;
        t = d.filePos + f.size;
      }
      const r = await is(e._reader, t, t + 4096);
      if (!r) return !1;
      const n = r.header,
        s = La(n.mpegVersionId, n.channel);
      let a = e._reader.requestSlice(r.startPos + s, 4);
      if ((A(a) && (a = await a), !a)) return !1;
      const o = R(a);
      if (o === Wa || o === Va) return !0;
      t = r.startPos + r.header.totalSize;
      const l = await is(e._reader, t, t + Gt);
      if (!l) return !1;
      const u = l.header;
      return !(n.channel !== u.channel || n.sampleRate !== u.sampleRate);
    }
    _createDemuxer(e) {
      return new Ru(e);
    }
    get name() {
      return "MP3";
    }
    get mimeType() {
      return "audio/mpeg";
    }
  }
  class Id extends st {
    async _canReadInput(e) {
      let t = e._reader.requestSlice(0, 12);
      if ((A(t) && (t = await t), !t)) return !1;
      const r = le(t, 4);
      return r !== "RIFF" && r !== "RIFX" && r !== "RF64"
        ? !1
        : (t.skip(4), le(t, 4) === "WAVE");
    }
    _createDemuxer(e) {
      return new Lu(e);
    }
    get name() {
      return "WAVE";
    }
    get mimeType() {
      return "audio/wav";
    }
  }
  class Ed extends st {
    async _canReadInput(e) {
      let t = e._reader.requestSlice(0, 4);
      return (A(t) && (t = await t), t ? le(t, 4) === "OggS" : !1);
    }
    _createDemuxer(e) {
      return new Wu(e);
    }
    get name() {
      return "Ogg";
    }
    get mimeType() {
      return "application/ogg";
    }
  }
  class _d extends st {
    async _canReadInput(e) {
      let t = 0;
      for (;;) {
        let n = e._reader.requestSlice(t, je);
        if ((A(n) && (n = await n), !n)) break;
        const s = Et(n);
        if (!s) break;
        t = n.filePos + s.size;
      }
      let r = e._reader.requestSlice(t, 4);
      return (A(r) && (r = await r), r ? le(r, 4) === "fLaC" : !1);
    }
    get name() {
      return "FLAC";
    }
    get mimeType() {
      return "audio/flac";
    }
    _createDemuxer(e) {
      return new Zu(e);
    }
  }
  class vd extends st {
    async _canReadInput(e) {
      let t = 0;
      for (;;) {
        let a = e._reader.requestSlice(t, je);
        if ((A(a) && (a = await a), !a)) break;
        const o = Et(a);
        if (!o) break;
        t = a.filePos + o.size;
      }
      let r = e._reader.requestSliceRange(t, nr, Ot);
      if ((A(r) && (r = await r), !r)) return !1;
      const n = Pt(r);
      if (
        !n ||
        ((t += n.frameLength),
        (r = e._reader.requestSliceRange(t, nr, Ot)),
        A(r) && (r = await r),
        !r)
      )
        return !1;
      const s = Pt(r);
      return s
        ? n.objectType === s.objectType &&
            n.samplingFrequencyIndex === s.samplingFrequencyIndex &&
            n.channelConfiguration === s.channelConfiguration
        : !1;
    }
    _createDemuxer(e) {
      return new qu(e);
    }
    get name() {
      return "ADTS";
    }
    get mimeType() {
      return "audio/aac";
    }
  }
  class Ad extends st {
    async _canReadInput(e) {
      const t = Ve + 16 + 1;
      let r = e._reader.requestSlice(0, t);
      if ((A(r) && (r = await r), !r)) return !1;
      const n = N(r, t);
      return (n[0] === 71 && n[Ve] === 71) || (n[0] === 71 && n[Ve + 16] === 71)
        ? !0
        : n[4] === 71 && n[4 + Ve + 4] === 71;
    }
    _createDemuxer(e) {
      return new sd(e);
    }
    get name() {
      return "MPEG Transport Stream";
    }
    get mimeType() {
      return "video/MP2T";
    }
  }
  class tc extends st {
    async _canReadInput(e) {
      let t = e._reader.requestSlice(0, 7);
      if ((A(t) && (t = await t), !t || !(le(t, 7) === "#EXTM3U"))) return !1;
      if (!(e._rootSource instanceof ar))
        throw new TypeError(
          "HLS inputs require `InputOptions.source` to be a PathedSource or a ref to one.",
        );
      return ((e._rootSource._usedForHls = !0), !0);
    }
    _createDemuxer(e) {
      return new Td(e);
    }
    get name() {
      return "HTTP Live Streaming (HLS)";
    }
    get mimeType() {
      return Co;
    }
  }
  const ls = [
      new tc(),
      new Sd(),
      new xd(),
      new ec(),
      new Pd(),
      new Id(),
      new Ed(),
      new _d(),
      new Cd(),
      new vd(),
      new Ad(),
    ],
    Bd = (i, e) => {
      if (!i || typeof i != "object")
        throw new TypeError(`${e}, when provided, must be an object.`);
      if (i.isobmff !== void 0) {
        if (!i.isobmff || typeof i.isobmff != "object")
          throw new TypeError(
            `${e}.isobmff, when provided, must be an object.`,
          );
        if (
          i.isobmff.resolveKeyId !== void 0 &&
          typeof i.isobmff.resolveKeyId != "function"
        )
          throw new TypeError(
            `${e}.isobmff.resolveKeyId, when provided, must be a function.`,
          );
      }
      if (i.hls !== void 0) {
        if (!i.hls || typeof i.hls != "object")
          throw new TypeError(`${e}.hls, when provided, must be an object.`);
        if (
          i.hls.offsetTimestampsByDateTime !== void 0 &&
          typeof i.hls.offsetTimestampsByDateTime != "boolean"
        )
          throw new TypeError(
            `${e}.hls.offsetTimestampsByDateTime, when provided, must be a boolean.`,
          );
      }
    };
  var Fd = function (i, e, t) {
      if (e != null) {
        if (typeof e != "object" && typeof e != "function")
          throw new TypeError("Object expected.");
        var r, n;
        if (t) {
          if (!Symbol.asyncDispose)
            throw new TypeError("Symbol.asyncDispose is not defined.");
          r = e[Symbol.asyncDispose];
        }
        if (r === void 0) {
          if (!Symbol.dispose)
            throw new TypeError("Symbol.dispose is not defined.");
          ((r = e[Symbol.dispose]), t && (n = r));
        }
        if (typeof r != "function")
          throw new TypeError("Object not disposable.");
        (n &&
          (r = function () {
            try {
              n.call(this);
            } catch (s) {
              return Promise.reject(s);
            }
          }),
          i.stack.push({ value: e, dispose: r, async: t }));
      } else t && i.stack.push({ async: !0 });
      return e;
    },
    Rd = (function (i) {
      return function (e) {
        function t(a) {
          ((e.error = e.hasError
            ? new i(a, e.error, "An error was suppressed during disposal.")
            : a),
            (e.hasError = !0));
        }
        var r,
          n = 0;
        function s() {
          for (; (r = e.stack.pop());)
            try {
              if (!r.async && n === 1)
                return ((n = 0), e.stack.push(r), Promise.resolve().then(s));
              if (r.dispose) {
                var a = r.dispose.call(r.value);
                if (r.async)
                  return (
                    (n |= 2),
                    Promise.resolve(a).then(s, function (o) {
                      return (t(o), s());
                    })
                  );
              } else n |= 1;
            } catch (o) {
              t(o);
            }
          if (n === 1)
            return e.hasError ? Promise.reject(e.error) : Promise.resolve();
          if (e.hasError) throw e.error;
        }
        return s();
      };
    })(
      typeof SuppressedError == "function"
        ? SuppressedError
        : function (i, e, t) {
            var r = new Error(t);
            return (
              (r.name = "SuppressedError"),
              (r.error = i),
              (r.suppressed = e),
              r
            );
          },
    );
  Sn();
  let ic = -1 / 0,
    rc = -1 / 0,
    or = null;
  typeof FinalizationRegistry < "u" &&
    (or = new FinalizationRegistry((i) => {
      const e = performance.now();
      i.type === "video"
        ? (e - ic >= 1e3 &&
            (L._error(
              "A VideoSample was garbage collected without first being closed. For proper resource management, make sure to call close() on all your VideoSamples as soon as you're done using them.",
            ),
            (ic = e)),
          typeof VideoFrame < "u" &&
            i.data instanceof VideoFrame &&
            i.data.close())
        : (e - rc >= 1e3 &&
            (L._error(
              "An AudioSample was garbage collected without first being closed. For proper resource management, make sure to call close() on all your AudioSamples as soon as you're done using them.",
            ),
            (rc = e)),
          typeof AudioData < "u" &&
            i.data instanceof AudioData &&
            i.data.close());
    }));
  class Zt {
    constructor() {
      ((this._referenceCount = 0), (this._lastAllocationBuffer = null));
    }
  }
  const us = [
      "I420",
      "I420P10",
      "I420P12",
      "I420A",
      "I420AP10",
      "I420AP12",
      "I422",
      "I422P10",
      "I422P12",
      "I422A",
      "I422AP10",
      "I422AP12",
      "I444",
      "I444P10",
      "I444P12",
      "I444A",
      "I444AP10",
      "I444AP12",
      "NV12",
      "RGBA",
      "RGBX",
      "BGRA",
      "BGRX",
    ],
    Md = new Set(us);
  class _e {
    get codedWidth() {
      return this.visibleRect.width;
    }
    get codedHeight() {
      return this.visibleRect.height;
    }
    get displayWidth() {
      return this.rotation % 180 === 0
        ? this.squarePixelWidth
        : this.squarePixelHeight;
    }
    get displayHeight() {
      return this.rotation % 180 === 0
        ? this.squarePixelHeight
        : this.squarePixelWidth;
    }
    get microsecondTimestamp() {
      return Math.trunc(_t * this.timestamp);
    }
    get microsecondDuration() {
      return Math.trunc(_t * this.duration);
    }
    get hasAlpha() {
      return this.format && this.format.includes("A");
    }
    constructor(e, t) {
      if (
        ((this._closed = !1),
        e instanceof ArrayBuffer ||
          (typeof SharedArrayBuffer < "u" && e instanceof SharedArrayBuffer) ||
          ArrayBuffer.isView(e))
      ) {
        if (!t || typeof t != "object")
          throw new TypeError("init must be an object.");
        if (t.format === void 0 || !Md.has(t.format))
          throw new TypeError("init.format must be one of: " + us.join(", "));
        if (!Number.isInteger(t.codedWidth) || t.codedWidth <= 0)
          throw new TypeError("init.codedWidth must be a positive integer.");
        if (!Number.isInteger(t.codedHeight) || t.codedHeight <= 0)
          throw new TypeError("init.codedHeight must be a positive integer.");
        if (t.rotation !== void 0 && ![0, 90, 180, 270].includes(t.rotation))
          throw new TypeError(
            "init.rotation, when provided, must be 0, 90, 180, or 270.",
          );
        if (t.flip !== void 0 && typeof t.flip != "boolean")
          throw new TypeError("init.flip, when provided, must be a boolean.");
        if (!Number.isFinite(t.timestamp))
          throw new TypeError("init.timestamp must be a number.");
        if (
          t.duration !== void 0 &&
          (!Number.isFinite(t.duration) || t.duration < 0)
        )
          throw new TypeError(
            "init.duration, when provided, must be a non-negative number.",
          );
        if (t.layout !== void 0) {
          if (!Array.isArray(t.layout))
            throw new TypeError(
              "init.layout, when provided, must be an array.",
            );
          for (const s of t.layout) {
            if (!s || typeof s != "object" || Array.isArray(s))
              throw new TypeError(
                "Each entry in init.layout must be an object.",
              );
            if (!Number.isInteger(s.offset) || s.offset < 0)
              throw new TypeError(
                "plane.offset must be a non-negative integer.",
              );
            if (!Number.isInteger(s.stride) || s.stride < 0)
              throw new TypeError(
                "plane.stride must be a non-negative integer.",
              );
          }
        }
        if (
          (t.visibleRect !== void 0 && Pn(t.visibleRect, "init.visibleRect"),
          t.displayWidth !== void 0 &&
            (!Number.isInteger(t.displayWidth) || t.displayWidth <= 0))
        )
          throw new TypeError(
            "init.displayWidth, when provided, must be a positive integer.",
          );
        if (
          t.displayHeight !== void 0 &&
          (!Number.isInteger(t.displayHeight) || t.displayHeight <= 0)
        )
          throw new TypeError(
            "init.displayHeight, when provided, must be a positive integer.",
          );
        if ((t.displayWidth !== void 0) != (t.displayHeight !== void 0))
          throw new TypeError(
            "init.displayWidth and init.displayHeight must be either both provided or both omitted.",
          );
        ((this.format = t.format),
          (this.rotation = t.rotation ?? 0),
          (this.flip = t.flip ?? !1),
          (this.timestamp = t.timestamp),
          (this.duration = t.duration ?? 0));
        const r = t.layout ?? zd(t.format, t.codedWidth, t.codedHeight);
        let n = t.colorSpace ?? null;
        (n === null &&
          (this.format === "RGBA" ||
          this.format === "RGBX" ||
          this.format === "BGRA" ||
          this.format === "BGRX"
            ? (n = {
                primaries: "bt709",
                transfer: "iec61966-2-1",
                matrix: "rgb",
                fullRange: !0,
              })
            : (n = {
                primaries: "bt709",
                transfer: "bt709",
                matrix: "bt709",
                fullRange: !1,
              })),
          (this.visibleRect = {
            left: t.visibleRect?.left ?? 0,
            top: t.visibleRect?.top ?? 0,
            width: t.visibleRect?.width ?? t.codedWidth,
            height: t.visibleRect?.height ?? t.codedHeight,
          }),
          t.displayWidth !== void 0
            ? ((this.squarePixelWidth =
                this.rotation % 180 === 0 ? t.displayWidth : t.displayHeight),
              (this.squarePixelHeight =
                this.rotation % 180 === 0 ? t.displayHeight : t.displayWidth))
            : ((this.squarePixelWidth = this.visibleRect.width),
              (this.squarePixelHeight = this.visibleRect.height)),
          (this._data = t._doNotCopy ? me(e) : me(e).slice()),
          (this._layout = r),
          (this.colorSpace = new ds(n)));
      } else if (typeof VideoFrame < "u" && e instanceof VideoFrame) {
        if (t?.rotation !== void 0 && ![0, 90, 180, 270].includes(t.rotation))
          throw new TypeError(
            "init.rotation, when provided, must be 0, 90, 180, or 270.",
          );
        if (t?.flip !== void 0 && typeof t.flip != "boolean")
          throw new TypeError("init.flip, when provided, must be a boolean.");
        if (t?.timestamp !== void 0 && !Number.isFinite(t?.timestamp))
          throw new TypeError(
            "init.timestamp, when provided, must be a number.",
          );
        if (
          t?.duration !== void 0 &&
          (!Number.isFinite(t.duration) || t.duration < 0)
        )
          throw new TypeError(
            "init.duration, when provided, must be a non-negative number.",
          );
        (t?.visibleRect !== void 0 && Pn(t.visibleRect, "init.visibleRect"),
          (this._data = e),
          (this._layout = null),
          (this.format = e.format),
          (this.visibleRect = {
            left: e.visibleRect?.x ?? 0,
            top: e.visibleRect?.y ?? 0,
            width: e.visibleRect?.width ?? e.codedWidth,
            height: e.visibleRect?.height ?? e.codedHeight,
          }),
          (this.rotation = t?.rotation ?? 0),
          (this.flip = t?.flip ?? !1),
          (this.squarePixelWidth = e.displayWidth),
          (this.squarePixelHeight = e.displayHeight),
          (this.timestamp = t?.timestamp ?? e.timestamp / 1e6),
          (this.duration = t?.duration ?? (e.duration ?? 0) / 1e6),
          (this.colorSpace = new ds(e.colorSpace)));
      } else if (
        (typeof HTMLImageElement < "u" && e instanceof HTMLImageElement) ||
        (typeof SVGImageElement < "u" && e instanceof SVGImageElement) ||
        (typeof ImageBitmap < "u" && e instanceof ImageBitmap) ||
        (typeof HTMLVideoElement < "u" && e instanceof HTMLVideoElement) ||
        (typeof HTMLCanvasElement < "u" && e instanceof HTMLCanvasElement) ||
        (typeof OffscreenCanvas < "u" && e instanceof OffscreenCanvas)
      ) {
        if (!t || typeof t != "object")
          throw new TypeError("init must be an object.");
        if (t.rotation !== void 0 && ![0, 90, 180, 270].includes(t.rotation))
          throw new TypeError(
            "init.rotation, when provided, must be 0, 90, 180, or 270.",
          );
        if (t.flip !== void 0 && typeof t.flip != "boolean")
          throw new TypeError("init.flip, when provided, must be a boolean.");
        if (!Number.isFinite(t.timestamp))
          throw new TypeError("init.timestamp must be a number.");
        if (
          t.duration !== void 0 &&
          (!Number.isFinite(t.duration) || t.duration < 0)
        )
          throw new TypeError(
            "init.duration, when provided, must be a non-negative number.",
          );
        if (
          (t.visibleRect !== void 0 && Pn(t.visibleRect, "init.visibleRect"),
          typeof VideoFrame < "u")
        )
          return new _e(
            new VideoFrame(e, {
              timestamp: Math.trunc(t.timestamp * _t),
              duration: Math.trunc((t.duration ?? 0) * _t) || void 0,
              visibleRect: t.visibleRect && {
                x: t.visibleRect.left,
                y: t.visibleRect.top,
                width: t.visibleRect.width,
                height: t.visibleRect.height,
              },
            }),
            t,
          );
        let r = 0,
          n = 0;
        if (
          ("naturalWidth" in e
            ? ((r = e.naturalWidth), (n = e.naturalHeight))
            : "videoWidth" in e
              ? ((r = e.videoWidth), (n = e.videoHeight))
              : "width" in e && ((r = Number(e.width)), (n = Number(e.height))),
          !r || !n)
        )
          throw new TypeError("Could not determine dimensions.");
        const s = t.visibleRect ?? { left: 0, top: 0, width: r, height: n },
          a = new OffscreenCanvas(s.width, s.height),
          o = a.getContext("2d", { alpha: ta(), willReadFrequently: !0 });
        if (!o)
          throw new Error(
            "OffscreenCanvas must have support for the '2d' context in order to create a VideoSample from this data.",
          );
        (o.drawImage(e, -s.left, -s.top),
          (this._data = a),
          (this._layout = null),
          (this.format = "RGBX"),
          (this.visibleRect = {
            left: 0,
            top: 0,
            width: s.width,
            height: s.height,
          }),
          (this.squarePixelWidth = s.width),
          (this.squarePixelHeight = s.height),
          (this.rotation = t.rotation ?? 0),
          (this.flip = t.flip ?? !1),
          (this.timestamp = t.timestamp),
          (this.duration = t.duration ?? 0),
          (this.colorSpace = new ds({
            matrix: "rgb",
            primaries: "bt709",
            transfer: "iec61966-2-1",
            fullRange: !0,
          })));
      } else if (e instanceof Zt) {
        if (!t || typeof t != "object")
          throw new TypeError("init must be an object.");
        if (t.rotation !== void 0 && ![0, 90, 180, 270].includes(t.rotation))
          throw new TypeError(
            "init.rotation, when provided, must be 0, 90, 180, or 270.",
          );
        if (t.flip !== void 0 && typeof t.flip != "boolean")
          throw new TypeError("init.flip, when provided, must be a boolean.");
        if (!Number.isFinite(t.timestamp))
          throw new TypeError("init.timestamp must be a number.");
        if (
          t.duration !== void 0 &&
          (!Number.isFinite(t.duration) || t.duration < 0)
        )
          throw new TypeError(
            "init.duration, when provided, must be a non-negative number.",
          );
        if (
          ((this._data = e),
          e._referenceCount++,
          (this.format = e.getFormat()),
          this.format !== null && !us.includes(this.format))
        )
          throw new TypeError(
            "getFormat() must return a VideoSamplePixelFormat or null.",
          );
        if (
          ((this.visibleRect = {
            left: 0,
            top: 0,
            width: e.getCodedWidth(),
            height: e.getCodedHeight(),
          }),
          !Number.isInteger(this.visibleRect.width) ||
            this.visibleRect.width <= 0)
        )
          throw new TypeError(
            "getCodedWidth() must return a positive integer.",
          );
        if (
          !Number.isInteger(this.visibleRect.height) ||
          this.visibleRect.height <= 0
        )
          throw new TypeError(
            "getCodedHeight() must return a positive integer.",
          );
        if (
          ((this.squarePixelWidth = e.getSquarePixelWidth()),
          !Number.isInteger(this.squarePixelWidth) ||
            this.squarePixelWidth <= 0)
        )
          throw new TypeError(
            "getSquarePixelWidth() must return a positive integer.",
          );
        if (
          ((this.squarePixelHeight = e.getSquarePixelHeight()),
          !Number.isInteger(this.squarePixelHeight) ||
            this.squarePixelHeight <= 0)
        )
          throw new TypeError(
            "getSquarePixelHeight() must return a positive integer.",
          );
        ((this.rotation = t.rotation ?? 0),
          (this.flip = t.flip ?? !1),
          (this.timestamp = t.timestamp),
          (this.duration = t.duration ?? 0),
          (this.colorSpace = e.getColorSpace()));
      } else
        throw new TypeError(
          "Invalid data type: Must be a BufferSource, CanvasImageSource, or VideoSampleResource.",
        );
      ((this.encodeOptions = t?.encodeOptions ?? {}),
        (this.pixelAspectRatio = di({
          num: this.squarePixelWidth * this.codedHeight,
          den: this.squarePixelHeight * this.codedWidth,
        })),
        or?.register(this, { type: "video", data: this._data }, this));
    }
    clone() {
      if (this._closed) throw new Error("VideoSample is closed.");
      return (
        g(this._data !== null),
        this._data instanceof Zt
          ? new _e(this._data, {
              timestamp: this.timestamp,
              duration: this.duration,
              rotation: this.rotation,
              flip: this.flip,
              encodeOptions: this.encodeOptions,
            })
          : lr(this._data)
            ? new _e(this._data.clone(), {
                timestamp: this.timestamp,
                duration: this.duration,
                rotation: this.rotation,
                flip: this.flip,
                encodeOptions: this.encodeOptions,
              })
            : this._data instanceof Uint8Array
              ? (g(this._layout),
                new _e(this._data, {
                  format: this.format,
                  layout: this._layout,
                  codedWidth: this.codedWidth,
                  codedHeight: this.codedHeight,
                  timestamp: this.timestamp,
                  duration: this.duration,
                  colorSpace: this.colorSpace,
                  rotation: this.rotation,
                  flip: this.flip,
                  visibleRect: this.visibleRect,
                  displayWidth: this.displayWidth,
                  displayHeight: this.displayHeight,
                  encodeOptions: this.encodeOptions,
                  _doNotCopy: !0,
                }))
              : new _e(this._data, {
                  format: this.format,
                  codedWidth: this.codedWidth,
                  codedHeight: this.codedHeight,
                  timestamp: this.timestamp,
                  duration: this.duration,
                  colorSpace: this.colorSpace,
                  rotation: this.rotation,
                  flip: this.flip,
                  visibleRect: this.visibleRect,
                  displayWidth: this.displayWidth,
                  displayHeight: this.displayHeight,
                  encodeOptions: this.encodeOptions,
                })
      );
    }
    close() {
      this._closed ||
        (or?.unregister(this),
        this._data instanceof Zt
          ? (this._data._referenceCount--,
            this._data._referenceCount === 0 && this._data.close())
          : lr(this._data)
            ? this._data.close()
            : (this._data = null),
        (this._closed = !0));
    }
    allocationSize(e = {}) {
      if ((ac(e), this._closed)) throw new Error("VideoSample is closed.");
      if ((e.format ?? this.format) == null)
        throw new Error("Cannot get allocation size when format is null.");
      return lr(this._data)
        ? this._data.allocationSize(e)
        : oc(this, e).allocationSize;
    }
    async copyTo(e, t = {}) {
      if (!kr(e))
        throw new TypeError(
          "destination must be an ArrayBuffer or an ArrayBuffer view.",
        );
      if ((ac(t), this._closed)) throw new Error("VideoSample is closed.");
      if ((t.format ?? this.format) == null)
        throw new Error("Cannot copy video sample data when format is null.");
      if ((g(this._data !== null), lr(this._data)))
        return this._data.copyTo(e, t);
      if (
        t.format &&
        !["RGBA", "RGBX", "BGRA", "BGRX"].includes(this.format) &&
        ["RGBA", "RGBX", "BGRA", "BGRX"].includes(t.format)
      )
        if (this._data instanceof Zt) {
          const l = { stack: [], error: void 0, hasError: !1 };
          try {
            const u = Fd(
              l,
              await this._data.toRgbSample(
                {
                  timestamp: this.timestamp,
                  duration: this.duration,
                  rotation: this.rotation,
                  flip: this.flip,
                },
                t.colorSpace ?? "srgb",
              ),
              !1,
            );
            if (!(u instanceof _e))
              throw new TypeError("toRgbSample() must return a VideoSample.");
            if (!["RGBA", "RGBX", "BGRA", "BGRX"].includes(u.format))
              throw new Error(
                `Sample returned by toRgbSample was expected to have an RGB format, got '${u.format}' instead.`,
              );
            return await u.copyTo(e, t);
          } catch (u) {
            ((l.error = u), (l.hasError = !0));
          } finally {
            Rd(l);
          }
        } else {
          if (typeof VideoFrame > "u")
            throw new Error(
              "For this sample, converting from a non-RGB to an RGB format requires VideoFrame to be defined.",
            );
          const l = this.toVideoFrame(),
            u = await l.copyTo(e, t);
          return (l.close(), u);
        }
      const r = oc(this, t);
      g(this.format);
      const n = me(e);
      if (n.byteLength < r.allocationSize)
        throw new TypeError(
          `Destination buffer too small. Required: ${r.allocationSize}, Available: ${n.byteLength}`,
        );
      const s = qr(this.format);
      let a;
      if (this._data instanceof Zt) {
        let l = this._data.getDataPlanes();
        if (
          (A(l) && (l = await l),
          !Array.isArray(l) ||
            l.some(
              (u) =>
                !(u.data instanceof Uint8Array) ||
                !Number.isInteger(u.stride) ||
                u.stride < 0,
            ))
        )
          throw new TypeError(
            'getDataPlanes() must return an array of objects with a Uint8Array "data" property and a non-negative integer "stride" property.',
          );
        a = l;
      } else if (this._data instanceof Uint8Array)
        (g(this._layout),
          g(this._layout.length === s.length),
          (a = this._layout.map((l, u) => {
            const d = Math.ceil(this.codedHeight / s[u].heightDivisor);
            return {
              data: this._data.subarray(l.offset, l.offset + l.stride * d),
              stride: l.stride,
            };
          })));
      else {
        const u = this._data.getContext("2d");
        g(u);
        const d = u.getImageData(0, 0, this.codedWidth, this.codedHeight);
        a = [{ data: me(d.data), stride: 4 * this.codedWidth }];
      }
      const o = [],
        c = s.length;
      for (let l = 0; l < c; l++) {
        const u = r.computedLayouts[l],
          d = a[l].stride,
          f = a[l].data;
        let h = u.sourceTop * d;
        h += u.sourceLeftBytes;
        let p = u.destinationOffset;
        const m = u.sourceWidthBytes,
          y = { offset: p, stride: u.destinationStride };
        for (let w = 0; w < u.sourceHeight; w++) {
          if (h + m > f.byteLength) throw new Error("Source buffer OOB read.");
          if (p + m > n.byteLength)
            throw new Error("Destination buffer OOB write.");
          const b = f.subarray(h, h + m);
          (n.set(b, p), (h += d), (p += u.destinationStride));
        }
        o.push(y);
      }
      if (t.format !== void 0) {
        const l = this.format.startsWith("RGB") !== t.format.startsWith("RGB"),
          u = this.format.includes("X") && t.format.includes("A");
        if (l || u)
          for (let d = 0; d < r.allocationSize; d += 4) {
            if (l) {
              const f = n[d],
                h = n[d + 2];
              ((n[d] = h), (n[d + 2] = f));
            }
            u && (n[d + 3] = 255);
          }
      }
      return o;
    }
    toVideoFrame() {
      if (this._closed) throw new Error("VideoSample is closed.");
      if ((g(this._data !== null), this._data instanceof Zt)) {
        if (this.format === null)
          throw new Error(
            "Cannot convert a VideoSampleResource-backed VideoSample to VideoFrame if format is null.",
          );
        const e = this._data.getDataPlanes();
        if (A(e))
          throw new Error(
            "Cannot convert a VideoSampleResource-backed VideoSample to VideoFrame if getDataPlanes() returns a promise.",
          );
        const t = e.reduce((a, o) => a + o.data.byteLength, 0),
          r = new Uint8Array(t);
        let n = 0;
        const s = [];
        for (const a of e)
          (r.set(a.data, n), s.push(n), (n += a.data.byteLength));
        return new VideoFrame(r, {
          format: this.format,
          layout: e.map((a, o) => ({ offset: s[o], stride: a.stride })),
          codedWidth: this.codedWidth,
          codedHeight: this.codedHeight,
          timestamp: this.microsecondTimestamp,
          duration: this.microsecondDuration,
          colorSpace: this.colorSpace,
          visibleRect: this.visibleRect,
          displayWidth: this.squarePixelWidth,
          displayHeight: this.squarePixelHeight,
        });
      } else
        return lr(this._data)
          ? new VideoFrame(this._data, {
              timestamp: this.microsecondTimestamp,
              duration: this.microsecondDuration || void 0,
            })
          : this._data instanceof Uint8Array
            ? (g(this._layout),
              new VideoFrame(this._data, {
                format: this.format,
                codedWidth: this.codedWidth,
                codedHeight: this.codedHeight,
                layout: this._layout,
                timestamp: this.microsecondTimestamp,
                duration: this.microsecondDuration || void 0,
                colorSpace: this.colorSpace,
                visibleRect: this.visibleRect,
                displayWidth: this.squarePixelWidth,
                displayHeight: this.squarePixelHeight,
              }))
            : new VideoFrame(this._data, {
                timestamp: this.microsecondTimestamp,
                duration: this.microsecondDuration || void 0,
              });
    }
    draw(e, t, r, n, s, a, o, c, l) {
      let u = 0,
        d = 0,
        f = this.displayWidth,
        h = this.displayHeight,
        p = 0,
        m = 0,
        y = this.displayWidth,
        w = this.displayHeight;
      if (
        (a !== void 0
          ? ((u = t),
            (d = r),
            (f = n),
            (h = s),
            (p = a),
            (m = o),
            c !== void 0 ? ((y = c), (w = l)) : ((y = f), (w = h)))
          : ((p = t), (m = r), n !== void 0 && ((y = n), (w = s))),
        !(
          (typeof CanvasRenderingContext2D < "u" &&
            e instanceof CanvasRenderingContext2D) ||
          (typeof OffscreenCanvasRenderingContext2D < "u" &&
            e instanceof OffscreenCanvasRenderingContext2D)
        ))
      )
        throw new TypeError(
          "context must be a CanvasRenderingContext2D or OffscreenCanvasRenderingContext2D.",
        );
      if (!Number.isFinite(u)) throw new TypeError("sx must be a number.");
      if (!Number.isFinite(d)) throw new TypeError("sy must be a number.");
      if (!Number.isFinite(f) || f < 0)
        throw new TypeError("sWidth must be a non-negative number.");
      if (!Number.isFinite(h) || h < 0)
        throw new TypeError("sHeight must be a non-negative number.");
      if (!Number.isFinite(p)) throw new TypeError("dx must be a number.");
      if (!Number.isFinite(m)) throw new TypeError("dy must be a number.");
      if (!Number.isFinite(y) || y < 0)
        throw new TypeError("dWidth must be a non-negative number.");
      if (!Number.isFinite(w) || w < 0)
        throw new TypeError("dHeight must be a non-negative number.");
      if (this._closed) throw new Error("VideoSample is closed.");
      ({
        sx: u,
        sy: d,
        sWidth: f,
        sHeight: h,
      } = this._unmapSourceRegion(u, d, f, h, this.rotation, this.flip));
      const b = this.toCanvasImageSource();
      e.save();
      const T = p + y / 2,
        S = m + w / 2;
      (e.translate(T, S),
        this.flip && e.scale(-1, 1),
        e.rotate((this.rotation * Math.PI) / 180));
      const k = this.rotation % 180 === 0 ? 1 : y / w;
      (e.scale(1 / k, k),
        e.drawImage(b, u, d, f, h, -y / 2, -w / 2, y, w),
        e.restore());
    }
    drawWithFit(e, t) {
      if (!(
        (typeof CanvasRenderingContext2D < "u" &&
          e instanceof CanvasRenderingContext2D) ||
        (typeof OffscreenCanvasRenderingContext2D < "u" &&
          e instanceof OffscreenCanvasRenderingContext2D)
      ))
        throw new TypeError(
          "context must be a CanvasRenderingContext2D or OffscreenCanvasRenderingContext2D.",
        );
      if (!t || typeof t != "object")
        throw new TypeError("options must be an object.");
      if (!["fill", "contain", "cover"].includes(t.fit))
        throw new TypeError(
          "options.fit must be 'fill', 'contain', or 'cover'.",
        );
      if (t.rotation !== void 0 && ![0, 90, 180, 270].includes(t.rotation))
        throw new TypeError(
          "options.rotation, when provided, must be 0, 90, 180, or 270.",
        );
      if (t.flip !== void 0 && typeof t.flip != "boolean")
        throw new TypeError("options.flip, when provided, must be a boolean.");
      t.crop !== void 0 && Hr(t.crop, "options.");
      const r = e.canvas.width,
        n = e.canvas.height,
        s = t.rotation ?? this.rotation,
        a = t.flip ?? this.flip,
        [o, c] =
          s % 180 === 0
            ? [this.squarePixelWidth, this.squarePixelHeight]
            : [this.squarePixelHeight, this.squarePixelWidth];
      let l = t.crop;
      l && (l = fs(l, o, c));
      let u, d, f, h;
      const {
        sx: p,
        sy: m,
        sWidth: y,
        sHeight: w,
      } = this._unmapSourceRegion(
        t.crop?.left ?? 0,
        t.crop?.top ?? 0,
        t.crop?.width ?? o,
        t.crop?.height ?? c,
        s,
        a,
      );
      if (t.fit === "fill") ((u = 0), (d = 0), (f = r), (h = n));
      else {
        const [T, S] = t.crop ? [t.crop.width, t.crop.height] : [o, c],
          k =
            t.fit === "contain"
              ? Math.min(r / T, n / S)
              : Math.max(r / T, n / S);
        ((f = T * k), (h = S * k), (u = (r - f) / 2), (d = (n - h) / 2));
      }
      e.save();
      const b = s % 180 === 0 ? 1 : f / h;
      (e.translate(r / 2, n / 2),
        a && e.scale(-1, 1),
        e.rotate((s * Math.PI) / 180),
        e.scale(1 / b, b),
        e.translate(-r / 2, -n / 2),
        e.drawImage(this.toCanvasImageSource(), p, m, y, w, u, d, f, h),
        e.restore());
    }
    _unmapSourceRegion(e, t, r, n, s, a) {
      return (
        a &&
          (e =
            (s % 180 === 0 ? this.squarePixelWidth : this.squarePixelHeight) -
            e -
            r),
        s === 90
          ? ([e, t, r, n] = [t, this.squarePixelHeight - e - r, n, r])
          : s === 180
            ? ([e, t] = [
                this.squarePixelWidth - e - r,
                this.squarePixelHeight - t - n,
              ])
            : s === 270 &&
              ([e, t, r, n] = [this.squarePixelWidth - t - n, e, n, r]),
        { sx: e, sy: t, sWidth: r, sHeight: n }
      );
    }
    _drawWithFitAndMipmapping(e, t, r) {
      const n = e.width,
        s = e.height,
        [a, o] =
          r.rotation % 180 === 0
            ? [this.squarePixelWidth, this.squarePixelHeight]
            : [this.squarePixelHeight, this.squarePixelWidth],
        c = r.crop ? r.crop.width : a,
        l = r.crop ? r.crop.height : o;
      let u = 0;
      2 * n < c &&
        2 * s < l &&
        (u = Math.floor(Math.log2(Math.min(c / n, l / s))));
      const d = n * 2 ** u,
        f = s * 2 ** u,
        {
          canvas: h,
          context: p,
          isNew: m,
        } = u > 0
          ? sc(d, f)
          : { canvas: e, context: t, isNew: r.targetIsFresh };
      ((p.imageSmoothingQuality = "high"),
        r.fillBlack
          ? ((p.fillStyle = "black"), p.fillRect(0, 0, d, f))
          : m || p.clearRect(0, 0, d, f),
        this.drawWithFit(p, {
          fit: r.fit,
          rotation: r.rotation,
          flip: r.flip,
          crop: r.crop,
        }),
        (p.globalCompositeOperation = "copy"));
      for (let y = u; y > 1; y--) {
        const w = n * 2 ** y,
          b = s * 2 ** y;
        p.drawImage(h, 0, 0, w, b, 0, 0, w / 2, b / 2);
      }
      ((p.globalCompositeOperation = "source-over"),
        u > 0 &&
          ((t.imageSmoothingQuality = "high"),
          (t.globalCompositeOperation = "copy"),
          t.drawImage(h, 0, 0, 2 * n, 2 * s, 0, 0, n, s),
          (t.globalCompositeOperation = "source-over")));
    }
    toCanvasImageSource() {
      if (this._closed) throw new Error("VideoSample is closed.");
      if (
        (g(this._data !== null),
        this._data instanceof Zt || this._data instanceof Uint8Array)
      ) {
        const e = this.toVideoFrame();
        return (queueMicrotask(() => e.close()), e);
      } else return this._data;
    }
    async transform(e) {
      if (!e || typeof e != "object")
        throw new TypeError("options must be an object.");
      if (e.width !== void 0 && (!Number.isInteger(e.width) || e.width <= 0))
        throw new TypeError(
          "options.width, when provided, must be a positive integer.",
        );
      if (e.height !== void 0 && (!Number.isInteger(e.height) || e.height <= 0))
        throw new TypeError(
          "options.height, when provided, must be a positive integer.",
        );
      if (
        e.roundDimensionsTo !== void 0 &&
        (!Number.isInteger(e.roundDimensionsTo) || e.roundDimensionsTo <= 0)
      )
        throw new TypeError(
          "options.roundDimensionsTo, when provided, must be a positive integer.",
        );
      if (e.fit !== void 0 && !["fill", "contain", "cover"].includes(e.fit))
        throw new TypeError(
          'options.fit, when provided, must be one of "fill", "contain", or "cover".',
        );
      if (e.width !== void 0 && e.height !== void 0 && e.fit === void 0)
        throw new TypeError(
          "When both options.width and options.height are provided, options.fit must also be provided.",
        );
      if (e.rotate !== void 0 && ![0, 90, 180, 270].includes(e.rotate))
        throw new TypeError(
          "options.rotate, when provided, must be 0, 90, 180 or 270.",
        );
      if (e.flip !== void 0 && typeof e.flip != "boolean")
        throw new TypeError("options.flip, when provided, must be a boolean.");
      if (
        (e.crop !== void 0 && Hr(e.crop, "options."),
        e.alpha !== void 0 && !["keep", "discard"].includes(e.alpha))
      )
        throw new TypeError(
          "options.alpha, when provided, must be 'keep' or 'discard'.",
        );
      const { rotation: t, flip: r } = Gs(
          this.rotation,
          this.flip,
          e.rotate ?? 0,
          e.flip ?? !1,
        ),
        [n, s] =
          t % 180 === 0
            ? [this.squarePixelWidth, this.squarePixelHeight]
            : [this.squarePixelHeight, this.squarePixelWidth];
      let a = e.crop;
      a && (a = fs(a, n, s));
      const o = a ? a.width : n,
        c = a ? a.height : s,
        l = o / c;
      let u, d;
      (e.width !== void 0 && e.height === void 0
        ? ((u = e.width), (d = u / l))
        : e.width === void 0 && e.height !== void 0
          ? ((d = e.height), (u = d * l))
          : e.width !== void 0 && e.height !== void 0
            ? ((u = e.width), (d = e.height))
            : ((u = o), (d = c)),
        (u = pn(u, e.roundDimensionsTo ?? 1)),
        (d = pn(d, e.roundDimensionsTo ?? 1)));
      const f = {
        width: u,
        height: d,
        fit: e.fit ?? "fill",
        rotation: t,
        flip: r,
        crop: a ?? { left: 0, top: 0, width: n, height: s },
        alpha: e.alpha ?? "keep",
      };
      for (const y of Dd) {
        let w = y(this, f);
        if ((A(w) && (w = await w), w !== null)) return w;
      }
      const { canvas: h, context: p, isNew: m } = sc(f.width, f.height);
      return (
        this._drawWithFitAndMipmapping(h, p, {
          fit: f.fit,
          rotation: f.rotation,
          flip: f.flip,
          crop: f.crop,
          targetIsFresh: m,
          fillBlack: f.alpha === "discard",
        }),
        new _e(h, {
          timestamp: this.timestamp,
          duration: this.duration,
          rotation: 0,
          flip: !1,
        })
      );
    }
    setRotation(e) {
      if (![0, 90, 180, 270].includes(e))
        throw new TypeError("newRotation must be 0, 90, 180, or 270.");
      this.rotation = e;
    }
    setFlip(e) {
      if (typeof e != "boolean")
        throw new TypeError("newFlip must be a boolean.");
      this.flip = e;
    }
    setTimestamp(e) {
      if (!Number.isFinite(e))
        throw new TypeError("newTimestamp must be a number.");
      this.timestamp = e;
    }
    setDuration(e) {
      if (!Number.isFinite(e) || e < 0)
        throw new TypeError("newDuration must be a non-negative number.");
      this.duration = e;
    }
    setEncodeOptions(e) {
      if (!e || typeof e != "object")
        throw new TypeError("newEncodeOptions must be an object.");
      this.encodeOptions = e;
    }
    [Symbol.dispose]() {
      this.close();
    }
  }
  const Dd = [],
    Od = 3,
    cr = [];
  let nc = 0;
  const sc = (i, e) => {
    for (const n of cr)
      if (n.canvas.width === i && n.canvas.height === e)
        return (
          (n.age = nc++),
          { canvas: n.canvas, context: n.context, isNew: !1 }
        );
    let t;
    if (typeof OffscreenCanvas < "u") t = new OffscreenCanvas(i, e);
    else {
      if (typeof window > "u" || typeof document > "u")
        throw new Error(
          "Cannot transform VideoSamples in this environment. Either run in an environment with OffscreenCanvas or HTMLCanvasElement, or supply a custom VideoSample transformer using registerVideoSampleTransformer().",
        );
      ((t = document.createElement("canvas")), (t.width = i), (t.height = e));
    }
    const r = t.getContext("2d", { alpha: !0, willReadFrequently: !1 });
    if (!r)
      throw new Error(
        "The '2d' canvas context is required to transform VideoSamples. Register a custom transformer using registerVideoSampleTransformer to work around this limitation.",
      );
    return (
      cr.length >= Od &&
        cr.splice(
          xn(cr, (n) => n.age),
          1,
        ),
      cr.push({ canvas: t, context: r, age: nc++ }),
      { canvas: t, context: r, isNew: !0 }
    );
  };
  class ds {
    constructor(e) {
      if (e !== void 0) {
        if (!e || typeof e != "object")
          throw new TypeError(
            "init.colorSpace, when provided, must be an object.",
          );
        const t = Object.keys(Ht);
        if (e.primaries != null && !t.includes(e.primaries))
          throw new TypeError(
            `init.colorSpace.primaries, when provided, must be one of ${t.join(", ")}.`,
          );
        const r = Object.keys(qt);
        if (e.transfer != null && !r.includes(e.transfer))
          throw new TypeError(
            `init.colorSpace.transfer, when provided, must be one of ${r.join(", ")}.`,
          );
        const n = Object.keys(jt);
        if (e.matrix != null && !n.includes(e.matrix))
          throw new TypeError(
            `init.colorSpace.matrix, when provided, must be one of ${n.join(", ")}.`,
          );
        if (e.fullRange != null && typeof e.fullRange != "boolean")
          throw new TypeError(
            "init.colorSpace.fullRange, when provided, must be a boolean.",
          );
      }
      ((this.primaries = e?.primaries ?? null),
        (this.transfer = e?.transfer ?? null),
        (this.matrix = e?.matrix ?? null),
        (this.fullRange = e?.fullRange ?? null));
    }
    toJSON() {
      return {
        primaries: this.primaries,
        transfer: this.transfer,
        matrix: this.matrix,
        fullRange: this.fullRange,
      };
    }
  }
  const lr = (i) => typeof VideoFrame < "u" && i instanceof VideoFrame,
    fs = (i, e, t) => {
      const r = Math.min(i.left, e),
        n = Math.min(i.top, t),
        s = Math.min(i.width, e - r),
        a = Math.min(i.height, t - n);
      return (g(s >= 0), g(a >= 0), { left: r, top: n, width: s, height: a });
    },
    Hr = (i, e) => {
      if (!i || typeof i != "object")
        throw new TypeError(e + "crop, when provided, must be an object.");
      if (!Number.isInteger(i.left) || i.left < 0)
        throw new TypeError(e + "crop.left must be a non-negative integer.");
      if (!Number.isInteger(i.top) || i.top < 0)
        throw new TypeError(e + "crop.top must be a non-negative integer.");
      if (!Number.isInteger(i.width) || i.width < 0)
        throw new TypeError(e + "crop.width must be a non-negative integer.");
      if (!Number.isInteger(i.height) || i.height < 0)
        throw new TypeError(e + "crop.height must be a non-negative integer.");
    },
    ac = (i) => {
      if (!i || typeof i != "object")
        throw new TypeError("options must be an object.");
      if (
        i.colorSpace !== void 0 &&
        !["display-p3", "srgb"].includes(i.colorSpace)
      )
        throw new TypeError(
          "options.colorSpace, when provided, must be 'display-p3' or 'srgb'.",
        );
      if (i.format !== void 0 && typeof i.format != "string")
        throw new TypeError("options.format, when provided, must be a string.");
      if (i.layout !== void 0) {
        if (!Array.isArray(i.layout))
          throw new TypeError(
            "options.layout, when provided, must be an array.",
          );
        for (const e of i.layout) {
          if (!e || typeof e != "object")
            throw new TypeError(
              "Each entry in options.layout must be an object.",
            );
          if (!Number.isInteger(e.offset) || e.offset < 0)
            throw new TypeError("plane.offset must be a non-negative integer.");
          if (!Number.isInteger(e.stride) || e.stride < 0)
            throw new TypeError("plane.stride must be a non-negative integer.");
        }
      }
      if (i.rect !== void 0) {
        if (!i.rect || typeof i.rect != "object")
          throw new TypeError(
            "options.rect, when provided, must be an object.",
          );
        if (
          i.rect.x !== void 0 &&
          (!Number.isInteger(i.rect.x) || i.rect.x < 0)
        )
          throw new TypeError(
            "options.rect.x, when provided, must be a non-negative integer.",
          );
        if (
          i.rect.y !== void 0 &&
          (!Number.isInteger(i.rect.y) || i.rect.y < 0)
        )
          throw new TypeError(
            "options.rect.y, when provided, must be a non-negative integer.",
          );
        if (
          i.rect.width !== void 0 &&
          (!Number.isInteger(i.rect.width) || i.rect.width < 0)
        )
          throw new TypeError(
            "options.rect.width, when provided, must be a non-negative integer.",
          );
        if (
          i.rect.height !== void 0 &&
          (!Number.isInteger(i.rect.height) || i.rect.height < 0)
        )
          throw new TypeError(
            "options.rect.height, when provided, must be a non-negative integer.",
          );
      }
    },
    zd = (i, e, t) => {
      const r = qr(i),
        n = [];
      let s = 0;
      for (const a of r) {
        const o = Math.ceil(e / a.widthDivisor),
          c = Math.ceil(t / a.heightDivisor),
          l = o * a.sampleBytes,
          u = l * c;
        (n.push({ offset: s, stride: l }), (s += u));
      }
      return n;
    },
    qr = (i) => {
      const e = (t, r, n, s, a) => {
        const o = [
          { sampleBytes: t, widthDivisor: 1, heightDivisor: 1 },
          { sampleBytes: r, widthDivisor: n, heightDivisor: s },
          { sampleBytes: r, widthDivisor: n, heightDivisor: s },
        ];
        return (
          a && o.push({ sampleBytes: t, widthDivisor: 1, heightDivisor: 1 }),
          o
        );
      };
      switch (i) {
        case "I420":
          return e(1, 1, 2, 2, !1);
        case "I420P10":
        case "I420P12":
          return e(2, 2, 2, 2, !1);
        case "I420A":
          return e(1, 1, 2, 2, !0);
        case "I420AP10":
        case "I420AP12":
          return e(2, 2, 2, 2, !0);
        case "I422":
          return e(1, 1, 2, 1, !1);
        case "I422P10":
        case "I422P12":
          return e(2, 2, 2, 1, !1);
        case "I422A":
          return e(1, 1, 2, 1, !0);
        case "I422AP10":
        case "I422AP12":
          return e(2, 2, 2, 1, !0);
        case "I444":
          return e(1, 1, 1, 1, !1);
        case "I444P10":
        case "I444P12":
          return e(2, 2, 1, 1, !1);
        case "I444A":
          return e(1, 1, 1, 1, !0);
        case "I444AP10":
        case "I444AP12":
          return e(2, 2, 1, 1, !0);
        case "NV12":
          return [
            { sampleBytes: 1, widthDivisor: 1, heightDivisor: 1 },
            { sampleBytes: 2, widthDivisor: 2, heightDivisor: 2 },
          ];
        case "RGBA":
        case "RGBX":
        case "BGRA":
        case "BGRX":
          return [{ sampleBytes: 4, widthDivisor: 1, heightDivisor: 1 }];
        default:
          (xe(i), g(!1));
      }
    },
    oc = (i, e) => {
      const t = { left: 0, top: 0, width: i.codedWidth, height: i.codedHeight },
        r = e.rect,
        n = Nd(t, r, i.codedWidth, i.codedHeight, i.format),
        s = e.layout;
      let a;
      if (!e.format || e.format === i.format) a = i.format;
      else if (["RGBA", "RGBX", "BGRA", "BGRX"].includes(e.format))
        a = e.format;
      else throw new Error("NotSupportedError: Invalid destination format.");
      return Wd(n, a, s);
    },
    Nd = (i, e, t, r, n) => {
      const s = { ...i };
      if (e !== void 0) {
        if (e.width === 0 || e.height === 0)
          throw new TypeError("visibleRect dimensions cannot be zero.");
        if ((e.x || 0) + (e.width || 0) > t)
          throw new TypeError("visibleRect exceeds codedWidth.");
        if ((e.y || 0) + (e.height || 0) > r)
          throw new TypeError("visibleRect exceeds codedHeight.");
        ((s.x = e.x || 0),
          (s.y = e.y || 0),
          (s.width = e.width || 0),
          (s.height = e.height || 0));
      }
      if (!Ud(n, s))
        throw new TypeError("visibleRect alignment is invalid for the format.");
      return s;
    },
    Ud = (i, e) => {
      if (i === null) return !0;
      const t = qr(i);
      for (let r = 0; r < t.length; r++) {
        const n = t[r],
          s = n.widthDivisor,
          a = n.heightDivisor;
        if ((e.x || 0) % s !== 0 || (e.y || 0) % a !== 0) return !1;
      }
      return !0;
    },
    Wd = (i, e, t) => {
      const r = qr(e),
        n = r.length;
      if (t !== void 0 && t.length !== n)
        throw new TypeError(`Layout must have ${n} planes.`);
      let s = 0;
      const a = [],
        o = [];
      for (let c = 0; c < n; c++) {
        const l = r[c],
          u = l.sampleBytes,
          d = l.widthDivisor,
          f = l.heightDivisor,
          h = {
            destinationOffset: 0,
            destinationStride: 0,
            sourceTop: 0,
            sourceHeight: 0,
            sourceLeftBytes: 0,
            sourceWidthBytes: 0,
          };
        if (
          ((h.sourceTop = Math.ceil(Math.trunc(i.y || 0) / f)),
          (h.sourceHeight = Math.ceil(Math.trunc(i.height || 0) / f)),
          (h.sourceLeftBytes = Math.floor(Math.trunc(i.x || 0) / d) * u),
          (h.sourceWidthBytes = Math.floor(Math.trunc(i.width || 0) / d) * u),
          t !== void 0)
        ) {
          const y = t[c];
          if (y.stride < h.sourceWidthBytes)
            throw new TypeError(`Stride for plane ${c} is too small.`);
          ((h.destinationOffset = y.offset), (h.destinationStride = y.stride));
        } else
          ((h.destinationOffset = s),
            (h.destinationStride = h.sourceWidthBytes));
        const m = h.destinationStride * h.sourceHeight + h.destinationOffset;
        if (m > 4294967295)
          throw new TypeError("Allocation size exceeds limit.");
        (o.push(m), (s = Math.max(s, m)));
        for (let y = 0; y < c; y++) {
          const w = a[y];
          if (!(o[c] <= w.destinationOffset || o[y] <= h.destinationOffset))
            throw new TypeError("Planes overlap.");
        }
        a.push(h);
      }
      return { allocationSize: s, computedLayouts: a };
    },
    jr = new Set([
      "f32",
      "f32-planar",
      "s16",
      "s16-planar",
      "s32",
      "s32-planar",
      "u8",
      "u8-planar",
    ]);
  class ur {
    constructor() {
      this._referenceCount = 0;
    }
  }
  class Ie {
    get microsecondTimestamp() {
      return Math.trunc(_t * this.timestamp);
    }
    get microsecondDuration() {
      return Math.trunc(_t * this.duration);
    }
    constructor(e) {
      if (((this._closed = !1), dr(e))) {
        if (e.format === null)
          throw new TypeError("AudioData with null format is not supported.");
        ((this._data = e),
          (this.format = e.format),
          (this.sampleRate = e.sampleRate),
          (this.numberOfFrames = e.numberOfFrames),
          (this.numberOfChannels = e.numberOfChannels),
          (this.timestamp = e.timestamp / 1e6),
          (this.duration = e.numberOfFrames / e.sampleRate));
      } else if (e instanceof ur) {
        if (
          ((this._data = e),
          e._referenceCount++,
          (this.format = e.getFormat()),
          !jr.has(this.format))
        )
          throw new TypeError("getFormat() must return an AudioSampleFormat.");
        if (
          ((this.sampleRate = e.getSampleRate()),
          !Number.isInteger(this.sampleRate) || this.sampleRate <= 0)
        )
          throw new TypeError(
            "getSampleRate() must return a positive integer.",
          );
        if (
          ((this.numberOfFrames = e.getNumberOfFrames()),
          !Number.isInteger(this.numberOfFrames) || this.numberOfFrames < 0)
        )
          throw new TypeError(
            "getNumberOfFrames() must return a non-negative integer.",
          );
        if (
          ((this.numberOfChannels = e.getNumberOfChannels()),
          !Number.isInteger(this.numberOfChannels) ||
            this.numberOfChannels <= 0)
        )
          throw new TypeError(
            "getNumberOfChannels() must return a positive integer.",
          );
        if (
          ((this.timestamp = e.getTimestamp()),
          !Number.isFinite(this.timestamp))
        )
          throw new TypeError("getTimestamp() must return a finite number.");
        this.duration = this.numberOfFrames / this.sampleRate;
      } else {
        if (!e || typeof e != "object")
          throw new TypeError("Invalid AudioDataInit: must be an object.");
        if (!jr.has(e.format))
          throw new TypeError("Invalid AudioDataInit: invalid format.");
        if (!Number.isFinite(e.sampleRate) || e.sampleRate <= 0)
          throw new TypeError("Invalid AudioDataInit: sampleRate must be > 0.");
        if (!Number.isInteger(e.numberOfChannels) || e.numberOfChannels === 0)
          throw new TypeError(
            "Invalid AudioDataInit: numberOfChannels must be an integer > 0.",
          );
        if (!Number.isFinite(e?.timestamp))
          throw new TypeError("init.timestamp must be a number.");
        const t = e.data.byteLength / (Ct(e.format) * e.numberOfChannels);
        if (!Number.isInteger(t))
          throw new TypeError(
            "Invalid AudioDataInit: data size is not a multiple of frame size.",
          );
        ((this.format = e.format),
          (this.sampleRate = e.sampleRate),
          (this.numberOfFrames = t),
          (this.numberOfChannels = e.numberOfChannels),
          (this.timestamp = e.timestamp),
          (this.duration = t / e.sampleRate));
        let r;
        if (e.data instanceof ArrayBuffer) r = new Uint8Array(e.data);
        else if (ArrayBuffer.isView(e.data))
          r = new Uint8Array(
            e.data.buffer,
            e.data.byteOffset,
            e.data.byteLength,
          );
        else
          throw new TypeError(
            "Invalid AudioDataInit: data is not a BufferSource.",
          );
        const n = this.numberOfFrames * this.numberOfChannels * Ct(this.format);
        if (r.byteLength < n)
          throw new TypeError("Invalid AudioDataInit: insufficient data size.");
        this._data = r;
      }
      or?.register(this, { type: "audio", data: this._data }, this);
    }
    allocationSize(e) {
      if (!e || typeof e != "object")
        throw new TypeError("options must be an object.");
      if (!Number.isInteger(e.planeIndex) || e.planeIndex < 0)
        throw new TypeError("planeIndex must be a non-negative integer.");
      if (e.format !== void 0 && !jr.has(e.format))
        throw new TypeError("Invalid format.");
      if (
        e.frameOffset !== void 0 &&
        (!Number.isInteger(e.frameOffset) || e.frameOffset < 0)
      )
        throw new TypeError("frameOffset must be a non-negative integer.");
      if (
        e.frameCount !== void 0 &&
        (!Number.isInteger(e.frameCount) || e.frameCount < 0)
      )
        throw new TypeError("frameCount must be a non-negative integer.");
      if (this._closed) throw new Error("AudioSample is closed.");
      const t = e.format ?? this.format,
        r = e.frameOffset ?? 0;
      if (r >= this.numberOfFrames)
        throw new RangeError("frameOffset out of range");
      const n =
        e.frameCount !== void 0 ? e.frameCount : this.numberOfFrames - r;
      if (n > this.numberOfFrames - r)
        throw new RangeError("frameCount out of range");
      const s = Ct(t),
        a = Jt(t);
      if (a && e.planeIndex >= this.numberOfChannels)
        throw new RangeError("planeIndex out of range");
      if (!a && e.planeIndex !== 0)
        throw new RangeError("planeIndex out of range");
      return (a ? n : n * this.numberOfChannels) * s;
    }
    copyTo(e, t) {
      if (!kr(e))
        throw new TypeError(
          "destination must be an ArrayBuffer or an ArrayBuffer view.",
        );
      if (!t || typeof t != "object")
        throw new TypeError("options must be an object.");
      if (!Number.isInteger(t.planeIndex) || t.planeIndex < 0)
        throw new TypeError("planeIndex must be a non-negative integer.");
      if (t.format !== void 0 && !jr.has(t.format))
        throw new TypeError("Invalid format.");
      if (
        t.frameOffset !== void 0 &&
        (!Number.isInteger(t.frameOffset) || t.frameOffset < 0)
      )
        throw new TypeError("frameOffset must be a non-negative integer.");
      if (
        t.frameCount !== void 0 &&
        (!Number.isInteger(t.frameCount) || t.frameCount < 0)
      )
        throw new TypeError("frameCount must be a non-negative integer.");
      if (this._closed) throw new Error("AudioSample is closed.");
      const { format: r, frameCount: n, frameOffset: s } = t;
      let { planeIndex: a } = t,
        o = this.format;
      const c = r ?? this.format;
      if (!c) throw new Error("Destination format not determined");
      const l = this.numberOfFrames,
        u = this.numberOfChannels,
        d = s ?? 0;
      if (d >= l) throw new RangeError("frameOffset out of range");
      const f = n !== void 0 ? n : l - d;
      if (f > l - d) throw new RangeError("frameCount out of range");
      const h = Ct(c),
        p = Jt(c);
      if (p && a >= u) throw new RangeError("planeIndex out of range");
      if (!p && a !== 0) throw new RangeError("planeIndex out of range");
      const y = (p ? f : f * u) * h;
      if (e.byteLength < y)
        throw new RangeError("Destination buffer is too small");
      const w = j(e),
        b = lc(c);
      if (dr(this._data))
        if (Ui() && u > 2 && c !== o) {
          Ld(this._data, w, o, c, u, a, d, f);
          return;
        } else
          try {
            this._data.copyTo(e, {
              planeIndex: a,
              frameOffset: d,
              frameCount: f,
              format: c,
            });
            return;
          } catch (E) {
            if (c === "f32-planar") throw E;
            o = "f32-planar";
          }
      const T = cc(o),
        S = Ct(o),
        k = Jt(o);
      let P;
      if (this._data instanceof ur) {
        const E = (C) => {
          const _ = this._data.getDataPlane(C);
          if (!(_ instanceof Uint8Array))
            throw new TypeError("getDataPlane() must return a Uint8Array.");
          const v = l * S * (k ? 1 : u);
          if (_.byteLength !== v)
            throw new TypeError(
              `Data plane ${C} has invalid size. Expected exactly ${v} bytes, got ${_.byteLength} bytes.`,
            );
          return _;
        };
        if (k)
          if (p) ((P = E(a)), (a = 0));
          else {
            P = new Uint8Array(l * S * u);
            for (let C = 0; C < u; C++) {
              const _ = E(C);
              P.set(_, C * l * S);
            }
          }
        else P = E(0);
      } else if (this._data instanceof Uint8Array) P = this._data;
      else if ((g(o === "f32-planar"), p))
        ((P = new Uint8Array(
          this._data.allocationSize({ format: "f32-planar", planeIndex: a }),
        )),
          this._data.copyTo(P, { format: "f32-planar", planeIndex: a }),
          (a = 0));
      else {
        P = new Uint8Array(
          this._data.allocationSize({ format: "f32-planar", planeIndex: 0 }) *
            u,
        );
        for (let E = 0; E < u; E++)
          this._data.copyTo(P.subarray(E * l * S, (E + 1) * l * S), {
            format: "f32-planar",
            planeIndex: E,
          });
      }
      const I = j(P);
      for (let E = 0; E < f; E++)
        if (p) {
          const C = E * h;
          let _;
          k ? (_ = (a * l + (E + d)) * S) : (_ = ((E + d) * u + a) * S);
          const v = T(I, _);
          b(w, C, v);
        } else
          for (let C = 0; C < u; C++) {
            const v = (E * u + C) * h;
            let F;
            k ? (F = (C * l + (E + d)) * S) : (F = ((E + d) * u + C) * S);
            const D = T(I, F);
            b(w, v, D);
          }
    }
    clone() {
      if (this._closed) throw new Error("AudioSample is closed.");
      if (this._data instanceof ur) {
        const e = new Ie(this._data);
        return (e.setTimestamp(this.timestamp), e);
      } else if (dr(this._data)) {
        const e = new Ie(this._data.clone());
        return (e.setTimestamp(this.timestamp), e);
      } else
        return new Ie({
          format: this.format,
          sampleRate: this.sampleRate,
          numberOfFrames: this.numberOfFrames,
          numberOfChannels: this.numberOfChannels,
          timestamp: this.timestamp,
          data: this._data,
        });
    }
    trim(e, t = this.numberOfFrames) {
      if (!Number.isInteger(e) || e < 0)
        throw new TypeError("startFrame must be a non-negative integer.");
      if (!Number.isInteger(t) || t < 0)
        throw new TypeError("endFrame must be a non-negative integer.");
      if (e > this.numberOfFrames)
        throw new RangeError("startFrame out of range.");
      if (t > this.numberOfFrames)
        throw new RangeError("endFrame out of range.");
      if (t < e)
        throw new RangeError("endFrame must not be less than startFrame.");
      if (this._closed) throw new Error("AudioSample is closed.");
      const r = t - e,
        n = Ct(this.format);
      let s;
      if (Jt(this.format)) {
        const a = r * n;
        if (((s = new Uint8Array(a * this.numberOfChannels)), r > 0))
          for (let o = 0; o < this.numberOfChannels; o++)
            this.copyTo(s.subarray(o * a, (o + 1) * a), {
              planeIndex: o,
              format: this.format,
              frameOffset: e,
              frameCount: r,
            });
      } else
        ((s = new Uint8Array(r * this.numberOfChannels * n)),
          r > 0 &&
            this.copyTo(s, {
              planeIndex: 0,
              format: this.format,
              frameOffset: e,
              frameCount: r,
            }));
      return new Ie({
        data: s,
        format: this.format,
        sampleRate: this.sampleRate,
        numberOfChannels: this.numberOfChannels,
        timestamp: this.timestamp + e / this.sampleRate,
      });
    }
    close() {
      this._closed ||
        (or?.unregister(this),
        this._data instanceof ur
          ? (this._data._referenceCount--,
            this._data._referenceCount === 0 && this._data.close())
          : dr(this._data)
            ? this._data.close()
            : (this._data = new Uint8Array(0)),
        (this._closed = !0));
    }
    toAudioData() {
      if (this._closed) throw new Error("AudioSample is closed.");
      return this._data instanceof ur
        ? this._createAudioDataFromData()
        : dr(this._data)
          ? this._data.timestamp === this.microsecondTimestamp
            ? this._data.clone()
            : this._createAudioDataFromData()
          : new AudioData({
              format: this.format,
              sampleRate: this.sampleRate,
              numberOfFrames: this.numberOfFrames,
              numberOfChannels: this.numberOfChannels,
              timestamp: this.microsecondTimestamp,
              data:
                this._data.buffer instanceof ArrayBuffer
                  ? this._data.buffer
                  : this._data.slice(),
            });
    }
    _createAudioDataFromData() {
      if (Jt(this.format)) {
        const e = this.allocationSize({ planeIndex: 0, format: this.format }),
          t = new ArrayBuffer(e * this.numberOfChannels);
        for (let r = 0; r < this.numberOfChannels; r++)
          this.copyTo(new Uint8Array(t, r * e, e), {
            planeIndex: r,
            format: this.format,
          });
        return new AudioData({
          format: this.format,
          sampleRate: this.sampleRate,
          numberOfFrames: this.numberOfFrames,
          numberOfChannels: this.numberOfChannels,
          timestamp: this.microsecondTimestamp,
          data: t,
        });
      } else {
        const e = new ArrayBuffer(
          this.allocationSize({ planeIndex: 0, format: this.format }),
        );
        return (
          this.copyTo(e, { planeIndex: 0, format: this.format }),
          new AudioData({
            format: this.format,
            sampleRate: this.sampleRate,
            numberOfFrames: this.numberOfFrames,
            numberOfChannels: this.numberOfChannels,
            timestamp: this.microsecondTimestamp,
            data: e,
          })
        );
      }
    }
    toAudioBuffer() {
      if (this._closed) throw new Error("AudioSample is closed.");
      const e = new AudioBuffer({
          numberOfChannels: this.numberOfChannels,
          length: this.numberOfFrames,
          sampleRate: this.sampleRate,
        }),
        t = new Float32Array(
          this.allocationSize({ planeIndex: 0, format: "f32-planar" }) / 4,
        );
      for (let r = 0; r < this.numberOfChannels; r++)
        (this.copyTo(t, { planeIndex: r, format: "f32-planar" }),
          e.copyToChannel(t, r));
      return e;
    }
    setTimestamp(e) {
      if (!Number.isFinite(e))
        throw new TypeError("newTimestamp must be a number.");
      this.timestamp = e;
    }
    [Symbol.dispose]() {
      this.close();
    }
    static *_fromAudioBuffer(e, t) {
      if (!(e instanceof AudioBuffer))
        throw new TypeError("audioBuffer must be an AudioBuffer.");
      const r = 48e3 * 5,
        n = e.numberOfChannels,
        s = e.sampleRate,
        a = e.length,
        o = Math.floor(r / n);
      let c = 0,
        l = a;
      for (; l > 0;) {
        const u = Math.min(o, l),
          d = new Float32Array(n * u);
        for (let f = 0; f < n; f++)
          e.copyFromChannel(d.subarray(f * u, (f + 1) * u), f, c);
        (yield new Ie({
          format: "f32-planar",
          sampleRate: s,
          numberOfFrames: u,
          numberOfChannels: n,
          timestamp: t + c / s,
          data: d,
        }),
          (c += u),
          (l -= u));
      }
    }
    static fromAudioBuffer(e, t) {
      if (!(e instanceof AudioBuffer))
        throw new TypeError("audioBuffer must be an AudioBuffer.");
      const r = 48e3 * 5,
        n = e.numberOfChannels,
        s = e.sampleRate,
        a = e.length,
        o = Math.floor(r / n);
      let c = 0,
        l = a;
      const u = [];
      for (; l > 0;) {
        const d = Math.min(o, l),
          f = new Float32Array(n * d);
        for (let p = 0; p < n; p++)
          e.copyFromChannel(f.subarray(p * d, (p + 1) * d), p, c);
        const h = new Ie({
          format: "f32-planar",
          sampleRate: s,
          numberOfFrames: d,
          numberOfChannels: n,
          timestamp: t + c / s,
          data: f,
        });
        (u.push(h), (c += d), (l -= d));
      }
      return u;
    }
  }
  const Ct = (i) => {
      switch (i) {
        case "u8":
        case "u8-planar":
          return 1;
        case "s16":
        case "s16-planar":
          return 2;
        case "s32":
        case "s32-planar":
          return 4;
        case "f32":
        case "f32-planar":
          return 4;
        default:
          throw new Error("Unknown AudioSampleFormat");
      }
    },
    Jt = (i) => {
      switch (i) {
        case "u8-planar":
        case "s16-planar":
        case "s32-planar":
        case "f32-planar":
          return !0;
        default:
          return !1;
      }
    },
    cc = (i) => {
      switch (i) {
        case "u8":
        case "u8-planar":
          return (e, t) => (e.getUint8(t) - 128) / 128;
        case "s16":
        case "s16-planar":
          return (e, t) => e.getInt16(t, !0) / 32768;
        case "s32":
        case "s32-planar":
          return (e, t) => e.getInt32(t, !0) / 2147483648;
        case "f32":
        case "f32-planar":
          return (e, t) => e.getFloat32(t, !0);
      }
    },
    lc = (i) => {
      switch (i) {
        case "u8":
        case "u8-planar":
          return (e, t, r) =>
            e.setUint8(t, se(Math.round(r * 128) + 128, 0, 255));
        case "s16":
        case "s16-planar":
          return (e, t, r) =>
            e.setInt16(t, se(Math.round(r * 32768), -32768, 32767), !0);
        case "s32":
        case "s32-planar":
          return (e, t, r) =>
            e.setInt32(
              t,
              se(Math.round(r * 2147483648), -2147483648, 2147483647),
              !0,
            );
        case "f32":
        case "f32-planar":
          return (e, t, r) => e.setFloat32(t, r, !0);
      }
    },
    dr = (i) => typeof AudioData < "u" && i instanceof AudioData,
    Vd = (i) => {
      switch (i) {
        case "u8-planar":
          return "u8";
        case "s16-planar":
          return "s16";
        case "s32-planar":
          return "s32";
        case "f32-planar":
          return "f32";
        default:
          return i;
      }
    },
    Ld = (i, e, t, r, n, s, a, o) => {
      const c = cc(t),
        l = lc(r),
        u = Ct(t),
        d = Ct(r),
        f = Jt(t);
      if (Jt(r))
        if (f) {
          const p = new ArrayBuffer(o * u),
            m = j(p);
          i.copyTo(p, {
            planeIndex: s,
            frameOffset: a,
            frameCount: o,
            format: t,
          });
          for (let y = 0; y < o; y++) {
            const w = y * u,
              b = y * d,
              T = c(m, w);
            l(e, b, T);
          }
        } else {
          const p = new ArrayBuffer(o * n * u),
            m = j(p);
          i.copyTo(p, {
            planeIndex: 0,
            frameOffset: a,
            frameCount: o,
            format: t,
          });
          for (let y = 0; y < o; y++) {
            const w = (y * n + s) * u,
              b = y * d,
              T = c(m, w);
            l(e, b, T);
          }
        }
      else if (f) {
        const p = o * u,
          m = new ArrayBuffer(p),
          y = j(m);
        for (let w = 0; w < n; w++) {
          i.copyTo(m, {
            planeIndex: w,
            frameOffset: a,
            frameCount: o,
            format: t,
          });
          for (let b = 0; b < o; b++) {
            const T = b * u,
              S = (b * n + w) * d,
              k = c(y, T);
            l(e, S, k);
          }
        }
      } else {
        const p = new ArrayBuffer(o * n * u),
          m = j(p);
        i.copyTo(p, {
          planeIndex: 0,
          frameOffset: a,
          frameCount: o,
          format: t,
        });
        for (let y = 0; y < o; y++)
          for (let w = 0; w < n; w++) {
            const b = y * n + w,
              T = b * u,
              S = b * d,
              k = c(m, T);
            l(e, S, k);
          }
      }
    },
    Hd = (i, e) => {
      const t = i.allocationSize({ format: e, planeIndex: 0 }),
        r = new ArrayBuffer(t);
      return (
        i.copyTo(r, { format: e, planeIndex: 0 }),
        new Ie({
          data: r,
          format: e,
          numberOfChannels: i.numberOfChannels,
          sampleRate: i.sampleRate,
          timestamp: i.timestamp,
          duration: i.duration,
        })
      );
    };
  const uc = new Map(),
    dc = new Map(),
    qd = (i) => {
      if (!i || typeof i != "object")
        throw new TypeError("Encoding config must be an object.");
      if (!Fe.includes(i.codec))
        throw new TypeError(
          `Invalid video codec '${i.codec}'. Must be one of: ${Fe.join(", ")}.`,
        );
      const e = i.bitrate;
      if (i.quality === void 0 && e === void 0)
        throw new TypeError("config.quality must be provided.");
      if (i.quality !== void 0 && e !== void 0)
        throw new TypeError(
          "config.quality and config.bitrate cannot both be provided.",
        );
      if (i.quality !== void 0 && !(i.quality instanceof ke))
        throw new TypeError(
          "config.quality, when provided, must be a Quality.",
        );
      if (
        e !== void 0 &&
        !(e instanceof ke) &&
        (!Number.isInteger(e) || e <= 0)
      )
        throw new TypeError(
          "config.bitrate, when provided, must be a positive integer or a quality.",
        );
      if (
        i.keyFrameInterval !== void 0 &&
        (!Number.isFinite(i.keyFrameInterval) || i.keyFrameInterval < 0)
      )
        throw new TypeError(
          "config.keyFrameInterval, when provided, must be a non-negative number.",
        );
      if (
        i.sizeChangeBehavior !== void 0 &&
        !["deny", "passThrough", "fill", "contain", "cover"].includes(
          i.sizeChangeBehavior,
        )
      )
        throw new TypeError(
          "config.sizeChangeBehavior, when provided, must be 'deny', 'passThrough', 'fill', 'contain' or 'cover'.",
        );
      if (i.transform !== void 0) {
        if (typeof i.transform != "object" || !i.transform)
          throw new TypeError(
            "config.transform, when provided, must be an object.",
          );
        if (
          i.transform.width !== void 0 &&
          (!Number.isInteger(i.transform.width) || i.transform.width <= 0)
        )
          throw new TypeError(
            "config.transform.width, when provided, must be a positive integer.",
          );
        if (
          i.transform.height !== void 0 &&
          (!Number.isInteger(i.transform.height) || i.transform.height <= 0)
        )
          throw new TypeError(
            "config.transform.height, when provided, must be a positive integer.",
          );
        if (
          i.transform.fit !== void 0 &&
          !["fill", "contain", "cover"].includes(i.transform.fit)
        )
          throw new TypeError(
            'config.transform.fit, when provided, must be one of "fill", "contain", or "cover".',
          );
        if (
          i.transform.width !== void 0 &&
          i.transform.height !== void 0 &&
          i.transform.fit === void 0 &&
          !["fill", "contain", "cover"].includes(i.sizeChangeBehavior)
        )
          throw new TypeError(
            "When both config.transform.width and config.transform.height are provided, config.transform.fit must also be provided.",
          );
        if (
          i.transform.fit !== void 0 &&
          ["fill", "contain", "cover"].includes(i.sizeChangeBehavior) &&
          i.transform.fit !== i.sizeChangeBehavior
        )
          throw new TypeError(
            "config.transform.fit, when provided, cannot differ from config.sizeChangeBehavior when config.sizeChangeBehavior is 'fill', 'contain' or 'cover', as sizeChangeBehavior already determines the fitting algorithm.",
          );
        if (
          i.transform.rotate !== void 0 &&
          ![0, 90, 180, 270].includes(i.transform.rotate)
        )
          throw new TypeError(
            "config.transform.rotate, when provided, must be 0, 90, 180 or 270.",
          );
        if (i.transform.flip !== void 0 && typeof i.transform.flip != "boolean")
          throw new TypeError(
            "config.transform.flip, when provided, must be a boolean.",
          );
        if (
          (i.transform.crop !== void 0 &&
            Hr(i.transform.crop, "config.transform."),
          i.transform.process !== void 0 &&
            typeof i.transform.process != "function")
        )
          throw new TypeError(
            "config.transform.process, when provided, must be a function.",
          );
        if (
          i.transform.frameRate !== void 0 &&
          (!Number.isFinite(i.transform.frameRate) ||
            i.transform.frameRate <= 0)
        )
          throw new TypeError(
            "config.transform.frameRate, when provided, must be a finite positive number.",
          );
        if (
          i.transform.force !== void 0 &&
          typeof i.transform.force != "boolean"
        )
          throw new TypeError(
            "config.transform.force, when provided, must be a boolean.",
          );
      }
      if (
        i.onEncodedPacket !== void 0 &&
        typeof i.onEncodedPacket != "function"
      )
        throw new TypeError(
          "config.onEncodedPacket, when provided, must be a function.",
        );
      if (
        i.onEncoderConfig !== void 0 &&
        typeof i.onEncoderConfig != "function"
      )
        throw new TypeError(
          "config.onEncoderConfig, when provided, must be a function.",
        );
      if (
        i.onEncodedSample !== void 0 &&
        typeof i.onEncodedSample != "function"
      )
        throw new TypeError(
          "config.onEncodedSample, when provided, must be a function.",
        );
      fc(i.codec, i);
    },
    fc = (i, e) => {
      if (!e || typeof e != "object")
        throw new TypeError("Encoding options must be an object.");
      if (e.alpha !== void 0 && !["discard", "keep"].includes(e.alpha))
        throw new TypeError(
          "options.alpha, when provided, must be 'discard' or 'keep'.",
        );
      const t = e.bitrateMode;
      if (t !== void 0 && !["constant", "variable"].includes(t))
        throw new TypeError(
          "bitrateMode, when provided, must be 'constant' or 'variable'.",
        );
      if (
        e.latencyMode !== void 0 &&
        !["quality", "realtime"].includes(e.latencyMode)
      )
        throw new TypeError(
          "latencyMode, when provided, must be 'quality' or 'realtime'.",
        );
      if (e.fullCodecString !== void 0 && typeof e.fullCodecString != "string")
        throw new TypeError(
          "fullCodecString, when provided, must be a string.",
        );
      if (e.fullCodecString !== void 0 && yt(e.fullCodecString) !== i)
        throw new TypeError(
          `fullCodecString, when provided, must be a string that matches the specified codec (${i}).`,
        );
      if (
        e.hardwareAcceleration !== void 0 &&
        !["no-preference", "prefer-hardware", "prefer-software"].includes(
          e.hardwareAcceleration,
        )
      )
        throw new TypeError(
          "hardwareAcceleration, when provided, must be 'no-preference', 'prefer-hardware' or 'prefer-software'.",
        );
      if (e.scalabilityMode !== void 0 && typeof e.scalabilityMode != "string")
        throw new TypeError(
          "scalabilityMode, when provided, must be a string.",
        );
      if (e.contentHint !== void 0 && typeof e.contentHint != "string")
        throw new TypeError("contentHint, when provided, must be a string.");
    },
    hc = (i) => {
      const e = i.bitrateMode,
        t = i.quality._toVideoRateControl(i.codec, i.width, i.height, e),
        r = (s, a, o) => ({
          codec:
            i.fullCodecString ??
            iu(i.codec, i.width, i.height, o, i.alpha === "keep"),
          width: i.width,
          height: i.height,
          displayWidth: i.squarePixelWidth,
          displayHeight: i.squarePixelHeight,
          bitrate: s,
          bitrateMode: a,
          alpha: i.alpha ?? "discard",
          framerate: i.framerate,
          latencyMode: i.latencyMode,
          hardwareAcceleration: i.hardwareAcceleration,
          scalabilityMode: i.scalabilityMode,
          contentHint: i.contentHint,
          ...su(i.codec),
        }),
        n = [];
      return (
        t.quantizer !== null &&
          n.push({
            config: r(void 0, "quantizer", t.bitrate),
            quantizer: t.quantizer,
          }),
        t.bitrateMode !== "quantizer" &&
          n.push({
            config: r(t.bitrate, t.bitrateMode, t.bitrate),
            quantizer: null,
          }),
        g(n.length > 0),
        n
      );
    },
    jd = (i) => {
      if (!i || typeof i != "object")
        throw new TypeError("Encoding config must be an object.");
      if (!Ne.includes(i.codec))
        throw new TypeError(
          `Invalid audio codec '${i.codec}'. Must be one of: ${Ne.join(", ")}.`,
        );
      const e = i.bitrate;
      if (
        i.quality === void 0 &&
        e === void 0 &&
        !(pe.includes(i.codec) || i.codec === "flac")
      )
        throw new TypeError(
          "config.quality must be provided for compressed audio codecs.",
        );
      if (i.quality !== void 0 && e !== void 0)
        throw new TypeError(
          "config.quality and config.bitrate cannot both be provided.",
        );
      if (i.quality !== void 0 && !(i.quality instanceof ke))
        throw new TypeError(
          "config.quality, when provided, must be a Quality.",
        );
      if (
        e !== void 0 &&
        !(e instanceof ke) &&
        (!Number.isInteger(e) || e <= 0)
      )
        throw new TypeError(
          "config.bitrate, when provided, must be a positive integer or a quality.",
        );
      if (i.transform !== void 0) {
        if (typeof i.transform != "object" || !i.transform)
          throw new TypeError(
            "config.transform, when provided, must be an object.",
          );
        if (
          i.transform.numberOfChannels !== void 0 &&
          (!Number.isInteger(i.transform.numberOfChannels) ||
            i.transform.numberOfChannels <= 0)
        )
          throw new TypeError(
            "config.transform.numberOfChannels, when provided, must be a positive integer.",
          );
        if (
          i.transform.sampleRate !== void 0 &&
          (!Number.isInteger(i.transform.sampleRate) ||
            i.transform.sampleRate <= 0)
        )
          throw new TypeError(
            "config.transform.sampleRate, when provided, must be a positive integer.",
          );
        if (
          i.transform.sampleFormat !== void 0 &&
          !["u8", "s16", "s32", "f32"].includes(i.transform.sampleFormat)
        )
          throw new TypeError(
            "config.transform.sampleFormat, when provided, must be one of: u8, s16, s32, f32.",
          );
        if (
          i.transform.process !== void 0 &&
          typeof i.transform.process != "function"
        )
          throw new TypeError(
            "config.transform.process, when provided, must be a function.",
          );
      }
      if (
        i.onEncodedPacket !== void 0 &&
        typeof i.onEncodedPacket != "function"
      )
        throw new TypeError(
          "config.onEncodedPacket, when provided, must be a function.",
        );
      if (
        i.onEncoderConfig !== void 0 &&
        typeof i.onEncoderConfig != "function"
      )
        throw new TypeError(
          "config.onEncoderConfig, when provided, must be a function.",
        );
      if (
        i.onEncodedSample !== void 0 &&
        typeof i.onEncodedSample != "function"
      )
        throw new TypeError(
          "config.onEncodedSample, when provided, must be a function.",
        );
      mc(i.codec, i);
    },
    mc = (i, e) => {
      if (!e || typeof e != "object")
        throw new TypeError("Encoding options must be an object.");
      const t = e.bitrateMode;
      if (t !== void 0 && !["constant", "variable"].includes(t))
        throw new TypeError(
          "bitrateMode, when provided, must be 'constant' or 'variable'.",
        );
      if (e.fullCodecString !== void 0 && typeof e.fullCodecString != "string")
        throw new TypeError(
          "fullCodecString, when provided, must be a string.",
        );
      if (e.fullCodecString !== void 0 && yt(e.fullCodecString) !== i)
        throw new TypeError(
          `fullCodecString, when provided, must be a string that matches the specified codec (${i}).`,
        );
    },
    pc = (i) => {
      const e = i.bitrateMode;
      return {
        codec:
          i.fullCodecString ?? nu(i.codec, i.numberOfChannels, i.sampleRate),
        numberOfChannels: i.numberOfChannels,
        sampleRate: i.sampleRate,
        bitrate: i.quality?._toAudioBitrate(i.codec),
        bitrateMode: i.quality?._bitrateMode ?? e,
        ...au(i.codec),
      };
    };
  class ke {
    constructor(e) {
      if (
        ((typeof e == "number" || typeof e == "string") && (e = { quality: e }),
        !e || typeof e != "object")
      )
        throw new TypeError("options must be an object.");
      if (
        e.bitrateMode !== void 0 &&
        !["constant", "variable"].includes(e.bitrateMode)
      )
        throw new TypeError(
          "options.bitrateMode, when provided, must be 'constant' or 'variable'.",
        );
      if ("quality" in e) {
        if (
          typeof e.quality == "string"
            ? !(e.quality in gc)
            : typeof e.quality != "number" || Number.isNaN(e.quality)
        )
          throw new TypeError(
            "options.quality must be a number, or one of 'very-low', 'low', 'medium', 'high' or 'very-high'.",
          );
        if (e.preferBitrate !== void 0 && typeof e.preferBitrate != "boolean")
          throw new TypeError(
            "options.preferBitrate, when provided, must be a boolean.",
          );
        if ("bitrate" in e || "quantizer" in e)
          throw new TypeError(
            "options.quality cannot be combined with options.bitrate or options.quantizer.",
          );
        ((this._quality =
          typeof e.quality == "string" ? gc[e.quality] : e.quality),
          (this._preferBitrate = e.preferBitrate ?? !1),
          (this._bitrate = void 0),
          (this._quantizer = void 0));
      } else {
        if (
          e.bitrate !== void 0 &&
          (!Number.isInteger(e.bitrate) || e.bitrate <= 0)
        )
          throw new TypeError(
            "options.bitrate, when provided, must be a positive integer.",
          );
        if (
          e.quantizer !== void 0 &&
          (!Number.isInteger(e.quantizer) || e.quantizer < 0)
        )
          throw new TypeError(
            "options.quantizer, when provided, must be a non-negative integer.",
          );
        if (e.bitrate === void 0 && e.quantizer === void 0)
          throw new TypeError(
            "At least one of options.bitrate or options.quantizer must be set.",
          );
        if ("preferBitrate" in e)
          throw new TypeError(
            "options.preferBitrate can only be combined with options.quality.",
          );
        ((this._quality = void 0),
          (this._preferBitrate = !1),
          (this._bitrate = e.bitrate),
          (this._quantizer = e.quantizer));
      }
      this._bitrateMode = e.bitrateMode;
    }
    _toVideoRateControl(e, t, r, n) {
      const s = $d[e];
      let a = null,
        o = this._bitrateMode ?? n ?? "variable";
      if (this._quantizer !== void 0) {
        if (s)
          if (this._quantizer < s.min || this._quantizer > s.max) {
            if (this._bitrate === void 0)
              throw new Error(
                `Quantizer ${this._quantizer} is out of range for codec '${e}'; must be between ${s.min} and ${s.max}.`,
              );
          } else
            ((a = this._quantizer),
              this._bitrate === void 0 && (o = "quantizer"));
        else if (this._bitrate === void 0)
          throw new Error(
            `Codec '${e}' does not support quantizer-based encoding. Provide a bitrate in the Quality to define a fallback.`,
          );
      } else
        this._bitrate === void 0 &&
          s &&
          !this._preferBitrate &&
          (g(this._quality !== void 0),
          (a = se(
            Math.round(fl(s.worst, s.best, this._quality)),
            s.min,
            s.max,
          )));
      let c;
      if (this._bitrate !== void 0) c = this._bitrate;
      else {
        let l = this._quality;
        (l === void 0 &&
          (g(a !== null && s),
          (l = se((a - s.worst) / (s.best - s.worst), 0, 1))),
          (c = wc(e, t, r, hs(l))));
      }
      return { quantizer: a, bitrate: c, bitrateMode: o };
    }
    _toVideoBitrate(e, t, r) {
      return this._bitrate !== void 0
        ? this._bitrate
        : (g(this._quality !== void 0), wc(e, t, r, hs(this._quality)));
    }
    _toAudioBitrate(e) {
      if (pe.includes(e) || e === "flac") return;
      if (this._bitrate !== void 0) return this._bitrate;
      if (this._quality === void 0)
        throw new Error(
          "This Quality defines neither a quality level nor a bitrate and therefore cannot be used for audio encoding.",
        );
      const t = hs(this._quality),
        n = {
          aac: 128e3,
          opus: 64e3,
          mp3: 16e4,
          vorbis: 64e3,
          ac3: 384e3,
          eac3: 192e3,
          dts: 768e3,
        }[e];
      if (!n) throw new Error(`Unhandled codec: ${e}`);
      let s = n * t;
      return (
        e === "aac"
          ? (s = [96e3, 128e3, 16e4, 192e3].reduce((o, c) =>
              Math.abs(c - s) < Math.abs(o - s) ? c : o,
            ))
          : e === "opus" || e === "vorbis"
            ? (s = Math.max(6e3, s))
            : e === "mp3" &&
              (s = [
                8e3, 16e3, 24e3, 32e3, 4e4, 48e3, 64e3, 8e4, 96e3, 112e3, 128e3,
                16e4, 192e3, 224e3, 256e3, 32e4,
              ].reduce((o, c) => (Math.abs(c - s) < Math.abs(o - s) ? c : o))),
        Math.round(s / 1e3) * 1e3
      );
    }
  }
  const gc = {
      "very-low": 0,
      low: 0.25,
      medium: 0.5,
      high: 0.75,
      "very-high": 1,
    },
    $d = {
      avc: { min: 0, max: 51, worst: 41, best: 16 },
      hevc: { min: 0, max: 51, worst: 41, best: 16 },
      vp9: { min: 0, max: 63, worst: 52, best: 20 },
      av1: { min: 0, max: 255, worst: 208, best: 80 },
    },
    hs = (i) => 0.3 * Math.exp(2.5538 * i),
    wc = (i, e, t, r) => {
      const n = e * t,
        s = 1920 * 1080,
        a = 3e6,
        o = Math.pow(n / s, 0.95),
        c = a * o,
        l = {
          avc: 1,
          hevc: 0.6,
          vp9: 0.6,
          av1: 0.4,
          vp8: 1.2,
          prores: 22e7 / a,
        },
        d = c * l[i] * r;
      return Math.ceil(d / 1e3) * 1e3;
    },
    yc = (i, e) => {
      if (i === "avc") return { avc: { quantizer: e } };
      if (i === "hevc") return { hevc: { quantizer: e } };
      if (i === "vp9") return { vp9: { quantizer: e } };
      if (i === "av1") return { av1: { quantizer: e } };
      g(!1);
    },
    Kd = async (i, e = {}) => {
      const {
        width: t = 1280,
        height: r = 720,
        quality: n,
        bitrate: s,
        frameRate: a,
        ...o
      } = e;
      if (!Fe.includes(i)) return !1;
      if (!Number.isInteger(t) || t <= 0)
        throw new TypeError("width must be a positive integer.");
      if (!Number.isInteger(r) || r <= 0)
        throw new TypeError("height must be a positive integer.");
      if (n !== void 0 && !(n instanceof ke))
        throw new TypeError("quality, when provided, must be a Quality.");
      if (n !== void 0 && s !== void 0)
        throw new TypeError("quality and bitrate cannot both be provided.");
      if (
        s !== void 0 &&
        !(s instanceof ke) &&
        (!Number.isInteger(s) || s <= 0)
      )
        throw new TypeError("bitrate must be a positive integer or a quality.");
      if (a !== void 0 && (!Number.isFinite(a) || a <= 0))
        throw new TypeError(
          "frameRate, when provided, must be a finite positive number.",
        );
      fc(i, o);
      const c = Ei(n, s) ?? new ke("medium");
      let l;
      try {
        l = hc({
          codec: i,
          width: t,
          height: r,
          quality: c,
          framerate: a,
          ...o,
          alpha: "discard",
        });
      } catch {
        return !1;
      }
      const u = JSON.stringify(l),
        d = uc.get(u);
      if (d) return d;
      const f = (async () => {
        for (const { config: p } of l)
          if (Sc.some((m) => m.supports(i, p))) return !0;
        if (
          typeof VideoEncoder > "u" ||
          ((t % 2 === 1 || r % 2 === 1) && (i === "avc" || i === "hevc"))
        )
          return !1;
        for (const { config: p, quantizer: m } of l) {
          try {
            if (!(await VideoEncoder.isConfigSupported(p)).supported) continue;
          } catch {
            continue;
          }
          if (
            !ta() ||
            (await new Promise(async (w) => {
              try {
                const b = new VideoEncoder({
                  output: () => {},
                  error: () => w(!1),
                });
                b.configure(p);
                const T = new Uint8Array(t * r * 4),
                  S = new VideoFrame(T, {
                    format: "RGBA",
                    codedWidth: t,
                    codedHeight: r,
                    timestamp: 0,
                  });
                (b.encode(S, m !== null ? yc(i, m) : void 0),
                  S.close(),
                  await b.flush(),
                  w(!0));
              } catch {
                w(!1);
              }
            }))
          )
            return !0;
        }
        return !1;
      })();
      return (uc.set(u, f), f);
    },
    Gd = async (i, e = {}) => {
      const {
        numberOfChannels: t = 2,
        sampleRate: r = 48e3,
        quality: n,
        bitrate: s,
        ...a
      } = e;
      if (!Ne.includes(i)) return !1;
      if (!Number.isInteger(t) || t <= 0)
        throw new TypeError("numberOfChannels must be a positive integer.");
      if (!Number.isInteger(r) || r <= 0)
        throw new TypeError("sampleRate must be a positive integer.");
      if (n !== void 0 && !(n instanceof ke))
        throw new TypeError("quality, when provided, must be a Quality.");
      if (n !== void 0 && s !== void 0)
        throw new TypeError("quality and bitrate cannot both be provided.");
      if (
        s !== void 0 &&
        !(s instanceof ke) &&
        (!Number.isInteger(s) || s <= 0)
      )
        throw new TypeError("bitrate must be a positive integer.");
      mc(i, a);
      const o = Ei(n, s) ?? new ke("medium"),
        c = pc({
          codec: i,
          numberOfChannels: t,
          sampleRate: r,
          quality: o,
          ...a,
        }),
        l = JSON.stringify(c),
        u = dc.get(l);
      if (u) return u;
      const d = (async () => {
        if (xc.some((f) => f.supports(i, c)) || pe.includes(i)) return !0;
        if (typeof AudioEncoder > "u") return !1;
        try {
          return (await AudioEncoder.isConfigSupported(c)).supported === !0;
        } catch {
          return !1;
        }
      })();
      return (dc.set(l, d), d);
    },
    Ei = (i, e) => {
      if (i !== void 0) return i;
      if (e !== void 0) return e instanceof ke ? e : new ke({ bitrate: e });
    },
    bc = async (i = Ne, e) => {
      const t = await Promise.all(i.map((r) => Gd(r, e)));
      return i.filter((r, n) => t[n]);
    },
    Xd = async (i, e) => {
      for (const t of i) if (await Kd(t, e)) return t;
      return null;
    };
  const kc = [],
    Tc = [],
    Sc = [],
    xc = [];
  const Qd = (i) => {
      let r = i >> 2,
        n = 4096,
        s = 0,
        a = 12,
        o = 0;
      for (
        r < 0 && ((r = -r), (s = 128)), r += 33, r > 8191 && (r = 8191);
        (r & n) !== n && a >= 5;
      )
        ((n >>= 1), a--);
      return ((o = (r >> (a - 4)) & 15), ~(s | ((a - 5) << 4) | o) & 255);
    },
    Yd = (i) => {
      let t = 0,
        r = 0,
        n = ~i;
      (n & 128 && ((n &= -129), (t = -1)), (r = ((n & 240) >> 4) + 5));
      const s = ((1 << r) | ((n & 15) << (r - 4)) | (1 << (r - 5))) - 33;
      return (t === 0 ? s : -s) << 2;
    },
    Zd = (i) => {
      let t = 2048,
        r = 0,
        n = 11,
        s = 0,
        a = i >> 3;
      for (
        a < 0 ? (a = -a - 1) : (r = 128), a > 4095 && (a = 4095);
        (a & t) !== t && n >= 5;
      )
        ((t >>= 1), n--);
      return (
        (s = (a >> (n === 4 ? 1 : n - 4)) & 15),
        (r | ((n - 4) << 4) | s) ^ 85
      );
    },
    Jd = (i) => {
      let e = 0,
        t = 0,
        r = i ^ 85;
      (r & 128 && ((r &= -129), (e = -1)), (t = ((r & 240) >> 4) + 4));
      let n = 0;
      return (
        t !== 4
          ? (n = (1 << t) | ((r & 15) << (t - 4)) | (1 << (t - 5)))
          : (n = (r << 1) | 1),
        (e === 0 ? -n : n) << 3
      );
    };
  const ei = (i) => {
      if (!i || typeof i != "object")
        throw new TypeError("options must be an object.");
      if (i.metadataOnly !== void 0 && typeof i.metadataOnly != "boolean")
        throw new TypeError(
          "options.metadataOnly, when defined, must be a boolean.",
        );
      if (
        i.verifyKeyPackets !== void 0 &&
        typeof i.verifyKeyPackets != "boolean"
      )
        throw new TypeError(
          "options.verifyKeyPackets, when defined, must be a boolean.",
        );
      if (i.verifyKeyPackets && i.metadataOnly)
        throw new TypeError(
          "options.verifyKeyPackets and options.metadataOnly cannot be enabled together.",
        );
      if (i.skipLiveWait !== void 0 && typeof i.skipLiveWait != "boolean")
        throw new TypeError(
          "options.skipLiveWait, when defined, must be a boolean.",
        );
    },
    ti = (i) => {
      if (!gt(i)) throw new TypeError("timestamp must be a number.");
    },
    ms = (i, e, t) =>
      t.verifyKeyPackets
        ? e.then(async (r) => {
            if (!r || r.type === "delta") return r;
            const n = await i.determinePacketType(r);
            return (n && (r.type = n), r);
          })
        : e;
  class zt {
    constructor(e) {
      if (!(e instanceof fr))
        throw new TypeError("track must be an InputTrack.");
      this._track = e;
    }
    async getFirstPacket(e = {}) {
      if ((ei(e), this._track.input._disposed)) throw new ve();
      return ms(this._track, this._track._backing.getFirstPacket(e), e);
    }
    async getFirstKeyPacket(e = {}) {
      ei(e);
      const t = await this.getFirstPacket(e);
      return t ? (t.type === "key" ? t : this.getNextKeyPacket(t, e)) : null;
    }
    async getPacket(e, t = {}) {
      if ((ti(e), ei(t), this._track.input._disposed)) throw new ve();
      return ms(this._track, this._track._backing.getPacket(e, t), t);
    }
    async getNextPacket(e, t = {}) {
      if (!(e instanceof Z))
        throw new TypeError("packet must be an EncodedPacket.");
      if ((ei(t), this._track.input._disposed)) throw new ve();
      return ms(this._track, this._track._backing.getNextPacket(e, t), t);
    }
    async getKeyPacket(e, t = {}) {
      if ((ti(e), ei(t), this._track.input._disposed)) throw new ve();
      if (!t.verifyKeyPackets) return this._track._backing.getKeyPacket(e, t);
      const r = await this._track._backing.getKeyPacket(e, t);
      return (
        r &&
        (g(r.type === "key"),
        (await this._track.determinePacketType(r)) === "delta"
          ? this.getKeyPacket(
              r.timestamp - 1 / (await this._track.getTimeResolution()),
              t,
            )
          : r)
      );
    }
    async getNextKeyPacket(e, t = {}) {
      if (!(e instanceof Z))
        throw new TypeError("packet must be an EncodedPacket.");
      if ((ei(t), this._track.input._disposed)) throw new ve();
      if (!t.verifyKeyPackets)
        return this._track._backing.getNextKeyPacket(e, t);
      const r = await this._track._backing.getNextKeyPacket(e, t);
      return (
        r &&
        (g(r.type === "key"),
        (await this._track.determinePacketType(r)) === "delta"
          ? this.getNextKeyPacket(r, t)
          : r)
      );
    }
    packets(e, t, r = {}) {
      if (e !== void 0 && !(e instanceof Z))
        throw new TypeError("startPacket must be an EncodedPacket.");
      if (e !== void 0 && e.isMetadataOnly && !r?.metadataOnly)
        throw new TypeError(
          "startPacket can only be metadata-only if options.metadataOnly is enabled.",
        );
      if (t !== void 0 && !(t instanceof Z))
        throw new TypeError("endPacket must be an EncodedPacket.");
      if ((ei(r), this._track.input._disposed)) throw new ve();
      const n = [];
      let { promise: s, resolve: a } = ce(),
        { promise: o, resolve: c } = ce(),
        l = !1,
        u = !1,
        d = null,
        f = !1;
      const h = [],
        p = () => Math.max(2, h.length);
      (async () => {
        let y = e ?? (await this.getFirstPacket(r));
        for (
          ;
          y &&
          !u &&
          !this._track.input._disposed &&
          !(t && y.sequenceNumber >= t?.sequenceNumber);
        ) {
          if (n.length > p()) {
            (({ promise: o, resolve: c } = ce()), await o);
            continue;
          }
          (n.push(y),
            a(),
            ({ promise: s, resolve: a } = ce()),
            (y = await this.getNextPacket(y, r)));
        }
        ((l = !0), a());
      })().catch((y) => {
        f || ((d = y), (f = !0), a());
      });
      const m = this._track;
      return {
        async next() {
          for (;;) {
            if (m.input._disposed) throw new ve();
            if (u) return { value: void 0, done: !0 };
            if (f) throw d;
            if (n.length > 0) {
              const y = n.shift(),
                w = performance.now();
              for (h.push(w); h.length > 0 && w - h[0] >= 1e3;) h.shift();
              return (c(), { value: y, done: !1 });
            } else {
              if (l) return { value: void 0, done: !0 };
              await s;
            }
          }
        },
        async return() {
          return ((u = !0), c(), a(), { value: void 0, done: !0 });
        },
        async throw(y) {
          throw y;
        },
        [Symbol.asyncIterator]() {
          return this;
        },
      };
    }
  }
  class ps {
    constructor(e, t) {
      ((this.onSample = e), (this.onError = t));
    }
  }
  class Pc {
    mediaSamplesInRange(e = -1 / 0, t = 1 / 0, r) {
      (ti(e), ti(t));
      const n = [];
      let s = !1,
        a = null,
        { promise: o, resolve: c } = ce(),
        { promise: l, resolve: u } = ce(),
        d = !1,
        f = !1,
        h = !1,
        p = null,
        m = null,
        y = !1;
      const w = { ...r, verifyKeyPackets: !0, metadataOnly: !1 };
      (async () => {
        p = await this._createDecoder(
          (C) => {
            if ((u(), C.timestamp >= t && (f = !0), f)) {
              C.close();
              return;
            }
            (a && (C.timestamp > e ? (n.push(a), (s = !0)) : a.close()),
              C.timestamp >= e && (n.push(C), (s = !0)),
              (a = s ? null : C),
              n.length > 0 && (c(), ({ promise: o, resolve: c } = ce())));
          },
          (C) => {
            y || ((m = C), (y = !0), c());
          },
        );
        const S = this._createPacketSink(),
          k = (await S.getKeyPacket(e, w)) ?? (await S.getFirstKeyPacket(w));
        let P = k;
        const E = S.packets(k ?? void 0, void 0, w);
        for (await E.next(); P && !f && !this._track.input._disposed;) {
          const C = Cc(n.length);
          if (n.length + p.getDecodeQueueSize() > C) {
            (({ promise: l, resolve: u } = ce()), await l);
            continue;
          }
          p.decode(P);
          const _ = await E.next();
          if (_.done) break;
          P = _.value;
        }
        (await E.return(),
          !h && !this._track.input._disposed && (await p.flush()),
          !s && a && n.push(a),
          (d = !0),
          c());
      })()
        .catch((S) => {
          y || ((m = S), (y = !0), c());
        })
        .finally(() => {
          p?.close();
        });
      const b = this._track,
        T = () => {
          a?.close();
          for (const S of n) S.close();
        };
      return {
        async next() {
          for (;;) {
            if (b.input._disposed) throw ((h = !0), (f = !0), T(), new ve());
            if (h) return { value: void 0, done: !0 };
            if (y) throw ((h = !0), (f = !0), T(), m);
            if (n.length > 0) {
              const S = n.shift();
              return (u(), { value: S, done: !1 });
            } else if (!d) await o;
            else return { value: void 0, done: !0 };
          }
        },
        async return() {
          return (
            (h = !0),
            (f = !0),
            u(),
            c(),
            T(),
            { value: void 0, done: !0 }
          );
        },
        async throw(S) {
          throw S;
        },
        [Symbol.asyncIterator]() {
          return this;
        },
      };
    }
    mediaSamplesAtTimestamps(e, t) {
      ll(e);
      const r = cl(e),
        n = [],
        s = [];
      let { promise: a, resolve: o } = ce(),
        { promise: c, resolve: l } = ce(),
        u = !1,
        d = !1,
        f = null,
        h = null,
        p = !1;
      const m = (T) => {
          (s.push(T), o(), ({ promise: a, resolve: o } = ce()));
        },
        y = { ...t, verifyKeyPackets: !0, metadataOnly: !1 };
      (async () => {
        f = await this._createDecoder(
          (C) => {
            if ((l(), d)) {
              C.close();
              return;
            }
            let _ = 0;
            for (; n.length > 0 && C.timestamp - n[0] > -1e-10;)
              (_++, n.shift());
            if (_ > 0) for (let v = 0; v < _; v++) m(v < _ - 1 ? C.clone() : C);
            else C.close();
          },
          (C) => {
            p || ((h = C), (p = !0), o());
          },
        );
        const T = this._createPacketSink();
        let S = null,
          k = null,
          P = -1;
        const I = async () => {
            (g(k), g(f));
            let C = k;
            for (f.decode(C); C.sequenceNumber < P;) {
              const _ = Cc(s.length);
              for (; s.length + f.getDecodeQueueSize() > _ && !d;)
                (({ promise: c, resolve: l } = ce()), await c);
              if (d) break;
              const v = await T.getNextPacket(C, y);
              (g(v), f.decode(v), (C = v));
            }
            P = -1;
          },
          E = async () => {
            (g(f), await f.flush());
            for (let C = 0; C < n.length; C++) m(null);
            n.length = 0;
          };
        for await (const C of r) {
          if ((ti(C), d || this._track.input._disposed)) break;
          const _ = await T.getPacket(C, y),
            v = _ && (await T.getKeyPacket(C, y));
          if (!v) {
            (P !== -1 && (await I(), await E()), m(null), (S = null));
            continue;
          }
          (S &&
            (v.sequenceNumber !== k.sequenceNumber ||
              _.timestamp < S.timestamp) &&
            (await I(), await E()),
            n.push(_.timestamp),
            (P = Math.max(_.sequenceNumber, P)),
            (S = _),
            (k = v));
        }
        (!d &&
          !this._track.input._disposed &&
          (P !== -1 && (await I()), await E()),
          (u = !0),
          o());
      })()
        .catch((T) => {
          p || ((h = T), (p = !0), o());
        })
        .finally(() => {
          f?.close();
        });
      const w = this._track,
        b = () => {
          for (const T of s) T?.close();
        };
      return {
        async next() {
          for (;;) {
            if (w.input._disposed) throw ((d = !0), b(), new ve());
            if (d) return { value: void 0, done: !0 };
            if (p) throw ((d = !0), b(), h);
            if (s.length > 0) {
              const T = s.shift();
              return (g(T !== void 0), l(), { value: T, done: !1 });
            } else if (!u) await a;
            else return { value: void 0, done: !0 };
          }
        },
        async return() {
          return ((d = !0), l(), o(), b(), { value: void 0, done: !0 });
        },
        async throw(T) {
          throw T;
        },
        [Symbol.asyncIterator]() {
          return this;
        },
      };
    }
  }
  const Cc = (i) => (i === 0 ? 40 : 8);
  class ef extends ps {
    constructor(e, t, r, n, s, a, o) {
      (super(e, t),
        (this.codec = r),
        (this.decoderConfig = n),
        (this.rotation = s),
        (this.flip = a),
        (this.timeResolution = o),
        (this.decoder = null),
        (this.customDecoder = null),
        (this.customDecoderCallSerializer = new Sr()),
        (this.customDecoderQueueSize = 0),
        (this.inputTimestamps = []),
        (this.sampleQueue = []),
        (this.currentPacketIndex = 0),
        (this.raslSkipped = !1),
        (this.alphaDecoder = null),
        (this.alphaHadKeyframe = !1),
        (this.colorQueue = []),
        (this.alphaQueue = []),
        (this.merger = null),
        (this.decodedAlphaChunkCount = 0),
        (this.alphaDecoderQueueSize = 0),
        (this.nullAlphaFrameQueue = []),
        (this.currentAlphaPacketIndex = 0),
        (this.alphaRaslSkipped = !1),
        (this.finalSamples = []),
        (this.mergeAlphaPromises = []));
      const c = kc.find((l) => l.supports(r, n));
      if (c)
        ((this.customDecoder = new c()),
          (this.customDecoder.codec = r),
          (this.customDecoder.config = n),
          (this.customDecoder.onSample = (l) => {
            if (!(l instanceof _e))
              throw new TypeError(
                "The argument passed to onSample must be a VideoSample.",
              );
            this.finalizeAndEmitSample(l);
          }),
          (this.customDecoder.onError = (l) => {
            t(l);
          }),
          this.customDecoderCallSerializer
            .call(() => this.customDecoder.init())
            .catch((l) => t(l)));
      else {
        const l = (d) => {
          if (this.alphaQueue.length > 0) {
            const f = this.alphaQueue.shift();
            (g(f !== void 0), this.mergeAlpha(d, f));
          } else this.colorQueue.push(d);
        };
        if (Tn()) {
          if (r === "avc" && this.decoderConfig.description) {
            const d = Mn(me(this.decoderConfig.description));
            if (d && d.sequenceParameterSets.length > 0) {
              const f = pi(d.sequenceParameterSets[0]);
              f &&
                (f.frameMbsOnlyFlag === 0 &&
                  (this.decoderConfig = {
                    ...this.decoderConfig,
                    hardwareAcceleration: "prefer-software",
                  }),
                f.maxDecFrameBuffering !== 0 &&
                  f.bitstreamRestrictionFlag !== 1 &&
                  ((d.sequenceParameterSets[0] = wa(f)),
                  (this.decoderConfig = {
                    ...this.decoderConfig,
                    description: ma(d),
                  })));
            }
          }
          dn(this.decoderConfig.colorSpace) ||
            (this.decoderConfig = {
              ...this.decoderConfig,
              colorSpace: {
                primaries: this.decoderConfig.colorSpace?.primaries ?? "bt709",
                matrix: this.decoderConfig.colorSpace?.matrix ?? "bt709",
                transfer: this.decoderConfig.colorSpace?.transfer ?? "bt709",
                fullRange: this.decoderConfig.colorSpace?.fullRange ?? !1,
              },
            });
        }
        const u = new Error("Decoding error").stack;
        ((this.decoder = new VideoDecoder({
          output: (d) => {
            try {
              l(d);
            } catch (f) {
              this.onError(f);
            }
          },
          error: (d) => {
            ((d.stack = u), this.onError(d));
          },
        })),
          this.decoder.configure(this.decoderConfig));
      }
    }
    getDecodeQueueSize() {
      return this.customDecoder
        ? this.customDecoderQueueSize
        : (g(this.decoder),
          Math.max(
            this.decoder.decodeQueueSize,
            this.alphaDecoder?.decodeQueueSize ?? 0,
          ));
    }
    decode(e) {
      if (
        this.codec === "hevc" &&
        this.currentPacketIndex > 0 &&
        !this.raslSkipped
      ) {
        if (this.hasHevcRaslPicture(e.data)) return;
        this.raslSkipped = !0;
      }
      if (this.customDecoder)
        (this.customDecoderQueueSize++,
          this.customDecoderCallSerializer
            .call(() => this.customDecoder.decode(e))
            .catch((t) => this.onError(t))
            .finally(() => this.customDecoderQueueSize--));
      else {
        if (
          (g(this.decoder),
          Ui() || Zs(this.inputTimestamps, e.timestamp, (t) => t),
          Tn() && this.currentPacketIndex === 0)
        ) {
          if (this.codec === "avc") {
            const t = [];
            let r = !1;
            for (const s of fa(e.data, this.decoderConfig)) {
              const a = ji(e.data[s.offset]);
              if (((r ||= a >= 1 && a <= 5), a === be.AUD)) {
                if (r) break;
                t.length = 0;
              }
              (a >= 20 && a <= 31) ||
                t.push(e.data.subarray(s.offset, s.offset + s.length));
            }
            if (!this.decoderConfig.description)
              for (let s = 0; s < t.length; s++) {
                const a = t[s];
                if (ji(a[0]) !== be.SPS) continue;
                const o = pi(a);
                o &&
                  o.maxDecFrameBuffering !== 0 &&
                  o.bitstreamRestrictionFlag !== 1 &&
                  (t[s] = wa(o));
                break;
              }
            const n = xl(t, this.decoderConfig);
            e = new Z(n, e.type, e.timestamp, e.duration);
          } else if (this.codec === "hevc") {
            const t = Fl(e.data, this.decoderConfig);
            t && (e = new Z(t, e.type, e.timestamp, e.duration));
          }
        }
        (this.decoder.decode(e.toEncodedVideoChunk()), this.decodeAlphaData(e));
      }
      this.currentPacketIndex++;
    }
    decodeAlphaData(e) {
      if (!e.sideData.alpha) {
        this.pushNullAlphaFrame();
        return;
      }
      if ((this.merger || (this.merger = new tf()), !this.alphaDecoder)) {
        const r = (s) => {
            if (this.colorQueue.length > 0) {
              const a = this.colorQueue.shift();
              (g(a !== void 0), this.mergeAlpha(a, s));
            } else this.alphaQueue.push(s);
            for (
              this.decodedAlphaChunkCount++;
              this.nullAlphaFrameQueue.length > 0 &&
              this.nullAlphaFrameQueue[0] === this.decodedAlphaChunkCount;
            )
              if (
                (this.nullAlphaFrameQueue.shift(), this.colorQueue.length > 0)
              ) {
                const a = this.colorQueue.shift();
                (g(a !== void 0), this.mergeAlpha(a, null));
              } else this.alphaQueue.push(null);
            this.alphaDecoderQueueSize--;
          },
          n = new Error("Decoding error").stack;
        ((this.alphaDecoder = new VideoDecoder({
          output: (s) => {
            try {
              r(s);
            } catch (a) {
              this.onError(a);
            }
          },
          error: (s) => {
            ((s.stack = n), this.onError(s));
          },
        })),
          this.alphaDecoder.configure({
            ...this.decoderConfig,
            colorSpace: {
              fullRange: !0,
              matrix: "bt709",
              primaries: "bt709",
              transfer: "bt709",
            },
          }));
      }
      const t = vr(this.codec, this.decoderConfig, e.sideData.alpha);
      if (
        (this.alphaHadKeyframe || (this.alphaHadKeyframe = t === "key"),
        this.alphaHadKeyframe)
      ) {
        if (
          this.codec === "hevc" &&
          this.currentAlphaPacketIndex > 0 &&
          !this.alphaRaslSkipped
        ) {
          if (this.hasHevcRaslPicture(e.sideData.alpha)) {
            this.pushNullAlphaFrame();
            return;
          }
          this.alphaRaslSkipped = !0;
        }
        (this.currentAlphaPacketIndex++,
          this.alphaDecoder.decode(e.alphaToEncodedVideoChunk(t ?? e.type)),
          this.alphaDecoderQueueSize++);
      } else this.pushNullAlphaFrame();
    }
    pushNullAlphaFrame() {
      this.alphaDecoderQueueSize === 0
        ? this.alphaQueue.push(null)
        : this.nullAlphaFrameQueue.push(
            this.decodedAlphaChunkCount + this.alphaDecoderQueueSize,
          );
    }
    hasHevcRaslPicture(e) {
      for (const t of Ir(e, this.decoderConfig)) {
        const r = gi(e[t.offset]);
        if (r === de.RASL_N || r === de.RASL_R) return !0;
      }
      return !1;
    }
    sampleHandler(e) {
      if (Ui()) {
        if (
          this.sampleQueue.length > 0 &&
          e.timestamp >= ne(this.sampleQueue).timestamp
        ) {
          for (const t of this.sampleQueue) this.finalizeAndEmitSample(t);
          this.sampleQueue.length = 0;
        }
        Zs(this.sampleQueue, e, (t) => t.timestamp);
      } else {
        const t = this.inputTimestamps.shift();
        (g(t !== void 0), e.setTimestamp(t), this.finalizeAndEmitSample(e));
      }
    }
    finalizeAndEmitSample(e) {
      (e.setTimestamp(
        Math.round(e.timestamp * this.timeResolution) / this.timeResolution,
      ),
        e.setDuration(
          Math.round(e.duration * this.timeResolution) / this.timeResolution,
        ),
        e.setRotation(this.rotation),
        e.setFlip(this.flip),
        this.onSample(e));
    }
    async mergeAlpha(e, t) {
      const r = ce();
      this.mergeAlphaPromises.push(r.promise);
      const n = { sample: null };
      this.finalSamples.push(n);
      try {
        if (!t) n.sample = new _e(e);
        else {
          g(this.merger);
          const s = await this.merger.merge(e, t);
          n.sample = new _e(s);
        }
        for (
          ;
          this.finalSamples.length > 0 && this.finalSamples[0].sample !== null;
        ) {
          const s = this.finalSamples.shift();
          this.sampleHandler(s.sample);
        }
      } catch (s) {
        (fn(this.finalSamples, n), this.onError(s));
      } finally {
        (fn(this.mergeAlphaPromises, r.promise), r.resolve());
      }
    }
    async flush() {
      if (
        (this.customDecoder
          ? await this.customDecoderCallSerializer.call(() =>
              this.customDecoder.flush(),
            )
          : (g(this.decoder),
            await Promise.all([
              this.decoder.flush(),
              this.alphaDecoder?.flush(),
            ]),
            await Promise.all(this.mergeAlphaPromises),
            this.colorQueue.forEach((e) => e.close()),
            (this.colorQueue.length = 0),
            this.alphaQueue.forEach((e) => e?.close()),
            (this.alphaQueue.length = 0),
            (this.alphaHadKeyframe = !1),
            (this.decodedAlphaChunkCount = 0),
            (this.alphaDecoderQueueSize = 0),
            (this.nullAlphaFrameQueue.length = 0),
            (this.currentAlphaPacketIndex = 0),
            (this.alphaRaslSkipped = !1)),
        Ui())
      ) {
        for (const e of this.sampleQueue) this.finalizeAndEmitSample(e);
        this.sampleQueue.length = 0;
      }
      ((this.currentPacketIndex = 0), (this.raslSkipped = !1));
    }
    close() {
      this.customDecoder
        ? this.customDecoderCallSerializer.call(() =>
            this.customDecoder.close(),
          )
        : (g(this.decoder),
          this.decoder.state !== "closed" && this.decoder.close(),
          this.alphaDecoder &&
            this.alphaDecoder.state !== "closed" &&
            this.alphaDecoder.close(),
          this.colorQueue.forEach((e) => e.close()),
          (this.colorQueue.length = 0),
          this.alphaQueue.forEach((e) => e?.close()),
          (this.alphaQueue.length = 0),
          this.merger?.close());
      for (const e of this.sampleQueue) e.close();
      this.sampleQueue.length = 0;
    }
  }
  let gs = null;
  class tf {
    constructor() {
      ((this.workers = []),
        (this.nextWorkerIndex = 0),
        (this.pendingRequests = new Map()),
        (this.nextRequestId = 0));
    }
    merge(e, t) {
      if (this.workers.length === 0) {
        if (!gs) {
          const o = new Blob([`(${rf.toString()})()`], {
            type: "application/javascript",
          });
          gs = URL.createObjectURL(o);
        }
        const a = se(navigator.hardwareConcurrency, 1, 4);
        for (let o = 0; o < a; o++) {
          const c = new Worker(gs);
          (c.addEventListener("message", (l) => {
            const u = l.data,
              d = this.pendingRequests.get(u.id);
            d &&
              (this.pendingRequests.delete(u.id),
              "error" in u ? d.reject(new Error(u.error)) : d.resolve(u.frame));
          }),
            c.addEventListener("error", (l) => {
              const u = new Error(
                l.message || "Color/alpha merge worker error.",
              );
              for (const d of this.pendingRequests.values()) d.reject(u);
              this.pendingRequests.clear();
            }),
            this.workers.push(c));
        }
      }
      const r = this.nextRequestId++,
        n = ce();
      this.pendingRequests.set(r, n);
      const s = this.workers[this.nextWorkerIndex];
      return (
        (this.nextWorkerIndex =
          (this.nextWorkerIndex + 1) % this.workers.length),
        s.postMessage({ id: r, color: e, alpha: t }, { transfer: [e, t] }),
        n.promise
      );
    }
    close() {
      for (const t of this.workers) t.terminate();
      this.workers.length = 0;
      const e = new Error("Color/alpha merger closed.");
      for (const t of this.pendingRequests.values()) t.reject(e);
      this.pendingRequests.clear();
    }
  }
  const rf = () => {
      let i = null,
        e = null,
        t = Promise.resolve();
      self.addEventListener("message", (c) => {
        const { id: l, color: u, alpha: d } = c.data;
        t = t.then(async () => {
          try {
            const f = await r(u, d);
            self.postMessage({ id: l, frame: f }, { transfer: [f] });
          } catch (f) {
            self.postMessage({ id: l, error: f.message });
          } finally {
            (u.close(), d.close());
          }
        });
      });
      const r = async (c, l) => {
          const u = c.format,
            d = l.format;
          if (!u || !d)
            throw new Error(
              "CPU color/alpha merging requires a known VideoFrame format.",
            );
          const f = u.includes("P10"),
            h = u.includes("P12"),
            p = d.includes("P10"),
            m = d.includes("P12");
          if (p !== f || m !== h)
            throw new Error(
              `CPU color/alpha merging requires the alpha frame to have the same bit depth as the color frame (color: '${u}', alpha: '${d}').`,
            );
          if (u === "RGBX" || u === "RGBA" || u === "BGRX" || u === "BGRA")
            return await n(c, l, u);
          if (
            u === "I420" ||
            u === "I420P10" ||
            u === "I420P12" ||
            u === "I422" ||
            u === "I422P10" ||
            u === "I422P12" ||
            u === "I444" ||
            u === "I444P10" ||
            u === "I444P12"
          )
            return await s(c, l, u);
          if (u === "NV12") return await a(c, l);
          throw new Error(
            `CPU color/alpha merging does not support format '${u}'.`,
          );
        },
        n = async (c, l, u) => {
          const d = c.visibleRect?.width ?? c.codedWidth,
            f = c.visibleRect?.height ?? c.codedHeight,
            h = d * f,
            p = new Uint8Array(h * 4);
          await c.copyTo(p);
          const m = await o(l, d, f, 1);
          for (let b = 0, T = 3; b < h; b++, T += 4) p[T] = m[b];
          const w = {
            format: u === "RGBX" || u === "RGBA" ? "RGBA" : "BGRA",
            codedWidth: d,
            codedHeight: f,
            timestamp: c.timestamp,
            duration: c.duration ?? void 0,
            transfer: [p.buffer],
          };
          return new VideoFrame(p, w);
        },
        s = async (c, l, u) => {
          const d = c.visibleRect?.width ?? c.codedWidth,
            f = c.visibleRect?.height ?? c.codedHeight,
            h = u.includes("P10"),
            p = u.includes("P12"),
            m = h || p ? 2 : 1;
          let y, w;
          u.startsWith("I420")
            ? ((y = Math.ceil(d / 2)), (w = Math.ceil(f / 2)))
            : u.startsWith("I422")
              ? ((y = Math.ceil(d / 2)), (w = f))
              : ((y = d), (w = f));
          const b = d * f,
            T = y * w,
            S = b * m,
            k = T * m,
            P = b * m,
            I = S + 2 * k + P,
            E = new Uint8Array(I);
          await c.copyTo(E);
          const C = await o(l, d, f, m),
            _ = S + 2 * k;
          E.set(C, _);
          const F = {
            format: u.slice(0, 4) + "A" + u.slice(4),
            codedWidth: d,
            codedHeight: f,
            timestamp: c.timestamp,
            duration: c.duration ?? void 0,
            transfer: [E.buffer],
          };
          return new VideoFrame(E, F);
        },
        a = async (c, l) => {
          const u = c.visibleRect?.width ?? c.codedWidth,
            d = c.visibleRect?.height ?? c.codedHeight,
            f = u * d,
            h = Math.ceil(u / 2),
            p = Math.ceil(d / 2),
            m = h * p,
            y = c.allocationSize();
          ((!e || e.byteLength !== y) && (e = new Uint8Array(y)),
            await c.copyTo(e));
          const w = new Uint8Array(f + 2 * m + f);
          w.set(e.subarray(0, f), 0);
          const b = f,
            T = f + m,
            S = f;
          for (let I = 0; I < m; I++)
            ((w[b + I] = e[S + I * 2]), (w[T + I] = e[S + I * 2 + 1]));
          const k = await o(l, u, d, 1);
          w.set(k, f + 2 * m);
          const P = {
            format: "I420A",
            codedWidth: u,
            codedHeight: d,
            timestamp: c.timestamp,
            duration: c.duration ?? void 0,
            transfer: [w.buffer],
          };
          return new VideoFrame(w, P);
        },
        o = async (c, l, u, d) => {
          const f = c.allocationSize();
          ((!i || i.byteLength !== f) && (i = new Uint8Array(f)),
            await c.copyTo(i));
          const h = c.format;
          if (h === "RGBA" || h === "BGRA" || h === "RGBX" || h === "BGRX") {
            const p = h === "RGBA" || h === "RGBX" ? 0 : 2,
              m = l * u;
            for (let y = 0; y < m; y++) i[y] = i[y * 4 + p];
            return i.subarray(0, m);
          } else {
            const p = i.subarray(0, l * u * d);
            if (c.colorSpace.fullRange === !1) {
              const m = h.includes("P12") ? 12 : h.includes("P10") ? 10 : 8,
                y = 16 << (m - 8),
                w = 235 << (m - 8),
                b = (1 << m) - 1,
                T = d === 2 ? new Uint16Array(p.buffer, 0, l * u) : p;
              for (let S = 0; S < T.length; S++) {
                const k = Math.round(((T[S] - y) * b) / (w - y));
                T[S] = Math.min(Math.max(k, 0), b);
              }
            }
            return p;
          }
        };
    },
    nf = (i) => {
      if (!i || typeof i != "object")
        throw new TypeError("decoderOptions must be an object.");
      if (
        i.hardwareAcceleration !== void 0 &&
        !["no-preference", "prefer-hardware", "prefer-software"].includes(
          i.hardwareAcceleration,
        )
      )
        throw new TypeError(
          "decoderOptions.hardwareAcceleration, when provided, must be 'no-preference', 'prefer-hardware' or 'prefer-software'.",
        );
      if (
        i.optimizeForLatency !== void 0 &&
        typeof i.optimizeForLatency != "boolean"
      )
        throw new TypeError(
          "decoderOptions.optimizeForLatency, when provided, must be a boolean.",
        );
    };
  class Ic extends Pc {
    constructor(e, t = {}) {
      if (!(e instanceof ys))
        throw new TypeError("videoTrack must be an InputVideoTrack.");
      (nf(t), super(), (this._track = e), (this._decoderOptions = t));
    }
    async _createDecoder(e, t) {
      if (!(await this._track.canDecode()))
        throw typeof VideoDecoder > "u"
          ? new Error(xr("VideoDecoder"))
          : new Error(
              "This video track cannot be decoded in this environment. Make sure to check decodability before using a track.",
            );
      const r = await this._track.getCodec(),
        n = await this._track.getRotation(),
        s = await this._track.getFlip();
      let a = await this._track.getDecoderConfig();
      const o = await this._track.getTimeResolution();
      return (
        g(r && a),
        (a = {
          ...a,
          hardwareAcceleration: this._decoderOptions.hardwareAcceleration,
          optimizeForLatency: this._decoderOptions.optimizeForLatency,
        }),
        new ef(e, t, r, a, n, s, o)
      );
    }
    _createPacketSink() {
      return new zt(this._track);
    }
    async getSample(e, t = {}) {
      ti(e);
      for await (const r of this.mediaSamplesAtTimestamps([e], t)) return r;
      throw new Error("Internal error: Iterator returned nothing.");
    }
    samples(e, t, r = {}) {
      return this.mediaSamplesInRange(e, t, r);
    }
    samplesAtTimestamps(e, t = {}) {
      return this.mediaSamplesAtTimestamps(e, t);
    }
  }
  class sf extends ps {
    constructor(e, t, r, n) {
      (super(e, t),
        (this.decoder = null),
        (this.customDecoder = null),
        (this.customDecoderCallSerializer = new Sr()),
        (this.customDecoderQueueSize = 0),
        (this.currentTimestamp = null),
        (this.expectedFirstTimestamp = null),
        (this.timestampOffset = 0));
      const s = (o) => {
          let c = o.timestamp;
          (this.expectedFirstTimestamp !== null &&
            this.currentTimestamp === null &&
            (this.timestampOffset = this.expectedFirstTimestamp - c),
            (c += this.timestampOffset),
            (this.currentTimestamp === null ||
              Math.abs(c - this.currentTimestamp) >= o.duration) &&
              (this.currentTimestamp = c));
          const l = this.currentTimestamp;
          if (((this.currentTimestamp += o.duration), o.numberOfFrames === 0)) {
            o.close();
            return;
          }
          const u = n.sampleRate;
          (o.setTimestamp(Math.round(l * u) / u), e(o));
        },
        a = Tc.find((o) => o.supports(r, n));
      if (a)
        ((this.customDecoder = new a()),
          (this.customDecoder.codec = r),
          (this.customDecoder.config = n),
          (this.customDecoder.onSample = (o) => {
            if (!(o instanceof Ie))
              throw new TypeError(
                "The argument passed to onSample must be an AudioSample.",
              );
            s(o);
          }),
          (this.customDecoder.onError = (o) => {
            t(o);
          }),
          this.customDecoderCallSerializer
            .call(() => this.customDecoder.init())
            .catch((o) => t(o)));
      else {
        const o = new Error("Decoding error").stack;
        ((this.decoder = new AudioDecoder({
          output: (c) => {
            try {
              s(new Ie(c));
            } catch (l) {
              this.onError(l);
            }
          },
          error: (c) => {
            ((c.stack = o), this.onError(c));
          },
        })),
          this.decoder.configure(n));
      }
    }
    getDecodeQueueSize() {
      return this.customDecoder
        ? this.customDecoderQueueSize
        : (g(this.decoder), this.decoder.decodeQueueSize);
    }
    decode(e) {
      this.customDecoder
        ? (this.customDecoderQueueSize++,
          this.customDecoderCallSerializer
            .call(() => this.customDecoder.decode(e))
            .catch((t) => this.onError(t))
            .finally(() => this.customDecoderQueueSize--))
        : (g(this.decoder),
          (this.expectedFirstTimestamp ??= e.timestamp),
          this.decoder.decode(e.toEncodedAudioChunk()));
    }
    async flush() {
      (this.customDecoder
        ? await this.customDecoderCallSerializer.call(() =>
            this.customDecoder.flush(),
          )
        : (g(this.decoder), await this.decoder.flush()),
        (this.currentTimestamp = null),
        (this.expectedFirstTimestamp = null),
        (this.timestampOffset = 0));
    }
    close() {
      this.customDecoder
        ? this.customDecoderCallSerializer.call(() =>
            this.customDecoder.close(),
          )
        : (g(this.decoder),
          this.decoder.state !== "closed" && this.decoder.close());
    }
  }
  class af extends ps {
    constructor(e, t, r) {
      (super(e, t),
        (this.decoderConfig = r),
        (this.currentTimestamp = null),
        g(pe.includes(r.codec)),
        (this.codec = r.codec));
      const { dataType: n, sampleSize: s, littleEndian: a } = Ge(this.codec);
      switch (((this.inputSampleSize = s), s)) {
        case 1:
          n === "unsigned"
            ? (this.readInputValue = (o, c) => o.getUint8(c) - 2 ** 7)
            : n === "signed"
              ? (this.readInputValue = (o, c) => o.getInt8(c))
              : n === "ulaw"
                ? (this.readInputValue = (o, c) => Yd(o.getUint8(c)))
                : n === "alaw"
                  ? (this.readInputValue = (o, c) => Jd(o.getUint8(c)))
                  : g(!1);
          break;
        case 2:
          n === "unsigned"
            ? (this.readInputValue = (o, c) => o.getUint16(c, a) - 2 ** 15)
            : n === "signed"
              ? (this.readInputValue = (o, c) => o.getInt16(c, a))
              : g(!1);
          break;
        case 3:
          n === "unsigned"
            ? (this.readInputValue = (o, c) => Tr(o, c, a) - 2 ** 23)
            : n === "signed"
              ? (this.readInputValue = (o, c) => ul(o, c, a))
              : g(!1);
          break;
        case 4:
          n === "unsigned"
            ? (this.readInputValue = (o, c) => o.getUint32(c, a) - 2 ** 31)
            : n === "signed"
              ? (this.readInputValue = (o, c) => o.getInt32(c, a))
              : n === "float"
                ? (this.readInputValue = (o, c) => o.getFloat32(c, a))
                : g(!1);
          break;
        case 8:
          n === "float"
            ? (this.readInputValue = (o, c) => o.getFloat64(c, a))
            : g(!1);
          break;
        default:
          (xe(s), g(!1));
      }
      switch (s) {
        case 1:
          n === "ulaw" || n === "alaw"
            ? ((this.outputSampleSize = 2),
              (this.outputFormat = "s16"),
              (this.writeOutputValue = (o, c, l) => o.setInt16(c, l, !0)))
            : ((this.outputSampleSize = 1),
              (this.outputFormat = "u8"),
              (this.writeOutputValue = (o, c, l) => o.setUint8(c, l + 2 ** 7)));
          break;
        case 2:
          ((this.outputSampleSize = 2),
            (this.outputFormat = "s16"),
            (this.writeOutputValue = (o, c, l) => o.setInt16(c, l, !0)));
          break;
        case 3:
          ((this.outputSampleSize = 4),
            (this.outputFormat = "s32"),
            (this.writeOutputValue = (o, c, l) => o.setInt32(c, l << 8, !0)));
          break;
        case 4:
          ((this.outputSampleSize = 4),
            n === "float"
              ? ((this.outputFormat = "f32"),
                (this.writeOutputValue = (o, c, l) => o.setFloat32(c, l, !0)))
              : ((this.outputFormat = "s32"),
                (this.writeOutputValue = (o, c, l) => o.setInt32(c, l, !0))));
          break;
        case 8:
          ((this.outputSampleSize = 4),
            (this.outputFormat = "f32"),
            (this.writeOutputValue = (o, c, l) => o.setFloat32(c, l, !0)));
          break;
        default:
          (xe(s), g(!1));
      }
    }
    getDecodeQueueSize() {
      return 0;
    }
    decode(e) {
      const t = j(e.data),
        r =
          e.byteLength /
          this.decoderConfig.numberOfChannels /
          this.inputSampleSize,
        n = r * this.decoderConfig.numberOfChannels * this.outputSampleSize,
        s = new ArrayBuffer(n),
        a = new DataView(s);
      for (let u = 0; u < r * this.decoderConfig.numberOfChannels; u++) {
        const d = u * this.inputSampleSize,
          f = u * this.outputSampleSize,
          h = this.readInputValue(t, d);
        this.writeOutputValue(a, f, h);
      }
      const o = r / this.decoderConfig.sampleRate;
      (this.currentTimestamp === null ||
        Math.abs(e.timestamp - this.currentTimestamp) >= o) &&
        (this.currentTimestamp = e.timestamp);
      const c = this.currentTimestamp;
      this.currentTimestamp += o;
      const l = new Ie({
        format: this.outputFormat,
        data: s,
        numberOfChannels: this.decoderConfig.numberOfChannels,
        sampleRate: this.decoderConfig.sampleRate,
        numberOfFrames: r,
        timestamp: c,
      });
      this.onSample(l);
    }
    async flush() {}
    close() {}
  }
  class of extends Pc {
    constructor(e) {
      if (!(e instanceof bs))
        throw new TypeError("audioTrack must be an InputAudioTrack.");
      (super(), (this._track = e));
    }
    async _createDecoder(e, t) {
      if (!(await this._track.canDecode()))
        throw typeof AudioDecoder > "u"
          ? new Error(xr("AudioDecoder"))
          : new Error(
              "This audio track cannot be decoded in this environment. Make sure to check decodability before using a track.",
            );
      const r = await this._track.getCodec(),
        n = await this._track.getDecoderConfig();
      return (
        g(r && n),
        pe.includes(n.codec) ? new af(e, t, n) : new sf(e, t, r, n)
      );
    }
    _createPacketSink() {
      return new zt(this._track);
    }
    async getSample(e, t = {}) {
      ti(e);
      for await (const r of this.mediaSamplesAtTimestamps([e], t)) return r;
      throw new Error("Internal error: Iterator returned nothing.");
    }
    samples(e, t, r = {}) {
      return this.mediaSamplesInRange(e, t, r);
    }
    samplesAtTimestamps(e, t = {}) {
      return this.mediaSamplesAtTimestamps(e, t);
    }
  }
  class fr {
    constructor(e, t) {
      ((this.input = e), (this._backing = t));
    }
    isVideoTrack() {
      return this instanceof ys;
    }
    isAudioTrack() {
      return this instanceof bs;
    }
    get id() {
      return this._backing.getId();
    }
    get number() {
      return this._backing.getNumber();
    }
    async getInternalCodecId() {
      return this._backing.getInternalCodecId();
    }
    get internalCodecId() {
      return ae(
        this._backing.getInternalCodecId(),
        "internalCodecId",
        "getInternalCodecId",
      );
    }
    async getLanguageCode() {
      return this._backing.getLanguageCode();
    }
    get languageCode() {
      return ae(
        this._backing.getLanguageCode(),
        "languageCode",
        "getLanguageCode",
      );
    }
    async getName() {
      return this._backing.getName();
    }
    get name() {
      return ae(this._backing.getName(), "name", "getName");
    }
    async getTimeResolution() {
      return this._backing.getTimeResolution();
    }
    get timeResolution() {
      return ae(
        this._backing.getTimeResolution(),
        "timeResolution",
        "getTimeResolution",
      );
    }
    async isRelativeToUnixEpoch() {
      return this._backing.isRelativeToUnixEpoch();
    }
    async getUnixTimeForTimestamp(e) {
      return this._backing.getUnixTimeForTimestamp(e);
    }
    async hasUnixTimeMapping() {
      return (
        (await this._backing.getUnixTimeForTimestamp(
          await this.getFirstTimestamp(),
        )) !== null
      );
    }
    async getDisposition() {
      return this._backing.getDisposition();
    }
    get disposition() {
      return ae(
        this._backing.getDisposition(),
        "disposition",
        "getDisposition",
      );
    }
    async getBitrate() {
      return this._backing.getBitrate();
    }
    async getAverageBitrate() {
      return this._backing.getAverageBitrate();
    }
    async getFirstTimestamp() {
      return (
        (await this._backing.getFirstPacket({ metadataOnly: !0 }))?.timestamp ??
        0
      );
    }
    async computeDuration(e) {
      const t = await this._backing.getPacket(1 / 0, {
          metadataOnly: !0,
          ...e,
        }),
        r = (t?.timestamp ?? 0) + (t?.duration ?? 0);
      return li(r, await this.getTimeResolution());
    }
    async getDurationFromMetadata(e = {}) {
      return this._backing.getDurationFromMetadata(e);
    }
    async computePacketStats(e = 1 / 0, t) {
      const r = new zt(this);
      let n = 1 / 0,
        s = -1 / 0,
        a = 0,
        o = 0;
      for await (const c of r.packets(void 0, void 0, {
        metadataOnly: !0,
        ...t,
      })) {
        if (a >= e && c.timestamp >= s) break;
        ((n = Math.min(n, c.timestamp)),
          (s = Math.max(s, c.timestamp + c.duration)),
          a++,
          (o += c.byteLength));
      }
      return {
        packetCount: a,
        averagePacketRate: a ? Number((a / (s - n)).toPrecision(16)) : 0,
        averageBitrate: a ? Number(((8 * o) / (s - n)).toPrecision(16)) : 0,
      };
    }
    async isLive() {
      return (await this._backing.getLiveRefreshInterval()) !== null;
    }
    async getLiveRefreshInterval() {
      return this._backing.getLiveRefreshInterval();
    }
    canBePairedWith(e) {
      if (!(e instanceof fr))
        throw new TypeError("other must be an InputTrack.");
      return this.input !== e.input || this === e
        ? !1
        : (this._backing.getPairingMask() & e._backing.getPairingMask()) !== 0n;
    }
    async getPairableTracks(e) {
      return this.input.getTracks(
        ii({ filter: (t) => t.canBePairedWith(this) }, e),
      );
    }
    async getPairableVideoTracks(e) {
      return this.input.getVideoTracks(
        ii({ filter: (t) => t.canBePairedWith(this) }, e),
      );
    }
    async getPairableAudioTracks(e) {
      return this.input.getAudioTracks(
        ii({ filter: (t) => t.canBePairedWith(this) }, e),
      );
    }
    async getPrimaryPairableVideoTrack(e) {
      return this.input.getPrimaryVideoTrack(
        ii({ filter: (t) => t.canBePairedWith(this) }, e),
      );
    }
    async getPrimaryPairableAudioTrack(e) {
      return this.input.getPrimaryAudioTrack(
        ii({ filter: (t) => t.canBePairedWith(this) }, e),
      );
    }
    async hasPairableTrack(e) {
      e &&= ws(e);
      const t = await this.input.getTracks();
      for (const r of t)
        if (this.canBePairedWith(r) && (!e || (await e(r)))) return !0;
      return !1;
    }
    hasPairableVideoTrack(e) {
      return (
        (e &&= ws(e)),
        this.hasPairableTrack(
          async (t) => t.isVideoTrack() && (!e || (await e(t))),
        )
      );
    }
    hasPairableAudioTrack(e) {
      return (
        (e &&= ws(e)),
        this.hasPairableTrack(
          async (t) => t.isAudioTrack() && (!e || (await e(t))),
        )
      );
    }
  }
  const ae = (i, e, t) => {
      if (A(i))
        throw new Error(
          `'${e}' is deprecated and not available synchronously for this track. Use the preferred '${t}()' instead.`,
        );
      return i;
    },
    ws = (i) => {
      if (i !== void 0 && typeof i != "function")
        throw new TypeError("predicate, when provided, must be a function.");
      return i
        ? (e) => {
            const t = (n) => {
                if (typeof n != "boolean")
                  throw new TypeError(
                    "predicate must return or resolve to a boolean value.",
                  );
                return n;
              },
              r = i(e);
            return A(r) ? r.then(t) : t(r);
          }
        : void 0;
    };
  class ys extends fr {
    constructor(e, t) {
      (super(e, t), (this._pixelAspectRatioCache = null));
    }
    get type() {
      return "video";
    }
    async getCodec() {
      return this._backing.getCodec();
    }
    get codec() {
      return ae(this._backing.getCodec(), "codec", "getCodec");
    }
    async hasOnlyKeyPackets() {
      return (
        (await this._backing.getHasOnlyKeyPackets?.()) ??
        (await this._backing.getCodec()) === "prores"
      );
    }
    async getCodedWidth() {
      return this._backing.getCodedWidth();
    }
    get codedWidth() {
      return ae(this._backing.getCodedWidth(), "codedWidth", "getCodedWidth");
    }
    async getCodedHeight() {
      return this._backing.getCodedHeight();
    }
    get codedHeight() {
      return ae(
        this._backing.getCodedHeight(),
        "codedHeight",
        "getCodedHeight",
      );
    }
    async getTransformationMatrix() {
      return this._backing.getTransformationMatrix();
    }
    async getRotation() {
      return Ri(await this._backing.getTransformationMatrix());
    }
    get rotation() {
      return Ri(
        ae(this._backing.getTransformationMatrix(), "rotation", "getRotation"),
      );
    }
    async getFlip() {
      return js(await this._backing.getTransformationMatrix());
    }
    async getSquarePixelWidth() {
      return this._backing.getSquarePixelWidth();
    }
    get squarePixelWidth() {
      return ae(
        this._backing.getSquarePixelWidth(),
        "squarePixelWidth",
        "getSquarePixelWidth",
      );
    }
    async getSquarePixelHeight() {
      return this._backing.getSquarePixelHeight();
    }
    get squarePixelHeight() {
      return ae(
        this._backing.getSquarePixelHeight(),
        "squarePixelHeight",
        "getSquarePixelHeight",
      );
    }
    async getPixelAspectRatio() {
      return (this._pixelAspectRatioCache ??= di({
        num: (await this.getSquarePixelWidth()) * (await this.getCodedHeight()),
        den: (await this.getSquarePixelHeight()) * (await this.getCodedWidth()),
      }));
    }
    get pixelAspectRatio() {
      return (this._pixelAspectRatioCache ??= di({
        num:
          ae(
            this._backing.getSquarePixelWidth(),
            "pixelAspectRatio",
            "getPixelAspectRatio",
          ) *
          ae(
            this._backing.getCodedHeight(),
            "pixelAspectRatio",
            "getPixelAspectRatio",
          ),
        den:
          ae(
            this._backing.getSquarePixelHeight(),
            "pixelAspectRatio",
            "getPixelAspectRatio",
          ) *
          ae(
            this._backing.getCodedWidth(),
            "pixelAspectRatio",
            "getPixelAspectRatio",
          ),
      }));
    }
    async getDisplayWidth() {
      const e = await this._backing.getMetadataDisplayWidth?.();
      return (
        e ??
        ((await this.getRotation()) % 180 === 0
          ? this.getSquarePixelWidth()
          : this.getSquarePixelHeight())
      );
    }
    get displayWidth() {
      const e = this._backing.getMetadataDisplayWidth?.();
      if (e !== void 0) {
        const n = ae(e, "displayWidth", "getDisplayWidth");
        if (n !== null) return n;
      }
      const r =
        Ri(
          ae(
            this._backing.getTransformationMatrix(),
            "displayWidth",
            "getDisplayWidth",
          ),
        ) %
          180 ===
        0
          ? this._backing.getSquarePixelWidth()
          : this._backing.getSquarePixelHeight();
      return ae(r, "displayWidth", "getDisplayWidth");
    }
    async getDisplayHeight() {
      const e = await this._backing.getMetadataDisplayHeight?.();
      return (
        e ??
        ((await this.getRotation()) % 180 === 0
          ? this.getSquarePixelHeight()
          : this.getSquarePixelWidth())
      );
    }
    get displayHeight() {
      const e = this._backing.getMetadataDisplayHeight?.();
      if (e !== void 0) {
        const n = ae(e, "displayHeight", "getDisplayHeight");
        if (n !== null) return n;
      }
      const r =
        Ri(
          ae(
            this._backing.getTransformationMatrix(),
            "displayHeight",
            "getDisplayHeight",
          ),
        ) %
          180 ===
        0
          ? this._backing.getSquarePixelHeight()
          : this._backing.getSquarePixelWidth();
      return ae(r, "displayHeight", "getDisplayHeight");
    }
    async getColorSpace() {
      return this._backing.getColorSpace();
    }
    async hasHighDynamicRange() {
      const e = await this._backing.getColorSpace();
      return (
        e.primaries === "bt2020" ||
        e.primaries === "smpte432" ||
        e.transfer === "pq" ||
        e.transfer === "hlg" ||
        e.matrix === "bt2020-ncl"
      );
    }
    async canBeTransparent() {
      return this._backing.canBeTransparent();
    }
    async getDecoderConfig() {
      return this._backing.getDecoderConfig();
    }
    async getCodecParameterString() {
      const e = await this._backing.getMetadataCodecParameterString?.();
      return e ?? (await this._backing.getDecoderConfig())?.codec ?? null;
    }
    async canDecode() {
      try {
        const e = await this._backing.getDecoderConfig();
        if (!e) return !1;
        const t = await this._backing.getCodec();
        return (
          g(t !== null),
          kc.some((n) => n.supports(t, e))
            ? !0
            : typeof VideoDecoder > "u"
              ? !1
              : (await VideoDecoder.isConfigSupported(e)).supported === !0
        );
      } catch (e) {
        return (L._error("Error during decodability check:", e), !1);
      }
    }
    async determinePacketType(e) {
      if (!(e instanceof Z))
        throw new TypeError("packet must be an EncodedPacket.");
      if (e.isMetadataOnly)
        throw new TypeError(
          "packet must not be metadata-only to determine its type.",
        );
      const t = await this.getCodec();
      if (t === null) return null;
      const r = await this.getDecoderConfig();
      return (g(r), vr(t, r, e.data));
    }
    async computeFrameRateMetrics(e = {}) {
      if (!e || typeof e != "object")
        throw new TypeError("options must be an object.");
      if (
        e.targetPacketCount !== void 0 &&
        (!gt(e.targetPacketCount) || e.targetPacketCount < 0)
      )
        throw new TypeError(
          "options.targetPacketCount must be a non-negative number.",
        );
      const t = await this.getTimeResolution(),
        r = e.targetPacketCount ?? 256,
        n = new zt(this),
        s = [];
      let a = -1 / 0,
        o = 0;
      for await (const _ of n.packets(void 0, void 0, { metadataOnly: !0 })) {
        if (s.length >= r && _.timestamp >= a) break;
        (s.push(_.timestamp), (a = Math.max(a, _.timestamp)), o++);
      }
      const c = new Float64Array(s.length);
      for (let _ = 0; _ < s.length; _++) c[_] = Math.round(s[_] * t);
      c.sort();
      let l = 1;
      for (let _ = 1; _ < c.length; _++) c[_] !== c[l - 1] && (c[l++] = c[_]);
      if (l < 2)
        return {
          underlyingFrameRate: null,
          bestGuessFrameRate: t,
          minFrameRate: t,
          maxFrameRate: t,
          averageFrameRate: t,
          medianFrameRate: t,
          frameRateIsConstant: !0,
          probedPacketCount: o,
        };
      const u = c.subarray(0, l),
        d = cf(u, t),
        f = d ?? t,
        h = d !== null ? t / d : null,
        p = new Map();
      let m = 1 / 0,
        y = -1 / 0,
        w = 0;
      for (let _ = 1; _ < l; _++) {
        const v = u[_] - u[_ - 1],
          F = h !== null ? Math.max(1, Math.round(v / h)) : v;
        (p.set(F, (p.get(F) ?? 0) + 1),
          (m = Math.min(m, F)),
          (y = Math.max(y, F)),
          (w += F));
      }
      const b = l - 1,
        T = [...p.keys()].sort((_, v) => _ - v),
        S = (b - 1) >> 1,
        k = b >> 1;
      let P = 0,
        I = 0,
        E = 0;
      for (const _ of T)
        if (((E += p.get(_)), P === 0 && E > S && (P = _), E > k)) {
          I = _;
          break;
        }
      const C = (f / P + f / I) / 2;
      return {
        underlyingFrameRate: d,
        bestGuessFrameRate: d !== null ? d : lf(C),
        minFrameRate: f / y,
        maxFrameRate: f / m,
        averageFrameRate: (f * b) / w,
        medianFrameRate: C,
        frameRateIsConstant: d !== null && m === 1 && y === 1,
        probedPacketCount: o,
      };
    }
  }
  class bs extends fr {
    constructor(e, t) {
      super(e, t);
    }
    get type() {
      return "audio";
    }
    async getCodec() {
      return this._backing.getCodec();
    }
    get codec() {
      return ae(this._backing.getCodec(), "codec", "getCodec");
    }
    async hasOnlyKeyPackets() {
      return (await this._backing.getHasOnlyKeyPackets?.()) ?? !0;
    }
    async getNumberOfChannels() {
      return this._backing.getNumberOfChannels();
    }
    get numberOfChannels() {
      return ae(
        this._backing.getNumberOfChannels(),
        "numberOfChannels",
        "getNumberOfChannels",
      );
    }
    async getSampleRate() {
      return this._backing.getSampleRate();
    }
    get sampleRate() {
      return ae(this._backing.getSampleRate(), "sampleRate", "getSampleRate");
    }
    async getDecoderConfig() {
      return this._backing.getDecoderConfig();
    }
    async getCodecParameterString() {
      const e = await this._backing.getMetadataCodecParameterString?.();
      return e ?? (await this._backing.getDecoderConfig())?.codec ?? null;
    }
    async canDecode() {
      try {
        const e = await this._backing.getDecoderConfig();
        if (!e) return !1;
        const t = await this._backing.getCodec();
        return (
          g(t !== null),
          Tc.some((r) => r.supports(t, e)) || pe.includes(e.codec)
            ? !0
            : typeof AudioDecoder > "u"
              ? !1
              : (await AudioDecoder.isConfigSupported(e)).supported === !0
        );
      } catch (e) {
        return (L._error("Error during decodability check:", e), !1);
      }
    }
    async determinePacketType(e) {
      if (!(e instanceof Z))
        throw new TypeError("packet must be an EncodedPacket.");
      return (await this.getCodec()) === null ? null : "key";
    }
  }
  const Ec = (i) => -(i ?? -1 / 0),
    hr = (i) => -i,
    mr = (i) => {
      if (typeof i != "object" || !i)
        throw new TypeError("query must be an object.");
      if (i.filter !== void 0 && typeof i.filter != "function")
        throw new TypeError("query.filter, when provided, must be a function.");
      if (i.sortBy !== void 0 && typeof i.sortBy != "function")
        throw new TypeError("query.sortBy, when provided, must be a function.");
      return {
        filter: i.filter
          ? (e) => {
              const t = (n) => {
                  if (typeof n != "boolean")
                    throw new TypeError(
                      "query.filter must return or resolve to a boolean.",
                    );
                  return n;
                },
                r = i.filter(e);
              return A(r) ? r.then(t) : t(r);
            }
          : void 0,
        sortBy: i.sortBy
          ? (e) => {
              const t = (n) => {
                  if (
                    typeof n != "number" &&
                    (!Array.isArray(n) || !n.every((s) => typeof s == "number"))
                  )
                    throw new TypeError(
                      "query.sortBy must return or resolve to a number or an array of numbers.",
                    );
                  return n;
                },
                r = i.sortBy(e);
              return A(r) ? r.then(t) : t(r);
            }
          : void 0,
      };
    },
    ii = (i, e) => ({
      filter:
        i?.filter || e?.filter
          ? (t) => {
              const r = i?.filter?.(t) ?? !0,
                n = (s) => (s === !1 ? !1 : (e?.filter?.(t) ?? !0));
              return A(r) ? r.then(n) : n(r);
            }
          : void 0,
      sortBy:
        i?.sortBy || e?.sortBy
          ? (t) => {
              const r = i?.sortBy?.(t) ?? [],
                n = e?.sortBy?.(t) ?? [],
                s = (a, o) => [
                  ...(Array.isArray(a) ? a : [a]),
                  ...(Array.isArray(o) ? o : [o]),
                ];
              return A(r) || A(n)
                ? Promise.all([r, n]).then(([a, o]) => s(a, o))
                : s(r, n);
            }
          : void 0,
    }),
    ks = async (i, e) => {
      let t = i;
      if (e?.filter) {
        const a = i.map((c) => e.filter(c));
        if (a.some((c) => A(c))) {
          const c = await Promise.all(a);
          t = i.filter((l, u) => c[u]);
        } else t = i.filter((c, l) => a[l]);
      }
      if (!e?.sortBy) return t;
      const r = t.map((a) => e.sortBy(a)),
        s = r.some((a) => A(a)) ? await Promise.all(r) : r;
      return t
        .map((a, o) => ({ track: a, sortValue: s[o] }))
        .sort((a, o) => {
          const c = Array.isArray(a.sortValue) ? a.sortValue : [a.sortValue],
            l = Array.isArray(o.sortValue) ? o.sortValue : [o.sortValue],
            u = Math.max(c.length, l.length);
          for (let d = 0; d < u; d++) {
            const f = c[d] ?? 0,
              h = l[d] ?? 0;
            if (f !== h) return f - h;
          }
          return 0;
        })
        .map((a) => a.track);
    },
    cf = (i, e) => {
      const n = 1.000000001,
        s = 1e3,
        a = [
          12,
          15,
          20,
          24e3 / 1001,
          24,
          25,
          3e4 / 1001,
          30,
          48,
          50,
          6e4 / 1001,
          60,
          100,
          12e4 / 1001,
          120,
          144,
          240,
        ];
      if (i.length < 2) return null;
      const o = new Float64Array(i.length - 1);
      for (let I = 1; I < i.length; I++) {
        const E = i[I] - i[I - 1];
        if (!(E > 0)) return null;
        o[I - 1] = E;
      }
      const c = o.slice();
      c.sort();
      let l = c[Math.floor(c.length * 0.05)];
      for (let I = 0; I < 6; I++) {
        let E = 0,
          C = 0;
        for (const v of o) {
          const F = Math.max(1, Math.round(v / l));
          Math.abs(v - F * l) >= n || ((E += v), (C += F));
        }
        if (C === 0) return null;
        const _ = E / C;
        if (Math.abs(_ - l) <= 1e-12 * Math.max(1, l)) {
          l = _;
          break;
        }
        l = _;
      }
      let u = 0,
        d = 0,
        f = 0;
      for (const I of o) {
        const E = Math.max(1, Math.round(I / l));
        Math.abs(I - E * l) >= n || (u++, (d += I), (f += E));
      }
      if (u / o.length < 0.98) return null;
      l = d / f;
      const h = 1 / Math.min(f, s),
        p = Math.max(Number.EPSILON, l - h),
        m = l + h,
        y = e / m,
        w = e / p,
        b = e / l;
      let T = null,
        S = 1 / 0;
      for (const I of a) {
        if (I < y || I > w) continue;
        const E = Math.abs(I / b - 1);
        E < S && ((T = I), (S = E));
      }
      if (T === null) {
        const I = _c(p, m, 1e6),
          E = _c(y, w, 1e6);
        if (E && (!I || E.den < I.den || (E.den === I.den && E.num <= I.num)))
          T = E.num / E.den;
        else if (I) T = (e * I.den) / I.num;
        else return null;
      }
      const k = e / T;
      let P = 0;
      for (const I of o) {
        const E = Math.max(1, Math.round(I / k));
        Math.abs(I - E * k) < n && P++;
      }
      return P / o.length < 0.98 ? null : T;
    },
    _c = (i, e, t) => {
      for (let r = 1; r <= t; r++) {
        const n = Math.floor(i * r) + 1;
        if (n / r < e) return di({ num: n, den: r });
      }
      return null;
    },
    lf = (i) => {
      const e = [
          23.976023976023978, 29.970029970029973, 59.940059940059946,
          119.88011988011989,
        ],
        t = [12, 15, 20, 24, 25, 30, 48, 50, 60, 100, 120, 144, 240],
        r = 5e-4,
        n = 0.025;
      for (const o of e) if (Math.abs(o / i - 1) <= r) return o;
      let s = i,
        a = 1 / 0;
      for (const o of t) {
        const c = Math.abs(o / i - 1);
        c <= n && c < a && ((s = o), (a = c));
      }
      return s;
    };
  Sn();
  const uf = 1,
    df = 2;
  class ri extends Li {
    get disposed() {
      return this._disposed;
    }
    constructor(e) {
      if (
        (super(),
        (this._demuxerPromise = null),
        (this._format = null),
        (this._trackBackingsCache = null),
        (this._backingToTrack = new Map()),
        (this._disposed = !1),
        (this._nextSourceCacheAge = 0),
        (this._sourceRefs = []),
        (this._sourceCache = []),
        (this._sourceCachePromises = []),
        (this._onFormatDetermined = null),
        !e || typeof e != "object")
      )
        throw new TypeError("options must be an object.");
      if (
        !Array.isArray(e.formats) ||
        e.formats.some((t) => !(t instanceof st))
      )
        throw new TypeError("options.formats must be an array of InputFormat.");
      if (!(e.source instanceof nt || e.source instanceof os))
        throw new TypeError("options.source must be a Source or SourceRef.");
      if (e.source instanceof nt && e.source._disposed)
        throw new TypeError("options.source must not be a disposed Source.");
      if (e.initInput !== void 0 && !(e.initInput instanceof ri))
        throw new TypeError(
          "options.initInput, when provided, must be an Input.",
        );
      (e.formatOptions !== void 0 && Bd(e.formatOptions, "formatOptions"),
        (this._formats = e.formats),
        (this._initInput = e.initInput ?? null),
        (this._formatOptions = e.formatOptions ?? {}),
        e.source instanceof nt
          ? (this._rootRef = e.source.ref())
          : (this._rootRef = e.source),
        this._sourceRefs.push(this._rootRef));
    }
    get _rootSource() {
      return this._rootRef.source;
    }
    async _getSourceUncached(e) {
      g(this._rootSource instanceof ar);
      const t = await this._rootSource._resolveRequest(e);
      return (
        this._emit("source", {
          source: t.source,
          request: e,
          isRoot: e.isRoot,
        }),
        t
      );
    }
    _getSourceCached(e, t = uf) {
      const r = this._sourceCache.find(
        (a) => a.cacheGroup === t && Ho(a.request, e),
      );
      if (r) return (r.age++, Promise.resolve(r.sourceRef.source.ref()));
      const n = this._sourceCachePromises.find(
        (a) => a.cacheGroup === t && Ho(a.request, e),
      );
      if (n) return n.promise.then((a) => a.sourceRef.source.ref());
      const s = (async () => {
        const a = await this._getSourceUncached(e);
        if (
          Vi(
            this._sourceCache,
            (d) => d.cacheGroup === t && d.sourceRef.source._refCount === 1,
          ) >= 4
        ) {
          const d = xn(this._sourceCache, (h) =>
            h.cacheGroup === t && h.sourceRef.source._refCount === 1
              ? h.age
              : 1 / 0,
          );
          g(d !== -1);
          const f = this._sourceCache[d];
          (this._sourceCache.splice(d, 1),
            f.sourceRef.free(),
            fn(this._sourceRefs, f.sourceRef));
        }
        this._sourceRefs.push(a);
        const l = this._sourceCachePromises.findIndex((d) => d.request === e);
        return (
          g(l !== -1),
          this._sourceCachePromises.splice(l, 1),
          {
            request: e,
            sourceRef: a,
            age: this._nextSourceCacheAge++,
            cacheGroup: t,
          }
        );
      })();
      return (
        this._sourceCachePromises.push({
          request: e,
          cacheGroup: t,
          promise: s,
        }),
        s.then((a) => {
          const o = a.sourceRef.source.ref();
          return (this._sourceCache.push(a), o);
        })
      );
    }
    _getDemuxer() {
      return (this._demuxerPromise ??= (async () => {
        ((this._reader = new $r(this._rootSource)),
          this._emit("source", {
            source: this._rootSource,
            request: null,
            isRoot: !0,
          }));
        for (const e of this._formats)
          if (await e._canReadInput(this))
            return (
              (this._format = e),
              this._onFormatDetermined?.(e),
              e._createDemuxer(this)
            );
        throw new vc();
      })());
    }
    get source() {
      return this._rootSource;
    }
    async getFormat() {
      return (await this._getDemuxer(), g(this._format), this._format);
    }
    async canRead() {
      try {
        return (await this._getDemuxer(), !0);
      } catch (e) {
        if (e instanceof vc) return !1;
        throw e;
      }
    }
    async getFirstTimestamp(e) {
      e ??= await this.getTracks();
      const t = e.filter((s) => s !== null);
      if (t.length === 0) return 0;
      const r = await Promise.all(
          t.map((s) => s._backing.getFirstPacket({ metadataOnly: !0 })),
        ),
        n = Math.min(...r.map((s) => s?.timestamp ?? 1 / 0));
      return n === 1 / 0 ? 0 : n;
    }
    async computeDuration(e, t) {
      e ??= await this.getTracks();
      const r = e.filter((s) => s !== null);
      if (r.length === 0) return 0;
      const n = await Promise.all(r.map((s) => s.computeDuration(t)));
      return Math.max(...n);
    }
    async getDurationFromMetadata(e, t) {
      e ??= await this.getTracks();
      const r = e.filter((a) => a !== null),
        s = (
          await Promise.all(r.map((a) => a.getDurationFromMetadata(t)))
        ).filter((a) => a !== null);
      return s.length === 0 ? null : Math.max(...s);
    }
    async getTracks(e) {
      e &&= mr(e);
      const r = (await this._getTrackBackings()).map((n) =>
        this._wrapBackingAsTrack(n),
      );
      return ks(r, e);
    }
    async getVideoTracks(e) {
      e &&= mr(e);
      const r = (await this.getTracks()).filter((n) => n.isVideoTrack());
      return ks(r, e);
    }
    async getAudioTracks(e) {
      e &&= mr(e);
      const r = (await this.getTracks()).filter((n) => n.isAudioTrack());
      return ks(r, e);
    }
    async getPrimaryVideoTrack(e) {
      e &&= mr(e);
      const t = ii(e, {
        sortBy: async (n) => [
          hr((await n.getDisposition()).default),
          hr(await n.hasPairableAudioTrack()),
          hr(!(await n.hasOnlyKeyPackets())),
          Ec(await n.getBitrate()),
        ],
      });
      return (await this.getVideoTracks(t))[0] ?? null;
    }
    async getPrimaryAudioTrack(e) {
      e &&= mr(e);
      const t = await this.getPrimaryVideoTrack(),
        r = ii(e, {
          sortBy: async (s) => [
            hr(!t || s.canBePairedWith(t)),
            hr((await s.getDisposition()).default),
            Ec(await s.getBitrate()),
          ],
        });
      return (await this.getAudioTracks(r))[0] ?? null;
    }
    async _getTrackBackings() {
      const e = await this._getDemuxer();
      return (this._trackBackingsCache ??= await e.getTrackBackings());
    }
    _wrapBackingAsTrack(e) {
      const t = this._backingToTrack.get(e);
      if (t) return t;
      const n = e.getType() === "video" ? new ys(this, e) : new bs(this, e);
      return (this._backingToTrack.set(e, n), n);
    }
    async getMimeType() {
      return (await this._getDemuxer()).getMimeType();
    }
    async getMetadataTags() {
      return (await this._getDemuxer()).getMetadataTags();
    }
    dispose() {
      if (!this._disposed) {
        this._disposed = !0;
        for (const e of this._sourceRefs) e.free();
        ((this._sourceRefs.length = 0),
          this._demuxerPromise &&
            this._demuxerPromise.then((e) => e.dispose()).catch(() => {}));
      }
    }
    [Symbol.dispose]() {
      this.dispose();
    }
  }
  class vc extends Error {
    constructor(e = "Input has an unsupported or unrecognizable format.") {
      (super(e), (this.name = "UnsupportedInputFormatError"));
    }
  }
  class ve extends Error {
    constructor(e = "Input has been disposed.") {
      (super(e), (this.name = "InputDisposedError"));
    }
  }
  class $r {
    constructor(e) {
      this.source = e;
    }
    get fileSize() {
      const e = this.source._getFileSize();
      if (e === void 0)
        throw new Error("Reading file size too early; read required first.");
      return e;
    }
    get fileSizeNonStrict() {
      return this.source._getFileSize() ?? null;
    }
    requestSlice(e, t) {
      if (this.source._disposed) throw new ve();
      if (
        e < 0 ||
        (this.fileSizeNonStrict !== null && e + t > this.fileSizeNonStrict)
      )
        return null;
      if (t === 0) {
        const s = new Uint8Array(0);
        return new Me(s, j(s), 0, e, e);
      }
      const r = e + t,
        n = this.source._read(e, r, Vo, Lo);
      return A(n)
        ? n.then((s) => (s ? new Me(s.bytes, s.view, s.offset, e, r) : null))
        : n
          ? new Me(n.bytes, n.view, n.offset, e, r)
          : null;
    }
    requestSliceRange(e, t, r) {
      if (this.source._disposed) throw new ve();
      if (e < 0) return null;
      if (this.fileSizeNonStrict !== null)
        return this.requestSlice(e, se(this.fileSizeNonStrict - e, t, r));
      {
        const n = this.requestSlice(e, r),
          s = (a) =>
            a ||
            (g(this.fileSizeNonStrict !== null),
            this.requestSlice(e, se(this.fileSizeNonStrict - e, t, r)));
        return A(n) ? n.then(s) : s(n);
      }
    }
    requestEntireFile() {
      if (this.fileSizeNonStrict !== null)
        return this.requestSlice(0, this.fileSizeNonStrict);
      const e = 1024;
      return (async () => {
        const t = [];
        let r = 0;
        for (;;) {
          if (t.length === 1 && this.fileSizeNonStrict !== null)
            return this.requestSlice(0, this.fileSizeNonStrict);
          let a = this.requestSliceRange(r, 0, e);
          if ((A(a) && (a = await a), !a || a.length === 0)) break;
          const o = N(a, a.length);
          (t.push(o), (r += a.length));
        }
        const n = new Uint8Array(r);
        let s = 0;
        for (const a of t) (n.set(a, s), (s += a.length));
        return new Me(n, j(n), 0, 0, r);
      })();
    }
  }
  class Me {
    constructor(e, t, r, n, s) {
      ((this.bytes = e),
        (this.view = t),
        (this.offset = r),
        (this.start = n),
        (this.end = s),
        (this.bufferPos = n - r));
    }
    static tempFromBytes(e) {
      return new Me(e, j(e), 0, 0, e.length);
    }
    get length() {
      return this.end - this.start;
    }
    get filePos() {
      return this.offset + this.bufferPos;
    }
    set filePos(e) {
      this.bufferPos = e - this.offset;
    }
    get remainingLength() {
      return Math.max(this.end - this.filePos, 0);
    }
    skip(e) {
      this.bufferPos += e;
    }
    slice(e, t = this.end - e) {
      if (e < this.start || e + t > this.end)
        throw new RangeError("Slicing outside of original slice.");
      return new Me(this.bytes, this.view, this.offset, e, e + t);
    }
  }
  const ze = (i, e) => {
      if (i.filePos < i.start || i.filePos + e > i.end)
        throw new RangeError(
          `Tried reading [${i.filePos}, ${i.filePos + e}), but slice is [${i.start}, ${i.end}). This is likely an internal error, please report it alongside the file that caused it.`,
        );
    },
    N = (i, e) => {
      ze(i, e);
      const t = i.bytes.subarray(i.bufferPos, i.bufferPos + e);
      return ((i.bufferPos += e), t);
    },
    U = (i) => (ze(i, 1), i.view.getUint8(i.bufferPos++)),
    pr = (i, e) => {
      ze(i, 2);
      const t = i.view.getUint16(i.bufferPos, e);
      return ((i.bufferPos += 2), t);
    },
    fe = (i) => {
      ze(i, 2);
      const e = i.view.getUint16(i.bufferPos, !1);
      return ((i.bufferPos += 2), e);
    },
    It = (i) => {
      ze(i, 3);
      const e = Tr(i.view, i.bufferPos, !1);
      return ((i.bufferPos += 3), e);
    },
    Ts = (i) => {
      ze(i, 2);
      const e = i.view.getInt16(i.bufferPos, !1);
      return ((i.bufferPos += 2), e);
    },
    Nt = (i, e) => {
      ze(i, 4);
      const t = i.view.getUint32(i.bufferPos, e);
      return ((i.bufferPos += 4), t);
    },
    R = (i) => {
      ze(i, 4);
      const e = i.view.getUint32(i.bufferPos, !1);
      return ((i.bufferPos += 4), e);
    },
    _i = (i) => {
      ze(i, 4);
      const e = i.view.getUint32(i.bufferPos, !0);
      return ((i.bufferPos += 4), e);
    },
    ni = (i) => {
      ze(i, 4);
      const e = i.view.getInt32(i.bufferPos, !1);
      return ((i.bufferPos += 4), e);
    },
    ff = (i) => {
      ze(i, 4);
      const e = i.view.getInt32(i.bufferPos, !0);
      return ((i.bufferPos += 4), e);
    },
    Ac = (i, e) => {
      let t, r;
      return (
        e
          ? ((t = Nt(i, !0)), (r = Nt(i, !0)))
          : ((r = Nt(i, !1)), (t = Nt(i, !1))),
        r * 4294967296 + t
      );
    },
    Le = (i) => {
      const e = R(i),
        t = R(i);
      return e * 4294967296 + t;
    },
    hf = (i) => {
      const e = ni(i),
        t = R(i);
      return e * 4294967296 + t;
    },
    mf = (i) => {
      const e = _i(i);
      return ff(i) * 4294967296 + e;
    },
    pf = (i) => {
      ze(i, 4);
      const e = i.view.getFloat32(i.bufferPos, !1);
      return ((i.bufferPos += 4), e);
    },
    Bc = (i) => {
      ze(i, 8);
      const e = i.view.getFloat64(i.bufferPos, !1);
      return ((i.bufferPos += 8), e);
    },
    le = (i, e) => {
      ze(i, e);
      let t = "";
      for (let r = 0; r < e; r++)
        t += String.fromCharCode(i.bytes[i.bufferPos++]);
      return t;
    },
    Fc = (i, e, t) =>
      Oe.decode(N(i, e))
        .split(
          `
`,
        )
        .map((s) => s.trim())
        .filter((s) => s.length > 0 && !t?.ignore?.(s));
  var si;
  (function (i) {
    ((i[(i.Unsynchronisation = 128)] = "Unsynchronisation"),
      (i[(i.ExtendedHeader = 64)] = "ExtendedHeader"),
      (i[(i.ExperimentalIndicator = 32)] = "ExperimentalIndicator"),
      (i[(i.Footer = 16)] = "Footer"));
  })(si || (si = {}));
  var Te;
  (function (i) {
    ((i[(i.ISO_8859_1 = 0)] = "ISO_8859_1"),
      (i[(i.UTF_16_WITH_BOM = 1)] = "UTF_16_WITH_BOM"),
      (i[(i.UTF_16_BE_NO_BOM = 2)] = "UTF_16_BE_NO_BOM"),
      (i[(i.UTF_8 = 3)] = "UTF_8"));
  })(Te || (Te = {}));
  const Kr = 128,
    je = 10,
    vi = [
      "Blues",
      "Classic rock",
      "Country",
      "Dance",
      "Disco",
      "Funk",
      "Grunge",
      "Hip-hop",
      "Jazz",
      "Metal",
      "New age",
      "Oldies",
      "Other",
      "Pop",
      "Rhythm and blues",
      "Rap",
      "Reggae",
      "Rock",
      "Techno",
      "Industrial",
      "Alternative",
      "Ska",
      "Death metal",
      "Pranks",
      "Soundtrack",
      "Euro-techno",
      "Ambient",
      "Trip-hop",
      "Vocal",
      "Jazz & funk",
      "Fusion",
      "Trance",
      "Classical",
      "Instrumental",
      "Acid",
      "House",
      "Game",
      "Sound clip",
      "Gospel",
      "Noise",
      "Alternative rock",
      "Bass",
      "Soul",
      "Punk",
      "Space",
      "Meditative",
      "Instrumental pop",
      "Instrumental rock",
      "Ethnic",
      "Gothic",
      "Darkwave",
      "Techno-industrial",
      "Electronic",
      "Pop-folk",
      "Eurodance",
      "Dream",
      "Southern rock",
      "Comedy",
      "Cult",
      "Gangsta",
      "Top 40",
      "Christian rap",
      "Pop/funk",
      "Jungle music",
      "Native US",
      "Cabaret",
      "New wave",
      "Psychedelic",
      "Rave",
      "Showtunes",
      "Trailer",
      "Lo-fi",
      "Tribal",
      "Acid punk",
      "Acid jazz",
      "Polka",
      "Retro",
      "Musical",
      "Rock 'n' roll",
      "Hard rock",
      "Folk",
      "Folk rock",
      "National folk",
      "Swing",
      "Fast fusion",
      "Bebop",
      "Latin",
      "Revival",
      "Celtic",
      "Bluegrass",
      "Avantgarde",
      "Gothic rock",
      "Progressive rock",
      "Psychedelic rock",
      "Symphonic rock",
      "Slow rock",
      "Big band",
      "Chorus",
      "Easy listening",
      "Acoustic",
      "Humour",
      "Speech",
      "Chanson",
      "Opera",
      "Chamber music",
      "Sonata",
      "Symphony",
      "Booty bass",
      "Primus",
      "Porn groove",
      "Satire",
      "Slow jam",
      "Club",
      "Tango",
      "Samba",
      "Folklore",
      "Ballad",
      "Power ballad",
      "Rhythmic Soul",
      "Freestyle",
      "Duet",
      "Punk rock",
      "Drum solo",
      "A cappella",
      "Euro-house",
      "Dance hall",
      "Goa music",
      "Drum & bass",
      "Club-house",
      "Hardcore techno",
      "Terror",
      "Indie",
      "Britpop",
      "Negerpunk",
      "Polsk punk",
      "Beat",
      "Christian gangsta rap",
      "Heavy metal",
      "Black metal",
      "Crossover",
      "Contemporary Christian",
      "Christian rock",
      "Merengue",
      "Salsa",
      "Thrash metal",
      "Anime",
      "Jpop",
      "Synthpop",
      "Christmas",
      "Art rock",
      "Baroque",
      "Bhangra",
      "Big beat",
      "Breakbeat",
      "Chillout",
      "Downtempo",
      "Dub",
      "EBM",
      "Eclectic",
      "Electro",
      "Electroclash",
      "Emo",
      "Experimental",
      "Garage",
      "Global",
      "IDM",
      "Illbient",
      "Industro-Goth",
      "Jam Band",
      "Krautrock",
      "Leftfield",
      "Lounge",
      "Math rock",
      "New romantic",
      "Nu-breakz",
      "Post-punk",
      "Post-rock",
      "Psytrance",
      "Shoegaze",
      "Space rock",
      "Trop rock",
      "World music",
      "Neoclassical",
      "Audiobook",
      "Audio theatre",
      "Neue Deutsche Welle",
      "Podcast",
      "Indie rock",
      "G-Funk",
      "Dubstep",
      "Garage rock",
      "Psybient",
    ],
    gf = (i, e) => {
      const t = i.filePos;
      ((e.raw ??= {}), (e.raw.TAG ??= N(i, Kr - 3)), (i.filePos = t));
      const r = Ai(i, 30);
      r && (e.title ??= r);
      const n = Ai(i, 30);
      n && (e.artist ??= n);
      const s = Ai(i, 30);
      s && (e.album ??= s);
      const a = Ai(i, 4),
        o = Number.parseInt(a, 10);
      Number.isInteger(o) && o > 0 && (e.date ??= new Date(String(o)));
      const c = N(i, 30);
      let l;
      if (c[28] === 0 && c[29] !== 0) {
        const d = c[29];
        (d > 0 && (e.trackNumber ??= d),
          i.skip(-30),
          (l = Ai(i, 28)),
          i.skip(2));
      } else (i.skip(-30), (l = Ai(i, 30)));
      l && (e.comment ??= l);
      const u = U(i);
      u < vi.length && (e.genre ??= vi[u]);
    },
    Ai = (i, e) => {
      const t = N(i, e),
        r = ui(t.indexOf(0), t.length),
        n = t.subarray(0, r);
      let s = "";
      for (let a = 0; a < n.length; a++) s += String.fromCharCode(n[a]);
      return s.trimEnd();
    },
    Et = (i) => {
      const e = i.filePos,
        t = le(i, 3),
        r = U(i),
        n = U(i),
        s = U(i),
        a = R(i);
      if (t !== "ID3" || r === 255 || n === 255 || (a & 2155905152) !== 0)
        return ((i.filePos = e), null);
      let o = jn(a);
      return (
        s & si.Footer && (o += je),
        { majorVersion: r, revision: n, flags: s, size: o }
      );
    },
    Gr = (i, e, t) => {
      if (![2, 3, 4].includes(e.majorVersion)) {
        L._warn(`Unsupported ID3v2 major version: ${e.majorVersion}`);
        return;
      }
      const r = e.flags & si.Footer ? e.size - je : e.size,
        n = N(i, r),
        s = new wf(e, n);
      if (
        (e.flags & si.Unsynchronisation &&
          e.majorVersion === 3 &&
          s.ununsynchronizeAll(),
        e.flags & si.ExtendedHeader)
      ) {
        const a = s.readU32();
        e.majorVersion === 3 ? (s.pos += a) : (s.pos += a - 4);
      }
      for (; s.pos <= s.bytes.length - s.frameHeaderSize();) {
        const a = s.readId3V2Frame();
        if (!a) break;
        const o = s.pos,
          c = s.pos + a.size;
        let l = !1,
          u = !1,
          d = !1;
        if (
          (e.majorVersion === 3
            ? ((l = !!(a.flags & 64)), (u = !!(a.flags & 128)))
            : e.majorVersion === 4 &&
              ((l = !!(a.flags & 4)),
              (u = !!(a.flags & 8)),
              (d = !!(a.flags & 2) || !!(e.flags & si.Unsynchronisation))),
          l)
        ) {
          (L._warn(`Skipping encrypted ID3v2 frame ${a.id}`), (s.pos = c));
          continue;
        }
        if (u) {
          (L._warn(`Skipping compressed ID3v2 frame ${a.id}`), (s.pos = c));
          continue;
        }
        if (
          (d && s.ununsynchronizeRegion(s.pos, c),
          (t.raw ??= {}),
          a.id === "TXXX")
        ) {
          const f = (t.raw.TXXX ??= {}),
            h = s.readId3V2TextEncoding(),
            p = s.readId3V2Text(h, c),
            m = s.readId3V2Text(h, c);
          f[p] ??= m;
        } else
          a.id[0] === "T"
            ? (t.raw[a.id] ??= s.readId3V2EncodingAndText(c))
            : (t.raw[a.id] ??= s.readBytes(a.size));
        switch (((s.pos = o), a.id)) {
          case "TIT2":
          case "TT2":
            t.title ??= s.readId3V2EncodingAndText(c);
            break;
          case "TIT3":
          case "TT3":
            t.description ??= s.readId3V2EncodingAndText(c);
            break;
          case "TPE1":
          case "TP1":
            t.artist ??= s.readId3V2EncodingAndText(c);
            break;
          case "TALB":
          case "TAL":
            t.album ??= s.readId3V2EncodingAndText(c);
            break;
          case "TPE2":
          case "TP2":
            t.albumArtist ??= s.readId3V2EncodingAndText(c);
            break;
          case "TRCK":
          case "TRK":
            {
              const h = s.readId3V2EncodingAndText(c).split("/"),
                p = Number.parseInt(h[0], 10),
                m = h[1] && Number.parseInt(h[1], 10);
              (Number.isInteger(p) && p > 0 && (t.trackNumber ??= p),
                m && Number.isInteger(m) && m > 0 && (t.tracksTotal ??= m));
            }
            break;
          case "TPOS":
          case "TPA":
            {
              const h = s.readId3V2EncodingAndText(c).split("/"),
                p = Number.parseInt(h[0], 10),
                m = h[1] && Number.parseInt(h[1], 10);
              (Number.isInteger(p) && p > 0 && (t.discNumber ??= p),
                m && Number.isInteger(m) && m > 0 && (t.discsTotal ??= m));
            }
            break;
          case "TCON":
          case "TCO":
            {
              const f = s.readId3V2EncodingAndText(c);
              let h = /^\((\d+)\)/.exec(f);
              if (h) {
                const p = Number.parseInt(h[1]);
                if (vi[p] !== void 0) {
                  t.genre ??= vi[p];
                  break;
                }
              }
              if (((h = /^\d+$/.exec(f)), h)) {
                const p = Number.parseInt(h[0]);
                if (vi[p] !== void 0) {
                  t.genre ??= vi[p];
                  break;
                }
              }
              t.genre ??= f;
            }
            break;
          case "TDRC":
          case "TDAT":
            {
              const f = s.readId3V2EncodingAndText(c),
                h = new Date(f);
              Number.isNaN(h.getTime()) || (t.date ??= h);
            }
            break;
          case "TYER":
          case "TYE":
            {
              const f = s.readId3V2EncodingAndText(c),
                h = Number.parseInt(f, 10);
              Number.isInteger(h) && (t.date ??= new Date(String(h)));
            }
            break;
          case "TBPM":
          case "TBP":
            {
              const f = s.readId3V2EncodingAndText(c),
                h = Number.parseInt(f, 10);
              Number.isInteger(h) && (t.beatsPerMinute ??= h);
            }
            break;
          case "USLT":
          case "ULT":
            {
              const f = s.readU8();
              ((s.pos += 3),
                s.readId3V2Text(f, c),
                (t.lyrics ??= s.readId3V2Text(f, c)));
            }
            break;
          case "COMM":
          case "COM":
            {
              const f = s.readU8();
              ((s.pos += 3),
                s.readId3V2Text(f, c),
                (t.comment ??= s.readId3V2Text(f, c)));
            }
            break;
          case "APIC":
          case "PIC":
            {
              const f = s.readId3V2TextEncoding();
              let h;
              if (e.majorVersion === 2) {
                const w = s.readAscii(3);
                h =
                  w === "PNG"
                    ? "image/png"
                    : w === "JPG"
                      ? "image/jpeg"
                      : "image/*";
              } else h = s.readId3V2Text(f, c);
              const p = s.readU8(),
                m = s.readId3V2Text(f, c).trimEnd(),
                y = c - s.pos;
              if (y >= 0) {
                const w = s.readBytes(y);
                (t.images || (t.images = []),
                  t.images.push({
                    data: w,
                    mimeType: h,
                    kind:
                      p === 3
                        ? "coverFront"
                        : p === 4
                          ? "coverBack"
                          : "unknown",
                    description: m,
                  }));
              }
            }
            break;
          default:
            s.pos += a.size;
            break;
        }
        s.pos = c;
      }
    };
  class wf {
    constructor(e, t) {
      ((this.header = e),
        (this.bytes = t),
        (this.pos = 0),
        (this.view = new DataView(t.buffer, t.byteOffset, t.byteLength)));
    }
    frameHeaderSize() {
      return this.header.majorVersion === 2 ? 6 : 10;
    }
    ununsynchronizeAll() {
      const e = [];
      for (let t = 0; t < this.bytes.length; t++) {
        const r = this.bytes[t];
        (e.push(r),
          r === 255 &&
            t !== this.bytes.length - 1 &&
            this.bytes[t] === 0 &&
            t++);
      }
      ((this.bytes = new Uint8Array(e)),
        (this.view = new DataView(this.bytes.buffer)));
    }
    ununsynchronizeRegion(e, t) {
      const r = [];
      for (let a = e; a < t; a++) {
        const o = this.bytes[a];
        (r.push(o), o === 255 && a !== t - 1 && this.bytes[a + 1] === 0 && a++);
      }
      const n = this.bytes.subarray(0, e),
        s = this.bytes.subarray(t);
      ((this.bytes = new Uint8Array(n.length + r.length + s.length)),
        this.bytes.set(n, 0),
        this.bytes.set(r, n.length),
        this.bytes.set(s, n.length + r.length),
        (this.view = new DataView(this.bytes.buffer)));
    }
    readBytes(e) {
      const t = this.bytes.subarray(this.pos, this.pos + e);
      return ((this.pos += e), t);
    }
    readU8() {
      const e = this.view.getUint8(this.pos);
      return ((this.pos += 1), e);
    }
    readU16() {
      const e = this.view.getUint16(this.pos, !1);
      return ((this.pos += 2), e);
    }
    readU24() {
      const e = this.view.getUint16(this.pos, !1),
        t = this.view.getUint8(this.pos + 2);
      return ((this.pos += 3), e * 256 + t);
    }
    readU32() {
      const e = this.view.getUint32(this.pos, !1);
      return ((this.pos += 4), e);
    }
    readAscii(e) {
      let t = "";
      for (let r = 0; r < e; r++)
        t += String.fromCharCode(this.view.getUint8(this.pos + r));
      return ((this.pos += e), t);
    }
    readId3V2Frame() {
      if (this.header.majorVersion === 2) {
        const e = this.readAscii(3);
        if (e === "\0\0\0") return null;
        const t = this.readU24();
        return { id: e, size: t, flags: 0 };
      } else {
        const e = this.readAscii(4);
        if (e === "\0\0\0\0") return null;
        const t = this.readU32();
        let r = this.header.majorVersion === 4 ? jn(t) : t;
        const n = this.readU16(),
          s = this.pos,
          a = (o) => {
            const c = this.pos + o;
            if (c > this.bytes.length) return !1;
            if (c <= this.bytes.length - this.frameHeaderSize()) {
              this.pos += o;
              const l = this.readAscii(4);
              if (l !== "\0\0\0\0" && !/[0-9A-Z]{4}/.test(l)) return !1;
            }
            return !0;
          };
        if (!a(r)) {
          const o = this.header.majorVersion === 4 ? t : jn(t);
          a(o) && (r = o);
        }
        return ((this.pos = s), { id: e, size: r, flags: n });
      }
    }
    readId3V2TextEncoding() {
      const e = this.readU8();
      if (e > 3) throw new Error(`Unsupported text encoding: ${e}`);
      return e;
    }
    readId3V2Text(e, t) {
      const r = this.pos,
        n = this.readBytes(t - this.pos);
      switch (e) {
        case Te.ISO_8859_1: {
          let s = "";
          for (let a = 0; a < n.length; a++) {
            const o = n[a];
            if (o === 0) {
              this.pos = r + a + 1;
              break;
            }
            s += String.fromCharCode(o);
          }
          return s;
        }
        case Te.UTF_16_WITH_BOM:
          if (n[0] === 255 && n[1] === 254) {
            const s = new Mi("utf-16le"),
              a = ui(
                n.findIndex((o, c) => o === 0 && n[c + 1] === 0 && c % 2 === 0),
                n.length,
              );
            return (
              (this.pos = r + Math.min(a + 2, n.length)),
              s.decode(n.subarray(2, a))
            );
          } else if (n[0] === 254 && n[1] === 255) {
            const s = new Mi("utf-16be"),
              a = ui(
                n.findIndex((o, c) => o === 0 && n[c + 1] === 0 && c % 2 === 0),
                n.length,
              );
            return (
              (this.pos = r + Math.min(a + 2, n.length)),
              s.decode(n.subarray(2, a))
            );
          } else {
            const s = ui(
              n.findIndex((a) => a === 0),
              n.length,
            );
            return (
              (this.pos = r + Math.min(s + 1, n.length)),
              Oe.decode(n.subarray(0, s))
            );
          }
        case Te.UTF_16_BE_NO_BOM: {
          const s = new Mi("utf-16be"),
            a = ui(
              n.findIndex((o, c) => o === 0 && n[c + 1] === 0 && c % 2 === 0),
              n.length,
            );
          return (
            (this.pos = r + Math.min(a + 2, n.length)),
            s.decode(n.subarray(0, a))
          );
        }
        case Te.UTF_8: {
          const s = ui(
            n.findIndex((a) => a === 0),
            n.length,
          );
          return (
            (this.pos = r + Math.min(s + 1, n.length)),
            Oe.decode(n.subarray(0, s))
          );
        }
      }
    }
    readId3V2EncodingAndText(e) {
      if (this.pos >= e) return "";
      const t = this.readId3V2TextEncoding();
      return this.readId3V2Text(t, e);
    }
  }
  class yf {
    constructor(e) {
      ((this.helper = new Uint8Array(8)),
        (this.helperView = j(this.helper)),
        (this.writer = e));
    }
    writeId3V2Tag(e) {
      const t = this.writer.getPos();
      (this.writeAscii("ID3"),
        this.writeU8(4),
        this.writeU8(0),
        this.writeU8(0),
        this.writeSynchsafeU32(0));
      const r = this.writer.getPos(),
        n = new Set();
      for (const { key: o, value: c } of Wi(e))
        switch (o) {
          case "title":
            (this.writeId3V2TextFrame("TIT2", c), n.add("TIT2"));
            break;
          case "description":
            (this.writeId3V2TextFrame("TIT3", c), n.add("TIT3"));
            break;
          case "artist":
            (this.writeId3V2TextFrame("TPE1", c), n.add("TPE1"));
            break;
          case "album":
            (this.writeId3V2TextFrame("TALB", c), n.add("TALB"));
            break;
          case "albumArtist":
            (this.writeId3V2TextFrame("TPE2", c), n.add("TPE2"));
            break;
          case "trackNumber":
            {
              const l =
                e.tracksTotal !== void 0
                  ? `${c}/${e.tracksTotal}`
                  : c.toString();
              (this.writeId3V2TextFrame("TRCK", l), n.add("TRCK"));
            }
            break;
          case "discNumber":
            {
              const l =
                e.discsTotal !== void 0 ? `${c}/${e.discsTotal}` : c.toString();
              (this.writeId3V2TextFrame("TPOS", l), n.add("TPOS"));
            }
            break;
          case "genre":
            (this.writeId3V2TextFrame("TCON", c), n.add("TCON"));
            break;
          case "date":
            (this.writeId3V2TextFrame("TDRC", c.toISOString().slice(0, 10)),
              n.add("TDRC"));
            break;
          case "beatsPerMinute":
            (this.writeId3V2TextFrame("TBPM", `${c}`), n.add("TBPM"));
            break;
          case "lyrics":
            (this.writeId3V2LyricsFrame(c), n.add("USLT"));
            break;
          case "comment":
            (this.writeId3V2CommentFrame(c), n.add("COMM"));
            break;
          case "images":
            {
              const l = { coverFront: 3, coverBack: 4, unknown: 0 };
              for (const u of c) {
                const d = l[u.kind] ?? 0,
                  f = u.description ?? "";
                this.writeId3V2ApicFrame(u.mimeType, d, f, u.data);
              }
            }
            break;
          case "tracksTotal":
          case "discsTotal":
            break;
          case "raw":
            break;
          default:
            xe(o);
        }
      if (e.raw)
        for (const o in e.raw) {
          const c = e.raw[o];
          if (c == null || o.length !== 4 || n.has(o)) continue;
          let l;
          if (typeof c == "string")
            if (ft(c)) {
              ((l = new Uint8Array(c.length + 2)), (l[0] = Te.ISO_8859_1));
              for (let d = 0; d < c.length; d++) l[d + 1] = c.charCodeAt(d);
            } else {
              const d = oe.encode(c);
              ((l = new Uint8Array(d.byteLength + 2)),
                (l[0] = Te.UTF_8),
                l.set(d, 1));
            }
          else if (c instanceof Uint8Array) l = c;
          else if (o === "TXXX" && oa(c)) {
            for (const u in c) {
              const d = c[u],
                f = ft(u) && ft(d),
                h = f ? null : oe.encode(u),
                p = f ? null : oe.encode(d),
                m = f ? u.length : h.byteLength,
                y = f ? d.length : p.byteLength,
                w = 1 + m + 1 + y + 1;
              (this.writeAscii("TXXX"),
                this.writeSynchsafeU32(w),
                this.writeU16(0),
                this.writeU8(f ? Te.ISO_8859_1 : Te.UTF_8),
                f
                  ? (this.writeIsoString(u), this.writeIsoString(d))
                  : (this.writer.write(h),
                    this.writeU8(0),
                    this.writer.write(p),
                    this.writeU8(0)));
            }
            continue;
          } else continue;
          (this.writeAscii(o),
            this.writeSynchsafeU32(l.byteLength),
            this.writeU16(0),
            this.writer.write(l));
        }
      const s = this.writer.getPos(),
        a = s - r;
      return (
        this.writer.seek(t + 6),
        this.writeSynchsafeU32(a),
        this.writer.seek(s),
        a + 10
      );
    }
    writeU8(e) {
      ((this.helper[0] = e), this.writer.write(this.helper.subarray(0, 1)));
    }
    writeU16(e) {
      (this.helperView.setUint16(0, e, !1),
        this.writer.write(this.helper.subarray(0, 2)));
    }
    writeU32(e) {
      (this.helperView.setUint32(0, e, !1),
        this.writer.write(this.helper.subarray(0, 4)));
    }
    writeAscii(e) {
      for (let t = 0; t < e.length; t++) this.helper[t] = e.charCodeAt(t);
      this.writer.write(this.helper.subarray(0, e.length));
    }
    writeSynchsafeU32(e) {
      this.writeU32(wu(e));
    }
    writeIsoString(e) {
      const t = new Uint8Array(e.length + 1);
      for (let r = 0; r < e.length; r++) t[r] = e.charCodeAt(r);
      this.writer.write(t);
    }
    writeUtf8String(e) {
      const t = oe.encode(e);
      (this.writer.write(t), this.writeU8(0));
    }
    writeId3V2TextFrame(e, t) {
      const r = ft(t),
        s = 1 + (r ? t.length : oe.encode(t).byteLength) + 1;
      (this.writeAscii(e),
        this.writeSynchsafeU32(s),
        this.writeU16(0),
        this.writeU8(r ? Te.ISO_8859_1 : Te.UTF_8),
        r ? this.writeIsoString(t) : this.writeUtf8String(t));
    }
    writeId3V2LyricsFrame(e) {
      const t = ft(e),
        r = "",
        n = 4 + r.length + 1 + e.length + 1;
      (this.writeAscii("USLT"),
        this.writeSynchsafeU32(n),
        this.writeU16(0),
        this.writeU8(t ? Te.ISO_8859_1 : Te.UTF_8),
        this.writeAscii("und"),
        t
          ? (this.writeIsoString(r), this.writeIsoString(e))
          : (this.writeUtf8String(r), this.writeUtf8String(e)));
    }
    writeId3V2CommentFrame(e) {
      const t = ft(e),
        r = t ? e.length : oe.encode(e).byteLength,
        n = "",
        s = 4 + n.length + 1 + r + 1;
      (this.writeAscii("COMM"),
        this.writeSynchsafeU32(s),
        this.writeU16(0),
        this.writeU8(t ? Te.ISO_8859_1 : Te.UTF_8),
        this.writeU8(117),
        this.writeU8(110),
        this.writeU8(100),
        t
          ? (this.writeIsoString(n), this.writeIsoString(e))
          : (this.writeUtf8String(n), this.writeUtf8String(e)));
    }
    writeId3V2ApicFrame(e, t, r, n) {
      const s = ft(e) && ft(r),
        a = s ? r.length : oe.encode(r).byteLength,
        o = 1 + e.length + 1 + 1 + a + 1 + n.byteLength;
      (this.writeAscii("APIC"),
        this.writeSynchsafeU32(o),
        this.writeU16(0),
        this.writeU8(s ? Te.ISO_8859_1 : Te.UTF_8),
        s ? this.writeIsoString(e) : this.writeUtf8String(e),
        this.writeU8(t),
        s ? this.writeIsoString(r) : this.writeUtf8String(r),
        this.writer.write(n));
    }
  }
  class Ss {
    constructor(e) {
      ((this.mutex = new ci()),
        (this.trackTimestampInfo = new WeakMap()),
        (this.output = e));
    }
    onTrackClose(e) {}
    validateTimestamp(e, t, r) {
      let n = this.trackTimestampInfo.get(e);
      if (n) {
        if (
          (r && (n.maxTimestampBeforeLastKeyPacket = n.maxTimestamp),
          n.maxTimestampBeforeLastKeyPacket !== null &&
            t < n.maxTimestampBeforeLastKeyPacket)
        )
          throw new Error(
            `Timestamps cannot be smaller than the largest timestamp of the previous GOP (a GOP begins with a key packet and ends right before the next key packet). Got ${t}s, but largest timestamp is ${n.maxTimestampBeforeLastKeyPacket}s.`,
          );
        n.maxTimestamp = Math.max(n.maxTimestamp, t);
      } else {
        if (!r) throw new Error("First packet must be a key packet.");
        ((n = { maxTimestamp: t, maxTimestampBeforeLastKeyPacket: null }),
          this.trackTimestampInfo.set(e, n));
      }
    }
  }
  const Xr = /<(?:(\d{2}):)?(\d{2}):(\d{2}).(\d{3})>/g,
    bf = /(?:(\d{2}):)?(\d{2}):(\d{2}).(\d{3})/,
    kf = (i) => {
      const e = bf.exec(i);
      if (!e) throw new Error("Expected match.");
      return (
        3600 * 1e3 * Number(e[1] || "0") +
        60 * 1e3 * Number(e[2]) +
        1e3 * Number(e[3]) +
        Number(e[4])
      );
    },
    Rc = (i) => {
      const e = Math.floor(i / 36e5),
        t = Math.floor((i % (3600 * 1e3)) / (60 * 1e3)),
        r = Math.floor((i % (60 * 1e3)) / 1e3),
        n = i % 1e3;
      return (
        e.toString().padStart(2, "0") +
        ":" +
        t.toString().padStart(2, "0") +
        ":" +
        r.toString().padStart(2, "0") +
        "." +
        n.toString().padStart(3, "0")
      );
    };
  class Qr {
    constructor(e) {
      ((this.writer = e),
        (this.helper = new Uint8Array(8)),
        (this.helperView = new DataView(this.helper.buffer)),
        (this.offsets = new WeakMap()));
    }
    writeU32(e) {
      (this.helperView.setUint32(0, e, !1),
        this.writer.write(this.helper.subarray(0, 4)));
    }
    writeU64(e) {
      (this.helperView.setUint32(0, Math.floor(e / 2 ** 32), !1),
        this.helperView.setUint32(4, e, !1),
        this.writer.write(this.helper.subarray(0, 8)));
    }
    writeAscii(e) {
      for (let t = 0; t < e.length; t++)
        (this.helperView.setUint8(t % 8, e.charCodeAt(t)),
          t % 8 === 7 && this.writer.write(this.helper));
      e.length % 8 !== 0 &&
        this.writer.write(this.helper.subarray(0, e.length % 8));
    }
    writeBox(e) {
      if (
        (this.offsets.set(e, this.writer.getPos()), e.contents && !e.children)
      )
        (this.writeBoxHeader(e, e.size ?? e.contents.byteLength + 8),
          this.writer.write(e.contents));
      else {
        const t = this.writer.getPos();
        if (
          (this.writeBoxHeader(e, 0),
          e.contents && this.writer.write(e.contents),
          e.children)
        )
          for (const s of e.children) s && this.writeBox(s);
        const r = this.writer.getPos(),
          n = e.size ?? r - t;
        (this.writer.seek(t), this.writeBoxHeader(e, n), this.writer.seek(r));
      }
    }
    writeBoxHeader(e, t) {
      (this.writeU32(e.largeSize ? 1 : t),
        this.writeAscii(e.type),
        e.largeSize && this.writeU64(t));
    }
    measureBoxHeader(e) {
      return 8 + (e.largeSize ? 8 : 0);
    }
    patchBox(e) {
      const t = this.offsets.get(e);
      g(t !== void 0);
      const r = this.writer.getPos();
      (this.writer.seek(t), this.writeBox(e), this.writer.seek(r));
    }
    measureBox(e) {
      if (e.contents && !e.children)
        return this.measureBoxHeader(e) + e.contents.byteLength;
      {
        let t = this.measureBoxHeader(e);
        if ((e.contents && (t += e.contents.byteLength), e.children))
          for (const r of e.children) r && (t += this.measureBox(r));
        return t;
      }
    }
  }
  const q = new Uint8Array(8),
    $e = new DataView(q.buffer),
    ue = (i) => [((i % 256) + 256) % 256],
    V = (i) => ($e.setUint16(0, i, !1), [q[0], q[1]]),
    xs = (i) => ($e.setInt16(0, i, !1), [q[0], q[1]]),
    Mc = (i) => ($e.setUint32(0, i, !1), [q[1], q[2], q[3]]),
    B = (i) => ($e.setUint32(0, i, !1), [q[0], q[1], q[2], q[3]]),
    at = (i) => ($e.setInt32(0, i, !1), [q[0], q[1], q[2], q[3]]),
    Qe = (i) => (
      $e.setUint32(0, Math.floor(i / 2 ** 32), !1),
      $e.setUint32(4, i, !1),
      [q[0], q[1], q[2], q[3], q[4], q[5], q[6], q[7]]
    ),
    Dc = (i) => (
      $e.setInt32(0, Math.floor(i / 2 ** 32), !1),
      $e.setUint32(4, i, !1),
      [q[0], q[1], q[2], q[3], q[4], q[5], q[6], q[7]]
    ),
    Oc = (i) => ($e.setInt16(0, 2 ** 8 * i, !1), [q[0], q[1]]),
    Ke = (i) => ($e.setInt32(0, 2 ** 16 * i, !1), [q[0], q[1], q[2], q[3]]),
    Ps = (i) => ($e.setInt32(0, 2 ** 30 * i, !1), [q[0], q[1], q[2], q[3]]),
    Cs = (i, e) => {
      const t = [];
      let r = i;
      do {
        let n = r & 127;
        ((r >>= 7), t.length > 0 && (n |= 128), t.push(n));
      } while (r > 0 || e);
      return t.reverse();
    },
    J = (i, e = !1) => {
      const t = Array(i.length)
        .fill(null)
        .map((r, n) => i.charCodeAt(n));
      return (e && t.push(0), t);
    },
    zc = (i) => [
      Ke(i[0]),
      Ke(i[1]),
      Ps(i[2]),
      Ke(i[3]),
      Ke(i[4]),
      Ps(i[5]),
      Ke(i[6]),
      Ke(i[7]),
      Ps(i[8]),
    ],
    W = (i, e, t) => ({
      type: i,
      contents: e && new Uint8Array(e.flat(10)),
      children: t,
    }),
    Q = (i, e, t, r, n) => W(i, [ue(e), Mc(t), r ?? []], n),
    Tf = (i) =>
      i.isQuickTime
        ? W("ftyp", [J("qt  "), B(512), J("qt  ")])
        : i.fragmented
          ? i.cmaf
            ? W("ftyp", [
                J("iso5"),
                B(512),
                J("iso5"),
                J("iso6"),
                J("mp41"),
                J("cmfc"),
                J("dash"),
              ])
            : W("ftyp", [J("iso5"), B(512), J("iso5"), J("iso6"), J("mp41")])
          : W("ftyp", [
              J("isom"),
              B(512),
              J("isom"),
              i.holdsAvc ? J("avc1") : [],
              J("mp41"),
            ]),
    Nc = () =>
      W("styp", [
        J("iso5"),
        B(0),
        J("iso5"),
        J("iso6"),
        J("mp41"),
        J("cmfc"),
        J("dash"),
      ]),
    Uc = (i, e) => {
      const t = Math.max(0, i.minWrittenTimestamp);
      let r = Math.max(0, i.maxWrittenEndTimestamp - t);
      return (
        Number.isFinite(r) || (r = 0),
        Q("sidx", 1, 0, [
          B(1),
          B(He),
          Qe(te(t, He)),
          Qe(0),
          V(0),
          V(1),
          B(e & 2147483647),
          B(te(r, He)),
          B(0),
        ])
      );
    },
    Yr = (i) => ({ type: "mdat", largeSize: i }),
    Sf = (i) => ({ type: "free", size: i }),
    gr = (i) =>
      W("moov", void 0, [
        xf(i.creationTime, i.trackDatas),
        ...i.trackDatas.map((e) => Pf(e, i.creationTime)),
        i.isFragmented ? oh(i.trackDatas) : null,
        kh(i),
      ]),
    xf = (i, e) => {
      const t = Math.max(
          0,
          ...e.map((a) =>
            Math.max(0, te(Bi(a), He) + te(a.startTimestampOffset ?? 0, He)),
          ),
        ),
        r = Math.max(0, ...e.map((a) => a.track.id)) + 1,
        n = !ut(i) || !ut(t),
        s = n ? Qe : B;
      return Q("mvhd", +n, 0, [
        s(i),
        s(i),
        B(He),
        s(t),
        Ke(1),
        Oc(1),
        Array(10).fill(0),
        zc(Fi),
        Array(24).fill(0),
        B(r),
      ]);
    },
    Pf = (i, e) => {
      const t = Fh(i),
        r = i.startTimestampOffset !== null && i.startTimestampOffset !== 0;
      return W("trak", void 0, [
        Cf(i, e),
        r ? If(i) : null,
        Ef(i, e),
        t.name !== void 0
          ? W("udta", void 0, [W("name", [...oe.encode(t.name)])])
          : null,
      ]);
    },
    Cf = (i, e) => {
      const t = Math.max(
          0,
          te(Bi(i), He) + te(i.startTimestampOffset ?? 0, He),
        ),
        r = !ut(e) || !ut(t),
        n = r ? Qe : B;
      let s;
      if (i.type === "video" && i.track.metadata.transformationMatrix)
        s = i.track.metadata.transformationMatrix;
      else if (i.type === "video") {
        const { rotation: c, flip: l } = i.track.metadata,
          u = ai(ln(c ?? 0), Ks(l ? -1 : 1, 1));
        s = qs(u, i.info.width, i.info.height);
      } else s = Fi;
      let a = 2;
      i.track.metadata.disposition?.default !== !1 && (a |= 1);
      const o =
        i.type === "video"
          ? 0
          : i.type === "audio"
            ? 1
            : i.type === "subtitle"
              ? 2
              : xe(i);
      return Q("tkhd", +r, a, [
        n(e),
        n(e),
        B(i.track.id),
        B(0),
        n(t),
        Array(8).fill(0),
        V(0),
        V(o),
        Oc(i.type === "audio" ? 1 : 0),
        V(0),
        zc(s),
        Ke(i.type === "video" ? i.info.width : 0),
        Ke(i.type === "video" ? i.info.height : 0),
      ]);
    },
    If = (i) => {
      const e = i.startTimestampOffset;
      if ((g(e !== null), e > 0)) {
        const t = te(e, He),
          r = te(Bi(i), He),
          n = !ut(t) || !ut(r),
          s = n ? Qe : B,
          a = n ? Dc : at;
        return W("edts", void 0, [
          Q("elst", n ? 1 : 0, 0, [
            B(2),
            s(t),
            a(-1),
            Ke(1),
            s(r),
            a(0),
            Ke(1),
          ]),
        ]);
      } else {
        const t = te(-e, i.timescale),
          r = Math.max(0, te(Bi(i), He) + te(e, He)),
          n = !rl(t) || !ut(r),
          s = n ? Qe : B,
          a = n ? Dc : at;
        return W("edts", void 0, [
          Q("elst", n ? 1 : 0, 0, [B(1), s(r), a(t), Ke(1)]),
        ]);
      }
    },
    Ef = (i, e) =>
      W("mdia", void 0, [_f(i, e), Is(!0, vf[i.type], Af[i.type]), Bf(i)]),
    _f = (i, e) => {
      const t = te(Bi(i), i.timescale),
        r = !ut(e) || !ut(t),
        n = r ? Qe : B;
      return Q("mdhd", +r, 0, [
        n(e),
        n(e),
        B(i.timescale),
        n(t),
        V($c(i.track.metadata.languageCode ?? Pe)),
        V(0),
      ]);
    },
    vf = { video: "vide", audio: "soun", subtitle: "text" },
    Af = {
      video: "MediabunnyVideoHandler",
      audio: "MediabunnySoundHandler",
      subtitle: "MediabunnyTextHandler",
    },
    Is = (i, e, t, r = "\0\0\0\0") =>
      Q("hdlr", 0, 0, [i ? J("mhlr") : B(0), J(e), J(r), B(0), B(0), J(t, !0)]),
    Bf = (i) => W("minf", void 0, [Ff[i.type](), Rf(), Of(i)]),
    Ff = {
      video: () => Q("vmhd", 0, 1, [V(0), V(0), V(0), V(0)]),
      audio: () => Q("smhd", 0, 0, [V(0), V(0)]),
      subtitle: () => Q("nmhd", 0, 0),
    },
    Rf = () => W("dinf", void 0, [Mf()]),
    Mf = () => Q("dref", 0, 0, [B(1)], [Df()]),
    Df = () => Q("url ", 0, 1),
    Of = (i) => {
      const e =
        i.compositionTimeOffsetTable.length > 1 ||
        i.compositionTimeOffsetTable.some(
          (t) => t.sampleCompositionTimeOffset !== 0,
        );
      return W("stbl", void 0, [
        zf(i),
        eh(i),
        e ? sh(i) : null,
        e ? ah(i) : null,
        ih(i),
        rh(i),
        nh(i),
        th(i),
      ]);
    },
    zf = (i) => {
      let e;
      if (i.type === "video")
        e = Nf(Ph(i.track.source._codec, i.info.decoderConfig.codec), i);
      else if (i.type === "audio") {
        const t = jc(
          i.track.source._codec,
          i.info.decoderConfig.codec,
          i.muxer.isQuickTime,
        );
        (g(t), (e = qf(t, i)));
      } else i.type === "subtitle" && (e = Zf(Eh[i.track.source._codec], i));
      return (g(e), Q("stsd", 0, 0, [B(1)], [e]));
    },
    Nf = (i, e) =>
      W(
        i,
        [
          Array(6).fill(0),
          V(1),
          V(0),
          V(0),
          Array(12).fill(0),
          V(e.info.width),
          V(e.info.height),
          B(4718592),
          B(4718592),
          B(0),
          V(1),
          ue(Hi.length),
          J(Hi),
          Array(31 - Hi.length).fill(0),
          V(e.info.hasAlphaChannel ? 32 : 24),
          xs(65535),
        ],
        [
          Ch[e.track.source._codec]?.(e) ?? null,
          Uf(e),
          Xs(e.info.decoderConfig.colorSpace) ? null : Wf(e),
          Es(e),
        ],
      ),
    Es = (i) =>
      i.avgBitrate === 0 && i.maxBitrate === 0
        ? null
        : W("btrt", [B(0), B(i.maxBitrate), B(i.avgBitrate)]),
    Uf = (i) =>
      i.info.pixelAspectRatio.num === i.info.pixelAspectRatio.den
        ? null
        : W("pasp", [
            B(i.info.pixelAspectRatio.num),
            B(i.info.pixelAspectRatio.den),
          ]),
    Wf = (i) => {
      const e = i.info.decoderConfig.colorSpace;
      return W("colr", [
        J(i.muxer.isQuickTime ? "nclc" : "nclx"),
        V(e?.primaries != null ? Ht[e.primaries] : 2),
        V(e?.transfer != null ? qt[e.transfer] : 2),
        V(e?.matrix != null ? jt[e.matrix] : 2),
        i.muxer.isQuickTime ? [] : ue((e?.fullRange ? 1 : 0) << 7),
      ]);
    },
    Vf = (i) =>
      i.info.decoderConfig &&
      W("avcC", [...me(i.info.decoderConfig.description)]),
    Lf = (i) =>
      i.info.decoderConfig &&
      W("hvcC", [...me(i.info.decoderConfig.description)]),
    Wc = (i) => {
      if (!i.info.decoderConfig) return null;
      const e = i.info.decoderConfig,
        t = e.codec.split("."),
        r = Number(t[1]),
        n = Number(t[2]),
        s = Number(t[3]),
        a = t[4] ? Number(t[4]) : 1,
        o = t[8] ? Number(t[8]) : Number(e.colorSpace?.fullRange ?? 0),
        c = (s << 4) + (a << 1) + o,
        l = t[5]
          ? Number(t[5])
          : e.colorSpace?.primaries
            ? Ht[e.colorSpace.primaries]
            : 1,
        u = t[6]
          ? Number(t[6])
          : e.colorSpace?.transfer
            ? qt[e.colorSpace.transfer]
            : 1,
        d = t[7]
          ? Number(t[7])
          : e.colorSpace?.matrix
            ? jt[e.colorSpace.matrix]
            : 1;
      return Q("vpcC", 1, 0, [ue(r), ue(n), ue(c), ue(l), ue(u), ue(d), V(0)]);
    },
    Hf = (i) => W("av1C", Oa(i.info.decoderConfig.codec)),
    qf = (i, e) => {
      let t = 0,
        r,
        n = 16;
      const s = pe.includes(e.track.source._codec);
      if (s) {
        const a = e.track.source._codec,
          { sampleSize: o } = Ge(a);
        ((n = 8 * o), n > 16 && (t = 1));
      }
      if ((e.muxer.isQuickTime && (t = 1), t === 0))
        r = [
          Array(6).fill(0),
          V(1),
          V(t),
          V(0),
          B(0),
          V(e.info.numberOfChannels),
          V(n),
          V(0),
          V(0),
          V(e.info.sampleRate < 2 ** 16 ? e.info.sampleRate : 0),
          V(0),
        ];
      else {
        const a = s ? 0 : -2;
        r = [
          Array(6).fill(0),
          V(1),
          V(t),
          V(0),
          B(0),
          V(e.info.numberOfChannels),
          V(Math.min(n, 16)),
          xs(a),
          V(0),
          V(e.info.sampleRate < 2 ** 16 ? e.info.sampleRate : 0),
          V(0),
          s
            ? [B(1), B(n / 8), B((e.info.numberOfChannels * n) / 8)]
            : [B(0), B(0), B(0)],
          B(2),
        ];
      }
      return W(i, r, [
        Ih(e.track.source._codec, e.muxer.isQuickTime)?.(e) ?? null,
        Es(e),
      ]);
    },
    _s = (i) => {
      let e;
      switch (i.track.source._codec) {
        case "aac":
          e = 64;
          break;
        case "mp3":
          e = 107;
          break;
        case "vorbis":
          e = 221;
          break;
        default:
          throw new Error(`Unhandled audio codec: ${i.track.source._codec}`);
      }
      let t = [
        ...ue(e),
        ...ue(21),
        ...Mc(0),
        ...B(i.maxBitrate),
        ...B(i.avgBitrate),
      ];
      if (i.info.decoderConfig.description) {
        const r = me(i.info.decoderConfig.description);
        t = [...t, ...ue(5), ...Cs(r.byteLength), ...r];
      }
      return (
        (t = [
          ...V(1),
          ...ue(0),
          ...ue(4),
          ...Cs(t.length),
          ...t,
          ...ue(6),
          ...ue(1),
          ...ue(2),
        ]),
        (t = [...ue(3), ...Cs(t.length), ...t]),
        Q("esds", 0, 0, t)
      );
    },
    Ut = (i) => W("wave", void 0, [jf(i), $f(i), W("\0\0\0\0")]),
    jf = (i) =>
      W("frma", [
        J(
          jc(
            i.track.source._codec,
            i.info.decoderConfig.codec,
            i.muxer.isQuickTime,
          ),
        ),
      ]),
    $f = (i) => {
      const { littleEndian: e } = Ge(i.track.source._codec);
      return W("enda", [V(+e)]);
    },
    Kf = (i) => {
      let e = i.info.numberOfChannels,
        t = 3840,
        r = i.info.sampleRate,
        n = 0,
        s = 0,
        a = new Uint8Array(0);
      const o = i.info.decoderConfig?.description;
      if (o) {
        g(o.byteLength >= 18);
        const c = me(o),
          l = On(c);
        ((e = l.outputChannelCount),
          (t = l.preSkip),
          (r = l.inputSampleRate),
          (n = l.outputGain),
          (s = l.channelMappingFamily),
          l.channelMappingTable && (a = l.channelMappingTable));
      }
      return W("dOps", [ue(0), ue(e), V(t), B(r), xs(n), ue(s), ...a]);
    },
    Gf = (i) => {
      const e = i.info.decoderConfig?.description;
      g(e);
      const t = me(e);
      return Q("dfLa", 0, 0, [...t.subarray(4)]);
    },
    ot = (i) => {
      const { littleEndian: e, sampleSize: t } = Ge(i.track.source._codec),
        r = +e;
      return Q("pcmC", 0, 0, [ue(r), ue(8 * t)]);
    },
    Xf = (i) => {
      g(i.info.primingPacket);
      const e = xa(i.info.primingPacket.data);
      if (!e)
        throw new Error(
          "Couldn't extract AC-3 frame info from the audio packet. Ensure the packets contain valid AC-3 sync frames (as specified in ETSI TS 102 366).",
        );
      const t = new Uint8Array(3),
        r = new $(t);
      return (
        r.writeBits(2, e.fscod),
        r.writeBits(5, e.bsid),
        r.writeBits(3, e.bsmod),
        r.writeBits(3, e.acmod),
        r.writeBits(1, e.lfeon),
        r.writeBits(5, e.bitRateCode),
        r.writeBits(5, 0),
        W("dac3", [...t])
      );
    },
    Qf = (i) => {
      g(i.info.primingPacket);
      const e = Ca(i.info.primingPacket.data);
      if (!e)
        throw new Error(
          "Couldn't extract E-AC-3 frame info from the audio packet. Ensure the packets contain valid E-AC-3 sync frames (as specified in ETSI TS 102 366).",
        );
      let t = 16;
      for (const a of e.substreams)
        ((t += 23), a.numDepSub > 0 ? (t += 9) : (t += 1));
      const r = Math.ceil(t / 8),
        n = new Uint8Array(r),
        s = new $(n);
      (s.writeBits(13, e.dataRate), s.writeBits(3, e.substreams.length - 1));
      for (const a of e.substreams)
        (s.writeBits(2, a.fscod),
          s.writeBits(5, a.bsid),
          s.writeBits(1, 0),
          s.writeBits(1, 0),
          s.writeBits(3, a.bsmod),
          s.writeBits(3, a.acmod),
          s.writeBits(1, a.lfeon),
          s.writeBits(3, 0),
          s.writeBits(4, a.numDepSub),
          a.numDepSub > 0 ? s.writeBits(9, a.chanLoc) : s.writeBits(1, 0));
      return W("dec3", [...n]);
    },
    Yf = (i) => {
      g(i.info.primingPacket);
      const e = Vn(i.info.primingPacket.data);
      if (!e)
        throw new Error(
          "Couldn't extract DTS frame info from the audio packet. Ensure the packets contain valid DTS frames as specified in ETSI TS 102 114.",
        );
      return W("ddts", [...Jl(e)]);
    },
    Zf = (i, e) =>
      W(i, [Array(6).fill(0), V(1)], [_h[e.track.source._codec](e), Es(e)]),
    Jf = (i) => W("vttC", [...oe.encode(i.info.config.description)]),
    eh = (i) =>
      Q("stts", 0, 0, [
        B(i.timeToSampleTable.length),
        i.timeToSampleTable.map((e) => [B(e.sampleCount), B(e.sampleDelta)]),
      ]),
    th = (i) => {
      if (i.samples.every((t) => t.type === "key")) return null;
      const e = [...i.samples.entries()].filter(([, t]) => t.type === "key");
      return Q("stss", 0, 0, [B(e.length), e.map(([t]) => B(t + 1))]);
    },
    ih = (i) =>
      Q("stsc", 0, 0, [
        B(i.compactlyCodedChunkTable.length),
        i.compactlyCodedChunkTable.map((e) => [
          B(e.firstChunk),
          B(e.samplesPerChunk),
          B(1),
        ]),
      ]),
    rh = (i) => {
      if (i.type === "audio" && i.info.requiresPcmTransformation) {
        const { sampleSize: e } = Ge(i.track.source._codec);
        return Q("stsz", 0, 0, [
          B(e * i.info.numberOfChannels),
          B(i.samples.reduce((t, r) => t + te(r.duration, i.timescale), 0)),
        ]);
      }
      return Q("stsz", 0, 0, [
        B(0),
        B(i.samples.length),
        i.samples.map((e) => B(e.size)),
      ]);
    },
    nh = (i) =>
      i.finalizedChunks.length > 0 && ne(i.finalizedChunks).offset >= 2 ** 32
        ? Q("co64", 0, 0, [
            B(i.finalizedChunks.length),
            i.finalizedChunks.map((e) => Qe(e.offset)),
          ])
        : Q("stco", 0, 0, [
            B(i.finalizedChunks.length),
            i.finalizedChunks.map((e) => B(e.offset)),
          ]),
    sh = (i) =>
      Q("ctts", 1, 0, [
        B(i.compositionTimeOffsetTable.length),
        i.compositionTimeOffsetTable.map((e) => [
          B(e.sampleCount),
          at(e.sampleCompositionTimeOffset),
        ]),
      ]),
    ah = (i) => {
      let e = 1 / 0,
        t = -1 / 0,
        r = 1 / 0,
        n = -1 / 0;
      (g(i.compositionTimeOffsetTable.length > 0), g(i.samples.length > 0));
      for (let a = 0; a < i.compositionTimeOffsetTable.length; a++) {
        const o = i.compositionTimeOffsetTable[a];
        ((e = Math.min(e, o.sampleCompositionTimeOffset)),
          (t = Math.max(t, o.sampleCompositionTimeOffset)));
      }
      for (let a = 0; a < i.samples.length; a++) {
        const o = i.samples[a];
        ((r = Math.min(r, te(o.timestamp, i.timescale))),
          (n = Math.max(n, te(o.timestamp + o.duration, i.timescale))));
      }
      const s = Math.max(-e, 0);
      return n >= 2 ** 31
        ? null
        : Q("cslg", 0, 0, [at(s), at(e), at(t), at(r), at(n)]);
    },
    oh = (i) => W("mvex", void 0, i.map(ch)),
    ch = (i) => Q("trex", 0, 0, [B(i.track.id), B(1), B(0), B(0), B(0)]),
    Vc = (i, e) => W("moof", void 0, [lh(i), ...e.map(uh)]),
    lh = (i) => Q("mfhd", 0, 0, [B(i)]),
    Lc = (i) => {
      let e = 0,
        t = 0;
      const r = 0,
        n = 0,
        s = i.type === "delta";
      return (
        (t |= +s),
        s ? (e |= 1) : (e |= 2),
        (e << 24) | (t << 16) | (r << 8) | n
      );
    },
    uh = (i) => W("traf", void 0, [dh(i), fh(i), hh(i)]),
    dh = (i) => {
      g(i.currentChunk);
      let e = 0;
      ((e |= 8), (e |= 16), (e |= 32), (e |= 131072));
      const t = i.currentChunk.samples[1] ?? i.currentChunk.samples[0],
        r = {
          duration: t.timescaleUnitsToNextSample,
          size: t.size,
          flags: Lc(t),
        };
      return Q("tfhd", 0, e, [
        B(i.track.id),
        B(r.duration),
        B(r.size),
        B(r.flags),
      ]);
    },
    fh = (i) => (
      g(i.currentChunk),
      Q("tfdt", 1, 0, [Qe(te(i.currentChunk.startTimestamp, i.timescale))])
    ),
    hh = (i) => {
      g(i.currentChunk);
      const e = i.currentChunk.samples.map((m) => m.timescaleUnitsToNextSample),
        t = i.currentChunk.samples.map((m) => m.size),
        r = i.currentChunk.samples.map(Lc),
        n = i.currentChunk.samples.map((m) =>
          te(m.timestamp - m.decodeTimestamp, i.timescale),
        ),
        s = new Set(e),
        a = new Set(t),
        o = new Set(r),
        c = new Set(n),
        l = o.size === 2 && r[0] !== r[1],
        u = s.size > 1,
        d = a.size > 1,
        f = !l && o.size > 1,
        h = c.size > 1 || [...c].some((m) => m !== 0);
      let p = 0;
      return (
        (p |= 1),
        (p |= 4 * +l),
        (p |= 256 * +u),
        (p |= 512 * +d),
        (p |= 1024 * +f),
        (p |= 2048 * +h),
        Q("trun", 1, p, [
          B(i.currentChunk.samples.length),
          B(i.currentChunk.offset - i.currentChunk.moofOffset || 0),
          l ? B(r[0]) : [],
          i.currentChunk.samples.map((m, y) => [
            u ? B(e[y]) : [],
            d ? B(t[y]) : [],
            f ? B(r[y]) : [],
            h ? at(n[y]) : [],
          ]),
        ])
      );
    },
    mh = (i) => W("mfra", void 0, [...i.map(ph), gh()]),
    ph = (i) =>
      Q("tfra", 1, 0, [
        B(i.track.id),
        B(63),
        B(i.finalizedChunks.length),
        i.finalizedChunks.map((t) => [
          Qe(te(t.samples[0].timestamp, i.timescale)),
          Qe(t.moofOffset),
          B(t.trafIndex + 1),
          B(1),
          B(1),
        ]),
      ]),
    gh = () => Q("mfro", 0, 0, [B(0)]),
    wh = () => W("vtte"),
    yh = (i, e, t, r, n) =>
      W("vttc", void 0, [
        n !== null ? W("vsid", [at(n)]) : null,
        t !== null ? W("iden", [...oe.encode(t)]) : null,
        e !== null ? W("ctim", [...oe.encode(Rc(e))]) : null,
        r !== null ? W("sttg", [...oe.encode(r)]) : null,
        W("payl", [...oe.encode(i)]),
      ]),
    bh = (i) => W("vtta", [...oe.encode(i)]),
    kh = (i) => {
      const e = [],
        t = i.format._options.metadataFormat ?? "auto",
        r = i.output._metadataTags;
      if (t === "mdir" || (t === "auto" && !i.isQuickTime)) {
        const n = Sh(r);
        n && e.push(n);
      } else if (t === "mdta") {
        const n = xh(r);
        n && e.push(n);
      } else
        (t === "udta" || (t === "auto" && i.isQuickTime)) &&
          Th(e, i.output._metadataTags);
      return e.length === 0 ? null : W("udta", void 0, e);
    },
    Th = (i, e) => {
      for (const { key: t, value: r } of Wi(e))
        switch (t) {
          case "title":
            i.push(ct("©nam", r));
            break;
          case "description":
            i.push(ct("©des", r));
            break;
          case "artist":
            i.push(ct("©ART", r));
            break;
          case "album":
            i.push(ct("©alb", r));
            break;
          case "albumArtist":
            i.push(ct("albr", r));
            break;
          case "genre":
            i.push(ct("©gen", r));
            break;
          case "date":
            i.push(ct("©day", r.toISOString().slice(0, 10)));
            break;
          case "comment":
            i.push(ct("©cmt", r));
            break;
          case "lyrics":
            i.push(ct("©lyr", r));
            break;
          case "raw":
            break;
          case "discNumber":
          case "discsTotal":
          case "trackNumber":
          case "tracksTotal":
          case "beatsPerMinute":
          case "images":
            break;
          default:
            xe(t);
        }
      if (e.raw)
        for (const t in e.raw) {
          const r = e.raw[t];
          r == null ||
            t.length !== 4 ||
            i.some((n) => n.type === t) ||
            (typeof r == "string"
              ? i.push(ct(t, r))
              : r instanceof Uint8Array && i.push(W(t, Array.from(r))));
        }
    },
    ct = (i, e) => {
      const t = oe.encode(e);
      return W(i, [V(t.length), V($c("und")), Array.from(t)]);
    },
    Hc = { "image/jpeg": 13, "image/png": 14, "image/bmp": 27 },
    qc = (i, e) => {
      const t = [];
      for (const { key: r, value: n } of Wi(i))
        switch (r) {
          case "title":
            t.push({ key: e ? "title" : "©nam", value: Ye(n) });
            break;
          case "description":
            t.push({ key: e ? "description" : "©des", value: Ye(n) });
            break;
          case "artist":
            t.push({ key: e ? "artist" : "©ART", value: Ye(n) });
            break;
          case "album":
            t.push({ key: e ? "album" : "©alb", value: Ye(n) });
            break;
          case "albumArtist":
            t.push({ key: e ? "album_artist" : "aART", value: Ye(n) });
            break;
          case "comment":
            t.push({ key: e ? "comment" : "©cmt", value: Ye(n) });
            break;
          case "genre":
            t.push({ key: e ? "genre" : "©gen", value: Ye(n) });
            break;
          case "beatsPerMinute":
            e || t.push({ key: "tmpo", value: W("data", [B(21), B(0), V(n)]) });
            break;
          case "lyrics":
            t.push({ key: e ? "lyrics" : "©lyr", value: Ye(n) });
            break;
          case "date":
            t.push({
              key: e ? "date" : "©day",
              value: Ye(n.toISOString().slice(0, 10)),
            });
            break;
          case "images":
            for (const s of n)
              s.kind === "coverFront" &&
                t.push({
                  key: "covr",
                  value: W("data", [
                    B(Hc[s.mimeType] ?? 0),
                    B(0),
                    Array.from(s.data),
                  ]),
                });
            break;
          case "trackNumber":
            if (e) {
              const s =
                i.tracksTotal !== void 0
                  ? `${n}/${i.tracksTotal}`
                  : n.toString();
              t.push({ key: "track", value: Ye(s) });
            } else
              t.push({
                key: "trkn",
                value: W("data", [
                  B(0),
                  B(0),
                  V(0),
                  V(n),
                  V(i.tracksTotal ?? 0),
                  V(0),
                ]),
              });
            break;
          case "discNumber":
            e ||
              t.push({
                key: "disc",
                value: W("data", [
                  B(0),
                  B(0),
                  V(0),
                  V(n),
                  V(i.discsTotal ?? 0),
                  V(0),
                ]),
              });
            break;
          case "tracksTotal":
          case "discsTotal":
            break;
          case "raw":
            break;
          default:
            xe(r);
        }
      if (i.raw)
        for (const r in i.raw) {
          const n = i.raw[r];
          n == null ||
            (!e && r.length !== 4) ||
            t.some((s) => s.key === r) ||
            (typeof n == "string"
              ? t.push({ key: r, value: Ye(n) })
              : n instanceof Uint8Array
                ? t.push({
                    key: r,
                    value: W("data", [B(0), B(0), Array.from(n)]),
                  })
                : n instanceof hi &&
                  t.push({
                    key: r,
                    value: W("data", [
                      B(Hc[n.mimeType] ?? 0),
                      B(0),
                      Array.from(n.data),
                    ]),
                  }));
        }
      return t;
    },
    Sh = (i) => {
      const e = qc(i, !1);
      return e.length === 0
        ? null
        : Q("meta", 0, 0, void 0, [
            Is(!1, "mdir", "", "appl"),
            W(
              "ilst",
              void 0,
              e.map((t) => W(t.key, void 0, [t.value])),
            ),
          ]);
    },
    xh = (i) => {
      const e = qc(i, !0);
      return e.length === 0
        ? null
        : W("meta", void 0, [
            Is(!1, "mdta", ""),
            Q(
              "keys",
              0,
              0,
              [B(e.length)],
              e.map((t) => W("mdta", [...oe.encode(t.key)])),
            ),
            W(
              "ilst",
              void 0,
              e.map((t, r) => {
                const n = String.fromCharCode(...B(r + 1));
                return W(n, void 0, [t.value]);
              }),
            ),
          ]);
    },
    Ye = (i) => W("data", [B(1), B(0), ...oe.encode(i)]),
    Ph = (i, e) => {
      switch (i) {
        case "avc":
          return e.startsWith("avc3") ? "avc3" : "avc1";
        case "hevc":
          return "hvc1";
        case "vp8":
          return "vp08";
        case "vp9":
          return "vp09";
        case "av1":
          return "av01";
        case "prores":
          return e;
      }
    },
    Ch = { avc: Vf, hevc: Lf, vp8: Wc, vp9: Wc, av1: Hf, prores: null },
    jc = (i, e, t) => {
      switch (i) {
        case "aac":
          return "mp4a";
        case "mp3":
          return "mp4a";
        case "opus":
          return "Opus";
        case "vorbis":
          return "mp4a";
        case "flac":
          return "fLaC";
        case "ulaw":
          return "ulaw";
        case "alaw":
          return "alaw";
        case "pcm-u8":
          return "raw ";
        case "pcm-s8":
          return "sowt";
        case "ac3":
          return "ac-3";
        case "eac3":
          return "ec-3";
        case "dts":
          return e;
      }
      if (t)
        switch (i) {
          case "pcm-s16":
            return "sowt";
          case "pcm-s16be":
            return "twos";
          case "pcm-s24":
            return "in24";
          case "pcm-s24be":
            return "in24";
          case "pcm-s32":
            return "in32";
          case "pcm-s32be":
            return "in32";
          case "pcm-f32":
            return "fl32";
          case "pcm-f32be":
            return "fl32";
          case "pcm-f64":
            return "fl64";
          case "pcm-f64be":
            return "fl64";
        }
      else
        switch (i) {
          case "pcm-s16":
            return "ipcm";
          case "pcm-s16be":
            return "ipcm";
          case "pcm-s24":
            return "ipcm";
          case "pcm-s24be":
            return "ipcm";
          case "pcm-s32":
            return "ipcm";
          case "pcm-s32be":
            return "ipcm";
          case "pcm-f32":
            return "fpcm";
          case "pcm-f32be":
            return "fpcm";
          case "pcm-f64":
            return "fpcm";
          case "pcm-f64be":
            return "fpcm";
        }
    },
    Ih = (i, e) => {
      switch (i) {
        case "aac":
          return _s;
        case "mp3":
          return _s;
        case "opus":
          return Kf;
        case "vorbis":
          return _s;
        case "flac":
          return Gf;
        case "ac3":
          return Xf;
        case "eac3":
          return Qf;
        case "dts":
          return Yf;
      }
      if (e)
        switch (i) {
          case "pcm-s24":
            return Ut;
          case "pcm-s24be":
            return Ut;
          case "pcm-s32":
            return Ut;
          case "pcm-s32be":
            return Ut;
          case "pcm-f32":
            return Ut;
          case "pcm-f32be":
            return Ut;
          case "pcm-f64":
            return Ut;
          case "pcm-f64be":
            return Ut;
        }
      else
        switch (i) {
          case "pcm-s16":
            return ot;
          case "pcm-s16be":
            return ot;
          case "pcm-s24":
            return ot;
          case "pcm-s24be":
            return ot;
          case "pcm-s32":
            return ot;
          case "pcm-s32be":
            return ot;
          case "pcm-f32":
            return ot;
          case "pcm-f32be":
            return ot;
          case "pcm-f64":
            return ot;
          case "pcm-f64be":
            return ot;
        }
      return null;
    },
    Eh = { webvtt: "wvtt" },
    _h = { webvtt: Jf },
    $c = (i) => {
      g(i.length === 3);
      let e = 0;
      for (let t = 0; t < 3; t++) ((e <<= 5), (e += i.charCodeAt(t) - 96));
      return e;
    };
  class vs {
    constructor(e, t) {
      if (
        ((this.finalized = !1),
        (this.started = !1),
        (this.pos = 0),
        (this.trackedWrites = null),
        (this.trackedStart = -1),
        (this.trackedEnd = -1),
        e._writerAcquired)
      )
        throw new Error("Can't have multiple Writers for the same Target.");
      ((this.target = e), e._setMonotonicity(t), (e._writerAcquired = !0));
    }
    start() {
      (g(!this.started), this.target._start(), (this.started = !0));
    }
    write(e) {
      (g(this.started && !this.finalized),
        this.maybeTrackWrites(e),
        this.target._write(e, this.pos),
        (this.pos += e.byteLength));
    }
    seek(e) {
      this.pos = e;
    }
    getPos() {
      return this.pos;
    }
    async flush() {
      return (g(this.started && !this.finalized), this.target._flush());
    }
    async finalize() {
      (g(this.started && !this.finalized),
        await this.target._finalize(),
        (this.finalized = !0));
    }
    maybeTrackWrites(e) {
      if (!this.trackedWrites) return;
      let t = this.getPos();
      if (t < this.trackedStart) {
        if (t + e.byteLength <= this.trackedStart) return;
        ((e = e.subarray(this.trackedStart - t)), (t = 0));
      }
      const r = t + e.byteLength - this.trackedStart;
      let n = this.trackedWrites.byteLength;
      for (; n < r;) n *= 2;
      if (n !== this.trackedWrites.byteLength) {
        const s = new Uint8Array(n);
        (s.set(this.trackedWrites, 0), (this.trackedWrites = s));
      }
      (this.trackedWrites.set(e, t - this.trackedStart),
        (this.trackedEnd = Math.max(this.trackedEnd, t + e.byteLength)));
    }
    startTrackingWrites() {
      ((this.trackedWrites = new Uint8Array(2 ** 10)),
        (this.trackedStart = this.getPos()),
        (this.trackedEnd = this.trackedStart));
    }
    stopTrackingWrites() {
      if (!this.trackedWrites)
        throw new Error(
          "Internal error: Can't get tracked writes since nothing was tracked.",
        );
      const t = {
        data: this.trackedWrites.subarray(
          0,
          this.trackedEnd - this.trackedStart,
        ),
        start: this.trackedStart,
        end: this.trackedEnd,
      };
      return ((this.trackedWrites = null), t);
    }
  }
  class lt extends Li {
    constructor() {
      (super(...arguments),
        (this._writerAcquired = !1),
        (this._monotonicity = null),
        (this.onwrite = null));
    }
    _setMonotonicity(e) {
      this._monotonicity !== !1 && (this._monotonicity = e);
    }
    _dispatchWrite(e, t) {
      (this.onwrite?.(e, t), this._emit("write", { start: e, end: t }));
    }
    slice(e) {
      if (!Number.isInteger(e) || e < 0)
        throw new TypeError("offset must be a non-negative integer.");
      return new Ah(this, e);
    }
  }
  const As = 2 ** 16,
    Bs = 2 ** 32;
  class wr extends lt {
    constructor(e = {}) {
      if (
        (super(),
        (this.buffer = null),
        (this._maxPos = 0),
        !e || typeof e != "object")
      )
        throw new TypeError(
          "BufferTarget options, when provided, must be an object.",
        );
      if (e.onFinalize !== void 0 && typeof e.onFinalize != "function")
        throw new TypeError(
          "options.onFinalize, when provided, must be a function.",
        );
      if (
        ((this._options = e),
        (this._supportsResize = "resize" in new ArrayBuffer(0)),
        this._supportsResize)
      )
        try {
          this._buffer = new ArrayBuffer(As, { maxByteLength: Bs });
        } catch {
          ((this._buffer = new ArrayBuffer(As)), (this._supportsResize = !1));
        }
      else this._buffer = new ArrayBuffer(As);
      this._bytes = new Uint8Array(this._buffer);
    }
    _ensureSize(e) {
      let t = this._buffer.byteLength;
      for (; t < e;) t *= 2;
      if (t !== this._buffer.byteLength) {
        if (t > Bs)
          throw new Error(
            `ArrayBuffer exceeded maximum size of ${Bs} bytes. Please consider using another target.`,
          );
        if (this._supportsResize) this._buffer.resize(t);
        else {
          const r = new ArrayBuffer(t),
            n = new Uint8Array(r);
          (n.set(this._bytes, 0), (this._buffer = r), (this._bytes = n));
        }
      }
    }
    _start() {}
    _write(e, t) {
      (this._ensureSize(t + e.byteLength),
        this._bytes.set(e, t),
        (this._maxPos = Math.max(this._maxPos, t + e.byteLength)),
        this._dispatchWrite(t, t + e.byteLength));
    }
    async _flush() {}
    async _finalize() {
      ((this.buffer = this._buffer.slice(0, this._maxPos)),
        this._options.onFinalize &&
          (await this._options.onFinalize(this.buffer)),
        this._emit("finalized"));
    }
    async _close() {}
    _getSlice(e, t) {
      return this._bytes.slice(e, t);
    }
  }
  class vh extends lt {
    _start() {}
    _write(e, t) {
      this._dispatchWrite(t, t + e.byteLength);
    }
    async _flush() {}
    async _finalize() {
      this._emit("finalized");
    }
    async _close() {}
  }
  class Ah extends lt {
    constructor(e, t) {
      (super(), (this._baseTarget = e), (this._offset = t));
    }
    _start() {}
    _write(e, t) {
      (this._baseTarget._write(e, this._offset + t),
        this._dispatchWrite(t, t + e.byteLength));
    }
    _flush() {
      return this._baseTarget._flush();
    }
    async _finalize() {
      this._emit("finalized");
    }
    async _close() {}
    _setMonotonicity(e) {
      (super._setMonotonicity(e), this._baseTarget._setMonotonicity(e));
    }
  }
  class Fs {
    constructor(e, t) {
      if (((this.rootPath = e), (this.getTarget = t), typeof e != "string"))
        throw new TypeError("rootPath must be a string.");
      if (typeof t != "function")
        throw new TypeError("getTarget must be a function.");
    }
  }
  const He = 57600,
    Bh = 2082844800,
    Fh = (i) => {
      const e = {},
        t = i.track;
      return (t.metadata.name !== void 0 && (e.name = t.metadata.name), e);
    },
    te = (i, e, t = !0) => {
      const r = i * e;
      return t ? Math.round(r) : r;
    },
    Bi = (i) => {
      if (i.samples.length === 0) return 0;
      let e = 1 / 0,
        t = -1 / 0;
      for (let r = 0; r < i.samples.length; r++) {
        const n = i.samples[r];
        (n.timestamp < e && (e = n.timestamp),
          n.timestamp + n.duration > t && (t = n.timestamp + n.duration));
      }
      return e === 1 / 0 ? 0 : t - e;
    };
  class Rh extends Ss {
    constructor(e, t) {
      (super(e),
        (this.writer = null),
        (this.boxWriter = null),
        (this.initWriter = null),
        (this.initBoxWriter = null),
        (this.auxTarget = new wr()),
        (this.auxWriter = new vs(this.auxTarget, !1)),
        (this.auxBoxWriter = new Qr(this.auxWriter)),
        (this.mdat = null),
        (this.ftypSize = null),
        (this.trackDatas = []),
        (this.allTracksKnown = ce()),
        (this.creationTime = Math.floor(Date.now() / 1e3) + Bh),
        (this.finalizedChunks = []),
        (this.wroteFragmentedHeader = !1),
        (this.nextFragmentNumber = 1),
        (this.maxWrittenTimestamp = -1 / 0),
        (this.minWrittenTimestamp = 1 / 0),
        (this.maxWrittenEndTimestamp = -1 / 0),
        (this.segmentHeaderSize = null),
        (this.format = t),
        (this.formatOptions = { ...t._options }),
        (this.isQuickTime = t instanceof Yc),
        (this.isCmaf = t instanceof Qc),
        (this.minimumFragmentDuration =
          this.formatOptions.minimumFragmentDuration ??
          (t instanceof Qc ? 1 / 0 : 1)),
        this.auxWriter.start());
    }
    async start() {
      const e = await this.mutex.acquire();
      if (
        (this.isCmaf
          ? ((this.fastStart = "fragmented"), (this.isFragmented = !0))
          : ((this.writer = await this.output._getRootWriter((r) =>
              this.formatOptions.fastStart !== void 0
                ? this.formatOptions.fastStart === "fragmented"
                : r instanceof wr,
            )),
            (this.boxWriter = new Qr(this.writer)),
            (this.fastStart =
              this.formatOptions.fastStart ??
              (this.writer.target instanceof wr ? "in-memory" : !1)),
            (this.isFragmented = this.fastStart === "fragmented")),
        this.isCmaf)
      ) {
        if (!this.output._hasInitTarget())
          throw new Error(
            "CMAF outputs require the initTarget field in OutputOptions to be set; the init segment will be written to it.",
          );
        const r = await this.output._getInitTarget(),
          n = new vs(r, !0);
        (n.start(), (this.initWriter = n), (this.initBoxWriter = new Qr(n)));
      }
      const t = this.output.tracks.some(
        (r) => r.isVideoTrack() && r.source._codec === "avc",
      );
      {
        const r = this.initBoxWriter ?? this.boxWriter;
        if (
          (g(r),
          this.formatOptions.onFtyp && r.writer.startTrackingWrites(),
          r.writeBox(
            Tf({
              isQuickTime: this.isQuickTime,
              holdsAvc: t,
              fragmented: this.isFragmented,
              cmaf: this.isCmaf,
            }),
          ),
          this.formatOptions.onFtyp)
        ) {
          const { data: n, start: s } = r.writer.stopTrackingWrites();
          this.formatOptions.onFtyp(n, s);
        }
        ((this.ftypSize = r.writer.getPos()),
          this.isCmaf && (await this.initWriter.flush()));
      }
      if (this.fastStart !== "in-memory")
        if (this.fastStart === "reserve") {
          for (const r of this.output.tracks)
            if (r.metadata.maximumPacketCount === void 0)
              throw new Error(
                "All tracks must specify maximumPacketCount in their metadata when using fastStart: 'reserve'.",
              );
        } else
          this.isFragmented ||
            (g(this.writer),
            g(this.boxWriter),
            this.formatOptions.onMdat && this.writer.startTrackingWrites(),
            (this.mdat = Yr(!0)),
            this.boxWriter.writeBox(this.mdat));
      await this.writer?.flush();
      for (const r of this.output.tracks)
        r.isVideoTrack() && r.metadata.decoderConfig
          ? this.getVideoTrackData(r, r.metadata.primingPacket ?? null, {
              decoderConfig: r.metadata.decoderConfig,
            })
          : r.isAudioTrack() &&
            r.metadata.decoderConfig &&
            this.getAudioTrackData(r, r.metadata.primingPacket ?? null, {
              decoderConfig: r.metadata.decoderConfig,
            });
      e();
    }
    allTracksAreKnown() {
      for (const e of this.output.tracks)
        if (!e.source._closed && !this.trackDatas.some((t) => t.track === e))
          return !1;
      return !0;
    }
    async getMimeType() {
      await this.allTracksKnown.promise;
      const e = this.trackDatas.map((t) =>
        t.type === "video" || t.type === "audio"
          ? t.info.decoderConfig.codec
          : { webvtt: "wvtt" }[t.track.source._codec],
      );
      return Ha({
        isQuickTime: this.isQuickTime,
        hasVideo: this.trackDatas.some((t) => t.type === "video"),
        hasAudio: this.trackDatas.some((t) => t.type === "audio"),
        codecStrings: e,
      });
    }
    getVideoTrackData(e, t, r) {
      const n = this.trackDatas.find((h) => h.track === e);
      if (n) return n;
      (Hn(r, e.source._codec), g(r), g(r.decoderConfig));
      const s = { ...r.decoderConfig };
      (g(s.codedWidth !== void 0), g(s.codedHeight !== void 0));
      let a = !1;
      if (e.source._codec === "avc" && !s.description) {
        if (!t)
          throw new Error(
            "No AVC description provided; you must therefore provide a priming packet.",
          );
        const h = $i(t.data);
        if (!h)
          throw new Error(
            "Couldn't extract an AVCDecoderConfigurationRecord from the AVC packet. Make sure the packets are in Annex B format (as specified in ITU-T-REC-H.264) when not providing a description, or provide a description (must be an AVCDecoderConfigurationRecord as specified in ISO 14496-15) and ensure the packets are in AVCC format.",
          );
        ((s.description = ma(h)), (a = !0));
      } else if (e.source._codec === "hevc" && !s.description) {
        if (!t)
          throw new Error(
            "No HEVC description provided; you must therefore provide a priming packet.",
          );
        const h = Ki(t.data);
        if (!h)
          throw new Error(
            "Couldn't extract an HEVCDecoderConfigurationRecord from the HEVC packet. Make sure the packets are in Annex B format (as specified in ITU-T-REC-H.265) when not providing a description, or provide a description (must be an HEVCDecoderConfigurationRecord as specified in ISO 14496-15) and ensure the packets are in HEVC format.",
          );
        ((s.description = Bl(h)), (a = !0));
      }
      const o = pl(1 / (e.metadata.frameRate ?? He), 1e6).den,
        c = s.displayAspectWidth,
        l = s.displayAspectHeight,
        u =
          c === void 0 || l === void 0
            ? { num: 1, den: 1 }
            : di({ num: c * s.codedHeight, den: l * s.codedWidth }),
        d = s.codec === "ap4h" || s.codec === "ap4x",
        f = {
          muxer: this,
          track: e,
          type: "video",
          info: {
            width: s.codedWidth,
            height: s.codedHeight,
            pixelAspectRatio: u,
            decoderConfig: s,
            requiresAnnexBTransformation: a,
            hasAlphaChannel: d,
          },
          timescale: o,
          samples: [],
          sampleQueue: [],
          timestampProcessingQueue: [],
          timeToSampleTable: [],
          compositionTimeOffsetTable: [],
          lastTimescaleUnits: null,
          lastSample: null,
          startTimestampOffset: null,
          finalizedChunks: [],
          currentChunk: null,
          compactlyCodedChunkTable: [],
          closed: !1,
          avgBitrate:
            e.source._nominalBitrate ?? e.metadata.averageBitrate ?? 0,
          maxBitrate: e.source._nominalBitrate ?? e.metadata.bitrate ?? 0,
        };
      return (
        this.trackDatas.push(f),
        this.trackDatas.sort((h, p) => h.track.id - p.track.id),
        this.allTracksAreKnown() && this.allTracksKnown.resolve(),
        f
      );
    }
    getAudioTrackData(e, t, r) {
      const n = this.trackDatas.find((c) => c.track === e);
      if (n) return n;
      (Xi(r, e.source._codec), g(r), g(r.decoderConfig));
      const s = { ...r.decoderConfig };
      let a = !1;
      if (e.source._codec === "aac" && !s.description) {
        if (!t)
          throw new Error(
            "No AAC description provided; you must therefore provide a priming packet.",
          );
        const c = Pt(Me.tempFromBytes(t.data));
        if (!c)
          throw new Error(
            "Couldn't parse ADTS header from the AAC packet. Make sure the packets are in ADTS format (as specified in ISO 13818-7) when not providing a description, or provide a description (must be an AudioSpecificConfig as specified in ISO 14496-3) and ensure the packets are raw AAC data.",
          );
        const l = At[c.samplingFrequencyIndex],
          u = mi[c.channelConfiguration];
        if (l === void 0 || u === void 0)
          throw new Error("Invalid ADTS frame header.");
        ((s.description = An({
          objectType: c.objectType,
          outputSampleRate: l,
          outputNumberOfChannels: u,
        })),
          (a = !0));
      }
      if (!t) {
        if (e.source._codec === "ac3" || e.source._codec === "eac3")
          throw new Error("AC-3/E-AC-3 require a priming packet.");
        if (e.source._codec === "dts")
          throw new Error("DTS requires a priming packet.");
      }
      const o = {
        muxer: this,
        track: e,
        type: "audio",
        info: {
          numberOfChannels: r.decoderConfig.numberOfChannels,
          sampleRate: r.decoderConfig.sampleRate,
          decoderConfig: s,
          requiresPcmTransformation:
            !this.isFragmented && pe.includes(e.source._codec),
          expectedNextPcmPacketTimestamp: null,
          requiresAdtsStripping: a,
          primingPacket: t,
        },
        timescale: s.sampleRate,
        samples: [],
        sampleQueue: [],
        timestampProcessingQueue: [],
        timeToSampleTable: [],
        compositionTimeOffsetTable: [],
        lastTimescaleUnits: null,
        lastSample: null,
        startTimestampOffset: null,
        finalizedChunks: [],
        currentChunk: null,
        compactlyCodedChunkTable: [],
        closed: !1,
        avgBitrate: e.source._nominalBitrate ?? e.metadata.averageBitrate ?? 0,
        maxBitrate: e.source._nominalBitrate ?? e.metadata.bitrate ?? 0,
      };
      return (
        this.trackDatas.push(o),
        this.trackDatas.sort((c, l) => c.track.id - l.track.id),
        this.allTracksAreKnown() && this.allTracksKnown.resolve(),
        o
      );
    }
    getSubtitleTrackData(e, t) {
      const r = this.trackDatas.find((s) => s.track === e);
      if (r) return r;
      (Ua(t), g(t), g(t.config));
      const n = {
        muxer: this,
        track: e,
        type: "subtitle",
        info: { config: t.config },
        timescale: 1e3,
        samples: [],
        sampleQueue: [],
        timestampProcessingQueue: [],
        timeToSampleTable: [],
        compositionTimeOffsetTable: [],
        lastTimescaleUnits: null,
        lastSample: null,
        startTimestampOffset: null,
        finalizedChunks: [],
        currentChunk: null,
        compactlyCodedChunkTable: [],
        closed: !1,
        avgBitrate: e.source._nominalBitrate ?? e.metadata.averageBitrate ?? 0,
        maxBitrate: e.source._nominalBitrate ?? e.metadata.bitrate ?? 0,
        lastCueEndTimestamp: null,
        cueQueue: [],
        nextSourceId: 0,
        cueToSourceId: new WeakMap(),
      };
      return (
        this.trackDatas.push(n),
        this.trackDatas.sort((s, a) => s.track.id - a.track.id),
        this.allTracksAreKnown() && this.allTracksKnown.resolve(),
        n
      );
    }
    async addEncodedVideoPacket(e, t, r) {
      const n = await this.mutex.acquire();
      try {
        const s = this.getVideoTrackData(e, t, r);
        let a = t.data;
        if (s.info.requiresAnnexBTransformation) {
          const c = [...qi(a)].map((l) =>
            a.subarray(l.offset, l.offset + l.length),
          );
          if (c.length === 0)
            throw new Error(
              "Failed to transform packet data. Make sure all packets are provided in Annex B format, as specified in ITU-T-REC-H.264 and ITU-T-REC-H.265.",
            );
          a = Rn(c, 4);
        }
        this.validateTimestamp(s.track, t.timestamp, t.type === "key");
        const o = this.createSampleForTrack(
          s,
          a,
          t.timestamp,
          t.duration,
          t.type,
        );
        await this.registerSample(s, o);
      } finally {
        n();
      }
    }
    async addEncodedAudioPacket(e, t, r) {
      const n = await this.mutex.acquire();
      try {
        const s = this.getAudioTrackData(e, t, r);
        let a = t.data;
        if (s.info.requiresAdtsStripping) {
          const u = Pt(Me.tempFromBytes(a));
          if (!u) throw new Error("Expected ADTS frame, didn't get one.");
          const d = u.crcCheck === null ? nr : Ot;
          a = a.subarray(d);
        }
        this.validateTimestamp(s.track, t.timestamp, t.type === "key");
        let o = t.timestamp,
          c = t.duration;
        if (s.info.requiresPcmTransformation) {
          const d =
            Ge(s.info.decoderConfig.codec).sampleSize * s.info.numberOfChannels;
          if (
            ((c = a.byteLength / d / s.info.sampleRate),
            s.info.expectedNextPcmPacketTimestamp !== null)
          ) {
            const f = o - s.info.expectedNextPcmPacketTimestamp;
            if (f < 0.01) o = s.info.expectedNextPcmPacketTimestamp;
            else {
              const h = await this.padWithSilence(
                s,
                s.info.expectedNextPcmPacketTimestamp,
                f,
              );
              o = s.info.expectedNextPcmPacketTimestamp + h;
            }
          }
          s.info.expectedNextPcmPacketTimestamp = o + c;
        }
        const l = this.createSampleForTrack(s, a, o, c, t.type);
        await this.registerSample(s, l);
      } finally {
        n();
      }
    }
    async padWithSilence(e, t, r) {
      const n = te(r, e.timescale);
      if (((r = n / e.timescale), n > 0)) {
        const { sampleSize: s, silentValue: a } = Ge(
            e.info.decoderConfig.codec,
          ),
          o = n * e.info.numberOfChannels,
          c = new Uint8Array(s * o).fill(a),
          l = this.createSampleForTrack(
            e,
            new Uint8Array(c.buffer),
            t,
            r,
            "key",
          );
        await this.registerSample(e, l);
      }
      return r;
    }
    async addSubtitleCue(e, t, r) {
      const n = await this.mutex.acquire();
      try {
        const s = this.getSubtitleTrackData(e, r);
        (this.validateTimestamp(s.track, t.timestamp, !0),
          e.source._codec === "webvtt" &&
            (s.cueQueue.push(t), await this.processWebVTTCues(s, t.timestamp)));
      } finally {
        n();
      }
    }
    async processWebVTTCues(e, t) {
      for (; e.cueQueue.length > 0;) {
        e.lastCueEndTimestamp ??= Math.min(0, e.cueQueue[0].timestamp);
        const r = new Set([]);
        for (const l of e.cueQueue)
          (g(l.timestamp <= t),
            g(e.lastCueEndTimestamp <= l.timestamp + l.duration),
            r.add(Math.max(l.timestamp, e.lastCueEndTimestamp)),
            r.add(l.timestamp + l.duration));
        const n = [...r].sort((l, u) => l - u),
          s = n[0],
          a = n[1] ?? s;
        if (t < a) break;
        if (e.lastCueEndTimestamp < s) {
          this.auxWriter.seek(0);
          const l = wh();
          this.auxBoxWriter.writeBox(l);
          const u = this.auxTarget._getSlice(0, this.auxWriter.getPos()),
            d = this.createSampleForTrack(
              e,
              u,
              e.lastCueEndTimestamp,
              s - e.lastCueEndTimestamp,
              "key",
            );
          (await this.registerSample(e, d), (e.lastCueEndTimestamp = s));
        }
        this.auxWriter.seek(0);
        for (let l = 0; l < e.cueQueue.length; l++) {
          const u = e.cueQueue[l];
          if (u.timestamp >= a) break;
          Xr.lastIndex = 0;
          const d = Xr.test(u.text),
            f = u.timestamp + u.duration;
          let h = e.cueToSourceId.get(u);
          if (
            (h === void 0 &&
              a < f &&
              ((h = e.nextSourceId++), e.cueToSourceId.set(u, h)),
            u.notes)
          ) {
            const m = bh(u.notes);
            this.auxBoxWriter.writeBox(m);
          }
          const p = yh(
            u.text,
            d ? s : null,
            u.identifier ?? null,
            u.settings ?? null,
            h ?? null,
          );
          (this.auxBoxWriter.writeBox(p), f === a && e.cueQueue.splice(l--, 1));
        }
        const o = this.auxTarget._getSlice(0, this.auxWriter.getPos()),
          c = this.createSampleForTrack(e, o, s, a - s, "key");
        (await this.registerSample(e, c), (e.lastCueEndTimestamp = a));
      }
    }
    createSampleForTrack(e, t, r, n, s) {
      return {
        timestamp: r,
        decodeTimestamp: r,
        duration: n,
        data: t,
        size: t.byteLength,
        type: s,
        timescaleUnitsToNextSample: te(n, e.timescale),
      };
    }
    processTimestamps(e, t) {
      if (e.timestampProcessingQueue.length === 0) return;
      if (e.type === "audio" && e.info.requiresPcmTransformation) {
        (g(!this.isFragmented),
          (e.startTimestampOffset ??= e.timestampProcessingQueue[0].timestamp));
        let n = 0;
        for (let s = 0; s < e.timestampProcessingQueue.length; s++) {
          const a = e.timestampProcessingQueue[s],
            o = te(a.duration, e.timescale);
          n += o;
        }
        if (e.timeToSampleTable.length === 0)
          e.timeToSampleTable.push({ sampleCount: n, sampleDelta: 1 });
        else {
          const s = ne(e.timeToSampleTable);
          s.sampleCount += n;
        }
        e.timestampProcessingQueue.length = 0;
        return;
      }
      const r = e.timestampProcessingQueue
        .map((n) => n.timestamp)
        .sort((n, s) => n - s);
      this.isFragmented
        ? (e.startTimestampOffset ??= Math.min(r[0], 0))
        : (e.startTimestampOffset ??= r[0]);
      for (let n = 0; n < e.timestampProcessingQueue.length; n++) {
        const s = e.timestampProcessingQueue[n];
        s.decodeTimestamp = r[n];
        const a = te(s.timestamp - s.decodeTimestamp, e.timescale),
          o = te(s.duration, e.timescale);
        if (e.lastTimescaleUnits !== null) {
          g(e.lastSample);
          const c = te(s.decodeTimestamp, e.timescale, !1),
            l = Math.round(c - e.lastTimescaleUnits);
          if (
            (g(l >= 0),
            (e.lastTimescaleUnits += l),
            (e.lastSample.timescaleUnitsToNextSample = l),
            !this.isFragmented)
          ) {
            let u = ne(e.timeToSampleTable);
            if ((g(u), u.sampleCount === 1)) {
              u.sampleDelta = l;
              const f = e.timeToSampleTable[e.timeToSampleTable.length - 2];
              f &&
                f.sampleDelta === l &&
                (f.sampleCount++, e.timeToSampleTable.pop(), (u = f));
            } else
              u.sampleDelta !== l &&
                (u.sampleCount--,
                e.timeToSampleTable.push(
                  (u = { sampleCount: 1, sampleDelta: l }),
                ));
            u.sampleDelta === o
              ? u.sampleCount++
              : e.timeToSampleTable.push({ sampleCount: 1, sampleDelta: o });
            const d = ne(e.compositionTimeOffsetTable);
            (g(d),
              d.sampleCompositionTimeOffset === a
                ? d.sampleCount++
                : e.compositionTimeOffsetTable.push({
                    sampleCount: 1,
                    sampleCompositionTimeOffset: a,
                  }));
          }
        } else
          ((e.lastTimescaleUnits = te(s.decodeTimestamp, e.timescale, !1)),
            this.isFragmented ||
              (e.timeToSampleTable.push({ sampleCount: 1, sampleDelta: o }),
              e.compositionTimeOffsetTable.push({
                sampleCount: 1,
                sampleCompositionTimeOffset: a,
              })));
        e.lastSample = s;
      }
      if (
        ((e.timestampProcessingQueue.length = 0),
        g(e.lastSample),
        g(e.lastTimescaleUnits !== null),
        t !== void 0 && e.lastSample.timescaleUnitsToNextSample === 0)
      ) {
        g(t.type === "key");
        const n = te(t.timestamp, e.timescale, !1),
          s = Math.round(n - e.lastTimescaleUnits);
        e.lastSample.timescaleUnitsToNextSample = s;
      }
    }
    async registerSample(e, t) {
      (t.type === "key" && this.processTimestamps(e, t),
        e.timestampProcessingQueue.push(t),
        this.isFragmented
          ? (e.sampleQueue.push(t), await this.interleaveSamples())
          : this.fastStart === "reserve"
            ? await this.registerSampleFastStartReserve(e, t)
            : await this.addSampleToTrack(e, t));
    }
    async addSampleToTrack(e, t) {
      if (
        !this.isFragmented &&
        (e.samples.push(t), this.fastStart === "reserve")
      ) {
        const n = e.track.metadata.maximumPacketCount;
        if ((g(n !== void 0), e.samples.length > n))
          throw new Error(
            `Track #${e.track.id} has already reached the maximum packet count (${n}). Either add less packets or increase the maximum packet count.`,
          );
      }
      let r = !1;
      if (!e.currentChunk) r = !0;
      else {
        e.currentChunk.startTimestamp = Math.min(
          e.currentChunk.startTimestamp,
          t.timestamp,
        );
        const n = t.timestamp - e.currentChunk.startTimestamp;
        if (this.isFragmented) {
          const s = this.trackDatas.every((a) => {
            if (e === a) return t.type === "key";
            const o = a.sampleQueue[0];
            return o ? o.type === "key" : a.closed;
          });
          n >= this.minimumFragmentDuration &&
            s &&
            t.timestamp > this.maxWrittenTimestamp &&
            ((r = !0), await this.finalizeFragment());
        } else r = n >= 0.5;
      }
      (r &&
        (e.currentChunk && (await this.finalizeCurrentChunk(e)),
        (e.currentChunk = {
          startTimestamp: t.timestamp,
          samples: [],
          offset: null,
          moofOffset: null,
          trafIndex: null,
        })),
        g(e.currentChunk),
        e.currentChunk.samples.push(t),
        this.isFragmented &&
          ((this.maxWrittenTimestamp = Math.max(
            this.maxWrittenTimestamp,
            t.timestamp,
          )),
          (this.maxWrittenEndTimestamp = Math.max(
            this.maxWrittenEndTimestamp,
            t.timestamp + t.duration,
          )),
          (this.minWrittenTimestamp = Math.min(
            this.minWrittenTimestamp,
            t.timestamp,
          ))));
    }
    async finalizeCurrentChunk(e) {
      if ((g(!this.isFragmented), g(this.writer), !e.currentChunk)) return;
      (e.finalizedChunks.push(e.currentChunk),
        this.finalizedChunks.push(e.currentChunk));
      let t = e.currentChunk.samples.length;
      if (
        (e.type === "audio" &&
          e.info.requiresPcmTransformation &&
          (t = e.currentChunk.samples.reduce(
            (r, n) => r + te(n.duration, e.timescale),
            0,
          )),
        (e.compactlyCodedChunkTable.length === 0 ||
          ne(e.compactlyCodedChunkTable).samplesPerChunk !== t) &&
          e.compactlyCodedChunkTable.push({
            firstChunk: e.finalizedChunks.length,
            samplesPerChunk: t,
          }),
        this.fastStart === "in-memory")
      ) {
        e.currentChunk.offset = 0;
        return;
      }
      e.currentChunk.offset = this.writer.getPos();
      for (const r of e.currentChunk.samples)
        (g(r.data), this.writer.write(r.data), (r.data = null));
      await this.writer.flush();
    }
    async interleaveSamples(e = !1) {
      if ((g(this.isFragmented), !(!e && !this.allTracksAreKnown())))
        e: for (;;) {
          let t = null,
            r = 1 / 0;
          for (const s of this.trackDatas) {
            if (!e && s.sampleQueue.length === 0 && !s.closed) break e;
            s.sampleQueue.length > 0 &&
              s.sampleQueue[0].timestamp < r &&
              ((t = s), (r = s.sampleQueue[0].timestamp));
          }
          if (!t) break;
          const n = t.sampleQueue.shift();
          await this.addSampleToTrack(t, n);
        }
    }
    async finalizeFragment(e = !this.isCmaf) {
      if ((g(this.isFragmented), !this.wroteFragmentedHeader)) {
        this.wroteFragmentedHeader = !0;
        const h = this.initBoxWriter ?? this.boxWriter;
        (g(h),
          this.formatOptions.onMoov && h.writer.startTrackingWrites(),
          this.ensureOneEnabledTrack());
        const p = gr(this);
        if ((h.writeBox(p), this.formatOptions.onMoov)) {
          const { data: m, start: y } = h.writer.stopTrackingWrites();
          this.formatOptions.onMoov(m, y);
        }
        if (this.isCmaf) {
          (g(this.initWriter),
            await this.initWriter.flush(),
            await this.initWriter.finalize(),
            (this.writer = await this.output._getRootWriter(!0)),
            (this.boxWriter = new Qr(this.writer)));
          const m = this.boxWriter.measureBox(Nc()),
            y = this.boxWriter.measureBox(Uc(this, 0));
          ((this.segmentHeaderSize = m + y),
            this.writer.seek(this.segmentHeaderSize));
        }
      }
      (g(this.writer), g(this.boxWriter));
      const t = this.trackDatas.filter((h) => h.currentChunk);
      if (t.length === 0) {
        e && (await this.writer.flush());
        return;
      }
      const r = this.nextFragmentNumber++,
        n = Vc(r, t),
        s = this.writer.getPos(),
        a = s + this.boxWriter.measureBox(n);
      let o = a + kt,
        c = 1 / 0;
      for (let h = 0; h < t.length; h++) {
        const p = t[h];
        (g(p.currentChunk),
          g(p.startTimestampOffset !== null),
          (p.currentChunk.offset = o),
          (p.currentChunk.moofOffset = s),
          (p.currentChunk.trafIndex = h),
          (p.currentChunk.startTimestamp -= p.startTimestampOffset));
        for (const m of p.currentChunk.samples)
          ((o += m.size),
            (m.timestamp -= p.startTimestampOffset),
            (m.decodeTimestamp -= p.startTimestampOffset));
        c = Math.min(c, p.currentChunk.startTimestamp);
      }
      const l = o - a,
        u = l >= 2 ** 32;
      if (u) for (const h of t) h.currentChunk.offset += Xt - kt;
      this.formatOptions.onMoof && this.writer.startTrackingWrites();
      const d = Vc(r, t);
      if ((this.boxWriter.writeBox(d), this.formatOptions.onMoof)) {
        const { data: h, start: p } = this.writer.stopTrackingWrites();
        this.formatOptions.onMoof(h, p, c);
      }
      (g(this.writer.getPos() === a),
        this.formatOptions.onMdat && this.writer.startTrackingWrites());
      const f = Yr(u);
      ((f.size = l),
        this.boxWriter.writeBox(f),
        this.writer.seek(a + (u ? Xt : kt)));
      for (const h of t)
        for (const p of h.currentChunk.samples)
          (this.writer.write(p.data), (p.data = null));
      if (this.formatOptions.onMdat) {
        const { data: h, start: p } = this.writer.stopTrackingWrites();
        this.formatOptions.onMdat(h, p);
      }
      for (const h of t)
        (h.finalizedChunks.push(h.currentChunk),
          this.finalizedChunks.push(h.currentChunk),
          (h.currentChunk = null));
      e && (await this.writer.flush());
    }
    async registerSampleFastStartReserve(e, t) {
      this.allTracksAreKnown()
        ? (this.mdat || (await this.createFastStartReserveMdat()),
          await this.addSampleToTrack(e, t))
        : e.sampleQueue.push(t);
    }
    async createFastStartReserveMdat() {
      (g(this.writer), g(this.boxWriter), this.ensureOneEnabledTrack());
      const e = gr(this),
        r =
          this.boxWriter.measureBox(e) +
          this.computeSampleTableSizeUpperBound() +
          4096;
      (g(this.ftypSize !== null),
        this.writer.seek(this.ftypSize + r),
        this.formatOptions.onMdat && this.writer.startTrackingWrites(),
        (this.mdat = Yr(!0)),
        this.boxWriter.writeBox(this.mdat));
      for (const n of this.trackDatas) {
        for (const s of n.sampleQueue) await this.addSampleToTrack(n, s);
        n.sampleQueue.length = 0;
      }
    }
    computeSampleTableSizeUpperBound() {
      g(this.fastStart === "reserve");
      let e = 0;
      for (const t of this.trackDatas) {
        const r = t.track.metadata.maximumPacketCount;
        (g(r !== void 0),
          (e += 8 * Math.ceil((2 / 3) * r)),
          (e += 4 * r),
          (e += 8 * Math.ceil((2 / 3) * r)),
          (e += 12 * Math.ceil((2 / 3) * r)),
          (e += 4 * r),
          (e += 8 * r));
      }
      return e;
    }
    async onTrackClose(e) {
      const t = await this.mutex.acquire(),
        r = this.trackDatas.find((n) => n.track === e);
      (r &&
        ((r.closed = !0),
        r.type === "subtitle" &&
          e.source._codec === "webvtt" &&
          (await this.processWebVTTCues(r, 1 / 0)),
        this.processTimestamps(r)),
        this.allTracksAreKnown() && this.allTracksKnown.resolve(),
        this.isFragmented && (await this.interleaveSamples()),
        t());
    }
    ensureOneEnabledTrack() {
      for (const e of ["video", "audio", "subtitle"]) {
        const t = this.trackDatas.filter((n) => n.type === e);
        if (t.length === 0) continue;
        if (!t.some((n) => n.track.metadata.disposition?.default !== !1)) {
          const n = t[0];
          n.track.metadata.disposition = {
            ...n.track.metadata.disposition,
            default: !0,
          };
        }
      }
    }
    async forceFragmentFinalization() {
      g(this.isFragmented);
      const e = await this.mutex.acquire();
      try {
        for (const t of this.trackDatas)
          (t.type === "subtitle" &&
            t.track.source._codec === "webvtt" &&
            (await this.processWebVTTCues(t, 1 / 0)),
            this.processTimestamps(t));
        (await this.interleaveSamples(!0), await this.finalizeFragment());
      } finally {
        e();
      }
    }
    async finalize() {
      const e = await this.mutex.acquire();
      (this.allTracksKnown.resolve(),
        this.ensureOneEnabledTrack(),
        !this.mdat &&
          this.fastStart === "reserve" &&
          (await this.createFastStartReserveMdat()));
      for (const t of this.trackDatas)
        ((t.closed = !0),
          t.type === "subtitle" &&
            t.track.source._codec === "webvtt" &&
            (await this.processWebVTTCues(t, 1 / 0)),
          this.processTimestamps(t));
      if (this.isFragmented)
        (await this.interleaveSamples(!0), await this.finalizeFragment(!1));
      else
        for (const t of this.trackDatas) {
          await this.finalizeCurrentChunk(t);
          const r = Bi(t);
          if (r > 0) {
            let o = 0;
            for (const c of t.samples) o += c.size;
            t.avgBitrate = Math.round((8 * o) / r);
          } else t.avgBitrate = 0;
          let n = 0,
            s = 0,
            a = 0;
          for (let o = 0; o < t.samples.length; o++) {
            const c = t.samples[o];
            for (
              s += c.size;
              c.decodeTimestamp - t.samples[n].decodeTimestamp >= 1;
            )
              ((s -= t.samples[n].size), n++);
            a = Math.max(a, s);
          }
          if (((t.maxBitrate = 8 * a), t.startTimestampOffset !== null))
            for (let o = 0; o < t.samples.length; o++) {
              const c = t.samples[o];
              ((c.timestamp -= t.startTimestampOffset),
                (c.decodeTimestamp -= t.startTimestampOffset));
            }
        }
      if ((g(this.writer), g(this.boxWriter), this.fastStart === "in-memory")) {
        this.mdat = Yr(!1);
        let t;
        for (let n = 0; n < 2; n++) {
          const s = gr(this),
            a = this.boxWriter.measureBox(s);
          t = this.boxWriter.measureBox(this.mdat);
          let o = this.writer.getPos() + a + t;
          for (const c of this.finalizedChunks) {
            c.offset = o;
            for (const { data: l } of c.samples)
              (g(l), (o += l.byteLength), (t += l.byteLength));
          }
          if (o < 2 ** 32) break;
          t >= 2 ** 32 && (this.mdat.largeSize = !0);
        }
        this.formatOptions.onMoov && this.writer.startTrackingWrites();
        const r = gr(this);
        if ((this.boxWriter.writeBox(r), this.formatOptions.onMoov)) {
          const { data: n, start: s } = this.writer.stopTrackingWrites();
          this.formatOptions.onMoov(n, s);
        }
        (this.formatOptions.onMdat && this.writer.startTrackingWrites(),
          (this.mdat.size = t),
          this.boxWriter.writeBox(this.mdat));
        for (const n of this.finalizedChunks)
          for (const s of n.samples)
            (g(s.data), this.writer.write(s.data), (s.data = null));
        if (this.formatOptions.onMdat) {
          const { data: n, start: s } = this.writer.stopTrackingWrites();
          this.formatOptions.onMdat(n, s);
        }
      } else if (this.isFragmented)
        if (this.isCmaf) {
          const t =
            this.segmentHeaderSize !== null
              ? this.writer.getPos() - this.segmentHeaderSize
              : 0;
          (this.writer.seek(0),
            this.boxWriter.writeBox(Nc()),
            this.boxWriter.writeBox(Uc(this, t)));
        } else {
          const t = this.writer.getPos(),
            r = mh(this.trackDatas);
          this.boxWriter.writeBox(r);
          const n = this.writer.getPos() - t;
          (this.writer.seek(this.writer.getPos() - 4),
            this.boxWriter.writeU32(n));
        }
      else {
        g(this.mdat);
        const t = this.boxWriter.offsets.get(this.mdat);
        g(t !== void 0);
        const r = this.writer.getPos() - t;
        if (
          ((this.mdat.size = r),
          (this.mdat.largeSize = r >= 2 ** 32),
          this.boxWriter.patchBox(this.mdat),
          this.formatOptions.onMdat)
        ) {
          const { data: s, start: a } = this.writer.stopTrackingWrites();
          this.formatOptions.onMdat(s, a);
        }
        const n = gr(this);
        if (this.fastStart === "reserve") {
          (g(this.ftypSize !== null),
            this.writer.seek(this.ftypSize),
            this.formatOptions.onMoov && this.writer.startTrackingWrites(),
            this.boxWriter.writeBox(n));
          const s =
            this.boxWriter.offsets.get(this.mdat) - this.writer.getPos();
          this.boxWriter.writeBox(Sf(s));
        } else
          (this.formatOptions.onMoov && this.writer.startTrackingWrites(),
            this.boxWriter.writeBox(n));
        if (this.formatOptions.onMoov) {
          const { data: s, start: a } = this.writer.stopTrackingWrites();
          this.formatOptions.onMoov(s, a);
        }
      }
      e();
    }
  }
  const Mh = -32768,
    Dh = 2 ** 15 - 1,
    Kc = 6,
    Gc = 5,
    Oh = { video: 1, audio: 2, subtitle: 17 };
  class zh extends Ss {
    constructor(e, t) {
      (super(e),
        (this.trackDatas = []),
        (this.allTracksKnown = ce()),
        (this.segment = null),
        (this.segmentInfo = null),
        (this.seekHead = null),
        (this.tracksElement = null),
        (this.tagsElement = null),
        (this.attachmentsElement = null),
        (this.segmentDuration = null),
        (this.cues = null),
        (this.currentCluster = null),
        (this.currentClusterStartMsTimestamp = null),
        (this.currentClusterMaxMsTimestamp = null),
        (this.trackDatasInCurrentCluster = new Map()),
        (this.startTimestamp = 1 / 0),
        (this.endTimestamp = -1 / 0),
        (this.warnedAboutTooNegativeTimestamp = !1),
        (this.format = t));
    }
    async start() {
      const e = await this.mutex.acquire();
      ((this.writer = await this.output._getRootWriter(
        !!this.format._options.appendOnly,
      )),
        (this.ebmlWriter = new _u(this.writer)),
        this.writeEBMLHeader(),
        this.createSegmentInfo(),
        this.createCues(),
        await this.writer.flush());
      for (const t of this.output.tracks)
        t.isVideoTrack() && t.metadata.decoderConfig
          ? this.getVideoTrackData(t, t.metadata.primingPacket ?? null, {
              decoderConfig: t.metadata.decoderConfig,
            })
          : t.isAudioTrack() &&
            t.metadata.decoderConfig &&
            this.getAudioTrackData(t, t.metadata.primingPacket ?? null, {
              decoderConfig: t.metadata.decoderConfig,
            });
      e();
    }
    writeEBMLHeader() {
      this.format._options.onEbmlHeader && this.writer.startTrackingWrites();
      const e = {
        id: x.EBML,
        data: [
          { id: x.EBMLVersion, data: 1 },
          { id: x.EBMLReadVersion, data: 1 },
          { id: x.EBMLMaxIDLength, data: 4 },
          { id: x.EBMLMaxSizeLength, data: 8 },
          {
            id: x.DocType,
            data: this.format instanceof Jr ? "webm" : "matroska",
          },
          { id: x.DocTypeVersion, data: 4 },
          { id: x.DocTypeReadVersion, data: 2 },
        ],
      };
      if ((this.ebmlWriter.writeEBML(e), this.format._options.onEbmlHeader)) {
        const { data: t, start: r } = this.writer.stopTrackingWrites();
        this.format._options.onEbmlHeader(t, r);
      }
    }
    maybeCreateSeekHead(e) {
      if (this.format._options.appendOnly) return;
      const t = new Uint8Array([28, 83, 187, 107]),
        r = new Uint8Array([21, 73, 169, 102]),
        n = new Uint8Array([22, 84, 174, 107]),
        s = new Uint8Array([25, 65, 164, 105]),
        a = new Uint8Array([18, 84, 195, 103]),
        o = {
          id: x.SeekHead,
          data: [
            {
              id: x.Seek,
              data: [
                { id: x.SeekID, data: r },
                {
                  id: x.SeekPosition,
                  size: 5,
                  data: e
                    ? this.ebmlWriter.offsets.get(this.segmentInfo) -
                      this.segmentDataOffset
                    : 0,
                },
              ],
            },
            {
              id: x.Seek,
              data: [
                { id: x.SeekID, data: n },
                {
                  id: x.SeekPosition,
                  size: 5,
                  data: e
                    ? this.ebmlWriter.offsets.get(this.tracksElement) -
                      this.segmentDataOffset
                    : 0,
                },
              ],
            },
            this.attachmentsElement
              ? {
                  id: x.Seek,
                  data: [
                    { id: x.SeekID, data: s },
                    {
                      id: x.SeekPosition,
                      size: 5,
                      data: e
                        ? this.ebmlWriter.offsets.get(this.attachmentsElement) -
                          this.segmentDataOffset
                        : 0,
                    },
                  ],
                }
              : null,
            this.tagsElement
              ? {
                  id: x.Seek,
                  data: [
                    { id: x.SeekID, data: a },
                    {
                      id: x.SeekPosition,
                      size: 5,
                      data: e
                        ? this.ebmlWriter.offsets.get(this.tagsElement) -
                          this.segmentDataOffset
                        : 0,
                    },
                  ],
                }
              : null,
            {
              id: x.Seek,
              data: [
                { id: x.SeekID, data: t },
                {
                  id: x.SeekPosition,
                  size: 5,
                  data: e
                    ? this.ebmlWriter.offsets.get(this.cues) -
                      this.segmentDataOffset
                    : 0,
                },
              ],
            },
          ],
        };
      this.seekHead = o;
    }
    createSegmentInfo() {
      const e = { id: x.Duration, data: new Yn(0) };
      this.segmentDuration = e;
      const t = {
        id: x.Info,
        data: [
          { id: x.TimestampScale, data: 1e6 },
          { id: x.MuxingApp, data: Hi },
          { id: x.WritingApp, data: Hi },
          this.format._options.appendOnly ? null : e,
        ],
      };
      this.segmentInfo = t;
    }
    createTracks() {
      const e = { id: x.Tracks, data: [] };
      this.tracksElement = e;
      for (const t of this.trackDatas) {
        let r = Re[t.track.source._codec];
        (g(r),
          t.type === "audio" &&
            t.track.source._codec === "dts" &&
            (t.info.decoderConfig.codec === "dtse"
              ? (r = "A_DTS/EXPRESS")
              : t.info.decoderConfig.codec === "dtsl" &&
                (r = "A_DTS/LOSSLESS")));
        let n = 0;
        if (t.type === "audio" && t.track.source._codec === "opus") {
          n = 1e6 * 80;
          const s = t.info.decoderConfig.description;
          if (s) {
            const a = me(s),
              o = On(a);
            n = Math.round(1e9 * (o.preSkip / Gi));
          }
        }
        e.data.push({
          id: x.TrackEntry,
          data: [
            { id: x.TrackNumber, data: t.track.id },
            { id: x.TrackUID, data: t.track.id },
            { id: x.TrackType, data: Oh[t.type] },
            t.track.metadata.disposition?.default === !1
              ? { id: x.FlagDefault, data: 0 }
              : null,
            t.track.metadata.disposition?.forced
              ? { id: x.FlagForced, data: 1 }
              : null,
            t.track.metadata.disposition?.hearingImpaired
              ? { id: x.FlagHearingImpaired, data: 1 }
              : null,
            t.track.metadata.disposition?.visuallyImpaired
              ? { id: x.FlagVisualImpaired, data: 1 }
              : null,
            t.track.metadata.disposition?.original
              ? { id: x.FlagOriginal, data: 1 }
              : null,
            t.track.metadata.disposition?.commentary
              ? { id: x.FlagCommentary, data: 1 }
              : null,
            { id: x.FlagLacing, data: 0 },
            { id: x.Language, data: t.track.metadata.languageCode ?? Pe },
            { id: x.CodecID, data: r },
            t.codecPrivate
              ? { id: x.CodecPrivate, data: me(t.codecPrivate) }
              : null,
            n > 0 ? { id: x.SeekPreRoll, data: n } : null,
            t.track.metadata.name !== void 0
              ? { id: x.Name, data: new Mt(t.track.metadata.name) }
              : null,
            t.type === "video" ? this.videoSpecificTrackInfo(t) : null,
            t.type === "audio" ? this.audioSpecificTrackInfo(t) : null,
            t.type === "subtitle" ? this.subtitleSpecificTrackInfo(t) : null,
          ],
        });
      }
    }
    videoSpecificTrackInfo(e) {
      const { frameRate: t, transformationMatrix: r } = e.track.metadata,
        n = [t ? { id: x.DefaultDuration, data: 1e9 / t } : null];
      let s, a, o;
      if (r) {
        s = Ri(r);
        const p = ai(ln(-s), r);
        ((a = Math.sign(p[0])),
          (o = Math.sign(p[4])),
          o === -1 && ((o = 1), (a = -a), (s = Lt(s + 180))));
      } else
        ((s = e.track.metadata.rotation ?? 0),
          (a = e.track.metadata.flip ? -1 : 1),
          (o = 1));
      const c = s ? Lt(a === -1 ? s : -s) : 0,
        l = Math.round(Math.acos(a) * cn),
        u = Math.round(Math.acos(o) * cn),
        d =
          !!e.info.aspectRatio &&
          e.info.aspectRatio.num * e.info.height !==
            e.info.aspectRatio.den * e.info.width,
        f = e.info.decoderConfig.colorSpace,
        h = {
          id: x.Video,
          data: [
            { id: x.PixelWidth, data: e.info.width },
            { id: x.PixelHeight, data: e.info.height },
            d ? { id: x.DisplayWidth, data: e.info.aspectRatio.num } : null,
            d ? { id: x.DisplayHeight, data: e.info.aspectRatio.den } : null,
            d ? { id: x.DisplayUnit, data: 3 } : null,
            e.info.alphaMode ? { id: x.AlphaMode, data: 1 } : null,
            Xs(f)
              ? null
              : {
                  id: x.Colour,
                  data: [
                    {
                      id: x.MatrixCoefficients,
                      data: f?.matrix != null ? jt[f.matrix] : 2,
                    },
                    {
                      id: x.TransferCharacteristics,
                      data: f?.transfer != null ? qt[f.transfer] : 2,
                    },
                    {
                      id: x.Primaries,
                      data: f?.primaries != null ? Ht[f.primaries] : 2,
                    },
                    {
                      id: x.Range,
                      data: f?.fullRange != null ? (f.fullRange ? 2 : 1) : 0,
                    },
                  ],
                },
            c || l || u
              ? {
                  id: x.Projection,
                  data: [
                    { id: x.ProjectionType, data: 0 },
                    l ? { id: x.ProjectionPoseYaw, data: new Yi(l) } : null,
                    u ? { id: x.ProjectionPosePitch, data: new Yi(u) } : null,
                    c
                      ? {
                          id: x.ProjectionPoseRoll,
                          data: new Yi(((c + 180) % 360) - 180),
                        }
                      : null,
                  ],
                }
              : null,
          ],
        };
      return (n.push(h), n);
    }
    audioSpecificTrackInfo(e) {
      const t = pe.includes(e.track.source._codec)
        ? Ge(e.track.source._codec)
        : null;
      return [
        {
          id: x.Audio,
          data: [
            { id: x.SamplingFrequency, data: new Yi(e.info.sampleRate) },
            { id: x.Channels, data: e.info.numberOfChannels },
            t ? { id: x.BitDepth, data: 8 * t.sampleSize } : null,
          ],
        },
      ];
    }
    subtitleSpecificTrackInfo(e) {
      return [];
    }
    maybeCreateTags() {
      const e = [],
        t = (s, a) => {
          e.push({
            id: x.SimpleTag,
            data: [
              { id: x.TagName, data: new Mt(s) },
              typeof a == "string"
                ? { id: x.TagString, data: new Mt(a) }
                : { id: x.TagBinary, data: a },
            ],
          });
        },
        r = this.output._metadataTags,
        n = new Set();
      for (const { key: s, value: a } of Wi(r))
        switch (s) {
          case "title":
            (t("TITLE", a), n.add("TITLE"));
            break;
          case "description":
            (t("DESCRIPTION", a), n.add("DESCRIPTION"));
            break;
          case "artist":
            (t("ARTIST", a), n.add("ARTIST"));
            break;
          case "album":
            (t("ALBUM", a), n.add("ALBUM"));
            break;
          case "albumArtist":
            (t("ALBUM_ARTIST", a), n.add("ALBUM_ARTIST"));
            break;
          case "genre":
            (t("GENRE", a), n.add("GENRE"));
            break;
          case "beatsPerMinute":
            (t("BPM", a.toString()), n.add("BPM"));
            break;
          case "comment":
            (t("COMMENT", a), n.add("COMMENT"));
            break;
          case "lyrics":
            (t("LYRICS", a), n.add("LYRICS"));
            break;
          case "date":
            (t("DATE", a.toISOString().slice(0, 10)), n.add("DATE"));
            break;
          case "trackNumber":
            {
              const o =
                r.tracksTotal !== void 0
                  ? `${a}/${r.tracksTotal}`
                  : a.toString();
              (t("PART_NUMBER", o), n.add("PART_NUMBER"));
            }
            break;
          case "discNumber":
            {
              const o =
                r.discsTotal !== void 0 ? `${a}/${r.discsTotal}` : a.toString();
              (t("DISC", o), n.add("DISC"));
            }
            break;
          case "tracksTotal":
          case "discsTotal":
            break;
          case "images":
          case "raw":
            break;
          default:
            xe(s);
        }
      if (r.raw)
        for (const s in r.raw) {
          const a = r.raw[s];
          a == null ||
            n.has(s) ||
            ((typeof a == "string" || a instanceof Uint8Array) && t(s, a));
        }
      e.length !== 0 &&
        (this.tagsElement = {
          id: x.Tags,
          data: [
            {
              id: x.Tag,
              data: [
                {
                  id: x.Targets,
                  data: [
                    { id: x.TargetTypeValue, data: 50 },
                    { id: x.TargetType, data: "MOVIE" },
                  ],
                },
                ...e,
              ],
            },
          ],
        });
    }
    maybeCreateAttachments() {
      const e = this.output._metadataTags,
        t = [],
        r = new Set(),
        n = e.images ?? [];
      for (const s of n) {
        let a = s.name;
        a === void 0 &&
          (a =
            (s.kind === "coverFront"
              ? "cover"
              : s.kind === "coverBack"
                ? "back"
                : "image") + (yl(s.mimeType) ?? ""));
        let o;
        for (;;) {
          o = 0n;
          for (let c = 0; c < 8; c++)
            ((o <<= 8n), (o |= BigInt(Math.floor(Math.random() * 256))));
          if (o !== 0n && !r.has(o)) break;
        }
        (r.add(o),
          t.push({
            id: x.AttachedFile,
            data: [
              s.description !== void 0
                ? { id: x.FileDescription, data: new Mt(s.description) }
                : null,
              { id: x.FileName, data: new Mt(a) },
              { id: x.FileMediaType, data: s.mimeType },
              { id: x.FileData, data: s.data },
              { id: x.FileUID, data: o },
            ],
          }));
      }
      for (const [s, a] of Object.entries(e.raw ?? {}))
        !(a instanceof Cn) ||
          !/^\d+$/.test(s) ||
          n.find((c) => c.mimeType === a.mimeType && na(c.data, a.data)) ||
          t.push({
            id: x.AttachedFile,
            data: [
              a.description !== void 0
                ? { id: x.FileDescription, data: new Mt(a.description) }
                : null,
              { id: x.FileName, data: new Mt(a.name ?? "") },
              { id: x.FileMediaType, data: a.mimeType ?? "" },
              { id: x.FileData, data: a.data },
              { id: x.FileUID, data: BigInt(s) },
            ],
          });
      t.length !== 0 &&
        (this.attachmentsElement = { id: x.Attachments, data: t });
    }
    createSegment() {
      (this.createTracks(),
        this.maybeCreateTags(),
        this.maybeCreateAttachments(),
        this.maybeCreateSeekHead(!1));
      const e = {
        id: x.Segment,
        size: this.format._options.appendOnly ? -1 : Kc,
        data: [
          this.seekHead,
          this.segmentInfo,
          this.tracksElement,
          this.attachmentsElement,
          this.tagsElement,
        ],
      };
      if (
        ((this.segment = e),
        this.format._options.onSegmentHeader &&
          this.writer.startTrackingWrites(),
        this.ebmlWriter.writeEBML(e),
        this.format._options.onSegmentHeader)
      ) {
        const { data: t, start: r } = this.writer.stopTrackingWrites();
        this.format._options.onSegmentHeader(t, r);
      }
    }
    createCues() {
      this.cues = { id: x.Cues, data: [] };
    }
    get segmentDataOffset() {
      return (g(this.segment), this.ebmlWriter.dataOffsets.get(this.segment));
    }
    allTracksAreKnown() {
      for (const e of this.output.tracks)
        if (!e.source._closed && !this.trackDatas.some((t) => t.track === e))
          return !1;
      return !0;
    }
    async getMimeType() {
      await this.allTracksKnown.promise;
      const e = this.trackDatas.map((t) =>
        t.type === "video" || t.type === "audio"
          ? t.info.decoderConfig.codec
          : { webvtt: "wvtt" }[t.track.source._codec],
      );
      return mo({
        isWebM: this.format instanceof Jr,
        hasVideo: this.trackDatas.some((t) => t.type === "video"),
        hasAudio: this.trackDatas.some((t) => t.type === "audio"),
        codecStrings: e,
      });
    }
    getVideoTrackData(e, t, r) {
      const n = this.trackDatas.find((l) => l.track === e);
      if (n) return n;
      (Hn(r, e.source._codec),
        g(r),
        g(r.decoderConfig),
        g(r.decoderConfig.codedWidth !== void 0),
        g(r.decoderConfig.codedHeight !== void 0));
      const s = r.decoderConfig.displayAspectWidth,
        a = r.decoderConfig.displayAspectHeight,
        o = s === void 0 || a === void 0 ? null : di({ num: s, den: a }),
        c = {
          track: e,
          type: "video",
          info: {
            width: r.decoderConfig.codedWidth,
            height: r.decoderConfig.codedHeight,
            aspectRatio: o,
            decoderConfig: r.decoderConfig,
            alphaMode:
              e.metadata.canBeTransparent || (t ? !!t.sideData.alpha : null),
          },
          chunkQueue: [],
          lastWrittenMsTimestamp: null,
          codecPrivate: r.decoderConfig.description ?? null,
          closed: !1,
        };
      return (
        e.source._codec === "vp9"
          ? (c.codecPrivate = new Uint8Array(ru(c.info.decoderConfig.codec)))
          : e.source._codec === "av1"
            ? (c.codecPrivate = new Uint8Array(Oa(c.info.decoderConfig.codec)))
            : e.source._codec === "prores" &&
              (c.codecPrivate = oe.encode(r.decoderConfig.codec)),
        this.trackDatas.push(c),
        this.trackDatas.sort((l, u) => l.track.id - u.track.id),
        this.allTracksAreKnown() && this.allTracksKnown.resolve(),
        c
      );
    }
    getAudioTrackData(e, t, r) {
      const n = this.trackDatas.find((c) => c.track === e);
      if (n) return n;
      (Xi(r, e.source._codec), g(r), g(r.decoderConfig));
      const s = { ...r.decoderConfig };
      let a = !1;
      if (e.source._codec === "aac" && !s.description) {
        if (!t)
          throw new Error(
            "No AAC description provided; you must therefore provide a priming packet.",
          );
        const c = Pt(Me.tempFromBytes(t.data));
        if (!c)
          throw new Error(
            "Couldn't parse ADTS header from the AAC packet. Make sure the packets are in ADTS format (as specified in ISO 13818-7) when not providing a description, or provide a description (must be an AudioSpecificConfig as specified in ISO 14496-3) and ensure the packets are raw AAC data.",
          );
        const l = At[c.samplingFrequencyIndex],
          u = mi[c.channelConfiguration];
        if (l === void 0 || u === void 0)
          throw new Error("Invalid ADTS frame header.");
        ((s.description = An({
          objectType: c.objectType,
          outputSampleRate: l,
          outputNumberOfChannels: u,
        })),
          (a = !0));
      }
      const o = {
        track: e,
        type: "audio",
        info: {
          numberOfChannels: r.decoderConfig.numberOfChannels,
          sampleRate: r.decoderConfig.sampleRate,
          decoderConfig: s,
          requiresAdtsStripping: a,
        },
        chunkQueue: [],
        lastWrittenMsTimestamp: null,
        codecPrivate: s.description ?? null,
        closed: !1,
      };
      return (
        this.trackDatas.push(o),
        this.trackDatas.sort((c, l) => c.track.id - l.track.id),
        this.allTracksAreKnown() && this.allTracksKnown.resolve(),
        o
      );
    }
    getSubtitleTrackData(e, t) {
      const r = this.trackDatas.find((s) => s.track === e);
      if (r) return r;
      (Ua(t), g(t), g(t.config));
      const n = {
        track: e,
        type: "subtitle",
        info: { config: t.config },
        chunkQueue: [],
        lastWrittenMsTimestamp: null,
        codecPrivate: oe.encode(t.config.description),
        closed: !1,
      };
      return (
        this.trackDatas.push(n),
        this.trackDatas.sort((s, a) => s.track.id - a.track.id),
        this.allTracksAreKnown() && this.allTracksKnown.resolve(),
        n
      );
    }
    async addEncodedVideoPacket(e, t, r) {
      const n = await this.mutex.acquire();
      try {
        const s = this.getVideoTrackData(e, t, r);
        s.info.alphaMode ??= !!t.sideData.alpha;
        let a = t.data;
        if (e.source._codec === "prores") {
          if (a.byteLength < 8)
            throw new Error(
              "ProRes packet too small, expected at least 8 bytes.",
            );
          a = a.subarray(8);
        }
        const o = t.type === "key";
        this.validateTimestamp(s.track, t.timestamp, o);
        let c = t.timestamp,
          l = t.duration;
        e.metadata.frameRate !== void 0 &&
          ((c = li(c, e.metadata.frameRate)),
          (l = li(l, e.metadata.frameRate)));
        const u = s.info.alphaMode ? (t.sideData.alpha ?? null) : null,
          d = this.createInternalChunk(a, c, l, t.type, u);
        (e.source._codec === "vp9" && this.fixVP9ColorSpace(s, d),
          s.chunkQueue.push(d),
          await this.interleaveChunks());
      } finally {
        n();
      }
    }
    async addEncodedAudioPacket(e, t, r) {
      const n = await this.mutex.acquire();
      try {
        const s = this.getAudioTrackData(e, t, r);
        let a = t.data;
        if (s.info.requiresAdtsStripping) {
          const l = Pt(Me.tempFromBytes(a));
          if (!l) throw new Error("Expected ADTS frame, didn't get one.");
          const u = l.crcCheck === null ? nr : Ot;
          a = a.subarray(u);
        }
        const o = t.type === "key";
        this.validateTimestamp(s.track, t.timestamp, o);
        const c = this.createInternalChunk(a, t.timestamp, t.duration, t.type);
        (s.chunkQueue.push(c), await this.interleaveChunks());
      } finally {
        n();
      }
    }
    async addSubtitleCue(e, t, r) {
      const n = await this.mutex.acquire();
      try {
        const s = this.getSubtitleTrackData(e, r);
        this.validateTimestamp(s.track, t.timestamp, !0);
        let a = t.text;
        const o = Math.round(t.timestamp * 1e3);
        ((Xr.lastIndex = 0),
          (a = a.replace(Xr, (d) => {
            const h = kf(d.slice(1, -1)) - o;
            return `<${Rc(h)}>`;
          })));
        const c = oe.encode(a),
          l = `${t.settings ?? ""}
${t.identifier ?? ""}
${t.notes ?? ""}`,
          u = this.createInternalChunk(
            c,
            t.timestamp,
            t.duration,
            "key",
            l.trim() ? oe.encode(l) : null,
          );
        (s.chunkQueue.push(u), await this.interleaveChunks());
      } finally {
        n();
      }
    }
    async interleaveChunks(e = !1) {
      if (!(!e && !this.allTracksAreKnown())) {
        e: for (;;) {
          let t = null,
            r = 1 / 0;
          for (const s of this.trackDatas) {
            if (!e && s.chunkQueue.length === 0 && !s.closed) break e;
            s.chunkQueue.length > 0 &&
              s.chunkQueue[0].timestamp < r &&
              ((t = s), (r = s.chunkQueue[0].timestamp));
          }
          if (!t) break;
          const n = t.chunkQueue.shift();
          this.writeBlock(t, n);
        }
        e || (await this.writer.flush());
      }
    }
    fixVP9ColorSpace(e, t) {
      if (
        t.type !== "key" ||
        !e.info.decoderConfig.colorSpace ||
        !e.info.decoderConfig.colorSpace.matrix
      )
        return;
      const r = new $(t.data);
      r.skipBits(2);
      const n = r.readBits(1),
        a = (r.readBits(1) << 1) + n;
      if (
        (a === 3 && r.skipBits(1),
        r.readBits(1) ||
          r.readBits(1) !== 0 ||
          (r.skipBits(2), r.readBits(24) !== 4817730))
      )
        return;
      a >= 2 && r.skipBits(1);
      const u = { rgb: 7, bt709: 2, bt470bg: 1, smpte170m: 3 }[
        e.info.decoderConfig.colorSpace.matrix
      ];
      nl(t.data, r.pos, r.pos + 3, u);
    }
    createInternalChunk(e, t, r, n, s = null) {
      return { data: e, type: n, timestamp: t, duration: r, additions: s };
    }
    writeBlock(e, t) {
      this.segment || this.createSegment();
      const r = Math.round(1e3 * t.timestamp),
        n = this.trackDatas.every((d) => {
          if (e === d) return t.type === "key";
          const f = d.chunkQueue[0];
          return f ? f.type === "key" : d.closed;
        });
      let s = !1;
      if (!this.currentCluster) s = !0;
      else {
        (g(this.currentClusterStartMsTimestamp !== null),
          g(this.currentClusterMaxMsTimestamp !== null));
        const d = r - this.currentClusterStartMsTimestamp;
        s =
          (n &&
            r > this.currentClusterMaxMsTimestamp &&
            d >= 1e3 * (this.format._options.minimumClusterDuration ?? 1)) ||
          d > Dh;
      }
      s && this.createNewCluster(r);
      const a = r - this.currentClusterStartMsTimestamp;
      if (a < Mh) {
        if (!this.warnedAboutTooNegativeTimestamp) {
          const d = this.format instanceof Jr ? "WebM" : "Matroska";
          (L._warn(
            `Packets had to be discarded because their timestamp is too negative to represent in ${d}.`,
          ),
            (this.warnedAboutTooNegativeTimestamp = !0));
        }
        return;
      }
      const o = new Uint8Array(4),
        c = new DataView(o.buffer);
      (c.setUint8(0, 128 | e.track.id), c.setInt16(1, a, !1));
      const l = Math.round(1e3 * t.duration);
      if (!!t.additions || e.type === "subtitle") {
        const d = {
          id: x.BlockGroup,
          data: [
            { id: x.Block, data: [o, t.data] },
            t.type === "delta"
              ? {
                  id: x.ReferenceBlock,
                  data: new ao(e.lastWrittenMsTimestamp - r),
                }
              : null,
            t.additions
              ? {
                  id: x.BlockAdditions,
                  data: [
                    {
                      id: x.BlockMore,
                      data: [
                        { id: x.BlockAddID, data: 1 },
                        { id: x.BlockAdditional, data: t.additions },
                      ],
                    },
                  ],
                }
              : null,
            l > 0 ? { id: x.BlockDuration, data: l } : null,
          ],
        };
        this.ebmlWriter.writeEBML(d);
      } else {
        c.setUint8(3, +(t.type === "key") << 7);
        const d = { id: x.SimpleBlock, data: [o, t.data] };
        this.ebmlWriter.writeEBML(d);
      }
      ((this.startTimestamp = Math.min(this.startTimestamp, r)),
        (this.endTimestamp = Math.max(this.endTimestamp, r + l)),
        (e.lastWrittenMsTimestamp = r),
        this.trackDatasInCurrentCluster.has(e) ||
          this.trackDatasInCurrentCluster.set(e, { firstMsTimestamp: r }),
        (this.currentClusterMaxMsTimestamp = Math.max(
          this.currentClusterMaxMsTimestamp,
          r,
        )));
    }
    createNewCluster(e) {
      ((e = Math.max(0, e)),
        this.currentCluster && this.finalizeCurrentCluster(),
        this.format._options.onCluster && this.writer.startTrackingWrites(),
        (this.currentCluster = {
          id: x.Cluster,
          size: this.format._options.appendOnly ? -1 : Gc,
          data: [{ id: x.Timestamp, data: e }],
        }),
        this.ebmlWriter.writeEBML(this.currentCluster),
        (this.currentClusterStartMsTimestamp = e),
        (this.currentClusterMaxMsTimestamp = e),
        this.trackDatasInCurrentCluster.clear());
    }
    finalizeCurrentCluster() {
      if ((g(this.currentCluster), !this.format._options.appendOnly)) {
        const n =
            this.writer.getPos() -
            this.ebmlWriter.dataOffsets.get(this.currentCluster),
          s = this.writer.getPos();
        (this.writer.seek(this.ebmlWriter.offsets.get(this.currentCluster) + 4),
          this.ebmlWriter.writeVarInt(n, Gc),
          this.writer.seek(s));
      }
      if (this.format._options.onCluster) {
        g(this.currentClusterStartMsTimestamp !== null);
        const { data: n, start: s } = this.writer.stopTrackingWrites();
        this.format._options.onCluster(
          n,
          s,
          this.currentClusterStartMsTimestamp / 1e3,
        );
      }
      const e =
          this.ebmlWriter.offsets.get(this.currentCluster) -
          this.segmentDataOffset,
        t = new Map();
      for (const [n, { firstMsTimestamp: s }] of this
        .trackDatasInCurrentCluster)
        (t.has(s) || t.set(s, []), t.get(s).push(n));
      const r = [...t.entries()].sort((n, s) => n[0] - s[0]);
      for (const [n, s] of r)
        (g(this.cues),
          this.cues.data.push({
            id: x.CuePoint,
            data: [
              { id: x.CueTime, data: Math.max(0, n) },
              ...s.map((a) => ({
                id: x.CueTrackPositions,
                data: [
                  { id: x.CueTrack, data: a.track.id },
                  { id: x.CueClusterPosition, data: e },
                ],
              })),
            ],
          }));
    }
    async onTrackClose(e) {
      const t = await this.mutex.acquire(),
        r = this.trackDatas.find((n) => n.track === e);
      (r && (r.closed = !0),
        this.allTracksAreKnown() && this.allTracksKnown.resolve(),
        await this.interleaveChunks(),
        t());
    }
    async finalize() {
      const e = await this.mutex.acquire();
      this.allTracksKnown.resolve();
      for (const t of this.trackDatas) t.closed = !0;
      if (
        (this.segment || this.createSegment(),
        await this.interleaveChunks(!0),
        this.currentCluster && this.finalizeCurrentCluster(),
        g(this.cues),
        this.ebmlWriter.writeEBML(this.cues),
        !this.format._options.appendOnly)
      ) {
        const t = this.writer.getPos() - this.segmentDataOffset;
        (this.writer.seek(this.ebmlWriter.offsets.get(this.segment) + 4),
          this.ebmlWriter.writeVarInt(t, Kc));
        const r = this.startTimestamp === 1 / 0 ? 0 : this.endTimestamp;
        ((this.segmentDuration.data = new Yn(r)),
          this.writer.seek(this.ebmlWriter.offsets.get(this.segmentDuration)),
          this.ebmlWriter.writeEBML(this.segmentDuration),
          g(this.seekHead),
          this.writer.seek(this.ebmlWriter.offsets.get(this.seekHead)),
          this.maybeCreateSeekHead(!0),
          this.ebmlWriter.writeEBML(this.seekHead));
      }
      e();
    }
  }
  class Nh {
    constructor(e) {
      ((this.writer = e),
        (this.helper = new Uint8Array(8)),
        (this.helperView = new DataView(this.helper.buffer)));
    }
    writeU16(e) {
      (this.helperView.setUint16(0, e, !0),
        this.writer.write(this.helper.subarray(0, 2)));
    }
    writeU32(e) {
      (this.helperView.setUint32(0, e, !0),
        this.writer.write(this.helper.subarray(0, 4)));
    }
    writeU64(e) {
      (this.helperView.setUint32(0, e, !0),
        this.helperView.setUint32(4, Math.floor(e / 2 ** 32), !0),
        this.writer.write(this.helper));
    }
    writeAscii(e) {
      this.writer.write(oe.encode(e));
    }
  }
  class Uh extends Ss {
    constructor(e, t) {
      (super(e),
        (this.headerWritten = !1),
        (this.dataSize = 0),
        (this.sampleRate = null),
        (this.sampleCount = 0),
        (this.riffSizePos = null),
        (this.dataSizePos = null),
        (this.ds64RiffSizePos = null),
        (this.ds64DataSizePos = null),
        (this.ds64SampleCountPos = null),
        (this.format = t),
        (this.isRf64 = !!t._options.large));
    }
    async start() {
      const e = await this.mutex.acquire();
      ((this.writer = await this.output._getRootWriter(!1)),
        (this.riffWriter = new Nh(this.writer)));
      const t = this.output.tracks[0];
      (g(t?.isAudioTrack()),
        t.metadata.decoderConfig &&
          (Xi({ decoderConfig: t.metadata.decoderConfig }, t.source._codec),
          this.writeHeader(t, t.metadata.decoderConfig),
          (this.sampleRate = t.metadata.decoderConfig.sampleRate),
          (this.headerWritten = !0)),
        e());
    }
    async getMimeType() {
      return "audio/wav";
    }
    async addEncodedVideoPacket() {
      throw new Error("WAVE does not support video.");
    }
    async addEncodedAudioPacket(e, t, r) {
      const n = await this.mutex.acquire();
      try {
        if (
          (this.headerWritten ||
            (Xi(r, e.source._codec),
            g(r),
            g(r.decoderConfig),
            this.writeHeader(e, r.decoderConfig),
            (this.sampleRate = r.decoderConfig.sampleRate),
            (this.headerWritten = !0)),
          this.validateTimestamp(e, t.timestamp, t.type === "key"),
          !this.isRf64 && this.writer.getPos() + t.data.byteLength >= 2 ** 32)
        )
          throw new Error(
            "Adding more audio data would exceed the maximum RIFF size of 4 GiB. To write larger files, use RF64 by setting `large: true` in the WavOutputFormatOptions.",
          );
        (this.writer.write(t.data),
          (this.dataSize += t.data.byteLength),
          (this.sampleCount += Math.round(t.duration * this.sampleRate)),
          await this.writer.flush());
      } finally {
        n();
      }
    }
    async addSubtitleCue() {
      throw new Error("WAVE does not support subtitles.");
    }
    writeHeader(e, t) {
      this.format._options.onHeader && this.writer.startTrackingWrites();
      let r;
      const n = e.source._codec,
        s = Ge(n);
      s.dataType === "ulaw"
        ? (r = ge.MULAW)
        : s.dataType === "alaw"
          ? (r = ge.ALAW)
          : s.dataType === "float"
            ? (r = ge.IEEE_FLOAT)
            : (r = ge.PCM);
      const a = t.numberOfChannels,
        o = t.sampleRate,
        c = s.sampleSize * a;
      if (
        (this.riffWriter.writeAscii(this.isRf64 ? "RF64" : "RIFF"),
        this.isRf64
          ? this.riffWriter.writeU32(4294967295)
          : ((this.riffSizePos = this.writer.getPos()),
            this.riffWriter.writeU32(0)),
        this.riffWriter.writeAscii("WAVE"),
        this.isRf64 &&
          (this.riffWriter.writeAscii("ds64"),
          this.riffWriter.writeU32(28),
          (this.ds64RiffSizePos = this.writer.getPos()),
          this.riffWriter.writeU64(0),
          (this.ds64DataSizePos = this.writer.getPos()),
          this.riffWriter.writeU64(0),
          (this.ds64SampleCountPos = this.writer.getPos()),
          this.riffWriter.writeU64(0),
          this.riffWriter.writeU32(0)),
        this.riffWriter.writeAscii("fmt "),
        this.riffWriter.writeU32(16),
        this.riffWriter.writeU16(r),
        this.riffWriter.writeU16(a),
        this.riffWriter.writeU32(o),
        this.riffWriter.writeU32(o * c),
        this.riffWriter.writeU16(c),
        this.riffWriter.writeU16(8 * s.sampleSize),
        !kl(this.output._metadataTags))
      ) {
        const l = this.format._options.metadataFormat ?? "info";
        l === "info"
          ? this.writeInfoChunk(this.output._metadataTags)
          : l === "id3"
            ? this.writeId3Chunk(this.output._metadataTags)
            : xe(l);
      }
      if (
        (this.riffWriter.writeAscii("data"),
        this.isRf64
          ? this.riffWriter.writeU32(4294967295)
          : ((this.dataSizePos = this.writer.getPos()),
            this.riffWriter.writeU32(0)),
        this.format._options.onHeader)
      ) {
        const { data: l, start: u } = this.writer.stopTrackingWrites();
        this.format._options.onHeader(l, u);
      }
    }
    writeInfoChunk(e) {
      const t = this.writer.getPos();
      (this.riffWriter.writeAscii("LIST"),
        this.riffWriter.writeU32(0),
        this.riffWriter.writeAscii("INFO"));
      const r = new Set(),
        n = (o, c) => {
          if (!ft(c)) {
            L._warn(
              `Didn't write tag '${o}' because '${c}' is not ISO 8859-1-compatible.`,
            );
            return;
          }
          const l = c.length + 1,
            u = new Uint8Array(l);
          for (let d = 0; d < c.length; d++) u[d] = c.charCodeAt(d);
          (this.riffWriter.writeAscii(o),
            this.riffWriter.writeU32(l),
            this.writer.write(u),
            l & 1 && this.writer.write(new Uint8Array(1)),
            r.add(o));
        };
      for (const { key: o, value: c } of Wi(e))
        switch (o) {
          case "title":
            (n("INAM", c), r.add("INAM"));
            break;
          case "artist":
            (n("IART", c), r.add("IART"));
            break;
          case "album":
            (n("IPRD", c), r.add("IPRD"));
            break;
          case "trackNumber":
            {
              const l =
                e.tracksTotal !== void 0
                  ? `${c}/${e.tracksTotal}`
                  : c.toString();
              (n("ITRK", l), r.add("ITRK"));
            }
            break;
          case "genre":
            (n("IGNR", c), r.add("IGNR"));
            break;
          case "date":
            (n("ICRD", c.toISOString().slice(0, 10)), r.add("ICRD"));
            break;
          case "comment":
            (n("ICMT", c), r.add("ICMT"));
            break;
          case "albumArtist":
          case "discNumber":
          case "tracksTotal":
          case "discsTotal":
          case "beatsPerMinute":
          case "description":
          case "lyrics":
          case "images":
            break;
          case "raw":
            break;
          default:
            xe(o);
        }
      if (e.raw)
        for (const o in e.raw) {
          const c = e.raw[o];
          c == null ||
            o.length !== 4 ||
            r.has(o) ||
            (typeof c == "string" && n(o, c));
        }
      const s = this.writer.getPos(),
        a = s - t - 8;
      (this.writer.seek(t + 4),
        this.riffWriter.writeU32(a),
        this.writer.seek(s),
        a & 1 && this.writer.write(new Uint8Array(1)));
    }
    writeId3Chunk(e) {
      const t = this.writer.getPos();
      (this.riffWriter.writeAscii("ID3 "), this.riffWriter.writeU32(0));
      const n = new yf(this.writer).writeId3V2Tag(e),
        s = this.writer.getPos();
      (this.writer.seek(t + 4),
        this.riffWriter.writeU32(n),
        this.writer.seek(s),
        n & 1 && this.writer.write(new Uint8Array(1)));
    }
    async finalize() {
      const e = await this.mutex.acquire();
      if (!this.headerWritten)
        throw new Error(
          "Cannot finalize an empty WAVE file: no packets were added and the track specified no decoderConfig in its metadata, so there's no telling what the file should look like.",
        );
      const t = this.writer.getPos();
      (this.isRf64
        ? (g(this.ds64RiffSizePos !== null),
          this.writer.seek(this.ds64RiffSizePos),
          this.riffWriter.writeU64(t - 8),
          g(this.ds64DataSizePos !== null),
          this.writer.seek(this.ds64DataSizePos),
          this.riffWriter.writeU64(this.dataSize),
          g(this.ds64SampleCountPos !== null),
          this.writer.seek(this.ds64SampleCountPos),
          this.riffWriter.writeU64(this.sampleCount))
        : (g(this.riffSizePos !== null),
          this.writer.seek(this.riffSizePos),
          this.riffWriter.writeU32(t - 8),
          g(this.dataSizePos !== null),
          this.writer.seek(this.dataSizePos),
          this.riffWriter.writeU32(this.dataSize)),
        e());
    }
  }
  class Wh {
    constructor(e) {
      ((this.sourceSampleRate = null),
        (this.sourceNumberOfChannels = null),
        (this.startTime = null),
        (this.bufferStartFrame = 0),
        (this.maxWrittenFrame = null),
        (this.targetSampleRate = e.targetSampleRate),
        (this.targetNumberOfChannels = e.targetNumberOfChannels),
        (this.onSample = e.onSample),
        (this.bufferSizeInFrames = Math.floor(this.targetSampleRate * 5)),
        (this.bufferSizeInSamples =
          this.bufferSizeInFrames * this.targetNumberOfChannels));
    }
    doChannelMixerSetup() {
      g(this.sourceNumberOfChannels !== null);
      const e = this.sourceNumberOfChannels,
        t = this.targetNumberOfChannels;
      e === 1 && t === 2
        ? (this.channelMixer = (r, n) => r[n * e])
        : e === 1 && t === 4
          ? (this.channelMixer = (r, n, s) => r[n * e] * +(s < 2))
          : e === 1 && t === 6
            ? (this.channelMixer = (r, n, s) => r[n * e] * +(s === 2))
            : e === 2 && t === 1
              ? (this.channelMixer = (r, n) => {
                  const s = n * e;
                  return 0.5 * (r[s] + r[s + 1]);
                })
              : e === 2 && t === 4
                ? (this.channelMixer = (r, n, s) => (s < 2 ? r[n * e + s] : 0))
                : e === 2 && t === 6
                  ? (this.channelMixer = (r, n, s) =>
                      s < 2 ? r[n * e + s] : 0)
                  : e === 4 && t === 1
                    ? (this.channelMixer = (r, n) => {
                        const s = n * e;
                        return 0.25 * (r[s] + r[s + 1] + r[s + 2] + r[s + 3]);
                      })
                    : e === 4 && t === 2
                      ? (this.channelMixer = (r, n, s) => {
                          const a = n * e;
                          return 0.5 * (r[a + s] + r[a + s + 2]);
                        })
                      : e === 4 && t === 6
                        ? (this.channelMixer = (r, n, s) => {
                            const a = n * e;
                            return s < 2
                              ? r[a + s]
                              : s === 2 || s === 3
                                ? 0
                                : r[a + s - 2];
                          })
                        : e === 6 && t === 1
                          ? (this.channelMixer = (r, n) => {
                              const s = n * e;
                              return (
                                Math.SQRT1_2 * (r[s] + r[s + 1]) +
                                r[s + 2] +
                                0.5 * (r[s + 4] + r[s + 5])
                              );
                            })
                          : e === 6 && t === 2
                            ? (this.channelMixer = (r, n, s) => {
                                const a = n * e;
                                return (
                                  r[a + s] +
                                  Math.SQRT1_2 * (r[a + 2] + r[a + s + 4])
                                );
                              })
                            : e === 6 && t === 4
                              ? (this.channelMixer = (r, n, s) => {
                                  const a = n * e;
                                  return s < 2
                                    ? r[a + s] + Math.SQRT1_2 * r[a + 2]
                                    : r[a + s + 2];
                                })
                              : (this.channelMixer = (r, n, s) =>
                                  s < e ? r[n * e + s] : 0);
    }
    ensureTempBufferSize(e) {
      let t = this.tempSourceBuffer.length;
      for (; t < e;) t *= 2;
      if (t !== this.tempSourceBuffer.length) {
        const r = new Float32Array(t);
        (r.set(this.tempSourceBuffer), (this.tempSourceBuffer = r));
      }
    }
    async add(e) {
      if (this.sourceSampleRate === null) {
        ((this.sourceSampleRate = e.sampleRate),
          (this.sourceNumberOfChannels = e.numberOfChannels),
          (this.startTime = e.timestamp));
        const l = Math.ceil(this.targetSampleRate / this.sourceSampleRate);
        ((this.bufferCapacityInFrames = this.bufferSizeInFrames + l),
          (this.outputBuffer = new Float32Array(
            this.bufferCapacityInFrames * this.targetNumberOfChannels,
          )),
          (this.tempSourceBuffer = new Float32Array(
            this.sourceSampleRate * this.sourceNumberOfChannels,
          )),
          this.doChannelMixerSetup());
      }
      g(this.startTime !== null);
      const t = e.numberOfFrames * e.numberOfChannels;
      this.ensureTempBufferSize(t);
      const r = e.allocationSize({ planeIndex: 0, format: "f32" }),
        n = new Float32Array(this.tempSourceBuffer.buffer, 0, r / 4);
      e.copyTo(n, { planeIndex: 0, format: "f32" });
      const s = e.timestamp - this.startTime,
        a = s + e.duration,
        o =
          Math.floor((s - 1 / this.sourceSampleRate) * this.targetSampleRate) +
          1,
        c = Math.ceil(a * this.targetSampleRate);
      for (let l = o; l < c; l++) {
        if (l < this.bufferStartFrame) continue;
        for (; l >= this.bufferStartFrame + this.bufferCapacityInFrames;)
          (await this.finalizeCurrentBuffer(),
            (this.bufferStartFrame += this.bufferSizeInFrames));
        const u = l - this.bufferStartFrame;
        g(u < this.bufferCapacityInFrames);
        const h = (l / this.targetSampleRate - s) * this.sourceSampleRate,
          p = Math.floor(h),
          m = Math.ceil(h),
          y = h - p;
        for (let w = 0; w < this.targetNumberOfChannels; w++) {
          let b = 0,
            T = 0;
          (p >= 0 && p < e.numberOfFrames && (b = this.channelMixer(n, p, w)),
            m >= 0 && m < e.numberOfFrames && (T = this.channelMixer(n, m, w)));
          const S = b + y * (T - b),
            k = u * this.targetNumberOfChannels + w;
          this.outputBuffer[k] += S;
        }
        this.maxWrittenFrame === null
          ? (this.maxWrittenFrame = u)
          : (this.maxWrittenFrame = Math.max(this.maxWrittenFrame, u));
      }
    }
    async finalizeCurrentBuffer() {
      if (this.maxWrittenFrame === null) return;
      g(this.startTime !== null);
      const e =
          Math.min(this.maxWrittenFrame + 1, this.bufferSizeInFrames) *
          this.targetNumberOfChannels,
        t = new Float32Array(e);
      t.set(this.outputBuffer.subarray(0, e));
      const r = new Ie({
        format: "f32",
        sampleRate: this.targetSampleRate,
        numberOfChannels: this.targetNumberOfChannels,
        timestamp:
          this.startTime + this.bufferStartFrame / this.targetSampleRate,
        data: t,
      });
      (await this.onSample(r),
        this.outputBuffer.copyWithin(0, this.bufferSizeInSamples),
        this.outputBuffer.fill(
          0,
          this.outputBuffer.length - this.bufferSizeInSamples,
        ),
        (this.maxWrittenFrame =
          this.maxWrittenFrame >= this.bufferSizeInFrames
            ? this.maxWrittenFrame - this.bufferSizeInFrames
            : null));
    }
    async finalize() {
      for (; this.maxWrittenFrame !== null;)
        (await this.finalizeCurrentBuffer(),
          (this.bufferStartFrame += this.bufferSizeInFrames));
    }
  }
  var Vh = function (i, e, t) {
      if (e != null) {
        if (typeof e != "object" && typeof e != "function")
          throw new TypeError("Object expected.");
        var r, n;
        if (t) {
          if (!Symbol.asyncDispose)
            throw new TypeError("Symbol.asyncDispose is not defined.");
          r = e[Symbol.asyncDispose];
        }
        if (r === void 0) {
          if (!Symbol.dispose)
            throw new TypeError("Symbol.dispose is not defined.");
          ((r = e[Symbol.dispose]), t && (n = r));
        }
        if (typeof r != "function")
          throw new TypeError("Object not disposable.");
        (n &&
          (r = function () {
            try {
              n.call(this);
            } catch (s) {
              return Promise.reject(s);
            }
          }),
          i.stack.push({ value: e, dispose: r, async: t }));
      } else t && i.stack.push({ async: !0 });
      return e;
    },
    Lh = (function (i) {
      return function (e) {
        function t(a) {
          ((e.error = e.hasError
            ? new i(a, e.error, "An error was suppressed during disposal.")
            : a),
            (e.hasError = !0));
        }
        var r,
          n = 0;
        function s() {
          for (; (r = e.stack.pop());)
            try {
              if (!r.async && n === 1)
                return ((n = 0), e.stack.push(r), Promise.resolve().then(s));
              if (r.dispose) {
                var a = r.dispose.call(r.value);
                if (r.async)
                  return (
                    (n |= 2),
                    Promise.resolve(a).then(s, function (o) {
                      return (t(o), s());
                    })
                  );
              } else n |= 1;
            } catch (o) {
              t(o);
            }
          if (n === 1)
            return e.hasError ? Promise.reject(e.error) : Promise.resolve();
          if (e.hasError) throw e.error;
        }
        return s();
      };
    })(
      typeof SuppressedError == "function"
        ? SuppressedError
        : function (i, e, t) {
            var r = new Error(t);
            return (
              (r.name = "SuppressedError"),
              (r.error = i),
              (r.suppressed = e),
              r
            );
          },
    );
  class Rs {
    constructor() {
      ((this._connectedTrack = null),
        (this._closingPromise = null),
        (this._closed = !1),
        (this._nominalBitrate = null));
    }
    _ensureValidAdd() {
      if (!this._connectedTrack)
        throw new Error("Source is not connected to an output track.");
      if (this._connectedTrack.output.state === "canceled")
        throw new Error("Output has been canceled.");
      if (
        this._connectedTrack.output.state === "finalizing" ||
        this._connectedTrack.output.state === "finalized"
      )
        throw new Error("Output has been finalized.");
      if (this._connectedTrack.output.state === "pending")
        throw new Error("Output has not started.");
      if (this._closed) throw new Error("Source is closed.");
    }
    async _start() {}
    async _flushAndClose(e) {}
    close() {
      if (this._closingPromise) return;
      const e = this._connectedTrack;
      if (!e)
        throw new Error(
          "Cannot call close without connecting the source to an output track.",
        );
      if (e.output.state === "pending")
        throw new Error("Cannot call close before output has been started.");
      this._closingPromise = (async () => {
        (await this._flushAndClose(!1),
          (this._closed = !0),
          !(
            e.output.state === "finalizing" || e.output.state === "finalized"
          ) && e.output._muxer.onTrackClose(e));
      })();
    }
    async _flushOrWaitForOngoingClose(e) {
      return (this._closingPromise ??= (async () => {
        (await this._flushAndClose(e), (this._closed = !0));
      })());
    }
  }
  class Ms extends Rs {
    constructor(e) {
      if ((super(), !Fe.includes(e)))
        throw new TypeError(
          `Invalid video codec '${e}'. Must be one of: ${Fe.join(", ")}.`,
        );
      this._codec = e;
    }
  }
  const Ds = (i, e) => {
    if (i.metadata.hasOnlyKeyPackets && e.type !== "key")
      throw new Error(
        "Cannot add non-key packets to a hasOnlyKeyPackets video track.",
      );
  };
  class Hh extends Ms {
    constructor(e) {
      super(e);
    }
    add(e, t) {
      if (!(e instanceof Z))
        throw new TypeError("packet must be an EncodedPacket.");
      if (e.isMetadataOnly)
        throw new TypeError("Metadata-only packets cannot be added.");
      if (t !== void 0 && (!t || typeof t != "object"))
        throw new TypeError("meta, when provided, must be an object.");
      return (
        this._ensureValidAdd(),
        Ds(this._connectedTrack, e),
        this._connectedTrack.output._muxer.addEncodedVideoPacket(
          this._connectedTrack,
          e,
          t,
        )
      );
    }
  }
  class qh {
    setError(e) {
      this.errorSet || ((this.error = e), (this.errorSet = !0));
    }
    constructor(e, t) {
      ((this.source = e),
        (this.encodingConfig = t),
        (this.ensureEncoderPromise = null),
        (this.encoderInitialized = !1),
        (this.encoder = null),
        (this.encoderConfig = null),
        (this.encodersReclaimed = !1),
        (this.muxer = null),
        (this.lastMultipleOfKeyFrameInterval = -1),
        (this.emittedEncoderPackets = 0),
        (this.codedWidth = null),
        (this.codedHeight = null),
        (this.outputWidth = null),
        (this.outputHeight = null),
        (this.frameRateLastSample = null),
        (this.frameRateLastTimestamp = null),
        (this.frameRateLastEndTimestamp = null),
        (this.preciseTimings = []),
        (this.customEncoder = null),
        (this.customEncoderCallSerializer = new Sr()),
        (this.customEncoderQueueSize = 0),
        (this.defaultEncodeOptions = {}),
        (this.alphaEncoder = null),
        (this.splitter = null),
        (this.splitterCreationFailed = !1),
        (this.alphaFrameQueue = []),
        (this.error = null),
        (this.errorSet = !1),
        (this.lastMuxerPromise = Promise.resolve()),
        (this.closed = !1));
    }
    async add(e, t, r) {
      const n = e;
      try {
        (this.checkForEncoderError(), this.source._ensureValidAdd());
        const s = this.encodingConfig,
          a = s.sizeChangeBehavior ?? "deny";
        let o = !1;
        if (this.codedWidth !== null && this.codedHeight !== null) {
          if (
            (e.codedWidth !== this.codedWidth ||
              e.codedHeight !== this.codedHeight) &&
            ((o = !0), a === "deny")
          )
            throw new Error(
              `Video sample size must remain constant. Expected ${this.codedWidth}x${this.codedHeight}, got ${e.codedWidth}x${e.codedHeight}. To allow the sample size to change over time, set \`sizeChangeBehavior\` to a value other than 'deny' in the encoding options.`,
            );
        } else
          ((this.codedWidth = e.codedWidth),
            (this.codedHeight = e.codedHeight));
        if (
          s.transform?.width !== void 0 ||
          s.transform?.height !== void 0 ||
          s.transform?.rotate !== void 0 ||
          s.transform?.flip !== void 0 ||
          s.transform?.crop !== void 0 ||
          s.transform?.force === !0 ||
          (o && a !== "passThrough")
        ) {
          let d = s.transform?.width,
            f = s.transform?.height,
            h = s.transform?.fit ?? "fill";
          o &&
            a !== "passThrough" &&
            (g(this.outputWidth),
            g(this.outputHeight),
            g(a !== "deny"),
            (d = this.outputWidth),
            (f = this.outputHeight),
            (h = a));
          const p = await e.transform({
            width: d,
            height: f,
            roundDimensionsTo: 2,
            crop: s.transform?.crop,
            rotate: s.transform?.rotate,
            flip: s.transform?.flip,
            fit: h,
            alpha: s.alpha,
          });
          ((this.outputWidth === null || this.outputHeight === null) &&
            ((this.outputWidth = p.displayWidth),
            (this.outputHeight = p.displayHeight)),
            t && e.close(),
            (e = p),
            (t = !0));
        } else
          (this.outputWidth === null || this.outputHeight === null) &&
            ((this.outputWidth = e.codedWidth),
            (this.outputHeight = e.codedHeight));
        const u = s.transform?.frameRate;
        if (u !== void 0) {
          const d = e.timestamp + e.duration,
            f = ea(e.timestamp, u);
          if (this.frameRateLastSample !== null)
            if (f <= this.frameRateLastTimestamp) {
              (this.frameRateLastSample.close(),
                (this.frameRateLastSample = e.clone()),
                (this.frameRateLastEndTimestamp = d));
              return;
            } else await this.padFrameRate(f, r);
          (e === n && ((e = e.clone()), (t = !0)),
            e.setTimestamp(f),
            e.setDuration(1 / u),
            this.frameRateLastSample?.close(),
            (this.frameRateLastSample = e.clone()),
            (this.frameRateLastTimestamp = f),
            (this.frameRateLastEndTimestamp = d));
        }
        await this.processAndEncode(e, r);
      } finally {
        t && e.close();
      }
    }
    async processAndEncode(e, t) {
      const r = this.encodingConfig;
      let n;
      if (r.transform?.process) {
        let s = r.transform.process(e);
        if ((A(s) && (s = await s), s === null)) return;
        Array.isArray(s) || (s = [s]);
        const a = [];
        try {
          for (const o of s)
            o instanceof _e
              ? a.push(o)
              : typeof VideoFrame < "u" && o instanceof VideoFrame
                ? a.push(new _e(o))
                : a.push(
                    new _e(o, { timestamp: e.timestamp, duration: e.duration }),
                  );
        } catch (o) {
          for (const c of a) c !== e && c.close();
          for (const c of s)
            ((c instanceof _e && c !== e) ||
              (typeof VideoFrame < "u" && c instanceof VideoFrame)) &&
              c.close();
          throw o;
        }
        n = a;
      } else n = [e];
      try {
        for (const s of n) {
          if (
            (this.encoderInitialized ||
              (this.ensureEncoderPromise || this.ensureEncoder(s),
              this.encoderInitialized || (await this.ensureEncoderPromise)),
            g(this.encoderInitialized),
            this.closed)
          )
            break;
          this.encodersReclaimed && this.recreateWebCodecsEncoders();
          const a = this.encodingConfig.keyFrameInterval ?? 2,
            o = Math.floor(s.timestamp / a),
            c = { ...this.defaultEncodeOptions, ...s.encodeOptions, ...t },
            l = {
              ...c,
              keyFrame:
                c.keyFrame !== void 0
                  ? c.keyFrame
                  : a === 0 || o !== this.lastMultipleOfKeyFrameInterval,
            };
          if (
            ((this.lastMultipleOfKeyFrameInterval = o),
            this.encodingConfig.onEncodedSample?.(s),
            this.customEncoder)
          ) {
            this.customEncoderQueueSize++;
            const u = s.clone(),
              d = this.customEncoderCallSerializer
                .call(() => this.customEncoder.encode(u, l))
                .catch((f) => this.setError(f))
                .finally(() => {
                  (this.customEncoderQueueSize--, u.close());
                });
            this.customEncoderQueueSize >= 4 && (await d);
          } else {
            g(this.encoder);
            const u = s.toVideoFrame(),
              d = X(
                this.preciseTimings,
                u.timestamp,
                (h) => h.microsecondTimestamp,
              ),
              f = d !== -1 ? this.preciseTimings[d] : null;
            if (
              (f && f.microsecondTimestamp === u.timestamp
                ? (f.timestamp !== s.timestamp && (f.timestampIsValid = !1),
                  f.duration !== s.duration && (f.durationIsValid = !1))
                : (this.preciseTimings.splice(d + 1, 0, {
                    microsecondTimestamp: u.timestamp,
                    timestamp: s.timestamp,
                    duration: s.duration,
                    timestampIsValid: !0,
                    durationIsValid: !0,
                  }),
                  this.preciseTimings.length > 128 &&
                    this.preciseTimings.shift()),
              this.alphaEncoder)
            )
              if (
                (!!u.format && !u.format.includes("A")) ||
                this.splitterCreationFailed
              ) {
                this.alphaFrameQueue.push(null);
                try {
                  this.encoder.encode(u, l);
                } finally {
                  u.close();
                }
              } else {
                this.splitter || (this.splitter = new jh());
                const { colorFrame: p, alphaFrame: m } =
                  await this.splitter.split(u);
                this.alphaFrameQueue.push(m);
                try {
                  this.encoder.encode(p, l);
                } finally {
                  p.close();
                }
              }
            else
              try {
                this.encoder.encode(u, l);
              } finally {
                u.close();
              }
            this.encoder.encodeQueueSize >= 4 &&
              (await new Promise((h) =>
                this.encoder.addEventListener("dequeue", h, { once: !0 }),
              ));
          }
          await this.lastMuxerPromise;
        }
      } finally {
        for (const s of n) s !== e && s.close();
      }
    }
    async padFrameRate(e, t) {
      const r = this.encodingConfig.transform.frameRate;
      g(this.frameRateLastSample);
      const n = Math.round((e - this.frameRateLastTimestamp) * r);
      for (let s = 1; s < n; s++) {
        const a = { stack: [], error: void 0, hasError: !1 };
        try {
          const o = Vh(a, this.frameRateLastSample.clone(), !1);
          (o.setTimestamp(this.frameRateLastTimestamp + s / r),
            o.setDuration(1 / r),
            await this.processAndEncode(o, t));
        } catch (o) {
          ((a.error = o), (a.hasError = !0));
        } finally {
          Lh(a);
        }
      }
    }
    ensureEncoder(e) {
      this.ensureEncoderPromise = (async () => {
        const t = Ei(this.encodingConfig.quality, this.encodingConfig.bitrate);
        g(t !== void 0);
        const r = hc({
          ...this.encodingConfig,
          quality: t,
          width: e.codedWidth,
          height: e.codedHeight,
          squarePixelWidth: e.squarePixelWidth,
          squarePixelHeight: e.squarePixelHeight,
          framerate: this.source._connectedTrack?.metadata.frameRate,
        });
        let n = null,
          s;
        for (const o of r) {
          const c = o.config;
          if (
            (this.encodingConfig.onEncoderConfig?.(c),
            (s = Sc.find((u) => u.supports(this.encodingConfig.codec, c))),
            s)
          ) {
            n = o;
            break;
          }
          if (typeof VideoEncoder > "u") continue;
          if (
            ((c.alpha = "discard"),
            this.encodingConfig.alpha === "keep" && (c.latencyMode = "quality"),
            (c.width % 2 === 1 || c.height % 2 === 1) &&
              (this.encodingConfig.codec === "avc" ||
                this.encodingConfig.codec === "hevc"))
          )
            throw new Error(
              `The dimensions ${c.width}x${c.height} are not supported for codec '${this.encodingConfig.codec}'; both width and height must be even numbers. Make sure to round your dimensions to the nearest even number.`,
            );
          try {
            if ((await VideoEncoder.isConfigSupported(c)).supported) {
              n = o;
              break;
            }
          } catch {}
        }
        if (!n) {
          if (typeof VideoEncoder > "u") throw new Error(xr("VideoEncoder"));
          const o = r[0].config,
            c = r.map(({ config: l, quantizer: u }) =>
              u !== null ? `quantizer ${u}` : `${l.bitrate} bps`,
            );
          throw new Error(
            `This specific encoder configuration (${o.codec}, ${c.join(" / ")}, ${o.width}x${o.height}, hardware acceleration: ${o.hardwareAcceleration ?? "no-preference"}) is not supported in this environment. Consider using another codec or changing your video parameters.`,
          );
        }
        const a = n.config;
        (n.quantizer !== null
          ? (this.defaultEncodeOptions = yc(
              this.encodingConfig.codec,
              n.quantizer,
            ))
          : (this.source._nominalBitrate = a.bitrate ?? null),
          s
            ? ((this.customEncoder = new s()),
              (this.customEncoder.codec = this.encodingConfig.codec),
              (this.customEncoder.config = a),
              (this.customEncoder.onPacket = (o, c) => {
                if (!(o instanceof Z))
                  throw new TypeError(
                    "The first argument passed to onPacket must be an EncodedPacket.",
                  );
                if (c !== void 0 && (!c || typeof c != "object"))
                  throw new TypeError(
                    "The second argument passed to onPacket must be an object or undefined.",
                  );
                (Ds(this.source._connectedTrack, o),
                  this.encodingConfig.onEncodedPacket?.(o, c),
                  (this.lastMuxerPromise = this.muxer
                    .addEncodedVideoPacket(this.source._connectedTrack, o, c)
                    .catch((l) => {
                      this.setError(l);
                    })));
              }),
              (this.customEncoder.onError = (o) => {
                this.setError(o);
              }),
              await this.customEncoder.init())
            : ((this.encoderConfig = a), this.createWebCodecsEncoders()),
          g(this.source._connectedTrack),
          (this.muxer = this.source._connectedTrack.output._muxer),
          (this.encoderInitialized = !0));
      })();
    }
    createWebCodecsEncoders() {
      const e = this.encoderConfig;
      g(e);
      const t = [],
        r = [];
      let n = 0,
        s = 0;
      const a = (c, l) => {
          if (gl(c)) {
            this.encodersReclaimed = !0;
            return;
          }
          ((c.stack = l), this.setError(c));
        },
        o = new Error("Encoding error").stack;
      if (
        ((this.encoder = new VideoEncoder({
          output: (c, l) => {
            if (!this.alphaEncoder) {
              this.addPacket(c, null, l);
              return;
            }
            const u = this.alphaFrameQueue.shift();
            (g(u !== void 0),
              u
                ? (this.alphaEncoder.encode(u, {
                    ...this.defaultEncodeOptions,
                    keyFrame: c.type === "key",
                  }),
                  s++,
                  u.close(),
                  t.push({ chunk: c, meta: l }))
                : s === 0
                  ? this.addPacket(c, null, l)
                  : (r.push(n + s), t.push({ chunk: c, meta: l })));
          },
          error: (c) => a(c, o),
        })),
        this.encoder.configure(e),
        this.encodingConfig.alpha === "keep")
      ) {
        const c = new Error("Encoding error").stack;
        ((this.alphaEncoder = new VideoEncoder({
          output: (l, u) => {
            s--;
            const d = t.shift();
            for (
              g(d !== void 0), this.addPacket(d.chunk, l, d.meta), n++;
              r.length > 0 && r[0] === n;
            ) {
              r.shift();
              const f = t.shift();
              (g(f !== void 0), this.addPacket(f.chunk, null, f.meta));
            }
          },
          error: (l) => a(l, c),
        })),
          this.alphaEncoder.configure(e));
      }
    }
    addPacket(e, t, r) {
      const n = {};
      if (t) {
        const u = new Uint8Array(t.byteLength);
        (t.copyTo(u), (n.alpha = u));
      }
      let s = Z.fromEncodedChunk(e, n);
      const a = this.encoderConfig;
      (g(a),
        r?.decoderConfig &&
          (r.decoderConfig.codec = Dr({
            width: a.width,
            height: a.height,
            codec: this.encodingConfig.codec,
            codecDescription: r.decoderConfig.description
              ? me(r.decoderConfig.description)
              : null,
            colorSpace: r.decoderConfig.colorSpace ?? null,
            avcType: a.codec.includes("avc1.")
              ? 1
              : a.codec.includes("avc3.")
                ? 3
                : null,
            avcCodecInfo:
              this.encodingConfig.codec === "avc" &&
              !r.decoderConfig.description
                ? $i(s.data)
                : null,
            hevcCodecInfo:
              this.encodingConfig.codec === "hevc" &&
              !r.decoderConfig.description
                ? Ki(s.data)
                : null,
            vp9CodecInfo:
              this.encodingConfig.codec === "vp9" ? Dn(s.data) : null,
            av1CodecInfo:
              this.encodingConfig.codec === "av1" ? _r(s.data) : null,
            proresFormat: null,
          })));
      const o = X(
          this.preciseTimings,
          e.timestamp,
          (u) => u.microsecondTimestamp,
        ),
        c = o !== -1 ? this.preciseTimings[o] : null;
      let l = null;
      (this.emittedEncoderPackets === 0 &&
        s.type === "delta" &&
        r?.decoderConfig &&
        (l = vr(this.encodingConfig.codec, r.decoderConfig, s.data)),
        ((c && c.microsecondTimestamp === e.timestamp) || l !== null) &&
          (s = s.clone({
            timestamp: c?.timestampIsValid ? c.timestamp : void 0,
            duration: c?.durationIsValid ? c.duration : void 0,
            type: l ?? void 0,
          })),
        Ds(this.source._connectedTrack, s),
        this.encodingConfig.onEncodedPacket?.(s, r),
        (this.lastMuxerPromise = this.muxer
          .addEncodedVideoPacket(this.source._connectedTrack, s, r)
          .catch((u) => {
            this.setError(u);
          })),
        this.emittedEncoderPackets++);
    }
    recreateWebCodecsEncoders() {
      (g(this.encoder),
        this.encoder.state !== "closed" && this.encoder.close(),
        this.alphaEncoder &&
          this.alphaEncoder.state !== "closed" &&
          this.alphaEncoder.close(),
        this.alphaFrameQueue.forEach((e) => e?.close()),
        (this.alphaFrameQueue.length = 0),
        (this.encodersReclaimed = !1),
        (this.lastMultipleOfKeyFrameInterval = -1),
        this.createWebCodecsEncoders());
    }
    async flushAndClose(e) {
      try {
        if (!e && (this.checkForEncoderError(), this.frameRateLastSample)) {
          const t = this.encodingConfig.transform.frameRate,
            r = ea(this.frameRateLastEndTimestamp, t);
          await this.padFrameRate(r);
        }
        ((this.closed = !0),
          e ||
            (this.customEncoder
              ? this.customEncoderCallSerializer.call(() =>
                  this.customEncoder.flush(),
                )
              : this.encoder &&
                !this.encodersReclaimed &&
                (await this.encoder.flush(),
                await this.alphaEncoder?.flush(),
                await sa(25))));
      } finally {
        ((this.closed = !0),
          this.frameRateLastSample?.close(),
          (this.frameRateLastSample = null),
          this.customEncoder
            ? await this.customEncoderCallSerializer
                .call(() => this.customEncoder.close())
                .catch((t) => this.setError(t))
            : this.encoder &&
              (this.encoder.state !== "closed" && this.encoder.close(),
              this.alphaEncoder &&
                this.alphaEncoder.state !== "closed" &&
                this.alphaEncoder.close(),
              this.alphaFrameQueue.forEach((t) => t?.close()),
              (this.alphaFrameQueue.length = 0),
              this.splitter?.close()));
      }
      e || this.checkForEncoderError();
    }
    getQueueSize() {
      return this.customEncoder
        ? this.customEncoderQueueSize
        : (this.encoder?.encodeQueueSize ?? 0);
    }
    checkForEncoderError() {
      if (this.errorSet) throw this.error;
    }
  }
  let Os = null;
  class jh {
    constructor() {
      ((this.worker = null),
        (this.pendingRequests = new Map()),
        (this.nextRequestId = 0));
    }
    split(e) {
      if (!this.worker) {
        if (!Os) {
          const n = new Blob([`(${$h.toString()})()`], {
            type: "application/javascript",
          });
          Os = URL.createObjectURL(n);
        }
        ((this.worker = new Worker(Os)),
          this.worker.addEventListener("message", (n) => {
            const s = n.data,
              a = this.pendingRequests.get(s.id);
            a &&
              (this.pendingRequests.delete(s.id),
              "error" in s
                ? a.reject(new Error(s.error))
                : a.resolve({
                    colorFrame: s.colorFrame,
                    alphaFrame: s.alphaFrame,
                  }));
          }),
          this.worker.addEventListener("error", (n) => {
            const s = new Error(
              n.message || "Color/alpha splitter worker error.",
            );
            for (const a of this.pendingRequests.values()) a.reject(s);
            this.pendingRequests.clear();
          }));
      }
      const t = this.nextRequestId++,
        r = ce();
      return (
        this.pendingRequests.set(t, r),
        this.worker.postMessage({ id: t, sourceFrame: e }, { transfer: [e] }),
        r.promise
      );
    }
    close() {
      (this.worker?.terminate(), (this.worker = null));
      const e = new Error("Color/alpha splitter closed.");
      for (const t of this.pendingRequests.values()) t.reject(e);
      this.pendingRequests.clear();
    }
  }
  const $h = () => {
    let i = null,
      e = Promise.resolve();
    self.addEventListener("message", (s) => {
      const { id: a, sourceFrame: o } = s.data;
      e = e.then(async () => {
        try {
          const { colorFrame: c, alphaFrame: l } = await t(o);
          self.postMessage(
            { id: a, colorFrame: c, alphaFrame: l },
            { transfer: [c, l] },
          );
        } catch (c) {
          self.postMessage({ id: a, error: c.message });
        } finally {
          o.close();
        }
      });
    });
    const t = async (s) => {
        const a = s.format;
        if (!a)
          throw new Error(
            "CPU color/alpha splitting requires a known VideoFrame format.",
          );
        const o = s.allocationSize();
        if (
          ((!i || i.byteLength !== o) && (i = new Uint8Array(o)),
          await s.copyTo(i),
          a === "RGBA" || a === "BGRA")
        )
          return r(i, a, s);
        if (
          a === "I420A" ||
          a === "I420AP10" ||
          a === "I420AP12" ||
          a === "I422A" ||
          a === "I422AP10" ||
          a === "I422AP12" ||
          a === "I444A" ||
          a === "I444AP10" ||
          a === "I444AP12"
        )
          return n(i, a, s);
        throw new Error(
          `CPU color/alpha splitting does not support format '${a}'.`,
        );
      },
      r = (s, a, o) => {
        const c = o.visibleRect?.width ?? o.codedWidth,
          l = o.visibleRect?.height ?? o.codedHeight,
          u = c * l,
          d = Math.ceil(c / 2),
          f = Math.ceil(l / 2),
          h = u + d * f * 2,
          p = new Uint8Array(h);
        for (let b = 0, T = 3; b < u; b++, T += 4) p[b] = s[T];
        p.fill(128, u);
        const m = new VideoFrame(s, {
            format: a === "RGBA" ? "RGBX" : "BGRX",
            codedWidth: c,
            codedHeight: l,
            timestamp: o.timestamp,
            duration: o.duration ?? void 0,
          }),
          y = {
            format: "I420",
            codedWidth: c,
            codedHeight: l,
            timestamp: o.timestamp,
            duration: o.duration ?? void 0,
            colorSpace: {
              fullRange: !0,
              matrix: "bt709",
              primaries: "bt709",
              transfer: "bt709",
            },
            transfer: [p.buffer],
          },
          w = new VideoFrame(p, y);
        return { colorFrame: m, alphaFrame: w };
      },
      n = (s, a, o) => {
        const c = o.visibleRect?.width ?? o.codedWidth,
          l = o.visibleRect?.height ?? o.codedHeight,
          u = a.includes("P10"),
          d = a.includes("P12"),
          f = u || d ? 2 : 1;
        let h, p;
        a.startsWith("I420")
          ? ((h = Math.ceil(c / 2)), (p = Math.ceil(l / 2)))
          : a.startsWith("I422")
            ? ((h = Math.ceil(c / 2)), (p = l))
            : ((h = c), (p = l));
        const m = c * l,
          y = h * p,
          w = m * f,
          b = y * f,
          T = m * f,
          S = w + b * 2,
          k = a.replace("A", ""),
          P = Math.ceil(c / 2),
          I = Math.ceil(l / 2),
          E = P * I,
          C = E * f,
          _ = T + 2 * C,
          v = new Uint8Array(_),
          F = S;
        v.set(s.subarray(F, F + T), 0);
        const D = T,
          z = u ? 512 : d ? 2048 : 128;
        f === 1 ? v.fill(z, D) : new Uint16Array(v.buffer, D, 2 * E).fill(z);
        const O = u ? "I420P10" : d ? "I420P12" : "I420",
          H = new VideoFrame(s.subarray(0, S), {
            format: k,
            codedWidth: c,
            codedHeight: l,
            timestamp: o.timestamp,
            duration: o.duration ?? void 0,
          }),
          ee = {
            format: O,
            codedWidth: c,
            codedHeight: l,
            timestamp: o.timestamp,
            duration: o.duration ?? void 0,
            colorSpace: {
              fullRange: !0,
              matrix: "bt709",
              primaries: "bt709",
              transfer: "bt709",
            },
            transfer: [v.buffer],
          },
          he = new VideoFrame(v, ee);
        return { colorFrame: H, alphaFrame: he };
      };
  };
  class Xc extends Ms {
    constructor(e) {
      (qd(e), super(e.codec), (this._encoder = new qh(this, e)));
    }
    add(e, t) {
      if (!(e instanceof _e))
        throw new TypeError("videoSample must be a VideoSample.");
      return this._encoder.add(e, !1, t);
    }
    _flushAndClose(e) {
      return this._encoder.flushAndClose(e);
    }
  }
  class zs extends Rs {
    constructor(e) {
      if ((super(), !Ne.includes(e)))
        throw new TypeError(
          `Invalid audio codec '${e}'. Must be one of: ${Ne.join(", ")}.`,
        );
      this._codec = e;
    }
  }
  class Kh extends zs {
    constructor(e) {
      super(e);
    }
    add(e, t) {
      if (!(e instanceof Z))
        throw new TypeError("packet must be an EncodedPacket.");
      if (e.isMetadataOnly)
        throw new TypeError("Metadata-only packets cannot be added.");
      if (t !== void 0 && (!t || typeof t != "object"))
        throw new TypeError("meta, when provided, must be an object.");
      return (
        this._ensureValidAdd(),
        this._connectedTrack.output._muxer.addEncodedAudioPacket(
          this._connectedTrack,
          e,
          t,
        )
      );
    }
  }
  class Gh {
    setError(e) {
      this.errorSet || ((this.error = e), (this.errorSet = !0));
    }
    constructor(e, t) {
      ((this.source = e),
        (this.encodingConfig = t),
        (this.ensureEncoderPromise = null),
        (this.encoderInitialized = !1),
        (this.encoder = null),
        (this.muxer = null),
        (this.lastNumberOfChannels = null),
        (this.lastSampleRate = null),
        (this.isPcmEncoder = !1),
        (this.outputSampleSize = null),
        (this.writeOutputValue = null),
        (this.customEncoder = null),
        (this.customEncoderCallSerializer = new Sr()),
        (this.customEncoderQueueSize = 0),
        (this.lastEndSampleIndex = null),
        (this.resampler = null),
        (this.error = null),
        (this.errorSet = !1),
        (this.lastMuxerPromise = Promise.resolve()),
        (this.closed = !1));
    }
    async add(e, t) {
      try {
        if (
          (this.checkForEncoderError(),
          this.source._ensureValidAdd(),
          this.lastNumberOfChannels !== null && this.lastSampleRate !== null)
        ) {
          if (
            e.numberOfChannels !== this.lastNumberOfChannels ||
            e.sampleRate !== this.lastSampleRate
          )
            throw new Error(
              `Audio parameters must remain constant. Expected ${this.lastNumberOfChannels} channels at ${this.lastSampleRate} Hz, got ${e.numberOfChannels} channels at ${e.sampleRate} Hz.`,
            );
        } else
          ((this.lastNumberOfChannels = e.numberOfChannels),
            (this.lastSampleRate = e.sampleRate));
        const r = this.encodingConfig;
        r.transform?.numberOfChannels !== void 0 ||
        r.transform?.sampleRate !== void 0
          ? (this.resampler ||
              (this.resampler = new Wh({
                targetNumberOfChannels:
                  r.transform.numberOfChannels ?? e.numberOfChannels,
                targetSampleRate: r.transform.sampleRate ?? e.sampleRate,
                onSample: async (s) => {
                  await this.processAndEncode(s, !0);
                },
              })),
            await this.resampler.add(e))
          : await this.processAndEncode(e, t);
      } finally {
        t && e.close();
      }
    }
    async processAndEncode(e, t) {
      const r = this.encodingConfig;
      if (
        r.transform?.sampleFormat !== void 0 &&
        Vd(e.format) !== r.transform.sampleFormat
      ) {
        const n = Hd(e, r.transform.sampleFormat);
        (t && e.close(), (e = n), (t = !0));
      }
      if (r.transform?.process)
        try {
          let n = r.transform.process(e);
          if ((A(n) && (n = await n), n === null)) return;
          Array.isArray(n) || (n = [n]);
          try {
            for (const s of n)
              if (!(s instanceof Ie))
                throw new TypeError(
                  "The audio process function must return an AudioSample, null, or an array of AudioSamples.",
                );
            for (const s of n) await this.encodeSample(s, !0);
          } finally {
            for (const s of n) s instanceof Ie && s.close();
          }
        } finally {
          t && e.close();
        }
      else await this.encodeSample(e, t);
    }
    async encodeSample(e, t) {
      try {
        if (
          (this.encoderInitialized ||
            (this.ensureEncoderPromise || this.ensureEncoder(e),
            this.encoderInitialized || (await this.ensureEncoderPromise)),
          g(this.encoderInitialized),
          this.closed)
        )
          return;
        {
          const r = Math.round(e.timestamp * e.sampleRate),
            n = Math.round((e.timestamp + e.duration) * e.sampleRate);
          if (this.lastEndSampleIndex === null) this.lastEndSampleIndex = n;
          else {
            const s = r - this.lastEndSampleIndex;
            if (s >= 64) {
              const a = new Ie({
                data: new Float32Array(s * e.numberOfChannels),
                format: "f32-planar",
                sampleRate: e.sampleRate,
                numberOfChannels: e.numberOfChannels,
                numberOfFrames: s,
                timestamp: this.lastEndSampleIndex / e.sampleRate,
              });
              await this.encodeSample(a, !0);
            }
            this.lastEndSampleIndex += e.numberOfFrames;
          }
        }
        if ((this.encodingConfig.onEncodedSample?.(e), this.customEncoder)) {
          this.customEncoderQueueSize++;
          const r = e.clone(),
            n = this.customEncoderCallSerializer
              .call(() => this.customEncoder.encode(r))
              .catch((s) => this.setError(s))
              .finally(() => {
                (this.customEncoderQueueSize--, r.close());
              });
          (this.customEncoderQueueSize >= 4 && (await n),
            await this.lastMuxerPromise);
        } else if (this.isPcmEncoder) await this.doPcmEncoding(e, t);
        else {
          g(this.encoder);
          const r = e.toAudioData();
          (this.encoder.encode(r),
            r.close(),
            t && e.close(),
            this.encoder.encodeQueueSize >= 4 &&
              (await new Promise((n) =>
                this.encoder.addEventListener("dequeue", n, { once: !0 }),
              )),
            await this.lastMuxerPromise);
        }
      } finally {
        t && e.close();
      }
    }
    async doPcmEncoding(e, t) {
      (g(this.outputSampleSize), g(this.writeOutputValue));
      const {
          numberOfChannels: r,
          numberOfFrames: n,
          sampleRate: s,
          timestamp: a,
        } = e,
        o = 2048,
        c = [];
      for (let m = 0; m < n; m += o) {
        const y = Math.min(o, e.numberOfFrames - m),
          w = y * r * this.outputSampleSize,
          b = new ArrayBuffer(w),
          T = new DataView(b);
        c.push({ frameCount: y, view: T });
      }
      const l = e.format === "s32" || e.format === "s32-planar",
        u = l ? "s32-planar" : "f32-planar",
        d = l ? 1 / 2147483648 : 1,
        f = e.allocationSize({ planeIndex: 0, format: u }),
        h = l ? new Int32Array(f / 4) : new Float32Array(f / 4);
      for (let m = 0; m < r; m++) {
        e.copyTo(h, { planeIndex: m, format: u });
        for (let y = 0; y < c.length; y++) {
          const { frameCount: w, view: b } = c[y];
          for (let T = 0; T < w; T++)
            this.writeOutputValue(
              b,
              (T * r + m) * this.outputSampleSize,
              h[y * o + T] * d,
            );
        }
      }
      t && e.close();
      const p = {
        decoderConfig: {
          codec: this.encodingConfig.codec,
          numberOfChannels: r,
          sampleRate: s,
        },
      };
      for (let m = 0; m < c.length; m++) {
        const { frameCount: y, view: w } = c[m],
          b = w.buffer,
          T = m * o,
          S = new Z(new Uint8Array(b), "key", a + T / s, y / s);
        (this.encodingConfig.onEncodedPacket?.(S, p),
          await this.muxer.addEncodedAudioPacket(
            this.source._connectedTrack,
            S,
            p,
          ));
      }
    }
    ensureEncoder(e) {
      this.ensureEncoderPromise = (async () => {
        const { numberOfChannels: t, sampleRate: r } = e,
          n = Ei(this.encodingConfig.quality, this.encodingConfig.bitrate),
          s = pc({
            numberOfChannels: t,
            sampleRate: r,
            ...this.encodingConfig,
            quality: n,
          });
        (this.encodingConfig.onEncoderConfig?.(s),
          (this.source._nominalBitrate = s.bitrate ?? null));
        const a = xc.find((o) => o.supports(this.encodingConfig.codec, s));
        if (a)
          ((this.customEncoder = new a()),
            (this.customEncoder.codec = this.encodingConfig.codec),
            (this.customEncoder.config = s),
            (this.customEncoder.onPacket = (o, c) => {
              if (!(o instanceof Z))
                throw new TypeError(
                  "The first argument passed to onPacket must be an EncodedPacket.",
                );
              if (c !== void 0 && (!c || typeof c != "object"))
                throw new TypeError(
                  "The second argument passed to onPacket must be an object or undefined.",
                );
              (this.encodingConfig.onEncodedPacket?.(o, c),
                (this.lastMuxerPromise = this.muxer
                  .addEncodedAudioPacket(this.source._connectedTrack, o, c)
                  .catch((l) => {
                    this.setError(l);
                  })));
            }),
            (this.customEncoder.onError = (o) => {
              this.setError(o);
            }),
            await this.customEncoder.init());
        else if (pe.includes(this.encodingConfig.codec)) this.initPcmEncoder();
        else {
          if (typeof AudioEncoder > "u") throw new Error(xr("AudioEncoder"));
          let o;
          try {
            o = (await AudioEncoder.isConfigSupported(s)).supported ?? !1;
          } catch {
            o = !1;
          }
          if (!o)
            throw new Error(
              `This specific encoder configuration (${s.codec}, ${s.bitrate} bps, ${s.numberOfChannels} channels, ${s.sampleRate} Hz) is not supported in this environment. Consider using another codec or changing your audio parameters.`,
            );
          const c = new Error("Encoding error").stack;
          ((this.encoder = new AudioEncoder({
            output: (l, u) => {
              if (this.encodingConfig.codec === "aac" && u?.decoderConfig) {
                let f = !1;
                if (
                  (!u.decoderConfig.description ||
                  u.decoderConfig.description.byteLength < 2
                    ? (f = !0)
                    : (f =
                        En(me(u.decoderConfig.description)).objectType === 0),
                  f)
                ) {
                  const h = Number(ne(s.codec.split(".")));
                  u.decoderConfig.description = An({
                    objectType: h,
                    outputNumberOfChannels: u.decoderConfig.numberOfChannels,
                    outputSampleRate: u.decoderConfig.sampleRate,
                  });
                }
              }
              let d = Z.fromEncodedChunk(l);
              ((d = d.clone({
                timestamp: li(d.timestamp, s.sampleRate),
                duration:
                  l.duration != null ? li(d.duration, s.sampleRate) : void 0,
              })),
                this.encodingConfig.onEncodedPacket?.(d, u),
                (this.lastMuxerPromise = this.muxer
                  .addEncodedAudioPacket(this.source._connectedTrack, d, u)
                  .catch((f) => {
                    this.setError(f);
                  })));
            },
            error: (l) => {
              ((l.stack = c), this.setError(l));
            },
          })),
            this.encoder.configure(s));
        }
        (g(this.source._connectedTrack),
          (this.muxer = this.source._connectedTrack.output._muxer),
          (this.encoderInitialized = !0));
      })();
    }
    initPcmEncoder() {
      this.isPcmEncoder = !0;
      const e = this.encodingConfig.codec,
        { dataType: t, sampleSize: r, littleEndian: n } = Ge(e);
      switch (((this.outputSampleSize = r), r)) {
        case 1:
          t === "unsigned"
            ? (this.writeOutputValue = (s, a, o) =>
                s.setUint8(a, se(Math.round(o * 128) + 128, 0, 255)))
            : t === "signed"
              ? (this.writeOutputValue = (s, a, o) => {
                  s.setInt8(a, se(Math.round(o * 128), -128, 127));
                })
              : t === "ulaw"
                ? (this.writeOutputValue = (s, a, o) => {
                    const c = se(Math.round(o * 32768), -32768, 32767);
                    s.setUint8(a, Qd(c));
                  })
                : t === "alaw"
                  ? (this.writeOutputValue = (s, a, o) => {
                      const c = se(Math.round(o * 32768), -32768, 32767);
                      s.setUint8(a, Zd(c));
                    })
                  : g(!1);
          break;
        case 2:
          t === "unsigned"
            ? (this.writeOutputValue = (s, a, o) =>
                s.setUint16(a, se(Math.round(o * 32768) + 32768, 0, 65535), n))
            : t === "signed"
              ? (this.writeOutputValue = (s, a, o) =>
                  s.setInt16(a, se(Math.round(o * 32768), -32768, 32767), n))
              : g(!1);
          break;
        case 3:
          t === "unsigned"
            ? (this.writeOutputValue = (s, a, o) =>
                mn(s, a, se(Math.round(o * 8388608) + 8388608, 0, 16777215), n))
            : t === "signed"
              ? (this.writeOutputValue = (s, a, o) =>
                  dl(s, a, se(Math.round(o * 8388608), -8388608, 8388607), n))
              : g(!1);
          break;
        case 4:
          t === "unsigned"
            ? (this.writeOutputValue = (s, a, o) =>
                s.setUint32(
                  a,
                  se(Math.round(o * 2147483648) + 2147483648, 0, 4294967295),
                  n,
                ))
            : t === "signed"
              ? (this.writeOutputValue = (s, a, o) =>
                  s.setInt32(
                    a,
                    se(Math.round(o * 2147483648), -2147483648, 2147483647),
                    n,
                  ))
              : t === "float"
                ? (this.writeOutputValue = (s, a, o) => s.setFloat32(a, o, n))
                : g(!1);
          break;
        case 8:
          t === "float"
            ? (this.writeOutputValue = (s, a, o) => s.setFloat64(a, o, n))
            : g(!1);
          break;
        default:
          (xe(r), g(!1));
      }
    }
    async flushAndClose(e) {
      try {
        (e ||
          (this.checkForEncoderError(),
          this.resampler && (await this.resampler.finalize())),
          (this.closed = !0),
          e ||
            (this.customEncoder
              ? this.customEncoderCallSerializer.call(() =>
                  this.customEncoder.flush(),
                )
              : this.encoder && (await this.encoder.flush())));
      } finally {
        ((this.closed = !0),
          (this.resampler = null),
          this.customEncoder
            ? await this.customEncoderCallSerializer
                .call(() => this.customEncoder.close())
                .catch((t) => this.setError(t))
            : this.encoder &&
              this.encoder.state !== "closed" &&
              this.encoder.close());
      }
      e || this.checkForEncoderError();
    }
    getQueueSize() {
      return this.customEncoder
        ? this.customEncoderQueueSize
        : this.isPcmEncoder
          ? 0
          : (this.encoder?.encodeQueueSize ?? 0);
    }
    checkForEncoderError() {
      if (this.errorSet) throw this.error;
    }
  }
  class Xh extends zs {
    constructor(e) {
      (jd(e), super(e.codec), (this._encoder = new Gh(this, e)));
    }
    add(e) {
      if (!(e instanceof Ie))
        throw new TypeError("audioSample must be an AudioSample.");
      return this._encoder.add(e, !1);
    }
    _flushAndClose(e) {
      return this._encoder.flushAndClose(e);
    }
  }
  class Qh extends Rs {
    constructor(e) {
      if ((super(), !$t.includes(e)))
        throw new TypeError(
          `Invalid subtitle codec '${e}'. Must be one of: ${$t.join(", ")}.`,
        );
      this._codec = e;
    }
  }
  class Zr {
    get supportsVideoRotationMetadata() {
      return this.supportsVideoTransformationMetadata;
    }
    getSupportedVideoCodecs() {
      return this.getSupportedCodecs().filter((e) => Fe.includes(e));
    }
    getSupportedAudioCodecs() {
      return this.getSupportedCodecs().filter((e) => Ne.includes(e));
    }
    getSupportedSubtitleCodecs() {
      return this.getSupportedCodecs().filter((e) => $t.includes(e));
    }
    _codecUnsupportedHint(e) {
      return "";
    }
    _isFragmentedIsobmff() {
      return !1;
    }
  }
  class Ns extends Zr {
    constructor(e = {}) {
      if (!e || typeof e != "object")
        throw new TypeError("options must be an object.");
      if (
        e.fastStart !== void 0 &&
        ![!1, "in-memory", "reserve", "fragmented"].includes(e.fastStart)
      )
        throw new TypeError(
          "options.fastStart, when provided, must be false, 'in-memory', 'reserve', or 'fragmented'.",
        );
      if (
        e.minimumFragmentDuration !== void 0 &&
        (!gt(e.minimumFragmentDuration) || e.minimumFragmentDuration < 0)
      )
        throw new TypeError(
          "options.minimumFragmentDuration, when provided, must be a non-negative number.",
        );
      if (e.onFtyp !== void 0 && typeof e.onFtyp != "function")
        throw new TypeError(
          "options.onFtyp, when provided, must be a function.",
        );
      if (e.onMoov !== void 0 && typeof e.onMoov != "function")
        throw new TypeError(
          "options.onMoov, when provided, must be a function.",
        );
      if (e.onMdat !== void 0 && typeof e.onMdat != "function")
        throw new TypeError(
          "options.onMdat, when provided, must be a function.",
        );
      if (e.onMoof !== void 0 && typeof e.onMoof != "function")
        throw new TypeError(
          "options.onMoof, when provided, must be a function.",
        );
      if (
        e.metadataFormat !== void 0 &&
        !["mdir", "mdta", "udta", "auto"].includes(e.metadataFormat)
      )
        throw new TypeError(
          "options.metadataFormat, when provided, must be either 'auto', 'mdir', 'mdta', or 'udta'.",
        );
      (super(), (this._options = e));
    }
    getSupportedTrackCounts() {
      return {
        video: { min: 0, max: 4294967295 },
        audio: { min: 0, max: 4294967295 },
        subtitle: { min: 0, max: 4294967295 },
        total: { min: 0, max: 4294967295 },
      };
    }
    get supportsVideoTransformationMetadata() {
      return !0;
    }
    get supportsTimestampedMediaData() {
      return !0;
    }
    get negativeTimestampSupport() {
      return "full";
    }
    _createMuxer(e) {
      return new Rh(e, this);
    }
    _isFragmentedIsobmff() {
      return this._options.fastStart === "fragmented";
    }
  }
  class Us extends Ns {
    constructor(e) {
      super(e);
    }
    get _name() {
      return "MP4";
    }
    get fileExtension() {
      return ".mp4";
    }
    get mimeType() {
      return "video/mp4";
    }
    getSupportedCodecs() {
      return [
        ...Fe,
        ...Bt,
        "pcm-s16",
        "pcm-s16be",
        "pcm-s24",
        "pcm-s24be",
        "pcm-s32",
        "pcm-s32be",
        "pcm-f32",
        "pcm-f32be",
        "pcm-f64",
        "pcm-f64be",
        ...$t,
      ];
    }
    _codecUnsupportedHint(e) {
      return new Yc().getSupportedCodecs().includes(e)
        ? " Switching to MOV will grant support for this codec."
        : "";
    }
  }
  class Qc extends Ns {
    constructor(e) {
      super(e);
    }
    get _name() {
      return "CMAF";
    }
    get fileExtension() {
      return ".m4s";
    }
    get mimeType() {
      return "video/mp4";
    }
    getSupportedCodecs() {
      return [
        ...Fe,
        ...Bt,
        "pcm-s16",
        "pcm-s16be",
        "pcm-s24",
        "pcm-s24be",
        "pcm-s32",
        "pcm-s32be",
        "pcm-f32",
        "pcm-f32be",
        "pcm-f64",
        "pcm-f64be",
        ...$t,
      ];
    }
  }
  class Yc extends Ns {
    constructor(e) {
      super(e);
    }
    get _name() {
      return "MOV";
    }
    get fileExtension() {
      return ".mov";
    }
    get mimeType() {
      return "video/quicktime";
    }
    getSupportedCodecs() {
      return [...Fe, ...Ne];
    }
    _codecUnsupportedHint(e) {
      return new Us().getSupportedCodecs().includes(e)
        ? " Switching to MP4 will grant support for this codec."
        : "";
    }
  }
  class Zc extends Zr {
    constructor(e = {}) {
      if (!e || typeof e != "object")
        throw new TypeError("options must be an object.");
      if (e.appendOnly !== void 0 && typeof e.appendOnly != "boolean")
        throw new TypeError(
          "options.appendOnly, when provided, must be a boolean.",
        );
      if (
        e.minimumClusterDuration !== void 0 &&
        (!gt(e.minimumClusterDuration) || e.minimumClusterDuration < 0)
      )
        throw new TypeError(
          "options.minimumClusterDuration, when provided, must be a non-negative number.",
        );
      if (e.onEbmlHeader !== void 0 && typeof e.onEbmlHeader != "function")
        throw new TypeError(
          "options.onEbmlHeader, when provided, must be a function.",
        );
      if (
        e.onSegmentHeader !== void 0 &&
        typeof e.onSegmentHeader != "function"
      )
        throw new TypeError(
          "options.onHeader, when provided, must be a function.",
        );
      if (e.onCluster !== void 0 && typeof e.onCluster != "function")
        throw new TypeError(
          "options.onCluster, when provided, must be a function.",
        );
      (super(), (this._options = e));
    }
    _createMuxer(e) {
      return new zh(e, this);
    }
    get _name() {
      return "Matroska";
    }
    getSupportedTrackCounts() {
      return {
        video: { min: 0, max: 127 },
        audio: { min: 0, max: 127 },
        subtitle: { min: 0, max: 127 },
        total: { min: 0, max: 127 },
      };
    }
    get fileExtension() {
      return ".mkv";
    }
    get mimeType() {
      return "video/x-matroska";
    }
    getSupportedCodecs() {
      return [
        ...Fe,
        ...Bt,
        ...pe.filter(
          (e) =>
            !["pcm-s8", "pcm-f32be", "pcm-f64be", "ulaw", "alaw"].includes(e),
        ),
        ...$t,
      ];
    }
    get supportsVideoTransformationMetadata() {
      return !1;
    }
    get supportsTimestampedMediaData() {
      return !0;
    }
    get negativeTimestampSupport() {
      return "prefer-non-negative";
    }
  }
  class Jr extends Zc {
    constructor(e) {
      super(e);
    }
    getSupportedCodecs() {
      return [
        ...Fe.filter((e) => ["vp8", "vp9", "av1"].includes(e)),
        ...Ne.filter((e) => ["opus", "vorbis"].includes(e)),
        ...$t,
      ];
    }
    get _name() {
      return "WebM";
    }
    get fileExtension() {
      return ".webm";
    }
    get mimeType() {
      return "video/webm";
    }
    _codecUnsupportedHint(e) {
      return new Zc().getSupportedCodecs().includes(e)
        ? " Switching to MKV will grant support for this codec."
        : "";
    }
  }
  class Yh extends Zr {
    constructor(e = {}) {
      if (!e || typeof e != "object")
        throw new TypeError("options must be an object.");
      if (e.large !== void 0 && typeof e.large != "boolean")
        throw new TypeError("options.large, when provided, must be a boolean.");
      if (
        e.metadataFormat !== void 0 &&
        !["info", "id3"].includes(e.metadataFormat)
      )
        throw new TypeError(
          "options.metadataFormat, when provided, must be either 'info' or 'id3'.",
        );
      if (e.onHeader !== void 0 && typeof e.onHeader != "function")
        throw new TypeError(
          "options.onHeader, when provided, must be a function.",
        );
      (super(), (this._options = e));
    }
    _createMuxer(e) {
      return new Uh(e, this);
    }
    get _name() {
      return "WAVE";
    }
    getSupportedTrackCounts() {
      return {
        video: { min: 0, max: 0 },
        audio: { min: 1, max: 1 },
        subtitle: { min: 0, max: 0 },
        total: { min: 1, max: 1 },
      };
    }
    get fileExtension() {
      return ".wav";
    }
    get mimeType() {
      return "audio/wav";
    }
    getSupportedCodecs() {
      return [
        ...pe.filter((e) =>
          [
            "pcm-s16",
            "pcm-s24",
            "pcm-s32",
            "pcm-f32",
            "pcm-f64",
            "pcm-u8",
            "ulaw",
            "alaw",
          ].includes(e),
        ),
      ];
    }
    get supportsVideoTransformationMetadata() {
      return !1;
    }
    get supportsTimestampedMediaData() {
      return !1;
    }
    get negativeTimestampSupport() {
      return null;
    }
  }
  const Jc = ["video", "audio", "subtitle"];
  class yr {
    constructor(e, t, r, n, s) {
      ((this.id = e),
        (this.output = t),
        (this.type = r),
        (this.source = n),
        (this.metadata = s));
    }
    isVideoTrack() {
      return this.type === "video";
    }
    isAudioTrack() {
      return this.type === "audio";
    }
    isSubtitleTrack() {
      return this.type === "subtitle";
    }
    canBePairedWith(e) {
      if (!(e instanceof yr))
        throw new TypeError("other must be an OutputTrack.");
      if (this === e) return !1;
      const t = aa(this.metadata.group),
        r = aa(e.metadata.group);
      for (const n of t)
        if (
          (this.type !== e.type && r.some((o) => n === o)) ||
          r.some((o) => n._pairedGroups.has(o))
        )
          return !0;
      return !1;
    }
  }
  class Zh extends yr {
    constructor(e, t, r, n) {
      super(e, t, "video", r, n);
    }
  }
  class Jh extends yr {
    constructor(e, t, r, n) {
      super(e, t, "audio", r, n);
    }
  }
  class em extends yr {
    constructor(e, t, r, n) {
      super(e, t, "subtitle", r, n);
    }
  }
  class Ze {
    constructor() {
      this._pairedGroups = new Set();
    }
    pairWith(e) {
      if (!(e instanceof Ze))
        throw new TypeError("other must be an OutputTrackGroup.");
      if (this === e) throw new TypeError("Cannot pair a group with itself.");
      (this._pairedGroups.add(e), e._pairedGroups.add(this));
    }
  }
  const Ws = (i) => {
    if (!i || typeof i != "object")
      throw new TypeError("metadata must be an object.");
    if (i.languageCode !== void 0 && !Ni(i.languageCode))
      throw new TypeError(
        "metadata.languageCode, when provided, must be a three-letter, ISO 639-2/T language code.",
      );
    if (i.name !== void 0 && typeof i.name != "string")
      throw new TypeError("metadata.name, when provided, must be a string.");
    if (
      (i.disposition !== void 0 && Tl(i.disposition),
      i.maximumPacketCount !== void 0 &&
        (!Number.isInteger(i.maximumPacketCount) || i.maximumPacketCount < 0))
    )
      throw new TypeError(
        "metadata.maximumPacketCount, when provided, must be a non-negative integer.",
      );
    if (i.bitrate !== void 0 && (!Number.isFinite(i.bitrate) || i.bitrate < 0))
      throw new TypeError(
        "metadata.bitrate, when provided, must be a non-negative number.",
      );
    if (
      i.averageBitrate !== void 0 &&
      (!Number.isFinite(i.averageBitrate) || i.averageBitrate < 0)
    )
      throw new TypeError(
        "metadata.averageBitrate, when provided, must be a non-negative number.",
      );
    if (
      i.group !== void 0 &&
      !(i.group instanceof Ze) &&
      (!Array.isArray(i.group) || i.group.some((e) => !(e instanceof Ze)))
    )
      throw new TypeError(
        "metadata.group, when provided, must be an OutputTrackGroup instance or an array of OutputTrackGroup instances.",
      );
  };
  class en extends Li {
    get target() {
      const e =
        "Output.target cannot be used when using PathedTarget with an async callback. Use the 'target' event instead.";
      if (this._rootTargetPromise) throw new TypeError(e);
      const t = this._getRootTarget();
      if (A(t)) throw new TypeError(e);
      return t;
    }
    constructor(e) {
      if (
        (super(),
        (this.state = "pending"),
        (this.defaultTrackGroup = new Ze()),
        (this.tracks = []),
        (this._onFinalize = null),
        (this._unfinalizedTargets = new Set()),
        (this._rootWriterPromise = null),
        (this._startPromise = null),
        (this._cancelPromise = null),
        (this._finalizePromise = null),
        (this._mutex = new ci()),
        (this._metadataTags = {}),
        (this._rootTarget = null),
        (this._rootTargetPromise = null),
        (this._firstMediaStreamTimestamp = null),
        !e || typeof e != "object")
      )
        throw new TypeError("options must be an object.");
      if (!(e.format instanceof Zr))
        throw new TypeError("options.format must be an OutputFormat.");
      if (!(e.target instanceof lt || e.target instanceof Fs))
        throw new TypeError(
          "options.target must be a Target or a PathedTarget.",
        );
      if (
        (e.target instanceof lt && this._rememberTarget(e.target),
        e.initTarget !== void 0 &&
          !(e.initTarget instanceof lt) &&
          typeof e.initTarget != "function")
      )
        throw new Error(
          "options.initTarget, when provided, must be a Target or a function that returns or resolves to a Target.",
        );
      if (e.onFinalize !== void 0 && typeof e.onFinalize != "function")
        throw new TypeError(
          "options.onFinalize, when provided, must be a function.",
        );
      ((this.format = e.format),
        (this._target = e.target),
        (this._onFinalize = e.onFinalize ?? null),
        (this._initTarget = e.initTarget ?? null),
        this._initTarget instanceof lt &&
          this._rememberTarget(this._initTarget),
        (this._muxer = e.format._createMuxer(this)));
    }
    _getTargetValidated(e) {
      g(this._target instanceof Fs);
      const t = this._target.getTarget(e),
        r = (n) => {
          if (!(n instanceof lt))
            throw new TypeError("getTarget must return a Target.");
          return n;
        };
      return A(t) ? t.then(r) : r(t);
    }
    async _getTarget(e) {
      g(this._target instanceof Fs);
      const t = await this._getTargetValidated(e);
      return (
        this._emit("target", { target: t, request: e, isRoot: e.isRoot }),
        this.state === "canceled" ? await t._close() : this._rememberTarget(t),
        t
      );
    }
    _rememberTarget(e) {
      (this._unfinalizedTargets.add(e),
        e.on("finalized", () => this._unfinalizedTargets.delete(e), {
          once: !0,
        }));
    }
    async _getInitTarget() {
      if ((g(this._initTarget !== null), this._initTarget instanceof lt))
        return this._initTarget;
      const e = await this._initTarget();
      return (
        this.state === "canceled" ? await e._close() : this._rememberTarget(e),
        e
      );
    }
    _hasInitTarget() {
      return this._initTarget !== null;
    }
    _getRootTarget() {
      if (this._rootTarget) return this._rootTarget;
      if (this._rootTargetPromise) return this._rootTargetPromise;
      if (this._target instanceof lt)
        return (
          this._emit("target", {
            target: this._target,
            request: null,
            isRoot: !0,
          }),
          (this._rootTarget = this._target),
          this._target
        );
      const e = {
          path: this._target.rootPath,
          isRoot: !0,
          mimeType: this.format.mimeType,
        },
        t = this._getTargetValidated(e),
        r = (n) => (
          this.state === "canceled" ? n._close() : this._rememberTarget(n),
          this._emit("target", { target: n, request: e, isRoot: !0 }),
          (this._rootTarget = n),
          n
        );
      return A(t) ? (this._rootTargetPromise = t.then(r)) : r(t);
    }
    _getRootWriter(e) {
      return (this._rootWriterPromise ??= (async () => {
        const t = await this._getRootTarget(),
          r = new vs(t, typeof e == "boolean" ? e : e(t));
        return (r.start(), r);
      })());
    }
    addVideoTrack(e, t = {}) {
      if (!(e instanceof Ms))
        throw new TypeError("source must be a VideoSource.");
      if (
        (Ws(t),
        t.rotation !== void 0 && ![0, 90, 180, 270].includes(t.rotation))
      )
        throw new TypeError(
          `Invalid video rotation: ${t.rotation}. Has to be 0, 90, 180 or 270.`,
        );
      if (t.flip !== void 0 && typeof t.flip != "boolean")
        throw new TypeError("metadata.flip, when provided, must be a boolean.");
      if (
        t.transformationMatrix !== void 0 &&
        (!Array.isArray(t.transformationMatrix) ||
          t.transformationMatrix.length !== 9 ||
          !t.transformationMatrix.every((n) => Number.isFinite(n)))
      )
        throw new TypeError(
          "metadata.transformationMatrix, when provided, must be an array of 9 finite numbers.",
        );
      if (
        t.frameRate !== void 0 &&
        (!Number.isFinite(t.frameRate) || t.frameRate <= 0)
      )
        throw new TypeError(
          `Invalid video frame rate: ${t.frameRate}. Must be a positive number.`,
        );
      if (
        t.hasOnlyKeyPackets !== void 0 &&
        typeof t.hasOnlyKeyPackets != "boolean"
      )
        throw new TypeError(
          "metadata.hasOnlyKeyPackets, when provided, must be a boolean.",
        );
      if (
        t.canBeTransparent !== void 0 &&
        typeof t.canBeTransparent != "boolean"
      )
        throw new TypeError(
          "metadata.canBeTransparent, when provided, must be a boolean.",
        );
      if (
        (t.decoderConfig !== void 0 &&
          Hn({ decoderConfig: t.decoderConfig }, e._codec),
        t.primingPacket !== void 0)
      ) {
        if (!(t.primingPacket instanceof Z))
          throw new TypeError(
            "metadata.primingPacket, when provided, must be an EncodedPacket.",
          );
        if (t.decoderConfig === void 0)
          throw new TypeError(
            "metadata.primingPacket can only be provided alongside metadata.decoderConfig.",
          );
      }
      const r = { ...t };
      return (
        (r.group ??= this.defaultTrackGroup),
        this._addTrack(new Zh(this.tracks.length + 1, this, e, r))
      );
    }
    addAudioTrack(e, t = {}) {
      if (!(e instanceof zs))
        throw new TypeError("source must be an AudioSource.");
      if (
        (Ws(t),
        t.decoderConfig !== void 0 &&
          Xi({ decoderConfig: t.decoderConfig }, e._codec),
        t.primingPacket !== void 0)
      ) {
        if (!(t.primingPacket instanceof Z))
          throw new TypeError(
            "metadata.primingPacket, when provided, must be an EncodedPacket.",
          );
        if (t.decoderConfig === void 0)
          throw new TypeError(
            "metadata.primingPacket can only be provided alongside metadata.decoderConfig.",
          );
      }
      const r = { ...t };
      return (
        (r.group ??= this.defaultTrackGroup),
        this._addTrack(new Jh(this.tracks.length + 1, this, e, r))
      );
    }
    addSubtitleTrack(e, t = {}) {
      if (!(e instanceof Qh))
        throw new TypeError("source must be a SubtitleSource.");
      Ws(t);
      const r = { ...t };
      return (
        (r.group ??= this.defaultTrackGroup),
        this._addTrack(new em(this.tracks.length + 1, this, e, r))
      );
    }
    setMetadataTags(e) {
      if ((In(e), this.state !== "pending"))
        throw new Error(
          "Cannot set metadata tags after output has been started or canceled.",
        );
      this._metadataTags = e;
    }
    _addTrack(e) {
      if (this.state !== "pending")
        throw new Error(
          "Cannot add track after output has been started or canceled.",
        );
      if (e.source._connectedTrack)
        throw new Error("Source is already used for a track.");
      const t = this.format.getSupportedTrackCounts(),
        r = this.tracks.reduce((a, o) => a + (o.type === e.type ? 1 : 0), 0),
        n = t[e.type].max;
      if (r === n)
        throw new Error(
          n === 0
            ? `${this.format._name} does not support ${e.type} tracks.`
            : `${this.format._name} does not support more than ${n} ${e.type} track${n === 1 ? "" : "s"}.`,
        );
      const s = t.total.max;
      if (this.tracks.length === s)
        throw new Error(
          `${this.format._name} does not support more than ${s} tracks${s === 1 ? "" : "s"} in total.`,
        );
      if (e.isVideoTrack()) {
        const a = this.format.getSupportedVideoCodecs();
        if (a.length === 0)
          throw new Error(
            `${this.format._name} does not support video tracks.` +
              this.format._codecUnsupportedHint(e.source._codec),
          );
        if (!a.includes(e.source._codec))
          throw new Error(
            `Codec '${e.source._codec}' cannot be contained within ${this.format._name}. Supported video codecs are: ${a.map((o) => `'${o}'`).join(", ")}.` +
              this.format._codecUnsupportedHint(e.source._codec),
          );
      } else if (e.isAudioTrack()) {
        const a = this.format.getSupportedAudioCodecs();
        if (a.length === 0)
          throw new Error(
            `${this.format._name} does not support audio tracks.` +
              this.format._codecUnsupportedHint(e.source._codec),
          );
        if (!a.includes(e.source._codec))
          throw new Error(
            `Codec '${e.source._codec}' cannot be contained within ${this.format._name}. Supported audio codecs are: ${a.map((o) => `'${o}'`).join(", ")}.` +
              this.format._codecUnsupportedHint(e.source._codec),
          );
      } else if (e.isSubtitleTrack()) {
        const a = this.format.getSupportedSubtitleCodecs();
        if (a.length === 0)
          throw new Error(
            `${this.format._name} does not support subtitle tracks.` +
              this.format._codecUnsupportedHint(e.source._codec),
          );
        if (!a.includes(e.source._codec))
          throw new Error(
            `Codec '${e.source._codec}' cannot be contained within ${this.format._name}. Supported subtitle codecs are: ${a.map((o) => `'${o}'`).join(", ")}.` +
              this.format._codecUnsupportedHint(e.source._codec),
          );
      }
      return (this.tracks.push(e), (e.source._connectedTrack = e), e);
    }
    hasEnoughTracks() {
      const e = this.format.getSupportedTrackCounts();
      for (const r of Jc) {
        const n = this.tracks.reduce((a, o) => a + (o.type === r ? 1 : 0), 0),
          s = e[r].min;
        if (n < s) return !1;
      }
      const t = e.total.min;
      return !(this.tracks.length < t);
    }
    async start() {
      const e = this.format.getSupportedTrackCounts();
      for (const r of Jc) {
        const n = this.tracks.reduce((a, o) => a + (o.type === r ? 1 : 0), 0),
          s = e[r].min;
        if (n < s)
          throw new Error(
            s === e[r].max
              ? `${this.format._name} requires exactly ${s} ${r} track${s === 1 ? "" : "s"}.`
              : `${this.format._name} requires at least ${s} ${r} track${s === 1 ? "" : "s"}.`,
          );
      }
      const t = e.total.min;
      if (this.tracks.length < t)
        throw new Error(
          t === e.total.max
            ? `${this.format._name} requires exactly ${t} track${t === 1 ? "" : "s"}.`
            : `${this.format._name} requires at least ${t} track${t === 1 ? "" : "s"}.`,
        );
      if (this.state === "canceled")
        throw new Error("Output has been canceled.");
      return this._startPromise
        ? (L._warn("Output has already been started."), this._startPromise)
        : (this._startPromise = (async () => {
            this.state = "started";
            const r = this._mutex.acquire();
            try {
              await this._muxer.start();
              const n = this.tracks.map((s) => s.source._start());
              await Promise.all(n);
            } finally {
              (await r)();
            }
          })());
    }
    getMimeType() {
      return this._muxer.getMimeType();
    }
    async cancel() {
      if (this.state === "canceled") return this._cancelPromise ?? void 0;
      if (this.state === "finalizing" || this.state === "finalized") {
        this.state === "finalized" &&
          L._warn("Output has already been finalized.");
        return;
      }
      return (this._cancelPromise = (async () => {
        this.state = "canceled";
        const e = await this._mutex.acquire();
        try {
          const t = this.tracks.map((r) =>
            r.source._flushOrWaitForOngoingClose(!0),
          );
          (await Promise.all(t),
            await Promise.all(
              [...this._unfinalizedTargets].map((r) => r._close()),
            ),
            this._unfinalizedTargets.clear());
        } finally {
          e();
        }
      })());
    }
    async finalize() {
      if (this.state === "pending")
        throw new Error("Cannot finalize before starting.");
      if (this.state === "canceled")
        throw new Error("Cannot finalize after canceling.");
      return this._finalizePromise
        ? (L._warn("Output has already been finalized."), this._finalizePromise)
        : (this._finalizePromise = (async () => {
            this.state = "finalizing";
            const e = await this._mutex.acquire();
            try {
              const t = this.tracks.map((r) =>
                r.source._flushOrWaitForOngoingClose(!1),
              );
              if (
                (await Promise.all(t),
                await this._muxer.finalize(),
                this._rootWriterPromise)
              ) {
                const r = await this._rootWriterPromise;
                r.finalized || (await r.flush(), await r.finalize());
              }
              (this._onFinalize && (await this._onFinalize()),
                (this.state = "finalized"));
            } catch (t) {
              throw ((this.state = "canceled"), t);
            } finally {
              (await Promise.all(
                [...this._unfinalizedTargets].map((t) =>
                  t._close().catch(() => {}),
                ),
              ),
                this._unfinalizedTargets.clear(),
                e());
            }
          })());
    }
  }
  var br = function (i, e, t) {
      if (e != null) {
        if (typeof e != "object" && typeof e != "function")
          throw new TypeError("Object expected.");
        var r, n;
        if (t) {
          if (!Symbol.asyncDispose)
            throw new TypeError("Symbol.asyncDispose is not defined.");
          r = e[Symbol.asyncDispose];
        }
        if (r === void 0) {
          if (!Symbol.dispose)
            throw new TypeError("Symbol.dispose is not defined.");
          ((r = e[Symbol.dispose]), t && (n = r));
        }
        if (typeof r != "function")
          throw new TypeError("Object not disposable.");
        (n &&
          (r = function () {
            try {
              n.call(this);
            } catch (s) {
              return Promise.reject(s);
            }
          }),
          i.stack.push({ value: e, dispose: r, async: t }));
      } else t && i.stack.push({ async: !0 });
      return e;
    },
    tn = (function (i) {
      return function (e) {
        function t(a) {
          ((e.error = e.hasError
            ? new i(a, e.error, "An error was suppressed during disposal.")
            : a),
            (e.hasError = !0));
        }
        var r,
          n = 0;
        function s() {
          for (; (r = e.stack.pop());)
            try {
              if (!r.async && n === 1)
                return ((n = 0), e.stack.push(r), Promise.resolve().then(s));
              if (r.dispose) {
                var a = r.dispose.call(r.value);
                if (r.async)
                  return (
                    (n |= 2),
                    Promise.resolve(a).then(s, function (o) {
                      return (t(o), s());
                    })
                  );
              } else n |= 1;
            } catch (o) {
              t(o);
            }
          if (n === 1)
            return e.hasError ? Promise.reject(e.error) : Promise.resolve();
          if (e.hasError) throw e.error;
        }
        return s();
      };
    })(
      typeof SuppressedError == "function"
        ? SuppressedError
        : function (i, e, t) {
            var r = new Error(t);
            return (
              (r.name = "SuppressedError"),
              (r.error = i),
              (r.suppressed = e),
              r
            );
          },
    );
  const rn = (i) => {
      if (!i || typeof i != "object")
        throw new TypeError("options.video, when provided, must be an object.");
      if (i?.discard !== void 0 && typeof i.discard != "boolean")
        throw new TypeError(
          "options.video.discard, when provided, must be a boolean.",
        );
      if (i?.forceTranscode !== void 0 && typeof i.forceTranscode != "boolean")
        throw new TypeError(
          "options.video.forceTranscode, when provided, must be a boolean.",
        );
      if (i?.codec !== void 0 && !Fe.includes(i.codec))
        throw new TypeError(
          `options.video.codec, when provided, must be one of: ${Fe.join(", ")}.`,
        );
      const e = i?.bitrate;
      if (i?.quality !== void 0 && !(i.quality instanceof ke))
        throw new TypeError(
          "options.video.quality, when provided, must be a Quality.",
        );
      if (i?.quality !== void 0 && e !== void 0)
        throw new TypeError(
          "options.video.quality and options.video.bitrate cannot both be provided.",
        );
      if (
        e !== void 0 &&
        !(e instanceof ke) &&
        (!Number.isInteger(e) || e <= 0)
      )
        throw new TypeError(
          "options.video.bitrate, when provided, must be a positive integer or a quality.",
        );
      if (i?.width !== void 0 && (!Number.isInteger(i.width) || i.width <= 0))
        throw new TypeError(
          "options.video.width, when provided, must be a positive integer.",
        );
      if (
        i?.height !== void 0 &&
        (!Number.isInteger(i.height) || i.height <= 0)
      )
        throw new TypeError(
          "options.video.height, when provided, must be a positive integer.",
        );
      if (i?.fit !== void 0 && !["fill", "contain", "cover"].includes(i.fit))
        throw new TypeError(
          "options.video.fit, when provided, must be one of 'fill', 'contain', or 'cover'.",
        );
      if (i?.width !== void 0 && i.height !== void 0 && i.fit === void 0)
        throw new TypeError(
          "When both options.video.width and options.video.height are provided, options.video.fit must also be provided.",
        );
      if (i?.rotate !== void 0 && ![0, 90, 180, 270].includes(i.rotate))
        throw new TypeError(
          "options.video.rotate, when provided, must be 0, 90, 180 or 270.",
        );
      if (i?.flip !== void 0 && typeof i.flip != "boolean")
        throw new TypeError(
          "options.video.flip, when provided, must be a boolean.",
        );
      if (
        i?.allowTransformationMetadata !== void 0 &&
        typeof i.allowTransformationMetadata != "boolean"
      )
        throw new TypeError(
          "options.video.allowTransformationMetadata, when provided, must be a boolean.",
        );
      if (
        i?.allowRotationMetadata !== void 0 &&
        typeof i.allowRotationMetadata != "boolean"
      )
        throw new TypeError(
          "options.video.allowRotationMetadata, when provided, must be a boolean.",
        );
      if (
        (i?.crop !== void 0 && Hr(i.crop, "options.video."),
        i?.frameRate !== void 0 &&
          (!Number.isFinite(i.frameRate) || i.frameRate <= 0))
      )
        throw new TypeError(
          "options.video.frameRate, when provided, must be a finite positive number.",
        );
      if (i?.alpha !== void 0 && !["discard", "keep"].includes(i.alpha))
        throw new TypeError(
          "options.video.alpha, when provided, must be either 'discard' or 'keep'.",
        );
      if (
        i?.keyFrameInterval !== void 0 &&
        (!Number.isFinite(i.keyFrameInterval) || i.keyFrameInterval < 0)
      )
        throw new TypeError(
          "options.video.keyFrameInterval, when provided, must be a non-negative number.",
        );
      if (i?.process !== void 0 && typeof i.process != "function")
        throw new TypeError(
          "options.video.process, when provided, must be a function.",
        );
      if (
        i?.processedWidth !== void 0 &&
        (!Number.isInteger(i.processedWidth) || i.processedWidth <= 0)
      )
        throw new TypeError(
          "options.video.processedWidth, when provided, must be a positive integer.",
        );
      if (
        i?.processedHeight !== void 0 &&
        (!Number.isInteger(i.processedHeight) || i.processedHeight <= 0)
      )
        throw new TypeError(
          "options.video.processedHeight, when provided, must be a positive integer.",
        );
      if (
        i?.hardwareAcceleration !== void 0 &&
        !["no-preference", "prefer-hardware", "prefer-software"].includes(
          i.hardwareAcceleration,
        )
      )
        throw new TypeError(
          "options.video.hardwareAcceleration, when provided, must be 'no-preference', 'prefer-hardware' or 'prefer-software'.",
        );
      if (
        i?.group !== void 0 &&
        !(
          i.group instanceof Ze ||
          (Array.isArray(i.group) && i.group.every((t) => t instanceof Ze))
        )
      )
        throw new TypeError(
          "options.video.group, when provided, must be an OutputTrackGroup or an array of OutputTrackGroups.",
        );
    },
    nn = (i) => {
      if (!i || typeof i != "object")
        throw new TypeError("options.audio, when provided, must be an object.");
      if (i?.discard !== void 0 && typeof i.discard != "boolean")
        throw new TypeError(
          "options.audio.discard, when provided, must be a boolean.",
        );
      if (i?.forceTranscode !== void 0 && typeof i.forceTranscode != "boolean")
        throw new TypeError(
          "options.audio.forceTranscode, when provided, must be a boolean.",
        );
      if (i?.codec !== void 0 && !Ne.includes(i.codec))
        throw new TypeError(
          `options.audio.codec, when provided, must be one of: ${Ne.join(", ")}.`,
        );
      const e = i?.bitrate;
      if (i?.quality !== void 0 && !(i.quality instanceof ke))
        throw new TypeError(
          "options.audio.quality, when provided, must be a Quality.",
        );
      if (i?.quality !== void 0 && e !== void 0)
        throw new TypeError(
          "options.audio.quality and options.audio.bitrate cannot both be provided.",
        );
      if (
        e !== void 0 &&
        !(e instanceof ke) &&
        (!Number.isInteger(e) || e <= 0)
      )
        throw new TypeError(
          "options.audio.bitrate, when provided, must be a positive integer or a quality.",
        );
      if (
        i?.numberOfChannels !== void 0 &&
        (!Number.isInteger(i.numberOfChannels) || i.numberOfChannels <= 0)
      )
        throw new TypeError(
          "options.audio.numberOfChannels, when provided, must be a positive integer.",
        );
      if (
        i?.sampleRate !== void 0 &&
        (!Number.isInteger(i.sampleRate) || i.sampleRate <= 0)
      )
        throw new TypeError(
          "options.audio.sampleRate, when provided, must be a positive integer.",
        );
      if (
        i?.sampleFormat !== void 0 &&
        !["u8", "s16", "s32", "f32"].includes(i.sampleFormat)
      )
        throw new TypeError(
          "options.audio.sampleFormat, when provided, must be one of: u8, s16, s32, f32.",
        );
      if (i?.process !== void 0 && typeof i.process != "function")
        throw new TypeError(
          "options.audio.process, when provided, must be a function.",
        );
      if (
        i?.processedNumberOfChannels !== void 0 &&
        (!Number.isInteger(i.processedNumberOfChannels) ||
          i.processedNumberOfChannels <= 0)
      )
        throw new TypeError(
          "options.audio.processedNumberOfChannels, when provided, must be a positive integer.",
        );
      if (
        i?.processedSampleRate !== void 0 &&
        (!Number.isInteger(i.processedSampleRate) || i.processedSampleRate <= 0)
      )
        throw new TypeError(
          "options.audio.processedSampleRate, when provided, must be a positive integer.",
        );
      if (
        i?.group !== void 0 &&
        !(
          i.group instanceof Ze ||
          (Array.isArray(i.group) && i.group.every((t) => t instanceof Ze))
        )
      )
        throw new TypeError(
          "options.audio.group, when provided, must be an OutputTrackGroup or an array of OutputTrackGroups.",
        );
    },
    Vs = 2,
    Ls = 48e3;
  class sn {
    get state() {
      return this._state;
    }
    static async init(e) {
      const t = new sn(e);
      return (await t._init(), t);
    }
    constructor(e) {
      if (
        ((this._state = "idle"),
        (this._timestampOffset = 0),
        (this._timestampOffsetAdjusted = !1),
        (this._copyTimestampPossible = new Map()),
        (this._copyStartPackets = new Map()),
        (this._nextOutputTrackId = 0),
        (this._outputTrackIds = []),
        (this._outputOwnTrackGroups = []),
        (this._trackPumps = []),
        (this._composable = !1),
        (this._executed = !1),
        (this._executionUntil = 1 / 0),
        (this._pauseRequested = !1),
        (this._synchronizer = new tm(this)),
        (this._totalDuration = null),
        (this._maxTimestamps = new Map()),
        (this.onProgress = void 0),
        (this._computeProgress = !1),
        (this._lastProgress = 0),
        (this.isValid = !1),
        (this.utilizedTracks = []),
        (this.discardedTracks = []),
        !e || typeof e != "object")
      )
        throw new TypeError("options must be an object.");
      if (!(e.input instanceof ri))
        throw new TypeError("options.input must be an Input.");
      if (!(e.output instanceof en))
        throw new TypeError("options.output must be an Output.");
      if (e.tracks !== void 0 && e.tracks !== "all" && e.tracks !== "primary")
        throw new TypeError(
          "options.tracks, when provided, must be either 'all' or 'primary'.",
        );
      if (e.composable !== void 0 && typeof e.composable != "boolean")
        throw new TypeError(
          "options.composable, when provided, must be a boolean.",
        );
      if (e.copy !== void 0 && e.copy !== !1) {
        if (!e.copy || typeof e.copy != "object")
          throw new TypeError(
            "options.copy, when provided, must be an object or false.",
          );
        if (
          e.copy.mode !== void 0 &&
          !["forced", "preferred"].includes(e.copy.mode)
        )
          throw new TypeError(
            "options.copy.mode, when provided, must be 'forced' or 'preferred'.",
          );
        if (
          e.copy.shiftTolerance !== void 0 &&
          (!gt(e.copy.shiftTolerance) || e.copy.shiftTolerance < 0)
        )
          throw new TypeError(
            "options.copy.shiftTolerance, when provided, must be a non-negative number.",
          );
        if (
          e.copy.boundaryPolicy !== void 0 &&
          !["expand", "shrink"].includes(e.copy.boundaryPolicy)
        )
          throw new TypeError(
            "options.copy.boundaryPolicy, when provided, must be 'expand' or 'shrink'.",
          );
        if (
          e.copy.boundaryTolerance !== void 0 &&
          (!gt(e.copy.boundaryTolerance) || e.copy.boundaryTolerance < 0)
        )
          throw new TypeError(
            "options.copy.boundaryTolerance, when provided, must be a non-negative number.",
          );
      }
      const t = e.composable ?? !1;
      if (t) {
        if (e.tags !== void 0)
          throw new TypeError(
            "options.tags cannot be set by a composable conversion; set metadata directly on the output instead.",
          );
        if (e.output.state !== "pending")
          throw new TypeError("options.output must not have been started yet.");
      } else if (
        e.output.tracks.length > 0 ||
        Object.keys(e.output._metadataTags).length > 0 ||
        e.output.state !== "pending"
      )
        throw new TypeError(
          "options.output must be fresh: no tracks or metadata tags added and not started.",
        );
      if (e.video !== void 0 && typeof e.video != "function")
        if (Array.isArray(e.video)) for (const r of e.video) rn(r);
        else rn(e.video);
      if (e.audio !== void 0 && typeof e.audio != "function")
        if (Array.isArray(e.audio)) for (const r of e.audio) nn(r);
        else nn(e.audio);
      if (e.trim !== void 0 && (!e.trim || typeof e.trim != "object"))
        throw new TypeError("options.trim, when provided, must be an object.");
      if (e.trim?.start !== void 0 && !Number.isFinite(e.trim.start))
        throw new TypeError(
          "options.trim.start, when provided, must be a finite number.",
        );
      if (e.trim?.end !== void 0 && !gt(e.trim.end))
        throw new TypeError(
          "options.trim.end, when provided, must be a number.",
        );
      if (
        e.trim?.start !== void 0 &&
        e.trim.end !== void 0 &&
        e.trim.start >= e.trim.end
      )
        throw new TypeError(
          "options.trim.start must be less than options.trim.end.",
        );
      if (
        e.tags !== void 0 &&
        (typeof e.tags != "object" || !e.tags) &&
        typeof e.tags != "function"
      )
        throw new TypeError(
          "options.tags, when provided, must be an object or a function.",
        );
      if (
        (typeof e.tags == "object" && In(e.tags),
        e.showWarnings !== void 0 && typeof e.showWarnings != "boolean")
      )
        throw new TypeError(
          "options.showWarnings, when provided, must be a boolean.",
        );
      ((this._options = e),
        (this._copyMode = e.copy === !1 ? !1 : (e.copy?.mode ?? "preferred")),
        (this._copyTimestampShiftTolerance =
          e.copy === !1 ? 0 : (e.copy?.shiftTolerance ?? 0)),
        (this._copyBoundaryPolicy =
          e.copy === !1 ? "expand" : (e.copy?.boundaryPolicy ?? "expand")),
        (this._copyBoundaryTolerance =
          e.copy === !1 ? 1 / 0 : (e.copy?.boundaryTolerance ?? 1 / 0)),
        (this._composable = t),
        (this.input = e.input),
        (this.output = e.output));
    }
    async _init() {
      const e = await this.input.getFormat();
      let t,
        r = this._options.tracks;
      if (
        (r === void 0 && (r = e.name.includes("(HLS)") ? "primary" : "all"),
        r === "all")
      )
        t = await this.input.getTracks();
      else if (r === "primary") {
        const l = await this.input.getPrimaryVideoTrack(),
          u = await this.input.getPrimaryAudioTrack();
        t = [l, u].filter((d) => d !== null);
      } else (xe(r), g(!1));
      const n = this.output.format.getSupportedTrackCounts();
      let s = 1,
        a = 1;
      const o = [],
        c = [];
      for (const l of t) {
        let u;
        if (l.isVideoTrack())
          if (this._options.video)
            if (typeof this._options.video == "function") {
              const h = (await this._options.video(l, s)) ?? {};
              if (Array.isArray(h)) for (const p of h) rn(p);
              else rn(h);
              ((u = Array.isArray(h) ? h : [h]), s++);
            } else
              u = Array.isArray(this._options.video)
                ? this._options.video
                : [this._options.video];
          else u = [{}];
        else if (l.isAudioTrack())
          if (this._options.audio)
            if (typeof this._options.audio == "function") {
              const h = (await this._options.audio(l, a)) ?? {};
              if (Array.isArray(h)) for (const p of h) nn(p);
              else nn(h);
              ((u = Array.isArray(h) ? h : [h]), a++);
            } else
              u = Array.isArray(this._options.audio)
                ? this._options.audio
                : [this._options.audio];
          else u = [{}];
        else g(!1);
        const d = u.filter((h) => h.discard);
        for (const h of d)
          this.discardedTracks.push({
            track: l,
            reason: "discarded_by_user",
            trackOptions: h,
          });
        if (u.length === d.length) {
          u.length === 0 &&
            this.discardedTracks.push({
              track: l,
              reason: "discarded_by_user",
              trackOptions: {},
            });
          continue;
        }
        const f = u.filter((h) => !h.discard);
        (o.push(l), c.push(f));
      }
      (this._options.trim?.start !== void 0
        ? (this._startTimestamp = this._options.trim.start)
        : (this._startTimestamp = Math.max(
            await this.input.getFirstTimestamp(o),
            0,
          )),
        (this._endTimestamp = Math.max(
          this._options.trim?.end ?? 1 / 0,
          this._startTimestamp,
        )),
        (this._timestampOffset = -this._startTimestamp));
      for (let l = 0; l < o.length; l++) {
        const u = o[l],
          d = c[l];
        for (const f of d) {
          if (this.output.tracks.length === n.total.max) {
            this.discardedTracks.push({
              track: u,
              reason: "max_track_count_reached",
              trackOptions: f,
            });
            continue;
          }
          if (
            this.output.tracks.reduce(
              (m, y) => m + (y.type === u.type ? 1 : 0),
              0,
            ) === n[u.type].max
          ) {
            this.discardedTracks.push({
              track: u,
              reason: "max_track_count_of_type_reached",
              trackOptions: f,
            });
            continue;
          }
          const p = this._nextOutputTrackId++;
          u.isVideoTrack()
            ? await this._processVideoTrack(u, f, p)
            : u.isAudioTrack()
              ? await this._processAudioTrack(u, f, p)
              : g(!1);
        }
      }
      for (let l = 0; l < this.utilizedTracks.length - 1; l++)
        for (let u = l + 1; u < this.utilizedTracks.length; u++) {
          const d = this.utilizedTracks[l],
            f = this.utilizedTracks[u],
            h = this._outputOwnTrackGroups[l],
            p = this._outputOwnTrackGroups[u];
          (g(h !== void 0),
            g(p !== void 0),
            h && p && d.canBePairedWith(f) && h.pairWith(p));
        }
      if (!this._composable) {
        const l = await this.input.getMetadataTags();
        let u;
        if (this._options.tags) {
          const h =
            typeof this._options.tags == "function"
              ? await this._options.tags(l)
              : this._options.tags;
          (In(h), (u = h));
        } else u = l;
        const d = e.mimeType === this.output.format.mimeType;
        (l.raw === u.raw && !d && delete u.raw, this.output.setMetadataTags(u));
      }
      if (
        (this._composable
          ? (this.isValid = !0)
          : (this.isValid =
              this.output.hasEnoughTracks() && this.output.tracks.length > 0),
        this._options.showWarnings ?? !0)
      ) {
        const l = [],
          u = this.discardedTracks.filter(
            (d) => d.reason !== "discarded_by_user",
          );
        (u.length > 0 &&
          l.push("Some tracks had to be discarded from the conversion:", u),
          this.isValid ||
            (l.length > 0 &&
              l.push(`

`),
            l.push(this._getInvalidityExplanation().join(""))),
          l.length > 0 && L._warn(...l));
      }
    }
    _getInvalidityExplanation() {
      const e = [];
      if (this.discardedTracks.length === 0)
        e.push("Due to missing tracks, this conversion cannot be executed.");
      else {
        const t =
          this.discardedTracks.every(
            (r) =>
              r.reason === "discarded_by_user" ||
              r.reason === "no_encodable_target_codec",
          ) &&
          this.discardedTracks.some(
            (r) => r.reason === "no_encodable_target_codec",
          );
        if (
          (e.push(
            "Due to discarded tracks, this conversion cannot be executed.",
          ),
          t)
        ) {
          const r = this.discardedTracks.flatMap((s) => {
              if (s.reason === "discarded_by_user") return [];
              let a;
              return (
                s.track.type === "video"
                  ? (a = this.output.format.getSupportedVideoCodecs())
                  : s.track.type === "audio"
                    ? (a = this.output.format.getSupportedAudioCodecs())
                    : (a = this.output.format.getSupportedSubtitleCodecs()),
                a.filter(
                  (o) => !s.trackOptions.codec || o === s.trackOptions.codec,
                )
              );
            }),
            n = [...new Set(r)];
          (n.length === 1
            ? e.push(`
Tracks were discarded because your environment is not able to encode '${n[0]}' with the provided parameters.`)
            : e.push(`
Tracks were discarded because your environment is not able to encode any of the codecs ${n.map((s) => `'${s}'`).join(", ")} with the provided parameters.`),
            n.includes("mp3") &&
              e.push(`
The @mediabunny/mp3-encoder extension package provides support for encoding MP3.`),
            n.includes("aac") &&
              e.push(`
The @mediabunny/aac-encoder extension package provides support for encoding AAC.`),
            (n.includes("ac3") || n.includes("eac3")) &&
              e.push(`
The @mediabunny/ac3 extension package provides support for encoding and decoding AC-3/E-AC-3.`),
            n.includes("flac") &&
              e.push(`
The @mediabunny/flac-encoder extension package provides support for encoding FLAC.`));
        } else
          e.push(`
Check the discardedTracks field for more info.`);
      }
      return e;
    }
    async execute(e = {}) {
      if (!e || typeof e != "object")
        throw new TypeError("options must be an object.");
      if (
        e.until !== void 0 &&
        (typeof e.until != "number" || Number.isNaN(e.until))
      )
        throw new TypeError("options.until, when provided, must be a number.");
      if (e.pauseSignal !== void 0 && !(e.pauseSignal instanceof AbortSignal))
        throw new TypeError(
          "options.pauseSignal, when provided, must be an AbortSignal.",
        );
      if (!this.isValid)
        throw new Error(
          `Cannot execute this conversion because its output configuration is invalid. Make sure to always check the isValid field before executing a conversion.
` + this._getInvalidityExplanation().join(""),
        );
      if (this._state === "executing")
        throw new Error(
          "Cannot call execute() while a previous call to execute() is still running.",
        );
      if (this._state === "canceled") throw new el();
      if (this._state === "done") return;
      if (this._composable && this.output.state === "pending")
        throw new Error(
          "A composable conversion requires the output to be started. Call start() on the output before executing the conversion.",
        );
      ((this._state = "executing"),
        (this._executionUntil = e.until ?? 1 / 0),
        (this._pauseRequested = e.pauseSignal?.aborted ?? !1));
      const t = () => {
        this._state === "executing" &&
          ((this._pauseRequested = !0), this._synchronizer.resolveAll());
      };
      e.pauseSignal?.addEventListener("abort", t);
      for (const n of this._trackPumps) n.done || (n.resolvers = ce());
      if (this._executed) for (const n of this._trackPumps) n.wake?.();
      else {
        this._executed = !0;
        for (const n of this._outputTrackIds)
          this._synchronizer.declareTrack(n);
        if (this.onProgress) {
          const s = [...new Set(this.utilizedTracks)].map(async (o) =>
              (await o.isLive())
                ? 1 / 0
                : ((await o.getDurationFromMetadata()) ??
                  (await o.computeDuration())),
            ),
            a = Math.max(0, ...(await Promise.all(s)));
          ((this._computeProgress = !0),
            (this._totalDuration = Math.min(
              a - this._startTimestamp,
              this._endTimestamp - this._startTimestamp,
            )));
          for (const o of this._outputTrackIds) this._maxTimestamps.set(o, 0);
          this.onProgress?.(0, 0);
        }
        this._composable || (await this.output.start());
        for (const n of this._trackPumps) n.start();
      }
      try {
        await Promise.all(this._trackPumps.map((n) => n.resolvers.promise));
      } catch (n) {
        throw (this._state !== "canceled" && this.cancel(), n);
      } finally {
        e.pauseSignal?.removeEventListener("abort", t);
      }
      if (this._state === "canceled") throw new el();
      const r = this._trackPumps.every((n) => n.done);
      if (
        ((this._state = r ? "done" : "idle"),
        r &&
          (this._composable || (await this.output.finalize()),
          this._computeProgress))
      ) {
        const n = Math.min(...this._maxTimestamps.values());
        this.onProgress?.(1, n);
      }
    }
    async cancel() {
      if (this._state !== "done") {
        if (this._state === "canceled") {
          L._warn("Conversion already canceled.");
          return;
        }
        this._state = "canceled";
        for (const e of this._trackPumps) e.wake?.();
        (this._synchronizer.resolveAll(),
          this._composable || (await this.output.cancel()));
      }
    }
    async _processVideoTrack(e, t, r) {
      const n = await e.getCodec();
      if (!n) {
        this.discardedTracks.push({
          track: e,
          reason: "unknown_source_codec",
          trackOptions: t,
        });
        return;
      }
      let s;
      const a = await e.getRotation(),
        o = await e.getFlip(),
        { rotation: c, flip: l } = Gs(a, o, t.rotate ?? 0, t.flip ?? !1);
      let u = c,
        d = l,
        f = t.rotate || t.flip ? null : await e.getTransformationMatrix();
      const h =
          this.output.format.supportsVideoTransformationMetadata &&
          (t.allowTransformationMetadata ?? t.allowRotationMetadata ?? !0),
        p = await e.getSquarePixelWidth(),
        m = await e.getSquarePixelHeight(),
        [y, w] = c % 180 === 0 ? [p, m] : [m, p];
      let b = t.crop;
      b && (b = fs(b, y, w));
      const [T, S] = b ? [b.width, b.height] : [y, w];
      let k = T,
        P = S;
      const I = k / P;
      t.width !== void 0 && t.height === void 0
        ? ((k = fi(t.width)), (P = fi(Math.round(k / I))))
        : t.width === void 0 && t.height !== void 0
          ? ((P = fi(t.height)), (k = fi(Math.round(P * I))))
          : t.width !== void 0 &&
            t.height !== void 0 &&
            ((k = fi(t.width)), (P = fi(t.height)));
      let E = this.output.format.getSupportedVideoCodecs();
      const C = t.alpha ?? "discard";
      let _ =
          !this._copyMode ||
          !!t.forceTranscode ||
          !!t.frameRate ||
          t.keyFrameInterval !== void 0 ||
          t.process !== void 0 ||
          t.quality !== void 0 ||
          t.bitrate !== void 0 ||
          !E.includes(n) ||
          (t.codec && t.codec !== n) ||
          k !== T ||
          P !== S ||
          (c !== 0 && !h) ||
          (l && !h) ||
          !!b,
        v = null;
      if (!_) {
        const Y = new zt(e);
        let G =
          (await Y.getKeyPacket(this._startTimestamp, {
            verifyKeyPackets: !0,
          })) ?? (await Y.getFirstKeyPacket({ verifyKeyPackets: !0 }));
        if (
          (G &&
            G.timestamp < this._startTimestamp &&
            G.timestamp + G.duration <= this._startTimestamp &&
            this._copyBoundaryPolicy === "shrink" &&
            (G = await Y.getNextKeyPacket(G, { verifyKeyPackets: !0 })),
          (v = G),
          G)
        ) {
          const ie =
            this._copyBoundaryPolicy === "shrink"
              ? Math.max(G.timestamp, this._startTimestamp)
              : G.timestamp;
          if (
            (this._copyBoundaryPolicy === "shrink"
              ? ie - this._startTimestamp
              : Math.max(this._startTimestamp - ie, 0)) >
            this._copyBoundaryTolerance
          )
            _ = !0;
          else if (this.output.format.supportsTimestampedMediaData) {
            if (
              this.output.format.negativeTimestampSupport !== "full" &&
              ie < this._startTimestamp
            ) {
              const we = Math.min(
                this._startTimestamp - ie,
                this._copyTimestampShiftTolerance,
              );
              ie + we >= this._startTimestamp ||
              (this.output.format.negativeTimestampSupport ===
                "prefer-non-negative" &&
                this._copyMode === "forced")
                ? (this._timestampOffset = Math.max(
                    this._timestampOffset,
                    -this._startTimestamp + we,
                  ))
                : (_ = !0);
            }
          } else if (this._timestampOffsetAdjusted)
            ie + this._timestampOffset === 0 || (_ = !0);
          else {
            const we = se(
              this._startTimestamp - ie,
              -this._copyTimestampShiftTolerance,
              this._copyTimestampShiftTolerance,
            );
            ie + we === this._startTimestamp
              ? ((this._timestampOffset = -this._startTimestamp + we),
                (this._timestampOffsetAdjusted = !0))
              : (_ = !0);
          }
        }
      }
      if (_ && this._copyMode === "forced") {
        this.discardedTracks.push({
          track: e,
          reason: "cannot_copy",
          trackOptions: t,
        });
        return;
      }
      if (_) {
        if (!(await e.canDecode())) {
          this.discardedTracks.push({
            track: e,
            reason: "undecodable_source_codec",
            trackOptions: t,
          });
          return;
        }
        t.codec && (E = E.filter((ye) => ye === t.codec));
        const G = Ei(t.quality, t.bitrate) ?? new ke("high"),
          ie = await Xd(E, {
            width: t.process && t.processedWidth ? t.processedWidth : k,
            height: t.process && t.processedHeight ? t.processedHeight : P,
            quality: G,
          });
        if (!ie) {
          this.discardedTracks.push({
            track: e,
            reason: "no_encodable_target_codec",
            trackOptions: t,
          });
          return;
        }
        const re = {
          codec: ie,
          quality: G,
          keyFrameInterval: t.keyFrameInterval,
          sizeChangeBehavior: t.fit ?? "passThrough",
          alpha: C,
          hardwareAcceleration: t.hardwareAcceleration,
          transform: {},
        };
        g(re.transform);
        let we =
          k !== T ||
          P !== S ||
          (c !== 0 && (!h || t.process !== void 0)) ||
          (l && (!h || t.process !== void 0)) ||
          !!b ||
          p !== (await e.getCodedWidth()) ||
          m !== (await e.getCodedHeight());
        if (!we) {
          const ye = { stack: [], error: void 0, hasError: !1 };
          try {
            const Be = new en({ format: new Us(), target: new vh() }),
              Wt = new Xc(re);
            (Be.addVideoTrack(Wt), await Be.start());
            const qe = new Ic(e),
              Se = br(ye, await qe.getSample(await e.getFirstTimestamp()), !1);
            if (Se)
              try {
                (await Wt.add(Se), Se.close(), await Be.finalize());
              } catch (Vt) {
                (L._warn(
                  "An error occurred when probing encoder support. Falling back to rerender path.",
                  Vt,
                ),
                  Be.cancel(),
                  (we = !0),
                  (re.transform.force = !0));
              }
            else await Be.cancel();
          } catch (Be) {
            ((ye.error = Be), (ye.hasError = !0));
          } finally {
            tn(ye);
          }
        }
        (t.frameRate && (re.transform.frameRate = t.frameRate),
          t.process && (re.transform.process = t.process),
          we &&
            ((u = 0),
            (d = !1),
            (f = null),
            (re.transform.width = k),
            (re.transform.height = P),
            (re.transform.fit = t.fit ?? "fill"),
            (re.transform.rotate = t.rotate),
            (re.transform.flip = t.flip),
            (re.transform.crop = b),
            (re.transform.alpha = C)));
        let De = null;
        re.onEncodedSample = (ye) => {
          De = ye.timestamp;
        };
        const Ae = new Xc(re);
        ((s = Ae),
          this._registerTrackPump(async (ye) => {
            const Be = new Ic(e);
            for await (const Wt of Be.samples(
              this._startTimestamp,
              this._endTimestamp,
            )) {
              const qe = { stack: [], error: void 0, hasError: !1 };
              try {
                const Se = br(qe, Wt, !1);
                if (this._state === "canceled") break;
                const Vt = Math.max(this._startTimestamp, Se.timestamp),
                  an = Math.min(this._endTimestamp, Se.timestamp + Se.duration);
                if (Vt >= an) continue;
                (Se.setTimestamp(Vt + this._timestampOffset),
                  Se.setDuration(an - Vt),
                  this._reportProgress(r, Se.timestamp + Se.duration),
                  await Ae.add(Se),
                  Se.close(),
                  De !== null &&
                    (this._synchronizer.shouldWait(r, De) &&
                      (await this._synchronizer.wait(De)),
                    await this._checkpoint(ye, De)));
              } catch (Se) {
                ((qe.error = Se), (qe.hasError = !0));
              } finally {
                tn(qe);
              }
            }
            (Ae.close(), this._synchronizer.closeTrack(r));
          }));
      } else {
        const Y = new Hh(n);
        ((s = Y),
          this._registerTrackPump(async (G) => {
            const ie = new zt(e),
              we = { decoderConfig: (await e.getDecoderConfig()) ?? void 0 };
            let De = null;
            if (v)
              for await (const Ae of ie.packets(v, void 0, {
                verifyKeyPackets: !0,
              })) {
                if (this._state === "canceled") break;
                if (Ae.timestamp >= this._endTimestamp) {
                  if (this._copyBoundaryPolicy === "shrink") break;
                  {
                    let Se = Ae,
                      Vt = !1;
                    const an = 6;
                    for (let il = 0; il < an; il++) {
                      const Hs = await ie.getNextPacket(Se, {
                        metadataOnly: !0,
                      });
                      if (!Hs) break;
                      if (Hs.timestamp < this._endTimestamp) {
                        Vt = !0;
                        break;
                      }
                      Se = Hs;
                    }
                    if (!Vt) break;
                  }
                }
                let ye = Ae.timestamp,
                  Be = Ae.timestamp + Ae.duration;
                (this._copyBoundaryPolicy === "shrink" &&
                  ((ye = Math.max(ye, this._startTimestamp)),
                  (Be = Math.min(Be, this._endTimestamp)),
                  (Be = Math.max(Be, ye))),
                  (ye += this._timestampOffset),
                  (Be += this._timestampOffset));
                let Wt = Ae.type;
                (Wt === "key" && De !== null && ye < De && (Wt = "delta"),
                  (De = Math.max(De ?? -1 / 0, ye)));
                const qe = Ae.clone({
                  timestamp: ye,
                  duration: Be - ye,
                  sideData: C === "discard" ? {} : Ae.sideData,
                  type: Wt,
                });
                (this._reportProgress(r, qe.timestamp + qe.duration),
                  await Y.add(qe, we),
                  this._synchronizer.shouldWait(r, qe.timestamp) &&
                    (await this._synchronizer.wait(qe.timestamp)),
                  await this._checkpoint(G, qe.timestamp));
              }
            (Y.close(), this._synchronizer.closeTrack(r));
          }));
      }
      let F = null;
      !t.group && !this._composable && (F = new Ze());
      const D = await e.getLanguageCode(),
        z = await e.getName(),
        O = await e.getDisposition(),
        H = _ ? null : await e.getBitrate(),
        ee = _ ? null : await e.getAverageBitrate(),
        he = C === "keep" && (await e.canBeTransparent());
      (this.output.addVideoTrack(s, {
        frameRate: t.frameRate,
        languageCode: Ni(D) ? D : void 0,
        name: z ?? void 0,
        disposition: O,
        rotation: u,
        flip: d,
        transformationMatrix: f ?? void 0,
        canBeTransparent: he,
        group: F ?? t.group,
        bitrate: H ?? void 0,
        averageBitrate: ee ?? void 0,
      }),
        this.utilizedTracks.push(e),
        this._outputTrackIds.push(r),
        this._outputOwnTrackGroups.push(F));
    }
    async _processAudioTrack(e, t, r) {
      const n = await e.getCodec();
      if (!n) {
        this.discardedTracks.push({
          track: e,
          reason: "unknown_source_codec",
          trackOptions: t,
        });
        return;
      }
      let s;
      const a = await e.getNumberOfChannels(),
        o = await e.getSampleRate();
      let c = t.numberOfChannels ?? a,
        l = t.sampleRate ?? o,
        u = this.output.format.getSupportedAudioCodecs(),
        d =
          !this._copyMode ||
          !!t.forceTranscode ||
          !!t.quality ||
          !!t.bitrate ||
          c !== a ||
          l !== o ||
          !u.includes(n) ||
          (!!t.codec && t.codec !== n) ||
          t.process !== void 0 ||
          t.sampleFormat !== void 0,
        f = null;
      if (!d) {
        const T = new zt(e);
        let S =
          (await T.getKeyPacket(this._startTimestamp)) ??
          (await T.getFirstKeyPacket());
        S &&
          ((this._copyBoundaryPolicy === "shrink" &&
            S.timestamp < this._startTimestamp) ||
            (this._copyBoundaryPolicy === "expand" &&
              S.timestamp + S.duration <= this._startTimestamp)) &&
          (S = await T.getNextKeyPacket(S));
        const k = Bt.includes(n) && n !== "flac";
        if (S && this._copyBoundaryPolicy === "expand" && k) {
          const P = await T.getKeyPacket(
            S.timestamp - 1 / (await e.getTimeResolution()),
          );
          P && (S = P);
        }
        if (((f = S), S))
          if (
            (this._copyBoundaryPolicy === "shrink"
              ? Math.max(S.timestamp - this._startTimestamp, 0)
              : Math.max(this._startTimestamp - S.timestamp, 0)) >
            this._copyBoundaryTolerance
          )
            d = !0;
          else if (this.output.format.supportsTimestampedMediaData) {
            if (
              this.output.format.negativeTimestampSupport !== "full" &&
              S.timestamp < this._startTimestamp
            ) {
              const I = Math.min(
                this._startTimestamp - S.timestamp,
                this._copyTimestampShiftTolerance,
              );
              S.timestamp + I >= this._startTimestamp ||
              (this.output.format.negativeTimestampSupport ===
                "prefer-non-negative" &&
                this._copyMode === "forced")
                ? (this._timestampOffset = Math.max(
                    this._timestampOffset,
                    -this._startTimestamp + I,
                  ))
                : (d = !0);
            }
          } else if (this._timestampOffsetAdjusted)
            S.timestamp + this._timestampOffset === 0 || (d = !0);
          else {
            const I = se(
              this._startTimestamp - S.timestamp,
              -this._copyTimestampShiftTolerance,
              this._copyTimestampShiftTolerance,
            );
            S.timestamp + I === this._startTimestamp
              ? ((this._timestampOffset = -this._startTimestamp + I),
                (this._timestampOffsetAdjusted = !0))
              : (d = !0);
          }
      }
      if (d && this._copyMode === "forced") {
        this.discardedTracks.push({
          track: e,
          reason: "cannot_copy",
          trackOptions: t,
        });
        return;
      }
      if (d) {
        if (!(await e.canDecode())) {
          this.discardedTracks.push({
            track: e,
            reason: "undecodable_source_codec",
            trackOptions: t,
          });
          return;
        }
        let S = null;
        t.codec && (u = u.filter((_) => _ === t.codec));
        const k = Ei(t.quality, t.bitrate) ?? new ke("high"),
          P = await bc(u, {
            numberOfChannels:
              t.process && t.processedNumberOfChannels
                ? t.processedNumberOfChannels
                : c,
            sampleRate:
              t.process && t.processedSampleRate ? t.processedSampleRate : l,
            quality: k,
          });
        if (
          !P.some((_) => Bt.includes(_)) &&
          u.some((_) => Bt.includes(_)) &&
          (c !== Vs || l !== Ls)
        ) {
          const v = (
            await bc(u, { numberOfChannels: Vs, sampleRate: Ls, quality: k })
          ).find((F) => Bt.includes(F));
          v && ((S = v), (c = Vs), (l = Ls));
        } else S = P[0] ?? null;
        if (S === null) {
          this.discardedTracks.push({
            track: e,
            reason: "no_encodable_target_codec",
            trackOptions: t,
          });
          return;
        }
        const I = {
          codec: S,
          quality: k,
          transform: { sampleFormat: t.sampleFormat, process: t.process },
        };
        (g(I.transform),
          c !== a && (I.transform.numberOfChannels = c),
          l !== o && (I.transform.sampleRate = l));
        let E = null;
        I.onEncodedSample = (_) => {
          E = _.timestamp;
        };
        const C = new Xh(I);
        ((s = C),
          this._registerTrackPump(async (_) => {
            let v = null;
            const F = new of(e);
            for await (const D of F.samples(
              this._startTimestamp,
              this._endTimestamp,
            )) {
              const z = { stack: [], error: void 0, hasError: !1 };
              try {
                const O = br(z, D, !1);
                if (this._state === "canceled") break;
                let H = 0,
                  ee = O.numberOfFrames;
                if (
                  (O.timestamp < this._startTimestamp &&
                    (H = Math.round(
                      (this._startTimestamp - O.timestamp) * O.sampleRate,
                    )),
                  O.timestamp + O.duration > this._endTimestamp &&
                    (ee = Math.round(
                      (this._endTimestamp - O.timestamp) * O.sampleRate,
                    )),
                  H >= ee)
                ) {
                  O.close();
                  continue;
                }
                let he;
                if (H > 0 || ee < O.numberOfFrames) {
                  const G = O.trim(H, ee);
                  if ((O.close(), (he = G), G.numberOfFrames === 0)) {
                    G.close();
                    continue;
                  }
                } else he = O;
                const Y = br(z, he, !1);
                if (
                  (Y.setTimestamp(Y.timestamp + this._timestampOffset),
                  v === null &&
                    (v =
                      Y.timestamp > 0 &&
                      !this.output.format.supportsTimestampedMediaData),
                  v)
                ) {
                  const G = { stack: [], error: void 0, hasError: !1 };
                  try {
                    const ie = Y.timestamp,
                      re = Math.round(ie * o),
                      we = Ct(O.format),
                      De = new Uint8Array(we * re * a);
                    (O.format === "u8" || O.format === "u8-planar") &&
                      De.fill(2 ** 7);
                    const Ae = br(
                      G,
                      new Ie({
                        data: De,
                        format: O.format,
                        numberOfChannels: a,
                        sampleRate: o,
                        timestamp: 0,
                      }),
                      !1,
                    );
                    (await this._registerAudioSample(_, Ae, C, r, () => E),
                      (v = !1));
                  } catch (ie) {
                    ((G.error = ie), (G.hasError = !0));
                  } finally {
                    tn(G);
                  }
                }
                await this._registerAudioSample(_, Y, C, r, () => E);
              } catch (O) {
                ((z.error = O), (z.hasError = !0));
              } finally {
                tn(z);
              }
            }
            (C.close(), this._synchronizer.closeTrack(r));
          }));
      } else {
        const T = new Kh(n);
        ((s = T),
          this._registerTrackPump(async (S) => {
            const k = new zt(e),
              I = { decoderConfig: (await e.getDecoderConfig()) ?? void 0 };
            let E = null;
            if (f)
              for await (const C of k.packets(f)) {
                if (
                  this._state === "canceled" ||
                  C.timestamp >= this._endTimestamp ||
                  (this._copyBoundaryPolicy === "shrink" &&
                    C.timestamp + C.duration > this._endTimestamp)
                )
                  break;
                const _ = C.timestamp + this._timestampOffset;
                if (E !== null && _ < E) continue;
                E = _;
                const v = C.clone({ timestamp: _, duration: C.duration });
                (this._reportProgress(r, v.timestamp + v.duration),
                  await T.add(v, I),
                  this._synchronizer.shouldWait(r, v.timestamp) &&
                    (await this._synchronizer.wait(v.timestamp)),
                  await this._checkpoint(S, v.timestamp));
              }
            (T.close(), this._synchronizer.closeTrack(r));
          }));
      }
      let h = null;
      !t.group && !this._composable && (h = new Ze());
      const p = await e.getLanguageCode(),
        m = await e.getName(),
        y = await e.getDisposition(),
        w = d ? null : await e.getBitrate(),
        b = d ? null : await e.getAverageBitrate();
      (this.output.addAudioTrack(s, {
        languageCode: Ni(p) ? p : void 0,
        name: m ?? void 0,
        disposition: y,
        group: h ?? t.group,
        bitrate: w ?? void 0,
        averageBitrate: b ?? void 0,
      }),
        this.utilizedTracks.push(e),
        this._outputTrackIds.push(r),
        this._outputOwnTrackGroups.push(h));
    }
    async _registerAudioSample(e, t, r, n, s) {
      (this._reportProgress(n, t.timestamp + t.duration),
        await r.add(t),
        t.close());
      const a = s();
      a !== null &&
        (this._synchronizer.shouldWait(n, a) &&
          (await this._synchronizer.wait(a)),
        await this._checkpoint(e, a));
    }
    _registerTrackPump(e) {
      const t = {
        done: !1,
        resolvers: ce(),
        wake: null,
        start: () => {
          e(t).then(
            () => {
              ((t.done = !0), t.resolvers.resolve());
            },
            (r) => {
              t.resolvers.reject(r);
            },
          );
        },
      };
      this._trackPumps.push(t);
    }
    async _checkpoint(e, t) {
      for (
        ;
        this._state !== "canceled" &&
        (t >= this._executionUntil || this._pauseRequested);
      ) {
        e.resolvers.resolve();
        const { promise: r, resolve: n } = ce();
        ((e.wake = n), await r);
      }
    }
    _reportProgress(e, t) {
      if (!this._computeProgress) return;
      (g(this._totalDuration !== null),
        this._maxTimestamps.set(e, Math.max(t, this._maxTimestamps.get(e))));
      const r = Math.min(...this._maxTimestamps.values()),
        n = se(r / this._totalDuration, 0, 1);
      n !== this._lastProgress &&
        ((this._lastProgress = n), this.onProgress?.(n, r));
    }
  }
  class el extends Error {
    constructor(e = "Conversion has been canceled.") {
      (super(e), (this.name = "ConversionCanceledError"));
    }
  }
  const tl = 1;
  class tm {
    constructor(e) {
      ((this.maxTimestamps = new Map()),
        (this.resolvers = []),
        (this.conversion = e));
    }
    declareTrack(e) {
      this.maxTimestamps.set(e, -1 / 0);
    }
    shouldWait(e, t) {
      const r = this.maxTimestamps.get(e);
      (g(r !== void 0), this.maxTimestamps.set(e, Math.max(t, r)));
      const n = this.computeMinAndMaybeResolve();
      return this.conversion._state === "canceled" ||
        this.conversion._pauseRequested ||
        t >= this.conversion._executionUntil
        ? !1
        : t - n > tl;
    }
    wait(e) {
      const { promise: t, resolve: r } = ce();
      return (this.resolvers.push({ timestamp: e, resolve: r }), t);
    }
    closeTrack(e) {
      (this.maxTimestamps.delete(e), this.computeMinAndMaybeResolve());
    }
    resolveAll() {
      for (const e of this.resolvers) e.resolve();
      this.resolvers.length = 0;
    }
    computeMinAndMaybeResolve() {
      let e = 1 / 0;
      for (const [, t] of this.maxTimestamps) e = Math.min(e, t);
      for (let t = 0; t < this.resolvers.length; t++) {
        const r = this.resolvers[t];
        r.timestamp - e < tl && (r.resolve(), this.resolvers.splice(t, 1), t--);
      }
      return e;
    }
  }
  function im(i) {
    if (i[0] === 26 && i[1] === 69 && i[2] === 223 && i[3] === 163)
      return "video/webm";
    if (i[4] === 102 && i[5] === 116 && i[6] === 121 && i[7] === 112)
      return "video/mp4";
  }
  self.onmessage = async (i) => {
    const e = new ri({ source: new cs(i.data.blob), formats: ls });
    try {
      const t = im(
        new Uint8Array(await i.data.blob.slice(0, 12).arrayBuffer()),
      );
      if (!t) throw new Error("仅支持 MP4 或 WebM 视频分离");
      const r = t === "video/webm",
        n = await e.getPrimaryAudioTrack();
      if (!n) throw new Error("视频中没有可分离的音轨");
      const s = await e.getPrimaryVideoTrack(),
        a = await Promise.all([n.getFirstTimestamp(), s?.getFirstTimestamp()]),
        o = Math.max(0, Math.min(...a.filter((S) => S !== void 0))),
        l =
          ((await e.getDurationFromMetadata([n])) ??
            (await e.computeDuration([n]))) - o,
        u = Math.max(0, i.data.startSeconds ?? 0),
        d = Math.min(l, i.data.endSeconds ?? l);
      if (!Number.isFinite(d) || d <= u) throw new Error("音频时间范围无效");
      const f = new wr(),
        h = new en({ format: new Yh(), target: f }),
        p = await sn.init({
          input: e,
          output: h,
          tracks: "primary",
          video: { discard: !0 },
          audio: { codec: "pcm-s16" },
          trim: u > 0 || d < l ? { start: o + u, end: o + d } : void 0,
          tags: {},
          showWarnings: !1,
        });
      if (!p.isValid) {
        const S = p.discardedTracks.some(
          (k) => k.reason === "undecodable_source_codec",
        );
        throw new Error(
          S
            ? "当前平台不支持解码该视频的音频编码"
            : "该视频的音频编码暂不支持分离",
        );
      }
      if ((await p.execute(), !f.buffer))
        throw new Error("音频分离没有生成结果");
      const m = new ri({ source: new cs(new Blob([f.buffer])), formats: ls });
      let y = 0;
      try {
        const S = await m.getPrimaryAudioTrack();
        if (!S) throw new Error("音频分离没有生成有效音轨");
        const k = Math.max(0, await S.getFirstTimestamp()),
          P = await m.computeDuration([S]);
        y = Math.round((P - k) * 1e3);
      } finally {
        m.dispose();
      }
      if (!Number.isFinite(y) || y <= 0)
        throw new Error("音频分离没有生成有效内容");
      const w = new ri({ source: new cs(i.data.blob), formats: ls }),
        b = new wr();
      try {
        const S = new en({ format: r ? new Jr() : new Us(), target: b }),
          k = await sn.init({
            input: w,
            output: S,
            tracks: "primary",
            audio: { discard: !0 },
            copy: { mode: "forced", shiftTolerance: 1 / 0 },
            trim: u > 0 || d < l ? { start: o + u, end: o + d } : void 0,
            tags: {},
            showWarnings: !1,
          });
        if (!k.isValid) {
          const P = k.discardedTracks.some((I) => I.reason === "cannot_copy");
          throw new Error(
            P
              ? `该视频编码无法复制到 ${r ? "WebM" : "MP4"} 无声容器`
              : "当前平台暂不支持生成该视频的无声版本",
          );
        }
        await k.execute();
      } finally {
        w.dispose();
      }
      if (!b.buffer) throw new Error("没有生成无声视频");
      const T = {
        audioBuffer: f.buffer,
        videoBuffer: b.buffer,
        videoMime: t,
        durationMs: y,
      };
      self.postMessage(T, { transfer: [f.buffer, b.buffer] });
    } catch (t) {
      const r = { error: t instanceof Error ? t.message : "音频分离失败" };
      self.postMessage(r);
    } finally {
      e.dispose();
    }
  };
})();
