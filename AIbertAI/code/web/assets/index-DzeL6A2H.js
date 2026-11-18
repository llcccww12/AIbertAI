import {
  D as U,
  t as J,
  a as Sn,
  C as vn,
  H as Cn,
  b as bn,
  c as yn,
  g as ht,
  d as j,
  e as ce,
  f as Ge,
  h as We,
  i as Ft,
  j as ct,
  U as Oi,
  k as Br,
  l as Ni,
  T as zi,
  m as Ui,
  E as _i,
  n as Bi,
  S as Vr,
  o as xe,
  p as ee,
  q as $r,
  r as Vi,
  s as Pe,
  u as Oe,
  v as Gr,
  w as Wr,
  x as qr,
  y as jr,
  z as Ye,
  A as Kr,
  B as Xr,
  F as Yr,
  G as Qr,
  I as Zr,
  J as Jr,
  K as eo,
  L as to,
  R as $i,
  M as no,
  N as io,
  O as ro,
  P as oo,
  Q as so,
  V as ao,
  W as lo,
  X as ho,
  Y as co,
} from "./pierre-ZKp3_yQg.js";
import {
  Z as Cd,
  _ as bd,
  $ as yd,
  a0 as Ld,
  a1 as xd,
  a2 as kd,
  a3 as wd,
  a4 as Ed,
  a5 as Rd,
  a6 as Td,
  a7 as Ad,
  a8 as Id,
  a9 as Hd,
  aa as Md,
  ab as Dd,
  ac as Fd,
  ad as Pd,
  ae as Od,
  af as Nd,
  ag as zd,
  ah as Ud,
  ai as _d,
  aj as Bd,
  ak as Vd,
  al as $d,
  am as Gd,
  an as Wd,
  ao as qd,
  ap as jd,
  aq as Kd,
  ar as Xd,
  as as Yd,
  at as Qd,
  au as Zd,
  av as Jd,
  aw as eh,
  ax as th,
  ay as nh,
  az as ih,
  aA as rh,
  aB as oh,
} from "./pierre-ZKp3_yQg.js";
import "./index-zDHqw1Zn.js";
const De = new Set();
let Le = null;
function W(e) {
  (De.add(e), (Le ??= requestAnimationFrame(Gi)));
}
function Ne(e) {
  De.delete(e) &&
    De.size === 0 &&
    Le != null &&
    (cancelAnimationFrame(Le), (Le = null));
}
function Xl() {
  (De.clear(),
    Le != null &&
      typeof cancelAnimationFrame == "function" &&
      cancelAnimationFrame(Le),
    (Le = null));
}
function Gi(e) {
  const t = new Set(De);
  De.clear();
  for (const n of t)
    try {
      n(e);
    } catch (i) {
      console.error(i);
    }
  De.size > 0 ? (Le = requestAnimationFrame(Gi)) : (Le = null);
}
function qe(e, t, n) {
  if (e === t || e == null || t == null) return e === t;
  const i = new Set(n),
    r = Object.keys(e),
    o = new Set(Object.keys(t));
  for (const s of r)
    if ((o.delete(s), !i.has(s) && (!(s in t) || e[s] !== t[s]))) return !1;
  for (const s of Array.from(o)) if (!i.has(s)) return !1;
  return !0;
}
function Ue(e, t) {
  return e == null || t == null || typeof e == "string" || typeof t == "string"
    ? e === t
    : e.dark === t.dark && e.light === t.light;
}
function Ln(e, t) {
  const n = e?.theme ?? U,
    i = t?.theme ?? U,
    r = On(e),
    o = On(t);
  return Ue(n, i) && qe(e, t, ["theme", "parseDiffOptions"]) && qe(r, o);
}
function On(e) {
  if (e != null && "parseDiffOptions" in e) return e.parseDiffOptions;
}
function Pt(e, t) {
  return (
    e?.start === t?.start &&
    e?.end === t?.end &&
    e?.side === t?.side &&
    e?.endSide === t?.endSide
  );
}
function rn({
  scrollTop: e,
  scrollHeight: t,
  height: n,
  fitPerfectly: i = !1,
  fitPerfectlyOverscroll: r = 0,
  overscrollSize: o,
}) {
  const s = n + o * 2,
    a = i ? n + r * 2 : s;
  if (((t = Math.max(t, a)), s >= t || i)) {
    const h = Math.max(e - r, 0),
      c = Math.min(e + a, t);
    return { top: h, bottom: Math.max(c, h) };
  }
  let l = e + n / 2 - s / 2,
    d = l + s;
  return (
    l < 0 && (l = 0),
    d > t && (d = t),
    (l = Math.floor(Math.max(l, 0))),
    { top: l, bottom: Math.ceil(Math.max(Math.min(d, t), l)) }
  );
}
function uo() {
  return typeof window > "u" || typeof window.matchMedia != "function"
    ? !1
    : window.matchMedia("(prefers-reduced-motion: reduce)").matches;
}
function X(e) {
  return { type: "text", value: e };
}
function R({ tagName: e, children: t = [], properties: n = {} }) {
  return { type: "element", tagName: e, properties: n, children: t };
}
function Ot({ name: e, width: t = 16, height: n = 16, properties: i }) {
  return R({
    tagName: "svg",
    properties: { width: t, height: n, viewBox: "0 0 16 16", ...i },
    children: [
      R({ tagName: "use", properties: { href: `#${e.replace(/^#/, "")}` } }),
    ],
  });
}
function fo(e) {
  let t = e.children[0];
  for (; t != null;) {
    if (t.type === "element" && t.tagName === "code") return t;
    "children" in t ? (t = t.children[0]) : (t = null);
  }
}
function $e(e) {
  return R({ tagName: "div", properties: { "data-gutter": "" }, children: e });
}
function Wi(e, t, n, i = {}) {
  return R({
    tagName: "div",
    properties: {
      "data-line-type": e,
      "data-column-number": t,
      "data-line-index": n,
      ...i,
    },
    children:
      t != null
        ? [
            R({
              tagName: "span",
              properties: { "data-line-number-content": "" },
              children: [X(`${t}`)],
            }),
          ]
        : void 0,
  });
}
function Z(e, t, n) {
  return R({
    tagName: "div",
    properties: {
      "data-gutter-buffer": t,
      "data-buffer-size": n,
      "data-line-type": t === "annotation" ? void 0 : e,
      style:
        t === "annotation"
          ? `grid-row: span ${n};`
          : `grid-row: span ${n};min-height:calc(${n} * 1lh);`,
    },
  });
}
function po() {
  return R({
    tagName: "button",
    properties: { "data-utility-button": "", type: "button" },
    children: [
      Ot({ name: "diffs-icon-plus", properties: { "data-icon": "" } }),
    ],
  });
}
function go(e, t) {
  return e.lineNumber === t.lineNumber && e.side === t.side;
}
function ft(e) {
  for (const t of e)
    if (
      t instanceof HTMLElement &&
      (t.hasAttribute("data-utility-button") ||
        t.hasAttribute("data-gutter-utility-slot") ||
        t.getAttribute("slot") === "gutter-utility-slot" ||
        t.getAttribute("name") === "gutter-utility-slot")
    )
      return !0;
  return !1;
}
var qi = class {
  mode;
  options;
  hoveredLine;
  hoveredToken;
  pre;
  gutterUtilityLine;
  gutterUtilityContainer;
  gutterUtilityButton;
  gutterUtilitySlot;
  interactiveLinesAttr = !1;
  interactiveLineNumbersAttr = !1;
  hasPointerListeners = !1;
  hasDocumentPointerListeners = !1;
  selectedRange = null;
  selectedRangeHighlightSide;
  selectedRangeLineNumberOnly = !1;
  editorActiveLine = null;
  editorActiveLineSide;
  editorLineNumberOnly = !1;
  proposedSelectedRange;
  renderedSelectedLinesState;
  renderedEditorActiveLineState;
  selectionAnchor;
  pointerSession = { mode: "idle" };
  constructor(e, t) {
    ((this.mode = e), (this.options = t));
  }
  setOptions(e) {
    this.options = e;
  }
  cleanUp() {
    (this.pre?.removeEventListener("click", this.handlePointerClick),
      this.pre?.removeEventListener("pointerdown", this.handlePointerDown),
      this.pre?.removeEventListener("pointermove", this.handlePointerMove),
      this.pre?.removeEventListener("pointerleave", this.handlePointerLeave),
      this.pre?.removeAttribute("data-interactive-lines"),
      this.pre?.removeAttribute("data-interactive-line-numbers"),
      (this.pre = void 0),
      this.gutterUtilityContainer?.remove(),
      (this.gutterUtilityLine = void 0),
      (this.gutterUtilityContainer = void 0),
      (this.gutterUtilityButton = void 0),
      (this.gutterUtilitySlot = void 0),
      this.clearHoveredLine(),
      this.clearHoveredToken(),
      this.detachDocumentPointerListeners(),
      this.clearPointerSession(),
      Ne(this.renderSelection),
      (this.interactiveLinesAttr = !1),
      (this.interactiveLineNumbersAttr = !1),
      (this.hasPointerListeners = !1),
      this.setSelectionDirty());
  }
  setup(e) {
    this.setSelectionDirty();
    const { usesCustomGutterUtility: t = !1, enableGutterUtility: n = !1 } =
      this.options;
    (this.pre !== e && (this.cleanUp(), (this.pre = e)),
      n
        ? this.ensureGutterUtilityNode(t)
        : this.gutterUtilityContainer != null &&
          (this.gutterUtilityContainer.remove(),
          (this.gutterUtilityLine = void 0),
          (this.gutterUtilityContainer = void 0),
          (this.gutterUtilityButton = void 0),
          (this.gutterUtilitySlot = void 0),
          this.pointerSession.mode === "gutterSelecting" &&
            (this.clearPointerSession(),
            this.detachDocumentPointerListeners())),
      this.syncPointerListeners(e),
      this.updateInteractiveLineAttributes(),
      this.renderSelection(),
      this.placeUtility());
  }
  setSelectionDirty() {
    ((this.renderedSelectedLinesState = void 0),
      (this.renderedEditorActiveLineState = void 0));
  }
  isSelectionDirty() {
    return (
      this.renderedSelectedLinesState === void 0 ||
      this.renderedEditorActiveLineState === void 0
    );
  }
  setSelection(e, t) {
    const n = !(
        e === this.selectedRange ||
        Pt(e ?? void 0, this.selectedRange ?? void 0)
      ),
      i = t?.lineNumberOnly ?? !1,
      r =
        t?.activeLineSide !== this.selectedRangeHighlightSide ||
        i !== this.selectedRangeLineNumberOnly;
    (!this.isSelectionDirty() && !n && !r) ||
      ((this.proposedSelectedRange = void 0),
      (this.selectedRange = e),
      (this.selectedRangeHighlightSide = t?.activeLineSide),
      (this.selectedRangeLineNumberOnly = i),
      this.renderSelection(),
      this.placeUtility(),
      n && t?.notify !== !1 && this.notifySelectionCommitted(e));
  }
  setEditorActiveLine(e, { lineNumberOnly: t = !1, side: n } = {}) {
    const i = e !== this.editorActiveLine,
      r = n !== this.editorActiveLineSide || t !== this.editorLineNumberOnly;
    (!this.isSelectionDirty() && !i && !r) ||
      ((this.editorActiveLine = e),
      (this.editorActiveLineSide = n),
      (this.editorLineNumberOnly = t),
      this.renderSelection());
  }
  getSelection() {
    return this.selectedRange;
  }
  getHoveredLine = () => {
    const e = this.gutterUtilityLine ?? this.hoveredLine;
    if (e != null) {
      if (this.mode === "diff" && e.type === "diff-line")
        return { lineNumber: e.lineNumber, side: e.annotationSide };
      if (this.mode === "file" && e.type === "line")
        return { lineNumber: e.lineNumber };
    }
  };
  handlePointerClick = (e) => {
    const {
      onHunkExpand: t,
      onLineClick: n,
      onLineNumberClick: i,
      onTokenClick: r,
      onMergeConflictActionClick: o,
    } = this.options;
    (t == null && n == null && i == null && o == null && r == null) ||
      (this.options.onGutterUtilityClick != null && ft(e.composedPath())) ||
      (me(
        this.options.__debugPointerEvents,
        "click",
        "FileDiff.DEBUG.handlePointerClick:",
        e,
      ),
      this.handlePointerEvent({ eventType: "click", event: e }));
  };
  handlePointerMove = (e) => {
    if (e.pointerType !== "mouse") return;
    const {
      lineHoverHighlight: t = "disabled",
      onLineEnter: n,
      onLineLeave: i,
      onTokenEnter: r,
      onTokenLeave: o,
      enableGutterUtility: s = !1,
    } = this.options;
    (t === "disabled" &&
      !s &&
      n == null &&
      i == null &&
      r == null &&
      o == null) ||
      (me(
        this.options.__debugPointerEvents,
        "move",
        "FileDiff.DEBUG.handlePointerMove:",
        e,
      ),
      this.handlePointerEvent({ eventType: "move", event: e }));
  };
  handlePointerLeave = (e) => {
    const { __debugPointerEvents: t } = this.options;
    if (
      (me(t, "move", "FileDiff.DEBUG.handlePointerLeave: no event"),
      this.hoveredLine == null && this.hoveredToken == null)
    ) {
      me(
        t,
        "move",
        "FileDiff.DEBUG.handlePointerLeave: returned early, no hovered line or token",
      );
      return;
    }
    (this.hoveredToken != null &&
      (this.options.onTokenLeave?.(this.hoveredToken, e),
      this.clearHoveredToken()),
      this.hoveredLine != null &&
        (this.options.onLineLeave?.({ ...this.hoveredLine, event: e }),
        this.clearHoveredLine()),
      this.placeUtility());
  };
  handlePointerEvent({ eventType: e, event: t }) {
    const { __debugPointerEvents: n } = this.options,
      i = t.composedPath();
    me(n, e, "FileDiff.DEBUG.handlePointerEvent:", {
      eventType: e,
      composedPath: i,
    });
    const r = this.resolvePointerTarget(i);
    me(
      n,
      e,
      "FileDiff.DEBUG.handlePointerEvent: resolvePointerTarget result:",
      r,
    );
    const {
      onLineClick: o,
      onLineNumberClick: s,
      onLineEnter: a,
      onLineLeave: l,
      onTokenClick: d,
      onTokenEnter: h,
      onTokenLeave: c,
      onHunkExpand: u,
      onMergeConflictActionClick: f,
    } = this.options;
    switch (e) {
      case "move": {
        const p = Wt(r) && this.hoveredLine?.lineElement === r.lineElement;
        ((At(r) && this.hoveredToken?.tokenElement === r.tokenElement) ||
          (this.hoveredToken != null &&
            (c?.(this.hoveredToken, t), this.clearHoveredToken()),
          At(r) &&
            (this.setHoveredToken(this.toTokenEventBaseProps(r)),
            h?.(this.hoveredToken, t))),
          p ||
            (this.hoveredLine != null &&
              (l?.({ ...this.hoveredLine, event: t }), this.clearHoveredLine()),
            Wt(r)
              ? (this.setHoveredLine(this.toEventBaseProps(r)),
                this.placeUtility(),
                a?.({ ...this.hoveredLine, event: t }))
              : this.placeUtility()));
        break;
      }
      case "click": {
        if (r == null) break;
        if (vo(r) && f != null) {
          f(r);
          break;
        }
        if (So(r) && u != null) {
          u(
            r.hunkIndex,
            r.all || t.shiftKey ? "both" : r.direction,
            r.all || t.shiftKey ? Number.POSITIVE_INFINITY : void 0,
          );
          break;
        }
        if (!Wt(r)) break;
        At(r) && d != null && d(this.toTokenEventBaseProps(r), t);
        const p = this.toEventBaseProps(r);
        s != null && r.numberColumn
          ? s({ ...p, event: t })
          : o?.({ ...p, event: t });
        break;
      }
    }
  }
  syncPointerListeners(e) {
    const {
        __debugPointerEvents: t,
        lineHoverHighlight: n = "disabled",
        onLineClick: i,
        onLineNumberClick: r,
        onLineEnter: o,
        onLineLeave: s,
        onTokenClick: a,
        onTokenEnter: l,
        onTokenLeave: d,
        onHunkExpand: h,
        onMergeConflictActionClick: c,
        enableGutterUtility: u = !1,
        enableLineSelection: f = !1,
        onGutterUtilityClick: p,
      } = this.options,
      g = p != null,
      m =
        n !== "disabled" ||
        i != null ||
        r != null ||
        o != null ||
        s != null ||
        a != null ||
        l != null ||
        d != null ||
        h != null ||
        c != null ||
        u ||
        f ||
        g;
    m && !this.hasPointerListeners
      ? (e.addEventListener("click", this.handlePointerClick),
        e.addEventListener("pointerdown", this.handlePointerDown),
        e.addEventListener("pointermove", this.handlePointerMove),
        e.addEventListener("pointerleave", this.handlePointerLeave),
        (this.hasPointerListeners = !0),
        me(
          t,
          "click",
          "FileDiff.DEBUG.attachEventListeners: Attaching click events for:",
          (() => {
            const b = [];
            return (
              (t === "both" || t === "click") &&
                (i != null && b.push("onLineClick"),
                r != null && b.push("onLineNumberClick"),
                h != null && b.push("expandable hunk separators"),
                c != null && b.push("merge conflict actions")),
              b
            );
          })(),
        ),
        me(
          t,
          "move",
          "FileDiff.DEBUG.attachEventListeners: Attaching pointer move event",
        ),
        me(
          t,
          "move",
          "FileDiff.DEBUG.attachEventListeners: Attaching pointer leave event",
        ))
      : !m &&
        this.hasPointerListeners &&
        (e.removeEventListener("click", this.handlePointerClick),
        e.removeEventListener("pointerdown", this.handlePointerDown),
        e.removeEventListener("pointermove", this.handlePointerMove),
        e.removeEventListener("pointerleave", this.handlePointerLeave),
        (this.hasPointerListeners = !1));
    const y =
        this.pointerSession.mode === "selecting" ||
        this.pointerSession.mode === "pendingSingleLineUnselect",
      S = this.pointerSession.mode === "gutterSelecting";
    ((!f && y) || (!g && S)) &&
      (this.clearPointerSession(),
      this.detachDocumentPointerListeners(),
      (this.selectionAnchor = void 0),
      this.clearPendingSingleLineState());
  }
  updateInteractiveLineAttributes() {
    if (this.pre == null) return;
    const {
        onLineClick: e,
        onLineNumberClick: t,
        enableLineSelection: n = !1,
      } = this.options,
      i = e != null,
      r = t != null || n;
    (i && !this.interactiveLinesAttr
      ? (this.pre.setAttribute("data-interactive-lines", ""),
        (this.interactiveLinesAttr = !0))
      : !i &&
        this.interactiveLinesAttr &&
        (this.pre.removeAttribute("data-interactive-lines"),
        (this.interactiveLinesAttr = !1)),
      r && !this.interactiveLineNumbersAttr
        ? (this.pre.setAttribute("data-interactive-line-numbers", ""),
          (this.interactiveLineNumbersAttr = !0))
        : !r &&
          this.interactiveLineNumbersAttr &&
          (this.pre.removeAttribute("data-interactive-line-numbers"),
          (this.interactiveLineNumbersAttr = !1)));
  }
  handlePointerDown = (e) => {
    if (
      (e.pointerType === "mouse" && e.button !== 0) ||
      this.pre == null ||
      this.pointerSession.mode !== "idle"
    )
      return;
    const t = e.composedPath();
    ft(t) && this.options.onGutterUtilityClick != null
      ? this.startGutterSelectionFromPointerDown(e)
      : (e.pointerType !== "mouse" && this.revealUtilityFromGutterPath(t),
        this.startLineSelectionFromPointerDown(e));
  };
  startLineSelectionFromPointerDown(e) {
    const { enableLineSelection: t = !1 } = this.options;
    if (!t) return;
    const n = this.resolveSelectionInfo(e, {
      source: "event-path",
      requireNumberColumn: !0,
    });
    if (n == null) return;
    const { pre: i } = this;
    if (i == null) return;
    const { lineNumber: r, eventSide: o, lineIndex: s } = n;
    if (e.shiftKey && this.selectedRange != null) {
      const a = this.getIndexesFromSelection(
        this.selectedRange,
        i.getAttribute("data-diff-type") === "split",
      );
      if (a == null) return;
      const l = a.start <= a.end ? s >= a.start : s <= a.end;
      ((this.selectionAnchor = {
        lineNumber: l ? this.selectedRange.start : this.selectedRange.end,
        side: l
          ? this.selectedRange.side
          : (this.selectedRange.endSide ?? this.selectedRange.side),
      }),
        this.updateSelection(r, o, !1),
        this.notifySelectionStart(this.getCurrentSelectionRange()),
        (this.pointerSession = { mode: "selecting", pointerId: e.pointerId }),
        this.attachDocumentPointerListeners());
      return;
    }
    if (this.selectedRange?.start === r && this.selectedRange?.end === r) {
      const a = { lineNumber: r, side: o };
      ((this.selectionAnchor = a),
        (this.pointerSession = {
          mode: "pendingSingleLineUnselect",
          pointerId: e.pointerId,
          anchor: a,
          pending: a,
        }),
        this.attachDocumentPointerListeners());
      return;
    }
    (this.options.controlledSelection === !0
      ? (this.proposedSelectedRange = null)
      : (this.selectedRange = null),
      this.placeUtility(),
      (this.selectionAnchor = { lineNumber: r, side: o }),
      this.updateSelection(r, o, !1),
      this.notifySelectionStart(this.getCurrentSelectionRange()),
      (this.pointerSession = { mode: "selecting", pointerId: e.pointerId }),
      this.attachDocumentPointerListeners());
  }
  startGutterSelectionFromPointerDown(e) {
    const { onGutterUtilityClick: t } = this.options;
    if (t == null) return;
    const n = this.currentSelectionEnds(),
      i =
        n?.bottom ??
        this.resolveSelectionPoint(e, {
          source: "event-path",
          excludeUtility: !1,
        }),
      r = n?.top ?? i;
    i == null ||
      r == null ||
      (e.preventDefault(),
      e.stopPropagation(),
      (this.pointerSession = {
        mode: "gutterSelecting",
        pointerId: e.pointerId,
        anchor: r,
        current: i,
      }),
      (this.selectionAnchor = { lineNumber: r.lineNumber, side: r.side }),
      this.updateSelection(i.lineNumber, i.side, !1),
      this.notifySelectionStart(this.getCurrentSelectionRange()),
      this.attachDocumentPointerListeners());
  }
  handleDocumentPointerMove = (e) => {
    switch (this.pointerSession.mode) {
      case "idle":
        return;
      case "gutterSelecting": {
        if (e.pointerId !== this.pointerSession.pointerId) return;
        e.preventDefault();
        const t = this.resolveSelectionPoint(e, {
          source: "coordinates-first",
        });
        if (t == null) return;
        ((this.pointerSession.current = t),
          this.updateSelection(t.lineNumber, t.side));
        return;
      }
      case "selecting": {
        if (e.pointerId !== this.pointerSession.pointerId) return;
        e.preventDefault();
        const t = this.resolveSelectionInfo(e, {
          source: "coordinates-first",
          requireNumberColumn: !1,
        });
        if (t == null || this.selectionAnchor == null) return;
        this.updateSelection(t.lineNumber, t.eventSide);
        return;
      }
      case "pendingSingleLineUnselect": {
        if (e.pointerId !== this.pointerSession.pointerId) return;
        e.preventDefault();
        const t = this.resolveSelectionInfo(e, {
          source: "coordinates-first",
          requireNumberColumn: !1,
        });
        if (t == null || this.selectionAnchor == null) return;
        const n = { lineNumber: t.lineNumber, side: t.eventSide };
        if (go(this.pointerSession.pending, n)) return;
        (this.updateSelection(t.lineNumber, t.eventSide, !1),
          this.notifySelectionStart(this.getCurrentSelectionRange()),
          this.notifySelectionChangeDelta(),
          (this.pointerSession = {
            mode: "selecting",
            pointerId: e.pointerId,
          }));
        return;
      }
    }
  };
  handleDocumentPointerUp = (e) => {
    const { onGutterUtilityClick: t } = this.options;
    switch (this.pointerSession.mode) {
      case "idle":
        return;
      case "gutterSelecting": {
        const { pointerSession: n } = this;
        if (e.pointerId !== n.pointerId) return;
        e.preventDefault();
        const i = this.resolveSelectionPoint(e, {
          source: "coordinates-first",
        });
        i != null &&
          ((n.current = i), this.updateSelection(i.lineNumber, i.side));
        const r = this.buildSelectedLineRange(n.anchor, n.current);
        (t?.({ ...r }),
          (this.selectionAnchor = void 0),
          this.notifySelectionEnd(r),
          this.notifySelectionCommitted(r),
          this.clearProposedSelection(),
          this.clearPointerSession(),
          this.detachDocumentPointerListeners());
        return;
      }
      case "pendingSingleLineUnselect":
        if (e.pointerId !== this.pointerSession.pointerId) return;
        (e.preventDefault(),
          this.updateSelection(null, void 0, !1),
          (this.selectionAnchor = void 0),
          this.clearPendingSingleLineState(),
          this.detachDocumentPointerListeners(),
          this.notifySelectionEnd(this.getCurrentSelectionRange()),
          this.notifySelectionCommitted(this.getCurrentSelectionRange()),
          this.clearProposedSelection());
        return;
      case "selecting":
        if (e.pointerId !== this.pointerSession.pointerId) return;
        (e.preventDefault(),
          (this.selectionAnchor = void 0),
          this.detachDocumentPointerListeners(),
          this.clearPointerSession(),
          this.notifySelectionEnd(this.getCurrentSelectionRange()),
          this.notifySelectionCommitted(this.getCurrentSelectionRange()),
          this.clearProposedSelection());
    }
  };
  handleDocumentPointerCancel = (e) => {
    switch (this.pointerSession.mode) {
      case "idle":
        return;
      case "gutterSelecting":
      case "selecting":
      case "pendingSingleLineUnselect":
        if (
          "pointerId" in this.pointerSession &&
          e.pointerId !== this.pointerSession.pointerId
        )
          return;
        ((this.selectionAnchor = void 0),
          this.clearProposedSelection(),
          this.clearPendingSingleLineState(),
          this.clearPointerSession(),
          this.detachDocumentPointerListeners());
    }
  };
  clearHoveredLine() {
    this.hoveredLine != null &&
      (this.hoveredLine.lineElement.removeAttribute("data-hovered"),
      this.hoveredLine.numberElement.removeAttribute("data-hovered"),
      (this.hoveredLine = void 0));
  }
  setHoveredLine(e) {
    const { lineHoverHighlight: t = "disabled" } = this.options;
    (this.hoveredLine != null && this.clearHoveredLine(),
      (this.hoveredLine = e),
      t !== "disabled" &&
        ((t === "both" || t === "line") &&
          this.hoveredLine.lineElement.setAttribute("data-hovered", ""),
        (t === "both" || t === "number") &&
          this.hoveredLine.numberElement.setAttribute("data-hovered", "")));
  }
  clearHoveredToken() {
    this.hoveredToken != null && (this.hoveredToken = void 0);
  }
  setHoveredToken(e) {
    (this.hoveredToken != null && this.clearHoveredToken(),
      (this.hoveredToken = e));
  }
  ensureGutterUtilityNode(e) {
    if (
      (this.gutterUtilityContainer == null &&
        ((this.gutterUtilityContainer = document.createElement("div")),
        this.gutterUtilityContainer.setAttribute(
          "data-gutter-utility-slot",
          "",
        )),
      e)
    )
      (this.gutterUtilityButton != null &&
        (this.gutterUtilityButton.remove(),
        (this.gutterUtilityButton = void 0)),
        this.gutterUtilitySlot == null &&
          ((this.gutterUtilitySlot = document.createElement("slot")),
          (this.gutterUtilitySlot.name = "gutter-utility-slot")),
        this.gutterUtilitySlot.parentNode !== this.gutterUtilityContainer &&
          this.gutterUtilityContainer.replaceChildren(this.gutterUtilitySlot));
    else {
      if (
        (this.gutterUtilitySlot?.remove(),
        (this.gutterUtilitySlot = void 0),
        this.gutterUtilityButton == null)
      ) {
        const t = document.createElement("div");
        t.innerHTML = J(po());
        const n = t.firstElementChild;
        if (!(n instanceof HTMLButtonElement))
          throw new Error(
            "InteractionManager.ensureGutterUtilityNode: Node element should be a button",
          );
        (n.remove(), (this.gutterUtilityButton = n));
      }
      this.gutterUtilityButton.parentNode !== this.gutterUtilityContainer &&
        this.gutterUtilityContainer.replaceChildren(this.gutterUtilityButton);
    }
  }
  revealUtilityFromGutterPath(e) {
    if (this.placeUtilityFromSelection()) return;
    const t = this.resolvePointerTarget(e);
    et(t) && t.numberColumn && this.showUtilityOnLine(this.toEventBaseProps(t));
  }
  placeUtility() {
    if (!this.placeUtilityFromSelection()) {
      if (this.hoveredLine != null) {
        this.showUtilityOnLine(this.hoveredLine);
        return;
      }
      this.hideUtility();
    }
  }
  placeUtilityFromSelection() {
    const e = this.currentSelectionEnds();
    if (e == null) return !1;
    const t = this.targetForSelectionPoint(e.bottom);
    return (
      t == null
        ? this.hideUtility()
        : this.showUtilityOnLine(this.toEventBaseProps(t)),
      !0
    );
  }
  showUtilityOnLine(e) {
    this.gutterUtilityContainer != null &&
      ((this.gutterUtilityLine = e),
      e.numberElement.appendChild(this.gutterUtilityContainer));
  }
  hideUtility() {
    (this.gutterUtilityContainer?.remove(), (this.gutterUtilityLine = void 0));
  }
  currentSelectionEnds() {
    const e = this.getCurrentSelectionRange();
    return e == null ? void 0 : this.selectionEnds(e);
  }
  selectionEnds(e) {
    const t = { lineNumber: e.start, side: e.side },
      n = { lineNumber: e.end, side: e.endSide ?? e.side },
      i = this.selectionPointRowIndex(t),
      r = this.selectionPointRowIndex(n);
    if (!(i == null || r == null))
      return i > r ? { top: n, bottom: t } : { top: t, bottom: n };
  }
  selectionPointRowIndex(e) {
    const t = this.getLineIndex(e.lineNumber, e.side);
    if (t != null) return this.isSplitDiff() ? t[1] : t[0];
  }
  targetForSelectionPoint(e) {
    if (this.pre == null) return;
    const t = this.getLineIndex(e.lineNumber, e.side);
    if (t == null) return;
    const n = this.mode === "diff" ? `${t[0]},${t[1]}` : `${t[0]}`,
      i = this.pre.querySelectorAll(
        `[data-column-number="${e.lineNumber}"][data-line-index="${n}"]`,
      );
    for (const r of i) {
      if (!(r instanceof HTMLElement)) continue;
      const o = this.resolvePointerTarget(Qe(r));
      if (
        et(o) &&
        !(this.mode === "diff" && e.side != null && o.side !== e.side)
      )
        return o;
    }
  }
  attachDocumentPointerListeners() {
    this.hasDocumentPointerListeners ||
      (document.addEventListener("pointermove", this.handleDocumentPointerMove),
      document.addEventListener("pointerup", this.handleDocumentPointerUp),
      document.addEventListener(
        "pointercancel",
        this.handleDocumentPointerCancel,
      ),
      (this.hasDocumentPointerListeners = !0));
  }
  detachDocumentPointerListeners() {
    this.hasDocumentPointerListeners &&
      (document.removeEventListener(
        "pointermove",
        this.handleDocumentPointerMove,
      ),
      document.removeEventListener("pointerup", this.handleDocumentPointerUp),
      document.removeEventListener(
        "pointercancel",
        this.handleDocumentPointerCancel,
      ),
      (this.hasDocumentPointerListeners = !1));
  }
  clearPointerSession() {
    this.pointerSession = { mode: "idle" };
  }
  clearPendingSingleLineState() {
    this.pointerSession.mode === "pendingSingleLineUnselect" &&
      (this.pointerSession = { mode: "idle" });
  }
  selectionInfoFromPath(e, t) {
    const n = this.resolvePointerTarget(e);
    if (et(n) && !(t && !n.numberColumn) && n.splitLineIndex != null)
      return {
        lineIndex: n.splitLineIndex,
        lineNumber: n.lineNumber,
        eventSide: this.mode === "diff" ? n.side : void 0,
      };
  }
  resolveSelectionInfo(e, t) {
    const n = this.resolveSelectionPath(e, t);
    return n != null
      ? this.selectionInfoFromPath(n, t.requireNumberColumn)
      : void 0;
  }
  selectionPointFromPath(e) {
    const t = this.resolvePointerTarget(e);
    if (et(t))
      return {
        lineNumber: t.lineNumber,
        side: this.mode === "diff" ? t.side : void 0,
      };
  }
  resolveSelectionPoint(e, t) {
    const n = this.resolveSelectionPath(e, t);
    return n != null ? this.selectionPointFromPath(n) : void 0;
  }
  resolveSelectionPath(e, t) {
    const n = t.excludeUtility !== !1;
    switch (t.source) {
      case "event-path":
        return this.pathFromEventPath(e.composedPath(), n);
      case "coordinates-first": {
        const i = this.pathFromCoordinates(e, n);
        return i !== void 0
          ? (i ?? void 0)
          : this.pathFromEventPath(e.composedPath(), n);
      }
    }
  }
  pathFromCoordinates(e, t) {
    const n = this.hitTest(e);
    if (n !== void 0)
      return n === null ? null : (this.pathFromElement(n, t) ?? null);
  }
  pathFromEventPath(e, t) {
    if (!(t && ft(e))) {
      for (const n of e)
        if (n instanceof Element) return this.pathFromElement(n, t);
    }
  }
  pathFromElement(e, t) {
    const n = Qe(e);
    if (t && ft(n)) return;
    const i = bo(e);
    return i != null ? Qe(i) : this.pathFromAnnotationSlot(e);
  }
  pathFromAnnotationSlot(e) {
    const t = yo(e)?.previousElementSibling;
    if (
      t instanceof HTMLElement &&
      (t.hasAttribute("data-line") || t.hasAttribute("data-column-number"))
    )
      return Qe(t);
    const n = xo(Lo(e));
    if (n == null) return;
    const i = this.targetForSelectionPoint(n);
    return i != null ? Qe(i.lineElement) : void 0;
  }
  hitTest(e) {
    if (!Number.isFinite(e.clientX) || !Number.isFinite(e.clientY)) return;
    const t = this.pre?.getRootNode(),
      n = zn(t) ? t : zn(document) ? document : void 0;
    if (n != null) return n.elementFromPoint(e.clientX, e.clientY);
  }
  getLineIndex(e, t) {
    const { getLineIndex: n } = this.options;
    return n != null ? n(e, t) : [e - 1, e - 1];
  }
  getCurrentSelectionRange() {
    return this.proposedSelectedRange !== void 0
      ? this.proposedSelectedRange
      : this.selectedRange;
  }
  clearProposedSelection() {
    this.proposedSelectedRange = void 0;
  }
  updateSelection(e, t, n = !0) {
    const i = this.getCurrentSelectionRange();
    let r;
    if (e == null) r = null;
    else {
      const o = this.selectionAnchor?.side ?? t,
        s = this.selectionAnchor?.lineNumber ?? e;
      r = this.buildSelectionRange(s, e, o, t);
    }
    Pt(i ?? void 0, r ?? void 0) ||
      ((this.selectedRangeHighlightSide = void 0),
      (this.selectedRangeLineNumberOnly = !1),
      this.options.controlledSelection === !0
        ? (this.proposedSelectedRange = r)
        : ((this.selectedRange = r), W(this.renderSelection)),
      this.placeUtility(),
      n && this.notifySelectionChangeDelta());
  }
  getIndexesFromSelection(e, t) {
    if (this.pre == null) return;
    const n = this.getLineIndex(e.start, e.side),
      i = this.getLineIndex(e.end, e.endSide ?? e.side);
    return n != null && i != null
      ? { start: t ? n[1] : n[0], end: t ? i[1] : i[0] }
      : void 0;
  }
  getSelectionRenderState() {
    return {
      selectedLines: {
        range: this.selectedRange,
        highlightSide: this.selectedRangeHighlightSide,
        lineNumberOnly: this.selectedRangeLineNumberOnly,
      },
      editorActiveLine: {
        range:
          this.editorActiveLine == null
            ? null
            : {
                start: this.editorActiveLine,
                end: this.editorActiveLine,
                side: this.editorActiveLineSide,
              },
        highlightSide: this.editorActiveLineSide,
        lineNumberOnly: this.editorLineNumberOnly,
      },
    };
  }
  resolveLineRenderRange(e, t, n) {
    if (e.range == null) return;
    const i = this.getIndexesFromSelection(e.range, t);
    if (i == null)
      throw (
        console.error({ rowRange: i, range: e.range }),
        new Error(`InteractionManager.renderSelection: No valid ${n} rowRange`)
      );
    return i;
  }
  getLineRenderColumns(e, t, n, i) {
    if (this.pre == null || (t == null && i == null)) return [];
    const { children: r } = this.pre;
    if (r.length > 2)
      throw (
        console.error(r),
        new Error(
          "InteractionManager.renderSelection: Somehow there are more than 2 code elements...",
        )
      );
    const o = [];
    for (const s of r) {
      const a = s.hasAttribute("data-deletions")
          ? "deletions"
          : s.hasAttribute("data-additions")
            ? "additions"
            : void 0,
        l =
          e != null &&
          t != null &&
          (e.highlightSide == null || a == null || e.highlightSide === a),
        d =
          n != null &&
          i != null &&
          (n.highlightSide == null || a == null || n.highlightSide === a);
      if (!l && !d) continue;
      const [h, c] = s.children;
      if (!(h instanceof HTMLElement) || !(c instanceof HTMLElement))
        throw new Error(
          "InteractionManager.renderSelection: missing gutter or content element",
        );
      if (c.children.length !== h.children.length)
        throw new Error(
          "InteractionManager.renderSelection: gutter and content children dont match, something is wrong",
        );
      o.push({
        content: c,
        gutter: h,
        renderEditorActiveLine: d,
        renderSelectedLines: l,
      });
    }
    return o;
  }
  renderSelection = () => {
    Ne(this.renderSelection);
    const e = this.getSelectionRenderState();
    if (this.pre == null) return;
    const { editorActiveLine: t, selectedLines: n } = e,
      i = !Un(this.renderedSelectedLinesState, n),
      r = !Un(this.renderedEditorActiveLineState, t);
    if (!i && !r) return;
    (i && (this.renderedSelectedLinesState = void 0),
      r && (this.renderedEditorActiveLineState = void 0));
    const o = this.pre.getAttribute("data-diff-type") === "split",
      s = i ? this.resolveLineRenderRange(n, o, "selected-lines") : void 0,
      a = r ? this.resolveLineRenderRange(t, o, "editor-active-line") : void 0,
      l = this.getLineRenderColumns(i ? n : void 0, s, r ? t : void 0, a),
      d = n.range == null || l.some((g) => g.renderSelectedLines),
      h = t.range == null || l.some((g) => g.renderEditorActiveLine);
    if (i)
      for (const g of this.pre.querySelectorAll("[data-selected-line]"))
        g.removeAttribute("data-selected-line");
    if (r)
      for (const g of this.pre.querySelectorAll("[data-editor-active-line]"))
        g.removeAttribute("data-editor-active-line");
    const c = s == null ? void 0 : Math.min(s.start, s.end),
      u = s == null ? void 0 : Math.max(s.start, s.end),
      f = c === u,
      p = a?.start;
    for (const g of l) {
      const {
          content: m,
          gutter: y,
          renderEditorActiveLine: S,
          renderSelectedLines: b,
        } = g,
        L = Math.max(b ? (u ?? -1 / 0) : -1 / 0, S ? (p ?? -1 / 0) : -1 / 0),
        v = m.children.length;
      for (let k = 0; k < v; k++) {
        const C = m.children[k],
          x = y.children[k];
        if (!(C instanceof HTMLElement) || !(x instanceof HTMLElement))
          continue;
        const A = this.parseLineIndex(C, o);
        if ((A ?? 0) > L) break;
        if (A != null) {
          if (b && c != null && u != null && A >= c && A <= u) {
            let w = f ? "single" : A === c ? "first" : A === u ? "last" : "";
            (x.setAttribute("data-selected-line", w),
              n.lineNumberOnly ||
                (C.setAttribute("data-selected-line", w),
                x.nextSibling instanceof HTMLElement &&
                  C.nextSibling instanceof HTMLElement &&
                  (C.nextSibling.hasAttribute("data-line-annotation") ||
                    C.nextSibling.hasAttribute(
                      "data-merge-conflict-actions",
                    )) &&
                  (f
                    ? ((w = "last"),
                      C.setAttribute("data-selected-line", "first"))
                    : A === c
                      ? (w = "")
                      : A === u && C.setAttribute("data-selected-line", ""),
                  C.nextSibling.setAttribute("data-selected-line", w),
                  x.nextSibling.setAttribute("data-selected-line", w))));
          }
          S &&
            A === p &&
            (x.setAttribute("data-editor-active-line", ""),
            t.lineNumberOnly || C.setAttribute("data-editor-active-line", ""));
        }
      }
    }
    (i && d && (this.renderedSelectedLinesState = n),
      r && h && (this.renderedEditorActiveLineState = t));
  };
  notifySelectionCommitted(e) {
    this.options.onLineSelected?.(e);
  }
  notifySelectionChangeDelta() {
    this.options.onLineSelectionChange?.(
      this.getCurrentSelectionRange() ?? null,
    );
  }
  notifySelectionStart(e) {
    this.options.onLineSelectionStart?.(e);
  }
  notifySelectionEnd(e) {
    this.options.onLineSelectionEnd?.(e);
  }
  toEventBaseProps(e) {
    return this.mode === "file"
      ? {
          type: "line",
          lineElement: e.lineElement,
          lineNumber: e.lineNumber,
          numberColumn: e.numberColumn,
          numberElement: e.numberElement,
        }
      : {
          type: "diff-line",
          annotationSide: e.side,
          lineType: e.lineType,
          lineElement: e.lineElement,
          numberElement: e.numberElement,
          lineNumber: e.lineNumber,
          numberColumn: e.numberColumn,
        };
  }
  toTokenEventBaseProps({
    lineCharEnd: e,
    lineCharStart: t,
    lineNumber: n,
    side: i,
    tokenElement: r,
    tokenText: o,
  }) {
    return this.mode === "file"
      ? {
          type: "token",
          lineCharEnd: e,
          lineCharStart: t,
          lineNumber: n,
          tokenElement: r,
          tokenText: o,
        }
      : {
          type: "token",
          lineCharEnd: e,
          lineCharStart: t,
          lineNumber: n,
          side: i,
          tokenElement: r,
          tokenText: o,
        };
  }
  buildSelectedLineRange(e, t) {
    return this.buildSelectionRange(e.lineNumber, t.lineNumber, e.side, t.side);
  }
  buildSelectionRange(e, t, n, i) {
    return {
      start: e,
      end: t,
      ...(n != null ? { side: n } : {}),
      ...(n !== i && i != null ? { endSide: i } : {}),
    };
  }
  resolvePointerTarget(e) {
    let t = !1,
      n,
      i,
      r,
      o,
      s,
      a,
      l,
      d,
      h,
      c;
    for (const f of e) {
      if (!(f instanceof HTMLElement)) continue;
      if (c == null && f.hasAttribute("data-merge-conflict-action")) {
        const y = f.getAttribute("data-merge-conflict-action") ?? void 0,
          S = f.getAttribute("data-merge-conflict-conflict-index") ?? void 0,
          b = S != null ? Number.parseInt(S, 10) : NaN;
        Co(y) &&
          Number.isFinite(b) &&
          (c = {
            kind: "merge-conflict-action",
            resolution: y,
            conflictIndex: b,
          });
      }
      if (a == null && f.hasAttribute("data-char")) {
        a = f;
        const y = f.getAttribute("data-char");
        if (y != null) {
          const S = Number.parseInt(y, 10);
          if (!Number.isNaN(S)) {
            const b = f.textContent ?? "",
              L = S + b.length;
            (b.trim() !== "" ||
              this.options.enableTokenInteractionsOnWhitespace === !0) &&
              (l = {
                tokenElement: a,
                lineCharStart: S,
                lineCharEnd: L,
                tokenText: b,
              });
            continue;
          }
        }
      }
      const p =
        s == null ? (f.getAttribute("data-column-number") ?? void 0) : void 0;
      if (p != null) {
        ((s = f),
          (h = Number.parseInt(p, 10)),
          (t = !0),
          (n = Bn(f)),
          (o = f.getAttribute("data-line-index") ?? void 0));
        continue;
      }
      const g = r == null ? (f.getAttribute("data-line") ?? void 0) : void 0;
      if (g != null) {
        ((r = f),
          (h = Number.parseInt(g, 10)),
          (n = Bn(f)),
          (o = f.getAttribute("data-line-index") ?? void 0));
        continue;
      }
      if (
        d == null &&
        (f.hasAttribute("data-expand-button") ||
          f.hasAttribute("data-unmodified-lines"))
      ) {
        d = {
          hunkIndex: void 0,
          direction: f.hasAttribute("data-expand-up")
            ? "up"
            : f.hasAttribute("data-expand-down")
              ? "down"
              : "both",
          all: f.hasAttribute("data-expand-all-button"),
        };
        continue;
      }
      const m =
        d != null ? (f.getAttribute("data-expand-index") ?? void 0) : void 0;
      if (d != null && m != null) {
        const y = Number.parseInt(m, 10);
        Number.isNaN(y) || (d.hunkIndex = y);
        continue;
      }
      if (i == null && f.hasAttribute("data-code")) {
        i = f;
        break;
      }
    }
    if (c != null) return c;
    if (d?.hunkIndex != null)
      return {
        type: "line-info",
        hunkIndex: d.hunkIndex,
        direction: d.direction,
        all: d.all,
      };
    if (
      ((r ??=
        o != null ? Nn(i, `[data-line][data-line-index="${o}"]`) : void 0),
      (s ??=
        o != null
          ? Nn(i, `[data-column-number][data-line-index="${o}"]`)
          : void 0),
      i == null ||
        r == null ||
        s == null ||
        n == null ||
        h == null ||
        Number.isNaN(h))
    )
      return;
    const u = this.parseLineIndex(r, this.isSplitDiff());
    return l != null
      ? this.mode === "file"
        ? {
            kind: "token",
            lineType: n,
            lineElement: r,
            lineNumber: h,
            numberColumn: t,
            numberElement: s,
            side: void 0,
            splitLineIndex: u,
            ...l,
          }
        : {
            kind: "token",
            lineType: n,
            lineElement: r,
            lineNumber: h,
            numberColumn: t,
            numberElement: s,
            side: _n(n, i),
            splitLineIndex: u,
            ...l,
          }
      : this.mode === "file"
        ? {
            kind: "line",
            lineType: n,
            lineElement: r,
            lineNumber: h,
            numberColumn: t,
            numberElement: s,
            side: void 0,
            splitLineIndex: u,
          }
        : {
            kind: "line",
            lineType: n,
            lineElement: r,
            lineNumber: h,
            numberColumn: t,
            numberElement: s,
            side: _n(n, i),
            splitLineIndex: u,
          };
  }
  isSplitDiff() {
    return this.pre?.getAttribute("data-diff-type") === "split";
  }
  parseLineIndex(e, t) {
    const n = (e.getAttribute("data-line-index") ?? "")
      .split(",")
      .map((i) => Number.parseInt(i, 10))
      .filter((i) => !Number.isNaN(i));
    if (t && n.length === 2) return n[1];
    if (!t) return n[0];
  }
};
function st(
  {
    enableTokenInteractionsOnWhitespace: e,
    enableGutterUtility: t,
    lineHoverHighlight: n,
    onGutterUtilityClick: i,
    onLineClick: r,
    onLineEnter: o,
    onLineLeave: s,
    onLineNumberClick: a,
    onTokenClick: l,
    onTokenEnter: d,
    onTokenLeave: h,
    renderGutterUtility: c,
    __debugPointerEvents: u,
    enableLineSelection: f,
    controlledSelection: p,
    onLineSelected: g,
    onLineSelectionStart: m,
    onLineSelectionChange: y,
    onLineSelectionEnd: S,
  },
  b,
  L,
  v,
) {
  return {
    enableTokenInteractionsOnWhitespace: e,
    enableGutterUtility: mo({
      enableGutterUtility: t,
      renderGutterUtility: c,
      onGutterUtilityClick: i,
    }),
    usesCustomGutterUtility: c != null,
    lineHoverHighlight: n,
    onGutterUtilityClick: i,
    onHunkExpand: b,
    onMergeConflictActionClick: v,
    onLineClick: r,
    onLineEnter: o,
    onLineLeave: s,
    onLineNumberClick: a,
    onTokenClick: l,
    onTokenEnter: d,
    onTokenLeave: h,
    __debugPointerEvents: u,
    enableLineSelection: f,
    controlledSelection: p,
    onLineSelected: g,
    onLineSelectionStart: m,
    onLineSelectionChange: y,
    onLineSelectionEnd: S,
    getLineIndex: L,
  };
}
function mo({
  enableGutterUtility: e,
  renderGutterUtility: t,
  onGutterUtilityClick: n,
}) {
  if (n != null && t != null)
    throw new Error(
      "Cannot use both 'onGutterUtilityClick' and 'renderGutterUtility'. Use only one gutter utility API.",
    );
  return e ?? !1;
}
function et(e) {
  return e != null && "kind" in e && e.kind === "line";
}
function At(e) {
  return e != null && "kind" in e && e.kind === "token";
}
function Wt(e) {
  return et(e) || At(e);
}
function So(e) {
  return "type" in e && e.type === "line-info";
}
function vo(e) {
  return "kind" in e && e.kind === "merge-conflict-action";
}
function Co(e) {
  return e === "current" || e === "incoming" || e === "both";
}
function Nn(e, t) {
  const n = e?.querySelector(t);
  return n instanceof HTMLElement ? n : void 0;
}
function Qe(e) {
  const t = [];
  let n = e;
  for (; n != null;) (t.push(n), (n = n.parentNode));
  return t;
}
function bo(e) {
  const t = e.closest("[data-line], [data-column-number]");
  if (t instanceof HTMLElement) return t;
  const n = e.closest(
    '[data-line-annotation], [data-gutter-buffer="annotation"]',
  );
  if (!(n instanceof HTMLElement)) return;
  const i = n.previousElementSibling;
  return i instanceof HTMLElement &&
    (i.hasAttribute("data-line") || i.hasAttribute("data-column-number"))
    ? i
    : void 0;
}
function yo(e) {
  const t = e.closest('[slot^="annotation-"]'),
    n = (
      t instanceof HTMLElement && t.assignedSlot != null
        ? t.assignedSlot
        : e.tagName === "SLOT"
          ? e
          : void 0
    )?.closest("[data-line-annotation]");
  return n instanceof HTMLElement ? n : void 0;
}
function Lo(e) {
  const t = e.closest('[slot^="annotation-"]');
  if (t instanceof HTMLElement) return t.getAttribute("slot") ?? void 0;
  if (e instanceof HTMLElement) {
    const n = e.getAttribute("name") ?? void 0;
    return n != null && n.startsWith("annotation-") ? n : void 0;
  }
}
function xo(e) {
  if (e == null) return;
  const t = /^annotation-(?:(additions|deletions)-)?(\d+)$/.exec(e);
  if (t == null) return;
  const n = Number.parseInt(t[2], 10);
  if (!(!Number.isFinite(n) || n <= 0)) return { lineNumber: n, side: t[1] };
}
function zn(e) {
  return e != null && typeof e.elementFromPoint == "function";
}
function ko(e, t) {
  return e === t || Pt(e ?? void 0, t ?? void 0);
}
function Un(e, t) {
  return (
    e?.highlightSide === t.highlightSide &&
    e?.lineNumberOnly === t.lineNumberOnly &&
    ko(e?.range ?? null, t.range)
  );
}
function _n(e, t) {
  switch (e) {
    case "change-deletion":
      return "deletions";
    case "change-addition":
      return "additions";
    default:
      return t.hasAttribute("data-deletions") ? "deletions" : "additions";
  }
}
function Bn(e) {
  const t = e.getAttribute("data-line-type");
  if (t != null)
    switch (t) {
      case "change-deletion":
      case "change-addition":
      case "context":
      case "context-expanded":
        return t;
      default:
        return;
    }
}
function me(e = "none", t, ...n) {
  switch (e) {
    case "none":
      return;
    case "both":
      break;
    case "click":
      if (t !== "click") return;
      break;
    case "move":
      if (t !== "move") return;
      break;
  }
  console.log(...n);
}
var ji = class Ce {
  static resizeObserver;
  static managersByElement = new Map();
  static getResizeObserver() {
    const t =
      Ce.resizeObserver ?? new ResizeObserver(Ce.handleSharedResizeEntries);
    return ((Ce.resizeObserver = t), t);
  }
  static handleSharedResizeEntries(t) {
    const n = new Map();
    for (const i of t) {
      const r = Ce.managersByElement.get(i.target);
      if (r == null) continue;
      const o = n.get(r);
      o == null ? n.set(r, [i]) : o.push(i);
    }
    for (const [i, r] of n) i.handleResizeEntries(r);
  }
  observedNodes = new Map();
  setup(t, { disableAnnotations: n, columnVariables: i = "apply" }) {
    const r = new Set(),
      o = i === "apply";
    let s = 0;
    const a = new Map(this.observedNodes);
    this.observedNodes.clear();
    for (const l of t.children) {
      if (s === 2) break;
      const d = (() => {
        if (l instanceof HTMLElement && l.tagName === "CODE") return l;
      })();
      if (d == null) continue;
      s++;
      let h = a.get(d);
      if (h != null && h.type !== "code")
        throw new Error(
          "ResizeManager.setup: somehow a code node is being used for an annotation, should be impossible",
        );
      let c = d.firstElementChild;
      (c instanceof HTMLElement || (c = null),
        h != null
          ? (this.observedNodes.set(d, h),
            a.delete(d),
            h.numberElement !== c
              ? (h.numberElement != null &&
                  (this.unobserve(h.numberElement), a.delete(h.numberElement)),
                c != null &&
                  (this.observe(c), a.delete(c), this.observedNodes.set(c, h)),
                (h.numberElement = c),
                (h.numberWidth = 0))
              : h.numberElement != null
                ? (a.delete(h.numberElement),
                  this.observedNodes.set(h.numberElement, h))
                : (h.numberWidth = 0),
            Ro(h, o))
          : ((h = {
              type: "code",
              codeElement: d,
              numberElement: c,
              codeWidth: "auto",
              numberWidth: 0,
              applyColumnVariables: o,
            }),
            this.observedNodes.set(d, h),
            this.observe(d),
            c != null && (this.observedNodes.set(c, h), this.observe(c))));
    }
    if (s > 1 && !n) {
      const l = t.querySelectorAll('[data-line-annotation*=","]'),
        d = new Map();
      for (const h of l) {
        if (!(h instanceof HTMLElement)) continue;
        const c = h.getAttribute("data-line-annotation") ?? "";
        if (!/^-?\d+,-?\d+$/.test(c)) {
          console.error(
            "DiffFileRenderer.setupResizeObserver: Invalid element or annotation",
            { lineAnnotation: c, element: h },
          );
          continue;
        }
        let u = d.get(c);
        (u == null && ((u = []), d.set(c, u)), u.push(h));
      }
      for (const [h, c] of d) {
        if (c.length !== 2) {
          console.error("DiffFileRenderer.setupResizeObserver: Bad Pair", h, c);
          continue;
        }
        const [u, f] = c,
          p = u.firstElementChild,
          g = f.firstElementChild;
        if (
          !(u instanceof HTMLElement) ||
          !(f instanceof HTMLElement) ||
          !(p instanceof HTMLElement) ||
          !(g instanceof HTMLElement)
        )
          continue;
        let m = a.get(p);
        if (m != null) {
          (this.observedNodes.set(p, m),
            this.observedNodes.set(g, m),
            a.delete(p),
            a.delete(g));
          continue;
        }
        const y = p.getBoundingClientRect().height,
          S = g.getBoundingClientRect().height;
        ((m = {
          type: "annotations",
          column1: { container: u, child: p, childHeight: y },
          column2: { container: f, child: g, childHeight: S },
          currentHeight: "auto",
        }),
          r.add({ child1: p, child2: g, item: m, newHeight: Math.max(y, S) }));
      }
      for (const h of r)
        (this.applyNewHeight(h.item, h.newHeight),
          this.observedNodes.set(h.child1, h.item),
          this.observedNodes.set(h.child2, h.item),
          this.observe(h.child1),
          this.observe(h.child2));
      r.clear();
    }
    for (const [l, d] of a)
      (this.unobserve(l), d.type === "code" ? To(d) : Ao(d));
    a.clear();
  }
  cleanUp() {
    for (const t of this.observedNodes.keys()) this.unobserve(t);
    this.observedNodes.clear();
  }
  observe(t) {
    const { managersByElement: n } = Ce,
      i = n.get(t);
    if (i !== this) {
      if (i != null && i !== this)
        throw new Error(
          "ResizeManager.observe: element is already owned by another ResizeManager",
        );
      (n.set(t, this), Ce.getResizeObserver().observe(t));
    }
  }
  unobserve(t) {
    const { managersByElement: n, resizeObserver: i } = Ce,
      r = n.get(t);
    if (r != null) {
      if (r !== this)
        throw new Error(
          "ResizeManager.unobserve: element is owned by another ResizeManager",
        );
      (n.delete(t),
        i?.unobserve(t),
        i != null &&
          n.size === 0 &&
          (i.disconnect(), (Ce.resizeObserver = void 0)));
    }
  }
  handleResizeEntries(t) {
    const n = new Map(),
      i = new Set();
    for (const r of t) {
      const { target: o, borderBoxSize: s, contentBoxSize: a } = r;
      if (!(o instanceof HTMLElement)) {
        console.error(
          "ResizeManager.handleResizeEntries: Invalid element for ResizeObserver",
          r,
        );
        continue;
      }
      const l = this.observedNodes.get(o);
      if (l == null) {
        console.error(
          "ResizeManager.handleResizeEntries: Not a valid observed node",
          r,
        );
        continue;
      }
      if (l.type === "annotations") {
        const d = (() => {
          if (o === l.column1.child) return l.column1;
          if (o === l.column2.child) return l.column2;
        })();
        if (d == null) {
          console.error(
            "ResizeManager.handleResizeEntries: Couldn't find a column for",
            { item: l, target: o },
          );
          continue;
        }
        ((d.childHeight = s[0].blockSize), i.add(l));
      } else if (l.type === "code") {
        const d = n.get(l) ?? {},
          h = a[0].inlineSize;
        (o === l.codeElement
          ? (d.codeInlineSize = h)
          : o === l.numberElement && (d.numberInlineSize = h),
          n.set(l, d));
      }
    }
    (this.applyAnnotationUpdates(i),
      i.clear(),
      this.applyColumnUpdates(n),
      n.clear());
  }
  applyAnnotationUpdates(t) {
    for (const n of t)
      this.applyNewHeight(
        n,
        Math.max(n.column1.childHeight, n.column2.childHeight),
      );
  }
  applyColumnUpdates = (t) => {
    for (const [n, i] of t) {
      const r = i.codeInlineSize != null ? wo(i.codeInlineSize) : n.codeWidth,
        o = i.numberInlineSize != null ? Eo(i.numberInlineSize) : n.numberWidth,
        s = r !== n.codeWidth,
        a = o !== n.numberWidth;
      (!s && !a) ||
        ((n.codeWidth = r),
        (n.numberWidth = o),
        n.applyColumnVariables &&
          Ki(n, { codeWidthChanged: s, numberWidthChanged: a }));
    }
  };
  applyNewHeight(t, n) {
    n !== t.currentHeight &&
      ((t.currentHeight = Math.max(n, 0)),
      t.column1.container.style.setProperty(
        "--diffs-annotation-min-height",
        `${t.currentHeight}px`,
      ),
      t.column2.container.style.setProperty(
        "--diffs-annotation-min-height",
        `${t.currentHeight}px`,
      ));
  }
};
function wo(e) {
  const t = Math.max(Math.floor(e), 0);
  return t === 0 ? "auto" : t;
}
function Eo(e) {
  return Math.max(Math.ceil(e), 0);
}
function Ro(e, t) {
  e.applyColumnVariables !== t &&
    ((e.applyColumnVariables = t),
    t ? Ki(e, { codeWidthChanged: !0, numberWidthChanged: !0 }) : Xi(e));
}
function Ki(e, { codeWidthChanged: t, numberWidthChanged: n }) {
  const { codeElement: i, codeWidth: r, numberWidth: o } = e;
  if (
    (t &&
      i.style.setProperty(
        "--diffs-column-width",
        `${typeof r == "number" ? `${r}px` : "auto"}`,
      ),
    n &&
      i.style.setProperty(
        "--diffs-column-number-width",
        `${o === 0 ? "auto" : `${o}px`}`,
      ),
    t || (n && r !== "auto"))
  ) {
    const s = typeof r == "number" ? Math.max(r - o, 0) : 0;
    i.style.setProperty(
      "--diffs-column-content-width",
      `${s > 0 ? `${s}px` : "auto"}`,
    );
  }
}
function Xi(e) {
  (e.codeElement.style.removeProperty("--diffs-column-content-width"),
    e.codeElement.style.removeProperty("--diffs-column-number-width"),
    e.codeElement.style.removeProperty("--diffs-column-width"));
}
function To(e) {
  e.codeElement.isConnected && Xi(e);
}
function Ao(e) {
  (e.column1.container.isConnected &&
    e.column1.container.style.removeProperty("--diffs-annotation-min-height"),
    e.column2.container.isConnected &&
      e.column2.container.style.removeProperty(
        "--diffs-annotation-min-height",
      ));
}
function Io(e) {
  return Sn.hasResolvedThemes(e);
}
function Re(e, t) {
  return (
    Ue(e.theme, t.theme) &&
    e.useTokenTransformer === t.useTokenTransformer &&
    e.tokenizeMaxLineLength === t.tokenizeMaxLineLength
  );
}
function Vt(e, t) {
  return e == null || t == null
    ? e === t
    : e.startingLine === t.startingLine &&
        e.totalLines === t.totalLines &&
        e.bufferBefore === t.bufferBefore &&
        e.bufferAfter === t.bufferAfter;
}
function on(e) {
  return R({
    tagName: "div",
    children: [
      R({
        tagName: "div",
        children: e.annotations?.map((t) =>
          R({ tagName: "slot", properties: { name: t } }),
        ),
        properties: { "data-annotation-content": "" },
      }),
    ],
    properties: { "data-line-annotation": `${e.hunkIndex},${e.lineIndex}` },
  });
}
function Ho(e) {
  switch (e) {
    case "file":
      return "diffs-icon-file-code";
    case "change":
      return "diffs-icon-symbol-modified";
    case "new":
      return "diffs-icon-symbol-added";
    case "deleted":
      return "diffs-icon-symbol-deleted";
    case "rename-pure":
    case "rename-changed":
      return "diffs-icon-symbol-moved";
  }
}
function Yi({ fileOrDiff: e, mode: t, stickyHeader: n }) {
  const i = "type" in e ? e : void 0,
    r = {
      "data-diffs-header": t,
      "data-change-type": i?.type,
      "data-sticky": n ? "" : void 0,
    };
  return R({
    tagName: "div",
    children: [
      t === "custom"
        ? R({ tagName: "slot", properties: { name: vn } })
        : Mo({
            name: e.name,
            prevName: "prevName" in e ? e.prevName : void 0,
            iconType: i?.type ?? "file",
          }),
      ...(t === "custom" ? [] : [Do(i)]),
    ],
    properties: r,
  });
}
function Mo({ name: e, prevName: t, iconType: n }) {
  const i = [
    R({ tagName: "slot", properties: { name: Cn } }),
    Ot({ name: Ho(n), properties: { "data-change-icon": n } }),
  ];
  return (
    t != null &&
      (i.push(
        R({
          tagName: "div",
          children: [R({ tagName: "bdi", children: [X(t)] })],
          properties: { "data-prev-name": "" },
        }),
      ),
      i.push(
        Ot({
          name: "diffs-icon-arrow-right-short",
          properties: { "data-rename-icon": "" },
        }),
      )),
    i.push(
      R({
        tagName: "div",
        children: [R({ tagName: "bdi", children: [X(e)] })],
        properties: { "data-title": "" },
      }),
    ),
    i.push(R({ tagName: "slot", properties: { name: bn } })),
    R({
      tagName: "div",
      children: i,
      properties: { "data-header-content": "" },
    })
  );
}
function Do(e) {
  const t = [];
  if (e != null) {
    let n = 0,
      i = 0;
    for (const r of e.hunks) ((n += r.additionLines), (i += r.deletionLines));
    ((i > 0 || n === 0) &&
      t.push(
        R({
          tagName: "span",
          children: [X(`-${i}`)],
          properties: { "data-deletions-count": "" },
        }),
      ),
      (n > 0 || i === 0) &&
        t.push(
          R({
            tagName: "span",
            children: [X(`+${n}`)],
            properties: { "data-additions-count": "" },
          }),
        ));
  }
  return (
    t.push(R({ tagName: "slot", properties: { name: yn } })),
    R({ tagName: "div", children: t, properties: { "data-metadata": "" } })
  );
}
function Qi(e) {
  return R({ tagName: "pre", properties: Fo(e) });
}
function Fo({
  diffIndicators: e,
  disableBackground: t,
  disableLineNumbers: n,
  overflow: i,
  split: r,
  totalLines: o,
  type: s,
  customProperties: a,
}) {
  return {
    ...a,
    "data-diff": s === "diff" ? "" : void 0,
    "data-file": s === "file" ? "" : void 0,
    "data-diff-type": s === "diff" ? (r ? "split" : "single") : void 0,
    "data-overflow": i,
    "data-disable-line-numbers": n ? "" : void 0,
    "data-background": t ? void 0 : "",
    "data-indicators": e === "bars" || e === "classic" ? e : void 0,
    style: `--diffs-min-number-column-width-default:${`${o}`.length}ch;`,
  };
}
function xn(e, { theme: t, preferredHighlighter: n = "shiki-js" }) {
  return { langs: [e ?? "text"], themes: ht(t), preferredHighlighter: n };
}
function fe(e) {
  return `annotation-${"side" in e ? `${e.side}-` : ""}${e.lineNumber}`;
}
function Nt(e) {
  return (
    e.startingLine === 0 &&
    e.totalLines === 1 / 0 &&
    e.bufferBefore === 0 &&
    e.bufferAfter === 0
  );
}
function Po(e, t, n) {
  const i = typeof n.lineInfo == "function" ? n.lineInfo(t) : n.lineInfo[t - 1];
  if (i == null) {
    const r = `processLine: line ${t}, contains no state.lineInfo`;
    throw (console.error(r, { node: e, line: t, state: n }), new Error(r));
  }
  return (
    (e.tagName = "div"),
    (e.properties["data-line"] = i.lineNumber),
    (e.properties["data-alt-line"] = i.altLineNumber),
    (e.properties["data-line-type"] = i.type),
    (e.properties["data-line-index"] = i.lineIndex),
    e.children.length === 0 &&
      e.children.push(
        X(`
`),
      ),
    e
  );
}
const pt = Symbol("no-token"),
  qt = Symbol("multiple-tokens");
function Zi(e) {
  const t = Oo(e);
  if (t != null) return t;
  let n = pt;
  const i = [];
  let r = [],
    o;
  const s = () => {
      if (r.length === 0 || o == null) {
        ((r = []), (o = void 0));
        return;
      }
      if (r.length === 1) {
        const l = r[0];
        if (l?.type === "element") {
          No(l, o);
          for (const d of l.children) It(d);
        } else It(l);
        (i.push(l), (r = []), (o = void 0));
        return;
      }
      for (const l of r) It(l);
      (i.push(
        R({ tagName: "span", properties: { "data-char": o }, children: r }),
      ),
        (r = []),
        (o = void 0));
    },
    a = (l) => {
      if (l !== pt) {
        if (l === qt) {
          n = qt;
          return;
        }
        if (n === pt) {
          n = l;
          return;
        }
        n !== l && (n = qt);
      }
    };
  for (const l of e.children) {
    const d = l.type === "element" ? Zi(l) : pt;
    if ((a(d), typeof d != "number")) {
      (s(), i.push(l));
      continue;
    }
    (o != null && o !== d && s(), (o ??= d), r.push(l));
  }
  return (s(), (e.children = i), n);
}
function Oo(e) {
  const t = e.properties["data-char"];
  if (typeof t == "number") return t;
}
function It(e) {
  if (e.type === "element") {
    e.properties["data-char"] = void 0;
    for (const t of e.children) It(t);
  }
}
function No(e, t) {
  e.properties["data-char"] = t;
}
function zo(e = {}) {
  const {
      classPrefix: t = "__shiki_",
      classSuffix: n = "",
      classReplacer: i = (a) => a,
    } = e,
    r = new Map();
  function o(a) {
    return Object.entries(a)
      .map(([l, d]) => `${l}:${d}`)
      .join(";");
  }
  function s(a) {
    const l = typeof a == "string" ? a : o(a);
    let d = t + Uo(l) + n;
    return (
      (d = i(d)),
      r.has(d) || r.set(d, typeof a == "string" ? a : { ...a }),
      d
    );
  }
  return {
    name: "@shikijs/transformers:style-to-class",
    pre(a) {
      if (!a.properties.style) return;
      const l = s(a.properties.style);
      (delete a.properties.style, this.addClassToHast(a, l));
    },
    tokens(a) {
      for (const l of a)
        for (const d of l) {
          if (!d.htmlStyle) continue;
          const h = s(d.htmlStyle);
          ((d.htmlStyle = {}),
            (d.htmlAttrs ||= {}),
            d.htmlAttrs.class
              ? (d.htmlAttrs.class += ` ${h}`)
              : (d.htmlAttrs.class = h));
        }
    },
    getClassRegistry() {
      return r;
    },
    getCSS() {
      let a = "";
      for (const [l, d] of r.entries())
        a += `.${l}{${typeof d == "string" ? d : o(d)}}`;
      return a;
    },
    clearRegistry() {
      r.clear();
    },
  };
}
function Uo(e, t = 0) {
  let n = 3735928559 ^ t,
    i = 1103547991 ^ t;
  for (let r = 0, o; r < e.length; r++)
    ((o = e.charCodeAt(r)),
      (n = Math.imul(n ^ o, 2654435761)),
      (i = Math.imul(i ^ o, 1597334677)));
  return (
    (n = Math.imul(n ^ (n >>> 16), 2246822507)),
    (n ^= Math.imul(i ^ (i >>> 13), 3266489909)),
    (i = Math.imul(i ^ (i >>> 16), 2246822507)),
    (i ^= Math.imul(n ^ (n >>> 13), 3266489909)),
    (4294967296 * (2097151 & i) + (n >>> 0)).toString(36).slice(0, 6)
  );
}
function Ji(e = !1, t = !1) {
  const n = { lineInfo: [] },
    i = [
      {
        line(r) {
          return (delete r.properties.class, r);
        },
        pre(r) {
          const o = fo(r),
            s = [];
          if (o != null) {
            let a = 1;
            for (const l of o.children)
              l.type === "element" && (e && Zi(l), s.push(Po(l, a, n)), a++);
            o.children = s;
          }
          return r;
        },
        ...(e
          ? {
              tokens(r) {
                for (const o of r) {
                  let s = 0;
                  for (const a of o) {
                    const l = a;
                    ((l.__lineChar ??= s), (s += a.content.length));
                  }
                }
              },
              preprocess(r, o) {
                o.mergeWhitespaces = "never";
              },
              span(r, o, s, a, l) {
                if (l?.offset != null && l.content != null) {
                  const d = l.__lineChar;
                  return (d != null && (r.properties["data-char"] = d), r);
                }
                return r;
              },
            }
          : null),
      },
    ];
  return (
    t && i.push(_o, Vn),
    e &&
      i.push({
        line: (r) => (
          r.type === "element" &&
            r.children.length === 0 &&
            r.children.push({
              type: "element",
              tagName: "br",
              properties: {},
              children: [],
            }),
          r
        ),
      }),
    { state: n, transformers: i, toClass: Vn }
  );
}
const Vn = zo({ classPrefix: "hl-" }),
  _o = {
    name: "token-style-normalizer",
    tokens(e) {
      for (const t of e)
        for (const n of t) {
          if (n.htmlStyle != null) continue;
          const i = {};
          (n.color != null && (i.color = n.color),
            n.bgColor != null && (i["background-color"] = n.bgColor),
            n.fontStyle != null &&
              n.fontStyle !== 0 &&
              ((n.fontStyle & 1) !== 0 && (i["font-style"] = "italic"),
              (n.fontStyle & 2) !== 0 && (i["font-weight"] = "bold"),
              (n.fontStyle & 4) !== 0 && (i["text-decoration"] = "underline")),
            Object.keys(i).length > 0 && (n.htmlStyle = i));
        }
    },
  };
function q(e) {
  return `--${e === "token" ? "diffs-token" : "diffs"}-`;
}
const Bo = /^#(?:[0-9a-f]{3}0|[0-9a-f]{6}00)$/i,
  Vo = /^0(?:\.0+)?%?$/;
function $o(e) {
  const t = e.indexOf("(");
  if (t <= 0 || !e.endsWith(")")) return;
  const n = e.slice(0, t).trim();
  if (!/^(?:rgb|rgba|hsl|hsla|hwb|lab|lch|oklab|oklch|color)$/i.test(n)) return;
  const i = e.slice(t + 1, -1).trim();
  if (i.length === 0) return;
  const r = i.lastIndexOf("/");
  if (r !== -1) return i.slice(r + 1).trim();
  if (/^(?:rgba|hsla)$/i.test(n)) {
    const o = i.split(",");
    if (o.length === 4) return o[3]?.trim();
  }
}
function Go(e) {
  const t = /^#([0-9a-f]{3}|[0-9a-f]{6}|[0-9a-f]{8})\b/i.exec(e.trim());
  if (t == null) return null;
  const n = t[1];
  let i,
    r = 1;
  return (
    n.length === 3
      ? (i = n
          .split("")
          .map((o) => o + o)
          .join(""))
      : n.length === 6
        ? (i = n)
        : ((i = n.slice(0, 6)), (r = parseInt(n.slice(6, 8), 16) / 255)),
    [
      parseInt(i.slice(0, 2), 16),
      parseInt(i.slice(2, 4), 16),
      parseInt(i.slice(4, 6), 16),
      r,
    ]
  );
}
function jt(e) {
  if (e == null) return null;
  const t = Go(e);
  if (t == null) return null;
  const n = t[0] / 255,
    i = t[1] / 255,
    r = t[2] / 255,
    o = (s) => (s <= 0.03928 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4);
  return 0.2126 * o(n) + 0.7152 * o(i) + 0.0722 * o(r);
}
function $n(e) {
  if (e == null) return !1;
  const t = e.trim().toLowerCase();
  if (t === "transparent" || Bo.test(t)) return !0;
  const n = $o(t);
  return n != null && Vo.test(n);
}
function Wo(e, t, n) {
  if (t == null || n == null) return !1;
  const i = jt(e),
    r = jt(t),
    o = jt(n);
  return i == null || r == null || o == null
    ? !1
    : Math.abs(i - o) < Math.abs(i - r);
}
const Gn = new WeakMap();
function Kt(e) {
  const t = Gn.get(e);
  if (t != null) return t;
  const n = e.colors ?? {},
    i = { ...n },
    r = n["editor.background"] ?? e.bg,
    o = n["editor.foreground"] ?? e.fg,
    s = n["sideBar.background"] ?? r,
    a = n["sideBar.foreground"] ?? o;
  (le(i, "editor.background", r),
    le(i, "editor.foreground", o),
    le(i, "sideBar.background", s),
    le(i, "sideBar.foreground", a),
    le(i, "input.background", n["input.background"] ?? s),
    le(
      i,
      "sideBarSectionHeader.foreground",
      n["sideBarSectionHeader.foreground"] ?? a,
    ),
    le(
      i,
      "list.activeSelectionForeground",
      n["list.activeSelectionForeground"] ?? a,
    ),
    le(
      i,
      "gitDecoration.addedResourceForeground",
      Xt(
        n["gitDecoration.addedResourceForeground"],
        n["terminal.ansiGreen"],
        n["editorGutter.addedBackground"],
      ),
    ),
    le(
      i,
      "gitDecoration.modifiedResourceForeground",
      Xt(
        n["gitDecoration.modifiedResourceForeground"],
        n["terminal.ansiBlue"],
        n["editorGutter.modifiedBackground"],
      ),
    ),
    le(
      i,
      "gitDecoration.deletedResourceForeground",
      Xt(
        n["gitDecoration.deletedResourceForeground"],
        n["terminal.ansiRed"],
        n["editorGutter.deletedBackground"],
      ),
    ));
  const l =
    ($n(n["list.focusOutline"]) ? void 0 : n["list.focusOutline"]) ??
    ($n(n.focusBorder) ? void 0 : n.focusBorder);
  l != null ? (i["list.focusOutline"] = l) : delete i["list.focusOutline"];
  const d = n["list.hoverBackground"];
  d != null && (qo(d, s) || Wo(d, s, a)) && delete i["list.hoverBackground"];
  const h = Object.freeze({ ...e, colors: Object.freeze(i) });
  return (Gn.set(e, h), h);
}
function le(e, t, n) {
  n != null && n !== "" && (e[t] = n);
}
function Xt(...e) {
  for (const t of e) if (t != null && t !== "") return t;
}
function qo(e, t) {
  return t != null && e.toLowerCase() === t.toLowerCase();
}
function kn({ theme: e = U, highlighter: t, prefix: n }) {
  let i = "";
  if (typeof e == "string") {
    const r = t.getTheme(e),
      o = Kt(r);
    ((i += `color:${o.fg};`),
      (i += `background-color:${o.bg};`),
      (i += `${q("global")}fg:${o.fg};`),
      (i += `${q("global")}bg:${o.bg};`),
      (i += Yt(r, n)));
  } else {
    let r = t.getTheme(e.dark),
      o = Kt(r);
    ((i += `${q("global")}dark:${o.fg};`),
      (i += `${q("global")}dark-bg:${o.bg};`),
      (i += Yt(r, "dark")),
      (r = t.getTheme(e.light)),
      (o = Kt(r)),
      (i += `${q("global")}light:${o.fg};`),
      (i += `${q("global")}light-bg:${o.bg};`),
      (i += Yt(r, "light")));
  }
  return i;
}
function Yt(e, t) {
  t = t != null ? `${t}-` : "";
  let n = "";
  const i =
    e.colors?.["gitDecoration.addedResourceForeground"] ??
    e.colors?.["terminal.ansiGreen"];
  i != null && (n += `${q("global")}${t}addition-color:${i};`);
  const r =
    e.colors?.["gitDecoration.deletedResourceForeground"] ??
    e.colors?.["terminal.ansiRed"];
  r != null && (n += `${q("global")}${t}deletion-color:${r};`);
  const o =
    e.colors?.["gitDecoration.modifiedResourceForeground"] ??
    e.colors?.["terminal.ansiBlue"];
  return (o != null && (n += `${q("global")}${t}modified-color:${o};`), n);
}
function sn(e) {
  let t = e.children[0];
  for (; t != null;) {
    if (t.type === "element" && t.tagName === "code") return t.children;
    "children" in t ? (t = t.children[0]) : (t = null);
  }
  throw (console.error(e), new Error("getLineNodes: Unable to find children"));
}
const Wn = 10,
  qn = 13,
  jo = 4,
  Ko = 40;
function Xo(e) {
  const t = [0];
  let n = e.indexOf("\r"),
    i = e.indexOf(`
`),
    r = 0,
    o = 0;
  for (; n !== -1 || i !== -1;) {
    let s;
    if (
      (n !== -1 && (i === -1 || n < i)
        ? i === n + 1
          ? ((s = i + 1),
            (n = e.indexOf("\r", s)),
            (i = e.indexOf(
              `
`,
              s,
            )))
          : ((s = n + 1), (n = e.indexOf("\r", s)))
        : ((s = i + 1),
          (i = e.indexOf(
            `
`,
            s,
          ))),
      t.push(s),
      ++o === jo)
    ) {
      if (s - r <= Ko) {
        for (let a = s; a < e.length; a++) {
          const l = e.charCodeAt(a);
          (l === Wn || l === qn) &&
            (l === qn && a + 1 < e.length && e.charCodeAt(a + 1) === Wn && a++,
            t.push(a + 1));
        }
        return t;
      }
      ((r = s), (o = 0));
    }
  }
  return t;
}
function tt(e) {
  const t = Xo(e);
  return Array.from({ length: t.length }, (n, i) => {
    const r = t[i],
      o = t[i + 1] ?? e.length;
    return e.slice(r, o);
  });
}
function je(e, t) {
  for (const n of t) e.push(n);
}
const Yo = { forcePlainText: !1 };
function Qo(
  e,
  t,
  { theme: n, tokenizeMaxLineLength: i, useTokenTransformer: r },
  { forcePlainText: o, startingLine: s, totalLines: a, lines: l } = Yo,
) {
  o ? ((s ??= 0), (a ??= 1 / 0)) : ((s = 0), (a = 1 / 0));
  const d = s > 0 || a < 1 / 0,
    { state: h, transformers: c } = Ji(r),
    u = o ? "text" : (e.lang ?? j(e.name)),
    f = typeof n == "string" ? t.getTheme(n).type : void 0,
    p = kn({ theme: n, highlighter: t });
  h.lineInfo = (S) => ({
    type: "context",
    lineIndex: S - 1 + s,
    lineNumber: S + s,
  });
  const g =
      typeof n == "string"
        ? {
            lang: u,
            theme: n,
            transformers: c,
            defaultColor: !1,
            cssVariablePrefix: q("token"),
            tokenizeMaxLineLength: i,
            tokenizeTimeLimit: 0,
          }
        : {
            lang: u,
            themes: n,
            transformers: c,
            defaultColor: !1,
            cssVariablePrefix: q("token"),
            tokenizeMaxLineLength: i,
            tokenizeTimeLimit: 0,
          },
    m = sn(t.codeToHast(Zo(d ? Jo(l ?? tt(e.contents), s, a) : e.contents), g)),
    y = d ? new Array(s) : m;
  return (d && je(y, m), { code: y, themeStyles: p, baseThemeType: f });
}
function Zo(e) {
  return e.replace(
    /\r(?!\n)/g,
    `
`,
  );
}
function Jo(e, t, n) {
  if (e.length === 0) return "";
  const i = Math.min(t + n, e.length);
  return e.slice(t, i).join("");
}
function $(e, t) {
  return e?.cacheKey != null || t?.cacheKey != null
    ? e?.cacheKey === t?.cacheKey
    : e?.contents === t?.contents && e?.name === t?.name && e?.lang === t?.lang;
}
const er = "-1,-1";
function an(e) {
  return e?.some((t) => t.lineNumber === 0) ?? !1;
}
function ln(e) {
  const t = e[0];
  return t != null && t.length > 0 ? t : void 0;
}
function $t(e) {
  return e.startingLine === 0 && e.totalLines > 0;
}
function tr(e, t) {
  return e.endsWith(`\r
`)
    ? t +
        `\r
`
    : e.endsWith("\r")
      ? t + "\r"
      : e.endsWith(`
`)
        ? t +
          `
`
        : t;
}
function nr(e, t) {
  return R({
    tagName: "div",
    children: e,
    properties: { "data-content": "", style: `grid-row: span ${t}` },
  });
}
function Ht(e) {
  return (e.lang ?? j(e.name)) === "text";
}
function gt(e, t) {
  return t.cacheKey == null
    ? e.file === t && e.sourceContents === t.contents
    : e.cacheKey === t.cacheKey;
}
let es = -1;
var ts = class {
  options;
  annotationSlotName;
  onRenderUpdate;
  workerManager;
  __id = `file-renderer:${++es}`;
  highlighter;
  file;
  renderCache;
  pendingHighlightResult;
  computedLang = "text";
  lineAnnotations = {};
  lineCache;
  pendingStructuralRows;
  textDocumentCache = new WeakMap();
  editSessionActive = !1;
  get fileCache() {
    return this.renderCache?.file;
  }
  constructor(e = { theme: U }, t = fe, n, i) {
    ((this.options = e),
      (this.annotationSlotName = t),
      (this.onRenderUpdate = n),
      (this.workerManager = i),
      i?.isWorkingPool() !== !0 &&
        (this.highlighter = ce(e.theme ?? U) ? Ge() : void 0));
  }
  setOptions(e) {
    this.options = e;
  }
  mergeOptions(e) {
    this.options = { ...this.options, ...e };
  }
  setLineAnnotations(e) {
    this.lineAnnotations = {};
    for (const t of e) {
      const n = this.lineAnnotations[t.lineNumber] ?? [];
      ((this.lineAnnotations[t.lineNumber] = n), n.push(t));
    }
  }
  cleanUp() {
    (this.recycle(),
      (this.workerManager = void 0),
      (this.onRenderUpdate = void 0));
  }
  beginEditSession(e, t) {
    const { editSessionActive: n, renderCache: i } = this;
    if (
      ((this.editSessionActive = !0),
      n || (this.pendingHighlightResult = void 0),
      (this.file = e),
      i == null || (n && i.file === e))
    )
      return;
    const { options: r } = this.getRenderOptions(e),
      o = i.file === e,
      s = t != null && $(i.file, t),
      { result: a } = i;
    if (!i.highlighted || a == null || !Re(i.options, r) || (!o && !s)) {
      (this.clearRenderCache(),
        (this.lineCache = void 0),
        (this.textDocumentCache = new WeakMap()));
      return;
    }
    if (o) return;
    this.renderCache = { ...i, file: e, result: { ...a, code: [...a.code] } };
    const { lineCache: l } = this;
    l != null && t != null && gt(l, t)
      ? (this.lineCache = {
          cacheKey: void 0,
          file: e,
          sourceContents: e.contents,
          lines: l.lines,
        })
      : (this.lineCache = void 0);
  }
  endEditSession(e) {
    ((this.editSessionActive = !1), (this.pendingHighlightResult = void 0));
    const { renderCache: t } = this;
    e == null || t == null || t.file === e || !$(t.file, e) || (t.file = e);
  }
  editorRenderReady() {
    return (
      this.renderCache?.options.useTokenTransformer === !0 &&
      this.renderCache.highlighted &&
      this.renderCache.result != null
    );
  }
  recycle() {
    (this.clearRenderCache(),
      (this.highlighter = void 0),
      this.workerManager?.cleanUpTasks(this),
      (this.lineCache = void 0),
      (this.file = void 0),
      this.endEditSession(),
      (this.textDocumentCache = new WeakMap()));
  }
  clearRenderCache() {
    ((this.pendingStructuralRows = void 0),
      (this.renderCache = void 0),
      (this.pendingHighlightResult = void 0));
  }
  hydrate(e) {
    this.file = e;
    const { options: t } = this.getRenderOptions(e),
      n = mt(this.getOrCreateLineCache(e).length, this.getTokenizeMaxLength());
    let i = this.workerManager?.getFileResultCache(e);
    (i != null && !Re(t, i.options) && (i = void 0),
      (this.renderCache ??= {
        file: e,
        hydrated: !0,
        options: t,
        highlighted: !n && !Ht(e),
        result: n ? void 0 : i?.result,
        renderRange: void 0,
      }),
      !this.editSessionActive && this.workerManager?.isWorkingPool() === !0
        ? this.renderCache.result == null &&
          !n &&
          this.workerManager.highlightFileAST(this, e)
        : this.highlighter == null &&
          ((this.computedLang = e.lang ?? j(e.name)),
          this.initializeHighlighter()));
  }
  getLocalHighlightTheme() {
    return (
      this.workerManager?.getFileRenderOptions().theme ??
      this.options.theme ??
      U
    );
  }
  getEffectiveCodeOptions() {
    const e =
      this.workerManager?.isWorkingPool() === !0
        ? this.workerManager.getFileRenderOptions()
        : void 0;
    return {
      theme: this.getLocalHighlightTheme(),
      tokenizeMaxLineLength:
        e?.tokenizeMaxLineLength ?? this.options.tokenizeMaxLineLength,
    };
  }
  getRenderOptions(e) {
    const t = (() => {
        if (this.workerManager?.isWorkingPool() === !0) {
          const r = this.workerManager.getFileRenderOptions();
          return this.editSessionActive && r.useTokenTransformer !== !0
            ? { ...r, useTokenTransformer: !0 }
            : r;
        }
        const { tokenizeMaxLineLength: i = 1e3 } = this.options;
        return {
          theme: this.getLocalHighlightTheme(),
          useTokenTransformer:
            this.editSessionActive || this.options.useTokenTransformer === !0,
          tokenizeMaxLineLength: i,
        };
      })(),
      { renderCache: n } = this;
    return n?.result == null
      ? { options: t, forceHighlight: !0 }
      : !$(e, n.file) || !Re(t, n.options)
        ? { options: t, forceHighlight: !0 }
        : { options: t, forceHighlight: !1 };
  }
  getFileForNextRender(e) {
    const { options: t } = this.getRenderOptions(e);
    if (this.getReadyRenderResult(e, t) != null) return e;
    const { renderCache: n } = this;
    if (n == null) return e;
    if ($(n.file, e)) return n.file;
    const i = tt(e.contents),
      r =
        e.contents.length === 0 ||
        Ht(e) ||
        mt(i.length, this.getTokenizeMaxLength());
    return this.canRenderFile(e, t, r) ? e : n.file;
  }
  canRenderFile(e, t, n) {
    const { renderCache: i } = this;
    return i == null || $(i.file, e)
      ? !0
      : n
        ? (i.result == null && i.hydrated !== !0) ||
          this.workerManager?.isWorkingPool() === !0 ||
          (this.highlighter != null && ce(t.theme))
        : i.result == null && i.hydrated !== !0
          ? !0
          : !this.editSessionActive &&
              this.workerManager?.isWorkingPool() === !0
            ? !i.highlighted
            : this.highlighter != null && ce(t.theme);
  }
  getOrCreateLineCache(e) {
    let { lineCache: t } = this;
    return (
      (t == null || !gt(t, e)) &&
        (t = {
          cacheKey: e.cacheKey,
          file: e,
          sourceContents: e.contents,
          lines: tt(e.contents),
        }),
      (this.lineCache = t),
      t.lines
    );
  }
  getLineCount(e) {
    const t = this.getOrCreateLineCache(e);
    return this.textDocumentCache.get(e)?.lineCount ?? t.length;
  }
  updateRenderCache(e, t, n = !1) {
    this.pendingStructuralRows = void 0;
    const { renderCache: i } = this;
    if (i == null) return;
    const { file: r, result: o } = i;
    if (o == null) return;
    const s = n ? new Map() : void 0;
    this.pendingStructuralRows = s;
    const a =
      this.lineCache != null && gt(this.lineCache, r) ? this.lineCache : void 0;
    for (const [l, d] of e) {
      if (s == null && a != null && l < a.lines.length) {
        const c = d.map((u) => u[2]).join("");
        a.lines[l] = tr(a.lines[l] ?? "", c);
      }
      const h = {
        type: "element",
        tagName: "div",
        properties: {
          "data-line": l + 1,
          "data-line-type": "context",
          "data-line-index": l,
        },
        children: d.map(([c, u, f]) =>
          c === 0 && u === ""
            ? f === ""
              ? { type: "element", tagName: "br", properties: {}, children: [] }
              : { type: "text", value: f }
            : {
                type: "element",
                tagName: "span",
                properties: { "data-char": c, style: `color:${u};` },
                children: [{ type: "text", value: f }],
              },
        ),
      };
      s != null ? s.set(l, h) : (o.code[l] = h);
    }
    ((o.baseThemeType = t),
      (i.isDirty = !0),
      s == null &&
        a != null &&
        ((r.contents = a.lines.join("")), (a.sourceContents = r.contents)));
  }
  applyDocumentChange(e) {
    const { pendingStructuralRows: t, renderCache: n } = this;
    if (((this.pendingStructuralRows = void 0), n == null)) return;
    const { file: i, result: r } = n;
    if (r == null) return;
    const o =
        this.lineCache != null && gt(this.lineCache, i)
          ? this.lineCache.lines
          : tt(i.contents),
      s = tt(e.getText());
    if (o.length !== s.length) {
      const a = Math.min(o.length, s.length);
      let l = 0;
      for (; l < a && o[l] === s[l];) l++;
      let d = 0;
      for (; d < a - l && o[o.length - 1 - d] === s[s.length - 1 - d];) d++;
      const h = r.code;
      r.code = new Array(s.length);
      for (let c = 0; c < l; c++) r.code[c] = h[c];
      for (let c = 0; c < d; c++)
        r.code[s.length - 1 - c] = h[o.length - 1 - c];
      if (t !== void 0) for (const [c, u] of t) c < s.length && (r.code[c] = u);
      for (let c = l; c < s.length - d; c++)
        r.code[c] ??= {
          type: "element",
          tagName: "div",
          properties: {
            "data-line": c + 1,
            "data-line-type": "context",
            "data-line-index": c,
          },
          children: [
            {
              type: "element",
              tagName: "span",
              properties: { "data-char": 0 },
              children: [{ type: "text", value: e.getLineText(c) }],
            },
          ],
        };
      for (let c = 0; c < r.code.length; c++) {
        const u = r.code[c];
        u?.type === "element" &&
          ((u.properties["data-line"] = c + 1),
          (u.properties["data-line-index"] = c));
      }
      n.isDirty = !0;
    }
    ((this.lineCache = {
      cacheKey: i.cacheKey,
      file: i,
      sourceContents: i.contents,
      lines: s,
    }),
      this.textDocumentCache.set(i, e),
      (i.contents = e.getText()));
  }
  renderFile(e = this.file, t = We) {
    if (((this.file = e), e == null)) {
      this.pendingHighlightResult = void 0;
      return;
    }
    let { options: n, forceHighlight: i } = this.getRenderOptions(e);
    const r = this.getReadyRenderResult(e, n);
    ((this.pendingHighlightResult = void 0),
      r != null &&
        ((this.renderCache = { ...r, file: e, renderRange: void 0 }), (i = !1)),
      (this.renderCache ??= {
        file: e,
        highlighted: !1,
        options: n,
        result: void 0,
        renderRange: void 0,
      }));
    const o = this.getOrCreateLineCache(e),
      s = e.contents.length > 0,
      a = !s || Ht(e) || mt(o.length, this.getTokenizeMaxLength()),
      l = this.canRenderFile(e, n, a),
      d = !$(e, this.renderCache.file),
      h = !Vt(this.renderCache.renderRange, t);
    if (!this.editSessionActive && this.workerManager?.isWorkingPool() === !0) {
      const c =
        this.renderCache.result == null &&
        this.renderCache.highlighted &&
        !a &&
        !d &&
        Nt(t);
      (l &&
        !c &&
        (a ||
          this.renderCache.result == null ||
          (!this.renderCache.highlighted && (d || h))) &&
        ((this.renderCache.file = e),
        (this.renderCache.options = n),
        (this.renderCache.highlighted = !1),
        (this.renderCache.result == null || d || h || i) &&
          (this.renderCache.result = this.workerManager.getPlainFileAST(
            e,
            t.startingLine,
            t.totalLines,
            o,
          )),
        (this.renderCache.renderRange = t)),
        !a &&
          s &&
          (!this.renderCache.highlighted || i) &&
          this.workerManager.highlightFileAST(this, e));
    } else {
      ((this.computedLang = e.lang ?? j(e.name)), (this.highlighter ??= Ge()));
      const c = this.highlighter != null && ce(n.theme),
        u = this.highlighter != null && Ft(this.computedLang),
        f = !a && u;
      if (
        l &&
        this.highlighter != null &&
        c &&
        (i ||
          a ||
          (!this.renderCache.highlighted && f) ||
          this.renderCache.result == null)
      ) {
        const { result: p, options: g } = this.renderFileWithHighlighter(
          e,
          this.highlighter,
          a || !u,
        );
        this.renderCache = {
          file: e,
          options: g,
          highlighted: f,
          result: p,
          renderRange: void 0,
        };
      }
      (!c || (!a && !u)) &&
        this.asyncHighlight(e).then(({ result: p, options: g }) => {
          this.applyHighlightResult(e, p, g, !a);
        });
    }
    return this.renderCache.result != null
      ? this.processFileResult(
          this.renderCache.file,
          t,
          this.renderCache.result,
        )
      : void 0;
  }
  async asyncRender(e, t = We) {
    this.file = e;
    const { result: n } = await this.asyncHighlight(e);
    return this.processFileResult(e, t, n);
  }
  async asyncHighlight(e) {
    const t = mt(
      this.getOrCreateLineCache(e).length,
      this.getTokenizeMaxLength(),
    );
    this.computedLang = t ? "text" : (e.lang ?? j(e.name));
    const n = this.highlighter != null && Io(ht(this.getLocalHighlightTheme())),
      i = t || (this.highlighter != null && Ft(this.computedLang));
    return (
      (this.highlighter == null || !n || !i) &&
        (this.highlighter = await this.initializeHighlighter()),
      this.renderFileWithHighlighter(e, this.highlighter, t)
    );
  }
  renderFileWithHighlighter(e, t, n = !1) {
    const { options: i } = this.getRenderOptions(e);
    return { result: Qo(e, t, i, { forcePlainText: n }), options: i };
  }
  processFileResult(e, t, { code: n, themeStyles: i, baseThemeType: r }) {
    const o = this.getLineCount(e),
      { disableFileHeader: s = !1 } = this.options,
      a = [],
      l = $e(),
      d = Math.min(t.startingLine + t.totalLines, o);
    let h = 0;
    const c = $t(t) ? ln(this.lineAnnotations) : void 0;
    c != null &&
      (l.children.push(Z("context", "annotation", 1)),
      a.push(
        on({
          hunkIndex: -1,
          lineIndex: -1,
          annotations: c.map((u) => this.annotationSlotName(u)),
        }),
      ),
      h++);
    for (let u = t.startingLine; u < d; u++) {
      const f = u + 1,
        p = n[u];
      if (p == null) {
        const m = "FileRenderer.processFileResult: Line doesnt exist";
        throw (
          console.error(m, { name: e.name, lineIndex: u, lineNumber: f }),
          new Error(m)
        );
      }
      (l.children.push(Wi("context", f, `${u}`)), a.push(p), h++);
      const g = this.lineAnnotations[f];
      g != null &&
        (l.children.push(Z("context", "annotation", 1)),
        a.push(
          on({
            hunkIndex: 0,
            lineIndex: f,
            annotations: g.map((m) => this.annotationSlotName(m)),
          }),
        ),
        h++);
    }
    return (
      (l.properties.style = `grid-row: span ${h}`),
      {
        file: e,
        gutterAST: l.children ?? [],
        contentAST: a,
        preAST: this.createPreElement(o),
        headerAST: s ? void 0 : this.renderHeader(e),
        totalLines: o,
        rowCount: h,
        themeStyles: i,
        baseThemeType: r,
        bufferBefore: t.bufferBefore,
        bufferAfter: t.bufferAfter,
        css: "",
      }
    );
  }
  renderHeader(e) {
    const { headerRenderMode: t = "default", stickyHeader: n = !1 } =
      this.options;
    return Yi({ fileOrDiff: e, mode: t, stickyHeader: n });
  }
  renderFullHTML(e) {
    return J(this.renderFullAST(e));
  }
  renderFullAST(e, t = []) {
    return (
      t.push(
        R({
          tagName: "code",
          children: this.renderCodeAST(e),
          properties: { "data-code": "" },
        }),
      ),
      { ...e.preAST, children: t }
    );
  }
  renderCodeAST(e) {
    const t = $e();
    return (
      (t.children = e.gutterAST),
      (t.properties.style = `grid-row: span ${e.rowCount}`),
      [t, nr(e.contentAST, e.rowCount)]
    );
  }
  renderPartialHTML(e, t = !1) {
    return t
      ? J(R({ tagName: "code", children: e, properties: { "data-code": "" } }))
      : J(e);
  }
  async initializeHighlighter() {
    return (
      (this.highlighter = await ct(
        xn(this.computedLang, {
          theme: this.getLocalHighlightTheme(),
          preferredHighlighter:
            this.workerManager?.getPreferredHighlighter() ??
            this.options.preferredHighlighter,
        }),
      )),
      this.highlighter
    );
  }
  onHighlightSuccess(e, t, n, i = !0) {
    this.editSessionActive || this.applyHighlightResult(e, t, n, i);
  }
  applyHighlightResult(e, t, n, i = !0) {
    const { file: r, renderCache: o } = this;
    r == null ||
      o == null ||
      !$(e, r) ||
      !Re(n, this.getRenderOptions(r).options) ||
      ((o.result == null ||
        !o.highlighted ||
        !Re(o.options, n) ||
        !$(o.file, r)) &&
        ((this.pendingHighlightResult = {
          file: r,
          options: n,
          highlighted: i,
          result: t,
        }),
        this.onRenderUpdate?.()));
  }
  getMatchingWorkerResultCache(e, t) {
    if (this.editSessionActive) return;
    const n = this.workerManager?.getFileResultCache(e);
    if (!(n == null || !Re(t, n.options))) return n;
  }
  getReadyRenderResult(e, t) {
    const { pendingHighlightResult: n } = this;
    if (n != null && $(n.file, e) && Re(n.options, t)) return n;
    const i = this.getMatchingWorkerResultCache(e, t);
    if (!(i == null || this.hasHighlightedRenderCache(e, t)))
      return { file: e, highlighted: !0, ...i };
  }
  hasHighlightedRenderCache(e, t) {
    const { renderCache: n } = this;
    return (
      n?.result != null && n.highlighted && $(e, n.file) && Re(t, n.options)
    );
  }
  onHighlightError(e) {
    console.error(e);
  }
  getTokenizeMaxLength() {
    return this.options.tokenizeMaxLength ?? 1e5;
  }
  createPreElement(e) {
    const { disableLineNumbers: t = !1, overflow: n = "scroll" } = this.options;
    return Qi({
      type: "file",
      diffIndicators: "none",
      disableBackground: !0,
      disableLineNumbers: t,
      overflow: n,
      split: !1,
      totalLines: e,
    });
  }
};
function mt(e, t) {
  return e > t;
}
const ir = `<svg data-icon-sprite aria-hidden="true" width="0" height="0">
  <symbol id="diffs-icon-arrow-right-short" viewBox="0 0 16 16">
    <path d="M8.47 4.22a.75.75 0 0 0 0 1.06l1.97 1.97H3.75a.75.75 0 0 0 0 1.5h6.69l-1.97 1.97a.75.75 0 1 0 1.06 1.06l3.25-3.25a.75.75 0 0 0 0-1.06L9.53 4.22a.75.75 0 0 0-1.06 0"/>
  </symbol>
  <symbol id="diffs-icon-brand-github" viewBox="0 0 16 16">
    <path d="M8 0c4.42 0 8 3.58 8 8a8.01 8.01 0 0 1-5.45 7.59c-.4.08-.55-.17-.55-.38 0-.27.01-1.13.01-2.2 0-.75-.25-1.23-.54-1.48 1.78-.2 3.65-.88 3.65-3.95 0-.88-.31-1.59-.82-2.15.08-.2.36-1.02-.08-2.12 0 0-.67-.22-2.2.82-.64-.18-1.32-.27-2-.27s-1.36.09-2 .27c-1.53-1.03-2.2-.82-2.2-.82-.44 1.1-.16 1.92-.08 2.12-.51.56-.82 1.28-.82 2.15 0 3.06 1.86 3.75 3.64 3.95-.23.2-.44.55-.51 1.07-.46.21-1.61.55-2.33-.66-.15-.24-.6-.83-1.23-.82-.67.01-.27.38.01.53.34.19.73.9.82 1.13.16.45.68 1.31 2.69.94 0 .67.01 1.3.01 1.49 0 .21-.15.45-.55.38A7.995 7.995 0 0 1 0 8c0-4.42 3.58-8 8-8"/>
  </symbol>
  <symbol id="diffs-icon-chevron" viewBox="0 0 16 16">
    <path d="M1.47 4.47a.75.75 0 0 1 1.06 0L8 9.94l5.47-5.47a.75.75 0 1 1 1.06 1.06l-6 6a.75.75 0 0 1-1.06 0l-6-6a.75.75 0 0 1 0-1.06"/>
  </symbol>
  <symbol id="diffs-icon-chevrons-narrow" viewBox="0 0 10 16">
    <path d="M4.47 2.22a.75.75 0 0 1 1.06 0l3.25 3.25a.75.75 0 0 1-1.06 1.06L5 3.81 2.28 6.53a.75.75 0 0 1-1.06-1.06zM1.22 9.47a.75.75 0 0 1 1.06 0L5 12.19l2.72-2.72a.75.75 0 0 1 1.06 1.06l-3.25 3.25a.75.75 0 0 1-1.06 0l-3.25-3.25a.75.75 0 0 1 0-1.06"/>
  </symbol>
  <symbol id="diffs-icon-diff-split" viewBox="0 0 16 16">
    <path d="M14 0H8.5v16H14a2 2 0 0 0 2-2V2a2 2 0 0 0-2-2m-1.5 6.5v1h1a.5.5 0 0 1 0 1h-1v1a.5.5 0 0 1-1 0v-1h-1a.5.5 0 0 1 0-1h1v-1a.5.5 0 0 1 1 0"/><path d="M2 0a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h5.5V0zm.5 7.5h3a.5.5 0 0 1 0 1h-3a.5.5 0 0 1 0-1" opacity=".3"/>
  </symbol>
  <symbol id="diffs-icon-diff-unified" viewBox="0 0 16 16">
    <path fill-rule="evenodd" d="M16 14a2 2 0 0 1-2 2H2a2 2 0 0 1-2-2V8.5h16zm-8-4a.5.5 0 0 0-.5.5v1h-1a.5.5 0 0 0 0 1h1v1a.5.5 0 0 0 1 0v-1h1a.5.5 0 0 0 0-1h-1v-1A.5.5 0 0 0 8 10" clip-rule="evenodd"/><path fill-rule="evenodd" d="M14 0a2 2 0 0 1 2 2v5.5H0V2a2 2 0 0 1 2-2zM6.5 3.5a.5.5 0 0 0 0 1h3a.5.5 0 0 0 0-1z" clip-rule="evenodd" opacity=".4"/>
  </symbol>
  <symbol id="diffs-icon-expand" viewBox="0 0 16 16">
    <path d="M3.47 5.47a.75.75 0 0 1 1.06 0L8 8.94l3.47-3.47a.75.75 0 1 1 1.06 1.06l-4 4a.75.75 0 0 1-1.06 0l-4-4a.75.75 0 0 1 0-1.06"/>
  </symbol>
  <symbol id="diffs-icon-expand-all" viewBox="0 0 16 16">
    <path d="M11.47 9.47a.75.75 0 1 1 1.06 1.06l-4 4a.75.75 0 0 1-1.06 0l-4-4a.75.75 0 1 1 1.06-1.06L8 12.94zM7.526 1.418a.75.75 0 0 1 1.004.052l4 4a.75.75 0 1 1-1.06 1.06L8 3.06 4.53 6.53a.75.75 0 1 1-1.06-1.06l4-4z"/>
  </symbol>
  <symbol id="diffs-icon-file-code" viewBox="0 0 16 16">
    <path d="M10.75 0c.199 0 .39.08.53.22l3.5 3.5c.14.14.22.331.22.53v9A2.75 2.75 0 0 1 12.25 16h-8.5A2.75 2.75 0 0 1 1 13.25V2.75A2.75 2.75 0 0 1 3.75 0zm-7 1.5c-.69 0-1.25.56-1.25 1.25v10.5c0 .69.56 1.25 1.25 1.25h8.5c.69 0 1.25-.56 1.25-1.25V5h-1.25A2.25 2.25 0 0 1 10 2.75V1.5z"/><path d="M7.248 6.19a.75.75 0 0 1 .063 1.058L5.753 9l1.558 1.752a.75.75 0 0 1-1.122.996l-2-2.25a.75.75 0 0 1 0-.996l2-2.25a.75.75 0 0 1 1.06-.063M8.69 7.248a.75.75 0 1 1 1.12-.996l2 2.25a.75.75 0 0 1 0 .996l-2 2.25a.75.75 0 1 1-1.12-.996L10.245 9z"/>
  </symbol>
  <symbol id="diffs-icon-plus" viewBox="0 0 16 16">
    <path d="M8 3a.75.75 0 0 1 .75.75v3.5h3.5a.75.75 0 0 1 0 1.5h-3.5v3.5a.75.75 0 0 1-1.5 0v-3.5h-3.5a.75.75 0 0 1 0-1.5h3.5v-3.5A.75.75 0 0 1 8 3"/>
  </symbol>
  <symbol id="diffs-icon-symbol-added" viewBox="0 0 16 16">
    <path d="M8 4a.75.75 0 0 1 .75.75v2.5h2.5a.75.75 0 0 1 0 1.5h-2.5v2.5a.75.75 0 0 1-1.5 0v-2.5h-2.5a.75.75 0 0 1 0-1.5h2.5v-2.5A.75.75 0 0 1 8 4"/><path d="M1.788 4.296c.196-.88.478-1.381.802-1.706s.826-.606 1.706-.802C5.194 1.588 6.387 1.5 8 1.5s2.806.088 3.704.288c.88.196 1.381.478 1.706.802s.607.826.802 1.706c.2.898.288 2.091.288 3.704s-.088 2.806-.288 3.704c-.195.88-.478 1.381-.802 1.706s-.826.607-1.706.802c-.898.2-2.091.288-3.704.288s-2.806-.088-3.704-.288c-.88-.195-1.381-.478-1.706-.802s-.606-.826-.802-1.706C1.588 10.806 1.5 9.613 1.5 8s.088-2.806.288-3.704M8 0C1.412 0 0 1.412 0 8s1.412 8 8 8 8-1.412 8-8-1.412-8-8-8"/>
  </symbol>
  <symbol id="diffs-icon-symbol-deleted" viewBox="0 0 16 16">
    <path d="M4 8a.75.75 0 0 1 .75-.75h6.5a.75.75 0 0 1 0 1.5h-6.5A.75.75 0 0 1 4 8"/><path d="M1.788 4.296c.196-.88.478-1.381.802-1.706s.826-.606 1.706-.802C5.194 1.588 6.387 1.5 8 1.5s2.806.088 3.704.288c.88.196 1.381.478 1.706.802s.607.826.802 1.706c.2.898.288 2.091.288 3.704s-.088 2.806-.288 3.704c-.195.88-.478 1.381-.802 1.706s-.826.607-1.706.802c-.898.2-2.091.288-3.704.288s-2.806-.088-3.704-.288c-.88-.195-1.381-.478-1.706-.802s-.606-.826-.802-1.706C1.588 10.806 1.5 9.613 1.5 8s.088-2.806.288-3.704M8 0C1.412 0 0 1.412 0 8s1.412 8 8 8 8-1.412 8-8-1.412-8-8-8"/>
  </symbol>
  <symbol id="diffs-icon-symbol-diffstat" viewBox="0 0 16 16">
    <path d="M1.788 4.296c.196-.88.478-1.381.802-1.706s.826-.606 1.706-.802C5.194 1.588 6.387 1.5 8 1.5s2.806.088 3.704.288c.88.196 1.381.478 1.706.802s.607.826.802 1.706c.2.898.288 2.091.288 3.704s-.088 2.806-.288 3.704c-.195.88-.478 1.381-.802 1.706s-.826.607-1.706.802c-.898.2-2.091.288-3.704.288s-2.806-.088-3.704-.288c-.88-.195-1.381-.478-1.706-.802s-.606-.826-.802-1.706C1.588 10.806 1.5 9.613 1.5 8s.088-2.806.288-3.704M8 0C1.412 0 0 1.412 0 8s1.412 8 8 8 8-1.412 8-8-1.412-8-8-8"/><path d="M8.75 4.296a.75.75 0 0 0-1.5 0V6.25h-2a.75.75 0 0 0 0 1.5h2v1.5h1.5v-1.5h2a.75.75 0 0 0 0-1.5h-2zM5.25 10a.75.75 0 0 0 0 1.5h5.5a.75.75 0 0 0 0-1.5z"/>
  </symbol>
  <symbol id="diffs-icon-symbol-ignored" viewBox="0 0 16 16">
    <path d="M1.5 8c0 1.613.088 2.806.288 3.704.196.88.478 1.381.802 1.706s.826.607 1.706.802c.898.2 2.091.288 3.704.288s2.806-.088 3.704-.288c.88-.195 1.381-.478 1.706-.802s.607-.826.802-1.706c.2-.898.288-2.091.288-3.704s-.088-2.806-.288-3.704c-.195-.88-.478-1.381-.802-1.706s-.826-.606-1.706-.802C10.806 1.588 9.613 1.5 8 1.5s-2.806.088-3.704.288c-.88.196-1.381.478-1.706.802s-.606.826-.802 1.706C1.588 5.194 1.5 6.387 1.5 8M0 8c0-6.588 1.412-8 8-8s8 1.412 8 8-1.412 8-8 8-8-1.412-8-8m11.53-2.47a.75.75 0 0 0-1.06-1.06l-6 6a.75.75 0 1 0 1.06 1.06z"/>
  </symbol>
  <symbol id="diffs-icon-symbol-modified" viewBox="0 0 16 16">
    <path d="M1.5 8c0 1.613.088 2.806.288 3.704.196.88.478 1.381.802 1.706s.826.607 1.706.802c.898.2 2.091.288 3.704.288s2.806-.088 3.704-.288c.88-.195 1.381-.478 1.706-.802s.607-.826.802-1.706c.2-.898.288-2.091.288-3.704s-.088-2.806-.288-3.704c-.195-.88-.478-1.381-.802-1.706s-.826-.606-1.706-.802C10.806 1.588 9.613 1.5 8 1.5s-2.806.088-3.704.288c-.88.196-1.381.478-1.706.802s-.606.826-.802 1.706C1.588 5.194 1.5 6.387 1.5 8M0 8c0-6.588 1.412-8 8-8s8 1.412 8 8-1.412 8-8 8-8-1.412-8-8m8 3a3 3 0 1 0 0-6 3 3 0 0 0 0 6"/>
  </symbol>
  <symbol id="diffs-icon-symbol-moved" viewBox="0 0 16 16">
    <path d="M1.788 4.296c.196-.88.478-1.381.802-1.706s.826-.606 1.706-.802C5.194 1.588 6.387 1.5 8 1.5s2.806.088 3.704.288c.88.196 1.381.478 1.706.802s.607.826.802 1.706c.2.898.288 2.091.288 3.704s-.088 2.806-.288 3.704c-.195.88-.478 1.381-.802 1.706s-.826.607-1.706.802c-.898.2-2.091.288-3.704.288s-2.806-.088-3.704-.288c-.88-.195-1.381-.478-1.706-.802s-.606-.826-.802-1.706C1.588 10.806 1.5 9.613 1.5 8s.088-2.806.288-3.704M8 0C1.412 0 0 1.412 0 8s1.412 8 8 8 8-1.412 8-8-1.412-8-8-8"/><path d="M8.495 4.695a.75.75 0 0 0-.05 1.06L10.486 8l-2.041 2.246a.75.75 0 0 0 1.11 1.008l2.5-2.75a.75.75 0 0 0 0-1.008l-2.5-2.75a.75.75 0 0 0-1.06-.051m-4 0a.75.75 0 0 0-.05 1.06l2.044 2.248-1.796 1.995a.75.75 0 0 0 1.114 1.004l2.25-2.5a.75.75 0 0 0-.002-1.007l-2.5-2.75a.75.75 0 0 0-1.06-.05"/>
  </symbol>
  <symbol id="diffs-icon-symbol-ref" viewBox="0 0 16 16">
    <path d="M1.5 8c0 1.613.088 2.806.288 3.704.196.88.478 1.381.802 1.706.286.286.71.54 1.41.73V1.86c-.7.19-1.124.444-1.41.73-.324.325-.606.826-.802 1.706C1.588 5.194 1.5 6.387 1.5 8m4 6.397c.697.07 1.522.103 2.5.103 1.613 0 2.806-.088 3.704-.288.88-.195 1.381-.478 1.706-.802s.607-.826.802-1.706c.2-.898.288-2.091.288-3.704s-.088-2.806-.288-3.704c-.195-.88-.478-1.381-.802-1.706s-.826-.606-1.706-.802C10.806 1.588 9.613 1.5 8 1.5c-.978 0-1.803.033-2.5.103zM0 8c0-6.588 1.412-8 8-8s8 1.412 8 8-1.412 8-8 8-8-1.412-8-8m7-2a1 1 0 0 1 1-1h3a1 1 0 0 1 1 1v1a1 1 0 0 1-1 1H8a1 1 0 0 1-1-1z"/>
  </symbol>
</svg>`;
function ns(e, t) {
  return e.lineNumber === t.lineNumber && e.metadata === t.metadata;
}
function rr(e, t) {
  return e == null || t == null
    ? e === t
    : is(e.customProperties, t.customProperties) &&
        e.type === t.type &&
        e.diffIndicators === t.diffIndicators &&
        e.disableBackground === t.disableBackground &&
        e.disableLineNumbers === t.disableLineNumbers &&
        e.overflow === t.overflow &&
        e.split === t.split &&
        e.totalLines === t.totalLines;
}
const jn = {};
function is(e = jn, t = jn) {
  if (e === t) return !0;
  const n = Object.keys(e),
    i = Object.keys(t);
  if (n.length !== i.length) return !1;
  for (const r of n) if (e[r] !== t[r]) return !1;
  return !0;
}
function wn(e) {
  const t = document.createElement("div");
  return (
    (t.dataset.annotationSlot = ""),
    (t.slot = e),
    (t.style.whiteSpace = "normal"),
    t
  );
}
function or() {
  const e = document.createElement("div");
  return (
    (e.slot = "gutter-utility-slot"),
    (e.style.position = "absolute"),
    (e.style.top = "0"),
    (e.style.bottom = "0"),
    (e.style.textAlign = "center"),
    (e.style.whiteSpace = "normal"),
    (e.style.touchAction = "none"),
    e
  );
}
function sr() {
  const e = document.createElement("style");
  return (e.setAttribute(Oi, ""), e);
}
var ar = `@layer base {
  :host {
    --diffs-font-fallback: "SF Mono", Monaco, Consolas, "Ubuntu Mono", "Liberation Mono",
      "Courier New", monospace;
    --diffs-header-font-fallback: system-ui, -apple-system, "Segoe UI", Roboto, "Helvetica Neue",
      "Noto Sans", "Liberation Sans", Arial, sans-serif;
    --diffs-mixer: light-dark(#000, #fff);
    --diffs-gap-fallback: 8px;
    --diffs-scrollbar-gutter-fallback: 6px;
    --diffs-scrollbar-gutter: var(--diffs-scrollbar-gutter-override, var(--diffs-scrollbar-gutter-measured, var(--diffs-scrollbar-gutter-fallback)));
    --diffs-added-light: #0dbe4e;
    --diffs-added-dark: #5ecc71;
    --diffs-modified-light: #009fff;
    --diffs-modified-dark: #69b1ff;
    --diffs-deleted-light: #ff2e3f;
    --diffs-deleted-dark: #ff6762;
    --diffs-warning-light: #d5a910;
    --diffs-warning-dark: #ffd452;
    color-scheme: light dark;
    font-family: var(--diffs-header-font-family, var(--diffs-header-font-fallback));
    font-size: var(--diffs-font-size, 13px);
    line-height: var(--diffs-line-height, 20px);
    font-feature-settings: var(--diffs-font-features);
    --diffs-bg: light-dark(var(--diffs-light-bg, #fff), var(--diffs-dark-bg, #000));
    --diffs-bg-buffer: var(--diffs-bg-buffer-override, light-dark(color-mix(in lab, var(--diffs-bg) 92%, var(--diffs-mixer)), color-mix(in lab, var(--diffs-bg) 92%, var(--diffs-mixer))));
    --diffs-bg-context: var(--diffs-bg-context-override, light-dark(color-mix(in lab, var(--diffs-bg) 98.5%, var(--diffs-mixer)), color-mix(in lab, var(--diffs-bg) 92.5%, var(--diffs-mixer))));
    --diffs-bg-context-gutter: var(--diffs-bg-context-gutter-override, light-dark(color-mix(in lab, var(--diffs-bg-context) 90%, var(--diffs-bg)), color-mix(in lab, var(--diffs-bg-context) 45%, var(--diffs-bg))));
    --diffs-bg-separator: var(--diffs-bg-separator-override, light-dark(color-mix(in lab, var(--diffs-bg) 96%, var(--diffs-mixer)), color-mix(in lab, var(--diffs-bg) 85%, var(--diffs-mixer))));
    --diffs-fg: light-dark(var(--diffs-light, #000), var(--diffs-dark, #fff));
    --diffs-fg-number: var(--diffs-fg-number-override, light-dark(color-mix(in lab, var(--diffs-fg) 65%, var(--diffs-bg)), color-mix(in lab, var(--diffs-fg) 65%, var(--diffs-bg))));
    --diffs-fg-conflict-marker: var(--diffs-fg-conflict-marker-override, var(--diffs-fg-number));
    --diffs-deletion-base: var(--diffs-deletion-color-override, light-dark(var(--diffs-light-deletion-color, var(--diffs-deletion-color, var(--diffs-deleted-light))), var(--diffs-dark-deletion-color, var(--diffs-deletion-color, var(--diffs-deleted-dark)))));
    --diffs-addition-base: var(--diffs-addition-color-override, light-dark(var(--diffs-light-addition-color, var(--diffs-addition-color, var(--diffs-added-light))), var(--diffs-dark-addition-color, var(--diffs-addition-color, var(--diffs-added-dark)))));
    --diffs-modified-base: var(--diffs-modified-color-override, light-dark(var(--diffs-light-modified-color, var(--diffs-modified-color, var(--diffs-modified-light))), var(--diffs-dark-modified-color, var(--diffs-modified-color, var(--diffs-modified-dark)))));
    --diffs-bg-deletion: var(--diffs-bg-deletion-override, light-dark(color-mix(in lab, var(--diffs-bg) 88%, var(--diffs-deletion-base)), color-mix(in lab, var(--diffs-bg) 80%, var(--diffs-deletion-base))));
    --diffs-bg-deletion-emphasis: var(--diffs-bg-deletion-emphasis-override, light-dark(rgb(from var(--diffs-deletion-base) r g b / .15), rgb(from var(--diffs-deletion-base) r g b / .2)));
    --diffs-bg-addition: var(--diffs-bg-addition-override, light-dark(color-mix(in lab, var(--diffs-bg) 88%, var(--diffs-addition-base)), color-mix(in lab, var(--diffs-bg) 80%, var(--diffs-addition-base))));
    --diffs-bg-addition-emphasis: var(--diffs-bg-addition-emphasis-override, light-dark(rgb(from var(--diffs-addition-base) r g b / .15), rgb(from var(--diffs-addition-base) r g b / .2)));
    --diffs-selection-base: var(--diffs-modified-base);
    --diffs-selection-number-fg: light-dark(color-mix(in lab, var(--diffs-selection-base) 65%, var(--diffs-mixer)), color-mix(in lab, var(--diffs-selection-base) 75%, var(--diffs-mixer)));
    background-color: var(--diffs-bg);
    color: var(--diffs-fg);
    display: block;
  }

  pre, code, [data-error-wrapper] {
    isolation: isolate;
    font-family: var(--diffs-font-family, var(--diffs-font-fallback));
    outline: none;
    margin: 0;
    padding: 0;
    display: block;
  }

  pre, code {
    background-color: var(--diffs-bg);
  }

  code {
    contain: content;
  }

  input, button {
    font-family: inherit;
    font-size: inherit;
    line-height: inherit;
  }

  *, :before, :after {
    box-sizing: border-box;
  }

  [data-icon-sprite] {
    display: none;
  }

  [data-diffs-header], [data-separator] {
    font-family: var(--diffs-header-font-family, var(--diffs-header-font-fallback));
  }

  [data-diffs-header][data-sticky] {
    z-index: 1;
    background-color: var(--diffs-bg);
    position: sticky;
    top: 0;
  }

  [data-file-info] {
    color: var(--fg);
    background-color: color-mix(in lab, var(--bg) 98%, var(--fg));
    border-block: 1px solid color-mix(in lab, var(--bg) 95%, var(--fg));
    padding: 10px;
    font-weight: 700;
  }

  [data-diff], [data-file] {
    --diffs-grid-number-column-width: minmax(min-content, max-content);
    --diffs-code-grid: var(--diffs-grid-number-column-width) 1fr;

    &[data-dehydrated] {
      --diffs-code-grid: var(--diffs-grid-number-column-width) minmax(0, 1fr);
    }

    &:hover [data-code]::-webkit-scrollbar-thumb {
      background-color: var(--diffs-bg-context);
    }
  }

  @supports (-webkit-touch-callout: none) {
    :host {
      --diffs-scrollbar-gutter-fallback: 0px;
    }
  }

  [data-line] span, [data-edit-prediction-suffix] span {
    color: light-dark(var(--diffs-token-light, var(--diffs-light)), var(--diffs-token-dark, var(--diffs-dark)));
    background-color: light-dark(var(--diffs-token-light-bg, inherit), var(--diffs-token-dark-bg, inherit));
    font-weight: light-dark(var(--diffs-token-light-font-weight, inherit), var(--diffs-token-dark-font-weight, inherit));
    font-style: light-dark(var(--diffs-token-light-font-style, inherit), var(--diffs-token-dark-font-style, inherit));
    text-decoration: light-dark(var(--diffs-token-light-text-decoration, inherit), var(--diffs-token-dark-text-decoration, inherit));
  }

  [data-line], [data-gutter-buffer], [data-column-number], [data-line-annotation], [data-no-newline], [data-merge-conflict], [data-merge-conflict-actions], [data-editor-overlay] {
    --diffs-computed-decoration-bg: var(--diffs-bg);
    --diffs-computed-diff-line-bg: var(--diffs-computed-decoration-bg);
    --diffs-computed-selected-line-bg: var(--diffs-computed-diff-line-bg);
    --diffs-computed-editor-active-line-bg: var(--diffs-computed-selected-line-bg);
    --diffs-computed-hovered-line-bg: var(--diffs-computed-editor-active-line-bg);
    --diffs-hover-mix-target: var(--diffs-bg-hover-override, var(--diffs-mixer));
    --diffs-line-bg: var(--diffs-computed-hovered-line-bg);
    color: var(--diffs-fg);
    background-color: var(--diffs-line-bg, var(--diffs-bg));
  }

  [data-line], [data-no-newline] {
    &[data-decoration-bg] {
      --mix-deco-light: 92%;
      --mix-deco-dark: 85%;

      &[data-decoration-bg-depth="2"] {
        --mix-deco-light: 88%;
        --mix-deco-dark: 80%;
      }

      &[data-decoration-bg-depth="3"] {
        --mix-deco-light: 85%;
        --mix-deco-dark: 78%;
      }

      --diffs-hover-mix-target: var(--diffs-decoration-bg);
      --diffs-computed-decoration-bg: light-dark(color-mix(in lab,
          var(--diffs-bg) var(--mix-deco-light),
          var(--diffs-decoration-bg)), color-mix(in lab,
          var(--diffs-bg) var(--mix-deco-dark),
          var(--diffs-decoration-bg)));
    }
  }

  [data-line-annotation], [data-gutter-buffer="annotation"] {
    --diffs-annotation-bg: var(--diffs-bg-context);
    --diffs-computed-decoration-bg: var(--diffs-annotation-bg);
    --diffs-hover-mix-target: var(--diffs-computed-editor-active-line-bg);
  }

  [data-merge-conflict-actions], [data-gutter-buffer="merge-conflict-action"], [data-gutter-buffer="merge-conflict-marker-base"], [data-gutter-buffer="merge-conflict-marker-separator"], [data-merge-conflict="marker-base"], [data-merge-conflict="marker-separator"] {
    --diffs-computed-decoration-bg: var(--diffs-bg-context);
    --diffs-hover-mix-target: var(--diffs-computed-editor-active-line-bg);
  }

  [data-gutter-buffer="merge-conflict-marker-start"], [data-merge-conflict="marker-start"] {
    --diffs-computed-decoration-bg: light-dark(color-mix(in lab,
        var(--diffs-bg) 78%,
        var(--conflict-bg-current-header-override, var(--diffs-addition-base))), color-mix(in lab,
        var(--diffs-bg) 68%,
        var(--conflict-bg-current-header-override, var(--diffs-addition-base))));
    --diffs-hover-mix-target: var(--diffs-computed-editor-active-line-bg);
  }

  [data-gutter-buffer="merge-conflict-marker-end"], [data-merge-conflict="marker-end"] {
    --diffs-computed-decoration-bg: light-dark(color-mix(in lab,
        var(--diffs-bg) 78%,
        var(--conflict-bg-incoming-header-override, var(--diffs-modified-base))), color-mix(in lab,
        var(--diffs-bg) 68%,
        var(--conflict-bg-incoming-header-override, var(--diffs-modified-base))));
    --diffs-hover-mix-target: var(--diffs-computed-editor-active-line-bg);
  }

  [data-has-merge-conflict] [data-line-annotation], [data-has-merge-conflict] [data-gutter-buffer="annotation"] {
    --diffs-computed-decoration-bg: var(--diffs-bg);
    --diffs-hover-mix-target: var(--diffs-computed-editor-active-line-bg);
  }

  :where([data-background]) {
    & [data-gutter-buffer], & [data-column-number] {
      --mix-light: 91%;
      --mix-dark: 85%;
    }

    & [data-line], & [data-no-newline] {
      --mix-light: 88%;
      --mix-dark: 80%;
    }

    & [data-gutter-buffer], & [data-column-number], & [data-line], & [data-no-newline] {
      --diffs-diff-line-mix-target: var(--diffs-bg);

      &[data-line-type="change-deletion"] {
        --diffs-diff-line-mix-target: var(--diffs-bg-deletion-override, var(--diffs-deletion-base));
        --diffs-hover-mix-target: var(--diffs-diff-line-mix-target);

        &:where([data-gutter-buffer], [data-column-number]) {
          color: var(--diffs-fg-number-deletion-override, var(--diffs-deletion-base));
          --diffs-diff-line-mix-target: var(--diffs-bg-deletion-number-override, var(--diffs-deletion-base));
        }

        --diffs-computed-diff-line-bg: light-dark(color-mix(in lab,
            var(--diffs-computed-decoration-bg) var(--mix-light),
            var(--diffs-diff-line-mix-target)), color-mix(in lab,
            var(--diffs-computed-decoration-bg) var(--mix-dark),
            var(--diffs-diff-line-mix-target)));
      }

      &[data-line-type="change-addition"] {
        --diffs-diff-line-mix-target: var(--diffs-bg-addition-override, var(--diffs-addition-base));
        --diffs-hover-mix-target: var(--diffs-diff-line-mix-target);

        &:where([data-gutter-buffer], [data-column-number]) {
          color: var(--diffs-fg-number-addition-override, var(--diffs-addition-base));
          --diffs-diff-line-mix-target: var(--diffs-bg-addition-number-override, var(--diffs-addition-base));
        }

        --diffs-computed-diff-line-bg: light-dark(color-mix(in lab,
            var(--diffs-computed-decoration-bg) var(--mix-light),
            var(--diffs-diff-line-mix-target)), color-mix(in lab,
            var(--diffs-computed-decoration-bg) var(--mix-dark),
            var(--diffs-diff-line-mix-target)));
      }

      &[data-merge-conflict="current"] {
        --diffs-diff-line-mix-target: var(--conflict-bg-current-override, var(--diffs-addition-base));
        --diffs-hover-mix-target: var(--diffs-diff-line-mix-target);

        &:where([data-gutter-buffer], [data-column-number]) {
          color: var(--diffs-fg-number-addition-override, var(--diffs-addition-base));
          --diffs-diff-line-mix-target: var(--conflict-bg-current-number-override, var(--diffs-addition-base));
        }

        --diffs-computed-diff-line-bg: light-dark(color-mix(in lab,
            var(--diffs-computed-decoration-bg) var(--mix-light),
            var(--diffs-diff-line-mix-target)), color-mix(in lab,
            var(--diffs-computed-decoration-bg) var(--mix-dark),
            var(--diffs-diff-line-mix-target)));
      }

      &[data-merge-conflict="incoming"] {
        --diffs-diff-line-mix-target: var(--conflict-bg-incoming-override, var(--diffs-modified-base));
        --diffs-hover-mix-target: var(--diffs-diff-line-mix-target);

        &:where([data-gutter-buffer], [data-column-number]) {
          color: var(--diffs-modified-base);
          --diffs-diff-line-mix-target: var(--conflict-bg-incoming-number-override, var(--diffs-modified-base));
        }

        --diffs-computed-diff-line-bg: light-dark(color-mix(in lab,
            var(--diffs-computed-decoration-bg) var(--mix-light),
            var(--diffs-diff-line-mix-target)), color-mix(in lab,
            var(--diffs-computed-decoration-bg) var(--mix-dark),
            var(--diffs-diff-line-mix-target)));
      }
    }
  }

  [data-gutter-buffer], [data-column-number], [data-line], [data-line-annotation], [data-merge-conflict], [data-merge-conflict-actions], [data-no-newline], [data-editor-overlay] {
    --diffs-selection-mix-target: var(--diffs-bg-selection-override, var(--diffs-selection-base));
    --diffs-selection-emphasis-mix-target: var(--diffs-selection-mix-target);

    &:where([data-editor-overlay]), &:where([data-line], [data-line-annotation], [data-merge-conflict], [data-merge-conflict-actions], [data-no-newline])[data-selected-line] {
      --mix-selection-light: 82%;
      --mix-selection-dark: 75%;
      --diffs-hover-mix-target: var(--diffs-selection-mix-target);
    }

    &:where([data-gutter-buffer][data-selected-line], [data-column-number]:is([data-selected-line], [data-editor-active-line])) {
      --mix-selection-light: 75%;
      --mix-selection-dark: 60%;
      --diffs-selection-mix-target: var(--diffs-bg-selection-number-override, var(--diffs-selection-base));
      --diffs-hover-mix-target: var(--diffs-selection-mix-target);
    }

    &:where([data-line][data-selected-line]):is([data-line-type="change-addition"], [data-line-type="change-deletion"]) {
      --diffs-selection-emphasis-mix-target: light-dark(color-mix(in lab,
          var(--diffs-diff-line-mix-target, var(--diffs-selection-mix-target))
            var(--mix-selection-light),
          var(--diffs-selection-mix-target)), var(--diffs-selection-mix-target));
    }

    &:where([data-editor-overlay]), &[data-selected-line] {
      --diffs-computed-selected-line-bg: light-dark(color-mix(in lab,
          var(--diffs-computed-diff-line-bg) var(--mix-selection-light),
          var(--diffs-selection-mix-target)), color-mix(in lab,
          var(--diffs-computed-diff-line-bg) var(--mix-selection-dark),
          var(--diffs-selection-mix-target)));
    }
  }

  [data-line][data-editor-active-line], [data-column-number][data-editor-active-line] {
    --diffs-computed-editor-active-line-bg: color-mix(in lab,
      var(--diffs-computed-selected-line-bg)
        var(--diffs-editor-active-line-source-mix, 100%),
      var(--diffs-selection-emphasis-mix-target));
  }

  @media (pointer: fine) {
    [data-line][data-hovered], [data-gutter-buffer][data-hovered], [data-column-number][data-hovered], [data-line-annotation][data-hovered], [data-no-newline][data-hovered], [data-merge-conflict][data-hovered], [data-merge-conflict-actions][data-hovered], [data-editor-overlay][data-hovered] {
      --diffs-computed-hovered-line-bg: light-dark(color-mix(in lab,
          var(--diffs-computed-editor-active-line-bg) 97%,
          var(--diffs-hover-mix-target)), color-mix(in lab,
          var(--diffs-computed-editor-active-line-bg) 91%,
          var(--diffs-hover-mix-target)));
    }
  }

  [data-gutter-buffer][data-selected-line], [data-column-number]:is([data-selected-line], [data-editor-active-line]) {
    color: var(--diffs-selection-number-fg);
  }

  [data-no-newline] {
    user-select: none;

    & span {
      opacity: .6;
    }
  }

  [data-diff-type="split"][data-overflow="scroll"] {
    grid-template-columns: 1fr 1fr;
    display: grid;

    & [data-additions] {
      border-left: 1px solid var(--diffs-bg);
    }

    & [data-deletions] {
      border-right: 1px solid var(--diffs-bg);
    }
  }

  [data-code] {
    grid-auto-flow: dense;
    grid-template-columns: var(--diffs-code-grid);
    overflow: var(--diffs-overflow-override, scroll) clip;
    overscroll-behavior-x: none;
    tab-size: var(--diffs-tab-size, 2);
    padding-top: var(--diffs-gap-block, var(--diffs-gap-fallback));
    padding-bottom: max(0px,
      calc(var(--diffs-gap-block, var(--diffs-gap-fallback)) -
          var(--diffs-scrollbar-gutter)));
    scrollbar-gutter: stable;
    align-self: flex-start;
    display: grid;
  }

  [data-diffs-scrollbar-measure] {
    opacity: 0;
    pointer-events: none;
    scrollbar-gutter: auto;
    grid-template-columns: none;
    width: 100px;
    height: 100px;
    padding: 0;
    position: absolute;
    top: -200px;
    left: -200px;
  }

  [data-container-size] {
    container-type: inline-size;
  }

  [data-code]::-webkit-scrollbar {
    width: 0;
    height: var(--diffs-scrollbar-gutter);
  }

  [data-code]::-webkit-scrollbar-track {
    background: none;
  }

  [data-code]::-webkit-scrollbar-thumb {
    background-color: #0000;
    background-clip: content-box;
    border: 1px solid #0000;
    border-radius: 3px;
  }

  [data-code]::-webkit-scrollbar-corner {
    background-color: #0000;
  }

  @supports ((-moz-appearance: none)) {
    [data-code] {
      scrollbar-width: thin;
      scrollbar-color: var(--diffs-bg-context) transparent;
      padding-bottom: var(--diffs-gap-block, var(--diffs-gap-fallback));
    }
  }

  [data-diffs-header] ~ [data-diff], [data-diffs-header] ~ [data-file] {
    & [data-code], &[data-overflow="wrap"], &[data-dehydrated][data-diff-type="split"][data-overflow="scroll"] {
      padding-top: 0;
    }
  }

  [data-gutter] {
    grid-template-rows: subgrid;
    grid-template-columns: subgrid;
    z-index: 3;
    background-color: var(--diffs-bg);
    grid-column: 1;
    display: grid;
    position: relative;

    & [data-gutter-buffer], & [data-column-number] {
      border-right: var(--diffs-gap-style, 2px solid var(--diffs-bg));
    }
  }

  [data-content] {
    grid-template-rows: subgrid;
    grid-template-columns: subgrid;
    background-color: var(--diffs-bg);
    grid-column: 2;
    min-width: 0;
    display: grid;
  }

  [data-diff-type="split"][data-overflow="wrap"], [data-dehydrated][data-diff-type="split"][data-overflow="scroll"] {
    grid-auto-flow: dense;
    grid-template-columns: repeat(2, var(--diffs-code-grid));
    padding-block: var(--diffs-gap-block, var(--diffs-gap-fallback));
    display: grid;

    & [data-code] {
      display: contents;
    }

    & [data-deletions] {
      & [data-gutter] {
        grid-column: 1;
      }

      & [data-content] {
        border-right: 1px solid var(--diffs-bg);
        grid-column: 2;
      }
    }

    & [data-additions] {
      & [data-gutter] {
        border-left: 1px solid var(--diffs-bg);
        grid-column: 3;
      }

      & [data-content] {
        grid-column: 4;
      }
    }
  }

  [data-dehydrated][data-diff-type="split"][data-overflow="scroll"] [data-content] {
    overflow: clip;
  }

  [data-overflow="scroll"] [data-gutter] {
    position: sticky;
    left: 0;
  }

  [data-interactive-lines] [data-line] {
    cursor: pointer;
  }

  [data-interactive-line-numbers] [data-column-number] {
    cursor: pointer;
    touch-action: none;
  }

  [data-content-buffer], [data-gutter-buffer] {
    user-select: none;
    min-height: 1lh;
    position: relative;
  }

  [data-gutter-buffer] {
    padding-left: 2ch;
    padding-right: 1ch;

    &:before {
      content: "";
      min-width: var(--diffs-min-number-column-width, var(--diffs-min-number-column-width-default, 3ch));
      display: block;
    }
  }

  [data-gutter-buffer="annotation"] {
    --diffs-annotation-bg: var(--diffs-bg-context-gutter);
    min-height: 0;
  }

  [data-gutter-buffer="buffer"] {
    --diffs-line-bg: var(--diffs-bg-context-gutter);
  }

  [data-content-buffer] {
    background-position: 5px 0;
    background-size: 8px 8px;
    background-origin: border-box;
    background-image: repeating-linear-gradient(-45deg,
      transparent,
      transparent calc(3px * 1.414),
      var(--diffs-bg-buffer) calc(3px * 1.414),
      var(--diffs-bg-buffer) calc(4px * 1.414));
    grid-column: 1;
  }

  [data-separator] {
    box-sizing: content-box;
    background-color: var(--diffs-bg);
  }

  [data-separator="simple"] {
    min-height: 4px;
  }

  [data-separator="line-info"], [data-separator="line-info-basic"], [data-separator="metadata"], [data-separator="simple"] {
    background-color: var(--diffs-bg-separator);
  }

  [data-separator="line-info"], [data-separator="line-info-basic"], [data-separator="metadata"] {
    height: 32px;
    position: relative;
  }

  [data-separator-wrapper] {
    user-select: none;
    fill: currentColor;
    background-color: var(--diffs-bg);
    align-items: center;
    height: 100%;
    display: flex;
    position: absolute;
    inset-inline: 0;
  }

  [data-content] [data-separator-wrapper] {
    display: none;
  }

  [data-separator="metadata"] [data-separator-wrapper] {
    background-color: var(--diffs-bg-separator);
    height: 100%;
    color: var(--diffs-fg-number);
    white-space: nowrap;
    text-overflow: ellipsis;
    min-width: min-content;
    padding-inline: 1ch;
    inset-inline: 100% auto;
    overflow: hidden;
  }

  [data-separator="line-info"] {
    margin-block: var(--diffs-gap-block, var(--diffs-gap-fallback));

    & [data-separator-wrapper] {
      min-width: 16px;
    }
  }

  [data-separator="line-info-basic"], [data-separator="metadata"] {
    margin-block: 0;
  }

  [data-separator="line-info"][data-separator-first] {
    margin-top: 0;
  }

  [data-separator="line-info"][data-separator-last] {
    margin-bottom: 0;
  }

  [data-expand-index] [data-separator-wrapper] {
    grid-template-columns: 32px auto;
    display: grid;
  }

  [data-expand-index] [data-separator-wrapper][data-separator-multi-button] {
    grid-template-columns: 32px 32px auto;
  }

  [data-expand-button], [data-separator-content] {
    background-color: var(--diffs-bg-separator);
    flex: none;
    align-items: center;
    display: flex;
  }

  [data-expand-index] [data-separator-content]:hover {
    cursor: pointer;
    text-decoration: underline;
  }

  [data-expand-button] {
    cursor: pointer;
    min-width: 32px;
    color: var(--diffs-fg-number);
    border-right: 2px solid var(--diffs-bg);
    flex-shrink: 0;
    justify-content: center;
    align-self: stretch;

    &:hover {
      color: var(--diffs-fg);
    }

    &[data-expand-all-button] {
      display: none;
    }
  }

  [data-expand-down] [data-icon] {
    transform: scaleY(-1);
  }

  [data-separator-content] {
    height: 100%;
    color: var(--diffs-fg-number);
    flex: auto;
    justify-content: flex-start;
    padding: 0 1ch;
    overflow: hidden;
  }

  [data-separator="line-info"], [data-separator="line-info-basic"] {
    & [data-separator-content] {
      user-select: none;
      height: 100%;
      overflow: clip;
    }
  }

  [data-unmodified-lines] {
    text-overflow: ellipsis;
    white-space: nowrap;
    flex: 0 auto;
    min-width: 0;
    display: block;
    overflow: hidden;
  }

  @supports (width: 1cqi) {
    [data-unified] {
      & [data-separator="line-info"] [data-separator-wrapper] {
        padding-inline: var(--diffs-gap-inline, var(--diffs-gap-fallback));
        width: 100cqi;

        & [data-separator-content] {
          border-radius: 6px;
        }
      }

      & [data-separator="line-info"][data-expand-index] [data-separator-wrapper] [data-separator-content] {
        border-top-left-radius: unset;
        border-bottom-left-radius: unset;
      }
    }

    [data-gutter] {
      & [data-separator="line-info"] [data-separator-wrapper] {
        padding-left: var(--diffs-gap-inline, var(--diffs-gap-fallback));
      }

      & [data-separator="line-info"] [data-separator-content] {
        border-top-left-radius: 6px;
        border-bottom-left-radius: 6px;
      }

      & [data-separator="line-info"][data-expand-index] [data-separator-content] {
        border-top-left-radius: unset;
        border-bottom-left-radius: unset;
      }
    }

    [data-additions] {
      & [data-content] [data-separator="line-info"] {
        background-color: var(--diffs-bg);

        & [data-separator-wrapper] {
          display: none;
        }
      }

      & [data-gutter] [data-separator="line-info"] [data-separator-wrapper] {
        background-color: var(--diffs-bg-separator);
        border-top-right-radius: 6px;
        border-bottom-right-radius: 6px;
        height: 100%;
        display: block;

        & [data-separator-content], & [data-expand-button] {
          display: none;
        }
      }
    }

    [data-overflow="scroll"] [data-additions] [data-gutter] [data-separator="line-info"] [data-separator-wrapper] {
      width: calc(100cqi - var(--diffs-gap-inline, var(--diffs-gap-fallback)));
    }

    [data-overflow="wrap"] [data-additions] [data-content] [data-separator="line-info"] [data-separator-wrapper] {
      background-color: var(--diffs-bg-separator);
      height: 100%;
      margin-right: var(--diffs-gap-inline, var(--diffs-gap-fallback));
      border-top-right-radius: 6px;
      border-bottom-right-radius: 6px;
      display: block;

      & [data-separator-content], & [data-expand-button] {
        display: none;
      }
    }

    [data-separator="line-info"] [data-separator-wrapper] {
      & [data-expand-both], & [data-expand-down], & [data-expand-up] {
        border-top-left-radius: 6px;
        border-bottom-left-radius: 6px;
      }
    }

    @media (pointer: fine) {
      [data-separator="line-info"] [data-separator-wrapper] {
        &[data-separator-multi-button] {
          & [data-expand-up] {
            border-top-left-radius: 6px;
            border-bottom-left-radius: unset;
          }

          & [data-expand-down] {
            border-bottom-left-radius: 6px;
            border-top-left-radius: unset;
          }
        }
      }
    }
  }

  @media (pointer: coarse) {
    [data-separator="line-info-basic"] [data-separator-wrapper][data-separator-multi-button] {
      grid-template-columns: 34px 34px auto;

      & [data-separator-content] {
        grid-column: unset;
        grid-row: unset;
      }
    }

    @supports (width: 1cqi) {
      [data-separator="line-info"] [data-separator-wrapper] {
        & [data-expand-both], & [data-expand-down], & [data-expand-up] {
          border-top-left-radius: 6px;
          border-bottom-left-radius: 6px;
        }

        &[data-separator-multi-button] {
          & [data-expand-up] {
            border-top-left-radius: 6px;
            border-bottom-left-radius: 6px;
          }

          & [data-expand-down] {
            border-bottom-left-radius: unset;
            border-top-left-radius: unset;
          }
        }
      }
    }
  }

  @media (pointer: fine) {
    [data-separator-wrapper][data-separator-multi-button] {
      grid-template-rows: 50% 50%;
      display: grid;

      & [data-separator-content] {
        grid-area: 1 / 2 / -1;
        min-width: min-content;
      }

      & [data-expand-button] {
        grid-column: 1;
      }
    }

    [data-separator="line-info"] [data-separator-wrapper], [data-separator="line-info"] [data-separator-wrapper][data-separator-multi-button] {
      grid-template-columns: 34px auto;
    }

    [data-separator="line-info-basic"][data-expand-index] [data-separator-wrapper] {
      grid-template-columns: 100% auto;
    }

    [data-separator="line-info"], [data-separator="line-info-basic"] {
      & [data-separator-multi-button] {
        & [data-expand-up] {
          border-bottom: 1px solid var(--diffs-bg);
          border-right: 2px solid var(--diffs-bg);
        }

        & [data-expand-down] {
          border-top: 1px solid var(--diffs-bg);
          border-right: 2px solid var(--diffs-bg);
        }
      }
    }
  }

  [data-additions] [data-gutter] [data-separator-wrapper], [data-additions] [data-separator="line-info-basic"] [data-separator-wrapper], [data-content] [data-separator-wrapper] {
    display: none;
  }

  [data-line-annotation] {
    min-height: var(--diffs-annotation-min-height, 0);
    z-index: 2;
  }

  [data-merge-conflict-actions] {
    z-index: 2;
  }

  [data-separator="custom"] {
    grid-template-columns: subgrid;
    display: grid;
  }

  [data-line], [data-column-number], [data-no-newline] {
    padding-inline: 1ch;
    position: relative;
  }

  [data-indicators="classic"] [data-line] {
    padding-inline-start: 2ch;
  }

  [data-indicators="classic"] {
    & [data-line-type="change-addition"], & [data-line-type="change-deletion"] {
      &[data-no-newline], &[data-line] {
        &:before {
          user-select: none;
          width: 1ch;
          height: 1lh;
          display: inline-block;
          position: absolute;
          top: 0;
          left: 0;
        }
      }
    }

    & [data-line-type="change-addition"] {
      &[data-line], &[data-no-newline] {
        &:before {
          content: "+";
          color: var(--diffs-addition-base);
        }
      }
    }

    & [data-line-type="change-deletion"] {
      &[data-line], &[data-no-newline] {
        &:before {
          content: "-";
          color: var(--diffs-deletion-base);
        }
      }
    }
  }

  [data-indicators="bars"] {
    & [data-line-type="change-deletion"], & [data-line-type="change-addition"] {
      &[data-column-number] {
        &:before {
          content: "";
          user-select: none;
          contain: strict;
          width: 4px;
          height: 100%;
          display: block;
          position: absolute;
          top: 0;
          left: 0;
        }
      }
    }

    & [data-line-type="change-deletion"] {
      &[data-column-number] {
        &:before {
          background-image: linear-gradient(0deg,
            var(--diffs-bg-deletion) 50%,
            var(--diffs-deletion-base) 50%);
          background-repeat: repeat;
          background-size: 2px 2px;
          background-size: calc(1lh / round(1lh / 2px))
            calc(1lh / round(1lh / 2px));
        }
      }
    }

    & [data-line-type="change-addition"] {
      &[data-column-number] {
        &:before {
          background-color: var(--diffs-addition-base);
        }
      }
    }
  }

  [data-overflow="wrap"] {
    & [data-line] {
      white-space: pre-wrap;
      word-break: break-word;
    }

    & [data-annotation-content] {
      word-break: break-word;
    }
  }

  [data-overflow="scroll"] [data-line] {
    white-space: pre;
    min-height: 1lh;
  }

  [data-column-number] {
    box-sizing: content-box;
    text-align: right;
    user-select: none;
    color: var(--diffs-fg-number);
    padding-left: 2ch;
  }

  [data-line-number-content] {
    min-width: var(--diffs-min-number-column-width, var(--diffs-min-number-column-width-default, 3ch));
    z-index: 1;
    display: inline-block;
    position: relative;
  }

  [data-disable-line-numbers] {
    & [data-gutter-buffer], & [data-column-number] {
      min-width: 4px;
      padding: 0;

      &:before {
        min-width: 0;
      }
    }

    & [data-line-number-content] {
      display: none;
    }

    & [data-gutter-utility-slot] {
      right: unset;
      justify-content: flex-start;
      left: 0;
    }

    &[data-indicators="bars"] [data-gutter-utility-slot] {
      left: 6px;
    }
  }

  [data-file][data-disable-line-numbers] {
    & [data-gutter-buffer], & [data-column-number] {
      border-right: 0;
      min-width: 0;
    }
  }

  [data-diff-span] {
    box-decoration-break: clone;
    border-radius: 3px;
  }

  [data-line-type="change-addition"] [data-diff-span] {
    background-color: var(--diffs-bg-addition-emphasis);
  }

  [data-line-type="change-deletion"] [data-diff-span] {
    background-color: var(--diffs-bg-deletion-emphasis);
  }

  [data-merge-conflict="marker-start"], [data-merge-conflict="marker-base"], [data-merge-conflict="marker-separator"], [data-merge-conflict="marker-end"] {
    color: var(--diffs-fg);
    padding-left: 1ch;
  }

  [data-merge-conflict="marker-start"], [data-merge-conflict="marker-end"] {
    align-items: center;
    display: flex;

    &:after {
      color: var(--diffs-fg-conflict-marker);
      font-size: .75rem;
      font-style: normal;
      line-height: 1.25rem;
      font-family: var(--diffs-header-font-family, var(--diffs-header-font-fallback));
      padding-left: 1ch;
    }
  }

  [data-merge-conflict="marker-start"]:after {
    content: "(Current Change)";
  }

  [data-merge-conflict="marker-end"]:after {
    content: "(Incoming Change)";
  }

  [data-merge-conflict-actions-content] {
    min-height: 1.75rem;
    font-family: var(--diffs-header-font-family, var(--diffs-header-font-fallback));
    color: var(--diffs-fg);
    align-items: center;
    gap: .25rem;
    padding-inline: .5rem;
    font-size: .75rem;
    line-height: 1.2;
    display: flex;
  }

  [data-merge-conflict-action] {
    appearance: none;
    color: var(--diffs-fg-number);
    font: inherit;
    cursor: pointer;
    background: none;
    border: 0;
    padding: 0;
    font-style: normal;
  }

  [data-merge-conflict-action]:hover {
    color: var(--diffs-fg);
  }

  [data-merge-conflict-action="current"]:hover {
    color: var(--diffs-addition-base);
  }

  [data-merge-conflict-action="incoming"]:hover {
    color: var(--diffs-modified-base);
  }

  [data-merge-conflict-action-separator] {
    color: var(--diffs-fg-number);
    opacity: .6;
    user-select: none;
  }

  [data-diffs-header="default"] {
    background-color: var(--diffs-bg);
    justify-content: space-between;
    align-items: center;
    gap: var(--diffs-gap-inline, var(--diffs-gap-fallback));
    min-height: calc(1lh + (var(--diffs-gap-block, var(--diffs-gap-fallback)) * 3));
    z-index: 2;
    flex-direction: row;
    padding-inline: 16px;
    display: flex;
    position: relative;
    top: 0;
  }

  [data-header-content] {
    align-items: center;
    gap: var(--diffs-gap-inline, var(--diffs-gap-fallback));
    white-space: nowrap;
    flex-direction: row;
    min-width: 0;
    display: flex;
  }

  [data-header-content] [data-prev-name], [data-header-content] [data-title] {
    text-overflow: ellipsis;
    white-space: nowrap;
    direction: rtl;
    min-width: 0;
    overflow: hidden;
  }

  [data-prev-name] {
    opacity: .7;
  }

  [data-rename-icon] {
    fill: currentColor;
    flex-grow: 0;
    flex-shrink: 0;
  }

  [data-diffs-header="default"] [data-metadata] {
    white-space: nowrap;
    align-items: center;
    gap: 1ch;
    display: flex;
  }

  [data-diffs-header="default"] [data-additions-count] {
    font-family: var(--diffs-font-family, var(--diffs-font-fallback));
    color: var(--diffs-addition-base);
  }

  [data-diffs-header="default"] [data-deletions-count] {
    font-family: var(--diffs-font-family, var(--diffs-font-fallback));
    color: var(--diffs-deletion-base);
  }

  [data-change-icon] {
    fill: currentColor;
    flex-shrink: 0;
  }

  [data-change-icon="change"], [data-change-icon="rename-pure"], [data-change-icon="rename-changed"] {
    color: var(--diffs-modified-base);
  }

  [data-change-icon="new"] {
    color: var(--diffs-addition-base);
  }

  [data-change-icon="deleted"] {
    color: var(--diffs-deletion-base);
  }

  [data-change-icon="file"] {
    opacity: .6;
  }

  [data-annotation-content] {
    z-index: 2;
    isolation: isolate;
    white-space: normal;
    align-self: flex-start;
    min-width: 0;
    display: flow-root;
    position: relative;
  }

  [data-overflow="scroll"] [data-annotation-content], [data-overflow="scroll"] [data-merge-conflict-actions-content] {
    width: var(--diffs-column-content-width, auto);
    left: var(--diffs-column-number-width, 0);
    position: sticky;
  }

  [data-annotation-slot] {
    text-wrap-mode: wrap;
    word-break: normal;
    white-space-collapse: collapse;
  }

  [data-gutter-utility-slot] {
    touch-action: none;
    justify-content: flex-end;
    display: flex;
    position: absolute;
    top: 0;
    bottom: 0;
    right: 0;
  }

  [data-utility-button] {
    appearance: none;
    cursor: pointer;
    width: 1lh;
    height: 1lh;
    font-size: var(--diffs-font-size, 13px);
    line-height: var(--diffs-line-height, 20px);
    background-color: var(--diffs-modified-base);
    color: var(--diffs-bg);
    fill: currentColor;
    z-index: 4;
    touch-action: none;
    border: none;
    border-radius: 4px;
    justify-content: center;
    align-items: center;
    margin-right: calc(-1lh + 1ch);
    padding: 0;
    display: flex;
    position: relative;

    &:before {
      content: "";
      display: block;
      position: absolute;
      inset: 0 0 0 -4px;
    }
  }

  [data-decoration-bar-stack] {
    pointer-events: none;
    isolation: isolate;
    z-index: 1;
    background-color: var(--diffs-decoration-bar-color, transparent);
    box-sizing: content-box;
    border-left: 2px solid var(--diffs-bg);
    border-right: 2px solid var(--diffs-bg);
    width: 6px;
    position: absolute;
    top: 0;
    bottom: 0;
    right: -2px;

    [data-decoration-bar-depth="1"] & {
      background-color: color-mix(in lab,
        var(--diffs-bg) 20%,
        var(--diffs-decoration-bar-color, transparent));
    }

    [data-decoration-bar-depth="2"] & {
      background-color: color-mix(in lab,
        var(--diffs-bg) 45%,
        var(--diffs-decoration-bar-color, transparent));
    }

    [data-decoration-bar-depth="3"] & {
      background-color: color-mix(in lab,
        var(--diffs-bg) 65%,
        var(--diffs-decoration-bar-color, transparent));
    }

    [data-decoration-bar-start] & {
      border-top-left-radius: 5px;
      border-top-right-radius: 5px;
    }

    [data-decoration-bar-end] & {
      z-index: 3;
      border-bottom-right-radius: 5px;
      border-bottom-left-radius: 5px;
    }
  }

  [data-placeholder] {
    contain: strict;
  }

  [data-error-wrapper] {
    padding: var(--diffs-gap-block, var(--diffs-gap-fallback))
      var(--diffs-gap-inline, var(--diffs-gap-fallback));
    scrollbar-width: none;
    max-height: 400px;
    overflow: auto;

    & [data-error-message] {
      color: var(--diffs-deletion-base);
      font-size: 18px;
      font-weight: bold;
    }

    & [data-error-stack] {
      color: var(--diffs-fg-number);
    }
  }
}

@layer theme, rendered, unsafe;
`;
let St;
function Ke(e) {
  if (St != null) return St;
  const t = e.host;
  if (typeof HTMLElement < "u" && t instanceof HTMLElement && !t.isConnected)
    return;
  const n = document.createElement("div");
  (n.setAttribute("data-code", ""), n.setAttribute(Br, "true"));
  const i = document.createElement("div");
  return (
    (i.style.position = "relative"),
    (i.style.width = "200%"),
    (i.style.height = "200%"),
    n.appendChild(i),
    e.appendChild(n),
    (St = Math.max(n.offsetHeight - n.clientHeight, 0)),
    n.remove(),
    St
  );
}
function lr(e) {
  return `${Ni}: ${e == null ? "var(--diffs-scrollbar-gutter-fallback)" : `${e}px`};`;
}
const En = "@layer base, theme, rendered, unsafe;",
  rs = new RegExp(`${ss(Ni)}\\s*:\\s*[^;]+;`);
function os(e) {
  return `${En}
${ar}
@layer theme {
  ${e}
}`;
}
function Rn(e) {
  return `${En}
@layer unsafe {
  ${e}
}`;
}
function Tn(e, t = "system", n) {
  return `${En}
@layer rendered {
  :host {${
    t === "system"
      ? ""
      : `
  color-scheme: ${t};`
  }
  ${lr(n)}
  ${e}
  }
}`;
}
function dr(e, t) {
  const n = lr(t);
  return e.replace(rs, n);
}
function ss(e) {
  return e.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}
const hr = new WeakMap();
function An(e) {
  const t = hr.get(e);
  return t ?? e;
}
function Yl(e, t) {
  hr.set(e, t);
}
function rt({
  code: e,
  pre: t,
  columnType: n,
  rowSpan: i,
  containerSize: r = !1,
} = {}) {
  return (
    e == null &&
      ((e = document.createElement("code")),
      e.setAttribute("data-code", ""),
      n != null && e.setAttribute(`data-${n}`, ""),
      t?.appendChild(e)),
    i != null
      ? e.style.setProperty("grid-row", `span ${i}`)
      : e.style.removeProperty("grid-row"),
    r
      ? e.setAttribute("data-container-size", "")
      : e.removeAttribute("data-container-size"),
    e
  );
}
function cr(e, t) {
  if (t == null) return;
  const n = e.shadowRoot ?? e.attachShadow({ mode: "open" });
  n.innerHTML === "" && (n.innerHTML = t);
}
function In(
  e,
  {
    type: t,
    diffIndicators: n,
    disableBackground: i,
    disableLineNumbers: r,
    overflow: o,
    split: s,
    totalLines: a,
    customProperties: l,
  },
) {
  if (l != null)
    for (const d in l) {
      const h = l[d];
      h != null && e.setAttribute(d, `${h}`);
    }
  switch (
    (t === "diff"
      ? (e.setAttribute("data-diff", ""), e.removeAttribute("data-file"))
      : (e.setAttribute("data-file", ""), e.removeAttribute("data-diff")),
    n)
  ) {
    case "bars":
    case "classic":
      e.setAttribute("data-indicators", n);
      break;
    case "none":
      e.removeAttribute("data-indicators");
      break;
  }
  return (
    r
      ? e.setAttribute("data-disable-line-numbers", "")
      : e.removeAttribute("data-disable-line-numbers"),
    i
      ? e.removeAttribute("data-background")
      : e.setAttribute("data-background", ""),
    t === "diff"
      ? e.setAttribute("data-diff-type", s ? "split" : "single")
      : e.removeAttribute("data-diff-type"),
    e.setAttribute("data-overflow", o),
    e.style.setProperty(
      "--diffs-min-number-column-width-default",
      `${`${a}`.length}ch`,
    ),
    e
  );
}
function at(e) {
  if (typeof HTMLStyleElement < "u" && e instanceof HTMLStyleElement) return !0;
  const t = e.tagName ?? e.nodeName;
  return typeof t == "string" && t.toLowerCase() === "style";
}
function zt(e, t, n) {
  const i = new Map();
  for (const r of e) {
    const o = An(r);
    i.set(o, n?.slotNames.get(o) ?? t(r));
  }
  return { provided: e, current: e, slotNames: i };
}
function ur(e, t, n) {
  for (const i of t) {
    const r = An(i);
    e.slotNames.has(r) || e.slotNames.set(r, n(i));
  }
  ((e.provided = t), (e.current = t));
}
function fr(e, t, n) {
  if (e == null) return n(t);
  const i = An(t);
  let r = e.slotNames.get(i);
  return (r == null && ((r = n(t)), e.slotNames.set(i, r)), r);
}
function Hn(e) {
  return (
    e?.useTokenTransformer === !0 ||
    e?.onTokenClick != null ||
    e?.onTokenEnter != null ||
    e?.onTokenLeave != null
  );
}
function Kn(e) {
  return {
    theme: e?.theme,
    disableLineNumbers: e?.disableLineNumbers,
    overflow: e?.overflow,
    themeType: e?.themeType,
    collapsed: e?.collapsed,
    disableFileHeader: e?.disableFileHeader,
    disableVirtualizationBuffers: e?.disableVirtualizationBuffers,
    stickyHeader: e?.stickyHeader,
    preferredHighlighter: e?.preferredHighlighter,
    useCSSClasses: e?.useCSSClasses,
    useTokenTransformer: Hn(e),
    tokenizeMaxLineLength: e?.tokenizeMaxLineLength,
    tokenizeMaxLength: e?.tokenizeMaxLength,
    unsafeCSS: e?.unsafeCSS,
    headerRenderMode: e?.renderCustomHeader != null ? "custom" : "default",
  };
}
function dn(e, t) {
  const n = e?.offsetHeight ?? 0;
  if (e == null || n === 0) {
    t();
    return;
  }
  e.style.minHeight = `${n}px`;
  try {
    (t(), e.offsetHeight);
  } finally {
    e.style.minHeight = "";
  }
}
function Mn({ shadowRoot: e, currentNode: t, themeCSS: n }) {
  if (n.trim() === "") {
    t?.remove();
    return;
  }
  return (
    (t ??= as()),
    (t.textContent = n),
    t.parentNode !== e && e.appendChild(t),
    t
  );
}
function as() {
  const e = document.createElement("style");
  return (e.setAttribute(zi, ""), e);
}
let ls;
function pr() {
  return (ls ??=
    ("safari" in window && "pushNotification" in window.safari) ||
    /^((?!chrome|android).)*safari/i.test(navigator.userAgent));
}
if (typeof HTMLElement < "u" && customElements.get("diffs-container") == null) {
  let e;
  class t extends HTMLElement {
    constructor() {
      if ((super(), this.shadowRoot != null)) return;
      const i = this.attachShadow({ mode: "open" });
      (e == null && ((e = new CSSStyleSheet()), e.replaceSync(ar)),
        (i.adoptedStyleSheets = [e]));
    }
    connectedCallback() {
      Ke(this.shadowRoot ?? this.attachShadow({ mode: "open" }));
    }
  }
  customElements.define(Ui, t);
}
const ds = [""];
function hs(e) {
  const t = { ...e };
  return (delete t.cacheKey, t);
}
let cs = -1;
var us = class {
  options;
  workerManager;
  isContainerManaged;
  static LoadedCustomComponent = !0;
  __id = `file:${++cs}`;
  type = "file";
  fileContainer;
  spriteSVG;
  pre;
  code;
  bufferBefore;
  bufferAfter;
  themeCSSStyle;
  appliedThemeCSS;
  hasAdoptedThemeCSS = !1;
  unsafeCSSStyle;
  appliedUnsafeCSS;
  gutterUtilityContent;
  errorWrapper;
  placeHolder;
  lastRenderedHeaderHTML;
  cachedHeaderHTML;
  appliedPreAttributes;
  lastRowCount;
  mounted = !1;
  headerElement;
  headerCustom;
  headerPrefix;
  headerFilenameSuffix;
  headerMetadata;
  fileRenderer;
  resizeManager;
  interactionManager;
  annotationCache = new Map();
  lineAnnotations = [];
  managersDirty = !1;
  file;
  editSession;
  renderedFile;
  renderRange;
  enabled = !0;
  editor;
  constructor(e = { theme: U }, t, n = !1) {
    ((this.options = e),
      (this.workerManager = t),
      (this.isContainerManaged = n),
      (this.fileRenderer = new ts(
        e,
        this.getAnnotationSlotName,
        this.handleHighlightRender,
        this.workerManager,
      )),
      (this.resizeManager = new ji()),
      (this.interactionManager = new qi("file", st(e))),
      this.workerManager?.subscribeToThemeChanges(this));
  }
  getAnnotationSlotName = (e) => fr(this.editSession?.annotations, e, fe);
  handleHighlightRender = () => {
    this.rerender();
  };
  rerender() {
    !this.enabled ||
      this.file == null ||
      this.render({
        file: this.file,
        forceRender: !0,
        renderRange: this.renderRange,
      });
  }
  getTheme() {
    return (
      this.workerManager?.getFileRenderOptions().theme ??
      this.options.theme ??
      U
    );
  }
  getLatestFile(e = this.file) {
    return this.editSession?.file ?? e;
  }
  getRenderedFile() {
    return this.renderedFile;
  }
  getLatestAnnotations() {
    return this.editSession?.annotations?.current ?? this.lineAnnotations;
  }
  isNewAnnotations(e) {
    const t = this.editSession?.annotations;
    return e === this.lineAnnotations
      ? !1
      : t == null || (e !== t.provided && e !== t.current);
  }
  updateExternalFile(e, t) {
    if ($(this.file, e)) return !1;
    const n = this.editSession != null;
    return (
      (this.file = e),
      n || this.editor != null
        ? this.installEditSession(
            e,
            n ? void 0 : this.editor?.__getDocumentContents(e),
            !0,
          )
        : (this.editSession = void 0),
      this.editSession?.annotations != null &&
        t != null &&
        ((this.lineAnnotations = t),
        (this.editSession.annotations = zt(
          t,
          fe,
          this.editSession.annotations,
        ))),
      !0
    );
  }
  installEditSession(e, t, n = !1) {
    const i =
        t == null ||
        (t.name === e.name && t.lang === e.lang && t.contents === e.contents),
      r = hs(t ?? e);
    ((this.editSession = {
      file: r,
      externalReplacement: n && i,
      annotations:
        this.editSession?.annotations ?? zt(this.lineAnnotations, fe),
    }),
      this.fileRenderer.beginEditSession(r, i ? e : void 0));
  }
  onThemeChange() {
    (this.fileRenderer.clearRenderCache(), this.rerender());
  }
  setOptions(e) {
    e != null &&
      ((this.options = e),
      (this.cachedHeaderHTML = void 0),
      this.syncInteractionOptions());
  }
  syncInteractionOptions() {
    this.interactionManager.setOptions(st(this.options));
  }
  mergeOptions(e) {
    this.options = { ...this.options, ...e };
  }
  setThemeType(e) {
    (this.options.themeType ?? "system") !== e &&
      (this.mergeOptions({ themeType: e }), this.applyCachedThemeState(e));
  }
  applyCachedThemeState(e) {
    if (
      typeof this.options.theme == "string" ||
      this.fileContainer == null ||
      this.appliedThemeCSS == null
    )
      return !1;
    const t = this.appliedThemeCSS.baseThemeType ?? e;
    return this.appliedThemeCSS.themeType === t
      ? !1
      : (this.applyThemeState(
          this.fileContainer,
          this.appliedThemeCSS.themeStyles,
          e,
          this.appliedThemeCSS.baseThemeType,
        ),
        !0);
  }
  hasThemeChanged() {
    return (
      this.appliedThemeCSS != null &&
      !Ue(this.appliedThemeCSS.theme, this.getTheme())
    );
  }
  getHoveredLine = () => this.interactionManager.getHoveredLine();
  setLineAnnotations(e) {
    const t = this.editSession?.annotations;
    if (t == null) {
      this.lineAnnotations = e;
      return;
    }
    this.isNewAnnotations(e) && ((this.lineAnnotations = e), ur(t, e, fe));
  }
  syncEditSessionAnnotationsFromEditor(e) {
    const t = this.editSession?.annotations;
    return t == null || e === t.current
      ? !1
      : ((t.current = e),
        this.fileRenderer.setLineAnnotations(e),
        this.renderAnnotations(),
        !0);
  }
  setSelectedLines(e, t) {
    this.interactionManager.setSelection(e, t);
  }
  setEditorActiveLine(e, t) {
    this.interactionManager.setEditorActiveLine(e, {
      lineNumberOnly: t?.lineNumberOnly,
      side: t?.side ?? "additions",
    });
  }
  getCodeScrollLeft() {
    return this.code?.scrollLeft ?? 0;
  }
  setCodeScrollLeft(e) {
    this.code != null && (this.code.scrollLeft = e);
  }
  __getEffectiveCodeOptions() {
    return { ...this.options, ...this.fileRenderer.getEffectiveCodeOptions() };
  }
  flushManagers() {
    if (!this.managersDirty || this.pre == null) {
      this.managersDirty = !1;
      return;
    }
    const { overflow: e = "scroll" } = this.options;
    (this.interactionManager.setup(this.pre),
      this.resizeManager.setup(this.pre, {
        disableAnnotations: e === "wrap",
        columnVariables: this.shouldApplyColumnVariables(e)
          ? "apply"
          : "measure",
      }),
      (this.managersDirty = !1));
  }
  shouldApplyColumnVariables(e) {
    return e === "scroll" && this.getLatestAnnotations().length > 0;
  }
  cleanUp(e = !1) {
    const t = this.editor;
    (this.emitPostRender(!0),
      t?.cleanUp(e ? "recycle" : "discard"),
      e || (this.editor = void 0),
      this.resizeManager.cleanUp(),
      this.interactionManager.cleanUp(),
      (this.managersDirty = !1),
      this.workerManager?.unsubscribeToThemeChanges(this),
      (this.renderRange = void 0),
      this.isContainerManaged || this.fileContainer?.remove(),
      (this.fileContainer = void 0),
      (this.mounted = !1),
      e || (this.lineAnnotations = []),
      this.clearAuxiliaryNodes(),
      (this.pre = void 0),
      (this.code = void 0),
      this.bufferBefore?.remove(),
      (this.bufferBefore = void 0),
      this.bufferAfter?.remove(),
      (this.bufferAfter = void 0),
      (this.appliedPreAttributes = void 0),
      (this.lastRowCount = void 0),
      (this.headerElement = void 0),
      (this.headerPrefix = void 0),
      (this.headerFilenameSuffix = void 0),
      (this.headerMetadata = void 0),
      (this.headerCustom = void 0),
      (this.lastRenderedHeaderHTML = void 0),
      e || (this.cachedHeaderHTML = void 0),
      this.errorWrapper?.remove(),
      (this.errorWrapper = void 0),
      (this.spriteSVG = void 0),
      (this.themeCSSStyle = void 0),
      (this.appliedThemeCSS = void 0),
      (this.hasAdoptedThemeCSS = !1),
      (this.unsafeCSSStyle = void 0),
      (this.appliedUnsafeCSS = void 0),
      this.placeHolder?.remove(),
      (this.placeHolder = void 0),
      e
        ? this.fileRenderer.recycle()
        : (this.fileRenderer.cleanUp(),
          (this.workerManager = void 0),
          (this.file = void 0),
          (this.editSession = void 0),
          (this.renderedFile = void 0)),
      (this.enabled = !1));
  }
  virtualizedSetup() {
    ((this.enabled = !0), this.workerManager?.subscribeToThemeChanges(this));
  }
  hydrate(e) {
    const {
      fileContainer: t,
      prerenderedHTML: n,
      preventEmit: i = !1,
      file: r,
      lineAnnotations: o,
    } = e;
    if (!this.enabled)
      throw new Error(
        "File.hydrate: attempting to call hydrate after cleaned up",
      );
    if (this.fileContainer != null)
      throw new Error(
        "File.hydrate: hydrate can only be called before the instance has rendered or hydrated",
      );
    this.hydrateElements(t, n);
    const s = this.editor != null;
    (s ||
    fs(this.pre, r, this.options.collapsed) ||
    ps(this.headerElement, r, this.options.disableFileHeader)
      ? this.render({ ...e, forceRender: s || e.forceRender, preventEmit: !0 })
      : this.hydrationSetup({ file: r, lineAnnotations: o }),
      i || this.emitPostRender());
  }
  hydrateElements(e, t) {
    (this.fileContainer !== e && this.emitPostRender(!0), cr(e, t));
    for (const n of Array.from(e.shadowRoot?.children ?? [])) {
      if (n instanceof SVGElement) {
        this.spriteSVG = n;
        continue;
      }
      if (n instanceof HTMLElement) {
        if (n instanceof HTMLPreElement) {
          ((this.pre = n), (this.appliedPreAttributes = void 0));
          continue;
        }
        if (n instanceof HTMLStyleElement && n.hasAttribute("data-theme-css")) {
          this.themeCSSStyle = n;
          continue;
        }
        if (
          n instanceof HTMLStyleElement &&
          n.hasAttribute("data-unsafe-css")
        ) {
          ((this.unsafeCSSStyle = n), (this.appliedUnsafeCSS = n.textContent));
          continue;
        }
        if ("diffsHeader" in n.dataset) {
          ((this.headerElement = n), (this.lastRenderedHeaderHTML = void 0));
          continue;
        }
      }
    }
    (this.pre != null &&
      (this.syncCodeNodeFromPre(this.pre),
      this.pre.removeAttribute("data-dehydrated")),
      (this.fileContainer = e),
      this.hydrateMeasuredScrollbar());
  }
  hydrationSetup({ file: e, lineAnnotations: t }) {
    ((this.lineAnnotations = t ?? this.lineAnnotations),
      (this.file = e),
      this.fileRenderer.setOptions(Kn(this.options)),
      this.syncInteractionOptions(),
      this.pre != null &&
        (this.fileRenderer.hydrate(e),
        (this.renderedFile = e),
        this.renderAnnotations(),
        this.renderGutterUtility(),
        this.injectUnsafeCSS(),
        (this.managersDirty = !0),
        this.flushManagers()));
  }
  getOrCreateLineCache(e = this.getLatestFile()) {
    return e != null ? this.fileRenderer.getOrCreateLineCache(e) : ds;
  }
  updateBuffers(e) {
    this.pre != null && this.applyBuffers(this.pre, e);
  }
  syncRenderViewToEditor() {
    const { editor: e, fileContainer: t, renderRange: n } = this,
      i = this.getLatestAnnotations(),
      r = this.getLatestFile();
    if (e == null || t == null || r == null) return;
    const o = (d) => {
        !this.enabled ||
          this.editor !== e ||
          this.fileContainer !== t ||
          this.getLatestFile() !== r ||
          e.__syncRenderView({
            highlighter: d,
            fileContainer: t,
            file: r,
            lineAnnotations: i,
            renderRange: n,
            externalDocument: this.editSession?.externalReplacement === !0,
          });
      },
      s = this.getTheme(),
      a = r.lang ?? j(r.name),
      l = Ge({ theme: s, lang: a });
    l != null
      ? o(l)
      : ct({
          themes: ht(s),
          langs: Array.from(new Set(["text", a])),
          preferredHighlighter:
            this.workerManager?.getPreferredHighlighter() ??
            this.options.preferredHighlighter,
        }).then(o);
  }
  __acknowledgeDocumentUpdate() {
    this.editSession != null && (this.editSession.externalReplacement = !1);
  }
  __acceptEditorChange(e) {
    const { lineAnnotations: t } = e;
    t != null && this.syncEditSessionAnnotationsFromEditor(t);
  }
  emitEditChange(e) {
    const { onEditChange: t } = this.options;
    t?.(e);
  }
  __captureDocumentSessionState() {}
  __attachEditor(e) {
    if (this.editor != null)
      throw new Error("File.__attachEditor: an editor is already attached");
    this.editor = e;
    const t = () => {
      ((this.editor = void 0), this.fileRenderer.endEditSession());
    };
    try {
      return (this.resumeEditorRendering(e), t);
    } catch (n) {
      throw (t(), n);
    }
  }
  __resumeEditor(e) {
    if (this.editor !== e)
      throw new Error("File.__resumeEditor: editor association changed");
    this.resumeEditorRendering(e);
  }
  resumeEditorRendering(e) {
    this.editSession != null
      ? this.fileRenderer.beginEditSession(this.editSession.file)
      : this.file != null &&
        this.installEditSession(this.file, e.__getDocumentContents(this.file));
    const t = this.editSession?.file;
    this.fileRenderer.editorRenderReady()
      ? (this.fileRenderer.fileCache === t && (this.renderedFile = t),
        this.syncRenderViewToEditor())
      : this.rerender();
  }
  __completeEditSession(e, t) {
    this.settleEditSession(t === "install", e);
  }
  settleEditSession(e, t) {
    const { editSession: n, file: i, lineAnnotations: r } = this;
    if (n == null || i == null) return;
    const { file: o, annotations: s } = n;
    if (this.editor != null)
      throw new Error(
        "File.__completeEditSession: detach the editor before completing the session",
      );
    const a = s?.current;
    let l,
      d = !1,
      h;
    if (t == null)
      throw new Error(
        "File.__completeEditSession: editor is required for completion",
      );
    const c = { ...o },
      u = {
        file: c,
        editor: t,
        originalFile: i,
        lineAnnotations: a,
        originalLineAnnotations: r,
      };
    Object.freeze(u);
    try {
      if (
        (t.__emitEditComplete(u), this.options.onEditComplete?.(u) === "accept")
      ) {
        if (c.cacheKey != null && c.cacheKey === i.cacheKey)
          throw new Error(
            "File.__completeEditSession: an accepted file must not reuse the replaced file cacheKey",
          );
        l = c;
      }
    } catch (g) {
      ((d = !0), (h = g));
    }
    (e &&
      l != null &&
      ((this.file = l), a != null && (this.lineAnnotations = a)),
      (this.editSession = void 0));
    const { renderedFile: f, file: p } = this;
    if (
      (this.fileRenderer.endEditSession(p),
      f != null && p != null && f !== p && $(f, p) && (this.renderedFile = p),
      e && this.fileContainer != null && this.rerender(),
      d)
    )
      throw h;
  }
  applyDocumentChange(e, t) {
    const n = this.editSession?.file;
    if (n == null)
      throw new Error(
        "File.applyDocumentChange: requires an active edit session",
      );
    (this.fileRenderer.beginEditSession(n),
      this.fileRenderer.applyDocumentChange(e),
      t != null && this.syncEditSessionAnnotationsFromEditor(t));
  }
  updateRenderCache(e, t, n) {
    const i = this.editSession?.file;
    if (i == null)
      throw new Error(
        "File.updateRenderCache: requires an active edit session",
      );
    (this.fileRenderer.beginEditSession(i),
      this.fileRenderer.updateRenderCache(e, t, n?.lineCountChangeInFlight));
  }
  render({
    file: e,
    fileContainer: t,
    forceRender: n = !1,
    preventEmit: i = !1,
    containerWrapper: r,
    deferManagers: o = !1,
    lineAnnotations: s,
    renderRange: a,
  }) {
    if (!this.enabled)
      throw new Error(
        "File.render: attempting to call render after cleaned up",
      );
    this.editor?.__postponeBgTokenizeToNextFrame();
    const { collapsed: l = !1, themeType: d = "system" } = this.options,
      h = l ? void 0 : a,
      c = this.renderRange,
      u = this.hasThemeChanged(),
      f =
        s != null && (s.length > 0 || this.getLatestAnnotations().length > 0)
          ? this.isNewAnnotations(s)
          : !1,
      p = !$(this.file, e);
    p && this.updateExternalFile(e, s);
    const g = this.getLatestFile(e) ?? e;
    if (!l && !n && Vt(h, this.renderRange) && !p && !f && !u) {
      const S = this.applyCachedThemeState(d);
      return (S && this.finalizeRender(), S);
    }
    ((this.renderRange = h),
      p && (this.cachedHeaderHTML = void 0),
      this.fileRenderer.setOptions(Kn(this.options)),
      this.syncInteractionOptions(),
      s != null && this.setLineAnnotations(s),
      this.fileRenderer.setLineAnnotations(this.getLatestAnnotations()));
    const { disableErrorHandling: m = !1, disableFileHeader: y = !1 } =
      this.options;
    if (
      (y &&
        (this.headerElement != null &&
          (this.headerElement.remove(),
          (this.headerElement = void 0),
          (this.lastRenderedHeaderHTML = void 0)),
        this.clearHeaderSlots()),
      (t = this.getOrCreateFileContainerNode(t, r)),
      this.applyCachedThemeState(d),
      l)
    ) {
      (this.removeRenderedCode(), this.clearAuxiliaryNodes());
      try {
        const S = this.fileRenderer.renderFile(g, _i);
        (S != null &&
          this.applyThemeState(t, S.themeStyles, d, S.baseThemeType),
          S?.headerAST != null && this.applyHeaderToDOM(S.headerAST, t, S.file),
          (this.renderedFile = S?.file ?? g),
          this.injectUnsafeCSS());
      } catch (S) {
        if (m) throw S;
        (console.error(S), S instanceof Error && this.applyErrorToDOM(S, t));
      }
      return (this.finalizeRender(), i || this.emitPostRender(), !0);
    }
    try {
      const S = this.getOrCreatePreNode(t);
      if (
        !this.canPartiallyRender(n, f, p || u || !$(this.renderedFile, g)) ||
        !this.applyPartialRender(g, c, h)
      ) {
        const b = this.fileRenderer.renderFile(g, h);
        if (b == null)
          return (
            this.workerManager?.isInitialized() === !1 &&
              this.workerManager.isWorkingPool() &&
              this.workerManager
                .initialize()
                .catch(() => {})
                .then(() => this.rerender()),
            !1
          );
        (this.applyThemeState(t, b.themeStyles, d, b.baseThemeType),
          b.headerAST != null && this.applyHeaderToDOM(b.headerAST, t, b.file),
          this.applyFullRender(b, S),
          (this.renderedFile = b.file));
      }
      (this.applyBuffers(S, h),
        this.injectUnsafeCSS(),
        this.renderAnnotations(),
        this.renderGutterUtility(),
        (this.managersDirty = !0),
        o || this.flushManagers(),
        this.finalizeRender(),
        this.editor != null && this.syncRenderViewToEditor());
    } catch (S) {
      if (m) throw S;
      (console.error(S), S instanceof Error && this.applyErrorToDOM(S, t));
    }
    return (i || this.emitPostRender(), !0);
  }
  finalizeRender() {}
  emitPostRender(e = !1) {
    const {
      fileContainer: t,
      options: { onPostRender: n },
    } = this;
    if (e) {
      if (!this.mounted || ((this.mounted = !1), t == null)) return;
      n?.(t, this, "unmount");
      return;
    }
    if (t == null) return;
    const i = this.mounted ? "update" : "mount";
    ((this.mounted = !0), n?.(t, this, i));
  }
  removeRenderedCode() {
    (this.resizeManager.cleanUp(),
      this.interactionManager.cleanUp(),
      this.bufferBefore?.remove(),
      (this.bufferBefore = void 0),
      this.bufferAfter?.remove(),
      (this.bufferAfter = void 0),
      this.code?.remove(),
      (this.code = void 0),
      this.pre?.remove(),
      (this.pre = void 0),
      (this.appliedPreAttributes = void 0),
      (this.lastRowCount = void 0));
  }
  clearAuxiliaryNodes() {
    for (const { element: e } of this.annotationCache.values()) e.remove();
    (this.annotationCache.clear(),
      this.gutterUtilityContent?.remove(),
      (this.gutterUtilityContent = void 0));
  }
  canPartiallyRender(e, t, n) {
    return !(e || t || n);
  }
  renderPlaceholder(e) {
    if (this.fileContainer == null) return !1;
    if (
      (this.emitPostRender(!0),
      this.cleanChildNodes(),
      this.placeHolder == null)
    ) {
      const t =
        this.fileContainer.shadowRoot ??
        this.fileContainer.attachShadow({ mode: "open" });
      ((this.placeHolder = document.createElement("div")),
        (this.placeHolder.dataset.placeholder = ""),
        t.appendChild(this.placeHolder));
    }
    return (this.placeHolder.style.setProperty("height", `${e}px`), !0);
  }
  async primeHighlightCache(e = this.file) {
    const { workerManager: t } = this;
    if (
      e == null ||
      t == null ||
      !t.isWorkingPool() ||
      e.cacheKey == null ||
      Ht(e)
    )
      return;
    const n = this.options.tokenizeMaxLength ?? 1e5;
    this.fileRenderer.getOrCreateLineCache(e).length > n ||
      (await t.primeFileHighlightCache(e).catch((i) => {
        console.error(i);
      }));
  }
  cleanChildNodes() {
    (this.resizeManager.cleanUp(),
      this.interactionManager.cleanUp(),
      this.clearAuxiliaryNodes(),
      this.bufferAfter?.remove(),
      this.bufferBefore?.remove(),
      this.code?.remove(),
      this.errorWrapper?.remove(),
      this.headerElement?.remove(),
      this.headerPrefix?.remove(),
      this.headerFilenameSuffix?.remove(),
      this.headerMetadata?.remove(),
      this.headerCustom?.remove(),
      this.pre?.remove(),
      this.spriteSVG?.remove(),
      this.themeCSSStyle?.remove(),
      this.unsafeCSSStyle?.remove(),
      (this.bufferAfter = void 0),
      (this.bufferBefore = void 0),
      (this.code = void 0),
      (this.errorWrapper = void 0),
      (this.headerElement = void 0),
      (this.headerPrefix = void 0),
      (this.headerFilenameSuffix = void 0),
      (this.headerMetadata = void 0),
      (this.headerCustom = void 0),
      (this.pre = void 0),
      (this.spriteSVG = void 0),
      (this.themeCSSStyle = void 0),
      (this.appliedThemeCSS = void 0),
      (this.hasAdoptedThemeCSS = !1),
      (this.unsafeCSSStyle = void 0),
      (this.appliedUnsafeCSS = void 0),
      (this.lastRenderedHeaderHTML = void 0),
      (this.lastRowCount = void 0),
      (this.mounted = !1));
  }
  renderAnnotations() {
    if (this.isContainerManaged || this.fileContainer == null) {
      for (const { element: i } of this.annotationCache.values()) i.remove();
      this.annotationCache.clear();
      return;
    }
    const e = new Map(this.annotationCache),
      { renderAnnotation: t } = this.options,
      n = this.getLatestAnnotations();
    if (t != null && n.length > 0)
      for (const [i, r] of n.entries()) {
        const o = this.getAnnotationSlotName(r),
          s = `${i}-${o}`;
        let a = this.annotationCache.get(s);
        if (a == null || !ns(r, a.annotation)) {
          a?.element.remove();
          const l = t(r);
          if (l == null) continue;
          ((a = { element: wn(o), annotation: r }),
            a.element.appendChild(l),
            this.fileContainer.appendChild(a.element),
            this.annotationCache.set(s, a));
        }
        e.delete(s);
      }
    for (const [i, { element: r }] of e.entries())
      (this.annotationCache.delete(i), r.remove());
  }
  renderGutterUtility() {
    const { renderGutterUtility: e } = this.options;
    if (this.fileContainer == null || e == null) {
      (this.gutterUtilityContent?.remove(),
        (this.gutterUtilityContent = void 0));
      return;
    }
    const t = e(this.interactionManager.getHoveredLine);
    if (t != null && this.gutterUtilityContent != null) return;
    if (t == null) {
      (this.gutterUtilityContent?.remove(),
        (this.gutterUtilityContent = void 0));
      return;
    }
    const n = or();
    (n.appendChild(t),
      this.fileContainer.appendChild(n),
      (this.gutterUtilityContent = n));
  }
  injectUnsafeCSS() {
    const { unsafeCSS: e } = this.options,
      t = this.fileContainer?.shadowRoot;
    if (t != null) {
      if (e == null || e === "") {
        (this.unsafeCSSStyle != null &&
          (this.unsafeCSSStyle.remove(), (this.unsafeCSSStyle = void 0)),
          (this.appliedUnsafeCSS = void 0));
        return;
      }
      (this.unsafeCSSStyle?.parentNode === t && this.appliedUnsafeCSS === e) ||
        ((this.unsafeCSSStyle ??= sr()),
        this.unsafeCSSStyle.parentNode !== t &&
          t.appendChild(this.unsafeCSSStyle),
        (this.unsafeCSSStyle.textContent = Rn(e)),
        (this.appliedUnsafeCSS = e));
    }
  }
  applyThemeState(e, t, n, i) {
    const r = e.shadowRoot ?? e.attachShadow({ mode: "open" }),
      o = i ?? n,
      s = this.getTheme(),
      a = typeof s == "string" ? s : { ...s },
      l = Ke(r);
    if (
      this.themeCSSStyle?.parentNode === r &&
      this.appliedThemeCSS?.themeStyles === t &&
      this.appliedThemeCSS.themeType === o &&
      this.appliedThemeCSS.scrollbarGutter === l
    ) {
      this.appliedThemeCSS.theme = a;
      return;
    }
    if (this.hasAdoptedThemeCSS && this.themeCSSStyle?.parentNode === r) {
      ((this.hasAdoptedThemeCSS = !1),
        (this.appliedThemeCSS = {
          theme: a,
          themeStyles: t,
          themeType: o,
          baseThemeType: i,
          scrollbarGutter: l,
        }));
      return;
    }
    ((this.themeCSSStyle = Mn({
      shadowRoot: r,
      currentNode: this.themeCSSStyle,
      themeCSS: Tn(t, o, l),
    })),
      (this.appliedThemeCSS =
        this.themeCSSStyle != null
          ? {
              theme: a,
              themeStyles: t,
              themeType: o,
              baseThemeType: i,
              scrollbarGutter: l,
            }
          : void 0));
  }
  hydrateMeasuredScrollbar() {
    const e = this.fileContainer?.shadowRoot;
    e == null ||
      this.themeCSSStyle == null ||
      (this.themeCSSStyle.textContent = dr(
        this.themeCSSStyle.textContent ?? "",
        Ke(e),
      ));
  }
  shouldGuardRebuildScroll() {
    return this.editor != null && pr();
  }
  applyFullRender(e, t) {
    (this.cleanupErrorWrapper(), this.applyPreNodeAttributes(t, e));
    const n = (this.code = rt({ code: this.code })),
      i = this.fileRenderer.renderCodeAST(e);
    this.editor?.__captureFocusForDOMReplacement();
    const r = () => {
      if (n.childElementCount >= 2)
        for (let o = 0; o < 2; o++) {
          const s = n.children[o],
            a = i[o];
          ((s.innerHTML = J(a.children)),
            (s.style.cssText = a.properties.style));
        }
      else n.innerHTML = J(i);
      t.contains(n) || t.replaceChildren(n);
    };
    (this.shouldGuardRebuildScroll() ? dn(t, r) : r(),
      (this.lastRowCount = e.rowCount));
  }
  applyPartialRender(e, t, n) {
    if (t == null || n == null) return !1;
    const { code: i } = this,
      r = i != null ? this.getColumns(i) : void 0;
    if (i == null || r == null) return !1;
    const o = t.startingLine,
      s = n.startingLine,
      a = t.totalLines === 1 / 0 ? Number.POSITIVE_INFINITY : o + t.totalLines,
      l = n.totalLines === 1 / 0 ? Number.POSITIVE_INFINITY : s + n.totalLines,
      d = Math.max(o, s),
      h = Math.min(a, l);
    if (h <= d) return !1;
    if (
      !this.trimDOMToOverlap(r.gutter, d, h) ||
      !this.trimDOMToOverlap(r.content, d, h)
    )
      throw new Error("File.applyPartialRender: failed to trim to overlap");
    let { length: c } = r.content.children;
    const u = (m, y) => {
        if (!(y <= 0))
          return this.fileRenderer.renderFile(e, {
            startingLine: m,
            totalLines: y,
            bufferBefore: 0,
            bufferAfter: 0,
          });
      },
      f = s < d ? u(s, d - s) : void 0;
    if (f === void 0 && s < d) return !1;
    const p =
        l === Number.POSITIVE_INFINITY
          ? Number.POSITIVE_INFINITY
          : Math.max(0, l - h),
      g = l > h ? u(h, p) : void 0;
    return g === void 0 && l > h
      ? !1
      : (this.cleanupErrorWrapper(),
        f != null &&
          (r.gutter.insertAdjacentHTML(
            "afterbegin",
            this.fileRenderer.renderPartialHTML(f.gutterAST),
          ),
          r.content.insertAdjacentHTML(
            "afterbegin",
            this.fileRenderer.renderPartialHTML(f.contentAST),
          ),
          (c += f.rowCount)),
        g != null &&
          (r.gutter.insertAdjacentHTML(
            "beforeend",
            this.fileRenderer.renderPartialHTML(g.gutterAST),
          ),
          r.content.insertAdjacentHTML(
            "beforeend",
            this.fileRenderer.renderPartialHTML(g.contentAST),
          ),
          (c += g.rowCount)),
        this.lastRowCount !== c &&
          (r.gutter.style.setProperty("grid-row", `span ${c}`),
          r.content.style.setProperty("grid-row", `span ${c}`),
          (this.lastRowCount = c)),
        !0);
  }
  getColumns(e) {
    const t = e.children[0],
      n = e.children[1];
    if (!(
      !(t instanceof HTMLElement) ||
      !(n instanceof HTMLElement) ||
      t.dataset.gutter == null ||
      n.dataset.content == null
    ))
      return { gutter: t, content: n };
  }
  trimDOMToOverlap(e, t, n) {
    const i = this.getDOMBoundaryIndices(e, [t, n]),
      r = i.get(t) ?? e.children.length,
      o = i.get(n) ?? e.children.length;
    if (r > o) return !1;
    for (let s = e.children.length - 1; s >= o; s -= 1) e.children[s]?.remove();
    for (let s = r - 1; s >= 0; s -= 1) e.children[s]?.remove();
    return !0;
  }
  getDOMBoundaryIndices(e, t) {
    const n = [...new Set(t)].sort((a, l) => a - l),
      i = new Map();
    if (n.length === 0) return i;
    let r = 0,
      o = n[r];
    const { children: s } = e;
    o === 0 && (i.set(0, 0), (r += 1), (o = n[r]));
    for (let a = 0; a < s.length; a += 1) {
      const l = s[a];
      if (!(l instanceof HTMLElement)) continue;
      const d = this.getLineIndexFromDOMNode(l);
      if (d != null) {
        for (; o != null && d >= o;) (i.set(o, a), (r += 1), (o = n[r]));
        if (r >= n.length) break;
      }
    }
    for (const a of n) i.has(a) || i.set(a, s.length);
    return i;
  }
  getLineIndexFromDOMNode(e) {
    const t = e.dataset.lineIndex;
    if (t == null) return;
    const n = Number(t);
    return Number.isNaN(n) ? void 0 : n;
  }
  applyBuffers(e, t) {
    if (t == null || this.shouldDisableVirtualizationBuffers()) {
      (this.bufferBefore != null &&
        (this.bufferBefore.remove(), (this.bufferBefore = void 0)),
        this.bufferAfter != null &&
          (this.bufferAfter.remove(), (this.bufferAfter = void 0)));
      return;
    }
    (t.bufferBefore > 0
      ? (this.bufferBefore == null &&
          ((this.bufferBefore = document.createElement("div")),
          (this.bufferBefore.dataset.virtualizerBuffer = "before"),
          e.before(this.bufferBefore)),
        this.bufferBefore.style.setProperty("height", `${t.bufferBefore}px`),
        this.bufferBefore.style.setProperty("contain", "strict"))
      : this.bufferBefore != null &&
        (this.bufferBefore.remove(), (this.bufferBefore = void 0)),
      t.bufferAfter > 0
        ? (this.bufferAfter == null &&
            ((this.bufferAfter = document.createElement("div")),
            (this.bufferAfter.dataset.virtualizerBuffer = "after"),
            e.after(this.bufferAfter)),
          this.bufferAfter.style.setProperty("height", `${t.bufferAfter}px`),
          this.bufferAfter.style.setProperty("contain", "strict"))
        : this.bufferAfter != null &&
          (this.bufferAfter.remove(), (this.bufferAfter = void 0)));
  }
  shouldDisableVirtualizationBuffers() {
    return this.options.disableVirtualizationBuffers ?? !1;
  }
  applyHeaderToDOM(e, t, n) {
    (this.cleanupErrorWrapper(),
      this.placeHolder?.remove(),
      (this.placeHolder = void 0));
    const i = this.cachedHeaderHTML ?? J(e);
    if (((this.cachedHeaderHTML = i), i !== this.lastRenderedHeaderHTML)) {
      const l = document.createElement("div");
      l.innerHTML = i;
      const d = l.firstElementChild;
      if (!(d instanceof HTMLElement)) return;
      (this.headerElement != null
        ? t.shadowRoot?.replaceChild(d, this.headerElement)
        : t.shadowRoot?.prepend(d),
        (this.headerElement = d),
        (this.lastRenderedHeaderHTML = i));
    }
    if (this.isContainerManaged) return;
    const {
      renderHeaderPrefix: r,
      renderHeaderFilenameSuffix: o,
      renderCustomHeader: s,
      renderHeaderMetadata: a,
    } = this.options;
    if (s != null) {
      const l = s(n) ?? void 0;
      ((this.headerCustom = this.upsertHeaderSlotElement(
        t,
        this.headerCustom,
        vn,
        l,
      )),
        this.headerPrefix?.remove(),
        this.headerFilenameSuffix?.remove(),
        this.headerMetadata?.remove(),
        (this.headerPrefix = void 0),
        (this.headerFilenameSuffix = void 0),
        (this.headerMetadata = void 0));
    } else {
      const l = r?.(n) ?? void 0,
        d = o?.(n) ?? void 0,
        h = a?.(n) ?? void 0;
      ((this.headerPrefix = this.upsertHeaderSlotElement(
        t,
        this.headerPrefix,
        Cn,
        l,
      )),
        (this.headerFilenameSuffix = this.upsertHeaderSlotElement(
          t,
          this.headerFilenameSuffix,
          bn,
          d,
        )),
        (this.headerMetadata = this.upsertHeaderSlotElement(
          t,
          this.headerMetadata,
          yn,
          h,
        )),
        this.headerCustom?.remove(),
        (this.headerCustom = void 0));
    }
  }
  clearHeaderSlots() {
    (this.headerPrefix?.remove(),
      this.headerFilenameSuffix?.remove(),
      this.headerMetadata?.remove(),
      this.headerCustom?.remove(),
      (this.headerPrefix = void 0),
      (this.headerFilenameSuffix = void 0),
      (this.headerMetadata = void 0),
      (this.headerCustom = void 0));
  }
  upsertHeaderSlotElement(e, t, n, i) {
    if (i == null) {
      t?.remove();
      return;
    }
    const r = t ?? this.createHeaderSlotElement(n);
    return (
      t == null && e.appendChild(r),
      this.replaceHeaderSlotContent(r, i),
      r
    );
  }
  replaceHeaderSlotContent(e, t) {
    (e.replaceChildren(),
      t instanceof Element ? e.appendChild(t) : (e.innerText = `${t}`));
  }
  createHeaderSlotElement(e) {
    const t = document.createElement("div");
    return ((t.slot = e), t);
  }
  getOrCreateFileContainerNode(e, t) {
    const { fileContainer: n } = this,
      i = e ?? n ?? document.createElement("diffs-container"),
      r = n !== i;
    return (
      n != null && r && this.editor?.__captureFocusForDOMReplacement(),
      r && this.emitPostRender(!0),
      (this.fileContainer = i),
      n != null &&
        r &&
        ((this.lastRenderedHeaderHTML = void 0), (this.headerElement = void 0)),
      t != null &&
        this.fileContainer.parentNode !== t &&
        t.appendChild(this.fileContainer),
      r && this.adoptReusableShellElements(this.fileContainer),
      this.ensureSpriteSVG(this.fileContainer),
      this.fileContainer
    );
  }
  adoptReusableShellElements(e) {
    const { shadowRoot: t } = e;
    if (t != null)
      for (const n of t.children)
        n instanceof SVGElement
          ? (this.spriteSVG ??= n)
          : at(n) && n.hasAttribute("data-theme-css")
            ? ((this.themeCSSStyle ??= n), (this.hasAdoptedThemeCSS = !0))
            : at(n) &&
              n.hasAttribute("data-unsafe-css") &&
              ((this.unsafeCSSStyle ??= n),
              (this.appliedUnsafeCSS ??= this.options.unsafeCSS ?? void 0));
  }
  ensureSpriteSVG(e) {
    const t = e.shadowRoot ?? e.attachShadow({ mode: "open" });
    if (this.spriteSVG == null) {
      const n = document.createElement("div");
      n.innerHTML = ir;
      const i = n.firstChild;
      i instanceof SVGElement && (this.spriteSVG = i);
    }
    this.spriteSVG != null &&
      this.spriteSVG.parentNode !== t &&
      t.appendChild(this.spriteSVG);
  }
  getOrCreatePreNode(e) {
    const t = e.shadowRoot ?? e.attachShadow({ mode: "open" });
    return (
      this.pre == null
        ? ((this.pre = document.createElement("pre")),
          (this.appliedPreAttributes = void 0),
          (this.code = void 0),
          t.appendChild(this.pre))
        : this.pre.parentNode !== t &&
          (this.editor?.__captureFocusForDOMReplacement(),
          e.shadowRoot?.appendChild(this.pre),
          (this.appliedPreAttributes = void 0)),
      this.placeHolder?.remove(),
      (this.placeHolder = void 0),
      this.pre
    );
  }
  syncCodeNodeFromPre(e) {
    this.code = void 0;
    for (const t of Array.from(e.children))
      if (t instanceof HTMLElement && t.hasAttribute("data-code")) {
        this.code = t;
        return;
      }
  }
  applyPreNodeAttributes(e, { totalLines: t }) {
    const { overflow: n = "scroll", disableLineNumbers: i = !1 } = this.options,
      r = {
        type: "file",
        split: !1,
        overflow: n,
        disableLineNumbers: i,
        diffIndicators: "none",
        disableBackground: !0,
        totalLines: t,
      };
    rr(r, this.appliedPreAttributes) ||
      (In(e, r), (this.appliedPreAttributes = r));
  }
  applyErrorToDOM(e, t) {
    (this.cleanupErrorWrapper(),
      this.pre?.remove(),
      (this.pre = void 0),
      (this.appliedPreAttributes = void 0));
    const n = t.shadowRoot ?? t.attachShadow({ mode: "open" });
    ((this.errorWrapper ??= document.createElement("div")),
      (this.errorWrapper.dataset.errorWrapper = ""),
      (this.errorWrapper.textContent = ""),
      n.appendChild(this.errorWrapper));
    const i = document.createElement("div");
    ((i.dataset.errorMessage = ""),
      (i.innerText = e.message),
      this.errorWrapper.appendChild(i));
    const r = document.createElement("pre");
    ((r.dataset.errorStack = ""),
      (r.innerText = e.stack ?? "No Error Stack"),
      this.errorWrapper.appendChild(r));
  }
  cleanupErrorWrapper() {
    (this.errorWrapper?.remove(), (this.errorWrapper = void 0));
  }
};
function fs(e, t, n = !1) {
  return !n && e == null && t != null;
}
function ps(e, t, n = !1) {
  return e == null && t != null && !n;
}
function Mt(e) {
  return { ...Bi, ...e };
}
function ue(e, t) {
  const n = gs(e, t);
  return t ? n : e.diffHeaderHeight + n;
}
function gs(e, t) {
  return e.paddingTop ?? (t ? e.spacing : 0);
}
function Ut(e) {
  return e.paddingBottom ?? e.spacing;
}
function ms(e) {
  switch (e) {
    case "simple":
      return 4;
    case "metadata":
    case "line-info":
    case "line-info-basic":
    case "custom":
      return 32;
  }
}
const Ss = 5e3,
  vt = new Map();
let vs = -1;
function Cs(e, t) {
  return (
    (e.overflow ?? "scroll") !== (t.overflow ?? "scroll") ||
    (e.collapsed ?? !1) !== (t.collapsed ?? !1) ||
    (e.disableLineNumbers ?? !1) !== (t.disableLineNumbers ?? !1) ||
    (e.disableFileHeader ?? !1) !== (t.disableFileHeader ?? !1) ||
    e.unsafeCSS !== t.unsafeCSS
  );
}
var bs = class extends us {
  virtualizer;
  metrics;
  __id = `virtualized-file:${++vs}`;
  renderType = "virtualized";
  top;
  height = 0;
  cache = {
    heights: new Map(),
    checkpoints: [],
    fileAnnotationHeight: 0,
    ghostTextRows: vt,
  };
  pendingRender;
  isVisible = !1;
  isSetup = !1;
  layoutDirty = !0;
  forceRenderOverride;
  currentCollapsed;
  constructor(e, t, n = Bi, i, r = !1) {
    (super(e, i, r), (this.virtualizer = t), (this.metrics = n));
  }
  setMetrics(e, t = !1) {
    const n = Mt(e);
    (!t && qe(this.metrics, n)) ||
      ((this.metrics = n), this.resetLayoutCache());
  }
  setLineAnnotations(e) {
    this.syncLineAnnotations(e) && this.resetLayoutCache();
  }
  syncLineAnnotations(e) {
    return e == null ||
      !this.isNewAnnotations(e) ||
      (e.length === 0 && this.getLatestAnnotations().length === 0)
      ? !1
      : (super.setLineAnnotations(e), !0);
  }
  syncEditSessionAnnotationsFromEditor(e) {
    return super.syncEditSessionAnnotationsFromEditor(e)
      ? (this.resetLayoutCache(), !0)
      : !1;
  }
  hasLineAnnotations() {
    return this.getLatestAnnotations().some((e) => e.lineNumber > 0);
  }
  hasUniformLineHeights() {
    const { overflow: e = "scroll" } = this.options;
    return (
      e === "scroll" &&
      !this.hasLineAnnotations() &&
      this.cache.heights.size === 0
    );
  }
  getLineHeight(e, t = !1) {
    const n = this.cache.heights.get(e);
    if (n != null) return n;
    const i = t ? 2 : 1;
    return this.metrics.lineHeight * i;
  }
  setOptions(e) {
    if (this.isAdvancedMode())
      throw new Error(
        "VirtualizedFile.setOptions cannot be used inside CodeView. Update CodeView options instead.",
      );
    if (e == null) return;
    const { options: t } = this,
      n = !Ln(t, e),
      i = Cs(t, e);
    (super.setOptions(e),
      i && this.resetLayoutCache(!0),
      n && (this.forceRenderOverride = !0),
      n && this.virtualizer.instanceChanged(this, i));
  }
  setThemeType(e) {
    if (this.isAdvancedMode())
      throw new Error(
        "VirtualizedFile.setThemeType cannot be used inside CodeView. Update CodeView options instead.",
      );
    super.setThemeType(e);
  }
  resetLayoutCache(e = !1, t = !0) {
    ((this.layoutDirty = !0),
      (this.cache.fileAnnotationHeight = 0),
      this.cache.heights.size > 0 && this.cache.heights.clear(),
      this.cache.checkpoints.length > 0 && (this.cache.checkpoints.length = 0),
      (this.cache.ghostTextRows = vt),
      this.renderRange != null && t && (this.renderRange = void 0),
      e && this.isSimpleMode() && this.computeApproximateSize());
  }
  applyGhostTextRows(e) {
    const { heights: t, ghostTextRows: n } = this.cache;
    if (n === e) return !1;
    const { lineHeight: i } = this.metrics;
    let r = !1;
    for (const o of new Set([...n.keys(), ...e.keys()])) {
      const s = n.get(o) ?? 0,
        a = e.get(o) ?? 0;
      if (a === s) continue;
      const l = (t.get(o) ?? i) - s * i + a * i;
      (l === i ? t.delete(o) : t.set(o, l), (r = !0));
    }
    return ((this.cache.ghostTextRows = e), r);
  }
  reconcileHeights() {
    let e = !1;
    if (this.fileContainer == null || this.getLayoutFile() == null)
      return (this.height !== 0 && (e = !0), (this.height = 0), e);
    const { overflow: t = "scroll" } = this.options;
    this.top = this.getVirtualizedTop();
    const n = this.editor?.__getGhostTextRows() ?? vt;
    if (
      ((e = this.applyGhostTextRows(n)),
      t === "scroll" &&
        this.getLatestAnnotations().length === 0 &&
        !this.isResizeDebuggingEnabled())
    )
      return (e && this.computeApproximateSize(!0), e);
    const { lineHeight: i } = this.metrics;
    if (this.code == null) return e;
    const r = this.code.children[1];
    if (!(r instanceof HTMLElement)) return e;
    const o = an(this.getLatestAnnotations());
    if (this.renderRange != null && o && $t(this.renderRange)) {
      const s = ys(r) ?? 0;
      s !== this.cache.fileAnnotationHeight &&
        ((this.cache.fileAnnotationHeight = s), (e = !0));
    } else
      !o &&
        this.cache.fileAnnotationHeight !== 0 &&
        ((this.cache.fileAnnotationHeight = 0), (e = !0));
    for (const s of r.children) {
      if (!(s instanceof HTMLElement)) continue;
      const a = s.dataset.lineIndex;
      if (a == null) continue;
      const l = Number(a);
      let d = s.getBoundingClientRect().height + (n.get(l) ?? 0) * i,
        h = !1;
      s.nextElementSibling instanceof HTMLElement &&
        ("lineAnnotation" in s.nextElementSibling.dataset ||
          "noNewline" in s.nextElementSibling.dataset) &&
        ("noNewline" in s.nextElementSibling.dataset && (h = !0),
        (d += s.nextElementSibling.getBoundingClientRect().height));
      const c = this.getLineHeight(l, h);
      d !== c &&
        ((e = !0),
        d === this.metrics.lineHeight * (h ? 2 : 1)
          ? this.cache.heights.delete(l)
          : this.cache.heights.set(l, d));
    }
    return (
      (e || this.isResizeDebuggingEnabled()) && this.computeApproximateSize(!0),
      e
    );
  }
  onRender = (e) =>
    this.fileContainer == null || this.file == null
      ? !1
      : (e && (this.top = this.getVirtualizedTop()),
        this.render({ file: this.file }));
  updateCodeViewLayout(e, t, n, i) {
    const r = !$(this.file, e);
    r && this.updateExternalFile(e, i);
    const {
      pendingRenderFile: o,
      layoutFileChanged: s,
      renderedFileChanged: a,
      annotationsChanged: l,
    } = this.updatePendingRender(e, i);
    let d = n?.resetFileLayoutCache === !0 || s || l;
    n?.metrics != null && ((this.metrics = n.metrics), (d = !0));
    const { collapsed: h = !1 } = this.options;
    return (
      this.currentCollapsed !== h && ((this.currentCollapsed = h), (d = !0)),
      d && this.resetLayoutCache(),
      r && (this.layoutDirty = !0),
      !this.forceRenderOverride &&
        (r || a || l) &&
        (this.forceRenderOverride = !0),
      (this.top = t),
      this.computeApproximateSize(!1, o),
      this.height
    );
  }
  getLayoutFile() {
    return this.pendingRender?.file ?? this.getRenderedFile();
  }
  getLinePosition(e) {
    const t = this.getLayoutFile();
    if (t == null || e < 1) return;
    const { disableFileHeader: n = !1, collapsed: i = !1 } = this.options,
      r = this.fileRenderer.getLineCount(t) - 1;
    let o = ue(this.metrics, n);
    if (i || r < 0) return { top: o, height: 0 };
    const s = Math.min(Math.max(e - 1, 0), r),
      { lineHeight: a } = this.metrics;
    if (((o += this.cache.fileAnnotationHeight), this.hasUniformLineHeights()))
      return { top: o + s * a, height: a };
    const l = this.getLayoutCheckpointBeforeLineIndex(s);
    o = l?.top ?? o;
    for (let d = l?.lineIndex ?? 0; d < s; d++) o += this.getLineHeight(d, !1);
    return { top: o, height: this.getLineHeight(s, !1) };
  }
  getEditorViewport() {
    return this.virtualizer.type === "simple"
      ? this.virtualizer.getRoot()
      : this.virtualizer.getContainerElement();
  }
  getNumericScrollAnchor(e) {
    const t = this.getLayoutFile();
    if (t == null || this.renderRange == null) return;
    const { disableFileHeader: n = !1, collapsed: i = !1 } = this.options;
    if (i || this.renderRange.totalLines <= 0) return;
    const r = this.fileRenderer.getLineCount(t) - 1;
    if (r < 0) return;
    const o = ue(this.metrics, n),
      s = Math.min(this.renderRange.startingLine, r),
      a = Math.min(s + this.renderRange.totalLines - 1, r);
    if (a < s) return;
    const { fileAnnotationHeight: l } = this.cache;
    if (this.hasUniformLineHeights()) {
      const { lineHeight: h } = this.metrics,
        c = o + (s === 0 ? l : this.renderRange.bufferBefore),
        u = s + Math.max(Math.ceil((e - c) / h), 0);
      return u > a ? void 0 : { lineNumber: u + 1, top: o + l + u * h };
    }
    let d = o + (s === 0 ? l : this.renderRange.bufferBefore);
    for (let h = s; h <= a; h++) {
      if (d >= e) return { lineNumber: h + 1, top: d };
      d += this.getLineHeight(h);
    }
  }
  getVirtualizedHeight() {
    return this.height;
  }
  getAdvancedStickySpecs(e) {
    const t = this.getLayoutFile();
    if (this.top == null || t == null) return;
    if (this.options.collapsed === !0)
      return { topOffset: this.top, height: this.height };
    const n =
      e != null
        ? this.computeRenderRangeFromWindow(t, this.top, e)
        : this.renderRange;
    if (n == null) return;
    const { bufferBefore: i, bufferAfter: r, totalLines: o } = n;
    let s = 0;
    if (o === 0) {
      const a = e ?? this.virtualizer.getWindowSpecs();
      this.top < a.top && (s = r);
    }
    return { topOffset: this.top + i + s, height: this.height - (i + r) };
  }
  cleanUp(e = !1) {
    const t = this.cache.ghostTextRows.size > 0;
    t && (this.layoutDirty = !0);
    const n =
      e &&
      this.isAdvancedMode() &&
      this.fileContainer != null &&
      (t || !$(this.getRenderedFile(), this.getLatestFile()));
    (this.fileContainer != null &&
      this.isSimpleMode() &&
      this.getSimpleVirtualizer()?.disconnect(this.fileContainer),
      e || this.resetLayoutCache(),
      (this.pendingRender = void 0),
      (this.isSetup = !1),
      super.cleanUp(e),
      n && this.virtualizer.instanceChanged(this, !0));
  }
  computeApproximateSize(e = !1, t = this.getLayoutFile()) {
    const n = this.isResizeDebuggingEnabled();
    if (!e && !this.layoutDirty && !n) return;
    const i = this.height === 0;
    if (((this.height = 0), (this.cache.checkpoints = []), t == null)) {
      this.layoutDirty = !1;
      return;
    }
    this.applyGhostTextRows(this.editor?.__getGhostTextRows() ?? vt);
    const { disableFileHeader: r = !1, collapsed: o = !1 } = this.options,
      { lineHeight: s } = this.metrics,
      a = this.fileRenderer.getLineCount(t),
      l = ue(this.metrics, r),
      d = Ut(this.metrics);
    if (((this.height += l), o)) {
      this.layoutDirty = !1;
      return;
    }
    if (
      ((this.height += this.cache.fileAnnotationHeight),
      this.hasUniformLineHeights())
    )
      this.height += a * s;
    else
      for (let h = 0; h < a; h++)
        (this.addLayoutCheckpoint(h, this.height),
          (this.height += this.getLineHeight(h, !1)));
    if ((a > 0 && (this.height += d), this.fileContainer != null && n && !i)) {
      const h = this.fileContainer.getBoundingClientRect();
      h.height !== this.height
        ? console.log(
            "VirtualizedFile.computeApproximateSize: computed height doesnt match",
            {
              name: t.name,
              elementHeight: h.height,
              computedHeight: this.height,
            },
          )
        : console.log(
            "VirtualizedFile.computeApproximateSize: computed height IS CORRECT",
          );
    }
    this.layoutDirty = !1;
  }
  setVisibility(e) {
    this.isAdvancedMode() ||
      this.fileContainer == null ||
      ((this.renderRange = void 0),
      e && !this.isVisible
        ? ((this.top = this.getVirtualizedTop()), (this.isVisible = !0))
        : !e && this.isVisible && ((this.isVisible = !1), this.rerender()));
  }
  rerender() {
    if (!this.enabled || this.file == null) return;
    const e = this.getLatestFile(),
      t = e == null ? void 0 : this.fileRenderer.getFileForNextRender(e);
    (this.isSimpleMode() && (this.pendingRender = void 0),
      (this.forceRenderOverride = !0),
      this.virtualizer.instanceChanged(this, !$(this.getRenderedFile(), t)));
  }
  syncGhostTextRows() {
    const e = this.getAdvancedVirtualizer();
    e != null
      ? (e.capturePendingLayoutAnchor(),
        (this.layoutDirty = !0),
        e.instanceChanged(this, !0))
      : this.getSimpleVirtualizer()?.requestHeightReconcile(this);
  }
  applyDocumentChange(e, t, n = !1) {
    const { renderRange: i } = this;
    (this.getAdvancedVirtualizer()?.capturePendingLayoutAnchor(),
      super.applyDocumentChange(e, t),
      this.getSimpleVirtualizer()?.markDOMDirty(),
      this.resetLayoutCache(this.isSimpleMode(), !1));
    const r = this.getRenderedFile();
    if (!this.isSimpleMode()) this.computeApproximateSize(!0);
    else if (n && i != null && r != null) {
      const o = this.virtualizer.getWindowSpecs(),
        s = this.computeRenderRangeFromWindow(r, this.top ?? 0, o);
      s.bufferAfter !== i.bufferAfter && this.updateBuffers(s);
    }
    ((this.forceRenderOverride = !0),
      this.virtualizer.instanceChanged(this, !0));
  }
  render({
    fileContainer: e,
    file: t,
    forceRender: n = !1,
    lineAnnotations: i,
    ...r
  }) {
    const o = !$(this.file, t);
    o && (this.updateExternalFile(t, i), (this.cachedHeaderHTML = void 0));
    const {
        pendingRenderFile: s,
        layoutFileChanged: a,
        renderedFileChanged: l,
        annotationsChanged: d,
      } = this.pendingRender != null &&
      this.pendingRender.latestFile === (this.getLatestFile(t) ?? t)
        ? {
            pendingRenderFile: this.pendingRender.file,
            layoutFileChanged: !1,
            renderedFileChanged: !1,
            annotationsChanged: !1,
          }
        : this.updatePendingRender(t, i),
      { forceRenderOverride: h, isSetup: c } = this;
    if (
      ((this.forceRenderOverride = void 0),
      (d || a) && this.resetLayoutCache(),
      (e = this.getOrCreateFileContainerNode(e)),
      c)
    )
      ((this.top ??= this.getVirtualizedTop()),
        a &&
          this.isSimpleMode() &&
          (this.getSimpleVirtualizer()?.markDOMDirty(),
          this.resetLayoutCache(!1),
          this.computeApproximateSize(!1, s)));
    else {
      this.computeApproximateSize(!1, s);
      const g = this.getSimpleVirtualizer();
      if (((this.top ??= this.getVirtualizedTop()), this.isAdvancedMode()))
        this.isVisible = !0;
      else {
        if (g == null)
          throw new Error(
            "VirtualizedFile.render: simple virtualizer is not available",
          );
        (g.connect(e, this),
          (this.isVisible = g.isInstanceVisible(this.top ?? 0, this.height)));
      }
      this.isSetup = !0;
    }
    if (!this.isVisible && this.isSimpleMode() && (!o || !c))
      return (
        (this.pendingRender = void 0),
        this.renderPlaceholder(this.height)
      );
    const u = this.virtualizer.getWindowSpecs(),
      f = this.top ?? 0,
      p = this.computeRenderRangeFromWindow(s, f, u);
    return super.render({
      file: t,
      fileContainer: e,
      renderRange: p,
      lineAnnotations: i,
      forceRender: (h ?? n) || d || l || o,
      ...r,
    });
  }
  finalizeRender() {
    if (this.getRenderedFile() !== this.pendingRender?.file)
      throw new Error(
        "VirtualizedFile.render: rendered a different file than its prepared layout",
      );
    ((this.pendingRender = void 0),
      this.isSimpleMode() &&
        this.getSimpleVirtualizer()?.requestHeightReconcile(this));
  }
  updatePendingRender(e, t) {
    const n = this.getLatestFile(e) ?? e,
      i = this.getRenderedFile(),
      r = this.pendingRender?.file ?? i,
      o = this.fileRenderer.getFileForNextRender(n);
    return (
      (this.pendingRender = { latestFile: n, file: o }),
      {
        pendingRenderFile: o,
        annotationsChanged: this.syncLineAnnotations(t),
        layoutFileChanged: !$(r, o),
        renderedFileChanged: !$(i, o),
      }
    );
  }
  syncVirtualizedTop() {
    this.top = this.getVirtualizedTop();
  }
  shouldDisableVirtualizationBuffers() {
    return this.isAdvancedMode() || super.shouldDisableVirtualizationBuffers();
  }
  shouldGuardRebuildScroll() {
    return !1;
  }
  isSimpleMode() {
    return this.virtualizer.type === "simple";
  }
  isAdvancedMode() {
    return this.virtualizer.type === "advanced";
  }
  addLayoutCheckpoint(e, t) {
    e % Ss === 0 && this.cache.checkpoints.push({ lineIndex: e, top: t });
  }
  getLayoutCheckpointBeforeLineIndex(e) {
    if (e <= 0 || this.cache.checkpoints.length === 0) return;
    let t = 0,
      n = this.cache.checkpoints.length - 1,
      i;
    for (; t <= n;) {
      const r = (t + n) >> 1,
        o = this.cache.checkpoints[r];
      if (o == null)
        throw new Error("VirtualizedFile: invalid checkpoint index");
      o.lineIndex <= e ? ((i = o), (t = r + 1)) : (n = r - 1);
    }
    return i;
  }
  getLayoutCheckpointBeforeTop(e, t) {
    let n = 0,
      i = this.cache.checkpoints.length - 1,
      r = -1;
    for (; n <= i;) {
      const o = (n + i) >> 1,
        s = this.cache.checkpoints[o];
      if (s == null)
        throw new Error("VirtualizedFile: invalid checkpoint index");
      s.top <= e ? ((r = o), (n = o + 1)) : (i = o - 1);
    }
    if (t == null) return r >= 0 ? this.cache.checkpoints[r] : void 0;
    for (let o = r; o >= 0; o--) {
      const s = this.cache.checkpoints[o];
      if (s == null)
        throw new Error("VirtualizedFile: invalid checkpoint index");
      if (s.lineIndex % t === 0) return s;
    }
  }
  getVirtualizedTop() {
    return this.virtualizer.type === "advanced"
      ? this.virtualizer.getLocalTopForInstance(this)
      : this.fileContainer != null
        ? this.virtualizer.getOffsetInScrollContainer(this.fileContainer)
        : 0;
  }
  getSimpleVirtualizer() {
    return this.virtualizer.type === "simple" ? this.virtualizer : void 0;
  }
  getAdvancedVirtualizer() {
    return this.virtualizer.type === "advanced" ? this.virtualizer : void 0;
  }
  isResizeDebuggingEnabled() {
    return this.getSimpleVirtualizer()?.config.resizeDebugging ?? !1;
  }
  computeRenderRangeFromWindow(e, t, { top: n, bottom: i }) {
    const { disableFileHeader: r = !1 } = this.options,
      { hunkLineCount: o, lineHeight: s } = this.metrics,
      a = this.fileRenderer.getLineCount(e),
      l = this.height,
      d = ue(this.metrics, r),
      h = a > 0 ? Ut(this.metrics) : 0,
      { fileAnnotationHeight: c } = this.cache,
      u = d + c,
      f = Math.max(0, l - d - c - h),
      p = an(this.getLatestAnnotations()),
      g = t + d,
      m = c > 0 && p && g < i && g + c > n;
    if (t < n - l || t > i)
      return {
        startingLine: 0,
        totalLines: 0,
        bufferBefore: 0,
        bufferAfter: l - d - h,
      };
    if (a <= o)
      return {
        startingLine: 0,
        totalLines: o,
        bufferBefore: 0,
        bufferAfter: 0,
      };
    const y = Math.ceil(Math.max(i - n, 0) / s),
      S = Math.ceil(y / o) * o + o,
      b = S / o,
      L = (n + i) / 2;
    if (this.hasUniformLineHeights()) {
      const V = t + u,
        te = V + f;
      if (!m && !(V < i && te > n))
        return {
          startingLine: 0,
          totalLines: 0,
          bufferBefore: 0,
          bufferAfter: l - d - h,
        };
      const ie = Math.floor(m && L < t + u ? 0 : (L - (t + u)) / s),
        re = Math.floor(ie / o) - Math.floor(b / 2),
        Ie = Math.ceil(a / o),
        K = Math.max(0, Math.min(re, Ie)) * o,
        _e = re < 0 ? S + re * o : S,
        Xe = K === 0 ? 0 : c + K * s,
        pe = Math.min(_e, a - K);
      return {
        startingLine: K,
        totalLines: _e,
        bufferBefore: Xe,
        bufferAfter: Math.max(0, (a - K - pe) * s),
      };
    }
    const v = b,
      k = [],
      C = this.getLayoutCheckpointBeforeTop(Math.max(0, n - t - S * s * 2), o);
    let x = t + (C?.top ?? u),
      A = C?.lineIndex ?? 0,
      w,
      O,
      T;
    const F = C?.lineIndex ?? 0;
    for (let V = F; V < a; V++) {
      const te = A % o === 0,
        ie = Math.floor(A / o);
      if (te && ((k[ie] = x - (t + u)), T != null)) {
        if (T <= 0) break;
        T--;
      }
      const re = this.getLineHeight(V, !1);
      (x > n - re && x < i && (w ??= ie),
        x + re > L && (O ??= ie),
        T == null && x >= i && te && (T = v),
        A++,
        (x += re));
    }
    if (w == null)
      if (m) ((w = 0), (O = 0));
      else
        return {
          startingLine: 0,
          totalLines: 0,
          bufferBefore: 0,
          bufferAfter: l - d - h,
        };
    O ??= w;
    const H = Math.round(O - b / 2),
      N = Math.max(0, Math.ceil(a / o) - b),
      E = Math.max(0, Math.min(H, N)),
      I = E * o,
      D = H < 0 ? S + H * o : S,
      G = k[E] ?? 0,
      M = I === 0 ? 0 : c + G,
      P = E + D / o,
      z = P < k.length ? f - k[P] : f - (x - t - u);
    return {
      startingLine: I,
      totalLines: D,
      bufferBefore: M,
      bufferAfter: Math.max(0, z),
    };
  }
};
function ys(e) {
  let t;
  for (const n of e.children)
    n instanceof HTMLElement &&
      n.dataset.lineAnnotation === er &&
      (t = Math.max(t ?? 0, n.getBoundingClientRect().height));
  return t;
}
function B(e, t) {
  return e?.cacheKey != null || t?.cacheKey != null
    ? e?.cacheKey === t?.cacheKey
    : e === t;
}
function ot(e, t) {
  return (
    e?.cacheKey === t?.cacheKey &&
    e?.contents === t?.contents &&
    e?.name === t?.name &&
    e?.lang === t?.lang
  );
}
function gr(e) {
  return e.map((t) => ({
    ...t,
    hunkContent: t.hunkContent.map((n) => ({ ...n })),
  }));
}
function mr(e) {
  return {
    ...e,
    hunks: gr(e.hunks),
    deletionLines: [...e.deletionLines],
    additionLines: [...e.additionLines],
  };
}
function lt(e) {
  return e !== "" ? e.split(Vr) : [];
}
function hn(e, t, n) {
  const i = e === "clone" ? mr(t) : t;
  if (!i.isPartial)
    throw new Error("hydratePartialDiff: fileDiff must be partial");
  switch (i.type) {
    case "change":
    case "rename-changed":
      return Ls(i, ks(i, n), Xn(i, n));
    case "rename-pure": {
      const r = Xn(i, n);
      ws(i, n);
      const o = lt(r.contents);
      return (
        (i.isPartial = !1),
        (i.deletionLines = o),
        (i.additionLines = o),
        Sr(i, null, r),
        i
      );
    }
  }
  throw new Error(
    `hydratePartialDiff: ${i.type} diffs cannot be hydrated from loaded files`,
  );
}
function Ls(e, t, n) {
  const i = lt(t.contents),
    r = lt(n.contents),
    {
      hunks: o,
      splitLineCount: s,
      unifiedLineCount: a,
    } = xs(e.hunks, r.length);
  return (
    (e.hunks = o),
    (e.splitLineCount = s),
    (e.unifiedLineCount = a),
    (e.isPartial = !1),
    (e.deletionLines = i),
    (e.additionLines = r),
    Sr(e, t, n),
    e
  );
}
function xs(e, t) {
  let n = 0,
    i = 0,
    r = 0;
  const o = [];
  for (const s of e) {
    const a = Math.max(s.additionStart - 1, 0),
      l = Math.max(s.deletionStart - 1, 0);
    let d = a,
      h = l,
      c = 0,
      u = 0,
      f = 0,
      p = 0;
    const g = [];
    for (const y of s.hunkContent) {
      if (y.type === "context") {
        (g.push({ ...y, additionLineIndex: d, deletionLineIndex: h }),
          (d += y.lines),
          (h += y.lines),
          (f += y.lines),
          (p += y.lines));
        continue;
      }
      (g.push({ ...y, additionLineIndex: d, deletionLineIndex: h }),
        (d += y.additions),
        (h += y.deletions),
        (c += y.additions),
        (u += y.deletions),
        (f += Math.max(y.additions, y.deletions)),
        (p += y.additions + y.deletions));
    }
    const m = Math.max(xe(s.additionStart, s.additionCount) - r, 0);
    (o.push({
      ...s,
      collapsedBefore: m,
      additionLineIndex: a,
      deletionLineIndex: l,
      additionLines: c,
      deletionLines: u,
      hunkContent: g,
      splitLineStart: n + m,
      unifiedLineStart: i + m,
      splitLineCount: f,
      unifiedLineCount: p,
    }),
      (n += m + f),
      (i += m + p),
      (r = ee(s.additionStart, s.additionCount)));
  }
  if (o.length > 0) {
    const s = o[o.length - 1],
      a = ee(s.additionStart, s.additionCount),
      l = Math.max(t - a, 0);
    ((n += l), (i += l));
  }
  return { hunks: o, splitLineCount: n, unifiedLineCount: i };
}
function ks(e, t) {
  if (t.oldFile == null)
    throw new Error(
      `hydratePartialDiff: ${e.type} diff for ${e.name} requires oldFile`,
    );
  return t.oldFile;
}
function Xn(e, t) {
  if (t.newFile == null)
    throw new Error(
      `hydratePartialDiff: ${e.type} diff for ${e.name} requires newFile`,
    );
  return t.newFile;
}
function ws(e, t) {
  if (t.oldFile !== null)
    throw new Error(
      `hydratePartialDiff: ${e.type} diff for ${e.name} requires oldFile to be null`,
    );
}
function Es(e, t, n) {
  return e.cacheKey != null ? `${e.cacheKey}:hydrated` : Rs(t, n);
}
function Sr(e, t, n) {
  const i = Es(e, t, n);
  if (i == null) {
    delete e.cacheKey;
    return;
  }
  e.cacheKey = i;
}
function Rs(e, t) {
  return e != null && t != null
    ? e.cacheKey != null && t.cacheKey != null
      ? $r("hydrated-files", e.cacheKey, t.cacheKey)
      : void 0
    : (e?.cacheKey ?? t?.cacheKey);
}
class Ts extends Vi {}
const As = new Ts();
function Is(e, t, n) {
  return As.diff(e, t, n);
}
const Yn =
  "a-zA-Z0-9_\\u{AD}\\u{C0}-\\u{D6}\\u{D8}-\\u{F6}\\u{F8}-\\u{2C6}\\u{2C8}-\\u{2D7}\\u{2DE}-\\u{2FF}\\u{1E00}-\\u{1EFF}";
class Hs extends Vi {
  tokenize(t) {
    const n = new RegExp(`(\\r?\\n)|[${Yn}]+|[^\\S\\n\\r]+|[^${Yn}]`, "ug");
    return t.match(n) || [];
  }
}
const Ms = new Hs();
function Ds(e, t, n) {
  return Ms.diff(e, t, n);
}
var Fs = class {
  isDeletionsScrolling = !1;
  isAdditionsScrolling = !1;
  timeoutId = -1;
  codeDeletions;
  codeAdditions;
  enabled = !1;
  cleanUp() {
    this.enabled &&
      (this.codeDeletions?.removeEventListener(
        "scroll",
        this.handleDeletionsScroll,
      ),
      this.codeAdditions?.removeEventListener(
        "scroll",
        this.handleAdditionsScroll,
      ),
      clearTimeout(this.timeoutId),
      (this.codeDeletions = void 0),
      (this.codeAdditions = void 0),
      (this.enabled = !1));
  }
  setup(e, t, n) {
    if (t == null || n == null)
      for (const i of e.children ?? [])
        i instanceof HTMLElement &&
          ("deletions" in i.dataset
            ? (t = i)
            : "additions" in i.dataset && (n = i));
    if (n == null || t == null) {
      this.cleanUp();
      return;
    }
    (this.codeDeletions !== t &&
      (this.codeDeletions?.removeEventListener(
        "scroll",
        this.handleDeletionsScroll,
      ),
      (this.codeDeletions = t),
      t.addEventListener("scroll", this.handleDeletionsScroll, {
        passive: !0,
      })),
      this.codeAdditions !== n &&
        (this.codeAdditions?.removeEventListener(
          "scroll",
          this.handleAdditionsScroll,
        ),
        (this.codeAdditions = n),
        n.addEventListener("scroll", this.handleAdditionsScroll, {
          passive: !0,
        })),
      (this.enabled = !0));
  }
  handleDeletionsScroll = () => {
    this.isAdditionsScrolling ||
      ((this.isDeletionsScrolling = !0),
      clearTimeout(this.timeoutId),
      (this.timeoutId = setTimeout(() => {
        this.isDeletionsScrolling = !1;
      }, 300)),
      this.codeAdditions?.scrollTo({ left: this.codeDeletions?.scrollLeft }));
  };
  handleAdditionsScroll = () => {
    this.isDeletionsScrolling ||
      ((this.isAdditionsScrolling = !0),
      clearTimeout(this.timeoutId),
      (this.timeoutId = setTimeout(() => {
        this.isAdditionsScrolling = !1;
      }, 300)),
      this.codeDeletions?.scrollTo({ left: this.codeAdditions?.scrollLeft }));
  };
};
function Se(e, t) {
  return (
    Ue(e.theme, t.theme) &&
    e.useTokenTransformer === t.useTokenTransformer &&
    e.tokenizeMaxLineLength === t.tokenizeMaxLineLength &&
    e.lineDiffType === t.lineDiffType &&
    e.maxLineDiffLength === t.maxLineDiffLength
  );
}
function Ct(e) {
  return R({
    tagName: "div",
    properties: {
      "data-content-buffer": "",
      "data-buffer-size": e,
      style: `grid-row: span ${e};min-height:calc(${e} * 1lh)`,
    },
  });
}
function bt(e) {
  return R({
    tagName: "div",
    children: [
      R({ tagName: "span", children: [X("No newline at end of file")] }),
    ],
    properties: {
      "data-no-newline": "",
      "data-line-type": e,
      "data-column-content": "",
    },
  });
}
function Qt(e) {
  return R({
    tagName: "div",
    children: [
      Ot({
        name: e === "both" ? "diffs-icon-expand-all" : "diffs-icon-expand",
        properties: { "data-icon": "" },
      }),
    ],
    properties: {
      role: "button",
      "data-expand-button": "",
      "data-expand-both": e === "both" ? "" : void 0,
      "data-expand-up": e === "up" ? "" : void 0,
      "data-expand-down": e === "down" ? "" : void 0,
    },
  });
}
function Be({
  type: e,
  content: t,
  expandIndex: n,
  chunked: i = !1,
  slotName: r,
  isFirstHunk: o,
  isLastHunk: s,
}) {
  let a = 0;
  const l = [];
  if (
    (e === "metadata" &&
      t != null &&
      l.push(
        R({
          tagName: "div",
          children: [X(t)],
          properties: { "data-separator-wrapper": "" },
        }),
      ),
    (e === "line-info" || e === "line-info-basic") && t != null)
  ) {
    const d = [];
    (n != null &&
      (i
        ? (o || (d.push(Qt("up")), a++), s || (d.push(Qt("down")), a++))
        : (d.push(Qt(!o && !s ? "both" : o ? "down" : "up")), a++)),
      d.push(
        R({
          tagName: "div",
          children: [
            R({
              tagName: "span",
              children: [X(t)],
              properties: { "data-unmodified-lines": "" },
            }),
          ],
          properties: { "data-separator-content": "" },
        }),
      ),
      i &&
        n != null &&
        d.push(
          R({
            tagName: "div",
            children: [X("Expand all")],
            properties: {
              role: "button",
              "data-expand-button": "",
              "data-expand-all-button": "",
            },
          }),
        ),
      l.push(
        R({
          tagName: "div",
          children: d,
          properties: {
            "data-separator-wrapper": "",
            "data-separator-multi-button": a > 1 ? "" : void 0,
          },
        }),
      ));
  }
  return (
    e === "custom" &&
      r != null &&
      l.push(R({ tagName: "slot", properties: { name: r } })),
    R({
      tagName: "div",
      children: l,
      properties: {
        "data-separator": l.length === 0 ? "simple" : e,
        "data-expand-index": n,
        "data-separator-first": o ? "" : void 0,
        "data-separator-last": s ? "" : void 0,
      },
    })
  );
}
function Ps(e, t) {
  return `hunk-separator-${e}-${t}`;
}
function Os(e) {
  const t = e.at(-1);
  return t == null
    ? 0
    : Math.max(
        ee(t.additionStart, t.additionCount),
        ee(t.deletionStart, t.deletionCount),
      );
}
function Qn({ line: e, spanStart: t, spanLength: n }) {
  return {
    start: { line: e, character: t },
    end: { line: e, character: t + n },
    properties: { "data-diff-span": "" },
    alwaysWrap: !0,
  };
}
function yt({
  item: e,
  arr: t,
  enableJoin: n,
  isNeutral: i = !1,
  isLastItem: r = !1,
}) {
  const o = t[t.length - 1];
  if (o == null || r || !n) {
    t.push([i ? 0 : 1, e.value]);
    return;
  }
  const s = o[0] === 0;
  if (i === s || (i && e.value.length === 1 && !s)) {
    o[1] += e.value;
    return;
  }
  t.push([i ? 0 : 1, e.value]);
}
function dt(e) {
  return [
    xe(e.additionStart, e.additionCount) + 1,
    ee(e.additionStart, e.additionCount) + 1,
  ];
}
function ke({
  isPartial: e,
  rangeSize: t,
  expandedHunks: n,
  hunkIndex: i,
  collapsedContextThreshold: r,
}) {
  const o = Math.max(t, 0);
  if (o === 0 || e)
    return {
      fromStart: 0,
      fromEnd: 0,
      rangeSize: o,
      collapsedLines: o,
      renderAll: !1,
    };
  if (n === !0 || o <= r)
    return {
      fromStart: o,
      fromEnd: 0,
      rangeSize: o,
      collapsedLines: 0,
      renderAll: !0,
    };
  const s = n?.get(i),
    a = Math.min(Math.max(s?.fromStart ?? 0, 0), o),
    l = Math.min(Math.max(s?.fromEnd ?? 0, 0), o),
    d = a + l,
    h = d >= o;
  return {
    fromStart: h ? o : a,
    fromEnd: h ? 0 : l,
    rangeSize: o,
    collapsedLines: Math.max(o - d, 0),
    renderAll: h,
  };
}
function Ns(e) {
  const t = e.hunks[e.hunks.length - 1];
  if (
    t == null ||
    e.isPartial ||
    e.additionLines.length === 0 ||
    e.deletionLines.length === 0
  )
    return !1;
  const n = e.additionLines.length - ee(t.additionStart, t.additionCount),
    i = e.deletionLines.length - ee(t.deletionStart, t.deletionCount);
  return n <= 0 && i <= 0 ? !1 : n !== i;
}
function vr({ fileDiff: e, errorPrefix: t }) {
  const n = e.hunks[e.hunks.length - 1];
  if (
    n == null ||
    e.isPartial ||
    e.additionLines.length === 0 ||
    e.deletionLines.length === 0
  )
    return 0;
  const i = e.additionLines.length - ee(n.additionStart, n.additionCount),
    r = e.deletionLines.length - ee(n.deletionStart, n.deletionCount);
  if (i <= 0 && r <= 0) return 0;
  if (i !== r)
    throw new Error(
      `${t}: trailing context mismatch (additions=${i}, deletions=${r}) for ${e.name}`,
    );
  return Math.min(i, r);
}
function we({
  fileDiff: e,
  hunkIndex: t,
  expandedHunks: n,
  collapsedContextThreshold: i,
  errorPrefix: r,
}) {
  if (t !== e.hunks.length - 1) return;
  const o = vr({ fileDiff: e, errorPrefix: r });
  if (o <= 0) return;
  if (n === !0 || o <= i)
    return {
      fromStart: o,
      fromEnd: 0,
      rangeSize: o,
      collapsedLines: 0,
      renderAll: !0,
    };
  const s = n?.get(e.hunks.length),
    a = Math.min(Math.max(s?.fromStart ?? 0, 0), o);
  return {
    fromStart: a,
    fromEnd: 0,
    rangeSize: o,
    collapsedLines: o - a,
    renderAll: a >= o,
  };
}
function Cr({
  fileDiff: e,
  lineNumber: t,
  expandedHunks: n,
  collapsedContextThreshold: i,
}) {
  if (n === !0 || e.isPartial) return !0;
  for (const [a, l] of e.hunks.entries()) {
    const [d, h] = dt(l);
    if (t < d) {
      const c = ke({
          isPartial: e.isPartial,
          rangeSize: l.collapsedBefore,
          expandedHunks: n,
          hunkIndex: a,
          collapsedContextThreshold: i,
        }),
        u = d - c.rangeSize;
      return c.renderAll || t < u + c.fromStart || t >= d - c.fromEnd;
    }
    if (t < h) return !0;
  }
  const r = we({
    fileDiff: e,
    hunkIndex: e.hunks.length - 1,
    expandedHunks: n,
    collapsedContextThreshold: i,
    errorPrefix: "isAdditionLineRenderable",
  });
  if (r == null || r.renderAll) return !0;
  const o = e.hunks[e.hunks.length - 1],
    [, s] = dt(o);
  return t < s + r.fromStart || t >= s + r.rangeSize;
}
function zs({
  fileDiff: e,
  lineNumber: t,
  direction: n,
  expandedHunks: i,
  collapsedContextThreshold: r,
}) {
  if (i === !0 || e.isPartial) return t;
  const o = [];
  let s = 1;
  for (const [l, d] of e.hunks.entries()) {
    const [h, c] = dt(d),
      u = ke({
        isPartial: e.isPartial,
        rangeSize: d.collapsedBefore,
        expandedHunks: i,
        hunkIndex: l,
        collapsedContextThreshold: r,
      }),
      f = h - u.rangeSize;
    (u.renderAll
      ? o.push([f, h])
      : (u.fromStart > 0 && o.push([f, f + u.fromStart]),
        u.fromEnd > 0 && o.push([h - u.fromEnd, h])),
      o.push([h, c]),
      (s = c));
  }
  const a = we({
    fileDiff: e,
    hunkIndex: e.hunks.length - 1,
    expandedHunks: i,
    collapsedContextThreshold: r,
    errorPrefix: "getNearestRenderableAdditionLine",
  });
  if (a != null) {
    const l = s;
    ((s = l + a.rangeSize),
      a.renderAll
        ? o.push([l, s])
        : a.fromStart > 0 && o.push([l, l + a.fromStart]));
  }
  if (t >= s) return t;
  if (n === "down") {
    for (const [l, d] of o) if (d > t) return Math.max(l, t);
    return;
  }
  for (let l = o.length - 1; l >= 0; l--) {
    const [d, h] = o[l];
    if (d <= t) return Math.min(h - 1, t);
  }
}
function br({ type: e, metrics: t }) {
  return t.hunkSeparatorHeight ?? ms(e);
}
function yr({ type: e, metrics: t }) {
  return e === "simple" || e === "metadata" || e === "line-info-basic"
    ? 0
    : t.spacing;
}
function Us({ type: e, hunkIndex: t, hunkSpecs: n }) {
  switch (e) {
    case "simple":
      return t > 0;
    case "metadata":
      return n != null;
    case "line-info":
    case "line-info-basic":
    case "custom":
      return !0;
  }
}
function _s(e) {
  return e !== "simple" && e !== "metadata";
}
function nt({ type: e, metrics: t, hunkIndex: n, hunkSpecs: i }) {
  if (!Us({ type: e, hunkIndex: n, hunkSpecs: i })) return;
  const r = br({ type: e, metrics: t }),
    o = yr({ type: e, metrics: t }),
    s = n > 0 ? o : 0,
    a = o;
  return { height: r, gapBefore: s, gapAfter: a, totalHeight: s + r + a };
}
function Me({ type: e, metrics: t }) {
  if (!_s(e)) return;
  const n = br({ type: e, metrics: t }),
    i = yr({ type: e, metrics: t });
  return { height: n, gapBefore: i, gapAfter: 0, totalHeight: i + n };
}
function Fe({
  diff: e,
  diffStyle: t,
  startingLine: n = 0,
  totalLines: i = 1 / 0,
  expandedHunks: r,
  collapsedContextThreshold: o = 1,
  callback: s,
}) {
  const a = Bs({
      diff: e,
      diffStyle: t,
      startingLine: n,
      expandedHunks: r,
      collapsedContextThreshold: o,
    }),
    l = {
      viewportStart: n,
      viewportEnd: n + i,
      isWindowedHighlight: n > 0 || i < 1 / 0,
      splitCount: a.splitCount,
      unifiedCount: a.unifiedCount,
      finalHunkIndex: e.hunks.length - 1,
      shouldBreak() {
        if (!l.isWindowedHighlight) return !1;
        const d = l.unifiedCount >= n + i,
          h = l.splitCount >= n + i;
        return t === "unified" ? d : (t === "split" || d) && h;
      },
      shouldSkip(d, h) {
        if (!l.isWindowedHighlight) return !1;
        const c = d > 0 && l.unifiedCount + d <= n,
          u = h > 0 && l.splitCount + h <= n;
        return t === "unified" ? c : (t === "split" || c) && u;
      },
      incrementCounts(d, h) {
        ((t === "unified" || t === "both") && (l.unifiedCount += d),
          (t === "split" || t === "both") && (l.splitCount += h));
      },
      isInWindow(d, h) {
        if (!l.isWindowedHighlight) return !0;
        const c = l.isInUnifiedWindow(d),
          u = l.isInSplitWindow(h);
        return t === "unified" ? c : t === "split" ? u : c || u;
      },
      isInUnifiedWindow(d) {
        return (
          !l.isWindowedHighlight ||
          (l.unifiedCount >= n - d && l.unifiedCount < n + i)
        );
      },
      isInSplitWindow(d) {
        return (
          !l.isWindowedHighlight ||
          (l.splitCount >= n - d && l.splitCount < n + i)
        );
      },
      emit(d, h = !1) {
        return (
          h ||
            (t === "unified"
              ? l.incrementCounts(1, 0)
              : t === "split"
                ? l.incrementCounts(0, 1)
                : l.incrementCounts(1, 1)),
          s(d) ?? !1
        );
      },
    };
  e: for (let d = a.hunkIndex; d < e.hunks.length; d++) {
    let S = function (T, F) {
        return m == null || m.collapsedLines <= 0 || m.fromStart + m.fromEnd > 0
          ? 0
          : t === "unified"
            ? T === h.unifiedLineStart + h.unifiedLineCount - 1
              ? m.collapsedLines
              : 0
            : F === h.splitLineStart + h.splitLineCount - 1
              ? m.collapsedLines
              : 0;
      },
      L = function () {
        return b ? 0 : ((b = !0), g.collapsedLines);
      };
    const h = e.hunks[d];
    if (h == null) throw new Error("iterateOverDiff: invalid hunk index");
    if (l.shouldBreak()) break;
    const c = xe(h.deletionStart, h.deletionCount),
      u = xe(h.additionStart, h.additionCount),
      f = !e.isPartial && h.deletionCount === 0 ? c : h.deletionLineIndex,
      p = !e.isPartial && h.additionCount === 0 ? u : h.additionLineIndex,
      g = ke({
        isPartial: e.isPartial,
        rangeSize: h.collapsedBefore,
        expandedHunks: r,
        hunkIndex: d,
        collapsedContextThreshold: o,
      }),
      m =
        d === l.finalHunkIndex
          ? we({
              fileDiff: e,
              hunkIndex: d,
              expandedHunks: r,
              collapsedContextThreshold: o,
              errorPrefix: "iterateOverDiff",
            })
          : void 0,
      y = g.fromStart + g.fromEnd;
    let b = g.collapsedLines === 0;
    if (l.shouldSkip(y, y)) (l.incrementCounts(y, y), L());
    else {
      let T = h.unifiedLineStart - g.rangeSize,
        F = h.splitLineStart - g.rangeSize,
        H = f - g.rangeSize,
        N = p - g.rangeSize,
        E = c + 1 - g.rangeSize,
        I = u + 1 - g.rangeSize;
      if (
        Lt(l, g.fromStart, t, (D) =>
          l.emit({
            hunkIndex: d,
            hunk: h,
            collapsedBefore: 0,
            collapsedAfter: 0,
            type: "context-expanded",
            deletionLine: {
              lineNumber: E + D,
              lineIndex: H + D,
              noEOFCR: !1,
              unifiedLineIndex: T + D,
              splitLineIndex: F + D,
            },
            additionLine: {
              unifiedLineIndex: T + D,
              splitLineIndex: F + D,
              lineIndex: N + D,
              lineNumber: I + D,
              noEOFCR: !1,
            },
          }),
        ) ||
        ((T = h.unifiedLineStart - g.fromEnd),
        (F = h.splitLineStart - g.fromEnd),
        (H = f - g.fromEnd),
        (N = p - g.fromEnd),
        (E = c + 1 - g.fromEnd),
        (I = u + 1 - g.fromEnd),
        Lt(
          l,
          g.fromEnd,
          t,
          (D) =>
            l.emit({
              hunkIndex: d,
              hunk: h,
              collapsedBefore: L(),
              collapsedAfter: 0,
              type: "context-expanded",
              deletionLine: {
                lineNumber: E + D,
                lineIndex: H + D,
                noEOFCR: !1,
                unifiedLineIndex: T + D,
                splitLineIndex: F + D,
              },
              additionLine: {
                unifiedLineIndex: T + D,
                splitLineIndex: F + D,
                lineIndex: N + D,
                lineNumber: I + D,
                noEOFCR: !1,
              },
            }),
          () => {
            L();
          },
        ))
      )
        break e;
    }
    let v = h.unifiedLineStart,
      k = h.splitLineStart,
      C = f,
      x = p,
      A = c + 1,
      w = u + 1;
    const O = h.hunkContent.at(-1);
    for (const T of h.hunkContent) {
      if (l.shouldBreak()) break e;
      const F = T === O;
      if (T.type === "context") {
        if (l.shouldSkip(T.lines, T.lines))
          (l.incrementCounts(T.lines, T.lines), L());
        else if (
          Lt(
            l,
            T.lines,
            t,
            (H) => {
              const N = F && H === T.lines - 1,
                E = v + H,
                I = k + H;
              return l.emit({
                hunkIndex: d,
                hunk: h,
                collapsedBefore: L(),
                collapsedAfter: S(E, I),
                type: "context",
                deletionLine: {
                  lineNumber: A + H,
                  lineIndex: C + H,
                  noEOFCR: N && h.noEOFCRDeletions,
                  unifiedLineIndex: E,
                  splitLineIndex: I,
                },
                additionLine: {
                  unifiedLineIndex: E,
                  splitLineIndex: I,
                  lineIndex: x + H,
                  lineNumber: w + H,
                  noEOFCR: N && h.noEOFCRAdditions,
                },
              });
            },
            () => {
              L();
            },
          )
        )
          break e;
        ((v += T.lines),
          (k += T.lines),
          (C += T.lines),
          (x += T.lines),
          (A += T.lines),
          (w += T.lines));
      } else {
        const H = Math.max(T.deletions, T.additions),
          N = T.deletions + T.additions;
        if (!l.shouldSkip(N, H)) {
          const E = Gs(l, T, t);
          (E[0]?.[0] ?? 0) > 0 && L();
          for (const [I, D] of E)
            for (let G = I; G < D; G++) {
              const M = S(
                v + G,
                t === "unified"
                  ? k + (G < T.deletions ? G : G - T.deletions)
                  : k + G,
              );
              if (
                l.emit(
                  Ws({
                    hunkIndex: d,
                    hunk: h,
                    collapsedBefore: L(),
                    collapsedAfter: M,
                    diffStyle: t,
                    index: G,
                    unifiedLineIndex: v,
                    splitLineIndex: k,
                    additionLineIndex: x,
                    deletionLineIndex: C,
                    additionLineNumber: w,
                    deletionLineNumber: A,
                    content: T,
                    isLastContent: F,
                    unifiedCount: N,
                    splitCount: H,
                  }),
                  !0,
                )
              )
                break e;
            }
        }
        (L(),
          l.incrementCounts(N, H),
          (v += N),
          (k += H),
          (C += T.deletions),
          (x += T.additions),
          (A += T.deletions),
          (w += T.additions));
      }
    }
    if (m != null) {
      const { collapsedLines: T, fromStart: F, fromEnd: H } = m,
        N = F + H;
      if (
        Lt(
          l,
          N,
          t,
          (E) => {
            const I = E === N - 1;
            return l.emit({
              hunkIndex: e.hunks.length,
              hunk: void 0,
              collapsedBefore: 0,
              collapsedAfter: I ? T : 0,
              type: "context-expanded",
              deletionLine: {
                lineNumber: A + E,
                lineIndex: C + E,
                noEOFCR: !1,
                unifiedLineIndex: v + E,
                splitLineIndex: k + E,
              },
              additionLine: {
                unifiedLineIndex: v + E,
                splitLineIndex: k + E,
                lineIndex: x + E,
                lineNumber: w + E,
                noEOFCR: !1,
              },
            });
          },
          void 0,
          () => l.shouldBreak(),
        )
      )
        break e;
    }
  }
}
function Bs({
  diff: e,
  diffStyle: t,
  startingLine: n,
  expandedHunks: i,
  collapsedContextThreshold: r,
}) {
  if (n <= 0 || t === "both")
    return { hunkIndex: 0, splitCount: 0, unifiedCount: 0 };
  const o = Vs({ diff: e, expandedHunks: i, collapsedContextThreshold: r });
  let s = 0,
    a = e.hunks.length - 1,
    l = e.hunks.length;
  for (; s <= a;) {
    const h = (s + a) >> 1,
      c = o[h + 1];
    if (c == null)
      throw new Error("iterateOverDiff: invalid hunk prefix index");
    (t === "unified" ? c.unifiedCount : c.splitCount) > n
      ? ((l = h), (a = h - 1))
      : (s = h + 1);
  }
  if (l >= e.hunks.length) {
    const h = o[e.hunks.length];
    if (h == null)
      throw new Error("iterateOverDiff: invalid terminal hunk prefix index");
    return {
      hunkIndex: e.hunks.length,
      splitCount: h.splitCount,
      unifiedCount: h.unifiedCount,
    };
  }
  const d = o[l];
  if (d == null)
    throw new Error("iterateOverDiff: invalid selected hunk prefix index");
  return {
    hunkIndex: l,
    splitCount: d.splitCount,
    unifiedCount: d.unifiedCount,
  };
}
function Vs({ diff: e, expandedHunks: t, collapsedContextThreshold: n }) {
  let i = 0,
    r = 0;
  const o = e.hunks.length - 1,
    s = [{ splitCount: 0, unifiedCount: 0 }];
  for (let a = 0; a < e.hunks.length; a++) {
    const l = e.hunks[a];
    if (l == null)
      throw new Error("iterateOverDiff: invalid hunk summary index");
    const d = ke({
        isPartial: e.isPartial,
        rangeSize: l.collapsedBefore,
        expandedHunks: t,
        hunkIndex: a,
        collapsedContextThreshold: n,
      }),
      h = d.fromStart + d.fromEnd;
    ((i += h + l.splitLineCount), (r += h + l.unifiedLineCount));
    const c =
      a === o
        ? we({
            fileDiff: e,
            hunkIndex: a,
            expandedHunks: t,
            collapsedContextThreshold: n,
            errorPrefix: "iterateOverDiff",
          })
        : void 0;
    if (c != null) {
      const u = c.fromStart + c.fromEnd;
      ((i += u), (r += u));
    }
    s.push({ splitCount: i, unifiedCount: r });
  }
  return s;
}
function $s(e, t, n) {
  if (!e.isWindowedHighlight || t <= 0) return [0, t];
  const i = [];
  function r(a) {
    const l = Math.max(0, e.viewportStart - a),
      d = Math.min(t, e.viewportEnd - a);
    d > l && i.push([l, d]);
  }
  if (
    (n !== "split" && r(e.unifiedCount),
    n !== "unified" && r(e.splitCount),
    i.length === 0)
  )
    return [0, 0];
  let o = i[0][0],
    s = i[0][1];
  for (let a = 1; a < i.length; a++) {
    const l = i[a];
    ((o = Math.min(o, l[0])), (s = Math.max(s, l[1])));
  }
  return [o, s];
}
function Lt(e, t, n, i, r, o) {
  const [s, a] = $s(e, t, n);
  s > 0 && (e.incrementCounts(s, s), r?.());
  let l = s;
  for (; l < t;) {
    if (o?.() === !0) return !0;
    if (l >= a) {
      e.incrementCounts(t - l, t - l);
      break;
    }
    if (e.isInWindow(0, 0)) {
      if (i(l) === !0) return !0;
    } else e.incrementCounts(1, 1);
    l++;
  }
  return !1;
}
function Gs(e, t, n) {
  if (!e.isWindowedHighlight)
    return [
      [
        0,
        n === "unified"
          ? t.deletions + t.additions
          : Math.max(t.deletions, t.additions),
      ],
    ];
  const i = n !== "split",
    r = n !== "unified",
    o = n === "unified" ? "unified" : "split",
    s = [];
  function a(c, u) {
    if (c + u <= e.viewportStart || c >= e.viewportEnd) return;
    const f = Math.max(0, e.viewportStart - c),
      p = Math.min(u, e.viewportEnd - c);
    return p > f ? [f, p] : void 0;
  }
  function l(c, u) {
    return o === "split"
      ? c
      : u === "additions"
        ? [c[0] + t.deletions, c[1] + t.deletions]
        : c;
  }
  function d(c, u) {
    if (c == null) return;
    const [f, p] = l(c, u);
    p > f && s.push([f, p]);
  }
  if (
    (i &&
      (d(a(e.unifiedCount, t.deletions), "deletions"),
      d(a(e.unifiedCount + t.deletions, t.additions), "additions")),
    r &&
      (d(a(e.splitCount, t.deletions), "deletions"),
      d(a(e.splitCount, t.additions), "additions")),
    s.length === 0)
  )
    return s;
  s.sort((c, u) => c[0] - u[0]);
  const h = [s[0]];
  for (const [c, u] of s.slice(1)) {
    const f = h[h.length - 1];
    c <= f[1] ? (f[1] = Math.max(f[1], u)) : h.push([c, u]);
  }
  return h;
}
function Ws({
  hunkIndex: e,
  hunk: t,
  collapsedAfter: n,
  collapsedBefore: i,
  diffStyle: r,
  index: o,
  unifiedLineIndex: s,
  splitLineIndex: a,
  additionLineIndex: l,
  deletionLineIndex: d,
  additionLineNumber: h,
  deletionLineNumber: c,
  content: u,
  isLastContent: f,
  unifiedCount: p,
  splitCount: g,
}) {
  const m = o < u.deletions ? s + o : void 0,
    y =
      r === "unified"
        ? o >= u.deletions
          ? s + o
          : void 0
        : o < u.additions
          ? s + u.deletions + o
          : void 0,
    S = r === "unified" ? a + (o < u.deletions ? o : o - u.deletions) : a + o,
    b = o < u.deletions ? d + o : void 0,
    L = o < u.deletions ? c + o : void 0,
    v =
      r === "unified"
        ? o >= u.deletions
          ? l + (o - u.deletions)
          : void 0
        : o < u.additions
          ? l + o
          : void 0,
    k =
      r === "unified"
        ? o >= u.deletions
          ? h + (o - u.deletions)
          : void 0
        : o < u.additions
          ? h + o
          : void 0,
    C =
      r === "unified"
        ? f && o === u.deletions - 1 && t.noEOFCRDeletions
        : f && o === g - 1 && t.noEOFCRDeletions,
    x =
      r === "unified"
        ? f && o === p - 1 && t.noEOFCRAdditions
        : f && o === g - 1 && t.noEOFCRAdditions,
    A =
      b != null && L != null && m != null
        ? {
            lineNumber: L,
            lineIndex: b,
            noEOFCR: C,
            unifiedLineIndex: m,
            splitLineIndex: S,
          }
        : void 0,
    w =
      v != null && k != null && y != null
        ? {
            unifiedLineIndex: y,
            splitLineIndex: S,
            lineIndex: v,
            lineNumber: k,
            noEOFCR: x,
          }
        : void 0;
  if (A == null && w != null)
    return {
      type: "change",
      hunkIndex: e,
      hunk: t,
      collapsedAfter: n,
      collapsedBefore: i,
      deletionLine: void 0,
      additionLine: w,
    };
  if (A != null && w == null)
    return {
      type: "change",
      hunkIndex: e,
      hunk: t,
      collapsedAfter: n,
      collapsedBefore: i,
      deletionLine: A,
      additionLine: void 0,
    };
  if (A == null || w == null)
    throw new Error("iterateOverDiff: missing change line data");
  return {
    type: "change",
    hunkIndex: e,
    hunk: t,
    collapsedAfter: n,
    collapsedBefore: i,
    deletionLine: A,
    additionLine: w,
  };
}
const qs = { forcePlainText: !1 };
function js(
  e,
  t,
  n,
  {
    forcePlainText: i,
    startingLine: r,
    totalLines: o,
    expandedHunks: s,
    collapsedContextThreshold: a = 1,
  } = qs,
) {
  i ? ((r ??= 0), (o ??= 1 / 0)) : ((r = 0), (o = 1 / 0));
  const l = r > 0 || o < 1 / 0,
    d = typeof n.theme == "string" ? t.getTheme(n.theme).type : void 0,
    h = kn({ theme: n.theme, highlighter: t }),
    c =
      i && !l && (e.unifiedLineCount > 1e3 || e.splitLineCount > 1e3)
        ? "none"
        : n.lineDiffType,
    u = { deletionLines: [], additionLines: [] },
    { maxLineDiffLength: f } = n,
    p = !i && !e.isPartial,
    g = i ? s : void 0,
    m = new Map();
  function y(b) {
    const L = p ? 0 : b,
      v = m.get(L) ?? Xs();
    return (m.set(L, v), v);
  }
  function S(b, L, v, k) {
    if (l) {
      let C = v.at(-1);
      ((C == null || C.targetIndex + C.count !== L) &&
        ((C = { targetIndex: L, originalOffset: k.length, count: 0 }),
        v.push(C)),
        C.count++);
    }
    k.push(b);
  }
  Fe({
    diff: e,
    diffStyle: "both",
    startingLine: r,
    totalLines: o,
    expandedHunks: l ? g : !0,
    collapsedContextThreshold: a,
    callback: ({ hunkIndex: b, additionLine: L, deletionLine: v, type: k }) => {
      const C = y(b),
        x = L != null ? L.splitLineIndex : v.splitLineIndex;
      (k === "change" &&
        L != null &&
        v != null &&
        Ks({
          additionLine: e.additionLines[L.lineIndex],
          deletionLine: e.deletionLines[v.lineIndex],
          deletionLineIndex: C.deletionContent.length,
          additionLineIndex: C.additionContent.length,
          deletionDecorations: C.deletionDecorations,
          additionDecorations: C.additionDecorations,
          lineDiffType: c,
          maxLineDiffLength: f,
        }),
        v != null &&
          (S(
            e.deletionLines[v.lineIndex],
            v.lineIndex,
            C.deletionSegments,
            C.deletionContent,
          ),
          C.deletionInfo.push({
            type: k === "change" ? "change-deletion" : k,
            lineNumber: v.lineNumber,
            altLineNumber: k === "change" ? void 0 : (L.lineNumber ?? void 0),
            lineIndex: `${v.unifiedLineIndex},${x}`,
          })),
        L != null &&
          (S(
            e.additionLines[L.lineIndex],
            L.lineIndex,
            C.additionSegments,
            C.additionContent,
          ),
          C.additionInfo.push({
            type: k === "change" ? "change-addition" : k,
            lineNumber: L.lineNumber,
            altLineNumber: k === "change" ? void 0 : (v.lineNumber ?? void 0),
            lineIndex: `${L.unifiedLineIndex},${x}`,
          })));
    },
  });
  for (const b of m.values()) {
    if (b.deletionContent.length === 0 && b.additionContent.length === 0)
      continue;
    const L = { name: e.prevName ?? e.name, contents: b.deletionContent.value },
      v = { name: e.name, contents: b.additionContent.value },
      { deletionLines: k, additionLines: C } = Ys({
        deletionFile: L,
        deletionInfo: b.deletionInfo,
        deletionDecorations: b.deletionDecorations,
        additionFile: v,
        additionInfo: b.additionInfo,
        additionDecorations: b.additionDecorations,
        highlighter: t,
        options: n,
        languageOverride: i ? "text" : e.lang,
      });
    if (p) {
      ((u.deletionLines = k), (u.additionLines = C));
      continue;
    }
    if (b.deletionSegments.length > 0)
      for (const x of b.deletionSegments)
        for (let A = 0; A < x.count; A++)
          u.deletionLines[x.targetIndex + A] = k[x.originalOffset + A];
    else je(u.deletionLines, k);
    if (b.additionSegments.length > 0)
      for (const x of b.additionSegments)
        for (let A = 0; A < x.count; A++)
          u.additionLines[x.targetIndex + A] = C[x.originalOffset + A];
    else je(u.additionLines, C);
  }
  return { code: u, themeStyles: h, baseThemeType: d };
}
function Ks({
  deletionLine: e,
  additionLine: t,
  deletionLineIndex: n,
  additionLineIndex: i,
  deletionDecorations: r,
  additionDecorations: o,
  lineDiffType: s,
  maxLineDiffLength: a,
}) {
  if (
    e == null ||
    t == null ||
    s === "none" ||
    ((e = Pe(e)), (t = Pe(t)), e.length > a || t.length > a)
  )
    return;
  const l = s === "char" ? Is(e, t) : Ds(e, t),
    d = [],
    h = [],
    c = s === "word-alt",
    u = l.at(-1);
  for (const p of l) {
    const g = p === u;
    !p.added && !p.removed
      ? (yt({ item: p, arr: d, enableJoin: c, isNeutral: !0, isLastItem: g }),
        yt({ item: p, arr: h, enableJoin: c, isNeutral: !0, isLastItem: g }))
      : p.removed
        ? yt({ item: p, arr: d, enableJoin: c, isLastItem: g })
        : yt({ item: p, arr: h, enableJoin: c, isLastItem: g });
  }
  let f = 0;
  for (const p of d)
    (p[0] === 1 &&
      r.push(Qn({ line: n, spanStart: f, spanLength: p[1].length })),
      (f += p[1].length));
  f = 0;
  for (const p of h)
    (p[0] === 1 &&
      o.push(Qn({ line: i, spanStart: f, spanLength: p[1].length })),
      (f += p[1].length));
}
function Xs() {
  return {
    deletionContent: {
      push(e) {
        ((this.value += e), this.length++);
      },
      value: "",
      length: 0,
    },
    additionContent: {
      push(e) {
        ((this.value += e), this.length++);
      },
      value: "",
      length: 0,
    },
    deletionInfo: [],
    additionInfo: [],
    deletionDecorations: [],
    additionDecorations: [],
    deletionSegments: [],
    additionSegments: [],
  };
}
function Ys({
  deletionFile: e,
  additionFile: t,
  deletionInfo: n,
  additionInfo: i,
  highlighter: r,
  deletionDecorations: o,
  additionDecorations: s,
  languageOverride: a,
  options: { theme: l, ...d },
}) {
  const h = a ?? j(e.name),
    c = a ?? j(t.name),
    { state: u, transformers: f } = Ji(d.useTokenTransformer),
    p =
      typeof l == "string"
        ? {
            ...d,
            lang: "text",
            theme: l,
            transformers: f,
            decorations: void 0,
            defaultColor: !1,
            cssVariablePrefix: q("token"),
            tokenizeTimeLimit: 0,
          }
        : {
            ...d,
            lang: "text",
            themes: l,
            transformers: f,
            decorations: void 0,
            defaultColor: !1,
            cssVariablePrefix: q("token"),
            tokenizeTimeLimit: 0,
          };
  return {
    deletionLines:
      e.contents === ""
        ? []
        : ((p.lang = h),
          (u.lineInfo = n),
          (p.decorations = o),
          sn(r.codeToHast(Pe(e.contents), p))),
    additionLines:
      t.contents === ""
        ? []
        : ((p.lang = c),
          (p.decorations = s),
          (u.lineInfo = i),
          sn(r.codeToHast(Pe(t.contents), p))),
  };
}
function ye(e, t) {
  const n = Oe(
    { name: e.prevName ?? e.name, contents: e.deletionLines.join("") },
    { name: e.name, contents: e.additionLines.join(""), lang: e.lang },
    t,
  );
  return {
    hunks: n.hunks,
    splitLineCount: n.splitLineCount,
    unifiedLineCount: n.unifiedLineCount,
    additionLines: n.additionLines,
    deletionLines: n.deletionLines,
    type: n.type,
  };
}
function Qs(e, t) {
  const n = Math.max(e, 1);
  let i = Array.from(
    { length: n },
    (r, o) => `${" ".repeat(o + 1)}
`,
  ).join("");
  return (
    i === t &&
      (i = Array.from(
        { length: n },
        (r, o) => `\0${" ".repeat(o)}
`,
      ).join("")),
    i
  );
}
function Zs(e) {
  for (const t of e) if (t.trim().length > 0) return !1;
  return !0;
}
function cn(e, t) {
  return t.length > 0 && t.length < e.deletionLines.length && Zs(t);
}
function _t(e, t, n) {
  const i = e.deletionLines.join(""),
    r = Qs(t.length, i),
    o = Oe(
      { name: e.prevName ?? e.name, contents: i },
      { name: e.name, contents: r, lang: e.lang },
      n,
    );
  return {
    hunks: o.hunks,
    splitLineCount: o.splitLineCount,
    unifiedLineCount: o.unifiedLineCount,
    additionLines: t,
    deletionLines: o.deletionLines,
    type: o.type,
  };
}
function Dt(e, t) {
  return _t(e, [""], t);
}
function Lr(e, t) {
  if (e.additionLines.length === 0) return Dt(e, t);
  if (cn(e, e.additionLines)) return _t(e, e.additionLines, t);
  const n = e.additionLines,
    i = ye(e, t);
  return (xr(i, n), i);
}
function Js(e) {
  return e.length > 1 && e.at(-1) === "";
}
function xr(e, t) {
  if (!Js(t)) return;
  const n = t.length - e.additionLines.length;
  if (n <= 0) return;
  const i = e.additionLines.length,
    r = e.hunks.at(-1);
  if (r != null && ee(r.additionStart, r.additionCount) === i) {
    for (const o of r.hunkContent)
      if (
        o.type === "change" &&
        o.additions < o.deletions &&
        o.additionLineIndex + o.additions === i
      ) {
        ((e.additionLines = t),
          (o.additions += n),
          (r.additionCount += n),
          (r.additionLines += n),
          Gt(e));
        return;
      }
  }
}
function ea(e, t, n) {
  if (e.isPartial || e.deletionLines.length !== e.additionLines.length)
    return ve(e, ye(e, n));
  const i = Array.from(t);
  if (i.length === 0)
    return ve(e, {
      hunks: e.hunks,
      splitLineCount: e.splitLineCount,
      unifiedLineCount: e.unifiedLineCount,
      type: e.type,
    });
  for (const o of i) {
    const s = e.additionLines[o],
      a = e.deletionLines[o];
    if (s == null || a == null || Pe(s) === Pe(a)) return ve(e, ye(e, n));
  }
  const r = ta(e, i);
  if (r.size === 0) return ve(e, ye(e, n));
  for (const o of r) if (!ia(e, o, n)) return ve(e, ye(e, n));
  return (
    Gt(e),
    Ns(e)
      ? ve(e, ye(e, n))
      : ve(e, {
          hunks: e.hunks,
          splitLineCount: e.splitLineCount,
          unifiedLineCount: e.unifiedLineCount,
          type: e.type,
        })
  );
}
function ve(e, t) {
  return (Object.assign(e, t), t);
}
function ta(e, t) {
  const n = new Set();
  for (const i of t) {
    const r = na(e, i);
    if (r == null) return new Set();
    n.add(r);
  }
  return n;
}
function na(e, t) {
  for (const [n, i] of e.hunks.entries()) {
    const r = i.additionLineIndex + i.additionCount;
    if (t >= i.additionLineIndex && t < r) return n;
  }
}
function ia(e, t, n) {
  const i = e.hunks[t];
  if (i == null) return !1;
  const r = e.deletionLines.slice(
      i.deletionLineIndex,
      i.deletionLineIndex + i.deletionCount,
    ),
    o = e.additionLines.slice(
      i.additionLineIndex,
      i.additionLineIndex + i.additionCount,
    ),
    s = Oe(
      { name: e.prevName ?? e.name, contents: r.join("") },
      { name: e.name, contents: o.join(""), lang: e.lang },
      { ...n, context: 0 },
    ),
    a = s.hunks[0];
  return a == null || s.hunks.length !== 1 ? !1 : (ra(i, a), kr(e, t), !0);
}
function kr(e, t) {
  const n = e.hunks[t];
  if (n == null) return;
  if (t !== e.hunks.length - 1) {
    ((n.noEOFCRAdditions = !1), (n.noEOFCRDeletions = !1));
    return;
  }
  const i = e.additionLines.at(-1),
    r = e.deletionLines.at(-1);
  ((n.noEOFCRAdditions =
    i != null &&
    i !== "" &&
    !i.endsWith(`
`)),
    (n.noEOFCRDeletions =
      r != null &&
      r !== "" &&
      !r.endsWith(`
`)));
}
function ra(e, t) {
  const n = e.additionLineIndex,
    i = e.deletionLineIndex;
  ((e.hunkContent = t.hunkContent.map((r) => wr(r, n, i))),
    (e.additionLineIndex = n + t.additionLineIndex),
    (e.additionStart = e.additionStart + t.additionLineIndex),
    (e.additionCount = t.additionCount),
    (e.additionLines = t.additionLines),
    t.deletionLineIndex >= 0 &&
      ((e.deletionLineIndex = i + t.deletionLineIndex),
      (e.deletionStart = e.deletionStart + t.deletionLineIndex)),
    (e.deletionCount = t.deletionCount),
    (e.deletionLines = t.deletionLines),
    (e.noEOFCRAdditions = t.noEOFCRAdditions),
    (e.noEOFCRDeletions = t.noEOFCRDeletions),
    Dn(e));
}
function wr(e, t, n) {
  return {
    ...e,
    additionLineIndex: e.additionLineIndex + t,
    deletionLineIndex: e.deletionLineIndex + n,
  };
}
function Dn(e) {
  let t = 0,
    n = 0;
  for (const i of e.hunkContent)
    i.type === "context"
      ? ((t += i.lines), (n += i.lines))
      : ((t += Math.max(i.additions, i.deletions)),
        (n += i.additions + i.deletions));
  ((e.splitLineCount = t), (e.unifiedLineCount = n));
}
function Gt(e) {
  let t = 0,
    n = 0,
    i = 0;
  for (const r of e.hunks)
    ((r.collapsedBefore = Math.max(
      xe(r.additionStart, r.additionCount) - i,
      0,
    )),
      (r.splitLineStart = t + r.collapsedBefore),
      (r.unifiedLineStart = n + r.collapsedBefore),
      Dn(r),
      (t += r.collapsedBefore + r.splitLineCount),
      (n += r.collapsedBefore + r.unifiedLineCount),
      (i = ee(r.additionStart, r.additionCount)));
  if (e.hunks.length > 0) {
    const r = e.hunks[e.hunks.length - 1],
      o = Math.max(
        e.additionLines.length - ee(r.additionStart, r.additionCount),
        0,
      );
    ((t += o), (n += o));
  }
  ((e.splitLineCount = t), (e.unifiedLineCount = n));
}
const Zn = new WeakMap();
function oa(e) {
  return e.length > 1 && e[e.length - 1] === "" ? e.slice(0, -1) : e;
}
function sa(e, t) {
  const n = Math.min(e.length, t.length);
  let i = 0;
  for (; i < n && e[i] === t[i];) i++;
  let r = e.length,
    o = t.length;
  for (; r > i && o > i && e[r - 1] === t[o - 1];) (r--, o--);
  if (!(i === r && i === o))
    return { start: i, deletionEnd: r, additionEnd: o };
}
function un(e, t) {
  const n = e.hunks,
    i = e.additionLines,
    r = oa(i),
    o = r === i ? e : { ...e, additionLines: r },
    s = ga(n, pa(o, t), e.deletionLines.length),
    a = va(o, s);
  if (
    ((e.additionLines = r),
    (e.hunks = a),
    (e.editSessionDirty = !0),
    La(e),
    xr(e, i),
    !!ba(n, e.hunks))
  )
    return { regions: s.map((l) => l.previousSpan) };
}
function aa(e, t, n, i) {
  const r = Array.from(new Set(t))
    .filter((l) => l >= 0 && l < e.additionLines.length)
    .sort((l, d) => l - d);
  if (r.length === 0) return;
  const { hunks: o } = e;
  let s,
    a = 0;
  for (const l of r) {
    for (; a < o.length;) {
      const c = o[a];
      if (l < Bt(c) + c.additionCount) break;
      a++;
    }
    const d = o[a],
      h = d == null ? void 0 : Bt(d);
    if (h == null || l < h || (s != null && s !== a)) return un(e, n);
    s = a;
  }
  if (s != null) {
    if (la(e, r, s, i, n)) {
      e.editSessionDirty = !0;
      return;
    }
    return un(e, n);
  }
}
function la(e, t, n, i, r) {
  if (i == null || r?.ignoreWhitespace === !0 || r?.stripTrailingCr === !0)
    return !1;
  const o = da(e),
    s = e.hunks[n];
  if (s.hunkContent.some(ha)) return !1;
  for (const a of t) {
    const l = i.get(a),
      d = e.additionLines[a];
    if (l == null || d == null || o.has(l) || o.has(d)) return !1;
    let h = !1;
    for (const c of s.hunkContent)
      if (
        c.type === "change" &&
        c.additions === c.deletions &&
        a >= c.additionLineIndex &&
        a < c.additionLineIndex + c.additions
      ) {
        h = !0;
        break;
      }
    if (!h) return !1;
  }
  return !0;
}
function da(e) {
  const t = Zn.get(e);
  if (t?.lines === e.deletionLines) return t.set;
  const n = new Set(e.deletionLines);
  return (Zn.set(e, { lines: e.deletionLines, set: n }), n);
}
function ha(e) {
  return e?.type === "change" && (e.additions === 0 || e.deletions === 0);
}
function ca(e, t) {
  const n = new Map(),
    { regions: i } = t;
  for (let r = 0; r <= i.length; r++) {
    const o = i[r - 1],
      s = i[r],
      a = r === 0 ? e.get(0) : o == null ? void 0 : e.get(o.lastIndex + 1),
      l = s == null ? void 0 : e.get(s.firstIndex),
      d = a?.fromStart ?? 0,
      h = l?.fromEnd ?? 0;
    (d > 0 || h > 0) && n.set(r, { fromStart: d, fromEnd: h });
  }
  return n;
}
function ua(e, t, n) {
  const i = [];
  if (e.isPartial) return i;
  for (const [o, s] of e.hunks.entries()) {
    const a = ke({
      isPartial: e.isPartial,
      rangeSize: s.collapsedBefore,
      expandedHunks: t,
      hunkIndex: o,
      collapsedContextThreshold: n,
    });
    if (a.rangeSize <= n) continue;
    const l = ze(s),
      d = l - a.rangeSize;
    (a.fromStart > 0 && i.push([d, d + a.fromStart]),
      a.fromEnd > 0 && i.push([l - a.fromEnd, l]));
  }
  const r = we({
    fileDiff: e,
    hunkIndex: e.hunks.length - 1,
    expandedHunks: t,
    collapsedContextThreshold: n,
    errorPrefix: "captureExpansionAnchors",
  });
  if (r != null && r.fromStart > 0 && r.rangeSize > n) {
    const o = e.hunks[e.hunks.length - 1],
      s = ze(o) + o.deletionCount;
    i.push([s, s + r.fromStart]);
  }
  return i;
}
function fa(e, t) {
  const n = new Map();
  if (t.length === 0) return n;
  const i = (o, s, a) => {
    if (a <= s) return;
    let l = 0,
      d = 0;
    for (const [h, c] of t)
      c <= s ||
        h >= a ||
        (h <= s && (l = Math.max(l, Math.min(c, a) - s)),
        c >= a && (d = Math.max(d, a - Math.max(h, s))));
    (l > 0 || d > 0) && n.set(o, { fromStart: l, fromEnd: d });
  };
  for (const [o, s] of e.hunks.entries()) {
    const a = ze(s);
    i(o, a - Math.max(s.collapsedBefore, 0), a);
  }
  const r = e.hunks[e.hunks.length - 1];
  return (
    r != null &&
      !e.isPartial &&
      e.deletionLines.length > 0 &&
      i(e.hunks.length, ze(r) + r.deletionCount, e.deletionLines.length),
    n
  );
}
function Jn(e, t) {
  return e.editSessionDirty !== !0
    ? !1
    : (delete e.editSessionDirty,
      Object.assign(
        e,
        e.additionLines.length <= 1 && e.additionLines.join("") === ""
          ? ye(e, t)
          : Lr(e, t),
      ),
      !0);
}
function pa(e, t) {
  if (sa(e.deletionLines, e.additionLines) == null) return [];
  const n = Oe(
      { name: e.prevName ?? e.name, contents: e.deletionLines.join("") },
      { name: e.name, contents: e.additionLines.join(""), lang: e.lang },
      t,
    ),
    i = [];
  let r = 0,
    o = 0;
  for (const s of n.hunks) {
    const a =
      s.additionCount > 0 ? s.additionLineIndex - r : s.deletionLineIndex - o;
    ((r += a), (o += a));
    for (const l of s.hunkContent) {
      if (l.type === "context") {
        ((r += l.lines), (o += l.lines));
        continue;
      }
      const d = wr(l, 0, 0);
      (d.additions === 0 && (d.additionLineIndex = r),
        d.deletions === 0 && (d.deletionLineIndex = o),
        i.push(d),
        (r += d.additions),
        (o += d.deletions));
    }
  }
  return i;
}
function ga(e, t, n) {
  const i = e.map((s, a) => {
      const l = ze(s);
      return {
        deletionStart: l,
        deletionEnd: l + s.deletionCount,
        blocks: [],
        previousSpan: { firstIndex: a, lastIndex: a },
      };
    }),
    r = [];
  let o = 0;
  for (const s of t) {
    const a = s.deletionLineIndex,
      l = a + s.deletions;
    for (; o < i.length && i[o].deletionEnd < a;) (r.push(i[o]), o++);
    let d = r.length > 0 && ma(a, l, r[r.length - 1]) ? r.pop() : void 0;
    for (; o < i.length && i[o].deletionStart <= l;) ((d = Sa(d, i[o])), o++);
    if (d == null) {
      let h = a,
        c = l;
      if ((s.deletions === 0 || s.additions === 0) && n > s.deletions) {
        const u = r[r.length - 1]?.deletionEnd ?? 0,
          f = i[o]?.deletionStart ?? n;
        a > u ? h-- : l < f && c++;
      }
      d = {
        deletionStart: h,
        deletionEnd: c,
        blocks: [],
        previousSpan: void 0,
      };
    }
    ((d.deletionStart = Math.min(d.deletionStart, a)),
      (d.deletionEnd = Math.max(d.deletionEnd, l)),
      d.blocks.push(s),
      r.push(d));
  }
  for (; o < i.length;) (r.push(i[o]), o++);
  return r;
}
function ma(e, t, n) {
  return e <= n.deletionEnd && t >= n.deletionStart;
}
function Sa(e, t) {
  return e == null
    ? t
    : ((e.deletionStart = Math.min(e.deletionStart, t.deletionStart)),
      (e.deletionEnd = Math.max(e.deletionEnd, t.deletionEnd)),
      e.blocks.push(...t.blocks),
      t.previousSpan != null &&
        ((e.previousSpan ??= { ...t.previousSpan }),
        (e.previousSpan.firstIndex = Math.min(
          e.previousSpan.firstIndex,
          t.previousSpan.firstIndex,
        )),
        (e.previousSpan.lastIndex = Math.max(
          e.previousSpan.lastIndex,
          t.previousSpan.lastIndex,
        ))),
      e);
}
function va(e, t) {
  const n = [];
  let i = 0,
    r = 0;
  for (const o of t) {
    const s = o.deletionStart - i;
    if (s < 0)
      throw new Error("buildRegionHunks: overlapping old-side regions");
    ((i += s), (r += s));
    const a = r,
      l = [];
    for (const h of o.blocks) {
      const c = h.deletionLineIndex - i,
        u = h.additionLineIndex - r;
      if (c < 0 || c !== u)
        throw new Error("buildRegionHunks: canonical block context mismatch");
      (ei(l, c, r, i),
        (i += c),
        (r += u),
        l.push({ ...h }),
        (i += h.deletions),
        (r += h.additions));
    }
    const d = o.deletionEnd - i;
    if (d < 0)
      throw new Error("buildRegionHunks: block exceeds its old-side region");
    (ei(l, d, r, i),
      (i += d),
      (r += d),
      n.push(
        Ca(
          e,
          {
            additionStart: a,
            additionEnd: r,
            deletionStart: o.deletionStart,
            deletionEnd: o.deletionEnd,
          },
          l,
        ),
      ));
  }
  if (e.deletionLines.length - i !== e.additionLines.length - r)
    throw new Error("buildRegionHunks: trailing context mismatch");
  return n;
}
function Ca(e, t, n) {
  const i = t.additionEnd - t.additionStart,
    r = t.deletionEnd - t.deletionStart;
  let o = 0,
    s = 0;
  for (const l of n)
    l.type === "change" && ((o += l.additions), (s += l.deletions));
  const a = {
    collapsedBefore: 0,
    additionStart: xt(t.additionStart, i),
    additionCount: i,
    additionLines: o,
    additionLineIndex: ni(t.additionStart, i),
    deletionStart: xt(t.deletionStart, r),
    deletionCount: r,
    deletionLines: s,
    deletionLineIndex: ni(t.deletionStart, r),
    hunkContent: n,
    hunkSpecs: `@@ -${xt(t.deletionStart, r)},${r} +${xt(t.additionStart, i)},${i} @@`,
    splitLineStart: 0,
    splitLineCount: 0,
    unifiedLineStart: 0,
    unifiedLineCount: 0,
    noEOFCRAdditions: !1,
    noEOFCRDeletions: !1,
  };
  return (Dn(a), a);
}
function ei(e, t, n, i) {
  t > 0 &&
    e.push({
      type: "context",
      lines: t,
      additionLineIndex: n,
      deletionLineIndex: i,
    });
}
function ba(e, t) {
  if (e.length !== t.length) return !0;
  for (let n = 0; n < e.length; n++) {
    const i = e[n],
      r = t[n];
    if (
      ze(i) !== ze(r) ||
      i.deletionCount !== r.deletionCount ||
      Bt(i) !== Bt(r) ||
      i.additionCount !== r.additionCount ||
      i.splitLineCount !== r.splitLineCount ||
      !ya(i, r)
    )
      return !0;
  }
  return !1;
}
function ya(e, t) {
  const n = ti(e),
    i = ti(t);
  for (;;) {
    const r = n.next(),
      o = i.next();
    if (r.done === !0 || o.done === !0) return r.done === o.done;
    if (r.value[0] !== o.value[0] || r.value[1] !== o.value[1]) return !1;
  }
}
function* ti(e) {
  for (const t of e.hunkContent) {
    if (t.type === "context") {
      for (let i = 0; i < t.lines; i++)
        yield [t.deletionLineIndex + i, t.additionLineIndex + i];
      continue;
    }
    const n = Math.max(t.deletions, t.additions);
    for (let i = 0; i < n; i++)
      yield [
        i < t.deletions ? t.deletionLineIndex + i : void 0,
        i < t.additions ? t.additionLineIndex + i : void 0,
      ];
  }
}
function Bt(e) {
  return xe(e.additionStart, e.additionCount);
}
function ze(e) {
  return xe(e.deletionStart, e.deletionCount);
}
function xt(e, t) {
  return t === 0 ? e : e + 1;
}
function ni(e, t) {
  return t === 0 ? e - 1 : e;
}
function La(e) {
  Gt(e);
  for (let t = 0; t < e.hunks.length; t++) kr(e, t);
}
function it(e) {
  const t = e.lang ?? j(e.name),
    n = e.lang ?? (e.prevName != null ? j(e.prevName) : "text");
  return t === "text" && n === "text";
}
let xa = -1;
var Er = class {
  options;
  annotationSlotName;
  onRenderUpdate;
  workerManager;
  __id = `diff-hunks-renderer:${++xa}`;
  highlighter;
  diff;
  expandedHunks = new Map();
  deletionAnnotations = {};
  additionAnnotations = {};
  computedLang = "text";
  renderCache;
  pendingHighlightResult;
  pendingStructuralRows;
  editSessionActive = !1;
  constructor(e = { theme: U }, t = fe, n, i) {
    ((this.options = e),
      (this.annotationSlotName = t),
      (this.onRenderUpdate = n),
      (this.workerManager = i),
      i?.isWorkingPool() !== !0 &&
        (this.highlighter = ce(e.theme ?? U) ? Ge() : void 0));
  }
  cleanUp() {
    (this.recycle(),
      this.expandedHunks.clear(),
      (this.workerManager = void 0),
      (this.onRenderUpdate = void 0));
  }
  recycle() {
    ((this.highlighter = void 0),
      (this.diff = void 0),
      this.clearRenderCache(),
      (this.additionAnnotations = {}),
      (this.deletionAnnotations = {}),
      this.workerManager?.cleanUpTasks(this),
      this.endEditSession());
  }
  beginEditSession(e, t) {
    const { editSessionActive: n, renderCache: i } = this;
    if (
      ((this.editSessionActive = !0),
      n || (this.pendingHighlightResult = void 0),
      (this.diff = e),
      !e.isPartial && e.additionLines.length === 0)
    ) {
      (Object.assign(e, Dt(e, this.options.parseDiffOptions)),
        this.markEditSessionPass(e),
        this.clearRenderCache());
      return;
    }
    if (i == null || (n && i.diff === e)) return;
    const { options: r } = this.getRenderOptions(e),
      o = i.diff === e,
      s = t != null && B(i.diff, t),
      { result: a } = i;
    if (!i.highlighted || a == null || !Se(i.options, r) || (!o && !s)) {
      this.clearRenderCache();
      return;
    }
    o ||
      (this.renderCache = {
        diff: e,
        options: r,
        highlighted: !0,
        result: {
          ...a,
          code: { ...a.code, additionLines: [...a.code.additionLines] },
        },
        renderRange: i.renderRange,
      });
  }
  endEditSession() {
    ((this.editSessionActive = !1), (this.pendingHighlightResult = void 0));
  }
  editorRenderReady() {
    return (
      this.renderCache?.options.useTokenTransformer === !0 &&
      this.renderCache.highlighted &&
      this.renderCache.result != null
    );
  }
  refreshHighlightedResult() {
    const { diff: e, renderCache: t, workerManager: n } = this;
    return e == null ||
      t == null ||
      !B(t.diff, e) ||
      it(e) ||
      Ze(e, this.getTokenizeMaxLength())
      ? Promise.resolve()
      : !this.editSessionActive &&
          n?.isWorkingPool() === !0 &&
          e.cacheKey != null
        ? (n.evictDiffFromCache(e.cacheKey),
          n
            .primeDiffHighlightCache(e)
            .then(() => {
              this.applyRefreshedResult(e, n.getDiffResultCache(e));
            })
            .catch((i) => this.onHighlightError(i)))
        : this.asyncHighlight(e)
            .then((i) => this.applyRefreshedResult(e, i))
            .catch((i) => this.onHighlightError(i));
  }
  applyRefreshedResult(e, t) {
    const { diff: n, renderCache: i } = this;
    if (
      t == null ||
      n == null ||
      i == null ||
      !B(n, e) ||
      !B(i.diff, e) ||
      this.editSessionActive
    )
      return;
    const { options: r } = this.getRenderOptions(n);
    Se(r, t.options) &&
      ((this.pendingHighlightResult = {
        diff: n,
        options: t.options,
        highlighted: !0,
        result: t.result,
      }),
      this.onRenderUpdate?.());
  }
  get diffCache() {
    return this.renderCache?.diff ?? this.diff;
  }
  clearRenderCache() {
    ((this.renderCache = void 0),
      (this.pendingHighlightResult = void 0),
      (this.pendingStructuralRows = void 0));
  }
  setOptions(e) {
    this.options = e;
  }
  mergeOptions(e) {
    this.options = { ...this.options, ...e };
  }
  expandHunk(e, t, n = this.getOptionsWithDefaults().expansionLineCount) {
    const i = {
      ...(this.expandedHunks.get(e) ?? { fromStart: 0, fromEnd: 0 }),
    };
    ((t === "up" || t === "both") && (i.fromStart += n),
      (t === "down" || t === "both") && (i.fromEnd += n),
      this.renderCache?.highlighted !== !0 && this.clearRenderCache(),
      this.expandedHunks.set(e, i));
  }
  getExpandedHunk(e) {
    return this.expandedHunks.get(e) ?? Gr;
  }
  getExpandedHunksMap() {
    return this.expandedHunks;
  }
  setExpandedHunksMap(e) {
    this.expandedHunks = e;
  }
  setLineAnnotations(e) {
    ((this.additionAnnotations = {}), (this.deletionAnnotations = {}));
    for (const t of e) {
      const n = (() => {
          switch (t.side) {
            case "deletions":
              return this.deletionAnnotations;
            case "additions":
              return this.additionAnnotations;
          }
        })(),
        i = n[t.lineNumber] ?? [];
      ((n[t.lineNumber] = i), i.push(t));
    }
  }
  updateRenderCache(e, t, n = !1) {
    this.pendingStructuralRows = void 0;
    const { renderCache: i } = this;
    if (i == null) return !1;
    const { result: r, diff: o } = i;
    if (r == null) return !1;
    if (o.isPartial)
      throw new Error("Could not update render cache for partial diff");
    const s = r.code.additionLines,
      a = (this.pendingStructuralRows = n ? new Map() : void 0),
      l = [],
      d = new Map();
    for (const [c, u] of e) {
      const f = s[c]?.properties ?? {},
        p = u.map((b) => b[2]).join(""),
        g = c < o.additionLines.length,
        m = g ? (o.additionLines[c] ?? "") : "",
        y = Pe(m);
      a == null &&
        g &&
        ((o.additionLines[c] = tr(m, p)), y !== p && (l.push(c), d.set(c, m)));
      const S = {
        type: "element",
        tagName: "div",
        properties: {
          "data-line": f["data-line"] ?? c + 1,
          "data-line-index": f["data-line-index"] ?? c,
          "data-line-type": f["data-line-type"] ?? "context",
        },
        children: u.map(([b, L, v]) =>
          b === 0 && L === ""
            ? v === ""
              ? { type: "element", tagName: "br", properties: {}, children: [] }
              : { type: "text", value: v }
            : {
                type: "element",
                tagName: "span",
                properties: { "data-char": b, style: `color:${L};` },
                children: [{ type: "text", value: v }],
              },
        ),
      };
      a != null ? a.set(c, S) : (s[c] = S);
    }
    let h = !1;
    if (l.length > 0)
      if (this.editSessionActive)
        if (o.additionLines.length <= 1 && o.additionLines.join("") === "")
          (this.applyRecomputePreservingSessionType(
            o,
            Dt(o, this.options.parseDiffOptions),
          ),
            (h = !0));
        else if (cn(o, o.additionLines))
          (this.applyRecomputePreservingSessionType(
            o,
            _t(o, o.additionLines, this.options.parseDiffOptions),
          ),
            (h = !0));
        else {
          const c = aa(o, l, this.options.parseDiffOptions, d);
          (this.applyExpansionRemap(c), (h = c != null));
        }
      else Object.assign(o, ea(o, l, this.options.parseDiffOptions));
    return ((r.baseThemeType = t), (i.isDirty = !0), h);
  }
  applyExpansionRemap(e) {
    e != null && (this.expandedHunks = ca(this.expandedHunks, e));
  }
  applyDocumentChange(e) {
    const { pendingStructuralRows: t, renderCache: n } = this;
    if (((this.pendingStructuralRows = void 0), n == null)) return;
    const { diff: i, result: r } = n;
    if (r == null) return;
    if (i.isPartial)
      throw new Error("Could not apply document change for partial diff");
    const { additionLines: o } = i;
    if (
      ((i.additionLines = Ra(e)),
      (r.code.additionLines = Ea(o, i.additionLines, r.code.additionLines, e)),
      i.additionLines.length <= 1 && i.additionLines.join("") === ""
        ? (this.applyRecomputePreservingSessionType(
            i,
            Dt(i, this.options.parseDiffOptions),
          ),
          (r.code.additionLines[0] = fn(0, e.getLineText(0))))
        : this.editSessionActive
          ? this.applySessionDocumentChange(i)
          : Object.assign(i, Lr(i, this.options.parseDiffOptions)),
      t != null)
    )
      for (const [s, a] of t)
        s < r.code.additionLines.length && (r.code.additionLines[s] = a);
    n.isDirty = !0;
  }
  applySessionDocumentChange(e) {
    const { parseDiffOptions: t } = this.options,
      n = e.additionLines;
    if (cn(e, n)) {
      this.applyRecomputePreservingSessionType(e, _t(e, n, t));
      return;
    }
    this.applyExpansionRemap(un(e, t));
  }
  applyRecomputePreservingSessionType(e, t) {
    const n = this.editSessionActive ? e.type : void 0;
    (Object.assign(e, t),
      n != null && (e.type = n),
      this.markEditSessionPass(e));
  }
  markEditSessionPass(e) {
    this.editSessionActive && (e.editSessionDirty = !0);
  }
  getUnifiedLineDecoration({ lineType: e }) {
    return { gutterLineType: e, contentProperties: { "data-line-type": e } };
  }
  getSplitLineDecoration({ side: e, type: t }) {
    const n =
      t === "change"
        ? e === "deletions"
          ? "change-deletion"
          : "change-addition"
        : t;
    return { gutterLineType: n, contentProperties: { "data-line-type": n } };
  }
  createAnnotationElement = (e) => on(e);
  getOptionsWithDefaults() {
    const {
      diffIndicators: e = "bars",
      diffStyle: t = "split",
      disableBackground: n = !1,
      disableFileHeader: i = !1,
      disableLineNumbers: r = !1,
      disableVirtualizationBuffers: o = !1,
      collapsed: s = !1,
      expandUnchanged: a = !1,
      collapsedContextThreshold: l = 1,
      expansionLineCount: d = 100,
      hunkSeparators: h = "line-info",
      lineDiffType: c = "word-alt",
      maxLineDiffLength: u = 1e3,
      overflow: f = "scroll",
      stickyHeader: p = !1,
      theme: g = U,
      headerRenderMode: m = "default",
      tokenizeMaxLineLength: y = 1e3,
      tokenizeMaxLength: S = Wr,
      useTokenTransformer: b = !1,
      useCSSClasses: L = !1,
    } = this.options;
    return {
      diffIndicators: e,
      diffStyle: t,
      disableBackground: n,
      disableFileHeader: i,
      disableLineNumbers: r,
      disableVirtualizationBuffers: o,
      collapsed: s,
      expandUnchanged: a,
      collapsedContextThreshold: l,
      expansionLineCount: d,
      hunkSeparators: h,
      lineDiffType: c,
      maxLineDiffLength: u,
      overflow: f,
      stickyHeader: p,
      theme: this.workerManager?.getDiffRenderOptions().theme ?? g,
      headerRenderMode: m,
      tokenizeMaxLineLength: y,
      tokenizeMaxLength: S,
      useTokenTransformer: b,
      useCSSClasses: L,
    };
  }
  async initializeHighlighter() {
    return (
      (this.highlighter = await ct(
        xn(this.computedLang, {
          theme: this.getLocalHighlightTheme(),
          preferredHighlighter:
            this.workerManager?.getPreferredHighlighter() ??
            this.options.preferredHighlighter,
        }),
      )),
      this.highlighter
    );
  }
  hydrate(e) {
    if (e == null) return;
    this.diff = e;
    const { options: t } = this.getRenderOptions(e),
      n = Ze(e, this.getTokenizeMaxLength());
    let i = this.workerManager?.getDiffResultCache(e);
    (i != null && !Se(t, i.options) && (i = void 0),
      (this.renderCache ??= {
        diff: e,
        hydrated: !0,
        highlighted: !n && !it(e),
        options: t,
        result: n ? void 0 : i?.result,
        renderRange: void 0,
      }),
      !this.editSessionActive && this.workerManager?.isWorkingPool() === !0
        ? this.renderCache.result == null &&
          !n &&
          this.workerManager.highlightDiffAST(this, this.diff)
        : this.highlighter == null &&
          ((this.computedLang = e.lang ?? j(e.name)),
          this.initializeHighlighter()));
  }
  getLocalHighlightTheme() {
    return (
      this.workerManager?.getDiffRenderOptions().theme ??
      this.options.theme ??
      U
    );
  }
  getEffectiveCodeOptions() {
    const e =
      this.workerManager?.isWorkingPool() === !0
        ? this.workerManager.getDiffRenderOptions()
        : void 0;
    return {
      theme: this.getLocalHighlightTheme(),
      tokenizeMaxLineLength:
        e?.tokenizeMaxLineLength ?? this.options.tokenizeMaxLineLength,
    };
  }
  getRenderOptions(e) {
    const t = (() => {
      if (this.workerManager?.isWorkingPool() === !0) {
        const a = this.workerManager.getDiffRenderOptions();
        return this.editSessionActive && a.useTokenTransformer !== !0
          ? { ...a, useTokenTransformer: !0 }
          : a;
      }
      const {
        theme: i,
        tokenizeMaxLineLength: r,
        lineDiffType: o,
        maxLineDiffLength: s,
      } = this.getOptionsWithDefaults();
      return {
        theme: i,
        useTokenTransformer:
          this.editSessionActive || this.options.useTokenTransformer === !0,
        tokenizeMaxLineLength: r,
        lineDiffType: o,
        maxLineDiffLength: s,
      };
    })();
    this.getOptionsWithDefaults();
    const { renderCache: n } = this;
    return n?.result == null
      ? { options: t, forceHighlight: !0 }
      : !B(e, n.diff) || !Se(t, n.options)
        ? { options: t, forceHighlight: !0 }
        : { options: t, forceHighlight: !1 };
  }
  getDiffForNextRender(e) {
    const { options: t } = this.getRenderOptions(e);
    if (this.getReadyRenderResult(e, t) != null || this.renderCache == null)
      return e;
    if (B(this.renderCache.diff, e)) return this.renderCache.diff;
    const n =
      !(e.additionLines.length > 0 || e.deletionLines.length > 0) ||
      it(e) ||
      Ze(e, this.getTokenizeMaxLength());
    return this.canRenderDiff(e, t, n) ? e : this.renderCache.diff;
  }
  canRenderDiff(e, t, n) {
    const { renderCache: i } = this;
    return i == null || B(i.diff, e)
      ? !0
      : n
        ? (i.result == null && i.hydrated !== !0) ||
          this.workerManager?.isWorkingPool() === !0 ||
          (this.highlighter != null && ce(t.theme))
        : i.result == null && i.hydrated !== !0
          ? !0
          : !this.editSessionActive &&
              this.workerManager?.isWorkingPool() === !0
            ? !i.highlighted
            : this.highlighter != null && ce(t.theme);
  }
  renderDiff(e = this.diff, t = We) {
    if (((this.diff = e), e == null)) {
      this.pendingHighlightResult = void 0;
      return;
    }
    const { expandUnchanged: n, collapsedContextThreshold: i } =
      this.getOptionsWithDefaults();
    let { options: r, forceHighlight: o } = this.getRenderOptions(e);
    const s = this.getReadyRenderResult(e, r);
    ((this.pendingHighlightResult = void 0),
      s != null &&
        ((this.renderCache = { ...s, diff: e, renderRange: void 0 }), (o = !1)),
      (this.renderCache ??= {
        diff: e,
        highlighted: !1,
        options: r,
        result: void 0,
        renderRange: void 0,
      }));
    const a = e.additionLines.length > 0 || e.deletionLines.length > 0,
      l = !a || it(e) || Ze(e, this.getTokenizeMaxLength()),
      d = this.canRenderDiff(e, r, l),
      h = !B(e, this.renderCache.diff),
      c = !Vt(this.renderCache.renderRange, t);
    if (!this.editSessionActive && this.workerManager?.isWorkingPool() === !0) {
      const u =
        this.renderCache.result == null &&
        this.renderCache.highlighted &&
        !l &&
        !h &&
        Nt(t);
      (d &&
        !u &&
        (l ||
          this.renderCache.result == null ||
          (!this.renderCache.highlighted && (h || c))) &&
        ((this.renderCache.diff = e),
        (this.renderCache.options = r),
        (this.renderCache.highlighted = !1),
        (this.renderCache.result == null || h || c || o) &&
          (this.renderCache.result = this.workerManager.getPlainDiffAST(
            e,
            t.startingLine,
            t.totalLines,
            Nt(t) || n ? !0 : this.expandedHunks,
            i,
          )),
        (this.renderCache.renderRange = t)),
        !l &&
          a &&
          (!this.renderCache.highlighted || o) &&
          this.workerManager.highlightDiffAST(this, e));
    } else {
      ((this.computedLang = e.lang ?? j(e.name)), (this.highlighter ??= Ge()));
      const u = this.highlighter != null && ce(r.theme),
        f = this.highlighter != null && Ft(this.computedLang),
        p = !l && f;
      if (
        d &&
        this.highlighter != null &&
        u &&
        (o ||
          l ||
          (!this.renderCache.highlighted && p) ||
          this.renderCache.result == null)
      ) {
        const { result: g, options: m } = this.renderDiffWithHighlighter(
          e,
          this.highlighter,
          l || !f,
        );
        this.renderCache = {
          diff: e,
          options: m,
          highlighted: p,
          result: g,
          renderRange: void 0,
        };
      }
      (!u || (!l && !f)) &&
        this.asyncHighlight(e).then(({ result: g, options: m }) => {
          this.applyHighlightResult(e, g, m, !l);
        });
    }
    return this.renderCache.result != null
      ? this.processDiffResult(
          this.renderCache.diff,
          t,
          this.renderCache.result,
        )
      : void 0;
  }
  async asyncRender(e, t = We) {
    this.diff = e;
    const { result: n } = await this.asyncHighlight(e);
    return this.processDiffResult(e, t, n);
  }
  createPreElement(e, t, n) {
    const {
      diffIndicators: i,
      disableBackground: r,
      disableLineNumbers: o,
      overflow: s,
    } = this.getOptionsWithDefaults();
    return Qi({
      type: "diff",
      diffIndicators: i,
      disableBackground: r,
      disableLineNumbers: o,
      overflow: s,
      split: e,
      totalLines: t,
      customProperties: n,
    });
  }
  async asyncHighlight(e) {
    const t = Ze(e, this.getTokenizeMaxLength());
    this.computedLang = t ? "text" : (e.lang ?? j(e.name));
    const n = this.highlighter != null && ce(this.getLocalHighlightTheme()),
      i = t || (this.highlighter != null && Ft(this.computedLang));
    return (
      (this.highlighter == null || !n || !i) &&
        (this.highlighter = await this.initializeHighlighter()),
      this.renderDiffWithHighlighter(e, this.highlighter, t)
    );
  }
  renderDiffWithHighlighter(e, t, n = !1) {
    const { options: i } = this.getRenderOptions(e),
      { collapsedContextThreshold: r } = this.getOptionsWithDefaults(),
      o = js(e, t, i, {
        forcePlainText: n,
        expandedHunks: n ? !0 : void 0,
        collapsedContextThreshold: r,
      });
    if (
      this.editSessionActive &&
      e.additionLines.length === 1 &&
      e.additionLines[0] === "" &&
      o.code.additionLines[0] == null
    ) {
      let s;
      if (
        (Fe({
          diff: e,
          diffStyle: "both",
          expandedHunks: n ? !0 : void 0,
          collapsedContextThreshold: r,
          callback: ({ additionLine: a }) => {
            if (a?.lineIndex === 0) return ((s = a), !0);
          },
        }),
        s == null)
      )
        throw new Error("DiffHunksRenderer: missing empty addition line");
      o.code.additionLines[0] = fn(0, "", s.unifiedLineIndex, s.splitLineIndex);
    }
    return { result: o, options: i };
  }
  onHighlightSuccess(e, t, n, i = !0) {
    this.editSessionActive || this.applyHighlightResult(e, t, n, i);
  }
  applyHighlightResult(e, t, n, i = !0) {
    const { diff: r, renderCache: o } = this;
    r == null ||
      o == null ||
      !B(r, e) ||
      !Se(n, this.getRenderOptions(r).options) ||
      ((o.result == null ||
        !o.highlighted ||
        !Se(o.options, n) ||
        !B(o.diff, r)) &&
        ((this.pendingHighlightResult = {
          diff: r,
          options: n,
          highlighted: i,
          result: t,
        }),
        this.onRenderUpdate?.()));
  }
  getMatchingWorkerResultCache(e, t) {
    if (this.editSessionActive) return;
    const n = this.workerManager?.getDiffResultCache(e);
    if (!(n == null || !Se(t, n.options))) return n;
  }
  getReadyRenderResult(e, t) {
    const { pendingHighlightResult: n } = this;
    if (n != null && B(n.diff, e) && Se(n.options, t)) return n;
    const i = this.getMatchingWorkerResultCache(e, t);
    if (!(i == null || this.hasHighlightedRenderCache(e, t)))
      return { diff: e, highlighted: !0, ...i };
  }
  hasHighlightedRenderCache(e, t) {
    const { renderCache: n } = this;
    return (
      n?.result != null && n.highlighted && B(e, n.diff) && Se(t, n.options)
    );
  }
  onHighlightError(e) {
    console.error(e);
  }
  getTokenizeMaxLength() {
    return this.options.tokenizeMaxLength ?? 1e5;
  }
  processDiffResult(e, t, { code: n, themeStyles: i, baseThemeType: r }) {
    const {
        diffStyle: o,
        disableFileHeader: s,
        expandUnchanged: a,
        expansionLineCount: l,
        collapsedContextThreshold: d,
        hunkSeparators: h,
      } = this.getOptionsWithDefaults(),
      c = this.renderCache?.isDirty ?? !1,
      u = o === "unified",
      f = Ta(e, this.options.loadDiffFiles != null),
      p = !e.isPartial || f;
    let g = [],
      m = [],
      y = [];
    const S = [],
      { additionLines: b, deletionLines: L } = n,
      v = {
        rowCount: 0,
        hunkSeparators: h,
        additionsContentAST: g,
        deletionsContentAST: m,
        unifiedContentAST: y,
        unifiedGutterAST: $e(),
        deletionsGutterAST: $e(),
        additionsGutterAST: $e(),
        expansionLineCount: l,
        hunkData: S,
        incrementRowCount(E = 1) {
          v.rowCount += E;
        },
        pushToGutter(E, I) {
          switch (E) {
            case "unified":
              v.unifiedGutterAST.children.push(I);
              break;
            case "deletions":
              v.deletionsGutterAST.children.push(I);
              break;
            case "additions":
              v.additionsGutterAST.children.push(I);
              break;
          }
        },
      },
      k = vr({
        fileDiff: e,
        errorPrefix: "DiffHunksRenderer.processDiffResult",
      }),
      C = {
        size: 0,
        side: void 0,
        increment() {
          this.size += 1;
        },
        flush() {
          if (o !== "unified") {
            if (this.size <= 0 || this.side == null) {
              ((this.side = void 0), (this.size = 0));
              return;
            }
            (this.side === "additions"
              ? (v.pushToGutter("additions", Z(void 0, "buffer", this.size)),
                g?.push(Ct(this.size)))
              : (v.pushToGutter("deletions", Z(void 0, "buffer", this.size)),
                m?.push(Ct(this.size))),
              (this.size = 0),
              (this.side = void 0));
          }
        },
      },
      x = (E, I, D, G, M) => {
        v.pushToGutter(E, Wi(I, D, G, M));
      };
    function A(E) {
      (C.flush(),
        o === "unified"
          ? Zt("unified", E, v)
          : (Zt("deletions", E, v), Zt("additions", E, v)));
    }
    (this.pushFileLevelAnnotations(e, o, t, v),
      Fe({
        diff: e,
        diffStyle: o,
        startingLine: t.startingLine,
        totalLines: t.totalLines,
        expandedHunks: a ? !0 : this.expandedHunks,
        collapsedContextThreshold: d,
        callback: ({
          hunkIndex: E,
          hunk: I,
          collapsedBefore: D,
          collapsedAfter: G,
          additionLine: M,
          deletionLine: P,
          type: z,
        }) => {
          const V = P != null ? P.splitLineIndex : M.splitLineIndex,
            te = M != null ? M.unifiedLineIndex : P.unifiedLineIndex;
          (o === "split" && z !== "change" && C.flush(),
            D > 0 &&
              A({
                hunkIndex: E,
                collapsedLines: D,
                rangeSize: Math.max(I?.collapsedBefore ?? 0, 0),
                hunkSpecs: I?.hunkSpecs,
                isFirstHunk: E === 0,
                isLastHunk: !1,
                isExpandable: p,
              }));
          const ie = o === "unified" ? te : V,
            re = {
              type: z,
              hunkIndex: E,
              lineIndex: ie,
              unifiedLineIndex: te,
              splitLineIndex: V,
              deletionLine: P,
              additionLine: M,
            };
          if (o === "unified") {
            const _ = this.getUnifiedInjectedRowsForLine?.(re);
            _?.before != null && ii(_.before, v);
            let Y = P != null ? L[P.lineIndex] : void 0,
              ne = M != null ? b[M.lineIndex] : void 0;
            if (Y == null && ne == null) {
              const se =
                "DiffHunksRenderer.processDiffResult: deletionLine and additionLine are null, something is wrong";
              throw (console.error(se, { file: e.name }), new Error(se));
            }
            const He =
                z === "change"
                  ? M != null
                    ? "change-addition"
                    : "change-deletion"
                  : z,
              oe = this.getUnifiedLineDecoration({
                type: z,
                lineType: He,
                additionLineIndex: M?.lineIndex,
                deletionLineIndex: P?.lineIndex,
              });
            (x(
              "unified",
              oe.gutterLineType,
              M != null ? M.lineNumber : P.lineNumber,
              `${te},${V}`,
              oe.gutterProperties,
            ),
              ne != null
                ? (ne = wt(
                    ne,
                    oe.contentProperties,
                    c && M != null
                      ? {
                          "data-line": M.lineNumber,
                          "data-line-index": `${te},${V}`,
                        }
                      : void 0,
                  ))
                : Y != null &&
                  (Y = wt(
                    Y,
                    oe.contentProperties,
                    c && P != null
                      ? {
                          "data-line": P.lineNumber,
                          "data-line-index": `${te},${V}`,
                        }
                      : void 0,
                  )),
              kt({
                diffStyle: "unified",
                type: z,
                deletionLine: Y,
                additionLine: ne,
                unifiedSpan: this.getAnnotations(
                  "unified",
                  P?.lineNumber,
                  M?.lineNumber,
                  E,
                  ie,
                ),
                createAnnotationElement: (se) =>
                  this.createAnnotationElement(se),
                context: v,
              }),
              _?.after != null && ii(_.after, v));
          } else {
            const _ = this.getSplitInjectedRowsForLine?.(re);
            _?.before != null && ri(_.before, v, C);
            let Y = P != null ? L[P.lineIndex] : void 0,
              ne = M != null ? b[M.lineIndex] : void 0;
            const He = this.getSplitLineDecoration({
                side: "deletions",
                type: z,
                lineIndex: P?.lineIndex,
              }),
              oe = this.getSplitLineDecoration({
                side: "additions",
                type: z,
                lineIndex: M?.lineIndex,
              });
            if (Y == null && ne == null) {
              const Q =
                "DiffHunksRenderer.processDiffResult: deletionLine and additionLine are null, something is wrong";
              throw (console.error(Q, { file: e.name }), new Error(Q));
            }
            const se = (() => {
              if (z === "change") {
                if (ne == null) return "additions";
                if (Y == null) return "deletions";
              }
            })();
            se != null
              ? (C.side != null && C.side !== se && C.flush(),
                (C.side = se),
                C.increment())
              : z === "change" && C.flush();
            const Ee = this.getAnnotations(
              "split",
              P?.lineNumber,
              M?.lineNumber,
              E,
              ie,
            );
            if ((Ee != null && C.size > 0 && C.flush(), P != null)) {
              const Q = wt(
                Y,
                He.contentProperties,
                c
                  ? {
                      "data-line": P.lineNumber,
                      "data-line-index": `${P.unifiedLineIndex},${V}`,
                    }
                  : void 0,
              );
              (x(
                "deletions",
                He.gutterLineType,
                P.lineNumber,
                `${P.unifiedLineIndex},${V}`,
                He.gutterProperties,
              ),
                Q != null && (Y = Q));
            }
            if (M != null) {
              const Q = wt(
                ne,
                oe.contentProperties,
                c
                  ? {
                      "data-line": M.lineNumber,
                      "data-line-index": `${M.unifiedLineIndex},${V}`,
                    }
                  : void 0,
              );
              (x(
                "additions",
                oe.gutterLineType,
                M.lineNumber,
                `${M.unifiedLineIndex},${V}`,
                oe.gutterProperties,
              ),
                Q != null && (ne = Q));
            }
            (kt({
              diffStyle: "split",
              type: z,
              additionLine: ne,
              deletionLine: Y,
              ...Ee,
              createAnnotationElement: (Q) => this.createAnnotationElement(Q),
              context: v,
            }),
              _?.after != null && ri(_.after, v, C));
          }
          const Ie =
              o === "split" &&
              I != null &&
              V === I.splitLineStart + I.splitLineCount - 1,
            K =
              E === e.hunks.length - 1 &&
              I != null &&
              (o === "split"
                ? V === I.splitLineStart + I.splitLineCount - 1
                : te === I.unifiedLineStart + I.unifiedLineCount - 1),
            _e = Ie ? I.noEOFCRDeletions : !1,
            Xe = Ie ? I.noEOFCRAdditions : !1,
            pe = (P?.noEOFCR ?? !1) || _e,
            ge = (M?.noEOFCR ?? !1) || Xe;
          if (ge || pe) {
            if ((o === "split" && C.flush(), pe)) {
              const _ =
                z === "context" || z === "context-expanded"
                  ? z
                  : "change-deletion";
              o === "unified"
                ? (v.unifiedContentAST.push(bt(_)),
                  v.pushToGutter("unified", Z(_, "metadata", 1)))
                : (v.deletionsContentAST.push(bt(_)),
                  v.pushToGutter("deletions", Z(_, "metadata", 1)),
                  ge ||
                    (v.pushToGutter("additions", Z(void 0, "buffer", 1)),
                    v.additionsContentAST.push(Ct(1))));
            }
            if (ge) {
              const _ =
                z === "context" || z === "context-expanded"
                  ? z
                  : "change-addition";
              o === "unified"
                ? (v.unifiedContentAST.push(bt(_)),
                  v.pushToGutter("unified", Z(_, "metadata", 1)))
                : (v.additionsContentAST.push(bt(_)),
                  v.pushToGutter("additions", Z(_, "metadata", 1)),
                  pe ||
                    (v.pushToGutter("deletions", Z(void 0, "buffer", 1)),
                    v.deletionsContentAST.push(Ct(1))));
            }
            v.incrementRowCount(1);
          }
          (h !== "simple" &&
            h !== "metadata" &&
            (G > 0 || (K && f)) &&
            A({
              hunkIndex: z === "context-expanded" ? E : E + 1,
              collapsedLines: K && f ? "unknown" : G,
              rangeSize: k,
              hunkSpecs: void 0,
              isFirstHunk: !1,
              isLastHunk: !0,
              isExpandable: p,
            }),
            v.incrementRowCount(1));
        },
      }),
      o === "split" && C.flush());
    const w = Math.max(
        Os(e.hunks),
        e.additionLines.length ?? 0,
        e.deletionLines.length ?? 0,
      ),
      O = t.bufferBefore > 0 || t.bufferAfter > 0,
      T = !u && e.type !== "deleted",
      F = !u && e.type !== "new",
      H = v.rowCount > 0 || O;
    ((g = T && H ? g : void 0),
      (m = F && H ? m : void 0),
      (y = u && H ? y : void 0));
    const N = this.createPreElement(m != null && g != null, w);
    return {
      fileDiff: e,
      unifiedGutterAST: u && H ? v.unifiedGutterAST.children : void 0,
      unifiedContentAST: y,
      deletionsGutterAST: F && H ? v.deletionsGutterAST.children : void 0,
      deletionsContentAST: m,
      additionsGutterAST: T && H ? v.additionsGutterAST.children : void 0,
      additionsContentAST: g,
      hunkData: S,
      preNode: N,
      themeStyles: i,
      baseThemeType: r,
      headerElement: s ? void 0 : this.renderHeader(e),
      totalLines: w,
      rowCount: v.rowCount,
      bufferBefore: t.bufferBefore,
      bufferAfter: t.bufferAfter,
      css: "",
    };
  }
  renderCodeAST(e, t) {
    const n =
        e === "unified"
          ? t.unifiedGutterAST
          : e === "deletions"
            ? t.deletionsGutterAST
            : t.additionsGutterAST,
      i =
        e === "unified"
          ? t.unifiedContentAST
          : e === "deletions"
            ? t.deletionsContentAST
            : t.additionsContentAST;
    if (n == null || i == null) return;
    const r = $e(n);
    return (
      (r.properties.style = `grid-row: span ${t.rowCount}`),
      [r, nr(i, t.rowCount)]
    );
  }
  renderFullAST(e, t = []) {
    const n = this.getOptionsWithDefaults().hunkSeparators === "line-info",
      i = this.renderCodeAST("unified", e);
    if (i != null)
      return (
        t.push(
          R({
            tagName: "code",
            children: i,
            properties: {
              "data-code": "",
              "data-container-size": n ? "" : void 0,
              "data-unified": "",
            },
          }),
        ),
        { ...e.preNode, children: t }
      );
    const r = this.renderCodeAST("deletions", e);
    r != null &&
      t.push(
        R({
          tagName: "code",
          children: r,
          properties: {
            "data-code": "",
            "data-container-size": n ? "" : void 0,
            "data-deletions": "",
          },
        }),
      );
    const o = this.renderCodeAST("additions", e);
    return (
      o != null &&
        t.push(
          R({
            tagName: "code",
            children: o,
            properties: {
              "data-code": "",
              "data-container-size": n ? "" : void 0,
              "data-additions": "",
            },
          }),
        ),
      { ...e.preNode, children: t }
    );
  }
  renderFullHTML(e, t = []) {
    return J(this.renderFullAST(e, t));
  }
  renderPartialHTML(e, t) {
    return t == null
      ? J(e)
      : J(
          R({
            tagName: "code",
            children: e,
            properties: {
              "data-code": "",
              "data-container-size":
                this.getOptionsWithDefaults().hunkSeparators === "line-info"
                  ? ""
                  : void 0,
              [`data-${t}`]: "",
            },
          }),
        );
  }
  pushFileLevelAnnotations(e, t, n, i) {
    if (!$t(n)) return;
    const r =
        e.type !== "new"
          ? this.getAnnotationNames(ln(this.deletionAnnotations))
          : [],
      o =
        e.type !== "deleted"
          ? this.getAnnotationNames(ln(this.additionAnnotations))
          : [];
    if (r.length === 0 && o.length === 0) return;
    const s = -1,
      a = -1,
      { createAnnotationElement: l } = this;
    if (t === "unified") {
      kt({
        diffStyle: t,
        type: "context",
        unifiedSpan: {
          type: "annotation",
          hunkIndex: s,
          lineIndex: a,
          annotations: r.concat(o),
        },
        createAnnotationElement: l,
        context: i,
      });
      return;
    }
    kt({
      diffStyle: t,
      type: "context",
      deletionSpan: {
        type: "annotation",
        hunkIndex: s,
        lineIndex: a,
        annotations: r,
      },
      additionSpan: {
        type: "annotation",
        hunkIndex: s,
        lineIndex: a,
        annotations: o,
      },
      createAnnotationElement: l,
      context: i,
    });
  }
  getAnnotations(e, t, n, i, r) {
    const o = {
      type: "annotation",
      hunkIndex: i,
      lineIndex: r,
      annotations: [],
    };
    if (t != null)
      for (const a of this.deletionAnnotations[t] ?? [])
        o.annotations.push(this.annotationSlotName(a));
    const s = {
      type: "annotation",
      hunkIndex: i,
      lineIndex: r,
      annotations: [],
    };
    if (n != null)
      for (const a of this.additionAnnotations[n] ?? [])
        (e === "unified" ? o : s).annotations.push(this.annotationSlotName(a));
    if (e === "unified") return o.annotations.length > 0 ? o : void 0;
    if (!(s.annotations.length === 0 && o.annotations.length === 0))
      return { deletionSpan: o, additionSpan: s };
  }
  getAnnotationNames(e) {
    return e?.map((t) => this.annotationSlotName(t)) ?? [];
  }
  renderHeader(e) {
    const { headerRenderMode: t, stickyHeader: n } =
      this.getOptionsWithDefaults();
    return Yi({ fileOrDiff: e, mode: t, stickyHeader: n });
  }
};
const ka = new Intl.PluralRules("en-US");
function wa(e) {
  return `${e} unmodified line${ka.select(e) === "one" ? "" : "s"}`;
}
function ii(e, t) {
  for (const n of e)
    (t.unifiedContentAST.push(n.content),
      t.pushToGutter("unified", n.gutter),
      t.incrementRowCount(1));
}
function ri(e, t, n) {
  for (const { deletion: i, addition: r } of e) {
    if (i == null && r == null) continue;
    const o =
      i != null && r != null ? void 0 : i == null ? "deletions" : "additions";
    ((o == null || n.side !== o) && n.flush(),
      i != null &&
        (t.deletionsContentAST.push(i.content),
        t.pushToGutter("deletions", i.gutter)),
      r != null &&
        (t.additionsContentAST.push(r.content),
        t.pushToGutter("additions", r.gutter)),
      o != null && ((n.side = o), n.increment()),
      t.incrementRowCount(1));
  }
}
function kt({
  diffStyle: e,
  type: t,
  deletionLine: n,
  additionLine: i,
  unifiedSpan: r,
  deletionSpan: o,
  additionSpan: s,
  createAnnotationElement: a,
  context: l,
}) {
  let d = !1;
  if (e === "unified") {
    if (
      (i != null
        ? l.unifiedContentAST.push(i)
        : n != null && l.unifiedContentAST.push(n),
      r != null)
    ) {
      const h =
        t === "change"
          ? n != null
            ? "change-deletion"
            : "change-addition"
          : t;
      (l.unifiedContentAST.push(a(r)),
        l.pushToGutter("unified", Z(h, "annotation", 1)),
        (d = !0));
    }
  } else if (e === "split") {
    if (
      (n != null && l.deletionsContentAST.push(n),
      i != null && l.additionsContentAST.push(i),
      o != null)
    ) {
      const h =
        t === "change" ? (n != null ? "change-deletion" : "context") : t;
      (l.deletionsContentAST.push(a(o)),
        l.pushToGutter("deletions", Z(h, "annotation", 1)),
        (d = !0));
    }
    if (s != null) {
      const h =
        t === "change" ? (i != null ? "change-addition" : "context") : t;
      (l.additionsContentAST.push(a(s)),
        l.pushToGutter("additions", Z(h, "annotation", 1)),
        (d = !0));
    }
  }
  d && l.incrementRowCount(1);
}
function Zt(
  e,
  {
    hunkIndex: t,
    collapsedLines: n,
    rangeSize: i,
    hunkSpecs: r,
    isFirstHunk: o,
    isLastHunk: s,
    isExpandable: a,
  },
  l,
) {
  if (typeof n == "number" && n <= 0) return;
  const d =
    e === "unified"
      ? l.unifiedContentAST
      : e === "deletions"
        ? l.deletionsContentAST
        : l.additionsContentAST;
  if (l.hunkSeparators === "metadata") {
    r != null &&
      (l.pushToGutter(
        e,
        Be({ type: "metadata", content: r, isFirstHunk: o, isLastHunk: s }),
      ),
      d.push(
        Be({ type: "metadata", content: r, isFirstHunk: o, isLastHunk: s }),
      ),
      e !== "additions" && l.incrementRowCount(1));
    return;
  }
  if (l.hunkSeparators === "simple") {
    t > 0 &&
      (l.pushToGutter(
        e,
        Be({ type: "simple", isFirstHunk: o, isLastHunk: !1 }),
      ),
      d.push(Be({ type: "simple", isFirstHunk: o, isLastHunk: !1 })),
      e !== "additions" && l.incrementRowCount(1));
    return;
  }
  const h = Ps(e, t),
    c = i > l.expansionLineCount,
    u = a ? t : void 0,
    f =
      typeof n == "number" ? wa(n) : "More unchanged context may be available";
  (l.pushToGutter(
    e,
    Be({
      type: l.hunkSeparators,
      content: f,
      expandIndex: u,
      chunked: c,
      slotName: h,
      isFirstHunk: o,
      isLastHunk: s,
    }),
  ),
    d.push(
      Be({
        type: l.hunkSeparators,
        content: f,
        expandIndex: u,
        chunked: c,
        slotName: h,
        isFirstHunk: o,
        isLastHunk: s,
      }),
    ),
    e !== "additions" && l.incrementRowCount(1),
    l.hunkData.push({
      slotName: h,
      hunkIndex: t,
      lines: typeof n == "number" ? n : 0,
      lineCountKnown: typeof n == "number",
      type: e,
      expandable: a ? { up: !o, down: !s, chunked: c } : void 0,
    }));
}
function wt(e, t, n) {
  return e == null || e.type !== "element" || (t == null && n == null)
    ? e
    : { ...e, properties: { ...e.properties, ...t, ...n } };
}
function oi(e) {
  return e.length > 0 && e[e.length - 1] === "" ? e.length - 1 : e.length;
}
function Ea(e, t, n, i) {
  const r = oi(e),
    o = oi(t),
    s = Math.min(r, o);
  let a = 0;
  for (; a < s && e[a] === t[a];) a++;
  let l = 0;
  for (; l < s - a && e[r - 1 - l] === t[o - 1 - l];) l++;
  const d = new Array(t.length);
  for (let h = 0; h < a; h++) d[h] = n[h];
  for (let h = 0; h < l; h++) d[o - 1 - h] = n[r - 1 - h];
  r < e.length && o < t.length && (d[t.length - 1] = n[e.length - 1]);
  for (let h = a; h < t.length; h++) d[h] ??= fn(h, i.getLineText(h));
  return d;
}
function fn(e, t, n = e, i = e) {
  return {
    type: "element",
    tagName: "div",
    properties: {
      "data-line": e + 1,
      "data-line-index": `${n},${i}`,
      "data-line-type": "context",
    },
    children: [
      {
        type: "element",
        tagName: "span",
        properties: { "data-char": 0 },
        children: [{ type: "text", value: t }],
      },
    ],
  };
}
function Ra(e) {
  const t = [];
  for (let n = 0; n < e.lineCount; n++) t.push(e.getLineText(n, !0));
  return t;
}
function Ze(e, t) {
  return Math.max(e.additionLines.length, e.deletionLines.length) > t;
}
function Ta(e, t) {
  return (
    e.isPartial && t && (e.type === "change" || e.type === "rename-changed")
  );
}
function Aa(e, t) {
  return (
    e.lineNumber === t.lineNumber &&
    e.side === t.side &&
    e.metadata === t.metadata
  );
}
function Ia(e, t) {
  return (
    e.slotName === t.slotName &&
    e.hunkIndex === t.hunkIndex &&
    e.lines === t.lines &&
    e.lineCountKnown === t.lineCountKnown &&
    e.type === t.type &&
    e.expandable?.chunked === t.expandable?.chunked &&
    e.expandable?.up === t.expandable?.up &&
    e.expandable?.down === t.expandable?.down
  );
}
async function pn(e, t = 300) {
  let n;
  try {
    await Promise.race([
      e(),
      new Promise((i) => {
        n = setTimeout(i, t);
      }),
    ]);
  } finally {
    n != null && clearTimeout(n);
  }
}
function gn({ oldFile: e, newFile: t }, n) {
  if (!(e === void 0 && t === void 0)) {
    if (e === void 0 || t === void 0)
      throw new Error(
        `${n}: Pass null for an intentionally missing oldFile or newFile side`,
      );
    if (e === null) {
      if (t === null)
        throw new Error(`${n}: You must pass oldFile, newFile, or both`);
      return { oldFile: e, newFile: t };
    }
    return t === null ? { oldFile: e, newFile: t } : { oldFile: e, newFile: t };
  }
}
function Ha(e) {
  return {
    theme: e?.theme,
    disableLineNumbers: e?.disableLineNumbers,
    overflow: e?.overflow,
    collapsed: e?.collapsed,
    disableFileHeader: e?.disableFileHeader,
    disableVirtualizationBuffers: e?.disableVirtualizationBuffers,
    stickyHeader: e?.stickyHeader,
    preferredHighlighter: e?.preferredHighlighter,
    useCSSClasses: e?.useCSSClasses,
    useTokenTransformer: Hn(e),
    tokenizeMaxLineLength: e?.tokenizeMaxLineLength,
    tokenizeMaxLength: e?.tokenizeMaxLength,
    diffStyle: e?.diffStyle,
    diffIndicators: e?.diffIndicators,
    disableBackground: e?.disableBackground,
    hunkSeparators:
      typeof e?.hunkSeparators == "function" ? "custom" : e?.hunkSeparators,
    expandUnchanged: e?.expandUnchanged,
    loadDiffFiles: e?.loadDiffFiles,
    collapsedContextThreshold: e?.collapsedContextThreshold,
    lineDiffType: e?.lineDiffType,
    maxLineDiffLength: e?.maxLineDiffLength,
    expansionLineCount: e?.expansionLineCount,
    headerRenderMode: e?.renderCustomHeader != null ? "custom" : "default",
  };
}
function Ma(e) {
  return (
    e.isPartial &&
    (e.type === "change" ||
      e.type === "rename-changed" ||
      e.type === "rename-pure")
  );
}
function Da(e) {
  const t = { ...e };
  return (delete t.cacheKey, t);
}
function si(e, t) {
  if (e.isPartial || t.isPartial)
    throw new Error(
      "FileDiff.shouldResetEditorForExternalDiff: diffs must be fully hydrated",
    );
  const n = e.lang ?? j(e.name),
    i = t.lang ?? j(t.name),
    r = e.type !== "new",
    o = t.type !== "new";
  return e.name !== t.name ||
    n !== i ||
    r !== o ||
    e.deletionLines.length !== t.deletionLines.length
    ? !0
    : e.deletionLines.some((s, a) => s !== t.deletionLines[a]);
}
function Fa(e, t) {
  return e.length === t.length && e.every((n, i) => n === t[i]);
}
function Pa(e, t) {
  const { oldFile: n } = e;
  return n == null
    ? t.type === "new"
    : t.type !== "new" &&
        n.name === (t.prevName ?? t.name) &&
        Fa(n.lines, t.deletionLines);
}
let Oa = -1;
var Rr = class {
  options;
  workerManager;
  isContainerManaged;
  static LoadedCustomComponent = !0;
  __id = `file-diff:${++Oa}`;
  type = "file-diff";
  fileContainer;
  spriteSVG;
  pre;
  codeUnified;
  codeDeletions;
  codeAdditions;
  bufferBefore;
  bufferAfter;
  themeCSSStyle;
  appliedThemeCSS;
  hasAdoptedThemeCSS = !1;
  unsafeCSSStyle;
  appliedUnsafeCSS;
  gutterUtilityContent;
  headerElement;
  headerPrefix;
  headerFilenameSuffix;
  headerMetadata;
  headerCustom;
  separatorCache = new Map();
  errorWrapper;
  placeHolder;
  hunksRenderer;
  resizeManager;
  scrollSyncManager;
  interactionManager;
  annotationCache = new Map();
  lineAnnotations = [];
  managersDirty = !1;
  deletionFile;
  additionFile;
  fileDiff;
  editSession;
  renderedDiff;
  renderRange;
  pendingFiles;
  appliedPreAttributes;
  headerCache = { lastRenderedHTML: void 0, html: void 0, fileDiff: void 0 };
  lastRowCount;
  mounted = !1;
  enabled = !0;
  editor;
  refreshViewTimeout;
  lineStateRefreshPending = !1;
  deferredSelectedLines;
  deferredEditorActiveLine;
  constructor(e = { theme: U }, t, n = !1) {
    ((this.options = e),
      (this.workerManager = t),
      (this.isContainerManaged = n),
      (this.hunksRenderer = this.createHunksRenderer(e)),
      (this.resizeManager = new ji()),
      (this.scrollSyncManager = new Fs()),
      (this.interactionManager = new qi(
        "diff",
        st(
          e,
          typeof e.hunkSeparators == "function" ||
            (e.hunkSeparators ?? "line-info") === "line-info" ||
            e.hunkSeparators === "line-info-basic"
            ? this.handleExpandHunk
            : void 0,
          this.getLineIndex,
        ),
      )),
      this.workerManager?.subscribeToThemeChanges(this),
      (this.enabled = !0));
  }
  handleHighlightRender = () => {
    this.rerender();
  };
  getTheme() {
    return (
      this.workerManager?.getDiffRenderOptions().theme ??
      this.options.theme ??
      U
    );
  }
  getHunksRendererOptions(e) {
    return Ha(e);
  }
  createHunksRenderer(e) {
    return new Er(
      this.getHunksRendererOptions(e),
      this.getAnnotationSlotName,
      this.handleHighlightRender,
      this.workerManager,
    );
  }
  getLineIndex = (e, t = "additions") =>
    this.getLineIndexForDiff(this.getDiffForLineIndex(), e, t);
  getDiffForLineIndex() {
    return this.getRenderedDiff();
  }
  getLineIndexForDiff(e, t, n) {
    if (e == null) return;
    const i = e.hunks.at(-1);
    let r, o;
    e: for (const s of e.hunks) {
      const a = n === "deletions" ? s.deletionStart : s.additionStart,
        l = n === "deletions" ? s.deletionCount : s.additionCount;
      let d = xe(a, l) + 1,
        h = s.splitLineStart,
        c = s.unifiedLineStart;
      if (t < d) {
        const u = d - t;
        ((r = Math.max(c - u, 0)), (o = Math.max(h - u, 0)));
        break e;
      }
      if (t >= d + l) {
        if (s === i) {
          const u = t - (d + l);
          ((r = c + s.unifiedLineCount + u), (o = h + s.splitLineCount + u));
          break e;
        }
        continue;
      }
      for (const u of s.hunkContent)
        if (u.type === "context")
          if (t < d + u.lines) {
            const f = t - d;
            ((o = h + f), (r = c + f));
            break e;
          } else ((d += u.lines), (h += u.lines), (c += u.lines));
        else {
          const f = n === "deletions" ? u.deletions : u.additions;
          if (t < d + f) {
            const p = t - d;
            ((r = c + (n === "additions" ? u.deletions : 0) + p), (o = h + p));
            break e;
          } else
            ((d += f),
              (h += Math.max(u.deletions, u.additions)),
              (c += u.deletions + u.additions));
        }
      break e;
    }
    if (!(r == null || o == null)) return [r, o];
  }
  setOptions(e) {
    e != null &&
      ((this.options = e),
      this.clearReusableHeader(),
      this.hunksRenderer.setOptions(this.getHunksRendererOptions(e)),
      this.syncInteractionOptions());
  }
  syncInteractionOptions() {
    this.interactionManager.setOptions(
      st(
        this.options,
        typeof this.options.hunkSeparators == "function" ||
          (this.options.hunkSeparators ?? "line-info") === "line-info" ||
          this.options.hunkSeparators === "line-info-basic"
          ? this.handleExpandHunk
          : void 0,
        this.getLineIndex,
      ),
    );
  }
  mergeOptions(e) {
    this.options = { ...this.options, ...e };
  }
  setThemeType(e) {
    (this.options.themeType ?? "system") !== e &&
      (this.mergeOptions({ themeType: e }), this.applyCachedThemeState(e));
  }
  applyCachedThemeState(e) {
    if (
      typeof this.options.theme == "string" ||
      this.fileContainer == null ||
      this.appliedThemeCSS == null
    )
      return !1;
    const t = this.appliedThemeCSS.baseThemeType ?? e;
    return this.appliedThemeCSS.themeType === t
      ? !1
      : (this.applyThemeState(
          this.fileContainer,
          this.appliedThemeCSS.themeStyles,
          e,
          this.appliedThemeCSS.baseThemeType,
        ),
        !0);
  }
  hasThemeChanged() {
    return (
      this.appliedThemeCSS != null &&
      !Ue(this.appliedThemeCSS.theme, this.getTheme())
    );
  }
  getHoveredLine = () => this.interactionManager.getHoveredLine();
  getAnnotationSlotName = (e) => fr(this.editSession?.annotations, e, fe);
  getLatestAnnotations() {
    return this.editSession?.annotations?.current ?? this.lineAnnotations;
  }
  isNewAnnotations(e) {
    const t = this.editSession?.annotations;
    return e === this.lineAnnotations
      ? !1
      : t == null || (e !== t.provided && e !== t.current);
  }
  setLineAnnotations(e) {
    const t = this.editSession?.annotations;
    if (t == null) {
      this.lineAnnotations = e;
      return;
    }
    this.isNewAnnotations(e) && ((this.lineAnnotations = e), ur(t, e, fe));
  }
  syncEditSessionAnnotationsFromEditor(e) {
    const t = this.editSession?.annotations;
    return t == null || e === t.current
      ? !1
      : ((t.current = e),
        this.hunksRenderer.setLineAnnotations(e),
        this.renderAnnotations(),
        !0);
  }
  canPartiallyRender(e, t, n) {
    return !(e || t || n || typeof this.options.hunkSeparators == "function");
  }
  setSelectedLines(e, t) {
    this.lineStateRefreshPending
      ? (this.deferredSelectedLines = [e, t])
      : this.interactionManager.setSelection(e, t);
  }
  setEditorActiveLine(e, t) {
    this.lineStateRefreshPending
      ? (this.deferredEditorActiveLine = [e, t])
      : this.interactionManager.setEditorActiveLine(e, {
          lineNumberOnly: t?.lineNumberOnly,
          side: t?.side ?? "additions",
        });
  }
  flushDeferredLineState() {
    const { deferredEditorActiveLine: e, deferredSelectedLines: t } = this;
    ((this.lineStateRefreshPending = !1),
      (this.deferredEditorActiveLine = void 0),
      (this.deferredSelectedLines = void 0),
      e != null && this.setEditorActiveLine(...e),
      t != null && this.interactionManager.setSelection(...t));
  }
  flushManagers() {
    if (!this.managersDirty || this.pre == null) {
      this.managersDirty = !1;
      return;
    }
    const { diffStyle: e = "split", overflow: t = "scroll" } = this.options;
    (this.interactionManager.setup(this.pre),
      this.resizeManager.setup(this.pre, {
        disableAnnotations: t === "wrap",
        columnVariables: this.shouldApplyColumnVariables(t)
          ? "apply"
          : "measure",
      }),
      t === "scroll" && e === "split"
        ? this.scrollSyncManager.setup(
            this.pre,
            this.codeDeletions,
            this.codeAdditions,
          )
        : this.scrollSyncManager.cleanUp(),
      (this.managersDirty = !1));
  }
  shouldApplyColumnVariables(e) {
    return typeof this.options.hunkSeparators == "function"
      ? !0
      : e === "scroll" &&
          (this.getLatestAnnotations().length > 0 ||
            this.pre?.hasAttribute("data-has-merge-conflict") === !0);
  }
  getCodeScrollLeft() {
    return Math.max(
      this.codeUnified?.scrollLeft ?? 0,
      this.codeDeletions?.scrollLeft ?? 0,
      this.codeAdditions?.scrollLeft ?? 0,
    );
  }
  setCodeScrollLeft(e) {
    (this.codeUnified != null && (this.codeUnified.scrollLeft = e),
      this.codeAdditions != null && (this.codeAdditions.scrollLeft = e),
      this.codeDeletions != null && (this.codeDeletions.scrollLeft = e));
  }
  __getEffectiveCodeOptions() {
    return { ...this.options, ...this.hunksRenderer.getEffectiveCodeOptions() };
  }
  cleanUp(e = !1) {
    const { editor: t } = this;
    (Ne(this.handleEditSessionRender),
      this.emitPostRender(!0),
      t?.cleanUp(e ? "recycle" : "discard"),
      e || (this.editor = void 0),
      this.resizeManager.cleanUp(),
      this.interactionManager.cleanUp(),
      this.scrollSyncManager.cleanUp(),
      (this.managersDirty = !1),
      this.workerManager?.unsubscribeToThemeChanges(this),
      (this.renderRange = void 0),
      e || (this.pendingFiles = void 0),
      this.isContainerManaged || this.fileContainer?.remove(),
      (this.fileContainer = void 0),
      (this.mounted = !1),
      e || (this.lineAnnotations = []),
      this.clearAuxiliaryNodes(),
      this.annotationCache.clear(),
      (this.pre = void 0),
      (this.codeUnified = void 0),
      (this.codeDeletions = void 0),
      (this.codeAdditions = void 0),
      this.bufferBefore?.remove(),
      (this.bufferBefore = void 0),
      this.bufferAfter?.remove(),
      (this.bufferAfter = void 0),
      (this.appliedPreAttributes = void 0),
      (this.headerElement = void 0),
      (this.headerPrefix = void 0),
      (this.headerFilenameSuffix = void 0),
      (this.headerMetadata = void 0),
      (this.headerCustom = void 0),
      this.placeHolder?.remove(),
      (this.placeHolder = void 0),
      (this.headerCache.lastRenderedHTML = void 0),
      e || this.clearReusableHeader(),
      this.errorWrapper?.remove(),
      (this.errorWrapper = void 0),
      (this.spriteSVG = void 0),
      (this.lastRowCount = void 0),
      (this.themeCSSStyle = void 0),
      (this.appliedThemeCSS = void 0),
      (this.hasAdoptedThemeCSS = !1),
      (this.unsafeCSSStyle = void 0),
      (this.appliedUnsafeCSS = void 0),
      e
        ? this.hunksRenderer.recycle()
        : (this.hunksRenderer.cleanUp(),
          (this.workerManager = void 0),
          (this.fileDiff = void 0),
          (this.editSession = void 0),
          (this.renderedDiff = void 0),
          (this.deletionFile = void 0),
          (this.additionFile = void 0)),
      this.refreshViewTimeout != null &&
        (clearTimeout(this.refreshViewTimeout),
        (this.refreshViewTimeout = void 0)),
      (this.lineStateRefreshPending = !1),
      (this.deferredEditorActiveLine = void 0),
      (this.deferredSelectedLines = void 0),
      (this.enabled = !1));
  }
  virtualizedSetup() {
    ((this.enabled = !0), this.workerManager?.subscribeToThemeChanges(this));
  }
  hydrate({
    fileContainer: e,
    prerenderedHTML: t,
    preventEmit: n = !1,
    lineAnnotations: i,
    fileDiff: r,
    ...o
  }) {
    if (!this.enabled)
      throw new Error(
        "FileDiff.hydrate: attempting to call hydrate after cleaned up",
      );
    if (this.fileContainer != null)
      throw new Error(
        "FileDiff.hydrate: hydrate can only be called before the instance has rendered or hydrated",
      );
    const s = gn(o, "FileDiff.hydrate"),
      a = s?.oldFile,
      l = s?.newFile;
    this.hydrateElements(e, t);
    const d = this.editor != null;
    (d ||
    Ua(
      this.pre,
      Na({ fileDiff: r, oldFile: a, newFile: l }),
      this.options.collapsed,
    ) ||
    _a(
      this.headerElement,
      za({ fileDiff: r, oldFile: a, newFile: l }),
      this.options.disableFileHeader,
    )
      ? this.render({
          ...o,
          fileContainer: e,
          lineAnnotations: i,
          fileDiff: r,
          forceRender: d || o.forceRender,
          preventEmit: !0,
        })
      : this.hydrationSetup({ fileDiff: r, lineAnnotations: i, ...s }),
      n || this.emitPostRender());
  }
  hydrateElements(e, t) {
    (this.fileContainer !== e && this.emitPostRender(!0), cr(e, t));
    for (const n of e.shadowRoot?.children ?? []) {
      if (n instanceof SVGElement) {
        this.spriteSVG = n;
        continue;
      }
      if (n instanceof HTMLElement) {
        if (n instanceof HTMLPreElement) {
          this.pre = n;
          for (const i of n.children)
            !(i instanceof HTMLElement) ||
              i.tagName.toLowerCase() !== "code" ||
              ("deletions" in i.dataset && (this.codeDeletions = i),
              "additions" in i.dataset && (this.codeAdditions = i),
              "unified" in i.dataset && (this.codeUnified = i));
          continue;
        }
        if ("diffsHeader" in n.dataset) {
          this.headerElement = n;
          continue;
        }
        if (n instanceof HTMLStyleElement && n.hasAttribute("data-theme-css")) {
          this.themeCSSStyle = n;
          continue;
        }
        if (
          n instanceof HTMLStyleElement &&
          n.hasAttribute("data-unsafe-css")
        ) {
          ((this.unsafeCSSStyle = n), (this.appliedUnsafeCSS = n.textContent));
          continue;
        }
      }
    }
    (this.pre != null &&
      (this.syncCodeNodesFromPre(this.pre),
      this.pre.removeAttribute("data-dehydrated")),
      (this.fileContainer = e),
      this.hydrateMeasuredScrollbar());
  }
  hydrationSetup({ fileDiff: e, oldFile: t, newFile: n, lineAnnotations: i }) {
    ((this.lineAnnotations = i ?? this.lineAnnotations),
      (this.additionFile = n),
      (this.deletionFile = t),
      (this.fileDiff =
        e ??
        (t !== void 0 && n !== void 0
          ? Oe(t, n, this.options.parseDiffOptions)
          : void 0)),
      this.pre != null &&
        (this.syncInteractionOptions(),
        this.hunksRenderer.hydrate(this.fileDiff),
        (this.renderedDiff = this.fileDiff),
        this.renderAnnotations(),
        this.renderGutterUtility(),
        this.injectUnsafeCSS(),
        (this.managersDirty = !0),
        this.flushManagers()));
  }
  rerender() {
    !this.enabled ||
      (this.fileDiff == null &&
        this.additionFile == null &&
        this.deletionFile == null) ||
      this.render({ forceRender: !0, renderRange: this.renderRange });
  }
  onThemeChange() {
    (this.hunksRenderer.clearRenderCache(), this.rerender());
  }
  handleExpandHunk = (e, t, n) => {
    this.expandHunk(e, t, n);
  };
  expandHunk = (e, t, n) => {
    (this.hunksRenderer.expandHunk(e, t, n),
      this.loadFilesIfNecessary(),
      this.rerender());
  };
  loadFilesIfNecessary() {
    const {
      fileDiff: e,
      options: { loadDiffFiles: t },
    } = this;
    if (e == null || t == null || !Ma(e) || this.pendingFiles?.fileDiff === e)
      return;
    const n = this.loadFilesForDiff(e, t),
      i = (this.pendingFiles = { fileDiff: e, promise: n }),
      r = () => {
        this.pendingFiles === i && (this.pendingFiles = void 0);
      };
    i.promise = n.finally(r);
  }
  async loadFilesForDiff(e, t) {
    try {
      const n = await t(e);
      if (!this.enabled || this.fileDiff !== e) return;
      await this.handleFilesLoaded(e, n);
    } catch (n) {
      if (this.options.disableErrorHandling === !0) throw n;
      console.error(n);
    }
  }
  async handleFilesLoaded(e, t) {
    if (!(this.fileDiff !== e || !e.isPartial)) {
      if (
        (hn("merge", e, t),
        this.setHydratedState(t),
        this.startHydratedEditSession(e))
      ) {
        this.rerender();
        return;
      }
      (await pn(() => this.primeHighlightCache(e)),
        !(!this.enabled || this.fileDiff !== e) && this.rerender());
    }
  }
  startHydratedEditSession(e) {
    if (e.isPartial)
      throw new Error(
        "FileDiff.startHydratedEditSession: diffs cannot be partial for editing",
      );
    if (this.fileDiff !== e) return !1;
    const { editor: t } = this;
    return this.editSession?.outgoingDiff != null
      ? (this.installEditSession(e), !0)
      : t == null || this.editSession != null
        ? !1
        : (this.installEditSession(
            e,
            t.__getDocumentContents(Jt(e)),
            t.__getDocumentSessionState(),
          ),
          !0);
  }
  updateExternalDiff(e, t) {
    if (B(this.fileDiff, e)) return !1;
    const n = this.editSession?.outgoingDiff ?? this.editSession?.diff;
    return (
      (this.fileDiff = e),
      n != null
        ? (e.isPartial
            ? this.loadFilesIfNecessary()
            : this.installEditSession(e),
          this.editSession != null && (this.editSession.outgoingDiff = n))
        : this.editor != null &&
          (e.isPartial
            ? this.loadFilesIfNecessary()
            : this.installEditSession(
                e,
                this.editor.__getDocumentContents(Jt(e)),
                this.editor.__getDocumentSessionState(),
              )),
      this.editSession?.annotations != null &&
        t != null &&
        ((this.lineAnnotations = t),
        (this.editSession.annotations = zt(
          t,
          fe,
          this.editSession.annotations,
        ))),
      !0
    );
  }
  installEditSession(e, t, n) {
    const i = e.additionLines.join(""),
      r = t != null ? lt(t.contents) : void 0,
      o = n != null && Pa(n, e);
    if (n != null && !o)
      throw new Error(
        "FileDiff: retained session cannot resume against a different old file",
      );
    if (t != null && t.contents !== i && !o)
      throw new Error(
        "FileDiff: retained edits are missing their diff session state",
      );
    const s =
        t == null ||
        (t.name === e.name && t.lang === e.lang && t.contents === i),
      a = Da(e);
    (t != null && r != null && !s && ((a.name = t.name), (a.lang = t.lang)),
      o &&
        n != null &&
        r != null &&
        ((a.additionLines = r),
        (a.type = n.type),
        (a.hunks = n.hunks),
        (a.editSessionDirty = !0),
        Gt(a)),
      (this.editSession = {
        diff: a,
        annotations:
          this.editSession?.annotations ?? zt(this.lineAnnotations, fe),
        outgoingDiff: this.editSession?.outgoingDiff,
      }),
      this.hunksRenderer.beginEditSession(a, s && !o ? e : void 0));
  }
  setHydratedState(e) {
    ((this.deletionFile = e.oldFile),
      (this.additionFile = e.newFile),
      this.workerManager?.cleanUpTasks(this.hunksRenderer),
      this.hunksRenderer.clearRenderCache());
  }
  render({
    fileDiff: e,
    deferManagers: t = !1,
    forceRender: n = !1,
    preventEmit: i = !1,
    lineAnnotations: r,
    fileContainer: o,
    containerWrapper: s,
    renderRange: a,
    ...l
  }) {
    if (!this.enabled)
      throw new Error(
        "FileDiff.render: attempting to call render after cleaned up",
      );
    const d = gn(l, "FileDiff.render"),
      h = d?.oldFile,
      c = d?.newFile;
    this.editor?.__postponeBgTokenizeToNextFrame();
    const {
        collapsed: u = !1,
        themeType: f = "system",
        expandUnchanged: p = !1,
      } = this.options,
      g = u ? void 0 : a,
      m = this.hasThemeChanged(),
      y = d != null,
      S = y && (!ai(h, this.deletionFile) || !ai(c, this.additionFile));
    let b = e != null && !B(e, this.fileDiff);
    const L =
      r != null && (r.length > 0 || this.getLatestAnnotations().length > 0)
        ? this.isNewAnnotations(r)
        : !1;
    if (
      !u &&
      Vt(g, this.renderRange) &&
      !n &&
      !L &&
      !m &&
      ((e != null && !b) || (e == null && !S))
    ) {
      const w = this.applyCachedThemeState(f);
      return (w && this.finalizeRender(), w);
    }
    let v;
    e == null &&
      y &&
      (S || this.fileDiff == null) &&
      (v = Oe(d.oldFile, d.newFile, this.options.parseDiffOptions));
    const { renderRange: k } = this;
    ((this.renderRange = g),
      y
        ? ((this.deletionFile = h), (this.additionFile = c))
        : e != null &&
          ((this.deletionFile = void 0), (this.additionFile = void 0)),
      e != null && b
        ? this.updateExternalDiff(e, r)
        : v != null && ((b = !0), this.updateExternalDiff(v, r)),
      b && this.clearReusableHeader(),
      r != null && this.setLineAnnotations(r));
    const C = this.getLatestDiff();
    if (C == null) return !1;
    (C.editSessionDirty === !0 &&
      this.shouldSelfHealEditSession() &&
      (Jn(C, this.options.parseDiffOptions),
      this.hunksRenderer.refreshHighlightedResult()),
      p && this.loadFilesIfNecessary(),
      this.hunksRenderer.setOptions(this.getHunksRendererOptions(this.options)),
      this.syncInteractionOptions(),
      this.hunksRenderer.setLineAnnotations(this.getLatestAnnotations()));
    const { disableErrorHandling: x = !1, disableFileHeader: A = !1 } =
      this.options;
    if (
      (A &&
        (this.headerElement != null &&
          (this.headerElement.remove(),
          (this.headerElement = void 0),
          (this.headerCache.lastRenderedHTML = void 0)),
        this.clearHeaderSlots()),
      (o = this.getOrCreateFileContainer(o, s)),
      this.applyCachedThemeState(f),
      u)
    ) {
      (this.removeRenderedCode(), this.clearAuxiliaryNodes());
      try {
        const w = this.hunksRenderer.renderDiff(C, _i);
        (w != null &&
          this.applyThemeState(o, w.themeStyles, f, w.baseThemeType),
          w?.headerElement != null &&
            this.applyHeaderToDOM(w.headerElement, o, w.fileDiff),
          this.renderSeparators([]),
          (this.renderedDiff = w?.fileDiff ?? C),
          this.injectUnsafeCSS());
      } catch (w) {
        if (x) throw w;
        (console.error(w), w instanceof Error && this.applyErrorToDOM(w, o));
      }
      return (this.finalizeRender(), i || this.emitPostRender(), !0);
    }
    try {
      const w = this.getOrCreatePreNode(o);
      if (!(
        this.canPartiallyRender(
          n,
          L,
          S || b || m || !B(this.renderedDiff, C),
        ) &&
        this.applyPartialRender({
          fileDiff: C,
          previousRenderRange: k,
          renderRange: g,
        })
      )) {
        const O = this.hunksRenderer.renderDiff(C, g);
        if (O == null)
          return (
            this.workerManager?.isInitialized() === !1 &&
              this.workerManager.isWorkingPool() &&
              this.workerManager
                .initialize()
                .catch(() => {})
                .then(() => this.rerender()),
            !1
          );
        (this.applyThemeState(o, O.themeStyles, f, O.baseThemeType),
          O.headerElement != null &&
            this.applyHeaderToDOM(O.headerElement, o, O.fileDiff),
          O.additionsContentAST != null ||
          O.deletionsContentAST != null ||
          O.unifiedContentAST != null
            ? this.applyHunksToDOM(w, O)
            : this.pre != null && (this.pre.remove(), (this.pre = void 0)),
          this.renderSeparators(O.hunkData),
          (this.renderedDiff = O.fileDiff));
      }
      (this.applyBuffers(w, g),
        this.injectUnsafeCSS(),
        this.renderAnnotations(),
        this.renderGutterUtility(),
        (this.managersDirty = !0),
        t || this.flushManagers(),
        this.finalizeRender(),
        this.editor != null && this.syncRenderViewToEditor());
    } catch (w) {
      if (x) throw w;
      (console.error(w), w instanceof Error && this.applyErrorToDOM(w, o));
    }
    return (i || this.emitPostRender(), !0);
  }
  finalizeRender() {}
  emitPostRender(e = !1) {
    const {
      fileContainer: t,
      options: { onPostRender: n },
    } = this;
    if (e) {
      if (!this.mounted || ((this.mounted = !1), t == null)) return;
      this.options.onPostRender?.(t, this, "unmount");
      return;
    }
    if (t == null) return;
    const i = this.mounted ? "update" : "mount";
    ((this.mounted = !0), n?.(t, this, i));
  }
  getLatestDiff(e = this.fileDiff) {
    return this.editSession?.diff ?? e;
  }
  getRenderedDiff() {
    return this.renderedDiff;
  }
  syncRenderViewToEditor() {
    const { editor: e, fileContainer: t } = this,
      n = this.getLatestAnnotations(),
      i = this.computeEditorRenderRange(this.renderRange),
      r = this.getLatestDiff();
    if (e == null || t == null || r == null || r.isPartial) return;
    const o = (d) => {
        if (
          !this.enabled ||
          this.editor !== e ||
          this.fileContainer !== t ||
          this.getLatestDiff() !== r
        )
          return;
        const h = this.editSession?.outgoingDiff,
          c = this.fileDiff,
          u = h != null && c != null && h !== r,
          f = u ? si(h, c) : !1;
        e.__syncRenderView({
          highlighter: d,
          fileContainer: t,
          fileDiff: r,
          lineAnnotations: n,
          renderRange: i,
          externalDocument: u,
          resetHistory: f,
        });
      },
      s = this.getTheme(),
      a = r.lang ?? j(r.name),
      l = Ge({ theme: s, lang: a });
    l != null
      ? o(l)
      : ct({
          themes: ht(s),
          langs: ["text", a],
          preferredHighlighter:
            this.workerManager?.getPreferredHighlighter() ??
            this.options.preferredHighlighter,
        }).then(o);
  }
  computeEditorRenderRange(e) {
    const t = this.getLatestDiff();
    if (e == null || t == null || Nt(e)) return e;
    const {
      diffStyle: n = "split",
      expandUnchanged: i = !1,
      collapsedContextThreshold: r = 1,
    } = this.options;
    let o, s;
    return (
      Fe({
        diff: t,
        diffStyle: n,
        startingLine: e.startingLine,
        totalLines: e.totalLines,
        expandedHunks: i ? !0 : this.hunksRenderer.getExpandedHunksMap(),
        collapsedContextThreshold: r,
        callback: ({ additionLine: a }) => {
          a != null && ((o ??= a.lineNumber), (s = a.lineNumber));
        },
      }),
      o == null || s == null
        ? { ...e, startingLine: 0, totalLines: 0 }
        : { ...e, startingLine: o - 1, totalLines: s - o + 1 }
    );
  }
  __acknowledgeDocumentUpdate() {
    this.editSession != null && (this.editSession.outgoingDiff = void 0);
  }
  __acceptEditorChange(e) {
    const { lineAnnotations: t } = e;
    t != null && this.syncEditSessionAnnotationsFromEditor(t);
  }
  emitEditChange(e) {
    const { onEditChange: t } = this.options;
    t?.(e);
  }
  __captureDocumentSessionState(e = !0) {
    let { fileDiff: t, editSession: { diff: n, outgoingDiff: i } = {} } = this;
    if (i != null && t != null) {
      if (!t.isPartial && si(i, t)) return;
      n = i;
    }
    if (!(n == null || n.isPartial))
      return {
        diffSession: {
          oldFile:
            n.type !== "new"
              ? {
                  name: n.prevName ?? n.name,
                  lines: e ? [...n.deletionLines] : n.deletionLines,
                }
              : null,
          type: n.type,
          hunks: e ? gr(n.hunks) : n.hunks,
        },
        hasChanges: n.editSessionDirty === !0,
      };
  }
  __attachEditor(e) {
    if (this.type !== "file-diff")
      throw new Error(
        `FileDiff.__attachEditor: cannot attach an editor to a "${this.type}" diff`,
      );
    if (this.editor != null)
      throw new Error("FileDiff.__attachEditor: an editor is already attached");
    const t = () => {
      ((this.editor = void 0), this.finishEditSession());
    };
    try {
      return (this.resumeEditorRendering(e), t);
    } catch (n) {
      throw (t(), n);
    }
  }
  __resumeEditor(e) {
    if (this.editor !== e)
      throw new Error("FileDiff.__resumeEditor: editor association changed");
    this.resumeEditorRendering(e);
  }
  resumeEditorRendering(e) {
    const { fileDiff: t } = this,
      n = this.editSession?.outgoingDiff;
    n != null &&
      t != null &&
      !t.isPartial &&
      this.editSession?.diff === n &&
      this.installEditSession(t);
    const i =
      this.editSession == null && t != null && !t.isPartial ? t : void 0;
    (i != null
      ? this.installEditSession(
          i,
          e.__getDocumentContents(Jt(i)),
          e.__getDocumentSessionState(),
        )
      : this.editSession != null &&
        this.hunksRenderer.beginEditSession(this.editSession.diff),
      (this.editor = e),
      this.fileDiff?.isPartial === !0 && this.loadFilesIfNecessary());
    const r = this.editSession?.diff;
    this.hunksRenderer.editorRenderReady()
      ? (r != null &&
          this.hunksRenderer.diffCache === r &&
          (this.renderedDiff = r),
        this.syncRenderViewToEditor())
      : this.rerender();
  }
  finishEditSession() {
    (this.hunksRenderer.endEditSession(), this.finalizeEditSessionHunks());
  }
  __completeEditSession(e, t) {
    this.settleEditSession(t === "install", e);
  }
  settleEditSession(e, t) {
    const { editSession: n, fileDiff: i, lineAnnotations: r } = this;
    if (n == null || i == null) return;
    const { diff: o, annotations: s } = n;
    if (this.editor != null)
      throw new Error(
        "FileDiff.__completeEditSession: detach the editor before completing the session",
      );
    (this.hunksRenderer.endEditSession(), this.finalizeEditSessionHunks());
    const a = s?.current;
    let l,
      d = null,
      h = null,
      c = !1,
      u;
    if (t == null)
      throw new Error(
        "FileDiff.__completeEditSession: editor is required for completion",
      );
    const f = mr(o),
      p = { name: f.name, contents: f.additionLines.join("") };
    f.lang != null && (p.lang = f.lang);
    const g = {
      fileDiff: f,
      editor: t,
      originalFileDiff: i,
      oldFile:
        f.type === "new"
          ? null
          : { name: f.prevName ?? f.name, contents: f.deletionLines.join("") },
      newFile: p,
      lineAnnotations: a,
      originalLineAnnotations: r,
    };
    Object.freeze(g);
    try {
      if (
        (t.__emitEditComplete(g), this.options.onEditComplete?.(g) === "accept")
      ) {
        if (f.cacheKey != null && f.cacheKey === i.cacheKey)
          throw new Error(
            "FileDiff.__completeEditSession: an accepted diff must not reuse the replaced diff cacheKey",
          );
        ((l = f), (d = g.oldFile), (h = g.newFile));
      }
    } catch (m) {
      ((c = !0), (u = m));
    }
    if (
      (e &&
        l != null &&
        ((this.fileDiff = l),
        this.additionFile != null &&
          ((this.deletionFile = d), (this.additionFile = h)),
        a != null && (this.lineAnnotations = a)),
      (this.editSession = void 0),
      e && this.fileContainer != null && this.rerender(),
      c)
    )
      throw u;
  }
  finalizeEditSessionHunks() {
    const e = this.getLatestDiff();
    if (e == null || e.editSessionDirty !== !0) return !1;
    const { collapsedContextThreshold: t = 1 } = this.options,
      n = ua(e, this.hunksRenderer.getExpandedHunksMap(), t);
    return (
      Jn(e, this.options.parseDiffOptions),
      this.hunksRenderer.setExpandedHunksMap(fa(e, n)),
      this.hunksRenderer.refreshHighlightedResult(),
      this.escalateEditSessionRender(),
      !0
    );
  }
  applyDocumentChange(e, t) {
    const n = this.editSession?.diff;
    if (n == null)
      throw new Error(
        "FileDiff.applyDocumentChange: requires an active edit session",
      );
    (this.detachAdditionLines(),
      this.hunksRenderer.beginEditSession(n),
      this.hunksRenderer.applyDocumentChange(e),
      t != null && this.syncEditSessionAnnotationsFromEditor(t),
      this.rerender(),
      this.interactionManager.setSelectionDirty());
  }
  updateRenderCache(e, t, n = {}) {
    const i = this.editSession?.diff;
    if (i == null)
      throw new Error(
        "FileDiff.updateRenderCache: requires an active edit session",
      );
    (this.detachAdditionLines(), this.hunksRenderer.beginEditSession(i));
    const { shouldRefreshDiffsView: r, lineCountChangeInFlight: o } = n;
    if (this.hunksRenderer.updateRenderCache(e, t, o)) {
      (this.refreshViewTimeout != null &&
        (clearTimeout(this.refreshViewTimeout),
        (this.refreshViewTimeout = void 0)),
        (this.lineStateRefreshPending = !0),
        this.escalateEditSessionRender());
      return;
    }
    r === !0 &&
      (this.refreshViewTimeout != null && clearTimeout(this.refreshViewTimeout),
      (this.lineStateRefreshPending = !0),
      (this.refreshViewTimeout = setTimeout(() => {
        ((this.refreshViewTimeout = void 0),
          this.options.diffStyle === "split"
            ? this.refreshSplitDiffView()
            : this.refreshUnifiedDiffView(),
          this.flushDeferredLineState());
      }, 150)));
  }
  detachAdditionLines() {
    const e = this.editSession?.diff,
      { fileDiff: t } = this;
    e != null &&
      (e.additionLines === t?.additionLines ||
        e.additionLines === e.deletionLines) &&
      (e.additionLines = [...e.additionLines]);
  }
  isLineRenderable(e) {
    const t = this.getRenderedDiff();
    if (t == null) return !0;
    const { expandUnchanged: n = !1, collapsedContextThreshold: i = 1 } =
      this.options;
    return Cr({
      fileDiff: t,
      lineNumber: e,
      expandedHunks: n ? !0 : this.hunksRenderer.getExpandedHunksMap(),
      collapsedContextThreshold: i,
    });
  }
  getNearestRenderableLine(e, t) {
    const n = this.getRenderedDiff();
    if (n == null) return e;
    const { expandUnchanged: i = !1, collapsedContextThreshold: r = 1 } =
      this.options;
    return zs({
      fileDiff: n,
      lineNumber: e,
      direction: t,
      expandedHunks: i ? !0 : this.hunksRenderer.getExpandedHunksMap(),
      collapsedContextThreshold: r,
    });
  }
  revealLine(e) {
    const t = this.getRenderedDiff(),
      {
        expandUnchanged: n = !1,
        collapsedContextThreshold: i = 1,
        expansionLineCount: r = 100,
      } = this.options;
    if (t == null || t.isPartial || n) return !1;
    const o = this.hunksRenderer.getExpandedHunksMap();
    for (const [d, h] of t.hunks.entries()) {
      const [c, u] = dt(h);
      if (e < c) {
        const f = ke({
            isPartial: t.isPartial,
            rangeSize: h.collapsedBefore,
            expandedHunks: o,
            hunkIndex: d,
            collapsedContextThreshold: i,
          }),
          p = c - f.rangeSize;
        if (f.renderAll || e < p + f.fromStart || e >= c - f.fromEnd) return !1;
        const g = e - (p + f.fromStart) + 1,
          m = c - f.fromEnd - e;
        return (
          g <= m
            ? this.expandHunk(d, "up", g + r)
            : this.expandHunk(d, "down", m + r),
          !0
        );
      }
      if (e < u) return !1;
    }
    const s = we({
      fileDiff: t,
      hunkIndex: t.hunks.length - 1,
      expandedHunks: o,
      collapsedContextThreshold: i,
      errorPrefix: "FileDiff.revealLine",
    });
    if (s == null || s.renderAll) return !1;
    const a = t.hunks[t.hunks.length - 1],
      [, l] = dt(a);
    return e < l + s.fromStart || e >= l + s.rangeSize
      ? !1
      : (this.expandHunk(t.hunks.length, "up", e - (l + s.fromStart) + 1 + r),
        !0);
  }
  shouldSelfHealEditSession() {
    return this.editor == null;
  }
  escalateEditSessionRender() {
    W(this.handleEditSessionRender);
  }
  handleEditSessionRender = () => {
    (this.rerender(), this.flushDeferredLineState());
  };
  removeRenderedCode() {
    (this.resizeManager.cleanUp(),
      this.scrollSyncManager.cleanUp(),
      this.interactionManager.cleanUp(),
      this.bufferBefore?.remove(),
      (this.bufferBefore = void 0),
      this.bufferAfter?.remove(),
      (this.bufferAfter = void 0),
      this.codeUnified?.remove(),
      (this.codeUnified = void 0),
      this.codeDeletions?.remove(),
      (this.codeDeletions = void 0),
      this.codeAdditions?.remove(),
      (this.codeAdditions = void 0),
      this.pre?.remove(),
      (this.pre = void 0),
      (this.appliedPreAttributes = void 0),
      (this.lastRowCount = void 0));
  }
  clearAuxiliaryNodes() {
    for (const { element: e } of this.separatorCache.values()) e.remove();
    this.separatorCache.clear();
    for (const { element: e } of this.annotationCache.values()) e.remove();
    (this.annotationCache.clear(),
      this.gutterUtilityContent?.remove(),
      (this.gutterUtilityContent = void 0));
  }
  renderPlaceholder(e) {
    if (this.fileContainer == null) return !1;
    if (
      (this.emitPostRender(!0),
      this.cleanChildNodes(),
      this.placeHolder == null)
    ) {
      const t =
        this.fileContainer.shadowRoot ??
        this.fileContainer.attachShadow({ mode: "open" });
      ((this.placeHolder = document.createElement("div")),
        (this.placeHolder.dataset.placeholder = ""),
        t.appendChild(this.placeHolder));
    }
    return (this.placeHolder.style.setProperty("height", `${e}px`), !0);
  }
  async primeHighlightCache(e = this.fileDiff) {
    const { workerManager: t } = this;
    if (
      e == null ||
      t == null ||
      !t.isWorkingPool() ||
      e.cacheKey == null ||
      it(e)
    )
      return;
    const n = this.options.tokenizeMaxLength ?? 1e5;
    Math.max(e.additionLines.length, e.deletionLines.length) > n ||
      (await t.primeDiffHighlightCache(e).catch((i) => {
        console.error(i);
      }));
  }
  cleanChildNodes() {
    (this.resizeManager.cleanUp(),
      this.scrollSyncManager.cleanUp(),
      this.interactionManager.cleanUp(),
      this.clearAuxiliaryNodes(),
      this.bufferAfter?.remove(),
      this.bufferBefore?.remove(),
      this.codeAdditions?.remove(),
      this.codeDeletions?.remove(),
      this.codeUnified?.remove(),
      this.errorWrapper?.remove(),
      this.headerElement?.remove(),
      this.headerPrefix?.remove(),
      this.headerFilenameSuffix?.remove(),
      this.headerMetadata?.remove(),
      this.headerCustom?.remove(),
      this.pre?.remove(),
      this.spriteSVG?.remove(),
      this.themeCSSStyle?.remove(),
      this.unsafeCSSStyle?.remove(),
      (this.bufferAfter = void 0),
      (this.bufferBefore = void 0),
      (this.codeAdditions = void 0),
      (this.codeDeletions = void 0),
      (this.codeUnified = void 0),
      (this.errorWrapper = void 0),
      (this.headerElement = void 0),
      (this.headerPrefix = void 0),
      (this.headerFilenameSuffix = void 0),
      (this.headerMetadata = void 0),
      (this.headerCustom = void 0),
      (this.pre = void 0),
      (this.spriteSVG = void 0),
      (this.themeCSSStyle = void 0),
      (this.appliedThemeCSS = void 0),
      (this.hasAdoptedThemeCSS = !1),
      (this.unsafeCSSStyle = void 0),
      (this.appliedUnsafeCSS = void 0),
      (this.headerCache.lastRenderedHTML = void 0),
      (this.lastRowCount = void 0),
      (this.mounted = !1));
  }
  renderSeparators(e) {
    const { hunkSeparators: t } = this.options;
    if (
      this.isContainerManaged ||
      this.fileContainer == null ||
      typeof t != "function"
    ) {
      for (const { element: i } of this.separatorCache.values()) i.remove();
      this.separatorCache.clear();
      return;
    }
    const n = new Map(this.separatorCache);
    for (const i of e) {
      const r = i.slotName;
      let o = this.separatorCache.get(r);
      if (o == null || !Ia(i, o.hunkData)) {
        o?.element.remove();
        const s = document.createElement("div");
        ((s.style.display = "contents"), (s.slot = i.slotName));
        const a = t(i, this);
        (a != null && s.appendChild(a),
          this.fileContainer.appendChild(s),
          (o = { element: s, hunkData: i }),
          this.separatorCache.set(r, o));
      }
      n.delete(r);
    }
    for (const [i, { element: r }] of n.entries())
      (this.separatorCache.delete(i), r.remove());
  }
  renderAnnotations() {
    if (this.isContainerManaged || this.fileContainer == null) {
      for (const { element: i } of this.annotationCache.values()) i.remove();
      this.annotationCache.clear();
      return;
    }
    const e = new Map(this.annotationCache),
      { renderAnnotation: t } = this.options,
      n = this.getLatestAnnotations();
    if (t != null && n.length > 0)
      for (const [i, r] of n.entries()) {
        const o = this.getAnnotationSlotName(r),
          s = `${i}-${o}`;
        let a = this.annotationCache.get(s);
        if (a == null || !Aa(r, a.annotation)) {
          a?.element.remove();
          const l = t(r);
          if (l == null) continue;
          ((a = { element: wn(o), annotation: r }),
            a.element.appendChild(l),
            this.fileContainer.appendChild(a.element),
            this.annotationCache.set(s, a));
        }
        e.delete(s);
      }
    for (const [i, { element: r }] of e.entries())
      (this.annotationCache.delete(i), r.remove());
  }
  renderGutterUtility() {
    const { renderGutterUtility: e } = this.options;
    if (this.fileContainer == null || e == null) {
      (this.gutterUtilityContent?.remove(),
        (this.gutterUtilityContent = void 0));
      return;
    }
    const t = e(this.interactionManager.getHoveredLine);
    if (t != null && this.gutterUtilityContent != null) return;
    if (t == null) {
      (this.gutterUtilityContent?.remove(),
        (this.gutterUtilityContent = void 0));
      return;
    }
    const n = or();
    (n.appendChild(t),
      this.fileContainer.appendChild(n),
      (this.gutterUtilityContent = n));
  }
  getOrCreateFileContainer(e, t) {
    const { fileContainer: n } = this,
      i = e ?? n ?? document.createElement("diffs-container"),
      r = n !== i;
    return (
      n != null && r && this.editor?.__captureFocusForDOMReplacement(),
      r && this.emitPostRender(!0),
      (this.fileContainer = i),
      n != null &&
        r &&
        ((this.headerCache.lastRenderedHTML = void 0),
        (this.headerElement = void 0)),
      t != null &&
        this.fileContainer.parentNode !== t &&
        t.appendChild(this.fileContainer),
      r && this.adoptReusableShellElements(this.fileContainer),
      this.ensureSpriteSVG(this.fileContainer),
      this.fileContainer
    );
  }
  adoptReusableShellElements(e) {
    const { shadowRoot: t } = e;
    if (t != null)
      for (const n of t.children)
        n instanceof SVGElement
          ? (this.spriteSVG ??= n)
          : at(n) && n.hasAttribute("data-theme-css")
            ? ((this.themeCSSStyle ??= n), (this.hasAdoptedThemeCSS = !0))
            : at(n) &&
              n.hasAttribute("data-unsafe-css") &&
              ((this.unsafeCSSStyle ??= n),
              (this.appliedUnsafeCSS ??= this.options.unsafeCSS ?? void 0));
  }
  ensureSpriteSVG(e) {
    const t = e.shadowRoot ?? e.attachShadow({ mode: "open" });
    if (this.spriteSVG == null) {
      const n = document.createElement("div");
      n.innerHTML = ir;
      const i = n.firstChild;
      i instanceof SVGElement && (this.spriteSVG = i);
    }
    this.spriteSVG != null &&
      this.spriteSVG.parentNode !== t &&
      t.appendChild(this.spriteSVG);
  }
  getOrCreatePreNode(e) {
    const t = e.shadowRoot ?? e.attachShadow({ mode: "open" });
    return (
      this.pre == null
        ? ((this.pre = document.createElement("pre")),
          (this.appliedPreAttributes = void 0),
          (this.codeUnified = void 0),
          (this.codeDeletions = void 0),
          (this.codeAdditions = void 0),
          t.appendChild(this.pre))
        : this.pre.parentNode !== t &&
          (this.editor?.__captureFocusForDOMReplacement(),
          t.appendChild(this.pre),
          (this.appliedPreAttributes = void 0)),
      this.placeHolder?.remove(),
      (this.placeHolder = void 0),
      this.pre
    );
  }
  syncCodeNodesFromPre(e) {
    ((this.codeUnified = void 0),
      (this.codeDeletions = void 0),
      (this.codeAdditions = void 0));
    for (const t of Array.from(e.children))
      t instanceof HTMLElement &&
        (t.hasAttribute("data-unified")
          ? (this.codeUnified = t)
          : t.hasAttribute("data-deletions")
            ? (this.codeDeletions = t)
            : t.hasAttribute("data-additions") && (this.codeAdditions = t));
  }
  applyHeaderToDOM(e, t, n) {
    (this.cleanupErrorWrapper(),
      this.placeHolder?.remove(),
      (this.placeHolder = void 0));
    const {
        headerCache: { fileDiff: i, html: r, lastRenderedHTML: o },
      } = this,
      s = (n !== this.editSession?.diff && B(i, n) ? r : void 0) ?? J(e);
    if (
      ((this.headerCache.html = s), (this.headerCache.fileDiff = n), s !== o)
    ) {
      const p = document.createElement("div");
      p.innerHTML = s;
      const g = p.firstElementChild;
      if (!(g instanceof HTMLElement)) return;
      (this.headerElement != null
        ? t.shadowRoot?.replaceChild(g, this.headerElement)
        : t.shadowRoot?.prepend(g),
        (this.headerElement = g),
        (this.headerCache.lastRenderedHTML = s));
    }
    if (this.isContainerManaged) return;
    const {
      renderCustomHeader: a,
      renderHeaderPrefix: l,
      renderHeaderFilenameSuffix: d,
      renderHeaderMetadata: h,
    } = this.options;
    if (a != null) {
      const p = a(n) ?? void 0;
      ((this.headerCustom = this.upsertHeaderSlotElement(
        t,
        this.headerCustom,
        vn,
        p,
      )),
        this.headerPrefix?.remove(),
        this.headerFilenameSuffix?.remove(),
        this.headerMetadata?.remove(),
        (this.headerPrefix = void 0),
        (this.headerFilenameSuffix = void 0),
        (this.headerMetadata = void 0));
      return;
    }
    const c = l?.(n) ?? void 0,
      u = d?.(n) ?? void 0,
      f = h?.(n) ?? void 0;
    ((this.headerPrefix = this.upsertHeaderSlotElement(
      t,
      this.headerPrefix,
      Cn,
      c,
    )),
      (this.headerFilenameSuffix = this.upsertHeaderSlotElement(
        t,
        this.headerFilenameSuffix,
        bn,
        u,
      )),
      (this.headerMetadata = this.upsertHeaderSlotElement(
        t,
        this.headerMetadata,
        yn,
        f,
      )),
      this.headerCustom?.remove(),
      (this.headerCustom = void 0));
  }
  clearReusableHeader() {
    ((this.headerCache.html = void 0), (this.headerCache.fileDiff = void 0));
  }
  clearHeaderSlots() {
    (this.headerPrefix?.remove(),
      this.headerFilenameSuffix?.remove(),
      this.headerMetadata?.remove(),
      this.headerCustom?.remove(),
      (this.headerPrefix = void 0),
      (this.headerFilenameSuffix = void 0),
      (this.headerMetadata = void 0),
      (this.headerCustom = void 0));
  }
  upsertHeaderSlotElement(e, t, n, i) {
    if (i == null) {
      t?.remove();
      return;
    }
    const r = t ?? this.createHeaderSlotElement(n);
    return (
      t == null && e.appendChild(r),
      this.replaceHeaderSlotContent(r, i),
      r
    );
  }
  replaceHeaderSlotContent(e, t) {
    (e.replaceChildren(),
      t instanceof Element ? e.appendChild(t) : (e.innerText = `${t}`));
  }
  createHeaderSlotElement(e) {
    const t = document.createElement("div");
    return ((t.slot = e), t);
  }
  injectUnsafeCSS() {
    const { unsafeCSS: e } = this.options,
      t = this.fileContainer?.shadowRoot;
    if (t != null) {
      if (e == null || e === "") {
        (this.unsafeCSSStyle != null &&
          (this.unsafeCSSStyle.remove(), (this.unsafeCSSStyle = void 0)),
          (this.appliedUnsafeCSS = void 0));
        return;
      }
      (this.unsafeCSSStyle?.parentNode === t && this.appliedUnsafeCSS === e) ||
        ((this.unsafeCSSStyle ??= sr()),
        this.unsafeCSSStyle.parentNode !== t &&
          t.appendChild(this.unsafeCSSStyle),
        (this.unsafeCSSStyle.textContent = Rn(e)),
        (this.appliedUnsafeCSS = e));
    }
  }
  applyThemeState(e, t, n, i) {
    const r = e.shadowRoot ?? e.attachShadow({ mode: "open" }),
      o = i ?? n,
      s = this.getTheme(),
      a = typeof s == "string" ? s : { ...s },
      l = Ke(r);
    if (
      this.themeCSSStyle?.parentNode === r &&
      this.appliedThemeCSS?.themeStyles === t &&
      this.appliedThemeCSS.themeType === o &&
      this.appliedThemeCSS.scrollbarGutter === l
    ) {
      this.appliedThemeCSS.theme = a;
      return;
    }
    if (this.hasAdoptedThemeCSS && this.themeCSSStyle?.parentNode === r) {
      ((this.hasAdoptedThemeCSS = !1),
        (this.appliedThemeCSS = {
          theme: a,
          themeStyles: t,
          themeType: o,
          baseThemeType: i,
          scrollbarGutter: l,
        }));
      return;
    }
    ((this.themeCSSStyle = Mn({
      shadowRoot: r,
      currentNode: this.themeCSSStyle,
      themeCSS: Tn(t, o, l),
    })),
      (this.appliedThemeCSS =
        this.themeCSSStyle != null
          ? {
              theme: a,
              themeStyles: t,
              themeType: o,
              baseThemeType: i,
              scrollbarGutter: l,
            }
          : void 0));
  }
  hydrateMeasuredScrollbar() {
    const e = this.fileContainer?.shadowRoot;
    e == null ||
      this.themeCSSStyle == null ||
      (this.themeCSSStyle.textContent = dr(
        this.themeCSSStyle.textContent ?? "",
        Ke(e),
      ));
  }
  shouldGuardRebuildScroll() {
    return this.editor != null && pr();
  }
  applyHunksToDOM(e, t) {
    this.shouldGuardRebuildScroll()
      ? dn(e, () => this.replaceCodeColumns(e, t))
      : this.replaceCodeColumns(e, t);
  }
  applyCodeColumnsInPlace(e, t, n) {
    const i = this.getColumnPair(e);
    if (i == null) return !1;
    const r = Te(t[0]),
      o = Te(t[1]);
    return r == null || o == null
      ? !1
      : ((i.gutter.innerHTML = J(r)),
        (i.content.innerHTML = J(o)),
        n !== this.lastRowCount &&
          (i.gutter.style.setProperty("grid-row", `span ${n}`),
          i.content.style.setProperty("grid-row", `span ${n}`)),
        !0);
  }
  replaceCodeColumns(e, t) {
    const { overflow: n = "scroll" } = this.options,
      i = (this.options.hunkSeparators ?? "line-info") === "line-info",
      r = n === "wrap" ? t.rowCount : void 0;
    (this.cleanupErrorWrapper(), this.applyPreNodeAttributes(e, t));
    let o = !1;
    const s = [],
      a = this.hunksRenderer.renderCodeAST("unified", t),
      l = this.hunksRenderer.renderCodeAST("deletions", t),
      d = this.hunksRenderer.renderCodeAST("additions", t);
    (this.editor?.__captureFocusForDOMReplacement(),
      a != null
        ? ((o =
            this.codeUnified == null ||
            this.codeAdditions != null ||
            this.codeDeletions != null),
          this.codeDeletions?.remove(),
          (this.codeDeletions = void 0),
          this.codeAdditions?.remove(),
          (this.codeAdditions = void 0),
          (this.codeUnified = rt({
            code: this.codeUnified,
            columnType: "unified",
            rowSpan: r,
            containerSize: i,
          })),
          this.applyCodeColumnsInPlace(this.codeUnified, a, t.rowCount) ||
            (this.codeUnified.innerHTML =
              this.hunksRenderer.renderPartialHTML(a)),
          s.push(this.codeUnified))
        : l != null || d != null
          ? (l != null
              ? ((o = this.codeDeletions == null || this.codeUnified != null),
                this.codeUnified?.remove(),
                (this.codeUnified = void 0),
                (this.codeDeletions = rt({
                  code: this.codeDeletions,
                  columnType: "deletions",
                  rowSpan: r,
                  containerSize: i,
                })),
                this.applyCodeColumnsInPlace(
                  this.codeDeletions,
                  l,
                  t.rowCount,
                ) ||
                  (this.codeDeletions.innerHTML =
                    this.hunksRenderer.renderPartialHTML(l)),
                s.push(this.codeDeletions))
              : (this.codeDeletions?.remove(), (this.codeDeletions = void 0)),
            d != null
              ? ((o =
                  o || this.codeAdditions == null || this.codeUnified != null),
                this.codeUnified?.remove(),
                (this.codeUnified = void 0),
                (this.codeAdditions = rt({
                  code: this.codeAdditions,
                  columnType: "additions",
                  rowSpan: r,
                  containerSize: i,
                })),
                this.applyCodeColumnsInPlace(
                  this.codeAdditions,
                  d,
                  t.rowCount,
                ) ||
                  (this.codeAdditions.innerHTML =
                    this.hunksRenderer.renderPartialHTML(d)),
                s.push(this.codeAdditions))
              : (this.codeAdditions?.remove(), (this.codeAdditions = void 0)))
          : (this.codeUnified?.remove(),
            (this.codeUnified = void 0),
            this.codeDeletions?.remove(),
            (this.codeDeletions = void 0),
            this.codeAdditions?.remove(),
            (this.codeAdditions = void 0)),
      s.length === 0 ? (e.textContent = "") : o && e.replaceChildren(...s),
      (this.lastRowCount = t.rowCount));
  }
  applyPartialRender({ fileDiff: e, previousRenderRange: t, renderRange: n }) {
    const {
      pre: i,
      codeUnified: r,
      codeAdditions: o,
      codeDeletions: s,
      options: { diffStyle: a = "split" },
    } = this;
    if (
      i == null ||
      t == null ||
      n == null ||
      !Number.isFinite(t.totalLines) ||
      !Number.isFinite(n.totalLines) ||
      this.lastRowCount == null
    )
      return !1;
    const l = this.getCodeColumns(a, r, s, o);
    if (l == null) return !1;
    const d = t.startingLine,
      h = n.startingLine,
      c = d + t.totalLines,
      u = h + n.totalLines,
      f = Math.max(d, h),
      p = Math.min(c, u);
    if (p <= f) return !1;
    const g = Math.max(0, f - d),
      m = Math.max(0, c - p),
      y = this.trimColumns({
        columns: l,
        trimStart: g,
        trimEnd: m,
        previousStart: d,
        overlapStart: f,
        overlapEnd: p,
        diffStyle: a,
      });
    if (y < 0)
      throw new Error("FileDiff.applyPartialRender: failed to trim to overlap");
    if (this.lastRowCount < y)
      throw new Error(
        "FileDiff.applyPartialRender: trimmed beyond DOM row count",
      );
    let S = this.lastRowCount - y;
    const b = (C, x) => {
        if (!(x <= 0))
          return this.hunksRenderer.renderDiff(e, {
            startingLine: C,
            totalLines: x,
            bufferBefore: 0,
            bufferAfter: 0,
          });
      },
      L = b(h, Math.max(f - h, 0));
    if (L == null && h < f) return !1;
    const v = b(p, Math.max(u - p, 0));
    if (v == null && u > p) return !1;
    const k = (C, x) => {
      if (C != null) {
        if (a === "unified" && !Array.isArray(l))
          this.insertPartialHTML(a, l, C, x);
        else if (a === "split" && Array.isArray(l))
          this.insertPartialHTML(a, l, C, x);
        else
          throw new Error(
            "FileDiff.applyPartialRender.applyChunk: invalid chunk application",
          );
        ((S += C.rowCount), (this.renderedDiff = C.fileDiff));
      }
    };
    return (
      this.cleanupErrorWrapper(),
      k(L, "afterbegin"),
      k(v, "beforeend"),
      this.lastRowCount !== S &&
        (this.applyRowSpan(a, l, S), (this.lastRowCount = S)),
      !0
    );
  }
  insertPartialHTML(e, t, n, i) {
    if (e === "unified" && !Array.isArray(t)) {
      const r = this.hunksRenderer.renderCodeAST("unified", n);
      this.renderPartialColumn(t, r, i);
    } else if (e === "split" && Array.isArray(t)) {
      const r = this.hunksRenderer.renderCodeAST("deletions", n),
        o = this.hunksRenderer.renderCodeAST("additions", n);
      (this.renderPartialColumn(t[0], r, i),
        this.renderPartialColumn(t[1], o, i));
    } else
      throw new Error(
        "FileDiff.insertPartialHTML: Invalid argument composition",
      );
  }
  refreshSplitDiffView() {
    const e = this.getLatestDiff();
    if (this.options.diffStyle !== "split" || e == null) return;
    const t = this.hunksRenderer.renderDiff(e, this.renderRange);
    if (t == null) return;
    const n = this.getCodeColumns(
      "split",
      this.codeUnified,
      this.codeDeletions,
      this.codeAdditions,
    );
    if (!Array.isArray(n)) return;
    const i = (r, o) => {
      if (o == null) return;
      const s = this.hunksRenderer.renderCodeAST(r, t),
        a = Te(s?.[0]),
        l = Te(s?.[1]);
      for (const [d, h] of [
        [o.gutter, a],
        [o.content, l],
      ])
        if (h != null && d.childElementCount === h.length)
          for (let c = 0; c < h.length; c++) {
            const u = d.children[c],
              f = h[c].properties["data-line-type"];
            f != null && u.dataset.lineType !== f && (u.dataset.lineType = f);
          }
    };
    (i("deletions", n[0]),
      i("additions", n[1]),
      (this.renderedDiff = t.fileDiff));
  }
  refreshUnifiedDiffView() {
    const e = this.getLatestDiff();
    if (this.options.diffStyle !== "unified" || e == null) return;
    const t = this.hunksRenderer.renderDiff(e, this.renderRange);
    if (t == null) return;
    const n = this.getCodeColumns(
      "unified",
      this.codeUnified,
      this.codeDeletions,
      this.codeAdditions,
    );
    if (n == null || Array.isArray(n)) return;
    const i = this.hunksRenderer.renderCodeAST("unified", t),
      r = Te(i?.[0]),
      o = Te(i?.[1]),
      s = () => {
        for (const [a, l] of [
          [n.gutter, r],
          [n.content, o],
        ])
          l != null && (a.innerHTML = J(l));
        (t.rowCount !== this.lastRowCount &&
          (this.applyRowSpan("unified", n, t.rowCount),
          (this.lastRowCount = t.rowCount)),
          (this.renderedDiff = t.fileDiff));
      };
    (this.shouldGuardRebuildScroll() ? dn(this.pre, s) : s(),
      this.renderSeparators(t.hunkData),
      (this.managersDirty = !0),
      this.flushManagers(),
      this.syncRenderViewToEditor());
  }
  renderPartialColumn(e, t, n) {
    if (e == null || t == null) return;
    const i = Te(t[0]),
      r = Te(t[1]);
    if (i == null || r == null)
      throw new Error("FileDiff.insertPartialHTML: Unexpected AST structure");
    const o = r.at(0);
    n === "beforeend" &&
      o?.type === "element" &&
      typeof o.properties["data-buffer-size"] == "number" &&
      this.mergeBuffersIfNecessary(
        o.properties["data-buffer-size"],
        e.content.children[e.content.children.length - 1],
        e.gutter.children[e.gutter.children.length - 1],
        i,
        r,
        !0,
      );
    const s = r.at(-1);
    (n === "afterbegin" &&
      s?.type === "element" &&
      typeof s.properties["data-buffer-size"] == "number" &&
      this.mergeBuffersIfNecessary(
        s.properties["data-buffer-size"],
        e.content.children[0],
        e.gutter.children[0],
        i,
        r,
        !1,
      ),
      e.gutter.insertAdjacentHTML(n, this.hunksRenderer.renderPartialHTML(i)),
      e.content.insertAdjacentHTML(n, this.hunksRenderer.renderPartialHTML(r)));
  }
  mergeBuffersIfNecessary(e, t, n, i, r, o) {
    if (!(t instanceof HTMLElement) || !(n instanceof HTMLElement)) return;
    const s = this.getBufferSize(t.dataset);
    s != null &&
      (o ? (i.shift(), r.shift()) : (i.pop(), r.pop()),
      this.updateBufferSize(t, s + e),
      this.updateBufferSize(n, s + e));
  }
  applyRowSpan(e, t, n) {
    const i = (r) => {
      r != null &&
        (r.gutter.style.setProperty("grid-row", `span ${n}`),
        r.content.style.setProperty("grid-row", `span ${n}`));
    };
    if (e === "unified" && !Array.isArray(t)) i(t);
    else if (e === "split" && Array.isArray(t)) (i(t[0]), i(t[1]));
    else throw new Error("dun fuuuuked up");
  }
  trimColumnRows(e, t, n) {
    let i = 0,
      r = 0,
      o = 0,
      s = !1;
    const a = n >= 0;
    if (e == null) return 0;
    const l = Array.from(e.content.children),
      d = Array.from(e.gutter.children);
    if (l.length !== d.length)
      throw new Error("FileDiff.trimColumnRows: columns do not match");
    for (; o < l.length && !(t <= 0 && !a && !s);) {
      const h = d[o],
        c = l[o];
      if ((o++, !(h instanceof HTMLElement) || !(c instanceof HTMLElement)))
        throw (
          console.error({ gutterElement: h, contentElement: c }),
          new Error("FileDiff.trimColumnRows: invalid row elements")
        );
      if (
        s &&
        ((s = !1),
        (h.dataset.gutterBuffer === "annotation" &&
          "lineAnnotation" in c.dataset) ||
          (h.dataset.gutterBuffer === "metadata" && "noNewline" in c.dataset))
      ) {
        (h.remove(), c.remove(), r++);
        continue;
      }
      if ("lineIndex" in h.dataset && "lineIndex" in c.dataset) {
        ((t > 0 || (a && i >= n)) &&
          (h.remove(), c.remove(), t > 0 && (t--, t === 0 && (s = !0)), r++),
          i++);
        continue;
      }
      if ("separator" in h.dataset && "separator" in c.dataset) {
        (t > 0 || (a && i >= n)) && (h.remove(), c.remove(), r++);
        continue;
      }
      if (
        h.dataset.gutterBuffer === "annotation" &&
        "lineAnnotation" in c.dataset
      ) {
        (t > 0 || (a && i >= n)) && (h.remove(), c.remove(), r++);
        continue;
      }
      if (h.dataset.gutterBuffer === "metadata" && "noNewline" in c.dataset) {
        (t > 0 || (a && i >= n)) && (h.remove(), c.remove(), r++);
        continue;
      }
      if (h.dataset.gutterBuffer === "buffer" && "contentBuffer" in c.dataset) {
        const u = this.getBufferSize(c.dataset);
        if (u == null)
          throw new Error("FileDiff.trimColumnRows: invalid element");
        if (t > 0) {
          const f = Math.min(t, u),
            p = u - f;
          (p > 0
            ? (this.updateBufferSize(h, p),
              this.updateBufferSize(c, p),
              (r += f))
            : (h.remove(), c.remove(), (r += u)),
            (t -= f),
            t === 0 && p === 0 && (s = !0));
        } else if (a) {
          const f = i,
            p = i + u - 1;
          if (n <= f) (h.remove(), c.remove(), (r += u));
          else if (n <= p) {
            const g = p - n + 1,
              m = u - g;
            (this.updateBufferSize(h, m),
              this.updateBufferSize(c, m),
              (r += g));
          }
        }
        i += u;
        continue;
      }
      throw (
        console.error({ gutterElement: h, contentElement: c }),
        new Error("FileDiff.trimColumnRows: unknown row elements")
      );
    }
    return r;
  }
  trimColumns({
    columns: e,
    diffStyle: t,
    overlapEnd: n,
    overlapStart: i,
    previousStart: r,
    trimEnd: o,
    trimStart: s,
  }) {
    const a = Math.max(0, i - r),
      l = n - r;
    if (l < 0)
      throw new Error("FileDiff.trimColumns: overlap ends before previous");
    const d = s > 0,
      h = o > 0;
    if (!d && !h) return 0;
    const c = d ? a : 0,
      u = h ? l : -1;
    if (t === "unified" && !Array.isArray(e))
      return this.trimColumnRows(e, c, u);
    if (t === "split" && Array.isArray(e)) {
      const f = this.trimColumnRows(e[0], c, u),
        p = this.trimColumnRows(e[1], c, u);
      if (e[0] != null && e[1] != null && f !== p)
        throw new Error("FileDiff.trimColumns: split columns out of sync");
      return e[0] != null ? f : p;
    } else
      throw (
        console.error({ diffStyle: t, columns: e }),
        new Error("FileDiff.trimColumns: Invalid columns for diffType")
      );
  }
  getBufferSize(e) {
    const t = Number.parseInt(e?.bufferSize ?? "", 10);
    return Number.isNaN(t) ? void 0 : t;
  }
  updateBufferSize(e, t) {
    ((e.dataset.bufferSize = `${t}`),
      e.style.setProperty("grid-row", `span ${t}`),
      e.style.setProperty("min-height", `calc(${t} * 1lh)`));
  }
  getColumnPair(e) {
    if (e == null) return;
    const t = e.children[0],
      n = e.children[1];
    if (!(
      !(t instanceof HTMLElement) ||
      !(n instanceof HTMLElement) ||
      t.dataset.gutter == null ||
      n.dataset.content == null
    ))
      return { gutter: t, content: n };
  }
  getCodeColumns(e, t, n, i) {
    if (e === "unified") return this.getColumnPair(t);
    {
      const r = this.getColumnPair(n),
        o = this.getColumnPair(i);
      return r != null || o != null ? [r, o] : void 0;
    }
  }
  updateBuffers(e) {
    this.pre != null && this.applyBuffers(this.pre, e);
  }
  applyBuffers(e, t) {
    if (t == null || this.shouldDisableVirtualizationBuffers()) {
      (this.bufferBefore != null &&
        (this.bufferBefore.remove(), (this.bufferBefore = void 0)),
        this.bufferAfter != null &&
          (this.bufferAfter.remove(), (this.bufferAfter = void 0)));
      return;
    }
    (t.bufferBefore > 0
      ? (this.bufferBefore == null &&
          ((this.bufferBefore = document.createElement("div")),
          (this.bufferBefore.dataset.virtualizerBuffer = "before"),
          e.before(this.bufferBefore)),
        this.bufferBefore.style.setProperty("height", `${t.bufferBefore}px`),
        this.bufferBefore.style.setProperty("contain", "strict"))
      : this.bufferBefore != null &&
        (this.bufferBefore.remove(), (this.bufferBefore = void 0)),
      t.bufferAfter > 0
        ? (this.bufferAfter == null &&
            ((this.bufferAfter = document.createElement("div")),
            (this.bufferAfter.dataset.virtualizerBuffer = "after"),
            e.after(this.bufferAfter)),
          this.bufferAfter.style.setProperty("height", `${t.bufferAfter}px`),
          this.bufferAfter.style.setProperty("contain", "strict"))
        : this.bufferAfter != null &&
          (this.bufferAfter.remove(), (this.bufferAfter = void 0)));
  }
  shouldDisableVirtualizationBuffers() {
    return this.options.disableVirtualizationBuffers ?? !1;
  }
  applyPreNodeAttributes(
    e,
    { additionsContentAST: t, deletionsContentAST: n, totalLines: i },
    r,
  ) {
    const {
        diffIndicators: o = "bars",
        disableBackground: s = !1,
        disableLineNumbers: a = !1,
        overflow: l = "scroll",
        diffStyle: d = "split",
      } = this.options,
      h = {
        type: "diff",
        diffIndicators: o,
        disableBackground: s,
        disableLineNumbers: a,
        overflow: l,
        split: d === "unified" ? !1 : t != null && n != null,
        totalLines: i,
        customProperties: r,
      };
    rr(h, this.appliedPreAttributes) ||
      (In(e, h), (this.appliedPreAttributes = h));
  }
  applyErrorToDOM(e, t) {
    (this.cleanupErrorWrapper(),
      this.pre?.remove(),
      (this.pre = void 0),
      (this.appliedPreAttributes = void 0));
    const n = t.shadowRoot ?? t.attachShadow({ mode: "open" });
    ((this.errorWrapper ??= document.createElement("div")),
      (this.errorWrapper.dataset.errorWrapper = ""),
      (this.errorWrapper.textContent = ""),
      n.appendChild(this.errorWrapper));
    const i = document.createElement("div");
    ((i.dataset.errorMessage = ""),
      (i.innerText = e.message),
      this.errorWrapper.appendChild(i));
    const r = document.createElement("pre");
    ((r.dataset.errorStack = ""),
      (r.innerText = e.stack ?? "No Error Stack"),
      this.errorWrapper.appendChild(r));
  }
  cleanupErrorWrapper() {
    (this.errorWrapper?.remove(), (this.errorWrapper = void 0));
  }
};
function Jt(e) {
  return { name: e.name, lang: e.lang, contents: e.additionLines.join("") };
}
function ai(e, t) {
  return e == null || t == null ? e == null && t == null : ot(e, t);
}
function Na({ fileDiff: e, oldFile: t, newFile: n }) {
  return (e != null && e.hunks.length > 0) || t != null || n != null;
}
function za({ fileDiff: e, oldFile: t, newFile: n }) {
  return e != null || t != null || n != null;
}
function Ua(e, t, n = !1) {
  return !n && e == null && t;
}
function _a(e, t, n = !1) {
  return e == null && t && !n;
}
function Te(e) {
  if (!(e == null || e.type !== "element")) return e.children ?? [];
}
function Ba({
  fileDiff: e,
  metrics: t,
  disableFileHeader: n,
  hunkSeparators: i,
  expandUnchanged: r,
  expandedHunks: o,
  collapsedContextThreshold: s,
  canHydratePartialDiff: a,
}) {
  let l = ue(t, n),
    d = l;
  const h = r ? !0 : o,
    c = e.hunks.length - 1;
  for (let u = 0; u < e.hunks.length; u++) {
    const f = e.hunks[u];
    if (f == null)
      throw new Error("computeEstimatedDiffHeights: invalid hunk index");
    const p = ke({
        isPartial: e.isPartial,
        rangeSize: f.collapsedBefore,
        expandedHunks: h,
        hunkIndex: u,
        collapsedContextThreshold: s,
      }),
      g = (p.fromStart + p.fromEnd) * t.lineHeight;
    if (((l += g), (d += g), p.collapsedLines > 0)) {
      const S =
        nt({ type: i, metrics: t, hunkIndex: u, hunkSpecs: f.hunkSpecs })
          ?.totalHeight ?? 0;
      ((l += S), (d += S));
    }
    ((l += f.splitLineCount * t.lineHeight),
      (d += f.unifiedLineCount * t.lineHeight));
    const m = Va(f);
    ((l += m.split * t.lineHeight), (d += m.unified * t.lineHeight));
    const y =
      u === c
        ? we({
            fileDiff: e,
            hunkIndex: u,
            expandedHunks: h,
            collapsedContextThreshold: s,
            errorPrefix: "computeEstimatedDiffHeights",
          })
        : void 0;
    if (y != null) {
      const S = (y.fromStart + y.fromEnd) * t.lineHeight;
      if (((l += S), (d += S), y.collapsedLines > 0)) {
        const b = Me({ type: i, metrics: t })?.totalHeight ?? 0;
        ((l += b), (d += b));
      }
    } else if (u === c && e.isPartial && a) {
      const S = Me({ type: i, metrics: t })?.totalHeight ?? 0;
      ((l += S), (d += S));
    }
  }
  if (e.hunks.length > 0) {
    const u = Ut(t);
    ((l += u), (d += u));
  }
  return { splitHeight: l, unifiedHeight: d };
}
function Va(e) {
  if (!e.noEOFCRAdditions && !e.noEOFCRDeletions)
    return { split: 0, unified: 0 };
  const t = e.hunkContent.at(-1);
  if (t == null) return { split: 0, unified: 0 };
  if (t.type === "context") {
    const n = t.lines > 0 ? 1 : 0;
    return { split: n, unified: n };
  }
  return $a(e, t);
}
function $a(e, t) {
  const n =
      (t.deletions > 0 && e.noEOFCRDeletions ? 1 : 0) +
      (t.additions > 0 && e.noEOFCRAdditions ? 1 : 0),
    i = t.deletions > 0 && e.noEOFCRDeletions,
    r = t.additions > 0 && e.noEOFCRAdditions;
  return { split: i || r ? 1 : 0, unified: n };
}
const mn = 3e3,
  Et = new Map();
let Ga = -1;
var Wa = class extends Rr {
  __id = `little-virtualized-file-diff:${++Ga}`;
  renderType = "virtualized";
  top;
  height = 0;
  metrics;
  cache = {
    heightDeltas: new Map(),
    measuredHeightDeltaTotal: 0,
    estimatedSplitHeight: void 0,
    estimatedUnifiedHeight: void 0,
    checkpoints: [],
    totalLines: 0,
    fileAnnotationHeight: 0,
    ghostTextRows: Et,
    ghostTextRowsByIndex: new Map(),
  };
  isVisible = !1;
  isSetup = !1;
  virtualizer;
  layoutDirty = !0;
  forceRenderOverride;
  currentCollapsed;
  pendingHydratedDiff;
  pendingExpansions;
  pendingRender;
  constructor(e, t, n, i, r = !1) {
    (super(e, i, r), (this.virtualizer = t), (this.metrics = Mt(n)));
  }
  setMetrics(e, t = !1) {
    const n = Mt(e);
    (!t && qe(this.metrics, n)) ||
      ((this.metrics = n),
      this.resetLayoutCache({ includeEstimatedHeights: !0 }));
  }
  setLineAnnotations(e) {
    this.syncLineAnnotations(e) &&
      this.resetLayoutCache({ includeEstimatedHeights: !1 });
  }
  syncLineAnnotations(e) {
    return e == null ||
      !this.isNewAnnotations(e) ||
      (e.length === 0 && this.getLatestAnnotations().length === 0)
      ? !1
      : (super.setLineAnnotations(e), !0);
  }
  syncEditSessionAnnotationsFromEditor(e) {
    return super.syncEditSessionAnnotationsFromEditor(e)
      ? (this.resetLayoutCache({ includeEstimatedHeights: !1 }), !0)
      : !1;
  }
  setFileAnnotationHeight(e) {
    const t = this.cache.fileAnnotationHeight;
    return e === t
      ? !1
      : ((this.cache.fileAnnotationHeight = e),
        (this.cache.measuredHeightDeltaTotal += e - t),
        !0);
  }
  hasFileAnnotations(e) {
    const t = this.getLatestAnnotations();
    return an(t)
      ? t.some((n) =>
          n.lineNumber !== 0
            ? !1
            : e.type === "new"
              ? n.side === "additions"
              : e.type === "deleted"
                ? n.side === "deletions"
                : !0,
        )
      : !1;
  }
  getLineHeight(e, t = !1) {
    return (
      this.getEstimatedLineHeight(t) + (this.cache.heightDeltas.get(e) ?? 0)
    );
  }
  getEstimatedLineHeight(e = !1) {
    const t = e ? 2 : 1;
    return this.metrics.lineHeight * t;
  }
  setOptions(e) {
    if (this.isAdvancedMode())
      throw new Error(
        "VirtualizedFileDiff.setOptions cannot be used inside CodeView. Update CodeView options instead.",
      );
    if (e == null) return;
    const { options: t } = this,
      n = !Ln(t, e),
      i = n && Qa(t, e);
    (super.setOptions(e),
      i &&
        this.resetLayoutCache({
          forceSimpleRecompute: !0,
          includeEstimatedHeights: Za(t, e),
        }),
      n && (this.forceRenderOverride = !0),
      n && this.isSimpleMode() && this.virtualizer.instanceChanged(this, i));
  }
  setThemeType(e) {
    if (this.isAdvancedMode())
      throw new Error(
        "VirtualizedFileDiff.setThemeType cannot be used inside CodeView. Update CodeView options instead.",
      );
    super.setThemeType(e);
  }
  resetLayoutCache({
    forceSimpleRecompute: e = !1,
    includeEstimatedHeights: t = !1,
    resetRenderRange: n = !0,
  } = {}) {
    ((this.layoutDirty = !0),
      (this.cache.fileAnnotationHeight = 0),
      this.cache.heightDeltas.clear(),
      (this.cache.measuredHeightDeltaTotal = 0),
      (this.cache.ghostTextRows = Et),
      this.cache.ghostTextRowsByIndex.size > 0 &&
        this.cache.ghostTextRowsByIndex.clear(),
      this.invalidateDerivedLayoutCache(t, n),
      e && this.isSimpleMode() && this.computeApproximateSize());
  }
  invalidateDerivedLayoutCache(e, t = !0) {
    ((this.layoutDirty = !0),
      this.cache.checkpoints.length > 0 && (this.cache.checkpoints.length = 0),
      this.cache.totalLines !== 0 && (this.cache.totalLines = 0),
      e &&
        ((this.cache.estimatedSplitHeight = void 0),
        (this.cache.estimatedUnifiedHeight = void 0)),
      this.renderRange != null && t && (this.renderRange = void 0));
  }
  applyGhostTextRows(e, t) {
    const {
      cache: { heightDeltas: n, ghostTextRows: i, ghostTextRowsByIndex: r },
      metrics: { lineHeight: o },
    } = this;
    if (i === t) return !1;
    const s = this.getDiffStyle(),
      a = new Map();
    for (const [d, h] of t) {
      const c = this.getLineIndexForDiff(e, d + 1, "additions");
      c != null && a.set(s === "split" ? c[1] : c[0], h);
    }
    let l = !1;
    for (const d of new Set([...r.keys(), ...a.keys()])) {
      const h = ((a.get(d) ?? 0) - (r.get(d) ?? 0)) * o;
      if (h === 0) continue;
      const c = (n.get(d) ?? 0) + h;
      (c === 0 ? n.delete(d) : n.set(d, c),
        (this.cache.measuredHeightDeltaTotal += h),
        (l = !0));
    }
    return (
      (this.cache.ghostTextRows = t),
      (this.cache.ghostTextRowsByIndex = a),
      l
    );
  }
  reconcileHeights() {
    let e = !1;
    const {
        options: { overflow: t = "scroll" },
        cache: { ghostTextRowsByIndex: n },
        metrics: { lineHeight: i },
      } = this,
      r = this.getRenderedDiff();
    if (this.fileContainer == null || r == null)
      return (this.height !== 0 && (e = !0), (this.height = 0), e);
    ((this.top = this.getVirtualizedTop()),
      (e = this.applyGhostTextRows(
        r,
        this.editor?.__getGhostTextRows() ?? Et,
      )));
    const o = this.getLatestAnnotations();
    if (t === "scroll" && o.length === 0 && !this.isResizeDebuggingEnabled())
      return (e && this.computeApproximateSize(!0), e);
    const s = this.getDiffStyle(),
      a =
        s === "split"
          ? [this.codeDeletions, this.codeAdditions]
          : [this.codeUnified],
      l = this.hasFileAnnotations(r);
    if (this.renderRange != null && l && $t(this.renderRange)) {
      const d = qa(a) ?? 0;
      this.setFileAnnotationHeight(d) && (e = !0);
    } else !l && this.setFileAnnotationHeight(0) && (e = !0);
    for (const d of a) {
      if (d == null) continue;
      const h = d.children[1];
      if (!(h instanceof HTMLElement)) continue;
      const c = s === "split" && d === this.codeDeletions;
      for (const u of h.children) {
        if (!(u instanceof HTMLElement)) continue;
        const f = u.dataset.lineIndex;
        if (f == null) continue;
        const p = el(f, s),
          g = n.get(p) ?? 0;
        if (c && g > 0) continue;
        let m = u.getBoundingClientRect().height + g * i,
          y = !1;
        u.nextElementSibling instanceof HTMLElement &&
          ("lineAnnotation" in u.nextElementSibling.dataset ||
            "noNewline" in u.nextElementSibling.dataset) &&
          ("noNewline" in u.nextElementSibling.dataset && (y = !0),
          (m += u.nextElementSibling.getBoundingClientRect().height));
        const S = this.getEstimatedLineHeight(y),
          b = this.cache.heightDeltas.get(p) ?? 0,
          L = m - S;
        L !== b &&
          ((e = !0),
          (this.cache.measuredHeightDeltaTotal += L - b),
          L === 0
            ? this.cache.heightDeltas.delete(p)
            : this.cache.heightDeltas.set(p, L));
      }
    }
    return (
      (e || this.isResizeDebuggingEnabled()) && this.computeApproximateSize(!0),
      e
    );
  }
  onRender = (e) =>
    this.fileContainer == null
      ? !1
      : (e && (this.top = this.getVirtualizedTop()), this.render());
  flushManagers() {
    (super.flushManagers(),
      this.lineStateRefreshPending && this.flushDeferredLineState());
  }
  updateCodeViewLayout(e, t, n, i) {
    let r = !1,
      o = !1;
    const {
      pendingExpansions: s,
      pendingHydratedDiff: a,
      options: { collapsed: l = !1 },
    } = this;
    if (
      (a != null &&
        ((this.pendingHydratedDiff = void 0),
        a.expectedDiff === e &&
          (Object.assign(e, a.nextDiff),
          this.setHydratedState(a.files),
          this.startHydratedEditSession(e),
          (this.forceRenderOverride = !0),
          (r = !0),
          (o = !0))),
      s != null)
    ) {
      this.pendingExpansions = void 0;
      for (const {
        hunkIndex: p,
        direction: g,
        expansionLineCountOverride: m,
      } of s)
        (this.hunksRenderer.expandHunk(p, g, m),
          (this.forceRenderOverride = !0),
          (o = !0));
    }
    const d = this.updateExternalDiff(e, i),
      {
        pendingRenderDiff: h,
        layoutDiffChanged: c,
        renderedDiffChanged: u,
        annotationsChanged: f,
      } = this.updatePendingRender(e, i);
    return (
      !this.forceRenderOverride &&
        (d || u || f) &&
        (this.forceRenderOverride = !0),
      (n?.resetDiffLayoutCache === !0 || c || f) && (r = !0),
      (c ||
        (n?.resetDiffLayoutCache === !0 && n.includeEstimatedDiffHeights)) &&
        (o = !0),
      n?.metrics != null &&
        ((this.metrics = Mt(n.metrics)), (r = !0), (o = !0)),
      this.currentCollapsed !== l && ((this.currentCollapsed = l), (r = !0)),
      r
        ? this.resetLayoutCache({ includeEstimatedHeights: o })
        : o && this.invalidateDerivedLayoutCache(!0),
      (this.top = t),
      this.computeApproximateSize(!1, h),
      this.height
    );
  }
  getLinePosition(e, t = "additions") {
    const n = this.getLayoutDiff();
    if (n == null || e < 1) return;
    const i = this.getLineIndexForDiff(n, e, t);
    if (i == null) return;
    const {
        disableFileHeader: r = !1,
        expandUnchanged: o = !1,
        collapsed: s = !1,
        collapsedContextThreshold: a = 1,
      } = this.options,
      l = this.getDiffStyle(),
      d = this.getHunkSeparatorType(),
      h = l === "split" ? i[1] : i[0];
    this.approximateLayoutCheckpoints(n);
    const c = ue(this.metrics, r),
      u = this.getLayoutCheckpointBeforeLineIndex(h);
    let f = u?.top ?? c + this.cache.fileAnnotationHeight;
    if (s) return { top: c, height: 0 };
    let p;
    return (
      Fe({
        diff: n,
        diffStyle: l,
        startingLine: u?.renderedLineIndex ?? 0,
        expandedHunks: o ? !0 : this.hunksRenderer.getExpandedHunksMap(),
        collapsedContextThreshold: a,
        callback: ({
          hunkIndex: g,
          hunk: m,
          collapsedBefore: y,
          collapsedAfter: S,
          deletionLine: b,
          additionLine: L,
        }) => {
          const v =
            l === "split"
              ? (L?.splitLineIndex ?? b?.splitLineIndex)
              : (L?.unifiedLineIndex ?? b?.unifiedLineIndex);
          if (v == null)
            throw new Error(
              "VirtualizedFileDiff.getLinePosition: missing line index data",
            );
          if (y > 0) {
            const C = nt({
              type: d,
              metrics: this.metrics,
              hunkIndex: g,
              hunkSpecs: m?.hunkSpecs,
            });
            if (C != null) {
              if (((f += C.gapBefore), h >= v - y && h < v))
                return ((p = { top: f, height: C.height }), !0);
              f += C.height + C.gapAfter;
            }
          }
          const k = this.getLineHeight(
            v,
            (L?.noEOFCR ?? !1) || (b?.noEOFCR ?? !1),
          );
          if (v === h) return ((p = { top: f, height: k }), !0);
          if (((f += k), S > 0)) {
            const C = Me({ type: d, metrics: this.metrics });
            if (C != null) {
              if (h > v && h <= v + S)
                return ((p = { top: f + C.gapBefore, height: C.height }), !0);
              f += C.totalHeight;
            }
          }
          return !1;
        },
      }),
      p
    );
  }
  getEditorViewport() {
    return this.virtualizer.type === "simple"
      ? this.virtualizer.getRoot()
      : this.virtualizer.getContainerElement();
  }
  getNumericScrollAnchor(e) {
    const t = this.getLayoutDiff();
    if (t == null) return;
    const {
      disableFileHeader: n = !1,
      expandUnchanged: i = !1,
      collapsed: r = !1,
      collapsedContextThreshold: o = 1,
    } = this.options;
    if (r) return;
    const s = this.getDiffStyle(),
      a = this.getHunkSeparatorType();
    this.approximateLayoutCheckpoints(t);
    const l = this.getLayoutCheckpointBeforeTop(e);
    let d = l?.top ?? ue(this.metrics, n) + this.cache.fileAnnotationHeight,
      h;
    return (
      Fe({
        diff: t,
        diffStyle: s,
        startingLine: l?.renderedLineIndex ?? 0,
        expandedHunks: i ? !0 : this.hunksRenderer.getExpandedHunksMap(),
        collapsedContextThreshold: o,
        callback: ({
          hunkIndex: c,
          hunk: u,
          collapsedBefore: f,
          collapsedAfter: p,
          deletionLine: g,
          additionLine: m,
        }) => {
          const y =
            s === "split"
              ? (m?.splitLineIndex ?? g?.splitLineIndex)
              : (m?.unifiedLineIndex ?? g?.unifiedLineIndex);
          if (y == null)
            throw new Error(
              "VirtualizedFileDiff.getNumericScrollAnchor: missing line index data",
            );
          if (f > 0) {
            const b = nt({
              type: a,
              metrics: this.metrics,
              hunkIndex: c,
              hunkSpecs: u?.hunkSpecs,
            });
            b != null && (d += b.totalHeight);
          }
          if (
            d >= e &&
            (g != null
              ? (h = { lineNumber: g.lineNumber, side: "deletions", top: d })
              : m != null &&
                (h = { lineNumber: m.lineNumber, side: "additions", top: d }),
            h != null)
          )
            return !0;
          const S = this.getLineHeight(
            y,
            (m?.noEOFCR ?? !1) || (g?.noEOFCR ?? !1),
          );
          if (((d += S), p > 0)) {
            const b = Me({ type: a, metrics: this.metrics });
            b != null && (d += b.totalHeight);
          }
          return !1;
        },
      }),
      h
    );
  }
  getVirtualizedHeight() {
    return this.height;
  }
  getAdvancedStickySpecs(e) {
    const t = this.getLayoutDiff();
    if (this.top == null || t == null) return;
    if (this.options.collapsed === !0)
      return { topOffset: this.top, height: this.height };
    const n =
      e != null
        ? this.computeRenderRangeFromWindow(t, this.top, e)
        : this.renderRange;
    if (n == null) return;
    const { bufferBefore: i, bufferAfter: r, totalLines: o } = n;
    let s = 0;
    if (o === 0) {
      const a = e ?? this.virtualizer.getWindowSpecs();
      this.top < a.top && (s = r);
    }
    return { topOffset: this.top + i + s, height: this.height - (i + r) };
  }
  cleanUp(e = !1) {
    const t = this.cache.ghostTextRows.size > 0;
    t && (this.layoutDirty = !0);
    const n =
      e &&
      this.isAdvancedMode() &&
      this.fileContainer != null &&
      (t || !B(this.getRenderedDiff(), this.getLatestDiff()));
    (this.fileContainer != null &&
      this.isSimpleMode() &&
      this.getSimpleVirtualizer()?.disconnect(this.fileContainer),
      e ||
        (this.resetLayoutCache({ includeEstimatedHeights: !0 }),
        (this.pendingExpansions = void 0),
        (this.pendingHydratedDiff = void 0)),
      (this.pendingRender = void 0),
      (this.isSetup = !1),
      super.cleanUp(e),
      n && this.virtualizer.instanceChanged(this, !0));
  }
  expandHunk = (e, t, n) => {
    this.fileDiff != null &&
      (this.isAdvancedMode()
        ? ((this.pendingExpansions ??= []),
          this.pendingExpansions.push({
            hunkIndex: e,
            direction: t,
            expansionLineCountOverride: n,
          }))
        : (this.hunksRenderer.expandHunk(e, t, n),
          this.resetLayoutCache({ includeEstimatedHeights: !0 }),
          this.computeApproximateSize()),
      this.loadFilesIfNecessary(),
      (this.forceRenderOverride = !0),
      this.virtualizer.instanceChanged(this, !0));
  };
  async handleFilesLoaded(e, t) {
    if (!(this.fileDiff !== e || !e.isPartial)) {
      if (this.isAdvancedMode()) {
        const n = hn("clone", e, t);
        if (
          (await pn(() => this.primeHighlightCache(n)),
          !this.enabled || this.fileDiff !== e)
        )
          return;
        this.pendingHydratedDiff = { expectedDiff: e, nextDiff: n, files: t };
      } else {
        if (
          (hn("merge", e, t),
          this.setHydratedState(t),
          !this.startHydratedEditSession(e) &&
            (await pn(() => this.primeHighlightCache(e)),
            !this.enabled || this.fileDiff !== e))
        )
          return;
        (this.resetLayoutCache({ includeEstimatedHeights: !0 }),
          this.computeApproximateSize());
      }
      ((this.forceRenderOverride = !0),
        this.virtualizer.instanceChanged(this, !0));
    }
  }
  loadFilesIfNecessary() {
    if (this.pendingHydratedDiff != null) {
      if (this.pendingHydratedDiff.expectedDiff === this.fileDiff) return;
      this.pendingHydratedDiff = void 0;
    }
    super.loadFilesIfNecessary();
  }
  isLineRenderable(e) {
    if (super.isLineRenderable(e)) return !0;
    const { pendingExpansions: t } = this,
      n = this.getRenderedDiff();
    if (t == null || t.length === 0 || n == null) return !1;
    const { expansionLineCount: i = 100, collapsedContextThreshold: r = 1 } =
        this.options,
      o = new Map(this.hunksRenderer.getExpandedHunksMap());
    for (const s of t) {
      const a = { ...(o.get(s.hunkIndex) ?? { fromStart: 0, fromEnd: 0 }) },
        l = s.expansionLineCountOverride ?? i;
      ((s.direction === "up" || s.direction === "both") && (a.fromStart += l),
        (s.direction === "down" || s.direction === "both") && (a.fromEnd += l),
        o.set(s.hunkIndex, a));
    }
    return Cr({
      fileDiff: n,
      lineNumber: e,
      expandedHunks: o,
      collapsedContextThreshold: r,
    });
  }
  invalidateEditSessionLayout() {
    (this.getSimpleVirtualizer()?.markDOMDirty(),
      this.resetLayoutCache({
        forceSimpleRecompute: this.isSimpleMode(),
        includeEstimatedHeights: !0,
        resetRenderRange: !1,
      }),
      this.isSimpleMode() || this.computeApproximateSize(!0),
      this.getSimpleVirtualizer()?.requestHeightReconcile(this));
  }
  escalateEditSessionRender() {
    (this.invalidateEditSessionLayout(),
      !(!this.enabled || this.fileDiff == null) &&
        ((this.forceRenderOverride = !0),
        this.virtualizer.instanceChanged(this, !0)));
  }
  shouldSelfHealEditSession() {
    return !this.isAdvancedMode() && super.shouldSelfHealEditSession();
  }
  setVisibility(e) {
    this.isAdvancedMode() ||
      this.fileContainer == null ||
      ((this.renderRange = void 0),
      e && !this.isVisible
        ? ((this.top = this.getVirtualizedTop()), (this.isVisible = !0))
        : !e && this.isVisible && ((this.isVisible = !1), this.rerender()));
  }
  rerender() {
    if (
      !this.enabled ||
      (this.fileDiff == null &&
        this.additionFile == null &&
        this.deletionFile == null)
    )
      return;
    const e = this.getLatestDiff(),
      t = e == null ? void 0 : this.hunksRenderer.getDiffForNextRender(e);
    (this.isSimpleMode() && (this.pendingRender = void 0),
      (this.forceRenderOverride = !0),
      this.virtualizer.instanceChanged(this, !B(this.getRenderedDiff(), t)));
  }
  syncGhostTextRows() {
    const e = this.getAdvancedVirtualizer();
    e != null
      ? (e.capturePendingLayoutAnchor(),
        (this.layoutDirty = !0),
        e.instanceChanged(this, !0))
      : this.getSimpleVirtualizer()?.requestHeightReconcile(this);
  }
  applyDocumentChange(e, t, n = !1) {
    const { renderRange: i } = this;
    (this.getAdvancedVirtualizer()?.capturePendingLayoutAnchor(),
      super.applyDocumentChange(e, t),
      this.getSimpleVirtualizer()?.markDOMDirty(),
      this.resetLayoutCache({
        forceSimpleRecompute: this.isSimpleMode(),
        includeEstimatedHeights: !0,
        resetRenderRange: !1,
      }));
    const r = this.getRenderedDiff();
    if (!this.isSimpleMode()) this.computeApproximateSize(!0);
    else if (n && i != null && r != null) {
      const o = this.virtualizer.getWindowSpecs(),
        s = this.computeRenderRangeFromWindow(r, this.top ?? 0, o);
      s.bufferAfter !== i.bufferAfter && this.updateBuffers(s);
    }
    ((this.forceRenderOverride = !0),
      this.virtualizer.instanceChanged(this, !0));
  }
  computeApproximateSize(e = !1, t = this.getLayoutDiff()) {
    const n = this.isResizeDebuggingEnabled();
    if (!e && !this.layoutDirty && !n) return;
    const i = this.height === 0;
    if (
      ((this.height = 0),
      (this.cache.checkpoints = []),
      (this.cache.totalLines = 0),
      t == null)
    ) {
      this.layoutDirty = !1;
      return;
    }
    this.applyGhostTextRows(t, this.editor?.__getGhostTextRows() ?? Et);
    const { disableFileHeader: r = !1, collapsed: o = !1 } = this.options,
      s = ue(this.metrics, r);
    if (((this.height += s), o)) {
      this.layoutDirty = !1;
      return;
    }
    ((this.height =
      this.getActiveEstimatedHeight(t) + this.cache.measuredHeightDeltaTotal),
      n && !i && this.validateComputedHeight(t),
      (this.layoutDirty = !1));
  }
  getLayoutDiff() {
    return this.pendingRender?.diff ?? this.getRenderedDiff();
  }
  getActiveEstimatedHeight(e = this.getLayoutDiff()) {
    this.ensureEstimatedDiffHeights(e);
    const t =
      this.getDiffStyle() === "split"
        ? this.cache.estimatedSplitHeight
        : this.cache.estimatedUnifiedHeight;
    if (t == null)
      throw new Error(
        "VirtualizedFileDiff.getActiveEstimatedHeight: missing estimated height",
      );
    return t;
  }
  ensureEstimatedDiffHeights(e = this.getLayoutDiff()) {
    if (e == null) {
      ((this.cache.estimatedSplitHeight = void 0),
        (this.cache.estimatedUnifiedHeight = void 0));
      return;
    }
    if (
      this.cache.estimatedSplitHeight != null &&
      this.cache.estimatedUnifiedHeight != null
    )
      return;
    const {
        disableFileHeader: t = !1,
        expandUnchanged: n = !1,
        collapsedContextThreshold: i = 1,
      } = this.options,
      { splitHeight: r, unifiedHeight: o } = Ba({
        fileDiff: e,
        metrics: this.metrics,
        disableFileHeader: t,
        hunkSeparators: this.getHunkSeparatorType(),
        expandUnchanged: n,
        expandedHunks: this.hunksRenderer.getExpandedHunksMap(),
        collapsedContextThreshold: i,
        canHydratePartialDiff: en(e, this.options.loadDiffFiles != null),
      });
    ((this.cache.estimatedSplitHeight = r),
      (this.cache.estimatedUnifiedHeight = o));
  }
  validateComputedHeight(e = this.getLayoutDiff()) {
    if (this.fileContainer == null || e == null) return;
    const t = this.fileContainer.getBoundingClientRect();
    t.height !== this.height
      ? console.log(
          "VirtualizedFileDiff.computeApproximateSize: computed height doesnt match",
          {
            name: e.name,
            elementHeight: t.height,
            computedHeight: this.height,
          },
        )
      : console.log(
          "VirtualizedFileDiff.computeApproximateSize: computed height IS CORRECT",
        );
  }
  render({
    fileContainer: e,
    fileDiff: t,
    forceRender: n = !1,
    lineAnnotations: i,
    ...r
  } = {}) {
    const o = gn(r, "VirtualizedFileDiff.render"),
      s = o != null,
      a = o?.oldFile,
      l = o?.newFile,
      d = s && (!hi(a, this.deletionFile) || !hi(l, this.additionFile));
    let h = t ?? this.fileDiff;
    t == null &&
      s &&
      (d || this.fileDiff == null) &&
      (h = Oe(o.oldFile, o.newFile, this.options.parseDiffOptions));
    const { forceRenderOverride: c, isSetup: u } = this;
    this.forceRenderOverride = void 0;
    const f = h != null && !B(this.fileDiff, h),
      p = f || d;
    if (((e = this.getOrCreateFileContainer(e)), h == null))
      return (
        console.error(
          "VirtualizedFileDiff.render: attempting to virtually render when we dont have the correct data",
        ),
        !1
      );
    f && this.updateExternalDiff(h, i);
    const {
      pendingRenderDiff: g,
      layoutDiffChanged: m,
      renderedDiffChanged: y,
      annotationsChanged: S,
    } = this.pendingRender != null &&
    this.pendingRender.latestDiff === (this.getLatestDiff(h) ?? h)
      ? {
          pendingRenderDiff: this.pendingRender.diff,
          layoutDiffChanged: !1,
          renderedDiffChanged: !1,
          annotationsChanged: !1,
        }
      : this.updatePendingRender(h, i);
    if (((S || m) && this.resetLayoutCache({ includeEstimatedHeights: m }), u))
      ((this.top ??= this.getVirtualizedTop()),
        m &&
          (this.getSimpleVirtualizer()?.markDOMDirty(),
          this.computeApproximateSize(!1, g)));
    else {
      this.computeApproximateSize(!1, g);
      const k = this.getSimpleVirtualizer();
      if (((this.top ??= this.getVirtualizedTop()), this.isAdvancedMode()))
        this.isVisible = !0;
      else {
        if (k == null)
          throw new Error(
            "VirtualizedFileDiff.render: simple virtualizer is not available",
          );
        (k.connect(e, this),
          (this.isVisible = k.isInstanceVisible(this.top ?? 0, this.height)));
      }
      this.isSetup = !0;
    }
    if (!this.isVisible && this.isSimpleMode() && (!p || !u))
      return (
        o != null && ((this.deletionFile = a), (this.additionFile = l)),
        f && this.clearReusableHeader(),
        (this.pendingRender = void 0),
        this.renderPlaceholder(this.height)
      );
    const b = this.virtualizer.getWindowSpecs(),
      L = this.top ?? 0,
      v = this.computeRenderRangeFromWindow(g, L, b);
    return super.render({
      fileDiff: h,
      fileContainer: e,
      renderRange: v,
      lineAnnotations: i,
      forceRender: (c ?? n) || S || y || f,
      ...o,
      ...r,
    });
  }
  finalizeRender() {
    if (this.getRenderedDiff() !== this.pendingRender?.diff)
      throw new Error(
        "VirtualizedFileDiff.render: rendered a different diff than its prepared layout",
      );
    ((this.pendingRender = void 0),
      this.isSimpleMode() &&
        this.getSimpleVirtualizer()?.requestHeightReconcile(this));
  }
  updatePendingRender(e, t) {
    const n = this.getLatestDiff(e) ?? e,
      i = this.getRenderedDiff(),
      { pendingRender: r } = this,
      o = r?.diff ?? i,
      s = this.hunksRenderer.getDiffForNextRender(n);
    return (
      (this.pendingRender = { latestDiff: n, diff: s }),
      {
        pendingRenderDiff: s,
        annotationsChanged: this.syncLineAnnotations(t),
        layoutDiffChanged: !B(o, s),
        renderedDiffChanged: !B(i, s),
      }
    );
  }
  syncVirtualizedTop() {
    this.top = this.getVirtualizedTop();
  }
  shouldDisableVirtualizationBuffers() {
    return this.isAdvancedMode() || super.shouldDisableVirtualizationBuffers();
  }
  shouldGuardRebuildScroll() {
    return !1;
  }
  isSimpleMode() {
    return this.virtualizer.type === "simple";
  }
  isAdvancedMode() {
    return this.virtualizer.type === "advanced";
  }
  getVirtualizedTop() {
    return this.virtualizer.type === "advanced"
      ? this.virtualizer.getLocalTopForInstance(this)
      : this.fileContainer != null
        ? this.virtualizer.getOffsetInScrollContainer(this.fileContainer)
        : 0;
  }
  getSimpleVirtualizer() {
    return this.virtualizer.type === "simple" ? this.virtualizer : void 0;
  }
  getAdvancedVirtualizer() {
    return this.virtualizer.type === "advanced" ? this.virtualizer : void 0;
  }
  isResizeDebuggingEnabled() {
    return this.getSimpleVirtualizer()?.config.resizeDebugging ?? !1;
  }
  getDiffStyle() {
    return this.options.diffStyle ?? "split";
  }
  getHunkSeparatorType() {
    return Ja(this.options.hunkSeparators);
  }
  approximateLayoutCheckpoints(e) {
    if (
      (!this.layoutDirty && this.cache.checkpoints.length > 0) ||
      e.hunks.length === 0 ||
      this.options.collapsed === !0
    )
      return;
    const {
        disableFileHeader: t = !1,
        expandUnchanged: n = !1,
        collapsedContextThreshold: i = 1,
      } = this.options,
      r = e.hunks.length - 1,
      o = en(e, this.options.loadDiffFiles != null),
      s = this.getDiffStyle(),
      a = this.getHunkSeparatorType(),
      l = n ? !0 : this.hunksRenderer.getExpandedHunksMap(),
      d = ja(this.cache.heightDeltas);
    let h = ue(this.metrics, t) + this.cache.fileAnnotationHeight,
      c = 0;
    const u = ({
      rowCount: f,
      startLineIndex: p,
      preSeparatorHeight: g = 0,
      postSeparatorHeight: m = 0,
      metadataOffsets: y = [],
    }) => {
      if (f <= 0) return;
      const S = c,
        b = c + f;
      let L = Ka(S);
      for (; L < b;) {
        const v = L - S,
          k =
            h +
            (v > 0 ? g : 0) +
            v * this.metrics.lineHeight +
            Xa(y, v) * this.metrics.lineHeight +
            li(d, p, p + v);
        (this.cache.checkpoints.push({
          renderedLineIndex: L,
          lineIndex: p + v,
          top: k,
        }),
          (L += mn));
      }
      ((h +=
        g +
        f * this.metrics.lineHeight +
        y.length * this.metrics.lineHeight +
        li(d, p, p + f) +
        m),
        (c = b));
    };
    for (let f = 0; f < e.hunks.length; f++) {
      const p = e.hunks[f];
      if (p == null)
        throw new Error(
          "VirtualizedFileDiff.approximateLayoutCheckpoints: invalid hunk index",
        );
      const g = ke({
          isPartial: e.isPartial,
          rangeSize: p.collapsedBefore,
          expandedHunks: l,
          hunkIndex: f,
          collapsedContextThreshold: i,
        }),
        m =
          g.collapsedLines > 0
            ? (nt({
                type: a,
                metrics: this.metrics,
                hunkIndex: f,
                hunkSpecs: p.hunkSpecs,
              })?.totalHeight ?? 0)
            : 0;
      u({
        rowCount: g.fromStart,
        startLineIndex:
          (s === "split" ? p.splitLineStart : p.unifiedLineStart) - g.rangeSize,
      });
      let y = m;
      (u({
        rowCount: g.fromEnd,
        startLineIndex:
          (s === "split" ? p.splitLineStart : p.unifiedLineStart) - g.fromEnd,
        preSeparatorHeight: y,
      }),
        g.fromEnd > 0 && (y = 0));
      const S =
          f === r
            ? we({
                fileDiff: e,
                hunkIndex: f,
                expandedHunks: l,
                collapsedContextThreshold: i,
                errorPrefix: "VirtualizedFileDiff",
              })
            : void 0,
        b =
          S != null && S.collapsedLines > 0
            ? (Me({ type: a, metrics: this.metrics })?.totalHeight ?? 0)
            : f === r && o
              ? (Me({ type: a, metrics: this.metrics })?.totalHeight ?? 0)
              : 0,
        L = S != null ? S.fromStart + S.fromEnd : 0,
        v = s === "split" ? p.splitLineCount : p.unifiedLineCount,
        k = s === "split" ? p.splitLineStart : p.unifiedLineStart;
      (u({
        rowCount: v,
        startLineIndex: k,
        preSeparatorHeight: y,
        postSeparatorHeight: L === 0 ? b : 0,
        metadataOffsets: Ya({ diffStyle: s, hunk: p, rowCount: v }),
      }),
        S != null &&
          L > 0 &&
          u({ rowCount: L, startLineIndex: k + v, postSeparatorHeight: b }));
    }
    this.cache.totalLines = c;
  }
  getLayoutCheckpointBeforeLineIndex(e) {
    if (e <= 0 || this.cache.checkpoints.length === 0) return;
    let t = 0,
      n = this.cache.checkpoints.length - 1,
      i;
    for (; t <= n;) {
      const r = (t + n) >> 1,
        o = this.cache.checkpoints[r];
      if (o == null)
        throw new Error("VirtualizedFileDiff: invalid checkpoint index");
      o.lineIndex <= e ? ((i = o), (t = r + 1)) : (n = r - 1);
    }
    return i;
  }
  getLayoutCheckpointBeforeTop(e, t) {
    let n = 0,
      i = this.cache.checkpoints.length - 1,
      r = -1;
    for (; n <= i;) {
      const o = (n + i) >> 1,
        s = this.cache.checkpoints[o];
      if (s == null)
        throw new Error("VirtualizedFileDiff: invalid checkpoint index");
      s.top <= e ? ((r = o), (n = o + 1)) : (i = o - 1);
    }
    if (t == null) return r >= 0 ? this.cache.checkpoints[r] : void 0;
    for (let o = r; o >= 0; o--) {
      const s = this.cache.checkpoints[o];
      if (s == null)
        throw new Error("VirtualizedFileDiff: invalid checkpoint index");
      if (s.renderedLineIndex % t === 0) return s;
    }
  }
  getExpandedLineCount(e, t) {
    let n = 0;
    if (e.isPartial) {
      for (const a of e.hunks)
        n += t === "split" ? a.splitLineCount : a.unifiedLineCount;
      return n;
    }
    const { expandUnchanged: i = !1, collapsedContextThreshold: r = 1 } =
        this.options,
      o = i ? !0 : this.hunksRenderer.getExpandedHunksMap();
    for (const [a, l] of e.hunks.entries()) {
      const d = t === "split" ? l.splitLineCount : l.unifiedLineCount;
      n += d;
      const h = Math.max(l.collapsedBefore, 0),
        {
          fromStart: c,
          fromEnd: u,
          renderAll: f,
        } = ke({
          isPartial: e.isPartial,
          rangeSize: h,
          expandedHunks: o,
          hunkIndex: a,
          collapsedContextThreshold: r,
        });
      h > 0 && (n += f ? h : c + u);
    }
    const s = we({
      fileDiff: e,
      hunkIndex: e.hunks.length - 1,
      expandedHunks: o,
      collapsedContextThreshold: r,
      errorPrefix: "VirtualizedFileDiff",
    });
    return (s != null && (n += s.fromStart + s.fromEnd), n);
  }
  getLayoutLineCount(e, t) {
    const n = this.getExpandedLineCount(e, t),
      i = t === "split" ? e.splitLineCount : e.unifiedLineCount;
    return Math.max(
      n,
      i,
      e.additionLines.length,
      e.deletionLines.length,
      this.cache.totalLines,
    );
  }
  computeRenderRangeFromWindow(e, t, { top: n, bottom: i }) {
    const {
        disableFileHeader: r = !1,
        expandUnchanged: o = !1,
        collapsedContextThreshold: s = 1,
      } = this.options,
      { hunkLineCount: a, lineHeight: l } = this.metrics,
      d = this.getDiffStyle(),
      h = this.getHunkSeparatorType(),
      c = en(e, this.options.loadDiffFiles != null),
      u = this.height;
    let f = this.getLayoutLineCount(e, d);
    const p = ue(this.metrics, r),
      g = e.hunks.length > 0 ? Ut(this.metrics) : 0,
      { fileAnnotationHeight: m } = this.cache,
      y = p + m,
      S = Math.max(0, u - p - m - g),
      b = this.hasFileAnnotations(e),
      L = t + p,
      v = m > 0 && b && L < i && L + m > n;
    if (t < n - u || t > i)
      return {
        startingLine: 0,
        totalLines: 0,
        bufferBefore: 0,
        bufferAfter: u - p - g,
      };
    if (f <= a || e.hunks.length === 0)
      return {
        startingLine: 0,
        totalLines: a,
        bufferBefore: 0,
        bufferAfter: 0,
      };
    (this.approximateLayoutCheckpoints(e), (f = this.getLayoutLineCount(e, d)));
    const k = Math.ceil(Math.max(i - n, 0) / l),
      C = Math.ceil(k / a) * a + a,
      x = C / a,
      A = x,
      w = [],
      O = (n + i) / 2,
      T = this.getLayoutCheckpointBeforeTop(Math.max(0, n - t - C * l * 2), a);
    let F = t + (T?.top ?? y),
      H = T?.renderedLineIndex ?? 0,
      N,
      E,
      I;
    if (
      (Fe({
        diff: e,
        diffStyle: d,
        startingLine: T?.renderedLineIndex ?? 0,
        expandedHunks: o ? !0 : this.hunksRenderer.getExpandedHunksMap(),
        collapsedContextThreshold: s,
        callback: ({
          hunkIndex: Ie,
          hunk: K,
          collapsedBefore: _e,
          collapsedAfter: Xe,
          deletionLine: pe,
          additionLine: ge,
        }) => {
          const _ = ge != null ? ge.splitLineIndex : pe.splitLineIndex,
            Y = ge != null ? ge.unifiedLineIndex : pe.unifiedLineIndex,
            ne = (ge?.noEOFCR ?? !1) || (pe?.noEOFCR ?? !1),
            He =
              Ie === e.hunks.length - 1 &&
              K != null &&
              (d === "split"
                ? _ === K.splitLineStart + K.splitLineCount - 1
                : Y === K.unifiedLineStart + K.unifiedLineCount - 1),
            oe =
              (_e > 0
                ? nt({
                    type: h,
                    metrics: this.metrics,
                    hunkIndex: Ie,
                    hunkSpecs: K?.hunkSpecs,
                  })
                : void 0
              )?.totalHeight ?? 0;
          F += oe;
          const se = H % a === 0,
            Ee = Math.floor(H / a);
          if (se && ((w[Ee] = F - (t + y + oe)), I != null)) {
            if (I <= 0) return !0;
            I--;
          }
          const Q = this.getLineHeight(d === "split" ? _ : Y, ne);
          if (
            (F > n - Q && F < i && (N ??= Ee),
            E == null && F + Q > O && (E = Ee),
            I == null && F >= i && se && (I = A),
            H++,
            (F += Q),
            Xe > 0 || (He && c))
          ) {
            const ut = Me({ type: h, metrics: this.metrics });
            ut != null &&
              (F < i && F + ut.totalHeight > n && (N ??= Ee),
              E == null && F + ut.totalHeight > O && (E = Ee),
              (F += ut.totalHeight));
          }
          return !1;
        },
      }),
      N == null)
    )
      if (v) ((N = 0), (E = 0));
      else
        return {
          startingLine: 0,
          totalLines: 0,
          bufferBefore: 0,
          bufferAfter: u - p - g,
        };
    E ??= N;
    const D = Math.round(E - x / 2),
      G = Math.max(0, Math.ceil(f / a) - x),
      M = Math.max(0, Math.min(D, G)),
      P = M * a,
      z = D < 0 ? C + D * a : C,
      V = w[M] ?? 0,
      te = P === 0 ? 0 : m + V,
      ie = M + z / a,
      re = ie < w.length ? S - w[ie] : S - (F - t - y);
    return {
      startingLine: P,
      totalLines: z,
      bufferBefore: te,
      bufferAfter: Math.max(0, re),
    };
  }
};
function qa(e) {
  let t;
  for (const n of e) {
    if (n == null) continue;
    const i = n.children[1];
    if (i instanceof HTMLElement)
      for (const r of i.children)
        r instanceof HTMLElement &&
          r.dataset.lineAnnotation === er &&
          (t = Math.max(t ?? 0, r.getBoundingClientRect().height));
  }
  return t;
}
function ja(e) {
  const t = Array.from(e).sort((o, s) => o[0] - s[0]),
    n = [],
    i = [0];
  let r = 0;
  for (const [o, s] of t) (n.push(o), (r += s), i.push(r));
  return { lineIndexes: n, prefixTotals: i };
}
function li({ lineIndexes: e, prefixTotals: t }, n, i) {
  if (n >= i || e.length === 0) return 0;
  const r = di(e, n);
  return (t[di(e, i)] ?? 0) - (t[r] ?? 0);
}
function di(e, t) {
  let n = 0,
    i = e.length;
  for (; n < i;) {
    const r = (n + i) >> 1,
      o = e[r];
    if (o == null) throw new Error("VirtualizedFileDiff: invalid prefix index");
    o < t ? (n = r + 1) : (i = r);
  }
  return n;
}
function Ka(e) {
  return Math.ceil(e / mn) * mn;
}
function Xa(e, t) {
  let n = 0;
  for (const i of e) i < t && n++;
  return n;
}
function Ya({ diffStyle: e, hunk: t, rowCount: n }) {
  if (n <= 0 || (!t.noEOFCRAdditions && !t.noEOFCRDeletions)) return [];
  const i = t.hunkContent.at(-1);
  if (i == null) return [];
  if (i.type === "context") return [n - 1];
  const r = Math.max(i.deletions, i.additions),
    o = i.deletions + i.additions;
  if (e === "split")
    return r > 0 && (t.noEOFCRAdditions || t.noEOFCRDeletions) ? [n - 1] : [];
  const s = [],
    a = n - o;
  return (
    i.deletions > 0 && t.noEOFCRDeletions && s.push(a + i.deletions - 1),
    i.additions > 0 && t.noEOFCRAdditions && s.push(n - 1),
    s
  );
}
function Qa(e, t) {
  return (
    (e.diffStyle ?? "split") !== (t.diffStyle ?? "split") ||
    (e.overflow ?? "scroll") !== (t.overflow ?? "scroll") ||
    (e.collapsed ?? !1) !== (t.collapsed ?? !1) ||
    (e.disableLineNumbers ?? !1) !== (t.disableLineNumbers ?? !1) ||
    (e.disableFileHeader ?? !1) !== (t.disableFileHeader ?? !1) ||
    (e.diffIndicators ?? "bars") !== (t.diffIndicators ?? "bars") ||
    (e.hunkSeparators ?? "line-info") !== (t.hunkSeparators ?? "line-info") ||
    !!e.loadDiffFiles != !!t.loadDiffFiles ||
    (e.expandUnchanged ?? !1) !== (t.expandUnchanged ?? !1) ||
    (e.collapsedContextThreshold ?? 1) !== (t.collapsedContextThreshold ?? 1) ||
    e.unsafeCSS !== t.unsafeCSS
  );
}
function Za(e, t) {
  return (
    (e.disableFileHeader ?? !1) !== (t.disableFileHeader ?? !1) ||
    (e.hunkSeparators ?? "line-info") !== (t.hunkSeparators ?? "line-info") ||
    !!e.loadDiffFiles != !!t.loadDiffFiles ||
    (e.expandUnchanged ?? !1) !== (t.expandUnchanged ?? !1) ||
    (e.collapsedContextThreshold ?? 1) !== (t.collapsedContextThreshold ?? 1)
  );
}
function en(e, t) {
  return (
    e.isPartial && t && (e.type === "change" || e.type === "rename-changed")
  );
}
function Ja(e) {
  return typeof e == "function" ? "custom" : (e ?? "line-info");
}
function hi(e, t) {
  return e == null || t == null ? e == null && t == null : ot(e, t);
}
function el(e, t) {
  const [n, i] = e.split(",").map(Number);
  return t === "split" ? i : n;
}
function tl(e, t) {
  return e == null || t == null
    ? e === t
    : e.header !== t.header || e.footer !== t.footer
      ? !1
      : nl(e.items, t.items);
}
function nl(e, t) {
  if (e == null || t == null) return e === t;
  if (e.length !== t.length) return !1;
  for (let n = 0; n < e.length; n++) {
    const i = e[n],
      r = t[n];
    if (
      i == null ||
      r == null ||
      i.id !== r.id ||
      i.type !== r.type ||
      i.element !== r.element ||
      i.instance !== r.instance ||
      i.version !== r.version
    )
      return !1;
  }
  return !0;
}
function il(e, t, n) {
  const i = document.createElement("div");
  return (
    (i.style.display = "flow-root"),
    e === "header"
      ? (i.setAttribute(qr, ""), t.before(i))
      : (i.setAttribute(jr, ""), t.after(i)),
    n?.observe(i),
    i
  );
}
function de(e) {
  const t = window.devicePixelRatio ?? 1;
  return Math.round(e * t) / t;
}
const rl = [
    "theme",
    "disableLineNumbers",
    "overflow",
    "themeType",
    "disableFileHeader",
    "disableVirtualizationBuffers",
    "preferredHighlighter",
    "useCSSClasses",
    "useTokenTransformer",
    "tokenizeMaxLineLength",
    "tokenizeMaxLength",
    "unsafeCSS",
    "diffStyle",
    "diffIndicators",
    "disableBackground",
    "expandUnchanged",
    "loadDiffFiles",
    "collapsedContextThreshold",
    "lineDiffType",
    "maxLineDiffLength",
    "expansionLineCount",
    "lineHoverHighlight",
    "enableTokenInteractionsOnWhitespace",
    "enableGutterUtility",
    "__debugPointerEvents",
    "enableLineSelection",
    "controlledSelection",
    "disableErrorHandling",
  ],
  ol = [
    "theme",
    "disableLineNumbers",
    "overflow",
    "themeType",
    "disableFileHeader",
    "disableVirtualizationBuffers",
    "preferredHighlighter",
    "useCSSClasses",
    "useTokenTransformer",
    "tokenizeMaxLineLength",
    "tokenizeMaxLength",
    "unsafeCSS",
    "lineHoverHighlight",
    "enableTokenInteractionsOnWhitespace",
    "enableGutterUtility",
    "__debugPointerEvents",
    "enableLineSelection",
    "controlledSelection",
    "disableErrorHandling",
  ],
  ci = [
    "renderCustomHeader",
    "renderHeaderPrefix",
    "renderHeaderFilenameSuffix",
    "renderHeaderMetadata",
    "renderAnnotation",
    "renderGutterUtility",
    "onPostRender",
    "onGutterUtilityClick",
    "onLineClick",
    "onLineNumberClick",
    "onLineEnter",
    "onLineLeave",
    "onTokenClick",
    "onTokenEnter",
    "onTokenLeave",
  ],
  ui = [
    "onLineSelected",
    "onLineSelectionStart",
    "onLineSelectionChange",
    "onLineSelectionEnd",
  ],
  Tr = Symbol("CodeView.itemOptionsState");
function fi(e, t) {
  Object.defineProperty(e, Tr, { configurable: !1, enumerable: !1, value: t });
}
function Ae(e) {
  return e[Tr];
}
function ae(e, t, n) {
  Object.defineProperty(e, t, {
    configurable: !1,
    enumerable: !0,
    get() {
      return n(this);
    },
  });
}
const sl = 120,
  pi = 1,
  gi = "--diffs-overflow-override",
  Fn = 12e6,
  Ar = 1e6,
  Ir = 2e6,
  al = Fn - Ir,
  mi = Fn - Ar;
function ll(e) {
  if (e == null)
    throw new Error(
      "CodeView: createEditor is required for items with edit: true",
    );
}
const dl = (() => {
  const { navigator: e } = globalThis,
    t = e.userAgent,
    n = /iP(?:hone|ad|od)/.test(t),
    i = e.platform === "MacIntel" && e.maxTouchPoints > 1;
  return (
    (n || i) &&
    /AppleWebKit/.test(t) &&
    /Safari/.test(t) &&
    !/(CriOS|FxiOS|EdgiOS|OPiOS)/.test(t)
  );
})();
var Ql = class be {
  static __STOP = !1;
  static __lastScrollPosition = 0;
  type = "advanced";
  config = {
    overscrollSize: 200,
    intersectionObserverMargin: 0,
    resizeDebugging: !1,
  };
  items = [];
  idToItem = new Map();
  selectedLines = null;
  itemEditors = new Map();
  attachedEditors = new Set();
  instanceToItem = new Map();
  layoutDirtyIndex;
  pendingLayoutReset;
  renderOptionsRevision = 0;
  slotCoordinator;
  slotSnapshot;
  scrollListeners = new Set();
  scrollHeight = 0;
  containerHeight = -1;
  scrollTop = 0;
  scrollPageOffset = 0;
  scrollDirty = !0;
  scrollInteractionFixTimer;
  pointerEventsDisabled = !1;
  codeOverflowFix = !1;
  height = 0;
  heightDirty = !0;
  windowSpecs = { top: 0, bottom: 0 };
  renderState = {
    scrollTop: -1,
    firstIndex: -1,
    lastIndex: -1,
    stickyHeight: 0,
    stickyTop: -1,
    stickyBottom: -1,
  };
  itemMetricsCache = Ye;
  fileOptionsPrototype;
  diffOptionsPrototype;
  pendingScrollTarget;
  pendingLayoutAnchor;
  shouldFixContainerFocus = !1;
  scrollAnimation;
  root;
  resizeObserver;
  container = document.createElement("div");
  stickyContainer = document.createElement("div");
  stickyOffset = document.createElement("div");
  header = { element: void 0, render: void 0, height: 0 };
  footer = { element: void 0, render: void 0, height: 0 };
  elementPool = [];
  elementPoolVersion = 0;
  elementPoolTracker = new WeakMap();
  pendingElementPool = [];
  options;
  workerManager;
  isReadySubscription;
  pendingHighlighterTheme;
  isContainerManaged;
  constructor(t = { theme: U }, n, i = !1) {
    ((this.options = t),
      this.computeMetricsCache(t.itemMetrics),
      (this.fileOptionsPrototype = this.createFileOptionsPrototype()),
      (this.diffOptionsPrototype = this.createDiffOptionsPrototype()),
      (this.workerManager = n),
      (this.isContainerManaged = i),
      (this.stickyOffset.style.contain = "layout size"),
      (this.stickyContainer.style.position = "sticky"),
      (this.stickyContainer.style.width = "100%"),
      (this.stickyContainer.style.contain = "layout style inline-size"),
      (this.stickyContainer.style.isolation = "isolate"),
      (this.stickyContainer.style.display = "flex"),
      (this.stickyContainer.style.flexDirection = "column"));
  }
  getLayout() {
    return this.options.layout ?? Kr;
  }
  getItemTopOffset() {
    return this.getLayout().paddingTop + this.header.height;
  }
  computeMetricsCache(t) {
    return (
      (this.itemMetricsCache = {
        hunkLineCount: t?.hunkLineCount ?? Ye.hunkLineCount,
        lineHeight: t?.lineHeight ?? Ye.lineHeight,
        diffHeaderHeight: t?.diffHeaderHeight ?? Ye.diffHeaderHeight,
        hunkSeparatorHeight: t?.hunkSeparatorHeight,
        spacing: t?.spacing ?? Ye.spacing,
        paddingTop: t?.paddingTop,
        paddingBottom: t?.paddingBottom,
      }),
      this.itemMetricsCache
    );
  }
  getSmoothScrollSettings() {
    return this.options.smoothScrollSettings ?? Xr;
  }
  shouldDisablePointerEvents() {
    return this.options.pointerEventsOnScroll !== !0;
  }
  shouldValidateItemHeights() {
    return Yr && this.options.__devOnlyValidateItemHeights === !0;
  }
  validateRenderedItemHeight(t) {
    if (!this.shouldValidateItemHeights() || t.element == null) return;
    const n = t.instance.getAdvancedStickySpecs();
    if (n == null) return;
    const i = n.height,
      r = t.element.getBoundingClientRect().height;
    i !== r &&
      console.error(
        "CodeView: reconciled item height does not match DOM height",
        {
          id: t.item.id,
          type: t.type,
          index: t.index,
          version: t.version,
          expectedHeight: i,
          actualHeight: r,
          delta: r - i,
          stickyTopOffset: n.topOffset,
          virtualizedHeight: t.instance.getVirtualizedHeight(),
          top: t.top,
          scrollTop: this.getScrollTop(),
          windowSpecs: { ...this.windowSpecs },
          element: t.element,
          instance: t.instance,
        },
      );
  }
  validateStickyContainerHeight() {
    if (!this.shouldValidateItemHeights()) return;
    const {
      firstIndex: t,
      lastIndex: n,
      stickyHeight: i,
      stickyTop: r,
      stickyBottom: o,
    } = this.renderState;
    if (t === -1 || n === -1) return;
    const s = this.stickyContainer.getBoundingClientRect().height;
    Math.abs(s - i) < pi ||
      console.error(
        "CodeView: sticky container height does not match computed layout",
        {
          computedStickyHeight: i,
          actualStickyHeight: s,
          delta: s - i,
          stickyTop: r,
          stickyBottom: o,
          firstIndex: t,
          lastIndex: n,
          firstStickySpecs: this.items[t]?.instance.getAdvancedStickySpecs(),
          lastStickySpecs: this.items[n]?.instance.getAdvancedStickySpecs(),
          scrollTop: this.getScrollTop(),
          scrollPageOffset: this.scrollPageOffset,
          windowSpecs: { ...this.windowSpecs },
          stickyContainer: this.stickyContainer,
        },
      );
  }
  clearScrollInteractionTimer() {
    this.scrollInteractionFixTimer != null &&
      (clearTimeout(this.scrollInteractionFixTimer),
      (this.scrollInteractionFixTimer = void 0));
  }
  suspendScrollInteractions() {
    (this.clearScrollInteractionTimer(),
      this.shouldDisablePointerEvents() &&
        !this.pointerEventsDisabled &&
        ((this.stickyContainer.style.pointerEvents = "none"),
        (this.pointerEventsDisabled = !0)),
      dl &&
        !this.codeOverflowFix &&
        (this.stickyContainer.style.setProperty(gi, "hidden"),
        (this.codeOverflowFix = !0)),
      (this.scrollInteractionFixTimer = setTimeout(
        this.restoreScrollInteractions,
        sl,
      )));
  }
  restoreScrollInteractions = () => {
    (this.clearScrollInteractionTimer(),
      this.pointerEventsDisabled &&
        (this.stickyContainer.style.removeProperty("pointer-events"),
        (this.pointerEventsDisabled = !1)),
      this.codeOverflowFix &&
        (this.stickyContainer.style.setProperty(gi, "auto"),
        (this.codeOverflowFix = !1)));
  };
  syncLayout() {
    const { gap: t, paddingBottom: n, paddingTop: i } = this.getLayout();
    ((this.stickyContainer.style.gap = `${t}px`),
      this.container?.style.setProperty("margin-top", `${i}px`),
      this.container?.style.setProperty("margin-bottom", `${n}px`));
  }
  reconcileHeaderFooterHosts() {
    const t = this.reconcileHost("header"),
      n = this.reconcileHost("footer");
    return t || n;
  }
  reconcileHost(t) {
    const { root: n, container: i } = this;
    if (n == null || i == null) return !1;
    const r = t === "header" ? this.header : this.footer,
      o =
        t === "header"
          ? this.options.renderCodeViewHeader
          : this.options.renderCodeViewFooter;
    if (o == null)
      return (
        r.element == null ||
          (this.resizeObserver?.unobserve(r.element),
          r.element.remove(),
          (r.element = void 0),
          (r.render = void 0),
          this.setHostHeight(r, 0)),
        !1
      );
    if (r.element != null && o === r.render) return !1;
    const s = r.element ?? il(t, i, this.resizeObserver);
    r.element = s;
    const a = o();
    return (
      a != null
        ? s.replaceChildren(a)
        : !this.isContainerManaged &&
          s.children.length > 0 &&
          (s.textContent = ""),
      (r.render = o),
      !0
    );
  }
  setHostHeight(t, n) {
    t.height !== n && ((t.height = n), (this.scrollDirty = !0));
  }
  measureMountedHosts() {
    (this.header.element != null &&
      this.setHostHeight(
        this.header,
        this.header.element.getBoundingClientRect().height,
      ),
      this.footer.element != null &&
        this.setHostHeight(
          this.footer,
          this.footer.element.getBoundingClientRect().height,
        ));
  }
  setup(t) {
    if (this.root != null) throw new Error("CodeView.setup: already setup");
    (this.workerManager?.subscribeToThemeChanges(this),
      (this.root = t),
      (this.root.style.overflowAnchor = "none"),
      this.root.hasAttribute("tabindex") || (this.root.tabIndex = -1),
      (this.container ??= document.createElement("div")),
      (this.container.style.contain = "layout style"),
      this.syncLayout(),
      this.container.appendChild(this.stickyOffset),
      this.container.appendChild(this.stickyContainer),
      this.root.appendChild(this.container),
      (this.scrollDirty = !0),
      (this.heightDirty = !0),
      (this.resizeObserver = new ResizeObserver(this.handleResize)),
      this.resizeObserver.observe(this.stickyContainer),
      this.root.addEventListener("scroll", this.handleScroll, { passive: !0 }),
      this.root.addEventListener("wheel", this.clearPendingScroll, {
        passive: !0,
      }),
      this.root.addEventListener("touchstart", this.clearPendingScroll, {
        passive: !0,
      }),
      this.root.addEventListener("pointerdown", this.clearPendingScroll, {
        passive: !0,
      }),
      this.root.addEventListener("keydown", this.clearPendingScroll, {
        passive: !0,
      }),
      this.resizeObserver.observe(this.root),
      this.render(!0),
      (window.__INSTANCE = this),
      (window.__TOGGLE = () => {
        be.__STOP
          ? ((be.__STOP = !1),
            this.scrollTo({
              type: "position",
              position: be.__lastScrollPosition,
              behavior: "instant",
            }))
          : ((be.__lastScrollPosition = this.getScrollTop()), (be.__STOP = !0));
      }));
  }
  reset() {
    (Ne(this.computeRenderRangeAndEmit),
      this.clearReadySubscription(),
      this.restoreScrollInteractions(),
      this.cleanAllRenderedItems());
    const t = [...this.itemEditors.values()].map((r) => {
      const o = this.idToItem.get(r.state.id);
      return (
        o != null && (r.state.removedItem = o.item),
        { record: r, instance: o?.instance }
      );
    });
    (this.itemEditors.clear(),
      this.attachedEditors.clear(),
      (this.selectedLines = null),
      (this.items.length = 0),
      this.idToItem.clear(),
      this.instanceToItem.clear());
    let n = !1,
      i;
    for (const { record: r, instance: o } of t)
      try {
        (r.editor.cleanUp("discard"), o?.cleanUp());
      } catch (s) {
        n || ((n = !0), (i = s));
      }
    if (
      ((this.layoutDirtyIndex = void 0),
      (this.pendingLayoutReset = void 0),
      (this.stickyContainer.textContent = ""),
      (this.stickyOffset.style.height = ""),
      this.container?.style.removeProperty("height"),
      (this.containerHeight = -1),
      (this.windowSpecs = { top: 0, bottom: 0 }),
      (this.pendingLayoutAnchor = void 0),
      (this.shouldFixContainerFocus = !1),
      (this.height = 0),
      (this.scrollTop = 0),
      (this.scrollPageOffset = 0),
      (this.scrollHeight = 0),
      (this.scrollDirty = !0),
      (this.heightDirty = !0),
      this.resetRenderState(),
      this.isContainerManaged || this.flushSlotCoordinator(),
      n)
    )
      throw i;
  }
  cleanUp() {
    let t = !1,
      n;
    try {
      this.reset();
    } catch (i) {
      ((t = !0), (n = i));
    }
    if (
      (this.clearElementPool(),
      this.restoreScrollInteractions(),
      this.workerManager?.unsubscribeToThemeChanges(this),
      this.resizeObserver?.disconnect(),
      (this.resizeObserver = void 0),
      this.root?.removeEventListener("scroll", this.handleScroll),
      this.root?.removeEventListener("wheel", this.clearPendingScroll),
      this.root?.removeEventListener("touchstart", this.clearPendingScroll),
      this.root?.removeEventListener("pointerdown", this.clearPendingScroll),
      this.root?.removeEventListener("keydown", this.clearPendingScroll),
      this.root?.style.removeProperty("overflow-anchor"),
      this.container?.remove(),
      this.stickyOffset.remove(),
      this.stickyContainer.remove(),
      (this.stickyContainer.textContent = ""),
      this.header.element?.remove(),
      (this.header.element = void 0),
      (this.header.render = void 0),
      (this.header.height = 0),
      this.footer.element?.remove(),
      (this.footer.element = void 0),
      (this.footer.render = void 0),
      (this.footer.height = 0),
      (this.root = void 0),
      (this.container = void 0),
      t)
    )
      throw n;
  }
  cleanAllRenderedItems() {
    if (this.renderState.firstIndex !== -1)
      for (
        let t = this.renderState.firstIndex;
        t <= this.renderState.lastIndex;
        t++
      ) {
        const n = this.items[t];
        if (n == null)
          throw new Error(
            `CodeView.cleanAllRenderedItems: Item does not exist at index: ${t}`,
          );
        this.releaseRenderedItem(n);
      }
  }
  primeScrollTarget(t) {
    if (t.type === "position") return;
    const n = this.idToItem.get(t.id);
    n?.instance.primeHighlightCache();
  }
  getElementPoolLimit() {
    const t = this.getHeight() + this.config.overscrollSize * 2,
      { diffHeaderHeight: n } = this.itemMetricsCache;
    return (
      Math.max(8, Math.ceil(t / Math.max(n, 10)) + 1) *
      (this.isContainerManaged ? 2 : 1)
    );
  }
  acquireElement() {
    this.promotePendingPooledElements();
    let t = this.elementPool.pop();
    for (; t != null && !this.isElementPoolGenerationCurrent(t);)
      t = this.elementPool.pop();
    return (
      (t ??= document.createElement(Ui)),
      this.markElementPoolGenerationCurrent(t),
      t
    );
  }
  releaseRenderedItem(t) {
    const { element: n } = t;
    (n != null &&
      this.renderedItemOwnsFocus(n) &&
      (this.shouldFixContainerFocus = !0),
      t.instance.cleanUp(!0),
      this.attachedEditors.delete(t.item.id),
      (t.element = void 0),
      n != null &&
        (n.remove(), this.cleanElement(n), this.queueElementForPool(n)));
  }
  renderedItemOwnsFocus(t) {
    const { activeElement: n } = document;
    return n === t || t.contains(n) || t.shadowRoot?.activeElement != null;
  }
  fixContainerFocus() {
    this.shouldFixContainerFocus &&
      ((this.shouldFixContainerFocus = !1),
      this.root?.focus({ preventScroll: !0 }));
  }
  cleanElement(t) {
    const { shadowRoot: n } = t;
    if (n != null) for (const i of Array.from(n.children)) pl(i) || i.remove();
    this.isContainerManaged || t.replaceChildren();
  }
  queueElementForPool(t) {
    const n = this.getElementPoolLimit();
    !this.isElementPoolGenerationCurrent(t) ||
      this.getElementPoolSize() >= n ||
      (this.isElementClean(t)
        ? this.elementPool.push(t)
        : this.pendingElementPool.push(t));
  }
  promotePendingPooledElements() {
    if (this.pendingElementPool.length === 0) return;
    const { pendingElementPool: t } = this;
    this.pendingElementPool = [];
    const n = this.getElementPoolLimit();
    for (const i of t)
      this.isElementPoolGenerationCurrent(i) &&
      this.isElementClean(i) &&
      this.elementPool.length < n
        ? this.elementPool.push(i)
        : this.isElementPoolGenerationCurrent(i) &&
          this.getElementPoolSize() < n &&
          this.pendingElementPool.push(i);
  }
  isElementClean(t) {
    return t.childNodes.length === 0;
  }
  getElementPoolSize() {
    return this.elementPool.length + this.pendingElementPool.length;
  }
  clearElementPool() {
    ((this.elementPool.length = 0), (this.pendingElementPool.length = 0));
  }
  invalidateElementPool() {
    (this.elementPoolVersion++, this.clearElementPool());
  }
  markElementPoolGenerationCurrent(t) {
    this.elementPoolTracker.set(t, this.elementPoolVersion);
  }
  isElementPoolGenerationCurrent(t) {
    return this.elementPoolTracker.get(t) === this.elementPoolVersion;
  }
  resolveEffectiveScrollBehavior(t, n) {
    return uo()
      ? "instant"
      : t.behavior !== "smooth-auto"
        ? (t.behavior ?? "instant")
        : Math.abs(n - this.getScrollTop()) <= this.getHeight() * 10
          ? "smooth"
          : "instant";
  }
  scrollTo(t) {
    if (this.root == null) return;
    const n = this.normalizeScrollTarget(t);
    if (n == null) return;
    const i = this.resolveScrollTargetTop(n);
    i != null &&
      (this.primeScrollTarget(n),
      this.resolveEffectiveScrollBehavior(n, i) === "smooth"
        ? (this.scrollAnimation ??= {
            position: this.getScrollTop(),
            velocity: 0,
            lastTimestamp: performance.now(),
          })
        : (this.scrollAnimation = void 0),
      this.suspendScrollInteractions(),
      (this.pendingLayoutAnchor = void 0),
      (this.pendingScrollTarget = n),
      this.render());
  }
  setSelectedLines(t, n) {
    this.applySelectedLines(t, n);
  }
  getSelectedLines() {
    return this.selectedLines;
  }
  clearSelectedLines(t) {
    this.applySelectedLines(null, t);
  }
  getItem(t) {
    return this.idToItem.get(t)?.item;
  }
  getEditor(t) {
    return this.itemEditors.get(t)?.editor;
  }
  updateItem(t) {
    const n = this.idToItem.get(t.id);
    return n == null
      ? (console.error(`CodeView.updateItem: unknown item id "${t.id}"`), !1)
      : this.syncItemRecord(n, t)
        ? (this.markItemLayoutDirty(n),
          (this.scrollDirty = !0),
          this.render(),
          this.syncItemEditors(),
          this.syncSelection(),
          !0)
        : !1;
  }
  updateItemId(t, n) {
    if (t === n) return !0;
    const i = this.idToItem.get(t);
    if (i == null)
      return (
        console.error(`CodeView.updateItemId: unknown item id "${t}"`),
        !1
      );
    if (this.idToItem.has(n))
      return (
        console.error(`CodeView.updateItemId: duplicate item id "${n}"`),
        !1
      );
    (this.idToItem.delete(t),
      (i.item.id = n),
      this.idToItem.set(n, i),
      this.updateItemOptionsId(i.instance.options, n),
      this.selectedLines?.id === t &&
        ((this.selectedLines = { ...this.selectedLines, id: n }),
        this.options.onSelectedLinesChange?.(this.selectedLines)));
    const r = this.itemEditors.get(t);
    return (
      r != null &&
        (this.itemEditors.delete(t),
        this.itemEditors.set(n, r),
        this.attachedEditors.delete(t) && this.attachedEditors.add(n)),
      this.renamePendingScrollTarget(t, n),
      this.renamePendingLayoutAnchor(t, n),
      this.render(),
      !0
    );
  }
  addItem(t) {
    this.addItems([t]);
  }
  addItems(t) {
    (this.appendItemsInternal(t), this.syncItemEditors(), this.syncSelection());
  }
  removeItem(t) {
    const n = this.idToItem.get(t);
    if (n == null)
      return (console.error(`CodeView.removeItem: unknown item id "${t}"`), !1);
    const i = [];
    for (const r of this.items) r !== n && i.push(r.item);
    return (this.setItems(i), !0);
  }
  setItems(t) {
    let n;
    (this.items.length === 0
      ? this.appendItemsInternal(t)
      : this.tryAppendItems(t) || (n = this.reconcileItems(t)),
      this.syncItemEditors(n),
      this.syncSelection());
  }
  appendItemsInternal(t, n = !0) {
    if (t.length === 0) return;
    const i = this.getLayout();
    let r = this.items.length === 0 ? 0 : this.scrollHeight + i.gap;
    const o = r;
    for (let s = 0; s < t.length; s++) {
      const a = t[s];
      if (a == null)
        throw new Error("CodeView.appendItemsInternal: missing input item");
      if (this.idToItem.has(a.id))
        throw new Error(`CodeView.addItem: duplicate id "${a.id}"`);
      const l = this.createItem(a, this.items.length, r);
      (this.items.push(l),
        this.idToItem.set(l.item.id, l),
        this.instanceToItem.set(l.instance, l),
        (l.height = hl(l)),
        (r += l.height + i.gap));
    }
    ((this.scrollHeight = r - i.gap),
      (this.scrollDirty = !0),
      n &&
        (this.canSkipRenderForAppend(o)
          ? this.syncContainerHeight()
          : this.render()));
  }
  canSkipRenderForAppend(t) {
    return (
      this.container != null &&
      this.renderState.firstIndex !== -1 &&
      this.pendingScrollTarget == null &&
      this.scrollAnimation == null &&
      this.layoutDirtyIndex == null &&
      t > this.windowSpecs.bottom
    );
  }
  onThemeChange() {
    this.invalidateElementPool();
  }
  setOptions(t) {
    if (t == null) return;
    this.capturePendingLayoutAnchor();
    const { options: n } = this,
      i = this.getLayout(),
      { itemMetricsCache: r } = this;
    (cl(n, t) && this.invalidateElementPool(), (this.options = t));
    const o = this.computeMetricsCache(t.itemMetrics),
      s = !qe(r, o),
      a = !qe(i, this.getLayout());
    a && this.syncLayout();
    const l = s || ul(n, t);
    if (l) {
      const h = this.pendingLayoutReset;
      this.pendingLayoutReset = {
        metrics: s ? o : h?.metrics,
        resetFileLayoutCache: !0,
        resetDiffLayoutCache: !0,
        includeEstimatedDiffHeights:
          h?.includeEstimatedDiffHeights === !0 || s || fl(n, t),
      };
    }
    ((a || l) && (this.markLayoutDirtyFromIndex(0), (this.scrollDirty = !0)),
      Ln(n, t) || this.renderOptionsRevision++,
      this.syncItemEditors());
    const d =
      n.renderCodeViewHeader !== t.renderCodeViewHeader ||
      n.renderCodeViewFooter !== t.renderCodeViewFooter;
    !this.isContainerManaged && (this.items.length > 0 || d) && this.render();
  }
  capturePendingLayoutAnchor(t = this.idToItem) {
    this.root == null ||
      this.items.length === 0 ||
      this.pendingScrollTarget != null ||
      (this.pendingLayoutAnchor = this.getScrollAnchor(this.getScrollTop(), t));
  }
  render(t = !1) {
    be.__STOP ||
      (t
        ? (Ne(this.computeRenderRangeAndEmit), this.computeRenderRangeAndEmit())
        : W(this.computeRenderRangeAndEmit));
  }
  isReady() {
    const { workerManager: t } = this;
    return t == null || t.getStats().workersFailed
      ? this.isSharedHighlighterReady()
      : t.isInitialized()
        ? (this.clearReadySubscription(), !0)
        : ((this.isReadySubscription ??= t.subscribeToStatChanges((n) => {
            (n.managerState !== "initialized" && !n.workersFailed) ||
              (this.clearReadySubscription(), this.render(!0));
          })),
          t.getStats().managerState === "waiting" &&
            t.initialize().catch(() => {}),
          !1);
  }
  clearReadySubscription() {
    this.isReadySubscription != null &&
      (this.isReadySubscription(),
      (this.isReadySubscription = void 0),
      (this.pendingHighlighterTheme = void 0));
  }
  isSharedHighlighterReady() {
    const t =
      this.workerManager?.getFileRenderOptions().theme ??
      this.options.theme ??
      U;
    return Qr() && ce(t)
      ? (this.clearReadySubscription(), !0)
      : (Ue(this.pendingHighlighterTheme, t) || this.clearReadySubscription(),
        (this.isReadySubscription ??= (() => {
          this.pendingHighlighterTheme = t;
          let n = !1;
          return (
            Zr({
              themes: ht(t),
              langs: [],
              preferredHighlighter: this.options.preferredHighlighter,
            }).then(
              () => {
                n || (this.clearReadySubscription(), this.render(!0));
              },
              (i) => {
                n || (this.clearReadySubscription(), console.error(i));
              },
            ),
            () => {
              n = !0;
            }
          );
        })()),
        !1);
  }
  instanceChanged(t, n) {
    const i = this.instanceToItem.get(t);
    if (i == null)
      throw new Error(
        "CodeView.instanceChanged: An instance has changed that is not registered",
      );
    (n && this.markItemLayoutDirty(i), this.render());
  }
  getWindowSpecs() {
    return this.windowSpecs;
  }
  getContainerElement() {
    return this.root;
  }
  getHeaderElement() {
    return this.header.element;
  }
  getFooterElement() {
    return this.footer.element;
  }
  getRenderedItems() {
    const { firstIndex: t, lastIndex: n } = this.renderState;
    if (t === -1 || n === -1 || n < t) return [];
    const i = [];
    for (let r = t; r <= n; r++) {
      const o = this.items[r];
      o?.element != null &&
        (o.type === "diff"
          ? i.push({
              id: o.item.id,
              type: "diff",
              item: o.item,
              version: o.version,
              element: o.element,
              instance: o.instance,
            })
          : i.push({
              id: o.item.id,
              type: "file",
              item: o.item,
              version: o.version,
              element: o.element,
              instance: o.instance,
            }));
    }
    return i;
  }
  setSlotCoordinator(t) {
    return t === this.slotCoordinator
      ? !1
      : ((this.slotCoordinator = t), (this.slotSnapshot = void 0), !0);
  }
  getSlotSnapshot(t) {
    return this.buildSlotSnapshot(t);
  }
  buildSlotSnapshot(t) {
    const n = ml(this.getRenderedItems(), t),
      { element: i } = this.header,
      { element: r } = this.footer;
    if (!(n == null && i == null && r == null))
      return { items: n, header: i, footer: r };
  }
  subscribeToScroll(t) {
    return (
      this.scrollListeners.add(t),
      () => {
        this.scrollListeners.delete(t);
      }
    );
  }
  getLocalTopForInstance(t) {
    const n = this.instanceToItem.get(t);
    if (n == null)
      throw new Error(
        "CodeView.getLocalTopForInstance: unknown virtualized instance",
      );
    return n.top;
  }
  getTopForItem(t) {
    const n = this.idToItem.get(t);
    if (n != null) return n.top + this.getItemTopOffset();
  }
  createItem(t, n, i) {
    const { itemMetricsCache: r } = this;
    if (t.type === "diff") {
      const s = new Wa(
        this.createDiffOptions(t.id),
        this,
        r,
        this.workerManager,
        this.isContainerManaged,
      );
      return {
        type: "diff",
        item: t,
        version: t.version,
        index: n,
        top: i,
        height: 0,
        element: void 0,
        renderedOptionsRevision: this.renderOptionsRevision,
        instance: s,
      };
    }
    const o = new bs(
      this.createFileOptions(t.id),
      this,
      r,
      this.workerManager,
      this.isContainerManaged,
    );
    return {
      type: "file",
      item: t,
      version: t.version,
      index: n,
      top: i,
      height: 0,
      element: void 0,
      renderedOptionsRevision: this.renderOptionsRevision,
      instance: o,
    };
  }
  applySelectedLines(t, n) {
    const { selectedLines: i } = this;
    (t == null && i == null) ||
      (t != null && i?.id === t.id && Pt(i.range, t.range)) ||
      (i != null &&
        i.id !== t?.id &&
        this.idToItem
          .get(i.id)
          ?.instance.setSelectedLines(null, { notify: !1 }),
      (this.selectedLines = t),
      this.idToItem
        .get(t?.id ?? "")
        ?.instance.setSelectedLines(t?.range ?? null, n));
  }
  syncSelection() {
    if (this.selectedLines == null) return;
    const t = this.idToItem.get(this.selectedLines.id);
    if (t == null) {
      ((this.selectedLines = null), this.options.onSelectedLinesChange?.(null));
      return;
    }
    t.instance.setSelectedLines(this.selectedLines.range, { notify: !1 });
  }
  isItemInEditMode(t) {
    return t.item.edit === !0 && t.item.collapsed !== !0;
  }
  attachItemEditor(t) {
    const { id: n } = t.item,
      { createEditor: i } = this.options;
    if (
      t.element == null ||
      this.attachedEditors.has(n) ||
      !this.isItemInEditMode(t)
    )
      return;
    const r = this.itemEditors.get(n);
    let o;
    try {
      if (r == null) {
        ll(i);
        const s = Ae(t.instance.options);
        if (s == null)
          throw new Error(
            "CodeView.attachItemEditor: instance options are missing their state",
          );
        const a = (d) => {
            const h = this.idToItem.get(s.id);
            h != null && this.options.onItemEditChange?.(d, h.item);
          },
          l = this.options.getEditStateKey?.(t.item);
        if (t.type === "diff") {
          const d = i("file-diff", { onChange: a }, l);
          ((o = d),
            this.itemEditors.set(n, {
              editor: d,
              state: s,
              dispose: d.edit(t.instance),
            }));
        } else {
          const d = i("file", { onChange: a }, l);
          ((o = d),
            this.itemEditors.set(n, {
              editor: d,
              state: s,
              dispose: d.edit(t.instance),
            }));
        }
      } else (t.type, (r.dispose = r.editor.edit(t.instance)));
      this.attachedEditors.add(n);
    } catch (s) {
      throw (o?.cleanUp(), this.releaseRenderedItem(t), s);
    }
  }
  syncItemEditors(t) {
    if (this.itemEditors.size === 0) return;
    const n = [];
    for (const [o, s] of this.itemEditors) {
      const a = this.idToItem.get(o),
        l = t?.get(o);
      if (l == null && a != null) {
        const { edit: d = !1, collapsed: h = !1 } = a.item;
        if (d) {
          h && this.attachedEditors.delete(o) && s.editor.cleanUp("recycle");
          continue;
        }
      } else l != null && (s.state.removedItem = l.item);
      (this.itemEditors.delete(o), this.attachedEditors.delete(o), n.push(s));
    }
    if (n.length === 0) return;
    let i = !1,
      r;
    for (const o of n) {
      try {
        o.dispose();
      } catch (a) {
        i || ((i = !0), (r = a));
      }
      const s = this.idToItem.get(o.state.id);
      s != null && this.markItemLayoutDirty(s);
    }
    if ((this.render(), i)) throw r;
  }
  requireItemFromState(t) {
    const n = this.idToItem.get(t.id)?.item ?? t.removedItem;
    if (n == null)
      throw new Error(`CodeView.onItemEditComplete: unknown item "${t.id}"`);
    return n;
  }
  completeFileItemEdit(t, n) {
    const i = this.requireItemFromState(t);
    if (i.type !== "file")
      throw new Error(
        `CodeView.onItemEditComplete: item "${t.id}" is not a file item`,
      );
    const r = {
      ...i,
      file: n.file,
      annotations: n.lineAnnotations,
      edit: !1,
      version: (i.version ?? 0) + 1,
    };
    return (this.options.onItemEditComplete?.(n, i, r) ?? "reject") === "reject"
      ? "reject"
      : (this.idToItem.has(i.id) && this.updateItem(r), "accept");
  }
  completeDiffItemEdit(t, n) {
    const i = this.requireItemFromState(t);
    if (i.type !== "diff")
      throw new Error(
        `CodeView.onItemEditComplete: item "${t.id}" is not a diff item`,
      );
    const r = {
      ...i,
      fileDiff: n.fileDiff,
      annotations: n.lineAnnotations,
      edit: !1,
      version: (i.version ?? 0) + 1,
    };
    return (this.options.onItemEditComplete?.(n, i, r) ?? "reject") === "reject"
      ? "reject"
      : (this.idToItem.has(i.id) && this.updateItem(r), "accept");
  }
  renamePendingScrollTarget(t, n) {
    const { pendingScrollTarget: i } = this;
    i == null ||
      i.type === "position" ||
      i.id !== t ||
      (this.pendingScrollTarget = { ...i, id: n });
  }
  renamePendingLayoutAnchor(t, n) {
    this.pendingLayoutAnchor?.id === t && (this.pendingLayoutAnchor.id = n);
  }
  createFileOptionsPrototype() {
    const t = {};
    for (const n of ol) ae(t, n, () => this.options[n]);
    (ae(t, "stickyHeader", () => this.options.stickyHeaders),
      ae(t, "collapsed", (n) => {
        const i = Ae(n);
        if (i != null) return this.getItemOptions(i, "file")?.item.collapsed;
      }),
      ae(t, "onEditComplete", (n) => {
        if (this.options.onItemEditComplete == null) return;
        const i = Ae(n);
        if (i != null) return (r) => this.completeFileItemEdit(i, r);
      }));
    for (const n of ci) this.defineItemSharedCallback(t, "file", n);
    for (const n of ui) this.defineItemSelectionCallback(t, "file", n);
    return t;
  }
  createDiffOptionsPrototype() {
    const t = {};
    for (const n of rl) ae(t, n, () => this.options[n]);
    (ae(t, "stickyHeader", () => this.options.stickyHeaders),
      ae(t, "hunkSeparators", () => this.options.hunkSeparators),
      ae(t, "collapsed", (n) => {
        const i = Ae(n);
        if (i != null) return this.getItemOptions(i, "diff")?.item.collapsed;
      }),
      ae(t, "onEditComplete", (n) => {
        if (this.options.onItemEditComplete == null) return;
        const i = Ae(n);
        if (i != null) return (r) => this.completeDiffItemEdit(i, r);
      }));
    for (const n of ci) this.defineItemSharedCallback(t, "diff", n);
    for (const n of ui) this.defineItemSelectionCallback(t, "diff", n);
    return t;
  }
  createFileOptions(t) {
    const n = Object.create(this.fileOptionsPrototype);
    return (fi(n, { id: t }), n);
  }
  createDiffOptions(t) {
    const n = Object.create(this.diffOptionsPrototype);
    return (fi(n, { id: t }), n);
  }
  updateItemOptionsId(t, n) {
    const i = Ae(t);
    if (i == null)
      throw new Error("CodeView.updateItemOptionsId: No valid state");
    i.id = n;
  }
  getItemOptions(t, n) {
    const i = this.idToItem.get(t.id);
    if (!(i == null || i.type !== n)) return i;
  }
  defineItemSharedCallback(t, n, i) {
    ae(t, i, (r) => {
      if (this.options[i] == null) return;
      const o = Ae(r);
      if (o == null) return;
      const s = (o.callbackCache ??= {});
      let a = s[i];
      return (
        a == null &&
          ((a = (...l) => {
            const d = this.getItemOptions(o, n);
            if (d == null) return;
            const h = this.options[i];
            return h?.(...l, d);
          }),
          (s[i] = a)),
        a
      );
    });
  }
  defineItemSelectionCallback(t, n, i) {
    ae(t, i, (r) => {
      const o = Ae(r);
      if (o == null) return;
      const s = (o.callbackCache ??= {});
      let a = s[i];
      return (
        a == null &&
          ((a = (l) => {
            const d = this.getItemOptions(o, n);
            if (d == null) return;
            const h = l == null ? null : { id: d.item.id, range: l };
            (this.options.controlledSelection !== !0 &&
              (l != null || this.selectedLines?.id === d.item.id) &&
              this.applySelectedLines(h, { notify: !1 }),
              this.options.onSelectedLinesChange?.(h));
            const c = this.options[i];
            return c?.(l, d);
          }),
          (s[i] = a)),
        a
      );
    });
  }
  markLayoutDirtyFromIndex(t) {
    this.layoutDirtyIndex = Math.min(this.layoutDirtyIndex ?? t, t);
  }
  markItemLayoutDirty(t) {
    if (this.items[t.index] !== t)
      throw new Error(
        `CodeView.markItemLayoutDirty: unknown item id "${t.item.id}"`,
      );
    this.markLayoutDirtyFromIndex(t.index);
  }
  tryAppendItems(t) {
    if (t.length <= this.items.length) return !1;
    for (let n = 0; n < this.items.length; n++) {
      const i = this.items[n];
      if (i == null)
        throw new Error("CodeView.tryAppendItems: missing existing item");
      const r = t[n];
      if (r == null || i.item.id !== r.id || i.type !== r.type) return !1;
    }
    for (let n = 0; n < this.items.length; n++) {
      const i = this.items[n];
      if (i == null)
        throw new Error("CodeView.tryAppendItems: missing existing item");
      const r = t[n];
      if (r == null)
        throw new Error(
          "CodeView.tryAppendItems: append candidate missing prefix item",
        );
      this.syncItemRecord(i, r) && this.markLayoutDirtyFromIndex(n);
    }
    return (
      this.appendItemsInternal(t.slice(this.items.length), !1),
      (this.scrollDirty = !0),
      this.render(),
      !0
    );
  }
  reconcileItems(t) {
    const { items: n, idToItem: i } = this,
      r = new Set(n),
      o = [],
      s = new Map(),
      a = new Map(),
      l = new Map();
    let d;
    for (let h = 0; h < t.length; h++) {
      const c = t[h];
      if (c == null)
        throw new Error("CodeView.reconcileItems: missing input item");
      if (s.has(c.id))
        throw new Error(`CodeView.setItems: duplicate id "${c.id}"`);
      const u = i.get(c.id),
        f = u != null && u.type === c.type ? u : this.createItem(c, h, 0);
      ((f.index = h),
        u != null && u.type === c.type
          ? (r.delete(u),
            this.syncItemRecord(f, c) && (d = Math.min(d ?? h, h)))
          : (d = Math.min(d ?? h, h)),
        n[h] !== f && (d = Math.min(d ?? h, h)),
        o.push(f),
        s.set(c.id, f),
        a.set(f.instance, f));
    }
    if (d == null) {
      if (r.size === 0) return;
      d = Math.max(o.length - 1, 0);
    }
    this.capturePendingLayoutAnchor(s);
    for (let h = 0; h < n.length; h++) {
      const c = n[h];
      if (c == null || !r.has(c)) continue;
      (l.set(c.item.id, c), this.releaseRenderedItem(c));
      const u = Math.max(o.length - 1, 0);
      d = Math.min(d ?? u, u);
    }
    return (
      (this.items = o),
      (this.idToItem = s),
      (this.instanceToItem = a),
      this.renderState.firstIndex >= o.length
        ? this.resetRenderState()
        : this.renderState.lastIndex >= o.length &&
          (this.renderState.lastIndex = o.length - 1),
      this.markLayoutDirtyFromIndex(d),
      (this.scrollDirty = !0),
      this.render(),
      l.size > 0 ? l : void 0
    );
  }
  syncItemRecord(t, n) {
    if (t.type !== n.type)
      throw new Error(
        `CodeView.syncItemRecord: type mismatch for id "${n.id}"`,
      );
    return t.version === n.version
      ? !1
      : ((t.item = n),
        (t.version = n.version),
        (t.renderedOptionsRevision = -1),
        !0);
  }
  getMaxScrollTopForHeight(t) {
    const { paddingBottom: n, paddingTop: i } = this.getLayout();
    return Math.max(
      i + this.header.height + t + this.footer.height + n - this.getHeight(),
      0,
    );
  }
  getMaxScrollTop() {
    return this.getMaxScrollTopForHeight(this.getScrollHeight());
  }
  shouldRebaseScroll() {
    return this.getMaxScrollTop() > mi;
  }
  getPagedScrollHeight() {
    return this.shouldRebaseScroll()
      ? Math.min(this.getScrollHeight(), Fn)
      : this.getScrollHeight();
  }
  getMaxPagedScrollTop() {
    return this.getMaxScrollTopForHeight(this.getPagedScrollHeight());
  }
  clampPagedScrollTop(t) {
    const n = this.getMaxPagedScrollTop();
    return Math.max(0, Math.min(t, n));
  }
  clampScrollTop(t) {
    const n = this.getMaxScrollTop();
    return Math.max(0, Math.min(t, n));
  }
  getMaxScrollPageOffset() {
    return Math.max(this.getMaxScrollTop() - this.getMaxPagedScrollTop(), 0);
  }
  clampScrollPageOffset(t) {
    const n = this.getMaxScrollPageOffset();
    return Math.max(0, Math.min(t, n));
  }
  resolveScrollPageWindow(t, n) {
    let i = de(this.clampPagedScrollTop(n)),
      r = this.clampScrollPageOffset(t - i);
    return (
      (i = de(this.clampPagedScrollTop(t - r))),
      (r = this.clampScrollPageOffset(t - i)),
      { pagedScrollTop: i, scrollPageOffset: r }
    );
  }
  resolvePagedScrollPosition(t) {
    if (!this.shouldRebaseScroll())
      return {
        pagedScrollTop: this.clampPagedScrollTop(t),
        scrollPageOffset: 0,
      };
    const n = this.clampScrollPageOffset(this.scrollPageOffset),
      i = t - n,
      r = this.getMaxPagedScrollTop(),
      o = this.getMaxScrollPageOffset(),
      s = i > mi && n < o,
      a = i < Ar && n > 0;
    return i < 0 || i > r || s || a
      ? this.resolveScrollPageWindow(t, a ? Math.min(al, r) : Ir)
      : {
          pagedScrollTop: de(this.clampPagedScrollTop(i)),
          scrollPageOffset: n,
        };
  }
  needsScrollPageUpdate(t) {
    const n = de(this.clampScrollTop(t)),
      { scrollPageOffset: i } = this.resolvePagedScrollPosition(n);
    return i !== this.scrollPageOffset;
  }
  getPagedLayoutTop(t) {
    return this.shouldRebaseScroll()
      ? Math.max(t - this.scrollPageOffset, 0)
      : t;
  }
  getStickyHeaderOffset() {
    return this.options.stickyHeaders === !0 &&
      this.options.disableFileHeader !== !0
      ? this.itemMetricsCache.diffHeaderHeight
      : 0;
  }
  getScrollTargetRect(t) {
    const n = this.idToItem.get(t.id);
    if (n == null) {
      console.warn(`CodeView.scrollTo: unknown item id "${t.id}"`);
      return;
    }
    if (t.type === "item") return { top: n.top, height: n.height };
    if (t.type === "range") {
      const r = this.getRangeScrollPosition(n, t);
      if (r == null) {
        console.warn(
          `CodeView.scrollTo: unable to resolve range ${Si(t.range)} for item "${t.id}"`,
        );
        return;
      }
      return { top: n.top + r.top, height: r.height };
    }
    const i = this.getLineScrollPosition(n, t);
    if (i == null) {
      console.warn(
        `CodeView.scrollTo: unable to resolve line ${t.lineNumber} for item "${t.id}"`,
      );
      return;
    }
    return { top: n.top + i.top, height: i.height };
  }
  normalizeScrollTarget(t) {
    if (t.type === "position" || t.align !== "nearest") return t;
    const n = this.getScrollTargetRect(t);
    if (n == null) return;
    const i = t.offset ?? 0,
      r = this.getItemTopOffset() + n.top,
      o = r + n.height,
      s = this.getScrollTop(),
      a =
        s +
        (t.type === "line" || t.type === "range"
          ? this.getStickyHeaderOffset()
          : 0),
      l = s + this.getHeight();
    if (!(r - i <= a && o + i >= l)) {
      if (r - i < a) return { ...t, align: "start" };
      if (o + i > l) return { ...t, align: "end" };
    }
  }
  resolveScrollTargetTop(t) {
    if (t.type === "position") {
      const r = this.clampScrollTop(t.position);
      return r !== t.position
        ? r
        : this.clampScrollTop(t.position - this.getStickyHeaderOffset());
    }
    const n = this.idToItem.get(t.id);
    if (n == null) {
      console.warn(`CodeView.scrollTo: unknown item id "${t.id}"`);
      return;
    }
    if (t.type === "item")
      return this.clampScrollTop(
        this.resolveAlignedScrollPosition(n.top, n.height, t.align, t.offset),
      );
    if (t.type === "range") {
      const r = this.getRangeScrollPosition(n, t);
      if (r == null) {
        console.warn(
          `CodeView.scrollTo: unable to resolve range ${Si(t.range)} for item "${t.id}"`,
        );
        return;
      }
      return this.clampScrollTop(
        this.resolveAlignedScrollPosition(
          n.top + r.top,
          r.height,
          t.align,
          t.offset,
          this.getStickyHeaderOffset(),
        ),
      );
    }
    const i = this.getLineScrollPosition(n, t);
    if (i == null) {
      console.warn(
        `CodeView.scrollTo: unable to resolve line ${t.lineNumber} for item "${t.id}"`,
      );
      return;
    }
    return this.clampScrollTop(
      this.resolveAlignedScrollPosition(
        n.top + i.top,
        i.height,
        t.align,
        t.offset,
        this.getStickyHeaderOffset(),
      ),
    );
  }
  resolveAlignedScrollPosition(t, n, i, r = 0, o = 0) {
    t += this.getItemTopOffset();
    const s = this.getHeight();
    return i === "center" && n + r < s
      ? t - (s - n) / 2 + r
      : i === "end"
        ? t - (s - n) + r
        : t - o - r;
  }
  getLineScrollPosition(t, n) {
    return t.type === "diff"
      ? t.instance.getLinePosition(n.lineNumber, n.side)
      : t.instance.getLinePosition(n.lineNumber);
  }
  getRangeScrollPosition(t, n) {
    const { range: i } = n,
      r = this.getLineScrollPosition(t, {
        type: "line",
        id: n.id,
        lineNumber: i.start,
        side: i.side,
      }),
      o = this.getLineScrollPosition(t, {
        type: "line",
        id: n.id,
        lineNumber: i.end,
        side: i.endSide ?? i.side,
      });
    if (r == null || o == null) return;
    const s = r.top,
      a = s + r.height,
      l = o.top,
      d = l + o.height,
      h = Math.min(s, l);
    return { top: h, height: Math.max(a, d) - h };
  }
  computeTargetScrollTopForFrame(t, n) {
    if (this.pendingScrollTarget == null) return t;
    const i = this.resolveScrollTargetTop(this.pendingScrollTarget);
    if (i == null) return t;
    const { scrollAnimation: r } = this;
    return r == null ? i : this.computeSpringStep(r, i, n).position;
  }
  computeSpringStep(t, n, i) {
    const r = Math.max(0, i - t.lastTimestamp),
      { omega: o } = this.getSmoothScrollSettings(),
      s = Math.exp(-o * r),
      a = t.position - n,
      l = t.velocity + o * a;
    return {
      position: n + (a + l * r) * s,
      velocity: (l * (1 - o * r) - o * a) * s,
    };
  }
  advanceScrollAnimation(t, n) {
    if (this.pendingScrollTarget == null) return;
    const i = this.resolveScrollTargetTop(this.pendingScrollTarget);
    if (i == null) {
      ((this.pendingScrollTarget = void 0), (this.scrollAnimation = void 0));
      return;
    }
    const r = this.scrollAnimation;
    if (r == null) return i;
    r.position += n;
    const { position: o, velocity: s } = this.computeSpringStep(r, i, t);
    ((r.lastTimestamp = t), (r.position = o), (r.velocity = s));
    const { positionEpsilon: a, velocityEpsilon: l } =
      this.getSmoothScrollSettings();
    return Math.abs(i - o) <= a && Math.abs(s) <= l
      ? ((r.position = i), (r.velocity = 0), (this.scrollAnimation = void 0), i)
      : r.position;
  }
  computeRenderRangeAndEmit = (t = performance.now()) => {
    if (be.__STOP || this.container == null || !this.isReady()) return;
    const n = this.getHeight(),
      i = this.getScrollTop();
    let r = i,
      o = this.pendingLayoutAnchor != null,
      s = this.getScrollAnchor(r);
    if (
      (this.layoutDirtyIndex != null &&
        (this.recomputeLayout(this.layoutDirtyIndex, this.pendingLayoutReset),
        (this.layoutDirtyIndex = void 0),
        (this.pendingLayoutReset = void 0),
        (o = !0)),
      o && s != null)
    ) {
      const C = this.resolveAnchoredScrollTop(s);
      if (C != null) {
        const x = C - r;
        ((r = C),
          this.scrollAnimation != null && (this.scrollAnimation.position += x));
      }
    }
    o && ((r = this.clampScrollTop(r)), this.syncContainerHeight());
    const a = this.computeTargetScrollTopForFrame(r, t),
      l =
        !o &&
        (this.renderState.scrollTop === -1 ||
          Math.abs(a - this.renderState.scrollTop) >
            n + this.config.overscrollSize * 2);
    (l && (s = void 0),
      (this.windowSpecs = rn({
        scrollTop: a - this.header.height,
        height: n,
        scrollHeight: this.getScrollHeight(),
        fitPerfectly: l,
        fitPerfectlyOverscroll: this.getFitPerfectlyOverscroll(),
        overscrollSize: this.config.overscrollSize,
      })));
    let d = i;
    ((this.pendingScrollTarget != null && a !== d) ||
      this.needsScrollPageUpdate(a)) &&
      (this.applyScrollFix(a, d, this.windowSpecs), (d = a));
    const { top: h, bottom: c } = this.windowSpecs,
      { firstIndex: u, lastIndex: f } = this.renderState;
    if (u >= 0)
      for (let C = u; C <= f; C++) {
        const x = this.items[C];
        if (x == null)
          throw new Error(
            `CodeView.computeRenderRangeAndEmit: No item at index: ${C}`,
          );
        (x.top > h - x.height && x.top <= c) || this.releaseRenderedItem(x);
      }
    const p = this.reconcileHeaderFooterHosts();
    let g;
    const m = new Set(),
      y = this.findFirstVisibleIndex(h),
      S = this.findLastVisibleIndex(c);
    for (let C = y; C <= S; C++) {
      const x = this.items[C];
      if (x == null)
        throw new Error("CodeView.computeRenderRangeAndEmit: missing item");
      const { instance: A } = x;
      (x.element == null
        ? ((x.element = this.acquireElement()),
          bi(this.stickyContainer, x.element, g),
          A.virtualizedSetup(),
          Ci(x, x.element) &&
            ((x.renderedOptionsRevision = this.renderOptionsRevision),
            m.add(x)),
          (g = x.element))
        : (bi(this.stickyContainer, x.element, g),
          Ci(
            x,
            void 0,
            x.renderedOptionsRevision !== this.renderOptionsRevision,
          ) &&
            ((x.renderedOptionsRevision = this.renderOptionsRevision),
            m.add(x)),
          (g = x.element)),
        x.item.edit === !0 && this.attachItemEditor(x));
    }
    ((this.renderState.firstIndex = y <= S ? y : -1),
      (this.renderState.lastIndex = S),
      this.flushSlotCoordinator(),
      this.flushManagers(m),
      p && this.measureMountedHosts(),
      this.reconcileRenderedItems(m),
      this.syncContainerHeight(),
      this.updateStickyPositioning());
    const b = s != null ? this.resolveAnchoredScrollTop(s) : void 0;
    s === this.pendingLayoutAnchor && (this.pendingLayoutAnchor = void 0);
    const L = b != null ? b - r : 0;
    let v = a,
      k = !1;
    if (this.pendingScrollTarget != null) {
      const C = this.advanceScrollAnimation(t, L);
      C != null ? ((v = C), (k = !0)) : (v = r);
    } else v = b ?? a;
    (v !== d && (this.applyScrollFix(v, d, this.windowSpecs), (d = v)),
      k &&
        this.pendingScrollTarget != null &&
        this.isPendingTargetSettled(this.pendingScrollTarget) &&
        ((this.pendingScrollTarget = void 0), (this.scrollAnimation = void 0)),
      (this.renderState.scrollTop = de(d)),
      this.updateStickyPositioning(),
      this.validateStickyContainerHeight(),
      this.fixContainerFocus(),
      (l || this.scrollAnimation != null) && this.render());
  };
  flushManagers(t) {
    for (const n of t) n.instance.flushManagers();
  }
  syncContainerHeight() {
    const t = this.getPagedScrollHeight();
    this.container == null ||
      this.containerHeight === t ||
      ((this.container.style.height = `${t}px`), (this.containerHeight = t));
  }
  getStickyBounds(t) {
    const { firstIndex: n, lastIndex: i } =
      t != null
        ? {
            firstIndex: this.findFirstVisibleIndex(t.top),
            lastIndex: this.findLastVisibleIndex(t.bottom),
          }
        : this.renderState;
    if (n === -1 || i === -1 || n > i) return;
    const r = this.items[n]?.instance.getAdvancedStickySpecs(t),
      o = this.items[i]?.instance.getAdvancedStickySpecs(t);
    if (!(r == null || o == null))
      return {
        stickyTop: this.getPagedLayoutTop(Math.max(r.topOffset, 0)),
        stickyBottom: this.getPagedLayoutTop(o.topOffset + o.height),
      };
  }
  applyStickyPositioning({ stickyTop: t, stickyBottom: n }) {
    const i = this.getHeight(),
      { itemMetricsCache: r } = this,
      o = n - t;
    ((this.renderState.stickyHeight = o),
      (this.renderState.stickyTop = t),
      (this.renderState.stickyBottom = n),
      (this.stickyOffset.style.height = `${t}px`));
    const s = ((Math.random() * r.lineHeight) >> 0) * -1,
      a = -Math.max(o + s, 0) + i;
    ((this.stickyContainer.style.top = `${a}px`),
      (this.stickyContainer.style.bottom = `${a + r.diffHeaderHeight}px`));
  }
  syncPagedScrollScaffolding(t) {
    this.syncContainerHeight();
    const n = this.getStickyBounds(t);
    if (n == null) {
      this.stickyOffset.style.height = `${this.getPagedLayoutTop(t.top)}px`;
      return;
    }
    this.applyStickyPositioning(n);
  }
  reconcileRenderedItems(t) {
    const { firstIndex: n, lastIndex: i } = this.renderState;
    if (n === -1) return;
    let r = -1,
      o = !1;
    for (let s = n; s < this.items.length && !(!o && s > i); s++) {
      const a = this.items[s];
      if (a == null)
        throw new Error("CodeView.reconcileRenderedItems: Invalid item");
      (r === -1
        ? (r = a.top)
        : a.top !== r &&
          ((a.top = r), a.instance.syncVirtualizedTop(), (o = !0)),
        (t == null ? s <= i : t.has(a)) &&
          (a.instance.reconcileHeights() &&
            ((o = !0), (a.height = a.instance.getVirtualizedHeight())),
          this.validateRenderedItemHeight(a)),
        (r += a.instance.getVirtualizedHeight()),
        s < this.items.length - 1 && (r += this.getLayout().gap));
    }
    o && r != null && ((this.scrollDirty = !0), (this.scrollHeight = r));
  }
  updateStickyPositioning() {
    const t = this.getStickyBounds();
    if (t == null) {
      this.renderState.firstIndex === -1 &&
        (this.stickyOffset.style.height = "");
      return;
    }
    const { stickyTop: n, stickyBottom: i } = t;
    (i - n === this.renderState.stickyHeight &&
      n === this.renderState.stickyTop &&
      i === this.renderState.stickyBottom) ||
      this.applyStickyPositioning(t);
  }
  handleScroll = () => {
    be.__STOP ||
      (this.suspendScrollInteractions(),
      (this.scrollDirty = !0),
      this.notifyScroll(),
      this.render());
  };
  clearPendingScroll = () => {
    ((this.pendingScrollTarget = void 0),
      (this.pendingLayoutAnchor = void 0),
      (this.scrollAnimation = void 0));
  };
  handleResize = (t) => {
    let n = !1;
    for (const i of t)
      if (i.target === this.stickyContainer) {
        const r = i.borderBoxSize[0].blockSize;
        if (Math.abs(r - this.renderState.stickyHeight) >= pi) {
          const o = this.getScrollTop(),
            s = this.getScrollAnchor(o);
          (this.reconcileRenderedItems(), this.updateStickyPositioning());
          const a = s != null ? this.resolveAnchoredScrollTop(s) : void 0;
          if (a != null) {
            const l = a - o;
            (this.applyScrollFix(a, o, this.windowSpecs),
              this.scrollAnimation != null &&
                (this.scrollAnimation.position += l));
          }
          (this.pendingScrollTarget != null &&
            this.isPendingTargetSettled(this.pendingScrollTarget) &&
            ((this.pendingScrollTarget = void 0),
            (this.scrollAnimation = void 0)),
            (n = !0));
        }
      } else if (
        i.target === this.header.element ||
        i.target === this.footer.element
      ) {
        const r = i.target === this.header.element ? this.header : this.footer,
          o = i.borderBoxSize[0].blockSize;
        if (o !== r.height) {
          const s = this.getScrollTop(),
            a = this.getScrollAnchor(s);
          this.setHostHeight(r, o);
          const l = a != null ? this.resolveAnchoredScrollTop(a) : void 0;
          if (l != null) {
            const d = l - s;
            (this.applyScrollFix(l, s, this.windowSpecs),
              this.scrollAnimation != null &&
                (this.scrollAnimation.position += d));
          }
          (this.pendingScrollTarget != null &&
            this.isPendingTargetSettled(this.pendingScrollTarget) &&
            ((this.pendingScrollTarget = void 0),
            (this.scrollAnimation = void 0)),
            (n = !0));
        }
      } else ((this.scrollDirty = !0), (this.heightDirty = !0), (n = !0));
    n && this.render(!0);
  };
  getScrollAnchorViewportTop(t, n) {
    return t < n ? n + this.getStickyHeaderOffset() : n;
  }
  getScrollAnchor(t, n = this.idToItem) {
    let i;
    const { pendingLayoutAnchor: r } = this;
    if (r != null) {
      const h = this.idToItem.get(r.id);
      if (h != null && n.get(r.id) === h) return r;
      h != null && (i = h);
    }
    if (t <= 0) return;
    const {
      firstIndex: o,
      lastIndex: s,
      stickyTop: a,
      stickyBottom: l,
    } = this.renderState;
    if (o === -1 || s === -1) return;
    const d = this.getHeight();
    if (!(a === -1 || l === -1)) {
      for (let h = o; h <= s; h++) {
        const c = this.items[h];
        if (c == null) continue;
        const u = this.getItemTopOffset() + c.top;
        if (u + c.height <= t) continue;
        if (u >= t + d) break;
        if (n.get(c.item.id) !== c) {
          i ??= c;
          continue;
        }
        if (u >= t)
          return { type: "item", id: c.item.id, viewportOffset: u - t };
        const f = this.getScrollAnchorViewportTop(u, t) - u,
          p = c.instance.getNumericScrollAnchor(f);
        if (p != null) {
          const g = u + p.top;
          return {
            type: "line",
            id: c.item.id,
            lineNumber: p.lineNumber,
            side: p.side,
            viewportOffset: g - t,
          };
        }
      }
      if (i != null)
        for (let h = i.index + 1; h < this.items.length; h++) {
          const c = this.items[h];
          if (c != null && n.get(c.item.id) === c)
            return { type: "item", id: c.item.id, viewportOffset: 0 };
        }
    }
  }
  resolveAnchoredScrollTop(t) {
    const n = this.idToItem.get(t.id);
    if (n == null) return;
    const i = this.getItemTopOffset();
    if (t.type === "item") {
      const s = i + n.top;
      return this.clampScrollTop(s - t.viewportOffset);
    }
    const r =
      n.type === "diff"
        ? n.instance.getLinePosition(t.lineNumber, t.side)
        : n.instance.getLinePosition(t.lineNumber);
    if (r == null) return;
    const o = i + n.top + r.top;
    return this.clampScrollTop(o - t.viewportOffset);
  }
  applyScrollFix(t, n, i) {
    if (this.root == null) return;
    const r = de(this.clampScrollTop(t)),
      o = de(n),
      { scrollPageOffset: s } = this,
      a = de(this.clampPagedScrollTop(o - s)),
      { pagedScrollTop: l, scrollPageOffset: d } =
        this.resolvePagedScrollPosition(r),
      h = l,
      c = s !== d;
    (r === this.renderState.scrollTop && r === o && h === a && !c) ||
      (this.suspendScrollInteractions(),
      (h !== a || c) &&
        ((this.scrollPageOffset = d), this.syncPagedScrollScaffolding(i)),
      h !== a && this.root.scrollTo({ top: h, behavior: "instant" }),
      (this.renderState.scrollTop = r),
      (this.scrollTop = r),
      (this.scrollDirty = !1));
  }
  isPendingTargetSettled(t) {
    const n = this.resolveScrollTargetTop(t);
    return n == null ? !0 : de(this.getScrollTop()) === de(n);
  }
  getScrollTop() {
    if (!this.scrollDirty) return this.scrollTop;
    this.scrollDirty = !1;
    const t = this.root?.scrollTop ?? 0;
    return (
      (this.scrollTop = this.clampScrollTop(t + this.scrollPageOffset)),
      this.scrollTop
    );
  }
  getHeight() {
    return this.heightDirty
      ? ((this.heightDirty = !1),
        (this.height = this.root?.getBoundingClientRect().height ?? 0),
        this.height)
      : this.height;
  }
  getScrollHeight() {
    return this.scrollHeight;
  }
  flushSlotCoordinator() {
    if (this.slotCoordinator == null) return;
    const t = this.buildSlotSnapshot(this.slotCoordinator);
    tl(this.slotSnapshot, t) ||
      ((this.slotSnapshot = t), this.slotCoordinator.onSnapshotChange(t));
  }
  notifyScroll() {
    if (this.scrollListeners.size === 0) return;
    const t = this.getScrollTop();
    for (const n of this.scrollListeners) n(t, this);
  }
  findFirstVisibleIndex(t) {
    let n = 0,
      i = this.items.length - 1,
      r = this.items.length;
    for (; n <= i;) {
      const o = (n + i) >> 1,
        s = this.items[o];
      if (s == null)
        throw new Error("CodeView.findFirstVisibleIndex: invalid item index");
      s.top + s.height > t ? ((r = o), (i = o - 1)) : (n = o + 1);
    }
    return r;
  }
  findLastVisibleIndex(t) {
    let n = 0,
      i = this.items.length - 1,
      r = -1;
    for (; n <= i;) {
      const o = (n + i) >> 1,
        s = this.items[o];
      if (s == null)
        throw new Error("CodeView.findLastVisibleIndex: invalid item index");
      s.top <= t ? ((r = o), (n = o + 1)) : (i = o - 1);
    }
    return r;
  }
  recomputeLayout(t = 0, n) {
    if (this.items.length === 0) {
      this.scrollHeight = 0;
      return;
    }
    const i = this.getLayout();
    let r = 0;
    if (t > 0) {
      const o = this.items[t - 1];
      if (o == null)
        throw new Error("CodeView.recomputeLayout: invalid dirty index");
      r = o.top + o.height + i.gap;
    }
    for (let o = t; o < this.items.length; o++) {
      const s = this.items[o];
      if (s == null)
        throw new Error("CodeView.recomputeLayout: invalid item index");
      ((s.top = r),
        s.type === "diff"
          ? (s.height = s.instance.updateCodeViewLayout(
              s.item.fileDiff,
              r,
              n,
              s.item.annotations ?? [],
            ))
          : (s.height = s.instance.updateCodeViewLayout(
              s.item.file,
              r,
              n,
              s.item.annotations ?? [],
            )),
        (r += s.height),
        o < this.items.length - 1 && (r += i.gap));
    }
    (r !== this.scrollHeight && (this.scrollDirty = !0),
      (this.scrollHeight = r));
  }
  resetRenderState() {
    ((this.renderState.scrollTop = -1),
      (this.renderState.firstIndex = -1),
      (this.renderState.lastIndex = -1),
      (this.renderState.stickyHeight = 0),
      (this.renderState.stickyTop = -1),
      (this.renderState.stickyBottom = -1));
  }
  getFitPerfectlyOverscroll() {
    return this.getLayout().gap + this.itemMetricsCache.diffHeaderHeight;
  }
};
function hl(e) {
  return (
    e.instance.cleanUp(!0),
    e.type === "diff"
      ? e.instance.updateCodeViewLayout(
          e.item.fileDiff,
          e.top,
          void 0,
          e.item.annotations ?? [],
        )
      : e.instance.updateCodeViewLayout(
          e.item.file,
          e.top,
          void 0,
          e.item.annotations ?? [],
        )
  );
}
function cl(e, t) {
  return (
    !Ue(e.theme ?? U, t.theme ?? U) ||
    (e.themeType ?? "system") !== (t.themeType ?? "system") ||
    e.unsafeCSS !== t.unsafeCSS
  );
}
function ul(e, t) {
  return (
    (e.overflow ?? "scroll") !== (t.overflow ?? "scroll") ||
    (e.disableLineNumbers ?? !1) !== (t.disableLineNumbers ?? !1) ||
    (e.disableFileHeader ?? !1) !== (t.disableFileHeader ?? !1) ||
    e.unsafeCSS !== t.unsafeCSS ||
    (e.diffStyle ?? "split") !== (t.diffStyle ?? "split") ||
    (e.diffIndicators ?? "bars") !== (t.diffIndicators ?? "bars") ||
    (e.hunkSeparators ?? "line-info") !== (t.hunkSeparators ?? "line-info") ||
    (e.expandUnchanged ?? !1) !== (t.expandUnchanged ?? !1) ||
    (e.collapsedContextThreshold ?? 1) !== (t.collapsedContextThreshold ?? 1)
  );
}
function fl(e, t) {
  return (
    (e.disableFileHeader ?? !1) !== (t.disableFileHeader ?? !1) ||
    (e.hunkSeparators ?? "line-info") !== (t.hunkSeparators ?? "line-info") ||
    (e.expandUnchanged ?? !1) !== (t.expandUnchanged ?? !1) ||
    (e.collapsedContextThreshold ?? 1) !== (t.collapsedContextThreshold ?? 1)
  );
}
function pl(e) {
  return e instanceof SVGElement
    ? !0
    : at(e) &&
        (e.hasAttribute("data-core-css") ||
          e.hasAttribute("data-theme-css") ||
          e.hasAttribute("data-unsafe-css"));
}
function Si(e) {
  const t = vi(e.start, e.side),
    n = vi(e.end, e.endSide ?? e.side);
  return t === n ? t : `${t}-${n}`;
}
function vi(e, t) {
  return t == null ? `${e}` : `${t === "deletions" ? "D" : "A"}${e}`;
}
function Ci(e, t, n = !1) {
  return e.type === "diff"
    ? e.instance.render({
        deferManagers: !0,
        fileContainer: t,
        fileDiff: e.item.fileDiff,
        forceRender: n,
        lineAnnotations: e.item.annotations ?? [],
      })
    : e.instance.render({
        deferManagers: !0,
        fileContainer: t,
        file: e.item.file,
        forceRender: n,
        lineAnnotations: e.item.annotations ?? [],
      });
}
function bi(e, t, n) {
  if (n == null) {
    e.firstChild !== t && e.prepend(t);
    return;
  }
  n.nextSibling !== t && n.after(t);
}
function gl(e) {
  return (e.annotations?.length ?? 0) > 0;
}
function ml(
  e,
  { hasHeaderRenderers: t, hasAnnotationRenderer: n, hasGutterRenderer: i },
) {
  if (e.length === 0) return;
  if (t || i) return e;
  if (!n) return;
  const r = [];
  for (const o of e) gl(o.item) && r.push(o);
  return r.length > 0 ? r : void 0;
}
var Sl = class Hr {
    options;
    tokensStable = [];
    tokensUnstable = [];
    lastUnstableCodeChunk = "";
    lastStableGrammarState;
    constructor(t) {
      this.options = t;
    }
    async enqueue(t) {
      const n = (this.lastUnstableCodeChunk + t).split(`
`),
        i = [];
      let r = [];
      const o = this.tokensUnstable.length;
      return (
        n.forEach((s, a) => {
          const l = a === n.length - 1,
            d = this.options.highlighter.codeToTokens(s, {
              ...this.options,
              grammarState: this.lastStableGrammarState,
            }),
            h = d.tokens[0];
          (l ||
            h.push({
              content: `
`,
              offset: 0,
            }),
            l
              ? ((r = h), (this.lastUnstableCodeChunk = s))
              : ((this.lastStableGrammarState = d.grammarState), i.push(...h)));
        }),
        je(this.tokensStable, i),
        (this.tokensUnstable = r),
        { recall: o, stable: i, unstable: r }
      );
    }
    close() {
      const t = this.tokensUnstable;
      return (
        (this.tokensUnstable = []),
        (this.lastUnstableCodeChunk = ""),
        (this.lastStableGrammarState = void 0),
        { stable: t }
      );
    }
    clear() {
      ((this.tokensStable = []),
        (this.tokensUnstable = []),
        (this.lastUnstableCodeChunk = ""),
        (this.lastStableGrammarState = void 0));
    }
    clone() {
      const t = new Hr(this.options);
      return (
        (t.lastUnstableCodeChunk = this.lastUnstableCodeChunk),
        (t.tokensUnstable = this.tokensUnstable),
        (t.tokensStable = this.tokensStable),
        (t.lastStableGrammarState = this.lastStableGrammarState),
        t
      );
    }
  },
  yi = class extends TransformStream {
    tokenizer;
    options;
    constructor(e) {
      const t = new Sl(e),
        { allowRecalls: n = !1 } = e;
      (super({
        async transform(i, r) {
          const { stable: o, unstable: s, recall: a } = await t.enqueue(i);
          n && a > 0 && r.enqueue({ recall: a });
          for (const l of o) r.enqueue(l);
          if (n) for (const l of s) r.enqueue(l);
        },
        async flush(i) {
          const { stable: r } = t.close();
          if (!n) for (const o of r) i.enqueue(o);
        },
      }),
        (this.tokenizer = t),
        (this.options = e));
    }
  };
function vl(e) {
  const t = document.createElement("span");
  return ((t.style = Jr(e.htmlStyle ?? eo(e))), (t.textContent = e.content), t);
}
let Cl = -1;
var Zl = class {
  options;
  __id = `file-stream:${++Cl}`;
  highlighter;
  stream;
  abortController;
  fileContainer;
  pre;
  code;
  gutterElement;
  contentElement;
  themeCSSStyle;
  appliedThemeCSS;
  currentRowCount = 0;
  constructor(e = { theme: U }) {
    ((this.options = e),
      (this.currentLineIndex = this.options.startingLineIndex ?? 1));
  }
  cleanUp() {
    (Ne(this.render),
      this.abortController?.abort(),
      (this.abortController = void 0));
  }
  setThemeType(e) {
    (this.options.themeType ?? "system") !== e &&
      ((this.options = { ...this.options, themeType: e }),
      !(
        typeof this.options.theme == "string" ||
        this.fileContainer == null ||
        this.appliedThemeCSS == null
      ) &&
        this.applyThemeState(
          this.fileContainer,
          this.appliedThemeCSS.themeStyles,
          e,
          this.appliedThemeCSS.baseThemeType,
        ));
  }
  async initializeHighlighter() {
    return (
      (this.highlighter = await ct(xn(this.options.lang, this.options))),
      this.highlighter
    );
  }
  queuedSetupArgs;
  async setup(e, t) {
    const n = this.queuedSetupArgs != null;
    if (((this.queuedSetupArgs = [e, t]), n)) return;
    this.highlighter ??= await this.initializeHighlighter();
    const [i, r] = this.queuedSetupArgs;
    this.queuedSetupArgs = void 0;
    const o = i;
    this.setupStream(o, r, this.highlighter);
  }
  setupStream(e, t, n) {
    const {
        disableLineNumbers: i = !1,
        overflow: r = "scroll",
        theme: o = U,
        themeType: s = "system",
      } = this.options,
      a = this.getOrCreateFileContainer();
    (a.parentElement == null && t.appendChild(a),
      (this.pre ??= document.createElement("pre")),
      this.pre.parentElement == null && a.shadowRoot?.appendChild(this.pre));
    const l = typeof o == "string" ? n.getTheme(o).type : void 0,
      d = kn({ theme: o, highlighter: n });
    this.applyThemeState(a, d, s, l);
    const h = In(this.pre, {
      type: "file",
      diffIndicators: "none",
      disableBackground: !0,
      disableLineNumbers: i,
      overflow: r,
      split: !1,
      totalLines: 0,
    });
    ((h.textContent = ""),
      (this.pre = h),
      (this.code = rt({ code: this.code, pre: h })),
      (this.gutterElement = void 0),
      (this.contentElement = void 0),
      (this.currentRowCount = 0),
      (this.currentLineElement = void 0),
      (this.currentLineIndex = this.options.startingLineIndex ?? 1),
      this.abortController?.abort(),
      (this.abortController = new AbortController()));
    const {
      onStreamStart: c,
      onStreamClose: u,
      onStreamAbort: f,
    } = this.options;
    (this.stream?.cancel().catch(() => {}),
      (this.stream = e),
      this.stream
        .pipeThrough(
          typeof o == "string"
            ? new yi({
                ...this.options,
                theme: o,
                highlighter: n,
                allowRecalls: !0,
                defaultColor: !1,
                cssVariablePrefix: q("token"),
                tokenizeTimeLimit: 0,
              })
            : new yi({
                ...this.options,
                themes: o,
                highlighter: n,
                allowRecalls: !0,
                defaultColor: !1,
                cssVariablePrefix: q("token"),
                tokenizeTimeLimit: 0,
              }),
        )
        .pipeTo(
          new WritableStream({
            start(p) {
              c?.(p);
            },
            close() {
              u?.();
            },
            abort(p) {
              f?.(p);
            },
            write: this.handleWrite,
          }),
          { signal: this.abortController.signal },
        )
        .catch((p) => {
          p.name !== "AbortError" && console.error("FileStream pipe error:", p);
        }));
  }
  queuedTokens = [];
  handleWrite = (e) => {
    ("recall" in e && this.queuedTokens.length >= e.recall
      ? (this.queuedTokens.length = this.queuedTokens.length - e.recall)
      : this.queuedTokens.push(e),
      W(this.render),
      this.options.onStreamWrite?.(e));
  };
  currentLineIndex;
  currentLineElement;
  render = () => {
    this.options.onPreRender?.(this);
    const { gutter: e, content: t } = this.getOrCreateStreamColumns(),
      n = document.createDocumentFragment(),
      i = document.createDocumentFragment();
    for (const r of this.queuedTokens)
      if ("recall" in r) {
        if (this.currentLineElement == null)
          throw new Error(
            "FileStream.render: no current line element, shouldnt be possible to get here",
          );
        if (r.recall > this.currentLineElement.childNodes.length)
          throw new Error(
            "FileStream.render: Token recall exceed the current line, there's probably a bug...",
          );
        for (let o = 0; o < r.recall; o++)
          this.currentLineElement.lastChild?.remove();
      } else {
        const o = vl(r);
        if (this.currentLineElement == null) {
          const { gutterLine: s, contentLine: a } = this.createLine();
          (n.appendChild(s), i.appendChild(a));
        }
        if (
          (this.currentLineElement?.appendChild(o),
          r.content ===
            `
`)
        ) {
          this.currentLineIndex++;
          const { gutterLine: s, contentLine: a } = this.createLine();
          (n.appendChild(s), i.appendChild(a));
        }
      }
    (n.childNodes.length > 0 && e.appendChild(n),
      i.childNodes.length > 0 && t.appendChild(i),
      (this.queuedTokens.length = 0),
      this.options.onPostRender?.(this));
  };
  getOrCreateStreamColumns() {
    if (this.code == null)
      throw new Error("FileStream: expected code element to exist");
    if (this.gutterElement != null && this.contentElement != null)
      return { gutter: this.gutterElement, content: this.contentElement };
    const e = document.createElement("div");
    e.dataset.gutter = "";
    const t = document.createElement("div");
    return (
      (t.dataset.content = ""),
      this.code.appendChild(e),
      this.code.appendChild(t),
      (this.gutterElement = e),
      (this.contentElement = t),
      { gutter: e, content: t }
    );
  }
  updateRowSpan() {
    (this.gutterElement != null &&
      (this.gutterElement.style.gridRow = `span ${this.currentRowCount}`),
      this.contentElement != null &&
        (this.contentElement.style.gridRow = `span ${this.currentRowCount}`));
  }
  createLine() {
    const e = this.currentLineIndex,
      t = `${e - 1}`,
      n = document.createElement("div");
    ((n.dataset.columnNumber = `${e}`),
      (n.dataset.lineType = "context"),
      (n.dataset.lineIndex = t));
    const i = document.createElement("span");
    ((i.dataset.lineNumberContent = ""),
      (i.textContent = `${e}`),
      n.appendChild(i));
    const r = document.createElement("div");
    return (
      (r.dataset.line = `${e}`),
      (r.dataset.lineType = "context"),
      (r.dataset.lineIndex = t),
      (this.currentRowCount += 1),
      this.updateRowSpan(),
      (this.currentLineElement = r),
      { gutterLine: n, contentLine: r }
    );
  }
  getOrCreateFileContainer(e) {
    return (e != null && e === this.fileContainer) ||
      (e == null && this.fileContainer != null)
      ? this.fileContainer
      : (this.fileContainer != null &&
          e != null &&
          e !== this.fileContainer &&
          ((this.themeCSSStyle = void 0), (this.appliedThemeCSS = void 0)),
        (this.fileContainer = e ?? document.createElement("diffs-container")),
        this.fileContainer);
  }
  applyThemeState(e, t, n, i) {
    const r = e.shadowRoot ?? e.attachShadow({ mode: "open" }),
      o = i ?? n,
      s = this.options.theme ?? U,
      a = typeof s == "string" ? s : { ...s },
      l = Ke(r);
    if (
      this.themeCSSStyle?.parentNode === r &&
      this.appliedThemeCSS?.themeStyles === t &&
      this.appliedThemeCSS.themeType === o &&
      this.appliedThemeCSS.scrollbarGutter === l
    ) {
      this.appliedThemeCSS.theme = a;
      return;
    }
    ((this.themeCSSStyle = Mn({
      shadowRoot: r,
      currentNode: this.themeCSSStyle,
      themeCSS: Tn(t, o, l),
    })),
      (this.appliedThemeCSS =
        this.themeCSSStyle != null
          ? {
              theme: a,
              themeStyles: t,
              themeType: o,
              baseThemeType: i,
              scrollbarGutter: l,
            }
          : void 0));
  }
};
function Mr({ hunkIndex: e, lineIndex: t, conflictIndex: n }) {
  return `merge-conflict-action-${e}-${t}-${n}`;
}
function Dr(e, t) {
  const n = t.hunks[e.hunkIndex];
  if (n != null)
    return { hunkIndex: e.hunkIndex, lineIndex: wl(n, e.startContentIndex) };
}
function Li(e, t = 6) {
  t = Math.max(t, 1);
  const n = {
      deletionLines: [],
      additionLines: [],
      conflictStack: [],
      conflictBuilders: [],
      actions: [],
      hunks: [],
      nextConflictIndex: 0,
      splitLineCount: 0,
      unifiedLineCount: 0,
      lastHunkEnd: 0,
      activeHunk: void 0,
      maxContextLines: t,
      maxContextLines2: t * 2,
    },
    i = e.contents,
    r = i.length;
  if (r > 0) {
    let c = 0,
      u = 0,
      f = i.indexOf(
        `
`,
        c,
      );
    for (; f !== -1;)
      (xi(n, i.slice(c, f + 1), u),
        (c = f + 1),
        u++,
        (f = i.indexOf(
          `
`,
          c,
        )));
    c < r && xi(n, i.slice(c), u);
  }
  if (n.conflictStack.length > 0)
    throw new Error(
      "parseMergeConflictDiffFromFile: unfinished merge conflict marker stack",
    );
  n.activeHunk != null &&
    n.activeHunk.hunkContent.length > 0 &&
    (Pn(n, n.activeHunk, "trailing"), Or(n));
  for (let c = 0; c < n.conflictBuilders.length; c++) {
    const u = n.conflictBuilders[c];
    if (u == null || !u.completed)
      throw new Error(
        `parseMergeConflictDiffFromFile: failed to build merge conflict action ${c}`,
      );
  }
  if (
    n.hunks.length > 0 &&
    n.additionLines.length > 0 &&
    n.deletionLines.length > 0
  ) {
    const c = n.hunks[n.hunks.length - 1],
      u = Math.max(
        n.additionLines.length - ee(c.additionStart, c.additionCount),
        0,
      );
    ((n.splitLineCount += u), (n.unifiedLineCount += u));
  }
  const o = n.deletionLines.join(""),
    s = n.additionLines.join(""),
    a = Ai(e, "current", o),
    l = Ai(e, "incoming", s);
  let d = "change";
  s === "" ? (d = "deleted") : o === "" && (d = "new");
  const h = {
    name: e.name,
    prevName: void 0,
    type: d,
    hunks: n.hunks,
    splitLineCount: n.splitLineCount,
    unifiedLineCount: n.unifiedLineCount,
    isPartial: !1,
    deletionLines: n.deletionLines,
    additionLines: n.additionLines,
    cacheKey: e.cacheKey != null ? `${e.cacheKey}:merge-conflict-diff` : void 0,
  };
  return {
    fileDiff: h,
    currentFile: a,
    incomingFile: l,
    actions: n.actions,
    markerRows: zr(h, n.actions),
  };
}
function xi(e, t, n) {
  const i = e.conflictStack[e.conflictStack.length - 1];
  if (i == null) {
    if (t.length >= 7 && t.charCodeAt(0) === 60 && Ti(t) === "start") {
      Ei(e, t, n);
      return;
    }
    ki(e, t);
    return;
  }
  const r = Ti(t);
  if (r === "start") {
    Ei(e, t, n);
    return;
  }
  if (r === "base") {
    ((i.stage = "base"), (i.baseMarkerLineIndex = n), (i.markerLines.base = t));
    return;
  }
  if (r === "separator") {
    ((i.stage = "incoming"),
      (i.separatorLineIndex = n),
      (i.markerLines.separator = t));
    return;
  }
  if (r === "end") {
    const o = e.conflictStack.pop();
    if (o == null)
      throw new Error(
        "parseMergeConflictDiffFromFile: encountered end marker before start marker",
      );
    Ll(e, o, n, t);
    return;
  }
  i.stage === "current"
    ? wi(e, "deletion", t, i.conflictIndex, "current")
    : i.stage === "base"
      ? ki(e, t, i.conflictIndex)
      : wi(e, "addition", t, i.conflictIndex, "incoming");
}
function Fr(e) {
  return (
    (e.activeHunk ??= Nr(
      e.additionLines.length + 1,
      e.deletionLines.length + 1,
    )),
    e.activeHunk
  );
}
function Pr(e, t, n, i) {
  const r = e.conflictBuilders[t];
  if (r == null)
    throw new Error(
      `parseMergeConflictDiffFromFile: failed to locate conflict action ${t}`,
    );
  const o = r.action,
    s = e.hunks.length;
  if (o.hunkIndex < 0) o.hunkIndex = s;
  else if (o.hunkIndex !== s)
    throw new Error(
      `parseMergeConflictDiffFromFile: conflict ${t} spans multiple hunks and cannot be anchored`,
    );
  if (
    (o.startContentIndex < 0 && (o.startContentIndex = i),
    (o.endContentIndex = i),
    (o.endMarkerContentIndex = i),
    n === "current")
  ) {
    o.currentContentIndex ??= i;
    return;
  }
  if (n === "base") {
    o.baseContentIndex ??= i;
    return;
  }
  o.incomingContentIndex = i;
}
function bl(e, t, n, i) {
  const r = e.hunkContent,
    o = r[r.length - 1];
  return o?.type === "change"
    ? (t === "addition" ? o.additions++ : o.deletions++, r.length - 1)
    : (r.push({
        type: "change",
        additions: t === "addition" ? 1 : 0,
        deletions: t === "deletion" ? 1 : 0,
        additionLineIndex: n,
        deletionLineIndex: i,
      }),
      r.length - 1);
}
function Pn(e, t, n) {
  let i = t.contextBufferCount,
    r = t.contextBufferAdditionStart,
    o = t.contextBufferDeletionStart;
  if (n === "leading" && i > e.maxContextLines) {
    const h = i - e.maxContextLines;
    ((r += h),
      (o += h),
      (i = e.maxContextLines),
      (t.additionStart += h),
      (t.deletionStart += h),
      (t.additionLineIndex += h),
      (t.deletionLineIndex += h));
  }
  if (
    (n === "trailing" && i > e.maxContextLines && (i = e.maxContextLines),
    i === 0)
  ) {
    ((t.contextBufferCount = 0), (t.contextBufferBaseConflicts = void 0));
    return;
  }
  const s = t.hunkContent,
    a = s[s.length - 1];
  let l;
  (a?.type === "context"
    ? ((a.lines += i), (l = s.length - 1))
    : (s.push({
        type: "context",
        lines: i,
        additionLineIndex: r,
        deletionLineIndex: o,
      }),
      (l = s.length - 1)),
    (t.additionCount += i),
    (t.deletionCount += i));
  const d = t.contextBufferBaseConflicts;
  if (d != null) {
    const h = r - t.contextBufferAdditionStart;
    for (const [c, u] of d) c >= h && c < h + i && Pr(e, u, "base", l);
  }
  ((t.contextBufferCount = 0), (t.contextBufferBaseConflicts = void 0));
}
function Or(e) {
  if (e.activeHunk == null) return;
  const t = e.activeHunk;
  if (((e.activeHunk = void 0), t.hunkContent.length === 0)) return;
  let n = 0,
    i = 0;
  for (const l of t.hunkContent)
    l.type === "context"
      ? ((n += l.lines), (i += l.lines))
      : ((n += Math.max(l.additions, l.deletions)),
        (i += l.additions + l.deletions));
  const r = t.additionCount === 0 ? t.additionStart - 1 : t.additionStart,
    o = t.deletionCount === 0 ? t.deletionStart - 1 : t.deletionStart,
    s = Math.max(xe(r, t.additionCount) - e.lastHunkEnd, 0),
    a = {
      collapsedBefore: s,
      additionStart: r,
      additionCount: t.additionCount,
      additionLines: t.additionLines,
      additionLineIndex: t.additionLineIndex,
      deletionStart: o,
      deletionCount: t.deletionCount,
      deletionLines: t.deletionLines,
      deletionLineIndex: t.deletionLineIndex,
      hunkContent: t.hunkContent,
      hunkContext: void 0,
      hunkSpecs: `@@ -${Ri(o, t.deletionCount)} +${Ri(r, t.additionCount)} @@
`,
      splitLineStart: e.splitLineCount + s,
      splitLineCount: n,
      unifiedLineStart: e.unifiedLineCount + s,
      unifiedLineCount: i,
      noEOFCRAdditions: !1,
      noEOFCRDeletions: !1,
    };
  (e.hunks.push(a),
    (e.splitLineCount += s + n),
    (e.unifiedLineCount += s + i),
    (e.lastHunkEnd = ee(r, t.additionCount)));
}
function yl(e) {
  if (e.activeHunk == null) return;
  const t = e.activeHunk,
    n = t.contextBufferCount,
    i = n - e.maxContextLines2,
    r = t.contextBufferAdditionStart + n - e.maxContextLines,
    o = t.contextBufferDeletionStart + n - e.maxContextLines;
  let s;
  if (t.contextBufferBaseConflicts != null) {
    const d = n - e.maxContextLines;
    for (const [h, c] of t.contextBufferBaseConflicts)
      h >= d && ((s ??= new Map()), s.set(h - d, c));
  }
  Pn(e, t, "trailing");
  const a = t.additionCount,
    l = t.deletionCount;
  (Or(e),
    (e.activeHunk = Nr(t.additionStart + a + i, t.deletionStart + l + i)),
    (e.activeHunk.contextBufferAdditionStart = r),
    (e.activeHunk.contextBufferDeletionStart = o),
    (e.activeHunk.contextBufferCount = e.maxContextLines),
    (e.activeHunk.contextBufferBaseConflicts = s));
}
function ki(e, t, n = -1) {
  const i = Fr(e);
  (i.contextBufferCount === 0 &&
    ((i.contextBufferAdditionStart = e.additionLines.length),
    (i.contextBufferDeletionStart = e.deletionLines.length)),
    e.additionLines.push(t),
    e.deletionLines.push(t),
    n >= 0 &&
      ((i.contextBufferBaseConflicts ??= new Map()),
      i.contextBufferBaseConflicts.set(i.contextBufferCount, n)),
    i.contextBufferCount++);
}
function wi(e, t, n, i, r) {
  let o = Fr(e);
  (o.hunkContent.length > 0 &&
    o.contextBufferCount > e.maxContextLines2 &&
    (yl(e), (o = e.activeHunk)),
    Pn(e, o, o.hunkContent.length === 0 ? "leading" : "before-change"));
  const s = e.additionLines.length,
    a = e.deletionLines.length;
  t === "addition" ? e.additionLines.push(n) : e.deletionLines.push(n);
  const l = bl(o, t, s, a);
  (t === "addition"
    ? (o.additionCount++, o.additionLines++)
    : (o.deletionCount++, o.deletionLines++),
    Pr(e, i, r, l));
}
function Ll(e, t, n, i) {
  if (t.separatorLineIndex == null || t.markerLines.separator == null)
    throw new Error(
      `parseMergeConflictDiffFromFile: conflict ${t.conflictIndex} is missing a separator marker`,
    );
  const r = e.conflictBuilders[t.conflictIndex];
  if (r == null)
    throw new Error(
      `parseMergeConflictDiffFromFile: failed to finalize conflict ${t.conflictIndex}`,
    );
  const o = r.action;
  ((o.markerLines.separator = t.markerLines.separator),
    (o.markerLines.end = i),
    t.markerLines.base != null && (o.markerLines.base = t.markerLines.base),
    (o.conflict = {
      conflictIndex: t.conflictIndex,
      startLineIndex: t.startLineIndex,
      startLineNumber: t.startLineIndex + 1,
      separatorLineIndex: t.separatorLineIndex,
      separatorLineNumber: t.separatorLineIndex + 1,
      endLineIndex: n,
      endLineNumber: n + 1,
      baseMarkerLineIndex: t.baseMarkerLineIndex,
      baseMarkerLineNumber:
        t.baseMarkerLineIndex != null ? t.baseMarkerLineIndex + 1 : void 0,
    }));
  const s = o.currentContentIndex ?? o.incomingContentIndex;
  if (
    ((o.currentContentIndex ??= s),
    (o.incomingContentIndex ??= s),
    o.startContentIndex < 0 && s != null && (o.startContentIndex = s),
    o.endContentIndex < 0 && s != null && (o.endContentIndex = s),
    o.endMarkerContentIndex < 0 && s != null && (o.endMarkerContentIndex = s),
    o.hunkIndex < 0 ||
      o.startContentIndex < 0 ||
      o.endContentIndex < 0 ||
      o.endMarkerContentIndex < 0)
  )
    throw new Error(
      `parseMergeConflictDiffFromFile: failed to anchor merge conflict ${t.conflictIndex}`,
    );
  ((e.actions[o.conflictIndex] = o), (r.completed = !0));
}
function Ei(e, t, n) {
  const i = e.nextConflictIndex;
  (e.nextConflictIndex++,
    e.conflictStack.push({
      conflictIndex: i,
      stage: "current",
      startLineIndex: n,
      markerLines: { start: t },
    }),
    (e.conflictBuilders[i] = {
      completed: !1,
      action: {
        conflict: {
          conflictIndex: i,
          startLineIndex: n,
          startLineNumber: n + 1,
          separatorLineIndex: n,
          separatorLineNumber: n + 1,
          endLineIndex: n,
          endLineNumber: n + 1,
          baseMarkerLineIndex: void 0,
          baseMarkerLineNumber: void 0,
        },
        conflictIndex: i,
        hunkIndex: -1,
        startContentIndex: -1,
        endContentIndex: -1,
        endMarkerContentIndex: -1,
        markerLines: { start: t, separator: "", end: "" },
      },
    }));
}
function Nr(e, t) {
  return {
    additionStart: e,
    deletionStart: t,
    additionCount: 0,
    deletionCount: 0,
    additionLines: 0,
    deletionLines: 0,
    additionLineIndex: Math.max(e - 1, 0),
    deletionLineIndex: Math.max(t - 1, 0),
    hunkContent: [],
    contextBufferAdditionStart: Math.max(e - 1, 0),
    contextBufferDeletionStart: Math.max(t - 1, 0),
    contextBufferCount: 0,
    contextBufferBaseConflicts: void 0,
  };
}
function Ri(e, t) {
  return t === 1 ? `${e}` : `${e},${t}`;
}
function Ti(e) {
  if (e.length < 7) return;
  const t = e.charCodeAt(0);
  if (t !== 60 && t !== 62 && t !== 61 && t !== 124) return;
  const n = xl(e);
  if (n < 7) return;
  let i = 1;
  for (; i < n && e.charCodeAt(i) === t;) i++;
  if (!(i < 7)) {
    if (t === 61) return i === n ? "separator" : void 0;
    if (!(i !== n && !kl(e.charCodeAt(i))))
      return t === 60 ? "start" : t === 62 ? "end" : "base";
  }
}
function xl(e) {
  let t = e.length;
  return (
    t > 0 && e.charCodeAt(t - 1) === 10 && t--,
    t > 0 && e.charCodeAt(t - 1) === 13 && t--,
    t
  );
}
function kl(e) {
  return e === 9 || e === 10 || e === 11 || e === 12 || e === 13 || e === 32;
}
function Ai(e, t, n) {
  return {
    ...e,
    contents: n,
    cacheKey: e.cacheKey != null ? `${e.cacheKey}:merge-conflict-${t}` : void 0,
  };
}
function zr(e, t) {
  const n = [],
    i = new Array(e.hunks.length),
    r = (s, a) => {
      const l = e.hunks[s];
      if (l == null) return 0;
      let d = i[s];
      if (d == null) {
        d = new Array(l.hunkContent.length + 1);
        let h = l.unifiedLineStart;
        d[0] = h;
        for (let c = 0; c < l.hunkContent.length; c++) {
          const u = l.hunkContent[c];
          ((h += u.type === "context" ? u.lines : u.deletions + u.additions),
            (d[c + 1] = h));
        }
        i[s] = d;
      }
      return d[Math.max(a, 0)] ?? l.unifiedLineStart;
    },
    o = (s, a) => {
      const l = r(s, a),
        d = i[s]?.[Math.max(a + 1, 0)] ?? r(s, a + 1);
      return Math.max(l, d - 1);
    };
  for (const s of t) {
    if (s == null) continue;
    const a = e.hunks[s.hunkIndex];
    if (a == null) continue;
    const l = r(s.hunkIndex, s.startContentIndex);
    if (
      (n.push(
        Ve(s, "marker-start", s.startContentIndex, s.markerLines.start, l),
      ),
      s.baseContentIndex != null)
    ) {
      const f = s.currentContentIndex,
        p = s.incomingContentIndex;
      if (f == null || p == null) continue;
      const g = s.markerLines.base;
      if (g == null) continue;
      const m = a.hunkContent[f],
        y = a.hunkContent[s.baseContentIndex],
        S = a.hunkContent[p];
      if (m?.type !== "change" || y?.type !== "context" || S?.type !== "change")
        continue;
      const b = r(s.hunkIndex, f),
        L = r(s.hunkIndex, p);
      (n.push(Ve(s, "marker-base", s.baseContentIndex, g, b + m.deletions)),
        n.push(
          Ve(
            s,
            "marker-separator",
            s.baseContentIndex,
            s.markerLines.separator,
            L,
          ),
          Ve(
            s,
            "marker-end",
            s.endMarkerContentIndex,
            s.markerLines.end,
            o(s.hunkIndex, s.endMarkerContentIndex),
          ),
        ));
      continue;
    }
    const d = s.currentContentIndex;
    if (d == null) continue;
    const h = a.hunkContent[d];
    if (h?.type !== "change") continue;
    const c = r(s.hunkIndex, d),
      u = h.deletions > 0 ? c + h.deletions : l;
    n.push(
      Ve(s, "marker-separator", d, s.markerLines.separator, u),
      Ve(
        s,
        "marker-end",
        s.endMarkerContentIndex,
        s.markerLines.end,
        o(s.hunkIndex, s.endMarkerContentIndex),
      ),
    );
  }
  return n;
}
function Ve(e, t, n, i, r) {
  return {
    type: t,
    hunkIndex: e.hunkIndex,
    contentIndex: n,
    conflictIndex: e.conflictIndex,
    lineText: i,
    lineIndex: r,
  };
}
function wl(e, t) {
  let n = e.unifiedLineStart;
  for (let i = 0; i < t; i++) {
    const r = e.hunkContent[i];
    n += r.type === "context" ? r.lines : r.deletions + r.additions;
  }
  return n;
}
var Ii = class extends Er {
  pendingConflictActions = [];
  pendingMarkerRows = [];
  injectedRows = new Map();
  options;
  constructor(e = { theme: U }, t, n, i) {
    (super(void 0, t, n, i), (this.options = e));
  }
  setConflictState(e, t, n) {
    ((this.pendingConflictActions = e),
      (this.pendingMarkerRows = t),
      this.syncInjectedRows(e, t, n));
  }
  syncInjectedRows(e, t, n) {
    this.injectedRows.clear();
    for (const i of e) {
      const r = i != null ? Dr(i, n) : void 0;
      if (i == null || r == null) continue;
      const o = {
        type: "actions",
        hunkIndex: r.hunkIndex,
        lineIndex: r.lineIndex,
        conflictIndex: i.conflictIndex,
      };
      this.addInjectedRow(o);
    }
    for (const i of t) this.addInjectedRow(i);
  }
  addInjectedRow(e) {
    const t = `${e.hunkIndex}:${e.lineIndex}`,
      n = this.injectedRows.get(t);
    n == null ? this.injectedRows.set(t, [e]) : n.push(e);
  }
  renderDiff(e, t = We) {
    return (
      e != null &&
        this.syncInjectedRows(
          this.pendingConflictActions,
          this.pendingMarkerRows,
          e,
        ),
      super.renderDiff(e, t)
    );
  }
  async asyncRender(e, t = We) {
    return (
      this.syncInjectedRows(
        this.pendingConflictActions,
        this.pendingMarkerRows,
        e,
      ),
      super.asyncRender(e, t)
    );
  }
  createPreElement(e, t) {
    return super.createPreElement(e, t, { "data-has-merge-conflict": "" });
  }
  getUnifiedLineDecoration({ type: e, lineType: t }) {
    const n =
      e === "change"
        ? t === "change-deletion"
          ? "current"
          : "incoming"
        : void 0;
    return {
      gutterLineType: e === "change" ? "context" : t,
      gutterProperties: Hi(n),
      contentProperties: Mi(e, n),
    };
  }
  getSplitLineDecoration({ side: e, type: t }) {
    const n =
      t === "change" ? (e === "deletions" ? "current" : "incoming") : void 0;
    return {
      gutterLineType: t === "change" ? "context" : t,
      gutterProperties: Hi(n),
      contentProperties: Mi(t, n),
    };
  }
  getUnifiedInjectedRowsForLine = (e) => {
    const t = this.injectedRows.get(`${e.hunkIndex}:${e.lineIndex}`);
    if (t == null || t.length === 0) return;
    const { mergeConflictActionsType: n } = this.getOptionsWithDefaults(),
      i = [],
      r = [];
    for (const o of t) {
      if (o.type === "actions") {
        i.push({
          content: El({
            row: o,
            includeDefaultActions: n === "default",
            includeSlot: !0,
          }),
          gutter: Di("action"),
        });
        continue;
      }
      (o.type === "marker-end" ? r : i).push({
        content: Rl(o),
        gutter: Di("marker", o.type),
      });
    }
    return {
      before: i.length > 0 ? i : void 0,
      after: r.length > 0 ? r : void 0,
    };
  };
  getOptionsWithDefaults() {
    const e = super.getOptionsWithDefaults();
    return (
      (e.diffStyle = "unified"),
      (e.lineDiffType = "none"),
      (e.mergeConflictActionsType =
        this.options.mergeConflictActionsType ?? "default"),
      e
    );
  }
};
function Hi(e) {
  return e != null ? { "data-merge-conflict": e } : void 0;
}
function Mi(e, t) {
  if (t != null) {
    if (e === "change")
      return t === "current" || t === "incoming"
        ? { "data-line-type": "context", "data-merge-conflict": t }
        : void 0;
    if (
      t === "marker-start" ||
      t === "marker-base" ||
      t === "marker-separator" ||
      t === "marker-end"
    )
      return { "data-merge-conflict": t };
  }
}
function Di(e, t) {
  const n = Z(void 0, "annotation", 1);
  return (
    (n.properties["data-gutter-buffer"] =
      e === "action"
        ? "merge-conflict-action"
        : `merge-conflict-${t ?? "marker"}`),
    n
  );
}
function El({ row: e, includeDefaultActions: t, includeSlot: n }) {
  const i = t ? Tl(e.conflictIndex) : [];
  return (
    i.push(
      R({
        tagName: "slot",
        properties: {
          name: Mr({
            hunkIndex: e.hunkIndex,
            lineIndex: e.lineIndex,
            conflictIndex: e.conflictIndex,
          }),
          "data-merge-conflict-action-slot": "",
        },
      }),
    ),
    R({
      tagName: "div",
      properties: { "data-merge-conflict-actions": "" },
      children: [
        R({
          tagName: "div",
          properties: { "data-merge-conflict-actions-content": "" },
          children: i,
        }),
      ],
    })
  );
}
function Rl(e) {
  return R({
    tagName: "div",
    properties: {
      "data-merge-conflict": e.type,
      "data-merge-conflict-marker-row": "",
    },
    children: [X(e.lineText.replace(/(?:\r\n|\n|\r)$/, ""))],
  });
}
function Tl(e) {
  return [
    tn({
      resolution: "current",
      label: "Accept current change",
      conflictIndex: e,
    }),
    Fi(),
    tn({
      resolution: "incoming",
      label: "Accept incoming change",
      conflictIndex: e,
    }),
    Fi(),
    tn({ resolution: "both", label: "Accept both", conflictIndex: e }),
  ];
}
function tn({ resolution: e, label: t, conflictIndex: n }) {
  return R({
    tagName: "button",
    properties: {
      type: "button",
      "data-merge-conflict-action": e,
      "data-merge-conflict-conflict-index": `${n}`,
    },
    children: [X(t)],
  });
}
function Fi() {
  return R({
    tagName: "span",
    properties: { "data-merge-conflict-action-separator": "" },
    children: [X("|")],
  });
}
function Al(e, t) {
  return (
    e.hunkIndex === t.hunkIndex &&
    e.startContentIndex === t.startContentIndex &&
    e.endContentIndex === t.endContentIndex &&
    e.currentContentIndex === t.currentContentIndex &&
    e.baseContentIndex === t.baseContentIndex &&
    e.incomingContentIndex === t.incomingContentIndex &&
    e.endMarkerContentIndex === t.endMarkerContentIndex &&
    e.conflictIndex === t.conflictIndex &&
    Il(e.conflict, t.conflict)
  );
}
function Il(e, t) {
  return (
    e.conflictIndex === t.conflictIndex &&
    e.startLineIndex === t.startLineIndex &&
    e.startLineNumber === t.startLineNumber &&
    e.separatorLineIndex === t.separatorLineIndex &&
    e.separatorLineNumber === t.separatorLineNumber &&
    e.endLineIndex === t.endLineIndex &&
    e.endLineNumber === t.endLineNumber &&
    e.baseMarkerLineIndex === t.baseMarkerLineIndex &&
    e.baseMarkerLineNumber === t.baseMarkerLineNumber
  );
}
let Hl = -1;
var Jl = class extends Rr {
  options;
  __id = `unresolved-file:${++Hl}`;
  type = "unresolved-file";
  computedCache = {
    file: void 0,
    fileDiff: void 0,
    actions: void 0,
    markerRows: void 0,
  };
  conflictActions = [];
  markerRows = [];
  conflictActionCache = new Map();
  constructor(e = { theme: U }, t, n = !1) {
    (super(void 0, t, n), (this.options = e), this.setOptions(e));
  }
  setOptions(e) {
    if (e != null) {
      if (e.onMergeConflictAction != null && e.onMergeConflictResolve != null)
        throw new Error(
          "UnresolvedFile: onMergeConflictAction and onMergeConflictResolve are mutually exclusive. Use only one callback.",
        );
      ((this.options = e),
        this.hunksRenderer.setOptions(this.getHunksRendererOptions(e)),
        this.syncInteractionOptions());
    }
  }
  syncInteractionOptions() {
    this.interactionManager.setOptions(
      st(
        this.options,
        typeof this.options.hunkSeparators == "function" ||
          (this.options.hunkSeparators ?? "line-info") === "line-info" ||
          this.options.hunkSeparators === "line-info-basic"
          ? this.handleExpandHunk
          : void 0,
        this.getLineIndex,
        this.handleMergeConflictActionClick,
      ),
    );
  }
  createHunksRenderer(e) {
    return new Ii(
      this.getHunksRendererOptions(e),
      this.getAnnotationSlotName,
      this.handleHighlightRender,
      this.workerManager,
    );
  }
  getHunksRendererOptions(e) {
    return _l(e, this.options);
  }
  applyPreNodeAttributes(e, t) {
    super.applyPreNodeAttributes(e, t, { "data-has-merge-conflict": "" });
  }
  cleanUp() {
    (this.emitPostRender(!0),
      this.clearMergeConflictActionCache(),
      (this.computedCache = {
        file: void 0,
        fileDiff: void 0,
        actions: void 0,
        markerRows: void 0,
      }),
      (this.conflictActions = []),
      super.cleanUp());
  }
  getOrComputeDiff({ file: e, fileDiff: t, actions: n, markerRows: i }) {
    const { maxContextLines: r, onMergeConflictAction: o } = this.options;
    e: if (o != null) {
      const s = t != null;
      if (s !== (n != null) || s !== (i != null))
        throw new Error(
          "UnresolvedFile.getOrComputeDiff: fileDiff, actions, and markerRows must be passed together",
        );
      if (t != null && n != null && i != null) {
        this.computedCache = {
          file: e ?? this.computedCache.file,
          fileDiff: t,
          actions: n,
          markerRows: i,
        };
        break e;
      } else if (e != null || this.computedCache.file != null) {
        if (
          e != null &&
          this.computedCache.file != null &&
          !ot(e, this.computedCache.file) &&
          this.computedCache.fileDiff != null &&
          this.computedCache.actions != null
        )
          throw new Error(
            "UnresolvedFile.getOrComputeDiff: file can only be used to initialize unresolved state once. Pass fileDiff and actions for subsequent updates.",
          );
        if (((e ??= this.computedCache.file), e == null))
          throw new Error(
            "UnresolvedFile.getOrComputeDiff: file is null, should be impossible",
          );
        if (
          !ot(e, this.computedCache.file) ||
          this.computedCache.fileDiff == null ||
          this.computedCache.actions == null
        ) {
          const a = Li(e, r);
          this.computedCache = {
            file: e,
            fileDiff: a.fileDiff,
            actions: a.actions,
            markerRows: a.markerRows,
          };
        }
        ((t = this.computedCache.fileDiff),
          (n = this.computedCache.actions),
          (i = this.computedCache.markerRows));
        break e;
      } else {
        ((t = this.computedCache.fileDiff),
          (n = this.computedCache.actions),
          (i = this.computedCache.markerRows));
        break e;
      }
    } else {
      if (t != null || n != null || i != null)
        throw new Error(
          "UnresolvedFile.getOrComputeDiff: fileDiff, actions, and markerRows are only usable in controlled mode, you must pass in `onMergeConflictAction`",
        );
      if (
        e != null &&
        this.computedCache.file != null &&
        !ot(e, this.computedCache.file)
      )
        throw new Error(
          "UnresolvedFile.getOrComputeDiff: uncontrolled unresolved files parse the file only once. Later updates must come from the cached diff state.",
        );
      if (
        ((this.computedCache.file ??= e),
        this.computedCache.fileDiff == null && this.computedCache.file != null)
      ) {
        const s = Li(this.computedCache.file, r);
        ((this.computedCache.fileDiff = s.fileDiff),
          (this.computedCache.actions = s.actions),
          (this.computedCache.markerRows = s.markerRows));
      }
      ((t = this.computedCache.fileDiff),
        (n = this.computedCache.actions),
        (i = this.computedCache.markerRows));
      break e;
    }
    if (!(t == null || n == null || i == null))
      return { fileDiff: t, actions: n, markerRows: i };
  }
  hydrate(e) {
    const {
        file: t,
        fileDiff: n,
        actions: i,
        markerRows: r,
        lineAnnotations: o,
        fileContainer: s,
        prerenderedHTML: a,
        preventEmit: l = !1,
      } = e,
      d = this.getOrComputeDiff({
        file: t,
        fileDiff: n,
        actions: i,
        markerRows: r,
      });
    d != null &&
      (this.hydrateElements(s, a),
      this.setActiveMergeConflictState(d.actions, d.markerRows),
      zl(this.pre, d.fileDiff, this.options.collapsed) ||
      Ul(this.headerElement, d.fileDiff, this.options.disableFileHeader)
        ? this.render({ ...e, preventEmit: !0 })
        : (this.hydrationSetup({ fileDiff: d.fileDiff, lineAnnotations: o }),
          this.pre != null && this.renderMergeConflictActionSlots()),
      l || this.emitPostRender());
  }
  rerender() {
    !this.enabled ||
      this.fileDiff == null ||
      this.render({ forceRender: !0, renderRange: this.renderRange });
  }
  render(e = {}) {
    const {
        file: t,
        fileDiff: n,
        actions: i,
        markerRows: r,
        lineAnnotations: o,
        preventEmit: s = !1,
        ...a
      } = e,
      l = this.getOrComputeDiff({
        file: t,
        fileDiff: n,
        actions: i,
        markerRows: r,
      });
    if (l == null) return !1;
    this.setActiveMergeConflictState(l.actions, l.markerRows);
    const d = super.render({
      ...a,
      fileDiff: l.fileDiff,
      lineAnnotations: o,
      preventEmit: !0,
    });
    return (
      d && (this.renderMergeConflictActionSlots(), s || this.emitPostRender()),
      d
    );
  }
  resolveConflict(e, t, n = this.computedCache.fileDiff) {
    const i = this.conflictActions[e];
    if (n == null || i == null) return;
    if (i.conflictIndex !== e)
      throw (
        console.error({ conflictIndex: e, action: i }),
        new Error(
          "UnresolvedFile.resolveConflict: conflictIndex and conflictAction don't match",
        )
      );
    const r = to(n, i, t),
      o = this.computedCache.file,
      {
        file: s,
        actions: a,
        markerRows: l,
      } = Ml({
        fileDiff: r,
        previousActions: this.conflictActions,
        resolvedConflictIndex: e,
        previousFile: o,
        resolution: t,
      });
    return { file: s, fileDiff: r, actions: a, markerRows: l };
  }
  resolveConflictAndRender(e, t) {
    const n = this.conflictActions[e];
    if (n == null) return;
    if (n.conflictIndex !== e)
      throw (
        console.error({ conflictIndex: e, action: n }),
        new Error(
          "UnresolvedFile.resolveConflictAndRender: conflictIndex and conflictAction don't match",
        )
      );
    const i = { resolution: t, conflict: n.conflict },
      {
        file: r,
        fileDiff: o,
        actions: s,
        markerRows: a,
      } = this.resolveConflict(e, t) ?? {};
    r == null ||
      o == null ||
      s == null ||
      a == null ||
      ((this.computedCache = {
        file: r,
        fileDiff: o,
        actions: s,
        markerRows: a,
      }),
      this.setActiveMergeConflictState(s, a),
      this.workerManager != null
        ? this.hunksRenderer.renderDiff(o)
        : this.render({ forceRender: !0 }),
      this.options.onMergeConflictResolve?.(r, i));
  }
  setActiveMergeConflictState(e = this.conflictActions, t = this.markerRows) {
    ((this.conflictActions = e),
      (this.markerRows = t),
      this.computedCache.fileDiff != null &&
        this.hunksRenderer instanceof Ii &&
        this.hunksRenderer.setConflictState(
          this.options.mergeConflictActionsType === "none" ? [] : e,
          t,
          this.computedCache.fileDiff,
        ));
  }
  handleMergeConflictActionClick = (e) => {
    const t = this.conflictActions[e.conflictIndex];
    if (t == null) return;
    if (t.conflictIndex !== e.conflictIndex)
      throw (
        console.error({ conflictIndex: e.conflictIndex, action: t }),
        new Error(
          "UnresolvedFile.handleMergeConflictActionClick: conflictIndex and conflictAction don't match",
        )
      );
    const n = { resolution: e.resolution, conflict: t.conflict };
    if (this.options.onMergeConflictAction != null) {
      this.options.onMergeConflictAction(n, this);
      return;
    }
    this.resolveConflictAndRender(e.conflictIndex, e.resolution);
  };
  renderMergeConflictActionSlots() {
    const { fileDiff: e } = this.computedCache;
    if (
      this.isContainerManaged ||
      this.fileContainer == null ||
      typeof this.options.mergeConflictActionsType != "function" ||
      this.conflictActions.length === 0 ||
      e == null
    ) {
      this.clearMergeConflictActionCache();
      return;
    }
    const t = new Map(this.conflictActionCache);
    for (let n = 0; n < this.conflictActions.length; n++) {
      const i = this.conflictActions[n];
      if (i == null) continue;
      if (i.conflictIndex !== n)
        throw (
          console.error({ conflictIndex: n, action: i }),
          new Error(
            "UnresolvedFile.renderMergeConflictActionSlots: conflictIndex and conflictAction don't match",
          )
        );
      const r = Dr(i, e);
      if (r == null) continue;
      const o = i.conflictIndex,
        s = Mr({
          hunkIndex: r.hunkIndex,
          lineIndex: r.lineIndex,
          conflictIndex: o,
        }),
        a = `${n}-${s}`;
      let l = this.conflictActionCache.get(a);
      if (l == null || !Al(l.action, i)) {
        l?.element.remove();
        const d = this.renderMergeConflictAction(i);
        if (d == null) continue;
        const h = wn(s);
        (h.appendChild(d),
          this.fileContainer.appendChild(h),
          (l = { element: h, action: i }),
          this.conflictActionCache.set(a, l));
      }
      t.delete(a);
    }
    for (const [n, { element: i }] of t.entries())
      (this.conflictActionCache.delete(n), i.remove());
  }
  renderMergeConflictAction(e) {
    if (typeof this.options.mergeConflictActionsType != "function") return;
    const t = this.options.mergeConflictActionsType(e, this);
    if (t != null) {
      if (t instanceof HTMLElement) return t;
      if (typeof DocumentFragment < "u" && t instanceof DocumentFragment) {
        const n = document.createElement("div");
        return ((n.style.display = "contents"), n.appendChild(t), n);
      }
    }
  }
  clearMergeConflictActionCache() {
    for (const { element: e } of this.conflictActionCache.values()) e.remove();
    this.conflictActionCache.clear();
  }
};
function Ml({
  fileDiff: e,
  previousActions: t,
  resolvedConflictIndex: n,
  previousFile: i,
  resolution: r,
}) {
  const o = t[n];
  if (o == null)
    throw new Error(
      "rebuildFileAndActions: missing resolved action for unresolved file rebuild",
    );
  const s = Pl(t, n, o, r),
    a = zr(e, s);
  return {
    file: Dl({
      fileDiff: e,
      resolvedAction: o,
      resolvedConflictIndex: n,
      previousFile: i,
      resolution: r,
    }),
    actions: s,
    markerRows: a,
  };
}
function Dl({
  resolvedAction: e,
  resolvedConflictIndex: t,
  previousFile: n,
  fileDiff: i,
  resolution: r,
}) {
  const o = lt(n?.contents ?? ""),
    { conflict: s } = e,
    a = Fl(o, s, r),
    l = [
      ...o.slice(0, s.startLineIndex),
      ...a,
      ...o.slice(s.endLineIndex + 1),
    ].join("");
  return {
    name: n?.name ?? i.name,
    contents: l,
    cacheKey: n?.cacheKey != null ? `${n.cacheKey}:mc-${t}-${r}` : void 0,
  };
}
function Fl(e, t, n) {
  const i = e.slice(
      t.startLineIndex + 1,
      t.baseMarkerLineIndex ?? t.separatorLineIndex,
    ),
    r = e.slice(t.separatorLineIndex + 1, t.endLineIndex);
  return n === "current" ? i : n === "incoming" ? r : [...i, ...r];
}
function Pl(e, t, n, i) {
  const r = Ol(n.conflict, i);
  return e.map((o, s) => {
    if (s !== t && o != null)
      return o.conflict.startLineIndex > n.conflict.endLineIndex
        ? { ...o, conflict: Nl(o.conflict, r) }
        : o;
  });
}
function Ol(e, t) {
  const n =
      (e.baseMarkerLineIndex ?? e.separatorLineIndex) - e.startLineIndex - 1,
    i = e.endLineIndex - e.separatorLineIndex - 1;
  return (
    (t === "current" ? n : t === "incoming" ? i : n + i) -
    (e.endLineIndex - e.startLineIndex + 1)
  );
}
function Nl(e, t) {
  return {
    ...e,
    startLineIndex: e.startLineIndex + t,
    startLineNumber: e.startLineNumber + t,
    separatorLineIndex: e.separatorLineIndex + t,
    separatorLineNumber: e.separatorLineNumber + t,
    endLineIndex: e.endLineIndex + t,
    endLineNumber: e.endLineNumber + t,
    baseMarkerLineIndex:
      e.baseMarkerLineIndex != null ? e.baseMarkerLineIndex + t : void 0,
    baseMarkerLineNumber:
      e.baseMarkerLineNumber != null ? e.baseMarkerLineNumber + t : void 0,
  };
}
function zl(e, t, n = !1) {
  return !n && e == null && t != null;
}
function Ul(e, t, n = !1) {
  return e == null && t != null && !n;
}
function _l(e, t) {
  const n = { ...t, ...e };
  return {
    ...n,
    useTokenTransformer: Hn(n),
    hunkSeparators:
      typeof e?.hunkSeparators == "function" ? "custom" : e?.hunkSeparators,
    mergeConflictActionsType:
      typeof e?.mergeConflictActionsType == "function"
        ? "custom"
        : e?.mergeConflictActionsType,
  };
}
function Bl(e, t) {
  return e == null || t == null
    ? e === t
    : e.top === t.top && e.bottom === t.bottom;
}
const Ur = 1e3,
  Vl = Ur * 4,
  $l = [0, 1e-6, 0.99999, 1],
  Gl = {
    overscrollSize: Ur,
    intersectionObserverMargin: Vl,
    resizeDebugging: !1,
  };
let Rt = 0,
  Wl = -1;
var ed = class he {
  static __STOP = !1;
  static __lastScrollPosition = 0;
  __id = `virtualizer-${++Wl}`;
  config;
  type = "simple";
  intersectionObserver;
  scrollTop = 0;
  height = 0;
  scrollHeight = 0;
  windowSpecs = { top: 0, bottom: 0 };
  root;
  contentContainer;
  resizeObserver;
  observers = new Map();
  visibleInstances = new Map();
  visibleInstancesDirty = !1;
  instancesChanged = new Set();
  reconcileQueue = new Set();
  scrollDirty = !0;
  heightDirty = !0;
  scrollHeightDirty = !0;
  renderedObservers = 0;
  connectQueue = new Map();
  constructor(t) {
    this.config = { ...Gl, ...t };
  }
  setup(t, n) {
    if (this.root == null) {
      ((this.root = t),
        (this.resizeObserver = new ResizeObserver(this.handleContainerResize)),
        (this.intersectionObserver = new IntersectionObserver(
          this.handleIntersectionChange,
          {
            root: this.root,
            threshold: $l,
            rootMargin: `${this.config.intersectionObserverMargin}px 0px ${this.config.intersectionObserverMargin}px 0px`,
          },
        )),
        t instanceof Document ? this.setupWindow() : this.setupElement(n),
        (window.__INSTANCE = this),
        (window.__TOGGLE = () => {
          he.__STOP
            ? ((he.__STOP = !1),
              (this.getScrollContainerElement() ?? window).scrollTo({
                top: he.__lastScrollPosition,
              }),
              W(this.computeRenderRangeAndEmit))
            : ((he.__lastScrollPosition = this.getScrollTop()),
              (he.__STOP = !0));
        }));
      for (const [i, r] of this.connectQueue.entries()) this.connect(i, r);
      (this.connectQueue.clear(),
        this.markDOMDirty(),
        W(this.computeRenderRangeAndEmit));
    }
  }
  instanceChanged(t, n) {
    (this.instancesChanged.add(t),
      n && this.markDOMDirty(),
      W(this.computeRenderRangeAndEmit));
  }
  requestHeightReconcile(t) {
    (this.reconcileQueue.add(t), W(this.computeRenderRangeAndEmit));
  }
  getWindowSpecs() {
    return (
      this.windowSpecs.top === 0 &&
        this.windowSpecs.bottom === 0 &&
        (this.windowSpecs = rn({
          scrollTop: this.getScrollTop(),
          height: this.getHeight(),
          scrollHeight: this.getScrollHeight(),
          overscrollSize: this.config.overscrollSize,
        })),
      this.windowSpecs
    );
  }
  getRoot() {
    return this.root;
  }
  isInstanceVisible(t, n) {
    const i = this.getScrollTop(),
      r = this.getHeight(),
      o = this.config.intersectionObserverMargin,
      s = i - o,
      a = i + r + o;
    return !(t < s - n || t > a);
  }
  handleContainerResize = (t) => {
    if (this.root == null) return;
    let n = !1;
    for (const i of t) {
      const r = i.borderBoxSize[0].blockSize;
      this.root instanceof Document
        ? r !== this.scrollHeight &&
          ((this.scrollHeightDirty = !0),
          (n = !0),
          this.config.resizeDebugging &&
            (console.log("Virtualizer: content size change", this.__id, {
              sizeChange: r - Rt,
              newSize: r,
            }),
            (Rt = r)))
        : i.target === this.root
          ? r !== this.height && ((this.heightDirty = !0), (n = !0))
          : i.target === this.contentContainer &&
            ((this.scrollHeightDirty = !0),
            (n = !0),
            this.config.resizeDebugging &&
              (console.log("Virtualizer: scroller size change", this.__id, {
                sizeChange: r - Rt,
                newSize: r,
              }),
              (Rt = r)));
    }
    n && W(this.computeRenderRangeAndEmit);
  };
  setupWindow() {
    if (this.root == null || !(this.root instanceof Document))
      throw new Error("Virtualizer.setupWindow: Invalid setup method");
    (window.addEventListener("scroll", this.handleWindowScroll, {
      passive: !0,
    }),
      window.addEventListener("resize", this.handleWindowResize, {
        passive: !0,
      }),
      this.resizeObserver?.observe(this.root.documentElement));
  }
  setupElement(t) {
    if (this.root == null || this.root instanceof Document)
      throw new Error("Virtualizer.setupElement: Invalid setup method");
    (this.root.addEventListener("scroll", this.handleElementScroll, {
      passive: !0,
    }),
      this.resizeObserver?.observe(this.root),
      (t ??= this.root.firstElementChild ?? void 0),
      t instanceof HTMLElement &&
        ((this.contentContainer = t), this.resizeObserver?.observe(t)));
  }
  cleanUp() {
    (Ne(this.computeRenderRangeAndEmit),
      this.resizeObserver?.disconnect(),
      (this.resizeObserver = void 0),
      this.intersectionObserver?.disconnect(),
      (this.intersectionObserver = void 0),
      this.root?.removeEventListener("scroll", this.handleElementScroll),
      window.removeEventListener("scroll", this.handleWindowScroll),
      window.removeEventListener("resize", this.handleWindowResize),
      (this.root = void 0),
      (this.contentContainer = void 0),
      this.observers.clear(),
      this.visibleInstances.clear(),
      this.instancesChanged.clear(),
      this.reconcileQueue.clear(),
      this.connectQueue.clear(),
      (this.visibleInstancesDirty = !1),
      (this.windowSpecs = { top: 0, bottom: 0 }),
      (this.scrollTop = 0),
      (this.height = 0),
      (this.scrollHeight = 0),
      (this.scrollDirty = !0),
      (this.heightDirty = !0),
      (this.scrollHeightDirty = !0));
  }
  getOffsetInScrollContainer(t) {
    return this.getScrollTop() + Je(t, this.getScrollContainerElement());
  }
  connect(t, n) {
    if (this.observers.has(t))
      throw new Error("Virtualizer.connect: instance is already connected...");
    return (
      this.intersectionObserver == null
        ? this.connectQueue.set(t, n)
        : (this.intersectionObserver.observe(t),
          this.observers.set(t, n),
          this.instancesChanged.add(n),
          this.markDOMDirty(),
          W(this.computeRenderRangeAndEmit)),
      () => this.disconnect(t)
    );
  }
  disconnect(t) {
    const n = this.observers.get(t);
    (this.connectQueue.delete(t),
      n != null &&
        (this.intersectionObserver?.unobserve(t),
        this.observers.delete(t),
        this.instancesChanged.delete(n),
        this.reconcileQueue.delete(n),
        this.visibleInstances.delete(t) && (this.visibleInstancesDirty = !0),
        this.markDOMDirty(),
        W(this.computeRenderRangeAndEmit)));
  }
  handleWindowResize = () => {
    he.__STOP ||
      window.innerHeight === this.height ||
      ((this.heightDirty = !0), W(this.computeRenderRangeAndEmit));
  };
  handleWindowScroll = () => {
    he.__STOP ||
      this.root == null ||
      !(this.root instanceof Document) ||
      ((this.scrollDirty = !0), W(this.computeRenderRangeAndEmit));
  };
  handleElementScroll = () => {
    he.__STOP ||
      this.root == null ||
      this.root instanceof Document ||
      ((this.scrollDirty = !0), W(this.computeRenderRangeAndEmit));
  };
  computeRenderRangeAndEmit = () => {
    if (he.__STOP) return;
    const t = this.heightDirty || this.scrollHeightDirty;
    let n = this.instancesChanged.size > 0;
    if (this.instancesChanged.size === 0) {
      const o = rn({
        scrollTop: this.getScrollTop(),
        height: this.getHeight(),
        scrollHeight: this.getScrollHeight(),
        overscrollSize: this.config.overscrollSize,
      });
      if (
        !t &&
        Bl(this.windowSpecs, o) &&
        this.renderedObservers === this.observers.size &&
        !this.visibleInstancesDirty &&
        this.reconcileQueue.size === 0
      )
        return;
      this.windowSpecs = o;
    }
    ((this.visibleInstancesDirty = !1),
      (this.renderedObservers = this.observers.size));
    const i = this.getScrollAnchor(this.height),
      r = new Set();
    for (const o of t
      ? this.observers.values()
      : this.visibleInstances.values())
      o.onRender(t) && r.add(o);
    for (const o of this.instancesChanged)
      r.has(o) || (o.onRender(t) && r.add(o));
    this.scrollFix(i);
    for (const o of r) (this.reconcileQueue.delete(o), o.reconcileHeights());
    for (const o of this.reconcileQueue) o.reconcileHeights() && (n = !0);
    (this.reconcileQueue.clear(),
      (n ||= this.instancesChanged.size > 0),
      n && this.markDOMDirty(),
      (n || t) && W(this.computeRenderRangeAndEmit),
      r.clear(),
      this.instancesChanged.clear());
  };
  scrollFix(t) {
    if (t == null) return;
    const n = this.getScrollContainerElement(),
      {
        lineIndex: i,
        lineOffset: r,
        fileElement: o,
        fileOffset: s,
        fileTypeOffset: a,
      } = t;
    if (i != null && r != null) {
      const d = o.shadowRoot?.querySelector(
        `[data-line][data-line-index="${i}"]`,
      );
      if (d instanceof HTMLElement) {
        const h = Je(d, n);
        if (h !== r) {
          const c = h - r;
          this.applyScrollFix(c);
        }
        return;
      }
    }
    const l = Je(o, n);
    if (a === "top") l !== s && this.applyScrollFix(l - s);
    else {
      const d = l + o.getBoundingClientRect().height;
      d !== s && this.applyScrollFix(d - s);
    }
  }
  applyScrollFix(t) {
    (this.root == null || this.root instanceof Document
      ? window.scrollTo({ top: window.scrollY + t, behavior: "instant" })
      : this.root.scrollTo({
          top: this.root.scrollTop + t,
          behavior: "instant",
        }),
      this.markDOMDirty());
  }
  getScrollAnchor(t) {
    const n = this.getScrollContainerElement();
    let i;
    for (const [r] of this.visibleInstances.entries()) {
      const o = Je(r, n),
        s = o + r.offsetHeight;
      let a, l;
      s <= 0 ? ((a = s), (l = "bottom")) : ((a = o), (l = "top"));
      let d, h;
      if (s > 0 && o < t)
        for (const u of r.shadowRoot?.querySelectorAll(
          "[data-line][data-line-index]",
        ) ?? []) {
          if (!(u instanceof HTMLElement)) continue;
          const f = u.dataset.lineIndex;
          if (f == null) continue;
          const p = Je(u, n);
          if (!(p < 0)) {
            ((d = f), (h = p));
            break;
          }
        }
      if (i?.lineOffset != null && h == null) continue;
      let c = !1;
      ((i == null ||
        (h != null && (i.lineOffset == null || h < i.lineOffset)) ||
        (h == null &&
          i.lineOffset == null &&
          ((a >= 0 && (i.fileOffset < 0 || a < i.fileOffset)) ||
            (a < 0 && i.fileOffset < 0 && a > i.fileOffset)))) &&
        (c = !0),
        c &&
          (i = {
            fileElement: r,
            fileTypeOffset: l,
            fileOffset: a,
            lineIndex: d,
            lineOffset: h,
          }));
    }
    return i;
  }
  handleIntersectionChange = (t) => {
    this.scrollDirty = !0;
    for (const { target: n, isIntersecting: i } of t) {
      if (!(n instanceof HTMLElement))
        throw new Error(
          "Virtualizer.handleIntersectionChange: target not an HTMLElement",
        );
      const r = this.observers.get(n);
      r != null &&
        (i && !this.visibleInstances.has(n)
          ? (r.setVisibility(!0),
            this.visibleInstances.set(n, r),
            (this.visibleInstancesDirty = !0))
          : !i &&
            this.visibleInstances.has(n) &&
            (r.setVisibility(!1),
            this.visibleInstances.delete(n),
            (this.visibleInstancesDirty = !0)));
    }
    this.visibleInstancesDirty && W(this.computeRenderRangeAndEmit);
  };
  getScrollTop() {
    if (!this.scrollDirty) return this.scrollTop;
    this.scrollDirty = !1;
    let t =
      this.root == null
        ? 0
        : this.root instanceof Document
          ? window.scrollY
          : this.root.scrollTop;
    return ((t = this.clampScrollTop(t)), (this.scrollTop = t), t);
  }
  scrollTo({ top: t, behavior: n = "auto" }) {
    const { root: i } = this;
    if (i == null) return;
    const r = this.clampScrollTop(t);
    (i instanceof Document
      ? window.scrollTo({ top: r, behavior: n })
      : i.scrollTo({ top: r, behavior: n }),
      (this.scrollDirty = !0),
      W(this.computeRenderRangeAndEmit));
  }
  clampScrollTop(t) {
    return Math.max(0, Math.min(t, this.getScrollHeight() - this.getHeight()));
  }
  getScrollHeight() {
    return this.scrollHeightDirty
      ? ((this.scrollHeightDirty = !1),
        (this.scrollHeight =
          this.root == null
            ? 0
            : this.root instanceof Document
              ? this.root.documentElement.scrollHeight
              : this.root.scrollHeight),
        this.scrollHeight)
      : this.scrollHeight;
  }
  getHeight() {
    return this.heightDirty
      ? ((this.heightDirty = !1),
        (this.height =
          this.root == null
            ? 0
            : this.root instanceof Document
              ? globalThis.innerHeight
              : this.root.getBoundingClientRect().height),
        this.height)
      : this.height;
  }
  markDOMDirty() {
    ((this.scrollDirty = !0),
      (this.scrollHeightDirty = !0),
      (this.heightDirty = !0));
  }
  getScrollContainerElement() {
    return this.root == null || this.root instanceof Document
      ? void 0
      : this.root;
  }
};
function Je(e, t) {
  const n = e.getBoundingClientRect(),
    i = t?.getBoundingClientRect().top ?? 0;
  return n.top - i;
}
function td(e) {
  const t = [];
  for (const n of e) {
    const i = $i.get(n);
    if (i == null)
      throw new Error(
        `getResolvedLanguages: ${n} is not resolved. Please resolve languages before calling getResolvedLanguages`,
      );
    t.push(i);
  }
  return t;
}
function nd(e) {
  for (const t of Array.isArray(e) ? e : [e]) if (!$i.has(t)) return !1;
  return !0;
}
async function id(e) {
  const t = [],
    n = [];
  for (const i of e) {
    if (i === "text" || i === "ansi") continue;
    const r = no(i) ?? io(i);
    "then" in r ? n.push(r) : t.push(r);
  }
  return (
    n.length > 0 &&
      (await Promise.all(n).then((i) => {
        for (const r of i) {
          if (r == null)
            throw new Error("resolvedLanguages: unable to resolve language");
          t.push(r);
        }
      })),
    t
  );
}
function rd(e) {
  return Sn.getResolvedThemes(e);
}
function od(e, t, n = !1) {
  const i = ro({
    name: e,
    variablePrefix: q("global"),
    variableDefaults: t,
    fontStyle: n,
  });
  oo(e, () => Promise.resolve(i));
}
async function sd(e) {
  for (const n of e) so(n);
  const t = await Sn.resolveThemes(e);
  for (let n = 0; n < e.length; n++) ao(e[n], t[n]);
  return t;
}
function _r(e) {
  return "side" in e;
}
function ql(e) {
  return !_r(e);
}
function ad(e) {
  const t = e[0];
  return t == null || _r(t);
}
function ld(e) {
  const t = e[0];
  return t == null || ql(t);
}
function dd(e, t) {
  return e == null || t == null
    ? e === t
    : e.busyWorkers === t.busyWorkers &&
        e.diffCacheSize === t.diffCacheSize &&
        e.fileCacheSize === t.fileCacheSize &&
        e.managerState === t.managerState &&
        e.activeTasks === t.activeTasks &&
        e.queuedTasks === t.queuedTasks &&
        e.themeSubscribers === t.themeSubscribers &&
        e.totalWorkers === t.totalWorkers &&
        e.workersFailed === t.workersFailed;
}
function hd(e) {
  const t = document.createElement("div");
  t.dataset.line = `${e}`;
  const n = document.createElement("div");
  ((n.dataset.columnNumber = ""), (n.textContent = `${e}`));
  const i = document.createElement("div");
  return (
    (i.dataset.columnContent = ""),
    t.appendChild(n),
    t.appendChild(i),
    { row: t, content: i }
  );
}
function cd(e, t = !1) {
  return R({
    tagName: "style",
    children: [X(t ? os(e) : Rn(e))],
    properties: { [lo]: t ? "" : void 0, [Oi]: t ? void 0 : "" },
  });
}
function ud(e) {
  return R({ tagName: "style", children: [X(e)], properties: { [zi]: "" } });
}
function fd(e) {
  return e.includes(`\r
`)
    ? "CRLF"
    : e.includes("\r")
      ? "CR"
      : e.includes(`
`)
        ? "LF"
        : "none";
}
function pd(e) {
  const t = ho(e);
  if (t.length !== 1)
    throw (
      console.error(t),
      new Error(
        "PatchDiff: Provided patch must include only 1 patch, with 1 diff",
      )
    );
  const { files: n } = t[0];
  if (n.length !== 1)
    throw (
      console.error(n),
      new Error("FileDiff: Provided patch must contain exactly 1 file diff")
    );
  return n[0];
}
function gd(e) {
  const t = e[0];
  if (t !== "+" && t !== "-" && t !== " " && t !== "\\") {
    console.error(
      `parseLineType: Invalid firstChar: "${t}", full line: "${e}"`,
    );
    return;
  }
  const n = e.substring(1);
  return {
    line:
      n === ""
        ? `
`
        : n,
    type:
      t === " "
        ? "context"
        : t === "\\"
          ? "metadata"
          : t === "+"
            ? "addition"
            : "deletion",
  };
}
function md(e, t = 10) {
  const n = [];
  let i;
  for (const o of e.split(`
`)) {
    const s = o.match(co);
    if (s != null) {
      i != null &&
        (i.hunkLines.length > 0 && (Tt(i, t, "trailing"), nn(i, n)),
        (i = void 0));
      const a = parseInt(s[3]),
        l = parseInt(s[1]),
        d = parseInt(s[4] ?? "1"),
        h = parseInt(s[2] ?? "1");
      isNaN(a) || isNaN(l) || isNaN(d) || isNaN(h)
        ? n.push(o)
        : (i = {
            additionStart: a,
            deletionStart: l,
            additionCount: 0,
            deletionCount: 0,
            hunkLines: [],
            contextLines: [],
          });
      continue;
    }
    if (i == null) {
      n.push(o);
      continue;
    }
    if (o.startsWith(" ")) i.contextLines.push(o);
    else if (o !== "") {
      if (i.hunkLines.length > 0 && i.contextLines.length > t * 2) {
        const a = i.contextLines.length - t * 2,
          l = i.contextLines.slice(-t);
        Tt(i, t, "trailing");
        const { additionCount: d, deletionCount: h } = i;
        (nn(i, n),
          (i = {
            additionStart: i.additionStart + d + a,
            deletionStart: i.deletionStart + h + a,
            deletionCount: 0,
            additionCount: 0,
            contextLines: l,
            hunkLines: [],
          }));
      }
      (Tt(i, t, i.hunkLines.length === 0 ? "leading" : "before-change"),
        i.hunkLines.push(o),
        o.startsWith("+")
          ? (i.additionCount += 1)
          : o.startsWith("-") && (i.deletionCount += 1));
    }
  }
  i != null && i.hunkLines.length > 0 && (Tt(i, t, "trailing"), nn(i, n));
  const r = n.join(`
`);
  return e.endsWith(`
`)
    ? `${r}
`
    : r;
}
function Tt(e, t, n) {
  if (n === "leading" && e.contextLines.length > t) {
    const i = e.contextLines.length - t;
    (e.contextLines.splice(0, i),
      (e.additionStart += i),
      (e.deletionStart += i));
  }
  return (
    n === "trailing" &&
      e.contextLines.length > t &&
      (e.contextLines.length = t),
    e.contextLines.length > 0 &&
      (je(e.hunkLines, e.contextLines),
      (e.additionCount += e.contextLines.length),
      (e.deletionCount += e.contextLines.length),
      (e.contextLines.length = 0)),
    e
  );
}
function nn(e, t) {
  (t.push(
    `@@ -${Pi(e.deletionStart, e.deletionCount)} +${Pi(e.additionStart, e.additionCount)} @@`,
  ),
    je(t, e.hunkLines));
}
function Pi(e, t) {
  return t === 1 ? `${e}` : `${e},${t}`;
}
export {
  Cd as ALTERNATE_FILE_NAMES_GIT,
  bd as AttachedLanguages,
  yd as AttachedThemes,
  rl as CODE_VIEW_DIFF_OPTION_KEYS,
  ol as CODE_VIEW_FILE_OPTION_KEYS,
  jr as CODE_VIEW_FOOTER_ATTRIBUTE,
  qr as CODE_VIEW_HEADER_ATTRIBUTE,
  Ld as COMMIT_METADATA_SPLIT,
  lo as CORE_CSS_ATTRIBUTE,
  vn as CUSTOM_HEADER_SLOT_ID,
  yi as CodeToTokenTransformStream,
  Ql as CodeView,
  Ye as DEFAULT_CODE_VIEW_FILE_METRICS,
  Kr as DEFAULT_CODE_VIEW_LAYOUT,
  xd as DEFAULT_COLLAPSED_CONTEXT_THRESHOLD,
  Gr as DEFAULT_EXPANDED_REGION,
  We as DEFAULT_RENDER_RANGE,
  Xr as DEFAULT_SMOOTH_SCROLL_SETTINGS,
  U as DEFAULT_THEMES,
  Wr as DEFAULT_TOKENIZE_MAX_LENGTH,
  Bi as DEFAULT_VIRTUAL_FILE_METRICS,
  Yr as DIFFS_DEVELOPMENT_BUILD,
  Ni as DIFFS_SCROLLBAR_GUTTER_MEASURED_PROPERTY,
  Br as DIFFS_SCROLLBAR_MEASURE_ATTRIBUTE,
  Ui as DIFFS_TAG_NAME,
  Er as DiffHunksRenderer,
  _i as EMPTY_RENDER_RANGE,
  kd as EXTENSION_TO_FILE_FORMAT,
  wd as FILENAME_HEADER_REGEX,
  Ed as FILENAME_HEADER_REGEX_GIT,
  Rd as FILE_CONTEXT_BLOB,
  us as File,
  Rr as FileDiff,
  ts as FileRenderer,
  Zl as FileStream,
  Td as GIT_DIFF_FILE_BREAK_REGEX,
  bn as HEADER_FILENAME_SUFFIX_SLOT_ID,
  yn as HEADER_METADATA_SLOT_ID,
  Cn as HEADER_PREFIX_SLOT_ID,
  co as HUNK_HEADER,
  Ad as INDEX_LINE_METADATA,
  qi as InteractionManager,
  Id as MERGE_CONFLICT_BASE_MARKER_REGEX,
  Hd as MERGE_CONFLICT_END_MARKER_REGEX,
  Md as MERGE_CONFLICT_SEPARATOR_MARKER_REGEX,
  Dd as MERGE_CONFLICT_START_MARKER_REGEX,
  Fd as RegisteredCustomLanguages,
  ji as ResizeManager,
  $i as ResolvedLanguages,
  Pd as ResolvingLanguages,
  Vr as SPLIT_WITH_NEWLINES,
  ir as SVGSpriteSheet,
  Fs as ScrollSyncManager,
  Sl as ShikiStreamTokenizer,
  zi as THEME_CSS_ATTRIBUTE,
  Od as UNIFIED_DIFF_FILE_BREAK_REGEX,
  Oi as UNSAFE_CSS_ATTRIBUTE,
  Jl as UnresolvedFile,
  mn as VIRTUALIZED_FILE_DIFF_LAYOUT_CHECKPOINT_INTERVAL,
  bs as VirtualizedFile,
  Wa as VirtualizedFileDiff,
  ed as Virtualizer,
  Aa as areDiffLineAnnotationsEqual,
  Se as areDiffRenderOptionsEqual,
  B as areDiffTargetsEqual,
  Re as areFileRenderOptionsEqual,
  ot as areFilesEqual,
  Ia as areHunkDataEqual,
  Ft as areLanguagesAttached,
  ns as areLineAnnotationsEqual,
  qe as areObjectsEqual,
  Ln as areOptionsEqual,
  rr as arePrePropertiesEqual,
  Vt as areRenderRangesEqual,
  Pt as areSelectionsEqual,
  ce as areThemesAttached,
  Ue as areThemesEqual,
  Bl as areVirtualWindowSpecsEqual,
  dd as areWorkerStatsEqual,
  Nd as attachResolvedLanguages,
  zd as attachResolvedThemes,
  Pe as cleanLastNewline,
  Ud as cleanUpResolvedLanguages,
  _d as cleanUpResolvedThemes,
  Xl as clearRenderQueue,
  mr as cloneFileDiffMetadata,
  gr as cloneHunks,
  Bd as codeToHtml,
  on as createAnnotationElement,
  wn as createAnnotationWrapperNode,
  ro as createCSSVariablesTheme,
  Qn as createDiffSpanDecoration,
  Ct as createEmptyRowBuffer,
  Yi as createFileHeaderElement,
  Z as createGutterGap,
  Wi as createGutterItem,
  or as createGutterUtilityContentNode,
  po as createGutterUtilityElement,
  $e as createGutterWrapper,
  R as createHastElement,
  Ot as createIconElement,
  bt as createNoNewlineElement,
  Qi as createPreElement,
  Fo as createPreWrapperProperties,
  hd as createRowNodes,
  Be as createSeparator,
  vl as createSpanFromToken,
  cd as createStyleElement,
  X as createTextNodeElement,
  ud as createThemeStyleElement,
  Ji as createTransformerWithState,
  sr as createUnsafeCSSStyleNode,
  rn as createWindowFromScrollPosition,
  Ne as dequeueRender,
  Vd as detachString,
  $d as diffAcceptRejectHunk,
  Gd as disposeHighlighter,
  fo as findCodeElement,
  q as formatCSSVariablePrefix,
  Wd as getCustomExtensionsMap,
  qd as getCustomExtensionsVersion,
  j as getFiletypeFromFileName,
  Ge as getHighlighterIfLoaded,
  xn as getHighlighterOptions,
  kn as getHighlighterThemeStyles,
  Ps as getHunkSeparatorSlotName,
  Ho as getIconForType,
  fe as getLineAnnotationName,
  An as getLineAnnotationSource,
  fd as getLineEndingType,
  sn as getLineNodes,
  rt as getOrCreateCodeNode,
  td as getResolvedLanguages,
  no as getResolvedOrResolveLanguage,
  jd as getResolvedOrResolveTheme,
  rd as getResolvedThemes,
  ct as getSharedHighlighter,
  pd as getSingularPatch,
  ht as getThemes,
  Os as getTotalLineCountFromHunks,
  _l as getUnresolvedDiffHunksRendererOptions,
  nd as hasResolvedLanguages,
  Io as hasResolvedThemes,
  hn as hydratePartialDiff,
  Nt as isDefaultRenderRange,
  _r as isDiffAnnotation,
  ad as isDiffAnnotationCollection,
  ql as isFileAnnotation,
  ld as isFileAnnotationCollection,
  Qr as isHighlighterLoaded,
  Kd as isHighlighterLoading,
  Xd as isHighlighterNull,
  Yd as isWorkerContext,
  Oe as parseDiffFromFile,
  gd as parseLineType,
  ho as parsePatchFiles,
  dr as patchScrollbarGutterSize,
  st as pluckInteractionOptions,
  uo as prefersReducedMotion,
  Zr as preloadHighlighter,
  cr as prerenderHTMLIfNecessary,
  Qd as processFile,
  Po as processLine,
  Zd as processPatch,
  yt as pushOrJoinSpan,
  W as queueRender,
  Yl as recordLineAnnotationSource,
  od as registerCustomCSSVariableTheme,
  Jd as registerCustomLanguage,
  oo as registerCustomTheme,
  eh as releaseStringDetachBuffer,
  js as renderDiffWithHighlighter,
  Qo as renderFileWithHighlighter,
  th as replaceCustomExtensions,
  to as resolveConflict,
  io as resolveLanguage,
  id as resolveLanguages,
  nh as resolveRegion,
  ih as resolveTheme,
  sd as resolveThemes,
  rh as setCustomExtension,
  oh as setLanguageOverride,
  In as setPreNodeProperties,
  md as trimPatchContext,
  os as wrapCoreCSS,
  Tn as wrapThemeCSS,
  Rn as wrapUnsafeCSS,
};
