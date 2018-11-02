package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"sync"
	"time"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/node"
	"github.com/docker/docker/api/types"
	logging "github.com/inconshreveable/log15"
	isatty "github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"github.com/stellar/go/keypair"
)

var (
	runCmd *cobra.Command
)

func parseRunFlags() {
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

	log.Debug("Starting to compose sebak network")
	log.Debug(fmt.Sprintf(
		`
number of nodes: %d
      log level: %s
sebak log level: %s
		`,
		flagNumberOfNodes,
		flagLogLevel,
		flagSebakLogLevel,
	))
}

func composeNetwork() map[string]*node.LocalNode {
	log.Debug("trying to compose network", "number of nodes", flagNumberOfNodes)

	var t int
	nc := int(flagNumberOfNodes) / len(config.DockerHosts)
	t = nc
	if int(flagNumberOfNodes)%len(config.DockerHosts) != 0 {
		t = nc + 1
	}

	nodes := map[string]*node.LocalNode{}
	for i, dh := range config.DockerHosts {
		if len(nodes) >= int(flagNumberOfNodes) {
			break
		}

		var port int = baseContainerPort
		for j := 0; j < t; j++ {
			if len(nodes) >= int(flagNumberOfNodes) {
				break
			}

			kp, _ := keypair.Random()
			endpoint, err := common.NewEndpointFromString(fmt.Sprintf(
				"https://%s:%d",
				dh.IP,
				port,
			))
			if err != nil {
				PrintError(runCmd, err)
			}

			nd, err := node.NewLocalNode(kp, endpoint, fmt.Sprintf("n%d-%d", i, j))
			if err != nil {
				PrintError(runCmd, err)
			}
			dh.Nodes = append(dh.Nodes, nd)
			nodes[kp.Address()] = nd

			log.Debug(
				"generate node",
				"address", node.MakeAlias(kp.Address()),
				"secret-seed", kp.Seed(),
				"endpoint", endpoint,
			)

			port += 1
		}
	}

	log.Debug("generate nodes", "nodes", len(nodes))

	for a0, n0 := range nodes {
		for a1, n1 := range nodes {
			if a0 == a1 {
				continue
			}
			n0.AddValidators(n1.ConvertToValidator())
		}
	}

	return nodes
}

func init() {
	runCmd = &cobra.Command{
		Use:   "run <config>",
		Short: "sebak composing network",
		Args:  cobra.ExactArgs(1),
		Run: func(c *cobra.Command, args []string) {
			var err error
			if config, err = parseConfig(args[0]); err != nil {
				PrintFlagsError(runCmd, "<config>", err)
			}

			parseRunFlags()

			if flagForceClean {
				for _, dh := range config.DockerHosts {
					if err = cleanDocker(dh.Client()); err != nil {
						PrintError(runCmd, err)
					}
				}
			}

			var wg sync.WaitGroup
			{ // check internal ip
				log.Debug("trying to get internal IP")
				wg.Add(len(config.DockerHosts))
				for _, dh := range config.DockerHosts {
					go func(d *DockerHost) {
						defer wg.Done()

						var ip string
						if ip, err = runContainerGettingIP(d.Client()); err != nil {
							PrintError(runCmd, fmt.Errorf("failed to get internal IP: %v", err))
						}
						d.IP = ip
					}(dh)
				}

				wg.Wait()
			}

			// compose network
			nodes := composeNetwork()

			commonKeypair, _ := keypair.Random()

			for _, dh := range config.DockerHosts {
				for _, nd := range dh.Nodes {
					_, err := runSEBAK(dh, nd, config.Genesis, commonKeypair)
					if err != nil {
						log.Error("failed to run container", "error", err)
						os.Exit(1)
					}
				}
			}

			// check status
			infos := map[string]types.Container{}
			var stop bool

			go func() {
				for {
					if stop == true {
						break
					}
					for _, dh := range config.DockerHosts {
						for _, nd := range dh.Nodes {
							name := makeContainerName(nd)
							if info, ok := infos[name]; ok && info.State == "exited" {
								continue
							}

							info, err := findContainer(dh.Client(), name)
							if err != nil {
								log.Error("something wrong")
								os.Exit(1)
							}
							fmt.Println("<", info.Names[0][1:], info.ID[:4], info.State, info.Status)
							infos[name] = info
						}
					}

					time.Sleep(time.Second)
				}
			}()

			select {
			case <-time.After(time.Second * 5):
				stop = true
			}

			for _, dh := range config.DockerHosts {
				for _, nd := range dh.Nodes {
					name := makeContainerName(nd)
					if info, ok := infos[name]; ok && info.State != "exited" {
						continue
					}

					var info types.Container
					info = infos[name]

					resp, err := dh.Client().ContainerLogs(
						context.Background(),
						info.ID,
						types.ContainerLogsOptions{
							ShowStdout: true,
							ShowStderr: true,
						},
					)
					if err != nil {
						log.Error("failed to get container log", "name", name, "error", err)
						continue
					}
					if b, err := ioutil.ReadAll(resp); err != nil {
						log.Error("failed to get container log", "name", name, "error", err)
						continue
					} else {
						fmt.Printf("= %s =========================================================\n", name)

						re := regexp.MustCompile("\x1B\\[([0-9]{1,3}((;[0-9]{1,3})*)?)?[m|K]")
						nb := string(re.ReplaceAll(b, []byte("")))

						limit := len(nb) - 1000
						if limit < 0 {
							limit = 0
						}

						fmt.Println("...\n" + nb[limit:])
					}
				}
			}

			for _, nd := range nodes {
				log.Debug("launch node", "alias", nd.Alias(), "endpoint", nd.Endpoint())
			}
		},
	}

	runCmd.Flags().UintVar(&flagNumberOfNodes, "n", 3, "number of node")
	runCmd.Flags().StringVar(&flagImageName, "image", flagImageName, "docker image name for sebak")
	runCmd.Flags().BoolVar(
		&flagForceClean,
		"force",
		flagForceClean,
		"remove the existing sebak containers",
	)
	runCmd.Flags().StringVar(&flagLogLevel, "log-level", flagLogLevel, "log level, {crit, error, warn, info, debug}")
	runCmd.Flags().StringVar(
		&flagSebakLogLevel,
		"sebak-log-level",
		flagSebakLogLevel,
		"sebak log level, {crit, error, warn, info, debug}",
	)

	rootCmd.AddCommand(runCmd)
}
