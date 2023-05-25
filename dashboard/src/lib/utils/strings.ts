export const formatJSON = (
  value: any,
  space: string | number | undefined = 2
) => {
  try {
    if (typeof value === "string") {
      value = JSON.parse(value);
    }

    return JSON.stringify(value, null, space);
  } catch (e) {
    return value;
  }
};
