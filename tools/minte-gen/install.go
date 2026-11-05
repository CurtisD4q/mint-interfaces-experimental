package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// Installing minte-gen for one user.
//
// The user layout keeps everything in the user's home:
//
//	~/.local/share/mint-interfaces-experimental/            the repository; apps in apps/<tool>/
//	~/.local/bin/mint-interfaces-experimental/minte-gen      the launcher
//
// The launcher runs the repository's own minte-gen with `go -C <repo> run
// ./tools/minte-gen`, so it is never out of date after a git pull. Go
// switches into the repository first; minte-gen switches back to the
// folder it was started from (callerDir), so relative paths and the
// folder an app starts in are yours, not the repository's.

// launcherMark is in every launcher minte-gen writes, so --install only
// ever replaces its own.
const launcherMark = "# minte-gen launcher, written by minte-gen --install"

// userDataDir is $XDG_DATA_HOME, or ~/.local/share.
func userDataDir() string {
	if d := os.Getenv("XDG_DATA_HOME"); d != "" && filepath.IsAbs(d) {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".local", "share")
}

// userInstallRoot is where the user layout keeps the repository.
func userInstallRoot() string {
	if d := userDataDir(); d != "" {
		return filepath.Join(d, "mint-interfaces-experimental")
	}
	return ""
}

// userLauncherDir is the folder the user layout's launcher is in.
func userLauncherDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".local", "bin", "mint-interfaces-experimental")
}

// launchRoot is the repository minte-gen was run in by `go -C <repo> run`,
// before it switched back to the caller's folder.
var launchRoot string

// Seams for tests.
var (
	executable = os.Executable
	getwd      = os.Getwd
	chdir      = os.Chdir
)

// callerDir puts minte-gen back in the folder it was started from, when it
// was started with `go -C <repo> run ./tools/minte-gen` (the launcher, or
// by hand). Go changes into the repository but leaves $PWD naming the
// caller's folder. Only a minte-gen built by go run, sitting in the
// repository, with $PWD a different real folder, moves.
func callerDir() {
	exe, err := executable()
	if err != nil || filepath.Base(exe) != "minte-gen" || !strings.Contains(exe, string(filepath.Separator)+"go-build") {
		return
	}
	wd, err := getwd()
	if err != nil || !isRepoRoot(wd) {
		return
	}
	pwd := os.Getenv("PWD")
	if pwd == "" || !filepath.IsAbs(pwd) {
		return
	}
	a, err1 := os.Stat(pwd)
	b, err2 := os.Stat(wd)
	if err1 != nil || err2 != nil || !a.IsDir() || os.SameFile(a, b) {
		return
	}
	if chdir(pwd) == nil {
		launchRoot = wd
	}
}

// shellQuote quotes s for a POSIX shell, in single quotes.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// launcherScript is the launcher for repo, running go (an absolute path
// when it was found, so the launcher works where PATH is short).
func launcherScript(goBin, repo string) string {
	return "#!/bin/sh\n" + launcherMark + "\n" +
		"exec " + shellQuote(goBin) + " -C " + shellQuote(repo) + " run ./tools/minte-gen \"$@\"\n"
}

