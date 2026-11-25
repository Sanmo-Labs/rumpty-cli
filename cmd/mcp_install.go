package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/spf13/cobra"

	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/config"
)

const (
	mcpScopeUser    = "user"
	mcpScopeProject = "project"
	mcpServerName   = "rumpty"
	mcpVSCodeInput  = "rumpty-key"
)

type mcpChange int

const (
	mcpUnchanged mcpChange = iota
	mcpAdded
	mcpReplaced
)

// mcpInstaller describes how to write one client's config file. scopes[0] is
// the default scope.
type mcpInstaller struct {
	scopes []string
	path   func(scope string) (string, error)
	merge  func(old []byte, force bool) ([]byte, mcpChange, error)
}

type mcpHTTPEntry struct {
	Type    string            `json:"type,omitempty"`
	URL     string            `json:"url,omitempty"`
	HTTPURL string            `json:"httpUrl,omitempty"`
	Headers map[string]string `json:"headers"`
}

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

func bearer(ref string) map[string]string {
	return map[string]string{"Authorization": "Bearer " + ref}
}

var (
	mcpEnvRef = "${" + config.EnvToken + "}"

	claudeCodeEntry = mustJSON(mcpHTTPEntry{Type: "http", URL: mcpURL, Headers: bearer(mcpEnvRef)})
	cursorEntry     = mustJSON(mcpHTTPEntry{URL: mcpURL, Headers: bearer("${env:" + config.EnvToken + "}")})
	vscodeEntry     = mustJSON(mcpHTTPEntry{Type: "http", URL: mcpURL, Headers: bearer("${input:" + mcpVSCodeInput + "}")})
	geminiEntry     = mustJSON(mcpHTTPEntry{HTTPURL: mcpURL, Headers: bearer(mcpEnvRef)})
	vscodeKeyInput  = mustJSON(struct {
		Type        string `json:"type"`
		ID          string `json:"id"`
		Description string `json:"description"`
		Password    bool   `json:"password"`
	}{"promptString", mcpVSCodeInput, "RumptyCloud API key", true})

	codexEntry = map[string]any{"url": mcpURL, "bearer_token_env_var": config.EnvToken}
	codexBlock = "[mcp_servers." + mcpServerName + "]\n" +
		"url = \"" + mcpURL + "\"\n" +
		"bearer_token_env_var = \"" + config.EnvToken + "\"\n"
)

func userOrProjectPath(userRel, projectRel string) func(string) (string, error) {
	return func(scope string) (string, error) {
		if scope == mcpScopeUser {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("find home directory: %w", err)
			}
			return filepath.Join(home, userRel), nil
		}
		wd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("find current directory: %w", err)
		}
		return filepath.Join(wd, projectRel), nil
	}
}

func codexConfigPath(string) (string, error) {
	if dir := strings.TrimSpace(os.Getenv("CODEX_HOME")); dir != "" {
		return filepath.Join(dir, "config.toml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home directory: %w", err)
	}
	return filepath.Join(home, ".codex", "config.toml"), nil
}

// claude-code user scope is handled by the claude CLI, so it has no entry here.
var mcpInstallers = map[string]mcpInstaller{
	"claude-code": {
		scopes: []string{mcpScopeProject, mcpScopeUser},
		path:   userOrProjectPath("", ".mcp.json"),
		merge:  jsonServerMerge("claude-code", "mcpServers", claudeCodeEntry, nil),
	},
	"cursor": {
		scopes: []string{mcpScopeUser, mcpScopeProject},
		path:   userOrProjectPath(filepath.Join(".cursor", "mcp.json"), filepath.Join(".cursor", "mcp.json")),
		merge:  jsonServerMerge("cursor", "mcpServers", cursorEntry, nil),
	},
	"vscode": {
		scopes: []string{mcpScopeProject},
		path:   userOrProjectPath("", filepath.Join(".vscode", "mcp.json")),
		merge:  jsonServerMerge("vscode", "servers", vscodeEntry, vscodeKeyInput),
	},
	"gemini": {
		scopes: []string{mcpScopeUser, mcpScopeProject},
		path:   userOrProjectPath(filepath.Join(".gemini", "settings.json"), filepath.Join(".gemini", "settings.json")),
		merge:  jsonServerMerge("gemini", "mcpServers", geminiEntry, nil),
	},
	"codex": {
		scopes: []string{mcpScopeUser},
		path:   codexConfigPath,
		merge:  mergeCodexTOML,
	},
}

