package app

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

const maxAppearanceSkinThemes = 16

var (
	appearanceSkinIDPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,62}[a-z0-9])?$`)
	appearanceColorPattern  = regexp.MustCompile(`^#[0-9a-fA-F]{6}(?:[0-9a-fA-F]{2})?$`)
)

// AppearanceSkinModeTokens is deliberately an allowlist instead of arbitrary CSS.
// Every value is validated as a six- or eight-digit hex color before it is exposed
// by the public appearance endpoint.
type AppearanceSkinModeTokens struct {
	Canvas                    string `json:"canvas"`
	Surface                   string `json:"surface"`
	SurfaceSubtle             string `json:"surfaceSubtle"`
	SurfaceRaised             string `json:"surfaceRaised"`
	Overlay                   string `json:"overlay"`
	Text                      string `json:"text"`
	TextMuted                 string `json:"textMuted"`
	Border                    string `json:"border"`
	Control                   string `json:"control"`
	ControlHover              string `json:"controlHover"`
	ControlActive             string `json:"controlActive"`
	ControlBorder             string `json:"controlBorder"`
	ControlFocus              string `json:"controlFocus"`
	ControlDisabledBackground string `json:"controlDisabledBackground"`
	ControlDisabledForeground string `json:"controlDisabledForeground"`
	SwitchChecked             string `json:"switchChecked"`
	SwitchCheckedHover        string `json:"switchCheckedHover"`
	SwitchCheckedHandle       string `json:"switchCheckedHandle"`
	SwitchUnchecked           string `json:"switchUnchecked"`
	SwitchUncheckedHover      string `json:"switchUncheckedHover"`
	SwitchUncheckedHandle     string `json:"switchUncheckedHandle"`
	Primary                   string `json:"primary"`
	PrimaryHover              string `json:"primaryHover"`
	PrimaryActive             string `json:"primaryActive"`
	PrimaryForeground         string `json:"primaryForeground"`
	Selected                  string `json:"selected"`
	SelectedHover             string `json:"selectedHover"`
	SelectedActive            string `json:"selectedActive"`
	SelectedForeground        string `json:"selectedForeground"`
	Icon                      string `json:"icon"`
	IconMuted                 string `json:"iconMuted"`
	IconActive                string `json:"iconActive"`
	Success                   string `json:"success"`
	Warning                   string `json:"warning"`
	Danger                    string `json:"danger"`
	DangerHover               string `json:"dangerHover"`
	DangerActive              string `json:"dangerActive"`
	DangerForeground          string `json:"dangerForeground"`
	Info                      string `json:"info"`
	Workspace                 string `json:"workspace"`
	WorkspaceGrid             string `json:"workspaceGrid"`
	AdminBackground           string `json:"adminBackground"`
	AdminSurface              string `json:"adminSurface"`
	AdminSubtle               string `json:"adminSubtle"`
	AdminStrong               string `json:"adminStrong"`
	AuthBackground            string `json:"authBackground"`
	AuthPanel                 string `json:"authPanel"`
	AuthCard                  string `json:"authCard"`
	AuthAccent                string `json:"authAccent"`
	AuthMuted                 string `json:"authMuted"`
}

type AppearanceSkinComponentTokens struct {
	ButtonRadius       int    `json:"buttonRadius"`
	InputRadius        int    `json:"inputRadius"`
	CardRadius         int    `json:"cardRadius"`
	OverlayRadius      int    `json:"overlayRadius"`
	MenuRadius         int    `json:"menuRadius"`
	CheckboxRadius     int    `json:"checkboxRadius"`
	ControlHeight      int    `json:"controlHeight"`
	ControlHeightSmall int    `json:"controlHeightSmall"`
	ControlHeightLarge int    `json:"controlHeightLarge"`
	BorderWidth        int    `json:"borderWidth"`
	FocusRingWidth     int    `json:"focusRingWidth"`
	IconSize           int    `json:"iconSize"`
	ButtonFontWeight   int    `json:"buttonFontWeight"`
	HoverLift          int    `json:"hoverLift"`
	MotionFast         int    `json:"motionFast"`
	MotionNormal       int    `json:"motionNormal"`
	ShadowStyle        string `json:"shadowStyle"`
}

