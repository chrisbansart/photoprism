package api

import (
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

		conf := get.Config()

		// Obtenir l'UID de la photo
		uid := clean.UID(c.Param("uid"))
		if uid == "" {
			Abort(c, http.StatusBadRequest, i18n.ErrNotFound)
			return
		}

		// Charger la photo depuis la base de données
		m, err := query.PhotoByUID(uid)
		if err != nil {
			AbortEntityNotFound(c)
			return
		}

		// Vérifier que la photo a un fichier
		if m.PhotoPath == "" || m.PhotoName == "" {
			Abort(c, http.StatusBadRequest, i18n.ErrSaveFailed)
			return
		}

		// Body JSON
		var req struct {
			SidecarData photoprism.ImageEdits `json:"sidecarData"`
		}

		if err := c.BindJSON(&req); err != nil {
			AbortBadRequest(c, err)
			return
		}

		// Construire le chemin complet de la photo
		photoPath := conf.OriginalsPath() + "/" + m.PhotoPath + "/" + m.PhotoName

		// Obtenir le chemin du sidecar
		sidecarPath := photoprism.GetSidecarPath(photoPath)

		// Sauvegarder les éditions dans le sidecar YAML
		if err := photoprism.SaveImageEditsToSidecar(sidecarPath, &req.SidecarData); err != nil {
			AbortSaveFailed(c)
			return
		}

		// Mettre à jour EditedAt dans la base de données
		if err := entity.Db().Model(m).Update("EditedAt", time.Now()).Error; err != nil {
			// on ignore l'erreur ici (warning silencieux)
		}

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

		conf := get.Config()

		uid := clean.UID(c.Param("uid"))
		if uid == "" {
			Abort(c, http.StatusBadRequest, i18n.ErrNotFound)
			return
		}

		m, err := query.PhotoByUID(uid)
		if err != nil {
			AbortEntityNotFound(c)
			return
		}

		if m.PhotoPath == "" || m.PhotoName == "" {
			Abort(c, http.StatusBadRequest, i18n.ErrNotFound)
			return
		}

		photoPath := conf.OriginalsPath() + "/" + m.PhotoPath + "/" + m.PhotoName
		sidecarPath := photoprism.GetSidecarPath(photoPath)

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

		conf := get.Config()

		uid := clean.UID(c.Param("uid"))
		if uid == "" {
			Abort(c, http.StatusBadRequest, i18n.ErrNotFound)
			return
		}

		m, err := query.PhotoByUID(uid)
		if err != nil {
			AbortEntityNotFound(c)
			return
		}

		if m.PhotoPath == "" || m.PhotoName == "" {
			Abort(c, http.StatusBadRequest, i18n.ErrDeleteFailed)
			return
		}

		photoPath := conf.OriginalsPath() + "/" + m.PhotoPath + "/" + m.PhotoName
		sidecarPath := photoprism.GetSidecarPath(photoPath)

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
