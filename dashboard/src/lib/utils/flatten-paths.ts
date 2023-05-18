import type { OpenAPIV3 } from "openapi-types";
import type { Endpoint, Method, Param } from "../../types";
import { sortedUniq, uniqBy } from "../utils";

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

      const key = `${doc.info.title}-${path}`;
      const endpoint: Endpoint = {
        id: key,
        api: doc.info.title,
        path,
        methods: [method.toUpperCase() as Method],
        doc,
      };

      if (!uniquePaths[key]) {
        uniquePaths[key] = endpoint;
      } else {
        uniquePaths[key] = {
          ...uniquePaths[key],
          methods: sortedUniq([
            ...uniquePaths[key].methods,
            ...endpoint.methods,
          ]),
        };
      }
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
