/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log"
	"os"
	"zion.com/zion/config"
)

var conf *config.Config

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "zion",
	Short: "zion vpn network tunnel",
	//Long:  ``,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//Run: func(cmd *cobra.Command, args []string) {},

}

func Execute() {

	err := rootCmd.Execute()
	if err != nil {
		fmt.Println("exit")
		os.Exit(1)
	}
}

func init() {

	v := viper.New()
	v.SetConfigFile("./config.yaml")

	if err := v.ReadInConfig(); err != nil {
		log.Println("v.ReadInConfig()", err)
		return
	}
	err := v.Unmarshal(&conf)
	if err != nil {
		log.Println("v.Unmarshal(&conf)", err)
		return
	}
	log.Println(conf.Server)

	log.Println()
	log.Println(conf.Client)
	rootCmd.CompletionOptions.DisableDescriptions = true
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.SetHelpCommand(&cobra.Command{
		Use:    "no-help",
		Hidden: true,
	})
}
