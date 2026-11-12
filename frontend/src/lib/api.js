// The API URL is the ApiUrl output of the CDK stack.
const baseUrl = (process.env.NEXT_PUBLIC_ENDPOINT || "").replace(/\/+$/, "");

/** Calls the PrimeDrive API as the signed-in user. */
export async function api(path, { token, method = "GET", body, accept } = {}) {
  const headers = {};
  if (token) headers.Authorization = `Bearer ${token}`;
  if (accept) headers.Accept = accept;
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

/** Reads a DynamoDB file item, as returned by GET /files. */
export function toFile(item) {
  return {
    key: item.key.S,
    title: fileName(item.key.S),
    status: item.status?.S || "",
    // The run changes every time the same file name is uploaded again.
    run: item.run?.S || "",
    execution: item.execution?.S || "",
    labels: (item.labels?.L || []).map((label) => label.M.name.S),
    reasons: (item.review?.M?.reasons?.L || []).map((reason) => reason.S),
    note: item.note?.S || "",
  };
}
