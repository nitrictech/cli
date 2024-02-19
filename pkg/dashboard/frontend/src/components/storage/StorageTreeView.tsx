import { type FC, useMemo } from 'react'
import TreeView, { type TreeItemType } from '../shared/TreeView'
import type { TreeItem, TreeItemIndex } from 'react-complex-tree'
import type { Bucket, Notification } from '@/types'
import { Tooltip, TooltipContent, TooltipTrigger } from '../ui/tooltip'
import { Badge } from '../ui/badge'
import { cn } from '@/lib/utils/cn'
import { getBucketNotifications } from '@/lib/utils/get-bucket-notifications'

export type StorageTreeItemType = TreeItemType<Bucket>

interface Props {
  buckets: Bucket[]
  notifications: Notification[]
  onSelect: (bucket: Bucket) => void
  initialItem: Bucket
}

const StorageTreeView: FC<Props> = ({
  buckets,
  notifications,
  onSelect,
  initialItem,
}) => {
  const treeItems: Record<
    TreeItemIndex,
    TreeItem<StorageTreeItemType>
  > = useMemo(() => {
    const rootItem: TreeItem = {
      index: 'root',
      isFolder: true,
      children: [],
      data: null,
    }

    const rootItems: Record<TreeItemIndex, TreeItem<StorageTreeItemType>> = {
      root: rootItem,
    }

    for (const bucket of buckets) {
      // add api if not added already
      rootItems[bucket.name] = {
        index: bucket.name,
        data: {
          label: bucket.name,
          data: bucket,
        },
      }

      rootItem.children!.push(bucket.name)
    }

    return rootItems
  }, [buckets])

  return (
    <TreeView<StorageTreeItemType>
      label="Buckets"
      initialItem={initialItem.name}
      items={treeItems}
      getItemTitle={(item) => item.data.label}
      onPrimaryAction={(items) => {
        if (items.data.data) {
          onSelect(items.data.data)
        }
      }}
      renderItemTitle={({ item }) => {
        const count = item.data.data
          ? getBucketNotifications(item.data.data, notifications).length
          : 0

        return (
          <span className="truncate">
            {count ? (
              <>
                {item.data.label}
                <Tooltip>
                  <TooltipTrigger asChild>
                    <span>
                      <Badge
                        className={cn(
                          'ml-2',
                          count > 0 ? 'bg-blue-600' : 'bg-orange-400',
                        )}
                      >
                        {count}
                      </Badge>
                    </span>
                  </TooltipTrigger>
                  <TooltipContent side="right">
                    <p>{count} notifications to this bucket</p>
                  </TooltipContent>
                </Tooltip>
              </>
            ) : (
              item.data.label
            )}
          </span>
        )
      }}
    />
  )
}

export default StorageTreeView
