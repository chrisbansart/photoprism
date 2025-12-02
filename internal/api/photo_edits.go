package api

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/photoprism/photoprism/internal/auth/acl"
	"github.com/photoprism/photoprism/internal/entity"
	"github.com/photoprism/photoprism/internal/entity/query"
	"github.com/photoprism/photoprism/internal/event"
	"github.com/photoprism/photoprism/internal/photoprism"
	"github.com/photoprism/photoprism/internal/photoprism/get"
	"github.com/photoprism/photoprism/pkg/clean"
	"github.com/photoprism/photoprism/pkg/i18n"
)

// SavePhotoEdits saves image modifications to the YAML sidecar file.
//
// POST /api/v1/photos/:uid/edits
func SavePhotoEdits(router *gin.RouterGroup) {
	router.POST("/photos/:uid/edits", func(c *gin.Context) {
		// Standard PhotoPrism auth: Auth(...) returns *entity.Session or nil.
		s := Auth(c, acl.ResourcePhotos, acl.ActionUpdate)
		if s == nil {
			AbortUnauthorized(c)
			return
		}

		// Get photo UID
		uid := clean.UID(c.Param("uid"))
		if uid == "" {
			Abort(c, http.StatusBadRequest, i18n.ErrNotFound)
			return
		}

		// Load photo from database
		m, err := query.PhotoByUID(uid)
		if err != nil {
			log.Errorf("api: photo not found for uid=%s: %v", uid, err)
			AbortEntityNotFound(c)
			return
		}

		// Get primary file of the photo
		primaryFile, err := m.PrimaryFile()
		if err != nil {
			log.Errorf("api: cannot find primary file for photo uid=%s: %v", uid, err)
			Abort(c, http.StatusBadRequest, i18n.ErrSaveFailed)
			return
		}

		// Verify file has a path
		if primaryFile.FileName == "" {
			log.Errorf("api: primary file has no filename for photo uid=%s", uid)
			Abort(c, http.StatusBadRequest, i18n.ErrSaveFailed)
			return
		}

		// JSON body
		var req struct {
			SidecarData photoprism.ImageEdits `json:"sidecarData"`
		}

		// Log raw body for debugging
		bodyBytes, _ := c.GetRawData()
		log.Infof("api: raw request body: %s", string(bodyBytes))

		// Reset body for BindJSON
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		if err := c.BindJSON(&req); err != nil {
			log.Errorf("api: failed to bind JSON: %v", err)
			AbortBadRequest(c, err)
			return
		}

		// Log for debugging
		log.Infof("api: photo edits received for uid=%s", uid)
		log.Infof("api: crop L=%.3f T=%.3f W=%.3f H=%.3f",
			req.SidecarData.Crop.Left, req.SidecarData.Crop.Top,
			req.SidecarData.Crop.Width, req.SidecarData.Crop.Height)
		log.Infof("api: rotation=%d° flip H=%v V=%v",
			req.SidecarData.Rotation,
			req.SidecarData.Flip.Horizontal,
			req.SidecarData.Flip.Vertical)

		// Get config for paths
		conf := get.Config()

		// Get sidecar YAML file path
		sidecarPath, relPath, err := m.YamlFileName(conf.OriginalsPath(), conf.SidecarPath())
		if err != nil {
			log.Errorf("api: cannot get sidecar path for photo uid=%s: %v", uid, err)
			Abort(c, http.StatusBadRequest, i18n.ErrSaveFailed)
			return
		}

		log.Infof("api: sidecar path: %s (relative: %s)", sidecarPath, relPath)

		// Save edits to sidecar YAML
		if err := photoprism.SaveImageEditsToSidecar(sidecarPath, &req.SidecarData); err != nil {
			AbortSaveFailed(c)
			return
		}

		// Update EditedAt in database
		if err := entity.Db().Model(&m).Update("EditedAt", time.Now()).Error; err != nil {
			log.Warnf("api: failed to update EditedAt for photo uid=%s: %v", uid, err)
		}

		// Note: Standard thumbnails are not deleted because the editor uses
		// a dedicated endpoint (/photos/:uid/preview/:size) that applies edits on-the-fly

		// Event & response
		event.SuccessMsg(i18n.MsgChangesSaved)
		PublishPhotoEvent(StatusUpdated, uid, c)

		c.JSON(http.StatusOK, gin.H{
			"message": "Image edits saved successfully",
			"photo":   m,
			"edits":   req.SidecarData,
		})
	})
}