type AppearanceSkinTokens struct {
	Light      AppearanceSkinModeTokens      `json:"light"`
	Dark       AppearanceSkinModeTokens      `json:"dark"`
	Components AppearanceSkinComponentTokens `json:"components"`
}

type AppearanceSkinTheme struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Locked      bool                 `json:"locked"`
	Tokens      AppearanceSkinTokens `json:"tokens"`
	Film        string               `json:"film,omitempty"`
	Poster      string               `json:"poster,omitempty"`
	FX          string               `json:"fx,omitempty"`
	Hue         float64              `json:"hue,omitempty"`
}

func defaultAppearanceSkinThemes() []AppearanceSkinTheme {
	return cinematicAppearanceSkins(defaultClassicAppearanceSkin())
}

func defaultClassicAppearanceSkin() AppearanceSkinTheme {
	return AppearanceSkinTheme{
		ID: "classic", Name: "经典黑白", Description: "项目原始样式 · 不可修改", Locked: true,
		Tokens: AppearanceSkinTokens{
			Light: AppearanceSkinModeTokens{
				Canvas: "#ffffff", Surface: "#ffffff", SurfaceSubtle: "#f7f7f7", SurfaceRaised: "#ececec", Overlay: "#ffffff", Text: "#171717", TextMuted: "#737373", Border: "#e5e5e5",
				Control: "#ffffff", ControlHover: "#f5f5f5", ControlActive: "#ececec", ControlBorder: "#d1d1d1", ControlFocus: "#171717", ControlDisabledBackground: "#f2f2f2", ControlDisabledForeground: "#a3a3a3", SwitchChecked: "#16a34a", SwitchCheckedHover: "#15803d", SwitchCheckedHandle: "#ffffff", SwitchUnchecked: "#b8b8b8", SwitchUncheckedHover: "#9f9f9f", SwitchUncheckedHandle: "#ffffff",
				Primary: "#171717", PrimaryHover: "#303030", PrimaryActive: "#404040", PrimaryForeground: "#ffffff", Selected: "#e8e8e8", SelectedHover: "#dedede", SelectedActive: "#d5d5d5", SelectedForeground: "#171717",
				Icon: "#3f3f46", IconMuted: "#a1a1aa", IconActive: "#171717", Success: "#16a34a", Warning: "#d97706", Danger: "#dc2626", DangerHover: "#b91c1c", DangerActive: "#991b1b", DangerForeground: "#ffffff", Info: "#2563eb", Workspace: "#ffffff", WorkspaceGrid: "#f3f3f3",
				AdminBackground: "#f3f4f6", AdminSurface: "#ffffff", AdminSubtle: "#f7f8fa", AdminStrong: "#eceff3", AuthBackground: "#08090c", AuthPanel: "#0b0c10", AuthCard: "#121318", AuthAccent: "#93c5fd", AuthMuted: "#8a8b91",
			},
			Dark: AppearanceSkinModeTokens{
				Canvas: "#0a0a0a", Surface: "#181818", SurfaceSubtle: "#202020", SurfaceRaised: "#2a2a2a", Overlay: "#1f1f20", Text: "#f5f5f5", TextMuted: "#a3a3a3", Border: "#2d2d2d",
				Control: "#202020", ControlHover: "#292929", ControlActive: "#333333", ControlBorder: "#4a4a4a", ControlFocus: "#f5f5f5", ControlDisabledBackground: "#252525", ControlDisabledForeground: "#737373", SwitchChecked: "#22c55e", SwitchCheckedHover: "#4ade80", SwitchCheckedHandle: "#071a0f", SwitchUnchecked: "#525252", SwitchUncheckedHover: "#686868", SwitchUncheckedHandle: "#f5f5f5",
				Primary: "#f5f5f5", PrimaryHover: "#ffffff", PrimaryActive: "#e5e5e5", PrimaryForeground: "#171717", Selected: "#2b2b2b", SelectedHover: "#343434", SelectedActive: "#3d3d3d", SelectedForeground: "#f5f5f5",
				Icon: "#d4d4d8", IconMuted: "#71717a", IconActive: "#ffffff", Success: "#4ade80", Warning: "#fbbf24", Danger: "#f87171", DangerHover: "#fca5a5", DangerActive: "#ef4444", DangerForeground: "#2b0808", Info: "#60a5fa", Workspace: "#181818", WorkspaceGrid: "#222222",
				AdminBackground: "#101010", AdminSurface: "#181818", AdminSubtle: "#202020", AdminStrong: "#2a2a2a", AuthBackground: "#08090c", AuthPanel: "#0b0c10", AuthCard: "#121318", AuthAccent: "#93c5fd", AuthMuted: "#8a8b91",
			},
			Components: appearanceSkinComponentPreset(6, 6, 12, 12, 8, 4),
		},
	}
}

