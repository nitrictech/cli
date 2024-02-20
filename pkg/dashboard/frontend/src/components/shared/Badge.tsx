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
        status === 'red' && 'bg-red-100 text-red-800',
        status === 'green' && 'bg-green-100 text-green-800',
        status === 'yellow' && 'bg-yellow-100 text-yellow-800',
        status === 'orange' && 'bg-orange-100 text-orange-800',
        status === 'blue' && 'bg-blue-100 text-blue-800',
        status === 'default' && 'bg-gray-100 text-gray-800',
        className,
      )}
      {...rest}
    >
      {children}
    </span>
  )
}

export default Badge