// GetPhotoEdits retrieves image modifications from the YAML sidecar file.
//
// GET /api/v1/photos/:uid/edits
func GetPhotoEdits(router *gin.RouterGroup) {
	router.GET("/photos/:uid/edits", func(c *gin.Context) {
		s := Auth(c, acl.ResourcePhotos, acl.ActionView)
		if s == nil {
			AbortUnauthorized(c)
			return
		}

		uid := clean.UID(c.Param("uid"))
		if uid == "" {
			Abort(c, http.StatusBadRequest, i18n.ErrNotFound)
			return
		}

		m, err := query.PhotoByUID(uid)
		if err != nil {
			log.Errorf("api: photo not found for uid=%s: %v", uid, err)
			AbortEntityNotFound(c)
			return
		}

		// Get config for paths
		conf := get.Config()

		// Get sidecar YAML file path
		sidecarPath, _, err := m.YamlFileName(conf.OriginalsPath(), conf.SidecarPath())
		if err != nil {
			log.Errorf("api: cannot get sidecar path for photo uid=%s: %v", uid, err)
			Abort(c, http.StatusBadRequest, i18n.ErrNotFound)
			return
		}

		// Load edits from sidecar YAML
		edits, err := photoprism.LoadImageEditsFromSidecar(sidecarPath)
		if err != nil {
			// Return default edits in case of error
			edits = &photoprism.ImageEdits{
				Crop: photoprism.CropEdits{
					Left:   0,
					Top:    0,
					Width:  1,
					Height: 1,
				},
				Rotation: 0,
				Flip: photoprism.FlipEdits{
					Horizontal: false,
					Vertical:   false,
				},
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"edits": edits,
		})
	})
}

// DeletePhotoEdits deletes image modifications from the YAML sidecar file.
//
// DELETE /api/v1/photos/:uid/edits
func DeletePhotoEdits(router *gin.RouterGroup) {
	router.DELETE("/photos/:uid/edits", func(c *gin.Context) {
		s := Auth(c, acl.ResourcePhotos, acl.ActionUpdate)
		if s == nil {
			AbortUnauthorized(c)
			return
		}

		uid := clean.UID(c.Param("uid"))
		if uid == "" {
			Abort(c, http.StatusBadRequest, i18n.ErrNotFound)
			return
		}

		m, err := query.PhotoByUID(uid)
		if err != nil {
			log.Errorf("api: photo not found for uid=%s: %v", uid, err)
			AbortEntityNotFound(c)
			return
		}

		// Get config for paths
		conf := get.Config()

		// Get sidecar YAML file path
		sidecarPath, _, err := m.YamlFileName(conf.OriginalsPath(), conf.SidecarPath())
		if err != nil {
			log.Errorf("api: cannot get sidecar path for photo uid=%s: %v", uid, err)
			Abort(c, http.StatusBadRequest, i18n.ErrDeleteFailed)
			return
		}

		// Delete edits from sidecar YAML
		if err := photoprism.DeleteImageEditsFromSidecar(sidecarPath); err != nil {
			AbortDeleteFailed(c)
			return
		}

		// Update EditedAt (we consider the photo as "modified")
		if err := entity.Db().Model(m).Update("EditedAt", time.Now()).Error; err != nil {
			// silent warning
		}

		event.SuccessMsg(i18n.MsgChangesSaved)
		PublishPhotoEvent(StatusUpdated, uid, c)

		c.JSON(http.StatusOK, gin.H{
			"message": "Image edits deleted successfully",
		})
	})
}
