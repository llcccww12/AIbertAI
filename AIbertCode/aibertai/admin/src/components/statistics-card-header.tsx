import type { ReactNode } from "react"
import { HugeiconsIcon, type IconSvgElement } from "@hugeicons/react"

import { CardHeader, CardTitle } from "@/components/ui/card"
import { cn } from "@/lib/utils"

export function StatisticsCardHeader({
  icon,
  title,
  qualifier,
  truncateTitle = false,
}: {
  icon: IconSvgElement
  title: ReactNode
  qualifier?: ReactNode
  truncateTitle?: boolean
}) {
  return (
    <CardHeader className="flex min-h-7 flex-row items-center gap-3 px-3 py-0">
      <HugeiconsIcon
        icon={icon}
        className="shrink-0 text-muted-foreground"
        size={16}
        strokeWidth={1.75}
        aria-hidden={truncateTitle || undefined}
      />
      <CardTitle
        role={truncateTitle ? "heading" : undefined}
        aria-level={truncateTitle ? 2 : undefined}
        title={truncateTitle && typeof title === "string" ? title : undefined}
        className={cn("min-w-0 flex-1 font-mono text-sm font-medium uppercase tracking-wide text-muted-foreground", truncateTitle && "truncate")}
      >
        {title}
      </CardTitle>
      {qualifier && (
        <span className="ms-auto shrink-0 font-mono text-xs text-muted-foreground/70">
          {qualifier}
        </span>
      )}
    </CardHeader>
  )
}
