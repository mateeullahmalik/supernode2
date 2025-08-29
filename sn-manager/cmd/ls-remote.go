package cmd

import (
	"fmt"

	"github.com/LumeraProtocol/supernode/v2/sn-manager/internal/config"
	"github.com/LumeraProtocol/supernode/v2/sn-manager/internal/github"
	"github.com/spf13/cobra"
)

var lsRemoteCmd = &cobra.Command{
	Use:   "ls-remote",
	Short: "List available SuperNode versions",
	RunE:  runLsRemote,
}

func runLsRemote(cmd *cobra.Command, args []string) error {
    client := github.NewClient(config.GitHubRepo)

    releases, err := client.ListReleases()
    if err != nil {
        return fmt.Errorf("failed to list releases: %w", err)
    }

    // Filter to stable (non-draft, non-prerelease)
    var stable []*github.Release
    for _, r := range releases {
        if !r.Draft && !r.Prerelease {
            stable = append(stable, r)
        }
    }

    if len(stable) == 0 {
        fmt.Println("No releases found")
        return nil
    }

    fmt.Println("Available versions:")
    for i, release := range stable {
        if i == 0 {
            fmt.Printf("  %s (latest) - %s\n", release.TagName, release.PublishedAt.Format("2006-01-02"))
        } else {
            fmt.Printf("  %s - %s\n", release.TagName, release.PublishedAt.Format("2006-01-02"))
        }
        if i >= 9 {
            break
        }
    }

	return nil
}
