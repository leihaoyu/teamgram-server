package core

import (
	"github.com/gogo/protobuf/types"
	"github.com/teamgram/proto/mtproto"
)

func stringValue(v string) *types.StringValue {
	return &types.StringValue{Value: v}
}

func int32Value(v int32) *types.Int32Value {
	return &types.Int32Value{Value: v}
}

// argb adds 0xFF alpha to a 24-bit RGB color, producing a signed int32 ARGB value.
// The iOS client uses UIColor(argb:) which reads bits 24-31 as alpha,
// so colors without 0xFF prefix appear fully transparent.
func argb(rgb int32) int32 {
	return int32(uint32(0xFF000000) | uint32(rgb))
}

// argbSlice applies argb() to each element
func argbSlice(colors []int32) []int32 {
	out := make([]int32, len(colors))
	for i, c := range colors {
		out[i] = argb(c)
	}
	return out
}

var (
	baseDay     = mtproto.MakeTLBaseThemeDay(&mtproto.BaseTheme{}).To_BaseTheme()
	baseClassic = mtproto.MakeTLBaseThemeClassic(&mtproto.BaseTheme{}).To_BaseTheme()
	baseNight   = mtproto.MakeTLBaseThemeNight(&mtproto.BaseTheme{}).To_BaseTheme()
	baseTinted  = mtproto.MakeTLBaseThemeTinted(&mtproto.BaseTheme{}).To_BaseTheme()
)

// makeGradientWallpaper creates a wallPaperNoFile with gradient colors for use in ThemeSettings
func makeGradientWallpaper(colors []int32) *mtproto.WallPaper {
	c := argbSlice(colors)
	settings := &mtproto.WallPaperSettings{
		BackgroundColor: int32Value(c[0]),
	}
	if len(c) > 1 {
		settings.SecondBackgroundColor = int32Value(c[1])
		settings.Rotation = int32Value(0) // must be set: shares flags.4 with SecondBackgroundColor
	}
	if len(c) > 2 {
		settings.ThirdBackgroundColor = int32Value(c[2])
	}
	if len(c) > 3 {
		settings.FourthBackgroundColor = int32Value(c[3])
	}
	return mtproto.MakeTLWallPaperNoFile(&mtproto.WallPaper{
		Id:       0,
		Default:  true,
		Settings: mtproto.MakeTLWallPaperSettings(settings).To_WallPaperSettings(),
	}).To_WallPaper()
}

func makeSettings(base *mtproto.BaseTheme, accent int32, msgColors []int32, wallpaperColors []int32) *mtproto.ThemeSettings {
	ts := &mtproto.ThemeSettings{
		BaseTheme:     base,
		AccentColor:   argb(accent),
		MessageColors: argbSlice(msgColors),
	}
	if len(wallpaperColors) > 0 {
		ts.Wallpaper = makeGradientWallpaper(wallpaperColors)
	}
	return mtproto.MakeTLThemeSettings(ts).To_ThemeSettings()
}

// chatThemeDef defines colors for a chat theme
type chatThemeDef struct {
	id       int64
	emoticon string
	// Day (light) mode
	dayAccent    int32
	dayMsgColors []int32
	dayWallpaper []int32 // gradient background for light mode
	// Night (dark) mode
	nightAccent    int32
	nightMsgColors []int32
	nightWallpaper []int32 // gradient background for dark mode
}

// makeChatTheme creates a chat theme with proper light and dark settings including wallpapers
func makeChatTheme(def chatThemeDef) *mtproto.Theme {
	settings := []*mtproto.ThemeSettings{
		makeSettings(baseDay, def.dayAccent, def.dayMsgColors, def.dayWallpaper),
		makeSettings(baseNight, def.nightAccent, def.nightMsgColors, def.nightWallpaper),
	}

	return mtproto.MakeTLTheme(&mtproto.Theme{
		Id:         def.id,
		AccessHash: def.id * 31,
		Slug:       def.emoticon,
		Title:      def.emoticon,
		ForChat:    true,
		Emoticon:   stringValue(def.emoticon),
		Settings:   settings,
	}).To_Theme()
}

