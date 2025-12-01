<template>
  <v-dialog :model-value="visible" fullscreen persistent theme="dark" class="p-dialog p-image-editor" @keydown.esc.exact.stop="onClose">
    <v-card class="image-editor-container">
      <!-- Toolbar -->
      <v-toolbar dark color="black" density="compact">
        <v-btn icon @click="onClose">
          <v-icon>mdi-close</v-icon>
        </v-btn>

        <v-toolbar-title>{{ $gettext("Edit Image") }}</v-toolbar-title>

        <v-spacer></v-spacer>
      </v-toolbar>
      <v-btn color="primary" variant="flat" @click="onSave">
        <v-icon start>mdi-content-save</v-icon>
        {{ $gettext("Save") }}
      </v-btn>
      <!-- Image originale avec overlays -->
      <v-card-text class="editor-content pa-0">
        <div class="image-wrapper" ref="imageWrapper">
          <!-- Canvas de prévisualisation -->
          <canvas v-if="isImageLoaded" ref="previewCanvas" class="preview-canvas"></canvas>

          <!-- Image originale (cachée, utilisée comme source) -->
          <img v-if="imageUrl" ref="imageElement" :src="imageUrl" class="editor-image-hidden" @load="onImageLoad" />

          <!-- Overlay de crop -->
          <div v-if="cropMode && isImageLoaded" class="crop-overlay" :style="cropOverlayStyle">
            <!-- Rectangle de crop -->
            <div class="crop-rect" :style="cropRectStyle" @mousedown.stop="startCropDrag">
              <!-- Poignées de redimensionnement -->
              <div class="crop-handle crop-handle-nw" @mousedown.stop="startCropResize('nw', $event)"></div>
              <div class="crop-handle crop-handle-ne" @mousedown.stop="startCropResize('ne', $event)"></div>
              <div class="crop-handle crop-handle-sw" @mousedown.stop="startCropResize('sw', $event)"></div>
              <div class="crop-handle crop-handle-se" @mousedown.stop="startCropResize('se', $event)"></div>
            </div>
          </div>
        </div>
      </v-card-text>

      <!-- Sidebar avec options -->
      <v-navigation-drawer permanent location="right" width="320" class="editor-sidebar">
        <v-card flat>
          <v-card-title>{{ $gettext("Edit Options") }}</v-card-title>

          <v-card-text>
            <!-- Bouton Crop -->
            <v-btn block :color="cropMode ? 'primary' : 'default'" :variant="cropMode ? 'flat' : 'outlined'" class="mb-4" @click="toggleCropMode">
              <v-icon start>mdi-crop</v-icon>
              {{ cropMode ? $gettext("Cropping...") : $gettext("Crop Image") }}
            </v-btn>

            <!-- Boutons Annuler / Valider (visibles seulement en mode crop) -->
            <v-expand-transition>
              <div v-if="cropMode" class="mb-4">
                <v-btn-group divided density="compact" variant="outlined" class="d-flex">
                  <v-btn @click="cancelCrop" color="error" style="flex: 1">
                    <v-icon>mdi-close</v-icon>
                    <span class="ml-1">{{ $gettext("Cancel") }}</span>
                  </v-btn>
                  <v-btn @click="applyCrop" color="success" style="flex: 1">
                    <v-icon>mdi-check</v-icon>
                    <span class="ml-1">{{ $gettext("Apply") }}</span>
                  </v-btn>
                </v-btn-group>
              </div>
            </v-expand-transition>

            <v-divider class="my-4"></v-divider>

            <!-- Rotation -->
            <div class="mb-4">
              <div class="text-subtitle-2 mb-2">{{ $gettext("Rotation") }}</div>
              <v-btn-group divided density="compact" variant="outlined" class="d-flex">
                <v-btn @click="rotate(-90)" :disabled="!isImageLoaded" style="flex: 1">
                  <v-icon>mdi-rotate-left</v-icon>
                  <span class="ml-1">-90°</span>
                </v-btn>
                <v-btn @click="rotate(90)" :disabled="!isImageLoaded" style="flex: 1">
                  <v-icon>mdi-rotate-right</v-icon>
                  <span class="ml-1">+90°</span>
                </v-btn>
              </v-btn-group>
            </div>

            <!-- Flip -->
            <div class="mb-4">
              <div class="text-subtitle-2 mb-2">{{ $gettext("Flip") }}</div>
              <v-btn-group divided density="compact" variant="outlined" class="d-flex">
                <v-btn @click="flip('horizontal')" :disabled="!isImageLoaded" style="flex: 1">
                  <v-icon>mdi-flip-horizontal</v-icon>
                  <span class="ml-1">{{ $gettext("Horizontal") }}</span>
                </v-btn>
                <v-btn @click="flip('vertical')" :disabled="!isImageLoaded" style="flex: 1">
                  <v-icon>mdi-flip-vertical</v-icon>
                  <span class="ml-1">{{ $gettext("Vertical") }}</span>
                </v-btn>
              </v-btn-group>
            </div>

            <!-- Reset -->
            <v-divider class="my-4"></v-divider>

            <v-btn block variant="outlined" color="warning" @click="reset">
              <v-icon start>mdi-refresh</v-icon>
              {{ $gettext("Reset All") }}
            </v-btn>

            <!-- Info sur les modifications -->
            <v-divider class="my-4"></v-divider>

            <div class="text-caption text-medium-emphasis">
              <div class="mb-2">
                <strong>{{ $gettext("Crop:") }}</strong>
                {{ cropInfo }}
              </div>
              <div class="mb-2">
                <strong>{{ $gettext("Rotation:") }}</strong>
                {{ edits.rotation }}°
              </div>
              <div class="mb-2">
                <strong>{{ $gettext("Flip:") }}</strong>
                {{ flipInfo }}
              </div>
            </div>

            <!-- Info sur le sidecar -->
            <v-alert type="info" variant="tonal" density="compact" class="mt-4">
              {{ $gettext("Changes will be saved in a YAML sidecar file without modifying the original image.") }}
            </v-alert>

            <!-- Debug YAML en temps réel -->
            <v-divider class="my-4"></v-divider>

            <div class="yaml-debug">
              <div class="d-flex align-center mb-2">
                <span class="text-subtitle-2 flex-grow-1">{{ $gettext("YAML Preview") }}</span>
                <v-btn icon size="x-small" variant="text" @click="copyYamlToClipboard" :title="$gettext('Copy to clipboard')">
                  <v-icon size="small">mdi-content-copy</v-icon>
                </v-btn>
              </div>
              <pre class="yaml-preview">{{ yamlPreview }}</pre>
            </div>
          </v-card-text>
        </v-card>
      </v-navigation-drawer>
    </v-card>
  </v-dialog>
