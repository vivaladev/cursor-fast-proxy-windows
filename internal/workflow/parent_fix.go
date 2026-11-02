package workflow

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/vivaladev/cursor-fast-proxy-windows/internal/apppaths"
	"github.com/vivaladev/cursor-fast-proxy-windows/internal/cursorfiles"
	"github.com/vivaladev/cursor-fast-proxy-windows/internal/fetchbox"
	"github.com/vivaladev/cursor-fast-proxy-windows/internal/listen"
	"github.com/vivaladev/cursor-fast-proxy-windows/internal/logx"
	"github.com/vivaladev/cursor-fast-proxy-windows/internal/procenv"
	"github.com/vivaladev/cursor-fast-proxy-windows/internal/settings"
	"github.com/vivaladev/cursor-fast-proxy-windows/internal/singboxcfg"
	"github.com/vivaladev/cursor-fast-proxy-windows/internal/sshrun"
	"github.com/vivaladev/cursor-fast-proxy-windows/internal/taskplan"
)

func parent(path string) string {
	i := strings.LastIndexAny(path, `/\`)
	if i <= 0 {
		return "."
	}
	return path[:i]
}