type appearanceSkinPalette struct {
	canvas, surface, subtle, raised, overlay, text, muted, border                                                        string
	primary, primaryHover, primaryActive, primaryForeground, selected, selectedHover, selectedActive, selectedForeground string
	switchChecked, switchCheckedHover, switchCheckedHandle, switchUnchecked, switchUncheckedHover, switchUncheckedHandle string
	success, warning, danger, dangerHover, dangerActive, dangerForeground, info                                          string
	workspace, grid, adminBackground, adminSurface, adminSubtle, adminStrong                                             string
	authBackground, authPanel, authCard, authAccent, authMuted                                                           string
}

func tintAppearanceSkinMode(value AppearanceSkinModeTokens, palette appearanceSkinPalette) AppearanceSkinModeTokens {
	value.Canvas, value.Surface, value.SurfaceSubtle, value.SurfaceRaised, value.Overlay = palette.canvas, palette.surface, palette.subtle, palette.raised, palette.overlay
	value.Text, value.TextMuted, value.Border = palette.text, palette.muted, palette.border
	value.Control, value.ControlHover, value.ControlActive, value.ControlBorder, value.ControlFocus = palette.surface, palette.subtle, palette.raised, palette.border, palette.primary
	value.ControlDisabledBackground, value.ControlDisabledForeground = palette.subtle, palette.muted
	value.SwitchChecked, value.SwitchCheckedHover, value.SwitchCheckedHandle = palette.switchChecked, palette.switchCheckedHover, palette.switchCheckedHandle
	value.SwitchUnchecked, value.SwitchUncheckedHover, value.SwitchUncheckedHandle = palette.switchUnchecked, palette.switchUncheckedHover, palette.switchUncheckedHandle
	value.Primary, value.PrimaryHover, value.PrimaryActive, value.PrimaryForeground = palette.primary, palette.primaryHover, palette.primaryActive, palette.primaryForeground
	value.Selected, value.SelectedHover, value.SelectedActive, value.SelectedForeground = palette.selected, palette.selectedHover, palette.selectedActive, palette.selectedForeground
	value.Icon, value.IconMuted, value.IconActive = palette.text, palette.muted, palette.primary
	value.Success, value.Warning, value.Danger = palette.success, palette.warning, palette.danger
	value.DangerHover, value.DangerActive, value.DangerForeground = palette.dangerHover, palette.dangerActive, palette.dangerForeground
	value.Info, value.Workspace, value.WorkspaceGrid = palette.info, palette.workspace, palette.grid
	value.AdminBackground, value.AdminSurface, value.AdminSubtle, value.AdminStrong = palette.adminBackground, palette.adminSurface, palette.adminSubtle, palette.adminStrong
	value.AuthBackground, value.AuthPanel, value.AuthCard, value.AuthAccent, value.AuthMuted = palette.authBackground, palette.authPanel, palette.authCard, palette.authAccent, palette.authMuted
	return value
}

