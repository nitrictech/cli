import CodeMirror, {
  type ReactCodeMirrorProps,
  type ReactCodeMirrorRef,
} from '@uiw/react-codemirror'
import { StreamLanguage } from '@codemirror/language'
import { parse } from '@prantlf/jsonlint'
import { linter, type Diagnostic } from '@codemirror/lint'
import { useEffect, useMemo, useRef, useState } from 'react'
import {
  ClipboardDocumentCheckIcon,
  ClipboardIcon,
  XCircleIcon,
} from '@heroicons/react/20/solid'
import { json } from '@codemirror/lang-json'
import { html } from '@codemirror/lang-html'
import { css } from '@codemirror/lang-css'
import { xml } from '@codemirror/lang-xml'
import { spreadsheet } from '@codemirror/legacy-modes/mode/spreadsheet'
import type { EditorView } from '@codemirror/view'
import { copyToClipboard } from '../../lib/utils/copy-to-clipboard'
import { formatJSON } from '../../lib/utils'

interface Props extends ReactCodeMirrorProps {
  contentType: string
  includeLinters?: boolean
  enableCopy?: boolean
}

function getLineNumber(str: string, index: number) {
  let lineNumber = 1
  for (let i = 0; i < index; i++) {
    if (str[i] === '\n') {
      lineNumber++
    }
  }
  return lineNumber
}

function getErrorPosition(
  text: string,
  lineNum: number,
  colNum: number,
): number {
  const lines = text.split('\n')
  const lineIdx = lineNum - 1
  const line = lines[lineIdx].substring(0, colNum)
  const prevLinesLength = lines.slice(0, lineIdx).join('\n').length
  return prevLinesLength + line.length
}

// Define a custom JSON linter function that uses jsonlint
const jsonLinter = (view: EditorView): Diagnostic[] => {
  const errors: Diagnostic[] = []
  const value = view.state.doc.toString()

  if (!value.trim()) return []

  try {
    parse(value, {
      allowDuplicateObjectKeys: false,
      allowSingleQuotedStrings: false,
    })
  } catch (e: any) {
    const errorLocation = e.message.match(/line (\d+), column (\d+)/)
    const lineNum = parseInt(errorLocation[1], 10)
    const colNum = parseInt(errorLocation[2], 10)
    const pos = getErrorPosition(value, lineNum, colNum)

    return [
      {
        from: pos,
        message: e.reason,
        severity: 'error',
        to: pos,
        actions: [],
      },
    ]
  }

  return errors
}

const CodeEditor: React.FC<Props> = ({
  contentType,
  content,
  readOnly,
  includeLinters,
  onChange,
  enableCopy,
  ...props
}) => {
  const editor = useRef<ReactCodeMirrorRef>(null)
  const [errors, setErrors] = useState<Diagnostic[]>([])

  const [copied, setCopied] = useState(false)
  const timeoutRef = useRef<any>()

  const handleCopyCode = () => {
    copyToClipboard(`${props.value}`.trim())
    setCopied(true)

    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current)
    }

    timeoutRef.current = setTimeout(() => {
      setCopied(false)
      timeoutRef.current = null
    }, 1000)
  }

  const extensions = useMemo(() => {
    switch (contentType) {
      case 'text/html':
        return [html()]
      case 'text/csv':
        return [StreamLanguage.define(spreadsheet)]
      case 'text/css':
        return [css()]
      case 'text/xml':
      case 'application/xml':
        return [xml()]
      case 'application/json':
        return includeLinters ? [json(), linter(jsonLinter)] : [json()]
    }

    return []
  }, [contentType])

  const handleOnChange: ReactCodeMirrorProps['onChange'] = (
    value,
    viewUpdate,
  ) => {
    if (typeof onChange === 'function') {
      // check validate
      if (includeLinters && contentType === 'application/json') {
        const errors = jsonLinter(viewUpdate.view)

        setErrors(errors)

        if (errors.length) {
          return // don't update state if in error
        }
      }

      onChange(value, viewUpdate)
    }
  }

  const handleFormat = () => {
    if (editor.current?.view && props.value) {
      const { view } = editor.current

      view.dispatch({
        changes: {
          from: 0,
          to: view.state.doc.length,
          insert: formatJSON(view.state.doc.toString()),
        },
      })
    }
  }

  return (
    <div className="relative">
      {enableCopy ? (
        <button
          aria-label="Copy Code"
          data-testid="copy-code"
          className="absolute right-0 top-0 z-50 m-4 h-4 w-4 text-white"
          onClick={handleCopyCode}
        >
          {copied ? <ClipboardDocumentCheckIcon /> : <ClipboardIcon />}
        </button>
      ) : null}
      {!readOnly && contentType === 'application/json' && (
        <div className="flex rounded-lg py-2">
          <button
            onClick={handleFormat}
            type="button"
            className="ml-auto rounded bg-white px-2 py-1 text-xs font-semibold text-gray-900 shadow-sm ring-1 ring-inset ring-gray-300 hover:bg-gray-50"
          >
            Format
          </button>
        </div>
      )}
      <div className="overflow-hidden rounded-lg">
        <CodeMirror
          ref={editor}
          height="350px"
          theme="dark"
          basicSetup={{
            foldGutter: true,
            lineNumbers: true,
          }}
          editable={!readOnly}
          readOnly={readOnly}
          extensions={extensions}
          onChange={handleOnChange}
          {...props}
        />
        {errors.length > 0 && (
          <div className="absolute bottom-0 right-0 m-2 rounded-md bg-red-50 p-2.5">
            <div className="flex items-center">
              <div className="flex-shrink-0">
                <XCircleIcon
                  className="h-5 w-5 text-red-400"
                  aria-hidden="true"
                />
              </div>
              <div className="ml-1">
                <div className="text-sm text-red-700">
                  Error Invalid JSON at line{' '}
                  {getLineNumber(
                    editor.current?.view?.state.doc.toString() || '',
                    errors[0].from,
                  )}
                </div>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

export default CodeEditor
