// Copyright 2021 Platform9 Systems Inc.

package cmd

import (
    "fmt"
    "github.com/spf13/cobra"
)

var (
    userName string
)

func init() {
    createDBCmd.PersistentFlags().StringVar(&serviceName, "service-name", "", "Name of the service (example qbert)")
    createDBCmd.MarkPersistentFlagRequired("service-name")
    createDBCmd.PersistentFlags().StringVar(&userName, "db-user", "", "Optional parameter to set name of DB user. Defaults to serviceName.")
    rootCmd.AddCommand(createDBCmd)
}

var createDBCmd = &cobra.Command{
    Use:   "create-db",
    Short: "Create DB for service",
    Long: `Create DB for service`,
    RunE: func(cmd *cobra.Command, args []string) error {
        cfgMgr := GetCfg()
        if userName == "" {
            userName = serviceName
        }
        dbPassword := cfgMgr.GetRandomPassword()
        dbCreated, err := cfgMgr.CreateDB(serviceName, userName)
        if err != nil {
            return err
        }
        grantsCreated, err := cfgMgr.CreateGrants(serviceName, userName, dbPassword)
        if err != nil {
            return err
        }
        if dbCreated || grantsCreated {
            err := cfgMgr.UpdateConsul(serviceName, userName, dbPassword)
            return err
        }
        fmt.Println("db_name: ", serviceName)
        return nil
    },
}
