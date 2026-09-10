// Command ccpet is the whole thing in one binary: the statusline, the pet's
// panel and the hook that feeds it. One process start, no interpreter, no
// shell front end.
//
//	ccpet statusline    read a refresh payload on stdin, print the footer
//	ccpet hook          read a hook payload on stdin, turn it into food
//	ccpet invade        the shooter: the pet you have is the cannon
//	ccpet arena on|off  let the game open itself while Claude works
//	ccpet               the pet's panel
//	ccpet feed|tests|commit|compact|task|overflow      a meal
//	ccpet count|day|record|session                     bookkeeping
//	ccpet lang es|en|auto                              what language it speaks
package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kyros-software/claude-code-themes/internal/hook"
	"github.com/kyros-software/claude-code-themes/internal/i18n"
	"github.com/kyros-software/claude-code-themes/internal/invaders"
	"github.com/kyros-software/claude-code-themes/internal/panel"
	"github.com/kyros-software/claude-code-themes/internal/pet"
	"github.com/kyros-software/claude-code-themes/internal/setup"
	"github.com/kyros-software/claude-code-themes/internal/statusline"
)

// version is stamped at build time with -ldflags "-X main.version=...".
var version = "dev"

func main() { os.Exit(run(os.Args, os.Stdin, os.Stdout, os.Stderr, time.Now())) }

// run is main with its edges handed in, so the dispatch can be tested. It
// returns the exit code rather than calling os.Exit, which is the only reason
// main is one line.
func run(argv []string, stdin io.Reader, stdout, stderr io.Writer, now time.Time) int {
	args := takeLang(argv[1:])

	// argv[0] can carry the command. ~/.claude/ccpet-statusline is a symlink to
	// this binary, which lets settings.json hold a bare path with no arguments:
	// the shape that works whether or not the host uses a shell.
	if strings.Contains(filepath.Base(argv[0]), "statusline") {
		if err := statusline.Run(stdin, stdout); err != nil {
			return 1
		}
		return 0
	}

	if len(args) > 0 {
		switch args[0] {
		case "statusline":
			if err := statusline.Run(stdin, stdout); err != nil {
				return 1
			}
			return 0
		case "hook":
			return hook.Run(stdin, pet.Path(), now)
		case "setup":
			return runSetup(args[1:], stdout, stderr)
		case "lang":
			return runLang(args[1:], stdout, stderr)
		case "invade":
			// The paths are resolved here, like every other verb: a package
			// under internal/ is handed where its state lives rather than
			// going and finding it, which is what lets the tests point them
			// somewhere harmless.
			return invaders.Run(args[1:], stdout, stderr, pet.Path(), invaders.SavePath(), now)
		case "arena":
			return runArena(args[1:], stdout, stderr)
		case "link":
			root := ""
			if len(args) > 1 {
				root = args[1]
			}
			setup.RunLink(stdout, root)
			return 0
		case "version", "--version", "-v":
			fmt.Fprintln(stdout, version)
			return 0
		case "-h", "--help", "help":
			fmt.Fprint(stdout, i18n.S().Usage)
			return 0
		}
	}
	return panel.Run(args, stdout, stderr, pet.Path(), now)
}

// runArena is the switch that lets a turn open the game: on, off, or say which
// it is. Three lines of state - one file that exists or does not - because what
// it guards is a program opening a window on somebody's desktop, and that is a
// thing to have said yes to out loud.
func runArena(args []string, stdout, stderr io.Writer) int {
	g := i18n.G()
	action := "status"
	if len(args) > 0 && args[0] != "" {
		action = args[0]
	}
	switch action {
	case "on", "off":
		if err := invaders.SetArena(action == "on"); err != nil {
			fmt.Fprintln(stderr, "ccpet:", err)
			return 1
		}
	case "status":
	default:
		fmt.Fprintln(stderr, g.ArenaUsage)
		return 2
	}
	if !invaders.ArenaOn() {
		fmt.Fprintln(stdout, g.ArenaIsOff)
		return 0
	}
	fmt.Fprintln(stdout, g.ArenaIsOn)
	// Said at the moment it is switched on, not the first time a turn silently
	// fails to open anything: there is nothing on this machine to open a window
	// with, and the arena will do nothing until there is.
	if !invaders.CanOpenAWindow() {
		fmt.Fprintln(stderr, "ccpet:", g.NoTerminal)
	}
	return 0
}

