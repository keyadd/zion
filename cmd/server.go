/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"zion.com/zion/conn/h2"
	"zion.com/zion/conn/ws"

	"github.com/spf13/cobra"
)

// zion vpn start server
var server = &cobra.Command{
	Use:   "server",
	Short: "zion vpn start server ",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("server start")
		if conf.Server.Type == "ws" {
			ws.StartServer(conf.Server)
		} else if conf.Server.Type == "h2" {
			h2.StartServer(conf.Server)
		} else {
			fmt.Println("请输入正确的类型")
		}
	},
}

func init() {
	rootCmd.AddCommand(server)
}
