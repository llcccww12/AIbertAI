import { useEffect, useState, type FormEvent } from "react"
import { useTranslation } from "react-i18next"

import { useAppToast } from "@/components/animated-toast-provider"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Field, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { api } from "@/lib/api"

type TeamCapacity = { max_members: number | null; active_members: number }

const endpoint = "/api/admin/v1/team-capacity"
const minLimit = 1
const maxLimit = 2_147_483_647

export function TeamCapacityCard({ className }: { className?: string }) {
  const { t } = useTranslation()
  const { showToast } = useAppToast()
  const [capacity, setCapacity] = useState<TeamCapacity>()
  const [draft, setDraft] = useState("")
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState("")
  const [revision, setRevision] = useState(0)

  useEffect(() => {
    const controller = new AbortController()
    api<TeamCapacity>(endpoint, { signal: controller.signal })
      .then((value) => {
        if (controller.signal.aborted) return
        setCapacity(value)
        setDraft(value.max_members?.toString() ?? "")
      })
      .catch((reason: Error) => {
        if (!controller.signal.aborted) setError(reason.message)
      })
    return () => controller.abort()
  }, [revision])

  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!capacity || saving) return
    const limit = draft === "" ? null : Number(draft)
    if (limit !== null && (!Number.isInteger(limit) || limit < minLimit || limit > maxLimit)) {
      setError(t("pages.teamCapacity.invalidLimit"))
      return
    }
    setError("")
    setSaving(true)
    try {
      const value = await api<TeamCapacity>(endpoint, {
        method: "PUT",
        body: JSON.stringify({ max_members: limit }),
      })
      setCapacity(value)
      setDraft(value.max_members?.toString() ?? "")
      showToast({ status: "success", title: t("pages.teamCapacity.saved") })
    } catch (reason) {
      setError((reason as Error).message)
    } finally {
      setSaving(false)
    }
  }

  return (
    <Card className={className}>
      <CardHeader>
        <CardTitle>{t("pages.teamCapacity.title")}</CardTitle>
        <CardDescription>{t("pages.teamCapacity.description")}</CardDescription>
      </CardHeader>
      <CardContent>
        <form className="space-y-3" onSubmit={save}>
          <Field>
            <FieldLabel htmlFor="team-capacity-limit">{t("pages.teamCapacity.limit")}</FieldLabel>
            <div className="flex flex-wrap items-center gap-2">
              <Input
                id="team-capacity-limit"
                type="number"
                min={minLimit}
                max={maxLimit}
                step={1}
                className="w-44"
                disabled={!capacity || saving}
                placeholder={t("pages.teamCapacity.unlimited")}
                value={draft}
                onChange={(event) => setDraft(event.target.value)}
              />
              <Button type="submit" loading={saving} disabled={!capacity || saving || draft === (capacity.max_members?.toString() ?? "")}>
                {t("pages.teamCapacity.save")}
              </Button>
            </div>
          </Field>
          {capacity && <p className="text-sm text-muted-foreground">{t("pages.teamCapacity.active", { count: capacity.active_members })}</p>}
          {error && <p role="alert" className="text-sm text-destructive">{error}</p>}
          {!capacity && error && <Button type="button" variant="outline" onClick={() => { setError(""); setRevision((value) => value + 1) }}>{t("pages.teamCapacity.retry")}</Button>}
        </form>
      </CardContent>
    </Card>
  )
}
