package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ibm-webmethods-continuous-delivery/7u-cli-tools/code/archictl/internal/archi"
	"github.com/ibm-webmethods-continuous-delivery/7u-cli-tools/code/archictl/internal/ops"
)

// wrapErr maps domain refusals from internal packages to exit code 1; everything else stays fatal (2).
func wrapErr(err error) error {
	var r *archi.Refusal
	if errors.As(err, &r) {
		return &refusedError{r.Msg}
	}
	return err
}

func println(cmd *cobra.Command, s string) { fmt.Fprintln(cmd.OutOrStdout(), s) }

// printJSON writes v as one JSON document (oracle-style separators).
func printJSON(cmd *cobra.Command, v any) { println(cmd, archi.PyJSON(v)) }

// readYAMLDir loads every *.yaml under dir keyed by its slash-separated relative path.
func readYAMLDir(dir string) (map[string]string, error) {
	files := map[string]string{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".yaml") {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(rel)] = string(b)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no .yaml files found in %s", dir)
	}
	return files, nil
}

// editCmd builds an edit command: lock, load, pre-validate, apply, post-validate, atomic write.
func editCmd(use, short string, args cobra.PositionalArgs, setup func(c *cobra.Command),
	apply func(c *cobra.Command, m *ops.Model, a []string) (string, error)) *cobra.Command {
	var out string
	c := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  args,
		RunE: func(c *cobra.Command, a []string) error {
			msg, err := ops.RunEdit(a[0], out, func(m *ops.Model) (string, error) {
				return apply(c, m, a[1:])
			})
			if err != nil {
				return wrapErr(err)
			}
			println(c, msg)
			return nil
		},
	}
	c.Flags().StringVar(&out, "out", "", "Write the edited model here instead of in place")
	if setup != nil {
		setup(c)
	}
	return c
}