// runSetup writes the one settings.json key a plugin cannot install by itself.
func runSetup(args []string, stdout, stderr io.Writer) int {
	action := "on"
	if len(args) > 0 && args[0] != "" {
		action = args[0]
	}
	root := ""
	if len(args) > 1 {
		root = args[1]
	}
	if root == "" {
		root = defaultRuntimeRoot()
	}

	var err error
	switch action {
	case "on":
		err = setup.On(stdout, root)
	case "off":
		err = setup.Off(stdout)
	case "status":
		err = setup.Status(stdout)
	case "install", "install-hooks":
		err = setup.Install(stdout, root, action == "install-hooks")
	case "uninstall":
		err = setup.Uninstall(stdout)
	default:
		fmt.Fprintln(stderr, i18n.S().SetupUsage)
		return 2
	}
	if err != nil {
		fmt.Fprintln(stderr, "ccpet:", err)
		return 1
	}
	return 0
}

// takeLang pulls a leading `--lang xx` or `--lang=xx` off the arguments and
// pins it for the rest of the process, so any command can be run in the other
// language without changing the setting: `ccpet --lang en` reads the panel in
// English and leaves the configured language alone. It has to happen before
// the dispatch, because the dispatch itself prints in whatever is set.
func takeLang(args []string) []string {
	for len(args) > 0 {
		arg := args[0]
		switch {
		case strings.HasPrefix(arg, "--lang="):
			i18n.Use(i18n.Lang(strings.TrimPrefix(arg, "--lang=")))
			args = args[1:]
		case arg == "--lang" && len(args) > 1:
			i18n.Use(i18n.Lang(args[1]))
			args = args[2:]
		default:
			return args
		}
	}
	return args
}

// runLang is the setting itself: with no argument it says what the theme
// speaks and who decided that, and with one it writes it down.
//
// `auto` is stored as `auto` rather than resolved and stored: somebody who
// asks for the locale to decide means every future session, not the language
// their locale happened to name this afternoon.
func runLang(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "" {
		lang, source := i18n.Setting()
		if lang == "" {
			lang = i18n.Default
		}
		fmt.Fprintf(stdout, i18n.S().LangIs, lang, source)
		return 0
	}
	want := i18n.Lang(strings.ToLower(args[0]))
	switch want {
	case i18n.ES, i18n.EN, i18n.Auto:
	default:
		fmt.Fprintln(stderr, i18n.S().LangUsage)
		return 2
	}
	if err := i18n.Save(want); err != nil {
		fmt.Fprintln(stderr, "ccpet:", err)
		return 1
	}
	// Printed AFTER the write, and in the language just chosen: the
	// confirmation is the first thing the new setting has to say.
	i18n.Use("")
	if want == i18n.Auto {
		// "es (auto)" and not a bare "es": what was written down is the
		// question, and the answer is only today's answer.
		fmt.Fprintf(stdout, i18n.S().LangIs, i18n.Current(), i18n.Auto)
		return 0
	}
	fmt.Fprintf(stdout, i18n.S().LangSaved, want)
	return 0
}

// defaultRuntimeRoot is the stable path the statusline is pointed at. The
// binary itself may live inside a version-stamped plugin directory, which is
// exactly the path that must not end up in settings.json.
func defaultRuntimeRoot() string {
	return filepath.Join(setup.ConfigDir(), "ccpet")
}
