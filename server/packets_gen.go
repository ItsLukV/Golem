// Code generated from the vanilla packet report. DO NOT EDIT.
//
// Source: https://github.com/mj41/mc26-data/blob/mc-26.2/packets.json
// Minecraft 26.2, protocol 776, world version 4903.
//
// The constants are untyped so they work both as Packet.PacketId (int) and in
// comparisons against the uint64 packet ID returned by connection.readPacket.

package server

// Handshake state, serverbound.
const (
	HandshakeSbIntention = 0x00 // minecraft:intention
)

// Status state, clientbound.
const (
	StatusCbStatusResponse = 0x00 // minecraft:status_response
	StatusCbPongResponse   = 0x01 // minecraft:pong_response
)

// Status state, serverbound.
const (
	StatusSbStatusRequest = 0x00 // minecraft:status_request
	StatusSbPingRequest   = 0x01 // minecraft:ping_request
)

// Login state, clientbound.
const (
	LoginCbLoginDisconnect  = 0x00 // minecraft:login_disconnect
	LoginCbHello            = 0x01 // minecraft:hello
	LoginCbLoginFinished    = 0x02 // minecraft:login_finished
	LoginCbLoginCompression = 0x03 // minecraft:login_compression
	LoginCbCustomQuery      = 0x04 // minecraft:custom_query
	LoginCbCookieRequest    = 0x05 // minecraft:cookie_request
)

// Login state, serverbound.
const (
	LoginSbHello             = 0x00 // minecraft:hello
	LoginSbKey               = 0x01 // minecraft:key
	LoginSbCustomQueryAnswer = 0x02 // minecraft:custom_query_answer
	LoginSbLoginAcknowledged = 0x03 // minecraft:login_acknowledged
	LoginSbCookieResponse    = 0x04 // minecraft:cookie_response
)

// Configuration state, clientbound.
const (
	ConfigCbCookieRequest         = 0x00 // minecraft:cookie_request
	ConfigCbCustomPayload         = 0x01 // minecraft:custom_payload
	ConfigCbDisconnect            = 0x02 // minecraft:disconnect
	ConfigCbFinishConfiguration   = 0x03 // minecraft:finish_configuration
	ConfigCbKeepAlive             = 0x04 // minecraft:keep_alive
	ConfigCbPing                  = 0x05 // minecraft:ping
	ConfigCbResetChat             = 0x06 // minecraft:reset_chat
	ConfigCbRegistryData          = 0x07 // minecraft:registry_data
	ConfigCbResourcePackPop       = 0x08 // minecraft:resource_pack_pop
	ConfigCbResourcePackPush      = 0x09 // minecraft:resource_pack_push
	ConfigCbStoreCookie           = 0x0A // minecraft:store_cookie
	ConfigCbTransfer              = 0x0B // minecraft:transfer
	ConfigCbUpdateEnabledFeatures = 0x0C // minecraft:update_enabled_features
	ConfigCbUpdateTags            = 0x0D // minecraft:update_tags
	ConfigCbSelectKnownPacks      = 0x0E // minecraft:select_known_packs
	ConfigCbCustomReportDetails   = 0x0F // minecraft:custom_report_details
	ConfigCbServerLinks           = 0x10 // minecraft:server_links
	ConfigCbClearDialog           = 0x11 // minecraft:clear_dialog
	ConfigCbShowDialog            = 0x12 // minecraft:show_dialog
	ConfigCbCodeOfConduct         = 0x13 // minecraft:code_of_conduct
)

// Configuration state, serverbound.
const (
	ConfigSbClientInformation   = 0x00 // minecraft:client_information
	ConfigSbCookieResponse      = 0x01 // minecraft:cookie_response
	ConfigSbCustomPayload       = 0x02 // minecraft:custom_payload
	ConfigSbFinishConfiguration = 0x03 // minecraft:finish_configuration
	ConfigSbKeepAlive           = 0x04 // minecraft:keep_alive
	ConfigSbPong                = 0x05 // minecraft:pong
	ConfigSbResourcePack        = 0x06 // minecraft:resource_pack
	ConfigSbSelectKnownPacks    = 0x07 // minecraft:select_known_packs
	ConfigSbCustomClickAction   = 0x08 // minecraft:custom_click_action
	ConfigSbAcceptCodeOfConduct = 0x09 // minecraft:accept_code_of_conduct
)

