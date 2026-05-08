package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	cli = "cu"
	api = "https://api.clickup.com/api/v2"
)

var defaultStatuses = map[string]string{
	"backlog": "BACKLOG",
	"todo":    "TO-DO",
	"start":   "IN PROGRESS",
	"review":  "REVIEW",
	"qa":      "QA/TESTING",
	"blocked": "BLOCKED",
	"release": "READY FOR RELEASE",
	"done":    "DONE",
}

var ui = newStyle(colorEnabled())

type style struct {
	enabled bool
}

func newStyle(enabled bool) style { return style{enabled: enabled} }

func colorEnabled() bool {
	info, err := os.Stdout.Stat()
	isTTY := err == nil && (info.Mode()&os.ModeCharDevice) != 0
	return colorEnabledFromEnv(isTTY)
}

func colorEnabledFromEnv(isTTY bool) bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("CU_PLAIN") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	return isTTY
}

func (s style) ansi(code, text string) string {
	if !s.enabled {
		return text
	}
	return "\x1b[" + code + "m" + text + "\x1b[0m"
}
func (s style) rgb(r, g, b int, text string) string {
	return s.ansi(fmt.Sprintf("38;2;%d;%d;%d", r, g, b), text)
}
func (s style) bg(r, g, b int, fg string, text string) string {
	if !s.enabled {
		return text
	}
	return s.ansi(fmt.Sprintf("48;2;%d;%d;%d;%s", r, g, b, fg), text)
}
func (s style) bold(text string) string   { return s.ansi("1", text) }
func (s style) dim(text string) string    { return s.ansi("2", text) }
func (s style) dark(text string) string   { return s.rgb(41, 45, 52, text) }
func (s style) purple(text string) string { return s.rgb(123, 104, 238, text) }
func (s style) pink(text string) string   { return s.rgb(253, 113, 175, text) }
func (s style) blue(text string) string   { return s.rgb(73, 204, 249, text) }
func (s style) yellow(text string) string { return s.rgb(255, 200, 0, text) }

func (s style) brand() string { return s.purple("c") + s.pink("u") }
func (s style) divider(width int) string {
	if width <= 0 {
		width = 62
	}
	chunk := width / 4
	if chunk < 8 {
		chunk = 8
	}
	return s.purple(strings.Repeat("─", chunk)) +
		s.pink(strings.Repeat("─", chunk)) +
		s.blue(strings.Repeat("─", chunk)) +
		s.yellow(strings.Repeat("─", width-(chunk*3)))
}
func (s style) statusPill(status string) string {
	label := " " + strings.ToUpper(strings.TrimSpace(status)) + " "
	if strings.TrimSpace(status) == "" {
		label = " UNKNOWN "
	}
	if !s.enabled {
		return "[" + strings.TrimSpace(label) + "]"
	}
	switch statusAccent(status) {
	case "pink":
		return s.bg(253, 113, 175, "38;2;255;255;255", label)
	case "blue":
		return s.bg(73, 204, 249, "38;2;41;45;52", label)
	case "yellow":
		return s.bg(255, 200, 0, "38;2;41;45;52", label)
	case "purple":
		return s.bg(123, 104, 238, "38;2;255;255;255", label)
	default:
		return s.bg(41, 45, 52, "38;2;255;255;255", label)
	}
}

func statusAccent(status string) string {
	st := strings.ToLower(status)
	switch {
	case strings.Contains(st, "blocked"):
		return "pink"
	case strings.Contains(st, "review") || strings.Contains(st, "qa") || strings.Contains(st, "testing"):
		return "blue"
	case strings.Contains(st, "done") || strings.Contains(st, "release") || strings.Contains(st, "ready"):
		return "yellow"
	case strings.Contains(st, "progress"):
		return "purple"
	default:
		return "dark"
	}
}

type Config struct {
	WorkspaceID   string            `json:"workspaceId"`
	UserID        string            `json:"userId"`
	DefaultListID string            `json:"defaultListId"`
	Statuses      map[string]string `json:"statuses"`
}

type App struct {
	ConfigPath string
	TokenPath  string
	Cfg        Config
	Token      string
	Client     *http.Client
}

func main() {
	app := newApp()
	cmd := "today"
	args := os.Args[1:]
	if len(args) > 0 {
		cmd = args[0]
		args = args[1:]
	}
	if err := app.run(cmd, args); err != nil {
		die(err.Error())
	}
}

func newApp() *App {
	home, _ := os.UserHomeDir()
	configDir := filepath.Join(home, ".config", "cu")
	configPath := filepath.Join(configDir, "config.json")
	tokenPath := filepath.Join(configDir, "token")
	cfg := readConfig(configPath)
	token := firstNonEmpty(os.Getenv("CLICKUP_API_TOKEN"), os.Getenv("CLICKUP_TOKEN"), readTokenFile(tokenPath))
	return &App{ConfigPath: configPath, TokenPath: tokenPath, Cfg: cfg, Token: token, Client: &http.Client{Timeout: 30 * time.Second}}
}

