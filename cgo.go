package main

/*
#cgo LDFLAGS: -L. -lqqcliput -framework Foundation -framework CoreGraphics -framework CoreImage -framework Vision
#include "qqcliput.h"
#include <stdlib.h>
*/
import "C"

import (
	"encoding/json"
	"unsafe"
)

func cFindQQWindow() uint32 {
	return uint32(C.find_qq_window())
}

func cOCRWindow(wid uint32) string {
	cstr := C.ocr_window(C.uint32_t(wid))
	if cstr == nil {
		return ""
	}
	defer C.free(unsafe.Pointer(cstr))
	return C.GoString(cstr)
}

func cWindowExists(wid uint32) bool {
	return C.window_exists(C.uint32_t(wid)) != 0
}

func cIsQQFrontmost() bool {
	return C.is_qq_frontmost() != 0
}

func cGetWindowBounds(wid uint32) (w int, h int) {
	cstr := C.get_window_bounds(C.uint32_t(wid))
	if cstr == nil {
		return 0, 0
	}
	defer C.free(unsafe.Pointer(cstr))
	var m map[string]int
	if err := json.Unmarshal([]byte(C.GoString(cstr)), &m); err != nil {
		return 0, 0
	}
	return m["w"], m["h"]
}

func cOCRWindowJSON(wid uint32) string {
	cstr := C.ocr_window_json(C.uint32_t(wid))
	if cstr == nil {
		return "[]"
	}
	defer C.free(unsafe.Pointer(cstr))
	return C.GoString(cstr)
}

func cOCRWindowRegionJSON(wid uint32, nx, ny, nw, nh float64) string {
	cstr := C.ocr_window_region_json(C.uint32_t(wid), C.double(nx), C.double(ny), C.double(nw), C.double(nh))
	if cstr == nil {
		return "[]"
	}
	defer C.free(unsafe.Pointer(cstr))
	return C.GoString(cstr)
}
