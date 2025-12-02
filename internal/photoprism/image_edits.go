package photoprism

import (
	"fmt"
	"image"
	"os"
	"path/filepath"

	"github.com/disintegration/imaging"
	"gopkg.in/yaml.v2"
)

// ImageEdits represents image modifications stored in the YAML sidecar
type ImageEdits struct {
	Crop     CropEdits `yaml:"Crop,omitempty" json:"crop,omitempty"`
	Rotation int       `yaml:"Rotation,omitempty" json:"rotation,omitempty"`
	Flip     FlipEdits `yaml:"Flip,omitempty" json:"flip,omitempty"`
	EditedAt string    `yaml:"EditedAt,omitempty" json:"editedAt,omitempty"`
	EditedBy string    `yaml:"EditedBy,omitempty" json:"editedBy,omitempty"`
}

// CropEdits represents a crop with relative coordinates (0-1)
type CropEdits struct {
	Left   float64 `yaml:"Left" json:"left"`
	Top    float64 `yaml:"Top" json:"top"`
	Width  float64 `yaml:"Width" json:"width"`
	Height float64 `yaml:"Height" json:"height"`
}

// FlipEdits represents horizontal and vertical flip
type FlipEdits struct {
	Horizontal bool `yaml:"Horizontal" json:"horizontal"`
	Vertical   bool `yaml:"Vertical" json:"vertical"`
}

// IsCropped returns true if a crop has been applied
func (c *CropEdits) IsCropped() bool {
	return c.Left != 0 || c.Top != 0 || c.Width != 1 || c.Height != 1
}

// IsFlipped returns true if a flip has been applied
func (f *FlipEdits) IsFlipped() bool {
	return f.Horizontal || f.Vertical
}

// HasEdits returns true if modifications have been applied
func (e *ImageEdits) HasEdits() bool {
	return e.Crop.IsCropped() || e.Rotation != 0 || e.Flip.IsFlipped()
}

