package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"

	"github.com/amarbel-llc/purse-first/libs/go-mcp/transport"
	"github.com/amarbel-llc/lux/internal/capabilities"
	"github.com/amarbel-llc/lux/internal/config"
	"github.com/amarbel-llc/lux/internal/config/filetype"
	"github.com/amarbel-llc/lux/internal/control"
	"github.com/amarbel-llc/lux/internal/formatter"
	"github.com/amarbel-llc/lux/internal/mcp"
	"github.com/amarbel-llc/lux/internal/server"
	"github.com/amarbel-llc/lux/internal/subprocess"
	"github.com/amarbel-llc/lux/internal/tools"
	luxtransport "github.com/amarbel-llc/lux/internal/transport"
)

var rootCmd = &cobra.Command{
	Use:   "lux",
	Short: "Lux: LSP Multiplexer",
	Long:  `Lux multiplexes LSP requests to multiple language servers based on file type.`,
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the LSP server",
	Long:  `Start the Lux LSP server, reading from stdin and writing to stdout.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		srv, err := server.New(cfg)
		if err != nil {
			return fmt.Errorf("creating server: %w", err)
		}

		return srv.Run(cmd.Context())
	},
}

var addBinary string
var addConfigPath string

var addCmd = &cobra.Command{
	Use:   "add <flake>",
	Short: "Add an LSP from a nix flake",
	Long:  `Add a new LSP to the configuration by bootstrapping it to discover capabilities.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		flake := args[0]
		return capabilities.Bootstrap(cmd.Context(), flake, addBinary, addConfigPath)
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured filetypes and their routing",
	Long:  `List all filetype configs showing extensions, LSP, formatters, and formatter mode.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		filetypes, err := filetype.LoadMerged()
		if err != nil {
			return fmt.Errorf("loading filetype configs: %w", err)
		}

		if len(filetypes) == 0 {
			fmt.Println("No filetype configs found")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "Filetype\tExtensions\tLSP\tFormatters\tMode")
		for _, ft := range filetypes {
			exts := strings.Join(ft.Extensions, ", ")
			lsp := ft.LSP
			if lsp == "" {
				lsp = "-"
			}
			fmts := strings.Join(ft.Formatters, ", ")
			if fmts == "" {
				fmts = "-"
			}
			mode := "-"
			if len(ft.Formatters) > 0 {
				mode = ft.EffectiveFormatterMode()
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", ft.Name, exts, lsp, fmts, mode)
		}
		w.Flush()
		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of running LSPs",
	Long:  `Connect to a running Lux server and show the status of all LSPs.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		client, err := control.NewClient(cfg.SocketPath())
		if err != nil {
			return fmt.Errorf("connecting to server: %w", err)
		}
		defer client.Close()

		return client.Status(os.Stdout)
	},
}

var startCmd = &cobra.Command{
	Use:   "start <name>",
	Short: "Eagerly start an LSP",
	Long:  `Start a configured LSP without waiting for a matching request.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		client, err := control.NewClient(cfg.SocketPath())
		if err != nil {
			return fmt.Errorf("connecting to server: %w", err)
		}
		defer client.Close()

		return client.Start(args[0])
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop <name>",
	Short: "Stop a running LSP",
	Long:  `Stop a running LSP to free resources.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		client, err := control.NewClient(cfg.SocketPath())
		if err != nil {
			return fmt.Errorf("connecting to server: %w", err)
		}
		defer client.Close()

		return client.Stop(args[0])
	},
}

var warmupCmd = &cobra.Command{
	Use:   "warmup [dir]",
	Short: "Pre-start LSPs for a directory",
	Long:  `Scan a directory for matching files and eagerly start the relevant LSPs.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := "."
		if len(args) > 0 {
			dir = args[0]
		}

		absDir, err := filepath.Abs(dir)
		if err != nil {
			return fmt.Errorf("resolving path: %w", err)
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		client, err := control.NewClient(cfg.SocketPath())
		if err != nil {
			return fmt.Errorf("connecting to server: %w", err)
		}
		defer client.Close()

		return client.Warmup(absDir)
	},
}

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Run as MCP server",
	Long:  `Run Lux as an MCP server, exposing LSP capabilities as MCP tools.`,
}

var mcpStdioCmd = &cobra.Command{
	Use:   "stdio",
	Short: "MCP over stdio",
	Long:  `Run MCP server reading from stdin and writing to stdout.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		t := transport.NewStdio(os.Stdin, os.Stdout)
		srv, err := mcp.New(cfg, t)
		if err != nil {
			return fmt.Errorf("creating MCP server: %w", err)
		}

		return srv.Run(cmd.Context())
	},
}

var mcpSSEAddr string

var mcpSSECmd = &cobra.Command{
	Use:   "sse",
	Short: "MCP over SSE",
	Long:  `Run MCP server using Server-Sent Events over HTTP.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		t := luxtransport.NewSSE(mcpSSEAddr)
		srv, err := mcp.New(cfg, t)
		if err != nil {
			return fmt.Errorf("creating MCP server: %w", err)
		}

		t.SetDocumentLifecycle(srv.DocumentManager())

		// Start HTTP server in background
		go func() {
			if err := t.Start(cmd.Context()); err != nil {
				fmt.Fprintf(os.Stderr, "SSE server error: %v\n", err)
			}
		}()

		fmt.Fprintf(os.Stderr, "MCP SSE server listening on %s\n", mcpSSEAddr)
		return srv.Run(cmd.Context())
	},
}

var mcpHTTPAddr string

var mcpHTTPCmd = &cobra.Command{
	Use:   "http",
	Short: "MCP over streamable HTTP",
	Long:  `Run MCP server using streamable HTTP transport.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		t := luxtransport.NewStreamableHTTP(mcpHTTPAddr)
		srv, err := mcp.New(cfg, t)
		if err != nil {
			return fmt.Errorf("creating MCP server: %w", err)
		}

		// Start HTTP server in background
		go func() {
			if err := t.Start(cmd.Context()); err != nil {
				fmt.Fprintf(os.Stderr, "HTTP server error: %v\n", err)
			}
		}()

		fmt.Fprintf(os.Stderr, "MCP HTTP server listening on %s\n", mcpHTTPAddr)
		return srv.Run(cmd.Context())
	},
}

