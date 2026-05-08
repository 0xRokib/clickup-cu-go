---
description: Fast ClickUp workflow using local cu command
argument-hint: "[today|list|show|start|stop|done|estimate|addtime|create|subtask|assign] [args]"
---
Run the local `cu` CLI command for this ClickUp request: $ARGUMENTS

Rules:
- Use bash, not MCP, unless `cu` reports unsupported command.
- Command format: `cu $ARGUMENTS`.
- If no args, run `cu today`.
- Return the command output concisely.
- If token missing, tell user to set `CLICKUP_API_TOKEN`.

Examples:
- `/cu-fast` -> `cu today`
- `/cu-fast start 86exgrxf2` -> `cu start 86exgrxf2`
- `/cu-fast stop` -> `cu stop`
- `/cu-fast estimate 86exgrxf2 2h` -> `cu estimate 86exgrxf2 2h`
