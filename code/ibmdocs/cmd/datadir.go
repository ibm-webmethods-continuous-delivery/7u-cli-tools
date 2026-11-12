package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var dataDirCmd = &cobra.Command{
	Use:   "data-dir",
	Short: "Print the resolved data folder",
	Long: `Prints the data folder that other commands would use, honouring --data and
the resolution order described in the root help. Only the path is written to
stdout. Use -v to see which rule selected it.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(cfg.DataDir)
	},
}

func init() {
	rootCmd.AddCommand(dataDirCmd)
}
