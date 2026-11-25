package commands

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/config"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
)

const (
	mcpURL       = "https://mcp.rumptycloud.com/mcp"
	mcpKeysURL   = "https://console.rumptycloud.com/profile?tab=api-keys"
	mcpDocsURL   = "https://docs.rumptycloud.com/mcp"
	mcpTransport = "streamable-http"
)

type mcpClient struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Setup string `json:"setup"`
	Check string `json:"check"`
	// UsesEnv is true when the setup reads the key from $RUMPTY_API_KEY.
	UsesEnv bool `json:"uses_env"`
}

// mcpClients holds setup steps for each client. Where the client can expand
// it, the key is read from $RUMPTY_API_KEY, so the steps never contain a key.
var mcpClients = []mcpClient{
	{
		ID:      "claude-code",
		UsesEnv: true,
		Name:    "Claude Code",
		Setup: `Add to .mcp.json in your project (Claude Code expands ${RUMPTY_API_KEY} when it starts):

{
  "mcpServers": {
    "rumpty": {
      "type": "http",
      "url": "` + mcpURL + `",
      "headers": { "Authorization": "Bearer ${RUMPTY_API_KEY}" }
    }
  }
}

Or add it for every project. Run this where RUMPTY_API_KEY is set; your shell fills
in the key, so it is saved in Claude Code's config:

claude mcp add --transport http --scope user rumpty ` + mcpURL + ` \
  --header "Authorization: Bearer $RUMPTY_API_KEY"`,
		Check: "Run `claude mcp list`; rumpty should show Connected.",
	},
	{
		ID:      "cursor",
		UsesEnv: true,
		Name:    "Cursor",
		Setup: `Add to ~/.cursor/mcp.json (all projects) or .cursor/mcp.json (one project):

{
  "mcpServers": {
    "rumpty": {
      "url": "` + mcpURL + `",
      "headers": { "Authorization": "Bearer ${env:RUMPTY_API_KEY}" }
    }
  }
}

Cursor reads RUMPTY_API_KEY from the environment it was started in. If you open
Cursor from the dock, set the variable where GUI apps see it, or start Cursor from
a terminal.`,
		Check: "Open the Output panel, pick MCP Logs, and check rumpty lists its tools.",
	},
	{
		ID:   "vscode",
		Name: "VS Code",
		Setup: `Add to .vscode/mcp.json. VS Code asks for the key once and stores it securely:

{
  "inputs": [
    { "type": "promptString", "id": "rumpty-key", "description": "RumptyCloud API key", "password": true }
  ],
  "servers": {
    "rumpty": {
      "type": "http",
      "url": "` + mcpURL + `",
      "headers": { "Authorization": "Bearer ${input:rumpty-key}" }
    }
  }
}`,
		Check: "Run MCP: List Servers from the command palette and start rumpty.",
	},
	{
		ID:      "codex",
		UsesEnv: true,
		Name:    "OpenAI Codex CLI",
		Setup: `codex mcp add rumpty --url ` + mcpURL + ` --bearer-token-env-var RUMPTY_API_KEY

Or in ~/.codex/config.toml:

[mcp_servers.rumpty]
url = "` + mcpURL + `"
bearer_token_env_var = "RUMPTY_API_KEY"`,
		Check: "Run `codex mcp list`, or /mcp inside Codex.",
	},
	{
		ID:      "gemini",
		UsesEnv: true,
		Name:    "Gemini CLI",
		Setup: `Run this where RUMPTY_API_KEY is set; your shell fills in the key, so it is saved
in Gemini's settings.json:

gemini mcp add --scope user --transport http \
  --header "Authorization: Bearer $RUMPTY_API_KEY" rumpty ` + mcpURL,
		Check: "Run `gemini mcp list`; rumpty should show Connected.",
	},
	{
		ID:   "zed",
		Name: "Zed",
		Setup: `Zed does not read environment variables here, so paste the key itself. Add to
settings.json:

"context_servers": {
  "rumpty": {
    "url": "` + mcpURL + `",
    "headers": { "Authorization": "Bearer <your API key>" }
  }
}`,
		Check: "In Settings, AI, MCP Servers, the dot next to rumpty turns green.",
	},
	{
		ID:   "claude-desktop",
		Name: "Claude Desktop",
		Setup: `Claude Desktop reaches remote servers with custom headers through mcp-remote
(needs Node.js). Add to claude_desktop_config.json and paste your key:

{
  "mcpServers": {
    "rumpty": {
      "command": "npx",
      "args": ["-y", "mcp-remote", "` + mcpURL + `", "--header", "Authorization:${AUTH_HEADER}"],
      "env": { "AUTH_HEADER": "Bearer <your API key>" }
    }
  }
}

Keep "Authorization:${AUTH_HEADER}" without spaces; the key goes in env.`,
		Check: "Restart Claude Desktop; rumpty appears under the tools menu.",
	},
}