func appearanceSkinComponentPreset(buttonRadius, inputRadius, cardRadius, overlayRadius, menuRadius, checkboxRadius int) AppearanceSkinComponentTokens {
	return AppearanceSkinComponentTokens{
		ButtonRadius: buttonRadius, InputRadius: inputRadius, CardRadius: cardRadius, OverlayRadius: overlayRadius, MenuRadius: menuRadius, CheckboxRadius: checkboxRadius,
		ControlHeight: 36, ControlHeightSmall: 30, ControlHeightLarge: 42, BorderWidth: 1, FocusRingWidth: 2, IconSize: 16, ButtonFontWeight: 500,
		HoverLift: 1, MotionFast: 120, MotionNormal: 180, ShadowStyle: "soft",
	}
}

func cloneAppearanceSkin(source AppearanceSkinTheme, id, name, description string) AppearanceSkinTheme {
	source.ID, source.Name, source.Description, source.Locked = id, name, description, false
	return source
}

func normalizeAppearanceSkinThemes(themes []AppearanceSkinTheme) []AppearanceSkinTheme {
	result := make([]AppearanceSkinTheme, len(themes))
	copy(result, themes)
	result = pruneRetiredAppearanceSkins(result)
	result = ensureCinematicAppearanceSkins(result)
	builtins := defaultAppearanceSkinThemes()
	for index := range result {
		result[index].ID = strings.ToLower(strings.TrimSpace(result[index].ID))
		result[index].Name = strings.TrimSpace(result[index].Name)
		result[index].Description = strings.TrimSpace(result[index].Description)
		result[index].Locked = isOfficialCinematicSkinID(result[index].ID)
		var fallback AppearanceSkinTokens
		for _, builtin := range builtins {
			if builtin.ID == result[index].ID {
				fallback = builtin.Tokens
				result[index].Name = builtin.Name
				result[index].Description = builtin.Description
				result[index].Film = builtin.Film
				result[index].Poster = builtin.Poster
				result[index].FX = builtin.FX
				result[index].Hue = builtin.Hue
				result[index].Tokens = builtin.Tokens
				break
			}
		}
		backfillAppearanceSkinModeColors(&result[index].Tokens.Light, fallback.Light)
		backfillAppearanceSkinModeColors(&result[index].Tokens.Dark, fallback.Dark)
		normalizeAppearanceSkinModeColors(&result[index].Tokens.Light)
		normalizeAppearanceSkinModeColors(&result[index].Tokens.Dark)
		result[index].Tokens.Components.ShadowStyle = strings.ToLower(strings.TrimSpace(result[index].Tokens.Components.ShadowStyle))
	}
	return result
}

func backfillAppearanceSkinModeColors(mode *AppearanceSkinModeTokens, fallback AppearanceSkinModeTokens) {
	for _, field := range []struct {
		value    *string
		fallback string
		derived  string
	}{
		{&mode.SwitchChecked, fallback.SwitchChecked, mode.Primary},
		{&mode.SwitchCheckedHover, fallback.SwitchCheckedHover, mode.PrimaryHover},
		{&mode.SwitchCheckedHandle, fallback.SwitchCheckedHandle, mode.PrimaryForeground},
		{&mode.SwitchUnchecked, fallback.SwitchUnchecked, mode.ControlBorder},
		{&mode.SwitchUncheckedHover, fallback.SwitchUncheckedHover, mode.ControlActive},
		{&mode.SwitchUncheckedHandle, fallback.SwitchUncheckedHandle, mode.SelectedForeground},
		{&mode.DangerHover, fallback.DangerHover, mode.Danger},
		{&mode.DangerActive, fallback.DangerActive, mode.Danger},
		{&mode.DangerForeground, fallback.DangerForeground, mode.PrimaryForeground},
	} {
		if strings.TrimSpace(*field.value) != "" {
			continue
		}
		if field.fallback != "" {
			*field.value = field.fallback
		} else {
			*field.value = field.derived
		}
	}
}

