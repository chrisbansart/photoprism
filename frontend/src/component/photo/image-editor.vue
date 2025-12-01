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

        <v-btn color="primary" variant="flat" @click="onSave">
          <v-icon start>mdi-content-save</v-icon>
          {{ $gettext("Save") }}
        </v-btn>
      </v-toolbar>

      <!-- Image avec rotation (sans crop temporairement) -->
      <v-card-text class="editor-content pa-0">
        <div class="image-wrapper">
          <img v-if="imageUrl" ref="imageElement" :src="imageUrl" :style="imageStyle" class="editor-image" @load="onImageLoad" />
        </div>
      </v-card-text>

      <!-- Sidebar avec options -->
      <v-navigation-drawer permanent location="right" width="320" class="editor-sidebar">
        <v-card flat>
          <v-card-title>{{ $gettext("Crop Options") }}</v-card-title>

          <v-card-text>
            <!-- Aspect Ratio (désactivé temporairement) -->
            <!--
            <v-select
              v-model="aspectRatio"
              :label="$gettext('Aspect Ratio')"
              :items="aspectRatioOptions"
              density="compact"
              variant="outlined"
              class="mb-4"
            ></v-select>
            -->

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
                <v-btn @click="flip(true, false)" :disabled="!isImageLoaded" style="flex: 1">
                  <v-icon>mdi-flip-horizontal</v-icon>
                  <span class="ml-1">{{ $gettext("Horizontal") }}</span>
                </v-btn>
                <v-btn @click="flip(false, true)" :disabled="!isImageLoaded" style="flex: 1">
                  <v-icon>mdi-flip-vertical</v-icon>
                  <span class="ml-1">{{ $gettext("Vertical") }}</span>
                </v-btn>
              </v-btn-group>
            </div>

            <!-- Zoom (désactivé temporairement) -->
            <!--
            <div class="mb-4">
              <div class="text-subtitle-2 mb-2">{{ $gettext('Zoom') }}</div>
              <v-slider
                v-model="zoom"
                :min="0.1"
                :max="3"
                :step="0.1"
                :disabled="!isImageLoaded"
                thumb-label
                density="compact"
                @update:model-value="onZoomChange"
              >
                <template #append>
                  <v-btn icon size="small" variant="text" :disabled="!isImageLoaded" @click="resetZoom">
                    <v-icon>mdi-restore</v-icon>
                  </v-btn>
                </template>
              </v-slider>
            </div>
            -->

            <!-- Preview Toggle -->
            <v-divider class="my-4"></v-divider>

            <v-switch v-model="showPreview" :label="$gettext('Show preview with changes')" color="primary" density="compact" hide-details></v-switch>

            <v-expand-transition>
              <div v-if="showPreview" class="mt-4">
                <canvas ref="previewCanvas" class="preview-canvas" @click="downloadPreview"></canvas>
                <div class="text-caption text-center mt-2 text-medium-emphasis">
                  {{ $gettext("Click to download preview") }}
                </div>
              </div>
            </v-expand-transition>

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
                {{ rotationAngle }}°
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
// import { Cropper } from 'vue-advanced-cropper';
// import 'vue-advanced-cropper/dist/style.css';

