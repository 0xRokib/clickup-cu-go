# clickup-cu JavaScript version

This is the **JavaScript/Node.js branch** of the ClickUp CLI.

Branches:

| Branch | Version | Path |
|---|---|---|
| `main` | Go native binary version | `github.com/0xRokib/clickup-cu-go/tree/main` |
| `js-version` | JavaScript/npm + Pi prompt version | `github.com/0xRokib/clickup-cu-go/tree/js-version` |

Fast zero-dependency ClickUp CLI for daily task control from terminal or Pi.

```bash
cu today
cu start TASK_ID
cu stop
cu estimate TASK_ID 2h
cu done TASK_ID
```

`TASK_ID`, `PARENT_TASK_ID`, `USER_ID`, `WORKSPACE_ID`, and `LIST_ID` are placeholders. Replace them with your ClickUp values.

## Why

Use `cu` when opening ClickUp just to check tasks, start/stop timers, change status, or set estimates feels slow.

`cu` talks directly to the ClickUp REST API:

```text
Terminal/Pi -> cu -> ClickUp REST API
```

It is faster and simpler than using MCP for daily repeated actions. Use ClickUp UI or MCP for rare/complex operations.

## Install

Requires Node.js 18+.

```bash
npm install -g clickup-cu
cu help
```

Use global install. Local install inside another project usually is not needed.

If another system `cu` exists, check PATH:

```bash
command -v cu      # macOS/Linux
where.exe cu       # Windows
```

## Setup

Get a ClickUp API token:

```text
ClickUp -> Profile/avatar -> Settings -> Apps -> API Token -> Generate
```

Save it:

```bash
cu token pk_your_token_here
```

Then create config:

```bash
cu init
```

Files:

```text
~/.config/cu/token
~/.config/cu/config.json
```

Both are written with owner-only permissions (`0600`). Environment variables also work and override the token file:

```bash
export CLICKUP_API_TOKEN="pk_your_token_here"
# or
export CLICKUP_TOKEN="pk_your_token_here"
```

Example config:

```json
{
  "workspaceId": "YOUR_WORKSPACE_ID",
  "userId": "YOUR_USER_ID",
  "defaultListId": "YOUR_LIST_ID",
  "statuses": {
    "backlog": "BACKLOG",
    "todo": "TO-DO",
    "start": "IN PROGRESS",
    "review": "REVIEW",
    "qa": "QA/TESTING",
    "blocked": "BLOCKED",
    "release": "READY FOR RELEASE",
    "done": "DONE"
  }
}
```

## Daily workflow

```bash
cu today                         # dashboard: timer + due/active/blocked/open tasks
cu list                          # list your open tasks
cu list all                      # list open tasks without assignee filter
cu list status "IN PROGRESS"     # list by status
cu show TASK_ID                  # show task details

cu start TASK_ID                 # set IN PROGRESS + start timer
cu progress TASK_ID              # set IN PROGRESS only
cu stop                          # stop current timer
cu done TASK_ID                  # set DONE
```

Status shortcuts:

```bash
cu backlog TASK_ID
cu todo TASK_ID
cu review TASK_ID
cu qa TASK_ID
cu testing TASK_ID
cu blocked TASK_ID
cu release TASK_ID
```

Time:

```bash
cu estimate TASK_ID 2h
cu estimate TASK_ID 2h 30m
cu estimate TASK_ID 45m
cu addtime TASK_ID 45m "backend work"
```

Create/update:

```bash
cu create "Fix login bug"
cu subtask PARENT_TASK_ID "Add API validation"
cu assign TASK_ID me
cu assign TASK_ID USER_ID ANOTHER_USER_ID
```

## Commands

| Command | Purpose |
|---|---|
| `cu today` | Daily dashboard |
| `cu list [all]` | List tasks |
| `cu list status "STATUS"` | List tasks by status |
| `cu show TASK_ID` | Show one task |
| `cu start TASK_ID` | Set start status + start timer |
| `cu stop` | Stop current timer |
| `cu backlog/todo/progress/review/qa/testing/blocked/release/done TASK_ID` | Change status |
| `cu estimate TASK_ID DURATION` | Set estimate |
| `cu addtime TASK_ID DURATION [note]` | Add manual time |
| `cu create "TITLE"` | Create task in default list |
| `cu subtask PARENT_TASK_ID "TITLE"` | Create subtask |
| `cu assign TASK_ID USER_ID_OR_ME...` | Add assignees |
| `cu token pk_your_token_here` | Save token |
| `cu init` | Create/show config |
| `cu help` | Show help |

Durations support `2h`, `30m`, `2h 30m`. Plain numbers mean minutes.

## Pi shortcut

This package includes `/cu-fast` for Pi.

```bash
pi install npm:clickup-cu
```

Restart Pi or run:

```text
/reload
```

Use:

```text
/cu-fast
/cu-fast today
/cu-fast start TASK_ID
/cu-fast stop
/cu-fast show TASK_ID
/cu-fast estimate TASK_ID 2h
/cu-fast done TASK_ID
```

`/cu-fast` runs local `cu` through bash, so install and configure the CLI too:

```bash
npm install -g clickup-cu
cu token pk_your_token_here
cu init
```

## REST mapping

| CLI action | REST API |
|---|---|
| `cu today` timer | `GET /team/{workspaceId}/time_entries/current` |
| `cu today/list` tasks | `GET /team/{workspaceId}/task?...` |
| `cu show TASK_ID` | `GET /task/{taskId}` |
| status commands | `PUT /task/{taskId}` |
| `cu start TASK_ID` timer | `POST /team/{workspaceId}/time_entries/start` |
| `cu stop` | `POST /team/{workspaceId}/time_entries/stop` |
| `cu estimate TASK_ID DURATION` | `PUT /task/{taskId}` with `time_estimate` |
| `cu addtime TASK_ID DURATION` | `POST /team/{workspaceId}/time_entries` |
| `cu create "TITLE"` | `POST /list/{defaultListId}/task` |
| `cu subtask PARENT_TASK_ID "TITLE"` | `POST /list/{listId}/task` with `parent` |
| `cu assign TASK_ID USER_ID_OR_ME...` | `POST /task/{taskId}/assignee` |

## Development

```bash
git clone https://github.com/YOUR_USER/cu.git
cd cu
npm install -g .
npm test
npm pack --dry-run
```

Publish:

```bash
npm version patch
npm publish --access public
```

## Security

- Never commit `~/.config/cu/token`.
- Never paste `pk_...` in chat, issues, logs, screenshots, or recordings.
- Each teammate should generate their own token.
- Rotate exposed tokens.