// SaveImageEditsToSidecar adds image modifications to the existing YAML sidecar file
func SaveImageEditsToSidecar(yamlPath string, edits *ImageEdits) error {
	// Read existing YAML content
	var existingData map[string]interface{}

	// Check if file exists
	if _, err := os.Stat(yamlPath); err == nil {
		// File exists, read it
		data, err := os.ReadFile(yamlPath)
		if err != nil {
			return fmt.Errorf("failed to read sidecar file: %w", err)
		}

		// Parse existing YAML
		if err := yaml.Unmarshal(data, &existingData); err != nil {
			return fmt.Errorf("failed to parse existing YAML: %w", err)
		}
	} else {
		// File doesn't exist, create new map
		existingData = make(map[string]interface{})
	}

	// Create ImageEdits structure
	imageEditsMap := make(map[string]interface{})

	// Add crop only if applied
	if edits.Crop.IsCropped() {
		imageEditsMap["Crop"] = map[string]interface{}{
			"Left":   edits.Crop.Left,
			"Top":    edits.Crop.Top,
			"Width":  edits.Crop.Width,
			"Height": edits.Crop.Height,
		}
	}

	// Add rotation only if different from 0
	if edits.Rotation != 0 {
		imageEditsMap["Rotation"] = edits.Rotation
	}

	// Add flip only if applied
	if edits.Flip.IsFlipped() {
		imageEditsMap["Flip"] = map[string]interface{}{
			"Horizontal": edits.Flip.Horizontal,
			"Vertical":   edits.Flip.Vertical,
		}
	}

	// Add metadata
	if edits.EditedAt != "" {
		imageEditsMap["EditedAt"] = edits.EditedAt
	}
	if edits.EditedBy != "" {
		imageEditsMap["EditedBy"] = edits.EditedBy
	}

	// Place everything under "ImageEdits" key
	existingData["ImageEdits"] = imageEditsMap

	// Serialize to YAML
	yamlData, err := yaml.Marshal(existingData)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	// Create directory if necessary
	dir := filepath.Dir(yamlPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(yamlPath, yamlData, 0644); err != nil {
		return fmt.Errorf("failed to write sidecar file: %w", err)
	}

	return nil
}

// LoadImageEditsFromSidecar loads image modifications from the YAML sidecar file
func LoadImageEditsFromSidecar(yamlPath string) (*ImageEdits, error) {
	// Check if file exists
	if _, err := os.Stat(yamlPath); os.IsNotExist(err) {
		// No sidecar, return default edits (no modifications)
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

	// Read file
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read sidecar file: %w", err)
	}

	// Parse YAML into generic map
	var yamlData map[string]interface{}
	if err := yaml.Unmarshal(data, &yamlData); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Extract edits
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

	// Look for data under "ImageEdits" key
	imageEditsData, hasImageEdits := yamlData["ImageEdits"].(map[interface{}]interface{})
	if !hasImageEdits {
		// No ImageEdits data, return default values
		return edits, nil
	}

	// Extract Crop
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

	// Extract Rotation
	if rotation, ok := imageEditsData["Rotation"].(int); ok {
		edits.Rotation = rotation
	}

	// Extract Flip
	if flipData, ok := imageEditsData["Flip"].(map[interface{}]interface{}); ok {
		if h, ok := flipData["Horizontal"].(bool); ok {
			edits.Flip.Horizontal = h
		}
		if v, ok := flipData["Vertical"].(bool); ok {
			edits.Flip.Vertical = v
		}
	}

	// Extract metadata
	if editedAt, ok := imageEditsData["EditedAt"].(string); ok {
		edits.EditedAt = editedAt
	}
	if editedBy, ok := imageEditsData["EditedBy"].(string); ok {
		edits.EditedBy = editedBy
	}

	return edits, nil
}

// DeleteImageEditsFromSidecar deletes image modifications from the YAML sidecar file
func DeleteImageEditsFromSidecar(yamlPath string) error {
	// Check if file exists
	if _, err := os.Stat(yamlPath); os.IsNotExist(err) {
		// No sidecar, nothing to do
		return nil
	}

	// Read file
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		return fmt.Errorf("failed to read sidecar file: %w", err)
	}

	// Parse YAML
	var yamlData map[string]interface{}
	if err := yaml.Unmarshal(data, &yamlData); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Delete complete ImageEdits section
	delete(yamlData, "ImageEdits")

	// If file is now empty, delete it
	if len(yamlData) == 0 {
		return os.Remove(yamlPath)
	}

	// Otherwise, rewrite file without edits
	newYamlData, err := yaml.Marshal(yamlData)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	if err := os.WriteFile(yamlPath, newYamlData, 0644); err != nil {
		return fmt.Errorf("failed to write sidecar file: %w", err)
	}

	return nil
}

// ApplyImageEdits applies edit transformations to an image
func ApplyImageEdits(img image.Image, edits *ImageEdits) image.Image {
	if edits == nil {
		return img
	}

	// 1. Apply crop first (relative coordinates 0-1)
	if edits.Crop.IsCropped() {
		bounds := img.Bounds()
		width := bounds.Dx()
		height := bounds.Dy()

		// Convert relative coordinates to pixels
		x0 := int(edits.Crop.Left * float64(width))
		y0 := int(edits.Crop.Top * float64(height))
		x1 := int((edits.Crop.Left + edits.Crop.Width) * float64(width))
		y1 := int((edits.Crop.Top + edits.Crop.Height) * float64(height))

		// Ensure coordinates are within bounds
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

		// Apply crop
		cropRect := image.Rect(x0, y0, x1, y1)
		img = imaging.Crop(img, cropRect)
	}

	// 2. Apply rotation (in degrees, clockwise)
	if edits.Rotation != 0 {
		// Normalize rotation between 0 and 360
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
			// For arbitrary rotations, use Rotate with interpolation
			img = imaging.Rotate(img, float64(-rotation), image.Black)
		}
	}

	// 3. Apply flips (after rotation)
	if edits.Flip.Horizontal {
		img = imaging.FlipH(img)
	}
	if edits.Flip.Vertical {
		img = imaging.FlipV(img)
	}

	return img
}

