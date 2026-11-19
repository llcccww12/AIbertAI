import { Loading03Icon } from "@hugeicons/core-free-icons"
import { HugeiconsIcon, type IconSvgElement } from "@hugeicons/react"
import { StatisticsCardHeader } from "@/components/statistics-card-header"
import { useTranslation } from "react-i18next"
import { Bar, BarChart, XAxis, YAxis } from "recharts"

import { Card, CardContent } from "@/components/ui/card"
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/components/ui/chart"

export type StatisticsTrendPoint = { date: string; count: number }

export function StatisticsTrendCard({
  icon,
  title,
  periodLabel,
  points,
  color,
  barClassName,
  loading,
  formatValue,
}: {
  icon: IconSvgElement
  title: string
  periodLabel: string
  points?: StatisticsTrendPoint[]
  color: string
  barClassName: string
  loading: boolean
  formatValue?: (value: number) => string
}) {
  const { t } = useTranslation()
  const config = { count: { label: title, color } } satisfies ChartConfig
  return (
    <Card
      className="min-w-0 gap-2 bg-muted/40 p-2 shadow-none ring-1 ring-border/70"
      role="figure"
      aria-label={title}
    >
      <StatisticsCardHeader icon={icon} title={title} qualifier={periodLabel} />
      <CardContent
        className="rounded-xl border border-border/60 bg-card p-4"
        aria-busy={loading}
      >
        {loading ? (
          <div className="flex h-32 items-center justify-center">
            <span
              role="status"
              aria-label={t("statistics.loading")}
              className="inline-flex items-center justify-center text-muted-foreground"
            >
              <HugeiconsIcon
                icon={Loading03Icon}
                className="animate-spin motion-reduce:animate-none"
                size={20}
                strokeWidth={1.75}
                aria-hidden="true"
              />
            </span>
          </div>
        ) : points ? (
          <ChartContainer className="h-32 w-full" config={config}>
            <BarChart
              accessibilityLayer
              data={points}
              margin={{ left: 2, right: 2, top: 8, bottom: 0 }}
            >
              <XAxis dataKey="date" hide />
              <YAxis
                hide
                domain={([min, max]: readonly [number, number]) =>
                  min === 0 && max === 0
                    ? [0, 1]
                    : [Math.min(0, min), Math.max(0, max)]
                }
              />
              <ChartTooltip
                cursor={false}
                content={
                  <ChartTooltipContent
                    valueFormatter={
                      formatValue
                        ? (value) =>
                            typeof value === "number"
                              ? formatValue(value)
                              : String(value)
                        : undefined
                    }
                  />
                }
              />
              <Bar
                dataKey="count"
                fill="var(--color-count)"
                className={barClassName}
                maxBarSize={8}
                minPointSize={3}
                radius={[2, 2, 0, 0]}
                isAnimationActive={false}
              />
            </BarChart>
          </ChartContainer>
        ) : (
          <div className="flex h-32 items-center justify-center text-2xl text-muted-foreground">
            —
          </div>
        )}
      </CardContent>
    </Card>
  )
}
