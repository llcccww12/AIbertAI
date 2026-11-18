var B = [
    [768, 879],
    [1155, 1158],
    [1160, 1161],
    [1425, 1469],
    [1471, 1471],
    [1473, 1474],
    [1476, 1477],
    [1479, 1479],
    [1536, 1539],
    [1552, 1557],
    [1611, 1630],
    [1648, 1648],
    [1750, 1764],
    [1767, 1768],
    [1770, 1773],
    [1807, 1807],
    [1809, 1809],
    [1840, 1866],
    [1958, 1968],
    [2027, 2035],
    [2305, 2306],
    [2364, 2364],
    [2369, 2376],
    [2381, 2381],
    [2385, 2388],
    [2402, 2403],
    [2433, 2433],
    [2492, 2492],
    [2497, 2500],
    [2509, 2509],
    [2530, 2531],
    [2561, 2562],
    [2620, 2620],
    [2625, 2626],
    [2631, 2632],
    [2635, 2637],
    [2672, 2673],
    [2689, 2690],
    [2748, 2748],
    [2753, 2757],
    [2759, 2760],
    [2765, 2765],
    [2786, 2787],
    [2817, 2817],
    [2876, 2876],
    [2879, 2879],
    [2881, 2883],
    [2893, 2893],
    [2902, 2902],
    [2946, 2946],
    [3008, 3008],
    [3021, 3021],
    [3134, 3136],
    [3142, 3144],
    [3146, 3149],
    [3157, 3158],
    [3260, 3260],
    [3263, 3263],
    [3270, 3270],
    [3276, 3277],
    [3298, 3299],
    [3393, 3395],
    [3405, 3405],
    [3530, 3530],
    [3538, 3540],
    [3542, 3542],
    [3633, 3633],
    [3636, 3642],
    [3655, 3662],
    [3761, 3761],
    [3764, 3769],
    [3771, 3772],
    [3784, 3789],
    [3864, 3865],
    [3893, 3893],
    [3895, 3895],
    [3897, 3897],
    [3953, 3966],
    [3968, 3972],
    [3974, 3975],
    [3984, 3991],
    [3993, 4028],
    [4038, 4038],
    [4141, 4144],
    [4146, 4146],
    [4150, 4151],
    [4153, 4153],
    [4184, 4185],
    [4448, 4607],
    [4959, 4959],
    [5906, 5908],
    [5938, 5940],
    [5970, 5971],
    [6002, 6003],
    [6068, 6069],
    [6071, 6077],
    [6086, 6086],
    [6089, 6099],
    [6109, 6109],
    [6155, 6157],
    [6313, 6313],
    [6432, 6434],
    [6439, 6440],
    [6450, 6450],
    [6457, 6459],
    [6679, 6680],
    [6912, 6915],
    [6964, 6964],
    [6966, 6970],
    [6972, 6972],
    [6978, 6978],
    [7019, 7027],
    [7616, 7626],
    [7678, 7679],
    [8203, 8207],
    [8234, 8238],
    [8288, 8291],
    [8298, 8303],
    [8400, 8431],
    [12330, 12335],
    [12441, 12442],
    [43014, 43014],
    [43019, 43019],
    [43045, 43046],
    [64286, 64286],
    [65024, 65039],
    [65056, 65059],
    [65279, 65279],
    [65529, 65531],
  ],
  qe = [
    [68097, 68099],
    [68101, 68102],
    [68108, 68111],
    [68152, 68154],
    [68159, 68159],
    [119143, 119145],
    [119155, 119170],
    [119173, 119179],
    [119210, 119213],
    [119362, 119364],
    [917505, 917505],
    [917536, 917631],
    [917760, 917999],
  ],
  g;
function $e(e, t) {
  let r = 0,
    i = t.length - 1,
    s;
  if (e < t[0][0] || e > t[i][1]) return !1;
  for (; i >= r;)
    if (((s = (r + i) >> 1), e > t[s][1])) r = s + 1;
    else if (e < t[s][0]) i = s - 1;
    else return !0;
  return !1;
}
var et = class {
    constructor() {
      if (((this.version = "6"), !g)) {
        ((g = new Uint8Array(65536)),
          g.fill(1),
          (g[0] = 0),
          g.fill(0, 1, 32),
          g.fill(0, 127, 160),
          g.fill(2, 4352, 4448),
          (g[9001] = 2),
          (g[9002] = 2),
          g.fill(2, 11904, 42192),
          (g[12351] = 1),
          g.fill(2, 44032, 55204),
          g.fill(2, 63744, 64256),
          g.fill(2, 65040, 65050),
          g.fill(2, 65072, 65136),
          g.fill(2, 65280, 65377),
          g.fill(2, 65504, 65511));
        for (let e = 0; e < B.length; ++e) g.fill(0, B[e][0], B[e][1] + 1);
      }
    }
    wcwidth(e) {
      return e < 32
        ? 0
        : e < 127
          ? 1
          : e < 65536
            ? g[e]
            : $e(e, qe)
              ? 0
              : (e >= 131072 && e <= 196605) || (e >= 196608 && e <= 262141)
                ? 2
                : 1;
    }
    charProperties(e, t) {
      let r = this.wcwidth(e),
        i = r === 0 && t !== 0;
      if (i) {
        let s = N.extractWidth(t);
        s === 0 ? (i = !1) : s > r && (r = s);
      }
      return N.createPropertyValue(0, r, i);
    }
  },
  tt = class {
    constructor() {
      ((this.listeners = []),
        (this.unexpectedErrorHandler = function (e) {
          setTimeout(() => {
            throw e.stack
              ? _e.isErrorNoTelemetry(e)
                ? new _e(
                    e.message +
                      `

` +
                      e.stack,
                  )
                : new Error(
                    e.message +
                      `

` +
                      e.stack,
                  )
              : e;
          }, 0);
        }));
    }
    addListener(e) {
      return (
        this.listeners.push(e),
        () => {
          this._removeListener(e);
        }
      );
    }
    emit(e) {
      this.listeners.forEach((t) => {
        t(e);
      });
    }
    _removeListener(e) {
      this.listeners.splice(this.listeners.indexOf(e), 1);
    }
    setUnexpectedErrorHandler(e) {
      this.unexpectedErrorHandler = e;
    }
    getUnexpectedErrorHandler() {
      return this.unexpectedErrorHandler;
    }
    onUnexpectedError(e) {
      (this.unexpectedErrorHandler(e), this.emit(e));
    }
    onUnexpectedExternalError(e) {
      this.unexpectedErrorHandler(e);
    }
  },
  rt = new tt();
