package lazyver

import (
	"fmt"

	"github.com/audryus/lazyver/internal/lazyver/sem"
	"github.com/spf13/cobra"
)

var semverCmd = &cobra.Command{
	Use:     "semver",
	Aliases: []string{"s"},
	Short:   "Semver versioning",
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		v := sem.Run(path)
		if print {
			fmt.Printf("v%d.%d.%d", v.Major, v.Minor, v.Patch)
		}
	},
}

func init() {
	semverCmd.Flags().StringVar(&path, "path", "./", "Path to repository folder")
	semverCmd.Flags().BoolVarP(&print, "output", "o", false, "Print version")
	rootCmd.AddCommand(semverCmd)
}
