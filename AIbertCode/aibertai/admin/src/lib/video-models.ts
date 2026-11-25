export const VIDEO_MODELS = [
  { provider: "xai", label: "Grok Imagine Video 1.5", model: "grok-imagine-video-1.5" },
  { provider: "minimax", label: "MiniMax H3", model: "MiniMax-H3" },
  { provider: "volcengine", label: "Seedance 2.0", model: "doubao-seedance-2-0-260128" },
  { provider: "volcengine", label: "Seedance 2.5", model: "doubao-seedance-2-5-260628" },
  { provider: "agnes", label: "Agnes Video 2.5", model: "agnes-video-2.5" },
  { provider: "agnes", label: "Agnes Video 2.5 Flash", model: "agnes-video-2.5-flash" },
] as const

export type VideoProvider = (typeof VIDEO_MODELS)[number]["provider"]
export type VideoMode = "text_to_video" | "image_to_video" | "last_frame_to_video" | "first_last_to_video" | "reference_image_to_video"
type VideoValue = string | number | boolean
export type VideoConfig = {
  modes: VideoMode[]
  defaults: Record<string, Record<string, VideoValue>>
  limits?: Record<string, Record<string, { choices?: VideoValue[] }>>
}
export type VideoCapabilities = {
  modes: VideoMode[]
  params: Array<{ name: string; type: "string" | "integer" | "boolean"; choices?: VideoValue[]; min?: number; max?: number }>
  param_rules?: Array<{ when: { modes?: VideoMode[] }; name: string; choices: VideoValue[] }>
}

export function videoResolutions(cap?: VideoCapabilities | null) {
  return cap?.params.find((param) => param.name === "resolution")?.choices?.filter((value): value is string => typeof value === "string") ?? []
}

// Resolutions a mode accepts, after the provider's per-mode rules.
export function videoModeResolutions(cap: VideoCapabilities, mode: VideoMode) {
  const rule = cap.param_rules?.find((item) => item.name === "resolution" && item.when.modes?.includes(mode))
  return (rule?.choices ?? videoResolutions(cap)).filter((value): value is string => typeof value === "string")
}

// The resolutions an admin left open, read back from the per-mode limits.
export function videoOpenResolutions(cap: VideoCapabilities, saved?: VideoConfig) {
  const all = videoResolutions(cap)
  const limited = Object.values(saved?.limits ?? {}).flatMap((limits) => limits.resolution?.choices ?? [])
  const open = all.filter((value) => limited.includes(value))
  return open.length ? open : all
}

export function videoDefaults(cap: VideoCapabilities, modes: VideoMode[], saved?: VideoConfig, resolutions?: string[]) {
  return Object.fromEntries(modes.map((mode) => {
    const values: Record<string, VideoValue> = {}
    for (const param of cap.params) {
      const rule = cap.param_rules?.find((item) => item.name === param.name && item.when.modes?.includes(mode))
      const choices = param.name === "resolution" && resolutions
        ? (rule?.choices ?? param.choices)?.filter((value) => resolutions.includes(value as string))
        : rule?.choices ?? param.choices
      const savedValue = saved?.defaults?.[mode]?.[param.name]
      const valid = savedValue !== undefined && (choices ? choices.includes(savedValue) : param.type === "boolean"
        ? typeof savedValue === "boolean"
        : typeof savedValue === "number" && Number.isInteger(savedValue) && savedValue >= (param.min ?? 0) && savedValue <= (param.max ?? Infinity))
      values[param.name] = savedValue !== undefined && valid
        ? savedValue
        : choices?.[0] ?? (param.type === "boolean" ? true : param.min ?? "")
    }
    return [mode, values]
  }))
}
