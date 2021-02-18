// Copyright 2020 Platform9 Systems Inc.

package cmd

import (
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	serviceName string
	ingressSuffix string
)

func init() {
	addKeystoneCmd.PersistentFlags().StringVar(&serviceName, "service-name", "", "Name of the service (example qbert)")
	addKeystoneCmd.MarkPersistentFlagRequired("service-name")
	addKeystoneCmd.PersistentFlags().StringVar(&ingressSuffix, "ingress-suffix", "", "Ingress suffix  (example qbert/v2)")
	addKeystoneCmd.MarkPersistentFlagRequired("ingress-suffix")
	rootCmd.AddCommand(addKeystoneCmd)
}

var addKeystoneCmd = &cobra.Command{
	Use:   "add-keystone",
	Short: "Keystone related commands",
	Long: `Keystone related commands`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgMgr := GetCfg()
		zap.L().Debug("Adding keystone endpoint")
		err := cfgMgr.AddKeystoneEndpoint(serviceName, ingressSuffix)
		if err != nil {
			zap.L().Info("Error adding keystone endpoint")
			return err
		}
		err = cfgMgr.AddKeystoneUser(serviceName)
		if err != nil {
			zap.L().Info("Error adding  keystone user")
			return err
		}
		return nil
	},
}
