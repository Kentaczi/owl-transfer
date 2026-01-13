#!/bin/bash

set -e

# QR File Transfer - Cross-Platform Build Script
# Builds for Windows, Linux, macOS on amd64 and arm64 architectures
# Creates native installers and distribution packages

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Version and metadata
VERSION="${VERSION:-1.0.0}"
RELEASE_DATE="$(date +%Y-%m-%d)"
PROJECT_NAME="qrtransfer"
AUTHOR="QR Transfer Team"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Build configuration
BUILD_DIR="build_output"
DIST_DIR="dist"
RELEASE_DIR="release"

# OS and architecture targets
TARGETS=(
    "linux:amd64:Linux x86_64"
    "linux:arm64:Linux ARM64"
    "darwin:amd64:macOS x86_64"
    "darwin:arm64:macOS ARM64"
    "windows:amd64:Windows x86_64"
    "windows:arm64:Windows ARM64"
)

# Installer configurations
declare -A DEB_ARCH_MAP=( ["amd64"]="amd64" ["arm64"]="arm64" )
declare -A RPM_ARCH_MAP=( ["amd64"]="x86_64" ["arm64"]="aarch64" )
declare -A MSI_ARCH_MAP=( ["amd64"]="x64" ["arm64"]="arm64" )

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check Go installation
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed. Please install Go 1.25 or later."
        exit 1
    fi
    
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    GO_MAJOR=$(echo $GO_VERSION | cut -d. -f1)
    GO_MINOR=$(echo $GO_VERSION | cut -d. -f2)
    
    if [ "$GO_MAJOR" -lt 1 ] || ([ "$GO_MAJOR" -eq 1 ] && [ "$GO_MINOR" -lt 25 ]); then
        log_error "Go 1.25 or later is required. Found: $GO_VERSION"
        exit 1
    fi
    
    log_success "Go $GO_VERSION found"
    
    # Check for required tools
    MISSING_TOOLS=""
    
    for tool in zip tar; do
        if ! command -v $tool &> /dev/null; then
            MISSING_TOOLS="$MISSING_TOOLS $tool"
        fi
    done
    
    if [ -n "$MISSING_TOOLS" ]; then
        log_warn "Missing tools:$MISSING_TOOLS"
        log_info "Some features may not work without these tools"
    fi
    
    # Check for platform-specific tools
    case "$(uname -s)" in
        Linux)
            if command -v fakeroot &> /dev/null; then
                log_info "fakeroot found - DEB packages can be built"
            fi
            ;;
        Darwin)
            log_info "macOS detected -DMG creation available"
            ;;
    esac
    
    log_success "Prerequisites check complete"
}

# Clean build directories
clean_build() {
    log_info "Cleaning build directories..."
    
    rm -rf "$BUILD_DIR" "$DIST_DIR" "$RELEASE_DIR"
    
    mkdir -p "$BUILD_DIR"
    mkdir -p "$DIST_DIR"
    mkdir -p "$RELEASE_DIR"
    
    log_success "Build directories cleaned"
}

# Build a single target
build_target() {
    local os=$1
    local arch=$2
    local desc=$3
    local output_name=$4
    
    log_info "Building for $os/$arch ($desc)..."
    
    # Set environment variables for cross-compilation
    export GOOS="$os"
    export GOARCH="$arch"
    export CGO_ENABLED=0
    
    # Determine output filename based on OS
    local binary_name="$PROJECT_NAME-$output_name"
    if [ "$os" = "windows" ]; then
        binary_name="$binary_name.exe"
    fi
    
    # Build the binary
    local build_args=(
        "-ldflags=-s -w"
        "-trimpath"
        "-buildmode=pie"
    )
    
    # Build sender
    if go build "${build_args[@]}" -o "$BUILD_DIR/sender-$os-$arch" ./cmd/sender; then
        log_success "Sender built: sender-$os-$arch"
    else
        log_error "Failed to build sender for $os/$arch"
        return 1
    fi
    
    # Build receiver
    if go build "${build_args[@]}" -o "$BUILD_DIR/receiver-$os-$arch" ./cmd/receiver; then
        log_success "Receiver built: receiver-$os-$arch"
    else
        log_error "Failed to build receiver for $os/$arch"
        return 1
    fi
    
    # Strip symbols if possible (reduces binary size)
    if command -v strip &> /dev/null; then
        strip "$BUILD_DIR/sender-$os-$arch" 2>/dev/null || true
        strip "$BUILD_DIR/receiver-$os-$arch" 2>/dev/null || true
        log_info "Symbols stripped for $os/$arch"
    fi
    
    return 0
}

