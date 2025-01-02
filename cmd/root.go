package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "serial-loopback",
	Short: "A serial port loopback testing tool",
	Long:  `A TUI-based tool for testing serial port communication using loopback mode.`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		panic(err)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
}

func initConfig() {
	viper.AutomaticEnv()
}
