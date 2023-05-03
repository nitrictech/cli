import Lottie from "lottie-react";
import loadingAnim from "./loading.animation.json";
import { PropsWithChildren, useEffect, useRef, useState } from "react";

interface Props extends PropsWithChildren {
  delay: number;
  conditionToShow: boolean;
}

const Loading: React.FC<Props> = ({ delay, conditionToShow, children }) => {
  const timeoutRef = useRef<NodeJS.Timeout>();
  const [showLoader, setShowLoader] = useState(true);

  useEffect(() => {
    if (delay) {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }

      timeoutRef.current = setTimeout(() => {
        setShowLoader(false);
      }, delay);
    }

    return () => clearTimeout(timeoutRef.current);
  }, [delay]);

  return showLoader || !conditionToShow ? (
    <div
      role="status"
      className="w-full h-full flex flex-col items-center justify-center"
    >
      <div className="w-40 mb-20">
        <Lottie
          initialSegment={[40, 149]}
          loop
          autoplay
          animationData={loadingAnim}
        />
      </div>
      <span className="sr-only">Loading...</span>
    </div>
  ) : (
    <>{children}</>
  );
};

export default Loading;
