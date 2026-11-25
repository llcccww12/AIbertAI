import { useEffect, useMemo, useState, type FormEvent } from "react"
import {
  AiChat02Icon,
  Alert02Icon,
  Coins01Icon,
  Download04Icon,
  FilterHorizontalIcon,
  Search02Icon,
  Task01Icon,
  TokenCircleIcon,
  UserMultiple02Icon,
} from "@hugeicons/core-free-icons"
import { HugeiconsIcon, type IconSvgElement } from "@hugeicons/react"
import { useTranslation } from "react-i18next"
import { TablePagination } from "@/components/table-pagination"
import { TableLoading } from "@/components/table-loading"
import { formatLastActivity, TABLE_DATE_TIME_OPTIONS } from "@/lib/date-time"
import { Link } from "react-router-dom"

import { AccountDialog } from "@/components/billing/account-dialog"
import { UserAvatar } from "@/components/user-avatar"
import { UserSearch } from "@/components/user-search"
import { useAppToast } from "@/components/animated-toast-provider"
import { useStatistics } from "@/hooks/use-statistics"
import { useTableQuery } from "@/hooks/use-table-query"
import { queryDate, queryDateValue } from "@/lib/table-query"
import { StatisticsFeedback } from "@/components/statistics-feedback"
import { DatePickerField } from "@/components/date-picker-field"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { StatisticsCardHeader } from "@/components/statistics-card-header"
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { api } from "@/lib/api"
import { base } from "@/lib/resources"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuOptionsContent,
  DropdownMenuGroup,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Input } from "@/components/ui/input"
import { CONSOLE_ROUTES } from "@/lib/routes"
import { ScrollArea } from "@/components/ui/scroll-area"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"

type TaskHistoryEntry = {
  id: string
  title: string
  user_id?: string
  user_name: string
  user_email: string
  user_avatar_url?: string | null
  started_at: string
  last_active_at: string
  turn_count: number | null
  expert_id?: string | null
  expert_name?: string | null
  model_name?: string | null
  client_type?: string
  active_seconds?: number
  failed_turns?: number
  interrupted_turns?: number
  input_tokens?: number
  output_tokens?: number
  files_changed?: number
  credits?: string
}

type TaskHistorySummary = {
  sessions: number
  members: number
  turns: number
  failed_turns: number
  failed_sessions: number
  tokens: number
  credits: string
  window_from: string
}

type ExpertOption = { id: string; name: string }

type TaskHistory = {
  items: TaskHistoryEntry[]
  has_next: boolean
  page: number
  page_size: number
}

type SearchField = "taskName" | "user" | "expert" | "status" | "dateRange"
type StatusFilter = "all" | "failed"

type TaskHistoryFilters = {
  taskName: string
  userName: string
  userId: string
  expertId: string
  status: StatusFilter
  startTime: Date | undefined
  endTime: Date | undefined
}

const DEFAULT_PAGE_SIZE = 50

const EMPTY_FILTERS: TaskHistoryFilters = {
  taskName: "",
  userName: "",
  userId: "",
  expertId: "",
  status: "all",
  startTime: undefined,
  endTime: undefined,
}

const TASK_QUERY_KEYS = [
  "task",
  "user",
  "user_id",
  "expert_id",
  "status",
  "from",
  "until",
] as const
const EXPORT_PAGE_SIZE = 500
const EXPORT_MAX_PAGES = 20
function taskFilterInput(filters: Record<string, string>): TaskHistoryFilters {
  return {
    ...EMPTY_FILTERS,
    taskName: filters.task ?? "",
    userName: filters.user_id ? "" : (filters.user ?? ""),
    userId: filters.user_id ?? "",
    expertId: filters.expert_id ?? "",
    status: filters.status === "failed" ? "failed" : "all",
    startTime: queryDate(filters.from),
    endTime: queryDate(filters.until, true),
  }
}
function activeTaskFields(filters: Record<string, string>): SearchField[] {
  return [
    ...(filters.task ? ["taskName" as const] : []),
    ...(filters.user || filters.user_id ? ["user" as const] : []),
    ...(filters.expert_id ? ["expert" as const] : []),
    ...(filters.status === "failed" ? ["status" as const] : []),
    ...(filters.from || filters.until ? ["dateRange" as const] : []),
  ]
}

