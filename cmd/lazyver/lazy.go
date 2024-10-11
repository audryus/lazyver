package lazyver

import (
	"fmt"

	"github.com/audryus/lazyver/internal/lazyver/lazy"
	"github.com/spf13/cobra"
)

var path string
var print bool

var lazyCmd = &cobra.Command{
	Use:     "lazy",
	Aliases: []string{"l"},
	Short:   "Lazy versioning",
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		v := lazy.Run(path)
		if print {
			fmt.Print(v)
		}
	},
}

func init() {
	lazyCmd.Flags().StringVar(&path, "path", "./", "Path to repository folder")
	lazyCmd.Flags().BoolVarP(&print, "output", "o", false, "Print version")
	rootCmd.AddCommand(lazyCmd)
}
