import { useEffect, useState, type FormEvent } from "react"
import { useTranslation } from "react-i18next"
import { Link } from "react-router-dom"

import { useAppToast } from "@/components/animated-toast-provider"
import { Button } from "@/components/ui/button"
import {
  Dialog, DialogClose, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger,
} from "@/components/ui/dialog"
import { Field, FieldDescription, FieldLabel } from "@/components/ui/field"
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { api } from "@/lib/api"
import { CONSOLE_ROUTES } from "@/lib/routes"

type EmbeddingModel = {
  id: string
  model_id: string
  display_name: string
  kind: string
  enabled: boolean
  embedding_config?: { dimension: number }
}

const copy = "pages.otherSettings.knowledgeBase"
const automatic = "automatic"

export function EmbeddingModelDialog({
  currentModelID, scope, onSave,
}: {
  currentModelID: string
  scope: "default" | "knowledgeBase"
  onSave: (id: string) => Promise<boolean>
}) {
  const { t } = useTranslation()
  const { showToast } = useAppToast()
  const [open, setOpen] = useState(false)
  const [options, setOptions] = useState<Array<{ value: string; label: string }>>([])
  const [selected, setSelected] = useState("")
  const [pending, setPending] = useState(false)
  const [loaded, setLoaded] = useState(false)
  const isDefault = scope === "default"
  const initial = currentModelID || (isDefault ? automatic : "")
  const title = t(`${copy}.${isDefault ? "embeddingModel" : "switchEmbedding"}`)
  const choices = isDefault ? [{ value: automatic, label: t(`${copy}.embeddingAutomatic`) }, ...options] : options

  useEffect(() => {
    if (!open) return
    const controller = new AbortController()
    api<{ models: EmbeddingModel[] }>("/api/admin/v1/models?ownership_type=system", { signal: controller.signal })
      .then(({ models }) => {
        if (controller.signal.aborted) return
        setOptions(models
          .filter((model) => model.kind === "embedding" && model.enabled && (model.embedding_config?.dimension ?? 0) > 0)
          .map((model) => ({ value: model.id, label: model.display_name || model.model_id })))
        setLoaded(true)
      })
      .catch((error: Error) => {
        if (!controller.signal.aborted) showToast({ status: "error", title: error.message })
      })
    return () => controller.abort()
  }, [open, showToast])

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (pending || !loaded || selected === initial || !choices.some(({ value }) => value === selected)) return
    setPending(true)
    try {
      if (await onSave(selected === automatic ? "" : selected)) {
        setOpen(false)
        showToast({ status: "success", title: t(`${copy}.modelSaved`) })
      }
    } catch (reason) {
      showToast({ status: "error", title: reason instanceof Error ? reason.message : String(reason) })
    } finally {
      setPending(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={(next) => {
      if (pending) return
      if (next) {
        setLoaded(false)
        setSelected(initial)
      }
      setOpen(next)
    }}>
      <DialogTrigger render={<Button type="button" variant={isDefault ? "secondary" : "outline"} />}>
        {isDefault ? t(`${copy}.configure`) : title}
      </DialogTrigger>
      <DialogContent className="sm:max-w-lg" closeLabel={t("common.close")}>
        <form className="flex flex-col gap-6" onSubmit={submit}>
          <DialogHeader>
            <DialogTitle>{title}</DialogTitle>
            <DialogDescription>{t(`${copy}.${isDefault ? "embeddingDefaultDescription" : "switchEmbeddingDescription"}`)}</DialogDescription>
          </DialogHeader>
          <Field>
            <FieldLabel htmlFor={`embedding-model-${scope}`}>{t(`${copy}.embeddingModel`)}</FieldLabel>
            <Select items={choices} value={selected} onValueChange={(value) => setSelected(value ?? "")} disabled={!loaded || pending}>
              <SelectTrigger id={`embedding-model-${scope}`} className="w-full">
                <SelectValue placeholder={t(`${copy}.embeddingSelect`)} />
              </SelectTrigger>
              <SelectContent alignItemWithTrigger={false}>
                <SelectGroup>
                  {choices.map(({ value, label }) => <SelectItem key={value} value={value}>{label}</SelectItem>)}
                </SelectGroup>
              </SelectContent>
            </Select>
            <FieldDescription>
              {t(`${copy}.embeddingRegistrationHint`)}{" "}
              <Link className="underline underline-offset-2" to={CONSOLE_ROUTES.models}>{t(`${copy}.manageEmbeddingModels`)}</Link>
            </FieldDescription>
          </Field>
          <DialogFooter>
            <DialogClose render={<Button type="button" variant="outline" disabled={pending} />}>{t(`${copy}.cancel`)}</DialogClose>
            <Button type="submit" loading={pending} disabled={!loaded || selected === initial || !choices.some(({ value }) => value === selected)}>
              {t(`${copy}.save`)}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