func normalizeAppearanceSkinModeColors(mode *AppearanceSkinModeTokens) {
	value := reflect.ValueOf(mode).Elem()
	for index := 0; index < value.NumField(); index++ {
		field := value.Field(index)
		field.SetString(strings.ToLower(strings.TrimSpace(field.String())))
	}
}

func validateAppearanceSkinThemes(themes []AppearanceSkinTheme, selectedID string) error {
	if len(themes) == 0 || len(themes) > maxAppearanceSkinThemes {
		return BadAuthRequest(fmt.Sprintf("皮肤主题数量必须为 1 到 %d 套", maxAppearanceSkinThemes))
	}
	seen := make(map[string]struct{}, len(themes))
	foundSelected := false
	for _, skin := range themes {
		if !appearanceSkinIDPattern.MatchString(skin.ID) {
			return BadAuthRequest("皮肤主题 ID 无效")
		}
		if _, exists := seen[skin.ID]; exists {
			return BadAuthRequest("皮肤主题 ID 不能重复")
		}
		seen[skin.ID] = struct{}{}
		if skin.ID == selectedID {
			foundSelected = true
		}
		if err := validateAppearanceSkinText(skin.Name, "皮肤主题名称", 40, true); err != nil {
			return err
		}
		if err := validateAppearanceSkinText(skin.Description, "皮肤主题说明", 100, false); err != nil {
			return err
		}
		if err := validateAppearanceSkinMode(skin.Tokens.Light); err != nil {
			return err
		}
		if err := validateAppearanceSkinMode(skin.Tokens.Dark); err != nil {
			return err
		}
		if err := validateAppearanceSkinComponents(skin.Tokens.Components); err != nil {
			return err
		}
		if err := validateAppearanceSkinMedia(skin.Film, "film"); err != nil {
			return err
		}
		if err := validateAppearanceSkinMedia(skin.Poster, "poster"); err != nil {
			return err
		}
	}
	if !foundSelected {
		return BadAuthRequest("当前启用的皮肤主题不存在")
	}
	return nil
}

func validateAppearanceSkinMode(mode AppearanceSkinModeTokens) error {
	value := reflect.ValueOf(mode)
	for index := 0; index < value.NumField(); index++ {
		if !appearanceColorPattern.MatchString(value.Field(index).String()) {
			return BadAuthRequest("皮肤颜色必须使用 6 或 8 位十六进制颜色")
		}
	}
	return nil
}

func validateAppearanceSkinComponents(value AppearanceSkinComponentTokens) error {
	for _, candidate := range []struct {
		value, min, max int
		label           string
	}{
		{value.ButtonRadius, 0, 32, "按钮圆角"}, {value.InputRadius, 0, 32, "输入框圆角"}, {value.CardRadius, 0, 40, "卡片圆角"}, {value.OverlayRadius, 0, 40, "弹层圆角"}, {value.MenuRadius, 0, 32, "菜单圆角"}, {value.CheckboxRadius, 0, 12, "勾选框圆角"},
		{value.ControlHeight, 30, 48, "控件高度"}, {value.ControlHeightSmall, 24, 40, "小控件高度"}, {value.ControlHeightLarge, 36, 56, "大控件高度"}, {value.BorderWidth, 1, 3, "描边宽度"}, {value.FocusRingWidth, 1, 4, "焦点环宽度"},
		{value.IconSize, 12, 24, "图标尺寸"}, {value.ButtonFontWeight, 400, 700, "按钮字重"}, {value.HoverLift, 0, 4, "悬停抬升"}, {value.MotionFast, 0, 400, "快速动效时长"}, {value.MotionNormal, 0, 800, "常规动效时长"},
	} {
		if candidate.value < candidate.min || candidate.value > candidate.max {
			return BadAuthRequest(fmt.Sprintf("%s必须在 %d 到 %d 之间", candidate.label, candidate.min, candidate.max))
		}
	}
	if value.ControlHeightSmall > value.ControlHeight || value.ControlHeight > value.ControlHeightLarge {
		return BadAuthRequest("控件高度须满足小号不大于标准、标准不大于大号")
	}
	if value.MotionFast > value.MotionNormal {
		return BadAuthRequest("快速动效时长不能大于常规动效时长")
	}
	if value.ShadowStyle != "none" && value.ShadowStyle != "soft" && value.ShadowStyle != "strong" {
		return BadAuthRequest("阴影风格无效")
	}
	return nil
}

