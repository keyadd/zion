/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"zion.com/zion/conn/ws"

	"github.com/spf13/cobra"
)

// zion vpn start server
var server = &cobra.Command{
	Use:   "server",
	Short: "zion vpn start server ",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("server called")
		ws.StartServer(conf.Server)
	},
}

func init() {
	rootCmd.AddCommand(server)
}
