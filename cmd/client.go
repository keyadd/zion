/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"syscall"
	h2client "zion.com/zion/conn/h2"
	"zion.com/zion/conn/ws"
	"zion.com/zion/route"
)

//var Config *config.Client

var globalBool bool //全局代理
var chinaBool bool  //绕开中国

// zion vpn start client
var client = &cobra.Command{
	Use:   "client",
	Short: "zion vpn start client ",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {

		c := make(chan os.Signal)

		//监听指定信号 ctrl+c kill
		signal.Notify(c, os.Interrupt, os.Kill, syscall.SIGUSR1, syscall.SIGUSR2)

		//阻塞直到有信号传入
		fmt.Println("启动")
		go func() {
			for s := range c {
				switch s {
				case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
					ExitFunc()
				case syscall.SIGUSR1:
					fmt.Println("usr1", s)
				case syscall.SIGUSR2:
					fmt.Println("usr2", s)
				default:
					fmt.Println("other", s)
				}
			}
		}()

		if globalBool == false {
			if conf.Client.Type == "ws" {
				ws.StartClient(conf.Client, globalBool)
			} else if conf.Client.Type == "h2" {
				h2client.StartClient(conf.Client, globalBool)
			} else {
				fmt.Println("请输入正确的类型")
			}

		} else if globalBool == true {

			if conf.Client.Type == "ws" {
				ws.StartClient(conf.Client, globalBool)
			} else if conf.Client.Type == "h2" {
				h2client.StartClient(conf.Client, globalBool)
			} else {
				fmt.Println("请输入正确的类型")
			}
		} else {
			fmt.Println("输入参数错误")
		}

	},
}

func ExitFunc() {
	fmt.Println("开始退出...")
	if globalBool == true {
		//if retract == false {
		route.RetractRoute()
		//}

	}
	os.Exit(0)
}
func init() {

	client.PersistentFlags().BoolVarP(&globalBool, "global", "g", false, "开启全局网关代理 ")
	client.PersistentFlags().BoolVarP(&chinaBool, "china", "c", false, "开启绕过中国代理")
	rootCmd.AddCommand(client)

}
