package utils

import (
	"database/sql/driver"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"reflect"
)

type IRepositoryModel interface {
	any
	PrimaryKey() string
}
type iRepositoryPrimaryKey interface {
	string | int | int64
}

type Repository[ModelType IRepositoryModel, PrimaryType iRepositoryPrimaryKey] struct {
	db    *gorm.DB
	model ModelType
}

func NewRepository[ModelType IRepositoryModel, PrimaryType iRepositoryPrimaryKey](db *gorm.DB) *Repository[ModelType, PrimaryType] {
	return &Repository[ModelType, PrimaryType]{
		db: db,
	}
}

func (r Repository[ModelType, PrimaryType]) Create(entities ...*ModelType) (err error) {
	if len(entities) > 0 {
		err = r.db.Omit(clause.Associations).Create(entities).Error
	}
	return
}

func (r Repository[ModelType, PrimaryType]) Delete(id PrimaryType) error {
	return r.db.Omit(clause.Associations).Delete(&r.model, id).Error
}

func (r Repository[ModelType, PrimaryType]) DeleteIn(ids []PrimaryType) error {
	return r.db.Omit(clause.Associations).Delete(&r.model, ids).Error
}

func (r Repository[ModelType, PrimaryType]) DeleteByField(field string, value any) error {
	builder := r.db.Omit(clause.Associations)
	return r.buildWhereCondition(builder, field, value).Delete(&r.model).Error
}

func (r Repository[ModelType, PrimaryType]) DeleteByConditions(conditions map[string]any) error {
	builder := r.db.Omit(clause.Associations)
	for k, v := range conditions {
		builder = r.buildWhereCondition(builder, k, v)
	}
	return builder.Delete(&r.model).Error
}

func (r Repository[ModelType, PrimaryType]) DeleteAll() error {
	builder := r.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Omit(clause.Associations)
	return builder.Delete(&r.model).Error
}

func (r Repository[ModelType, PrimaryType]) Save(entity *ModelType, fields ...string) error {
	return r.db.Omit(clause.Associations).Select(fields).Save(entity).Error
}

func (r Repository[ModelType, PrimaryType]) Update(id PrimaryType, field string, value any) error {
	return r.db.Omit(clause.Associations).Model(&r.model).Where(fmt.Sprintf("`%s`=?", r.model.PrimaryKey()), id).Select(field).Update(field, value).Error
}

func (r Repository[ModelType, PrimaryType]) UpdateIn(ids []PrimaryType, field string, value any) error {
	return r.db.Omit(clause.Associations).Model(&r.model).Where(fmt.Sprintf("`%s` in ?", r.model.PrimaryKey()), ids).Select(field).Update(field, value).Error
}

func (r Repository[ModelType, PrimaryType]) Updates(id PrimaryType, params map[string]any) error {
	selectedFields := make([]any, 0, len(params))
	for k := range params {
		selectedFields = append(selectedFields, k)
	}

	builder := r.db.Omit(clause.Associations).Model(&r.model).Where(fmt.Sprintf("`%s`=?", r.model.PrimaryKey()), id)
	if len(selectedFields) > 0 {
		builder = builder.Select(selectedFields[0], selectedFields[1:]...)
	}
	return builder.Updates(params).Error
}

func (r Repository[ModelType, PrimaryType]) UpdatesIn(ids []PrimaryType, params map[string]any) error {
	selectedFields := make([]any, 0, len(params))
	for k := range params {
		selectedFields = append(selectedFields, k)
	}

	builder := r.db.Omit(clause.Associations).Model(&r.model).Where(fmt.Sprintf("`%s` in ?", r.model.PrimaryKey()), ids)
	if len(selectedFields) > 0 {
		builder = builder.Select(selectedFields[0], selectedFields[1:]...)
	}
	return builder.Updates(params).Error
}

func (r Repository[ModelType, PrimaryType]) UpdateByConditions(conditions map[string]any, field string, value any) error {
	builder := r.db.Omit(clause.Associations).Model(&r.model)
	for k, v := range conditions {
		builder = r.buildWhereCondition(builder, k, v)
	}
	return builder.Select(field).Update(field, value).Error
}

func (r Repository[ModelType, PrimaryType]) UpdatesByConditions(conditions map[string]any, params map[string]any) error {
	selectedFields := make([]any, 0, len(params))
	for k := range params {
		selectedFields = append(selectedFields, k)
	}

	builder := r.db.Omit(clause.Associations).Model(&r.model)
	for k, v := range conditions {
		builder = r.buildWhereCondition(builder, k, v)
	}
	if len(selectedFields) > 0 {
		builder = builder.Select(selectedFields[0], selectedFields[1:]...)
	}
	return builder.Updates(params).Error
}

