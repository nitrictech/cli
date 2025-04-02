import { ExclamationCircleIcon, XMarkIcon } from '@heroicons/react/20/solid'
import { cn } from '@/lib/utils'
import React, { useId } from 'react'
import { Input } from '../ui/input'
import { Label } from '../ui/label'
import { Button } from '../ui/button'

export interface FieldRow {
  key: string
  value: string
}

interface Props {
  testId: string
  rows: FieldRow[]
  lockKeys?: boolean
  readOnly?: boolean
  valueRequired?: boolean
  valueErrors?: Record<number, FieldRow>
  setRows: (value: FieldRow[]) => void
  addRowLabel?: string
}

const FieldRows: React.FC<Props> = ({
  testId,
  rows,
  lockKeys,
  readOnly,
  setRows,
  valueErrors,
  valueRequired,
  addRowLabel = 'Add Row',
}) => {
  const id = useId()

  return (
    <div>
      <ul className="divide-y divide-border">
        {rows.map((r, i) => {
          const keyId = `${id}-${i}-key`
          const valueId = `${id}-${i}-value`
          const valueHasError = Boolean(valueErrors && valueErrors[i])

          return (
            <li key={i} className="group flex items-center gap-4 py-4">
              <div className="w-full">
                <Label htmlFor={keyId} className="sr-only">
                  Key
                </Label>
                <div className="mt-2 sm:col-span-2 sm:mt-0">
                  <Input
                    type="text"
                    data-testid={`${testId}-${i}-key`}
                    readOnly={lockKeys || readOnly}
                    placeholder="Key"
                    className="read-only:opacity-100"
                    onChange={(e) => {
                      const updatedRow: FieldRow = { ...r, key: e.target.value }
                      const newArr = [...rows]

                      newArr[i] = updatedRow

                      setRows(newArr)
                    }}
                    value={r.key}
                    name={keyId}
                    id={keyId}
                  />
                </div>
              </div>
              <div className={cn('w-full', lockKeys && 'mr-11')}>
                <Label htmlFor={valueId} className="sr-only">
                  {r.value}
                </Label>
                <div className="relative mt-2 sm:col-span-2 sm:mt-0">
                  <Input
                    type="text"
                    placeholder="Value"
                    readOnly={readOnly}
                    data-testid={`${testId}-${i}-value`}
                    onChange={(e) => {
                      const updatedRow: FieldRow = {
                        ...r,
                        value: e.target.value,
                      }
                      const newArr = [...rows]

                      newArr[i] = updatedRow

                      setRows(newArr)
                    }}
                    required={valueRequired}
                    name={valueId}
                    id={valueId}
                    value={r.value}
                    className={cn(
                      valueHasError &&
                        'text-red-900 !ring-red-500 placeholder:text-red-300',
                    )}
                  />
                  {valueHasError && (
                    <div
                      data-testid={`${testId}-${i}-value-error-icon`}
                      className="pointer-events-none absolute inset-y-0 right-0 flex items-center pr-3"
                    >
                      <ExclamationCircleIcon
                        className="h-5 w-5 text-red-500"
                        aria-hidden="true"
                      />
                    </div>
                  )}
                </div>
              </div>
              {!lockKeys && (
                <button
                  type="button"
                  onClick={() => {
                    const newArray = [...rows]
                    newArray.splice(i, 1)
                    setRows(newArray)
                  }}
                  aria-label="Remove row"
                  className={cn(
                    'rounded-full bg-gray-600 p-1 text-white shadow-sm hover:bg-blue-500 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-600',
                    'flex items-center opacity-30 transition-all group-hover:opacity-100',
                  )}
                >
                  <XMarkIcon className="h-5 w-5" aria-hidden="true" />
                </button>
              )}
            </li>
          )
        })}
        {rows.length === 0 && (
          <li className="mt-1 text-sm text-foreground">
            No rows to display. Click &apos;{addRowLabel}&apos; to begin adding
            data.
          </li>
        )}
      </ul>
      {!lockKeys && (
        <Button
          onClick={() => {
            setRows([
              ...rows,
              {
                key: '',
                value: '',
              },
            ])
          }}
          data-testid="add-row-btn"
          className="ml-auto mt-4 flex"
        >
          {addRowLabel}
        </Button>
      )}
    </div>
  )
}

export default FieldRows