function Y(e) {
  st(e) || rt.onUnexpectedError(e);
}
var ie = "Canceled";
function st(e) {
  return e instanceof it
    ? !0
    : e instanceof Error && e.name === ie && e.message === ie;
}
var it = class extends Error {
    constructor() {
      (super(ie), (this.name = this.message));
    }
  },
  _e = class ne extends Error {
    constructor(t) {
      (super(t), (this.name = "CodeExpectedError"));
    }
    static fromError(t) {
      if (t instanceof ne) return t;
      let r = new ne();
      return ((r.message = t.message), (r.stack = t.stack), r);
    }
    static isErrorNoTelemetry(t) {
      return t.name === "CodeExpectedError";
    }
  };
function nt(e, t) {
  let r = this,
    i = !1,
    s;
  return function () {
    return (i || ((i = !0), t || (s = e.apply(r, arguments))), s);
  };
}
var ot;
((e) => {
  function t(a) {
    return a < 0;
  }
  e.isLessThan = t;
  function r(a) {
    return a <= 0;
  }
  e.isLessThanOrEqual = r;
  function i(a) {
    return a > 0;
  }
  e.isGreaterThan = i;
  function s(a) {
    return a === 0;
  }
  ((e.isNeitherLessOrGreaterThan = s),
    (e.greaterThan = 1),
    (e.lessThan = -1),
    (e.neitherLessOrGreaterThan = 0));
})((ot ||= {}));
var We;
((e) => {
  function t(d) {
    return d && typeof d == "object" && typeof d[Symbol.iterator] == "function";
  }
  e.is = t;
  let r = Object.freeze([]);
  function i() {
    return r;
  }
  e.empty = i;
  function* s(d) {
    yield d;
  }
  e.single = s;
  function a(d) {
    return t(d) ? d : s(d);
  }
  e.wrap = a;
  function f(d) {
    return d || r;
  }
  e.from = f;
  function* _(d) {
    for (let v = d.length - 1; v >= 0; v--) yield d[v];
  }
  e.reverse = _;
  function m(d) {
    return !d || d[Symbol.iterator]().next().done === !0;
  }
  e.isEmpty = m;
  function L(d) {
    return d[Symbol.iterator]().next().value;
  }
  e.first = L;
  function k(d, v) {
    let w = 0;
    for (let y of d) if (v(y, w++)) return !0;
    return !1;
  }
  e.some = k;
  function z(d, v) {
    for (let w of d) if (v(w)) return w;
  }
  e.find = z;
  function* I(d, v) {
    for (let w of d) v(w) && (yield w);
  }
  e.filter = I;
  function* F(d, v) {
    let w = 0;
    for (let y of d) yield v(y, w++);
  }
  e.map = F;
  function* G(d, v) {
    let w = 0;
    for (let y of d) yield* v(y, w++);
  }
  e.flatMap = G;
  function* Q(...d) {
    for (let v of d) yield* v;
  }
  e.concat = Q;
  function Z(d, v, w) {
    let y = w;
    for (let D of d) y = v(y, D);
    return y;
  }
  e.reduce = Z;
  function* U(d, v, w = d.length) {
    for (
      v < 0 && (v += d.length),
        w < 0 ? (w += d.length) : w > d.length && (w = d.length);
      v < w;
      v++
    )
      yield d[v];
  }
  e.slice = U;
  function j(d, v = Number.POSITIVE_INFINITY) {
    let w = [];
    if (v === 0) return [w, d];
    let y = d[Symbol.iterator]();
    for (let D = 0; D < v; D++) {
      let S = y.next();
      if (S.done) return [w, e.empty()];
      w.push(S.value);
    }
    return [
      w,
      {
        [Symbol.iterator]() {
          return y;
        },
      },
    ];
  }
  e.consume = j;
  async function K(d) {
    let v = [];
    for await (let w of d) v.push(w);
    return Promise.resolve(v);
  }
  e.asyncToArray = K;
})((We ||= {}));
function ke(e) {
  if (We.is(e)) {
    let t = [];
    for (let r of e)
      if (r)
        try {
          r.dispose();
        } catch (i) {
          t.push(i);
        }
    if (t.length === 1) throw t[0];
    if (t.length > 1)
      throw new AggregateError(
        t,
        "Encountered errors while disposing of store",
      );
    return Array.isArray(e) ? [] : e;
  } else if (e) return (e.dispose(), e);
}
function at(...e) {
  return ze(() => ke(e));
}
function ze(e) {
  return {
    dispose: nt(() => {
      e();
    }),
  };
}
var Ue = class De {
  constructor() {
    ((this._toDispose = new Set()), (this._isDisposed = !1));
  }
  dispose() {
    this._isDisposed || ((this._isDisposed = !0), this.clear());
  }
  get isDisposed() {
    return this._isDisposed;
  }
  clear() {
    if (this._toDispose.size !== 0)
      try {
        ke(this._toDispose);
      } finally {
        this._toDispose.clear();
      }
  }
  add(t) {
    if (!t) return t;
    if (t === this) throw new Error("Cannot register a disposable on itself!");
    return (
      this._isDisposed
        ? De.DISABLE_DISPOSED_WARNING ||
          console.warn(
            new Error(
              "Trying to add a disposable to a DisposableStore that has already been disposed of. The added object will be leaked!",
            ).stack,
          )
        : this._toDispose.add(t),
      t
    );
  }
  delete(t) {
    if (t) {
      if (t === this) throw new Error("Cannot dispose a disposable on itself!");
      (this._toDispose.delete(t), t.dispose());
    }
  }
  deleteAndLeak(t) {
    t && this._toDispose.has(t) && (this._toDispose.delete(t), void 0);
  }
};
Ue.DISABLE_DISPOSED_WARNING = !1;
var ce = Ue,
  R = class {
    constructor() {
      ((this._store = new ce()), this._store);
    }
    dispose() {
      this._store.dispose();
    }
    _register(e) {
      if (e === this)
        throw new Error("Cannot register a disposable on itself!");
      return this._store.add(e);
    }
  };
