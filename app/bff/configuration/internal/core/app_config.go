package core

import (
	"github.com/teamgram/proto/mtproto"
)

func jsonBool(v bool) *mtproto.JSONValue {
	b := mtproto.BoolFalse
	if v {
		b = mtproto.BoolTrue
	}
	return mtproto.MakeTLJsonBool(&mtproto.JSONValue{
		Value_BOOL: b,
	}).To_JSONValue()
}

func jsonNumber(v float64) *mtproto.JSONValue {
	return mtproto.MakeTLJsonNumber(&mtproto.JSONValue{
		Value_FLOAT64: v,
	}).To_JSONValue()
}

func jsonString(v string) *mtproto.JSONValue {
	return mtproto.MakeTLJsonString(&mtproto.JSONValue{
		Value_STRING: v,
	}).To_JSONValue()
}

func jsonArray(values ...*mtproto.JSONValue) *mtproto.JSONValue {
	return mtproto.MakeTLJsonArray(&mtproto.JSONValue{
		Value_VECTORJSONVALUE: values,
	}).To_JSONValue()
}

func jsonStringArray(values ...string) *mtproto.JSONValue {
	arr := make([]*mtproto.JSONValue, len(values))
	for i, v := range values {
		arr[i] = jsonString(v)
	}
	return jsonArray(arr...)
}

func kv(key string, value *mtproto.JSONValue) *mtproto.JSONObjectValue {
	return mtproto.MakeTLJsonObjectValue(&mtproto.JSONObjectValue{
		Key:   key,
		Value: value,
	}).To_JSONObjectValue()
}

func buildAppConfig() *mtproto.JSONValue {
	return mtproto.MakeTLJsonObject(&mtproto.JSONValue{
		Value_VECTORJSONOBJECTVALUE: []*mtproto.JSONObjectValue{
			// Pinned dialogs limits
			kv("dialogs_pinned_limit_default", jsonNumber(5)),
			kv("dialogs_pinned_limit_premium", jsonNumber(10)),

			// Folder limits
			kv("dialogs_folder_pinned_limit_default", jsonNumber(100)),
			kv("dialogs_folder_pinned_limit_premium", jsonNumber(200)),

			// Channel limits
			kv("channels_limit_default", jsonNumber(500)),
			kv("channels_limit_premium", jsonNumber(1000)),

			// Public channel/group creation rate limit (seconds)
			kv("channels_public_limit_default", jsonNumber(10)),
			kv("channels_public_limit_premium", jsonNumber(20)),

			// Caption length
			kv("caption_length_limit_default", jsonNumber(1024)),
			kv("caption_length_limit_premium", jsonNumber(2048)),

			// Upload file size (bytes)
			kv("upload_max_fileparts_default", jsonNumber(4000)),
			kv("upload_max_fileparts_premium", jsonNumber(8000)),

			// About / bio length
			kv("about_length_limit_default", jsonNumber(70)),
			kv("about_length_limit_premium", jsonNumber(140)),

			// Saved GIFs limit
			kv("saved_gifs_limit_default", jsonNumber(200)),
			kv("saved_gifs_limit_premium", jsonNumber(400)),

			// Sticker favorites limit
			kv("stickers_faved_limit_default", jsonNumber(5)),
			kv("stickers_faved_limit_premium", jsonNumber(200)),

			// Message length limit
			kv("message_length_max", jsonNumber(4096)),

			// WebFile DC id
			kv("webfile_dc_id", jsonNumber(4)),

			// Feature flags
			kv("qr_login_camera", jsonBool(true)),
			kv("qr_login_code", jsonString("disabled")),
			kv("dialog_filters_enabled", jsonBool(true)),
			kv("dialog_filters_tooltip", jsonBool(false)),
			kv("autoarchive_setting_available", jsonBool(true)),

			// Chat read marks
			kv("chat_read_mark_size_threshold", jsonNumber(50)),
			kv("chat_read_mark_expire_period", jsonNumber(604800)), // 7 days in seconds

			// Group call limits
			kv("groupcall_video_participants_max", jsonNumber(30)),

			// Reactions
			kv("reactions_uniq_max", jsonNumber(11)),
			kv("reactions_in_chat_max", jsonNumber(100)),
			kv("reactions_default", jsonString("👍")),

			// Premium
			kv("premium_purchase_blocked", jsonBool(false)),
			kv("premium_bot_username", jsonString("PremiumBot")),
			kv("premium_promo_order", jsonStringArray(
				"double_limits",
				"more_upload",
				"faster_download",
				"voice_to_text",
				"no_ads",
				"unique_reactions",
				"premium_stickers",
				"advanced_chat_management",
				"profile_badge",
				"animated_userpics",
			)),

			// NSFW / restriction
			kv("ignore_restriction_reasons", jsonStringArray()),
			kv("restriction_add_platforms", jsonStringArray()),

			// GIF search
			kv("gif_search_branding", jsonString("powered by GIPHY")),
			kv("gif_search_emojies", jsonStringArray(
				"👍", "😘", "😍", "😡", "🥳",
				"😂", "😢", "😮", "😎", "👏",
			)),

			// Dice emojis
			kv("emojies_send_dice", jsonStringArray("🎲", "🎯", "🏀", "⚽", "🎰", "🎳")),
			kv("emojies_send_dice_success", jsonArray(
			// Frame numbers for showing "win" animation (not critical, can be empty)
			)),
			kv("emojies_sounds", mtproto.MakeTLJsonObject(&mtproto.JSONValue{
				Value_VECTORJSONOBJECTVALUE: []*mtproto.JSONObjectValue{},
			}).To_JSONValue()),

			// Autologin / URL auth
			kv("autologin_token", jsonString("")),
			kv("autologin_domains", jsonStringArray()),
			kv("url_auth_domains", jsonStringArray()),

			// Whitelisted domain suffixes for instant view
			kv("whitelisted_domain_suffixes", jsonStringArray("t.me")),

			// Round video limits
			kv("round_video_size", jsonNumber(384)),
			kv("round_video_encoding_bitrate", jsonNumber(1000)),

			// Forum
			kv("topics_pinned_limit", jsonNumber(5)),

			// Transcription
			kv("transcribe_audio_trial_weekly_number", jsonNumber(2)),
			kv("transcribe_audio_trial_duration_max", jsonNumber(300)),

			// Story limits
			kv("story_expiring_limit_default", jsonNumber(3)),
			kv("story_expiring_limit_premium", jsonNumber(30)),

			// Ringtone
			kv("ringtone_duration_max", jsonNumber(5)),
			kv("ringtone_size_max", jsonNumber(307200)), // 300KB

			// Recommended channels
			kv("recommended_channels_limit_default", jsonNumber(10)),
			kv("recommended_channels_limit_premium", jsonNumber(100)),

			// Channel color levels
			kv("channel_color_level_min", jsonNumber(5)),
			kv("channel_wallpaper_level_min", jsonNumber(9)),
			kv("channel_emoji_status_level_min", jsonNumber(8)),
			kv("channel_profile_color_level_min", jsonNumber(5)),
			kv("channel_bg_icon_level_min", jsonNumber(5)),

			// Group color levels
			kv("group_emoji_stickers_level_min", jsonNumber(4)),

			// Small queue max size
			kv("small_queue_max_active_operations_count", jsonNumber(5)),
			kv("large_queue_max_active_operations_count", jsonNumber(2)),
		},
	}).To_JSONValue()
}