export default {
  name: "PImageEditor",
  components: {
    // Cropper
  },
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
      aspectRatio: "free",
      aspectRatioOptions: [
        { value: "free", title: this.$gettext("Free") },
        { value: "1:1", title: "1:1 " + this.$gettext("(Square)") },
        { value: "4:3", title: "4:3" },
        { value: "16:9", title: "16:9" },
        { value: "3:2", title: "3:2" },
        { value: "2:3", title: "2:3 " + this.$gettext("(Portrait)") },
      ],
      zoom: 1,
      rotationAngle: 0,
      flipHorizontal: false,
      flipVertical: false,
      coordinates: null,
      originalImageSize: { width: 0, height: 0 },
      isImageLoaded: false, // Nouveau flag pour savoir si l'image est chargée
      showPreview: false,
      previewUpdateTimer: null,
    };
  },
  computed: {
    imageStyle() {
      let transform = "";

      // Rotation
      if (this.rotationAngle !== 0) {
        transform += `rotate(${this.rotationAngle}deg) `;
      }

      // Flip
      const scaleX = this.flipHorizontal ? -1 : 1;
      const scaleY = this.flipVertical ? -1 : 1;
      if (scaleX !== 1 || scaleY !== 1) {
        transform += `scale(${scaleX}, ${scaleY})`;
      }

      return {
        transform: transform.trim(),
      };
    },
    cropInfo() {
      return this.$gettext("Crop disabled (debug mode)");
    },
    flipInfo() {
      const flips = [];
      if (this.flipHorizontal) flips.push(this.$gettext("H"));
      if (this.flipVertical) flips.push(this.$gettext("V"));
      return flips.length > 0 ? flips.join(", ") : this.$gettext("None");
    },
    yamlPreview() {
      if (!this.isImageLoaded) {
        return "# Waiting for image to load...";
      }

      // Formater en YAML
      let yaml = "# Sidecar YAML Preview\n";
      yaml += "crop:\n";
      yaml += "  left: 0.000000  # Disabled in debug mode\n";
      yaml += "  top: 0.000000\n";
      yaml += "  width: 1.000000\n";
      yaml += "  height: 1.000000\n";
      yaml += `rotation: ${this.rotationAngle}\n`;
      yaml += "flip:\n";
      yaml += `  horizontal: ${this.flipHorizontal}\n`;
      yaml += `  vertical: ${this.flipVertical}\n`;

      if (this.aspectRatio !== "free") {
        yaml += `aspectRatio: "${this.aspectRatio}"\n`;
      }

      yaml += `editedAt: "${new Date().toISOString()}"\n`;
      yaml += `editedBy: "${this.$session?.user?.Name || "user"}"\n`;

      return yaml;
    },
  },
  watch: {
    visible(val) {
      if (val && this.model) {
        this.loadImage();
      }
    },
    showPreview(val) {
      if (val) {
        this.$nextTick(() => {
          this.updatePreview();
        });
      }
    },
    rotationAngle() {
      if (this.showPreview && this.isImageLoaded) {
        this.schedulePreviewUpdate();
      }
    },
    flipHorizontal() {
      if (this.showPreview && this.isImageLoaded) {
        this.schedulePreviewUpdate();
      }
    },
    flipVertical() {
      if (this.showPreview && this.isImageLoaded) {
        this.schedulePreviewUpdate();
      }
    },
  },
  mounted() {
    if (this.visible && this.model) {
      this.loadImage();
    }
  },
  methods: {
    loadImage() {
      if (!this.model) return;

      // Charger l'URL de l'image en haute résolution
      this.imageUrl = this.model.thumbnailUrl("fit_2048");

      // Réinitialiser les valeurs
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

      // Mettre à jour l'aperçu si activé
      if (this.showPreview) {
        this.$nextTick(() => {
          this.updatePreview();
        });
      }
    },

    rotate(angle) {
      this.rotationAngle = (this.rotationAngle + angle) % 360;
      if (this.rotationAngle < 0) this.rotationAngle += 360;
    },

    flip(horizontal, vertical) {
      if (horizontal) this.flipHorizontal = !this.flipHorizontal;
      if (vertical) this.flipVertical = !this.flipVertical;
    },

    onZoomChange(value) {
      // Désactivé temporairement
      console.log("Zoom:", value);
    },

    resetZoom() {
      this.zoom = 1;
    },

    reset() {
      this.aspectRatio = "free";
      this.zoom = 1;
      this.rotationAngle = 0;
      this.flipHorizontal = false;
      this.flipVertical = false;
    },

    onClose() {
      this.$emit("close");
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
      // Débounce pour éviter trop de mises à jour
      if (this.previewUpdateTimer) {
        clearTimeout(this.previewUpdateTimer);
      }
      this.previewUpdateTimer = setTimeout(() => {
        this.updatePreview();
      }, 300);
    },

    updatePreview() {
      const img = this.$refs.imageElement;
      const canvas = this.$refs.previewCanvas;

      if (!img || !canvas || !this.isImageLoaded) {
        return;
      }

      try {
        const ctx = canvas.getContext("2d");

        // Calculer les dimensions en tenant compte de la rotation
        let width = img.naturalWidth;
        let height = img.naturalHeight;

        // Échanger largeur et hauteur si rotation de 90 ou 270 degrés
        if (this.rotationAngle === 90 || this.rotationAngle === 270) {
          [width, height] = [height, width];
        }

        // Redimensionner pour tenir dans la sidebar
        const maxWidth = 280;
        const scale = Math.min(1, maxWidth / width);

        canvas.width = width * scale;
        canvas.height = height * scale;

        // Sauvegarder le contexte
        ctx.save();

        // Centrer l'origine de transformation
        ctx.translate(canvas.width / 2, canvas.height / 2);

        // Appliquer la rotation
        ctx.rotate((this.rotationAngle * Math.PI) / 180);

        // Appliquer le flip
        ctx.scale(this.flipHorizontal ? -1 : 1, this.flipVertical ? -1 : 1);

        // Dessiner l'image (centrée à l'origine)
        const drawWidth = img.naturalWidth * scale;
        const drawHeight = img.naturalHeight * scale;
        ctx.drawImage(img, -drawWidth / 2, -drawHeight / 2, drawWidth, drawHeight);

        // Restaurer le contexte
        ctx.restore();
      } catch (error) {
        console.error("Error updating preview:", error);
      }
    },

    downloadPreview() {
      const canvas = this.$refs.previewCanvas;

      if (!canvas) {
        return;
      }

      try {
        canvas.toBlob(
          (blob) => {
            const url = URL.createObjectURL(blob);
            const link = document.createElement("a");
            link.href = url;
            link.download = `preview_${this.model?.UID || "image"}_${Date.now()}.jpg`;
            document.body.appendChild(link);
            link.click();
            document.body.removeChild(link);
            URL.revokeObjectURL(url);

            this.$notify.success(this.$gettext("Preview downloaded"));
          },
          "image/jpeg",
          0.9
        );
      } catch (error) {
        console.error("Error downloading preview:", error);
        this.$notify.error(this.$gettext("Failed to download preview"));
      }
    },

    onSave() {
      if (!this.isImageLoaded) {
        this.$notify.error(this.$gettext("Image not loaded"));
        return;
      }

      try {
        // Créer l'objet de métadonnées pour le sidecar YAML
        const sidecarData = {
          crop: {
            left: 0,
            top: 0,
            width: 1,
            height: 1,
          },
          rotation: this.rotationAngle,
          flip: {
            horizontal: this.flipHorizontal,
            vertical: this.flipVertical,
          },
          aspectRatio: this.aspectRatio !== "free" ? this.aspectRatio : null,
          editedAt: new Date().toISOString(),
          editedBy: this.$session?.user?.Name || "user",
        };

        // Émettre l'événement save avec les données du sidecar
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
  }

  .editor-image {
    max-width: 100%;
    max-height: 100%;
    object-fit: contain;
    transition: transform 0.3s ease;
  }

  .editor-sidebar {
    background: #1e1e1e;
  }

  .yaml-debug {
    margin-top: 16px;
  }

  .preview-canvas {
    width: 100%;
    max-width: 280px;
    height: auto;
    border: 1px solid #333;
    border-radius: 4px;
    cursor: pointer;
    transition: border-color 0.2s;

    &:hover {
      border-color: #666;
    }
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
