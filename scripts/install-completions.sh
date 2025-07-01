#!/usr/bin/env bash
# Install shell completions for pb

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPLETIONS_DIR="$SCRIPT_DIR/completions"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

echo "Installing pb shell completions..."

# Detect the shell
if [ -n "$BASH_VERSION" ]; then
    SHELL_TYPE="bash"
elif [ -n "$ZSH_VERSION" ]; then
    SHELL_TYPE="zsh"
else
    echo -e "${YELLOW}Warning: Could not detect shell type. Please install manually.${NC}"
    exit 1
fi

# Install based on shell type
case "$SHELL_TYPE" in
    bash)
        echo "Detected Bash shell"
        
        # Find bash completions directory
        if [[ "$OSTYPE" == "darwin"* ]]; then
            # macOS
            if command -v brew &> /dev/null; then
                BASH_COMPLETION_DIR="$(brew --prefix)/etc/bash_completion.d"
            else
                BASH_COMPLETION_DIR="/usr/local/etc/bash_completion.d"
            fi
        else
            # Linux
            BASH_COMPLETION_DIR="/etc/bash_completion.d"
        fi

        # Create directory if it doesn't exist
        if [ ! -d "$BASH_COMPLETION_DIR" ]; then
            echo "Creating completion directory: $BASH_COMPLETION_DIR"
            sudo mkdir -p "$BASH_COMPLETION_DIR"
        fi

        # Copy completion file
        echo "Installing completion to: $BASH_COMPLETION_DIR/pb"
        sudo cp "$COMPLETIONS_DIR/pb.bash" "$BASH_COMPLETION_DIR/pb"
        
        echo -e "${GREEN}Bash completion installed successfully!${NC}"
        echo "Please restart your shell or run: source $BASH_COMPLETION_DIR/pb"
        ;;
        
    zsh)
        echo "Detected Zsh shell"
        
        # Find a suitable directory in fpath
        ZSH_COMPLETION_DIR=""
        for dir in /usr/local/share/zsh/site-functions /usr/share/zsh/site-functions ~/.zsh/completions; do
            if [ -d "$dir" ] || [ "$dir" = ~/.zsh/completions ]; then
                ZSH_COMPLETION_DIR="$dir"
                break
            fi
        done

        if [ -z "$ZSH_COMPLETION_DIR" ]; then
            # Create local completions directory
            ZSH_COMPLETION_DIR="$HOME/.zsh/completions"
            mkdir -p "$ZSH_COMPLETION_DIR"
            echo "Created local completions directory: $ZSH_COMPLETION_DIR"
            echo "Add the following to your ~/.zshrc:"
            echo "  fpath=(~/.zsh/completions \$fpath)"
        fi

        # Copy completion file (with underscore prefix)
        echo "Installing completion to: $ZSH_COMPLETION_DIR/_pb"
        if [[ "$ZSH_COMPLETION_DIR" == "$HOME"* ]]; then
            cp "$COMPLETIONS_DIR/pb.zsh" "$ZSH_COMPLETION_DIR/_pb"
        else
            sudo cp "$COMPLETIONS_DIR/pb.zsh" "$ZSH_COMPLETION_DIR/_pb"
        fi
        
        echo -e "${GREEN}Zsh completion installed successfully!${NC}"
        echo "Please restart your shell or run: autoload -U compinit && compinit"
        ;;
esac

# Alternative: Source directly from current shell
echo ""
echo "Alternatively, you can source completions directly by adding to your shell config:"
echo "  source $COMPLETIONS_DIR/pb.$SHELL_TYPE"