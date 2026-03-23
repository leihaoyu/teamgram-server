package core

import (
	"github.com/gogo/protobuf/types"
	"github.com/teamgram/proto/mtproto"
)

func stringValue(v string) *types.StringValue {
	return &types.StringValue{Value: v}
}

// makeChatTheme creates a chat theme with light and dark settings
func makeChatTheme(id int64, emoticon string, lightAccent int32, lightMsgColors []int32, darkAccent int32, darkMsgColors []int32) *mtproto.Theme {
	settings := []*mtproto.ThemeSettings{
		// Light (Day) settings
		mtproto.MakeTLThemeSettings(&mtproto.ThemeSettings{
			BaseTheme:     mtproto.MakeTLBaseThemeDay(&mtproto.BaseTheme{}).To_BaseTheme(),
			AccentColor:   lightAccent,
			MessageColors: lightMsgColors,
		}).To_ThemeSettings(),
		// Dark settings
		mtproto.MakeTLThemeSettings(&mtproto.ThemeSettings{
			BaseTheme:     mtproto.MakeTLBaseThemeNight(&mtproto.BaseTheme{}).To_BaseTheme(),
			AccentColor:   darkAccent,
			MessageColors: darkMsgColors,
		}).To_ThemeSettings(),
	}

	return mtproto.MakeTLTheme(&mtproto.Theme{
		Id:         id,
		AccessHash: id * 31,
		Slug:       emoticon,
		Title:      emoticon,
		ForChat:    true,
		Emoticon:   stringValue(emoticon),
		Settings:   settings,
	}).To_Theme()
}

// defaultChatThemes returns predefined emoticon-based chat themes
func defaultChatThemes() []*mtproto.Theme {
	return []*mtproto.Theme{
		makeChatTheme(20001, "❤", 0xf5524a, []int32{0xf57f87, 0xf5d1c3}, 0xf5524a, []int32{0x8b3a3a, 0x5c2626}),
		makeChatTheme(20002, "🏠", 0x7e5836, []int32{0xdbc3a0, 0xc9b289}, 0xc69b6d, []int32{0x4a3527, 0x3a2a1f}),
		makeChatTheme(20003, "🌿", 0x5da352, []int32{0xa5d68c, 0xc6e6a9}, 0x5da352, []int32{0x2e4a2a, 0x1f3a1b}),
		makeChatTheme(20004, "☀️", 0xe29e35, []int32{0xf5d97e, 0xf5e6ab}, 0xe29e35, []int32{0x5c4a1f, 0x4a3a15}),
		makeChatTheme(20005, "🍊", 0xe07e39, []int32{0xf5b87e, 0xf5d4a8}, 0xe07e39, []int32{0x5c3a1f, 0x4a2f15}),
		makeChatTheme(20006, "🌊", 0x3d8ed1, []int32{0x7ec4ea, 0xa8d8f0}, 0x3d8ed1, []int32{0x1f3a5c, 0x152f4a}),
		makeChatTheme(20007, "🌸", 0xd45a8f, []int32{0xf5a0c0, 0xf5c6d6}, 0xd45a8f, []int32{0x5c1f3a, 0x4a152f}),
		makeChatTheme(20008, "💜", 0x7b5ebd, []int32{0xc6b1ef, 0xd5c6f5}, 0x7b5ebd, []int32{0x3a2a5c, 0x2f1f4a}),
		makeChatTheme(20009, "🎮", 0x4a8e3f, []int32{0x85c77c, 0xaad6a2}, 0x4a8e3f, []int32{0x2a4a26, 0x1f3a1b}),
		makeChatTheme(20010, "🎄", 0xc74040, []int32{0xd47676, 0x72a550}, 0xc74040, []int32{0x5c2626, 0x2e4a2a}),
	}
}