func (a *App) run(cmd string, args []string) error {
	switch cmd {
	case "help", "--help", "-h":
		help(a)
		return nil
	case "init", "config":
		return a.cmdInit()
	case "token":
		if len(args) < 1 {
			return errors.New("usage: cu token pk_xxx")
		}
		return a.cmdToken(args[0])
	case "today":
		return a.cmdToday()
	case "list":
		return a.cmdList(args)
	case "show":
		return a.cmdShow(firstArg(args))
	case "start", "run":
		return a.cmdStart(firstArg(args))
	case "stop":
		return a.cmdStop()
	case "backlog", "todo", "review", "blocked", "release":
		return a.cmdSetStatus(firstArg(args), cmd, cmd)
	case "progress":
		return a.cmdSetStatus(firstArg(args), "start", "progress")
	case "qa", "testing":
		return a.cmdSetStatus(firstArg(args), "qa", "qa")
	case "done":
		return a.cmdSetStatus(firstArg(args), "done", "done")
	case "estimate":
		return a.cmdEstimate(args)
	case "addtime", "time":
		return a.cmdAddTime(args)
	case "create":
		return a.cmdCreate(args)
	case "subtask":
		return a.cmdSubtask(args)
	case "assign":
		return a.cmdAssign(args)
	default:
		return fmt.Errorf("unknown command: %s. Run cu help", cmd)
	}
}

func readConfig(path string) Config {
	var cfg Config
	b, err := os.ReadFile(path)
	if err == nil {
		_ = json.Unmarshal(b, &cfg)
	}
	if cfg.Statuses == nil {
		cfg.Statuses = map[string]string{}
	}
	return cfg
}

func readTokenFile(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func (a *App) requireToken() error {
	if strings.TrimSpace(a.Token) == "" {
		return errors.New("missing ClickUp token. Run: cu token pk_xxx  or export CLICKUP_API_TOKEN=\"pk_xxx\"")
	}
	return nil
}

func (a *App) requireWorkspace() (string, error) {
	if strings.TrimSpace(a.Cfg.WorkspaceID) == "" {
		return "", fmt.Errorf("missing workspaceId in %s. Run: cu init", a.ConfigPath)
	}
	return a.Cfg.WorkspaceID, nil
}

func (a *App) requireUser() (string, error) {
	if strings.TrimSpace(a.Cfg.UserID) == "" {
		return "", fmt.Errorf("missing userId in %s. Run: cu init", a.ConfigPath)
	}
	return a.Cfg.UserID, nil
}

func (a *App) status(key string) string {
	if v := strings.TrimSpace(a.Cfg.Statuses[key]); v != "" {
		return v
	}
	return defaultStatuses[key]
}

func (a *App) apiResult(method, endpoint string, body any) (map[string]any, error) {
	if err := a.requireToken(); err != nil {
		return nil, err
	}
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, api+endpoint, r)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", a.Token)
	req.Header.Set("Content-Type", "application/json")
	res, err := a.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	text, _ := io.ReadAll(res.Body)
	var out map[string]any
	if len(text) > 0 {
		if err := json.Unmarshal(text, &out); err != nil {
			out = map[string]any{"raw": string(text)}
		}
	} else {
		out = map[string]any{}
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		msg := firstNonEmpty(str(out["err"]), str(out["ECODE"]), str(out["raw"]), res.Status)
		return nil, fmt.Errorf("%s %s failed: %s", method, endpoint, clean(msg))
	}
	return out, nil
}

func (a *App) api(method, endpoint string, body any) (map[string]any, error) {
	return a.apiResult(method, endpoint, body)
}

func query(params map[string]any) string {
	v := url.Values{}
	for k, raw := range params {
		switch x := raw.(type) {
		case string:
			if x != "" {
				v.Add(k, x)
			}
		case bool:
			v.Add(k, strconv.FormatBool(x))
		case int:
			v.Add(k, strconv.Itoa(x))
		case []string:
			for _, s := range x {
				if s != "" {
					v.Add(k, s)
				}
			}
		}
	}
	if len(v) == 0 {
		return ""
	}
	return "?" + v.Encode()
}

func (a *App) getTask(id string) (map[string]any, error) {
	return a.api("GET", "/task/"+url.PathEscape(id), nil)
}

func (a *App) updateTask(id string, body map[string]any) (map[string]any, error) {
	return a.api("PUT", "/task/"+url.PathEscape(id), body)
}

