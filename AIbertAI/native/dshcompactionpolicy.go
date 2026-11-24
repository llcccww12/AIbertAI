package main

// Automatic compaction starts at 85% of the context window configured in the
// admin console, which is also the denominator of the composer context meter.
//
// compaction-basic triggers at floor(min(W*ratio, W-O-headroom)), where O is
// the per-request output cap the route *configures* and headroom defaults to
// 64K. Writing the admin output cap as a model entry maxTokens made it such a
// reservation, so a 200K route with a ~45K output cap compacted at ~90K.
//
// The pi-ai adapter already clamps every request's max_tokens to
// contextWindow - estimated prompt - 4096, so a fixed reservation is double
// counting. The route therefore declares the admin cap as the model's output
// capability (provider defaultMaxTokens), which pi-ai still sends and clamps,
// and the SDK reserves nothing. The remaining 15% is the headroom, so the
// trigger is exactly floor(W*0.85) for every window of 80K and more.
//
// Smaller windows need more than 15% to keep a usable answer: pi-ai leaves
// W - prompt - 4096 for output, so the headroom also covers that safety margin
// plus up to 8K of output, capped at half the window (the 16% verbatim tail
// always stays below the trigger).
const (
	dshCompactionThresholdPercent = 85
	dshCompactionSummaryCap       = 32768
	dshPiAISafetyTokens           = 4096
	dshCompactionAnswerRoom       = 8192
)

// dshRequestOutputTokens is the output capability written into the SDK route.
// A cap at or above the window would leave no prompt budget, so it falls back
// to a quarter of the window like the classic runtime does.
func dshRequestOutputTokens(model selectedModelConfig) int {
	output := max(0, model.MaxOutputTokens)
	if model.ContextWindowTokens > 0 && output >= model.ContextWindowTokens {
		return model.ContextWindowTokens / 4
	}
	return output
}

// dshCompactionPolicy returns the compaction-basic model override for one route,
// or nil when the route declares no window (the SDK then reports the missing
// capacity itself).
func dshCompactionPolicy(modelID string, window, output int) map[string]any {
	if window <= 0 {
		return nil
	}
	summary := output
	if summary <= 0 {
		summary = window / 8
	}
	// Explicit, because the SDK would otherwise size the summary call from the headroom.
	summary = max(1, min(summary, dshCompactionSummaryCap))
	answer := dshCompactionAnswerRoom
	if output > 0 {
		answer = min(answer, output)
	}
	headroom := max(window*(100-dshCompactionThresholdPercent)/100, min(dshPiAISafetyTokens+answer, window/2))
	return map[string]any{
		"provider":       "aibertai",
		"model":          modelID,
		"thresholdRatio": float64(dshCompactionThresholdPercent) / 100,
		"headroomTokens": headroom,
		"maxTokens":      summary,
	}
}

// dshCompactionThreshold mirrors compaction-basic's resolveCompactSpec for the
// route Patch writes (no configured request cap, so no reservation).
func dshCompactionThreshold(window, output int) int {
	policy := dshCompactionPolicy("", window, output)
	if policy == nil {
		return 0
	}
	ratio := float64(window) * policy["thresholdRatio"].(float64)
	return int(min(ratio, float64(window-policy["headroomTokens"].(int))))
}
