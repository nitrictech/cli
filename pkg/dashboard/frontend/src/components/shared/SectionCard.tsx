import { cn } from '@/lib/utils'
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '../ui/card'

interface SectionCardProps {
  title?: string
  description?: string
  children: React.ReactNode
  className?: string
  innerClassName?: string
  headerClassName?: string
  headerSiblings?: React.ReactNode
  footer?: React.ReactNode
}

const SectionCard = ({
  title,
  description,
  children,
  className,
  innerClassName,
  headerClassName,
  headerSiblings,
  footer,
}: SectionCardProps) => {
  return (
    <Card className={cn('px-4 py-5 sm:p-6', className)}>
      {title && (
        <CardHeader className={cn('relative mb-6 p-0', headerClassName)}>
          <div className="flex flex-row items-center justify-between">
            <CardTitle className={'text-xl font-semibold leading-6'}>
              {title}
            </CardTitle>
            {headerSiblings}
          </div>
          {description && <CardDescription>{description}</CardDescription>}
        </CardHeader>
      )}
      <CardContent className={cn('p-0', innerClassName)}>
        {children}
      </CardContent>
      {footer && <CardFooter className="mt-6 p-0">{footer}</CardFooter>}
    </Card>
  )
}

export default SectionCard
