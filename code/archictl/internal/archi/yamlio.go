package archi

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// All YAML scalars are strings; key order follows the Obj insertion order; output is block style.

func yamlScalar(s string) *yaml.Node {
	n := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: s}
	if strings.Contains(s, "\n") {
		if strings.Contains(s, "\r") {
			n.Style = yaml.DoubleQuotedStyle
		} else {
			n.Style = yaml.LiteralStyle
		}
	}
	return n
}

func toYAMLNode(v any) (*yaml.Node, error) {
	switch x := v.(type) {
	case nil:
		return yamlScalar(""), nil
	case string:
		return yamlScalar(x), nil
	case Obj:
		n := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		if x == nil {
			return n, nil
		}
		for _, k := range x.keys {
			vn, err := toYAMLNode(x.m[k])
			if err != nil {
				return nil, err
			}
			n.Content = append(n.Content, yamlScalar(k), vn)
		}
		return n, nil
	case []any:
		n := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		for _, e := range x {
			en, err := toYAMLNode(e)
			if err != nil {
				return nil, err
			}
			n.Content = append(n.Content, en)
		}
		return n, nil
	}
	return nil, fmt.Errorf("cannot serialise %T to YAML", v)
}

// DumpYAML renders an Obj / []any / string tree as YAML text.
func DumpYAML(v any) (string, error) {
	n, err := toYAMLNode(v)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(n); err != nil {
		return "", err
	}
	if err := enc.Close(); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func fromYAMLNode(n *yaml.Node) (any, error) {
	switch n.Kind {
	case yaml.DocumentNode:
		if len(n.Content) == 0 {
			return nil, errors.New("empty YAML document")
		}
		return fromYAMLNode(n.Content[0])
	case yaml.AliasNode:
		return fromYAMLNode(n.Alias)
	case yaml.ScalarNode:
		return n.Value, nil
	case yaml.SequenceNode:
		out := make([]any, 0, len(n.Content))
		for _, c := range n.Content {
			v, err := fromYAMLNode(c)
			if err != nil {
				return nil, err
			}
			out = append(out, v)
		}
		return out, nil
	case yaml.MappingNode:
		o := NewObj()
		for i := 0; i+1 < len(n.Content); i += 2 {
			kn := n.Content[i]
			if kn.Kind == yaml.AliasNode {
				kn = kn.Alias
			}
			if kn.Kind != yaml.ScalarNode {
				return nil, fmt.Errorf("line %d: mapping keys must be scalars", kn.Line)
			}
			v, err := fromYAMLNode(n.Content[i+1])
			if err != nil {
				return nil, err
			}
			o.Set(kn.Value, v)
		}
		return o, nil
	}
	return nil, fmt.Errorf("line %d: unsupported YAML node", n.Line)
}

// LoadYAML parses YAML text into Obj / []any / string; every scalar is a string.
func LoadYAML(s string) (any, error) {
	var n yaml.Node
	if err := yaml.Unmarshal([]byte(s), &n); err != nil {
		return nil, err
	}
	if n.Kind == 0 {
		return nil, errors.New("empty YAML document")
	}
	return fromYAMLNode(&n)
}
