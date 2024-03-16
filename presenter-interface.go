package utils

type IPresenter[ModelType IRepositoryModel, ResponseType any] interface {
	Present(entity *ModelType) *ResponseType
}
