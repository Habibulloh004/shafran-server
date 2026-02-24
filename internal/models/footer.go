package models

// FooterSettings stores dynamic footer content managed via admin panel.
// There should be only one row (singleton pattern).
type FooterSettings struct {
	BaseModel
	Address     string `json:"address"`
	Phone       string `json:"phone"`
	Phone2      string `json:"phone2"`
	Email       string `json:"email"`
	WorkingHours string `json:"working_hours"`
	WorkingHoursUz string `json:"working_hours_uz"`
	WorkingHoursRu string `json:"working_hours_ru"`
	WorkingHoursEn string `json:"working_hours_en"`
	WorkingHoursTitleUz string `json:"working_hours_title_uz"`
	WorkingHoursTitleRu string `json:"working_hours_title_ru"`
	WorkingHoursTitleEn string `json:"working_hours_title_en"`

	// Social links
	Telegram  string `json:"telegram"`
	Instagram string `json:"instagram"`
	Facebook  string `json:"facebook"`
	Youtube   string `json:"youtube"`
	Twitter   string `json:"twitter"`
	TikTok    string `json:"tiktok"`

	// Toggle social link visibility
	TelegramEnabled  bool `json:"telegram_enabled"`
	InstagramEnabled bool `json:"instagram_enabled"`
	FacebookEnabled  bool `json:"facebook_enabled"`
	YoutubeEnabled   bool `json:"youtube_enabled"`
	TwitterEnabled   bool `json:"twitter_enabled"`
	TikTokEnabled    bool `json:"tiktok_enabled"`

	// Extra text sections
	SubscribeTitleUz string `json:"subscribe_title_uz"`
	SubscribeTitleRu string `json:"subscribe_title_ru"`
	SubscribeTitleEn string `json:"subscribe_title_en"`
	CopyrightTextUz string `json:"copyright_text_uz"`
	CopyrightTextRu string `json:"copyright_text_ru"`
	CopyrightTextEn string `json:"copyright_text_en"`
	CopyrightText string `json:"copyright_text"`
}
