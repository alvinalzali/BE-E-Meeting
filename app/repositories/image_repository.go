package repositories

type ImageRepository interface {
}

type imageRepository struct {
}

func NewImageRepository() ImageRepository {
	return &imageRepository{}
}
