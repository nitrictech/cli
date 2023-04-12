import {
  JSONEditor as VanillaJSONEditor,
  JSONEditorPropsOptional,
} from "vanilla-jsoneditor";
import { useEffect, useRef } from "react";

export interface JSONEditorProps extends JSONEditorPropsOptional {
  content: { json?: any; text?: string };
}

const JSONEditor: React.FC<JSONEditorProps> = (props: JSONEditorProps) => {
  const refContainer = useRef<HTMLDivElement>(null);
  const refEditor = useRef<VanillaJSONEditor | null>(null);

  useEffect(() => {
    if (typeof window !== "undefined" && refContainer.current) {
      refEditor.current = new VanillaJSONEditor({
        target: refContainer.current,
        props: {
          mode: "text" as any,
          mainMenuBar: false,
          navigationBar: false,
          statusBar: false,
        },
      });
    }

    return () => {
      // destroy editor
      if (refEditor.current) {
        refEditor.current.destroy();
        refEditor.current = null;
      }
    };
  }, []);

  // update props
  useEffect(() => {
    if (refEditor.current) {
      refEditor.current.updateProps(props);
    }
  }, [props]);

  const handleFormat = async () => {
    if (refEditor.current) {
      const content = refEditor.current.get() as { text: string };
      const text = content.text;

      const newContent = {
        text: JSON.stringify(JSON.parse(text), null, 2),
      };

      await refEditor.current.update(newContent);
    }
  };

  return (
    <div className='relative flex flex-1 flex-col gap-2'>
      <div className='flex'>
        <h4 className='text-lg font-medium text-gray-900'>JSON Content</h4>
        <button
          onClick={handleFormat}
          type='button'
          className='rounded ml-auto bg-white px-2 py-1 text-xs font-semibold text-gray-900 shadow-sm ring-1 ring-inset ring-gray-300 hover:bg-gray-50'
        >
          Format
        </button>
      </div>
      <div ref={refContainer} />
    </div>
  );
};

export default JSONEditor;
