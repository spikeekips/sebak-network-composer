package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/spf13/cobra"
)

var (
	listCmd *cobra.Command
)

func init() {
	listCmd = &cobra.Command{
		Use:   "list <config>",
		Short: "list sebak containers",
		Args:  cobra.ExactArgs(1),
		Run: func(c *cobra.Command, args []string) {
			var err error
			if config, err = parseConfig(args[0]); err != nil {
				PrintFlagsError(runCmd, "<config>", err)
			}

			parseStartFlags()

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

			var containerNames []string
			for _, cls := range containers {
				for _, c := range cls {
					containerNames = append(containerNames, GetContainerName(c.Names))
				}
			}

			sort.Strings(containerNames)
			fmt.Printf("containers: %s\n", strings.Join(containerNames, ", "))

			for dhHost, cls := range containers {
				dh, found := config.GetDockerHost(dhHost)
				if !found {
					PrintError(stopCmd, fmt.Errorf("unknown host key found: %s", dhHost))
				}
				for _, c := range cls {
					b, _ := json.MarshalIndent(c, "", "  ")
					fmt.Println(strings.Repeat("=", 80))
					fmt.Println(dh.Host)
					fmt.Println(string(b))
				}
			}

			log.Debug("done")
		},
	}

	rootCmd.AddCommand(listCmd)
}
