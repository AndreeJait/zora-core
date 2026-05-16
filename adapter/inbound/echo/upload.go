package echo

import (
	"net/http"

	"github.com/AndreeJait/go-utility/v2/httpw/echow"
	"github.com/AndreeJait/go-utility/v2/responsew"
	"github.com/AndreeJait/zora-core/port/inbound/upload"
	"github.com/labstack/echo/v5"
)

// RegisterUploadRoutes registers the file upload route.
//
// @Summary      Upload a file to object storage
// @Description  Upload a file to MinIO and return the object key and presigned URL
// @Tags         upload
// @Accept       multipart/form-data
// @Produce      json
// @Param        file    formData  file    true  "File to upload"
// @Param        bucket  formData  string  false "Target bucket (default: zora-files)"
// @Param        prefix  formData  string  false "Path prefix (default: uploads)"
// @Success      200  {object}  upload.UploadResult
// @Failure      400  {object}  responsew.BaseResponse
// @Failure      500  {object}  responsew.BaseResponse
// @Security     ApiKeyAuth
// @Router       /api/v1/upload [post]
func RegisterUploadRoutes(r RouteRegistrar, uploadUC upload.UseCase) {
	r.POST("/api/v1/upload", echow.Bind(handleUpload(uploadUC)))
}

func handleUpload(uploadUC upload.UseCase) func(c *echo.Context) (any, error) {
	return func(c *echo.Context) (any, error) {
		file, err := (*c).FormFile("file")
		if err != nil {
			return nil, err
		}

		src, err := file.Open()
		if err != nil {
			return nil, err
		}
		defer src.Close()

		data := make([]byte, file.Size)
		if _, err := src.Read(data); err != nil {
			return nil, err
		}

		bucket := (*c).FormValue("bucket")
		prefix := (*c).FormValue("prefix")

		result, err := uploadUC.Upload(c.Request().Context(), bucket, prefix, file.Filename, file.Header.Get("Content-Type"), data)
		if err != nil {
			return nil, err
		}

		return responsew.Success(result, "File uploaded"), nil
	}
}

// Ensure http.StatusOK is used by the compiler.
var _ = http.StatusOK