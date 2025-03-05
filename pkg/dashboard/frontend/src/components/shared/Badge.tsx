import { cn } from '@/lib/utils'
import type { PropsWithChildren } from 'react'

interface Props extends PropsWithChildren {
  status: 'red' | 'green' | 'yellow' | 'orange' | 'blue' | 'default'
  className?: string
}

const Badge: React.FC<Props> = ({
  status = 'default',
  className,
  children,
  ...rest
}) => {
  return (
    <span
      className={cn(
        'inline-flex items-center justify-center rounded-full px-2.5 py-0.5 text-xs font-medium',
        status === 'red' && 'bg-destructive/10 text-destructive',
        status === 'green' && 'bg-primary/10 text-primary',
        status === 'yellow' && 'bg-warning/10 text-warning',
        status === 'orange' && 'bg-orange-500/10 text-orange-500',
        status === 'blue' && 'bg-blue-500/10 text-blue-500',
        status === 'default' && 'bg-muted text-muted-foreground',
        className,
      )}
      {...rest}
    >
      {children}
    </span>
  )
}

export default Badge
