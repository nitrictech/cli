import Lottie from "lottie-react";
import loadingAnim from "./loading.animation.json";
import {
  PropsWithChildren,
  Suspense,
  useEffect,
  useRef,
  useState,
} from "react";
import { cn } from "@/lib/utils";

interface Props extends PropsWithChildren {
  delay: number;
  conditionToShow: boolean;
  className?: string;
}

const Loading: React.FC<Props> = ({
  delay,
  conditionToShow,
  className,
  children,
}) => {
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

  const Loader = (
    <div
      role="status"
      className="w-full h-full flex flex-col items-center justify-center"
    >
      <div className={cn("w-40 my-20", className)}>
        <Lottie
          initialSegment={[40, 149]}
          loop
          autoplay
          animationData={loadingAnim}
        />
      </div>
      <span className="sr-only">Loading...</span>
    </div>
  );

  return showLoader || !conditionToShow ? (
    Loader
  ) : (
    <Suspense fallback={Loader}>{children}</Suspense>
  );
};

export default Loading;