var mcpInstallClaudeCmd = &cobra.Command{
	Use:   "install-claude",
	Short: "Install lux as MCP server in Claude Code",
	Long:  `Register lux as an MCP server using the claude CLI.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		luxPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("getting executable path: %w", err)
		}

		// Remove existing lux MCP server (ignore errors if it doesn't exist)
		removeCmd := exec.CommandContext(
			cmd.Context(),
			"claude", "mcp", "remove", "lux",
		)
		removeCmd.Stdout = os.Stdout
		removeCmd.Stderr = os.Stderr
		removeCmd.Stdin = os.Stdin
		_ = removeCmd.Run() // Ignore error - server may not exist yet

		claudeCmd := exec.CommandContext(
			cmd.Context(),
			"claude", "mcp", "add", "lux",
			"--", luxPath, "mcp", "stdio",
		)
		claudeCmd.Stdout = os.Stdout
		claudeCmd.Stderr = os.Stderr
		claudeCmd.Stdin = os.Stdin

		if err := claudeCmd.Run(); err != nil {
			return fmt.Errorf("running claude mcp add: %w", err)
		}

		return nil
	},
}

var formatStdout bool

var formatCmd = &cobra.Command{
	Use:   "format <file>",
	Short: "Format a file using configured formatters",
	Long:  `Format a file using external formatter programs configured in formatters.toml.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath, err := filepath.Abs(args[0])
		if err != nil {
			return fmt.Errorf("resolving path: %w", err)
		}

		filetypes, err := filetype.LoadMerged()
		if err != nil {
			return fmt.Errorf("loading filetype configs: %w", err)
		}

		fmtCfg, err := config.LoadMergedFormatters()
		if err != nil {
			return fmt.Errorf("loading formatter config: %w", err)
		}

		if err := fmtCfg.Validate(); err != nil {
			return fmt.Errorf("invalid formatter config: %w", err)
		}

		fmtMap := make(map[string]*config.Formatter)
		for i := range fmtCfg.Formatters {
			f := &fmtCfg.Formatters[i]
			if !f.Disabled {
				fmtMap[f.Name] = f
			}
		}

		router, err := formatter.NewRouter(filetypes, fmtMap)
		if err != nil {
			return fmt.Errorf("creating formatter router: %w", err)
		}

		match := router.Match(filePath)
		if match == nil {
			return fmt.Errorf("no formatter configured for %s", filePath)
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("reading file: %w", err)
		}

		executor := subprocess.NewNixExecutor()

		var result *formatter.Result
		switch match.Mode {
		case "chain":
			result, err = formatter.FormatChain(cmd.Context(), match.Formatters, filePath, content, executor)
		case "fallback":
			result, err = formatter.FormatFallback(cmd.Context(), match.Formatters, filePath, content, executor)
		default:
			return fmt.Errorf("unknown formatter mode: %s", match.Mode)
		}
		if err != nil {
			return err
		}

		if formatStdout {
			fmt.Print(result.Formatted)
			return nil
		}

		if !result.Changed {
			return nil
		}

		if err := os.WriteFile(filePath, []byte(result.Formatted), 0644); err != nil {
			return fmt.Errorf("writing file: %w", err)
		}

		return nil
	},
}

var version = "dev"

var genmanCmd = &cobra.Command{
	Use:    "genman <output-dir>",
	Short:  "Generate man pages",
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		header := &doc.GenManHeader{
			Title:   "LUX",
			Section: "1",
			Source:  "lux " + version,
			Manual:  "User Commands",
		}
		return doc.GenManTree(rootCmd, header, args[0])
	},
}

var generatePluginCmd = &cobra.Command{
	Use:    "generate-plugin [output-dir]",
	Short:  "Generate purse-first plugin artifacts",
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := tools.RegisterAll(nil)
		return app.GenerateAll(args[0])
	},
}

func init() {
	formatCmd.Flags().BoolVar(&formatStdout, "stdout", false, "Print formatted output to stdout instead of writing in-place")

	rootCmd.AddCommand(serveCmd)

	addCmd.Flags().StringVarP(&addBinary, "binary", "b", "",
		"Specify custom binary name or path within the flake (e.g., 'rust-analyzer' or 'bin/custom-lsp')")
	addCmd.Flags().StringVar(&addConfigPath, "config-path", "",
		"Write to a custom config file location instead of the default")
	rootCmd.AddCommand(addCmd)

	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(warmupCmd)
	rootCmd.AddCommand(formatCmd)

	mcpCmd.AddCommand(mcpStdioCmd)

	mcpSSECmd.Flags().StringVarP(&mcpSSEAddr, "addr", "a", ":8080", "Address to listen on")
	mcpCmd.AddCommand(mcpSSECmd)

	mcpHTTPCmd.Flags().StringVarP(&mcpHTTPAddr, "addr", "a", ":8081", "Address to listen on")
	mcpCmd.AddCommand(mcpHTTPCmd)

	mcpCmd.AddCommand(mcpInstallClaudeCmd)

	rootCmd.AddCommand(mcpCmd)
	rootCmd.AddCommand(genmanCmd)
	rootCmd.AddCommand(generatePluginCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
