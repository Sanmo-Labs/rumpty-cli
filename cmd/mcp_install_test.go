package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloverworks-hq/rumpty-cli/internal/config"
)

const testSecret = "rumpty_secret_value"

type installEnv struct {
	home, project string
}

func newInstallEnv(t *testing.T) installEnv {
	t.Helper()
	isolateCreds(t)
	env := installEnv{home: t.TempDir(), project: t.TempDir()}
	t.Setenv("HOME", env.home)
	t.Setenv("CODEX_HOME", "")
	t.Setenv(config.EnvToken, testSecret)
	t.Chdir(env.project)
	return env
}

func runInstall(t *testing.T, args ...string) (string, error) {
	t.Helper()
	return runMCP(t, append([]string{"install"}, args...)...)
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(b)
}

func assertNoSecretOnDisk(t *testing.T, dirs ...string) {
	t.Helper()
	for _, dir := range dirs {
		_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
			require.NoError(t, err)
			if !d.IsDir() {
				assert.NotContains(t, readFile(t, p), testSecret, p)
			}
			return nil
		})
	}
}

type jsonCase struct {
	client, scope string
	path          func(installEnv) string
	serversKey    string
	entry         json.RawMessage
}

var jsonCases = []jsonCase{
	{"claude-code", "project", func(e installEnv) string { return filepath.Join(e.project, ".mcp.json") }, "mcpServers", claudeCodeEntry},
	{"cursor", "user", func(e installEnv) string { return filepath.Join(e.home, ".cursor", "mcp.json") }, "mcpServers", cursorEntry},
	{"cursor", "project", func(e installEnv) string { return filepath.Join(e.project, ".cursor", "mcp.json") }, "mcpServers", cursorEntry},
	{"vscode", "project", func(e installEnv) string { return filepath.Join(e.project, ".vscode", "mcp.json") }, "servers", vscodeEntry},
	{"gemini", "user", func(e installEnv) string { return filepath.Join(e.home, ".gemini", "settings.json") }, "mcpServers", geminiEntry},
	{"gemini", "project", func(e installEnv) string { return filepath.Join(e.project, ".gemini", "settings.json") }, "mcpServers", geminiEntry},
}

func decodeServers(t *testing.T, data, key string) map[string]any {
	t.Helper()
	var doc map[string]any
	require.NoError(t, json.Unmarshal([]byte(data), &doc), data)
	servers, ok := doc[key].(map[string]any)
	require.True(t, ok, "missing %q in %s", key, data)
	return servers
}

func entryValue(t *testing.T, raw json.RawMessage) any {
	t.Helper()
	var v any
	require.NoError(t, json.Unmarshal(raw, &v))
	return v
}

func writeTestFile(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o750))
	require.NoError(t, os.WriteFile(path, []byte(content), mode))
}