func (a *App) listTasks(all bool, status string) ([]map[string]any, error) {
	team, err := a.requireWorkspace()
	if err != nil {
		return nil, err
	}
	includeClosed := strings.EqualFold(status, a.status("done"))
	params := map[string]any{"include_closed": strconv.FormatBool(includeClosed), "subtasks": "true", "page": 0}
	if !all {
		user, err := a.requireUser()
		if err != nil {
			return nil, err
		}
		params["assignees[]"] = []string{user}
	}
	if status != "" {
		params["statuses[]"] = []string{status}
	}
	data, err := a.api("GET", "/team/"+url.PathEscape(team)+"/task"+query(params), nil)
	if err != nil {
		return nil, err
	}
	return mapSlice(data["tasks"]), nil
}

func (a *App) currentTimer() (map[string]any, error) {
	team, err := a.requireWorkspace()
	if err != nil {
		return nil, err
	}
	return a.api("GET", "/team/"+url.PathEscape(team)+"/time_entries/current", nil)
}

func (a *App) startTimer(id string) (map[string]any, error) {
	team, err := a.requireWorkspace()
	if err != nil {
		return nil, err
	}
	return a.api("POST", "/team/"+url.PathEscape(team)+"/time_entries/start", map[string]any{"tid": id})
}

func (a *App) stopTimer() (map[string]any, error) {
	team, err := a.requireWorkspace()
	if err != nil {
		return nil, err
	}
	return a.api("POST", "/team/"+url.PathEscape(team)+"/time_entries/stop", map[string]any{})
}

func (a *App) addTime(id string, duration int64, note string) error {
	team, err := a.requireWorkspace()
	if err != nil {
		return err
	}
	end := time.Now().UnixMilli()
	start := end - duration
	_, err = a.api("POST", "/team/"+url.PathEscape(team)+"/time_entries", map[string]any{"tid": id, "start": start, "duration": duration, "description": note})
	return err
}

func (a *App) createTask(listID string, body map[string]any) (map[string]any, error) {
	return a.api("POST", "/list/"+url.PathEscape(listID)+"/task", body)
}

func (a *App) cmdToday() error {
	header("Today", "Timer, priority work, and next ClickUp moves")
	fmt.Printf("\n%s %s\n", ui.blue("●"), ui.bold("Running now"))
	if cur, err := a.currentTimer(); err == nil {
		entry := firstMap(cur["data"], cur["currentEntry"], cur)
		task := toMap(entry["task"])
		if len(entry) > 0 && (task["id"] != nil || entry["tid"] != nil) && (num(entry["duration"]) < 0 || entry["start"] != nil) {
			elapsed := abs(num(entry["duration"]))
			if elapsed == 0 && entry["start"] != nil {
				elapsed = float64(time.Now().UnixMilli()) - num(entry["start"])
			}
			fmt.Printf("  %s %s\n     %s %s %s\n", ui.purple("●"), ui.bold(clean(firstNonEmpty(str(task["name"]), str(entry["description"]), "active timer"))), ui.blue(firstNonEmpty(str(task["id"]), str(entry["tid"]))), ui.dim("•"), ui.yellow(fmtDuration(int64(elapsed))))
		} else {
			fmt.Printf("  %s\n", ui.dim("No active timer."))
		}
	} else {
		fmt.Printf("  %s\n", ui.dim("No active timer."))
	}
	tasks, err := a.listTasks(false, "")
	if err != nil {
		return err
	}
	groups := groupTasks(tasks)
	i := 1
	i = printGroup("Due / overdue", groups["due"], i)
	i = printGroup("Active", groups["active"], i)
	i = printGroup("Blocked", groups["blocked"], i)
	i = printGroup("Not started", groups["notStarted"], i)
	i = printGroup("Other", groups["other"], i)
	if i == 1 {
		fmt.Printf("\n%s %s\n", ui.blue("✓"), ui.bold("No open tasks found."))
	}
	fmt.Printf("\n%s %s\n", ui.yellow("→"), ui.bold("Next moves"))
	fmt.Printf("  %s %s %s %s %s %s %s\n", ui.blue("cu start <id>"), ui.dim("•"), ui.blue("cu show <id>"), ui.dim("•"), ui.blue("cu stop"), ui.dim("•"), ui.blue("cu release <id> • cu done <id>"))
	return nil
}

func (a *App) cmdList(args []string) error {
	all := contains(args, "all")
	status := ""
	for i, s := range args {
		if s == "status" && i+1 < len(args) {
			status = strings.Join(args[i+1:], " ")
			break
		}
	}
	tasks, err := a.listTasks(all, status)
	if err != nil {
		return err
	}
	sub := fmt.Sprintf("%d task(s)", len(tasks))
	if status != "" {
		sub += " with status " + status
	}
	if all {
		sub += " across workspace"
	} else {
		sub += " assigned to you"
	}
	header("Tasks", sub)
	if len(tasks) == 0 {
		success("Nothing to show.", "")
		return nil
	}
	for i, t := range tasks {
		taskRow(t, i+1)
	}
	return nil
}

