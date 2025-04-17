import { type PropsWithChildren, Children } from 'react'
import { Separator } from '../ui/separator'
import { cn } from '@/lib/utils'

interface Props extends PropsWithChildren {
  className?: string
}

const BreadCrumbs = ({ children, className }: Props) => {
  const childArray = Children.toArray(children)

  return (
    <nav
      aria-label="breadcrumb"
      className={cn('flex items-center gap-4', className)}
    >
      <ol className="flex gap-4">
        {childArray.map((child, index) => (
          <li key={index} className={'flex items-center justify-center gap-4'}>
            {index === childArray.length - 1 ? (
              <span className="text-foreground">{child}</span>
            ) : (
              <>
                <span className="text-muted-foreground hover:text-foreground">{child}</span>
                <Separator orientation="vertical" />
              </>
            )}
          </li>
        ))}
      </ol>
    </nav>
  )
}

export default BreadCrumbs