func validateAppearanceSkinMedia(value, kind string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if strings.Contains(value, "..") || strings.ContainsAny(value, " \t\r\n") || !strings.HasPrefix(value, "/bg/skin-") {
		return BadAuthRequest("皮肤影像路径无效")
	}
	if kind == "film" && !strings.HasSuffix(strings.ToLower(value), ".mp4") {
		return BadAuthRequest("皮肤循环影像必须是 mp4")
	}
	if kind == "poster" {
		lower := strings.ToLower(value)
		if !strings.HasSuffix(lower, ".jpg") && !strings.HasSuffix(lower, ".jpeg") && !strings.HasSuffix(lower, ".png") && !strings.HasSuffix(lower, ".webp") {
			return BadAuthRequest("皮肤封面必须是图片")
		}
	}
	return nil
}

func validateAppearanceSkinText(value, label string, maxRunes int, required bool) error {
	if required && value == "" {
		return BadAuthRequest(label + "不能为空")
	}
	if utf8.RuneCountInString(value) > maxRunes {
		return BadAuthRequest(fmt.Sprintf("%s不能超过 %d 个字符", label, maxRunes))
	}
	for _, char := range value {
		if unicode.IsControl(char) {
			return BadAuthRequest(label + "不能包含控制字符")
		}
	}
	return nil
}

func activeAppearanceSkin(themes []AppearanceSkinTheme, id string) AppearanceSkinTheme {
	for _, skin := range themes {
		if skin.ID == id {
			return skin
		}
	}
	for _, skin := range themes {
		if skin.ID == defaultAppearanceSkinID {
			return skin
		}
	}
	official := defaultAppearanceSkinThemes()
	if len(official) > 0 {
		return official[0]
	}
	return defaultClassicAppearanceSkin()
}