// installUser writes the user layout's launcher for repo.
func installUser(out, errOut io.Writer, repo string) int {
	if genEuid() == 0 {
		fmt.Fprintln(errOut, "minte-gen: --install sets minte-gen up for you; run it without sudo. For a system-wide install, see --install --system.")
		return 2
	}
	if repo == "" || strings.ContainsAny(repo, "\n\r") {
		fmt.Fprintln(errOut, noRootMessage)
		return 2
	}
	dir := userLauncherDir()
	if dir == "" {
		fmt.Fprintln(errOut, "minte-gen: cannot find your home folder")
		return 1
	}
	goBin := "go"
	if p, err := exec.LookPath("go"); err == nil {
		if abs, err := filepath.Abs(p); err == nil {
			goBin = abs
		}
	}
	path := filepath.Join(dir, "minte-gen")
	if fi, err := os.Lstat(path); err == nil {
		b, _ := os.ReadFile(path)
		if !fi.Mode().IsRegular() || !strings.Contains(string(b), launcherMark) {
			fmt.Fprintf(errOut, "minte-gen: %s is there already and minte-gen did not write it; move it away first\n", path)
			return 1
		}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fmt.Fprintf(errOut, "minte-gen: %v\n", err)
		return 1
	}
	tmp, err := os.CreateTemp(dir, ".minte-gen-*")
	if err != nil {
		fmt.Fprintf(errOut, "minte-gen: %v\n", err)
		return 1
	}
	_, werr := tmp.WriteString(launcherScript(goBin, repo))
	cerr := tmp.Close()
	if werr == nil && cerr == nil {
		werr = os.Chmod(tmp.Name(), 0o755)
	}
	if werr == nil && cerr == nil {
		werr = os.Rename(tmp.Name(), path)
	}
	if werr != nil || cerr != nil {
		_ = os.Remove(tmp.Name())
		fmt.Fprintf(errOut, "minte-gen: could not write %s: %v %v\n", path, werr, cerr)
		return 1
	}
	fmt.Fprintf(out, "Installed %s\n  runs minte-gen from %s\n", path, repo)
	if want := userInstallRoot(); want != "" && filepath.Clean(repo) != filepath.Clean(want) {
		fmt.Fprintf(out, "  (the usual place for it is %s; this works from where it is)\n", want)
	}
	if !onPath(dir) {
		fmt.Fprintf(out, "\n%s is not on your PATH. Add this line to ~/.profile (or ~/.bashrc, ~/.zshrc), then open a new terminal:\n\n  export PATH=\"$HOME/.local/bin/mint-interfaces-experimental:$PATH\"\n", dir)
	}
	return 0
}

// installSystemHelp prints the commands for the system-wide layout.
// minte-gen never runs sudo itself.
func installSystemHelp(out io.Writer, repo string) int {
	if repo == "" {
		repo = "mint-interfaces-experimental"
	}
	fmt.Fprintf(out, `To install Mint-Interfaces for everyone on this computer, run:

  sudo mv %s /usr/local/share/mint-interfaces-experimental
  sudo mkdir -p /usr/local/share/mint-interfaces-experimental/apps
  sudo chown -R "$USER:$USER" /usr/local/share/mint-interfaces-experimental/apps
  sudo tee /usr/local/bin/minte-gen >/dev/null << 'EOF'
  #!/bin/sh
  exec go -C /usr/local/share/mint-interfaces-experimental run ./tools/minte-gen "$@"
  EOF
  sudo chmod +x /usr/local/bin/minte-gen

minte-gen does not run these for you: sudo should only ever run what you typed.
`, shellQuote(repo))
	return 0
}

// onPath reports whether dir is one of the folders in $PATH.
func onPath(dir string) bool {
	for _, p := range filepath.SplitList(os.Getenv("PATH")) {
		if p != "" && filepath.Clean(p) == filepath.Clean(dir) {
			return true
		}
	}
	return false
}

// appLinkDirs are the folders a <tool>-mint link may be in.
func appLinkDirs() []string {
	dirs := []string{"/usr/local/bin"}
	if d := userLauncherDir(); d != "" {
		dirs = append(dirs, d)
	}
	return dirs
}

// othersCanSwap reports whether anyone but root could replace path before
// sudo runs it: it, or a folder above it, belongs to someone else or can
// be written by its group or everyone.
var othersCanSwap = func(path string) bool {
	p, err := filepath.Abs(path)
	if err != nil {
		return true
	}
	for {
		fi, err := os.Lstat(p)
		if err != nil {
			return true
		}
		st, ok := fi.Sys().(*syscall.Stat_t)
		if !ok || st.Uid != 0 || fi.Mode().Perm()&0o022 != 0 {
			return true
		}
		parent := filepath.Dir(p)
		if parent == p {
			return false
		}
		p = parent
	}
}
