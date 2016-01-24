package filter

import "testing"

func TestParseRepositoryTag(t *testing.T) {

	repo, tag := parseRepositoryTag("localhost.localdomain:5000/samalba/hipache:latest")
	if tag != "latest" {
		t.Errorf("repo=%s tag=%s", repo, tag)
	}
	repo, tag = parseRepositoryTag("localhost:5000/foo/bar@sha256:bc8813ea7b3603864987522f02a76101c17ad122e1c46d790efc0fca78ca7bfb")
	if tag != "sha256:bc8813ea7b3603864987522f02a76101c17ad122e1c46d790efc0fca78ca7bfb" {
		t.Logf("repo=%s tag=%s", repo, tag)
	}
	repo, tag = parseRepositoryTag("localhost:5000/foo/bar")
	if tag != "" {
		t.Logf("repo=%s tag=%s", repo, tag)
	}
	repo, tag = parseRepositoryTag("localhost:5000/foo/bar:latest")
	t.Logf("repo=%s tag=%s", repo, tag)
	if tag != "latest" {
		t.Logf("repo=%s tag=%s", repo, tag)
	}
}