func (a *App) cmdShow(id string) error {
	if id == "" {
		return errors.New("usage: cu show <task-id>")
	}
	t, err := a.getTask(id)
	if err != nil {
		return err
	}
	header(compact(firstNonEmpty(str(t["name"]), "Task"), 72), str(t["id"]))
	taskRow(t, 0)
	fmt.Printf("\n%s %s\n", ui.blue("◆"), ui.bold("Details"))
	fmt.Printf("  %s  %s\n", ui.dim("Assignees"), assignees(t))
	if n := int64(num(t["time_estimate"])); n > 0 {
		fmt.Printf("  %s   %s\n", ui.dim("Estimate"), ui.yellow(fmtDuration(n)))
	} else {
		fmt.Printf("  %s   %s\n", ui.dim("Estimate"), ui.dim("none"))
	}
	if n := int64(num(t["time_spent"])); n > 0 {
		fmt.Printf("  %s      %s\n", ui.dim("Spent"), ui.purple(fmtDuration(n)))
	} else {
		fmt.Printf("  %s      %s\n", ui.dim("Spent"), ui.dim("0m"))
	}
	if p := str(t["parent"]); p != "" {
		fmt.Printf("  %s     %s\n", ui.dim("Parent"), p)
	}
	desc := firstNonEmpty(str(t["description"]), str(t["text_content"]))
	if desc != "" {
		fmt.Printf("\n%s %s\n", ui.purple("◆"), ui.bold("Description"))
		fmt.Printf("  %s\n", compact(desc, 800))
	}
	return nil
}

func (a *App) cmdSetStatus(id, key, label string) error {
	if id == "" {
		return fmt.Errorf("usage: cu %s <task-id>", label)
	}
	st := a.status(key)
	t, err := a.updateTask(id, map[string]any{"status": st})
	if err != nil {
		return err
	}
	success(fmt.Sprintf("%s %s", label, id), fmt.Sprintf("%s • %s", statusName(t), firstNonEmpty(str(t["url"]), taskURL(id))))
	return nil
}

func (a *App) cmdStart(id string) error {
	if id == "" {
		return errors.New("usage: cu start <task-id>")
	}
	statusMsg := "status unchanged"
	if _, err := a.updateTask(id, map[string]any{"status": a.status("start")}); err == nil {
		statusMsg = "status " + a.status("start")
	}
	out, err := a.startTimer(id)
	if err != nil {
		return err
	}
	task := firstMap(pathVal(out, "data.task"), pathVal(out, "timeEntry.task"), out["task"])
	if len(task) == 0 {
		task, _ = a.getTask(id)
	}
	success("Started "+id, str(task["name"]))
	fmt.Printf("%s Timer active %s %s\n  %s\n", ui.yellow("→"), ui.dim("•"), statusMsg, ui.blue(taskURL(id)))
	return nil
}

func (a *App) cmdStop() error {
	out, err := a.stopTimer()
	if err != nil {
		return err
	}
	e := firstMap(out["data"], out["timeEntry"], out)
	task := toMap(e["task"])
	duration := int64(num(e["duration_ms"]))
	if duration == 0 {
		duration = int64(num(e["duration"]))
	}
	if duration == 0 && e["start"] != nil && e["end"] != nil {
		duration = int64(num(e["end"]) - num(e["start"]))
	}
	name := strings.TrimSpace(firstNonEmpty(str(task["id"]), "") + " — " + str(task["name"]))
	success("Stopped "+name, "Duration: "+fmtDuration(duration))
	return nil
}

func (a *App) cmdEstimate(args []string) error {
	if len(args) < 2 {
		return errors.New("usage: cu estimate <task-id> <duration>")
	}
	ms, err := parseDuration(strings.Join(args[1:], " "))
	if err != nil {
		return err
	}
	t, err := a.updateTask(args[0], map[string]any{"time_estimate": ms})
	if err != nil {
		return err
	}
	success("Estimate set "+args[0], fmt.Sprintf("%s • %s", fmtDuration(ms), firstNonEmpty(str(t["url"]), taskURL(args[0]))))
	return nil
}

func (a *App) cmdAddTime(args []string) error {
	if len(args) < 2 {
		return errors.New("usage: cu addtime <task-id> <duration> [note]")
	}
	durText, note := splitDurationAndNote(args[1:])
	ms, err := parseDuration(durText)
	if err != nil {
		return err
	}
	if err := a.addTime(args[0], ms, note); err != nil {
		return err
	}
	detail := fmtDuration(ms)
	if note != "" {
		detail += " • " + note
	}
	success("Added time "+args[0], detail)
	return nil
}

