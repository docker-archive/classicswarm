package libcluster

import "testing"

func TestImageNameParsing(t *testing.T) {
	var i *ImageInfo

	i = parseImageName("foo:bar")
	if i.Name != "foo" || i.Tag != "bar" {
		t.Fatalf("Parsing failed: %#v", i)
	}

	i = parseImageName("foo")
	if i.Name != "foo" || i.Tag != "latest" {
		t.Fatalf("Parsing failed: %#v", i)
	}
}
