/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"zion.com/zion/conn/websocket"

	"github.com/spf13/cobra"
)

// zion vpn start server
var server = &cobra.Command{
	Use:   "server",
	Short: "zion vpn start server ",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("server called")
		websocket.StartServer(conf.Server)
	},
}

func init() {
	rootCmd.AddCommand(server)
}
