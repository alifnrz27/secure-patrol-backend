package imageutil

// JPEGOrientation reads the EXIF orientation tag (0x0112) of a JPEG; 1 when absent.
func JPEGOrientation(data []byte) int {
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return 1
	}

	for i := 2; i+4 <= len(data); {
		if data[i] != 0xFF {
			return 1
		}
		marker := data[i+1]
		if marker == 0xDA || marker == 0xD9 { // start of scan / end of image
			return 1
		}
		length := int(data[i+2])<<8 | int(data[i+3])
		if length < 2 || i+2+length > len(data) {
			return 1
		}
		segment := data[i+4 : i+2+length]
		if marker == 0xE1 && len(segment) > 6 && string(segment[:6]) == "Exif\x00\x00" {
			return tiffOrientation(segment[6:])
		}
		i += 2 + length
	}
	return 1
}

func tiffOrientation(tiff []byte) int {
	if len(tiff) < 8 {
		return 1
	}

	var u16 func([]byte) uint16
	var u32 func([]byte) uint32
	switch string(tiff[:2]) {
	case "II":
		u16 = func(b []byte) uint16 { return uint16(b[0]) | uint16(b[1])<<8 }
		u32 = func(b []byte) uint32 { return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24 }
	case "MM":
		u16 = func(b []byte) uint16 { return uint16(b[1]) | uint16(b[0])<<8 }
		u32 = func(b []byte) uint32 { return uint32(b[3]) | uint32(b[2])<<8 | uint32(b[1])<<16 | uint32(b[0])<<24 }
	default:
		return 1
	}

	offset := int(u32(tiff[4:8]))
	if offset < 8 || offset+2 > len(tiff) {
		return 1
	}
	entries := int(u16(tiff[offset : offset+2]))
	for e := 0; e < entries; e++ {
		entry := offset + 2 + e*12
		if entry+12 > len(tiff) {
			return 1
		}
		if u16(tiff[entry:entry+2]) == 0x0112 {
			value := int(u16(tiff[entry+8 : entry+10]))
			if value >= 1 && value <= 8 {
				return value
			}
			return 1
		}
	}
	return 1
}
