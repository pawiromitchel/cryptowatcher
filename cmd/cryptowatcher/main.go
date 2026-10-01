// Command cryptowatcher is a terminal dashboard for crypto and stock prices.
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

// version is set at build time: -ldflags "-X main.version=v1.2.3".
var version = "dev"

func main() {
	os.Exit(run())
}

func run() int {
	useMock := flag.Bool("mock", false, "use synthetic offline data instead of live APIs")
	interval := flag.Int("interval", 0, "refresh interval in seconds (minimum 5; overrides the config file)")
	showConfig := flag.Bool("config-path", false, "print the configuration file path and exit")
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("cryptowatcher", version)
		return 0
	}

	if *showConfig {
		path, err := config.GetConfigPath()
		if err != nil {
			fmt.Fprintf(os.Stderr, "cryptowatcher: resolving config path: %v\n", err)
			return 1
		}
		fmt.Println(path)
		return 0
	}

	cfg, cfgErr := config.Load()
	if *interval > 0 {
		if *interval < 5 {
			fmt.Fprintln(os.Stderr, "cryptowatcher: -interval must be at least 5 seconds")
			return 2
		}
		cfg.RefreshInterval = *interval
	}

	var pf fetcher.PriceFetcher
	if *useMock {
		pf = fetcher.NewMockFetcher()
	} else {
		pf = fetcher.NewMultiFetcher(
			fetcher.NewCoinbaseFetcher(),
			fetcher.NewCoinGeckoFetcher(),
			fetcher.NewYahooFetcher(),
		)
	}

	m := ui.NewModel(cfg, pf)
	if cfgErr != nil {
		m = m.WithNotice("Config warning: "+cfgErr.Error(), true)
	}

	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "cryptowatcher: %v\n", err)
		return 1
	}
	return 0
}
