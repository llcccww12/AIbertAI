import { useState } from "react"
import { useTranslation } from "react-i18next"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import catalog from "@/lib/dsh-global-plugins.json"

type Plugin = { id: string; state: string; zh?: string; detail?: string; en?: string; description?: string; sessionTools?: string[] }

export function DshGlobalPluginsPage() {
  const { i18n, t } = useTranslation()
  const [search, setSearch] = useState("")
  const chinese = i18n.language.startsWith("zh")
  const term = search.trim().toLowerCase()
  const rows = (catalog.plugins as Plugin[]).filter(plugin =>
    [plugin.id, plugin.zh, plugin.detail, plugin.en, plugin.description, ...(plugin.sessionTools ?? [])]
      .some(text => text?.toLowerCase().includes(term))
  )
  return (
    <section className="flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto px-4 pb-6 md:px-6">
      <header className="space-y-1">
        <h1 className="text-xl font-semibold">{t("dshGlobal.title")}</h1>
        <p className="max-w-3xl text-sm text-muted-foreground">{t("dshGlobal.scope")}</p>
      </header>
      <div className="flex flex-wrap items-center justify-between gap-2">
        <Input type="search" className="w-full sm:max-w-64" aria-label={t("dshGlobal.search")} placeholder={t("dshGlobal.search")} value={search} onChange={event => setSearch(event.target.value)} />
        <p className="text-xs text-muted-foreground">{t("dshGlobal.source", { version: catalog.version, commit: catalog.source.slice(0, 7) })}</p>
      </div>
      <div className="rounded-lg border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t("dshGlobal.plugin")}</TableHead>
              <TableHead>{t("dshGlobal.description")}</TableHead>
              <TableHead>{t("dshGlobal.policy")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {rows.map(plugin => (
              <TableRow key={plugin.id}>
                <TableCell>
                  <div className="font-medium">{(chinese ? plugin.zh : plugin.en) || plugin.id}</div>
                  {((chinese ? plugin.zh : plugin.en) || plugin.id) !== plugin.id && <code className="text-xs text-muted-foreground">{plugin.id}</code>}
                </TableCell>
                <TableCell className="min-w-40 whitespace-normal text-muted-foreground">
                  <p>{(chinese && plugin.detail) || plugin.description || "—"}</p>
                  {plugin.sessionTools?.map(tool => (
                    <p className="mt-1 text-xs" key={tool}>{t("dshGlobal.sessionTool", { tool })}</p>
                  ))}
                </TableCell>
                <TableCell><Badge variant={plugin.state === "default" ? "secondary" : "outline"}>{t("dshGlobal." + plugin.state)}</Badge></TableCell>
              </TableRow>
            ))}
            {rows.length === 0 && <TableRow><TableCell colSpan={3} className="h-20 text-center text-muted-foreground">{t("dshGlobal.empty")}</TableCell></TableRow>}
          </TableBody>
        </Table>
      </div>
    </section>
  )
}
