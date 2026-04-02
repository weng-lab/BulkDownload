Prefer explicit Go over abstraction.

Inline code by default. Do not introduce tiny helpers unless they clearly remove repeated or hard-to-follow logic.

Optimize for local readability. Code should make sense in one pass.

Concrete structs first. Interfaces only at the consumer.

Handle errors explicitly at the call site.

Use synchronous code unless concurrency is required.

Prefer stdlib.

Before editing, check:
- Can I keep this inline?
- Does the naming match nearby code?
- Is this the smallest change that works?