func (r Repository[ModelType, PrimaryType]) UpdateAll(field string, value any) error {
	return r.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Omit(clause.Associations).Model(&r.model).Update(field, value).Error
}

func (r Repository[ModelType, PrimaryType]) UpdatesAll(params map[string]any) error {
	selectedFields := make([]any, 0, len(params))
	for k := range params {
		selectedFields = append(selectedFields, k)
	}

	builder := r.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Omit(clause.Associations).Model(&r.model)
	if len(selectedFields) > 0 {
		builder = builder.Select(selectedFields[0], selectedFields[1:]...)
	}
	return builder.Updates(params).Error
}

func (r Repository[ModelType, PrimaryType]) Get(id PrimaryType, preloads ...string) (*ModelType, error) {
	var (
		err error
		m   ModelType
	)
	if err = r.buildPreloads(r.db, preloads...).Take(&m, fmt.Sprintf("`%s`=?", r.model.PrimaryKey()), id).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r Repository[ModelType, PrimaryType]) GetIn(values []PrimaryType, preloads ...string) ([]*ModelType, error) {
	return r.GetInOrderByLimitOffset(values, "", -1, -1, preloads...)
}

func (r Repository[ModelType, PrimaryType]) GetInLimit(values []PrimaryType, limit int, preloads ...string) ([]*ModelType, error) {
	return r.GetInOrderByLimitOffset(values, "", limit, -1, preloads...)
}

func (r Repository[ModelType, PrimaryType]) GetInOffset(values []PrimaryType, offset int, preloads ...string) ([]*ModelType, error) {
	return r.GetInOrderByLimitOffset(values, "", -1, offset, preloads...)
}

func (r Repository[ModelType, PrimaryType]) GetInLimitOffset(values []PrimaryType, limit, offset int, preloads ...string) ([]*ModelType, error) {
	return r.GetInOrderByLimitOffset(values, "", limit, offset, preloads...)
}

func (r Repository[ModelType, PrimaryType]) GetInOrderByLimitOffset(values []PrimaryType, orderBy string, limit, offset int, preloads ...string) ([]*ModelType, error) {
	var (
		err error
		m   []*ModelType
	)
	builder := r.buildPreloads(r.db, preloads...)
	builder = r.buildOrderByLimitOffset(builder, orderBy, limit, offset)
	err = builder.Find(&m, values).Error
	return m, err
}

func (r Repository[ModelType, PrimaryType]) GetFirst(preloads ...string) (*ModelType, error) {
	var (
		err error
		m   ModelType
	)
	builder := r.buildPreloads(r.db, preloads...)
	err = builder.First(&m).Error
	return &m, err
}

func (r Repository[ModelType, PrimaryType]) GetFirstByField(field string, value any, preloads ...string) (*ModelType, error) {
	var (
		err error
		m   ModelType
	)
	if err = r.buildWhereCondition(r.buildPreloads(r.db, preloads...), field, value).First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r Repository[ModelType, PrimaryType]) GetFirstByConditions(conditions map[string]any, preloads ...string) (*ModelType, error) {
	var (
		err error
		m   ModelType
	)
	builder := r.buildPreloads(r.db, preloads...)
	for k, v := range conditions {
		builder = r.buildWhereCondition(builder, k, v)
	}
	if err = builder.First(&m).Error; err != nil {
		return nil, err
	}
	return &m, err
}

func (r Repository[ModelType, PrimaryType]) GetFirstOrderByLimitOffset(orderBy string, limit, offset int, preloads ...string) (*ModelType, error) {
	var (
		err error
		m   ModelType
	)
	err = r.buildOrderByLimitOffset(r.buildPreloads(r.db, preloads...), orderBy, limit, offset).First(&m).Error
	return &m, err
}

func (r Repository[ModelType, PrimaryType]) GetFirstByFieldOrderByLimitOffset(field string, value any, orderBy string, limit, offset int, preloads ...string) (*ModelType, error) {
	var (
		err error
		m   *ModelType
	)
	builder := r.buildPreloads(r.db, preloads...)
	builder = r.buildWhereCondition(builder, field, value)
	builder = r.buildOrderByLimitOffset(builder, orderBy, limit, offset)
	err = builder.First(&m).Error
	return m, err
}

func (r Repository[ModelType, PrimaryType]) GetFirstByConditionsOrderByLimitOffset(conditions map[string]any, orderBy string, limit, offset int, preloads ...string) (*ModelType, error) {
	var (
		err error
		m   *ModelType
	)
	builder := r.buildPreloads(r.db, preloads...)
	for k, v := range conditions {
		builder = r.buildWhereCondition(builder, k, v)
	}
	builder = r.buildOrderByLimitOffset(builder, orderBy, limit, offset)
	err = builder.First(&m).Error
	return m, err
}