var mcpManualOnly = map[string]string{
	"zed":            "Zed can't read the key from an environment variable, so its settings need the key itself, and rumpty never writes API keys to files",
	"claude-desktop": "Claude Desktop can't read the key from an environment variable, so its config needs the key itself, and rumpty never writes API keys to files",
}

// Swapped in tests.
var (
	mcpLookPath   = exec.LookPath
	mcpRunCommand = func(ctx context.Context, out, errOut io.Writer, name string, args ...string) error {
		c := exec.CommandContext(ctx, name, args...)
		c.Stdout, c.Stderr = out, errOut
		return c.Run()
	}
)

type mcpInstallOptions struct {
	scope  string
	dryRun bool
	force  bool
}

func newMCPInstallCmd(rt *app.Runtime) *cobra.Command {
	var opts mcpInstallOptions
	ids := make([]string, 0, len(mcpClients))
	for _, c := range mcpClients {
		ids = append(ids, c.ID)
	}
	cmd := &cobra.Command{
		Use:   "install <client>",
		Short: "Add Rumpty MCP to an AI client's config file",
		Long: "Add the rumpty server to an MCP client's config file, keeping everything else in it.\n\n" +
			"Clients: claude-code, cursor, vscode, gemini, codex. For zed and claude-desktop it prints\n" +
			"the setup steps instead, because those clients need the key pasted into their config.\n\n" +
			"The config never contains your API key: it points at $" + config.EnvToken + " (or, for VS Code,\n" +
			"a prompt that stores the key securely).\n\n" +
			"Scopes: user (all projects) or project (files in the current directory).\n" +
			"Default: project for claude-code and vscode, user for cursor, gemini and codex.\n" +
			"vscode supports project only; codex supports user only.",
		Example: `  rumpty mcp install cursor
  rumpty mcp install claude-code --scope project
  rumpty mcp install gemini --dry-run
  rumpty mcp install codex --force`,
		Args:      cobra.ExactArgs(1),
		ValidArgs: ids,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMCPInstall(cmd.Context(), rt.Streams.Out, rt.Streams.ErrOut, args[0], opts)
		},
	}
	cmd.Flags().StringVar(&opts.scope, "scope", "", "Where to install: user or project (default depends on the client)")
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "Show the change without writing anything")
	cmd.Flags().BoolVar(&opts.force, "force", false, "Replace an existing rumpty entry that differs")
	_ = cmd.RegisterFlagCompletionFunc("scope", cobra.FixedCompletions([]string{mcpScopeUser, mcpScopeProject}, cobra.ShellCompDirectiveNoFileComp))
	return cmd
}

func runMCPInstall(ctx context.Context, out, errOut io.Writer, id string, opts mcpInstallOptions) error {
	client, ok := findMCPClient(id)
	if !ok {
		ids := make([]string, 0, len(mcpClients))
		for _, c := range mcpClients {
			ids = append(ids, c.ID)
		}
		return config.NewUsageError("unknown client %q; use one of: %s", id, strings.Join(ids, ", "))
	}
	info := newMCPInfo()

	if why, manual := mcpManualOnly[client.ID]; manual {
		printMCPClient(out, &info, client)
		return fmt.Errorf("%s; follow the steps above to set up %s by hand", why, client.Name)
	}

	inst := mcpInstallers[client.ID]
	scope := strings.ToLower(strings.TrimSpace(opts.scope))
	if scope == "" {
		scope = inst.scopes[0]
	}
	if scope != mcpScopeUser && scope != mcpScopeProject {
		return config.NewUsageError("unknown scope %q; use user or project", opts.scope)
	}
	supported := false
	for _, s := range inst.scopes {
		supported = supported || s == scope
	}
	if !supported {
		return config.NewUsageError("%s supports --scope %s only", client.ID, strings.Join(inst.scopes, " or "))
	}

	if client.ID == "claude-code" && scope == mcpScopeUser {
		return installClaudeCodeUser(ctx, out, errOut, &info, client, opts)
	}

	path, err := inst.path(scope)
	if err != nil {
		return err
	}
	old, err := os.ReadFile(path)
	exists := err == nil
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("read %s: %w", path, err)
	}
	updated, change, err := inst.merge(old, opts.force)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}

	if change == mcpUnchanged {
		fmt.Fprintf(out, "%s already has the rumpty server; nothing changed.\n", path)
		return nil
	}
	verb := "add"
	if change == mcpReplaced {
		verb = "replace"
	}
	if opts.dryRun {
		state := ""
		if !exists {
			state = " (new file)"
		}
		fmt.Fprintf(out, "Dry run: would %s rumpty in %s%s. Nothing was written.\n\n", verb, path, state)
		fmt.Fprint(out, lineDiff(string(old), string(updated)))
		return nil
	}

	mode := os.FileMode(0o644)
	if scope == mcpScopeUser {
		mode = 0o600
	}
	if now, err := os.ReadFile(path); (err == nil) != exists || !bytes.Equal(now, old) {
		return fmt.Errorf("%s changed while rumpty was updating it; run the command again", path)
	}
	if err := writeFileAtomic(path, updated, mode); err != nil {
		return err
	}
	past := map[mcpChange]string{mcpAdded: "Added", mcpReplaced: "Replaced"}[change]
	fmt.Fprintf(out, "%s rumpty in %s (%s, %s scope).\n\n", past, path, client.Name, scope)
	if scope == mcpScopeProject {
		if _, err := os.Stat(".git"); err != nil {
			fmt.Fprintf(out, "Note: the current directory isn't a git repository root. %s reads project config from the folder you open, so run this there.\n\n", client.Name)
		}
	}
	printMCPInstallNext(out, &info, client)
	return nil
}

