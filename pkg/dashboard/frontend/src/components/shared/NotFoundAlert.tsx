import React, { type PropsWithChildren } from 'react'
import { Alert } from '../ui/alert'
import { ExclamationTriangleIcon } from '@heroicons/react/24/outline'

interface Props extends PropsWithChildren {
  className?: string
}

const NotFoundAlert: React.FC<Props> = ({ children, className }) => {
  return (
    <Alert variant="warning" className={className}>
      <div className="flex">
        <div className="flex-shrink-0">
          <ExclamationTriangleIcon className="h-5 w-5" aria-hidden="true" />
        </div>
        <div className="ml-3 flex-1 md:flex md:justify-between">
          <p className="text-sm leading-4">{children}</p>
        </div>
      </div>
    </Alert>
  )
}

export default NotFoundAlert
