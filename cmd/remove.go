package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/spf13/cobra"
)

var (
	removeCmd *cobra.Command
)

func init() {
	removeCmd = &cobra.Command{
		Use:   "remove <config>",
		Short: "remove sebak containers",
		Args:  cobra.ExactArgs(1),
		Run: func(c *cobra.Command, args []string) {
			var err error
			if config, err = parseConfig(args[0]); err != nil {
				PrintFlagsError(runCmd, "<config>", err)
			}

			parseStopFlags()

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
					PrintError(removeCmd, fmt.Errorf("unknown host key found: %s", dhHost))
				}
				for _, c := range cls {
					go func(c types.Container) {
						defer wg.Done()
						err := dh.Client().ContainerRemove(context.Background(), c.ID, types.ContainerRemoveOptions{Force: true})
						if err != nil {
							log.Error("failed to stop", "error", err)
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

	rootCmd.AddCommand(removeCmd)
}
