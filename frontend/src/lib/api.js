// The API URL is the ApiUrl output of the CDK stack.
const baseUrl = (process.env.NEXT_PUBLIC_ENDPOINT || "").replace(/\/+$/, "");

/** Calls the PrimeDrive API as the signed-in user. */
export async function api(path, { token, method = "GET", body } = {}) {
  const headers = {};
  if (token) headers.Authorization = `Bearer ${token}`;
  if (body) headers["Content-Type"] = "application/json";

  return fetch(`${baseUrl}${path}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  });
}

/** File name of an object key of the form users/<sub>/<name>. */
export function fileName(key) {
  return key.split("/").slice(2).join("/");
}