// Play state, clientbound.
const (
	PlayCbBundleDelimiter          = 0x00 // minecraft:bundle_delimiter
	PlayCbAddEntity                = 0x01 // minecraft:add_entity
	PlayCbAnimate                  = 0x02 // minecraft:animate
	PlayCbAwardStats               = 0x03 // minecraft:award_stats
	PlayCbBlockChangedAck          = 0x04 // minecraft:block_changed_ack
	PlayCbBlockDestruction         = 0x05 // minecraft:block_destruction
	PlayCbBlockEntityData          = 0x06 // minecraft:block_entity_data
	PlayCbBlockEvent               = 0x07 // minecraft:block_event
	PlayCbBlockUpdate              = 0x08 // minecraft:block_update
	PlayCbBossEvent                = 0x09 // minecraft:boss_event
	PlayCbChangeDifficulty         = 0x0A // minecraft:change_difficulty
	PlayCbChunkBatchFinished       = 0x0B // minecraft:chunk_batch_finished
	PlayCbChunkBatchStart          = 0x0C // minecraft:chunk_batch_start
	PlayCbChunksBiomes             = 0x0D // minecraft:chunks_biomes
	PlayCbClearTitles              = 0x0E // minecraft:clear_titles
	PlayCbCommandSuggestions       = 0x0F // minecraft:command_suggestions
	PlayCbCommands                 = 0x10 // minecraft:commands
	PlayCbContainerClose           = 0x11 // minecraft:container_close
	PlayCbContainerSetContent      = 0x12 // minecraft:container_set_content
	PlayCbContainerSetData         = 0x13 // minecraft:container_set_data
	PlayCbContainerSetSlot         = 0x14 // minecraft:container_set_slot
	PlayCbCookieRequest            = 0x15 // minecraft:cookie_request
	PlayCbCooldown                 = 0x16 // minecraft:cooldown
	PlayCbCustomChatCompletions    = 0x17 // minecraft:custom_chat_completions
	PlayCbCustomPayload            = 0x18 // minecraft:custom_payload
	PlayCbDamageEvent              = 0x19 // minecraft:damage_event
	PlayCbDebugBlockValue          = 0x1A // minecraft:debug/block_value
	PlayCbDebugChunkValue          = 0x1B // minecraft:debug/chunk_value
	PlayCbDebugEntityValue         = 0x1C // minecraft:debug/entity_value
	PlayCbDebugEvent               = 0x1D // minecraft:debug/event
	PlayCbDebugSample              = 0x1E // minecraft:debug_sample
	PlayCbDeleteChat               = 0x1F // minecraft:delete_chat
	PlayCbDisconnect               = 0x20 // minecraft:disconnect
	PlayCbDisguisedChat            = 0x21 // minecraft:disguised_chat
	PlayCbEntityEvent              = 0x22 // minecraft:entity_event
	PlayCbEntityPositionSync       = 0x23 // minecraft:entity_position_sync
	PlayCbExplode                  = 0x24 // minecraft:explode
	PlayCbForgetLevelChunk         = 0x25 // minecraft:forget_level_chunk
	PlayCbGameEvent                = 0x26 // minecraft:game_event
	PlayCbGameRuleValues           = 0x27 // minecraft:game_rule_values
	PlayCbGameTestHighlightPos     = 0x28 // minecraft:game_test_highlight_pos
	PlayCbMountScreenOpen          = 0x29 // minecraft:mount_screen_open
	PlayCbHurtAnimation            = 0x2A // minecraft:hurt_animation
	PlayCbInitializeBorder         = 0x2B // minecraft:initialize_border
	PlayCbKeepAlive                = 0x2C // minecraft:keep_alive
	PlayCbLevelChunkWithLight      = 0x2D // minecraft:level_chunk_with_light
	PlayCbLevelEvent               = 0x2E // minecraft:level_event
	PlayCbLevelParticles           = 0x2F // minecraft:level_particles
	PlayCbLightUpdate              = 0x30 // minecraft:light_update
	PlayCbLogin                    = 0x31 // minecraft:login
	PlayCbLowDiskSpaceWarning      = 0x32 // minecraft:low_disk_space_warning
	PlayCbMapItemData              = 0x33 // minecraft:map_item_data
	PlayCbMerchantOffers           = 0x34 // minecraft:merchant_offers
	PlayCbMoveEntityPos            = 0x35 // minecraft:move_entity_pos
	PlayCbMoveEntityPosRot         = 0x36 // minecraft:move_entity_pos_rot
	PlayCbMoveMinecartAlongTrack   = 0x37 // minecraft:move_minecart_along_track
	PlayCbMoveEntityRot            = 0x38 // minecraft:move_entity_rot
	PlayCbMoveVehicle              = 0x39 // minecraft:move_vehicle
	PlayCbOpenBook                 = 0x3A // minecraft:open_book
	PlayCbOpenScreen               = 0x3B // minecraft:open_screen
	PlayCbOpenSignEditor           = 0x3C // minecraft:open_sign_editor
	PlayCbPing                     = 0x3D // minecraft:ping
	PlayCbPongResponse             = 0x3E // minecraft:pong_response
	PlayCbPlaceGhostRecipe         = 0x3F // minecraft:place_ghost_recipe
	PlayCbPlayerAbilities          = 0x40 // minecraft:player_abilities
	PlayCbPlayerChat               = 0x41 // minecraft:player_chat
	PlayCbPlayerCombatEnd          = 0x42 // minecraft:player_combat_end
	PlayCbPlayerCombatEnter        = 0x43 // minecraft:player_combat_enter
	PlayCbPlayerCombatKill         = 0x44 // minecraft:player_combat_kill
	PlayCbPlayerInfoRemove         = 0x45 // minecraft:player_info_remove
	PlayCbPlayerInfoUpdate         = 0x46 // minecraft:player_info_update
	PlayCbPlayerLookAt             = 0x47 // minecraft:player_look_at
	PlayCbPlayerPosition           = 0x48 // minecraft:player_position
	PlayCbPlayerRotation           = 0x49 // minecraft:player_rotation
	PlayCbRecipeBookAdd            = 0x4A // minecraft:recipe_book_add
	PlayCbRecipeBookRemove         = 0x4B // minecraft:recipe_book_remove
	PlayCbRecipeBookSettings       = 0x4C // minecraft:recipe_book_settings
	PlayCbRemoveEntities           = 0x4D // minecraft:remove_entities
	PlayCbRemoveMobEffect          = 0x4E // minecraft:remove_mob_effect
	PlayCbResetScore               = 0x4F // minecraft:reset_score
	PlayCbResourcePackPop          = 0x50 // minecraft:resource_pack_pop
	PlayCbResourcePackPush         = 0x51 // minecraft:resource_pack_push
	PlayCbRespawn                  = 0x52 // minecraft:respawn
	PlayCbRotateHead               = 0x53 // minecraft:rotate_head
	PlayCbSectionBlocksUpdate      = 0x54 // minecraft:section_blocks_update
	PlayCbSelectAdvancementsTab    = 0x55 // minecraft:select_advancements_tab
	PlayCbServerData               = 0x56 // minecraft:server_data
	PlayCbSetActionBarText         = 0x57 // minecraft:set_action_bar_text
	PlayCbSetBorderCenter          = 0x58 // minecraft:set_border_center
	PlayCbSetBorderLerpSize        = 0x59 // minecraft:set_border_lerp_size
	PlayCbSetBorderSize            = 0x5A // minecraft:set_border_size
	PlayCbSetBorderWarningDelay    = 0x5B // minecraft:set_border_warning_delay
	PlayCbSetBorderWarningDistance = 0x5C // minecraft:set_border_warning_distance
	PlayCbSetCamera                = 0x5D // minecraft:set_camera
	PlayCbSetChunkCacheCenter      = 0x5E // minecraft:set_chunk_cache_center
	PlayCbSetChunkCacheRadius      = 0x5F // minecraft:set_chunk_cache_radius
	PlayCbSetCursorItem            = 0x60 // minecraft:set_cursor_item
	PlayCbSetDefaultSpawnPosition  = 0x61 // minecraft:set_default_spawn_position
	PlayCbSetDisplayObjective      = 0x62 // minecraft:set_display_objective
	PlayCbSetEntityData            = 0x63 // minecraft:set_entity_data
	PlayCbSetEntityLink            = 0x64 // minecraft:set_entity_link
	PlayCbSetEntityMotion          = 0x65 // minecraft:set_entity_motion
	PlayCbSetEquipment             = 0x66 // minecraft:set_equipment
	PlayCbSetExperience            = 0x67 // minecraft:set_experience
	PlayCbSetHealth                = 0x68 // minecraft:set_health
	PlayCbSetHeldSlot              = 0x69 // minecraft:set_held_slot
	PlayCbSetObjective             = 0x6A // minecraft:set_objective
	PlayCbSetPassengers            = 0x6B // minecraft:set_passengers
	PlayCbSetPlayerInventory       = 0x6C // minecraft:set_player_inventory
	PlayCbSetPlayerTeam            = 0x6D // minecraft:set_player_team
	PlayCbSetScore                 = 0x6E // minecraft:set_score
	PlayCbSetSimulationDistance    = 0x6F // minecraft:set_simulation_distance
	PlayCbSetSubtitleText          = 0x70 // minecraft:set_subtitle_text
	PlayCbSetTime                  = 0x71 // minecraft:set_time
	PlayCbSetTitleText             = 0x72 // minecraft:set_title_text
	PlayCbSetTitlesAnimation       = 0x73 // minecraft:set_titles_animation
	PlayCbSoundEntity              = 0x74 // minecraft:sound_entity
	PlayCbSound                    = 0x75 // minecraft:sound
	PlayCbStartConfiguration       = 0x76 // minecraft:start_configuration
	PlayCbStopSound                = 0x77 // minecraft:stop_sound
	PlayCbStoreCookie              = 0x78 // minecraft:store_cookie
	PlayCbSystemChat               = 0x79 // minecraft:system_chat
	PlayCbTabList                  = 0x7A // minecraft:tab_list
	PlayCbTagQuery                 = 0x7B // minecraft:tag_query
	PlayCbTakeItemEntity           = 0x7C // minecraft:take_item_entity
	PlayCbTeleportEntity           = 0x7D // minecraft:teleport_entity
	PlayCbTestInstanceBlockStatus  = 0x7E // minecraft:test_instance_block_status
	PlayCbTickingState             = 0x7F // minecraft:ticking_state
	PlayCbTickingStep              = 0x80 // minecraft:ticking_step
	PlayCbTransfer                 = 0x81 // minecraft:transfer
	PlayCbUpdateAdvancements       = 0x82 // minecraft:update_advancements
	PlayCbUpdateAttributes         = 0x83 // minecraft:update_attributes
	PlayCbUpdateMobEffect          = 0x84 // minecraft:update_mob_effect
	PlayCbUpdateRecipes            = 0x85 // minecraft:update_recipes
	PlayCbUpdateTags               = 0x86 // minecraft:update_tags
	PlayCbProjectilePower          = 0x87 // minecraft:projectile_power
	PlayCbCustomReportDetails      = 0x88 // minecraft:custom_report_details
	PlayCbServerLinks              = 0x89 // minecraft:server_links
	PlayCbWaypoint                 = 0x8A // minecraft:waypoint
	PlayCbClearDialog              = 0x8B // minecraft:clear_dialog
	PlayCbShowDialog               = 0x8C // minecraft:show_dialog
)

