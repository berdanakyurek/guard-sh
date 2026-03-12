# guard-sh zsh integration

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
    warning=$(guard-sh check "$cmd")
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

zle -N _guard_zsh_accept_line
bindkey "^M" _guard_zsh_accept_line
bindkey "^J" _guard_zsh_accept_line
