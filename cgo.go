package main

/*
#cgo LDFLAGS: -L. -lqqcliput -framework Foundation -framework CoreGraphics -framework CoreImage -framework Vision
#include "qqcliput.h"
#include <stdlib.h>
*/
import "C"

import "unsafe"

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

func cOCRWindowJSON(wid uint32) string {
	cstr := C.ocr_window_json(C.uint32_t(wid))
	if cstr == nil {
		return "[]"
	}
	defer C.free(unsafe.Pointer(cstr))
	return C.GoString(cstr)
}