</template>

<script>
export default {
  name: "PImageEditor",
  components: {},
  props: {
    visible: {
      type: Boolean,
      default: false,
    },
    model: {
      type: Object,
      default: null,
    },
  },
  emits: ["close", "save"],
  data() {
    return {
      imageUrl: "",
      isImageLoaded: false,
      // État des éditions (source de vérité unique - YAML)
      edits: {
        crop: {
          left: 0,
          top: 0,
          width: 1,
          height: 1,
        },
        rotation: 0,
        flip: {
          horizontal: false,
          vertical: false,
        },
      },
      // UI state pour le crop
      cropMode: false,
      cropRect: {
        x: 0,
        y: 0,
        width: 0,
        height: 0,
      },
      savedCrop: null, // Sauvegarde du crop avant d'entrer en mode crop
      cropDragging: false,
      cropResizing: false,
      cropResizeHandle: null,
      cropDragStart: { x: 0, y: 0 },
      cropRectStart: { x: 0, y: 0, width: 0, height: 0 },
      imageRect: { x: 0, y: 0, width: 0, height: 0 },
      originalImageSize: { width: 0, height: 0 },
      previewUpdateTimer: null,
    };
  },
  computed: {
    cropOverlayStyle() {
      if (!this.imageRect.width) return {};

      return {
        position: "absolute",
        left: `${this.imageRect.x}px`,
        top: `${this.imageRect.y}px`,
        width: `${this.imageRect.width}px`,
        height: `${this.imageRect.height}px`,
      };
    },
    cropRectStyle() {
      return {
        left: `${this.cropRect.x}px`,
        top: `${this.cropRect.y}px`,
        width: `${this.cropRect.width}px`,
        height: `${this.cropRect.height}px`,
      };
    },
    cropInfo() {
      if (this.edits.crop.width === 1 && this.edits.crop.height === 1) {
        return this.$gettext("No crop");
      }
      const w = Math.round(this.edits.crop.width * 100);
      const h = Math.round(this.edits.crop.height * 100);
      return `${w}% × ${h}%`;
    },
    flipInfo() {
      const flips = [];
      if (this.edits.flip.horizontal) flips.push(this.$gettext("H"));
      if (this.edits.flip.vertical) flips.push(this.$gettext("V"));
      return flips.length > 0 ? flips.join(", ") : this.$gettext("None");
    },
    yamlPreview() {
      if (!this.isImageLoaded) {
        return "# Waiting for image to load...";
      }

      // Formater en YAML
      let yaml = "# Sidecar YAML Preview\n";
      yaml += "crop:\n";
      yaml += `  left: ${this.edits.crop.left.toFixed(6)}\n`;
      yaml += `  top: ${this.edits.crop.top.toFixed(6)}\n`;
      yaml += `  width: ${this.edits.crop.width.toFixed(6)}\n`;
      yaml += `  height: ${this.edits.crop.height.toFixed(6)}\n`;
      yaml += `rotation: ${this.edits.rotation}\n`;
      yaml += "flip:\n";
      yaml += `  horizontal: ${this.edits.flip.horizontal}\n`;
      yaml += `  vertical: ${this.edits.flip.vertical}\n`;
      yaml += `editedAt: "${new Date().toISOString()}"\n`;
      yaml += `editedBy: "${this.$session?.user?.Name || "user"}"\n`;

      return yaml;
    },
  },
  watch: {
    "visible"(val) {
      if (val && this.model) {
        this.loadImage();
      }
    },
    "cropMode"(val) {
      if (val) {
        this.$nextTick(() => {
          this.updateImageRect();
        });
      }
    },
    "edits.crop": {
      handler() {
        if (this.isImageLoaded) {
          this.schedulePreviewUpdate();
        }
      },
      deep: true,
    },
    "edits.rotation"() {
      if (this.isImageLoaded) {
        this.schedulePreviewUpdate();
      }
    },
    "edits.flip": {
      handler() {
        if (this.isImageLoaded) {
          this.schedulePreviewUpdate();
        }
      },
      deep: true,
    },
  },
  mounted() {
    if (this.visible && this.model) {
      this.loadImage();
    }

    // Event listeners pour le crop
    document.addEventListener("mousemove", this.onCropMouseMove);
    document.addEventListener("mouseup", this.onCropMouseUp);
  },
  beforeUnmount() {
    document.removeEventListener("mousemove", this.onCropMouseMove);
    document.removeEventListener("mouseup", this.onCropMouseUp);
  },
  methods: {
    loadImage() {
      if (!this.model) return;

      // Charger l'URL de l'image originale en haute résolution
      this.imageUrl = this.model.thumbnailUrl("fit_2048");

      // Réinitialiser
      this.reset();
      this.isImageLoaded = false;
    },

    onImageLoad() {
      const img = this.$refs.imageElement;
      if (!img) return;

      this.originalImageSize = {
        width: img.naturalWidth,
        height: img.naturalHeight,
      };

      this.isImageLoaded = true;

      // Mettre à jour la prévisualisation
      this.$nextTick(() => {
        this.updatePreview();
        this.updateImageRect();
      });
    },

    updateImageRect() {
      const canvas = this.$refs.previewCanvas;
      if (!canvas) return;

      const rect = canvas.getBoundingClientRect();
      const wrapper = this.$refs.imageWrapper;
      const wrapperRect = wrapper.getBoundingClientRect();

      this.imageRect = {
        x: rect.left - wrapperRect.left,
        y: rect.top - wrapperRect.top,
        width: rect.width,
        height: rect.height,
      };
    },

    toggleCropMode() {
      this.cropMode = !this.cropMode;

      if (this.cropMode) {
        // En mode crop, on doit temporairement afficher l'image complète
        // et positionner le rectangle selon le crop actuel du YAML

        // Sauvegarder le crop actuel pour pouvoir le restaurer si on annule
        this.savedCrop = { ...this.edits.crop };

        // Temporairement réinitialiser le crop pour afficher l'image complète
        this.edits.crop = {
          left: 0,
          top: 0,
          width: 1,
          height: 1,
        };

        // Forcer la mise à jour du canvas
        this.$nextTick(() => {
          this.updatePreview();

          // Attendre que le canvas soit redimensionné
          this.$nextTick(() => {
            this.updateImageRect();

            // Positionner le rectangle selon le crop sauvegardé
            if (this.savedCrop.width < 1 || this.savedCrop.height < 1) {
              // Un crop existe, le restaurer comme position du rectangle
              this.cropRect = {
                x: this.savedCrop.left * this.imageRect.width,
                y: this.savedCrop.top * this.imageRect.height,
                width: this.savedCrop.width * this.imageRect.width,
                height: this.savedCrop.height * this.imageRect.height,
              };
            } else {
              // Pas de crop, initialiser à 80% centré
              const margin = 0.1;
              this.cropRect = {
                x: this.imageRect.width * margin,
                y: this.imageRect.height * margin,
                width: this.imageRect.width * (1 - 2 * margin),
                height: this.imageRect.height * (1 - 2 * margin),
              };
            }
          });
        });
      }
    },

    cancelCrop() {
      // Restaurer le crop sauvegardé
      if (this.savedCrop) {
        this.edits.crop = { ...this.savedCrop };
        this.savedCrop = null;
      }

      this.cropMode = false;

      // Rafraîchir l'affichage avec le crop original
      this.$nextTick(() => {
        this.updatePreview();
      });
    },

    applyCrop() {
      if (!this.cropRect.width || !this.cropRect.height) {
        this.$notify.warn(this.$gettext("Please select a crop area"));
        return;
      }

      // Mettre à jour le YAML (source de vérité)
      this.edits.crop = {
        left: this.cropRect.x / this.imageRect.width,
        top: this.cropRect.y / this.imageRect.height,
        width: this.cropRect.width / this.imageRect.width,
        height: this.cropRect.height / this.imageRect.height,
      };

      // Nettoyer la sauvegarde
      this.savedCrop = null;

      // Sortir du mode crop
      this.cropMode = false;

      // Forcer la mise à jour de la prévisualisation
      this.$nextTick(() => {
        this.updatePreview();
      });

      this.$notify.success(this.$gettext("Crop applied"));
    },

    rotate(angle) {
      this.edits.rotation = (this.edits.rotation + angle) % 360;
      if (this.edits.rotation < 0) this.edits.rotation += 360;
    },

    flip(direction) {
      if (direction === "horizontal") {
        this.edits.flip.horizontal = !this.edits.flip.horizontal;
      } else if (direction === "vertical") {
        this.edits.flip.vertical = !this.edits.flip.vertical;
      }
    },

    reset() {
      this.edits = {
        crop: {
          left: 0,
          top: 0,
          width: 1,
          height: 1,
        },
        rotation: 0,
        flip: {
          horizontal: false,
          vertical: false,
        },
      };
      this.cropMode = false;
      this.cropRect = { x: 0, y: 0, width: 0, height: 0 };

      this.$notify.info(this.$gettext("All changes reset"));
    },

    // Gestion du drag/resize du crop
    startCropDrag(event) {
      this.cropDragging = true;
      this.cropDragStart = {
        x: event.clientX,
        y: event.clientY,
      };
      this.cropRectStart = { ...this.cropRect };
      event.preventDefault();
    },

    startCropResize(handle, event) {
      this.cropResizing = true;
      this.cropResizeHandle = handle;
      this.cropDragStart = {
        x: event.clientX,
        y: event.clientY,
      };
      this.cropRectStart = { ...this.cropRect };
      event.preventDefault();
    },

    onCropMouseMove(event) {
      if (this.cropDragging) {
        const dx = event.clientX - this.cropDragStart.x;
        const dy = event.clientY - this.cropDragStart.y;

        this.cropRect.x = Math.max(0, Math.min(this.imageRect.width - this.cropRect.width, this.cropRectStart.x + dx));
        this.cropRect.y = Math.max(0, Math.min(this.imageRect.height - this.cropRect.height, this.cropRectStart.y + dy));
      } else if (this.cropResizing) {
        const dx = event.clientX - this.cropDragStart.x;
        const dy = event.clientY - this.cropDragStart.y;

        switch (this.cropResizeHandle) {
          case "nw":
            const newWidthNW = this.cropRectStart.width - dx;
            const newHeightNW = this.cropRectStart.height - dy;
            if (newWidthNW >= 50 && this.cropRectStart.x + dx >= 0) {
              this.cropRect.x = this.cropRectStart.x + dx;
              this.cropRect.width = newWidthNW;
            }
            if (newHeightNW >= 50 && this.cropRectStart.y + dy >= 0) {
              this.cropRect.y = this.cropRectStart.y + dy;
              this.cropRect.height = newHeightNW;
            }
            break;

          case "ne":
            this.cropRect.width = Math.max(50, Math.min(this.imageRect.width - this.cropRect.x, this.cropRectStart.width + dx));
            const newHeightNE = this.cropRectStart.height - dy;
            if (newHeightNE >= 50 && this.cropRectStart.y + dy >= 0) {
              this.cropRect.y = this.cropRectStart.y + dy;
              this.cropRect.height = newHeightNE;
            }
            break;

          case "sw":
            const newWidthSW = this.cropRectStart.width - dx;
            if (newWidthSW >= 50 && this.cropRectStart.x + dx >= 0) {
              this.cropRect.x = this.cropRectStart.x + dx;
              this.cropRect.width = newWidthSW;
            }
            this.cropRect.height = Math.max(50, Math.min(this.imageRect.height - this.cropRect.y, this.cropRectStart.height + dy));
            break;

          case "se":
            this.cropRect.width = Math.max(50, Math.min(this.imageRect.width - this.cropRect.x, this.cropRectStart.width + dx));
            this.cropRect.height = Math.max(50, Math.min(this.imageRect.height - this.cropRect.y, this.cropRectStart.height + dy));
            break;
        }
      }
    },

    onCropMouseUp() {
      this.cropDragging = false;
      this.cropResizing = false;
      this.cropResizeHandle = null;
    },

    copyYamlToClipboard() {
      if (navigator.clipboard && this.yamlPreview) {
        navigator.clipboard
          .writeText(this.yamlPreview)
          .then(() => {
            this.$notify.success(this.$gettext("YAML copied to clipboard"));
          })
          .catch((err) => {
            console.error("Failed to copy:", err);
            this.$notify.error(this.$gettext("Failed to copy"));
          });
      }
    },

    schedulePreviewUpdate() {
      if (this.previewUpdateTimer) {
        clearTimeout(this.previewUpdateTimer);
      }
      this.previewUpdateTimer = setTimeout(() => {
        this.updatePreview();
      }, 100);
    },

    updatePreview() {
      const img = this.$refs.imageElement;
      const canvas = this.$refs.previewCanvas;

      if (!img || !canvas || !this.isImageLoaded) {
        return;
      }

      try {
        const ctx = canvas.getContext("2d");

        // Calculer la zone source basée sur l'image ORIGINALE entière
        let sourceX = this.edits.crop.left * img.naturalWidth;
        let sourceY = this.edits.crop.top * img.naturalHeight;
        let sourceWidth = this.edits.crop.width * img.naturalWidth;
        let sourceHeight = this.edits.crop.height * img.naturalHeight;

        // Calculer les dimensions après rotation
        let displayWidth = sourceWidth;
        let displayHeight = sourceHeight;

        if (this.edits.rotation === 90 || this.edits.rotation === 270) {
          [displayWidth, displayHeight] = [displayHeight, displayWidth];
        }

        // Calculer l'échelle pour s'adapter au wrapper
        const wrapper = this.$refs.imageWrapper;
        const maxWidth = wrapper.clientWidth - 40;
        const maxHeight = wrapper.clientHeight - 40;
        const scale = Math.min(1, maxWidth / displayWidth, maxHeight / displayHeight);

        // Ajuster les dimensions du canvas
        canvas.width = displayWidth * scale;
        canvas.height = displayHeight * scale;

        // Clear canvas
        ctx.clearRect(0, 0, canvas.width, canvas.height);

        // Sauvegarder le contexte
        ctx.save();

        // Se déplacer au centre
        ctx.translate(canvas.width / 2, canvas.height / 2);

        // Appliquer la rotation
        ctx.rotate((this.edits.rotation * Math.PI) / 180);

        // Appliquer le flip
        ctx.scale(this.edits.flip.horizontal ? -1 : 1, this.edits.flip.vertical ? -1 : 1);

        // Dessiner la portion croppée de l'image originale
        const drawWidth = sourceWidth * scale;
        const drawHeight = sourceHeight * scale;

        ctx.drawImage(
          img,
          sourceX,
          sourceY,
          sourceWidth,
          sourceHeight, // Source (crop de l'original)
          -drawWidth / 2,
          -drawHeight / 2,
          drawWidth,
          drawHeight // Destination (centré)
        );

        // Restaurer le contexte
        ctx.restore();

        // Mettre à jour imageRect pour l'overlay de crop
        // Important: attendre que le canvas soit rendu
        this.$nextTick(() => {
          const canvasRect = canvas.getBoundingClientRect();
          const wrapperRect = wrapper.getBoundingClientRect();

          this.imageRect = {
            x: canvasRect.left - wrapperRect.left,
            y: canvasRect.top - wrapperRect.top,
            width: canvasRect.width,
            height: canvasRect.height,
          };
        });
      } catch (error) {
        console.error("Error updating preview:", error);
      }
    },

    onClose() {
      this.$emit("close");
    },

    onSave() {
      if (!this.isImageLoaded) {
        this.$notify.error(this.$gettext("Image not loaded"));
        return;
      }

      try {
        // Ajouter les métadonnées
        const sidecarData = {
          ...this.edits,
          editedAt: new Date().toISOString(),
          editedBy: this.$session?.user?.Name || "user",
        };

        // Émettre l'événement save avec les données du YAML
        this.$emit("save", {
          model: this.model,
          sidecarData: sidecarData,
        });

        this.$notify.success(this.$gettext("Changes saved to sidecar file"));
      } catch (error) {
        console.error("Error saving changes:", error);
        this.$notify.error(this.$gettext("Failed to save changes"));
      }
    },
  },
};
</script>

