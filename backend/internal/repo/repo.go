package repo

import "gorm.io/gorm"

// gormDB abstracts *gorm.DB for repo implementations.
type gormDB interface {
	Create(value interface{}) *gorm.DB
	Save(value interface{}) *gorm.DB
	First(dest interface{}, conds ...interface{}) *gorm.DB
	Find(dest interface{}, conds ...interface{}) *gorm.DB
	Where(query interface{}, args ...interface{}) *gorm.DB
	Model(value interface{}) *gorm.DB
	Select(query interface{}, args ...interface{}) *gorm.DB
	Scan(dest interface{}) *gorm.DB
	Delete(value interface{}, conds ...interface{}) *gorm.DB
	Joins(query string, args ...interface{}) *gorm.DB
	Table(name string, args ...interface{}) *gorm.DB
	Order(value interface{}) *gorm.DB
	Limit(limit int) *gorm.DB
	Count(count *int64) *gorm.DB
	Update(column string, value interface{}) *gorm.DB
}

// Ensure *gorm.DB implements gormDB.
var _ gormDB = (*gorm.DB)(nil)
