import type { FieldRow } from "../../components/shared";
import type { Endpoint } from "../../types";

export function generatePath(
  endpoint: Endpoint,
  pathParams: FieldRow[],
  queryParams: FieldRow[]
) {
  let transformedPath = endpoint.path;

  pathParams.forEach((p) => {
    transformedPath = transformedPath.replace(
      `{${p.key}}`,
      encodeURIComponent(p.value)
    );
  });

  if (queryParams.length) {
    const searchParams = new URLSearchParams();

    queryParams.forEach((param) => {
      if (param.key) {
        searchParams.append(param.key, param.value);
      }
    });

    const queryPath = searchParams.toString();

    if (queryPath) {
      transformedPath = `${transformedPath}?${queryPath.replace(/^(\?)/, "")}`;
    }
  }

  return transformedPath;
}