func TestMCPInstallJSONClients(t *testing.T) {
	for _, tc := range jsonCases {
		args := []string{tc.client, "--scope", tc.scope}
		t.Run(tc.client+"/"+tc.scope, func(t *testing.T) {
			t.Run("fresh file", func(t *testing.T) {
				env := newInstallEnv(t)
				out, err := runInstall(t, args...)
				require.NoError(t, err)
				path := tc.path(env)
				assert.Contains(t, out, path)
				data := readFile(t, path)
				assert.Equal(t, entryValue(t, tc.entry), decodeServers(t, data, tc.serversKey)["rumpty"])
				if tc.client == "vscode" {
					assert.Contains(t, data, `"id": "rumpty-key"`)
				}
				fi, err := os.Stat(path)
				require.NoError(t, err)
				want := os.FileMode(0o644)
				if tc.scope == "user" {
					want = 0o600
				}
				assert.Equal(t, want, fi.Mode().Perm())
				assertNoSecretOnDisk(t, env.home, env.project)
			})

			t.Run("merge keeps other servers and keys", func(t *testing.T) {
				env := newInstallEnv(t)
				path := tc.path(env)
				existing := `{
    "theme": "dark",
    "` + tc.serversKey + `": {
        "other": { "url": "https://example.com/mcp", "big": 12345678901234567890 }
    },
    "zzz": [1, 2]
}
`
				writeTestFile(t, path, existing, 0o640)
				_, err := runInstall(t, args...)
				require.NoError(t, err)
				data := readFile(t, path)
				servers := decodeServers(t, data, tc.serversKey)
				assert.Contains(t, servers, "other")
				assert.Equal(t, entryValue(t, tc.entry), servers["rumpty"])
				assert.Contains(t, data, "12345678901234567890", "numbers keep their exact text")
				assert.Less(t, strings.Index(data, `"theme"`), strings.Index(data, `"zzz"`), "key order is kept")
				assert.Contains(t, data, "\n    \"theme\"", "indent is kept")
				fi, err := os.Stat(path)
				require.NoError(t, err)
				assert.Equal(t, os.FileMode(0o640), fi.Mode().Perm(), "mode is kept")
				assertNoSecretOnDisk(t, env.home, env.project)
			})

			t.Run("identical entry is a no-op", func(t *testing.T) {
				env := newInstallEnv(t)
				_, err := runInstall(t, args...)
				require.NoError(t, err)
				before := readFile(t, tc.path(env))
				out, err := runInstall(t, args...)
				require.NoError(t, err)
				assert.Contains(t, out, "nothing changed")
				assert.Equal(t, before, readFile(t, tc.path(env)))
			})

			t.Run("conflicting entry needs --force", func(t *testing.T) {
				env := newInstallEnv(t)
				path := tc.path(env)
				existing := `{"` + tc.serversKey + `": {"rumpty": {"url": "https://old.example.com"}, "other": {"url": "x"}}}`
				writeTestFile(t, path, existing, 0o644)
				_, err := runInstall(t, args...)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "--force")
				assert.Equal(t, existing, readFile(t, path))

				out, err := runInstall(t, append(args, "--force")...)
				require.NoError(t, err)
				assert.Contains(t, out, "Replaced")
				servers := decodeServers(t, readFile(t, path), tc.serversKey)
				assert.Equal(t, entryValue(t, tc.entry), servers["rumpty"])
				assert.Contains(t, servers, "other")
			})

			t.Run("invalid JSON is refused untouched", func(t *testing.T) {
				for _, bad := range []string{
					"{\n  // my servers\n  \"" + tc.serversKey + "\": {}\n}\n",
					`{"` + tc.serversKey + `": {},}`,
					`[1, 2]`,
					`{"` + tc.serversKey + `": []}`,
					`{"` + tc.serversKey + `": {"a": {}}, "` + tc.serversKey + `": {"b": {}}}`,
					`{"` + tc.serversKey + `": {"rumpty": ` + string(tc.entry) + `, "rumpty": {"url": "evil"}}}`,
					"\xef\xbb\xbf{}",
				} {
					env := newInstallEnv(t)
					path := tc.path(env)
					writeTestFile(t, path, bad, 0o644)
					_, err := runInstall(t, append(args, "--force")...)
					require.Error(t, err, bad)
					assert.Equal(t, bad, readFile(t, path))
				}
			})

			t.Run("dry run writes nothing", func(t *testing.T) {
				env := newInstallEnv(t)
				out, err := runInstall(t, append(args, "--dry-run")...)
				require.NoError(t, err)
				assert.Contains(t, out, "Dry run")
				assert.Contains(t, out, "+ ")
				assert.Contains(t, out, mcpURL)
				assert.NotContains(t, out, testSecret)
				assert.NoFileExists(t, tc.path(env))

				path := tc.path(env)
				existing := `{"` + tc.serversKey + `": {"rumpty": {"url": "old"}}}`
				writeTestFile(t, path, existing, 0o644)
				_, err = runInstall(t, append(args, "--dry-run", "--force")...)
				require.NoError(t, err)
				assert.Equal(t, existing, readFile(t, path))
			})
		})
	}
}

func TestMCPInstallVSCodeKeepsExistingInput(t *testing.T) {
	env := newInstallEnv(t)
	path := filepath.Join(env.project, ".vscode", "mcp.json")
	existing := `{"inputs": [{"type": "promptString", "id": "rumpty-key", "description": "mine"}, {"id": "other"}], "servers": {}}`
	writeTestFile(t, path, existing, 0o644)
	_, err := runInstall(t, "vscode")
	require.NoError(t, err)
	var doc struct {
		Inputs []map[string]any `json:"inputs"`
	}
	require.NoError(t, json.Unmarshal([]byte(readFile(t, path)), &doc))
	require.Len(t, doc.Inputs, 2)
	assert.Equal(t, "mine", doc.Inputs[0]["description"])
}

