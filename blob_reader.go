package menmos

import (
	"io"
)

type rangeReader struct {
	BlobID string
	Client *Client

	RangeStart int64
	RangeEnd   int64
}

func (r *rangeReader) Read(buf []byte) (int, error) {
	if r.RangeStart > r.RangeEnd {
		return 0, io.EOF
	}

	requestedDataLength := int64(len(buf))
	byteRange := (r.RangeEnd - r.RangeStart) + 1

	var lengthToRead int64
	if requestedDataLength <= byteRange {
		lengthToRead = requestedDataLength
	} else {
		lengthToRead = byteRange
	}

	rangeEnd := (r.RangeStart + lengthToRead) - 1

	responseReader, err := r.Client.readRange(r.BlobID, r.RangeStart, rangeEnd)
	if err != nil {
		return 0, err
	}
	defer responseReader.Close()

	readCount, err := io.ReadFull(responseReader, buf[:lengthToRead])
	if err != nil {
		return 0, err
	}

	r.RangeStart += int64(readCount)
	return readCount, nil
}

func (r *rangeReader) Close() error {
	return nil
}