func (a *App) cmdCreate(args []string) error {
	listID := strings.TrimSpace(a.Cfg.DefaultListID)
	if listID == "" {
		return fmt.Errorf("missing defaultListId in %s", a.ConfigPath)
	}
	name := strings.TrimSpace(strings.Join(args, " "))
	if name == "" {
		return errors.New("usage: cu create <task title>")
	}
	t, err := a.createTask(listID, map[string]any{"name": name})
	if err != nil {
		return err
	}
	success("Created "+str(t["id"]), str(t["name"]))
	fmt.Println("  " + ui.blue(firstNonEmpty(str(t["url"]), taskURL(str(t["id"])))))
	return nil
}

func (a *App) cmdSubtask(args []string) error {
	if len(args) < 2 {
		return errors.New("usage: cu subtask <parent-task-id> <title>")
	}
	parent := args[0]
	name := strings.TrimSpace(strings.Join(args[1:], " "))
	p, err := a.getTask(parent)
	if err != nil {
		return err
	}
	listID := str(pathVal(p, "list.id"))
	if listID == "" {
		listID = a.Cfg.DefaultListID
	}
	t, err := a.createTask(listID, map[string]any{"name": name, "parent": parent})
	if err != nil {
		return err
	}
	success("Created subtask "+str(t["id"]), str(t["name"]))
	fmt.Println("  " + ui.blue(firstNonEmpty(str(t["url"]), taskURL(str(t["id"])))))
	return nil
}

func (a *App) cmdAssign(args []string) error {
	if len(args) < 2 {
		return errors.New("usage: cu assign <task-id> <user-id...>  # use numeric user IDs")
	}
	id := args[0]
	users := []int{}
	for _, x := range args[1:] {
		if x == "me" {
			u, err := a.requireUser()
			if err != nil {
				return err
			}
			x = u
		}
		if n, err := strconv.Atoi(x); err == nil && n > 0 {
			users = append(users, n)
		}
	}
	if len(users) == 0 {
		return errors.New("assign currently needs numeric user IDs or me")
	}
	t, err := a.getTask(id)
	if err != nil {
		return err
	}
	seen := map[int]bool{}
	merged := []int{}
	for _, a := range mapSlice(t["assignees"]) {
		if n := int(num(a["id"])); n > 0 && !seen[n] {
			seen[n] = true
			merged = append(merged, n)
		}
	}
	for _, n := range users {
		if !seen[n] {
			seen[n] = true
			merged = append(merged, n)
		}
	}
	out, err := a.updateTask(id, map[string]any{"assignees": merged})
	if err != nil {
		return err
	}
	parts := []string{}
	for _, n := range merged {
		parts = append(parts, strconv.Itoa(n))
	}
	success("Assigned "+id, strings.Join(parts, ", ")+" • "+firstNonEmpty(str(out["url"]), taskURL(id)))
	return nil
}

func (a *App) cmdToken(token string) error {
	if !strings.HasPrefix(strings.TrimSpace(token), "pk_") {
		return errors.New("usage: cu token pk_xxx")
	}
	if err := os.MkdirAll(filepath.Dir(a.TokenPath), 0700); err != nil {
		return err
	}
	if err := os.WriteFile(a.TokenPath, []byte(strings.TrimSpace(token)+"\n"), 0600); err != nil {
		return err
	}
	_ = os.Chmod(a.TokenPath, 0600)
	success("Token saved", a.TokenPath)
	if env := firstNonEmpty(os.Getenv("CLICKUP_API_TOKEN"), os.Getenv("CLICKUP_TOKEN")); env != "" && env != token {
		fmt.Printf("%s %s\n", ui.yellow("→"), "A different CLICKUP_API_TOKEN/CLICKUP_TOKEN is exported and will override this file token.")
	}
	return nil
}

func (a *App) cmdInit() error {
	if err := os.MkdirAll(filepath.Dir(a.ConfigPath), 0700); err != nil {
		return err
	}
	found := a.discoverConfig()
	sample := Config{
		WorkspaceID:   firstUseful(a.Cfg.WorkspaceID, found.WorkspaceID),
		UserID:        firstUseful(a.Cfg.UserID, found.UserID),
		DefaultListID: firstUseful(a.Cfg.DefaultListID, found.DefaultListID),
		Statuses:      map[string]string{},
	}
	for k, v := range defaultStatuses {
		sample.Statuses[k] = v
	}
	for k, v := range a.Cfg.Statuses {
		if strings.TrimSpace(v) != "" {
			sample.Statuses[k] = v
		}
	}
	shouldWrite := !fileExists(a.ConfigPath) || !useful(a.Cfg.WorkspaceID) || !useful(a.Cfg.UserID) || !useful(a.Cfg.DefaultListID)
	if shouldWrite {
		b, _ := json.MarshalIndent(sample, "", "  ")
		if err := os.WriteFile(a.ConfigPath, append(b, '\n'), 0600); err != nil {
			return err
		}
	}
	_ = os.Chmod(a.ConfigPath, 0600)
	header("Config", a.ConfigPath)
	b, _ := os.ReadFile(a.ConfigPath)
	fmt.Println(clean(string(b)))
	if a.Token == "" {
		fmt.Printf("%s Set token first for auto-discovery: %s\n", ui.yellow("→"), ui.blue("cu token pk_xxx"))
	} else if sample.WorkspaceID == "" || sample.UserID == "" {
		fmt.Printf("%s Could not auto-discover all IDs. Check token permissions or set missing values manually.\n", ui.yellow("→"))
	} else if sample.DefaultListID == "" {
		fmt.Printf("%s Workspace/user detected. Set defaultListId manually if you want cu create.\n", ui.yellow("→"))
	}
	fmt.Printf("  %s %s\n", ui.dim("Token stored at"), a.TokenPath)
	return nil
}