export function TaskHistoryPage() {
  const { i18n, t } = useTranslation()
  const { showToast } = useAppToast()
  const tableQuery = useTableQuery(TASK_QUERY_KEYS, DEFAULT_PAGE_SIZE)
  const { filters, filterQuery, page, pageSize, setPage, setPageSize } =
    tableQuery
  const [previousQuery, setPreviousQuery] = useState(filterQuery)
  const [filterInput, setFilterInput] = useState<TaskHistoryFilters>(() =>
    taskFilterInput(filters)
  )
  const [visibleFields, setVisibleFields] = useState<Set<SearchField>>(
    () =>
      new Set(["taskName", "user", "status", ...activeTaskFields(filters)])
  )
  const [now, setNow] = useState(() => Date.now())
  const [accountUserID, setAccountUserID] = useState<string | null>(null)
  const locale = i18n.resolvedLanguage ?? i18n.language
  if (previousQuery !== filterQuery) {
    setPreviousQuery(filterQuery)
    setFilterInput(taskFilterInput(filters))
    setVisibleFields(
      (current) => new Set([...current, ...activeTaskFields(filters)])
    )
  }

  useEffect(() => {
    const timer = window.setInterval(() => setNow(Date.now()), 60_000)
    return () => window.clearInterval(timer)
  }, [])

  const filterParams = new URLSearchParams({
    task: filters.task ?? "",
    user: filters.user_id ? "" : (filters.user ?? ""),
  })
  if (filters.user_id) filterParams.set("user_id", filters.user_id)
  if (filters.expert_id) filterParams.set("expert_id", filters.expert_id)
  if (filters.status === "failed") filterParams.set("status", "failed")
  if (filters.from) filterParams.set("from", filters.from)
  if (filters.until) filterParams.set("until", filters.until)
  const query = new URLSearchParams(filterParams)
  query.set("page", String(page))
  query.set("page_size", String(pageSize))
  const request = useStatistics<TaskHistory>(
    `/api/admin/v1/statistics/history?${query}`
  )
  const summary = useStatistics<TaskHistorySummary>(
    `/api/admin/v1/statistics/history/summary?${filterParams}`,
    0,
    { retryAction: false }
  )
  const [experts, setExperts] = useState<ExpertOption[]>([])
  useEffect(() => {
    const controller = new AbortController()
    api<{ items?: ExpertOption[] }>(
      `${base}/experts?ownership_type=system&limit=200`,
      { signal: controller.signal }
    )
      .then((data) => setExperts(data.items ?? []))
      .catch(() => {})
    return () => controller.abort()
  }, [])
  const [exporting, setExporting] = useState(false)
  const currentPage = request.data?.page ?? page
  useEffect(() => {
    if (
      !request.loading &&
      !request.error &&
      request.data?.page === page &&
      page > 1 &&
      request.data.items.length === 0
    )
      setPage(1, { replace: true })
  }, [request.loading, request.error, request.data, page, setPage])
  const visibleTasks = request.data?.items ?? []
  const pendingCounts = visibleTasks.some((task) => task.turn_count === null)
  const reload = request.reload
  useEffect(() => {
    if (!pendingCounts || request.loading || request.error) return
    const timer = window.setInterval(() => {
      if (document.visibilityState !== "hidden") reload()
    }, 5_000)
    return () => window.clearInterval(timer)
  }, [pendingCounts, request.loading, request.error, reload])
  const numberFormatter = useMemo(() => new Intl.NumberFormat(locale), [locale])
  const relativeFormatter = useMemo(
    () => new Intl.RelativeTimeFormat(locale, { numeric: "auto" }),
    [locale]
  )
  const dateFormatter = useMemo(
    () => new Intl.DateTimeFormat(locale, TABLE_DATE_TIME_OPTIONS),
    [locale]
  )
  const compactFormatter = useMemo(
    () =>
      new Intl.NumberFormat(locale, {
        notation: "compact",
        maximumFractionDigits: 1,
      }),
    [locale]
  )
  const creditFormatter = useMemo(
    () => new Intl.NumberFormat(locale, { maximumFractionDigits: 2 }),
    [locale]
  )
  const formatCredits = (value?: string) =>
    value == null ? "—" : creditFormatter.format(Number(value))
  const formatDuration = (seconds?: number) => {
    if (seconds == null) return "—"
    if (seconds < 60) return t("pages.taskHistory.duration.seconds", { count: seconds })
    const minutes = Math.round(seconds / 60)
    if (minutes < 60) return t("pages.taskHistory.duration.minutes", { count: minutes })
    return t("pages.taskHistory.duration.hours", {
      hours: Math.floor(minutes / 60),
      minutes: minutes % 60,
    })
  }
  const statusItems: { value: StatusFilter; label: string }[] = [
    { value: "all", label: t("pages.taskHistory.filters.statusAll") },
    { value: "failed", label: t("pages.taskHistory.filters.statusFailed") },
  ]
  const expertItems = [
    { value: "", label: t("pages.taskHistory.filters.expertAll") },
    ...experts.map((expert) => ({ value: expert.id, label: expert.name })),
  ]
  const exportCsv = async () => {
    setExporting(true)
    try {
      const rows: TaskHistoryEntry[] = []
      let truncated = false
      for (let exportPage = 1; ; exportPage++) {
        const pageQuery = new URLSearchParams(filterParams)
        pageQuery.set("page", String(exportPage))
        pageQuery.set("page_size", String(EXPORT_PAGE_SIZE))
        const data = await api<TaskHistory>(
          `/api/admin/v1/statistics/history?${pageQuery}`
        )
        rows.push(...data.items)
        if (!data.has_next) break
        if (exportPage >= EXPORT_MAX_PAGES) {
          truncated = true
          break
        }
      }
      const cell = (value: unknown) => {
        const text = value == null ? "" : String(value)
        return /[",\n\r]/.test(text) ? `"${text.replaceAll('"', '""')}"` : text
      }
      const header = [
        t("pages.taskHistory.columns.startedAt"),
        t("pages.taskHistory.columns.user"),
        "Email",
        t("pages.taskHistory.columns.task"),
        t("pages.taskHistory.columns.expert"),
        t("pages.taskHistory.columns.model"),
        t("pages.taskHistory.columns.conversationCount"),
        t("pages.taskHistory.columns.failedTurnsHeader"),
        `${t("pages.taskHistory.columns.tokens")} ${t("pages.taskHistory.columns.inputTokens")}`,
        `${t("pages.taskHistory.columns.tokens")} ${t("pages.taskHistory.columns.outputTokens")}`,
        t("pages.taskHistory.columns.credits"),
        `${t("pages.taskHistory.columns.activeTime")} (s)`,
        t("pages.taskHistory.columns.lastActiveAt"),
        "ID",
      ]
      const lines = [header, ...rows.map((row) => [
        dateFormatter.format(new Date(row.started_at)),
        row.user_name,
        row.user_email,
        row.title,
        row.expert_name ?? "",
        row.model_name ?? "",
        row.turn_count ?? "",
        row.failed_turns ?? "",
        row.input_tokens ?? "",
        row.output_tokens ?? "",
        row.credits ?? "",
        row.active_seconds ?? "",
        dateFormatter.format(new Date(row.last_active_at)),
        row.id,
      ])].map((line) => line.map(cell).join(","))
      const blob = new Blob(["\ufeff" + lines.join("\r\n")], {
        type: "text/csv;charset=utf-8",
      })
      const url = URL.createObjectURL(blob)
      const link = document.createElement("a")
      link.href = url
      link.download = `session-history-${new Date().toISOString().slice(0, 10)}.csv`
      link.click()
      URL.revokeObjectURL(url)
      if (truncated)
        showToast({
          status: "info",
          title: t("pages.taskHistory.filters.exportTruncated", {
            count: EXPORT_PAGE_SIZE * EXPORT_MAX_PAGES,
          }),
        })
    } catch (error) {
      showToast({
        status: "error",
        title: error instanceof Error ? error.message : String(error),
      })
    } finally {
      setExporting(false)
    }
  }

  const fieldOptions: { key: SearchField; label: string }[] = [
    { key: "taskName", label: t("pages.taskHistory.filters.taskName") },
    { key: "user", label: t("pages.billingDetails.filters.userPlaceholder") },
    { key: "expert", label: t("pages.taskHistory.filters.expert") },
    { key: "status", label: t("pages.taskHistory.filters.status") },
    {
      key: "dateRange",
      label: `${t("pages.taskHistory.filters.startTime")} – ${t("pages.taskHistory.filters.endTime")}`,
    },
  ]
  const updateTextFilterInput = (key: "taskName", value: string) => {
    setFilterInput((current) => ({ ...current, [key]: value }))
  }
  const applySearch = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (filterInput.userName.trim() && !filterInput.userId) {
      event.currentTarget.reportValidity()
      return
    }
    const next = {
      task: filterInput.taskName,
      user: "",
      user_id: filterInput.userId.trim(),
      expert_id: filterInput.expertId,
      status: filterInput.status === "failed" ? "failed" : "",
      from: queryDateValue(filterInput.startTime, filters.from, "from"),
      until: queryDateValue(
        filterInput.endTime,
        filters.until,
        "exclusiveUntil"
      ),
    }
    if (!tableQuery.commit(next)) {
      request.reload()
      summary.reload()
    }
  }
  const openAccount = (task: TaskHistoryEntry) => {
    if (!task.user_id) {
      showToast({
        status: "error",
        title: "会话历史缺少用户 ID，请更新后端服务",
      })
      return
    }
    setAccountUserID(task.user_id)
  }

  return (
    <section className="data-table-page-layout">
      <StatisticsFeedback {...request} />
      <div className="grid shrink-0 grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-6">
        {(
          [
            [Task01Icon, "sessions", summary.data && numberFormatter.format(summary.data.sessions)],
            [UserMultiple02Icon, "members", summary.data && numberFormatter.format(summary.data.members)],
            [AiChat02Icon, "turns", summary.data && numberFormatter.format(summary.data.turns)],
            [
              Alert02Icon,
              "failedSessions",
              summary.data &&
                `${numberFormatter.format(summary.data.failed_sessions)}${
                  summary.data.sessions
                    ? ` · ${Math.round((summary.data.failed_sessions * 100) / summary.data.sessions)}%`
                    : ""
                }`,
            ],
            [TokenCircleIcon, "tokens", summary.data && compactFormatter.format(summary.data.tokens)],
            [Coins01Icon, "credits", summary.data && formatCredits(summary.data.credits)],
          ] as [IconSvgElement, string, string | undefined][]
        ).map(([icon, key, value], index) => (
          <SummaryTile
            key={key}
            icon={icon}
            label={t(`pages.taskHistory.summary.${key}`)}
            qualifier={
              index === 0 && !filters.from
                ? t("pages.taskHistory.summary.last30Days")
                : undefined
            }
            value={summary.error ? "—" : value}
            loading={summary.loading && !summary.data}
            danger={key === "failedSessions" && !!summary.data?.failed_sessions}
          />
        ))}
      </div>
      <Card className="min-h-0 flex-1">
        <CardContent className="min-h-0 flex-1 gap-4 px-0">
          <form onSubmit={applySearch} className="px-[var(--card-spacing)]">
            <FieldGroup className="flex-row flex-wrap items-end gap-3">
              {visibleFields.has("taskName") && (
                <Field className="sm:w-44">
                  <FieldLabel
                    htmlFor="task-history-task-name"
                    className="sr-only"
                  >
                    {t("pages.taskHistory.filters.taskName")}
                  </FieldLabel>
                  <Input
                    id="task-history-task-name"
                    value={filterInput.taskName}
                    onChange={(event) =>
                      updateTextFilterInput("taskName", event.target.value)
                    }
                    placeholder={t(
                      "pages.taskHistory.filters.taskNamePlaceholder"
                    )}
                  />
                </Field>
              )}
              {visibleFields.has("user") && (
                <Field className="w-auto">
                  <FieldLabel htmlFor="task-history-user" className="sr-only">
                    {t("pages.billingDetails.filters.userPlaceholder")}
                  </FieldLabel>
                  <UserSearch
                    id="task-history-user"
                    selectionOnly
                    userID={filterInput.userId}
                    query={filterInput.userName}
                    onChange={({ userID, query }) =>
                      setFilterInput((current) => ({
                        ...current,
                        userId: userID,
                        userName: query,
                      }))
                    }
                  />
                </Field>
              )}
              {visibleFields.has("expert") && (
                <Field className="sm:w-44">
                  <FieldLabel className="sr-only">
                    {t("pages.taskHistory.filters.expert")}
                  </FieldLabel>
                  <Select
                    items={expertItems}
                    value={filterInput.expertId}
                    onValueChange={(value) =>
                      setFilterInput((current) => ({
                        ...current,
                        expertId: value ?? "",
                      }))
                    }
                  >
                    <SelectTrigger aria-label={t("pages.taskHistory.filters.expert")}>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent alignItemWithTrigger={false} align="start">
                      <SelectGroup>
                        {expertItems.map((item) => (
                          <SelectItem key={item.value} value={item.value}>
                            {item.label}
                          </SelectItem>
                        ))}
                      </SelectGroup>
                    </SelectContent>
                  </Select>
                </Field>
              )}
              {visibleFields.has("status") && (
                <Field className="sm:w-40">
                  <FieldLabel className="sr-only">
                    {t("pages.taskHistory.filters.status")}
                  </FieldLabel>
                  <Select
                    items={statusItems}
                    value={filterInput.status}
                    onValueChange={(value) =>
                      setFilterInput((current) => ({
                        ...current,
                        status: value === "failed" ? "failed" : "all",
                      }))
                    }
                  >
                    <SelectTrigger aria-label={t("pages.taskHistory.filters.status")}>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent alignItemWithTrigger={false} align="start">
                      <SelectGroup>
                        {statusItems.map((item) => (
                          <SelectItem key={item.value} value={item.value}>
                            {item.label}
                          </SelectItem>
                        ))}
                      </SelectGroup>
                    </SelectContent>
                  </Select>
                </Field>
              )}
              {visibleFields.has("dateRange") && (
                <>
                  <DatePickerField
                    id="task-history-start-time"
                    className="sm:w-40"
                    label={t("pages.taskHistory.filters.startTime")}
                    placeholder={t("pages.taskHistory.filters.startTime")}
                    locale={locale}
                    value={filterInput.startTime}
                    onChange={(value) =>
                      setFilterInput((current) => ({
                        ...current,
                        startTime: value,
                      }))
                    }
                    disabled={
                      filterInput.endTime
                        ? { after: filterInput.endTime }
                        : undefined
                    }
                  />
                  <DatePickerField
                    id="task-history-end-time"
                    className="sm:w-40"
                    label={t("pages.taskHistory.filters.endTime")}
                    placeholder={t("pages.taskHistory.filters.endTime")}
                    locale={locale}
                    value={filterInput.endTime}
                    onChange={(value) =>
                      setFilterInput((current) => ({
                        ...current,
                        endTime: value,
                      }))
                    }
                    disabled={
                      filterInput.startTime
                        ? { before: filterInput.startTime }
                        : undefined
                    }
                  />
                </>
              )}
              <DropdownMenu>
                <DropdownMenuTrigger
                  render={<Button type="button" variant="outline" />}
                >
                  <HugeiconsIcon
                    icon={FilterHorizontalIcon}
                    data-icon="inline-start"
                  />
                  {t("common.more")}
                </DropdownMenuTrigger>
                <DropdownMenuOptionsContent align="start">
                  <DropdownMenuGroup>
                    {fieldOptions.map(({ key, label }) => (
                      <DropdownMenuCheckboxItem
                        key={key}
                        checked={visibleFields.has(key)}
                        onCheckedChange={(checked) => {
                          setVisibleFields((current) => {
                            const next = new Set(current)
                            if (checked) next.add(key)
                            else next.delete(key)
                            return next
                          })
                          if (!checked && key === "user")
                            setFilterInput((current) => ({
                              ...current,
                              userId: "",
                              userName: "",
                            }))
                          if (!checked && key === "expert")
                            setFilterInput((current) => ({
                              ...current,
                              expertId: "",
                            }))
                          if (!checked && key === "status")
                            setFilterInput((current) => ({
                              ...current,
                              status: "all",
                            }))
                        }}
                      >
                        {label}
                      </DropdownMenuCheckboxItem>
                    ))}
                  </DropdownMenuGroup>
                </DropdownMenuOptionsContent>
              </DropdownMenu>
              <Field className="sm:w-auto">
                <FieldLabel className="sr-only">
                  {t("pages.taskHistory.filters.search")}
                </FieldLabel>
                <Button type="submit" className="w-full xl:w-auto">
                  <HugeiconsIcon icon={Search02Icon} data-icon="inline-start" />
                  {t("pages.taskHistory.filters.search")}
                </Button>
              </Field>
              <Button
                type="button"
                variant="outline"
                className="ms-auto"
                disabled={exporting}
                onClick={() => void exportCsv()}
              >
                <HugeiconsIcon icon={Download04Icon} data-icon="inline-start" />
                {exporting
                  ? t("pages.taskHistory.filters.exporting")
                  : t("pages.taskHistory.filters.export")}
              </Button>
            </FieldGroup>
          </form>
          <ScrollArea
            horizontal
            className="data-table-scroll"
          >
            <Table
              className={
                !visibleTasks.length
                  ? "h-full min-w-6xl table-fixed"
                  : "min-w-6xl table-fixed"
              }
              aria-label={t("pages.taskHistory.title")}
              aria-busy={request.loading}
            >
              <TableHeader className="sticky top-0 z-10">
                <TableRow>
                  <TableHead className="w-44 ps-[var(--card-spacing)]">
                    {t("pages.taskHistory.columns.startedAt")}
                  </TableHead>
                  <TableHead className="w-44">
                    {t("pages.taskHistory.columns.user")}
                  </TableHead>
                  <TableHead>{t("pages.taskHistory.columns.task")}</TableHead>
                  <TableHead className="w-28">
                    {t("pages.taskHistory.columns.conversationCount")}
                  </TableHead>
                  <TableHead className="w-24">
                    {t("pages.taskHistory.columns.tokens")}
                  </TableHead>
                  <TableHead className="w-24">
                    {t("pages.taskHistory.columns.credits")}
                  </TableHead>
                  <TableHead className="w-28">
                    {t("pages.taskHistory.columns.activeTime")}
                  </TableHead>
                  <TableHead className="w-36 pe-[var(--card-spacing)]">
                    {t("pages.taskHistory.columns.lastActiveAt")}
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody
                className={!visibleTasks.length ? "h-full" : undefined}
              >
                {visibleTasks.length > 0 ? (
                  visibleTasks.map((task) => (
                    <TableRow key={task.id}>
                      <TableCell className="ps-[var(--card-spacing)] text-muted-foreground">
                        <span
                          className="block truncate"
                          title={dateFormatter.format(
                            new Date(task.started_at)
                          )}
                        >
                          {dateFormatter.format(new Date(task.started_at))}
                        </span>
                      </TableCell>
                      <TableCell>
                        <button
                          type="button"
                          className="flex w-full min-w-0 cursor-pointer items-center gap-2 rounded-sm text-left underline-offset-2 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring [&:enabled:hover_span]:underline"
                          title={task.user_email}
                          onClick={() => openAccount(task)}
                        >
                          <UserAvatar src={task.user_avatar_url} />
                          <span className="min-w-0 truncate">
                            {task.user_name}
                          </span>
                        </button>
                      </TableCell>
                      <TableCell>
                        <Link
                          to={`${CONSOLE_ROUTES.conversationHistory}?task_id=${encodeURIComponent(task.id)}`}
                          className="block min-w-0 cursor-pointer truncate rounded-sm underline-offset-2 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring [&:hover]:underline"
                          title={task.title}
                        >
                          {task.title || "—"}
                        </Link>
                        {(task.expert_name || task.model_name) && (
                          <span
                            className="mt-0.5 block truncate text-xs text-muted-foreground"
                            title={[task.expert_name, task.model_name].filter(Boolean).join(" · ")}
                          >
                            {[task.expert_name, task.model_name]
                              .filter(Boolean)
                              .join(" · ")}
                          </span>
                        )}
                      </TableCell>
                      <TableCell className="tabular-nums">
                        <span className="block truncate">
                          {task.turn_count == null
                            ? "—"
                            : numberFormatter.format(task.turn_count)}
                        </span>
                        {!!task.failed_turns && (
                          <span className="block truncate text-xs text-destructive">
                            {t("pages.taskHistory.columns.failedTurns", {
                              count: task.failed_turns,
                            })}
                          </span>
                        )}
                      </TableCell>
                      <TableCell
                        className="truncate tabular-nums"
                        title={
                          task.input_tokens == null
                            ? undefined
                            : `${t("pages.taskHistory.columns.inputTokens")} ${numberFormatter.format(task.input_tokens)} / ${t("pages.taskHistory.columns.outputTokens")} ${numberFormatter.format(task.output_tokens ?? 0)}`
                        }
                      >
                        {task.input_tokens == null
                          ? "—"
                          : compactFormatter.format(
                              task.input_tokens + (task.output_tokens ?? 0)
                            )}
                      </TableCell>
                      <TableCell className="truncate tabular-nums">
                        {formatCredits(task.credits)}
                      </TableCell>
                      <TableCell className="truncate tabular-nums text-muted-foreground">
                        {formatDuration(task.active_seconds)}
                      </TableCell>
                      <TableCell
                        className="whitespace-nowrap pe-[var(--card-spacing)] text-muted-foreground"
                        title={dateFormatter.format(
                          new Date(task.last_active_at)
                        )}
                      >
                        <span className="block truncate">
                          {formatLastActivity(
                            task.last_active_at,
                            now,
                            locale,
                            relativeFormatter
                          )}
                        </span>
                      </TableCell>
                    </TableRow>
                  ))
                ) : (
                  <TableRow className="h-full">
                    <TableCell
                      colSpan={8}
                      className="h-full text-center text-muted-foreground"
                    >
                      {request.loading ? (
                        <TableLoading />
                      ) : request.error ? (
                        "—"
                      ) : (
                        t("pages.taskHistory.empty")
                      )}
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </ScrollArea>
          <TablePagination
            page={currentPage}
            pageSize={pageSize}
            itemCount={visibleTasks.length}
            hasNext={request.data?.has_next ?? false}
            loading={request.loading}
            error={Boolean(request.error)}
            onPageSizeChange={(size) => {
              setPageSize(size)
            }}
            onPrevious={() => setPage(Math.max(1, page - 1))}
            onNext={() => setPage(page + 1)}
          />
        </CardContent>
      </Card>
      {accountUserID && (
        <AccountDialog
          key={accountUserID}
          userId={accountUserID}
          onClose={() => setAccountUserID(null)}
        />
      )}
    </section>
  )
}

function SummaryTile({
  icon,
  label,
  qualifier,
  value,
  loading,
  danger,
}: {
  icon: IconSvgElement
  label: string
  qualifier?: string
  value?: string
  loading: boolean
  danger?: boolean
}) {
  return (
    <Card className="gap-1.5 bg-muted/40 p-2 shadow-none ring-1 ring-border/70">
      <StatisticsCardHeader icon={icon} title={label} qualifier={qualifier} truncateTitle />
      <CardContent
        className="min-h-12 flex-row items-center rounded-xl border border-border/60 bg-card px-4 py-2.5"
        aria-busy={loading}
      >
        <p
          className={
            "min-w-0 truncate text-xl font-medium tabular-nums tracking-tight" +
            (danger ? " text-destructive" : "")
          }
          title={value}
        >
          {loading ? "…" : (value ?? "—")}
        </p>
      </CardContent>
    </Card>
  )
}
