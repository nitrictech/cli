import type { ApiHistoryItem } from '../../types'
import { formatJSON } from '../../lib/utils'
import { useState } from 'react'
import { Tabs } from '../shared'
import CodeEditor from './CodeEditor'
import APIResponseContent from './APIResponseContent'
import TableGroup from '../shared/TableGroup'
import HistoryAccordion from '../shared/HistoryAccordion'

interface Props {
  history: ApiHistoryItem[]
  selectedRequest: {
    method: string
    path: string
  }
}

const checkEquivalentPaths = (matcher: string, path: string): boolean => {
  // If the paths are equal regardless of query params
  if (path.split('?').length > 1 && matcher.split('?').length > 1) {
    return path.split('?')[0] === matcher.split('?')[0]
  }

  const regex = matcher.replace(/{(.*)}/, '(.*)')
  return path.match(regex) !== null
}

const APIHistory: React.FC<Props> = ({ history, selectedRequest }) => {
  const requestHistory = history
    .sort((a, b) => b.time - a.time)
    .filter(({ event }) => event.request && event.response)
    .filter(({ event }) =>
      checkEquivalentPaths(
        selectedRequest.path ?? '',
        event.request.path ?? '',
      ),
    )
    .filter(({ event }) => event.request.method === selectedRequest.method)

  if (!requestHistory.length) {
    return <p>There is no history.</p>
  }

  return (
    <HistoryAccordion
      items={requestHistory.map((h) => ({
        label: h.event.api + h.event.request.path,
        time: h.time,
        status: h.event?.response?.status,
        content: <ApiHistoryAccordionContent {...h} />,
      }))}
    />
  )
}

function isJSON(data: string | undefined) {
  if (!data) return false
  try {
    JSON.parse(data)
  } catch (e) {
    return false
  }
  return true
}

const ApiHistoryAccordionContent: React.FC<ApiHistoryItem> = ({
  event: { request, response },
}) => {
  const [tabIndex, setTabIndex] = useState(0)

  const isJson = isJSON(atob(request.body?.toString() ?? ''))

  const tabs = [{ name: 'Headers' }, { name: 'Response' }]

  const jsonTabs = [...tabs, { name: 'Payload' }]

  return (
    <div className="flex flex-col py-4">
      <div className="bg-white sm:rounded-lg">
        <Tabs
          tabs={isJson ? jsonTabs : tabs}
          index={tabIndex}
          setIndex={setTabIndex}
        />
        <div className="py-5">
          {tabIndex === 0 && (
            <TableGroup
              headers={['Key', 'Value']}
              rowDataClassName="max-w-[100px]"
              groups={[
                {
                  name: 'Request Headers',
                  rows: Object.entries(request.headers)
                    .filter(([key, value]) => key && value)
                    .map(([key, value]) => [
                      key.toLowerCase(),
                      value.join(', '),
                    ]),
                },
                {
                  name: 'Response Headers',
                  rows: Object.entries(response.headers ?? [])
                    .filter(([key, value]) => key && value)
                    .map(([key, value]) => [key.toLowerCase(), value]),
                },
              ]}
            />
          )}
          {tabIndex === 1 && (
            <div className="flex flex-col gap-8">
              <div className="flex flex-col gap-2">
                <p className="text-md font-semibold">Response Data</p>
                <APIResponseContent
                  response={{ ...response, data: atob(response.data) }}
                />
              </div>
            </div>
          )}
          {tabIndex === 2 && (
            <div className="flex flex-col gap-8">
              <div className="flex flex-col gap-2">
                <p className="text-md font-semibold">Request Body</p>
                <CodeEditor
                  contentType="application/json"
                  readOnly={true}
                  value={formatJSON(
                    JSON.parse(atob(request.body?.toString() ?? '')),
                  )}
                  title="Request Body"
                />
              </div>
              <div className="flex flex-col gap-2">
                {request.queryParams && (
                  <TableGroup
                    headers={['Key', 'Value']}
                    rowDataClassName="max-w-[100px]"
                    groups={[
                      {
                        name: 'Query Params',
                        rows: request.queryParams
                          .filter(({ key, value }) => key && value)
                          .map(({ key, value }) => [key, value]),
                      },
                    ]}
                  />
                )}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

export default APIHistory
