package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	logging "github.com/inconshreveable/log15"
	isatty "github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

var (
	logsCmd *cobra.Command
)

func parseLogsFlags() {
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

	if _, err := os.Stat(flagOutputDirectory); os.IsNotExist(err) {
		if err := os.Mkdir(flagOutputDirectory, 0755); err != nil {
			PrintFlagsError(logsCmd, "--output-directory", err)
		}
	}
}

func getContainerLogs(cli *client.Client, id, path string) error {
	reader, err := cli.ContainerLogs(context.Background(), id, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Since:      flagLogsSince,
		Tail:       flagLogsTail,
	})
	if err != nil {
		return err
	}

	output, err := os.Create(path)
	if err != nil {
		return err
	}

	_, err = io.Copy(output, reader)
	if err != nil && err != io.EOF {
		return err
	}

	return nil
}

func init() {
	logsCmd = &cobra.Command{
		Use:   "logs <config>",
		Short: "logs sebak containers",
		Args:  cobra.ExactArgs(1),
		Run: func(c *cobra.Command, args []string) {
			var err error
			if config, err = parseConfig(args[0]); err != nil {
				PrintFlagsError(runCmd, "<config>", err)
			}

			parseLogsFlags()

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
					PrintError(logsCmd, fmt.Errorf("unknown host key found: %s", dhHost))
				}
				for _, c := range cls {
					go func(c types.Container) {
						defer wg.Done()
						err := getContainerLogs(
							dh.Client(),
							c.ID,
							filepath.Join(flagOutputDirectory, GetContainerName(c.Names)+".log"),
						)
						if err != nil {
							log.Error("failed to download logs", "error", err)
						}
					}(c)
				}
			}

			ch := Ticker()
			wg.Wait()
			ch <- true
			log.Debug("done")

			if flagVerbose {
				for _, c := range containerNames {
					fname := filepath.Join(flagOutputDirectory, c+".log")
					fmt.Printf("= %s ================\n", fname)

					f, _ := os.Open(fname)
					fi, _ := f.Stat()

					n := fi.Size() - maxLogsVerbose
					if n > 0 {
						f.Seek(n, 0)
					}

					buf := make([]byte, maxLogsVerbose) // define your buffer size here.
					for {
						n, err := f.Read(buf)
						if err == io.EOF {
							break
						}
						if err != nil {
							fmt.Printf("read %d bytes: %v\n", n, err)
							break
						}
					}

					fmt.Fprintf(os.Stdout, "%s", string(buf))
				}
			}
		},
	}

	var err error
	var currentDirectory string
	if currentDirectory, err = os.Getwd(); err != nil {
		PrintFlagsError(logsCmd, "--ouptut-directory", err)
	}
	if currentDirectory, err = filepath.Abs(currentDirectory); err != nil {
		PrintFlagsError(logsCmd, "--ouptut-directory", err)
	}

	logsCmd.Flags().StringVar(&flagLogLevel, "log-level", flagLogLevel, "log level, {crit, error, warn, info, debug}")
	logsCmd.Flags().StringVar(
		&flagOutputDirectory,
		"ouput-directory",
		fmt.Sprintf("%s/%s", currentDirectory, time.Now().Format("20060102T150405")),
		"output directory",
	)
	logsCmd.Flags().StringVar(&flagLogsSince, "since", flagLogsSince, "since")
	logsCmd.Flags().BoolVar(&flagVerbose, "verbose", flagVerbose, "verbose")
	logsCmd.Flags().StringVar(&flagLogsTail, "tail", flagLogsTail, "tail")
	logsCmd.Flags().StringVar(&flagLogsHead, "head", flagLogsHead, "head")

	rootCmd.AddCommand(logsCmd)
}
