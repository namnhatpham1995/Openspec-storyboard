//go:build !windows

package directorybrowser

func platformLocations(home string) ([]Location, error) {
	return []Location{
		{Name: "Home", Path: home},
		{Name: "Filesystem", Path: "/"},
	}, nil
}
