BINARY     := src-tauri/binaries/aisha-backend
# Detect target triple: prefer rustc, fall back to uname
RUST_HOST  := $(shell rustc -vV 2>/dev/null | awk '/^host:/{print $$2}')
ifneq ($(RUST_HOST),)
  ARCH := $(RUST_HOST)
else
  _UARCH := $(shell uname -m)
  ifeq ($(_UARCH),arm64)
    ARCH := aarch64-apple-darwin
  else
    ARCH := x86_64-apple-darwin
  endif
endif

.PHONY: build-backend dev-backend dev build setup icons

## Build Go backend and place sidecar binary for Tauri
build-backend:
	@mkdir -p src-tauri/binaries
	cd backend && go build -o ../$(BINARY)-$(ARCH) .
	@echo "Built: $(BINARY)-$(ARCH)"

## Run Go backend standalone (no Tauri)
dev-backend:
	cd backend && go run .

## Install all deps
setup:
	npm install
	cd frontend && npm install

## Generate placeholder icons (requires ImageMagick)
icons:
	@mkdir -p src-tauri/icons
	convert -size 256x256 xc:#7c6aff -font Helvetica -pointsize 80 \
	  -fill white -gravity center -annotate 0 "T" \
	  src-tauri/icons/256x256.png 2>/dev/null || \
	  python3 scripts/gen_icons.py
	convert src-tauri/icons/256x256.png -resize 32x32   src-tauri/icons/32x32.png
	convert src-tauri/icons/256x256.png -resize 128x128 src-tauri/icons/128x128.png
	convert src-tauri/icons/256x256.png src-tauri/icons/icon.icns 2>/dev/null || true
	convert src-tauri/icons/256x256.png src-tauri/icons/icon.ico  2>/dev/null || true

## Full dev mode (runs Tauri + Vite + Go backend via sidecar)
dev: build-backend
	npm run dev

## Production build
build: build-backend setup
	npm run build
