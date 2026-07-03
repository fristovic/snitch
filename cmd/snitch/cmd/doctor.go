package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/fristovic/snitch/internal/config"
	"github.com/fristovic/snitch/internal/ipc"
	"github.com/fristovic/snitch/internal/platform"
	"github.com/fristovic/snitch/internal/version"
	"github.com/spf13/cobra"
)

var uninstallPurge bool

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check Snitch + Cursor install health",
	RunE: func(cmd *cobra.Command, args []string) error {
		ok := true
		printCheck := func(name string, pass bool, detail string) {
			mark := "ok"
			if !pass {
				mark = "FAIL"
				ok = false
			}
			if detail != "" {
				fmt.Printf("[%s] %s — %s\n", mark, name, detail)
			} else {
				fmt.Printf("[%s] %s\n", mark, name)
			}
		}

		printCheck("snitch CLI", fileExists(selfPath()), version.Version)
		snitchdPath, snitchdOnPath := findBinary("snitchd")
		printCheck("snitchd binary", snitchdOnPath, snitchdPath)

		sock := resolveSocket()
		client, err := ipc.Connect(sock)
		running := err == nil
		if running {
			_ = client.Close()
		}
		printCheck("snitchd running", running, sock)

		menubarPlist := filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents", "com.snitch.menubar.plist")
		printCheck("Snitch Bar LaunchAgent", fileExists(menubarPlist), menubarPlist)

		shareApp := filepath.Join(os.Getenv("HOME"), ".local", "share", "snitch", "Snitch Bar.app")
		printCheck("Snitch Bar.app", fileExists(filepath.Join(shareApp, "Contents", "MacOS", "snitchbar")), shareApp)

		cursorApp := fileExists("/Applications/Cursor.app")
		cursorData := fileExists(filepath.Join(os.Getenv("HOME"), ".cursor"))
		printCheck("Cursor installed", cursorApp || cursorData, "app or ~/.cursor")

		paths, _ := platform.Resolve()
		cfg, _ := config.Load(paths.ConfigPath)
		watch := cfg.Cursor.TranscriptWatchPath
		if watch == "" {
			watch = filepath.Join(os.Getenv("HOME"), ".cursor", "projects")
		}
		printCheck("transcript watch path", fileExists(watch), watch)

		if !ok {
			fmt.Println("\nSome checks failed. Run: snitch start")
			return nil
		}
		fmt.Println("\nAll checks passed. Use Start/Stop Snitching in the menu bar to pause or resume.")
		return nil
	},
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove Snitch daemon and binaries",
	RunE: func(cmd *cobra.Command, args []string) error {
		plist := filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents", "com.snitch.daemon.plist")
		_ = exec.Command("launchctl", "bootout", fmt.Sprintf("gui/%d/com.snitch.daemon", os.Getuid())).Run()
		_ = exec.Command("launchctl", "unload", plist).Run()
		if fileExists(plist) {
			_ = os.Remove(plist)
			fmt.Println("removed legacy daemon LaunchAgent")
		}
		menubarPlist := filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents", "com.snitch.menubar.plist")
		_ = exec.Command("launchctl", "bootout", fmt.Sprintf("gui/%d/com.snitch.menubar", os.Getuid())).Run()
		_ = exec.Command("launchctl", "unload", menubarPlist).Run()
		if fileExists(menubarPlist) {
			_ = os.Remove(menubarPlist)
			fmt.Println("removed Snitch Bar LaunchAgent")
		}
		_ = exec.Command("brew", "services", "stop", "snitch").Run()

		shareApp := filepath.Join(os.Getenv("HOME"), ".local", "share", "snitch", "Snitch Bar.app")
		if fileExists(shareApp) {
			_ = os.RemoveAll(shareApp)
			fmt.Println("removed Snitch Bar.app")
		}

		for _, name := range []string{"snitch", "snitchd", "snitchbar"} {
			if p, ok := findBinary(name); ok {
				if err := os.Remove(p); err == nil {
					fmt.Printf("removed %s\n", p)
				}
			}
		}

		if uninstallPurge {
			dataDir := filepath.Join(os.Getenv("HOME"), ".snitch")
			if err := os.RemoveAll(dataDir); err == nil {
				fmt.Printf("removed %s\n", dataDir)
			}
		} else {
			fmt.Println("kept ~/.snitch data (use --purge to remove)")
		}
		fmt.Println("Homebrew users: brew uninstall snitch")
		return nil
	},
}

func init() {
	uninstallCmd.Flags().BoolVar(&uninstallPurge, "purge", false, "Also remove ~/.snitch data")
	rootCmd.AddCommand(doctorCmd, uninstallCmd)
}

func fileExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

func selfPath() string {
	p, err := os.Executable()
	if err != nil {
		return ""
	}
	return p
}

func findBinary(name string) (string, bool) {
	p, err := exec.LookPath(name)
	if err != nil {
		return "", false
	}
	return p, true
}
