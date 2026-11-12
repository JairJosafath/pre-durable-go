// Turns an execution history into what the dev panel draws: the operations
// as an indented tree, and a time axis to place their bars on.

// A durable execution is milliseconds of work and, sometimes, hours of
// waiting. On a true time axis the work would be invisible. So every stretch
// between two moments counts for at most this long; anything longer is drawn
// as a break in the axis.
const LONGEST_STRETCH_MS = 1200;
// A stretch always gets a little room, so a 5 ms step is still a visible bar.
const SHORTEST_STRETCH_MS = 60;

const time = (value) => (value ? new Date(value).getTime() : null);

/** Operations in tree order, each with its depth below the handler. */
export function tree(operations) {
  const ids = new Set(operations.map((operation) => operation.id));
  const children = new Map();

  for (const operation of operations) {
    // An operation whose parent is not in the list hangs under the handler.
    const parent = ids.has(operation.parentId) ? operation.parentId : "";
    children.set(parent, [...(children.get(parent) || []), operation]);
  }

  const rows = [];
  const walk = (parent, depth) => {
    for (const operation of children.get(parent) || []) {
      rows.push({ ...operation, depth });
      walk(operation.id, depth + 1);
    }
  };
  walk("", 0);

  return rows;
}

/**
 * A scale from a moment to a position between 0 and 1, with long idle
 * stretches squeezed. `breaks` are the positions where time was squeezed out.
 */
export function axis(history, now) {
  const moments = new Set();
  const add = (value) => value != null && moments.add(value);

  add(time(history.execution?.startedAt));
  add(time(history.execution?.endedAt));
  for (const span of [...(history.operations || []), ...(history.invocations || [])]) {
    add(time(span.startedAt));
    add(time(span.endedAt));
  }
  if (!history.execution?.endedAt) add(now);

  const sorted = [...moments].sort((a, b) => a - b);
  const positions = [0];
  const squeezed = [];

  for (let index = 1; index < sorted.length; index++) {
    const gap = sorted[index] - sorted[index - 1];
    const drawn = Math.min(Math.max(gap, SHORTEST_STRETCH_MS), LONGEST_STRETCH_MS);
    if (gap > LONGEST_STRETCH_MS * 2) squeezed.push({ index, gap });
    positions.push(positions[index - 1] + drawn);
  }

  const total = positions[positions.length - 1] || 1;

  const at = (moment) => {
    if (moment == null || sorted.length === 0) return 0;
    if (moment <= sorted[0]) return 0;
    if (moment >= sorted[sorted.length - 1]) return 1;

    let index = 1;
    while (sorted[index] < moment) index++;

    const share = (moment - sorted[index - 1]) / (sorted[index] - sorted[index - 1]);

    return (positions[index - 1] + share * (positions[index] - positions[index - 1])) / total;
  };

  return {
    // The bar of a span, as left edge and width in percent.
    bar(span) {
      const start = at(time(span.startedAt));
      const end = at(time(span.endedAt) ?? now);

      return { left: start * 100, width: Math.max(0, end - start) * 100 };
    },
    breaks: squeezed.map(({ index, gap }) => ({
      at: ((positions[index - 1] + positions[index]) / 2 / total) * 100,
      gap,
    })),
  };
}

/** How long a span took, or has been taking. */
export function elapsed(span, now) {
  const start = time(span.startedAt);
  if (start == null) return null;

  return (time(span.endedAt) ?? now) - start;
}

/** The operations that are being worked on right now, innermost only. */
export function current(operations) {
  const open = operations.filter((operation) => operation.status === "RUNNING" || operation.status === "RETRYING");
  const parents = new Set(open.map((operation) => operation.parentId));

  return open.filter((operation) => !parents.has(operation.id));
}
