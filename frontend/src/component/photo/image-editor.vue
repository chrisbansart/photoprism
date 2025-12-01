<template>
  <v-dialog :model-value="visible" fullscreen persistent theme="dark" class="p-dialog p-image-editor" @keydown.esc.exact.stop="onClose">
    <v-card class="image-editor-container">
      <!-- Header avec les contrôles -->
      <v-toolbar dark color="black" density="compact">
        <v-btn icon @click="onClose">
          <v-icon>mdi-close</v-icon>
        </v-btn>

        <v-toolbar-title>{{ $gettext("Edit Image") }}</v-toolbar-title>

        <v-spacer></v-spacer>

        <!-- Boutons d'outils -->
        <v-btn icon :class="{ active: tool === 'crop' }" @click="tool = 'crop'">
          <v-icon>mdi-crop</v-icon>
        </v-btn>

        <v-btn icon :class="{ active: tool === 'rotate' }" @click="tool = 'rotate'">
          <v-icon>mdi-rotate-right</v-icon>
        </v-btn>

        <v-divider vertical class="mx-2"></v-divider>

        <v-btn color="primary" variant="flat" @click="onSave">
          {{ $gettext("Save") }}
        </v-btn>
      </v-toolbar>

      <!-- Zone d'édition de l'image -->
      <v-card-text class="editor-content pa-0">
        <div ref="editorContainer" class="editor-canvas-container">
          <!-- L'image sera affichée ici -->
          <img v-if="imageUrl" ref="imageElement" :src="imageUrl" class="editor-image" @load="onImageLoad" />

          <!-- Overlay pour le crop -->
          <div v-if="tool === 'crop' && imageLoaded" class="crop-overlay">
            <div ref="cropBox" class="crop-box" :style="cropBoxStyle" @mousedown="startDrag">
              <!-- Poignées de redimensionnement -->
              <div class="crop-handle crop-handle-nw" @mousedown.stop="startResize('nw')"></div>
              <div class="crop-handle crop-handle-ne" @mousedown.stop="startResize('ne')"></div>
              <div class="crop-handle crop-handle-sw" @mousedown.stop="startResize('sw')"></div>
              <div class="crop-handle crop-handle-se" @mousedown.stop="startResize('se')"></div>
            </div>
          </div>
        </div>
      </v-card-text>

      <!-- Panneau latéral avec les options -->
      <v-navigation-drawer v-if="tool === 'crop'" permanent location="right" width="300" class="editor-sidebar">
        <v-card flat>
          <v-card-title>{{ $gettext("Crop Options") }}</v-card-title>
          <v-card-text>
            <v-select
              v-model="aspectRatio"
              :label="$gettext('Aspect Ratio')"
              :items="aspectRatioOptions"
              density="compact"
              @update:model-value="updateCropBox"
            ></v-select>

            <div class="mt-4">
              <div class="text-caption">{{ $gettext("Dimensions") }}</div>
              <div class="text-body-2">{{ cropWidth }} × {{ cropHeight }} px</div>
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
      tool: "crop",
      imageUrl: "",
      imageLoaded: false,
      aspectRatio: "free",
      aspectRatioOptions: [
        { value: "free", title: this.$gettext("Free") },
        { value: "1:1", title: "1:1 (Square)" },
        { value: "4:3", title: "4:3" },
        { value: "16:9", title: "16:9" },
        { value: "3:2", title: "3:2" },
      ],
      // Crop box position and size
      cropBox: {
        x: 0,
        y: 0,
        width: 0,
        height: 0,
      },
      // Image dimensions
      imageRect: {
        width: 0,
        height: 0,
        x: 0,
        y: 0,
      },
      // Drag state
      isDragging: false,
      isResizing: false,
      resizeHandle: null,
      dragStart: { x: 0, y: 0 },
      cropStart: { x: 0, y: 0, width: 0, height: 0 },
    };
  },
  computed: {
    cropBoxStyle() {
      return {
        left: `${this.cropBox.x}px`,
        top: `${this.cropBox.y}px`,
        width: `${this.cropBox.width}px`,
        height: `${this.cropBox.height}px`,
      };
    },
    cropWidth() {
      return Math.round(this.cropBox.width);
    },
    cropHeight() {
      return Math.round(this.cropBox.height);
    },
  },
  watch: {
    visible(val) {
      if (val && this.model) {
        this.loadImage();
      }
    },
    model(val) {
      if (val && this.visible) {
        this.loadImage();
      }
    },
  },
  mounted() {
    if (this.visible && this.model) {
      this.loadImage();
    }

    // Ajouter les event listeners pour le drag
    document.addEventListener("mousemove", this.onMouseMove);
    document.addEventListener("mouseup", this.onMouseUp);
  },
  beforeUnmount() {
    document.removeEventListener("mousemove", this.onMouseMove);
    document.removeEventListener("mouseup", this.onMouseUp);
  },
  methods: {
    loadImage() {
      if (!this.model) return;

      // Charger l'URL de l'image en haute résolution
      this.imageUrl = this.model.thumbnailUrl("fit_2048");
      this.imageLoaded = false;
    },

    onImageLoad() {
      this.imageLoaded = true;
      this.$nextTick(() => {
        this.initializeCropBox();
      });
    },

    initializeCropBox() {
      const img = this.$refs.imageElement;
      if (!img) return;

      const rect = img.getBoundingClientRect();
      this.imageRect = {
        x: rect.left,
        y: rect.top,
        width: rect.width,
        height: rect.height,
      };

      // Initialiser le crop box au centre avec 80% de la taille de l'image
      const margin = 0.1;
      this.cropBox = {
        x: rect.width * margin,
        y: rect.height * margin,
        width: rect.width * (1 - 2 * margin),
        height: rect.height * (1 - 2 * margin),
      };
    },

    updateCropBox() {
      if (this.aspectRatio === "free") return;

      const [w, h] = this.aspectRatio.split(":").map(Number);
      const ratio = w / h;

      // Ajuster la hauteur en fonction de la largeur actuelle
      const newHeight = this.cropBox.width / ratio;

      if (newHeight <= this.imageRect.height - this.cropBox.y) {
        this.cropBox.height = newHeight;
      } else {
        // Si la hauteur dépasse, ajuster la largeur
        this.cropBox.height = this.imageRect.height - this.cropBox.y;
        this.cropBox.width = this.cropBox.height * ratio;
      }
    },

    startDrag(event) {
      this.isDragging = true;
      this.dragStart = { x: event.clientX, y: event.clientY };
      this.cropStart = { ...this.cropBox };
    },

    startResize(handle) {
      return (event) => {
        this.isResizing = true;
        this.resizeHandle = handle;
        this.dragStart = { x: event.clientX, y: event.clientY };
        this.cropStart = { ...this.cropBox };
      };
    },

    onMouseMove(event) {
      if (this.isDragging) {
        const dx = event.clientX - this.dragStart.x;
        const dy = event.clientY - this.dragStart.y;

        this.cropBox.x = Math.max(0, Math.min(this.imageRect.width - this.cropBox.width, this.cropStart.x + dx));
        this.cropBox.y = Math.max(0, Math.min(this.imageRect.height - this.cropBox.height, this.cropStart.y + dy));
      } else if (this.isResizing) {
        const dx = event.clientX - this.dragStart.x;
        const dy = event.clientY - this.dragStart.y;

        // Redimensionner selon la poignée utilisée
        switch (this.resizeHandle) {
          case "se": // Sud-est (bas-droite)
            this.cropBox.width = Math.max(50, Math.min(this.imageRect.width - this.cropBox.x, this.cropStart.width + dx));
            this.cropBox.height = Math.max(50, Math.min(this.imageRect.height - this.cropBox.y, this.cropStart.height + dy));
            break;
          case "sw": // Sud-ouest (bas-gauche)
            const newWidth = this.cropStart.width - dx;
            if (newWidth >= 50 && this.cropStart.x + dx >= 0) {
              this.cropBox.x = this.cropStart.x + dx;
              this.cropBox.width = newWidth;
            }
            this.cropBox.height = Math.max(50, Math.min(this.imageRect.height - this.cropBox.y, this.cropStart.height + dy));
            break;
          case "ne": // Nord-est (haut-droite)
            this.cropBox.width = Math.max(50, Math.min(this.imageRect.width - this.cropBox.x, this.cropStart.width + dx));
            const newHeight = this.cropStart.height - dy;
            if (newHeight >= 50 && this.cropStart.y + dy >= 0) {
              this.cropBox.y = this.cropStart.y + dy;
              this.cropBox.height = newHeight;
            }
            break;
          case "nw": // Nord-ouest (haut-gauche)
            const newWidthNW = this.cropStart.width - dx;
            if (newWidthNW >= 50 && this.cropStart.x + dx >= 0) {
              this.cropBox.x = this.cropStart.x + dx;
              this.cropBox.width = newWidthNW;
            }
            const newHeightNW = this.cropStart.height - dy;
            if (newHeightNW >= 50 && this.cropStart.y + dy >= 0) {
              this.cropBox.y = this.cropStart.y + dy;
              this.cropBox.height = newHeightNW;
            }
            break;
        }

        // Maintenir le ratio si nécessaire
        if (this.aspectRatio !== "free") {
          this.updateCropBox();
        }
      }
    },

    onMouseUp() {
      this.isDragging = false;
      this.isResizing = false;
      this.resizeHandle = null;
    },

    onClose() {
      this.$emit("close");
    },

    onSave() {
      // Calculer les coordonnées relatives de crop (0-1)
      const cropData = {
        x: this.cropBox.x / this.imageRect.width,
        y: this.cropBox.y / this.imageRect.height,
        width: this.cropBox.width / this.imageRect.width,
        height: this.cropBox.height / this.imageRect.height,
      };

      this.$emit("save", {
        model: this.model,
        tool: this.tool,
        cropData: cropData,
      });

      this.$notify.success(this.$gettext("Changes saved"));
      this.onClose();
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
    background: #000;
  }

  .editor-content {
    flex: 1;
    overflow: hidden;
    display: flex;
    align-items: center;
    justify-content: center;
    position: relative;
  }

  .editor-canvas-container {
    position: relative;
    max-width: 100%;
    max-height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .editor-image {
    max-width: 100%;
    max-height: calc(100vh - 64px);
    display: block;
    user-select: none;
  }

  .crop-overlay {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    pointer-events: none;
  }

  .crop-box {
    position: absolute;
    border: 2px solid #fff;
    box-shadow: 0 0 0 9999px rgba(0, 0, 0, 0.5);
    cursor: move;
    pointer-events: all;
  }

  .crop-handle {
    position: absolute;
    width: 12px;
    height: 12px;
    background: #fff;
    border: 2px solid #000;
    border-radius: 50%;
  }

  .crop-handle-nw {
    top: -6px;
    left: -6px;
    cursor: nw-resize;
  }

  .crop-handle-ne {
    top: -6px;
    right: -6px;
    cursor: ne-resize;
  }

  .crop-handle-sw {
    bottom: -6px;
    left: -6px;
    cursor: sw-resize;
  }

  .crop-handle-se {
    bottom: -6px;
    right: -6px;
    cursor: se-resize;
  }

  .editor-sidebar {
    background: #1e1e1e;
  }

  .v-btn.active {
    background: rgba(255, 255, 255, 0.1);
  }
}
</style>
