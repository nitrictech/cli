import type { Topic, Subscriber } from '@/types'

export const getTopicSubscriptions = (
  topic: Topic,
  subscriptions: Subscriber[],
) => subscriptions.filter((n) => n.topic === topic.name)