func printMCPInstallNext(w io.Writer, info *mcpInfo, c mcpClient) {
	if c.ID == "vscode" {
		fmt.Fprintf(w, "Create an API key at %s\nVS Code asks for it the first time it starts rumpty and stores it securely.\n", info.CreateKey)
	} else {
		printMCPKeyStep(w, info, c.UsesEnv)
	}
	fmt.Fprintf(w, "\nRestart %s so it loads the server. %s\n", c.Name, c.Check)
}

func claudeCodeUserArgs() []string {
	return []string{"mcp", "add", "--transport", "http", "--scope", "user", mcpServerName, mcpURL,
		"--header", "Authorization: Bearer " + mcpEnvRef}
}

func installClaudeCodeUser(ctx context.Context, out, errOut io.Writer, info *mcpInfo, client mcpClient, opts mcpInstallOptions) error {
	add := claudeCodeUserArgs()
	remove := []string{"mcp", "remove", "--scope", "user", mcpServerName}
	shown := "claude mcp add --transport http --scope user " + mcpServerName + " " + mcpURL +
		" \\\n  --header 'Authorization: Bearer " + mcpEnvRef + "'"
	printCmds := func() {
		if opts.force {
			fmt.Fprintf(out, "  claude %s\n", strings.Join(remove, " "))
		}
		fmt.Fprintf(out, "  %s\n", shown)
	}

	bin, err := mcpLookPath("claude")
	if opts.dryRun {
		fmt.Fprintf(out, "Dry run: would run\n\n")
		printCmds()
		fmt.Fprintf(out, "\nThe single quotes keep %s unexpanded, so Claude Code reads the key when it starts.\n", mcpEnvRef)
		return nil
	}
	if err != nil {
		fmt.Fprintf(out, "The claude command isn't on your PATH. Run this where it is:\n\n")
		printCmds()
		fmt.Fprintln(out)
		return errors.New("claude not found on PATH; run the command above to finish")
	}
	if opts.force {
		// Fails when there is nothing to remove; the add below reports real problems.
		_ = mcpRunCommand(ctx, io.Discard, io.Discard, bin, remove...)
	}
	if err := mcpRunCommand(ctx, out, errOut, bin, add...); err != nil {
		if opts.force {
			fmt.Fprintf(out, "\nThe old rumpty entry may already be removed. To add it, run:\n\n  %s\n\n", shown)
			return fmt.Errorf("claude mcp add failed (%v)", err)
		}
		return fmt.Errorf("claude mcp add failed (%v); if rumpty already exists, rerun with --force to replace it", err)
	}
	fmt.Fprintf(out, "\nAdded rumpty to Claude Code (user scope).\n\n")
	printMCPInstallNext(out, info, client)
	return nil
}

var errMCPConflict = errors.New("it already has a different rumpty server; rerun with --force to replace it")

