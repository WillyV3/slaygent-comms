#!/usr/bin/env bash
#
# Slaygent Communication Suite Installation Script
# Installs both the TUI manager and messaging system
#

set -euo pipefail

# Configuration
readonly SCRIPT_NAME="install.sh"
readonly PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# TUI Configuration
readonly TUI_BINARY_NAME="slaygent-manager"
readonly TUI_ALIAS="slay"
readonly TUI_SOURCE_DIR="${PROJECT_ROOT}/app/tui"
readonly TUI_BINARY="${PROJECT_ROOT}/app/tui/bin/${TUI_BINARY_NAME}"

# Messenger Configuration
readonly MSG_BINARY_NAME="msg"
readonly MSG_SOURCE="${PROJECT_ROOT}/app/messenger/msg.go"
readonly MSG_BINARY="${PROJECT_ROOT}/app/messenger/bin/${MSG_BINARY_NAME}"

# Common Configuration
readonly REGISTRY_DIR="${HOME}/.slaygent"
readonly REGISTRY_PATH="${REGISTRY_DIR}/registry.json"
readonly INSTALL_DIR="${HOME}/.local/bin"
readonly TUI_INSTALLED="${INSTALL_DIR}/${TUI_BINARY_NAME}"
readonly MSG_INSTALLED="${INSTALL_DIR}/${MSG_BINARY_NAME}"

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly CYAN='\033[0;36m'
readonly BOLD='\033[1m'
readonly NC='\033[0m'

# Output functions
print_header() {
    echo -e "\n${CYAN}${BOLD}=== $1 ===${NC}"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_info() {
    echo -e "${BLUE}→${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1" >&2
}

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Detect shell configuration file
get_shell_config() {
    if [[ -n "${SHELL:-}" ]]; then
        case "${SHELL}" in
            */zsh)
                echo "${HOME}/.zshrc"
                ;;
            */bash)
                echo "${HOME}/.bashrc"
                ;;
            *)
                echo "${HOME}/.profile"
                ;;
        esac
    else
        echo "${HOME}/.profile"
    fi
}

# Detect operating system and package manager
detect_os() {
    if [[ "$OSTYPE" == "darwin"* ]]; then
        OS="macos"
        if command_exists brew; then
            PKG_MANAGER="brew"
        else
            print_error "Homebrew not found on macOS"
            print_info "Install from: https://brew.sh"
            exit 1
        fi
    elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
        OS="linux"
        if command_exists apt; then
            PKG_MANAGER="apt"
        elif command_exists dnf; then
            PKG_MANAGER="dnf"
        elif command_exists yum; then
            PKG_MANAGER="yum"
        elif command_exists pacman; then
            PKG_MANAGER="pacman"
        elif command_exists zypper; then
            PKG_MANAGER="zypper"
        else
            print_error "No supported package manager found"
            print_info "Supported: apt, dnf, yum, pacman, zypper"
            exit 1
        fi
    else
        print_error "Unsupported operating system: $OSTYPE"
        exit 1
    fi

    print_info "Detected OS: $OS, Package Manager: $PKG_MANAGER"
}

# Install package using detected package manager
install_package() {
    local package="$1"
    local linux_package="$2"

    case "$PKG_MANAGER" in
        brew)
            brew install "$package"
            ;;
        apt)
            sudo apt update && sudo apt install -y "${linux_package:-$package}"
            ;;
        dnf)
            sudo dnf install -y "${linux_package:-$package}"
            ;;
        yum)
            sudo yum install -y "${linux_package:-$package}"
            ;;
        pacman)
            sudo pacman -S --noconfirm "${linux_package:-$package}"
            ;;
        zypper)
            sudo zypper install -y "${linux_package:-$package}"
            ;;
        *)
            print_error "Unsupported package manager: $PKG_MANAGER"
            exit 1
            ;;
    esac
}