<style lang="scss">
.p-image-editor {
  .image-editor-container {
    height: 100vh;
    display: flex;
    flex-direction: column;
    background: #1e1e1e;
  }

  .editor-content {
    flex: 1;
    overflow: hidden;
    position: relative;
    background: #000;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .image-wrapper {
    width: 100%;
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 20px;
    position: relative;
  }

  .preview-canvas {
    display: block;
    max-width: 100%;
    max-height: 100%;
  }

  .editor-image-hidden {
    display: none;
  }

  .crop-overlay {
    pointer-events: none;
    z-index: 10;
  }

  .crop-rect {
    position: absolute;
    border: 2px solid #fff;
    box-shadow: 0 0 0 9999px rgba(0, 0, 0, 0.5);
    cursor: move;
    pointer-events: all;

    &::before {
      content: "";
      position: absolute;
      top: 50%;
      left: 50%;
      transform: translate(-50%, -50%);
      width: 20px;
      height: 20px;
      border: 2px solid #fff;
      border-radius: 50%;
      background: rgba(255, 255, 255, 0.2);
    }
  }

  .crop-handle {
    position: absolute;
    width: 16px;
    height: 16px;
    background: #fff;
    border: 2px solid #000;
    border-radius: 50%;

    &.crop-handle-nw {
      top: -8px;
      left: -8px;
      cursor: nw-resize;
    }

    &.crop-handle-ne {
      top: -8px;
      right: -8px;
      cursor: ne-resize;
    }

    &.crop-handle-sw {
      bottom: -8px;
      left: -8px;
      cursor: sw-resize;
    }

    &.crop-handle-se {
      bottom: -8px;
      right: -8px;
      cursor: se-resize;
    }
  }

  .editor-sidebar {
    background: #1e1e1e;
  }

  .yaml-debug {
    margin-top: 16px;
  }

  .yaml-preview {
    background: #0d0d0d;
    border: 1px solid #333;
    border-radius: 4px;
    padding: 12px;
    font-family: "Courier New", Courier, monospace;
    font-size: 11px;
    line-height: 1.4;
    color: #a0a0a0;
    overflow-x: auto;
    max-height: 300px;
    overflow-y: auto;
    white-space: pre;

    &::-webkit-scrollbar {
      width: 8px;
      height: 8px;
    }

    &::-webkit-scrollbar-track {
      background: #1a1a1a;
    }

    &::-webkit-scrollbar-thumb {
      background: #444;
      border-radius: 4px;

      &:hover {
        background: #555;
      }
    }
  }
}
</style>