// makeAppearanceTheme creates a theme for the appearance settings page (shown as accent color dots)
func makeAppearanceTheme(id int64, slug string, title string, baseTheme *mtproto.BaseTheme, accentColor int32, msgColors []int32) *mtproto.Theme {
	return mtproto.MakeTLTheme(&mtproto.Theme{
		Id:         id,
		AccessHash: id * 37,
		Slug:       slug,
		Title:      title,
		Settings: []*mtproto.ThemeSettings{
			mtproto.MakeTLThemeSettings(&mtproto.ThemeSettings{
				BaseTheme:     baseTheme,
				AccentColor:   accentColor,
				MessageColors: msgColors,
			}).To_ThemeSettings(),
		},
	}).To_Theme()
}

// defaultAppearanceThemes returns themes for the iOS appearance settings page.
// Themes with settings are displayed as accent color swatches.
func defaultAppearanceThemes() []*mtproto.Theme {
	dayBase := mtproto.MakeTLBaseThemeDay(&mtproto.BaseTheme{}).To_BaseTheme()
	classicBase := mtproto.MakeTLBaseThemeClassic(&mtproto.BaseTheme{}).To_BaseTheme()
	nightBase := mtproto.MakeTLBaseThemeNight(&mtproto.BaseTheme{}).To_BaseTheme()
	tintedBase := mtproto.MakeTLBaseThemeTinted(&mtproto.BaseTheme{}).To_BaseTheme()

	return []*mtproto.Theme{
		// Day theme accent colors
		makeAppearanceTheme(30001, "day-blue", "Blue", dayBase, 0x3e88f7, []int32{0x4fae4e, 0x51b5a8}),
		makeAppearanceTheme(30002, "day-red", "Red", dayBase, 0xf83b4c, []int32{0xf83b4c}),
		makeAppearanceTheme(30003, "day-orange", "Orange", dayBase, 0xfa5e16, []int32{0xfa5e16}),
		makeAppearanceTheme(30004, "day-yellow", "Yellow", dayBase, 0xffc402, []int32{0xffc402}),
		makeAppearanceTheme(30005, "day-green", "Green", dayBase, 0x3dbd4d, []int32{0x3dbd4d}),
		makeAppearanceTheme(30006, "day-cyan", "Cyan", dayBase, 0x29b6f6, []int32{0x29b6f6}),
		makeAppearanceTheme(30007, "day-pink", "Pink", dayBase, 0xeb6ca4, []int32{0xeb6ca4}),
		makeAppearanceTheme(30008, "day-purple", "Purple", dayBase, 0x7b68ee, []int32{0x7b68ee}),

		// Classic theme accent colors
		makeAppearanceTheme(30101, "classic-blue", "Classic Blue", classicBase, 0x3e88f7, []int32{0x4fae4e, 0x51b5a8}),
		makeAppearanceTheme(30102, "classic-red", "Classic Red", classicBase, 0xf83b4c, []int32{0xf83b4c}),
		makeAppearanceTheme(30103, "classic-orange", "Classic Orange", classicBase, 0xfa5e16, []int32{0xfa5e16}),
		makeAppearanceTheme(30104, "classic-green", "Classic Green", classicBase, 0x3dbd4d, []int32{0x3dbd4d}),

		// Night theme accent colors
		makeAppearanceTheme(30201, "night-blue", "Night Blue", nightBase, 0x3e88f7, []int32{0x3e88f7}),
		makeAppearanceTheme(30202, "night-red", "Night Red", nightBase, 0xf83b4c, []int32{0xf83b4c}),
		makeAppearanceTheme(30203, "night-green", "Night Green", nightBase, 0x3dbd4d, []int32{0x3dbd4d}),
		makeAppearanceTheme(30204, "night-purple", "Night Purple", nightBase, 0x7b68ee, []int32{0x7b68ee}),

		// Tinted (dark blue) theme accent colors
		makeAppearanceTheme(30301, "tinted-blue", "Tinted Blue", tintedBase, 0x3e88f7, []int32{0x3e88f7}),
		makeAppearanceTheme(30302, "tinted-red", "Tinted Red", tintedBase, 0xf83b4c, []int32{0xf83b4c}),
		makeAppearanceTheme(30303, "tinted-green", "Tinted Green", tintedBase, 0x3dbd4d, []int32{0x3dbd4d}),
		makeAppearanceTheme(30304, "tinted-purple", "Tinted Purple", tintedBase, 0x7b68ee, []int32{0x7b68ee}),
	}
}
