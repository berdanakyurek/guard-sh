# guard-sh fish integration

set -g _GUARD_SCRIPT_PATH (status filename)

function _guard_execute
    set -l cmd (commandline)

    # Empty buffer — pass through
    if test -z (string trim -- $cmd)
        commandline -f execute
        return
    end

    if not command -v guard-sh &>/dev/null
        commandline -f execute
        return
    end

    set -l warning (command guard-sh check $cmd 2>/dev/null)
    set -l exit_code $status

    if test $exit_code -eq 0
        commandline -f execute
        return
    end

    printf '\nguard-sh: %s ' $warning
    set -l confirm (read --nchars 1 --silent)
    printf '\n'

    if string match -qr '^[Yy]$' -- $confirm
        commandline -f execute
    else
        commandline -f repaint
    end
end

function _guard_enable
    bind \n _guard_execute
    bind \r _guard_execute
    echo "guard-sh: enabled"
end

function _guard_disable
    bind --erase \n
    bind --erase \r
    echo "guard-sh: disabled"
end

function _guard_global_on
    set -l rc "$HOME/.config/fish/config.fish"
    set -l source_line "source $_GUARD_SCRIPT_PATH"
    set -l on_line "guard-sh on"

    if not grep -qF $source_line $rc 2>/dev/null
        printf '\n# guard-sh\n%s\n%s\n' $source_line $on_line >> $rc
        echo "guard-sh: enabled globally in $rc"
        return
    end

    if grep -qF $on_line $rc 2>/dev/null
        echo "guard-sh: already enabled globally in $rc"
    else
        echo $on_line >> $rc
        echo "guard-sh: enabled globally in $rc"
    end
end

function _guard_global_off
    set -l rc "$HOME/.config/fish/config.fish"
    set -l on_line "guard-sh on"

    if grep -qF $on_line $rc 2>/dev/null
        grep -vF $on_line $rc > $rc.guardtmp && mv $rc.guardtmp $rc
        echo "guard-sh: disabled globally in $rc"
    else
        echo "guard-sh: not enabled globally in $rc"
    end
end

function guard-sh
    switch $argv[1]
        case on
            if test "$argv[2]" = "--global"
                _guard_global_on
            else
                _guard_enable
            end
        case off
            if test "$argv[2]" = "--global"
                _guard_global_off
            else
                _guard_disable
            end
        case status
            set -l session off
            bind \n 2>/dev/null | string match -q '*_guard_execute*' && set session on
            set -l global off
            grep -qF "guard-sh on" "$HOME/.config/fish/config.fish" 2>/dev/null && set global on
            command guard-sh status --session="$session" --global="$global"
        case '*'
            command guard-sh $argv
    end
end
