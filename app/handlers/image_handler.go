package handlers

import (
	"BE-E-MEETING/app/usecases"
	"github.com/labstack/echo/v4"
	"net/http"
)

type ImageHandler struct {
	imageUsecase usecases.ImageUsecase
}

func NewImageHandler(imageUsecase usecases.ImageUsecase) *ImageHandler {
	return &ImageHandler{imageUsecase: imageUsecase}
}

// UploadImage godoc
// @Summary Save an image
// @Description Upload an image to temp folder and return its URL
// @Tags Image
// @Accept multipart/form-data
// @Produce json
// @Param image formData file true "Image file"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uploads [post]
func (h *ImageHandler) UploadImage(c echo.Context) error {
	file, err := c.FormFile("image")
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Failed to upload image"})
	}

	baseURL := c.Scheme() + "://" + c.Request().Host
	imageURL, err := h.imageUsecase.UploadImage(file, baseURL)
	if err != nil {
		if e, ok := err.(*usecases.UseCaseError); ok {
			return c.JSON(e.Code, echo.Map{"error": e.Message})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "internal server error"})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"message":  "Image uploaded successfully",
		"imageURL": imageURL,
	})
}