// defaultChatThemes returns predefined emoticon-based chat themes
// Each theme has Day (light) and Night (dark) settings with proper wallpaper backgrounds
func defaultChatThemes() []*mtproto.Theme {
	return []*mtproto.Theme{
		makeChatTheme(chatThemeDef{
			id: 20001, emoticon: "❤",
			dayAccent: 0xf5524a, dayMsgColors: []int32{0xf57f87, 0xf5d1c3},
			dayWallpaper: []int32{0xfce3df, 0xf9d7d5, 0xf9eae2, 0xfce8d4},
			nightAccent:  0xf5524a, nightMsgColors: []int32{0x8b3a3a, 0x5c2626},
			nightWallpaper: []int32{0x3b1518, 0x3d1a1a, 0x2e1114, 0x2c1315},
		}),
		makeChatTheme(chatThemeDef{
			id: 20002, emoticon: "🏠",
			dayAccent: 0x7e5836, dayMsgColors: []int32{0xdbc3a0, 0xc9b289},
			dayWallpaper: []int32{0xf0e6d8, 0xede3d1, 0xece0cc, 0xf2e7d6},
			nightAccent:  0xc69b6d, nightMsgColors: []int32{0x4a3527, 0x3a2a1f},
			nightWallpaper: []int32{0x2d2319, 0x2a201a, 0x26201b, 0x2a2118},
		}),
		makeChatTheme(chatThemeDef{
			id: 20003, emoticon: "🌿",
			dayAccent: 0x5da352, dayMsgColors: []int32{0xa5d68c, 0xc6e6a9},
			dayWallpaper: []int32{0xe0f0d8, 0xd8ebd0, 0xdaf0d0, 0xe2f0dc},
			nightAccent:  0x5da352, nightMsgColors: []int32{0x2e4a2a, 0x1f3a1b},
			nightWallpaper: []int32{0x14261a, 0x16281c, 0x122418, 0x14261a},
		}),
		makeChatTheme(chatThemeDef{
			id: 20004, emoticon: "☀️",
			dayAccent: 0xc98315, dayMsgColors: []int32{0xf5d97e, 0xf5e6ab},
			dayWallpaper: []int32{0xfaf2d8, 0xf8efcc, 0xfaf0d0, 0xfcf4dc},
			nightAccent:  0xe29e35, nightMsgColors: []int32{0x5c4a1f, 0x4a3a15},
			nightWallpaper: []int32{0x2c2410, 0x2a2112, 0x28200e, 0x2a2210},
		}),
		makeChatTheme(chatThemeDef{
			id: 20005, emoticon: "🍊",
			dayAccent: 0xd67722, dayMsgColors: []int32{0xf5b87e, 0xf5d4a8},
			dayWallpaper: []int32{0xfaeadc, 0xf8e4d2, 0xfae6d4, 0xfcecde},
			nightAccent:  0xe07e39, nightMsgColors: []int32{0x5c3a1f, 0x4a2f15},
			nightWallpaper: []int32{0x2c1a0e, 0x2a1810, 0x28160c, 0x2a1a0e},
		}),
		makeChatTheme(chatThemeDef{
			id: 20006, emoticon: "🌊",
			dayAccent: 0x3d8ed1, dayMsgColors: []int32{0x7ec4ea, 0xa8d8f0},
			dayWallpaper: []int32{0xdcecf8, 0xd4e8f4, 0xd8ecf6, 0xe0eefa},
			nightAccent:  0x3d8ed1, nightMsgColors: []int32{0x1f3a5c, 0x152f4a},
			nightWallpaper: []int32{0x0e1a2c, 0x10182a, 0x0c1628, 0x0e1a2a},
		}),
		makeChatTheme(chatThemeDef{
			id: 20007, emoticon: "🌸",
			dayAccent: 0xd45a8f, dayMsgColors: []int32{0xf5a0c0, 0xf5c6d6},
			dayWallpaper: []int32{0xfae0ec, 0xf8dce8, 0xfae0ea, 0xfce4ee},
			nightAccent:  0xd45a8f, nightMsgColors: []int32{0x5c1f3a, 0x4a152f},
			nightWallpaper: []int32{0x2c0e1a, 0x2a1018, 0x280c16, 0x2a0e1a},
		}),
		makeChatTheme(chatThemeDef{
			id: 20008, emoticon: "💜",
			dayAccent: 0x7b5ebd, dayMsgColors: []int32{0xc6b1ef, 0xd5c6f5},
			dayWallpaper: []int32{0xe8e0f6, 0xe4dcf2, 0xe6def4, 0xeae2f8},
			nightAccent:  0x7b5ebd, nightMsgColors: []int32{0x3a2a5c, 0x2f1f4a},
			nightWallpaper: []int32{0x1a0e2c, 0x18102a, 0x160c28, 0x1a0e2a},
		}),
		makeChatTheme(chatThemeDef{
			id: 20009, emoticon: "🎮",
			dayAccent: 0x4a8e3f, dayMsgColors: []int32{0x85c77c, 0xaad6a2},
			dayWallpaper: []int32{0xdcf0d8, 0xd6ecd0, 0xd8eed2, 0xdef2da},
			nightAccent:  0x4a8e3f, nightMsgColors: []int32{0x2a4a26, 0x1f3a1b},
			nightWallpaper: []int32{0x12260e, 0x142810, 0x10240c, 0x12260e},
		}),
		makeChatTheme(chatThemeDef{
			id: 20010, emoticon: "🎄",
			dayAccent: 0xc74040, dayMsgColors: []int32{0xd47676, 0x72a550},
			dayWallpaper: []int32{0xf0dcd8, 0xe8e8d4, 0xecdcd6, 0xf0e4d8},
			nightAccent:  0xc74040, nightMsgColors: []int32{0x5c2626, 0x2e4a2a},
			nightWallpaper: []int32{0x2c1010, 0x1a2a14, 0x281212, 0x1e2c16},
		}),
	}
}

// accentColorDef defines accent colors for all 4 base themes
type accentColorDef struct {
	dayAccent     int32
	dayMsg        []int32
	classicAccent int32
	classicMsg    []int32
	nightAccent   int32
	nightMsg      []int32
	tintedAccent  int32
	tintedMsg     []int32
}

