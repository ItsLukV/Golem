package registries

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/itslukv/minecraftserver/data"
)

type TagRegistry struct {
	ID   string
	Tags map[string][]int
}

func LoadTags(regs []Registry) ([]TagRegistry, error) {
	ids, err := idTables(regs)
	if err != nil {
		return nil, err
	}
	raw := map[string]map[string][]json.RawMessage{}
	err = fs.WalkDir(data.FS, "minecraft/tags", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || path.Ext(p) != ".json" {
			return err
		}

		rel := strings.TrimSuffix(strings.TrimPrefix(p, "minecraft/tags/"), ".json")
		regName, tagName, ok := splitTagPath(rel, ids)
		if !ok {
			return nil
		}
		data, err := data.FS.ReadFile(p)
		if err != nil {
			return err
		}
		var f struct {
			Values []json.RawMessage `json:"values"`
		}
		if err := json.Unmarshal(data, &f); err != nil {
			return err
		}
		if raw[regName] == nil {
			raw[regName] = map[string][]json.RawMessage{}
		}
		raw[regName]["minecraft:"+tagName] = f.Values
		return nil
	})
	if err != nil {
		return nil, err
	}

	var out []TagRegistry
	for regName, tags := range raw {
		table := ids[regName]
		resolved := map[string][]int{}
		var resolve func(tag string, stack map[string]bool) ([]int, error)
		resolve = func(tag string, stack map[string]bool) ([]int, error) {
			if r, ok := resolved[tag]; ok {
				return r, nil
			}
			values, ok := tags[tag]
			if !ok {
				return nil, fmt.Errorf("unknown tag #%s", tag)
			}
			if stack[tag] {
				return nil, fmt.Errorf("tag cycle at #%s", tag)
			}
			stack[tag] = true
			seen := map[int]bool{}
			var list []int
			for _, v := range values {
				id, required, err := parseTagValue(v)
				if err != nil {
					return nil, fmt.Errorf("#%s: %w", tag, err)
				}
				var add []int
				if strings.HasPrefix(id, "#") {
					sub, err := resolve(strings.TrimPrefix(id, "#"), stack)
					if err != nil {
						if !required {
							continue
						}
						return nil, err
					}
					add = sub
				} else if n, ok := table[id]; ok {
					add = []int{n}
				} else if required {
					return nil, fmt.Errorf("#%s: unknown entry %s", tag, id)
				}
				for _, n := range add {
					if !seen[n] {
						seen[n] = true
						list = append(list, n)
					}
				}
			}
			delete(stack, tag)
			resolved[tag] = list
			return list, nil
		}
		for tag := range tags {
			if _, err := resolve(tag, map[string]bool{}); err != nil {
				return nil, fmt.Errorf("registry %s: %w", regName, err)
			}
		}
		out = append(out, TagRegistry{ID: "minecraft:" + regName, Tags: resolved})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func parseTagValue(v json.RawMessage) (id string, required bool, err error) {
	if err := json.Unmarshal(v, &id); err == nil {
		return id, true, nil
	}
	var obj struct {
		ID       string `json:"id"`
		Required *bool  `json:"required"`
	}
	if err := json.Unmarshal(v, &obj); err != nil {
		return "", false, fmt.Errorf("bad tag value %s", v)
	}
	return obj.ID, obj.Required == nil || *obj.Required, nil
}

func splitTagPath(rel string, ids map[string]map[string]int) (reg, id string, ok bool) {
	parts := strings.Split(rel, "/")
	for i := len(parts) - 1; i >= 1; i-- {
		name := strings.Join(parts[:i], "/")
		if _, known := ids[name]; known {
			return name, strings.Join(parts[i:], "/"), true
		}
	}
	return "", "", false
}

func idTables(regs []Registry) (map[string]map[string]int, error) {
	var builtin map[string]struct {
		Entries map[string]struct {
			ProtocolID int `json:"protocol_id"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(data.RegistriesJSON, &builtin); err != nil {
		return nil, fmt.Errorf("registries.json: %w", err)
	}
	ids := map[string]map[string]int{}
	for name, r := range builtin {
		m := map[string]int{}
		for e, v := range r.Entries {
			m[e] = v.ProtocolID
		}
		ids[strings.TrimPrefix(name, "minecraft:")] = m
	}

	for _, r := range regs {
		m := map[string]int{}
		for i, e := range r.Entries {
			m[e.ID] = int(i)
		}
		ids[strings.TrimPrefix(r.ID, "minecraft:")] = m
	}

	return ids, nil
}
