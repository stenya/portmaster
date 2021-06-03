package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	releaseCmd = &cobra.Command{
		Use:   "release",
		Short: "Release scans the distribution directory and creates registry indexes and the symlink structure",
		RunE:  release,
	}
	preReleaseCmd = &cobra.Command{
		Use:   "prerelease",
		Short: "Stage scans the specified directory and loads the indexes - it then creates a staging index with all files newer than the stable and beta indexes",
		Args:  cobra.ExactArgs(1),
		RunE:  prerelease,
	}
	resetPreReleases  bool
	includeUnreleased bool
)

func init() {
	rootCmd.AddCommand(releaseCmd)
	rootCmd.AddCommand(preReleaseCmd)
	preReleaseCmd.Flags().BoolVar(&resetPreReleases, "reset", false, "Reset pre-release assets")
}

func release(cmd *cobra.Command, args []string) error {
	return writeIndex(
		"stable",
		getChannelVersions("", false),
	)
}

func prerelease(cmd *cobra.Command, args []string) error {
	channel := args[0]

	// Check if we want to reset instead.
	if resetPreReleases {
		return removeFilesFromIndex(getChannelVersions(channel, true))
	}

	return writeIndex(
		channel,
		getChannelVersions(channel, false),
	)
}

func writeIndex(channel string, versions map[string]string) error {
	// Export versions and format them.
	versionData, err := json.MarshalIndent(versions, "", " ")
	if err != nil {
		return err
	}

	// Build destination path.
	indexFilePath := filepath.Join(registry.StorageDir().Path, channel+".json")

	// Print preview.
	fmt.Printf("%s (%s):\n", channel, indexFilePath)
	fmt.Println(string(versionData))

	// Ask for confirmation.
	if !confirm("\nDo you want to write this index?") {
		fmt.Println("aborted...")
		return nil
	}

	// Write new index to disk.
	err = ioutil.WriteFile(indexFilePath, versionData, 0644) //nolint:gosec // 0644 is intended
	if err != nil {
		return err
	}
	fmt.Printf("written %s\n", indexFilePath)

	return nil
}

func removeFilesFromIndex(versions map[string]string) error {
	// Print preview.
	fmt.Println("To be deleted:")
	for _, filePath := range versions {
		fmt.Println(filePath)
	}

	// Ask for confirmation.
	if !confirm("\nDo you want to delete these files?") {
		fmt.Println("aborted...")
		return nil
	}

	// Delete files.
	for _, filePath := range versions {
		err := os.Remove(filePath)
		if err != nil {
			return err
		}
	}

	fmt.Println("deleted")
	return nil
}

func getChannelVersions(channel string, storagePath bool) map[string]string {
	// Sort all versions.
	registry.SelectVersions()
	export := registry.Export()

	// Go through all versions and save the highest version, if not stable or beta.
	versions := make(map[string]string)
	for _, rv := range export {
		for _, v := range rv.Versions {
			// Ignore versions that don't match the release channel.
			if v.SemVer().Prerelease() != channel {
				// Stop at the first stable version, nothing should ever be selected
				// beyond that.
				if v.SemVer().Prerelease() == "" {
					break
				}

				continue
			}

			// Add highest version of matching release channel.
			if storagePath {
				versions[rv.Identifier] = rv.GetFile().Path()
			} else {
				versions[rv.Identifier] = v.VersionNumber
			}

			break
		}
	}

	return versions
}