// makeAppearanceTheme creates a theme with settings for ALL 4 base themes
// so it shows as an accent color dot regardless of which base theme the user selects
func makeAppearanceTheme(id int64, slug string, title string, def accentColorDef) *mtproto.Theme {
	return mtproto.MakeTLTheme(&mtproto.Theme{
		Id:         id,
		AccessHash: id * 37,
		Slug:       slug,
		Title:      title,
		Default:    true,
		Settings: []*mtproto.ThemeSettings{
			makeSettings(baseClassic, def.classicAccent, def.classicMsg, nil),
			makeSettings(baseDay, def.dayAccent, def.dayMsg, nil),
			makeSettings(baseNight, def.nightAccent, def.nightMsg, nil),
			makeSettings(baseTinted, def.tintedAccent, def.tintedMsg, nil),
		},
	}).To_Theme()
}

// defaultAppearanceThemes returns themes for the iOS appearance settings page.
// Each theme has settings for all 4 base themes so the accent color dot
// shows up regardless of which base theme the user is currently on.
func defaultAppearanceThemes() []*mtproto.Theme {
	return []*mtproto.Theme{
		makeAppearanceTheme(30001, "blue", "Blue", accentColorDef{
			dayAccent: 0x3e88f7, dayMsg: []int32{0x4fae4e, 0x51b5a8},
			classicAccent: 0x3e88f7, classicMsg: []int32{0x4fae4e, 0x51b5a8},
			nightAccent: 0x3e88f7, nightMsg: []int32{0x3e88f7},
			tintedAccent: 0x3e88f7, tintedMsg: []int32{0x3e88f7},
		}),
		makeAppearanceTheme(30002, "red", "Red", accentColorDef{
			dayAccent: 0xf83b4c, dayMsg: []int32{0xf83b4c},
			classicAccent: 0xf83b4c, classicMsg: []int32{0xf83b4c},
			nightAccent: 0xf83b4c, nightMsg: []int32{0xf83b4c},
			tintedAccent: 0xf83b4c, tintedMsg: []int32{0xf83b4c},
		}),
		makeAppearanceTheme(30003, "orange", "Orange", accentColorDef{
			dayAccent: 0xfa5e16, dayMsg: []int32{0xfa5e16},
			classicAccent: 0xfa5e16, classicMsg: []int32{0xfa5e16},
			nightAccent: 0xfa5e16, nightMsg: []int32{0xfa5e16},
			tintedAccent: 0xfa5e16, tintedMsg: []int32{0xfa5e16},
		}),
		makeAppearanceTheme(30004, "yellow", "Yellow", accentColorDef{
			// Day/Classic: darker amber-gold for visibility on white background (~4.7:1 contrast)
			dayAccent: 0xa06800, dayMsg: []int32{0xa06800},
			classicAccent: 0xa06800, classicMsg: []int32{0xa06800},
			// Night/Tinted: bright yellow works well on dark backgrounds
			nightAccent: 0xffc402, nightMsg: []int32{0xffc402},
			tintedAccent: 0xffc402, tintedMsg: []int32{0xffc402},
		}),
		makeAppearanceTheme(30005, "green", "Green", accentColorDef{
			dayAccent: 0x3dbd4d, dayMsg: []int32{0x3dbd4d},
			classicAccent: 0x3dbd4d, classicMsg: []int32{0x3dbd4d},
			nightAccent: 0x3dbd4d, nightMsg: []int32{0x3dbd4d},
			tintedAccent: 0x3dbd4d, tintedMsg: []int32{0x3dbd4d},
		}),
		makeAppearanceTheme(30006, "cyan", "Cyan", accentColorDef{
			// Day/Classic: darker cyan-blue for visibility on white background (~4.8:1 contrast)
			dayAccent: 0x007aad, dayMsg: []int32{0x007aad},
			classicAccent: 0x007aad, classicMsg: []int32{0x007aad},
			// Night/Tinted: bright cyan works well on dark backgrounds
			nightAccent: 0x29b6f6, nightMsg: []int32{0x29b6f6},
			tintedAccent: 0x29b6f6, tintedMsg: []int32{0x29b6f6},
		}),
		makeAppearanceTheme(30007, "pink", "Pink", accentColorDef{
			dayAccent: 0xeb6ca4, dayMsg: []int32{0xeb6ca4},
			classicAccent: 0xeb6ca4, classicMsg: []int32{0xeb6ca4},
			nightAccent: 0xeb6ca4, nightMsg: []int32{0xeb6ca4},
			tintedAccent: 0xeb6ca4, tintedMsg: []int32{0xeb6ca4},
		}),
		makeAppearanceTheme(30008, "purple", "Purple", accentColorDef{
			dayAccent: 0x7b68ee, dayMsg: []int32{0x7b68ee},
			classicAccent: 0x7b68ee, classicMsg: []int32{0x7b68ee},
			nightAccent: 0x7b68ee, nightMsg: []int32{0x7b68ee},
			tintedAccent: 0x7b68ee, tintedMsg: []int32{0x7b68ee},
		}),
	}
}
