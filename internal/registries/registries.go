package registries

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/itslukv/minecraftserver/data"
)

var SyncedRegistries = []string{
	"banner_pattern", "chat_type", "damage_type", "dialog", "dimension_type",
	"enchantment", "instrument", "jukebox_song", "painting_variant",
	"sulfur_cube_archetype", "test_environment", "test_instance", "timeline",
	"trim_material", "trim_pattern", "world_clock", "worldgen/biome",
	"cat_variant", "cat_sound_variant", "chicken_variant", "chicken_sound_variant",
	"cow_variant", "cow_sound_variant", "frog_variant", "pig_variant",
	"pig_sound_variant", "wolf_variant", "wolf_sound_variant",
	"zombie_nautilus_variant",
}

type Entry struct {
	ID   string
	Data any
}

type Registry struct {
	ID      string
	Entries []Entry
}

func LoadRegistries() ([]Registry, error) {
	regs := make([]Registry, 0, len(SyncedRegistries))
	for _, name := range SyncedRegistries {
		root := path.Join("minecraft", name)
		reg := Registry{ID: "minecraft:" + name}

		err := fs.WalkDir(data.FS, root, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || path.Ext(p) != ".json" {
				return nil
			}
			raw, err := data.FS.ReadFile(p)
			if err != nil {
				return err
			}
			dec := json.NewDecoder(bytes.NewReader(raw))
			dec.UseNumber()
			var v any
			if err := dec.Decode(&v); err != nil {
				return fmt.Errorf("%s: %w", p, err)
			}
			id := strings.TrimSuffix(strings.TrimPrefix(p, root+"/"), ".json")
			reg.Entries = append(reg.Entries, Entry{ID: "minecraft:" + id, Data: v})
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("registry %s: %w", name, err)
		}
		regs = append(regs, reg)
	}
	return regs, nil
}
