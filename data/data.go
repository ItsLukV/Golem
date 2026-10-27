package data

import "embed"

//go:embed minecraft/banner_pattern minecraft/chat_type minecraft/damage_type
//go:embed minecraft/dialog minecraft/dimension_type minecraft/enchantment
//go:embed minecraft/instrument minecraft/jukebox_song minecraft/painting_variant
//go:embed minecraft/sulfur_cube_archetype minecraft/test_environment
//go:embed minecraft/test_instance minecraft/timeline minecraft/trim_material
//go:embed minecraft/trim_pattern minecraft/world_clock minecraft/worldgen/biome
//go:embed minecraft/cat_variant minecraft/cat_sound_variant
//go:embed minecraft/chicken_variant minecraft/chicken_sound_variant
//go:embed minecraft/cow_variant minecraft/cow_sound_variant minecraft/frog_variant
//go:embed minecraft/pig_variant minecraft/pig_sound_variant
//go:embed minecraft/wolf_variant minecraft/wolf_sound_variant
//go:embed minecraft/zombie_nautilus_variant
//go:embed minecraft/tags
var FS embed.FS

//go:embed minecraft/registries.json
var RegistriesJSON []byte
