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

// SavePhotoEdits sauvegarde les modifications d'image dans le fichier sidecar YAML.
//
// POST /api/v1/photos/:uid/edits
func SavePhotoEdits(router *gin.RouterGroup) {
	router.POST("/photos/:uid/edits", func(c *gin.Context) {
		// Auth standard PhotoPrism : Auth(...) renvoie *entity.Session ou nil.
		s := Auth(c, acl.ResourcePhotos, acl.ActionUpdate)
		if s == nil {
			AbortUnauthorized(c)
			return
		}

		// Obtenir l'UID de la photo
		uid := clean.UID(c.Param("uid"))
		if uid == "" {
			Abort(c, http.StatusBadRequest, i18n.ErrNotFound)
			return
		}

		// Charger la photo depuis la base de données
		m, err := query.PhotoByUID(uid)
		if err != nil {
			log.Errorf("api: photo not found for uid=%s: %v", uid, err)
			AbortEntityNotFound(c)
			return
		}

		// Obtenir le fichier principal de la photo
		primaryFile, err := m.PrimaryFile()
		if err != nil {
			log.Errorf("api: cannot find primary file for photo uid=%s: %v", uid, err)
			Abort(c, http.StatusBadRequest, i18n.ErrSaveFailed)
			return
		}

		// Vérifier que le fichier a un chemin
		if primaryFile.FileName == "" {
			log.Errorf("api: primary file has no filename for photo uid=%s", uid)
			Abort(c, http.StatusBadRequest, i18n.ErrSaveFailed)
			return
		}

		// Body JSON
		var req struct {
			SidecarData photoprism.ImageEdits `json:"sidecarData"`
		}

		// Log du body brut pour debug
		bodyBytes, _ := c.GetRawData()
		log.Infof("api: raw request body: %s", string(bodyBytes))

		// Réinitialiser le body pour BindJSON
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		if err := c.BindJSON(&req); err != nil {
			log.Errorf("api: failed to bind JSON: %v", err)
			AbortBadRequest(c, err)
			return
		}

		// Log pour debugging
		log.Infof("api: photo edits received for uid=%s", uid)
		log.Infof("api: crop L=%.3f T=%.3f W=%.3f H=%.3f",
			req.SidecarData.Crop.Left, req.SidecarData.Crop.Top,
			req.SidecarData.Crop.Width, req.SidecarData.Crop.Height)
		log.Infof("api: rotation=%d° flip H=%v V=%v",
			req.SidecarData.Rotation,
			req.SidecarData.Flip.Horizontal,
			req.SidecarData.Flip.Vertical)

		// Obtenir la config pour les chemins
		conf := get.Config()

		// Obtenir le chemin du fichier sidecar YAML
		sidecarPath, relPath, err := m.YamlFileName(conf.OriginalsPath(), conf.SidecarPath())
		if err != nil {
			log.Errorf("api: cannot get sidecar path for photo uid=%s: %v", uid, err)
			Abort(c, http.StatusBadRequest, i18n.ErrSaveFailed)
			return
		}

		log.Infof("api: sidecar path: %s (relative: %s)", sidecarPath, relPath)

		// Sauvegarder les éditions dans le sidecar YAML
		if err := photoprism.SaveImageEditsToSidecar(sidecarPath, &req.SidecarData); err != nil {
			AbortSaveFailed(c)
			return
		}

		// Mettre à jour EditedAt dans la base de données
		if err := entity.Db().Model(&m).Update("EditedAt", time.Now()).Error; err != nil {
			log.Warnf("api: failed to update EditedAt for photo uid=%s: %v", uid, err)
		}

		// Note: Les thumbnails standards ne sont pas supprimés car l'éditeur utilise
		// un endpoint dédié (/photos/:uid/preview/:size) qui applique les éditions à la volée

		// Événement & réponse
		event.SuccessMsg(i18n.MsgChangesSaved)
		PublishPhotoEvent(StatusUpdated, uid, c)

		c.JSON(http.StatusOK, gin.H{
			"message": "Image edits saved successfully",
			"photo":   m,
			"edits":   req.SidecarData,
		})
	})
}

// GetPhotoEdits récupère les modifications d'image depuis le fichier sidecar YAML.
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

		// Obtenir la config pour les chemins
		conf := get.Config()

		// Obtenir le chemin du fichier sidecar YAML
		sidecarPath, _, err := m.YamlFileName(conf.OriginalsPath(), conf.SidecarPath())
		if err != nil {
			log.Errorf("api: cannot get sidecar path for photo uid=%s: %v", uid, err)
			Abort(c, http.StatusBadRequest, i18n.ErrNotFound)
			return
		}

		// Charger les éditions depuis le sidecar YAML
		edits, err := photoprism.LoadImageEditsFromSidecar(sidecarPath)
		if err != nil {
			// Retourner des éditions par défaut en cas d'erreur
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

// DeletePhotoEdits supprime les modifications d'image du fichier sidecar YAML.
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

		// Obtenir la config pour les chemins
		conf := get.Config()

		// Obtenir le chemin du fichier sidecar YAML
		sidecarPath, _, err := m.YamlFileName(conf.OriginalsPath(), conf.SidecarPath())
		if err != nil {
			log.Errorf("api: cannot get sidecar path for photo uid=%s: %v", uid, err)
			Abort(c, http.StatusBadRequest, i18n.ErrDeleteFailed)
			return
		}

		// Supprimer les éditions du sidecar YAML
		if err := photoprism.DeleteImageEditsFromSidecar(sidecarPath); err != nil {
			AbortDeleteFailed(c)
			return
		}

		// Mettre à jour EditedAt (on considère que la photo est "modifiée")
		if err := entity.Db().Model(m).Update("EditedAt", time.Now()).Error; err != nil {
			// warning silencieux
		}

		event.SuccessMsg(i18n.MsgChangesSaved)
		PublishPhotoEvent(StatusUpdated, uid, c)

		c.JSON(http.StatusOK, gin.H{
			"message": "Image edits deleted successfully",
		})
	})
}
