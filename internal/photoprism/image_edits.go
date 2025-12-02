package photoprism

import (
	"fmt"
	"image"
	"os"
	"path/filepath"

	"github.com/disintegration/imaging"
	"gopkg.in/yaml.v2"
)

// ImageEdits représente les modifications d'image stockées dans le sidecar YAML
type ImageEdits struct {
	Crop     CropEdits `yaml:"Crop,omitempty" json:"crop,omitempty"`
	Rotation int       `yaml:"Rotation,omitempty" json:"rotation,omitempty"`
	Flip     FlipEdits `yaml:"Flip,omitempty" json:"flip,omitempty"`
	EditedAt string    `yaml:"EditedAt,omitempty" json:"editedAt,omitempty"`
	EditedBy string    `yaml:"EditedBy,omitempty" json:"editedBy,omitempty"`
}

// CropEdits représente un crop avec coordonnées relatives (0-1)
type CropEdits struct {
	Left   float64 `yaml:"Left" json:"left"`
	Top    float64 `yaml:"Top" json:"top"`
	Width  float64 `yaml:"Width" json:"width"`
	Height float64 `yaml:"Height" json:"height"`
}

// FlipEdits représente les flip horizontal et vertical
type FlipEdits struct {
	Horizontal bool `yaml:"Horizontal" json:"horizontal"`
	Vertical   bool `yaml:"Vertical" json:"vertical"`
}

// IsCropped retourne true si un crop a été appliqué
func (c *CropEdits) IsCropped() bool {
	return c.Left != 0 || c.Top != 0 || c.Width != 1 || c.Height != 1
}

// IsFlipped retourne true si un flip a été appliqué
func (f *FlipEdits) IsFlipped() bool {
	return f.Horizontal || f.Vertical
}

// HasEdits retourne true si des modifications ont été appliquées
func (e *ImageEdits) HasEdits() bool {
	return e.Crop.IsCropped() || e.Rotation != 0 || e.Flip.IsFlipped()
}

// SaveImageEditsToSidecar ajoute les modifications d'image au fichier sidecar YAML existant
func SaveImageEditsToSidecar(yamlPath string, edits *ImageEdits) error {
	// Lire le contenu YAML existant
	var existingData map[string]interface{}

	// Vérifier si le fichier existe
	if _, err := os.Stat(yamlPath); err == nil {
		// Le fichier existe, le lire
		data, err := os.ReadFile(yamlPath)
		if err != nil {
			return fmt.Errorf("failed to read sidecar file: %w", err)
		}

		// Parser le YAML existant
		if err := yaml.Unmarshal(data, &existingData); err != nil {
			return fmt.Errorf("failed to parse existing YAML: %w", err)
		}
	} else {
		// Le fichier n'existe pas, créer une nouvelle map
		existingData = make(map[string]interface{})
	}

	// Créer la structure ImageEdits
	imageEditsMap := make(map[string]interface{})

	// Ajouter le crop seulement s'il est appliqué
	if edits.Crop.IsCropped() {
		imageEditsMap["Crop"] = map[string]interface{}{
			"Left":   edits.Crop.Left,
			"Top":    edits.Crop.Top,
			"Width":  edits.Crop.Width,
			"Height": edits.Crop.Height,
		}
	}

	// Ajouter la rotation seulement si différente de 0
	if edits.Rotation != 0 {
		imageEditsMap["Rotation"] = edits.Rotation
	}

	// Ajouter le flip seulement s'il est appliqué
	if edits.Flip.IsFlipped() {
		imageEditsMap["Flip"] = map[string]interface{}{
			"Horizontal": edits.Flip.Horizontal,
			"Vertical":   edits.Flip.Vertical,
		}
	}

	// Ajouter les métadonnées
	if edits.EditedAt != "" {
		imageEditsMap["EditedAt"] = edits.EditedAt
	}
	if edits.EditedBy != "" {
		imageEditsMap["EditedBy"] = edits.EditedBy
	}

	// Placer tout sous la clé "ImageEdits"
	existingData["ImageEdits"] = imageEditsMap

	// Sérialiser en YAML
	yamlData, err := yaml.Marshal(existingData)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	// Créer le répertoire si nécessaire
	dir := filepath.Dir(yamlPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Écrire le fichier
	if err := os.WriteFile(yamlPath, yamlData, 0644); err != nil {
		return fmt.Errorf("failed to write sidecar file: %w", err)
	}

	return nil
}

// LoadImageEditsFromSidecar charge les modifications d'image depuis le fichier sidecar YAML
func LoadImageEditsFromSidecar(yamlPath string) (*ImageEdits, error) {
	// Vérifier si le fichier existe
	if _, err := os.Stat(yamlPath); os.IsNotExist(err) {
		// Pas de sidecar, retourner des éditions par défaut (pas de modifications)
		return &ImageEdits{
			Crop: CropEdits{
				Left:   0,
				Top:    0,
				Width:  1,
				Height: 1,
			},
			Rotation: 0,
			Flip: FlipEdits{
				Horizontal: false,
				Vertical:   false,
			},
		}, nil
	}

	// Lire le fichier
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read sidecar file: %w", err)
	}

	// Parser le YAML dans une map générique
	var yamlData map[string]interface{}
	if err := yaml.Unmarshal(data, &yamlData); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Extraire les éditions
	edits := &ImageEdits{
		Crop: CropEdits{
			Left:   0,
			Top:    0,
			Width:  1,
			Height: 1,
		},
		Rotation: 0,
		Flip: FlipEdits{
			Horizontal: false,
			Vertical:   false,
		},
	}

	// Chercher les données sous la clé "ImageEdits"
	imageEditsData, hasImageEdits := yamlData["ImageEdits"].(map[interface{}]interface{})
	if !hasImageEdits {
		// Pas de données ImageEdits, retourner les valeurs par défaut
		return edits, nil
	}

	// Extraire Crop
	if cropData, ok := imageEditsData["Crop"].(map[interface{}]interface{}); ok {
		if left, ok := cropData["Left"].(float64); ok {
			edits.Crop.Left = left
		}
		if top, ok := cropData["Top"].(float64); ok {
			edits.Crop.Top = top
		}
		if width, ok := cropData["Width"].(float64); ok {
			edits.Crop.Width = width
		}
		if height, ok := cropData["Height"].(float64); ok {
			edits.Crop.Height = height
		}
	}

	// Extraire Rotation
	if rotation, ok := imageEditsData["Rotation"].(int); ok {
		edits.Rotation = rotation
	}

	// Extraire Flip
	if flipData, ok := imageEditsData["Flip"].(map[interface{}]interface{}); ok {
		if h, ok := flipData["Horizontal"].(bool); ok {
			edits.Flip.Horizontal = h
		}
		if v, ok := flipData["Vertical"].(bool); ok {
			edits.Flip.Vertical = v
		}
	}

	// Extraire métadonnées
	if editedAt, ok := imageEditsData["EditedAt"].(string); ok {
		edits.EditedAt = editedAt
	}
	if editedBy, ok := imageEditsData["EditedBy"].(string); ok {
		edits.EditedBy = editedBy
	}

	return edits, nil
}