func (a *App) discoverConfig() Config {
	found := Config{}
	if a.Token == "" {
		return found
	}
	if user, err := a.apiResult("GET", "/user", nil); err == nil {
		found.UserID = str(pathVal(user, "user.id"))
	}
	if teams, err := a.apiResult("GET", "/team", nil); err == nil {
		list := mapSlice(teams["teams"])
		preferred := firstUseful(a.Cfg.WorkspaceID)
		team := map[string]any{}
		for _, t := range list {
			if str(t["id"]) == preferred {
				team = t
				break
			}
		}
		if len(team) == 0 && len(list) > 0 {
			team = list[0]
		}
		found.WorkspaceID = str(team["id"])
		members := mapSlice(team["members"])
		if found.UserID == "" && len(members) == 1 {
			found.UserID = str(pathVal(members[0], "user.id"))
		}
	}
	found.DefaultListID = firstUseful(a.Cfg.DefaultListID)
	if found.DefaultListID == "" {
		found.DefaultListID = a.discoverFirstListID(found.WorkspaceID)
	}
	return found
}

func (a *App) discoverFirstListID(teamID string) string {
	if teamID == "" {
		return ""
	}
	spacesData, err := a.apiResult("GET", "/team/"+url.PathEscape(teamID)+"/space"+query(map[string]any{"archived": "false"}), nil)
	if err != nil {
		return ""
	}
	for _, space := range mapSlice(spacesData["spaces"]) {
		spaceID := str(space["id"])
		if direct, err := a.apiResult("GET", "/space/"+url.PathEscape(spaceID)+"/list"+query(map[string]any{"archived": "false"}), nil); err == nil {
			lists := mapSlice(direct["lists"])
			if len(lists) > 0 && str(lists[0]["id"]) != "" {
				return str(lists[0]["id"])
			}
		}
		folders, err := a.apiResult("GET", "/space/"+url.PathEscape(spaceID)+"/folder"+query(map[string]any{"archived": "false"}), nil)
		if err != nil {
			continue
		}
		for _, folder := range mapSlice(folders["folders"]) {
			if nested, err := a.apiResult("GET", "/folder/"+url.PathEscape(str(folder["id"]))+"/list"+query(map[string]any{"archived": "false"}), nil); err == nil {
				lists := mapSlice(nested["lists"])
				if len(lists) > 0 && str(lists[0]["id"]) != "" {
					return str(lists[0]["id"])
				}
			}
		}
	}
	return ""
}

func help(a *App) {
	header("ClickUp CLI", "Fast daily task control from terminal or Pi")
	helpSection("Daily", "purple")
	helpLine("cu today", "Dashboard: timer + my open tasks")
	helpLine("cu list [all|status X]", "List tasks")
	helpLine("cu show <id>", "Show task details")
	helpSection("Workflow statuses", "blue")
	helpLine("cu backlog <id>", "Set BACKLOG")
	helpLine("cu todo <id>", "Set TO-DO")
	helpLine("cu start <id>", "Set IN PROGRESS + start timer")
	helpLine("cu progress <id>", "Set IN PROGRESS without timer")
	helpLine("cu review <id>", "Set REVIEW")
	helpLine("cu qa | testing <id>", "Set QA/TESTING")
	helpLine("cu blocked <id>", "Set BLOCKED")
	helpLine("cu release <id>", "Set READY FOR RELEASE")
	helpLine("cu done <id>", "Set DONE")
	helpSection("Time", "yellow")
	helpLine("cu stop", "Stop current timer")
	helpLine("cu estimate <id> <dur>", "Set estimate (2h, 30m, 2h 30m)")
	helpLine("cu addtime <id> <dur> [note]", "Add manual time entry")
	helpSection("Create/update", "pink")
	helpLine("cu create <title>", "Create in default list")
	helpLine("cu subtask <parent> <title>", "Create subtask")
	helpLine("cu assign <id> <user-id|me...>", "Add assignees")
	helpSection("Setup", "purple")
	helpLine("cu init", "Create/show config")
	helpLine("cu token <pk_xxx>", "Save token locally")
	fmt.Printf("\n%s %s\n", ui.dim("Configured status keys:"), "backlog, todo, start, review, qa, blocked, release, done")
	fmt.Printf("%s %s\n", ui.dim("Config:"), a.ConfigPath)
	fmt.Printf("%s  CLICKUP_API_TOKEN or %s\n", ui.dim("Token:"), a.TokenPath)
}

