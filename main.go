package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	_ "github.com/anti-raid/evil-befall/pkg/api_all"
	"github.com/anti-raid/evil-befall/pkg/router"
	_ "github.com/anti-raid/evil-befall/pkg/routes"
	statelib "github.com/anti-raid/evil-befall/pkg/state"
	"github.com/anti-raid/shellcli/shell"
)

type cliData struct {
	State *statelib.State
}

func envOrBool(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	return fallback
}

func envOrString(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	return fallback
}

func main() {
	// Create a new state
	var mouseEnabled = envOrBool("MOUSE_ENABLED", "false") == "true"
	var pasteEnabled = envOrBool("PASTE_ENABLED", "true") == "true"
	var fullscreen = envOrBool("FULLSCREEN", "true") == "true"
	var persist = envOrString("PERSIST", "evil-befall-cfg.json")

	// Set state.Prefs
	state, err := statelib.NewState(statelib.UserPref{
		MouseEnabledInTView:      mouseEnabled,
		PasteEnabledInTView:      pasteEnabled,
		FullscreenEnabledInTView: fullscreen,
		Persist: func() *string {
			if persist == "" || persist == "false" {
				return nil
			}

			return &persist
		}(),
	})

	if err != nil {
		slog.Error("Failed to create state:", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Create command list
	var commands = make(map[string]*shell.Command[cliData])

	for _, route := range router.Routes() {
		cmd := &shell.Command[cliData]{
			Name:        route.Command(),
			Description: route.Description(),
			Args:        route.Arguments(),
			Run: func(cli *shell.ShellCli[cliData], args map[string]string) error {
				return router.Goto(route.Command(), cli.Data.State, args)
			},
		}

		// Default completion
		var completion = func(a *shell.ShellCli[cliData], line string, args map[string]string) ([]string, error) {
			return shell.ArgBasedCompletionHandler(a, cmd, line, args)
		}

		completer, ok := route.(router.CompletableRoute)

		if ok {
			completion = func(a *shell.ShellCli[cliData], line string, args map[string]string) ([]string, error) {
				return completer.Completion(a.Data.State, line, args)
			}
		}

		cmd.Completer = completion

		commands[route.Command()] = cmd
	}

	root := &shell.ShellCli[cliData]{
		Data: &cliData{
			State: state,
		},
		Prompter: func(r *shell.ShellCli[cliData]) string {
			return "evil-befall> "
		},
		Commands:         commands,
		HistoryPath:      "evil-befall-history.txt",
		DebugCompletions: envOrBool("DEBUG_COMPLETIONS", "false") == "true",
	}

	root.AddCommand("help", root.Help())
	root.AddCommand("getcompletion", root.GetCompletion())

	// Handle --command args
	command := flag.String("command", "", "Command to run. If unset, will run as shell")
	flag.Parse()

	if command != nil && *command != "" {
		err := root.Init()

		if err != nil {
			fmt.Println("Error initializing cli: ", err)
			os.Exit(1)
		}

		cancel, err := root.ExecuteCommands(*command)

		if err != nil {
			fmt.Println("Error:", err)
		}

		if cancel {
			fmt.Println("Exiting...")
		}

		return
	}

	root.Run()
}
