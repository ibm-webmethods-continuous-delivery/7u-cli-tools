package cmd

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"

	"github.com/ibm-webmethods-continuous-delivery/7u-cli-tools/code/archictl/internal/archi"
	"github.com/ibm-webmethods-continuous-delivery/7u-cli-tools/code/archictl/internal/ops"
)

func newSplitCmd() *cobra.Command {
	var out string
	var ids bool
	c := &cobra.Command{
		Use:   "split <model.archimate> --out <dir> [--ids]",
		Short: "Convert an Archi model to a directory of compact YAML files",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, a []string) error {
			text, err := archi.ReadXML(a[0])
			if err != nil {
				return err
			}
			files, err := archi.SplitToFiles(text, !ids)
			if err != nil {
				return wrapErr(err)
			}
			names := make([]string, 0, len(files))
			for n := range files {
				names = append(names, n)
			}
			sort.Strings(names)
			for _, n := range names {
				p := filepath.Join(out, filepath.FromSlash(n))
				if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
					return err
				}
				if err := ops.WriteFileAtomic(p, []byte(files[n])); err != nil {
					return err
				}
			}
			return nil
		},
	}
	c.Flags().StringVar(&out, "out", "", "Output directory (required)")
	c.Flags().BoolVar(&ids, "ids", false, "Write relation endpoints as ids instead of names")
	_ = c.MarkFlagRequired("out")
	return c
}

func newJoinCmd() *cobra.Command {
	var out string
	var noRules bool
	c := &cobra.Command{
		Use:   "join <dir> --out <model.archimate> [--no-rules]",
		Short: "Rebuild an Archi model from a YAML directory (validates first)",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, a []string) error {
			files, err := readYAMLDir(a[0])
			if err != nil {
				return err
			}
			text, err := archi.JoinFromFiles(files, true, !noRules)
			if err != nil {
				return wrapErr(err)
			}
			return ops.WriteFileAtomic(out, []byte(text))
		},
	}
	c.Flags().StringVar(&out, "out", "", "Output .archimate file (required)")
	c.Flags().BoolVar(&noRules, "no-rules", false, "Skip the relationship legality check")
	_ = c.MarkFlagRequired("out")
	return c
}

func newRoundtripCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "roundtrip <model.archimate>",
		Short: "Split then join in memory and report whether the canonical form is unchanged",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, a []string) error {
			text, err := archi.ReadXML(a[0])
			if err != nil {
				return err
			}
			for _, mode := range []struct {
				name  string
				names bool
			}{{"names", true}, {"ids", false}} {
				ok, err := archi.RoundTrip(text, mode.names)
				if err != nil {
					return wrapErr(err)
				}
				if !ok {
					return refused("roundtrip (%s mode): canonical form differs", mode.name)
				}
			}
			println(c, "OK")
			return nil
		},
	}
}

func init() {
	rootCmd.AddCommand(newSplitCmd(), newJoinCmd(), newRoundtripCmd())
}