func parseDuration(input string) (int64, error) {
	s := strings.TrimSpace(input)
	re := regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*(h|hr|hrs|hour|hours|m|min|mins|minute|minutes)`)
	matches := re.FindAllStringSubmatch(s, -1)
	var ms float64
	for _, m := range matches {
		n, _ := strconv.ParseFloat(m[1], 64)
		u := strings.ToLower(m[2])
		if strings.HasPrefix(u, "h") {
			ms += n * 3600000
		} else {
			ms += n * 60000
		}
	}
	if ms == 0 {
		if regexp.MustCompile(`^\d+$`).MatchString(s) {
			n, _ := strconv.ParseInt(s, 10, 64)
			ms = float64(n * 60000)
		}
	}
	if ms == 0 {
		return 0, fmt.Errorf("could not parse duration: %s", input)
	}
	return int64(ms + 0.5), nil
}

func fmtDuration(ms int64) string {
	if ms < 0 {
		ms = -ms
	}
	h := ms / 3600000
	m := (ms % 3600000) / 60000
	if (ms%3600000)%60000 >= 30000 {
		m++
	}
	if m == 60 {
		h++
		m = 0
	}
	if h > 0 && m > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	if h > 0 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dm", m)
}

func splitDurationAndNote(args []string) (string, string) {
	if len(args) == 0 {
		return "", ""
	}
	dur := []string{}
	i := 0
	for i < len(args) {
		tok := args[i]
		if regexp.MustCompile(`(?i)^\d+(?:\.\d+)?(h|hr|hrs|hour|hours|m|min|mins|minute|minutes)?$`).MatchString(tok) {
			dur = append(dur, tok)
			i++
			if !regexp.MustCompile(`(?i)\d`).MatchString(tok) {
				break
			}
			continue
		}
		if regexp.MustCompile(`(?i)^(h|hr|hrs|hour|hours|m|min|mins|minute|minutes)$`).MatchString(tok) && len(dur) > 0 {
			dur = append(dur, tok)
			i++
			continue
		}
		break
	}
	if len(dur) == 0 {
		dur = []string{args[0]}
		i = 1
	}
	return strings.Join(dur, " "), strings.Join(args[i:], " ")
}

func groupTasks(tasks []map[string]any) map[string][]map[string]any {
	today := time.Now().Format("2006-01-02")
	groups := map[string][]map[string]any{"due": {}, "active": {}, "blocked": {}, "notStarted": {}, "other": {}}
	for _, t := range tasks {
		st := strings.ToLower(statusName(t))
		d := due(t)
		switch {
		case d != "" && d <= today:
			groups["due"] = append(groups["due"], t)
		case strings.Contains(st, "blocked"):
			groups["blocked"] = append(groups["blocked"], t)
		case strings.Contains(st, "progress") || strings.Contains(st, "review") || strings.Contains(st, "qa") || strings.Contains(st, "testing") || strings.Contains(st, "ready") || strings.Contains(st, "release"):
			groups["active"] = append(groups["active"], t)
		case strings.Contains(st, "backlog") || strings.Contains(st, "to-do") || strings.Contains(st, "todo") || strings.Contains(st, "not started"):
			groups["notStarted"] = append(groups["notStarted"], t)
		default:
			groups["other"] = append(groups["other"], t)
		}
	}
	return groups
}

func printGroup(title string, arr []map[string]any, start int) int {
	if len(arr) == 0 {
		return start
	}
	fmt.Printf("\n%s %s\n", ui.purple("◆"), ui.bold(fmt.Sprintf("%s %d", title, len(arr))))
	for i, t := range arr {
		taskRow(t, start+i)
	}
	return start + len(arr)
}

func taskRow(t map[string]any, i int) {
	prefix := ""
	if i > 0 {
		prefix = ui.dim(fmt.Sprintf("%2d ", i))
	}
	fmt.Printf("  %s%s %s\n", prefix, ui.statusPill(statusName(t)), ui.bold(compact(str(t["name"]), 80)))
	fmt.Printf("     %s\n", taskMeta(t))
	fmt.Printf("     %s\n", ui.blue(firstNonEmpty(str(t["url"]), taskURL(str(t["id"])))))
}

func taskMeta(t map[string]any) string {
	bits := []string{ui.blue(str(t["id"]))}
	if p := priority(t); p != "" {
		bits = append(bits, ui.pink(p))
	}
	if d := due(t); d != "" {
		bits = append(bits, ui.yellow("due "+d))
	}
	if l := str(pathVal(t, "list.name")); l != "" {
		bits = append(bits, ui.dim(l))
	}
	return strings.Join(bits, ui.dim(" • "))
}

func statusName(t map[string]any) string {
	if s := str(t["status"]); s != "" {
		return s
	}
	if s := str(pathVal(t, "status.status")); s != "" {
		return s
	}
	return "unknown"
}

func priority(t map[string]any) string {
	if s := str(t["priority"]); s != "" {
		return s
	}
	return str(pathVal(t, "priority.priority"))
}

func due(t map[string]any) string {
	v := str(t["due_date"])
	if v == "" {
		return ""
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil || n <= 0 {
		return ""
	}
	return time.UnixMilli(n).UTC().Format("2006-01-02")
}

func assignees(t map[string]any) string {
	items := []string{}
	for _, a := range mapSlice(t["assignees"]) {
		items = append(items, firstNonEmpty(str(a["username"]), str(a["email"]), str(a["id"])))
	}
	if len(items) == 0 {
		return "none"
	}
	return strings.Join(items, ", ")
}

func header(title, subtitle string) {
	fmt.Printf("\n%s %s\n", ui.brand(), ui.bold(clean(title)))
	if subtitle != "" {
		fmt.Printf("   %s\n", ui.dim(clean(subtitle)))
	}
	fmt.Println(ui.divider(62))
}

func helpSection(title, accent string) {
	marker := ui.purple("◆")
	switch accent {
	case "blue":
		marker = ui.blue("◆")
	case "yellow":
		marker = ui.yellow("◆")
	case "pink":
		marker = ui.pink("◆")
	}
	fmt.Printf("\n%s %s\n", marker, ui.bold(title))
}

func helpLine(command, desc string) {
	fmt.Printf("  %s %s\n", ui.blue(fmt.Sprintf("%-32s", command)), ui.dim(desc))
}
func success(message, detail string) {
	fmt.Printf("%s %s\n", ui.blue("✓"), ui.bold(clean(message)))
	if detail != "" {
		fmt.Printf("  %s\n", ui.dim(clean(detail)))
	}
}
func die(msg string) {
	fmt.Fprintf(os.Stderr, "\n%s %s\n  %s\n  %s %s\n\n", ui.pink("✕"), ui.bold(cli+" error"), clean(msg), ui.dim("Help:"), ui.blue(cli+" help"))
	os.Exit(1)
}

func taskURL(id string) string { return "https://app.clickup.com/t/" + id }
func firstArg(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
func firstUseful(vals ...string) string {
	for _, v := range vals {
		if useful(v) {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
func useful(v string) bool { return strings.TrimSpace(v) != "" && strings.TrimSpace(v) != "test" }
func contains(xs []string, v string) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}
func fileExists(path string) bool { _, err := os.Stat(path); return err == nil }
func compact(text string, max int) string {
	s := strings.Join(strings.Fields(clean(text)), " ")
	if max > 0 && len([]rune(s)) > max {
		r := []rune(s)
		return string(r[:max-1]) + "…"
	}
	return s
}
func clean(text string) string {
	s := regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`).ReplaceAllString(str(text), "")
	s = strings.Map(func(r rune) rune {
		if (r >= 0 && r < 9) || (r > 10 && r < 32) || r == 127 {
			return -1
		}
		return r
	}, s)
	return s
}
func str(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case fmt.Stringer:
		return x.String()
	case float64:
		if x == float64(int64(x)) {
			return strconv.FormatInt(int64(x), 10)
		}
		return strconv.FormatFloat(x, 'f', -1, 64)
	case int:
		return strconv.Itoa(x)
	case int64:
		return strconv.FormatInt(x, 10)
	case bool:
		return strconv.FormatBool(x)
	default:
		return fmt.Sprintf("%v", v)
	}
}
func num(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case int:
		return float64(x)
	case int64:
		return float64(x)
	case string:
		n, _ := strconv.ParseFloat(x, 64)
		return n
	default:
		return 0
	}
}
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
func toMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}
func mapSlice(v any) []map[string]any {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(arr))
	for _, item := range arr {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}
func firstMap(vals ...any) map[string]any {
	for _, v := range vals {
		m := toMap(v)
		if len(m) > 0 {
			return m
		}
	}
	return map[string]any{}
}
func pathVal(m map[string]any, path string) any {
	var cur any = m
	for _, part := range strings.Split(path, ".") {
		cm, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = cm[part]
	}
	return cur
}