func (r Repository[ModelType, PrimaryType]) GetLast(preloads ...string) (*ModelType, error) {
	var (
		err error
		m   ModelType
	)
	builder := r.buildPreloads(r.db, preloads...)
	err = builder.Last(&m).Error
	return &m, err
}

func (r Repository[ModelType, PrimaryType]) GetLastByField(field string, value any, preloads ...string) (*ModelType, error) {
	var (
		err error
		m   ModelType
	)
	builder := r.buildPreloads(r.db, preloads...)
	if err = builder.Where(r.buildWhereCondition(builder, field, value)).Last(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r Repository[ModelType, PrimaryType]) GetLastByConditions(conditions map[string]any, preloads ...string) (*ModelType, error) {
	var (
		err error
		m   ModelType
	)
	builder := r.buildPreloads(r.db, preloads...)
	for k, v := range conditions {
		builder = r.buildWhereCondition(builder, k, v)
	}
	if err = builder.Last(&m).Error; err != nil {
		return nil, err
	}
	return &m, err
}

func (r Repository[ModelType, PrimaryType]) GetLastOrderByLimitOffset(orderBy string, limit, offset int, preloads ...string) (*ModelType, error) {
	var (
		err error
		m   ModelType
	)
	err = r.buildOrderByLimitOffset(r.buildPreloads(r.db, preloads...), orderBy, limit, offset).Last(&m).Error
	return &m, err
}

func (r Repository[ModelType, PrimaryType]) GetLastByFieldOrderByLimitOffset(field string, value any, orderBy string, limit, offset int, preloads ...string) (*ModelType, error) {
	var (
		err error
		m   *ModelType
	)
	builder := r.buildPreloads(r.db, preloads...)
	builder = r.buildWhereCondition(builder, field, value)
	builder = r.buildOrderByLimitOffset(builder, orderBy, limit, offset)
	err = builder.Last(&m).Error
	return m, err
}

func (r Repository[ModelType, PrimaryType]) GetLastByConditionsOrderByLimitOffset(conditions map[string]any, orderBy string, limit, offset int, preloads ...string) (*ModelType, error) {
	var (
		err error
		m   *ModelType
	)
	builder := r.buildPreloads(r.db, preloads...)
	for k, v := range conditions {
		builder = r.buildWhereCondition(builder, k, v)
	}
	builder = r.buildOrderByLimitOffset(builder, orderBy, limit, offset)
	err = builder.Last(&m).Error
	return m, err
}

func (r Repository[ModelType, PrimaryType]) GetAll(preloads ...string) ([]*ModelType, error) {
	return r.GetAllOrderByLimitOffset("", -1, -1, preloads...)
}

func (r Repository[ModelType, PrimaryType]) GetAllLimit(limit int, preloads ...string) ([]*ModelType, error) {
	return r.GetAllOrderByLimitOffset("", limit, -1, preloads...)
}

func (r Repository[ModelType, PrimaryType]) GetAllOffset(offset int, preloads ...string) ([]*ModelType, error) {
	return r.GetAllOrderByLimitOffset("", -1, offset, preloads...)
}

func (r Repository[ModelType, PrimaryType]) GetAllLimitOffset(limit, offset int, preloads ...string) ([]*ModelType, error) {
	return r.GetAllOrderByLimitOffset("", limit, offset, preloads...)
}

func (r Repository[ModelType, PrimaryType]) GetAllOrderByLimitOffset(orderBy string, limit, offset int, preloads ...string) ([]*ModelType, error) {
	var (
		err error
		m   []*ModelType
	)
	builder := r.buildPreloads(r.db, preloads...)
	builder = r.buildOrderByLimitOffset(builder, orderBy, limit, offset)
	err = builder.Find(&m).Error
	return m, err
}

func (r Repository[ModelType, PrimaryType]) GetByField(field string, value any, preloads ...string) (*ModelType, error) {
	var (
		err error
		m   ModelType
	)
	builder := r.buildPreloads(r.db, preloads...)
	if err = builder.Where(r.buildWhereCondition(builder, field, value)).Take(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r Repository[ModelType, PrimaryType]) GetByConditions(conditions map[string]any, preloads ...string) (*ModelType, error) {
	var (
		err error
		m   ModelType
	)
	builder := r.buildPreloads(r.db, preloads...)
	for k, v := range conditions {
		builder = r.buildWhereCondition(builder, k, v)
	}
	if err = builder.Take(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r Repository[ModelType, PrimaryType]) GetAllByField(field string, value any, preloads ...string) (m []*ModelType, err error) {
	return r.GetAllByFieldOrderByLimitOffset(field, value, "", -1, -1, preloads...)
}

func (r Repository[ModelType, PrimaryType]) GetAllByFieldLimit(field string, value any, limit int, preloads ...string) ([]*ModelType, error) {
	return r.GetAllByFieldOrderByLimitOffset(field, value, "", limit, -1, preloads...)
}

func (r Repository[ModelType, PrimaryType]) GetAllByFieldOffset(field string, value any, offset int, preloads ...string) ([]*ModelType, error) {
	return r.GetAllByFieldOrderByLimitOffset(field, value, "", -1, offset, preloads...)
}

func (r Repository[ModelType, PrimaryType]) GetAllByFieldLimitOffset(field string, value any, limit, offset int, preloads ...string) ([]*ModelType, error) {
	return r.GetAllByFieldOrderByLimitOffset(field, value, "", limit, offset, preloads...)
}

func (r Repository[ModelType, PrimaryType]) GetAllByFieldOrderByLimitOffset(field string, value any, orderBy string, limit, offset int, preloads ...string) ([]*ModelType, error) {
	var (
		err error
		m   []*ModelType
	)
	builder := r.buildPreloads(r.db, preloads...)
	builder = r.buildOrderByLimitOffset(builder, orderBy, limit, offset)
	err = builder.Where(r.buildWhereCondition(builder, field, value)).Find(&m).Error
	return m, err
}

func (r Repository[ModelType, PrimaryType]) GetAllByConditions(conditions map[string]any, preloads ...string) ([]*ModelType, error) {
	return r.GetAllByConditionsOrderByLimitOffset(conditions, "", -1, -1, preloads...)
}

func (r Repository[ModelType, PrimaryType]) GetAllByConditionsLimit(conditions map[string]any, limit int, preloads ...string) ([]*ModelType, error) {
	return r.GetAllByConditionsOrderByLimitOffset(conditions, "", limit, -1, preloads...)
}

func (r Repository[ModelType, PrimaryType]) GetAllByConditionsOffset(conditions map[string]any, offset int, preloads ...string) ([]*ModelType, error) {
	return r.GetAllByConditionsOrderByLimitOffset(conditions, "", -1, offset, preloads...)
}

func (r Repository[ModelType, PrimaryType]) GetAllByConditionsLimitOffset(conditions map[string]any, limit, offset int, preloads ...string) ([]*ModelType, error) {
	return r.GetAllByConditionsOrderByLimitOffset(conditions, "", limit, offset, preloads...)
}

func (r Repository[ModelType, PrimaryType]) GetAllByConditionsOrderByLimitOffset(conditions map[string]any, orderBy string, limit, offset int, preloads ...string) ([]*ModelType, error) {
	var (
		err error
		m   = make([]*ModelType, 0)
	)
	builder := r.buildPreloads(r.db, preloads...)
	builder = r.buildOrderByLimitOffset(builder, orderBy, limit, offset)
	for k, v := range conditions {
		builder = r.buildWhereCondition(builder, k, v)
	}
	if err = builder.Find(&m).Error; err != nil {
		return nil, err
	}
	return m, nil
}

func (r Repository[ModelType, PrimaryType]) CountByField(field string, value any) (count int64, err error) {
	builder := r.db.Model(&r.model)
	if err = builder.Where(r.buildWhereCondition(builder, field, value)).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r Repository[ModelType, PrimaryType]) CountByConditions(conditions map[string]any) (count int64, err error) {
	builder := r.db.Model(&r.model)
	for k, v := range conditions {
		builder = r.buildWhereCondition(builder, k, v)
	}
	if err = builder.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r Repository[ModelType, PrimaryType]) DBConn() *gorm.DB {
	return r.db
}

func (r Repository[ModelType, PrimaryType]) buildWhereCondition(builder *gorm.DB, k string, v any) *gorm.DB {
	if valuer, ok := v.(driver.Valuer); ok {
		v, _ = valuer.Value()
	}
	if v == nil {
		return builder.Where(fmt.Sprintf("`%s` IS NULL", k))
	}

	t := reflect.TypeOf(v)
	switch t.Kind() {
	case reflect.Slice:
		fallthrough
	case reflect.Array:
		builder = builder.Where(fmt.Sprintf("`%s` in ?", k), v)
	default:
		if condition, ok := v.(clause.Expression); ok {
			builder = builder.Where(condition)
		} else {
			builder = builder.Where(fmt.Sprintf("`%s`=?", k), v)
		}
	}
	return builder
}

func (r Repository[ModelType, PrimaryType]) buildPreloads(builder *gorm.DB, preloads ...string) *gorm.DB {
	for _, v := range preloads {
		builder = builder.Preload(v)
	}
	return builder
}

func (r Repository[ModelType, PrimaryType]) buildOrderByLimitOffset(builder *gorm.DB, orderBy string, limit, offset int) *gorm.DB {
	if orderBy != "" {
		builder = builder.Order(orderBy)
	}
	if limit > 0 {
		builder = builder.Limit(limit)
	}
	if offset > 0 {
		builder = builder.Offset(offset)
	}
	return builder
}
