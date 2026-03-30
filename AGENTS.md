Prefer clear, explicit Go over DRY abstractions. Repeat yourself if it makes the code easier to read top-to-bottom. A little duplication is far cheaper than the wrong abstraction.

Do not define interfaces preemptively. Define interfaces at the consumer, not the producer. A concrete struct is the default.

Handle errors explicitly at each call site. Do not write generic error-handling wrappers or must()-style panic helpers. Wrap errors with fmt.Errorf("doing X: %w", err).

Do not introduce goroutines or channels unless the task is genuinely concurrent. A synchronous for-loop is almost always the right first answer.

Prefer stdlib (net/http, encoding/json, log/slog) over third-party packages unless there's a concrete reason.
