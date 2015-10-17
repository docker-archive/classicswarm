// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package webdav

import (
	"encoding/xml"
	"net/http"
	"os"
	"strconv"
)

// PropSystem manages the properties of named resources. It allows finding
// and setting properties as defined in RFC 4918.
//
// The elements in a resource name are separated by slash ('/', U+002F)
// characters, regardless of host operating system convention.
type PropSystem interface {
	// Find returns the status of properties named propnames for resource name.
	//
	// Each Propstat must have a unique status and each property name must
	// only be part of one Propstat element.
	Find(name string, propnames []xml.Name) ([]Propstat, error)

	// TODO(rost) PROPPATCH.
	// TODO(nigeltao) merge Find and Allprop?

	// Allprop returns the properties defined for resource name and the
	// properties named in include. The returned Propstats are handled
	// as in Find.
	//
	// Note that RFC 4918 defines 'allprop' to return the DAV: properties
	// defined within the RFC plus dead properties. Other live properties
	// should only be returned if they are named in 'include'.
	//
	// See http://www.webdav.org/specs/rfc4918.html#METHOD_PROPFIND
	Allprop(name string, include []xml.Name) ([]Propstat, error)

	// Propnames returns the property names defined for resource name.
	Propnames(name string) ([]xml.Name, error)

	// TODO(rost) COPY/MOVE/DELETE.
}

// Propstat describes a XML propstat element as defined in RFC 4918.
// See http://www.webdav.org/specs/rfc4918.html#ELEMENT_propstat
type Propstat struct {
	// Props contains the properties for which Status applies.
	Props []Property

	// Status defines the HTTP status code of the properties in Prop.
	// Allowed values include, but are not limited to the WebDAV status
	// code extensions for HTTP/1.1.
	// http://www.webdav.org/specs/rfc4918.html#status.code.extensions.to.http11
	Status int

	// XMLError contains the XML representation of the optional error element.
	// XML content within this field must not rely on any predefined
	// namespace declarations or prefixes. If empty, the XML error element
	// is omitted.
	XMLError string

	// ResponseDescription contains the contents of the optional
	// responsedescription field. If empty, the XML element is omitted.
	ResponseDescription string
}

// memPS implements an in-memory PropSystem. It supports all of the mandatory
// live properties of RFC 4918.
type memPS struct {
	// TODO(rost) memPS will get writeable in the next CLs.
	fs FileSystem
	ls LockSystem
}

// NewMemPS returns a new in-memory PropSystem implementation.
func NewMemPS(fs FileSystem, ls LockSystem) PropSystem {
	return &memPS{fs: fs, ls: ls}
}

type propfindFn func(*memPS, string, os.FileInfo) (string, error)

// davProps contains all supported DAV: properties and their optional
// propfind functions. A nil value indicates a hidden, protected property.
var davProps = map[xml.Name]propfindFn{
	xml.Name{Space: "DAV:", Local: "resourcetype"}:       (*memPS).findResourceType,
	xml.Name{Space: "DAV:", Local: "displayname"}:        (*memPS).findDisplayName,
	xml.Name{Space: "DAV:", Local: "getcontentlength"}:   (*memPS).findContentLength,
	xml.Name{Space: "DAV:", Local: "getlastmodified"}:    (*memPS).findLastModified,
	xml.Name{Space: "DAV:", Local: "creationdate"}:       nil,
	xml.Name{Space: "DAV:", Local: "getcontentlanguage"}: nil,

	// TODO(rost) ETag and ContentType will be defined the next CL.
	// xml.Name{Space: "DAV:", Local: "getcontenttype"}:     (*memPS).findContentType,
	// xml.Name{Space: "DAV:", Local: "getetag"}:            (*memPS).findEtag,

	// TODO(nigeltao) Lock properties will be defined later.
	// xml.Name{Space: "DAV:", Local: "lockdiscovery"}: nil, // TODO(rost)
	// xml.Name{Space: "DAV:", Local: "supportedlock"}: nil, // TODO(rost)
}

func (ps *memPS) Find(name string, propnames []xml.Name) ([]Propstat, error) {
	fi, err := ps.fs.Stat(name)
	if err != nil {
		return nil, err
	}

	pm := make(map[int]Propstat)
	for _, pn := range propnames {
		p := Property{XMLName: pn}
		s := http.StatusNotFound
		if fn := davProps[pn]; fn != nil {
			xmlvalue, err := fn(ps, name, fi)
			if err != nil {
				return nil, err
			}
			s = http.StatusOK
			p.InnerXML = []byte(xmlvalue)
		}
		pstat := pm[s]
		pstat.Props = append(pstat.Props, p)
		pm[s] = pstat
	}

	pstats := make([]Propstat, 0, len(pm))
	for s, pstat := range pm {
		pstat.Status = s
		pstats = append(pstats, pstat)
	}
	return pstats, nil
}

func (ps *memPS) Propnames(name string) ([]xml.Name, error) {
	fi, err := ps.fs.Stat(name)
	if err != nil {
		return nil, err
	}
	propnames := make([]xml.Name, 0, len(davProps))
	for pn, findFn := range davProps {
		// TODO(rost) ETag and ContentType will be defined the next CL.
		// memPS implements ETag as the concatenated hex values of a file's
		// modification time and size. This is not a reliable synchronization
		// mechanism for directories, so we do not advertise getetag for
		// DAV collections. Other property systems may do how they please.
		if fi.IsDir() && pn.Space == "DAV:" && pn.Local == "getetag" {
			continue
		}
		if findFn != nil {
			propnames = append(propnames, pn)
		}
	}
	return propnames, nil
}

func (ps *memPS) Allprop(name string, include []xml.Name) ([]Propstat, error) {
	propnames, err := ps.Propnames(name)
	if err != nil {
		return nil, err
	}
	// Add names from include if they are not already covered in propnames.
	nameset := make(map[xml.Name]bool)
	for _, pn := range propnames {
		nameset[pn] = true
	}
	for _, pn := range include {
		if !nameset[pn] {
			propnames = append(propnames, pn)
		}
	}
	return ps.Find(name, propnames)
}

func (ps *memPS) findResourceType(name string, fi os.FileInfo) (string, error) {
	if fi.IsDir() {
		return `<collection xmlns="DAV:"/>`, nil
	}
	return "", nil
}

func (ps *memPS) findDisplayName(name string, fi os.FileInfo) (string, error) {
	if slashClean(name) == "/" {
		// Hide the real name of a possibly prefixed root directory.
		return "", nil
	}
	return fi.Name(), nil
}

func (ps *memPS) findContentLength(name string, fi os.FileInfo) (string, error) {
	return strconv.FormatInt(fi.Size(), 10), nil
}

func (ps *memPS) findLastModified(name string, fi os.FileInfo) (string, error) {
	return fi.ModTime().Format(http.TimeFormat), nil
}
