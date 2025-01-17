import { cn } from '@/lib/utils'
import type {
  ForwardRefExoticComponent,
  InputHTMLAttributes,
  SVGProps,
} from 'react'
import { Input } from '../ui/input'
import { Label } from '../ui/label'

interface Props extends InputHTMLAttributes<HTMLInputElement> {
  label: string
  hideLabel?: boolean
  id: string
  icon?: ForwardRefExoticComponent<SVGProps<SVGSVGElement>>
}

const TextField = ({
  label,
  hideLabel,
  id,
  icon: Icon,
  ...inputProps
}: Props) => {
  return (
    <div className="w-full">
      <Label htmlFor={id} className={hideLabel ? 'sr-only' : undefined}>
        {label}
      </Label>
      <div
        className={cn('relative rounded-md shadow-sm', !hideLabel && 'mt-2')}
      >
        {Icon && (
          <div className="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3">
            <Icon className="h-5 w-5 text-gray-400" aria-hidden="true" />
          </div>
        )}
        <Input
          {...inputProps}
          id={id}
          className={cn(Icon && 'pl-10', inputProps.className)}
        />
      </div>
    </div>
  )
}

export default TextField