# Build all targets
build_all() {
    log_info "Starting cross-platform build..."
    log_info "Version: $VERSION, Release Date: $RELEASE_DATE"
    echo ""
    
    local failed=0
    
    for target in "${TARGETS[@]}"; do
        IFS=':' read -r os arch desc <<< "$target"
        
        # Determine output name based on OS
        case "$os" in
            linux)
                output_name="linux-${arch}"
                ;;
            darwin)
                output_name="macos-${arch}"
                ;;
            windows)
                output_name="windows-${arch}"
                ;;
        esac
        
        if ! build_target "$os" "$arch" "$desc" "$output_name"; then
            failed=$((failed + 1))
        fi
        
        echo ""
    done
    
    if [ $failed -gt 0 ]; then
        log_error "$failed build(s) failed"
        return 1
    fi
    
    log_success "All targets built successfully"
    
    # List built binaries
    echo ""
    log_info "Built binaries:"
    ls -lh "$BUILD_DIR"/
    
    return 0
}

# Create directory structure for packaging
create_package_structure() {
    local os=$1
    local arch=$2
    local pkg_dir="$DIST_DIR/$os-$arch"
    
    mkdir -p "$pkg_dir"
    mkdir -p "$pkg_dir/bin"
    mkdir -p "$pkg_dir/docs"
    mkdir -p "$pkg_dir/examples"
    
    # Copy binaries
    cp "$BUILD_DIR/sender-$os-$arch" "$pkg_dir/bin/"
    cp "$BUILD_DIR/receiver-$os-$arch" "$pkg_dir/bin/"
    
    # Rename binaries to platform-appropriate names
    if [ "$os" = "windows" ]; then
        mv "$pkg_dir/bin/sender-$os-$arch" "$pkg_dir/bin/${PROJECT_NAME}-sender.exe"
        mv "$pkg_dir/bin/receiver-$os-$arch" "$pkg_dir/bin/${PROJECT_NAME}-receiver.exe"
    else
        mv "$pkg_dir/bin/sender-$os-$arch" "$pkg_dir/bin/${PROJECT_NAME}-sender"
        mv "$pkg_dir/bin/receiver-$os-$arch" "$pkg_dir/bin/${PROJECT_NAME}-receiver"
        chmod +x "$pkg_dir/bin/"*
    fi
    
    # Create README
    cat > "$pkg_dir/README.md" << EOF
# QR File Transfer v${VERSION}

## Overview

A Go-based tool for transferring files between air-gapped systems using color-based QR codes.

## Quick Start

### ${os^} ${arch^}

1. Open a terminal in the bin/ directory
2. Run the sender on the source machine: \`./${PROJECT_NAME}-sender\`
3. Run the receiver on the destination machine: \`./${PROJECT_NAME}-receiver\`
4. Follow the on-screen instructions

## Features

- QR code-based file transfer
- Reed-Solomon error correction
- Configurable redundancy levels
- Cross-platform support

## Documentation

See the main README.md in the project root for full documentation.

## License

MIT License

## Version

- Version: ${VERSION}
- Release Date: ${RELEASE_DATE}
EOF
    
    # Copy license
    if [ -f "LICENSE" ]; then
        cp "LICENSE" "$pkg_dir/"
    else
        cat > "$pkg_dir/LICENSE" << EOF
MIT License

Copyright (c) ${RELEASE_DATE} ${AUTHOR}

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
EOF
    fi
    
    echo "$pkg_dir"
}

# Create archive packages
create_archives() {
    log_info "Creating archive packages..."
    
    for target in "${TARGETS[@]}"; do
        IFS=':' read -r os arch desc <<< "$target"
        
        log_info "Creating archive for $os/$arch..."
        
        local pkg_dir=$(create_package_structure "$os" "$arch")
        local archive_name="${PROJECT_NAME}-${VERSION}-${os}-${arch}"
        
        # Create tar.gz archive
        if command -v tar &> /dev/null; then
            tar -czf "$DIST_DIR/${archive_name}.tar.gz" -C "$DIST_DIR" "$(basename "$pkg_dir")"
            log_success "Created: ${archive_name}.tar.gz"
        fi
        
        # Create zip archive (preferred for Windows)
        if command -v zip &> /dev/null; then
            cd "$DIST_DIR"
            zip -rq "${archive_name}.zip" "$(basename "$pkg_dir")"
            cd - > /dev/null
            log_success "Created: ${archive_name}.zip"
        fi
    done
    
    log_success "Archive packages created"
}

# Create Debian package
create_deb_package() {
    local os=$1
    local arch=$2
    
    if ! command -v fakeroot &> /dev/null; then
        log_warn "fakeroot not found - skipping DEB package for $os/$arch"
        return 1
    fi
    
    log_info "Creating DEB package for $os/$arch..."
    
    local pkg_dir="$DIST_DIR/${PROJECT_NAME}-${VERSION}-${os}-${arch}"
    local deb_pkg_dir="$DIST_DIR/deb-build"
    
    mkdir -p "$deb_pkg_dir/DEBIAN"
    
    # Copy package contents
    mkdir -p "$deb_pkg_dir/usr/bin"
    mkdir -p "$deb_pkg_dir/usr/share/doc/${PROJECT_NAME}"
    mkdir -p "$deb_pkg_dir/usr/share/man/man1"
    
    # Copy binaries
    cp "$BUILD_DIR/sender-$os-$arch" "$deb_pkg_dir/usr/bin/"
    cp "$BUILD_DIR/receiver-$os-$arch" "$deb_pkg_dir/usr/bin/"
    
    local arch_pkg=${DEB_ARCH_MAP[$arch]:-$arch}
    
    # Create control file
    cat > "$deb_pkg_dir/DEBIAN/control" << EOF
Package: ${PROJECT_NAME}
Version: ${VERSION}
Section: utils
Priority: optional
Architecture: ${arch_pkg}
Depends: libc6, libgtk-3-0, libgl1-mesa-glx (optional for GUI)
Maintainer: ${AUTHOR} <${AUTHOR}@example.com>
Description: QR File Transfer Tool
 A Go-based tool for transferring files between air-gapped systems
 using color-based QR codes displayed on screen.
Homepage: https://github.com/example/qrtransfer
EOF
    
    # Create conffiles
    touch "$deb_pkg_dir/DEBIAN/conffiles"
    
    # Copy documentation
    cp "$pkg_dir/README.md" "$deb_pkg_dir/usr/share/doc/${PROJECT_NAME}/"
    cp "$pkg_dir/LICENSE" "$deb_pkg_dir/usr/share/doc/${PROJECT_NAME}/"
    
    # Create postinst script
    cat > "$deb_pkg_dir/DEBIAN/postinst" << 'POSTINST'
#!/bin/bash
set -e
case "$1" in
    configure)
        # Make binaries executable
        chmod 755 /usr/bin/qrtransfer-*
        ;;
    abort-upgrade|abort-remove|abort-deconfigure)
        ;;
    *)
        ;;
esac
exit 0
POSTINST
    chmod +x "$deb_pkg_dir/DEBIAN/postinst"
    
    # Create prerm script
    cat > "$deb_pkg_dir/DEBIAN/prerm" << 'PRERM'
#!/bin/bash
set -e
case "$1" in
    remove|purge)
        ;;
    upgrade|failed-upgrade)
        ;;
    *)
        ;;
esac
exit 0
PRERM
    chmod +x "$deb_pkg_dir/DEBIAN/prerm"
    
    # Build the package
    local deb_name="${PROJECT_NAME}_${VERSION}-1_${arch_pkg}.deb"
    
    fakeroot dpkg-deb --build "$deb_pkg_dir" "$DIST_DIR/${deb_name}" 2>/dev/null || \
        dpkg-deb --build "$deb_pkg_dir" "$DIST_DIR/${deb_name}"
    
    log_success "Created DEB package: ${deb_name}"
    
    rm -rf "$deb_pkg_dir"
}

# Create RPM package
create_rpm_package() {
    local os=$1
    local arch=$2
    
    if ! command -v rpmbuild &> /dev/null; then
        log_warn "rpmbuild not found - skipping RPM package for $os/$arch"
        return 1
    fi
    
    log_info "Creating RPM package for $os/$arch..."
    
    local arch_pkg=${RPM_ARCH_MAP[$arch]:-$arch}
    local rpm_name="${PROJECT_NAME}-${VERSION}-1.${arch_pkg}"
    
    # Create RPM build directory structure
    local rpm_dir="$HOME/rpmbuild"
    mkdir -p "$rpm_dir/BUILD"
    mkdir -p "$rpm_dir/BUILDROOT"
    mkdir -p "$rpm_dir/RPMS"
    mkdir -p "$rpm_dir/SOURCES"
    mkdir -p "$rpm_dir/SPECS"
    mkdir -p "$rpm_dir/SRPMS"
    
    # Create source tarball
    local source_dir="$DIST_DIR/${PROJECT_NAME}-${VERSION}"
    cp -r "$source_dir" "$DIST_DIR/${PROJECT_NAME}-${VERSION}-tmp"
    
    tar -czf "$rpm_dir/SOURCES/${PROJECT_NAME}-${VERSION}.tar.gz" -C "$DIST_DIR" "${PROJECT_NAME}-${VERSION}-tmp"
    rm -rf "$DIST_DIR/${PROJECT_NAME}-${VERSION}-tmp"
    
    # Create spec file
    cat > "$rpm_dir/SPECS/${PROJECT_NAME}.spec" << SPECEOF
Name:           ${PROJECT_NAME}
Version:        ${VERSION}
Release:        1%{?dist}
Summary:        QR File Transfer Tool
License:        MIT
URL:            https://github.com/example/qrtransfer
Source0:        %{name}-%{version}.tar.gz

BuildRequires:  golang >= 1.25
Requires:       gtk3, mesa-libGL

%description
A Go-based tool for transferring files between air-gapped systems
using color-based QR codes displayed on screen.

%prep
%setup -q

%build
go build -ldflags="-s -w" -buildmode=pie -o bin/%{name}-sender ./cmd/sender
go build -ldflags="-s -w" -buildmode=pie -o bin/%{name}-receiver ./cmd/receiver

%install
mkdir -p %{buildroot}/usr/bin
mkdir -p %{buildroot}/usr/share/doc/%{name}
mkdir -p %{buildroot}/usr/share/man/man1

install -m 755 bin/%{name}-sender %{buildroot}/usr/bin/
install -m 755 bin/%{name}-receiver %{buildroot}/usr/bin/
install -m 644 README.md %{buildroot}/usr/share/doc/%{name}/
install -m 644 LICENSE %{buildroot}/usr/share/doc/%{name}/

%files
/usr/bin/%{name}-sender
/usr/bin/%{name}-receiver
%doc README.md LICENSE

%changelog
* $(date '+%a %b %d %Y') ${AUTHOR} - ${VERSION}-1
- Initial package release
SPECEOF
    
    # Build RPM
    rpmbuild --target "${arch_pkg}-redhat-linux-gnu" -bb "$rpm_dir/SPECS/${PROJECT_NAME}.spec"
    
    # Copy RPM to dist directory
    find "$rpm_dir/RPMS" -name "*.rpm" -exec cp {} "$DIST_DIR/" \;
    
    log_success "Created RPM package for $os/$arch"
}

# Create Windows MSI package using WiX Toolset
create_msi_package() {
    local os=$1
    local arch=$2
    
    if ! command -v candle &> /dev/null; then
        log_warn "WiX Toolset not found - skipping MSI package for $os/$arch"
        return 1
    fi
    
    log_info "Creating MSI package for $os/$arch..."
    
    # WiX source file
    cat > "$DIST_DIR/${PROJECT_NAME}.wxs" << 'WIXEOF'
<?xml version="1.0" encoding="UTF-8"?>
<Wix xmlns="http://schemas.microsoft.com/wix/2006/wi">
    <Product Id="*" Name="QR File Transfer" Language="1033" 
             Version="1.0.0" Manufacturer="QR Transfer Team" 
             UpgradeCode="PUT-GUID-HERE">
        <Package InstallerVersion="200" Compressed="yes" InstallScope="perMachine" />
        
        <MajorUpgrade DowngradeErrorMessage="A newer version is already installed." />
        <MediaTemplate EmbedCab="yes" />
        
        <Feature Id="ProductFeature" Title="QR File Transfer" Level="1">
            <ComponentGroupRef Id="ProductComponents" />
            <ComponentRef Id="ApplicationShortcut" />
        </Feature>
        
        <UIRef Id="WixUI_InstallDir" />
        <Property Id="WIXUI_INSTALLDIR" Value="INSTALLFOLDER" />
        
        <Directory Id="TARGETDIR" Name="SourceDir">
            <Directory Id="ProgramFilesFolder">
                <Directory Id="INSTALLFOLDER" Name="QR File Transfer" />
            </Directory>
            <Directory Id="ProgramMenuFolder">
                <Directory Id="ApplicationProgramsFolder" Name="QR File Transfer" />
            </Directory>
        </Directory>
        
        <DirectoryRef Id="INSTALLFOLDER">
            <Component Id="ProductComponents" Guid="PUT-GUID-HERE">
                <File Id="SenderEXE" Source="sender.exe" KeyPath="yes" />
                <File Id="ReceiverEXE" Source="receiver.exe" />
            </Component>
        </DirectoryRef>
        
        <DirectoryRef Id="ApplicationProgramsFolder">
            <Component Id="ApplicationShortcut" Guid="PUT-GUID-HERE">
                <Shortcut Id="ApplicationStartMenuShortcutSender" 
                          Name="QR Transfer Sender" 
                          Description="QR File Transfer Sender"
                          Target="[INSTALLFOLDER]sender.exe" WorkingDirectory="INSTALLFOLDER" />
                <Shortcut Id="ApplicationStartMenuShortcutReceiver" 
                          Name="QR Transfer Receiver" 
                          Description="QR File Transfer Receiver"
                          Target="[INSTALLFOLDER]receiver.exe" WorkingDirectory="INSTALLFOLDER" />
                <RemoveFolder Id="CleanUpShortCut" Directory="ApplicationProgramsFolder" On="uninstall" />
                <RegistryValue Root="HKCU" KeyPath="yes" Type="string" Value="" />
            </Component>
        </DirectoryRef>
    </Product>
</Wix>
WIXEOF
    
    log_warn "MSI package generation requires WiX Toolset configuration"
    log_info "Please install WiX Toolset and customize the GUIDs"
}

# Create macOS DMG package
create_dmg_package() {
    local os=$1
    local arch=$2
    
    if [ "$(uname -s)" != "Darwin" ]; then
        log_warn "DMG creation only supported on macOS - skipping for $os/$arch"
        return 1
    fi
    
    log_info "Creating DMG package for $os/$arch..."
    
    local dmg_name="${PROJECT_NAME}-${VERSION}-macos-${arch}"
    local dmg_dir="$DIST_DIR/dmg-temp"
    
    mkdir -p "$dmg_dir/${PROJECT_NAME}"
    
    # Copy binaries
    cp "$BUILD_DIR/sender-$os-$arch" "$dmg_dir/${PROJECT_NAME}/"
    cp "$BUILD_DIR/receiver-$os-$arch" "$dmg_dir/${PROJECT_NAME}/"
    
    # Copy documentation
    cp "$DIST_DIR/${PROJECT_NAME}-${VERSION}-${os}-${arch}/README.md" "$dmg_dir/${PROJECT_NAME}/"
    
    # Create symlinks for applications
    ln -sf "/Applications/${PROJECT_NAME}.app" "$dmg_dir/"
    
    # Create DMG using hdiutil
    if command -v hdiutil &> /dev/null; then
        hdiutil create -format UDZO -srcfolder "$dmg_dir" "$DIST_DIR/${dmg_name}.dmg"
        log_success "Created DMG package: ${dmg_name}.dmg"
    else
        log_warn "hdiutil not found - cannot create DMG"
    fi
    
    rm -rf "$dmg_dir"
}

# Create all installer packages
create_installers() {
    log_info "Creating installer packages..."
    
    for target in "${TARGETS[@]}"; do
        IFS=':' read -r os arch desc <<< "$target"
        
        # Create archives for all platforms
        create_package_structure "$os" "$arch" > /dev/null
        
        # Platform-specific installers
        case "$os" in
            linux)
                create_deb_package "$os" "$arch"
                create_rpm_package "$os" "$arch"
                ;;
            windows)
                create_msi_package "$os" "$arch"
                ;;
            darwin)
                create_dmg_package "$os" "$arch"
                ;;
        esac
    done
    
    log_success "Installer packages created"
}

# Create release archive with all platforms
create_release_archive() {
    log_info "Creating release archive..."
    
    local release_file="${PROJECT_NAME}-v${VERSION}-release"
    
    # Create temporary directory structure for release
    mkdir -p "$RELEASE_DIR"
    
    # Copy all distribution files
    cp -r "$DIST_DIR"/* "$RELEASE_DIR/"
    
    # Create main release manifest
    cat > "$RELEASE_DIR/MANIFEST.txt" << EOF
QR File Transfer v${VERSION} Release Manifest
==============================================
Release Date: ${RELEASE_DATE}
Total Binaries: $(ls -1 "$BUILD_DIR" | wc -l)

Included Packages:
EOF
    
    for f in "$DIST_DIR"/*; do
        if [ -f "$f" ]; then
            echo "  - $(basename "$f")" >> "$RELEASE_DIR/MANIFEST.txt"
            echo "    Size: $(du -h "$f" | cut -f1)" >> "$RELEASE_DIR/MANIFEST.txt"
        fi
    done
    
    cat >> "$RELEASE_DIR/MANIFEST.txt" << EOF

Installation Instructions:
--------------------------
1. Extract the archive for your target platform
2. See README.md for detailed instructions
3. Run the appropriate binary for your system

Quick Start:
- Linux/macOS: ./bin/qrtransfer-sender / ./bin/qrtransfer-receiver
- Windows: bin\qrtransfer-sender.exe / bin\qrtransfer-receiver.exe

For full documentation, visit:
https://github.com/example/qrtransfer
EOF
    
    # Create tar.gz of entire release
    cd "$DIST_DIR"
    tar -czf "../${release_file}.tar.gz" *
    cd - > /dev/null
    
    log_success "Release archive created: ${release_file}.tar.gz"
}

# Display build summary
show_summary() {
    echo ""
    echo "=============================================="
    echo "  BUILD COMPLETE"
    echo "=============================================="
    echo ""
    echo "Version: $VERSION"
    echo "Release Date: $RELEASE_DATE"
    echo ""
    echo "Build Directory: $BUILD_DIR"
    echo "Distribution: $DIST_DIR"
    echo "Release Archive: $RELEASE_DIR"
    echo ""
    echo "Built Binaries:"
    ls -lh "$BUILD_DIR" | tail -n +4
    echo ""
    echo "Distribution Packages:"
    ls -lh "$DIST_DIR" | tail -n +4
    echo ""
    
    echo "Usage Examples:"
    echo "  Linux (amd64):   $DIST_DIR/${PROJECT_NAME}-${VERSION}-linux-amd64/"
    echo "  macOS (arm64):   $DIST_DIR/${PROJECT_NAME}-${VERSION}-darwin-arm64/"
    echo "  Windows (amd64): $DIST_DIR/${PROJECT_NAME}-${VERSION}-windows-amd64.zip"
    echo ""
}

# Main function
main() {
    echo "=============================================="
    echo "  QR File Transfer - Cross-Platform Build"
    echo "=============================================="
    echo ""
    
    # Parse command line arguments
    case "${1:-all}" in
        clean)
            clean_build
            ;;
        deps)
            check_prerequisites
            ;;
        build)
            check_prerequisites
            clean_build
            build_all
            ;;
        archives)
            create_archives
            ;;
        installers)
            create_installers
            ;;
        release)
            check_prerequisites
            clean_build
            build_all
            create_archives
            create_installers
            create_release_archive
            show_summary
            ;;
        all)
            check_prerequisites
            clean_build
            build_all
            create_archives
            create_installers
            show_summary
            ;;
        help|-h|--help)
            echo "Usage: $0 [command]"
            echo ""
            echo "Commands:"
            echo "  clean       - Clean build directories"
            echo "  deps        - Check prerequisites"
            echo "  build       - Build all binaries (default)"
            echo "  archives    - Create archive packages only"
            echo "  installers  - Create installer packages only"
            echo "  release     - Create full release with all packages"
            echo "  all         - Build everything (same as release)"
            echo ""
            echo "Environment Variables:"
            echo "  VERSION     - Set version string (default: 1.0.0)"
            echo ""
            echo "Examples:"
            echo "  $0                    # Build all binaries"
            echo "  $0 clean              # Clean build directories"
            echo "  VERSION=2.0.0 $0 release  # Build version 2.0.0"
            echo ""
            ;;
        *)
            log_error "Unknown command: $1"
            echo "Run '$0 help' for usage information"
            exit 1
            ;;
    esac
}

# Run main function
main "$@"