// jsonServerMerge returns a merge that sets serversKey.rumpty to entry and, if
// input is set, adds it to "inputs" unless an input with the same id exists.
// Other keys keep their order and values; only whitespace is normalized.
func jsonServerMerge(id, serversKey string, entry, input json.RawMessage) func([]byte, bool) ([]byte, mcpChange, error) {
	return func(old []byte, force bool) ([]byte, mcpChange, error) {
		var top []jsonMember
		if len(bytes.TrimSpace(old)) > 0 {
			if bytes.HasPrefix(old, []byte("\xef\xbb\xbf")) {
				return nil, 0, fmt.Errorf("starts with a byte order mark, so rumpty won't rewrite it; add the entry shown by rumpty mcp %s by hand", id)
			}
			if !json.Valid(old) {
				return nil, 0, fmt.Errorf("not plain JSON (comments or trailing commas?), so rumpty won't rewrite it; add the entry shown by rumpty mcp %s by hand", id)
			}
			var err error
			if top, err = parseJSONObject(old); err != nil {
				return nil, 0, err
			}
		}

		var servers []jsonMember
		if raw, ok := getMember(top, serversKey); ok && string(raw) != "null" {
			var err error
			if servers, err = parseJSONObject(raw); err != nil {
				return nil, 0, fmt.Errorf("%q: %w", serversKey, err)
			}
		}
		change := mcpAdded
		if cur, ok := getMember(servers, mcpServerName); ok {
			if jsonEqual(cur, entry) {
				change = mcpUnchanged
			} else if !force {
				return nil, 0, errMCPConflict
			} else {
				change = mcpReplaced
			}
		}

		inputChanged := false
		if input != nil {
			var inputs []json.RawMessage
			raw, ok := getMember(top, "inputs")
			if ok {
				if err := json.Unmarshal(raw, &inputs); err != nil {
					return nil, 0, errors.New(`"inputs" is not a list`)
				}
			}
			if !hasInputID(inputs, mcpVSCodeInput) {
				inputs = append(inputs, input)
				parts := make([][]byte, len(inputs))
				for i, in := range inputs {
					parts[i] = in
				}
				b := append([]byte{'['}, bytes.Join(parts, []byte{','})...)
				top = setMember(top, "inputs", append(b, ']'))
				inputChanged = true
			}
		}
		if change == mcpUnchanged && !inputChanged {
			return old, mcpUnchanged, nil
		}
		if change == mcpUnchanged {
			change = mcpAdded
		}

		encServers, err := encodeJSONObject(setMember(servers, mcpServerName, entry))
		if err != nil {
			return nil, 0, err
		}
		encTop, err := encodeJSONObject(setMember(top, serversKey, encServers))
		if err != nil {
			return nil, 0, err
		}
		out, err := indentJSON(encTop, detectIndent(old))
		if err != nil {
			return nil, 0, err
		}
		return out, change, nil
	}
}

type jsonMember struct {
	Key   string
	Value json.RawMessage
}

func parseJSONObject(data []byte) ([]jsonMember, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return nil, errors.New("expected a JSON object")
	}
	var members []jsonMember
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, ok := tok.(string)
		if !ok {
			return nil, errors.New("expected an object key")
		}
		var v json.RawMessage
		if err := dec.Decode(&v); err != nil {
			return nil, err
		}
		if _, dup := getMember(members, key); dup {
			return nil, fmt.Errorf("has the key %q more than once, so rumpty won't rewrite it; remove the duplicate first", key)
		}
		members = append(members, jsonMember{Key: key, Value: v})
	}
	if _, err := dec.Token(); err != nil {
		return nil, err
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, errors.New("unexpected data after the JSON object")
	}
	return members, nil
}

func getMember(members []jsonMember, key string) (json.RawMessage, bool) {
	for _, m := range members {
		if m.Key == key {
			return m.Value, true
		}
	}
	return nil, false
}

// setMember replaces the member named key or appends it. Keys are unique; parseJSONObject refuses duplicates.
func setMember(members []jsonMember, key string, v json.RawMessage) []jsonMember {
	out := append([]jsonMember(nil), members...)
	for i := range out {
		if out[i].Key == key {
			out[i].Value = v
			return out
		}
	}
	return append(out, jsonMember{Key: key, Value: v})
}

func encodeJSONObject(members []jsonMember) (json.RawMessage, error) {
	var b bytes.Buffer
	b.WriteByte('{')
	for i, m := range members {
		if i > 0 {
			b.WriteByte(',')
		}
		enc := json.NewEncoder(&b)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(m.Key); err != nil {
			return nil, err
		}
		b.Truncate(b.Len() - 1)
		b.WriteByte(':')
		b.Write(m.Value)
	}
	b.WriteByte('}')
	return b.Bytes(), nil
}

