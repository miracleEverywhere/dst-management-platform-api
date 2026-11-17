package dao

import (
	"dst-management-platform-api/database/models"
	"time"

	"gorm.io/gorm"
)

// DiagnosisDAO 是 AI 诊断历史与修复审计的数据访问层。
type DiagnosisDAO struct {
	db *gorm.DB
}

func NewDiagnosisDAO(db *gorm.DB) *DiagnosisDAO {
	return &DiagnosisDAO{db: db}
}

func (d *DiagnosisDAO) CreateRecord(record *models.DiagnosisRecord) error {
	return d.db.Create(record).Error
}

func (d *DiagnosisDAO) UpdateRecord(record *models.DiagnosisRecord) error {
	return d.db.Save(record).Error
}

func (d *DiagnosisDAO) GetRecord(id string) (*models.DiagnosisRecord, error) {
	var record models.DiagnosisRecord
	err := d.db.Where("id = ?", id).First(&record).Error
	return &record, err
}

// ListRecords 按房间范围、诊断范围与状态分页查询历史。
// roomIDs 为 nil 表示不限制房间（管理员），空切片表示没有可见房间。
func (d *DiagnosisDAO) ListRecords(roomIDs []int, scope, status string, page, pageSize int) (*PaginatedResult[models.DiagnosisRecord], error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
	}
	if roomIDs != nil && len(roomIDs) == 0 {
		return &PaginatedResult[models.DiagnosisRecord]{
			Data: []models.DiagnosisRecord{}, Page: page, PageSize: pageSize,
		}, nil
	}

	query := d.db.Model(&models.DiagnosisRecord{})
	if roomIDs != nil {
		query = query.Where("room_id IN ?", roomIDs)
	}
	if scope != "" {
		query = query.Where("scope = ?", scope)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	var records []models.DiagnosisRecord
	if err := query.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&records).Error; err != nil {
		return nil, err
	}
	if records == nil {
		records = []models.DiagnosisRecord{}
	}

	return &PaginatedResult[models.DiagnosisRecord]{
		Data: records, Page: page, PageSize: pageSize, TotalCount: total,
	}, nil
}

func (d *DiagnosisDAO) CreateActionRecord(record *models.DiagnosisActionRecord) error {
	return d.db.Create(record).Error
}

func (d *DiagnosisDAO) UpdateActionRecord(record *models.DiagnosisActionRecord) error {
	return d.db.Save(record).Error
}

func (d *DiagnosisDAO) ListActionRecords(diagnosisID string) ([]models.DiagnosisActionRecord, error) {
	var records []models.DiagnosisActionRecord
	err := d.db.Where("diagnosis_id = ?", diagnosisID).Order("id ASC").Find(&records).Error
	if records == nil {
		records = []models.DiagnosisActionRecord{}
	}
	return records, err
}

// DeleteOlderThan 清理超过保留期的诊断历史。
func (d *DiagnosisDAO) DeleteOlderThan(before time.Time) error {
	return d.db.Where("created_at < ?", before).Delete(&models.DiagnosisRecord{}).Error
}

// MarkInterrupted 将服务异常退出时遗留的 running 记录标记为 interrupted。
func (d *DiagnosisDAO) MarkInterrupted(before time.Time) error {
	return d.db.Model(&models.DiagnosisRecord{}).
		Where("status = ? AND created_at < ?", "running", before).
		Update("status", "interrupted").Error
}
