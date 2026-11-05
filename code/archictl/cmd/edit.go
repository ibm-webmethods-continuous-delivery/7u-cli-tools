package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ibm-webmethods-continuous-delivery/7u-cli-tools/code/archictl/internal/archi"
	"github.com/ibm-webmethods-continuous-delivery/7u-cli-tools/code/archictl/internal/ops"
)

func isDir(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

func newAddCmd() *cobra.Command {
	var folder, doc string
	return editCmd("add <model> <Type> <name> [--folder F] [--doc D]", "Add an element",
		cobra.ExactArgs(3),
		func(c *cobra.Command) {
			c.Flags().StringVar(&folder, "folder", "", "Sub-folder path, e.g. Processes/Claims (created if missing)")
			c.Flags().StringVar(&doc, "doc", "", "Documentation text")
		},
		func(c *cobra.Command, m *ops.Model, a []string) (string, error) {
			e, err := ops.Add(m, a[0], a[1], folder, doc)
			if err != nil {
				return "", err
			}
			return "added " + m.Label(m.ByID[e.Str("id")]), nil
		})
}

func newConnectCmd() *cobra.Command {
	var name string
	return editCmd("connect <model> <Type> <source> <target> [--name N]", "Add a relationship",
		cobra.ExactArgs(4),
		func(c *cobra.Command) {
			c.Flags().StringVar(&name, "name", "", "Relationship name")
		},
		func(c *cobra.Command, m *ops.Model, a []string) (string, error) {
			r, err := ops.Connect(m, a[0], a[1], a[2], name)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("added %s #%s", m.ByID[r.Str("id")].Type(), r.Str("id")), nil
		})
}

func newRenameCmd() *cobra.Command {
	return editCmd("rename <model> <ref> <new-name>", "Rename an element",
		cobra.ExactArgs(3), nil,
		func(c *cobra.Command, m *ops.Model, a []string) (string, error) {
			e, old, err := ops.Rename(m, a[0], a[1])
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("renamed %s -> %s #%s", archi.PyRepr(old), archi.PyRepr(a[1]), e.Str("id")), nil
		})
}

func newDeleteCmd() *cobra.Command {
	return editCmd("delete <model> <ref>", "Delete an element with its relations and view items",
		cobra.ExactArgs(2), nil,
		func(c *cobra.Command, m *ops.Model, a []string) (string, error) {
			n, err := m.Resolve(a[0])
			if err != nil {
				return "", err
			}
			typ := n.Type()
			e, rels, nodes, err := ops.Delete(m, a[0])
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("deleted %s %s #%s; %d relationship(s) and %d view item(s) removed",
				typ, e.Str("name"), e.Str("id"), rels, nodes), nil
		})
}

func newAddToViewCmd() *cobra.Command {
	var x, y int
	return editCmd("add-to-view <model> <view> <ref> [--x X --y Y]", "Place an element on a view and draw its relations",
		cobra.ExactArgs(3),
		func(c *cobra.Command) {
			c.Flags().IntVar(&x, "x", 0, "X position (with --y)")
			c.Flags().IntVar(&y, "y", 0, "Y position (with --x)")
		},
		func(c *cobra.Command, m *ops.Model, a []string) (string, error) {
			var px, py *int
			if c.Flags().Changed("x") && c.Flags().Changed("y") {
				px, py = &x, &y
			}
			view, e, node, n, err := ops.AddToView(m, a[0], a[1], px, py)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("added %s to view %s with %d connection(s) #%s",
				archi.PyRepr(e.Str("name")), archi.PyRepr(view.Str("name")), n, node.Str("id")), nil
		})
}

func init() {
	rootCmd.AddCommand(newAddCmd(), newConnectCmd(), newRenameCmd(), newDeleteCmd(), newAddToViewCmd())
}
