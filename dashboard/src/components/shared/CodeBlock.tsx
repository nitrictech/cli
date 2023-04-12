import classNames from "classnames";
import React from "react";
import Highlight, { defaultProps, Language } from "prism-react-renderer";
//import theme from "prism-react-renderer/themes/nightOwl";
import {
  ClipboardDocumentCheckIcon,
  ClipboardIcon,
} from "@heroicons/react/20/solid";

interface Props extends React.ComponentProps<"div"> {
  className?: string;
  enableCopy?: boolean;
  language?: Language;
}

const copyToClipboard = (str: string, callback = () => {}) => {
  const focused = window.document.hasFocus();
  if (focused) {
    window.navigator?.clipboard?.writeText(str).then(() => {
      callback();
    });
  } else {
    console.warn("Unable to copy to clipboard");
  }
};

const isPlainToken = (content: string) => /[a-zA-Z0-9]/.test(content.trim());

export const CodeBlock: React.FC<React.PropsWithChildren<Props>> = ({
  className = "",
  enableCopy = true,
  children,
  language = "typescript",
  ...props
}) => {
  const [copied, setCopied] = React.useState(false);
  const timeoutRef = React.useRef<any>();
  language = language || (className.replace(/language-/, "") as Language);

  const handleCopyCode = () => {
    copyToClipboard(`${children}`.trim());
    setCopied(true);

    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current);
    }

    timeoutRef.current = setTimeout(() => {
      setCopied(false);
      timeoutRef.current = null;
    }, 1000);
  };

  return (
    <div
      className={classNames(
        "bg-gray-800 rounded-xl w-full text-neutral-content p-5",
        className
      )}
      {...props}
    >
      <Highlight
        {...defaultProps}
        code={`${children}`.trim()}
        language={language}
        // theme={theme}
      >
        {({ tokens, getLineProps, getTokenProps, className }) => (
          <div className='w-full relative'>
            <pre
              className={classNames(
                className,
                "flex overflow-x-auto flex-col py-4"
              )}
            >
              {enableCopy ? (
                <button
                  aria-label='Copy Code'
                  className='w-4 h-4 absolute top-0 text-white right-0'
                  onClick={handleCopyCode}
                >
                  {copied ? <ClipboardDocumentCheckIcon /> : <ClipboardIcon />}
                </button>
              ) : null}
              {tokens.map((line, i) => {
                let insideObject = false;
                const { className, ...rest } = getLineProps({ line, key: i });

                return (
                  <span key={i} className={className} {...rest}>
                    <span>
                      {line.map((token, key, tokens) => {
                        const classes: string[] = [];

                        if (token.content.trim() === "{") {
                          insideObject = true;
                        }

                        if (
                          language === "yaml" &&
                          token.types.includes("punctuation") &&
                          token.content.trim() !== ":" &&
                          key > 0 &&
                          isPlainToken(tokens[key - 1]?.content) &&
                          !insideObject
                        ) {
                          classes.push("yaml-colored-punct");
                        }

                        const { className, ...tokenProps } = getTokenProps({
                          token,
                          key,
                        });

                        classes.unshift(className);

                        return (
                          <span
                            key={key}
                            className={classes.join(" ")}
                            {...tokenProps}
                          />
                        );
                      })}
                    </span>
                  </span>
                );
              })}
            </pre>
          </div>
        )}
      </Highlight>
    </div>
  );
};
