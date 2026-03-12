# guard-sh bash integration

_guard_bash_accept_line() {
    local cmd="$READLINE_LINE"

    if [[ -z "${cmd// /}" ]]; then
        return
    fi

    if ! command -v guard-sh &>/dev/null; then
        history -s "$cmd"
        printf '\n'
        READLINE_LINE=""
        eval "$cmd"
        return
    fi

    local warning
    warning=$(guard-sh check "$cmd")
    local exit_code=$?

    if [[ $exit_code -ne 0 ]]; then
        printf '\nguard-sh: %s ' "$warning"
        local confirm
        read -rn1 confirm
        printf '\n'
        if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
            READLINE_LINE=""
            READLINE_POINT=0
            return
        fi
    fi

    history -s "$cmd"
    printf '\n'
    READLINE_LINE=""
    eval "$cmd"
}

bind -x '"\C-m": _guard_bash_accept_line'
bind -x '"\C-j": _guard_bash_accept_line'
