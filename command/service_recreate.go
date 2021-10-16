package command

import (
	"github.com/spf13/cobra"
)

func init() {
	serviceCmd.AddCommand(recreateCmd)
	recreateCmd.Flags().StringVarP(&source, "service", "s", "", "Recreate single service")
}

var recreateCmd = &cobra.Command{
	Use:   "recreate",
	Short: "Recreate containers",
	Long:  `Recreate containers.`,
	Run: func(cmd *cobra.Command, args []string) {
		down()
		up()
	},
}