func TestMCPInstallFollowsSymlink(t *testing.T) {
	env := newInstallEnv(t)
	real := filepath.Join(env.home, "dotfiles", "mcp.json")
	writeTestFile(t, real, `{"mcpServers": {}}`, 0o600)
	link := filepath.Join(env.home, ".cursor", "mcp.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(link), 0o750))
	require.NoError(t, os.Symlink(real, link))

	_, err := runInstall(t, "cursor")
	require.NoError(t, err)
	fi, err := os.Lstat(link)
	require.NoError(t, err)
	assert.NotZero(t, fi.Mode()&os.ModeSymlink, "the symlink is kept")
	assert.Contains(t, decodeServers(t, readFile(t, real), "mcpServers"), "rumpty")
	fi, err = os.Stat(real)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), fi.Mode().Perm())
}

func TestMCPInstallNullServersAndInputBytes(t *testing.T) {
	env := newInstallEnv(t)
	path := filepath.Join(env.project, ".vscode", "mcp.json")
	writeTestFile(t, path, `{"inputs": [{"id": "a", "description": "<x> & y"}], "servers": null}`, 0o644)
	_, err := runInstall(t, "vscode")
	require.NoError(t, err)
	data := readFile(t, path)
	assert.Contains(t, data, `"<x> & y"`)
	assert.Contains(t, decodeServers(t, data, "servers"), "rumpty")
}

func TestMCPInstallDryRunRedactsOtherSecrets(t *testing.T) {
	env := newInstallEnv(t)
	path := filepath.Join(env.home, ".cursor", "mcp.json")
	existing := `{"mcpServers": {"other": {"url": "x", "headers": {"Authorization": "Bearer other_secret"}, "env": {"GITHUB_TOKEN": "ghp_other"}}}}`
	writeTestFile(t, path, existing, 0o600)
	out, err := runInstall(t, "cursor", "--dry-run")
	require.NoError(t, err)
	assert.NotContains(t, out, "other_secret")
	assert.NotContains(t, out, "ghp_other")
	assert.Contains(t, out, "<redacted>")
	assert.Contains(t, out, "${env:RUMPTY_API_KEY}")
	assert.Equal(t, existing, readFile(t, path))
}

func codexPath(env installEnv) string { return filepath.Join(env.home, ".codex", "config.toml") }

