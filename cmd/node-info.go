package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"

	"github.com/docker/docker/api/types"
	logging "github.com/inconshreveable/log15"
	isatty "github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

var (
	nodeInfoCmd *cobra.Command
)

func parseNodeInoFlags() {
	if len(flagImageName) < 1 {
		PrintFlagsError(buildCmd, "--image", fmt.Errorf("empty image name"))
	}

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
	nodeInfoCmd = &cobra.Command{
		Use:   "node-info <config>",
		Short: "node-info running sebak nodes",
		Args:  cobra.ExactArgs(1),
		Run: func(c *cobra.Command, args []string) {
			var err error
			if config, err = parseConfig(args[0]); err != nil {
				PrintFlagsError(runCmd, "<config>", err)
			}

			parseNodeInoFlags()

			// get container info
			var endpoints []string
			var containers []types.Container
			for _, dh := range config.DockerHosts {
				cl, err := findContainersByPrefix(dh.Client(), "scn.")
				if err != nil {
					log.Error("failed to get containers", "error", err)
					os.Exit(1)
				}
				containers = append(containers, cl...)

				for _, c := range cl {
					if c.State == "exited" {
						continue
					}

					j, err := dh.Client().ContainerInspect(context.Background(), c.ID)
					if err != nil {
						log.Error("failed to inspect containers", "error", err)
						os.Exit(1)
					}

					for _, e := range j.Config.Env {
						if !strings.HasPrefix(e, "SEBAK_PUBLISH=") {
							continue
						}

						du, _ := url.Parse(dh.Host)
						dhost, _, _ := net.SplitHostPort(du.Host)
						u, _ := url.Parse(strings.Replace(e, "SEBAK_PUBLISH=", "", 1))
						_, port, _ := net.SplitHostPort(u.Host)

						u.Host = fmt.Sprintf("%s:%s", dhost, port)

						endpoints = append(endpoints, u.String())
					}
				}
			}

			if len(endpoints) < 1 {
				PrintError(nodeInfoCmd, fmt.Errorf("containers not found"))
			}

			// get publish endpoint
			for _, endpoint := range endpoints {
				b, err := HTTPGet(endpoint)
				if err != nil {
					log.Error("failed to get response", "endpoint", endpoint)
				}
				if flagVerbose {
					var m map[string]interface{}
					json.Unmarshal(b, &m)
					b, _ = json.Marshal(m)
					fmt.Printf("%s == %s\n", endpoint, string(b))
				}
			}
		},
	}

	nodeInfoCmd.Flags().StringVar(&flagLogLevel, "log-level", flagLogLevel, "log level, {crit, error, warn, info, debug}")
	nodeInfoCmd.Flags().StringVar(&flagImageName, "image", flagImageName, "docker image name for sebak")
	nodeInfoCmd.Flags().BoolVar(&flagVerbose, "verbose", flagVerbose, "verbose")

	rootCmd.AddCommand(nodeInfoCmd)
}
