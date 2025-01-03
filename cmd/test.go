package cmd

import (
	"fmt"
	"tui/tui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	port     string
	baudRate int
	count    int
)

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Run serial port loopback test",
	Run: func(cmd *cobra.Command, args []string) {
		model, err := tui.NewSerialModel(port, baudRate, count)
		if err != nil {
			fmt.Printf("Error initializing serial port: %v\n", err)
			return
		}

		p := tea.NewProgram(model)
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error running TUI: %v\n", err)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(testCmd)

	testCmd.Flags().StringVarP(&port, "port", "p", "/dev/ttyUSB0", "Serial port device")
	testCmd.Flags().IntVarP(&baudRate, "baud", "b", 115200, "Baud rate")
	testCmd.Flags().IntVarP(&count, "count", "c", 1000, "Number of test iterations")

	viper.BindPFlag("port", testCmd.Flags().Lookup("port"))
	viper.BindPFlag("baud", testCmd.Flags().Lookup("baud"))
	viper.BindPFlag("count", testCmd.Flags().Lookup("count"))
}
