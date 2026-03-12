# guard-sh bash integration

[[ $- == *i* ]] || return

_GUARD_SCRIPT_PATH="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/$(basename "${BASH_SOURCE[0]}")"

shopt -s extdebug

_GUARD_BUSY=0
_GUARD_IN_PROMPT=1

_guard_debug_trap() {
    [[ $BASH_SUBSHELL -gt 0 ]] && return 0
    [[ $_GUARD_BUSY -eq 1 ]] && return 0
    [[ $_GUARD_IN_PROMPT -eq 1 ]] && return 0
    [[ "$BASH_COMMAND" == _guard_* ]] && return 0
    [[ "$BASH_COMMAND" == guard-sh* ]] && return 0

    command -v guard-sh &>/dev/null || return 0

    local cmd="$BASH_COMMAND"
    _GUARD_BUSY=1

    local warning
    warning=$(command guard-sh check "$cmd" 2>/dev/null)
    local exit_code=$?
    _GUARD_BUSY=0

    if [[ $exit_code -ne 0 ]]; then
        printf 'guard-sh: %s ' "$warning"
        local confirm
        read -rn1 confirm </dev/tty
        printf '\n'
        if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
            return 1
        fi
    fi
    return 0
}

_guard_prompt_begin() { _GUARD_IN_PROMPT=1; }
_guard_prompt_end()   { _GUARD_IN_PROMPT=0; }

_guard_enable() {
    trap '_guard_debug_trap' DEBUG
    _GUARD_IN_PROMPT=0
    echo "guard-sh: enabled"
}

_guard_disable() {
    trap - DEBUG
    _GUARD_IN_PROMPT=1
    echo "guard-sh: disabled"
}

_guard_global_on() {
    local rc="$HOME/.bashrc"
    local source_line="source \"$_GUARD_SCRIPT_PATH\""
    local on_line="guard-sh on"

    if ! grep -qF "$source_line" "$rc" 2>/dev/null; then
        printf '\n# guard-sh\n%s\n%s\n' "$source_line" "$on_line" >> "$rc"
        echo "guard-sh: enabled globally in $rc"
        return
    fi

    if grep -qF "$on_line" "$rc" 2>/dev/null; then
        echo "guard-sh: already enabled globally in $rc"
    else
        echo "$on_line" >> "$rc"
        echo "guard-sh: enabled globally in $rc"
    fi
}

_guard_global_off() {
    local rc="$HOME/.bashrc"
    local on_line="guard-sh on"

    if grep -qF "$on_line" "$rc" 2>/dev/null; then
        grep -vF "$on_line" "$rc" > "${rc}.guardtmp" && mv "${rc}.guardtmp" "$rc"
        echo "guard-sh: disabled globally in $rc"
    else
        echo "guard-sh: not enabled globally in $rc"
    fi
}

# Works whether the trap is active or not
guard-sh() {
    _GUARD_BUSY=1
    case "$1" in
        on)  [[ "$2" == "--global" ]] && _guard_global_on || _guard_enable ;;
        off) [[ "$2" == "--global" ]] && _guard_global_off || _guard_disable ;;
        *)   command guard-sh "$@" ;;
    esac
    _GUARD_BUSY=0
}

PROMPT_COMMAND="_guard_prompt_begin${PROMPT_COMMAND:+; $PROMPT_COMMAND}; _guard_prompt_end"
