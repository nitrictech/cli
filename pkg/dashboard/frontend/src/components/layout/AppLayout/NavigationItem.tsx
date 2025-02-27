import { cn } from '@/lib/utils'
import React from 'react'

export interface NavigationItemProps {
  name: string
  href: string
  icon: React.ForwardRefExoticComponent<
    Omit<React.SVGProps<SVGSVGElement>, 'ref'> & {
      title?: string
      titleId?: string
    } & React.RefAttributes<SVGSVGElement>
  >
  onClick?: () => void
  routePath: string
}

const NavigationItem: React.FC<NavigationItemProps> = ({
  name,
  href,
  icon: Icon,
  onClick,
  routePath,
}) => {
  const isActive = href === routePath

  return (
    <li key={name}>
      <a
        href={href}
        onClick={onClick}
        aria-current={isActive}
        target={href.startsWith('http') ? '_blank' : undefined}
        rel={href.startsWith('http') ? 'noopener noreferrer' : undefined}
        className={cn(
          'group relative flex h-12 w-12 items-center gap-x-3 rounded-md p-3 text-sm font-semibold leading-6 transition-all group-data-[state=expanded]:w-full',
          isActive
            ? 'bg-accent text-accent-foreground'
            : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground', 
          'dark:hover:bg-accent' 
        )}
      >
        <Icon className="h-6 w-6 shrink-0" aria-hidden="true" />
        <span
          className={cn(
            'min-w-[120px] text-sm',
            'absolute left-7 group-data-[state=expanded]:left-12',
            'opacity-0 group-data-[state=expanded]:opacity-100',
            'transition-all',
          )}
        >
          {name}
        </span>
      </a>
    </li>
  )
}

export default NavigationItem
