import { useMemo } from 'react'
import { installIconSprite } from 'sdk/client/ui-theme/src/client/icon-sprite.ts'
import { AgentPresetSeat } from 'sdk/client/ui-agent-preset/src/client/AgentPresetSeat.tsx'
import { zh, en } from 'sdk/client/ui-agent-preset/src/client/locales.ts'

// A new composer can load before a conversation or the files panel installs the document sprite.
if (typeof document !== 'undefined' && !document.querySelector('[data-dsh-icon-sprite]')) {
  installIconSprite({ effect: mount => mount(), on: () => {} }, undefined)
}

const noop = () => {}
const useMainView = select => select({ retainedBy: { mainView: 1 } })
const useDeveloperTools = select => select(false)

export default function EmployeePreset({ experts, expertId, busy, onSelect, language }) {
  // Platform expert IDs remain opaque. The empty platform ID selects DSH Standard.
  // The canvas is not a preset: Standard opens it from the design categories,
  // so canvas sessions (monkeyai-design) show as Standard.
  const options = useMemo(() => [{ id: 'standard' }, ...experts.filter(expert => expert.id !== 'monkeyai-design').map(expert => ({
    id: 'expert:' + expert.id,
    name: expert.name,
    description: expert.description,
  }))], [experts])
  const current = expertId && expertId !== 'monkeyai-design' ? 'expert:' + expertId : 'standard'
  const locale = language?.startsWith('zh') ? zh : en
  return <span className="dsh-native-theme" data-dsh-preset>
    <AgentPresetSeat useSessionRetainInfo={useMainView} useDeveloperTools={useDeveloperTools}
      load={noop} introduced={noop} dismissRefusal={noop}
      useAgentPresetSeat={read => read({ options, current, busy, error: null, introduce: false })}
      select={id => { if (id !== current) onSelect(id === 'standard' ? '' : id.slice(7)) }}
      t={key => locale[key]} />
  </span>
}