func indentJSON(data []byte, indent string) ([]byte, error) {
	var compact, out bytes.Buffer
	if err := json.Compact(&compact, data); err != nil {
		return nil, err
	}
	if err := json.Indent(&out, compact.Bytes(), "", indent); err != nil {
		return nil, err
	}
	out.WriteByte('\n')
	return out.Bytes(), nil
}

// detectIndent returns the indent of the first indented line, or two spaces.
func detectIndent(data []byte) string {
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimLeft(line, " \t")
		if trimmed != "" && len(trimmed) < len(line) {
			return line[:len(line)-len(trimmed)]
		}
	}
	return "  "
}

func jsonEqual(a, b json.RawMessage) bool {
	var x, y any
	if json.Unmarshal(a, &x) != nil || json.Unmarshal(b, &y) != nil {
		return false
	}
	return reflect.DeepEqual(x, y)
}

func hasInputID(inputs []json.RawMessage, id string) bool {
	for _, in := range inputs {
		var v struct {
			ID string `json:"id"`
		}
		if json.Unmarshal(in, &v) == nil && v.ID == id {
			return true
		}
	}
	return false
}

var (
	codexHeaderRe = regexp.MustCompile(`^\[\s*mcp_servers\s*\.\s*("rumpty"|'rumpty'|rumpty)\s*\]\s*(#.*)?$`)
	codexKeyRe    = regexp.MustCompile(`(?m)^\s*("mcp_servers"|'mcp_servers'|mcp_servers)\s*[.=]`)
	codexSubRe    = regexp.MustCompile(`^\[\[?\s*mcp_servers\s*\.\s*("rumpty"|'rumpty'|rumpty)\s*\.`)
)

// mergeCodexTOML edits config.toml as text so comments and layout survive, then
// parses the result to prove only [mcp_servers.rumpty] changed.
func mergeCodexTOML(old []byte, force bool) ([]byte, mcpChange, error) {
	before, err := decodeTOML(old)
	if err != nil {
		return nil, 0, err
	}
	cur, exists := codexRumpty(before)
	change := mcpAdded
	var out string
	switch {
	case exists && reflect.DeepEqual(cur, codexEntry):
		return old, mcpUnchanged, nil
	case exists && !force:
		return nil, 0, errMCPConflict
	case exists:
		change = mcpReplaced
		lines := strings.SplitAfter(string(old), "\n")
		start := -1
		for i, l := range lines {
			if codexHeaderRe.MatchString(strings.TrimSpace(l)) {
				start = i
				break
			}
		}
		if start < 0 {
			return nil, 0, errors.New("rumpty is defined in a form rumpty can't safely replace; edit [mcp_servers.rumpty] by hand")
		}
		end := len(lines)
		for i := start + 1; i < len(lines); i++ {
			t := strings.TrimSpace(lines[i])
			if strings.HasPrefix(t, "[") && !codexSubRe.MatchString(t) {
				end = i
				break
			}
		}
		// Blank and comment lines just above the next table belong to it.
		for end-1 > start {
			t := strings.TrimSpace(lines[end-1])
			if t != "" && !strings.HasPrefix(t, "#") {
				break
			}
			end--
		}
		rest := strings.Join(lines[end:], "")
		if rest != "" && strings.TrimSpace(lines[end]) != "" {
			rest = "\n" + rest
		}
		out = strings.Join(lines[:start], "") + codexBlock + rest
	case codexKeyRe.Match(old):
		// TOML forbids extending an inline or dotted-key table with a header,
		// but not every parser enforces it.
		return nil, 0, errors.New("mcp_servers is defined inline; add [mcp_servers.rumpty] by hand (see rumpty mcp codex)")
	default:
		out = string(old)
		if out != "" {
			if !strings.HasSuffix(out, "\n") {
				out += "\n"
			}
			out += "\n"
		}
		out += codexBlock
	}

	after, err := decodeTOML([]byte(out))
	if err == nil {
		got, _ := codexRumpty(after)
		if reflect.DeepEqual(got, codexEntry) && reflect.DeepEqual(withoutRumpty(before), withoutRumpty(after)) {
			return []byte(out), change, nil
		}
	}
	return nil, 0, errors.New("couldn't add [mcp_servers.rumpty] without changing other settings; add it by hand (see rumpty mcp codex)")
}