func TestMCPInstallCodex(t *testing.T) {
	existing := `# my codex config
model = "o3"

[mcp_servers.other]
url = "https://example.com/mcp" # trailing comment

[profiles.fast]
model = "mini"
`
	conflicting := `model = "o3"

[mcp_servers.rumpty]
url = "https://old.example.com"
http_headers = { "Authorization" = "Bearer pasted" }

[mcp_servers.rumpty.env]
A = "b"

# about profiles
[profiles.fast]
model = "mini"
`
	tests := []struct {
		name      string
		existing  string
		args      []string
		wantErr   string
		wantSame  bool
		wantParts []string
	}{
		{name: "fresh file", wantParts: []string{codexBlock}},
		{name: "merge keeps comments and tables", existing: existing, wantParts: []string{existing, "\n" + codexBlock}},
		{name: "missing trailing newline", existing: `model = "o3"`, wantParts: []string{"model = \"o3\"\n\n" + codexBlock}},
		{name: "identical entry is a no-op", existing: "model = \"o3\"\n\n" + codexBlock, wantSame: true},
		{name: "conflict refused", existing: conflicting, wantErr: "--force", wantSame: true},
		{name: "conflict replaced with --force", existing: conflicting, args: []string{"--force"}, wantParts: []string{
			"model = \"o3\"\n\n" + codexBlock + "\n# about profiles\n[profiles.fast]\nmodel = \"mini\"\n",
		}},
		{name: "invalid TOML refused", existing: "model = \n[[", args: []string{"--force"}, wantErr: "TOML", wantSame: true},
		{name: "inline table refused", existing: `mcp_servers = { other = { url = "x" } }` + "\n", wantErr: "by hand", wantSame: true},
		{name: "dotted root keys refused", existing: `mcp_servers.other.url = "x"` + "\n", wantErr: "by hand", wantSame: true},
		{name: "existing mcp_servers table extended", existing: "[mcp_servers]\nother.url = \"x\"\n", wantParts: []string{"[mcp_servers]\nother.url = \"x\"\n\n" + codexBlock}},
		{name: "dotted form refused even with --force", existing: "[mcp_servers]\nrumpty.url = \"old\"\n", args: []string{"--force"}, wantErr: "by hand", wantSame: true},
		{name: "dry run writes nothing", existing: existing, args: []string{"--dry-run"}, wantSame: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := newInstallEnv(t)
			path := codexPath(env)
			defer func() {
				if tc.wantErr == "" && !tc.wantSame {
					got := readFile(t, path)
					assert.NotContains(t, got, "old.example.com")
					assert.NotContains(t, got, "rumpty.env")
					assert.Equal(t, 1, strings.Count(got, "[mcp_servers.rumpty]"))
				}
			}()
			if tc.existing != "" {
				writeTestFile(t, path, tc.existing, 0o644)
			}
			out, err := runInstall(t, append([]string{"codex"}, tc.args...)...)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
			} else {
				require.NoError(t, err)
			}
			if tc.wantSame {
				if tc.existing == "" {
					assert.NoFileExists(t, path)
				} else {
					assert.Equal(t, tc.existing, readFile(t, path))
				}
			}
			if len(tc.wantParts) > 0 {
				got := readFile(t, path)
				for _, p := range tc.wantParts {
					assert.Contains(t, got, p)
				}
				assert.Contains(t, out, path)
			}
			assertNoSecretOnDisk(t, env.home, env.project)
		})
	}
}

func TestMCPInstallCodexHome(t *testing.T) {
	env := newInstallEnv(t)
	dir := filepath.Join(env.home, "custom-codex")
	t.Setenv("CODEX_HOME", dir)
	_, err := runInstall(t, "codex")
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(dir, "config.toml"))
}

type fakeClaude struct {
	calls [][]string
	err   error
}

func stubClaude(t *testing.T, found bool) *fakeClaude {
	t.Helper()
	f := &fakeClaude{}
	oldLook, oldRun := mcpLookPath, mcpRunCommand
	t.Cleanup(func() { mcpLookPath, mcpRunCommand = oldLook, oldRun })
	mcpLookPath = func(string) (string, error) {
		if !found {
			return "", errors.New("not found")
		}
		return "/fake/claude", nil
	}
	mcpRunCommand = func(_ context.Context, _, _ io.Writer, name string, args ...string) error {
		f.calls = append(f.calls, append([]string{name}, args...))
		if args[1] == "add" {
			return f.err
		}
		return nil
	}
	return f
}

