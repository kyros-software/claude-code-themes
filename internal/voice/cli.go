package voice

// `ccpet voice ...`.

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/kyros-software/claude-code-themes/internal/pet"
	"github.com/kyros-software/claude-code-themes/internal/session"
)

const usage = "uso: ccpet voice {on|off|status|say} | ccpet voice shape {on|off}\n"

// Run is `ccpet voice ...`.
func Run(args []string, stdout io.Writer, statePath string, now time.Time) int {
	path := PathBeside(statePath)
	action := "status"
	if len(args) > 0 {
		action = args[0]
	}
	switch action {
	case "on", "off":
		on := action == "on"
		if !update(path, func(f *File) bool { f.Enabled = on; return true }) {
			fmt.Fprintln(stdout, "no he podido escribir "+path)
			return 1
		}
		if on {
			if _, err := exec.LookPath("claude"); err != nil {
				fmt.Fprintln(stdout, "voz encendida, pero no encuentro `claude` en el PATH: no dirá nada nuevo")
				return 0
			}
			fmt.Fprintln(stdout, "voz encendida · haiku desempata la forma cuando las reglas dudan y escribe tres frases cada media hora de trabajo")
		} else {
			fmt.Fprintln(stdout, "voz apagada · la forma de cada sesión la eligen las reglas, y habla el repertorio")
		}
		return 0
	case "shape":
		// The per-session shapes, which need no model and are on by default.
		if len(args) < 2 || (args[1] != "on" && args[1] != "off") {
			fmt.Fprint(stdout, usage)
			return 2
		}
		fixed := args[1] == "off"
		if !update(path, func(f *File) bool { f.Fixed = fixed; return true }) {
			fmt.Fprintln(stdout, "no he podido escribir "+path)
			return 1
		}
		if fixed {
			fmt.Fprintln(stdout, "forma fija · todas las sesiones llevan la forma del árbol")
		} else {
			fmt.Fprintln(stdout, "forma por sesión · cada sesión lleva la que pide su trabajo")
		}
		return 0
	case "status":
		f := Load(path)
		if f.Fixed {
			fmt.Fprintln(stdout, "forma fija (la del árbol)")
		} else {
			fmt.Fprintln(stdout, "forma por sesión")
		}
		state := "apagada"
		if f.Enabled {
			state = "encendida"
		}
		fmt.Fprintln(stdout, "voz "+state)
		if !f.Enabled {
			fmt.Fprintln(stdout, "enciéndela con `ccpet voice on` (o /pet-voice on): usa haiku con tu sesión de claude")
		}
		// Each live session, with the shape it wears and what it last said.
		paths, _ := filepath.Glob(filepath.Join(session.TmpDir(), session.Prefix+"voice-*"))
		sort.Strings(paths)
		for _, p := range paths {
			if strings.HasSuffix(p, ".lock") {
				continue
			}
			// A session that died without a SessionEnd leaves its file for
			// the daily sweep; it is not a session any more.
			if info, err := os.Stat(p); err != nil || now.Sub(info.ModTime()) > 2*ShapeEvery {
				continue
			}
			sf := Load(p)
			if sf.Mark == "" && (len(sf.Lines) == 0 || !f.Enabled) {
				continue
			}
			shape := "-"
			if _, ok := pet.Unlocks[sf.Mark]; ok {
				shape = pet.Name(sf.Mark)
			}
			said := "aún no ha dicho nada"
			if !f.Enabled {
				said = "por reglas"
			} else if len(sf.Lines) > 0 {
				// Cleaned like the statusline cleans them: the file is on
				// disk and this goes to a terminal.
				var shown []string
				for _, line := range sf.Lines {
					if line = Clean(line); line != "" {
						shown = append(shown, line)
					}
				}
				said = time.Unix(sf.At, 0).Format("15:04") + " · " + strings.Join(shown, " / ")
			}
			fmt.Fprintf(stdout, "sesión · %-13s · %s\n", shape, said)
		}
		return 0
	case "say":
		// The hook's detached half, and the way to try it by hand. Works with
		// the voice off, on purpose: it is how one finds out what it sounds
		// like before switching it on.
		var c Context
		json.Unmarshal([]byte(os.Getenv(ctxEnv)), &c)
		// From the hook it asks only if the session's lines are spent; typed
		// by hand it always asks.
		lines := Say(statePath, c, now, os.Getenv(ChildEnv) == "")
		if len(lines) == 0 {
			fmt.Fprintln(stdout, "(sin frases: el modelo no ha respondido o ninguna cabía)")
			return 1
		}
		for _, line := range lines {
			fmt.Fprintln(stdout, line)
		}
		return 0
	}
	fmt.Fprint(stdout, usage)
	return 2
}
