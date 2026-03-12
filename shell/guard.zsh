# guard-sh zsh integration

_GUARD_SCRIPT_PATH="${${(%):-%x}:A}"

_guard_zsh_accept_line() {
    local cmd="$BUFFER"

    if [[ -z "${cmd//[[:space:]]/}" ]]; then
        zle .accept-line
        return
    fi

    if ! command -v guard-sh &>/dev/null; then
        zle .accept-line
        return
    fi

    local warning
    warning=$(command guard-sh check "$cmd")
    local exit_code=$?

    if [[ $exit_code -eq 0 ]]; then
        zle .accept-line
        return
    fi

    printf '\nguard-sh: %s ' "$warning"
    local confirm
    read -rk1 confirm
    printf '\n'

    if [[ "$confirm" == [Yy] ]]; then
        zle .accept-line
    else
        zle reset-prompt
    fi
}

_guard_enable() {
    zle -N _guard_zsh_accept_line
    bindkey "^M" _guard_zsh_accept_line
    bindkey "^J" _guard_zsh_accept_line
    echo "guard-sh: enabled"
}

_guard_disable() {
    bindkey "^M" accept-line
    bindkey "^J" accept-line
    echo "guard-sh: disabled"
}

_guard_global_on() {
    local rc="$HOME/.zshrc"
    local line="source \"$_GUARD_SCRIPT_PATH\""
    if grep -qF "$line" "$rc" 2>/dev/null; then
        echo "guard-sh: already enabled globally in $rc"
    else
        printf '\n# guard-sh\n%s\n' "$line" >> "$rc"
        echo "guard-sh: enabled globally in $rc"
    fi
}

_guard_global_off() {
    local rc="$HOME/.zshrc"
    local line="source \"$_GUARD_SCRIPT_PATH\""
    if grep -qF "$line" "$rc" 2>/dev/null; then
        grep -v "^# guard-sh$" "$rc" | grep -vF "$line" > "${rc}.guardtmp" && mv "${rc}.guardtmp" "$rc"
        echo "guard-sh: disabled globally in $rc"
    else
        echo "guard-sh: not found in $rc"
    fi
}

guard-sh() {
    case "$1" in
        on)
            [[ "$2" == "--global" ]] && _guard_global_on || _guard_enable
            ;;
        off)
            [[ "$2" == "--global" ]] && _guard_global_off || _guard_disable
            ;;
        *)
            command guard-sh "$@"
            ;;
    esac
}

_guard_enable