# Check and install prerequisites
check_prerequisites() {
    print_header "Checking Prerequisites"

    detect_os

    # Check and install required tools
    local tools_to_install=()

    if ! command_exists go; then
        print_warning "Go not found"
        tools_to_install+=("go:golang")
    else
        local go_version
        go_version=$(go version | awk '{print $3}' | sed 's/go//')
        print_info "Go version: ${go_version}"
    fi

    if ! command_exists tmux; then
        print_warning "tmux not found"
        tools_to_install+=("tmux:tmux")
    else
        print_info "tmux version: $(tmux -V)"
    fi

    if ! command_exists fd; then
        print_warning "fd not found"
        if [[ "$OS" == "linux" ]]; then
            tools_to_install+=("fd:fd-find")
        else
            tools_to_install+=("fd:fd")
        fi
    else
        print_info "fd version: $(fd --version | head -1)"
    fi

    # Install missing tools
    if [[ ${#tools_to_install[@]} -gt 0 ]]; then
        print_info "Installing missing tools..."

        for tool_spec in "${tools_to_install[@]}"; do
            local brew_name="${tool_spec%%:*}"
            local linux_name="${tool_spec##*:}"

            print_info "Installing ${brew_name}..."
            if install_package "$brew_name" "$linux_name"; then
                print_success "${brew_name} installed"
            else
                print_error "Failed to install ${brew_name}"
                exit 1
            fi
        done

        # Special handling for fd on Ubuntu/Debian - create symlink
        if [[ "$PKG_MANAGER" == "apt" ]] && ! command_exists fd; then
            if command_exists fdfind; then
                print_info "Creating fd symlink for fdfind"
                mkdir -p "$HOME/.local/bin"
                ln -sf "$(which fdfind)" "$HOME/.local/bin/fd"
                print_success "fd symlink created"
            fi
        fi
    fi

    print_success "All prerequisites met"
}

# Build TUI manager
build_tui() {
    print_header "Building TUI Manager"

    if [[ ! -d "${TUI_SOURCE_DIR}" ]]; then
        print_error "TUI source directory not found: ${TUI_SOURCE_DIR}"
        exit 1
    fi

    print_info "Building ${TUI_BINARY_NAME} from ${TUI_SOURCE_DIR}"

    (
        cd "${TUI_SOURCE_DIR}"
        mkdir -p bin
        if go build -o "bin/${TUI_BINARY_NAME}"; then
            print_success "TUI build successful"
        else
            print_error "TUI build failed"
            exit 1
        fi
    )
}

# Build messenger
build_messenger() {
    print_header "Building Messenger"

    if [[ ! -f "${MSG_SOURCE}" ]]; then
        print_error "Messenger source not found: ${MSG_SOURCE}"
        exit 1
    fi

    print_info "Installing SQLite driver..."
    (
        cd "$(dirname "${MSG_SOURCE}")"
        go get github.com/mattn/go-sqlite3
    )

    print_info "Building ${MSG_BINARY_NAME} from ${MSG_SOURCE}"

    (
        cd "$(dirname "${MSG_SOURCE}")"
        mkdir -p bin
        if go build -o "bin/${MSG_BINARY_NAME}" .; then
            print_success "Messenger build successful"
        else
            print_error "Messenger build failed"
            exit 1
        fi
    )
}

# Install binaries
install_binaries() {
    print_header "Installing Binaries"

    # Create install directory if needed
    if [[ ! -d "${INSTALL_DIR}" ]]; then
        print_info "Creating ${INSTALL_DIR}"
        mkdir -p "${INSTALL_DIR}"
    fi

    # Verify source binaries exist
    if [[ ! -f "${TUI_BINARY}" ]]; then
        print_error "TUI binary not found at ${TUI_BINARY}"
        exit 1
    fi

    if [[ ! -f "${MSG_BINARY}" ]]; then
        print_error "Messenger binary not found at ${MSG_BINARY}"
        exit 1
    fi

    # Install TUI (overwrite if exists - no prompts for automation)
    print_info "Installing ${TUI_BINARY_NAME} to ${TUI_INSTALLED}"
    if cp "${TUI_BINARY}" "${TUI_INSTALLED}" && chmod +x "${TUI_INSTALLED}"; then
        print_success "TUI installed successfully"
    else
        print_error "Failed to install TUI"
        exit 1
    fi

    # Install Messenger (overwrite if exists - no prompts for automation)
    print_info "Installing ${MSG_BINARY_NAME} to ${MSG_INSTALLED}"
    if cp "${MSG_BINARY}" "${MSG_INSTALLED}" && chmod +x "${MSG_INSTALLED}"; then
        print_success "Messenger installed successfully"
    else
        print_error "Failed to install Messenger"
        exit 1
    fi
}

# Configure shell aliases
configure_aliases() {
    print_header "Configuring Shell Aliases"

    local shell_config
    shell_config=$(get_shell_config)

    print_info "Shell configuration: ${shell_config}"

    local aliases_added=false

    # Check and add TUI alias
    if [[ -f "${shell_config}" ]] && grep -q "alias ${TUI_ALIAS}=" "${shell_config}"; then
        local existing_alias=$(grep "alias ${TUI_ALIAS}=" "${shell_config}")
        print_info "TUI alias '${TUI_ALIAS}' already exists: ${existing_alias}"

        # Check if it points to the right command
        if echo "${existing_alias}" | grep -q "slaygent-manager"; then
            print_success "TUI alias correctly configured"
        else
            print_warning "TUI alias exists but points to different command: ${existing_alias}"
            print_info "Consider updating to: alias ${TUI_ALIAS}='slaygent-manager'"
        fi
    else
        if [[ "${aliases_added}" == false ]]; then
            echo "" >> "${shell_config}"
            echo "# Slaygent Communication Suite" >> "${shell_config}"
            aliases_added=true
        fi
        echo "alias ${TUI_ALIAS}='slaygent-manager'" >> "${shell_config}"
        print_success "TUI alias '${TUI_ALIAS}' added: alias ${TUI_ALIAS}='slaygent-manager'"
    fi

    # Check if msg command is accessible and log details
    print_info "Checking msg command accessibility..."
    if command -v msg >/dev/null 2>&1; then
        local msg_location=$(which msg)
        print_success "msg command found at: ${msg_location}"
        if [[ "${msg_location}" == "${MSG_INSTALLED}" ]]; then
            print_success "msg points to correct installation"
        else
            print_warning "msg found at different location: ${msg_location}"
            print_info "Expected location: ${MSG_INSTALLED}"
        fi
    else
        print_warning "msg command not found in PATH"
        print_info "Installed at: ${MSG_INSTALLED}"
        print_info "PATH may need to be refreshed"
    fi

    if [[ "${aliases_added}" == true ]] || grep -q "Slaygent Communication Suite" "${shell_config}"; then
        print_info "Run 'source ${shell_config}' or restart your shell to use aliases"
    fi
}

# Update PATH if needed
update_path() {
    print_header "Checking PATH"

    if [[ ":${PATH}:" == *":${INSTALL_DIR}:"* ]]; then
        print_success "${INSTALL_DIR} already in PATH"
        return
    fi

    local shell_config
    shell_config=$(get_shell_config)

    print_info "Adding ${INSTALL_DIR} to PATH"
    {
        echo ""
        echo "# Add local bin to PATH"
        echo "export PATH=\"\${HOME}/.local/bin:\${PATH}\""
    } >> "${shell_config}"

    print_success "PATH updated"
}

# Initialize registry if needed
initialize_registry() {
    print_header "Checking Registry"

    # Create .slaygent directory if it doesn't exist
    if [[ ! -d "${REGISTRY_DIR}" ]]; then
        print_info "Creating ${REGISTRY_DIR}"
        mkdir -p "${REGISTRY_DIR}"
    fi

    if [[ -f "${REGISTRY_PATH}" ]]; then
        local agent_count
        agent_count=$(grep -c '"name"' "${REGISTRY_PATH}" 2>/dev/null || echo "0")
        print_success "Registry exists with ${agent_count} registered agents"
    else
        print_info "Creating empty registry"
        echo "[]" > "${REGISTRY_PATH}"
        print_success "Registry initialized at ${REGISTRY_PATH}"
    fi
}

# Run initial sync to set up CLAUDE.md files
run_initial_sync() {
    print_header "Running Initial Sync"

    # Find the sync script
    local sync_script=""
    if [[ -f "app/scripts/sync-claude.sh" ]]; then
        sync_script="app/scripts/sync-claude.sh"
    elif [[ -f "scripts/sync-claude.sh" ]]; then
        sync_script="scripts/sync-claude.sh"
    else
        print_warning "Sync script not found, skipping initial sync"
        return 0
    fi

    print_info "Running basic sync to initialize CLAUDE.md files..."

    # Run sync script with auto-confirmation
    if bash -c "echo 'y' | bash ${sync_script}" 2>/dev/null; then
        print_success "Initial sync completed successfully"
    else
        print_warning "Initial sync failed, but installation can continue"
        print_info "You can run sync manually later with: slay -> s"
    fi
}

# Verify installation
verify_installation() {
    print_header "Verifying Installation"

    local all_good=true

    # Check TUI
    if [[ -x "${TUI_INSTALLED}" ]]; then
        print_success "TUI binary installed and executable"
    else
        print_error "TUI binary not found or not executable"
        all_good=false
    fi

    # Check Messenger
    if [[ -x "${MSG_INSTALLED}" ]]; then
        print_success "Messenger binary installed and executable"
    else
        print_error "Messenger binary not found or not executable"
        all_good=false
    fi

    # Test messenger
    if "${MSG_INSTALLED}" --status >/dev/null 2>&1; then
        print_success "Messenger working correctly"
    else
        print_warning "Messenger test failed - tmux may not be running"
    fi

    if [[ "${all_good}" == true ]]; then
        return 0
    else
        return 1
    fi
}


# Show completion message
show_completion() {
    print_header "Installation Complete"

    echo -e "\n${GREEN}${BOLD}Success!${NC} Slaygent Communication Suite installed.\n"

    echo -e "${CYAN}${BOLD}Commands:${NC}"
    echo -e "  ${BOLD}${TUI_ALIAS}${NC}                        - Launch TUI manager"
    echo -e "  ${BOLD}${MSG_BINARY_NAME} <agent> <message>${NC}   - Send message to agent"
    echo -e "  ${BOLD}${MSG_BINARY_NAME} --status${NC}            - Show agent status"

    echo -e "\n${CYAN}${BOLD}Ready to use:${NC}"
    echo -e "  ${BOLD}${TUI_ALIAS}${NC}  - Launch TUI manager"

    echo -e "\n${BLUE}Happy coding with Slaygent!${NC}"
}

# Main installation flow
main() {
    echo -e "${CYAN}${BOLD}"
    echo "╔════════════════════════════════════════╗"
    echo "║   Slaygent Communication Suite         ║"
    echo "║   TUI Manager & Messaging System       ║"
    echo "╚════════════════════════════════════════╝"
    echo -e "${NC}"

    check_prerequisites
    build_tui
    build_messenger
    install_binaries
    configure_aliases
    update_path
    initialize_registry
    run_initial_sync

    if verify_installation; then
        show_completion

        # Auto-source shell config
        local shell_config
        shell_config=$(get_shell_config)
        print_info "Auto-sourcing shell config..."
        exec "$SHELL" -l
    else
        print_error "Installation verification failed"
        print_info "Please check the errors above and try again"
        exit 1
    fi
}

# Run main function
main "$@"