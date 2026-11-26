// Static checks for the AIbert running indicator: own copy, own icon, no DeepSeek whale.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');

const dir = path.join(__dirname, '..', '..', 'AIbertAI', 'code', 'web', 'dsh', 'native-chat');
const js = fs.readFileSync(path.join(dir, 'reply.js'), 'utf8');
const css = fs.readFileSync(path.join(dir, 'reply.css'), 'utf8');

// Copy: the running label and the running clock must be ours, in both real locales.
assert.match(js, /"chat\.deepDiving":"AIbert 正在思考"/, 'zh label');
assert.match(js, /"chat\.deepDivingFor":"AIbert 正在思考，用时 \{duration\}"/, 'zh clock');
assert.match(js, /"chat\.deepDiving":"AIbert is thinking"/, 'en label');
assert.match(js, /"chat\.deepDivingFor":"AIbert is thinking for \{duration\}"/, 'en clock');
assert.doesNotMatch(js, /\\u6DF1\\u5EA6\\u6C42\\u7D22\\u4E2D/, 'zh copy still says 深度求索中');
assert.doesNotMatch(js, /Deep diving/, 'en copy still says Deep diving');
// Team mode keeps its own sentence for the same clock (waiting for a member).
assert.match(js, /"chat\.deepDiving":l,"chat\.deepDivingFor":l\+\(i==="zh"\?"\\uFF0C\\u672C\\u8F6E\\u5DF2\\u7528 \{duration\}":" \\xB7 \{duration\} elapsed this turn"\)/, 'team mode copy');

// Icon: the whale path, mask asset and both whale spans are gone.
assert.doesNotMatch(js, /M8\.844 13\.742/, 'whale path still bundled');
assert.doesNotMatch(js, /runningWhale/, 'whale classes still bundled');
assert.doesNotMatch(css, /running-whale/, 'whale asset still referenced');
assert.doesNotMatch(js, /runningFrag/, 'old fragment icon still bundled');
assert.equal(fs.existsSync(path.join(dir, 'running-whale@2x-4VWCUEON.png')), false, 'whale png still shipped');

// The spark: twelve rounded rays of unequal length, brand orange, breathing.
const icon = /className:"ChatView_runningBloomPetals",stroke:"#E2804F",strokeWidth:1\.15,strokeLinecap:"round",d:"([^"]+)"/.exec(js);
assert(icon, 'spark path missing');
const rays = icon[1].match(/M8 8L[\d.]+ [\d.]+/g) || [];
assert.equal(rays.length, 12, 'expected twelve rays, got ' + rays.length);
const lengths = new Set(rays.map(ray => {
  const [, x, y] = ray.match(/M8 8L([\d.]+) ([\d.]+)/);
  return (Math.hypot(Number(x) - 8, Number(y) - 8)).toFixed(1);
}));
assert(lengths.size > 1, 'rays all share one length');
assert.match(js, /viewBox:"0 0 16 16"/, 'icon viewBox');

// Breathing lives in CSS and collapses under reduced motion.
assert.match(css, /\.ChatView_runningBloom\{[^}]*width:100%/, 'bloom rule missing');
assert.match(css, /@keyframes ChatView_bloomBreath\{/, 'breathing keyframes missing');
assert.match(css, /@keyframes ChatView_bloomBreath\{[^}]*\}[^}]*\}[^}]*\}/, 'breathing keyframes incomplete');
assert.ok(css.includes('@media(prefers-reduced-motion:no-preference){.ChatView_runningBloom{animation:ChatView_bloomBreath'), 'breathing not behind no-preference');
assert.doesNotMatch(css, /prefers-reduced-motion:reduce\{[^}]*ChatView_runningBloom[^}]*animation:/, 'breathing not disabled for reduced motion');
assert.match(css, /@media\(forced-colors:active\)\{\.ChatView_runningBloomPetals\{stroke:CanvasText\}\}/, 'no forced-colors fallback');

// The fallback stays readable for screen readers and keeps the live region.
assert.match(js, /role:"status","aria-live":"polite"/, 'live region lost');
console.log('AIbert running indicator: own copy, own spark, breathing behind reduced-motion, forced-colors fallback present');
