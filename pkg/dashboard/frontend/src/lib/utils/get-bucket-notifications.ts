import type { Bucket, Notification } from '@/types'

export const getBucketNotifications = (
  bucket: Bucket,
  notifications: Notification[],
) => notifications.filter((n) => n.bucket === bucket.name)