// Play state, serverbound.
const (
	PlaySbAcceptTeleportation       = 0x00 // minecraft:accept_teleportation
	PlaySbAttack                    = 0x01 // minecraft:attack
	PlaySbBlockEntityTagQuery       = 0x02 // minecraft:block_entity_tag_query
	PlaySbBundleItemSelected        = 0x03 // minecraft:bundle_item_selected
	PlaySbChangeDifficulty          = 0x04 // minecraft:change_difficulty
	PlaySbChangeGameMode            = 0x05 // minecraft:change_game_mode
	PlaySbChatAck                   = 0x06 // minecraft:chat_ack
	PlaySbChatCommand               = 0x07 // minecraft:chat_command
	PlaySbChatCommandSigned         = 0x08 // minecraft:chat_command_signed
	PlaySbChat                      = 0x09 // minecraft:chat
	PlaySbChatSessionUpdate         = 0x0A // minecraft:chat_session_update
	PlaySbChunkBatchReceived        = 0x0B // minecraft:chunk_batch_received
	PlaySbClientCommand             = 0x0C // minecraft:client_command
	PlaySbClientTickEnd             = 0x0D // minecraft:client_tick_end
	PlaySbClientInformation         = 0x0E // minecraft:client_information
	PlaySbCommandSuggestion         = 0x0F // minecraft:command_suggestion
	PlaySbConfigurationAcknowledged = 0x10 // minecraft:configuration_acknowledged
	PlaySbContainerButtonClick      = 0x11 // minecraft:container_button_click
	PlaySbContainerClick            = 0x12 // minecraft:container_click
	PlaySbContainerClose            = 0x13 // minecraft:container_close
	PlaySbContainerSlotStateChanged = 0x14 // minecraft:container_slot_state_changed
	PlaySbCookieResponse            = 0x15 // minecraft:cookie_response
	PlaySbCustomPayload             = 0x16 // minecraft:custom_payload
	PlaySbDebugSubscriptionRequest  = 0x17 // minecraft:debug_subscription_request
	PlaySbEditBook                  = 0x18 // minecraft:edit_book
	PlaySbEntityTagQuery            = 0x19 // minecraft:entity_tag_query
	PlaySbInteract                  = 0x1A // minecraft:interact
	PlaySbJigsawGenerate            = 0x1B // minecraft:jigsaw_generate
	PlaySbKeepAlive                 = 0x1C // minecraft:keep_alive
	PlaySbLockDifficulty            = 0x1D // minecraft:lock_difficulty
	PlaySbMovePlayerPos             = 0x1E // minecraft:move_player_pos
	PlaySbMovePlayerPosRot          = 0x1F // minecraft:move_player_pos_rot
	PlaySbMovePlayerRot             = 0x20 // minecraft:move_player_rot
	PlaySbMovePlayerStatusOnly      = 0x21 // minecraft:move_player_status_only
	PlaySbMoveVehicle               = 0x22 // minecraft:move_vehicle
	PlaySbPaddleBoat                = 0x23 // minecraft:paddle_boat
	PlaySbPickItemFromBlock         = 0x24 // minecraft:pick_item_from_block
	PlaySbPickItemFromEntity        = 0x25 // minecraft:pick_item_from_entity
	PlaySbPingRequest               = 0x26 // minecraft:ping_request
	PlaySbPlaceRecipe               = 0x27 // minecraft:place_recipe
	PlaySbPlayerAbilities           = 0x28 // minecraft:player_abilities
	PlaySbPlayerAction              = 0x29 // minecraft:player_action
	PlaySbPlayerCommand             = 0x2A // minecraft:player_command
	PlaySbPlayerInput               = 0x2B // minecraft:player_input
	PlaySbPlayerLoaded              = 0x2C // minecraft:player_loaded
	PlaySbPong                      = 0x2D // minecraft:pong
	PlaySbRecipeBookChangeSettings  = 0x2E // minecraft:recipe_book_change_settings
	PlaySbRecipeBookSeenRecipe      = 0x2F // minecraft:recipe_book_seen_recipe
	PlaySbRenameItem                = 0x30 // minecraft:rename_item
	PlaySbResourcePack              = 0x31 // minecraft:resource_pack
	PlaySbSeenAdvancements          = 0x32 // minecraft:seen_advancements
	PlaySbSelectTrade               = 0x33 // minecraft:select_trade
	PlaySbSetBeacon                 = 0x34 // minecraft:set_beacon
	PlaySbSetCarriedItem            = 0x35 // minecraft:set_carried_item
	PlaySbSetCommandBlock           = 0x36 // minecraft:set_command_block
	PlaySbSetCommandMinecart        = 0x37 // minecraft:set_command_minecart
	PlaySbSetCreativeModeSlot       = 0x38 // minecraft:set_creative_mode_slot
	PlaySbSetGameRule               = 0x39 // minecraft:set_game_rule
	PlaySbSetJigsawBlock            = 0x3A // minecraft:set_jigsaw_block
	PlaySbSetStructureBlock         = 0x3B // minecraft:set_structure_block
	PlaySbSetTestBlock              = 0x3C // minecraft:set_test_block
	PlaySbSignUpdate                = 0x3D // minecraft:sign_update
	PlaySbSpectatorAction           = 0x3E // minecraft:spectator_action
	PlaySbSwing                     = 0x3F // minecraft:swing
	PlaySbTeleportToEntity          = 0x40 // minecraft:teleport_to_entity
	PlaySbTestInstanceBlockAction   = 0x41 // minecraft:test_instance_block_action
	PlaySbUseItemOn                 = 0x42 // minecraft:use_item_on
	PlaySbUseItem                   = 0x43 // minecraft:use_item
	PlaySbCustomClickAction         = 0x44 // minecraft:custom_click_action
)
