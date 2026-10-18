package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/Paksonite/gitnest/v2/internal/state"
	"github.com/Paksonite/gitnest/v2/internal/update"
)

// updateCmd installs the latest release the same way gitnest was installed
// (--check only says whether there is one).
func updateCmd(args []string) error {
	check, force := false, false
	for _, a := range args {
		switch a {
		case "--check", "-c":
			check = true
		case "--force", "-f":
			force = true
		default:
			return fmt.Errorf("update: unknown flag %q (update [--check] [--force])", a)
		}
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	cur, _, _ := buildVersion()
	if exe, err := update.Executable(); err == nil && update.IsDevBinary(exe) {
		return updateDev(ctx, exe, cur, check)
	}
	checkCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	rel, newer, err := update.Check(checkCtx, cur, state.UpdateCachePath(), 0)
	cancel()
	if err != nil {
		return err
	}
	exe, err := update.Executable()
	if err != nil {
		return err
	}
	method := update.Detect(exe, update.PackageOwner, update.GopathBin())

	switch {
	case newer:
		fmt.Printf("gitnest %s is available (you have %s)\n", rel.Version, cur)
	case force:
		fmt.Printf("gitnest %s is the latest release (you have %s); reinstalling\n", rel.Version, cur)
	case !update.IsRelease(cur):
		fmt.Printf("gitnest %s is a development build; the latest release is %s (update --force installs it)\n", cur, rel.Version)
		return nil
	default:
		fmt.Printf("gitnest %s is up to date\n", cur)
		return nil
	}
	fmt.Printf("installed with: %s (%s)\n", method, exe)
	if rel.URL != "" {
		fmt.Println("release notes:  " + rel.URL)
	}
	if check {
		fmt.Println("\nrun `gitnest update` to install it")
		return nil
	}
	fmt.Println()
	if err := update.Apply(ctx, method, rel, exe, update.TerminalRunner, os.Stdout); err != nil {
		return fmt.Errorf("update: %w", err)
	}
	fmt.Printf("\ngitnest %s installed ✓  (restart gitnest to use it)\n", rel.Version)
	return nil
}

// updateDev updates a gitnest-dev preview build from the rolling dev
// pre-release; release versions are left alone.
func updateDev(ctx context.Context, exe, cur string, check bool) error {
	_, commit, _ := buildVersion()
	checkCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	latest, newer, err := update.CheckDev(checkCtx, commit, state.DevUpdateCachePath(), 0)
	cancel()
	if err != nil {
		return err
	}
	built := latest.Published.Local().Format("2006-01-02 15:04")
	if !newer {
		fmt.Printf("gitnest-dev %s is the latest dev build (built %s)\n", cur, built)
		return nil
	}
	fmt.Printf("gitnest-dev %s is available (built %s); you have %s\n", latest.Version(), built, cur)
	if check {
		fmt.Println("run `gitnest-dev update` to install it")
		return nil
	}
	fmt.Println()
	changed, err := update.ApplyDev(ctx, exe, os.Stdout)
	if err != nil {
		return fmt.Errorf("update: %w", err)
	}
	if !changed { // the dev release moved on between the check and the download
		fmt.Println("already the latest dev build")
		return nil
	}
	fmt.Printf("\ngitnest-dev %s → %s installed ✓  (restart it to use it)\n", cur, latest.Version())
	return nil
}
