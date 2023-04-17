import CodeMirror, {
  ReactCodeMirrorProps,
  ReactCodeMirrorRef,
} from "@uiw/react-codemirror";
import { StreamLanguage } from "@codemirror/language";
import { parse } from "@prantlf/jsonlint";
import { linter, Diagnostic } from "@codemirror/lint";
import { useMemo, useRef, useState } from "react";
import { XCircleIcon } from "@heroicons/react/20/solid";
import { json } from "@codemirror/lang-json";
import { html } from "@codemirror/lang-html";
import { css } from "@codemirror/lang-css";
import { xml } from "@codemirror/lang-xml";
import { spreadsheet } from "@codemirror/legacy-modes/mode/spreadsheet";
import type { EditorView } from "@codemirror/view";

interface Props extends ReactCodeMirrorProps {
  contentType: string;
  includeLinters?: boolean;
}

function getLineNumber(str: string, index: number) {
  let lineNumber = 1;
  for (let i = 0; i < index; i++) {
    if (str[i] === "\n") {
      lineNumber++;
    }
  }
  return lineNumber;
}

function getErrorPosition(
  text: string,
  lineNum: number,
  colNum: number
): number {
  const lines = text.split("\n");
  const lineIdx = lineNum - 1;
  const line = lines[lineIdx].substring(0, colNum);
  const prevLinesLength = lines.slice(0, lineIdx).join("\n").length;
  return prevLinesLength + line.length;
}

// Define a custom JSON linter function that uses jsonlint
const jsonLinter = (view: EditorView): Diagnostic[] => {
  const errors: Diagnostic[] = [];
  const value = view.state.doc.toString();

  if (!value.trim()) return [];

  try {
    parse(value, {
      allowDuplicateObjectKeys: false,
      allowSingleQuotedStrings: false,
    });
  } catch (e: any) {
    const errorLocation = e.message.match(/line (\d+), column (\d+)/);
    const lineNum = parseInt(errorLocation[1], 10);
    const colNum = parseInt(errorLocation[2], 10);
    const pos = getErrorPosition(value, lineNum, colNum);

    return [
      {
        from: pos,
        message: e.reason,
        severity: "error",
        to: pos,
        actions: [],
      },
    ];
  }

  return errors;
};

const CodeEditor: React.FC<Props> = ({
  contentType,
  readOnly,
  includeLinters,
  onChange,
  ...props
}) => {
  const editor = useRef<ReactCodeMirrorRef>(null);
  const [errors, setErrors] = useState<Diagnostic[]>([]);

  const extensions = useMemo(() => {
    switch (contentType) {
      case "text/html":
        return [html()];
      case "text/csv":
        return [StreamLanguage.define(spreadsheet)];
      case "text/css":
        return [css()];
      case "text/xml":
        return [xml()];
      case "application/json":
        return includeLinters ? [json(), linter(jsonLinter)] : [json()];
    }

    return [];
  }, [contentType]);

  const handleOnChange: ReactCodeMirrorProps["onChange"] = (
    value,
    viewUpdate
  ) => {
    if (typeof onChange === "function") {
      // check validate
      if (includeLinters && contentType === "application/json") {
        const errors = jsonLinter(viewUpdate.view);

        setErrors(errors);

        if (errors.length) {
          return; // don't update state if in error
        }
      }

      onChange(value, viewUpdate);
    }
  };

  const handleFormat = () => {
    if (editor.current?.view && props.value) {
      const { view } = editor.current;

      view.dispatch({
        changes: {
          from: 0,
          to: view.state.doc.length,
          insert: JSON.stringify(
            JSON.parse(view.state.doc.toString()),
            null,
            2
          ),
        },
      });
    }
  };

  return (
    <div className='rounded-lg relative overflow-hidden'>
      {!readOnly && contentType === "application/json" && (
        <div className='flex mb-2'>
          <h4 className='text-lg font-medium text-gray-900'>JSON Content</h4>
          <button
            onClick={handleFormat}
            type='button'
            className='rounded ml-auto bg-white px-2 py-1 text-xs font-semibold text-gray-900 shadow-sm ring-1 ring-inset ring-gray-300 hover:bg-gray-50'
          >
            Format
          </button>
        </div>
      )}
      <CodeMirror
        ref={editor}
        height='350px'
        theme='dark'
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
        <div className='rounded-md bottom-0 right-0 m-2 absolute bg-red-50 p-2.5'>
          <div className='flex items-center'>
            <div className='flex-shrink-0'>
              <XCircleIcon
                className='h-5 w-5 text-red-400'
                aria-hidden='true'
              />
            </div>
            <div className='ml-1'>
              <div className='text-sm text-red-700'>
                Error Invalid JSON at line{" "}
                {getLineNumber(
                  editor.current?.view?.state.doc.toString() || "",
                  errors[0].from
                )}
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default CodeEditor;
