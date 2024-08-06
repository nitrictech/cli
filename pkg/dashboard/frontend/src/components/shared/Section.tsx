import { cn } from '@/lib/utils'

interface SectionProps {
  title?: string
  children: React.ReactNode
  innerClassName?: string
  headerClassName?: string
  headerSiblings?: React.ReactNode
}

const Section = ({
  title,
  children,
  innerClassName,
  headerClassName,
  headerSiblings,
}: SectionProps) => {
  return (
    <section className="rounded-lg bg-white shadow">
      <div className={cn('px-4 py-5 sm:p-6', innerClassName)}>
        <div className="sm:flex sm:items-start sm:justify-between">
          <div className="relative w-full">
            {title && (
              <div
                className={cn(
                  'mb-4 flex items-center justify-between',
                  headerClassName,
                )}
              >
                <h3 className={'text-xl font-semibold leading-6 text-gray-900'}>
                  {title}
                </h3>
                {headerSiblings}
              </div>
            )}
            {children}
          </div>
        </div>
      </div>
    </section>
  )
}

export default Section
