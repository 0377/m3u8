package crypt

import "errors"

var ErrNotImplemented = errors.New("hook not implemented")

type KeyMeta struct {
	Method string
	URI    string
	IV     string
}

type Context struct {
	M3U8URL    string
	SegmentURI string
	SegmentIdx int
	Method     string
	KeyMeta    KeyMeta
	Key        []byte
	IV         []byte
}

type KeyMaterial struct {
	Key []byte
	IV  []byte
}
