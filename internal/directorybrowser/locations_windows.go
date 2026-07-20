//go:build windows

package directorybrowser

import (
	"fmt"

	"golang.org/x/sys/windows"
)

func platformLocations(home string) ([]Location, error) {
	mask, err := windows.GetLogicalDrives()
	if err != nil {
		return nil, fmt.Errorf("discovering Windows drives: %w", err)
	}
	locations := []Location{{Name: "Home", Path: home}}
	for index := 0; index < 26; index++ {
		if mask&(1<<index) == 0 {
			continue
		}
		root := fmt.Sprintf("%c:\\", 'A'+index)
		locations = append(locations, Location{Name: root, Path: root})
	}
	return locations, nil
}