func TestMCPInstallClaudeCodeUser(t *testing.T) {
	t.Run("runs claude mcp add with an env reference", func(t *testing.T) {
		env := newInstallEnv(t)
		f := stubClaude(t, true)
		out, err := runInstall(t, "claude-code", "--scope", "user")
		require.NoError(t, err)
		require.Len(t, f.calls, 1)
		assert.Equal(t, append([]string{"/fake/claude"}, claudeCodeUserArgs()...), f.calls[0])
		assert.Contains(t, f.calls[0], "Authorization: Bearer ${RUMPTY_API_KEY}")
		assert.NotContains(t, strings.Join(f.calls[0], " "), testSecret)
		assert.Contains(t, out, "Added rumpty")
		assertNoSecretOnDisk(t, env.home, env.project)
	})
	t.Run("force removes first", func(t *testing.T) {
		newInstallEnv(t)
		f := stubClaude(t, true)
		_, err := runInstall(t, "claude-code", "--scope", "user", "--force")
		require.NoError(t, err)
		require.Len(t, f.calls, 2)
		assert.Equal(t, "remove", f.calls[0][2])
	})
	t.Run("add failure suggests --force", func(t *testing.T) {
		newInstallEnv(t)
		f := stubClaude(t, true)
		f.err = errors.New("exit status 1")
		_, err := runInstall(t, "claude-code", "--scope", "user")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--force")
	})
	t.Run("force add failure says how to finish", func(t *testing.T) {
		newInstallEnv(t)
		f := stubClaude(t, true)
		f.err = errors.New("exit status 1")
		out, err := runInstall(t, "claude-code", "--scope", "user", "--force")
		require.Error(t, err)
		assert.Contains(t, out, "may already be removed")
		assert.Contains(t, out, "claude mcp add")
	})
	t.Run("missing binary prints the command", func(t *testing.T) {
		newInstallEnv(t)
		f := stubClaude(t, false)
		out, err := runInstall(t, "claude-code", "--scope", "user")
		require.Error(t, err)
		assert.Empty(t, f.calls)
		assert.Contains(t, out, "--header 'Authorization: Bearer ${RUMPTY_API_KEY}'")
		assert.NotContains(t, out, testSecret)
	})
	t.Run("dry run runs nothing", func(t *testing.T) {
		newInstallEnv(t)
		f := stubClaude(t, true)
		out, err := runInstall(t, "claude-code", "--scope", "user", "--dry-run")
		require.NoError(t, err)
		assert.Empty(t, f.calls)
		assert.Contains(t, out, "claude mcp add")
	})
}

func TestMCPInstallManualClients(t *testing.T) {
	for _, id := range []string{"zed", "claude-desktop"} {
		t.Run(id, func(t *testing.T) {
			env := newInstallEnv(t)
			out, err := runInstall(t, id)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "never writes API keys")
			assert.Contains(t, out, mcpURL, "prints the setup steps")
			assert.NotContains(t, out, testSecret)
			entries, err := os.ReadDir(env.home)
			require.NoError(t, err)
			assert.Empty(t, entries)
		})
	}
}

func TestMCPInstallUsageErrors(t *testing.T) {
	for _, args := range [][]string{
		{"notepad"},
		{"cursor", "--scope", "global"},
		{"vscode", "--scope", "user"},
		{"codex", "--scope", "project"},
	} {
		newInstallEnv(t)
		_, err := runInstall(t, args...)
		require.Error(t, err, args)
		assert.True(t, config.IsUsageError(err), "%v: %v", args, err)
	}
	newInstallEnv(t)
	_, err := runInstall(t)
	require.Error(t, err)
}

func TestMCPInstallEntriesMatchPrintedSetup(t *testing.T) {
	for id, entry := range map[string]json.RawMessage{"claude-code": claudeCodeEntry, "cursor": cursorEntry, "vscode": vscodeEntry} {
		c, ok := findMCPClient(id)
		require.True(t, ok)
		var compact bytes.Buffer
		require.NoError(t, json.Compact(&compact, entry))
		var v map[string]any
		require.NoError(t, json.Unmarshal(entry, &v))
		headers, ok := v["headers"].(map[string]any)
		require.True(t, ok)
		assert.Contains(t, c.Setup, headers["Authorization"], id)
	}
	c, _ := findMCPClient("codex")
	assert.Contains(t, c.Setup, strings.TrimSuffix(codexBlock, "\n"))
	c, _ = findMCPClient("vscode")
	assert.Contains(t, c.Setup, `"id": "rumpty-key"`)
}

func TestLineDiff(t *testing.T) {
	assert.Equal(t, "+ a\n+ b\n", lineDiff("", "a\nb\n"))
	assert.Equal(t, "  ...\n  c\n- d\n+ D\n  e\n  ...\n", lineDiff("a\nb\nc\nd\ne\nf\ng\n", "a\nb\nc\nD\ne\nf\ng\n"))
	assert.Equal(t, `"Authorization": "<redacted>", "x": "y"`, redactSecrets(`"Authorization": "Bearer abc", "x": "y"`))
	assert.Equal(t, `bearer_token_env_var = "RUMPTY_API_KEY"`, redactSecrets(`bearer_token_env_var = "RUMPTY_API_KEY"`))
	assert.Equal(t, `api_key = "<redacted>"`, redactSecrets(`api_key = "sk-1"`))
}
