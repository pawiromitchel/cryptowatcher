package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"cryptowatcher/internal/config"
	"cryptowatcher/internal/fetcher"
	"cryptowatcher/internal/ui"
)

func main() {
	useMock := flag.Bool("mock", false, "Use mock data fetcher for testing")
	intervalFlag := flag.Int("interval", 0, "Override price refresh interval in seconds")
	showConfig := flag.Bool("config-path", false, "Print path to configuration file and exit")
	flag.Parse()

	if *showConfig {
		path, err := config.GetConfigPath()
		if err != nil {
			fmt.Printf("Error resolving config path: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(path)
		return
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load configuration: %v\n", err)
	}

	if *intervalFlag > 0 {
		cfg.RefreshInterval = *intervalFlag
	}

	var priceFetcher fetcher.PriceFetcher
	if *useMock {
		priceFetcher = fetcher.NewMockFetcher()
	} else {
		priceFetcher = fetcher.NewMultiFetcher(
			fetcher.NewCoinbaseFetcher(),
			fetcher.NewPythFetcher(),
		)
	}

	p := tea.NewProgram(
		ui.NewModel(cfg, priceFetcher),
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Application error: %v\n", err)
		os.Exit(1)
	}
}
