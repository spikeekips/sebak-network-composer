package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	logging "github.com/inconshreveable/log15"
	isatty "github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

var (
	buildCmd *cobra.Command
)

func parseBuildFlags() {
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

func buildImage(cli *client.Client, path string, fromSource bool) error {
	ctx, _ := archive.TarWithOptions(path, &archive.TarOptions{})

	dockerfile := "Dockerfile"
	if fromSource {
		dockerfile = "Dockerfile.from-source"
	}

	buildOptions := types.ImageBuildOptions{
		Tags:           []string{flagImageName},
		SuppressOutput: false,
		NoCache:        false,
		Remove:         true,
		Dockerfile:     dockerfile,
	}

	resp, err := cli.ImageBuild(context.Background(), ctx, buildOptions)
	if err != nil {
		return err
	}
	_, _ = ioutil.ReadAll(resp.Body)

	return nil
}

func init() {
	buildCmd = &cobra.Command{
		Use:   "build <config>",
		Short: "build sebak image",
		Args:  cobra.ExactArgs(1),
		Run: func(c *cobra.Command, args []string) {
			var err error
			if config, err = parseConfig(args[0]); err != nil {
				PrintFlagsError(runCmd, "<config>", err)
			}

			parseBuildFlags()

			logImage := log.New(logging.Ctx{"image": flagImageName})

			var wg sync.WaitGroup

			if flagForceClean {
				logImage.Debug("trying to remove image")

				wg.Add(len(config.DockerHosts))
				for _, dh := range config.DockerHosts {
					go func(d *DockerHost) {
						defer wg.Done()

						if err := removeImage(d.Client(), flagImageName); err != nil {
							logImage.Error("failed to remove image", "error", err)
							return
						}
					}(dh)
				}
				wg.Wait()
				logImage.Debug("successfully removed docker image")
			}

			logImage.Debug("trying to build image")
			wg.Add(len(config.DockerHosts))

			var foundErrors []error
			ch := Ticker()
			for _, dh := range config.DockerHosts {
				go func(d *DockerHost) {
					defer wg.Done()
					if err := buildImage(d.Client(), config.DockerPath, flagBuildFromSource); err != nil {
						foundErrors = append(foundErrors, err)
						logImage.Error("failed to build image", "error", err)
						return
					}
				}(dh)
			}

			wg.Wait()
			ch <- true
			if len(foundErrors) > 0 {
				logImage.Error("failed to create docker image")
			} else {
				logImage.Debug("successfully created docker image")
			}
		},
	}

	buildCmd.Flags().StringVar(&flagLogLevel, "log-level", flagLogLevel, "log level, {crit, error, warn, info, debug}")
	buildCmd.Flags().BoolVar(&flagBuildFromSource, "source", flagBuildFromSource, "build from source")
	buildCmd.Flags().StringVar(&flagImageName, "image", flagImageName, "docker image name for sebak")
	buildCmd.Flags().BoolVar(&flagForceClean, "force", flagForceClean, "remove the existing sebak containers")

	rootCmd.AddCommand(buildCmd)
}
