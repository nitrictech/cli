import {
  Panel,
  useReactFlow,
  getRectOfNodes,
  getTransformForBounds,
} from "reactflow";
import { toPng } from "html-to-image";
import { Button } from "../ui/button";

const filter = (node: HTMLElement) => {
  return !node.classList?.contains("nitric-remove-on-share");
};

function downloadImage(projectName: string, dataUrl: string) {
  const a = document.createElement("a");

  a.setAttribute("download", `${projectName}.png`);
  a.setAttribute("href", dataUrl);
  a.click();
  a.remove();
}

const imageWidth = 1024;
const imageHeight = 768;

function ShareButton({ projectName }: { projectName: string }) {
  const { getNodes } = useReactFlow();
  const onClick = async () => {
    // we calculate a transform for the nodes so that all nodes are visible
    // we then overwrite the transform of the `.react-flow__viewport` element
    // with the style option of the html-to-image library
    const nodesBounds = getRectOfNodes(getNodes());
    const transform = getTransformForBounds(
      nodesBounds,
      imageWidth,
      imageHeight,
      0.5,
      2
    );

    const el = document.querySelector(".react-flow__viewport");

    if (el) {
      const dataUrl = await toPng(el as HTMLElement, {
        backgroundColor: "#fff",
        width: imageWidth,
        height: imageHeight,
        style: {
          width: `${imageWidth}px`,
          height: `${imageHeight}px`,
          transform: `translate(${transform[0]}px, ${transform[1]}px) scale(${transform[2]})`,
        },
        filter,
      });

      downloadImage(projectName, dataUrl);
    }
  };

  return (
    <Panel position="top-right">
      <Button onClick={onClick}>Share</Button>
    </Panel>
  );
}

export default ShareButton;
