import type {
  EventHistoryItem,
  EventResource,
  TopicHistoryItem,
} from '../../types'
import { formatJSON } from '@/lib/utils'
import CodeEditor from '../apis/CodeEditor'
import HistoryAccordion from '../shared/HistoryAccordion'

interface Props {
  history: EventHistoryItem[]
  selectedWorker: EventResource
  workerType: 'schedules' | 'topics' | 'jobs'
}

const EventsHistory: React.FC<Props> = ({
  selectedWorker,
  workerType,
  history,
}) => {
  const requestHistory = history
    .sort((a, b) => b.time - a.time)
    .filter((h) => h.event)
    .filter((h) => h.event.name === selectedWorker.name)

  if (!requestHistory.length) {
    return <p>There is no history.</p>
  }

  return (
    <div className="pb-10">
      <HistoryAccordion
        items={requestHistory.map((h) => {
          let payload = ''

          if (workerType === 'topics' || workerType === 'jobs') {
            payload = (h.event as TopicHistoryItem['event']).payload
          }

          const formattedPayload = payload ? formatJSON(payload) : ''

          return {
            label: h.event.name,
            time: h.time,
            success: Boolean(h.event.success),
            content: formattedPayload ? (
              <div className="flex flex-col gap-8">
                <div className="flex flex-col gap-2">
                  <p className="text-md font-semibold">Payload</p>
                  <CodeEditor
                    contentType="application/json"
                    readOnly={true}
                    value={formattedPayload}
                    title="Payload"
                  />
                </div>
              </div>
            ) : undefined,
          }
        })}
      />
    </div>
  )
}

export default EventsHistory
