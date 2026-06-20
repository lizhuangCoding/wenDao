package upload

import (
	"errors"

	"github.com/gin-gonic/gin"

	"wenDao/internal/pkg/response"
	"wenDao/internal/service"
	"wenDao/internal/svcerrors"
)

// UploadHandler 上传处理器
type UploadHandler struct {
	uploadService service.UploadService
}

// NewUploadHandler 创建上传处理器实例
func NewUploadHandler(uploadService service.UploadService) *UploadHandler {
	return &UploadHandler{
		uploadService: uploadService,
	}
}

// UploadImage 上传图片（管理员，用于 Markdown 编辑器）
func (h *UploadHandler) UploadImage(c *gin.Context) {
	// 获取上传的文件
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.InvalidParams(c, "请选择要上传的图片文件")
		return
	}
	defer file.Close()

	// 从 context 获取当前用户 ID
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "登录状态已失效，请重新登录后上传")
		return
	}

	// 上传图片并添加版权水印。封面图使用更靠近中心的水印，避免前台裁剪后不可见。
	uploadFunc := h.uploadService.UploadArticleImage
	if c.PostForm("usage") == "cover" {
		uploadFunc = h.uploadService.UploadCoverImage
	}
	upload, err := uploadFunc(file, header, userID.(int64))
	if err != nil {
		if errors.Is(err, svcerrors.ErrFileTypeNotAllowed) {
			response.InvalidParams(c, "图片格式不支持，请上传 jpg、png、gif 或 webp 图片")
		} else if errors.Is(err, svcerrors.ErrFileSizeExceedsLimit) {
			response.InvalidParams(c, err.Error())
		} else {
			response.InternalError(c, "图片上传失败，请稍后重试")
		}
		return
	}

	// 返回上传结果
	response.Success(c, gin.H{
		"url":      upload.FilePath,
		"filename": upload.Filename,
		"size":     upload.FileSize,
	})
}
