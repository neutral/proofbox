#compdef pb
# pb zsh completion script

# This script provides zsh completion for the pb command.
# To use it:
# 1. Ensure compinit is enabled in your .zshrc: autoload -U compinit && compinit
# 2. Copy this file to a directory in your $fpath (e.g., /usr/share/zsh/site-functions/)
# 3. Name it _pb (with underscore prefix)

_pb() {
    local -a commands
    commands=(
        'init:Initialize a new ProofBox database'
        'put:Insert or update a key-value pair'
        'get:Retrieve a value by key'
        'delete:Delete a key from the tree'
        'root:Show the current root hash'
        'prove:Generate a Merkle proof for a key'
        'verify:Verify a Merkle proof'
        'stats:Show database statistics'
        'batch:Execute batch operations from file'
        'export-keys:Export specific keys to a file'
        'import:Import tree data from a file'
        'repl:Start an interactive REPL session'
        'completion:Generate shell completion script'
        'help:Show help information'
    )

    local -a global_flags
    global_flags=(
        '--db[Database path]:path:_files'
        '--json[Output in JSON format]'
        '--verbose[Enable verbose output]'
        '--help[Show help information]'
    )

    _arguments -C \
        "${global_flags[@]}" \
        '1: :->command' \
        '*:: :->args'

    case $state in
        command)
            _describe -t commands 'pb commands' commands
            ;;
        args)
            case $words[1] in
                put)
                    _arguments \
                        '--hex-key[Interpret key as hex-encoded bytes]' \
                        '--file-value[Read value from file]:file:_files' \
                        "${global_flags[@]}" \
                        '1:key:' \
                        '2:value:'
                    ;;
                get)
                    _arguments \
                        '--hex-key[Interpret key as hex-encoded bytes]' \
                        '--version[Version to retrieve]:version:' \
                        "${global_flags[@]}" \
                        '1:key:'
                    ;;
                delete)
                    _arguments \
                        '--hex-key[Interpret key as hex-encoded bytes]' \
                        "${global_flags[@]}" \
                        '1:key:'
                    ;;
                prove)
                    _arguments \
                        '--hex-key[Interpret key as hex-encoded bytes]' \
                        '--version[Version to generate proof for]:version:' \
                        '--output[Write proof to file]:file:_files' \
                        "${global_flags[@]}" \
                        '1:key:'
                    ;;
                verify)
                    _arguments \
                        "${global_flags[@]}" \
                        '1:proof file:_files -g "*.proof"'
                    ;;
                batch)
                    _arguments \
                        '--dry-run[Validate without executing]' \
                        '--skip-errors[Continue on errors]' \
                        "${global_flags[@]}" \
                        '1:batch file:_files -g "*.json"'
                    ;;
                export-keys)
                    _arguments \
                        '--keys[File containing keys to export]:file:_files' \
                        '--format[Export format]:format:(json csv)' \
                        '--output[Output file]:file:_files' \
                        '--hex[Include hex encoding]' \
                        '--version[Version to export]:version:' \
                        '--keys-format[Format of input keys]:format:(text hex)' \
                        "${global_flags[@]}"
                    ;;
                import)
                    _arguments \
                        '--format[Import format]:format:(json csv)' \
                        '--skip-errors[Continue on import errors]' \
                        '--validate[Validate without importing]' \
                        "${global_flags[@]}" \
                        '1:import file:_files -g "*.(json|csv)"'
                    ;;
                completion)
                    _arguments \
                        '1:shell:(bash zsh fish powershell)'
                    ;;
                init|root|stats|repl|help)
                    _arguments "${global_flags[@]}"
                    ;;
            esac
            ;;
    esac
}

_pb "$@"