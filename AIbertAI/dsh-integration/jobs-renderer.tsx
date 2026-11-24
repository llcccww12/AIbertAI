import { useMemo, useSyncExternalStore } from 'react'
import { ClientJobsModel } from 'sdk/api/job-controller/src/client/model.ts'
import { JobListAction } from 'sdk/client/ui-jobs/src/client/JobListAction.tsx'
import { zh, en } from 'sdk/client/ui-jobs/src/client/locales.ts'

// ACP is request/response. Only watch the mounted session and only read
// output for the expanded row. DSH's model keeps the observer cursor separate
// from the agent's consuming reads.
function watch(poll, failed) {
  let active = true
  let timer
  const current = () => active
  async function tick() {
    let delay
    try { delay = await poll(current) }
    catch (err) { if (active) failed(err); delay = 1800 }
    if (active && delay !== undefined) timer = setTimeout(tick, delay)
  }
  void tick()
  return () => { active = false; clearTimeout(timer) }
}

export default function EmployeeJobs({ sessionId, request, language }) {
  const adapter = useMemo(() => {
    const model = new ClientJobsModel()
    const subscribe = model.subscribe.bind(model)
    const snapshot = model.getSnapshot.bind(model)
    return {
      useJobs: select => select(useSyncExternalStore(subscribe, snapshot)),
      watchRows: () => watch(async current => {
        const { jobs } = await request({ sessionId, action: 'list' })
        if (current()) model.rowsReplaced(sessionId, jobs)
        return jobs.some(job => job.status === 'running' || job.status === 'stopping') ? 750 : 1500
      }, () => model.rowsDropped(sessionId)),
      observe: (_, jobId) => {
        let opened = false
        const stop = watch(async current => {
          const data = await request({ sessionId, action: 'output', jobId, from: model.cursorOf(jobId) })
          if (!current()) return
          if (!opened) {
            model.observeOpened(jobId, { type: 'opened', job: data.job, from: data.from })
            opened = true
          }
          if (data.chunks.length || data.lossy) model.observeOutput(jobId, { type: 'output', ...data })
          if (data.job.status === 'running' || data.job.status === 'stopping') return 400
          model.observeSettled(jobId)
        }, err => { opened = false; model.observeFailed(jobId, err) })
        return () => { stop(); model.observeStopped(jobId) }
      },
      killJob: async (_, jobId) => {
        const { result } = await request({ sessionId, action: 'kill', jobId })
        return result === 'requested' || result === 'already-finished'
      },
    }
  }, [request, sessionId])
  const dictionary = language?.startsWith('zh') ? zh : en
  const t = (key, params) => (dictionary[key] ?? key).replace(/\{(\w+)\}/g, (token, name) => params?.[name] ?? token)
  return <span className="dsh-native-theme"><JobListAction sessionId={sessionId} {...adapter} t={t} /></span>
}
