package cmd

import (
	"github.com/spf13/cobra"

	"github.com/ibm-webmethods-continuous-delivery/7u-cli-tools/code/archictl/internal/archi"
	"github.com/ibm-webmethods-continuous-delivery/7u-cli-tools/code/archictl/internal/ops"
)

func newFindCmd() *cobra.Command {
	var typ string
	c := &cobra.Command{
		Use:   "find <model> [text] [--type T]",
		Short: "Case-insensitive search of element and relationship names (or exact id)",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(c *cobra.Command, a []string) error {
			m, err := ops.LoadModel(a[0])
			if err != nil {
				return err
			}
			q := ""
			if len(a) > 1 {
				q = a[1]
			}
			items := ops.Find(m, q, typ)
			if cfg.JSON {
				printJSON(c, items)
			} else if len(items) > 0 {
				println(c, ops.FormatFind(items))
			}
			return nil
		},
	}
	c.Flags().StringVar(&typ, "type", "", "Restrict to this element or relationship type")
	return c
}

func newShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <model> <ref>",
		Short: "Element details: documentation, properties, relations in/out, views",
		Args:  cobra.ExactArgs(2),
		RunE: func(c *cobra.Command, a []string) error {
			m, err := ops.LoadModel(a[0])
			if err != nil {
				return err
			}
			d, err := ops.Show(m, a[1])
			if err != nil {
				return wrapErr(err)
			}
			if cfg.JSON {
				printJSON(c, d)
			} else {
				println(c, ops.FormatShow(m, d))
			}
			return nil
		},
	}
}

func newOrphansCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "orphans <model>",
		Short: "Elements and relationships that appear in no view",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, a []string) error {
			m, err := ops.LoadModel(a[0])
			if err != nil {
				return err
			}
			d := ops.Orphans(m)
			if cfg.JSON {
				printJSON(c, d)
			} else {
				println(c, ops.FormatOrphans(d))
			}
			return nil
		},
	}
}

func newUnusedCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unused <model>",
		Short: "Views that contain no model elements",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, a []string) error {
			m, err := ops.LoadModel(a[0])
			if err != nil {
				return err
			}
			items := ops.Unused(m)
			if cfg.JSON {
				printJSON(c, items)
			} else {
				println(c, ops.FormatUnused(items))
			}
			return nil
		},
	}
}

func newValidateCmd() *cobra.Command {
	var noRules bool
	c := &cobra.Command{
		Use:   "validate <model-or-dir> [--no-rules]",
		Short: "Reference and relationship-legality validation",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, a []string) error {
			var obj archi.Obj
			if isDir(a[0]) {
				files, err := readYAMLDir(a[0])
				if err != nil {
					return err
				}
				if obj, err = archi.JoinToObj(files); err != nil {
					return wrapErr(err)
				}
			} else {
				m, err := ops.LoadModel(a[0])
				if err != nil {
					return err
				}
				obj = m.Root
			}
			errs := archi.ValidateObj(obj, !noRules)
			if cfg.JSON {
				list := make([]any, len(errs))
				for i, e := range errs {
					list[i] = e
				}
				printJSON(c, archi.O("ok", len(errs) == 0, "errors", list))
			} else if len(errs) == 0 {
				println(c, "OK")
			} else {
				for _, e := range errs {
					println(c, e)
				}
			}
			if len(errs) > 0 {
				return refused("%d validation error(s)", len(errs))
			}
			return nil
		},
	}
	c.Flags().BoolVar(&noRules, "no-rules", false, "Skip the relationship legality check")
	return c
}

func init() {
	rootCmd.AddCommand(newFindCmd(), newShowCmd(), newOrphansCmd(), newUnusedCmd(), newValidateCmd())
}