R.None = Object.freeze({ dispose() {} });
var lt =
    globalThis.performance && typeof globalThis.performance.now == "function",
  ht = class Ce {
    static create(t) {
      return new Ce(t);
    }
    constructor(t) {
      ((this._now =
        lt && t === !1
          ? Date.now
          : globalThis.performance.now.bind(globalThis.performance)),
        (this._startTime = this._now()),
        (this._stopTime = -1));
    }
    stop() {
      this._stopTime = this._now();
    }
    reset() {
      ((this._startTime = this._now()), (this._stopTime = -1));
    }
    elapsed() {
      return this._stopTime !== -1
        ? this._stopTime - this._startTime
        : this._now() - this._startTime;
    }
  },
  ut;
((e) => {
  e.None = () => R.None;
  function t(h, n) {
    return z(h, () => {}, 0, void 0, !0, void 0, n);
  }
  e.defer = t;
  function r(h) {
    return (n, l = null, o) => {
      let u = !1,
        c;
      return (
        (c = h(
          (p) => {
            if (!u) return (c ? c.dispose() : (u = !0), n.call(l, p));
          },
          null,
          o,
        )),
        u && c.dispose(),
        c
      );
    };
  }
  e.once = r;
  function i(h, n, l) {
    return L((o, u = null, c) => h((p) => o.call(u, n(p)), null, c), l);
  }
  e.map = i;
  function s(h, n, l) {
    return L(
      (o, u = null, c) =>
        h(
          (p) => {
            (n(p), o.call(u, p));
          },
          null,
          c,
        ),
      l,
    );
  }
  e.forEach = s;
  function a(h, n, l) {
    return L((o, u = null, c) => h((p) => n(p) && o.call(u, p), null, c), l);
  }
  e.filter = a;
  function f(h) {
    return h;
  }
  e.signal = f;
  function _(...h) {
    return (n, l = null, o) => {
      let u = at(...h.map((c) => c((p) => n.call(l, p))));
      return k(u, o);
    };
  }
  e.any = _;
  function m(h, n, l, o) {
    let u = l;
    return i(h, (c) => ((u = n(u, c)), u), o);
  }
  e.reduce = m;
  function L(h, n) {
    let l,
      o = {
        onWillAddFirstListener() {
          l = h(u.fire, u);
        },
        onDidRemoveLastListener() {
          l?.dispose();
        },
      },
      u = new P(o);
    return (n?.add(u), u.event);
  }
  function k(h, n) {
    return (n instanceof Array ? n.push(h) : n && n.add(h), h);
  }
  function z(h, n, l = 100, o = !1, u = !1, c, p) {
    let b,
      x,
      C,
      T = 0,
      M,
      Be = {
        leakWarningThreshold: c,
        onWillAddFirstListener() {
          b = h((Ye) => {
            (T++,
              (x = n(x, Ye)),
              o && !C && (V.fire(x), (x = void 0)),
              (M = () => {
                let Je = x;
                ((x = void 0),
                  (C = void 0),
                  (!o || T > 1) && V.fire(Je),
                  (T = 0));
              }),
              typeof l == "number"
                ? (clearTimeout(C), (C = setTimeout(M, l)))
                : C === void 0 && ((C = 0), queueMicrotask(M)));
          });
        },
        onWillRemoveListener() {
          u && T > 0 && M?.();
        },
        onDidRemoveLastListener() {
          ((M = void 0), b.dispose());
        },
      },
      V = new P(Be);
    return (p?.add(V), V.event);
  }
  e.debounce = z;
  function I(h, n = 0, l) {
    return e.debounce(
      h,
      (o, u) => (o ? (o.push(u), o) : [u]),
      n,
      void 0,
      !0,
      void 0,
      l,
    );
  }
  e.accumulate = I;
  function F(h, n = (o, u) => o === u, l) {
    let o = !0,
      u;
    return a(
      h,
      (c) => {
        let p = o || !n(c, u);
        return ((o = !1), (u = c), p);
      },
      l,
    );
  }
  e.latch = F;
  function G(h, n, l) {
    return [e.filter(h, n, l), e.filter(h, (o) => !n(o), l)];
  }
  e.split = G;
  function Q(h, n = !1, l = [], o) {
    let u = l.slice(),
      c = h((x) => {
        u ? u.push(x) : b.fire(x);
      });
    o && o.add(c);
    let p = () => {
        (u?.forEach((x) => b.fire(x)), (u = null));
      },
      b = new P({
        onWillAddFirstListener() {
          c || ((c = h((x) => b.fire(x))), o && o.add(c));
        },
        onDidAddFirstListener() {
          u && (n ? setTimeout(p) : p());
        },
        onDidRemoveLastListener() {
          (c && c.dispose(), (c = null));
        },
      });
    return (o && o.add(b), b.event);
  }
  e.buffer = Q;
  function Z(h, n) {
    return (l, o, u) => {
      let c = n(new j());
      return h(
        function (p) {
          let b = c.evaluate(p);
          b !== U && l.call(o, b);
        },
        void 0,
        u,
      );
    };
  }
  e.chain = Z;
  let U = Symbol("HaltChainable");
  class j {
    constructor() {
      this.steps = [];
    }
    map(n) {
      return (this.steps.push(n), this);
    }
    forEach(n) {
      return (this.steps.push((l) => (n(l), l)), this);
    }
    filter(n) {
      return (this.steps.push((l) => (n(l) ? l : U)), this);
    }
    reduce(n, l) {
      let o = l;
      return (this.steps.push((u) => ((o = n(o, u)), o)), this);
    }
    latch(n = (l, o) => l === o) {
      let l = !0,
        o;
      return (
        this.steps.push((u) => {
          let c = l || !n(u, o);
          return ((l = !1), (o = u), c ? u : U);
        }),
        this
      );
    }
    evaluate(n) {
      for (let l of this.steps) if (((n = l(n)), n === U)) break;
      return n;
    }
  }
  function K(h, n, l = (o) => o) {
    let o = (...b) => p.fire(l(...b)),
      u = () => h.on(n, o),
      c = () => h.removeListener(n, o),
      p = new P({ onWillAddFirstListener: u, onDidRemoveLastListener: c });
    return p.event;
  }
  e.fromNodeEventEmitter = K;
  function d(h, n, l = (o) => o) {
    let o = (...b) => p.fire(l(...b)),
      u = () => h.addEventListener(n, o),
      c = () => h.removeEventListener(n, o),
      p = new P({ onWillAddFirstListener: u, onDidRemoveLastListener: c });
    return p.event;
  }
  e.fromDOMEventEmitter = d;
  function v(h) {
    return new Promise((n) => r(h)(n));
  }
  e.toPromise = v;
  function w(h) {
    let n = new P();
    return (
      h
        .then(
          (l) => {
            n.fire(l);
          },
          () => {
            n.fire(void 0);
          },
        )
        .finally(() => {
          n.dispose();
        }),
      n.event
    );
  }
  e.fromPromise = w;
  function y(h, n) {
    return h((l) => n.fire(l));
  }
  e.forward = y;
  function D(h, n, l) {
    return (n(l), h((o) => n(o)));
  }
  e.runAndSubscribe = D;
  class S {
    constructor(n, l) {
      ((this._observable = n), (this._counter = 0), (this._hasChanged = !1));
      let o = {
        onWillAddFirstListener: () => {
          n.addObserver(this);
        },
        onDidRemoveLastListener: () => {
          n.removeObserver(this);
        },
      };
      ((this.emitter = new P(o)), l && l.add(this.emitter));
    }
    beginUpdate(n) {
      this._counter++;
    }
    handlePossibleChange(n) {}
    handleChange(n, l) {
      this._hasChanged = !0;
    }
    endUpdate(n) {
      (this._counter--,
        this._counter === 0 &&
          (this._observable.reportChanges(),
          this._hasChanged &&
            ((this._hasChanged = !1),
            this.emitter.fire(this._observable.get()))));
    }
  }
  function je(h, n) {
    return new S(h, n).emitter.event;
  }
  e.fromObservable = je;
  function Ke(h) {
    return (n, l, o) => {
      let u = 0,
        c = !1,
        p = {
          beginUpdate() {
            u++;
          },
          endUpdate() {
            (u--, u === 0 && (h.reportChanges(), c && ((c = !1), n.call(l))));
          },
          handlePossibleChange() {},
          handleChange() {
            c = !0;
          },
        };
      (h.addObserver(p), h.reportChanges());
      let b = {
        dispose() {
          h.removeObserver(p);
        },
      };
      return (o instanceof ce ? o.add(b) : Array.isArray(o) && o.push(b), b);
    };
  }
  e.fromObservableLight = Ke;
})((ut ||= {}));
var oe = class ae {
  constructor(t) {
    ((this.listenerCount = 0),
      (this.invocationCount = 0),
      (this.elapsedOverall = 0),
      (this.durations = []),
      (this.name = `${t}_${ae._idPool++}`),
      ae.all.add(this));
  }
  start(t) {
    ((this._stopWatch = new ht()), (this.listenerCount = t));
  }
  stop() {
    if (this._stopWatch) {
      let t = this._stopWatch.elapsed();
      (this.durations.push(t),
        (this.elapsedOverall += t),
        (this.invocationCount += 1),
        (this._stopWatch = void 0));
    }
  }
};
((oe.all = new Set()), (oe._idPool = 0));
var dt = oe,
  ct = -1,
  Ne = class Me {
    constructor(t, r, i = (Me._idPool++).toString(16).padStart(3, "0")) {
      ((this._errorHandler = t),
        (this.threshold = r),
        (this.name = i),
        (this._warnCountdown = 0));
    }
    dispose() {
      this._stacks?.clear();
    }
    check(t, r) {
      let i = this.threshold;
      if (i <= 0 || r < i) return;
      this._stacks || (this._stacks = new Map());
      let s = this._stacks.get(t.value) || 0;
      if (
        (this._stacks.set(t.value, s + 1),
        (this._warnCountdown -= 1),
        this._warnCountdown <= 0)
      ) {
        this._warnCountdown = i * 0.5;
        let [a, f] = this.getMostFrequentStack(),
          _ = `[${this.name}] potential listener LEAK detected, having ${r} listeners already. MOST frequent listener (${f}):`;
        (console.warn(_), console.warn(a));
        let m = new pt(_, a);
        this._errorHandler(m);
      }
      return () => {
        let a = this._stacks.get(t.value) || 0;
        this._stacks.set(t.value, a - 1);
      };
    }
    getMostFrequentStack() {
      if (!this._stacks) return;
      let t,
        r = 0;
      for (let [i, s] of this._stacks) (!t || r < s) && ((t = [i, s]), (r = s));
      return t;
    }
  };