func cinematicAppearanceSkins(classic AppearanceSkinTheme) []AppearanceSkinTheme {
	type spec struct {
		id, name, description, fx string
		hue                       float64
		light, dark               appearanceSkinPalette
	}
	specs := []spec{
		{id: "apex", name: "奇点", description: "熔金奇点 · 深空加载", fx: "rings", hue: 38,
			dark: cinematicPalette("#070709", "#1a1a20", "#f4f1ea", "#8a8680", "#d9e6f2", "#0b0c10"),
			light: cinematicPalette("#f3efe6", "#fffdf8", "#1a1814", "#6f6a64", "#2c333c", "#f4f1ea")},
		{id: "nexus", name: "冰核", description: "冰晶反应堆 · 冷青加载", fx: "radar", hue: 192,
			dark: cinematicPalette("#031018", "#0c2230", "#d8f6ff", "#6a93a4", "#5ee7ff", "#032028"),
			light: cinematicPalette("#dff4fa", "#ffffff", "#073040", "#4a7380", "#0490a8", "#ffffff")},
		{id: "lumen", name: "霓虹", description: "雨夜霓虹 · 电光加载", fx: "rain", hue: 312,
			dark: cinematicPalette("#08080c", "#1b1b24", "#f5f5f7", "#8b8b99", "#4d7dff", "#ffffff"),
			light: cinematicPalette("#f6f7f9", "#ffffff", "#171717", "#6b7280", "#2563eb", "#ffffff")},
		{id: "prism", name: "棱镜", description: "分光晶体 · 虹彩加载", fx: "shards", hue: 268,
			dark: cinematicPalette("#0e0a18", "#221a34", "#f0eaff", "#9a90b8", "#c4b5ff", "#1a1230"),
			light: cinematicPalette("#f5f2fb", "#ffffff", "#211b35", "#716a86", "#6656d9", "#ffffff")},
		{id: "radix", name: "全息", description: "全息网格 · 扫描加载", fx: "scan", hue: 168,
			dark: cinematicPalette("#051410", "#123028", "#d8fff0", "#6fa392", "#3ee0b0", "#042018"),
			light: cinematicPalette("#eaf6f2", "#ffffff", "#142026", "#4e6c62", "#087f76", "#ffffff")},
		{id: "helix", name: "螺旋", description: "光之螺旋 · 双色加载", fx: "helix", hue: 262,
			dark: cinematicPalette("#070c18", "#162244", "#dce8ff", "#7d8eaa", "#6ea8ff", "#071028"),
			light: cinematicPalette("#e8eef8", "#ffffff", "#12182a", "#5a6780", "#1d4ed8", "#ffffff")},
		{id: "ink", name: "墨核", description: "墨潮核心 · 高对比加载", fx: "ink", hue: 220,
			dark: cinematicPalette("#12100e", "#26201a", "#f3eadc", "#8a8278", "#e8dcc8", "#1a1612"),
			light: cinematicPalette("#f4ecde", "#fffaf0", "#1c1814", "#6e655c", "#1c1814", "#fffaf0")},
		{id: "ember", name: "熔核", description: "熔炉核心 · 余烬加载", fx: "sparks", hue: 22,
			dark: cinematicPalette("#140c08", "#301c12", "#f8e6d2", "#a08870", "#e08a4a", "#2a1008"),
			light: cinematicPalette("#fbf4eb", "#fffdf9", "#35261f", "#806b61", "#b94f2f", "#fffaf6")},
		{id: "veil", name: "极光", description: "极光帷幕 · 绿紫加载", fx: "aurora", hue: 148,
			dark: cinematicPalette("#121214", "#222228", "#ececec", "#8a8d94", "#c8ccd4", "#121214"),
			light: cinematicPalette("#ececef", "#fbfbfc", "#1c1c20", "#6a6d74", "#3a3c42", "#ffffff")},
		{id: "echo", name: "深渊", description: "深海回声 · 靛蓝加载", fx: "sonar", hue: 222,
			dark: cinematicPalette("#061016", "#142834", "#d4f0ff", "#6f8fa4", "#3ec8e8", "#041820"),
			light: cinematicPalette("#e4f1f2", "#ffffff", "#0e2a32", "#4e6e76", "#0e7490", "#ffffff")},
	}
	out := make([]AppearanceSkinTheme, 0, len(specs))
	for _, item := range specs {
		skin := cloneAppearanceSkin(classic, item.id, item.name, item.description)
		skin.Locked = true
		skin.Film = "/bg/skin-" + item.id + ".mp4"
		skin.Poster = "/bg/skin-" + item.id + ".jpg"
		skin.FX = item.fx
		skin.Hue = item.hue
		skin.Tokens.Dark = tintAppearanceSkinMode(skin.Tokens.Dark, item.dark)
		skin.Tokens.Light = tintAppearanceSkinMode(skin.Tokens.Light, item.light)
		skin.Tokens.Components = appearanceSkinComponentPreset(8, 8, 14, 16, 10, 4)
		out = append(out, skin)
	}
	return out
}

