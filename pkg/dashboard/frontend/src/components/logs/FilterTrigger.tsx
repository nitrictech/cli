import React from 'react'
import { useSidebar } from '../ui/sidebar'
import { Button } from '../ui/button'
import { FunnelIcon } from '@heroicons/react/24/outline'

const FilterTrigger: React.FC = () => {
  const { toggleSidebar } = useSidebar()

  return (
    <Button
      data-testid="filter-logs-btn"
      variant="outline"
      onClick={toggleSidebar}
      size="icon"
      title="Toggle filters"
    >
      <FunnelIcon className="h-5 w-5 text-gray-500" />
    </Button>
  )
}

export default FilterTrigger
