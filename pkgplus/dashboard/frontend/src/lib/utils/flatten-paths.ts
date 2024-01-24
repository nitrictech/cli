import type { OpenAPIV3 } from "openapi-types";
import type { Endpoint, Method, Param } from "../../types";
import { uniqBy } from "../utils";

export function flattenPaths(doc: OpenAPIV3.Document): Endpoint[] {
  const uniquePaths: Record<string, Endpoint> = {};
  const params: Param[] = [];

  Object.entries(doc.paths).forEach(([path, pathItem]) => {
    Object.entries(pathItem as any).forEach(([method, value]) => {
      if (method === "parameters") {
        params.push({
          path,
          value: value as OpenAPIV3.ParameterObject[],
        });
        return;
      }

      method = method.toUpperCase();
      const key = `${doc.info.title}-${path}-${method}`;
      const endpoint: Endpoint = {
        id: key,
        api: doc.info.title,
        path,
        method: method as Method,
        doc,
      };

      uniquePaths[key] = endpoint;
    });
  });

  return uniqBy(
    Object.entries(uniquePaths).map(([_, value]) => {
      const param = params.find((param) => param.path == value.path);

      if (param) {
        value.params = value.params ? [...value.params, param] : [param];
      }

      return value;
    }),
    "id"
  );
}
