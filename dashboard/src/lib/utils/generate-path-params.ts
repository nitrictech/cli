import type { APIRequest, Endpoint } from "../../types";
import type { FieldRow } from "../shared/FieldRows";

export const generatePathParams = (endpoint: Endpoint, request: APIRequest) => {
  const pathParams: FieldRow[] = [];

  if (endpoint.params?.length) {
    endpoint.params.forEach((p) => {
      p.value.forEach((v) => {
        if (v.in === "path") {
          const existing = request.pathParams.find((pp) => pp.key === v.name);

          pathParams.push({
            key: v.name,
            value: existing?.value || "",
          });
        }
      });
    });
  }

  return pathParams;
};