func cinematicPalette(canvas, surface, text, muted, primary, primaryFg string) appearanceSkinPalette {
	return appearanceSkinPalette{
		canvas: canvas, surface: surface, subtle: surface, raised: surface, overlay: surface, text: text, muted: muted, border: muted,
		primary: primary, primaryHover: primary, primaryActive: primary, primaryForeground: primaryFg, selected: surface, selectedHover: surface, selectedActive: surface, selectedForeground: text,
		switchChecked: primary, switchCheckedHover: primary, switchCheckedHandle: primaryFg, switchUnchecked: muted, switchUncheckedHover: muted, switchUncheckedHandle: text,
		success: "#4ade80", warning: "#fbbf24", danger: "#f87171", dangerHover: "#fca5a5", dangerActive: "#ef4444", dangerForeground: primaryFg, info: primary,
		workspace: surface, grid: muted, adminBackground: canvas, adminSurface: surface, adminSubtle: surface, adminStrong: surface,
		authBackground: canvas, authPanel: surface, authCard: surface, authAccent: primary, authMuted: muted,
	}
}

func defaultEnabledCinematicSkinIDs() []string {
	return []string{"apex", "nexus", "lumen", "prism", "radix", "helix", "ink", "ember", "veil", "echo"}
}

func isOfficialCinematicSkinID(id string) bool {
	for _, item := range defaultEnabledCinematicSkinIDs() {
		if item == id {
			return true
		}
	}
	return false
}

func appearanceSkinIDExists(themes []AppearanceSkinTheme, id string) bool {
	for _, theme := range themes {
		if theme.ID == id {
			return true
		}
	}
	return false
}

func pruneRetiredAppearanceSkins(themes []AppearanceSkinTheme) []AppearanceSkinTheme {
	out := make([]AppearanceSkinTheme, 0, len(themes))
	for _, theme := range themes {
		if isOfficialCinematicSkinID(theme.ID) {
			out = append(out, theme)
		}
	}
	return out
}

func ensureCinematicAppearanceSkins(themes []AppearanceSkinTheme) []AppearanceSkinTheme {
	seen := make(map[string]struct{}, len(themes))
	for _, theme := range themes {
		seen[theme.ID] = struct{}{}
	}
	classic := defaultClassicAppearanceSkin()
	for _, extra := range cinematicAppearanceSkins(classic) {
		if _, exists := seen[extra.ID]; exists {
			continue
		}
		if len(themes) >= maxAppearanceSkinThemes {
			break
		}
		themes = append(themes, extra)
	}
	return themes
}

func normalizeEnabledSkinIDs(ids []string, themes []AppearanceSkinTheme) []string {
	if ids == nil {
		ids = defaultEnabledCinematicSkinIDs()
	}
	allowed := make(map[string]struct{}, len(themes))
	for _, theme := range themes {
		allowed[theme.ID] = struct{}{}
	}
	out := make([]string, 0, len(ids))
	seen := map[string]struct{}{}
	for _, id := range ids {
		id = strings.ToLower(strings.TrimSpace(id))
		if _, ok := allowed[id]; !ok {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func normalizeAppearanceDefaultMode(value string) string {
	if strings.EqualFold(strings.TrimSpace(value), "light") {
		return "light"
	}
	return "dark"
}

func publicWorkspaceSkins(value AppearanceSetting) []AppearanceSkinTheme {
	out := make([]AppearanceSkinTheme, 0, len(value.SkinThemes))
	for _, theme := range value.SkinThemes {
		if !isOfficialCinematicSkinID(theme.ID) {
			continue
		}
		item := theme
		item.Film = ""
		out = append(out, item)
	}
	return out
}

func publicEnabledSkins(value AppearanceSetting) []AppearanceSkinTheme {
	wanted := make(map[string]struct{}, len(value.EnabledSkins))
	for _, id := range value.EnabledSkins {
		wanted[id] = struct{}{}
	}
	out := make([]AppearanceSkinTheme, 0, len(value.EnabledSkins))
	for _, theme := range value.SkinThemes {
		if _, ok := wanted[theme.ID]; !ok {
			continue
		}
		if theme.Film == "" && theme.Poster == "" {
			continue
		}
		out = append(out, theme)
	}
	return out
}