Ne._idPool = 1;
var ft = Ne,
  vt = class Oe {
    constructor(t) {
      this.value = t;
    }
    static create() {
      let t = new Error();
      return new Oe(t.stack ?? "");
    }
    print() {
      console.warn(
        this.value
          .split(
            `
`,
          )
          .slice(2).join(`
`),
      );
    }
  },
  pt = class extends Error {
    constructor(e, t) {
      (super(e), (this.name = "ListenerLeakError"), (this.stack = t));
    }
  },
  wt = class extends Error {
    constructor(e, t) {
      (super(e), (this.name = "ListenerRefusalError"), (this.stack = t));
    }
  },
  _t = 0,
  J = class {
    constructor(e) {
      ((this.value = e), (this.id = _t++));
    }
  },
  mt = 2,
  bt,
  P = class {
    constructor(e) {
      ((this._size = 0),
        (this._options = e),
        (this._leakageMon = this._options?.leakWarningThreshold
          ? new ft(
              e?.onListenerError ?? Y,
              this._options?.leakWarningThreshold ?? ct,
            )
          : void 0),
        (this._perfMon = this._options?._profName
          ? new dt(this._options._profName)
          : void 0),
        (this._deliveryQueue = this._options?.deliveryQueue));
    }
    dispose() {
      this._disposed ||
        ((this._disposed = !0),
        this._deliveryQueue?.current === this && this._deliveryQueue.reset(),
        this._listeners && ((this._listeners = void 0), (this._size = 0)),
        this._options?.onDidRemoveLastListener?.(),
        this._leakageMon?.dispose());
    }
    get event() {
      return (
        (this._event ??= (e, t, r) => {
          if (
            this._leakageMon &&
            this._size > this._leakageMon.threshold ** 2
          ) {
            let f = `[${this._leakageMon.name}] REFUSES to accept new listeners because it exceeded its threshold by far (${this._size} vs ${this._leakageMon.threshold})`;
            console.warn(f);
            let _ = this._leakageMon.getMostFrequentStack() ?? [
                "UNKNOWN stack",
                -1,
              ],
              m = new wt(
                `${f}. HINT: Stack shows most frequent listener (${_[1]}-times)`,
                _[0],
              );
            return ((this._options?.onListenerError || Y)(m), R.None);
          }
          if (this._disposed) return R.None;
          t && (e = e.bind(t));
          let i = new J(e),
            s;
          (this._leakageMon &&
            this._size >= Math.ceil(this._leakageMon.threshold * 0.2) &&
            ((i.stack = vt.create()),
            (s = this._leakageMon.check(i.stack, this._size + 1))),
            this._listeners
              ? this._listeners instanceof J
                ? ((this._deliveryQueue ??= new gt()),
                  (this._listeners = [this._listeners, i]))
                : this._listeners.push(i)
              : (this._options?.onWillAddFirstListener?.(this),
                (this._listeners = i),
                this._options?.onDidAddFirstListener?.(this)),
            this._size++);
          let a = ze(() => {
            (s?.(), this._removeListener(i));
          });
          return (
            r instanceof ce ? r.add(a) : Array.isArray(r) && r.push(a),
            a
          );
        }),
        this._event
      );
    }
    _removeListener(e) {
      if ((this._options?.onWillRemoveListener?.(this), !this._listeners))
        return;
      if (this._size === 1) {
        ((this._listeners = void 0),
          this._options?.onDidRemoveLastListener?.(this),
          (this._size = 0));
        return;
      }
      let t = this._listeners,
        r = t.indexOf(e);
      if (r === -1)
        throw (
          console.log("disposed?", this._disposed),
          console.log("size?", this._size),
          console.log("arr?", JSON.stringify(this._listeners)),
          new Error("Attempted to dispose unknown listener")
        );
      (this._size--, (t[r] = void 0));
      let i = this._deliveryQueue.current === this;
      if (this._size * mt <= t.length) {
        let s = 0;
        for (let a = 0; a < t.length; a++)
          t[a]
            ? (t[s++] = t[a])
            : i &&
              (this._deliveryQueue.end--,
              s < this._deliveryQueue.i && this._deliveryQueue.i--);
        t.length = s;
      }
    }
    _deliver(e, t) {
      if (!e) return;
      let r = this._options?.onListenerError || Y;
      if (!r) {
        e.value(t);
        return;
      }
      try {
        e.value(t);
      } catch (i) {
        r(i);
      }
    }
    _deliverQueue(e) {
      let t = e.current._listeners;
      for (; e.i < e.end;) this._deliver(t[e.i++], e.value);
      e.reset();
    }
    fire(e) {
      if (
        (this._deliveryQueue?.current &&
          (this._deliverQueue(this._deliveryQueue), this._perfMon?.stop()),
        this._perfMon?.start(this._size),
        this._listeners)
      )
        if (this._listeners instanceof J) this._deliver(this._listeners, e);
        else {
          let t = this._deliveryQueue;
          (t.enqueue(this, e, this._listeners.length), this._deliverQueue(t));
        }
      this._perfMon?.stop();
    }
    hasListeners() {
      return this._size > 0;
    }
  },
  gt = class {
    constructor() {
      ((this.i = -1), (this.end = 0));
    }
    enqueue(e, t, r) {
      ((this.i = 0), (this.end = r), (this.current = e), (this.value = t));
    }
    reset() {
      ((this.i = this.end), (this.current = void 0), (this.value = void 0));
    }
  },
  N = class H {
    constructor() {
      ((this._providers = Object.create(null)),
        (this._active = ""),
        (this._onChange = new P()),
        (this.onChange = this._onChange.event));
      let t = new et();
      (this.register(t),
        (this._active = t.version),
        (this._activeProvider = t));
    }
    static extractShouldJoin(t) {
      return (t & 1) !== 0;
    }
    static extractWidth(t) {
      return (t >> 1) & 3;
    }
    static extractCharKind(t) {
      return t >> 3;
    }
    static createPropertyValue(t, r, i = !1) {
      return ((t & 16777215) << 3) | ((r & 3) << 1) | (i ? 1 : 0);
    }
    dispose() {
      this._onChange.dispose();
    }
    get versions() {
      return Object.keys(this._providers);
    }
    get activeVersion() {
      return this._active;
    }
    set activeVersion(t) {
      if (!this._providers[t])
        throw new Error(`unknown Unicode version "${t}"`);
      ((this._active = t),
        (this._activeProvider = this._providers[t]),
        this._onChange.fire(t));
    }
    register(t) {
      this._providers[t.version] = t;
    }
    wcwidth(t) {
      return this._activeProvider.wcwidth(t);
    }
    getStringCellWidth(t) {
      let r = 0,
        i = 0,
        s = t.length;
      for (let a = 0; a < s; ++a) {
        let f = t.charCodeAt(a);
        if (55296 <= f && f <= 56319) {
          if (++a >= s) return r + this.wcwidth(f);
          let L = t.charCodeAt(a);
          56320 <= L && L <= 57343
            ? (f = (f - 55296) * 1024 + L - 56320 + 65536)
            : (r += this.wcwidth(L));
        }
        let _ = this.charProperties(f, i),
          m = H.extractWidth(_);
        (H.extractShouldJoin(_) && (m -= H.extractWidth(i)), (r += m), (i = _));
      }
      return r;
    }
    charProperties(t, r) {
      return this._activeProvider.charProperties(t, r);
    }
  },
  fe = 0,
  Se = -3,
  O = class {
    constructor() {
      ((this.table = new Uint16Array(16)), (this.trans = new Uint16Array(288)));
    }
  },
  yt = class {
    constructor(e, t) {
      ((this.tag = 0),
        (this.bitcount = 0),
        (this.destLen = 0),
        (this.sourceIndex = 0),
        (this.source = e),
        (this.dest = t),
        (this.ltree = new O()),
        (this.dtree = new O()));
    }
  },
  Te = new O(),
  Ve = new O(),
  ve = new Uint8Array(30),
  pe = new Uint16Array(30),
  He = new Uint8Array(30),
  Xe = new Uint16Array(30),
  xt = new Uint8Array([
    16, 17, 18, 0, 8, 7, 9, 6, 10, 5, 11, 4, 12, 3, 13, 2, 14, 1, 15,
  ]),
  me = new O(),
  E = new Uint8Array(320);