// DeleteImageEditsFromSidecar supprime les modifications d'image du fichier sidecar YAML
func DeleteImageEditsFromSidecar(yamlPath string) error {
	// Vérifier si le fichier existe
	if _, err := os.Stat(yamlPath); os.IsNotExist(err) {
		// Pas de sidecar, rien à faire
		return nil
	}

	// Lire le fichier
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		return fmt.Errorf("failed to read sidecar file: %w", err)
	}

	// Parser le YAML
	var yamlData map[string]interface{}
	if err := yaml.Unmarshal(data, &yamlData); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Supprimer la section ImageEdits complète
	delete(yamlData, "ImageEdits")

	// Si le fichier est maintenant vide, le supprimer
	if len(yamlData) == 0 {
		return os.Remove(yamlPath)
	}

	// Sinon, réécrire le fichier sans les éditions
	newYamlData, err := yaml.Marshal(yamlData)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	if err := os.WriteFile(yamlPath, newYamlData, 0644); err != nil {
		return fmt.Errorf("failed to write sidecar file: %w", err)
	}

	return nil
}

// ApplyImageEdits applique les transformations d'édition à une image
func ApplyImageEdits(img image.Image, edits *ImageEdits) image.Image {
	if edits == nil {
		return img
	}

	// 1. Appliquer le crop en premier (coordonnées relatives 0-1)
	if edits.Crop.IsCropped() {
		bounds := img.Bounds()
		width := bounds.Dx()
		height := bounds.Dy()

		// Convertir les coordonnées relatives en pixels
		x0 := int(edits.Crop.Left * float64(width))
		y0 := int(edits.Crop.Top * float64(height))
		x1 := int((edits.Crop.Left + edits.Crop.Width) * float64(width))
		y1 := int((edits.Crop.Top + edits.Crop.Height) * float64(height))

		// S'assurer que les coordonnées sont dans les limites
		if x0 < 0 {
			x0 = 0
		}
		if y0 < 0 {
			y0 = 0
		}
		if x1 > width {
			x1 = width
		}
		if y1 > height {
			y1 = height
		}

		// Appliquer le crop
		cropRect := image.Rect(x0, y0, x1, y1)
		img = imaging.Crop(img, cropRect)
	}

	// 2. Appliquer la rotation (en degrés, sens horaire)
	if edits.Rotation != 0 {
		// Normaliser la rotation entre 0 et 360
		rotation := edits.Rotation % 360
		if rotation < 0 {
			rotation += 360
		}

		switch rotation {
		case 90:
			img = imaging.Rotate90(img)
		case 180:
			img = imaging.Rotate180(img)
		case 270:
			img = imaging.Rotate270(img)
		default:
			// Pour les rotations arbitraires, utiliser Rotate avec interpolation
			img = imaging.Rotate(img, float64(-rotation), image.Black)
		}
	}

	// 3. Appliquer les flips (après rotation)
	if edits.Flip.Horizontal {
		img = imaging.FlipH(img)
	}
	if edits.Flip.Vertical {
		img = imaging.FlipV(img)
	}

	return img
}