type mcpInfo struct {
	URL        string      `json:"url"`
	Transport  string      `json:"transport"`
	AuthHeader string      `json:"auth_header"`
	AuthScheme string      `json:"auth_scheme"`
	KeyEnv     string      `json:"key_env"`
	KeySet     bool        `json:"key_set"`
	CreateKey  string      `json:"create_key_url"`
	Docs       string      `json:"docs_url"`
	Clients    []mcpClient `json:"clients"`
}

func newMCPCmd(rt *app.Runtime) *cobra.Command {
	var output string
	ids := make([]string, 0, len(mcpClients))
	for _, c := range mcpClients {
		ids = append(ids, c.ID)
	}
	cmd := &cobra.Command{
		Use:   "mcp [client]",
		Short: "Show how to connect an AI client to Rumpty MCP",
		Long: "Print the Rumpty MCP server details and setup steps for an MCP client.\n\n" +
			"Clients: " + strings.Join(ids, ", ") + ".\n\n" +
			"Where a client supports it, the steps read your API key from $" + config.EnvToken + ". This command never prints a key.\n\n" +
			"To write the config for you, use rumpty mcp install <client>.",
		Example: `  rumpty mcp
  rumpty mcp cursor
  rumpty mcp claude-code
  rumpty mcp -o json
  rumpty mcp install cursor`,
		Args:      cobra.MaximumNArgs(1),
		ValidArgs: ids,
		RunE: func(cmd *cobra.Command, args []string) error {
			format, err := outputFormat(output)
			if err != nil {
				return err
			}
			info := newMCPInfo()
			out := rt.Streams.Out
			if len(args) == 1 {
				client, ok := findMCPClient(args[0])
				if !ok {
					return config.NewUsageError("unknown client %q; use one of: %s", args[0], strings.Join(ids, ", "))
				}
				if format == term.OutputJSON {
					info.Clients = []mcpClient{client}
					return term.WriteJSON(out, info)
				}
				printMCPClient(out, &info, client)
				return nil
			}
			if format == term.OutputJSON {
				return term.WriteJSON(out, info)
			}
			printMCPOverview(out, &info)
			return nil
		},
	}
	addOutputFlag(cmd, &output)
	cmd.AddCommand(newMCPInstallCmd(rt))
	return cmd
}

func newMCPInfo() mcpInfo {
	return mcpInfo{
		URL:        mcpURL,
		Transport:  mcpTransport,
		AuthHeader: "Authorization",
		AuthScheme: "Bearer",
		KeyEnv:     config.EnvToken,
		KeySet:     strings.TrimSpace(os.Getenv(config.EnvToken)) != "",
		CreateKey:  mcpKeysURL,
		Docs:       mcpDocsURL,
		Clients:    mcpClients,
	}
}

func findMCPClient(id string) (mcpClient, bool) {
	id = strings.ToLower(strings.TrimSpace(id))
	for _, c := range mcpClients {
		if c.ID == id {
			return c, true
		}
	}
	return mcpClient{}, false
}

func printMCPKeyStep(w io.Writer, info *mcpInfo, usesEnv bool) {
	if !usesEnv {
		fmt.Fprintf(w, "Create an API key at %s\n", info.CreateKey)
		return
	}
	if info.KeySet {
		fmt.Fprintf(w, "$%s is set in this shell. Make sure your client is started with it too.\n", info.KeyEnv)
		return
	}
	fmt.Fprintf(w, "Create an API key at %s\nthen export it before starting your client:\n\n  export %s=<your API key>\n", info.CreateKey, info.KeyEnv)
}

func printMCPOverview(w io.Writer, info *mcpInfo) {
	fmt.Fprintf(w, "Rumpty MCP\n\n")
	fmt.Fprintf(w, "  URL        %s\n", info.URL)
	fmt.Fprintf(w, "  Transport  Streamable HTTP\n")
	fmt.Fprintf(w, "  Auth       %s: %s <API key>\n\n", info.AuthHeader, info.AuthScheme)
	printMCPKeyStep(w, info, true)
	fmt.Fprintf(w, "\nThe agent can do what the key's permissions and your workspace role allow.\n\n")
	fmt.Fprintf(w, "Setup steps for a client:\n\n")
	for _, c := range info.Clients {
		fmt.Fprintf(w, "  rumpty mcp %-15s %s\n", c.ID, c.Name)
	}
	fmt.Fprintf(w, "\nDocs: %s\n", info.Docs)
}

func printMCPClient(w io.Writer, info *mcpInfo, c mcpClient) {
	fmt.Fprintf(w, "Connect %s to Rumpty MCP\n\n", c.Name)
	printMCPKeyStep(w, info, c.UsesEnv)
	fmt.Fprintf(w, "\n%s\n\n%s\n\nDocs: %s\n", c.Setup, c.Check, info.Docs)
}
