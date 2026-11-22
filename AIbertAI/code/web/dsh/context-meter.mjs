const categories = ['systemTokens', 'toolsTokens', 'messageTokens'];
const isCount = value => Number.isSafeInteger(value) && value >= 0;

export function contextReading(measurement) {
  if (!isCount(measurement?.used) || !isCount(measurement?.window) || measurement.window === 0) return undefined;
  const { used, window: size, breakdown } = measurement;
  const hasBreakdown = breakdown && categories.every(key => isCount(breakdown[key]));
  return { used, size, ...(hasBreakdown ? { breakdown } : {}) };
}

export function contextSegments({ used, size, breakdown }) {
  const percent = Math.min(100, used / size * 100);
  const total = breakdown && categories.reduce((sum, key) => sum + breakdown[key], 0);
  if (!total) return [{ key: 'total', width: percent }];
  return categories.map(key => ({ key, width: percent * breakdown[key] / total }));
}

export function formatContextTokens(count) {
  for (const [base, suffix] of [[1_000_000, 'M'], [1_000, 'K']]) {
    if (count >= base) return (Math.round(count / base * 10) / 10) + suffix;
  }
  return String(count);
}
