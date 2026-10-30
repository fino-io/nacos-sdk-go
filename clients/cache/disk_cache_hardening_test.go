package cache

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHardeningSnapshotDeletion(t *testing.T) {
	for _, existing := range []bool{false, true} {
		for _, content := range []string{"", " \n\t"} {
			t.Run(content+map[bool]string{false: "missing", true: "existing"}[existing], func(t *testing.T) {
				name := filepath.Join(t.TempDir(), "snapshot")
				if existing {
					require.NoError(t, os.WriteFile(name, []byte("old"), 0600))
				}
				require.NoError(t, writeConfigToFile(name, content, ConfigContent))
				_, err := os.Stat(name)
				require.True(t, os.IsNotExist(err), "empty snapshot must stay deleted")
			})
		}
	}
}

func TestHardeningSnapshotDeletionFailure(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "child"), []byte("keep"), 0600))
	require.Error(t, writeConfigToFile(dir, "", ConfigContent), "nonempty directory removal must fail")
}

func TestHardeningSnapshotErrorRedaction(t *testing.T) {
	secret := "password=private-value"
	err := writeConfigToFile(filepath.Join(t.TempDir(), "missing", "config"), secret, ConfigContent)
	require.Error(t, err)
	require.NotContains(t, err.Error(), secret)
}

func TestHardeningSnapshotContent(t *testing.T) {
	name := filepath.Join(t.TempDir(), "config")
	content := "  value=中文 & +\n"
	require.NoError(t, writeConfigToFile(name, content, ConfigContent))
	data, err := os.ReadFile(name)
	require.NoError(t, err)
	require.Equal(t, content, string(data))
	require.NoError(t, writeConfigToFile(name, "replacement\n", ConfigContent))
	data, err = os.ReadFile(name)
	require.NoError(t, err)
	require.Equal(t, "replacement\n", string(data))
}
