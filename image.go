package libcluster

import (
	"strings"
)

type ImageInfo struct {
	Name string
	Tag  string
}

// Parse an image name in the format of "name:tag" and return an ImageInfo
// struct. If no tag is defined, assume "latest".
func parseImageName(name string) *ImageInfo {
	imageInfo := &ImageInfo{
		Name: name,
		Tag:  "latest",
	}

	img := strings.Split(name, ":")
	if len(img) == 2 {
		imageInfo.Name = img[0]
		imageInfo.Tag = img[1]
	}

	return imageInfo
}
