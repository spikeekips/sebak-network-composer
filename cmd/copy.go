package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	logging "github.com/inconshreveable/log15"
	isatty "github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

var (
	copyCmd *cobra.Command
)

func parseCopyFlags() {
	{
		var err error
		var logLevel logging.Lvl
		if logLevel, err = logging.LvlFromString(flagLogLevel); err != nil {
			fmt.Printf("invalid `log-level`: %v\n", err)
			os.Exit(1)
		}

		var formatter logging.Format
		if isatty.IsTerminal(os.Stdout.Fd()) {
			formatter = logging.TerminalFormat()
		} else {
			formatter = logging.JsonFormatEx(false, true)
		}
		logHandler := logging.StreamHandler(os.Stdout, formatter)

		log = logging.New("module", "main")
		log.SetHandler(logging.LvlFilterHandler(logLevel, logHandler))
	}

	{
		var err error
		if _, err = logging.LvlFromString(flagSebakLogLevel); err != nil {
			fmt.Printf("invalid `sebak-log-level`: %v\n", err)
			os.Exit(1)
		}
	}
}

func init() {
	copyCmd = &cobra.Command{
		Use:   "copy <config> <source> <output>",
		Short: "copy from sebak containers",
		Args:  cobra.ExactArgs(3),
		Run: func(c *cobra.Command, args []string) {
			var err error
			if config, err = parseConfig(args[0]); err != nil {
				PrintFlagsError(runCmd, "<config>", err)
			}

			if len(args[1]) < 1 {
				PrintFlagsError(copyCmd, "--source", fmt.Errorf("must be given"))
			}

			if _, err := os.Stat(args[2]); os.IsNotExist(err) {
				if err := os.Mkdir(args[2], 0755); err != nil {
					PrintFlagsError(copyCmd, "--output", err)
				}
			}

			flagSourceDirectory = args[1]
			flagOutputDirectory = args[2]

			parseCopyFlags()

			// get container info
			containers := map[string][]types.Container{}
			var numContainers int
			for _, dh := range config.DockerHosts {
				cl, err := findContainersByPrefix(dh.Client(), "scn.")
				if err != nil {
					log.Error("failed to get containers", "error", err)
					os.Exit(1)
				}
				if _, found := containers[dh.Host]; !found {
				}
				containers[dh.Host] = append(containers[dh.Host], cl...)
				numContainers++
			}

			var wg sync.WaitGroup

			var containerNames []string
			for _, cls := range containers {
				for _, c := range cls {
					containerNames = append(containerNames, GetContainerName(c.Names))
				}
			}

			fmt.Printf("containers: %s\n", strings.Join(containerNames, ", "))

			wg.Add(len(containerNames))
			for dhHost, cls := range containers {
				dh, found := config.GetDockerHost(dhHost)
				if !found {
					PrintError(copyCmd, fmt.Errorf("unknown host key found: %s", dhHost))
				}
				for _, c := range cls {
					go func(c types.Container) {
						defer wg.Done()
						err := copyFromContainer(
							dh.Client(),
							c.ID,
							flagSourceDirectory,
							filepath.Join(flagOutputDirectory, GetContainerName(c.Names)),
						)
						if err != nil {
							log.Error("failed to download source", "error", err)
						}
					}(c)
				}
			}

			ch := Ticker()
			wg.Wait()
			ch <- true
			log.Debug("done")
		},
	}

	var err error
	var currentDirectory string
	if currentDirectory, err = os.Getwd(); err != nil {
		PrintFlagsError(copyCmd, "--ouptut-directory", err)
	}
	if currentDirectory, err = filepath.Abs(currentDirectory); err != nil {
		PrintFlagsError(copyCmd, "--ouptut-directory", err)
	}

	copyCmd.Flags().StringVar(&flagLogLevel, "log-level", flagLogLevel, "log level, {crit, error, warn, info, debug}")
	copyCmd.Flags().StringVar(
		&flagSourceDirectory,
		"source",
		"",
		"source directory",
	)

	copyCmd.Flags().StringVar(
		&flagOutputDirectory,
		"output",
		fmt.Sprintf("%s/%s", currentDirectory, time.Now().Format("20060102T150405")),
		"output directory",
	)

	rootCmd.AddCommand(copyCmd)
}
