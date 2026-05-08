# cu Go version

Fast ClickUp CLI for team daily workflow. Written in Go. One native binary.

Branches:

| Branch | Version | Path |
|---|---|---|
| `main` | Go native binary version | `github.com/0xRokib/clickup-cu-go/tree/main` |
| `js-version` | JavaScript/npm + Pi prompt version | `github.com/0xRokib/clickup-cu-go/tree/js-version` |

Use the JS branch:

```bash
git clone git@github.com:0xRokib/clickup-cu-go.git
cd clickup-cu-go
git switch js-version
```

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
$(go env GOPATH)/bin/cu help
```

`go install` puts the binary here:

```bash
$(go env GOPATH)/bin/cu
```

On macOS/Linux, another system command named `cu` may already exist (often `/usr/bin/cu`). Put Go bin **first** in `PATH` so our ClickUp CLI wins.

For zsh/macOS default:

```bash
echo 'export PATH="$(go env GOPATH)/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
which -a cu
cu help
```

For bash/Linux:

```bash
echo 'export PATH="$(go env GOPATH)/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
which -a cu
cu help
```

Expected first path:

```text
/Users/YOUR_NAME/go/bin/cu      # macOS default
/home/YOUR_NAME/go/bin/cu       # Linux default
```

### Update installed Go version

Normal update:

```bash
go install github.com/0xRokib/clickup-cu-go/cmd/cu@latest
cu help
```

Force latest from GitHub if Go proxy/cache gives an old build:

```bash
GOPROXY=direct GONOSUMDB=github.com/0xRokib/clickup-cu-go go install github.com/0xRokib/clickup-cu-go/cmd/cu@main
cu help
```

If `cu help` runs the wrong command, test directly:

```bash
$(go env GOPATH)/bin/cu help
```

Then make sure Go bin is first in `PATH` as shown above.

### Option 2: build from repo

```bash
git clone git@github.com:0xRokib/clickup-cu-go.git
cd clickup-cu-go
go build -o cu ./cmd/cu
sudo mv cu /usr/local/bin/cu
cu help
```

Update later if you built from repo:

```bash
cd clickup-cu-go
git pull
go build -o cu ./cmd/cu
sudo mv cu /usr/local/bin/cu
cu help
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

## Output style

`cu` uses ClickUp-inspired terminal colors when running in an interactive terminal.

Plain output:

```bash
NO_COLOR=1 cu today
# or
CU_PLAIN=1 cu today
```

Force color if your terminal shows plain text:

```bash
CU_COLOR=1 cu help
# or
FORCE_COLOR=1 cu help
```

Make force color permanent if needed:

```bash
# macOS/zsh
echo 'export CU_COLOR=1' >> ~/.zshrc
source ~/.zshrc

# Linux/bash
echo 'export CU_COLOR=1' >> ~/.bashrc
source ~/.bashrc
```

If `CU_COLOR=1` still shows plain text, force reinstall latest first:

```bash
GOPROXY=direct GONOSUMDB=github.com/0xRokib/clickup-cu-go go install github.com/0xRokib/clickup-cu-go/cmd/cu@main
CU_COLOR=1 cu help
```

## Team troubleshooting

### `cu: command not found` or wrong `cu` runs

Check all matching commands:

```bash
which -a cu
```

If you see `/usr/bin/cu` before Go's path, shell is running the system `cu`, not this ClickUp CLI.

Test the Go-installed binary directly:

```bash
$(go env GOPATH)/bin/cu help
```

Fix by putting Go bin first in `PATH`.

zsh/macOS:

```bash
echo 'export PATH="$(go env GOPATH)/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

bash/Linux:

```bash
echo 'export PATH="$(go env GOPATH)/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

Then verify:

```bash
which -a cu
cu help
```

The first `cu` should be under your Go bin path, usually `~/go/bin/cu`.

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
