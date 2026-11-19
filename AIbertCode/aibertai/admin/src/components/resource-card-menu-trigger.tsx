import type { ComponentProps } from "react"
import { MoreHorizontalIcon } from "@hugeicons/core-free-icons"
import { HugeiconsIcon } from "@hugeicons/react"

import { Button } from "@/components/ui/button"
import { DropdownMenuTrigger } from "@/components/ui/dropdown-menu"

export function ResourceCardMenuTrigger({
  label,
  onClick,
  strokeWidth,
}: {
  label: string
  onClick?: ComponentProps<typeof Button>["onClick"]
  strokeWidth?: number
}) {
  return (
    <DropdownMenuTrigger
      render={
        <Button type="button" variant="ghost" size="icon-sm" aria-label={label} onClick={onClick} />
      }
    >
      <HugeiconsIcon icon={MoreHorizontalIcon} strokeWidth={strokeWidth} />
    </DropdownMenuTrigger>
  )
}
