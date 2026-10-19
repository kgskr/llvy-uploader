package platform

import (
	"golang.org/x/sys/windows"
	"llvy-uploader/internal/discovery"
	"path/filepath"
)

func Candidates() ([]discovery.Candidate, error) {
	docs, err := windows.KnownFolderPath(windows.FOLDERID_Documents, 0)
	var candidates []discovery.Candidate
	if err == nil {
		candidates = append(candidates, discovery.Candidate{Path: filepath.Join(docs, "League of Legends", "Replays"), Source: "Windows 문서 폴더"})
	}
	for letter := 'C'; letter <= 'Z'; letter++ {
		root := string(letter) + ":\\"
		p, _ := windows.UTF16PtrFromString(root)
		if windows.GetDriveType(p) != windows.DRIVE_FIXED {
			continue
		}
		game := filepath.Join(root, "Riot Games", "League of Legends")
		candidates = append(candidates, discovery.Candidate{Path: filepath.Join(game, "Replays"), Source: "설치 경로의 Replays 폴더"}, discovery.Candidate{Path: game, Source: "설치 경로의 리플레이"})
	}
	for _, folder := range []*windows.KNOWNFOLDERID{windows.FOLDERID_ProgramFiles, windows.FOLDERID_ProgramFilesX86} {
		if base, e := windows.KnownFolderPath(folder, 0); e == nil {
			game := filepath.Join(base, "Riot Games", "League of Legends")
			candidates = append(candidates, discovery.Candidate{Path: filepath.Join(game, "Replays"), Source: "Program Files의 Replays 폴더"}, discovery.Candidate{Path: game, Source: "설치 경로의 리플레이"})
		}
	}
	return candidates, err
}
