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

export default function EmployeePreset({ experts, expertId, designName, busy, onSelect, language }) {
  // Platform expert IDs remain opaque. The empty platform ID selects DSH Standard.
  const options = useMemo(() => [{ id: 'standard' }, ...experts.map(expert => ({
    id: 'expert:' + expert.id,
    name: expert.id === 'monkeyai-design' ? designName : expert.name,
    description: expert.description,
  }))], [experts, designName])
  const current = expertId ? 'expert:' + expertId : 'standard'
  const locale = language?.startsWith('zh') ? zh : en
  return <span className="dsh-native-theme" data-dsh-preset>
    <AgentPresetSeat useSessionRetainInfo={useMainView} useDeveloperTools={useDeveloperTools}
      load={noop} introduced={noop} dismissRefusal={noop}
      useAgentPresetSeat={read => read({ options, current, busy, error: null, introduce: false })}
      select={id => { if (id !== current) onSelect(id === 'standard' ? '' : id.slice(7)) }}
      t={key => locale[key]} />
  </span>
}
