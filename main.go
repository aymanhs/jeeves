package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/aymanhs/sys-tui/systemd"
	"github.com/aymanhs/sys-tui/tui"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Flags
	userFlag := flag.Bool("user", false, "Connect to the user systemd manager")
	systemFlag := flag.Bool("system", false, "Connect to the system systemd manager (default)")
	helpFlag := flag.Bool("help", false, "Show help information")
	flag.BoolVar(helpFlag, "h", false, "Show help information")

	flag.Parse()

	if *helpFlag {
		fmt.Println("sys-tui: A TUI for managing systemd services using D-Bus")
		fmt.Println("\nUsage:")
		fmt.Println("  sys-tui [flags]")
		fmt.Println("\nFlags:")
		fmt.Println("  --user      Connect to the user systemd manager (no root required)")
		fmt.Println("  --system    Force connect to the system systemd manager (requires root/sudo)")
		fmt.Println("  -h, --help  Show this help information")
		os.Exit(0)
	}

	var mode *systemd.Mode
	if *userFlag {
		m := systemd.UserMode
		mode = &m
	} else if *systemFlag {
		m := systemd.SystemMode
		mode = &m
	}

	// Connect to systemd D-Bus
	client, err := systemd.NewClient(mode)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to systemd: %v\n", err)
		if mode == nil || *mode == systemd.SystemMode {
			fmt.Fprintln(os.Stderr, "\nTip: Managing system units usually requires root. Try running: sudo ./sys-tui")
			fmt.Fprintln(os.Stderr, "Alternatively, run with --user to manage user-level services: ./sys-tui --user")
		}
		os.Exit(1)
	}
	defer client.Close()

	// Initialize Bubble Tea program
	p := tea.NewProgram(tui.NewModel(client), tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		log.Fatalf("Error running program: %v", err)
	}
}
