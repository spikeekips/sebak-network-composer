package cmd

import (
	"os"

	logging "github.com/inconshreveable/log15"
	"github.com/spf13/cobra"
)

const (
	basePort                  int    = 12345
	baseContainerPort         int    = 12000
	networkID                 string = "test sebak-network"
	dockerContainerNamePrefix string = "scn."
)

const (
	defaultLogLevel        logging.Lvl = logging.LvlInfo
	defaultSebakLogLevel   logging.Lvl = logging.LvlDebug
	defaultDockerImageName string      = "boscoin/sebak-network-composer:latest"
)

var (
	log            logging.Logger
	config         *Config
	maxLogsVerbose int64 = 10000

	flagLogLevel        string = defaultLogLevel.String()
	flagSebakLogLevel   string = defaultSebakLogLevel.String()
	flagImageName       string = defaultDockerImageName
	flagForceClean      bool   = false
	flagBuildFromSource bool
	flagVerbose         bool
	flagSourceDirectory string
	flagOutputDirectory string
	flagLogsSince       string
	flagLogsTail        string
	flagLogsHead        string
)

var rootCmd = &cobra.Command{
	Use:   os.Args[0],
	Short: "sebak-network-composer",
	Run: func(c *cobra.Command, args []string) {
		if len(args) < 1 {
			c.Usage()
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		PrintFlagsError(rootCmd, "", err)
	}
}

func SetArgs(s []string) {
	rootCmd.SetArgs(s)
}
