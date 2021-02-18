// Copyright 2020 Platform9 Systems Inc.

package cmd

import (
	"github.com/platform9/go-firkinize/pkg/cfg"
	"os"
	"go.uber.org/zap"
	"github.com/spf13/cobra"
)

var (
	// Consul server address
	consulHostPort string

	// Consul server security token
	consulToken string

	// Consul scheme
	consulScheme string

	// Customer ID
	customerID string

	// regionID (if the service is region specific)
	regionID string

	// debug log level
	debugLog bool

	cfgMgr *cfg.CfgMgr

	rootCmd = &cobra.Command{
		Use:   "firkinize",
		Short: "A utility to get/set various configuration to Platform9 config store",
		Long: `A simple utility that hides the complexity associated with Platform9
		config store i.e. consul/vault as of today.`,
		PersistentPreRunE: func (cmd *cobra.Command, args []string) error {
			var err error
			setupLogs()
			cfgMgr, err = cfg.Setup(consulHostPort,consulScheme, consulToken,customerID,regionID)
			if err != nil {
				er("can't setup consul")
				return err
			}
			return nil
		},
	}
)


func GetCfg() *cfg.CfgMgr {
	return cfgMgr
}

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func setupLogs() {
	var config zap.Config
	if debugLog {
		config = zap.NewDevelopmentConfig()
	} else {
		config = zap.NewProductionConfig()
	}
	config.OutputPaths = []string{"stderr"}
	logger, _ := config.Build()
	zap.ReplaceGlobals(logger)
	zap.L().Debug("Debug log enabled")
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&debugLog, "debug", false,"Enable debug logging")
	rootCmd.PersistentFlags().StringVar(&consulHostPort, "consul-host-port", os.Getenv("CONFIG_HOST_AND_PORT"), "Where to connect to consul server")
	rootCmd.MarkPersistentFlagRequired("consul-host-port")
	rootCmd.PersistentFlags().StringVar(&consulToken, "consul-token", os.Getenv("CONSUL_HTTP_TOKEN"), "Security token to talk to consul server")
	rootCmd.MarkPersistentFlagRequired("consul-token")
	rootCmd.PersistentFlags().StringVar(&consulScheme, "consul-scheme", os.Getenv("CONFIG_SCHEME"), "Consul API scheme can be http/https/jrpc")
	rootCmd.MarkPersistentFlagRequired("consul-scheme")
	rootCmd.PersistentFlags().StringVar(&customerID, "customer-id", os.Getenv("CUSTOMER_ID"), "ID of the customer under which it is operating")
	rootCmd.MarkPersistentFlagRequired("customer-id")
	rootCmd.PersistentFlags().StringVar(&regionID, "region-id", os.Getenv("REGION_ID"), "ID of the region under which it is operating")
}

func er(msg interface{}) {
	os.Exit(1)
}

