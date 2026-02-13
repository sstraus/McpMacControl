# MCPMacControl Makefile
# Builds the MCP server as a signed .app bundle for proper macOS TCC registration

APP_NAME=MCPMacControl
BINARY_NAME=mcpmaccontrol
APP_BUNDLE=$(APP_NAME).app
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUNDLE_ID=com.sstraus.mcpmaccontrol

# Code signing identity: override with SIGN_IDENTITY env var for your own cert.
# Auto-detection order: Developer ID > MCPMacControl-Dev > ad-hoc.
SIGN_IDENTITY?=-

# Build output directories
BUILD_DIR=build
APP_DIR=$(BUILD_DIR)/$(APP_BUNDLE)

.PHONY: all clean build test lint install verify-sign init cert tcc-reset notarize release

all: build

# Build the .app bundle with signed binary
build:
	@# Auto-detect best available signing certificate
	@if [ "$(SIGN_IDENTITY)" != "-" ]; then \
		SIGN_ID="$(SIGN_IDENTITY)"; \
	else \
		SIGN_ID=$$(security find-identity -v -p codesigning 2>/dev/null | grep "Developer ID Application:" | head -1 | sed 's/.*"\(.*\)".*/\1/'); \
		if [ -z "$$SIGN_ID" ]; then \
			SIGN_ID=$$(security find-identity -v -p codesigning 2>/dev/null | grep "MCPMacControl-Dev" | head -1 | sed 's/.*"\(.*\)".*/\1/'); \
		fi; \
		if [ -z "$$SIGN_ID" ]; then \
			echo "WARNING: Using ad-hoc signing — permissions will reset on every build. Run 'make cert' to fix."; \
			SIGN_ID="-"; \
		fi; \
	fi; \
	echo "Using certificate: $$SIGN_ID"; \
	echo "Building MCP server..."; \
	go build -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME) . ; \
	echo "Creating .app bundle..."; \
	mkdir -p $(APP_DIR)/Contents/MacOS; \
	mkdir -p $(APP_DIR)/Contents/Resources; \
	cp $(BUILD_DIR)/$(BINARY_NAME) $(APP_DIR)/Contents/MacOS/$(BINARY_NAME); \
	cp bundle/Info.plist $(APP_DIR)/Contents/Info.plist; \
	if [ -f bundle/AppIcon.icns ]; then \
		cp bundle/AppIcon.icns $(APP_DIR)/Contents/Resources/; \
	fi; \
	echo "Signing .app bundle (identity: $$SIGN_ID)..."; \
	codesign --force --sign "$$SIGN_ID" \
		--identifier "$(BUNDLE_ID)" \
		--options runtime \
		$(APP_DIR); \
	echo "Build complete: $(APP_DIR)"

# Run tests
test:
	go test -v ./...

# Run linter
lint:
	golangci-lint run

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)

# Install to /Applications
install: build
	@echo "Installing to /Applications..."
	cp -r $(APP_DIR) /Applications/
	@echo "Installed to /Applications/$(APP_BUNDLE)"
	@echo "MCP binary path: /Applications/$(APP_BUNDLE)/Contents/MacOS/$(BINARY_NAME)"

# Register macOS permissions (run once after build, or after rebuild with ad-hoc signing)
init: build
	@echo "Launching $(APP_BUNDLE) to register permissions..."
	@echo "Grant Accessibility and Screen Recording when prompted."
	open -a $(APP_DIR) --args --init
	@sleep 5
	@echo "Done. You can now use the MCP server."

# Verify code signature
verify-sign:
	codesign -dvvv $(APP_DIR)

# Create a self-signed code signing certificate for stable TCC permissions.
# One-time setup — the certificate is stored in your login keychain and lasts 10 years.
# Without this, ad-hoc signing generates a new CDHash per build, which resets
# macOS permissions (Accessibility, Screen Recording) on every rebuild.
CERT_NAME=MCPMacControl-Dev
cert:
	@# Check if certificate already exists
	@if security find-identity -v -p codesigning 2>/dev/null | grep -q "$(CERT_NAME)"; then \
		echo "Certificate '$(CERT_NAME)' already exists in keychain."; \
		exit 0; \
	fi
	@echo "Creating self-signed code signing certificate '$(CERT_NAME)'..."
	@TMPDIR=$$(mktemp -d); \
	trap "rm -rf $$TMPDIR" EXIT; \
	openssl req -x509 -newkey rsa:2048 -sha256 \
		-keyout "$$TMPDIR/key.pem" -out "$$TMPDIR/cert.pem" \
		-days 3650 -nodes \
		-subj "/CN=$(CERT_NAME)" \
		-addext "keyUsage=digitalSignature" \
		-addext "extendedKeyUsage=codeSigning" 2>/dev/null; \
	openssl pkcs12 -export -legacy -inkey "$$TMPDIR/key.pem" -in "$$TMPDIR/cert.pem" \
		-out "$$TMPDIR/cert.p12" -passout pass:temp 2>/dev/null; \
	security import "$$TMPDIR/cert.p12" -k ~/Library/Keychains/login.keychain-db \
		-P temp -T /usr/bin/codesign; \
	security add-trusted-cert -d -r trustRoot -p codeSign \
		-k ~/Library/Keychains/login.keychain-db "$$TMPDIR/cert.pem"; \
	security set-key-partition-list -S apple-tool:,apple:,codesign: -s \
		-k "" ~/Library/Keychains/login.keychain-db 2>/dev/null || true; \
	echo "Certificate '$(CERT_NAME)' created successfully."
	@echo "Verify with: security find-identity -v -p codesigning"

# Notarize the built .app bundle with Apple (requires stored credentials).
# First run: xcrun notarytool store-credentials "MCPMacControl" --apple-id YOUR_ID --team-id YOUR_TEAM
notarize: build
	@echo "Creating zip for notarization..."
	ditto -c -k --keepParent $(APP_DIR) /tmp/$(APP_NAME).zip
	@echo "Submitting to Apple notary service..."
	xcrun notarytool submit /tmp/$(APP_NAME).zip --keychain-profile "MCPMacControl" --wait
	@echo "Stapling notarization ticket..."
	xcrun stapler staple $(APP_DIR)
	@rm -f /tmp/$(APP_NAME).zip
	@echo "Notarization complete."

# Build, sign, notarize, and create distributable zip.
release: notarize
	@echo "Creating distributable zip..."
	ditto -c -k --keepParent $(APP_DIR) $(BUILD_DIR)/$(APP_NAME).app.zip
	@echo "Release artifact: $(BUILD_DIR)/$(APP_NAME).app.zip"
	@ls -lh $(BUILD_DIR)/$(APP_NAME).app.zip

# Reset TCC permissions for this app. Useful after switching from ad-hoc to cert signing.
tcc-reset:
	@echo "Resetting TCC permissions for $(BUNDLE_ID)..."
	tccutil reset Accessibility $(BUNDLE_ID)
	tccutil reset ScreenCapture $(BUNDLE_ID)
	@echo "TCC entries cleared. Re-launch the app to re-grant permissions."