func decodeTOML(data []byte) (map[string]any, error) {
	doc := map[string]any{}
	if err := toml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("not valid TOML (%v); rumpty won't rewrite it", err)
	}
	return doc, nil
}

func codexRumpty(doc map[string]any) (any, bool) {
	servers, ok := doc["mcp_servers"].(map[string]any)
	if !ok {
		return nil, false
	}
	v, ok := servers[mcpServerName]
	return v, ok
}

func withoutRumpty(doc map[string]any) map[string]any {
	out := make(map[string]any, len(doc))
	for k, v := range doc {
		out[k] = v
	}
	if servers, ok := doc["mcp_servers"].(map[string]any); ok {
		s := make(map[string]any, len(servers))
		for k, v := range servers {
			if k != mcpServerName {
				s[k] = v
			}
		}
		if len(s) == 0 {
			delete(out, "mcp_servers")
		} else {
			out["mcp_servers"] = s
		}
	}
	return out
}

// writeFileAtomic writes via a temp file and rename in the target's directory.
// It follows a symlink to its target and keeps an existing file's mode.
func writeFileAtomic(path string, data []byte, newMode os.FileMode) error {
	target, mode := path, newMode
	if fi, err := os.Lstat(path); err == nil {
		if fi.Mode()&os.ModeSymlink != 0 {
			if target, err = filepath.EvalSymlinks(path); err != nil {
				return fmt.Errorf("resolve %s: %w", path, err)
			}
			if fi, err = os.Stat(target); err != nil {
				return err
			}
		}
		mode = fi.Mode().Perm()
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	dir := filepath.Dir(target)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(target)+".rumpty-*")
	if err != nil {
		return fmt.Errorf("write %s: %w", target, err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write %s: %w", target, err)
	}
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write %s: %w", target, err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write %s: %w", target, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("write %s: %w", target, err)
	}
	if err := os.Rename(tmp.Name(), target); err != nil { //nolint:gosec // target is the client config path rumpty chose, not user input
		return fmt.Errorf("write %s: %w", target, err)
	}
	return nil
}

// lineDiff returns the changed lines of a and b with one line of context.
func lineDiff(a, b string) string {
	al, bl := splitLines(a), splitLines(b)
	n, m := len(al), len(bl)
	if n*m > 4_000_000 {
		return "(file too large to show a diff)\n"
	}
	lcs := make([][]int, n+1)
	for i := range lcs {
		lcs[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if al[i] == bl[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else {
				lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
			}
		}
	}
	type op struct {
		kind byte
		line string
	}
	var ops []op
	i, j := 0, 0
	for i < n || j < m {
		switch {
		case i < n && j < m && al[i] == bl[j]:
			ops = append(ops, op{' ', al[i]})
			i++
			j++
		case i < n && (j == m || lcs[i+1][j] >= lcs[i][j+1]):
			ops = append(ops, op{'-', al[i]})
			i++
		default:
			ops = append(ops, op{'+', bl[j]})
			j++
		}
	}
	const contextLines = 1
	show := make([]bool, len(ops))
	for k, o := range ops {
		if o.kind == ' ' {
			continue
		}
		for d := k - contextLines; d <= k+contextLines; d++ {
			if d >= 0 && d < len(ops) {
				show[d] = true
			}
		}
	}
	var sb strings.Builder
	skipped := false
	for k, o := range ops {
		if !show[k] {
			skipped = true
			continue
		}
		if skipped {
			sb.WriteString("  ...\n")
			skipped = false
		}
		sb.WriteByte(o.kind)
		sb.WriteByte(' ')
		sb.WriteString(redactSecrets(o.line))
		sb.WriteByte('\n')
	}
	if skipped {
		sb.WriteString("  ...\n")
	}
	return sb.String()
}

var secretValueRe = regexp.MustCompile(`(?i)("?[\w.-]*(authori[sz]ation|token|secret|password|api[_-]?key)[\w.-]*"?\s*[:=]\s*)"((?:[^"\\]|\\.)*)"`)

// redactSecrets hides values of secret-looking keys, such as other servers'
// headers, so dry-run output is safe to paste. Variable references stay.
func redactSecrets(line string) string {
	return secretValueRe.ReplaceAllStringFunc(line, func(m string) string {
		sub := secretValueRe.FindStringSubmatch(m)
		if strings.Contains(sub[3], "${") || sub[3] == config.EnvToken {
			return m
		}
		return sub[1] + `"<redacted>"`
	})
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
}
