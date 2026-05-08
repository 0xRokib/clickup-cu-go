# cu

Fast ClickUp CLI for team daily workflow. Written in Go. One native binary.

```bash
cu today
cu start TASK_ID
cu stop
cu estimate TASK_ID 2h
cu done TASK_ID
```

## Quick install for team

### Option 1: install with Go

Requires Go 1.22+.

```bash
go install github.com/0xRokib/clickup-cu-go/cmd/cu@latest
cu help
```

If `cu` is not found, add Go bin to your shell:

```bash
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.zshrc
source ~/.zshrc
cu help
```

For bash users, use `~/.bashrc` instead of `~/.zshrc`.

### Option 2: build from repo

```bash
git clone git@github.com:0xRokib/clickup-cu-go.git
cd clickup-cu-go
go build -o cu ./cmd/cu
sudo mv cu /usr/local/bin/cu
cu help
```

Update later:

```bash
cd clickup-cu-go
git pull
go build -o cu ./cmd/cu
sudo mv cu /usr/local/bin/cu
```

## First-time setup

### 1. Get ClickUp token

ClickUp:

```text
Profile/avatar -> Settings -> Apps -> API Token -> Generate
```

### 2. Save token

```bash
cu token pk_your_token_here
```

This creates:

```text
~/.config/cu/token
```

You can also use env var instead:

```bash
export CLICKUP_API_TOKEN="pk_your_token_here"
```

### 3. Create config

```bash
cu init
```

This creates:

```text
~/.config/cu/config.json
```

If auto-discovery misses anything, edit the file:

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

Status names must match your ClickUp workspace exactly.

## Daily commands

```bash
cu today                         # dashboard: timer + tasks
cu list                          # my open tasks
cu list all                      # all open tasks in workspace
cu list status "IN PROGRESS"     # tasks by status
cu show TASK_ID                  # task details

cu start TASK_ID                 # set IN PROGRESS + start timer
cu progress TASK_ID              # set IN PROGRESS only
cu stop                          # stop current timer
cu done TASK_ID                  # set DONE
```

## Status shortcuts

```bash
cu backlog TASK_ID
cu todo TASK_ID
cu review TASK_ID
cu qa TASK_ID
cu testing TASK_ID
cu blocked TASK_ID
cu release TASK_ID
cu done TASK_ID
```

## Time commands

```bash
cu estimate TASK_ID 2h
cu estimate TASK_ID 2h 30m
cu estimate TASK_ID 45m
cu estimate TASK_ID 45        # plain number = minutes

cu addtime TASK_ID 45m "backend work"
cu addtime TASK_ID 2h 30m "debugging"
```

## Create/update commands

```bash
cu create "Fix login bug"
cu subtask PARENT_TASK_ID "Add API validation"
cu assign TASK_ID me
cu assign TASK_ID USER_ID ANOTHER_USER_ID
```

`me` uses `userId` from config.

## All commands

| Command | Purpose |
|---|---|
| `cu today` | Daily dashboard |
| `cu list [all]` | List tasks |
| `cu list status "STATUS"` | List tasks by status |
| `cu show TASK_ID` | Show task details |
| `cu start TASK_ID` | Set start status + start timer |
| `cu stop` | Stop timer |
| `cu progress TASK_ID` | Set start status without timer |
| `cu backlog/todo/review/qa/testing/blocked/release/done TASK_ID` | Change status |
| `cu estimate TASK_ID DURATION` | Set estimate |
| `cu addtime TASK_ID DURATION [note]` | Add manual time |
| `cu create "TITLE"` | Create task in default list |
| `cu subtask PARENT_TASK_ID "TITLE"` | Create subtask |
| `cu assign TASK_ID USER_ID_OR_ME...` | Add assignees |
| `cu token pk_xxx` | Save token |
| `cu init` | Create/show config |
| `cu help` | Show help |

## Team troubleshooting

### `cu: command not found`

Check install path:

```bash
which cu
```

If installed with Go, add Go bin to `PATH`:

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

### `missing ClickUp token`

```bash
cu token pk_your_token_here
```

### `missing workspaceId` or `missing userId`

```bash
cu init
```

Then edit:

```text
~/.config/cu/config.json
```

### Status update fails

Your team workspace may use different status names. Fix `statuses` in config.

### `cu create` fails

Set `defaultListId` in config.

## Development

```bash
go test ./...
go run ./cmd/cu help
go build -o cu ./cmd/cu
gofmt -w cmd/cu/*.go
```

## Security

- Never commit or share `~/.config/cu/token`.
- Never paste `pk_...` tokens in chat, issues, logs, screenshots, or recordings.
- Each teammate should create their own token.
- Rotate exposed tokens immediately.

## License

MIT
