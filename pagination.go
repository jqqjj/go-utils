package utils

import (
	"fmt"
	"gorm.io/gorm"
	"strings"
)

type PaginationResponse[ResponseType any] struct {
	Page    int             `json:"page"`
	PerPage int             `json:"per_page"`
	Total   int64           `json:"total"`
	List    []*ResponseType `json:"list"`
}

type Pagination[ModelType IRepositoryModel, PrimaryType iRepositoryPrimaryKey, ResponseType any] struct {
	response PaginationResponse[ModelType]
	repo     *Repository[ModelType, PrimaryType]

	conditions map[string]any
	scope      func(db *gorm.DB) *gorm.DB

	sortOnlyColumns []string
	sort            string
	descending      string
	preloads        []string

	presenter IPresenter[ModelType, ResponseType]
}

func NewPagination[ModelType IRepositoryModel, PrimaryType iRepositoryPrimaryKey, ResponseType any](
	repo *Repository[ModelType, PrimaryType], conditions map[string]any, presenter IPresenter[ModelType, ResponseType],
) *Pagination[ModelType, PrimaryType, ResponseType] {
	return &Pagination[ModelType, PrimaryType, ResponseType]{
		repo:       repo,
		conditions: conditions,
		presenter:  presenter,
	}
}

func (p *Pagination[ModelType, PrimaryType, ResponseType]) SetPresenter(presenter IPresenter[ModelType, ResponseType]) *Pagination[ModelType, PrimaryType, ResponseType] {
	p.presenter = presenter
	return p
}

func (p *Pagination[ModelType, PrimaryType, ResponseType]) SetPreloads(preloads ...string) *Pagination[ModelType, PrimaryType, ResponseType] {
	p.preloads = preloads
	return p
}

func (p *Pagination[ModelType, PrimaryType, ResponseType]) SetSort(column string, descending bool) *Pagination[ModelType, PrimaryType, ResponseType] {
	if descending {
		return p.setSort(column, "desc")
	} else {
		return p.setSort(column, "asc")
	}
}

func (p *Pagination[ModelType, PrimaryType, ResponseType]) setSort(column, descending string) *Pagination[ModelType, PrimaryType, ResponseType] {
	if strings.ToLower(descending) == "desc" {
		p.descending = "desc"
	} else {
		p.descending = "asc"
	}
	p.sort = column
	return p
}

func (p *Pagination[ModelType, PrimaryType, ResponseType]) SetSortOnlyColumns(columns []string) *Pagination[ModelType, PrimaryType, ResponseType] {
	p.sortOnlyColumns = columns
	return p
}

func (p *Pagination[ModelType, PrimaryType, ResponseType]) SetScope(scope func(db *gorm.DB) *gorm.DB) *Pagination[ModelType, PrimaryType, ResponseType] {
	p.scope = scope
	return p
}

func (p *Pagination[ModelType, PrimaryType, ResponseType]) Paginate(page, perPage int) (*PaginationResponse[ResponseType], error) {
	var (
		err        error
		count      int64
		m          ModelType
		entities   []*ModelType
		collection = make([]*ResponseType, 0)
		builder    = p.repo.db
	)

	//构造条件
	if p.scope != nil {
		builder = p.scope(builder)
	}
	for k, v := range p.conditions {
		builder = p.repo.buildWhereCondition(builder, k, v)
	}

	//查询总数
	if err = builder.Model(&m).Count(&count).Error; err != nil {
		return nil, err
	}

	//preloads
	if len(p.preloads) > 0 {
		builder = p.repo.buildPreloads(builder, p.preloads...)
	}

	//排序
	if p.sort != "" && p.descending != "" {
		if len(p.sortOnlyColumns) > 0 {
			for _, v := range p.sortOnlyColumns {
				if v == p.sort {
					builder = builder.Order(fmt.Sprintf("%s %s", p.sort, p.descending))
					break
				}
			}
		} else {
			builder = builder.Order(fmt.Sprintf("%s %s", p.sort, p.descending))
		}
	}

	//分页设置
	if perPage <= 0 {
		perPage = 15
	}
	if page < 1 {
		page = 1
	}
	builder = builder.Limit(perPage).Offset(perPage * (page - 1))
	if err = builder.Find(&entities).Error; err != nil {
		return nil, err
	}

	for _, v := range entities {
		collection = append(collection, p.presenter.Present(v))
	}
	return &PaginationResponse[ResponseType]{
		Page:    page,
		PerPage: perPage,
		Total:   count,
		List:    collection,
	}, nil
}
