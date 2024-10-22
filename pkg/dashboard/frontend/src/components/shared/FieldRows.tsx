import { ExclamationCircleIcon, XMarkIcon } from '@heroicons/react/20/solid'
import { cn } from '@/lib/utils'
import React, { useEffect, useId } from 'react'
import { Input } from '../ui/input'
import { Label } from '../ui/label'

export interface FieldRow {
  key: string
  value: string
}

interface Props {
  testId: string
  rows: FieldRow[]
  lockKeys?: boolean
  readOnly?: boolean
  canClearRow?: boolean
  valueRequired?: boolean
  valueErrors?: Record<number, FieldRow>
  setRows: (value: FieldRow[]) => void
}

const FieldRows: React.FC<Props> = ({
  testId,
  rows,
  lockKeys,
  readOnly,
  setRows,
  valueErrors,
  valueRequired,
  canClearRow = true,
}) => {
  const id = useId()

  useEffect(() => {
    if (
      !lockKeys &&
      (rows[rows.length - 1].key || rows[rows.length - 1].value)
    ) {
      setRows([
        ...rows,
        {
          key: '',
          value: '',
        },
      ])
    }
  }, [rows])

  return (
    <ul className="divide-y divide-gray-200">
      {rows.map((r, i) => {
        const keyId = `${id}-${i}-key`
        const valueId = `${id}-${i}-value`
        const valueHasError = Boolean(valueErrors && valueErrors[i])

        return (
          <li
            key={i}
            className="group relative grid grid-cols-2 items-center gap-4 py-4"
          >
            <div>
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
            <div className="pr-8">
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
            {canClearRow && (
              <button
                type="button"
                onClick={() => {
                  const newArray = [...rows]
                  newArray.splice(i, 1)
                  setRows(newArray)
                }}
                className={cn(
                  'absolute right-0 hidden rounded-full bg-gray-600 p-1 text-white shadow-sm hover:bg-blue-500 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-600',
                  rows.length > 1 && (r.key || r.value)
                    ? 'group-hover:block'
                    : '',
                )}
              >
                <XMarkIcon className="h-5 w-5" aria-hidden="true" />
              </button>
            )}
          </li>
        )
      })}
    </ul>
  )
}

export default FieldRows
