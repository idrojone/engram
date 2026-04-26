package main

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/Gentleman-Programming/engram/internal/cloud"
	"github.com/Gentleman-Programming/engram/internal/cloud/auth"
	"github.com/Gentleman-Programming/engram/internal/cloud/cloudserver"
	"github.com/Gentleman-Programming/engram/internal/cloud/cloudstore"
	"github.com/Gentleman-Programming/engram/internal/mcp"
	"github.com/Gentleman-Programming/engram/internal/server"
	"github.com/Gentleman-Programming/engram/internal/setup"
	"github.com/Gentleman-Programming/engram/internal/store"
	"github.com/Gentleman-Programming/engram/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
	mcp_server "github.com/mark3labs/mcp-go/server"
)

const Version = "0.1.0-dev"

// Dependency injection for tests
var (
	exitFunc            = os.Exit
	runTeaProgram       = func(p *tea.Program) (tea.Model, error) { return p.Run() }
	newTeaProgram       = func(m tea.Model) *tea.Program { return tea.NewProgram(m) }
	newTUIModel         = func(s *store.Store) tui.Model { return tui.New(s, Version) }
	storeNew            = func(cfg store.Config) (*store.Store, error) { return store.New(cfg) }
	newMCPServerWithConfig = func(s *store.Store, cfg mcp.MCPConfig, tools map[string]bool) *mcp_server.MCPServer {
		return mcp.NewServerWithConfig(s, cfg, tools)
	}
	serveMCP = func(s *mcp_server.MCPServer) error {
		return mcp_server.ServeStdio(s)
	}
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		exitFunc(1)
	}

	cfg, _ := store.DefaultConfig()

	cmd := os.Args[1]
	switch cmd {
	case "serve":
		cmdServe(cfg)
	case "mcp":
		cmdMCP(cfg)
	case "tui":
		cmdTUI(cfg)
	case "search":
		cmdSearch(cfg)
	case "save":
		cmdSave(cfg)
	case "timeline":
		cmdTimeline(cfg)
	case "context":
		cmdContext(cfg)
	case "stats":
		cmdStats(cfg)
	case "setup":
		cmdSetup(cfg)
	case "cloud":
		cmdCloud(cfg)
	case "version":
		fmt.Printf("engram %s (%s/%s)\n", Version, runtime.GOOS, runtime.GOARCH)
	case "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "engram: unknown command %q\n", cmd)
		printUsage()
		exitFunc(1)
	}
}

func cmdServe(cfg store.Config) {
	port := 7437
	if v := os.Getenv("ENGRAM_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			port = p
		}
	}
	if len(os.Args) > 2 {
		if p, err := strconv.Atoi(os.Args[2]); err == nil {
			port = p
		}
	}

	// In the new cloud-first architecture, 'serve' can run either as a local proxy
	// or as the central cloud server. We detect based on environment.
	if os.Getenv("ENGRAM_DATABASE_URL") != "" {
		// Run as central cloud server
		fmt.Printf("Starting Engram Cloud server on port %d...\n", port)
		
		cloudCfg := cloud.ConfigFromEnv()
		cStore, err := cloudstore.New(cloudCfg)
		if err != nil {
			fatal(err)
		}

		authSvc, err := auth.NewService(cStore, cloudCfg.JWTSecret)
		if err != nil {
			fatal(err)
		}

		srv := cloudserver.New(cStore, authSvc, port)
		if err := srv.Start(); err != nil {
			fatal(err)
		}
		return
	}

	// Run as local proxy server (legacy local mode)
	s, err := storeNew(cfg)
	if err != nil {
		fatal(err)
	}
	defer s.Close()

	fmt.Printf("Starting Engram API server on http://localhost:%d...\n", port)
	srv := server.New(s, port)

	if err := srv.Start(); err != nil {
		fatal(err)
	}
}

func cmdMCP(cfg store.Config) {
	toolsFilter := ""
	for i := 2; i < len(os.Args); i++ {
		if strings.HasPrefix(os.Args[i], "--tools=") {
			toolsFilter = strings.TrimPrefix(os.Args[i], "--tools=")
		} else if os.Args[i] == "--tools" && i+1 < len(os.Args) {
			toolsFilter = os.Args[i+1]
			i++
		}
	}

	s, err := storeNew(cfg)
	if err != nil {
		fatal(err)
	}
	defer s.Close()

	mcpCfg := mcp.MCPConfig{}
	allowlist := resolveMCPTools(toolsFilter)
	mcpSrv := newMCPServerWithConfig(s, mcpCfg, allowlist)

	if err := serveMCP(mcpSrv); err != nil {
		fatal(err)
	}
}

func resolveMCPTools(filter string) map[string]bool {
	if filter == "" || filter == "all" {
		return nil // all tools
	}
	result := make(map[string]bool)
	parts := strings.Split(filter, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		switch p {
		case "agent":
			for k := range mcp.ProfileAgent {
				result[k] = true
			}
		case "admin":
			for k := range mcp.ProfileAdmin {
				result[k] = true
			}
		default:
			result[p] = true
		}
	}
	return result
}

func cmdTUI(cfg store.Config) {
	s, err := storeNew(cfg)
	if err != nil {
		fatal(err)
	}
	defer s.Close()

	model := newTUIModel(s)
	p := newTeaProgram(model)
	if _, err := runTeaProgram(p); err != nil {
		fatal(err)
	}
}