function Re(e, t, r, i) {
  var s, a;
  for (s = 0; s < r; ++s) e[s] = 0;
  for (s = 0; s < 30 - r; ++s) e[s + r] = (s / r) | 0;
  for (a = i, s = 0; s < 30; ++s) ((t[s] = a), (a += 1 << e[s]));
}
function Lt(e, t) {
  var r;
  for (r = 0; r < 7; ++r) e.table[r] = 0;
  for (e.table[7] = 24, e.table[8] = 152, e.table[9] = 112, r = 0; r < 24; ++r)
    e.trans[r] = 256 + r;
  for (r = 0; r < 144; ++r) e.trans[24 + r] = r;
  for (r = 0; r < 8; ++r) e.trans[168 + r] = 280 + r;
  for (r = 0; r < 112; ++r) e.trans[176 + r] = 144 + r;
  for (r = 0; r < 5; ++r) t.table[r] = 0;
  for (t.table[5] = 32, r = 0; r < 32; ++r) t.trans[r] = r;
}
var be = new Uint16Array(16);
function q(e, t, r, i) {
  var s, a;
  for (s = 0; s < 16; ++s) e.table[s] = 0;
  for (s = 0; s < i; ++s) e.table[t[r + s]]++;
  for (e.table[0] = 0, a = 0, s = 0; s < 16; ++s)
    ((be[s] = a), (a += e.table[s]));
  for (s = 0; s < i; ++s) t[r + s] && (e.trans[be[t[r + s]]++] = s);
}
function Et(e) {
  e.bitcount-- || ((e.tag = e.source[e.sourceIndex++]), (e.bitcount = 7));
  var t = e.tag & 1;
  return ((e.tag >>>= 1), t);
}
function A(e, t, r) {
  if (!t) return r;
  for (; e.bitcount < 24;)
    ((e.tag |= e.source[e.sourceIndex++] << e.bitcount), (e.bitcount += 8));
  var i = e.tag & (65535 >>> (16 - t));
  return ((e.tag >>>= t), (e.bitcount -= t), i + r);
}
function le(e, t) {
  for (; e.bitcount < 24;)
    ((e.tag |= e.source[e.sourceIndex++] << e.bitcount), (e.bitcount += 8));
  var r = 0,
    i = 0,
    s = 0,
    a = e.tag;
  do
    ((i = 2 * i + (a & 1)),
      (a >>>= 1),
      ++s,
      (r += t.table[s]),
      (i -= t.table[s]));
  while (i >= 0);
  return ((e.tag = a), (e.bitcount -= s), t.trans[r + i]);
}
function At(e, t, r) {
  var i, s, a, f, _, m;
  for (i = A(e, 5, 257), s = A(e, 5, 1), a = A(e, 4, 4), f = 0; f < 19; ++f)
    E[f] = 0;
  for (f = 0; f < a; ++f) {
    var L = A(e, 3, 0);
    E[xt[f]] = L;
  }
  for (q(me, E, 0, 19), _ = 0; _ < i + s;) {
    var k = le(e, me);
    switch (k) {
      case 16:
        var z = E[_ - 1];
        for (m = A(e, 2, 3); m; --m) E[_++] = z;
        break;
      case 17:
        for (m = A(e, 3, 3); m; --m) E[_++] = 0;
        break;
      case 18:
        for (m = A(e, 7, 11); m; --m) E[_++] = 0;
        break;
      default:
        E[_++] = k;
        break;
    }
  }
  (q(t, E, 0, i), q(r, E, i, s));
}
function ge(e, t, r) {
  for (;;) {
    var i = le(e, t);
    if (i === 256) return fe;
    if (i < 256) e.dest[e.destLen++] = i;
    else {
      var s, a, f, _;
      for (
        i -= 257,
          s = A(e, ve[i], pe[i]),
          a = le(e, r),
          f = e.destLen - A(e, He[a], Xe[a]),
          _ = f;
        _ < f + s;
        ++_
      )
        e.dest[e.destLen++] = e.dest[_];
    }
  }
}
function Pt(e) {
  for (var t, r, i; e.bitcount > 8;) (e.sourceIndex--, (e.bitcount -= 8));
  if (
    ((t = e.source[e.sourceIndex + 1]),
    (t = 256 * t + e.source[e.sourceIndex]),
    (r = e.source[e.sourceIndex + 3]),
    (r = 256 * r + e.source[e.sourceIndex + 2]),
    t !== (~r & 65535))
  )
    return Se;
  for (e.sourceIndex += 4, i = t; i; --i)
    e.dest[e.destLen++] = e.source[e.sourceIndex++];
  return ((e.bitcount = 0), fe);
}
function Wt(e, t) {
  var r = new yt(e, t),
    i,
    s,
    a;
  do {
    switch (((i = Et(r)), (s = A(r, 2, 0)), s)) {
      case 0:
        a = Pt(r);
        break;
      case 1:
        a = ge(r, Te, Ve);
        break;
      case 2:
        (At(r, r.ltree, r.dtree), (a = ge(r, r.ltree, r.dtree)));
        break;
      default:
        a = Se;
    }
    if (a !== fe) throw new Error("Data error");
  } while (!i);
  return r.destLen < r.dest.length
    ? typeof r.dest.slice == "function"
      ? r.dest.slice(0, r.destLen)
      : r.dest.subarray(0, r.destLen)
    : r.dest;
}
Lt(Te, Ve);
Re(ve, pe, 4, 3);
Re(He, Xe, 2, 1);
ve[28] = 0;
pe[28] = 258;
var ye = Wt,
  we = 11,
  W = 5,
  kt = we - W,
  zt = 65536 >> we,
  Ut = 1 << kt,
  Dt = Ut - 1,
  X = 2,
  Ct = 1 << W,
  $ = Ct - 1,
  Ie = 65536 >> W,
  Nt = 1024 >> W,
  Mt = Ie + Nt,
  Ot = Mt,
  St = 32,
  Tt = Ot + St,
  Vt = 1 << X,
  Ht = new Uint8Array(new Uint32Array([305419896]).buffer)[0] === 18,
  Xt = class {
    constructor(e) {
      let t = new DataView(e.buffer);
      ((this.highStart = t.getUint32(0, !0)),
        (this.errorValue = t.getUint32(4, !0)));
      let r = t.getUint32(8, !0);
      if (
        ((e = e.subarray(12)),
        (e = ye(e, new Uint8Array(r))),
        (e = ye(e, new Uint8Array(r))),
        Ht)
      ) {
        let i = e.length;
        for (let s = 0; s < i; s += 4) {
          let a = e[s];
          ((e[s] = e[s + 3]), (e[s + 3] = a));
          let f = e[s + 1];
          ((e[s + 1] = e[s + 2]), (e[s + 2] = f));
        }
      }
      this.data = new Uint32Array(e.buffer);
    }
    get(e) {
      let t;
      return e < 0 || e > 1114111
        ? this.errorValue
        : e < 55296 || (e > 56319 && e <= 65535)
          ? ((t = (this.data[e >> W] << X) + (e & $)), this.data[t])
          : e <= 65535
            ? ((t = (this.data[Ie + ((e - 55296) >> W)] << X) + (e & $)),
              this.data[t])
            : e < this.highStart
              ? ((t = this.data[Tt - zt + (e >> we)]),
                (t = this.data[t + ((e >> W) & Dt)]),
                (t = (t << X) + (e & $)),
                this.data[t])
              : this.data[this.data.length - Vt];
    }
  },
  Rt = Xt,
  It =
    "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";
