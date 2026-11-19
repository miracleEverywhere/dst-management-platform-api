package models

type System struct {
	Key   string `gorm:"primaryKey;not null"`
	Value string `gorm:"not null"`
}

func (System) TableName() string {
	return "system"
}

var (
	JwtSecret        = "jwt_secret"
	CloudKey         = "cloud_key"
	AIBaseSettingKey = "ai_base_setting"
	// DiagnosisAISettingKey 保存 AI 诊断专用的 LLM 配置。
	DiagnosisAISettingKey = "ai_diagnosis_setting"
	ReadmeKey             = "readme"
)