func cmdSearch(cfg store.Config) {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: engram search <query> [--limit N]")
		exitFunc(1)
	}

	query := os.Args[2]
	limit := 10
	for i := 3; i < len(os.Args); i++ {
		if os.Args[i] == "--limit" && i+1 < len(os.Args) {
			limit, _ = strconv.Atoi(os.Args[i+1])
			i++
		}
	}

	s, err := storeNew(cfg)
	if err != nil {
		fatal(err)
	}
	defer s.Close()

	results, err := s.Search(query, store.SearchOptions{Limit: limit})
	if err != nil {
		fatal(err)
	}

	for _, r := range results {
		fmt.Printf("[%d] %s (%s)\n", r.ID, r.Title, r.Type)
		fmt.Printf("    %s\n\n", truncate(r.Content, 200))
	}
}

func cmdSave(cfg store.Config) {
	if len(os.Args) < 4 {
		fmt.Fprintln(os.Stderr, "usage: engram save <title> <content> [--type TYPE]")
		exitFunc(1)
	}

	title := os.Args[2]
	content := os.Args[3]
	obsType := "manual"
	for i := 4; i < len(os.Args); i++ {
		if os.Args[i] == "--type" && i+1 < len(os.Args) {
			obsType = os.Args[i+1]
			i++
		}
	}

	s, err := storeNew(cfg)
	if err != nil {
		fatal(err)
	}
	defer s.Close()

	id, err := s.AddObservation(store.AddObservationParams{
		Title:   title,
		Content: content,
		Type:    obsType,
	})
	if err != nil {
		fatal(err)
	}

	fmt.Printf("Saved observation %d\n", id)
}

func cmdTimeline(cfg store.Config) {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: engram timeline <observation_id>")
		exitFunc(1)
	}

	id, err := strconv.ParseInt(os.Args[2], 10, 64)
	if err != nil {
		fatal(err)
	}

	s, err := storeNew(cfg)
	if err != nil {
		fatal(err)
	}
	defer s.Close()

	tl, err := s.Timeline(id, 5, 5)
	if err != nil {
		fatal(err)
	}

	for _, entry := range tl.Before {
		fmt.Printf("[%s] %s\n", entry.ID, entry.Title)
	}
	fmt.Printf(">>> [%d] %s <<<\n", tl.Focus.ID, tl.Focus.Title)
	for _, entry := range tl.After {
		fmt.Printf("[%s] %s\n", entry.ID, entry.Title)
	}
}

func cmdContext(cfg store.Config) {
	s, err := storeNew(cfg)
	if err != nil {
		fatal(err)
	}
	defer s.Close()

	// Context usually means recent observations
	obs, err := s.AllObservations("", "", 10)
	if err != nil {
		fatal(err)
	}

	for _, o := range obs {
		fmt.Printf("[%d] %s\n", o.ID, o.Title)
	}
}

func cmdStats(cfg store.Config) {
	s, err := storeNew(cfg)
	if err != nil {
		fatal(err)
	}
	defer s.Close()

	stats, err := s.Stats()
	if err != nil {
		fatal(err)
	}

	fmt.Printf("Observations: %d\n", stats.TotalObservations)
	fmt.Printf("Sessions:     %d\n", stats.TotalSessions)
	fmt.Printf("Prompts:      %d\n", stats.TotalPrompts)
}

func cmdSetup(cfg store.Config) {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: engram setup <agent_name>")
		exitFunc(1)
	}

	agent := os.Args[2]
	_, err := setup.Install(agent)
	if err != nil {
		fatal(err)
	}

	fmt.Printf("Setup complete for %s\n", agent)
}

func cmdCloud(cfg store.Config) {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: engram cloud <subcommand>")
		fmt.Fprintln(os.Stderr, "subcommands: status, config")
		exitFunc(1)
	}

	sub := os.Args[2]
	switch sub {
	case "status":
		fmt.Printf("Cloud URL:   %s\n", cfg.BaseURL)
		fmt.Printf("Cloud Token: %s\n", mask(cfg.APIKey))
	case "config":
		fmt.Println("To configure cloud, set ENGRAM_API_URL and ENGRAM_API_KEY environment variables.")
	default:
		fmt.Fprintf(os.Stderr, "unknown cloud subcommand %q\n", sub)
	}
}

func mask(s string) string {
	if len(s) <= 8 {
		return "********"
	}
	return s[:4] + "...." + s[len(s)-4:]
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "engram: %v\n", err)
	exitFunc(1)
}

func printUsage() {
	fmt.Printf("engram %s - agentic coding memory\n\n", Version)
	fmt.Println("Usage: engram <command> [args]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  serve [port]   Start HTTP API server")
	fmt.Println("  mcp            Start MCP server (stdio)")
	fmt.Println("  tui            Launch terminal UI")
	fmt.Println("  search <q>     Search memories")
	fmt.Println("  save <t> <c>   Save a memory")
	fmt.Println("  timeline <id>  Show context around a memory")
	fmt.Println("  context        Show recent context")
	fmt.Println("  stats          Show system stats")
	fmt.Println("  setup <agent>  Setup agent integration")
	fmt.Println("  cloud <subcp>  Cloud commands")
	fmt.Println("  version        Print version")
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "..."
}