function Ft(e) {
  if (typeof Buffer < "u") return Buffer.from(e, "base64");
  let t = atob(e),
    r = new Uint8Array(t.length);
  for (let i = 0; i < r.length; ++i) r[i] = t.charCodeAt(i);
  return r;
}
var Gt = new Rt(Ft(It)),
  he = 15,
  ue = 0,
  Qt = 48,
  Zt = 4,
  jt = 0,
  Fe = 1,
  Ge = 2,
  de = 3,
  Kt = 4,
  ee = 5,
  te = 6,
  re = 7,
  xe = 8,
  se = 9,
  Le = 10,
  Bt = 11,
  Yt = 32;
function Ee(e) {
  return (e & Qt) >> Zt;
}
function Jt(e, t) {
  let r = (e & he) >> ue,
    i = (t & he) >> ue;
  return qt(r, i) ? (i === de ? Yt : i + 16) : i - 16;
}
function qt(e, t) {
  return (
    (e >= ee &&
      e <= se &&
      ((e == ee && (t == ee || t == te || t == xe || t == se)) ||
        ((e == xe || e == te) && (t == te || t == re)) ||
        ((e == se || e == re) && t == re))) ||
    t == Ge ||
    t == Le ||
    e == Fe ||
    t == Kt ||
    (e == Le && t == Bt) ||
    (t == de && e == de)
  );
}
function Ae(e) {
  return Gt.get(e);
}
var Qe = class Ze {
  constructor(t = !0) {
    ((this.ambiguousCharsAreWide = !1),
      (this.version = t ? "15-graphemes" : "15"),
      (this.handleGraphemes = t));
  }
  charProperties(t, r) {
    if (t >= 32 && t < 127 && r >> 3 === 0) return Ze._plainNarrowProperties;
    let i = Ae(t),
      s = Ee(i),
      a = !1;
    if (
      (s >= 2
        ? (s = s === 3 || this.ambiguousCharsAreWide || t === 65039 ? 2 : 1)
        : (s = 1),
      r !== 0)
    ) {
      let f = N.extractWidth(r);
      (this.handleGraphemes
        ? (i = Jt(N.extractCharKind(r), i))
        : (i = s === 0 ? 1 : 0),
        (a = i > 0),
        a && (f > s ? (s = f) : i === 32 && (s = 2)));
    }
    return N.createPropertyValue(i, s, a);
  }
  wcwidth(t) {
    let r = Ae(t),
      i = Ee(r),
      s = (r & he) >> ue;
    return s === Ge || s === Fe
      ? 0
      : i >= 2 && (i === 3 || this.ambiguousCharsAreWide)
        ? 2
        : 1;
  }
};
Qe._plainNarrowProperties = N.createPropertyValue(jt, 1, !1);
var Pe = Qe,
  $t = class {
    constructor() {
      this._oldVersion = "";
    }
    activate(e) {
      (this._provider15 || (this._provider15 = new Pe(!1)),
        this._provider15Graphemes || (this._provider15Graphemes = new Pe(!0)));
      let t = e.unicode;
      ((this._unicode = t),
        t.register(this._provider15),
        t.register(this._provider15Graphemes),
        (this._oldVersion = t.activeVersion),
        (t.activeVersion = "15-graphemes"));
    }
    dispose() {
      this._unicode && (this._unicode.activeVersion = this._oldVersion);
    }
  };
export { $t as UnicodeGraphemesAddon };
