import type { FieldRow } from "../../components/shared";

export function generatePath(
  path: string,
  pathParams: FieldRow[],
  queryParams: FieldRow[]
) {
  pathParams.forEach((p) => {
    path = path.replace(`{${p.key}}`, encodeURIComponent(p.value));
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
      path = `${path}?${queryPath.replace(/^(\?)/, "")}`;
    }
  }

  return path;
}
