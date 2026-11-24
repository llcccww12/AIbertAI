import { useEffect, useState } from 'react'
import { OpenTargetButton } from 'sdk/client/ui-open-in-app/src/client/OpenTargetButton.tsx'
import { zh, en } from 'sdk/client/ui-open-in-app/src/client/locales.ts'

export function EmployeeOpenTarget({ scope, path, bridge, language, reportError }) {
  const [target, setTarget] = useState(null)
  const identity = JSON.stringify([scope.accountId, scope.sessionId, path])
  useEffect(() => {
    let current = true
    bridge(scope, path, 'inspect').then(result => {
      if (current) setTarget({ identity, application: result.application })
    }).catch(() => { if (current) setTarget(null) })
    return () => { current = false }
  }, [identity, bridge])
  if (target?.identity !== identity) return null
  const dictionary = language?.startsWith('zh') ? zh : en
  const t = (key, values) => dictionary[key].replace(/\{(\w+)\}/g, (token, name) => values?.[name] ?? token)
  return <OpenTargetButton kind="file" applications={target.application ? [target.application] : []}
    defaultId={target.application?.id} failed={false} t={t}
    execute={async op => {
      const reveal = op.kind === 'reveal'
      try {
        await bridge(scope, path, reveal ? 'reveal' : 'open')
        return { kind: reveal ? 'revealed' : 'opened' }
      } catch {
        return { kind: 'failed', failure: reveal ? 'revealError' : 'openError' }
      }
    }} report={outcome => { if (outcome.kind === 'failed') reportError(t('path.' + outcome.failure)) }} />
}
